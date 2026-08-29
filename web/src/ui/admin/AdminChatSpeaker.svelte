<script lang="ts">
  // 源：ui/AdminViews.tsx:1435-1474
  import type { PublicPlayer } from "../../shared/types";
  import { safePlayerStats, genderStyle, titleStyle, displayPlayerName } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import PlayerAvatar from "../shell/PlayerAvatar.svelte";
  import GiveawayChip from "../shell/GiveawayChip.svelte";
  import ModeChip from "../shell/ModeChip.svelte";

  let { player, fallbackName, isLobby, scopeLabel }: {
    player?: PublicPlayer;
    fallbackName: string;
    isLobby: boolean;
    scopeLabel: string;
  } = $props();

  const stats = $derived(player ? safePlayerStats(player) : undefined);
  const punished = $derived(Boolean(player?.nameWarPunished && player?.nameWarPenaltyName));
</script>

<PlayerAvatar {player} size={24} />
{#if player}
  <span class="gender-chip" style={styleString(genderStyle(player))} title={player.factionLabel}>{player.genderLabel || "未知"}</span>
  <span class="title-chip" style={styleString(titleStyle(stats?.titleColors))}>{stats?.title || "无称号"}</span>
  <strong class={punished ? "name-war-pill" : ""}>{displayPlayerName(player)}</strong>
  <GiveawayChip {player} />
  <ModeChip {player} />
  {#if player.nameWarEnabled && player.nameWarPunished && !player.extremeModeEnabled}
    <span class="mode-chip">⚔️ 名争</span>
  {/if}
  <span class={`admin-chat-presence ${player.connected ? "online" : "offline"}`}>{player.connected ? "在线" : "离线"}</span>
{:else}
  <strong>{fallbackName}</strong>
  <span class="admin-chat-presence offline">离线</span>
{/if}
<span class={`admin-chat-scope ${isLobby ? "lobby" : "room"}`}>{scopeLabel}</span>
