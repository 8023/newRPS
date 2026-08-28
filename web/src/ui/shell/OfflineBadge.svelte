<script lang="ts">
  // 源：ui/AppViews.tsx:2831-2837
  import type { PublicPlayer } from "../../shared/types";
  import { useNow } from "../../lib/useNow.svelte";

  let { player }: { player: PublicPlayer } = $props();
  const ticking = $derived(!player.connected && Boolean(player.disconnectExpiresAt));
  const now = useNow(() => 1000, () => ticking);
  const seconds = $derived(player.disconnectExpiresAt ? Math.max(0, Math.ceil((player.disconnectExpiresAt - now.value) / 1000)) : 0);
</script>

{#if !player.connected}
  <em class="offline-badge">离线 {seconds}s</em>
{/if}
