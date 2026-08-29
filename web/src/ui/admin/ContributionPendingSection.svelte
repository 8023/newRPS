<script lang="ts">
  // 源：ui/AdminContributionReview.tsx:157-178
  import type { ContributionStatus } from "../../shared/types";
  import type { PendingOverviewResponse } from "../../lib/contributionAdmin";
  import ContributionOverviewRow from "./ContributionOverviewRow.svelte";

  let { data, onJump }: {
    data: PendingOverviewResponse | null;
    onJump: (kind: "task" | "series", status: ContributionStatus) => void;
  } = $props();
</script>

<div class="stack">
  <div class="admin-review-section">
    <h3>总览</h3>
    {#if data}
      <p class="hint">
        已发布随机任务 <strong>{data.poolStats?.randomTasks ?? 0}</strong> 条
        （含系列任务里同时进随机池的步骤） · 已发布系列任务 <strong>{data.poolStats?.series ?? 0}</strong> 组
      </p>
      <ContributionOverviewRow kind="task" label="随机任务" counts={data.counts?.task} {onJump} />
      <ContributionOverviewRow kind="series" label="系列任务" counts={data.counts?.series} {onJump} />
    {:else}
      <p class="hint">加载中…</p>
    {/if}
  </div>
</div>
