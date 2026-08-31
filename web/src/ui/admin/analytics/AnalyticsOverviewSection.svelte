<script lang="ts">
  // 顶部 KPI + 用户/会话/页面浏览/新老用户/新老设备趋势 + 留存热力图。
  // 源：ui/AnalyticsPanel.tsx 493-636（AnalyticsPanel 主体前半段）。
  import type { AnalyticsRangeView } from "../../../shared/types";
  import { formatDurationMs, trendRows } from "../../../lib/analyticsDashboard";
  import LineTrendChart from "../../../lib/charts/LineTrendChart.svelte";
  import StatTile from "./StatTile.svelte";
  import ChartCard from "./ChartCard.svelte";
  import RetentionHeatmap from "./RetentionHeatmap.svelte";

  let { data }: { data: AnalyticsRangeView } = $props();
  const kpi = $derived(data.kpi);
  const trends = $derived(trendRows(data));
</script>

<div class="analytics-stats-row">
  <StatTile label="日活 (均)" value={kpi?.dauValue || 0} delta={kpi?.deltaDau || 0} spark={kpi?.sparkDau || []} />
  <StatTile label="会话数" value={kpi?.sessions || 0} delta={kpi?.deltaSessions || 0} spark={kpi?.sparkSessions || []} />
  <StatTile label="页面浏览" value={kpi?.pageviews || 0} delta={kpi?.deltaPageviews || 0} spark={kpi?.sparkPageviews || []} />
  <StatTile label="新访客" value={kpi?.newVisitors || 0} delta={kpi?.deltaNewVisitors || 0} spark={kpi?.sparkNewVisitors || []} />
  <StatTile label="平均会话" value={kpi?.avgSessionMs || 0} delta={kpi?.deltaAvgSessionMs || 0} spark={kpi?.sparkAvgSessionMs || []} format={formatDurationMs} />
  <StatTile label="峰值在线" value={kpi?.peakOnline || 0} delta={kpi?.deltaPeakOnline || 0} spark={kpi?.sparkPeakOnline || []} />
</div>

<div class="analytics-row-2">
  <ChartCard title="用户与会话">
    <LineTrendChart data={trends} x="day" series={[
      { key: "dau", label: "日活", color: "var(--chart-1)" },
      { key: "sessions", label: "会话", color: "var(--chart-2)" },
      { key: "loggedDau", label: "登录用户", color: "var(--chart-3)" }
    ]} />
  </ChartCard>

  <ChartCard title="页面浏览量">
    <LineTrendChart data={trends} x="day" series={[{ key: "pageviews", label: "页面浏览", color: "var(--chart-1)" }]} />
  </ChartCard>
</div>

<div class="analytics-row-2">
  <ChartCard title="新老用户">
    <LineTrendChart data={trends} x="day" showPercent series={[
      { key: "newUsers", label: "新用户", color: "var(--chart-1)" },
      { key: "oldLogin", label: "老用户登录", color: "var(--chart-3)" }
    ]} />
  </ChartCard>

  <ChartCard title="新老设备">
    <LineTrendChart data={trends} x="day" showPercent series={[
      { key: "newVisitors", label: "新设备", color: "var(--chart-1)" },
      { key: "returning", label: "老设备", color: "var(--chart-3)" }
    ]} />
  </ChartCard>
</div>

<ChartCard title="用户留存热力图 (D0–D14)">
  <RetentionHeatmap matrix={data.retention?.matrix || []} cohorts={data.retention?.cohorts || []} offsets={data.retention?.offsets || []} />
</ChartCard>
