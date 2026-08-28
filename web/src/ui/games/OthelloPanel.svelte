<script lang="ts">
  // 源：ui/AppViews.tsx:2428-2566
  import type { RoomSnapshot, PublicPlayer } from "../../shared/types";
  import { occupantDisplay, othelloDeltaText, othelloThemeStyle } from "../../lib/gameDisplay";
  import { useNow } from "../../lib/useNow.svelte";
  import { styleString } from "../../lib/style";
  import GameClockBar from "../shell/GameClockBar.svelte";
  import OthelloSettlementCard from "./OthelloSettlementCard.svelte";

  let { room, me, onMove, onSettle, onRestart, onReady, onRequestUndo, onRespondUndo, onRequestSurrender, onRespondSurrender, onEscape }: {
    room: RoomSnapshot;
    me: PublicPlayer;
    onMove: (row: number, col: number) => void;
    onSettle: (mode: "normal" | "giveaway" | "tribute") => void;
    onRestart: () => void;
    onReady: () => void;
    onRequestUndo: () => void;
    onRespondUndo: (accept: boolean) => void;
    onRequestSurrender: () => void;
    onRespondSurrender: (accept: boolean) => void;
    onEscape: () => void;
  } = $props();

  const state = $derived(room.othello);
  const boardTheme = $derived(styleString(othelloThemeStyle(room.settings.othelloBoardTheme)));
  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null);
  const pending = $derived(state?.pendingSettlement);
  const isMyTurn = $derived(Boolean(state && mySeat && state.turn === mySeat && room.phase === "choosing" && !state.ended && !pending && !state.undoRequest));
  const legalKeys = $derived(new Set((state?.legalMoves || []).map((move) => `${move.row}-${move.col}`)));
  const turnName = $derived(state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const waitingForReady = $derived(room.phase === "ready" && Boolean(room.seats.A && room.seats.B));
  const drawingFirst = $derived(waitingForReady && room.ready.A && room.ready.B);
  const myReady = $derived(mySeat ? room.ready[mySeat] : false);
  const undoLimit = $derived(room.settings.othelloUndoLimit ?? 0);
  const undoRequest = $derived(state?.undoRequest);
  const undoToMe = $derived(Boolean(mySeat && undoRequest?.toSeat === mySeat));
  const undoFromMe = $derived(Boolean(mySeat && undoRequest?.fromSeat === mySeat));
  const undoFromName = $derived(undoRequest ? occupantDisplay(room.seats[undoRequest.fromSeat]) : "");
  const now = useNow(() => 1000, () => Boolean(undoRequest));
  const undoSecondsLeft = $derived(undoRequest ? Math.max(0, Math.ceil((undoRequest.expiresAt - now.value) / 1000)) : 0);
  const myUndoCount = $derived(mySeat ? (state?.undoCount?.[mySeat] ?? 0) : 0);
  const moveCount = $derived(state ? Math.max(0, state.blackCount + state.whiteCount - 4) : 0);
  const canRequestUndo = $derived(Boolean(
    mySeat && state && room.phase === "choosing" && !state.ended && !pending && !state.undoRequest && !state.surrenderRequest &&
    state.turn === mySeat && moveCount >= 2 && myUndoCount < undoLimit
  ));
  const canSurrender = $derived(Boolean(mySeat && state && room.phase === "choosing" && !state.ended && !pending && !state.undoRequest));
  const surrenderRequest = $derived(state?.surrenderRequest);
  const surrenderFromMe = $derived(Boolean(mySeat && surrenderRequest?.fromSeat === mySeat));
  const surrenderToMe = $derived(Boolean(mySeat && surrenderRequest?.toSeat === mySeat));
  const surrenderFromName = $derived(surrenderRequest ? occupantDisplay(room.seats[surrenderRequest.fromSeat]) : "");
  const board = $derived(state?.board || Array.from({ length: 8 }, () => Array.from({ length: 8 }, () => null)));
</script>

<div class="othello-panel">
  <div class="othello-head">
    <div>
      <h3>⚫⚪ 黑白棋</h3>
      <p class="hint">
        {#if !room.seats.A || !room.seats.B}等待两个战斗席坐满。
        {:else if drawingFirst}正在随机执黑先手...
        {:else if waitingForReady}双方准备后随机决定谁执黑先手。
        {:else if state?.ended}{room.resultText || "对局结束"}
        {:else if pending}正在结算本手白给/上贡。
        {:else}{isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}{/if}
      </p>
    </div>
    {#if state}
      <div class="othello-turn-card">
        <span>{state.blackSeat === state.turn ? "⚫ 黑棋" : "⚪ 白棋"}</span>
        <strong>{state.blackCount} : {state.whiteCount}</strong>
        {#if room.settings.enableRanked && state.rankedDelta}<small>本局排位：黑 {othelloDeltaText(state, "black")} / 白 {othelloDeltaText(state, "white")}</small>{/if}
      </div>
    {/if}
  </div>
  {#if waitingForReady}
    <div class={`othello-ready-card ${drawingFirst ? "drawing" : ""}`}>
      <div class="othello-draw-animation" aria-hidden="true"><span>⚫</span><span>⚪</span><span>⚫</span></div>
      <div>
        <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
        <p class="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
      </div>
      {#if mySeat && !myReady && !drawingFirst}<button class="primary" onclick={onReady}>准备</button>{/if}
      {#if mySeat && myReady && !drawingFirst}<button disabled>等待对方</button>{/if}
    </div>
  {/if}
  {#if state?.ended && mySeat && room.phase === "result"}
    <button class="primary othello-restart-button" onclick={onRestart}>再来一局</button>
  {/if}
  {#if undoRequest}
    <div class={`othello-surrender-card ${undoToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
        <p class="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
      </div>
      {#if undoToMe}
        <div class="othello-surrender-actions">
          <button class="primary" onclick={() => onRespondUndo(true)}>同意悔棋</button>
          <button class="soft-button" onclick={() => onRespondUndo(false)}>拒绝</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if canSurrender && surrenderRequest}
    <div class={`othello-surrender-card ${surrenderToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{surrenderFromMe ? "已申请认输" : `${surrenderFromName} 申请认输`}</strong>
        <p class="hint">{surrenderFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
      </div>
      {#if surrenderToMe}
        <div class="othello-surrender-actions">
          <button class="primary" onclick={() => onRespondSurrender(true)}>同意认输</button>
          <button class="soft-button" onclick={() => onRespondSurrender(false)}>拒绝，继续下棋</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if canRequestUndo || canSurrender}
    <div class="othello-risk-actions">
      {#if canRequestUndo}
        <button class="soft-button othello-surrender-button" onclick={onRequestUndo}>申请悔棋<br />本局剩 {undoLimit - myUndoCount} 次</button>
      {/if}
      {#if canSurrender}<button class="soft-button othello-surrender-button" disabled={Boolean(surrenderRequest)} onclick={onRequestSurrender}>申请认输</button>{/if}
      {#if canSurrender}<button class="soft-button danger-soft othello-surrender-button" onclick={onEscape}>逃跑</button>{/if}
    </div>
  {/if}
  {#if pending}
    <OthelloSettlementCard {room} {me} {pending} {onSettle} />
  {/if}
  <GameClockBar {room} {state} moveSeconds={room.settings.othelloMoveSeconds} gameMinutes={room.settings.othelloGameMinutes} />
  <div class="othello-board" role="grid" aria-label="黑白棋棋盘" style={boardTheme}>
    {#each board as row, rowIndex (rowIndex)}
      {#each row as cell, colIndex (colIndex)}
        {@const legal = legalKeys.has(`${rowIndex}-${colIndex}`)}
        <div class="othello-cell-wrap">
          <button
            type="button"
            class={`othello-cell ${cell || ""} ${legal ? "legal" : ""}`}
            disabled={!isMyTurn || !legal}
            onclick={() => onMove(rowIndex, colIndex)}
            aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列`}
          ></button>
          {#if cell}<span class={`othello-disc ${cell}`}></span>{/if}
          {#if !cell && legal}<span class="othello-legal-dot"></span>{/if}
        </div>
      {/each}
    {/each}
  </div>
</div>
