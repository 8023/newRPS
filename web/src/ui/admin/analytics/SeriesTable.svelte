<script lang="ts">
  // 源：ui/AnalyticsPanel.tsx 的 SeriesTable。
  import type { AnalyticsNamedSeries } from "../../../shared/types";
  import { labelFor, formatNum } from "../../../lib/analyticsDashboard";

  let { days, series, labels }: { days: string[]; series: AnalyticsNamedSeries[]; labels?: Record<string, string> } = $props();
</script>

{#if series.length === 0}
  <p class="empty">暂无数据</p>
{:else}
  <div class="analytics-table-scroll">
    <table class="analytics-data-table">
      <thead>
        <tr><th>日期</th>{#each series as s (s.key)}<th>{labelFor(labels, s.key)}</th>{/each}</tr>
      </thead>
      <tbody>
        {#each days as d, i (d)}
          <tr><td>{d}</td>{#each series as s (s.key)}<td>{formatNum(s.values[i] || 0)}</td>{/each}</tr>
        {/each}
      </tbody>
    </table>
  </div>
{/if}
