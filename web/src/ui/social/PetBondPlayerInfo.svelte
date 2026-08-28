<script lang="ts">
  // 源：ui/AppViews.tsx:4365-4373
  import type { PublicPlayer } from "../../shared/types";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";

  let { player, size = 28, showStatus = false }: { player: PublicPlayer; size?: number; showStatus?: boolean } = $props();
</script>

<span class="pet-bond-player-info">
  <PlayerAvatar {player} {size} />
  <PlayerBadge {player} compact />
  {#if showStatus && !player.connected}<span class="online-dot offline">离线</span>{/if}
</span>
