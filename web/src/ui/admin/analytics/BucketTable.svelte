<script lang="ts">
  // 源：ui/AnalyticsPanel.tsx 的 BucketTable。
  import type { AnalyticsBucket } from "../../../shared/types";
  import { formatNum } from "../../../lib/analyticsDashboard";

  let { rows, valueLabel = "数量" }: { rows: AnalyticsBucket[]; valueLabel?: string } = $props();
</script>

{#if rows.length === 0}
  <p class="empty">暂无数据</p>
{:else}
  <table class="analytics-data-table">
    <thead><tr><th>项</th><th>{valueLabel}</th></tr></thead>
    <tbody>
      {#each rows as r (r.key)}<tr><td>{r.key || "(空)"}</td><td>{formatNum(r.value)}</td></tr>{/each}
    </tbody>
  </table>
{/if}
