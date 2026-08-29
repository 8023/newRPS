<script lang="ts">
  /**
   * 堆叠/分组柱状图：惩罚任务发布构成、热门惩罚标签选中/拒绝对照、对局数/开房数按游戏堆叠
   * 都是这一种形态。写法参照 LayerChart 官方 charts/BarChart 的核心组合（Chart+Bars），
   * `rounded` 用 `context.series.isStackTop(seriesKey, d)` 逐行判断——只有真正露在柱子顶端
   * 的那一段带圆角，与原 React 版手写的 StackedSegment 是同一视觉语言，这个判断本身就是
   * LayerChart 内置能力，不需要再手搓。源：ui/AnalyticsPanel.tsx 多处 <BarChart stackId="a">。
   */
  import { Chart, Bars } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS, formatDayTick } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";

  let { data, x, series, height = 220, layout = "stack", angledLabels = false, showPercent = false, showTotal = true }: {
    data: Record<string, string | number>[];
    x: string;
    series: { key: string; label: string; color?: string }[];
    height?: number;
    layout?: "stack" | "group";
    /** 类目较多时横轴文字倾斜显示（热门惩罚标签）。 */
    angledLabels?: boolean;
    showPercent?: boolean;
    /** 「惩罚任务」那张图原版刻意不显示合计（发布量=三段之和，重复信息），故可关。 */
    showTotal?: boolean;
  } = $props();

  const resolved = $derived(series.map((s, i) => ({ ...s, color: s.color ?? CHART_COLORS[i % CHART_COLORS.length] })));
</script>

<div class="layerchart-card" style={`height:${height}px`}>
  {#if data.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <Chart
      {data}
      x={x}
      xScale={scaleBand().padding(0.35)}
      y={series.map((s) => s.key)}
      yNice
      series={resolved.map((s) => ({ key: s.key, value: s.key, label: s.label, color: s.color }))}
      seriesLayout={layout}
      padding={{ left: 8, bottom: angledLabels ? 48 : 24, top: 8, right: 8 }}
      axis
      grid={{ x: false }}
      legend={{ placement: "top" }}
      tooltipContext={{ mode: "band" }}
      props={{
        xAxis: {
          format: (v: unknown) => formatDayTick(String(v)),
          // 倾斜标签的图（热门惩罚标签，最多 10 条）要全显；日期轴则按像素间距抽稀，
          // 否则 30 天会渲染 30 个标签叠成一片（band 轴的 tickSpacing 默认是 null）。
          tickSpacing: angledLabels ? null : 60,
          tickLabelProps: angledLabels ? { class: "layerchart-tick", textAnchor: "end", transform: "rotate(-25)" } : { class: "layerchart-tick" }
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
    </Chart>
  {/if}
</div>
