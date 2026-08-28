<script lang="ts">
  // 源：ui/AppViews.tsx:301-307
  import type { PublicPlayer } from "../../shared/types";

  let { player }: { player: PublicPlayer } = $props();
  const showNameWar = $derived(Boolean(player.nameWarEnabled && !player.nameWarPunished));
</script>

{#if player.extremeModeEnabled && showNameWar}
  <span class="mode-chip">⚡⚔️ 极限名争</span>
{:else if player.extremeModeEnabled}
  <span class="mode-chip">⚡ 极限</span>
{:else if showNameWar}
  <span class="mode-chip">⚔️ 名争</span>
{/if}
