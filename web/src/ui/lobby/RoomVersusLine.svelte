<script lang="ts">
  // 源：ui/AppViews.tsx:1496-1533（RoomVersusSeat 内联合并，仅这一处调用）
  import type { LobbySnapshot } from "../../shared/types";
  import PlayerBadge from "../shell/PlayerBadge.svelte";

  let { room }: { room: LobbySnapshot["rooms"][number] } = $props();
</script>

{#if room.gameId === "liarsdice"}
  <!-- 大话骰是 N 人动态名单，没有 A/B 对阵；用骰子行取代 VS 行。 -->
  <div class="room-versus-line liarsdice-line" title={`大话骰 · ${room.players} 人参战`}>
    <span aria-hidden="true">🎲</span>
    <b>{room.players > 0 ? `${room.players} 人参战` : "等待玩家加入"}</b>
  </div>
{:else if room.gameId === "coinflip"}
  <!-- 猜硬币只用参战席 A（没有 B），用单人行取代 VS 行。 -->
  {@const flipper = room.versus.A?.player}
  <div class="room-versus-line coinflip-line" title={flipper ? `${flipper.name} 独自挑战` : "等待玩家坐下"}>
    <span aria-hidden="true">🪙</span>
    <b>{flipper ? `${flipper.name} 独自挑战` : "等待玩家坐下"}</b>
  </div>
{:else}
  {@const left = room.versus.A}
  {@const right = room.versus.B}
  {@const leftName = left?.player?.name || "等待玩家"}
  {@const rightName = right?.player?.name || "等待玩家"}
  <div class="room-versus-line" title={`${leftName} VS ${rightName}`}>
    {#if left}<PlayerBadge player={left.player} compact />{:else}<span class="empty">等待玩家</span>{/if}
    <b>VS</b>
    {#if right}<PlayerBadge player={right.player} compact />{:else}<span class="empty">等待玩家</span>{/if}
  </div>
{/if}
