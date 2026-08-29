<script lang="ts">
  /**
   * 「对局时长」专用图：左轴堆叠柱是各游戏当天单局均值分钟数，右轴折线是全站单房均值
   * 分钟数——两者口径、量级都不同，与 DualAxisLineChart 同样用两个重叠 Chart 各自独立
   * 缩放，只是底层这次换成堆叠柱而不是折线。源：ui/AnalyticsPanel.tsx 的 RoundDurationChart。
   */
  import { Chart, Bars, Spline } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS, formatDayTick } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";
  import ChartLegend from "./ChartLegend.svelte";

  let { data, x, barSeries, line, height = 264 }: {
    data: Record<string, string | number>[];
    x: string;
    barSeries: { key: string; label: string }[];
    line: { key: string; label: string; color: string };
    height?: number;
  } = $props();

  const resolvedBars = $derived(barSeries.map((s, i) => ({ ...s, color: CHART_COLORS[i % CHART_COLORS.length] })));
  const minutes = (n: number) => `${n.toFixed(1)} 分钟`;
</script>

<div class="layerchart-card layerchart-dual-axis layerchart-composed layerchart-with-legend" style={`height:${height}px`}>
  {#if data.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <div class="layerchart-plot"><Chart
      {data}
      x={x}
      xScale={scaleBand().padding(0.35)}
      y={barSeries.map((s) => s.key)}
      yNice
      series={resolvedBars.map((s) => ({ key: s.key, value: s.key, label: s.label, color: s.color }))}
      seriesLayout="stack"
      padding={{ left: 8, bottom: 24, top: 8, right: 32 }}
      axis
      grid={{ x: false }}
      tooltipContext={{ mode: "band" }}
      props={{
        xAxis: { format: (v: unknown) => formatDayTick(String(v)), tickSpacing: 60, tickLabelProps: { class: "layerchart-tick" } },
        yAxis: { tickLabelProps: { class: "layerchart-tick" } }
      }}
    >
      {#snippet marks({ context })}
        {#each context.series.visibleSeries as s (s.key)}
          <Bars seriesKey={s.key} radius={4} rounded={(d: any) => (context.series.isStackTop(s.key, d) ? "edge" : "none")} />
        {/each}
      {/snippet}
      <!-- 柱（各游戏单局均值）与折线（全站单房均值）单位相同但口径不同、相加无意义，
           故不传 showTotal/showPercent——与原 React 版 AnalyticsTooltip 的调用一致。 -->
      {#snippet tooltip({ context })}
        <AnalyticsTooltip {context} xKey={x} series={[...resolvedBars, line]} valueFormat={minutes} />
      {/snippet}
    </Chart>
    <div class="layerchart-dual-axis-overlay">
      <Chart
        {data}
        x={x}
        xScale={scaleBand().padding(0.35)}
        y={line.key}
        yNice
        series={[{ key: line.key, value: line.key, label: line.label, color: line.color }]}
        padding={{ left: 8, bottom: 24, top: 8, right: 32 }}
        axis="y"
        grid={false}
        tooltipContext={false}
        pointerEvents={false}
        props={{ yAxis: { placement: "right", tickLabelProps: { class: "layerchart-tick" } } }}
      >
        {#snippet marks({ context })}
          <Spline seriesKey={line.key} stroke={line.color} strokeWidth={2.25} />
        {/snippet}
      </Chart>
    </div></div>
    <ChartLegend items={[...resolvedBars.map((s) => ({ key: s.key, label: s.label, color: s.color })), { key: line.key, label: line.label, color: line.color }]} />
  {/if}
</div>
