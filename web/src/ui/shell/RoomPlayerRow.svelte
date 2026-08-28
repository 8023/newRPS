<script lang="ts">
  // 源：ui/AppViews.tsx:2810-2829
  import type { PublicPlayer } from "../../shared/types";
  import { safePlayerStats, winRateText } from "../../lib/playerDisplay";
  import PlayerAvatar from "./PlayerAvatar.svelte";
  import PlayerBadge from "./PlayerBadge.svelte";
  import OfflineBadge from "./OfflineBadge.svelte";

  let { player, role }: { player: PublicPlayer; role: string } = $props();
  const stats = $derived(safePlayerStats(player));
</script>

<div class="room-player-row">
  <div class="room-player-main">
    <div class="room-player-identity">
      <PlayerAvatar {player} size={28} />
      <PlayerBadge {player} />
    </div>
    <div class="room-player-tags">
      <em>{role}</em>
      <OfflineBadge {player} />
    </div>
  </div>
  <small class="room-player-stats">
    全局：{stats.wins}胜 {stats.losses}负 {stats.draws}平 · {stats.punishments}惩罚 · 胜率 {winRateText(player)} · {stats.rankedPoints}分
  </small>
</div>
