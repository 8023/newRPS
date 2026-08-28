<script lang="ts">
  // 源：ui/AppViews.tsx:2915-2926
  import type { RoomSnapshot } from "../../shared/types";
  let { proof, isMine, requireConfirm }: { proof: RoomSnapshot["proofs"][number]; isMine: boolean; requireConfirm: boolean } = $props();
</script>

{#if proof.status === "rejected"}
  <p class="task-card danger"><b>{isMine ? "对方要求你重做" : "已要求对方重做"}</b>{proof.redoTaskText || "请重新完成任务。"}</p>
{:else if proof.status === "approved" || proof.confirmedBy}
  {#if proof.rejectReason === "双方互相放过，下一局正常开始。"}
    <p class="task-card success"><b>双方互相放过</b>下一局正常开始。</p>
  {:else}
    <p class="task-card success"><b>{proof.rejectReason === "对方选择放过你" ? "对方选择放过你" : isMine ? "对方已确认完成" : "你已确认完成"}</b>{requireConfirm ? "等待系统进入下一局。" : "准备进入下一局。"}</p>
  {/if}
{:else}
  <p class="task-card"><b>证明已提交</b>{requireConfirm ? (isMine ? "等待对方验证。" : "请验证对方证明。") : "准备进入下一局。"}</p>
{/if}
