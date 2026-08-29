<script lang="ts">
  // 「共建审核」分区：待处理总览 + 随机任务/系列任务两个审核队列。
  // 源：ui/AdminContributionReview.tsx:82-148。
  import { untrack } from "svelte";
  import type { ContributionStatus } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { reviewQueueCounts, type JumpTarget, type PendingOverviewResponse } from "../../lib/contributionAdmin";
  import { adminStore } from "../../lib/stores/adminStore.svelte";
  import { sessionStore } from "../../lib/stores/sessionStore.svelte";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import ContributionPendingSection from "./ContributionPendingSection.svelte";
  import ContributionReviewKindPanel from "./ContributionReviewKindPanel.svelte";

  const config = $derived(sessionStore.config!);

  let section = $state<"pending" | "task" | "series">("pending");
  let jumpTarget = $state<({ kind: "task" | "series" } & JumpTarget) | null>(null);
  // 总览数据提到这一层统一拉取：三个顶部选项卡各自的摘要数字（待处理条数、随机任务/系列任务
  // 使用中条数）不管当前停在哪个板块都要显示，不能只在「待处理」板块挂载时才有数据。
  let overview = $state<PendingOverviewResponse | null>(null);

  async function refreshOverview() {
    try {
      const res = await ask<PendingOverviewResponse>("admin:action", { action: "contributionPendingOverview", password: adminStore.password });
      overview = res;
      adminStore.contributionCounts = reviewQueueCounts(res);
    } catch (e) {
      uiStore.notify(e instanceof Error ? e.message : "加载失败");
    }
  }
  // 只在挂载时拉一次（原 React 版 deps 是 [overviewNonce]）；之后由子面板的 onChanged
  // 直接调用 refreshOverview。untrack 是必需的——refreshOverview 同步读 adminStore.password
  // 拼请求体，否则登录后在常驻的口令框里每敲一个字符都会重发一次总览请求。
  $effect(() => {
    untrack(() => refreshOverview());
  });

  function jumpTo(kind: "task" | "series", status: ContributionStatus) {
    section = kind;
    jumpTarget = { kind, status, nonce: Date.now() };
  }

  // 「待处理」= 随机任务 + 系列任务待审条数之和；「随机任务」「系列任务」=
  // 各自当前已批准（正在使用）的投稿数——系列任务的每一步不是独立投稿，天然不会被随机
  // 任务这边重复计入。
  const queueCounts = $derived(overview ? reviewQueueCounts(overview) : null);
  const pendingTotal = $derived(queueCounts ? queueCounts.pending : null);
  const taskApproved = $derived(overview?.counts?.task?.approved ?? null);
  const seriesApproved = $derived(overview?.counts?.series?.approved ?? null);
</script>

<div class="admin-contribution contribute-panel">
  <div class="contribute-tabs" role="tablist">
    <button type="button" class={`small${section === "pending" ? " active" : ""}`} onclick={() => (section = "pending")}>
      <span><strong>待处理</strong></span>
      <em>{pendingTotal == null ? "…" : `${pendingTotal} 条`}</em>
    </button>
    <button type="button" class={`small${section === "task" ? " active" : ""}`} onclick={() => (section = "task")}>
      <span><strong>随机任务</strong></span>
      <em>{taskApproved == null ? "…" : `${taskApproved} 条`}</em>
    </button>
    <button type="button" class={`small${section === "series" ? " active" : ""}`} onclick={() => (section = "series")}>
      <span><strong>系列任务</strong></span>
      <em>{seriesApproved == null ? "…" : `${seriesApproved} 组`}</em>
    </button>
  </div>
  {#if section === "pending"}
    <ContributionPendingSection data={overview} onJump={jumpTo} />
  {:else}
    <!-- {#key section}：与原版 React key={section} 语义一致——随机任务/系列任务两个队列
         切换时整个面板（含内部的当前选中投稿、排序、搜索词等状态）要完全重建，而不是
         复用同一实例只换 props，避免"随机任务详情"残留进系列任务视图。 -->
    {#key section}
      <ContributionReviewKindPanel
        kind={section}
        {config}
        password={adminStore.password}
        onError={(m) => uiStore.notify(m)}
        onChanged={refreshOverview}
        jumpTarget={jumpTarget && jumpTarget.kind === section ? jumpTarget : null}
      />
    {/key}
  {/if}
</div>
