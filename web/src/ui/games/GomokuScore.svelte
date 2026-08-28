<script lang="ts">
  // 源：ui/GomokuPanel.tsx:38-53
  import type { RoomSnapshot, SeatKey } from "../../shared/types";
  import { gomokuDeltaText } from "../../lib/gameDisplay";
  let { room }: { room: RoomSnapshot } = $props();
  const state = $derived(room.gomoku);
  const bothReady = $derived(room.ready.A && room.ready.B);
  const whiteSeat = $derived<SeatKey | null>(state ? (state.blackSeat === "A" ? "B" : "A") : null);
</script>

{#if !state}
  <p class="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>
{:else}
  <div class="gomoku-score-mini">
    <span>⚫ {state.blackSeat}</span>
    <span>⚪ {whiteSeat}</span>
    {#if room.settings.enableRanked && state.rankedDelta}
      <span class="gomoku-live-rank">黑 {gomokuDeltaText(state, state.blackSeat)} / 白 {gomokuDeltaText(state, whiteSeat!)}</span>
    {/if}
    <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${state.blackSeat === state.turn ? "黑" : "白"}`}</strong>
  </div>
{/if}
