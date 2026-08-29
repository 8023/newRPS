<script lang="ts">
  // 用户留存热力图：纯 CSS Grid，不经过任何图表库——与原版一致，这类"逐格上色的表格"
  // 用图表库反而更绕。源：ui/AnalyticsPanel.tsx 的 RetentionHeatmap。
  import { styleString } from "../../../lib/style";

  let { matrix, cohorts, offsets }: { matrix: number[][]; cohorts: string[]; offsets: number[] } = $props();

  // 只展示最近 14 个 cohort × offset 0-14，避免 30×30 过密
  const maxC = 14;
  const maxO = 15;
  const start = $derived(Math.max(0, cohorts.length - maxC));
  const cols = $derived(offsets.slice(0, maxO));
  const rows = $derived(cohorts.slice(start));
  const data = $derived(matrix.slice(start).map((r) => r.slice(0, maxO)));
</script>

{#if matrix.length === 0}
  <p class="empty">暂无留存数据（需有成功注册的用户）</p>
{:else}
  <div class="analytics-heatmap-wrap">
    <div class="analytics-heatmap" style={styleString({ gridTemplateColumns: `72px repeat(${cols.length}, minmax(18px, 1fr))` })}>
      <div class="analytics-heatmap-corner"></div>
      {#each cols as o (o)}<div class="analytics-heatmap-colhead">D{o}</div>{/each}
      {#each rows as c, ri (c)}
        <div class="analytics-heatmap-rowhead">{c.slice(5)}</div>
        {#each cols as o, ci (o)}
          {@const v = data[ri]?.[ci] ?? 0}
          {@const level = v <= 0 ? 0 : Math.min(6, Math.ceil(v / 16.7))}
          <div class={`analytics-heatmap-cell seq-${level}`} title={`${c} · D${o}: ${v.toFixed(1)}%`}>{v > 0 ? Math.round(v) : ""}</div>
        {/each}
      {/each}
    </div>
  </div>
{/if}
