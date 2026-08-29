<script lang="ts">
  // 全站排行榜弹窗。源：ui/AppViews.tsx:3740-3905。players 保留为 prop
  // （App.svelte 传入的是带 TTL 缓存刷新节奏的 leaderboardSource 快照，是渲染时机
  // 特化的数据，不适合提升为全局 store 字段）。onClose 直接操作 uiStore。
  import type { PublicPlayer } from "../../shared/types";
  import Crown from "@lucide/svelte/icons/crown";
  import {
    GAME_LEADERBOARD_TABS, type GlobalLeaderboardTab, gameWLDOf, isGameLeaderboardTab, leaderboardPlayers,
    rosterCache, ROSTER_CACHE_TTL_MS, fetchRosterPages, setRosterCache
  } from "../../lib/leaderboard";
  import { safePlayerStats, winRateText, totalOnlineMsOf, contributionApprovedCountOf } from "../../lib/playerDisplay";
  import { formatOnlineDuration } from "../../lib/format";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";
  import LeaderboardExtra from "./LeaderboardExtra.svelte";

  let { players }: { players: PublicPlayer[] } = $props();

  function close() {
    uiStore.leaderboardOpen = false;
  }

  let tab = $state<GlobalLeaderboardTab>("positive");
  let now = $state(Date.now());
  // 全站档案（含长期离线 + 分游戏战绩）按需分页拉取，不依赖大厅实时快照。
  let roster = $state<PublicPlayer[] | null>(rosterCache?.data ?? null);
  let rosterLoading = $state(!rosterCache);
  let rosterError = $state("");

  $effect(() => {
    let alive = true;
    const cached = rosterCache;
    const cacheFresh = !!cached && Date.now() - cached.fetchedAt < ROSTER_CACHE_TTL_MS;
    rosterError = "";
    if (cached) {
      // 有缓存（哪怕已过期）先展示，避免每次点开都对着空列表等待。
      roster = cached.data;
      rosterLoading = false;
    } else {
      rosterLoading = true;
    }
    if (cacheFresh) {
      return () => { alive = false; };
    }
    (async () => {
      try {
        const data = await fetchRosterPages(() => alive);
        if (!alive || !data) return;
        setRosterCache(data);
        roster = data;
      } catch (error) {
        if (!alive) return;
        // 已有缓存兜底数据时，后台刷新失败不打断展示，静默忽略即可。
        if (!cached) {
          rosterError = error instanceof Error ? error.message : "加载排行榜失败";
          roster = null;
        }
      } finally {
        if (alive) rosterLoading = false;
      }
    })();
    return () => { alive = false; };
  });

  const source = $derived(roster && roster.length ? roster : players);

  $effect(() => {
    const hasTimer = source.some((player) =>
      (player.nameWarRenameProtectedUntil && player.nameWarRenameProtectedUntil > Date.now()) ||
      (player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > Date.now())
    );
    if (!hasTimer) return;
    const timer = window.setInterval(() => { now = Date.now(); }, 1000);
    return () => window.clearInterval(timer);
  });

  const ranked = $derived(leaderboardPlayers(source, tab).slice(0, 50));
  const gameTabMeta = $derived(GAME_LEADERBOARD_TABS.find((item) => item.id === tab));
  const title = $derived.by(() => {
    if (tab === "positive") return "当前正分榜";
    if (tab === "negative") return "当前负分榜";
    if (tab === "historyPositive") return "历史最高分榜";
    if (tab === "historyNegative") return "历史最低分榜";
    if (tab === "extremePositive") return "极限正分榜";
    if (tab === "extremeNegative") return "极限负分榜";
    if (tab === "nameWar") return "名字争夺战榜";
    if (tab === "giveaway") return "白给榜";
    if (tab === "totalWins") return "总局数榜";
    if (tab === "onlineTime") return "在线时长榜";
    if (tab === "contribution") return "共建榜";
    return gameTabMeta?.title || "排行榜";
  });

  const mainTabs: Array<{ id: GlobalLeaderboardTab; label: string }> = [
    { id: "positive", label: "当前正" }, { id: "negative", label: "当前负" },
    { id: "historyPositive", label: "历史正" }, { id: "historyNegative", label: "历史负" },
    { id: "extremePositive", label: "极限正" }, { id: "extremeNegative", label: "极限负" },
    { id: "nameWar", label: "名争" }, { id: "giveaway", label: "白给" },
    { id: "totalWins", label: "总局数" }, { id: "onlineTime", label: "在线时长" }, { id: "contribution", label: "共建" }
  ];
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="modal-backdrop leaderboard-backdrop" onclick={(event) => { if (event.target === event.currentTarget) close(); }}>
  <section class="leaderboard-modal">
    <div class="modal-title">
      <div>
        <h2><Crown size={20} /> 排行榜</h2>
        <p class="hint">打开时从服务端拉取全站档案（含离线），每类最多显示 50 名。总榜胜负平为各游戏合计。</p>
      </div>
      <button type="button" class="icon-button" onclick={close}>×</button>
    </div>
    <div class="segmented leaderboard-tabs">
      {#each mainTabs as item (item.id)}
        <button class={tab === item.id ? "active" : ""} onclick={() => (tab = item.id)}>{item.label}</button>
      {/each}
      {#each GAME_LEADERBOARD_TABS as item (item.id)}
        <button class={tab === item.id ? "active" : ""} onclick={() => (tab = item.id)}>{item.label}</button>
      {/each}
    </div>
    <div class="global-leaderboard-list">
      <h3>{title}</h3>
      {#if rosterLoading}<p class="hint">正在加载全站档案…</p>{/if}
      {#if rosterError}<p class="hint">{rosterError}（已降级为大厅实时名单）</p>{/if}
      {#each ranked as player, index (`${tab}-${player.id}`)}
        {@const stats = safePlayerStats(player)}
        {@const game = isGameLeaderboardTab(tab) ? gameWLDOf(player, tab) : null}
        {@const showGameStats = Boolean(game) || tab === "totalWins"}
        {@const wins = game ? game.wins : stats.wins}
        {@const losses = game ? game.losses : stats.losses}
        {@const draws = game ? game.draws : stats.draws}
        {@const decisive = wins + losses}
        {@const rate = decisive === 0 ? 0 : Math.round((wins / decisive) * 100)}
        <article class="global-rank-card">
          <div class="global-rank-main">
            <span class="rank-index">#{index + 1}</span>
            <PlayerAvatar {player} size={24} />
            <PlayerBadge {player} compact />
            <span class={`online-dot ${player.connected ? "online" : "offline"}`}>{player.connected ? "在线" : "离线"}</span>
          </div>
          <div class="global-rank-stats">
            {#if tab === "onlineTime"}
              <span>累计在线 {formatOnlineDuration(totalOnlineMsOf(player))}</span>
              <span>{stats.wins}胜 {stats.losses}负 {stats.draws}平</span>
            {:else if tab === "contribution"}
              <span>共建通过 {contributionApprovedCountOf(player)} 条</span>
            {:else if showGameStats}
              <span>{wins}胜 {losses}负 {draws}平</span>
              <span>总局 {wins + losses + draws}</span>
              <span>胜率 {rate}%</span>
            {:else}
              <span>{stats.rankedPoints} 分</span>
              <span>历史最高 {stats.highestScore}</span>
              <span>历史最低 {stats.lowestScore}</span>
              <span>{stats.wins}胜 {stats.losses}负 {stats.draws}平</span>
              <span>{stats.punishments} 惩罚</span>
              <span>胜率 {winRateText(player)}</span>
            {/if}
          </div>
          <LeaderboardExtra {player} {tab} {now} />
        </article>
      {/each}
      {#if ranked.length === 0}<p class="empty">暂无玩家上榜</p>{/if}
    </div>
  </section>
</div>
