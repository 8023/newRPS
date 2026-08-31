<script lang="ts">
  /**
   * 堆叠/分组柱状图：惩罚任务发布构成、对局数/开房数按游戏堆叠都是这一种形态（横轴恒为
   * 日期；类目型的对照图走横向的 HBarChart）。写法参照 LayerChart 官方 charts/BarChart 的核心组合（Chart+Bars），
   * `rounded` 用 `context.series.isStackTop(seriesKey, d)` 逐行判断——只有真正露在柱子顶端
   * 的那一段带圆角，与原 React 版手写的 StackedSegment 是同一视觉语言，这个判断本身就是
   * LayerChart 内置能力，不需要再手搓。源：ui/AnalyticsPanel.tsx 多处 <BarChart stackId="a">。
   */
  import { Chart, Bars } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS, PLOT_HEIGHT_TREND, VALUE_AXIS_GUTTER, formatDayTick } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";
  import ChartLegend from "./ChartLegend.svelte";

  let { data, x, series, height = PLOT_HEIGHT_TREND, layout = "stack", showPercent = false, showTotal = true }: {
    data: Record<string, string | number>[];
    x: string;
    series: { key: string; label: string; color?: string }[];
    height?: number;
    layout?: "stack" | "group";
    showPercent?: boolean;
    /** 「惩罚任务」那张图原版刻意不显示合计（发布量=三段之和，重复信息），故可关。 */
    showTotal?: boolean;
  } = $props();

  const resolved = $derived(series.map((s, i) => ({ ...s, color: s.color ?? CHART_COLORS[i % CHART_COLORS.length] })));
</script>

<div class="layerchart-card layerchart-with-legend" style={`min-height:${height}px`}>
  {#if data.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <div class="layerchart-plot"><Chart
      {data}
      x={x}
      xScale={scaleBand().padding(0.35)}
      y={series.map((s) => s.key)}
      yNice
      series={resolved.map((s) => ({ key: s.key, value: s.key, label: s.label, color: s.color }))}
      seriesLayout={layout}
      padding={{ left: VALUE_AXIS_GUTTER, bottom: 24, top: 8, right: 8 }}
      axis
      grid={{ x: false }}
      tooltipContext={{ mode: "band" }}
      props={{
        xAxis: {
          format: (v: unknown) => formatDayTick(String(v)),
          // band 轴的 tickSpacing 默认是 null（分类轴全显），30 天数据会把 30 个日期标签
          // 叠在一起糊成一片；给个像素间距让它按宽度自动抽稀。
          tickSpacing: 60,
          tickLabelProps: { class: "layerchart-tick" }
        },
        yAxis: { tickLabelProps: { class: "layerchart-tick" } }
      }}
    >
      {#snippet marks({ context })}
        {#each context.series.visibleSeries as s (s.key)}
          <Bars
            seriesKey={s.key}
            radius={4}
            rounded={(d: any) => (context.series.isStackTop(s.key, d) ? "edge" : "none")}
          />
        {/each}
      {/snippet}
      {#snippet tooltip({ context })}
        <AnalyticsTooltip {context} xKey={x} series={resolved} {showTotal} {showPercent} />
      {/snippet}
    </Chart></div>
    <ChartLegend items={resolved.map((s) => ({ key: s.key, label: s.label, color: s.color }))} />
  {/if}
</div>
