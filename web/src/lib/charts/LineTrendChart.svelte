<script lang="ts">
  /**
   * 单 Y 轴多折线趋势图：数据分析面板里最常见的图表形态（用户与会话/页面浏览/新老用户/
   * 新老设备/聊天人数……），基于 LayerChart 的 Chart + Spline 组合——Chart 声明式接收
   * axis/grid/legend/tooltipContext 就会自动渲染坐标轴、图例、默认 tooltip，不需要像
   * recharts 那样手工拼 <XAxis>/<YAxis>/<Tooltip>；写法参照 LayerChart 官方 charts/LineChart
   * 的核心组合（Chart+Spline，见 node_modules/layerchart 源码），未直接依赖其未对外导出的
   * 高阶封装（LineChart.base.svelte 引用了包内部 $lib 别名，无法从已发布的 npm 包外部导入）。
   * 源：ui/AnalyticsPanel.tsx 内多处 <ResponsiveContainer><LineChart>...</LineChart></ResponsiveContainer>。
   */
  import { Chart, Spline } from "layerchart/svg";
  import { scalePoint } from "d3-scale";
  import { CHART_COLORS, formatDayTick } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";

  let { data, x, series, height = 240, valueFormat, showPercent = false }: {
    data: Record<string, string | number>[];
    x: string;
    series: { key: string; label: string; color?: string }[];
    height?: number;
    valueFormat?: (n: number) => string;
    showPercent?: boolean;
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
      xScale={scalePoint().padding(0.5)}
      y={series.map((s) => s.key)}
      yNice
      series={resolved.map((s) => ({ key: s.key, value: s.key, label: s.label, color: s.color }))}
      padding={{ left: 8, bottom: 24, top: 8, right: 8 }}
      axis
      grid={{ x: false }}
      legend={{ placement: "top" }}
      tooltipContext={{ mode: "band" }}
      props={{
        xAxis: {
          format: (v: unknown) => formatDayTick(String(v)),
          // band/point 轴的 tickSpacing 默认是 null（分类轴全显），30 天数据会把 30 个日期
          // 标签叠在一起糊成一片；给个像素间距让它按宽度自动抽稀，等价于 recharts 的默认行为。
          tickSpacing: 60,
          tickLabelProps: { class: "layerchart-tick" }
        },
        yAxis: { tickLabelProps: { class: "layerchart-tick" } }
      }}
    >
      {#snippet marks({ context })}
        {#each context.series.visibleSeries as s (s.key)}
          <Spline seriesKey={s.key} stroke={s.color} strokeWidth={2} />
        {/each}
      {/snippet}
      {#snippet tooltip({ context })}
        <AnalyticsTooltip {context} xKey={x} series={resolved} {showPercent} {valueFormat} />
      {/snippet}
    </Chart>
  {/if}
</div>
