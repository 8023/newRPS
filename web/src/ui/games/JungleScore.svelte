<script lang="ts">
  // 源：ui/JunglePanel.tsx:130-152
  import type { RoomSnapshot } from "../../shared/types";
  import { jungleDeltaText, jungleSideBrief, jungleSideLabel } from "../../lib/gameDisplay";
  let { room }: { room: RoomSnapshot } = $props();
  const state = $derived(room.jungle);
  const bothReady = $derived(room.ready.A && room.ready.B);
</script>

{#if !state}
  <p class="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>
{:else}
  <div class="jungle-score-mini">
    <span class="jungle-score-side"><em>玩家 A</em><b>{jungleSideBrief("A")}</b></span>
    <span class="jungle-score-side"><em>玩家 B</em><b>{jungleSideBrief("B")}</b></span>
    {#if room.settings.enableRanked && state.rankedDelta}
      <span class="jungle-live-rank">白 {jungleDeltaText(state, "A")} / 黑 {jungleDeltaText(state, "B")}</span>
    {/if}
    <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${jungleSideLabel(state.turn)}`}</strong>
  </div>
{/if}
