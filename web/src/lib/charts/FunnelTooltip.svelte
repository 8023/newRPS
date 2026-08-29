<script lang="ts">
  /**
   * 转化漏斗专用气泡：绝对设备 UV + 相对上一级 / 相对访问 的转化率。
   * 漏斗图的全部分析价值就在这两个转化率上，通用气泡只报一个绝对值等于没画漏斗。
   * 源：ui/AnalyticsPanel.tsx:1010-1049 的 FunnelTooltip。
   */
  import { Tooltip } from "layerchart/svg";
  import { formatNum } from "../analyticsDashboard";

  let { context, steps }: {
    context: any;
    /** 已按 visit→lobby→room→round→finish 排好序的各层，用来算相对上一级/相对访问。 */
    steps: { key: string; value: number }[];
  } = $props();

  function pct(num: number, den: number) {
    return den <= 0 ? "—" : `${((num / den) * 100).toFixed(1)}%`;
  }
</script>

<Tooltip.Root {context} variant="none" class="analytics-tooltip-shell">
  {#snippet children({ data }: { data: { key?: string; value?: number } })}
    {@const label = data?.key ?? ""}
    {@const value = Number(data?.value) || 0}
    {@const idx = steps.findIndex((s) => s.key === label)}
    {@const top = steps[0]?.value || 0}
    {@const prev = idx > 0 ? (steps[idx - 1]?.value || 0) : 0}
    <div class="analytics-tooltip">
      {#if label !== ""}
        <div class="analytics-tooltip-label">{label}</div>
      {/if}
      <div class="analytics-tooltip-row">
        <span class="analytics-tooltip-swatch" style="background:var(--chart-2)"></span>
        <span>设备 UV</span>
        <strong>{formatNum(value)}</strong>
      </div>
      {#if idx > 0}
        <div class="analytics-tooltip-row analytics-tooltip-meta">
          <span>相对上一级</span>
          <strong>{pct(value, prev)}</strong>
        </div>
      {/if}
      <div class="analytics-tooltip-row analytics-tooltip-meta">
        <span>相对访问</span>
        <strong>{idx === 0 ? "100%" : pct(value, top)}</strong>
      </div>
    </div>
  {/snippet}
</Tooltip.Root>
