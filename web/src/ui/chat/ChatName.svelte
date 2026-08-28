<script lang="ts">
  // 源：ui/AppViews.tsx:3166-3180
  import type { PublicPlayer } from "../../shared/types";
  import { safePlayerStats, genderStyle, titleStyle, displayPlayerName } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import ModeChip from "../shell/ModeChip.svelte";
  import GiveawayChip from "../shell/GiveawayChip.svelte";

  let { player }: { player: PublicPlayer } = $props();
  const punished = $derived(Boolean(player.nameWarPunished && player.nameWarPenaltyName));
  const stats = $derived(safePlayerStats(player));
</script>

<span class="chat-name">
  <span class="chat-gender" style={styleString(genderStyle(player))}>{player.genderLabel}</span>
  <span class="chat-title" style={styleString(titleStyle(stats.titleColors))}>{stats.title}</span>
  <b class={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</b>
  <ModeChip {player} />
  <GiveawayChip {player} />
  {#if !player.connected}<span class="chat-offline">离线</span>{/if}
</span>
