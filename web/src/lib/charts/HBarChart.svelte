<script lang="ts">
  /**
   * 横向条形图：浏览器/操作系统/来源/省份/ISP/热门系列任务/转化漏斗共用的形态——数值轴在
   * 下方横轴，类目（浏览器名、来源域名……）在纵轴。源：ui/AnalyticsPanel.tsx 的 HBarCard
   * 非圆环分支与转化漏斗图（recharts <BarChart layout="vertical">，"vertical" 指类目轴纵向，
   * 与本组件命名的"横向条形"是同一件事，只是叫法相反）。
   */
  import type { Snippet } from "svelte";
  import { Chart, Bars } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";

  let { rows, height = 188, color = CHART_COLORS[0], valueLabel = "数量", showPercent = false, maxLabelChars = 10, tooltip }: {
    rows: { key: string; value: number }[];
    height?: number;
    color?: string;
    valueLabel?: string;
    /** 占比分母是全部类目合计（不只是画出来的 Top N），与原版一致。 */
    showPercent?: boolean;
    /** SVG 纵轴不支持 CSS 省略号；截断只影响轴标签，悬停气泡仍展示完整名称。 */
    maxLabelChars?: number;
    /** 需要完全自定义气泡内容时传入（转化漏斗要展示相对上一级/相对访问的转化率）。 */
    tooltip?: Snippet<[{ context: any }]>;
  } = $props();

  const fullTotal = $derived(rows.reduce((sum, r) => sum + r.value, 0));
  const axisLabel = (value: unknown) => {
    const chars = Array.from(String(value));
    return chars.length > maxLabelChars ? `${chars.slice(0, maxLabelChars).join("")}…` : chars.join("");
  };
</script>

<div class="layerchart-card layerchart-hbar" style={`height:${height}px`}>
  {#if rows.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <Chart
      data={rows}
      x="value"
      xDomain={[0, null]}
      y="key"
      yScale={scaleBand().padding(0.35)}
      xNice
      padding={{ left: 100, bottom: 20, top: 4, right: 8 }}
      axis
      grid={{ y: false }}
      tooltipContext={{ mode: "band" }}
      props={{
        yAxis: { format: axisLabel, tickLabelProps: { class: "layerchart-tick" } },
        xAxis: { tickLabelProps: { class: "layerchart-tick" } }
      }}
    >
      {#snippet marks()}
        <Bars radius={4} rounded="right" fill={color} />
      {/snippet}
      {#snippet tooltip(snippetProps)}
        {#if tooltip}
          {@render tooltip(snippetProps)}
        {:else}
          <AnalyticsTooltip
            context={snippetProps.context}
            xKey="key"
            series={[{ key: "value", label: valueLabel, color }]}
            {showPercent}
            percentTotal={fullTotal}
          />
        {/if}
      {/snippet}
    </Chart>
  {/if}
</div>
