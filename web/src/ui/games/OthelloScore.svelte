<script lang="ts">
  // 源：ui/AppViews.tsx:2219-2233
  import type { RoomSnapshot } from "../../shared/types";
  import { othelloDeltaText } from "../../lib/gameDisplay";
  let { room }: { room: RoomSnapshot } = $props();
  const state = $derived(room.othello);
  const bothReady = $derived(room.ready.A && room.ready.B);
</script>

{#if !state}
  <p class="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>
{:else}
  <div class="othello-score-mini">
    <span>⚫ {state.blackCount}</span>
    <span>⚪ {state.whiteCount}</span>
    {#if room.settings.enableRanked && state.rankedDelta}
      <span class="othello-live-rank">黑 {othelloDeltaText(state, "black")} / 白 {othelloDeltaText(state, "white")}</span>
    {/if}
    <strong>{state.ended ? room.resultText || "对局结束" : `轮到${state.blackSeat === state.turn ? "黑棋" : "白棋"}`}</strong>
  </div>
{/if}
