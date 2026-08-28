<script lang="ts">
  // 源：ui/AppViews.tsx:1440-1443
  let { multiplier }: { multiplier?: number } = $props();
</script>

{#if multiplier && multiplier > 1}
  <span class="rank-multiplier-badge">x{multiplier}</span>
{/if}
