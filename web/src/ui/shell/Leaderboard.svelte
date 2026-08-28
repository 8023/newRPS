<script lang="ts">
  // 源：ui/AppViews.tsx:3524-3546。大厅侧栏小榜单（区别于点击「排行榜」打开的全站弹窗）。
  import Crown from "@lucide/svelte/icons/crown";
  import type { PublicPlayer } from "../../shared/types";
  import { safePlayerStats, winRateText } from "../../lib/playerDisplay";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import CollapseToggle from "./CollapseToggle.svelte";
  import PlayerAvatar from "./PlayerAvatar.svelte";
  import PlayerBadge from "./PlayerBadge.svelte";

  let { title, players }: { title: string; players: PublicPlayer[] } = $props();
  const collapse = useMobileCollapse("leaderboard");
</script>

<div class={`panel leaderboard-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <div class="panel-header-row">
    <h2><Crown size={18} /> {title}</h2>
    <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label={title} />
  </div>
  <div class={`leaderboard-list mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    {#each players as player, index (player.id)}
      {@const stats = safePlayerStats(player)}
      <p class="rank-row rich">
        <span>{index + 1}. <PlayerAvatar {player} size={22} /> <PlayerBadge {player} compact /></span>
        <small>{stats.wins}胜 {stats.losses}负 {stats.draws}平 · {stats.punishments}惩罚</small>
        <b>{winRateText(player)} · {stats.rankedPoints}分</b>
      </p>
    {/each}
    {#if players.length === 0}<p class="empty">暂无在线玩家</p>{/if}
  </div>
</div>
