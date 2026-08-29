<script lang="ts">
  // 图表卡片外壳：标题 + 图表内容。详细数值统一由图表悬停气泡展示。
  // 源：ui/AnalyticsPanel.tsx 的 ChartCard。
  import type { Snippet } from "svelte";

  let { title, children }: { title: string; children: Snippet } = $props();
</script>

<div class="analytics-card">
  <div class="analytics-card-head">
    <h3>{title}</h3>
  </div>
  {@render children()}
</div>
