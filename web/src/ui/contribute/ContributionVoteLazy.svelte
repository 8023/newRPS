<script lang="ts">
  // 正式任务一经分配即可评价；证明是否提交、通过或驳回都不改变投票资格，也不回滚票数。
  // eventId 是任务版本和双方资格的稳定边界，只有它变化时才需要重新获取评价卡片。
  // 源：ui/AppViews.tsx:5241-5254
  import type { VoteCard } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import ContributionVote from "./ContributionVote.svelte";

  let { eventId, onError }: { eventId: string; onError?: (message: string) => void } = $props();

  let card = $state<VoteCard | null>(null);

  $effect(() => {
    const currentEventId = eventId;
    let cancelled = false;
    ask<VoteCard>("contribution:votePreview", { eventId: currentEventId }).then((next) => {
      if (!cancelled) card = next;
    }).catch(() => {
      if (!cancelled) card = null;
    });
    return () => { cancelled = true; };
  });
</script>

{#if card && card.targetId}
  <ContributionVote {card} onError={onError ?? (() => undefined)} />
{/if}
