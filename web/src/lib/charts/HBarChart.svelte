<script lang="ts">
  /**
   * 横向条形图：浏览器/操作系统/来源/省份/ISP/热门惩罚标签/热门系列任务/转化漏斗共用的
   * 形态——数值轴在下方横轴，类目（浏览器名、来源域名……）在纵轴。源：ui/AnalyticsPanel.tsx
   * 的 HBarCard 非圆环分支与转化漏斗图（recharts <BarChart layout="vertical">，"vertical"
   * 指类目轴纵向，与本组件命名的"横向条形"是同一件事，只是叫法相反）。
   *
   * 传 series 时变成多系列横向堆叠柱（热门惩罚标签的选中/拒绝对照），柱子下方跟统一图例；
   * 不传就是单色单系列。两种形态共用同一套留白与截断规则，几张图并排时左沿必然对齐。
   *
   * 类目轴留白是全站固定值（AXIS_GUTTER），不跟着各图自己最长的标签走：让它自适应的话，
   * "来源 Top10"会被 www.google.com、"ISP Top10"会被一长串机构名各自撑到不同宽度，同一屏
   * 几张图的左边缘就参差不齐。宁可让个别超长名字打省略号（完整名称见悬停），也要左沿一致。
   */
  import type { Snippet } from "svelte";
  import { Chart, Bars, Text } from "layerchart/svg";
  import { scaleBand } from "d3-scale";
  import { CHART_COLORS, PLOT_HEIGHT_DISTRIBUTION } from "../analyticsDashboard";
  import AnalyticsTooltip from "./AnalyticsTooltip.svelte";
  import ChartLegend from "./ChartLegend.svelte";

  let {
    rows,
    series,
    height = PLOT_HEIGHT_DISTRIBUTION,
    color = CHART_COLORS[0],
    valueLabel = "数量",
    showLegend = false,
    showPercent = false,
    showTotal = true,
    tooltip
  }: {
    /** 单系列时是 { key, value }；传了 series 则每行是 key + 各系列字段。 */
    rows: Record<string, string | number>[];
    /** 多系列（堆叠）时的系列定义；不传即单系列单色。 */
    series?: { key: string; label: string; color?: string }[];
    height?: number;
    color?: string;
    valueLabel?: string;
    /** 单系列也画一条图例。用于让它和同一行里带图例的图表横轴对齐（热门系列任务 vs
        热门惩罚标签）；没有这种搭配的图不必开，绘图区可以把整张卡吃满。 */
    showLegend?: boolean;
    /** 占比分母：单系列取全部类目合计，多系列取当前这一行各系列之和。 */
    showPercent?: boolean;
    /** 多系列气泡里的「合计」行。 */
    showTotal?: boolean;
    /** 需要完全自定义气泡内容时传入（转化漏斗要展示相对上一级/相对访问的转化率）。 */
    tooltip?: Snippet<[{ context: any }]>;
  } = $props();

  /** .layerchart-tick 的字号与字重；下面量文字宽度要用同一套。 */
  const TICK_FONT = "450 10.5px Inter, \"PingFang SC\", \"Microsoft YaHei\", system-ui, sans-serif";
  /**
   * 类目轴留白（含标签与柱子之间的 10px 呼吸位）。取值对齐「浏览器」「省份 Top10」这类
   * 短标签图原本的宽度——那是留白看着最紧凑的一档；再窄连 Windows / 11.11.1.11 都要打
   * 省略号就过头了。更长的名字一律省略号 + 悬停看全名。
   */
  const AXIS_GUTTER = 62;
  /** 窄屏（后台单列布局）时按卡片宽度收一收，别让留白吃掉整根柱子。 */
  const GUTTER_MAX_RATIO = 0.42;

  /**
   * 用离屏 canvas 按真实字体量宽度：轴标签是 SVG 文本，拿不到 CSS 省略号，截断位置只能自己
   * 算。以前按字符类别估算，中英文混排时会差出好几个像素（数字尤其偏），省略号早一个字晚
   * 一个字都不好看。canvas 环境不可用时（测试/SSR）退回粗略估算，只影响截断位置，不影响
   * 布局——留白本身是固定值。
   */
  let measureCtx: CanvasRenderingContext2D | null | undefined;
  function labelWidth(text: string): number {
    if (measureCtx === undefined) {
      measureCtx = typeof document === "undefined" ? null : document.createElement("canvas").getContext("2d");
      if (measureCtx) measureCtx.font = TICK_FONT;
    }
    if (measureCtx) return measureCtx.measureText(text).width;
    let width = 0;
    for (const ch of text) width += (ch.codePointAt(0) ?? 0) > 0x2e80 ? 10.5 : 5.7;
    return width;
  }

  const resolved = $derived(
    series?.map((s, i) => ({ ...s, color: s.color ?? CHART_COLORS[i % CHART_COLORS.length] }))
  );
  /** 图例项：多系列列出各系列，单系列只在 showLegend 时列出唯一那条。 */
  const legendItems = $derived(
    resolved?.map((s) => ({ key: s.key, label: s.label, color: s.color })) ??
      (showLegend ? [{ key: "value", label: valueLabel, color }] : null)
  );
  /** 单系列的占比分母是全部类目合计（不只是画出来的 Top N），与原版一致。 */
  const fullTotal = $derived(series ? 0 : rows.reduce((sum, r) => sum + (Number(r.value) || 0), 0));

  let boxWidth = $state(0);
  const gutter = $derived(Math.min(AXIS_GUTTER, Math.round((boxWidth || 320) * GUTTER_MAX_RATIO)));
  /** 留白里真正能写字的宽度（扣掉标签与柱子之间的呼吸位）。 */
  const budget = $derived(gutter - 10);

  const axisLabel = (value: unknown) => {
    const text = String(value);
    if (labelWidth(text) <= budget) return text;
    const room = budget - labelWidth("…");
    let out = "";
    let used = 0;
    for (const ch of text) {
      const w = labelWidth(ch);
      if (used + w > room) break;
      out += ch;
      used += w;
    }
    return `${out}…`;
  };
</script>

{#snippet categoryTickLabel({ props, index }: { props: Record<string, unknown>; index: number })}
  {@const full = String(rows[index]?.key ?? props.value ?? "")}
  <!-- 被省略号截断的标签额外挂一个原生 <title>：鼠标停在文字上就能看到完整名称。
       LayerChart 的悬停气泡只覆盖绘图区（TooltipContext 的 isOutsidePlotArea 会主动
       忽略 padding 区域），够不到左侧留白里的轴标签，所以这一层不能省。
       band 轴的刻度就是按数据顺序排的整个 domain，index 与 rows 下标一一对应。 -->
  {#if labelWidth(full) > budget}
    <g><title>{full}</title><Text {...props} /></g>
  {:else}
    <Text {...props} />
  {/if}
{/snippet}

<div
  class="layerchart-card layerchart-hbar {legendItems ? 'layerchart-with-legend' : ''}"
  style={`min-height:${height}px`}
  bind:clientWidth={boxWidth}
>
  {#if rows.length === 0}
    <p class="empty">暂无数据</p>
  {:else}
    <div class="layerchart-plot">
      <!-- xDomain 只在单系列时钉住 [0, null]（把数值轴锚在 0）。堆叠时不能给：那样数值轴会
           按"单个系列的最大值"定界，叠在后面的那一段（拒绝）会一路画到卡片外面去，盖住旁边
           那张图；不传则由 LayerChart 按堆叠合计推导，天然也是从 0 起。 -->
      <Chart
        data={rows}
        x={resolved ? resolved.map((s) => s.key) : "value"}
        xDomain={resolved ? undefined : [0, null]}
        y="key"
        yScale={scaleBand().padding(0.35)}
        xNice
        series={resolved?.map((s) => ({ key: s.key, value: s.key, label: s.label, color: s.color }))}
        seriesLayout={resolved ? "stack" : undefined}
        padding={{ left: gutter, bottom: 20, top: 4, right: 8 }}
        axis
        grid={{ y: false }}
        tooltipContext={{ mode: "band" }}
        props={{
          yAxis: { format: axisLabel, tickLabel: categoryTickLabel, tickLabelProps: { class: "layerchart-tick" } },
          xAxis: { tickLabelProps: { class: "layerchart-tick" } }
        }}
      >
        {#snippet marks({ context })}
          {#if resolved}
            {#each context.series.visibleSeries as s (s.key)}
              <!-- 横向堆叠：只有真正露在右端的那一段带圆角，与竖版 StackedBarChart 同一套判断。 -->
              <Bars
                seriesKey={s.key}
                radius={4}
                rounded={(d: any) => (context.series.isStackTop(s.key, d) ? "edge" : "none")}
              />
            {/each}
          {:else}
            <Bars radius={4} rounded="right" fill={color} />
          {/if}
        {/snippet}
        {#snippet tooltip(snippetProps)}
          {#if tooltip}
            {@render tooltip(snippetProps)}
          {:else if resolved}
            <AnalyticsTooltip context={snippetProps.context} xKey="key" series={resolved} {showTotal} {showPercent} />
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
    </div>
    {#if legendItems}
      <ChartLegend items={legendItems} />
    {/if}
  {/if}
</div>
