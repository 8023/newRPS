<script lang="ts">
  /**
   * 「对局时长」专用图：左轴堆叠柱是各游戏当天单局均值分钟数，右轴折线是全站单房均值
   * 分钟数——两者口径、量级都不同，与 DualAxisLineChart 同样用两个重叠 Chart 各自独立
   * 缩放，只是底层这次换成堆叠柱而不是折线。源：ui/AnalyticsPanel.tsx 的 RoundDurationChart。
   */
  import { Chart, Bars, Spline } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS, formatDayTick } from "../analyticsDashboard";

  let { data, x, barSeries, line, height = 260 }: {
    data: Record<string, string | number>[];
    x: string;
    barSeries: { key: string; label: string }[];
    line: { key: string; label: string; color: string };
    height?: number;
  } = $props();
</script>

<div class="layerchart-card layerchart-dual-axis" style={`height:${height}px`}>
  {#if data.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <Chart
      {data}
      x={x}
      xScale={scaleBand().padding(0.35)}
      y={barSeries.map((s) => s.key)}
      yNice
      series={barSeries.map((s, i) => ({ key: s.key, value: s.key, label: s.label, color: CHART_COLORS[i % CHART_COLORS.length] }))}
      seriesLayout="stack"
      padding={{ left: 8, bottom: 24, top: 8, right: 32 }}
      axis
      grid={{ x: false }}
      legend={{ placement: "top" }}
      tooltipContext={{ mode: "band" }}
      props={{
        xAxis: { format: (v: unknown) => formatDayTick(String(v)), tickLabelProps: { class: "layerchart-tick" } },
        yAxis: { tickLabelProps: { class: "layerchart-tick" } },
        tooltip: { item: { format: (v: unknown) => `${Number(v).toFixed(1)} 分钟` } }
      }}
    >
      {#snippet marks({ context })}
        {#each context.series.visibleSeries as s (s.key)}
          <Bars seriesKey={s.key} radius={4} rounded={(d: any) => (context.series.isStackTop(s.key, d) ? "edge" : "none")} />
        {/each}
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
          <Spline seriesKey={line.key} stroke={line.color} strokeWidth={2} />
        {/snippet}
      </Chart>
    </div>
    <div class="layerchart-dual-axis-legend">
      <span><i style={`background:${line.color}`}></i>{line.label}</span>
    </div>
  {/if}
</div>
