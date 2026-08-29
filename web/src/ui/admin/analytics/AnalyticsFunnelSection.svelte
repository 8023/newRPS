<script lang="ts">
  // 转化漏斗 + 明细（页面浏览/来源/省份/最近会话）。最近会话是懒加载的独立请求，只在切到
  // 这个子页签时才拉取——与主快照的 60s 轮询分开，避免明细表格长期占带宽。
  // 源：ui/AnalyticsPanel.tsx 452-469、934-996（detailTab 状态与相关 effect）。
  import type { AnalyticsRangeView, AnalyticsSessionBrief } from "../../../shared/types";
  import { ask } from "../../../lib/rpc";
  import { formatDurationMs, funnelOrdered, labelFor, relabelBuckets, DEVICE_LABELS, VIEW_LABELS } from "../../../lib/analyticsDashboard";
  import HBarChart from "../../../lib/charts/HBarChart.svelte";
  import FunnelTooltip from "../../../lib/charts/FunnelTooltip.svelte";
  import ChartCard from "./ChartCard.svelte";
  import BucketTable from "./BucketTable.svelte";

  let { data, days }: { data: AnalyticsRangeView; days: number } = $props();

  const funnelRows = $derived(funnelOrdered(data.funnel || []));
  const viewRows = $derived(relabelBuckets((data.viewPv || []).slice(0, 20), VIEW_LABELS));
  const refRows = $derived((data.referrers || []).slice(0, 20));
  const provRows = $derived((data.provinces || []).slice(0, 20));

  let detailTab = $state<"views" | "refs" | "prov" | "sess">("views");
  let sessions = $state<AnalyticsSessionBrief[]>([]);
  let sessionsGen = 0;

  async function loadSessions(forDays: number) {
    const gen = ++sessionsGen;
    try {
      const detail = await ask<{ recentSessions?: AnalyticsSessionBrief[] }>("admin:analyticsDetail", { days: forDays });
      if (gen !== sessionsGen) return;
      sessions = detail.recentSessions || [];
    } catch {
      // 主面板数据仍可用，明细失败时保留旧值
    }
  }

  // 「最近会话」是独立于主快照的懒加载明细：切到该页签、或在该页签上改时间范围时重新拉取
  // （对应原 React 版 deps [detailTab, days]），并按 60s 续拉，复现原版主轮询里
  // `if (detailTabRef.current === "sess") 一并刷新明细` 的实时性。离开页签即停表。
  $effect(() => {
    if (detailTab !== "sess") return;
    const forDays = days;
    void loadSessions(forDays);
    const timer = setInterval(() => void loadSessions(forDays), 60_000);
    return () => clearInterval(timer);
  });
</script>

<ChartCard title="转化漏斗">
  {#snippet table()}<BucketTable rows={funnelRows} />{/snippet}
  <HBarChart rows={funnelRows} height={200} color="var(--chart-2)">
    {#snippet tooltip({ context })}
      <FunnelTooltip {context} steps={funnelRows} />
    {/snippet}
  </HBarChart>
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
