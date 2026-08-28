<script lang="ts">
  // 源：ui/AppViews.tsx:2235-2249
  import type { RoomSnapshot } from "../../shared/types";
  import { tictactoeDeltaText } from "../../lib/gameDisplay";
  let { room }: { room: RoomSnapshot } = $props();
  const state = $derived(room.tictactoe);
  const bothReady = $derived(room.ready.A && room.ready.B);
</script>

{#if !state}
  <p class="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>
{:else}
  <div class="tictactoe-score-mini">
    <span>❌ {state.xSeat === "A" ? "A" : "B"}</span>
    <span>⭕ {state.xSeat === "A" ? "B" : "A"}</span>
    {#if room.settings.enableRanked && state.rankedDelta}
      <span class="tictactoe-live-rank">X {tictactoeDeltaText(state, "X")} / O {tictactoeDeltaText(state, "O")}</span>
    {/if}
    <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${state.xSeat === state.turn ? "X" : "O"}`}</strong>
  </div>
{/if}
