<script lang="ts">
  // 源：ui/AppViews.tsx:1479-1482
  let { enabled }: { enabled?: boolean } = $props();
</script>

{#if enabled}
  <span class="rank-multiplier-badge extreme-ranked-badge">⚡ 极限</span>
{/if}
