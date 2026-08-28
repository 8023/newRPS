<script lang="ts">
  // 白给自救板：宣言提交 + 点赞点踩投票。源：ui/AppViews.tsx:4151-4263。
  // config/players/me 原为 props；players 保留为 prop（Lobby 传 lobby.players，
  // 语义上是"当前在线名单"这类上下文数据，不适合提升为全局 store）。
  import type { GiveawayVoteQuota, PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { socket } from "../../ws";
  import { formatDuration } from "../../lib/format";
  import { formatGiveawayValue } from "../../lib/playerDisplay";
  import { describeGiveawayVoteQuota } from "../../lib/giveaway";
  import { useMobileCollapse } from "../../lib/useMobileCollapse.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import CollapseToggle from "../shell/CollapseToggle.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";

  const config = $derived(sessionStore.config!);
  const me = $derived(sessionStore.me!.player);
  const players = $derived(sessionStore.lobby!.players);

  let text = $state("");
  let now = $state(Date.now());
  let bondRevision = $state(0);
  // 投票额度按 actor→target 独立计算（见服务端 giveawayVoteRulesFor），不再是 me 身上的全局字段，
  // 走 giveaway:voteQuotas 单独查询、只对自己可见（不进大厅广播）。
  let quotas = $state<Record<string, GiveawayVoteQuota>>({});
  const collapse = useMobileCollapse("giveaway");

  const activeBoards = $derived(
    players
      .filter((player) => player.giveawayBoardText && player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > now)
      .sort((a, b) => (b.giveawayBoardSubmittedAt || 0) - (a.giveawayBoardSubmittedAt || 0))
  );
  const myActiveBoard = $derived(activeBoards.find((player) => player.id === me.id));
  const canSubmit = $derived(Boolean(me.giveawayEnabled && (me.giveawayValue || 0) > 0 && !myActiveBoard));
  const voteTargetIds = $derived(activeBoards.filter((player) => player.id !== me.id).map((player) => player.id).sort());
  const voteTargetKey = $derived(voteTargetIds.join(","));
  const voteRulesKey = $derived([
    config.giveaway.likeVoteLimitPerHour, config.giveaway.likeVoteValue, config.giveaway.dislikeVoteLimitPerHour, config.giveaway.dislikeVoteValue,
    config.giveaway.petLikeVoteLimitPerHour, config.giveaway.petLikeVoteValue, config.giveaway.petDislikeVoteLimitPerHour, config.giveaway.petDislikeVoteValue,
    config.giveaway.masterLikeVoteLimitPerHour, config.giveaway.masterLikeVoteValue, config.giveaway.masterDislikeVoteLimitPerHour, config.giveaway.masterDislikeVoteValue
  ].join("|"));

  $effect(() => {
    const hasExpiry = players.some((player) => player.giveawayBoardExpiresAt && player.giveawayBoardExpiresAt > Date.now());
    if (!hasExpiry) return;
    const timer = window.setInterval(() => { now = Date.now(); }, 1000);
    return () => window.clearInterval(timer);
  });

  $effect(() => {
    const onPetBondUpdate = () => { bondRevision += 1; };
    const onConnect = () => { bondRevision += 1; };
    socket.on("petbond:update", onPetBondUpdate);
    socket.on("connect", onConnect);
    return () => {
      socket.off("petbond:update", onPetBondUpdate);
      socket.off("connect", onConnect);
    };
  });

  $effect(() => {
    void voteTargetKey;
    void voteRulesKey;
    void bondRevision;
    if (!voteTargetIds.length) return;
    let cancelled = false;
    ask<{ quotas: Record<string, GiveawayVoteQuota> }>("giveaway:voteQuotas", { targetIds: voteTargetIds })
      .then((result) => { if (!cancelled) quotas = { ...quotas, ...result.quotas }; })
      .catch(() => { /* 额度展示是锦上添花，静默失败即可，不打断投票操作本身 */ });
    return () => { cancelled = true; };
  });

  async function submitBoard() {
    const cleanText = text.trim();
    if (!cleanText) return;
    try {
      const result = await ask<{ player: PublicPlayer }>("giveaway:submitBoard", { text: cleanText });
      if (result.player.id === me.id) text = "";
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "上板失败");
    }
  }

  async function vote(targetId: string, voteType: "like" | "dislike") {
    try {
      const result = await ask<{ ok: boolean; quota: GiveawayVoteQuota }>("giveaway:vote", { targetId, vote: voteType });
      if (result.quota) quotas = { ...quotas, [targetId]: result.quota };
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "操作失败");
    }
  }
</script>

<div class={`panel giveaway-panel ${collapse.collapsed ? "collapsed" : ""}`}>
  <div class="panel-title compact-title">
    <h2>🫴 {config.giveaway.panelTitle}</h2>
    <span class="panel-title-actions">
      <span>{activeBoards.length} 条</span>
      <CollapseToggle collapsed={collapse.collapsed} onToggle={collapse.toggle} label={config.giveaway.panelTitle} />
    </span>
  </div>
  <div class={`mobile-collapsible-body ${collapse.collapsed ? "collapsed" : ""}`}>
    <div class="giveaway-hints">
      <p class="hint">{config.giveaway.panelDescription}</p>
      {#if myActiveBoard}<p class="hint">你已经上板，过期后才能重新提交。当前剩余 {formatDuration((myActiveBoard.giveawayBoardExpiresAt || now) - now)}。</p>{/if}
      {#if !canSubmit && me.giveawayEnabled}<p class="hint">你的白给值已经是 {formatGiveawayValue(me.giveawayValue || 0)}%，归零后自动关闭白给模式。</p>{/if}
    </div>
    {#if canSubmit}
      <div class="giveaway-submit">
        <textarea value={text} maxlength="300" oninput={(event) => (text = event.currentTarget.value)} placeholder={config.giveaway.submitPlaceholder}></textarea>
        <button class="primary small" onclick={submitBoard}>上板 12 小时</button>
      </div>
    {/if}
    <div class="giveaway-board-list">
      {#each activeBoards as player (player.id)}
        {@const isSelf = player.id === me.id}
        {@const expiresText = player.giveawayBoardExpiresAt ? formatDuration(player.giveawayBoardExpiresAt - now) : ""}
        {@const rawQuota = isSelf ? undefined : quotas[player.id]}
        {@const quota = describeGiveawayVoteQuota(rawQuota, now)}
        <article class="giveaway-card">
          <div class="giveaway-card-head"><PlayerBadge {player} /></div>
          <p class="giveaway-board-text">{player.giveawayBoardText}</p>
          <div class="giveaway-card-meta">
            <span>👍 {player.giveawayBoardLikes || 0} · 👎 {player.giveawayBoardDislikes || 0} · 剩余 {expiresText}</span>
          </div>
          {#if quota}<div class="giveaway-card-meta">{quota.text}</div>{/if}
          <div class="giveaway-actions">
            <button disabled={isSelf || (quota ? quota.likesLeft <= 0 : false)} onclick={() => vote(player.id, "like")}>👍 -{formatGiveawayValue(rawQuota?.likeValue ?? config.giveaway.likeVoteValue)}%</button>
            <button disabled={isSelf || (quota ? quota.dislikesLeft <= 0 : false)} onclick={() => vote(player.id, "dislike")}>👎 +{formatGiveawayValue(rawQuota?.dislikeValue ?? config.giveaway.dislikeVoteValue)}%</button>
          </div>
        </article>
      {/each}
      {#if activeBoards.length === 0}<p class="empty">{config.giveaway.emptyText}</p>{/if}
    </div>
  </div>
</div>
