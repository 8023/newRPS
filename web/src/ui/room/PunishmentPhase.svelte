<script lang="ts">
  // 惩罚阶段卡片网格：证明提交/审核、任务发布——原 React 版这部分状态（proofText/
  // proofImage/redoInputs/taskInputs）直接堆在 Room 组件里，现在整段下沉到这个自
  // 包含的组件，Room.svelte 不再需要知道这些字段的存在。
  // 源：ui/AppViews.tsx:2007-2131（punish-box 区块）。
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { tokenKey } from "../../lib/constants";
  import { prepareProofImageForUpload } from "../../lib/proofImage";
  import {
    canAssignPunishmentTask, canReviewPunishmentProof, punishedPlayerName, punishedPlayerNames,
    roomPlayerById, taskTextOnly
  } from "../../lib/gameDisplay";
  import { displayPlayerName } from "../../lib/playerDisplay";
  import { styleString } from "../../lib/style";
  import { uiStore } from "../../lib/stores/uiStore.svelte";
  import PlayerBadge from "../shell/PlayerBadge.svelte";
  import ProofImage from "../shell/ProofImage.svelte";
  import PunishmentStatus from "../shell/PunishmentStatus.svelte";
  import ContributionVoteLazy from "../contribute/ContributionVoteLazy.svelte";
  import Upload from "@lucide/svelte/icons/upload";

  let { room, me, onOpenImage }: { room: RoomSnapshot; me: PublicPlayer; onOpenImage: (url: string) => void } = $props();

  const punishedNames = $derived(punishedPlayerNames(room));
  const punishedIds = $derived(room.punishedPlayerIds || []);

  let proofText = $state("");
  let proofImage = $state("");
  let proofSubmitting = $state(false);
  let imageUploading = $state(false);
  let redoInputs = $state<Record<string, string>>({});
  let taskInputs = $state<Record<string, string>>({});

  $effect(() => {
    if (room.settings.allowProofImage === false) proofImage = "";
  });

  async function uploadImage(file: File) {
    let localPreview = "";
    imageUploading = true;
    try {
      const uploadFile = await prepareProofImageForUpload(file);
      // 立刻本地预览（blob，不显示「加载中」）
      localPreview = URL.createObjectURL(uploadFile);
      proofImage = localPreview;
      const form = new FormData();
      form.append("token", localStorage.getItem(tokenKey) || "");
      form.append("image", uploadFile, uploadFile.name.endsWith(".webp") ? uploadFile.name : "proof.webp");
      const response = await fetch("/api/proof-image", { method: "POST", body: form });
      let data: { message?: string; imageUrl?: string } = {};
      try {
        data = await response.json();
      } catch {
        throw new Error(response.ok ? "服务器响应无效" : `上传失败（${response.status}）`);
      }
      if (!response.ok) throw new Error(data.message || "上传失败");
      if (!data.imageUrl) throw new Error("服务器未返回图片地址");
      // 提交必须用服务端 URL；展示切到远端（ProofImage 已处理缓存 onLoad）
      proofImage = data.imageUrl;
    } catch (error) {
      proofImage = "";
      uiStore.notify(error instanceof Error ? error.message : "图片上传失败");
    } finally {
      imageUploading = false;
      if (localPreview) {
        const url = localPreview;
        window.setTimeout(() => URL.revokeObjectURL(url), 3000);
      }
    }
  }

  async function submitProof() {
    const text = proofText.trim();
    if (!text) {
      uiStore.notify("请填写文字证明");
      return;
    }
    // 图片还在上传时 proofImage 是本地 blob 预览，服务端 safeUploadURL 会直接拒绝——
    // 弱网/HEIC 转码慢时很容易点提交就报「图片地址无效」，这里等上传完成再放行。
    if (imageUploading) {
      uiStore.notify("图片正在上传，请稍候再提交");
      return;
    }
    if (proofSubmitting) return;
    proofSubmitting = true;
    try {
      await ask("punishment:submit", { text, imageUrl: proofImage || "" });
      proofText = "";
      proofImage = "";
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "提交失败");
    } finally {
      proofSubmitting = false;
    }
  }

  async function reviewProof(playerId: string, action: "approve" | "forgive" | "reject") {
    try {
      await ask("punishment:review", { playerId, action, redoTaskText: redoInputs[playerId] });
      redoInputs = { ...redoInputs, [playerId]: "" };
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "审核失败");
    }
  }

  async function assignPunishmentTask(playerId: string) {
    try {
      await ask("punishment:assignTask", { playerId, taskText: taskInputs[playerId] });
      taskInputs = { ...taskInputs, [playerId]: "" };
    } catch (error) {
      uiStore.notify(error instanceof Error ? error.message : "发布任务失败");
    }
  }
</script>

<div class="punish-box">
  <div class="punish-head">
    <span>🎲 惩罚阶段</span>
    <strong>{punishedNames.length ? punishedNames.join("、") : "等待同步"}</strong>
  </div>
  <div class={`punishment-card-grid ${punishedIds.length === 1 ? "single" : "double"}`}>
    {#each punishedIds as playerId (playerId)}
      {@const punishedPlayer = roomPlayerById(room, playerId)}
      {@const proof = (room.proofs || []).find((item) => item.playerId === playerId)}
      {@const task = room.roundHistory[0]?.punishmentTasks?.find((item) => item.playerId === playerId)}
      {@const taskAssignerPlayer = task?.assignedBy ? roomPlayerById(room, task.assignedBy) : undefined}
      {@const isMine = playerId === me.id}
      {@const taskText = proof?.redoTaskText || task?.taskText || ""}
      {@const taskAssigned = Boolean(taskText.trim())}
      {@const canAssignTask = Boolean(room.settings.punishmentSource === "player" && task && canAssignPunishmentTask(room, me.id, playerId, task.assignedBy) && !taskAssigned)}
      {@const canSubmit = isMine && taskAssigned && (!proof || !proof.status || proof.status === "rejected")}
      {@const canReview = Boolean(canReviewPunishmentProof(room, me.id, playerId) && proof && proof.status !== "approved" && proof.status !== "rejected")}
      {@const taskCardStyle = task?.backgroundImage ? styleString({ "--task-bg": `url(${task.backgroundImage})`, "--task-bg-opacity": String(task.backgroundOpacity ?? 0.22) }) : undefined}
      {@const assignerName = taskAssignerPlayer ? displayPlayerName(taskAssignerPlayer) : task?.assignedByName}
      <div class="punishment-card">
        <div class="punishment-card-title">
          <h4>{#if punishedPlayer}<PlayerBadge player={punishedPlayer} compact />{:else}{punishedPlayerName(room, playerId)}{/if} {isMine ? "（你）" : ""}</h4>
          <em>{proof?.status === "approved" ? "已完成" : proof?.status === "pending" ? "待审核" : proof?.status === "rejected" ? "重做中" : taskAssigned ? "待提交" : "等任务"}</em>
        </div>
        {#if taskAssigned && task}
          <div class={`task-card designed-task-card ${task.backgroundImage ? "has-task-background" : ""}`} style={taskCardStyle}>
            <b>{assignerName ? `由 ${assignerName} 发布给${isMine ? "你" : "对方"}的任务` : (isMine ? "你的任务" : "对方任务")}</b>
            <p>{taskTextOnly(taskText, task.factionLabel)}</p>
            {#if task.eventId}<div class="task-card-vote-row"><ContributionVoteLazy eventId={task.eventId} onError={(m) => uiStore.notify(m)} /></div>{/if}
          </div>
        {/if}
        {#if !taskAssigned && room.settings.punishmentSource === "player"}
          <div class="assign-task-box">
            {#if isMine}<p class="hint">等待对方发布任务，发布后你就可以提交证明。</p>{/if}
            {#if canAssignTask}
              <p class="hint">请给对方发布一个本局临时惩罚任务。</p>
              <textarea value={taskInputs[playerId] || ""} oninput={(event) => (taskInputs = { ...taskInputs, [playerId]: event.currentTarget.value })} placeholder="例如：面向镜头比个耶并拍照证明"></textarea>
              <button class="primary" onclick={() => assignPunishmentTask(playerId)}>发布任务</button>
            {/if}
            {#if !isMine && !canAssignTask}<p class="hint">等待任务发布。</p>{/if}
          </div>
        {/if}
        {#if proof}<PunishmentStatus {proof} {isMine} requireConfirm={room.settings.requireOpponentConfirm} />{/if}
        {#if taskAssigned && task?.backgroundImage}
          <button type="button" class="history-proof-image-button punishment-task-image-button" title="点击查看完整任务图" onclick={() => onOpenImage(task.backgroundImage!)}>
            <img src={task.backgroundImage} alt="任务图片" loading="lazy" decoding="async" />
          </button>
        {/if}
        {#if proof?.text}
          <div class="proof">
            <b>已提交证明</b>
            <p>{proof.text}</p>
            {#if proof.imageUrl}
              <button type="button" class="history-proof-image-button" title="点击放大" onclick={() => onOpenImage(proof.imageUrl!)}>
                <ProofImage src={proof.imageUrl} alt="惩罚证明" />
              </button>
            {/if}
          </div>
        {/if}
        {#if canSubmit}
          <div class="proof-submit-card">
            <b>{proof?.status === "rejected" ? "重新提交证明" : "提交完成证明"}</b>
            <textarea value={proofText} oninput={(event) => (proofText = event.currentTarget.value)} placeholder={proof?.status === "rejected" ? "重新提交你的惩罚完成证明" : "写下你的惩罚完成证明"}></textarea>
            <p class="hint">文字证明必须填写；照片证明可以不交。</p>
            {#if room.settings.allowProofImage !== false}
              <label class="upload">
                <Upload size={16} /> 上传图片证明
                <!-- Chrome 的 accept="image/*" 通配符会用内置图片扩展名表过滤文件选择器，
                     该表不含 heic/heif，导致即使同时列了 .heic/.heif 也可能被隐藏；
                     这里去掉通配符、只列具体扩展名/MIME，纯粹影响系统选择器的可见文件，
                     不改变选完文件之后的校验/压缩/上传流程。 -->
                <input type="file" accept=".jpg,.jpeg,.png,.webp,.heic,.heif,.gif,.bmp,image/jpeg,image/png,image/webp,image/gif,image/bmp" onchange={(event) => event.currentTarget.files?.[0] && uploadImage(event.currentTarget.files[0]).catch((error) => uiStore.notify(error.message))} />
              </label>
              {#if proofImage}
                <button type="button" class="history-proof-image-button" title="点击放大" onclick={() => onOpenImage(proofImage)}>
                  <ProofImage className="proof-preview" src={proofImage} alt="惩罚证明" />
                </button>
              {/if}
            {:else}
              <p class="hint">本房间关闭了图片证明，只需要提交文字证明。</p>
            {/if}
            <button type="button" class="primary" disabled={proofSubmitting || imageUploading} onclick={() => void submitProof()}>
              {proofSubmitting ? "提交中…" : imageUploading ? "图片上传中…" : "提交证明"}
            </button>
          </div>
        {/if}
        {#if canReview}
          <div class="review-box">
            <div class="toolbar">
              <button class="primary" onclick={() => reviewProof(playerId, "approve")}>确认完成</button>
              <button onclick={() => reviewProof(playerId, "forgive")}>放过对方</button>
            </div>
            <p class="hint">放过对方后，对方下一局可能会受到一点命运安排。双方互相放过时会互相抵消。</p>
            <textarea value={redoInputs[playerId] || ""} oninput={(event) => (redoInputs = { ...redoInputs, [playerId]: event.currentTarget.value })} placeholder="不通过时，给对方发布一个新任务"></textarea>
            <button onclick={() => reviewProof(playerId, "reject")}>不通过，发布新任务</button>
          </div>
        {/if}
      </div>
    {/each}
  </div>
</div>
