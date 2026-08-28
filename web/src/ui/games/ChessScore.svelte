<script lang="ts">
  // 源：ui/ChessPanel.tsx:108-130
  import type { RoomSnapshot } from "../../shared/types";
  import { chessDeltaText, chessSideLabel } from "../../lib/gameDisplay";
  let { room }: { room: RoomSnapshot } = $props();
  const state = $derived(room.chess);
  const bothReady = $derived(room.ready.A && room.ready.B);
</script>

{#if !state}
  <p class="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>
{:else}
  <div class="chess-score-mini">
    <span class="chess-score-side"><em>玩家 A</em><b>{chessSideLabel(state, "A")}</b></span>
    <span class="chess-score-side"><em>玩家 B</em><b>{chessSideLabel(state, "B")}</b></span>
    {#if room.settings.enableRanked && state.rankedDelta}
      <span class="chess-live-rank">A {chessDeltaText(state, "A")} / B {chessDeltaText(state, "B")}</span>
    {/if}
    <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${chessSideLabel(state, state.turn)}`}</strong>
  </div>
{/if}
