<script module lang="ts">
  import type { VoteCard } from "../../shared/types";

  export function formatVoteStats(card: Pick<VoteCard, "hasVotes" | "displayRatio" | "voteCount">) {
    if (!card.hasVotes) return `暂无评价 · ${card.voteCount} 人评价`;
    return `${card.voteCount} 人评价 · 点赞 ${card.displayRatio ?? 0}%`;
  }
</script>

<script lang="ts">
  // 源：ui/ContributionVote.tsx
  import { untrack } from "svelte";
  import { ask } from "../../lib/rpc";

  let { card, onError, onVoted }: {
    card: VoteCard;
    onError: (message: string) => void;
    onVoted?: (next: VoteCard) => void;
  } = $props();

  let busy = $state(false);
  let current = $state(untrack(() => card));
  // card prop 只在首次挂载时用于初始化；父组件若因任务切换或重新取数传入新 card，仍需把
  // prop 同步进本地状态。vote() 自己的乐观更新不受影响，因为父组件不会同时回传旧 card。
  $effect(() => {
    current = card;
  });

  async function vote(value: 1 | -1) {
    busy = true;
    try {
      const next = await ask<VoteCard>("contribution:vote", { eventId: current.eventId, vote: value });
      current = next;
      onVoted?.(next);
    } catch (e) {
      onError(e instanceof Error ? e.message : "评价失败");
    } finally {
      busy = false;
    }
  }

  // 点赞率/贡献者在投票前是"剧透"：只有玩家自己点了赞或踩之后才展示，避免看到评价影响判断。
  const voted = $derived(current.myVote !== 0);
</script>

<div class="contribution-vote">
  {#if voted}
    <p class="contribution-vote-result">
      <span class="vote-result-contributor">由 {current.contributorDisplayName} 贡献</span>
      <span class="vote-result-stats">{formatVoteStats(current)}</span>
    </p>
  {:else if current.canVote}
    <div class="vote-actions">
      <span>评价任务：</span>
      <button type="button" disabled={busy} onclick={() => vote(1)}>赞 👍</button>
      <button type="button" disabled={busy} onclick={() => vote(-1)}>踩 👎</button>
    </div>
  {:else if current.cannotVoteReason}
    <p class="hint">{current.cannotVoteReason}</p>
  {/if}
</div>
