<script lang="ts">
  // 源：ui/AppViews.tsx:3181-3189。锤子剪刀布对战比分区的小结算展示。
  import type { RoomSnapshot } from "../../shared/types";
  import { choiceText, historyResultText } from "../../lib/gameDisplay";

  let { room }: { room: RoomSnapshot } = $props();
  const latest = $derived(room.roundHistory[0]);
</script>

{#if !latest || (room.phase !== "result" && room.phase !== "punishment")}
  <p class="settlement-placeholder">等待结算</p>
{:else}
  <div class="settlement-pop">
    <span>{choiceText(latest.moveA)} <b>vs</b> {choiceText(latest.moveB)}</span>
    <strong>{latest.resultText || latest.resultLabel || historyResultText(latest.result)}</strong>
  </div>
{/if}
