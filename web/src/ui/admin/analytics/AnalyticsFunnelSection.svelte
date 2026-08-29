<script lang="ts">
  // 转化漏斗 + 明细（页面浏览/来源/省份/最近会话）。最近会话是懒加载的独立请求，只在切到
  // 这个子页签时才拉取——与主快照的 60s 轮询分开，避免明细表格长期占带宽。
  // 源：ui/AnalyticsPanel.tsx 452-469、934-996（detailTab 状态与相关 effect）。
  import type { AnalyticsRangeView, AnalyticsSessionBrief } from "../../../shared/types";
  import { ask } from "../../../lib/rpc";
  import { formatDurationMs, funnelOrdered, labelFor, relabelBuckets, DEVICE_LABELS, VIEW_LABELS } from "../../../lib/analyticsDashboard";
  import HBarChart from "../../../lib/charts/HBarChart.svelte";
  import ChartCard from "./ChartCard.svelte";
  import BucketTable from "./BucketTable.svelte";

  let { data, days }: { data: AnalyticsRangeView; days: number } = $props();

  const funnelRows = $derived(funnelOrdered(data.funnel || []));
  const viewRows = $derived(relabelBuckets((data.viewPv || []).slice(0, 20), VIEW_LABELS));
  const refRows = $derived((data.referrers || []).slice(0, 20));
  const provRows = $derived((data.provinces || []).slice(0, 20));

  let detailTab = $state<"views" | "refs" | "prov" | "sess">("views");
  let sessions = $state<AnalyticsSessionBrief[]>([]);
  let sessionsLoaded = false;

  async function loadSessions() {
    try {
      const detail = await ask<{ recentSessions?: AnalyticsSessionBrief[] }>("admin:analyticsDetail", { days });
      sessionsLoaded = true;
      sessions = detail.recentSessions || [];
    } catch {
      // 主面板数据仍可用，明细失败时保留旧值
    }
  }

  // 首次切到「最近会话」时立即拉取；之后同一 tab 不重复拉（父级 60s 轮询若想刷新明细，
  // 由用户手动重新切换页签触发，与原版按 days 变化重拉的语义一致——days 变化时本组件
  // 随父级一起重建（父级用 days 做 key），无需额外处理。
  $effect(() => {
    if (detailTab === "sess" && !sessionsLoaded) void loadSessions();
  });
</script>

<ChartCard title="转化漏斗">
  {#snippet table()}<BucketTable rows={funnelRows} />{/snippet}
  <HBarChart rows={funnelRows} height={200} color="var(--chart-2)" />
</ChartCard>

<div class="analytics-card">
  <div class="analytics-card-head">
    <h3>明细</h3>
    <div class="analytics-tabs">
      {#each [["views", "页面浏览"], ["refs", "来源"], ["prov", "省份"], ["sess", "最近会话"]] as [id, label] (id)}
        <button type="button" class={detailTab === id ? "active" : ""} onclick={() => (detailTab = id as typeof detailTab)}>{label}</button>
      {/each}
    </div>
  </div>
  {#if detailTab === "views"}<BucketTable rows={viewRows} />{/if}
  {#if detailTab === "refs"}<BucketTable rows={refRows} />{/if}
  {#if detailTab === "prov"}<BucketTable rows={provRows} />{/if}
  {#if detailTab === "sess"}
    <table class="analytics-data-table">
      <thead><tr><th>访客</th><th>玩家</th><th>时长</th><th>设备</th><th>省份</th><th>浏览</th></tr></thead>
      <tbody>
        {#each sessions as s, i (i)}
          <tr>
            <td>{s.visitor}</td><td>{s.playerName || "—"}</td><td>{formatDurationMs(s.durationMs)}</td>
            <td>{labelFor(DEVICE_LABELS, s.device)}</td><td>{s.province || "—"}</td><td>{s.pageviews}</td>
          </tr>
        {/each}
        {#if sessions.length === 0}<tr><td colspan="6" class="empty">暂无会话（前端埋点上线后产生）</td></tr>{/if}
      </tbody>
    </table>
  {/if}
</div>
