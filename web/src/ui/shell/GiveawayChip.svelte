<script lang="ts">
  // 源：ui/AppViews.tsx:316-323
  import type { PublicPlayer } from "../../shared/types";
  import { shouldShowGiveawayValue, formatGiveawayValue } from "../../lib/playerDisplay";

  let { player }: { player: PublicPlayer } = $props();
</script>

{#if shouldShowGiveawayValue(player)}
  <span class="giveaway-chip">白给 {formatGiveawayValue(player.giveawayValue || 0)}%</span>
{/if}
