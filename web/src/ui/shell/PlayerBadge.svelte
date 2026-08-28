<script lang="ts">
  // 源：ui/AppViews.tsx:287-299
  import type { PublicPlayer } from "../../shared/types";
  import { safePlayerStats, displayPlayerName, genderStyle, titleStyle } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import ModeChip from "./ModeChip.svelte";
  import GiveawayChip from "./GiveawayChip.svelte";

  let { player, compact = false }: { player: PublicPlayer; compact?: boolean } = $props();
  const stats = $derived(safePlayerStats(player));
  const punished = $derived(Boolean(player.nameWarPunished && player.nameWarPenaltyName));
</script>

<span class={`player-badge ${compact ? "compact" : ""}`}>
  <span class="gender-chip" style={styleString(genderStyle(player))} title={player.factionLabel}>{player.genderLabel}</span>
  <span class="title-chip" style={styleString(titleStyle(stats.titleColors))}>{stats.title}</span>
  <strong class={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</strong>
  <ModeChip {player} />
  <GiveawayChip {player} />
</span>
