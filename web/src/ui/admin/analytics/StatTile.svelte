<script lang="ts">
  // 顶部 KPI 卡：数值 + 环比 + 迷你趋势线。源：ui/AnalyticsPanel.tsx 的 StatTile。
  import Sparkline from "../../../lib/charts/Sparkline.svelte";
  import { formatDelta, formatNum } from "../../../lib/analyticsDashboard";

  let { label, value, delta, spark, format = formatNum }: {
    label: string; value: number; delta: number; spark: number[]; format?: (n: number) => string;
  } = $props();

  const good = $derived(delta > 0);
  const bad = $derived(delta < 0);
</script>

<div class="analytics-stat">
  <span class="analytics-stat-label">{label}</span>
  <strong class="analytics-stat-value">{format(value)}</strong>
  <span class={`analytics-stat-delta${good ? " good" : ""}${bad ? " bad" : ""}`}>{formatDelta(delta)}</span>
  <Sparkline data={spark} color={good ? "var(--chart-good)" : bad ? "var(--chart-critical)" : "var(--chart-1)"} />
</div>
