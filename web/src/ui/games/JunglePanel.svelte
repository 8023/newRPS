<script lang="ts">
  // 源：ui/JunglePanel.tsx
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { occupantDisplay, jungleDeltaText, jungleSideLabel, jungleThemeStyle } from "../../lib/gameDisplay";
  import { ANIMAL_EMOJI, ANIMAL_LABEL, COLS, ROWS, isDen, isTrap, isWater, legalDests, parseCell } from "../../lib/jungleRules";
  import { useNow } from "../../lib/useNow.svelte";
  import { styleString } from "../../lib/style";
  import GameClockBar from "../shell/GameClockBar.svelte";

  let { room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void } = $props();

  const gameState = $derived(room.jungle);
  const boardTheme = $derived(styleString(jungleThemeStyle(room.settings.jungleBoardTheme)));
  let busy = $state(false);
  let selected = $state<{ row: number; col: number } | null>(null);
  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null);

  $effect(() => {
    void gameState?.moveCount;
    void room.phase;
    selected = null;
  });

  async function act(event: string, payload: unknown = {}) {
    if (busy) return;
    busy = true;
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      busy = false;
    }
  }

  const board = $derived(gameState?.board || Array.from({ length: ROWS }, () => Array.from({ length: COLS }, () => null as string | null)));
  const isMyTurn = $derived(Boolean(mySeat && gameState && gameState.turn === mySeat && room.phase === "choosing" && !gameState.ended && !gameState.resignRequest && !gameState.undoRequest));
  const destKeys = $derived.by(() => {
    if (!isMyTurn || !selected || !mySeat) return new Set<string>();
    return new Set(legalDests(board, mySeat, selected.row, selected.col).map((p) => `${p.row}-${p.col}`));
  });

  function onCellClick(row: number, col: number) {
    if (!isMyTurn || !mySeat || busy) return;
    const key = `${row}-${col}`;
    if (selected && destKeys.has(key)) {
      const from = selected;
      selected = null;
      act("jungle:move", { fromRow: from.row, fromCol: from.col, toRow: row, toCol: col });
      return;
    }
    const piece = parseCell(board[row][col]);
    if (piece && piece.side === mySeat) {
      selected = selected && selected.row === row && selected.col === col ? null : { row, col };
      return;
    }
    selected = null;
  }

  const turnName = $derived(gameState?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const waitingForReady = $derived(room.phase === "ready" && Boolean(room.seats.A && room.seats.B));
  const drawingFirst = $derived(waitingForReady && room.ready.A && room.ready.B);
  const myReady = $derived(mySeat ? room.ready[mySeat] : false);
  const undoLimit = $derived(room.settings.jungleUndoLimit ?? 0);
  const undoRequest = $derived(gameState?.undoRequest);
  const undoToMe = $derived(Boolean(mySeat && undoRequest?.toSeat === mySeat));
  const undoFromMe = $derived(Boolean(mySeat && undoRequest?.fromSeat === mySeat));
  const undoFromName = $derived(undoRequest ? occupantDisplay(room.seats[undoRequest.fromSeat]) : "");
  const now = useNow(() => 1000, () => Boolean(undoRequest));
  const undoSecondsLeft = $derived(undoRequest ? Math.max(0, Math.ceil((undoRequest.expiresAt - now.value) / 1000)) : 0);
  const myUndoCount = $derived(mySeat ? (gameState?.undoCount?.[mySeat] ?? 0) : 0);
  const canRequestUndo = $derived(Boolean(
    gameState && mySeat && room.phase === "choosing" && !gameState.ended && !gameState.undoRequest && !gameState.resignRequest &&
    gameState.turn === mySeat && gameState.moveCount >= 2 && myUndoCount < undoLimit
  ));
  const resignRequest = $derived(gameState?.resignRequest);
  const resignToMe = $derived(Boolean(mySeat && resignRequest?.toSeat === mySeat));
  const resignFromMe = $derived(Boolean(mySeat && resignRequest?.fromSeat === mySeat));
  const resignFromName = $derived(resignRequest ? occupantDisplay(room.seats[resignRequest.fromSeat]) : "");
  const canRequestResign = $derived(Boolean(gameState && mySeat && room.phase === "choosing" && !gameState.ended && !gameState.undoRequest && !gameState.resignRequest));

  const statusHint = $derived.by(() => {
    if (!room.seats.A || !room.seats.B) return "等待两个战斗席坐满。";
    if (drawingFirst) return "正在随机先手...";
    if (waitingForReady) return "双方准备后随机决定谁先手；A 执下方白棋⚪，B 执上方黑棋⚫。";
    if (gameState?.ended) return room.resultText || "对局结束";
    if (isMyTurn) return selected ? "点击目标格走子。" : "点击想要移动的棋子。";
    if (mySeat) return `轮到对方（ ${turnName} ）走子。`;
    return `轮到 ${turnName} 走子。`;
  });

  const rankOrder = ["elephant", "lion", "tiger", "leopard", "wolf", "dog", "cat", "rat"] as const;
</script>

<div class="jungle-panel">
  <div class="jungle-head">
    <div>
      <h3>🦁 斗兽棋</h3>
      <p class="hint">{statusHint}</p>
    </div>
    {#if gameState}
      <div class="jungle-turn-card">
        <span>轮到 {jungleSideLabel(gameState.turn)}</span>
        {#if room.settings.enableRanked && gameState.rankedDelta}<small>本局排位：白 {jungleDeltaText(gameState, "A")} / 黑 {jungleDeltaText(gameState, "B")}</small>{/if}
      </div>
    {/if}
  </div>
  {#if waitingForReady}
    <div class={`jungle-ready-card ${drawingFirst ? "drawing" : ""}`}>
      <div class="jungle-draw-animation" aria-hidden="true"><span>🦁</span><span>🐘</span><span>🐯</span></div>
      <div>
        <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
        <p class="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
      </div>
      {#if mySeat && !myReady && !drawingFirst}<button class="primary" disabled={busy} onclick={() => act("jungle:ready")}>准备</button>{/if}
      {#if mySeat && myReady && !drawingFirst}<button disabled>等待对方</button>{/if}
    </div>
  {/if}
  {#if gameState?.ended && mySeat && room.phase === "result"}
    <button class="primary jungle-restart-button" disabled={busy} onclick={() => act("jungle:restart")}>再来一局</button>
  {/if}
  {#if undoRequest}
    <div class={`jungle-request-card ${undoToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
        <p class="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
      </div>
      {#if undoToMe}
        <div class="jungle-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("jungle:undoRespond", { accept: true })}>同意悔棋</button>
          <button class="soft-button" disabled={busy} onclick={() => act("jungle:undoRespond", { accept: false })}>拒绝</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if resignRequest}
    <div class={`jungle-request-card ${resignToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
        <p class="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
      </div>
      {#if resignToMe}
        <div class="jungle-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("jungle:resignRespond", { accept: true })}>同意认输</button>
          <button class="soft-button" disabled={busy} onclick={() => act("jungle:resignRespond", { accept: false })}>拒绝，继续下棋</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if canRequestUndo || canRequestResign}
    <div class="jungle-risk-actions">
      {#if canRequestUndo}<button class="soft-button" disabled={busy} onclick={() => act("jungle:undoRequest")}>申请悔棋<br />本局剩 {undoLimit - myUndoCount} 次</button>{/if}
      {#if canRequestResign}<button class="soft-button danger-soft" disabled={busy} onclick={() => act("jungle:resignRequest")}>申请认输</button>{/if}
    </div>
  {/if}
  <GameClockBar
    {room}
    state={gameState ? { turn: gameState.turn, blackSeat: "A", ended: gameState.ended, moveDeadlineAt: gameState.moveDeadlineAt, clockDeadlineAt: gameState.clockDeadlineAt, clockRemaining: gameState.clockRemaining } : undefined}
    moveSeconds={room.settings.jungleMoveSeconds}
    gameMinutes={room.settings.jungleGameMinutes}
    labels={{ primary: "下方⬇ 白", secondary: "上方⬆ 黑" }}
  />
  <div class="jungle-board" role="grid" aria-label="斗兽棋棋盘" style={boardTheme}>
    {#each board as row, rowIndex (rowIndex)}
      {#each row as cell, colIndex (colIndex)}
        {@const key = `${rowIndex}-${colIndex}`}
        {@const piece = parseCell(cell)}
        {@const water = isWater(rowIndex, colIndex)}
        {@const denA = isDen(rowIndex, colIndex, "A")}
        {@const denB = isDen(rowIndex, colIndex, "B")}
        {@const trapA = isTrap(rowIndex, colIndex, "A")}
        {@const trapB = isTrap(rowIndex, colIndex, "B")}
        {@const isSelected = Boolean(selected && selected.row === rowIndex && selected.col === colIndex)}
        {@const isDest = destKeys.has(key)}
        {@const isLastFrom = Boolean(gameState?.lastFrom && gameState.lastFrom.row === rowIndex && gameState.lastFrom.col === colIndex)}
        {@const isLastTo = Boolean(gameState?.lastTo && gameState.lastTo.row === rowIndex && gameState.lastTo.col === colIndex)}
        {@const terrain = water ? "water" : denA || denB ? "den" : trapA || trapB ? "trap" : "land"}
        <button
          type="button"
          class={["jungle-cell", terrain, isSelected ? "selected" : "", isDest ? "legal" : "", isLastFrom ? "last-from" : "", isLastTo ? "last-to" : ""].filter(Boolean).join(" ")}
          disabled={!isMyTurn}
          onclick={() => onCellClick(rowIndex, colIndex)}
          aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列${piece ? `，${piece.side}方${ANIMAL_LABEL[piece.animal]}` : ""}`}
        >
          {#if denA || denB}<span class="jungle-terrain-mark" aria-hidden="true">穴</span>{/if}
          {#if (trapA || trapB) && !piece}<span class="jungle-terrain-mark trap-mark" aria-hidden="true">陷</span>{/if}
          {#if water && !piece}<span class="jungle-terrain-mark water-mark" aria-hidden="true">～</span>{/if}
          {#if piece}<span class={`jungle-piece side-${piece.side}`} title={ANIMAL_LABEL[piece.animal]} aria-hidden="true">{ANIMAL_EMOJI[piece.animal]}</span>{/if}
        </button>
      {/each}
    {/each}
  </div>
  <div class="jungle-rules-hint hint">
    <p class="jungle-rank-line" aria-label="等级：象大于狮大于虎大于豹大于狼大于狗大于猫大于鼠">
      <span class="jungle-rank-label">等级</span>
      {#each rankOrder as animal, index (animal)}
        <span class="jungle-rank-item">
          {#if index > 0}<span class="jungle-rank-gt" aria-hidden="true">&gt;</span>{/if}
          <span class="jungle-rank-icon" title={ANIMAL_LABEL[animal]}>{ANIMAL_EMOJI[animal]}</span>
        </span>
      {/each}
    </p>
    <p>
      🐀 可吃 🐘（不能从水中上岸吃），🐘 不能吃 🐀；仅 🐀 可入水；🦁🐅 可跳河（河中有 🐀 则不能跳）；先入对方兽穴或令对方无子可走者胜。
      敌兽进入己方陷阱，可被任意己方兽吃掉。
    </p>
  </div>
</div>
