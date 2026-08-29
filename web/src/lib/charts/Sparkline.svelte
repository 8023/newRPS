<script lang="ts">
  // 统计卡片下方的迷你趋势线：无坐标轴/图例/tooltip，纯装饰性。源：ui/AnalyticsPanel.tsx 的 Sparkline。
  import { Chart, Spline } from "layerchart/svg";
  import { scalePoint } from "d3-scale";

  let { data, color = "var(--chart-1)" }: { data: number[]; color?: string } = $props();
  const rows = $derived(data.map((v, i) => ({ i, v })));
</script>

<div class="analytics-spark">
  {#if data.length}
    <Chart
      data={rows}
      x="i"
      y="v"
      xScale={scalePoint()}
      yNice
      padding={0}
      axis={false}
      grid={false}
      tooltipContext={false}
      pointerEvents={false}
    >
      {#snippet marks()}
        <Spline stroke={color} strokeWidth={2} />
      {/snippet}
    </Chart>
  {/if}
</div>
