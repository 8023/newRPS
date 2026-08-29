<script lang="ts">
  // 图表卡片外壳：标题 + 「图表/表格」切换按钮（有 table snippet 时才显示）。
  // 源：ui/AnalyticsPanel.tsx 的 ChartCard。
  import type { Snippet } from "svelte";

  let { title, table, children }: { title: string; table?: Snippet; children: Snippet } = $props();
  let asTable = $state(false);
</script>

<div class="analytics-card">
  <div class="analytics-card-head">
    <h3>{title}</h3>
    {#if table}
      <button type="button" class="analytics-table-toggle" onclick={() => (asTable = !asTable)}>{asTable ? "图表" : "表格"}</button>
    {/if}
  </div>
  {#if asTable && table}
    {@render table()}
  {:else}
    {@render children()}
  {/if}
</div>
