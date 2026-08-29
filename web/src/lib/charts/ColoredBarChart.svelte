<script lang="ts">
  // 会话时长分布 / 证明耗时分布：单系列竖向柱状图，但每根柱子按顺序取不同的低饱和度色，
  // 不是同一个颜色——与 HBarChart（横向、单色）区分开。源：ui/AnalyticsPanel.tsx 里两处
  // 用 <Cell fill={SEQ_COLORS[i % SEQ_COLORS.length]} /> 逐格上色的 BarChart。
  import { Chart, Bars } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { SEQ_COLORS } from "../analyticsDashboard";

  let { rows, height = 220 }: { rows: { key: string; value: number }[]; height?: number } = $props();
</script>

<div class="layerchart-card" style={`height:${height}px`}>
  {#if rows.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <Chart
      data={rows}
      x="key"
      y="value"
      c="key"
      cRange={SEQ_COLORS}
      xScale={scaleBand().padding(0.3)}
      yNice
      padding={{ left: 8, bottom: 20, top: 8, right: 8 }}
      axis
      grid={{ x: false }}
      tooltipContext={{ mode: "band" }}
      props={{ xAxis: { tickLabelProps: { class: "layerchart-tick" } }, yAxis: { tickLabelProps: { class: "layerchart-tick" } } }}
    >
      {#snippet marks()}
        <!-- 不传 fill：Bars 内部会用 Chart 的 c/cRange 色阶按每行的 key 自动逐格上色
             （等价于 recharts 的 <Cell fill=... /> 逐格覆盖，见 Bars.base.svelte 的
             `fill ?? c.series?.color ?? ctx.cGet(d)` 兜底链）。 -->
        <Bars radius={4} rounded="top" />
      {/snippet}
    </Chart>
  {/if}
</div>
