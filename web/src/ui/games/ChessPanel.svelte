<script lang="ts">
  // 源：ui/ChessPanel.tsx
  import ChessKnight from "@lucide/svelte/icons/chess-knight";
  import type { ChessPiece, RoomSnapshot, PublicPlayer, SeatKey } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { occupantDisplay, chessDeltaText, chessSideLabel, chessThemeStyle, parseChessCell } from "../../lib/gameDisplay";
  import { useNow } from "../../lib/useNow.svelte";
  import { styleString } from "../../lib/style";
  import GameClockBar from "../shell/GameClockBar.svelte";

  const SIZE = 8;
  const FILES = ["a", "b", "c", "d", "e", "f", "g", "h"];
  // U+FE0E 强制使用可重着色的文本字形，避免 Safari 把部分棋子回退成黑色彩色字体。
  const PIECE_GLYPH: Record<string, string> = { king: "♚︎", queen: "♛︎", rook: "♜︎", bishop: "♝︎", knight: "♞︎", pawn: "♟︎" };
  const PIECE_LABEL: Record<string, string> = { king: "王", queen: "后", rook: "车", bishop: "象", knight: "马", pawn: "兵" };
  const PROMOTE_OPTIONS: Array<{ id: ChessPiece; label: string }> = [
    { id: "queen", label: "后" }, { id: "rook", label: "车" }, { id: "bishop", label: "象" }, { id: "knight", label: "马" }
  ];

  let { room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void } = $props();

  const gameState = $derived(room.chess);
  const boardTheme = $derived(styleString(chessThemeStyle(room.settings.chessBoardTheme)));
  let busy = $state(false);
  let selected = $state<{ row: number; col: number } | null>(null);
  let pendingPromote = $state<{ from: { row: number; col: number }; to: { row: number; col: number } } | null>(null);
  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null);
  const myColor = $derived(gameState && mySeat ? (gameState.whiteSeat === mySeat ? "white" : "black") : null);

  $effect(() => {
    void gameState?.moveCount;
    void room.phase;
    selected = null;
    pendingPromote = null;
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

  const board = $derived(gameState?.board || Array.from({ length: SIZE }, () => Array.from({ length: SIZE }, () => null as string | null)));
  const isMyTurn = $derived(Boolean(mySeat && gameState && gameState.turn === mySeat && room.phase === "choosing" && !gameState.ended && !gameState.resignRequest && !gameState.undoRequest));
  const dests = $derived.by(() => {
    if (!isMyTurn || !selected || !gameState) return [] as Array<{ row: number; col: number; capture: boolean; promote?: string }>;
    return (gameState.legalMoves || [])
      .filter((m) => Number(m.from?.row) === selected!.row && Number(m.from?.col) === selected!.col)
      .map((m) => {
        const row = Number(m.to?.row);
        const col = Number(m.to?.col);
        const fromPiece = parseChessCell(board[selected!.row]?.[selected!.col]);
        return { row, col, promote: m.promote, capture: Boolean(parseChessCell(board[row]?.[col])) || (fromPiece?.piece === "pawn" && col !== selected!.col) };
      });
  });
  const destKeys = $derived(new Set(dests.map((p) => `${p.row}-${p.col}`)));
  const captureKeys = $derived(new Set(dests.filter((p) => p.capture).map((p) => `${p.row}-${p.col}`)));

  function onCellClick(row: number, col: number) {
    if (!isMyTurn || !mySeat || busy) return;
    const key = `${row}-${col}`;
    if (selected && destKeys.has(key)) {
      const options = dests.filter((d) => d.row === row && d.col === col);
      const needsPromote = options.some((d) => d.promote);
      if (needsPromote) {
        pendingPromote = { from: selected, to: { row, col } };
        return;
      }
      const from = selected;
      selected = null;
      act("chess:move", { fromRow: from.row, fromCol: from.col, toRow: row, toCol: col });
      return;
    }
    const piece = parseChessCell(board[row][col]);
    if (piece && piece.color === myColor) {
      selected = selected && selected.row === row && selected.col === col ? null : { row, col };
      return;
    }
    selected = null;
  }

  function confirmPromote(promote: ChessPiece) {
    if (!pendingPromote) return;
    const { from, to } = pendingPromote;
    pendingPromote = null;
    selected = null;
    act("chess:move", { fromRow: from.row, fromCol: from.col, toRow: to.row, toCol: to.col, promote });
  }

  const turnName = $derived(gameState?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const waitingForReady = $derived(room.phase === "ready" && Boolean(room.seats.A && room.seats.B));
  const drawingFirst = $derived(waitingForReady && room.ready.A && room.ready.B);
  const myReady = $derived(mySeat ? room.ready[mySeat] : false);
  const undoLimit = $derived(room.settings.chessUndoLimit ?? 0);
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
  const kingInCheck = $derived(Boolean(gameState?.inCheck && gameState.turn));
  const checkSquare = $derived.by(() => {
    if (!gameState?.inCheck) return null;
    const color = gameState.whiteSeat === gameState.turn ? "white" : "black";
    for (let r = 0; r < SIZE; r++) {
      for (let c = 0; c < SIZE; c++) {
        const p = parseChessCell(board[r][c]);
        if (p && p.color === color && p.piece === "king") return { row: r, col: c };
      }
    }
    return null;
  });

  const statusHint = $derived.by(() => {
    if (!room.seats.A || !room.seats.B) return "等待两个战斗席坐满。";
    if (drawingFirst) return "正在随机谁执白先手...";
    if (waitingForReady) return "双方准备后随机决定谁执白先走。";
    if (gameState?.ended) return room.resultText || "对局结束";
    if (isMyTurn) return pendingPromote ? "选择升变成什么棋子。" : selected ? "点击目标格走子。" : "点击想要移动的棋子。";
    if (mySeat) return `轮到对方（ ${turnName} ）走子。`;
    return `轮到 ${turnName} 走子。`;
  });

  const blackSeat = $derived<SeatKey>(gameState ? (gameState.whiteSeat === "A" ? "B" : "A") : "B");
</script>

<div class="chess-panel">
  <div class="chess-head">
    <div>
      <h3 class="chess-title"><ChessKnight size={19} strokeWidth={2.1} aria-hidden="true" />国际象棋</h3>
      <p class="hint">{statusHint}{kingInCheck && !gameState?.ended ? (isMyTurn ? " 你已被将军。" : mySeat ? " 对方已被将军。" : " 当前行棋方已被将军。") : ""}</p>
    </div>
    {#if gameState}
      <div class="chess-turn-card">
        <span>轮到 {chessSideLabel(gameState, gameState.turn)}</span>
        {#if room.settings.enableRanked && gameState.rankedDelta}<small>本局排位：A {chessDeltaText(gameState, "A")} / B {chessDeltaText(gameState, "B")}</small>{/if}
      </div>
    {/if}
  </div>
  {#if waitingForReady}
    <div class={`chess-ready-card ${drawingFirst ? "drawing" : ""}`}>
      <div class="chess-draw-animation" aria-hidden="true"><span>♔</span><span>♕</span><span>♞</span></div>
      <div>
        <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
        <p class="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
      </div>
      {#if mySeat && !myReady && !drawingFirst}<button class="primary" disabled={busy} onclick={() => act("chess:ready")}>准备</button>{/if}
      {#if mySeat && myReady && !drawingFirst}<button disabled>等待对方</button>{/if}
    </div>
  {/if}
  {#if gameState?.ended && mySeat && room.phase === "result"}
    <button class="primary chess-restart-button" disabled={busy} onclick={() => act("chess:restart")}>再来一局</button>
  {/if}
  {#if undoRequest}
    <div class={`chess-request-card ${undoToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
        <p class="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
      </div>
      {#if undoToMe}
        <div class="chess-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("chess:undoRespond", { accept: true })}>同意悔棋</button>
          <button class="soft-button" disabled={busy} onclick={() => act("chess:undoRespond", { accept: false })}>拒绝</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if resignRequest}
    <div class={`chess-request-card ${resignToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
        <p class="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
      </div>
      {#if resignToMe}
        <div class="chess-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("chess:resignRespond", { accept: true })}>同意认输</button>
          <button class="soft-button" disabled={busy} onclick={() => act("chess:resignRespond", { accept: false })}>拒绝，继续下棋</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if canRequestUndo || canRequestResign}
    <div class="chess-risk-actions">
      {#if canRequestUndo}<button class="soft-button" disabled={busy} onclick={() => act("chess:undoRequest")}>申请悔棋<br />本局剩 {undoLimit - myUndoCount} 次</button>{/if}
      {#if canRequestResign}<button class="soft-button danger-soft" disabled={busy} onclick={() => act("chess:resignRequest")}>申请认输</button>{/if}
    </div>
  {/if}
  {#if pendingPromote}
    <div class="chess-request-card needs-action">
      <div><strong>兵升变</strong><p class="hint">选择把兵变成哪种棋子。</p></div>
      <div class="chess-request-actions">
        {#each PROMOTE_OPTIONS as opt (opt.id)}
          <button class="primary" disabled={busy} onclick={() => confirmPromote(opt.id)}>{opt.label}</button>
        {/each}
        <button class="soft-button" disabled={busy} onclick={() => (pendingPromote = null)}>取消</button>
      </div>
    </div>
  {/if}
  <GameClockBar
    {room}
    state={gameState ? { turn: gameState.turn, blackSeat, ended: gameState.ended, moveDeadlineAt: gameState.moveDeadlineAt, clockDeadlineAt: gameState.clockDeadlineAt, clockRemaining: gameState.clockRemaining } : undefined}
    moveSeconds={room.settings.chessMoveSeconds}
    gameMinutes={room.settings.chessGameMinutes}
    labels={{ primary: "⚫ 黑", secondary: "⚪ 白" }}
  />
  <div class="chess-board-wrap" style={boardTheme}>
    <div class="chess-ranks" aria-hidden="true">
      {#each Array.from({ length: SIZE }, (_, i) => i) as i (i)}<span>{SIZE - i}</span>{/each}
    </div>
    <div class="chess-board" role="grid" aria-label="国际象棋棋盘">
      {#each Array.from({ length: SIZE }, (_, row) => row) as row (row)}
        {#each Array.from({ length: SIZE }, (_, col) => col) as col (col)}
          {@const key = `${row}-${col}`}
          {@const cell = board[row]?.[col]}
          {@const piece = parseChessCell(cell)}
          {@const isLight = (row + col) % 2 === 0}
          {@const isSelected = Boolean(selected && selected.row === row && selected.col === col)}
          {@const isDest = destKeys.has(key)}
          {@const isCapture = captureKeys.has(key)}
          {@const isLastFrom = Boolean(gameState?.lastFrom && gameState.lastFrom.row === row && gameState.lastFrom.col === col)}
          {@const isLastTo = Boolean(gameState?.lastTo && gameState.lastTo.row === row && gameState.lastTo.col === col)}
          {@const isCheck = Boolean(checkSquare && checkSquare.row === row && checkSquare.col === col)}
          <button
            type="button"
            class={[
              "chess-cell", isLight ? "light" : "dark", isSelected ? "selected" : "", isDest ? "legal" : "",
              isCapture ? "capture" : "", isLastFrom ? "last-from" : "", isLastTo ? "last-to" : "", isCheck ? "in-check" : ""
            ].filter(Boolean).join(" ")}
            disabled={!isMyTurn}
            onclick={() => onCellClick(row, col)}
            aria-label={`${FILES[col]}${8 - row}${piece ? `，${piece.color === "white" ? "白" : "黑"}${PIECE_LABEL[piece.piece] || piece.piece}` : ""}`}
          >
            {#if piece}<span class={`chess-piece color-${piece.color}`} aria-hidden="true">{PIECE_GLYPH[piece.piece] || "?"}</span>{/if}
          </button>
        {/each}
      {/each}
    </div>
    <div class="chess-files" aria-hidden="true">
      {#each Array.from({ length: SIZE }, (_, i) => i) as i (i)}<span>{FILES[i]}</span>{/each}
    </div>
  </div>
  <div class="chess-rules-hint hint">
    <p>白方先走。王车易位、吃过路兵、兵到底线升变都按常规规则。将死对方的王即胜；逼和、子力不足、五十步或三次重复局面算平局。</p>
  </div>
</div>
