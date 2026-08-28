<script lang="ts">
  // 源：ui/AppViews.tsx:2648-2773
  import type { RoomSnapshot } from "../../shared/types";
  import { normalizeRoundHistoryItem } from "../../lib/normalize";
  import { historyProofStatusLabel, historyResultText, historySeatLabel, taskTextOnly } from "../../lib/gameDisplay";
  import { styleString } from "../../lib/style";
  import { coinFaceLabel } from "../games/CoinFlipPanel.svelte";
  import ProofImage from "./ProofImage.svelte";

  let { item, onOpenImage }: { item: RoomSnapshot["roundHistory"][number]; onOpenImage: (imageUrl: string) => void } = $props();

  const safe = $derived(normalizeRoundHistoryItem(item));
  const proofByPlayer = $derived(new Map(safe.proofs.map((proof) => [proof.playerId, proof])));
  const taskPlayerIds = $derived(new Set(safe.punishmentTasks.map((task) => task.playerId)));
  const looseProofs = $derived(safe.proofs.filter((proof) => !taskPlayerIds.has(proof.playerId)));
</script>

<article class="history-card">
  <header class="history-card-head">
    <div>
      <b>第 {safe.round} 局</b>
      <small>{new Date(safe.at).toLocaleTimeString()}</small>
    </div>
    <div class="history-tags">
      {#if safe.gameId === "othello"}<em>⚫⚪ 黑白棋</em>{/if}
      {#if safe.gameId === "tictactoe"}<em>❌⭕ 井字棋</em>{/if}
      {#if safe.gameId === "gomoku"}<em>●○ 五子棋</em>{/if}
      {#if safe.gameId === "jungle"}<em>🦁 斗兽棋</em>{/if}
      {#if safe.gameId === "chess"}<em>♔ 国际象棋</em>{/if}
      {#if safe.gameId === "liarsdice"}<em>🎲 大话骰</em>{/if}
      {#if safe.gameId === "coinflip"}<em>🪙 猜硬币</em>{/if}
      {#if safe.ranked}<em>🏆 {safe.gameId === "othello" ? `${safe.stake}分/子${safe.rankMultiplier && safe.rankMultiplier > 1 ? ` ×${safe.rankMultiplier}` : ""}` : `${safe.stake}分${safe.rankMultiplier && safe.rankMultiplier > 1 ? ` ×${safe.rankMultiplier}` : ""}`}</em>{/if}
      {#if safe.extremeRanked}<em>⚡ 极限</em>{/if}
      {#if safe.punishedNames.length > 0}<em>🎲 惩罚</em>{/if}
    </div>
  </header>
  <div class="history-duel">
    <div class="history-side">
      <span>{safe.playerA}</span>
      <strong>{historySeatLabel(safe, "A", coinFaceLabel)}</strong>
    </div>
    <div class="history-result">
      <small>{safe.gameId === "othello" && safe.othelloScore ? `${safe.othelloScore.black} : ${safe.othelloScore.white}` : safe.gameId === "tictactoe" ? "3 × 3" : safe.gameId === "gomoku" ? "15 × 15" : safe.gameId === "jungle" ? "7 × 9" : safe.gameId === "chess" ? "8 × 8" : safe.gameId === "coinflip" ? "🪙" : "VS"}</small>
      <b>{safe.resultLabel || historyResultText(safe.result)}</b>
    </div>
    <div class="history-side">
      <span>{safe.playerB}</span>
      <strong>{historySeatLabel(safe, "B", coinFaceLabel)}</strong>
    </div>
  </div>
  {#if safe.punishedNames.length > 0}
    <section class="history-section">
      <!-- 发布任务 + 完成证明包在同一个 .history-task-group 圆角矩形里，底色统一为白色，
           中间靠 .history-proof.inline 顶部的虚线分隔（见 styles.css），不再是各自独立
           的两个圆角矩形。 -->
      {#each safe.punishmentTasks as task (task.playerId)}
        {@const proof = proofByPlayer.get(task.playerId)}
        {@const taskStyle = task.backgroundImage ? styleString({ "--task-bg": `url(${task.backgroundImage})`, "--task-bg-opacity": String(task.backgroundOpacity ?? 0.22) }) : undefined}
        <div class="history-task-group">
          <div class={`history-task ${task.backgroundImage ? "has-task-background" : ""}`} style={taskStyle}>
            <small>{task.playerName} 的任务{task.assignedByName ? ` · ${task.assignedByName} 发布` : ""}</small>
            <p>{task.taskText ? taskTextOnly(task.taskText, task.factionLabel) : "等待玩家发布任务"}</p>
          </div>
          {#if proof}
            <div class="history-proof inline">
              <span>完成证明 · {historyProofStatusLabel(proof)}</span>
              {#if proof.taskText && proof.taskText !== task.taskText}<small>对应任务：{proof.taskText}</small>{/if}
              <p>{proof.text || "（无文字）"}</p>
              {#if proof.rejectReason}<small>审核备注：{proof.rejectReason}</small>{/if}
              {#if proof.redoTaskText}<small>重做任务：{proof.redoTaskText}</small>{/if}
              {#if proof.imageUrl}
                <button class="history-proof-image-button" onclick={() => onOpenImage(proof.imageUrl!)}>
                  <ProofImage src={proof.imageUrl} alt="惩罚证明" />
                </button>
              {/if}
            </div>
          {:else}
            <div class="history-proof inline muted"><span>完成证明</span><p>尚未提交</p></div>
          {/if}
        </div>
      {/each}
    </section>
  {/if}
  {#if looseProofs.length > 0}
    <section class="history-section">
      <b>其它完成证明</b>
      {#each looseProofs as proof (proof.playerId)}
        <div class="history-proof">
          <span>{proof.playerName || "玩家"} · {historyProofStatusLabel(proof)}</span>
          {#if proof.taskText}<small>任务：{proof.taskText}</small>{/if}
          <p>{proof.text || "（无文字）"}</p>
          {#if proof.rejectReason}<small>审核备注：{proof.rejectReason}</small>{/if}
          {#if proof.imageUrl}
            <button class="history-proof-image-button" onclick={() => onOpenImage(proof.imageUrl!)}>
              <ProofImage src={proof.imageUrl} alt="惩罚证明" />
            </button>
          {/if}
        </div>
      {/each}
    </section>
  {/if}
</article>
