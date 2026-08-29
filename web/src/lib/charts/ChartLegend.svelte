<script lang="ts">
  /**
   * 数据分析图表的统一静态图例。刻意不使用 LayerChart 内置的可点击 button 图例：
   * 后台统计只需要辨认系列，筛选会让不同图表的交互与视觉状态变得不一致。
   */
  let { items }: { items: { key: string; label: string; color: string; value?: string }[] } = $props();
</script>

<ul class="analytics-chart-legend" aria-label="图例">
  {#each items as item (item.key)}
    <li>
      <i aria-hidden="true" style={`background:${item.color}`}></i>
      <span>{item.label}</span>
      {#if item.value}<em>{item.value}</em>{/if}
    </li>
  {/each}
</ul>
