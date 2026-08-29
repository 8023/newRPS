<script lang="ts">
  /**
   * 数据分析图表的悬停气泡：复刻原 React 版 AnalyticsPanel.tsx 的 AnalyticsTooltip
   * （含「总计」行、每行占比、可自定义数值格式），并沿用它留在 styles.css 里的
   * .analytics-tooltip* 一套类名，因此外观与迁移前逐像素一致。
   *
   * 为什么不直接用 LayerChart 的 DefaultTooltip：
   *   1. 它的合计行标签硬编码英文 "total"（见 charts/DefaultTooltip.svelte），
   *      在全中文后台里会突兀，而该标签只能靠 props.tooltip.item 覆盖——那会把所有
   *      系列行的标签一起改掉，没法只改合计行。
   *   2. 它的表头对日期用宿主 locale 渲染成 8/25/2026 这类美式格式；原版气泡里
   *      固定展示完整的 yyyy-mm-dd（横轴刻度才缩写成 mm/dd）。
   *   3. 它没有「每行占当日/全类目合计的百分比」这一层，而原版多张图都开了 showPercent。
   * 源：ui/AnalyticsPanel.tsx:110-154。
   */
  import { Tooltip } from "layerchart/svg";
  import { formatNum } from "../analyticsDashboard";

  type SeriesDef = { key: string; label: string; color?: string };

  let {
    context,
    xKey,
    series,
    showTotal = false,
    showPercent = false,
    percentTotal,
    valueFormat
  }: {
    context: any;
    /** 行对象里充当表头的字段（日期图是 "day"，类目图是 "key"）。 */
    xKey: string;
    series: SeriesDef[];
    /** 堆叠图悬停时附带当行各系列之和。 */
    showTotal?: boolean;
    /** 每行附带占比：分母默认取当前悬停行各系列之和，也可用 percentTotal 指定外部分母
        （如单系列柱状图，需要占「全部类目合计」而非「当前这一根柱子」的比例）。 */
    showPercent?: boolean;
    percentTotal?: number;
    /** 均值类图表（如「对局时长」的分钟数，常见 <1 的小数）需要保留小数位。 */
    valueFormat?: (n: number) => string;
  } = $props();

  const fmt = $derived(valueFormat ?? formatNum);

  function rowsFor(data: Record<string, unknown> | null | undefined) {
    if (!data) return [];
    return series.map((s) => ({ ...s, value: Number(data[s.key]) || 0 }));
  }
</script>

<Tooltip.Root {context} variant="none" class="analytics-tooltip-shell">
  {#snippet children({ data }: { data: Record<string, unknown> })}
    {@const rows = rowsFor(data)}
    {@const localTotal = rows.reduce((sum, r) => sum + r.value, 0)}
    {@const denom = percentTotal != null ? percentTotal : localTotal}
    {@const label = data?.[xKey]}
    <div class="analytics-tooltip">
      {#if label != null && label !== ""}
        <div class="analytics-tooltip-label">{label}</div>
      {/if}
      {#each rows as row (row.key)}
        {@const pct = showPercent && denom > 0 ? `${((row.value / denom) * 100).toFixed(1)}%` : null}
        <div class="analytics-tooltip-row">
          <span class="analytics-tooltip-swatch" style={`background:${row.color ?? "var(--chart-1)"}`}></span>
          <span>{row.label}</span>
          <strong>
            {fmt(row.value)}
            {#if pct != null}<span class="analytics-tooltip-pct"> ({pct})</span>{/if}
          </strong>
        </div>
      {/each}
      {#if showTotal}
        <div class="analytics-tooltip-row analytics-tooltip-total">
          <span class="analytics-tooltip-swatch analytics-tooltip-swatch-total"></span>
          <span>总计</span>
          <strong>{fmt(localTotal)}</strong>
        </div>
      {/if}
    </div>
  {/snippet}
</Tooltip.Root>
