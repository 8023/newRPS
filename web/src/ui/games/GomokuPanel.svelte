<script lang="ts">
  // 源：ui/GomokuPanel.tsx
  import type { RoomSnapshot, PublicPlayer, SeatKey } from "../../shared/types";
  import { ask } from "../../lib/rpc";
  import { occupantDisplay, gomokuDeltaText, gomokuThemeStyle } from "../../lib/gameDisplay";
  import { useNow } from "../../lib/useNow.svelte";
  import { styleString } from "../../lib/style";
  import GameClockBar from "../shell/GameClockBar.svelte";

  let { room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void } = $props();

  // 仅靠 `(hover: none) and (pointer: coarse)` 媒体查询判断触屏在 Android 上并不可靠：
  // 大量国产 Android 浏览器/内嵌 WebView（如微信 X5 内核、UC、QQ 浏览器、部分厂商 ROM 自带浏览器）
  // 出于兼容旧网页鼠标交互的考虑，会把触屏错误上报为 hover:hover / pointer:fine，
  // 导致该查询判断为假、二次确认功能被跳过；iOS 只有 Safari(WebKit) 一种引擎，上报始终准确，
  // 因此现象是"iPhone 正常，安卓大多失效"。改用 ontouchstart/maxTouchPoints 兜底即可覆盖这些设备。
  const isTouchDevice = typeof window !== "undefined" && (
    window.matchMedia("(hover: none) and (pointer: coarse)").matches ||
    navigator.maxTouchPoints > 0 ||
    "ontouchstart" in window
  );

  const gameState = $derived(room.gomoku);
  const boardTheme = $derived(styleString(gomokuThemeStyle(room.settings.gomokuBoardTheme)));
  let busy = $state(false);
  let pendingCell = $state<{ row: number; col: number } | null>(null);
  const mySeat = $derived(room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null);

  $effect(() => {
    void gameState?.moveCount;
    void room.phase;
    pendingCell = null;
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

  function placeStone(rowIndex: number, colIndex: number) {
    if (isTouchDevice) {
      if (pendingCell && pendingCell.row === rowIndex && pendingCell.col === colIndex) {
        pendingCell = null;
        act("gomoku:move", { row: rowIndex, col: colIndex });
      } else {
        pendingCell = { row: rowIndex, col: colIndex };
      }
      return;
    }
    act("gomoku:move", { row: rowIndex, col: colIndex });
  }

  const isMyTurn = $derived(Boolean(mySeat && gameState && gameState.turn === mySeat && room.phase === "choosing" && !gameState.ended && !gameState.undoRequest && !gameState.resignRequest));
  const giveawayArmed = $derived(Boolean(mySeat && gameState?.giveawaySeat === mySeat));
  const showGiveawayControl = $derived(Boolean(mySeat && gameState && me.giveawayEnabled && room.seats.A && room.seats.B && room.phase === "choosing" && !gameState.ended));
  const canChooseGiveaway = $derived(Boolean(showGiveawayControl && isMyTurn));
  const turnName = $derived(gameState?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B));
  const waitingForReady = $derived(room.phase === "ready" && Boolean(room.seats.A && room.seats.B));
  const drawingFirst = $derived(waitingForReady && room.ready.A && room.ready.B);
  const myReady = $derived(mySeat ? room.ready[mySeat] : false);
  const blackSeat = $derived(gameState?.blackSeat);
  const whiteSeat = $derived<SeatKey | null>(blackSeat ? (blackSeat === "A" ? "B" : "A") : null);
  const lastMove = $derived(gameState?.moves.length ? gameState.moves[gameState.moves.length - 1] : null);
  const winningKeys = $derived(new Set((gameState?.winningLine || []).map((p) => `${p.row}-${p.col}`)));

  // 缺省按 0（禁止悔棋）而不是 3：wire.ts/pbconv 的 fillRoomSettingsDefaults 已经把
  // "键缺失"统一按真值 0 补齐（建房时必然是 0/1/3/10 之一，缺失只可能是零值被 protojson
  // 省略），这里若退回 3 会跟服务端 game_gomoku.go 的实际校验（用的是真实值，不做兜底）
  // 不一致——显示"还能悔棋"，点了却被服务端拒绝。
  const undoLimit = $derived(room.settings.gomokuUndoLimit ?? 0);
  const canRequestUndo = $derived(Boolean(
    gameState && mySeat && room.phase === "choosing" && !gameState.ended && !gameState.undoRequest && !gameState.resignRequest && !giveawayArmed &&
    gameState.turn === mySeat && gameState.moveCount >= 2 && (gameState.undoCount?.[mySeat] ?? 0) < undoLimit
  ));
  const canRequestResign = $derived(Boolean(gameState && mySeat && room.phase === "choosing" && !gameState.ended && !gameState.undoRequest && !gameState.resignRequest && !giveawayArmed));

  const undoRequest = $derived(gameState?.undoRequest);
  const undoToMe = $derived(Boolean(mySeat && undoRequest?.toSeat === mySeat));
  const undoFromMe = $derived(Boolean(mySeat && undoRequest?.fromSeat === mySeat));
  const undoFromName = $derived(undoRequest ? occupantDisplay(room.seats[undoRequest.fromSeat]) : "");
  const now = useNow(() => 1000, () => Boolean(undoRequest));
  const undoSecondsLeft = $derived(undoRequest ? Math.max(0, Math.ceil((undoRequest.expiresAt - now.value) / 1000)) : 0);

  const resignRequest = $derived(gameState?.resignRequest);
  const resignToMe = $derived(Boolean(mySeat && resignRequest?.toSeat === mySeat));
  const resignFromMe = $derived(Boolean(mySeat && resignRequest?.fromSeat === mySeat));
  const resignFromName = $derived(resignRequest ? occupantDisplay(room.seats[resignRequest.fromSeat]) : "");

  const board = $derived(gameState?.board || Array.from({ length: 15 }, () => Array.from({ length: 15 }, () => null)));
</script>

<div class="gomoku-panel">
  <div class="gomoku-head">
    <div>
      <h3>●○ 五子棋</h3>
      <p class="hint">
        {#if !room.seats.A || !room.seats.B}等待两个战斗席坐满。
        {:else if drawingFirst}正在随机执黑先手...
        {:else if waitingForReady}双方准备后随机决定谁执黑先手。
        {:else if gameState?.ended}{room.resultText || "对局结束"}
        {:else if giveawayArmed}本手已白给，请选择落点；落子将视为对方棋子。
        {:else}{isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}{/if}
      </p>
    </div>
    {#if gameState}
      <div class="gomoku-turn-card">
        <span>{gameState.blackSeat === gameState.turn ? "⚫ 黑棋" : "⚪ 白棋"}</span>
        {#if room.settings.enableRanked && gameState.rankedDelta}<small>本局排位：黑 {gomokuDeltaText(gameState, blackSeat as SeatKey)} / 白 {whiteSeat ? gomokuDeltaText(gameState, whiteSeat) : ""}</small>{/if}
      </div>
    {/if}
  </div>
  {#if waitingForReady}
    <div class={`gomoku-ready-card ${drawingFirst ? "drawing" : ""}`}>
      <div class="gomoku-draw-animation" aria-hidden="true"><span>⚫</span><span>⚪</span><span>⚫</span></div>
      <div>
        <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
        <p class="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
      </div>
      {#if mySeat && !myReady && !drawingFirst}<button class="primary" disabled={busy} onclick={() => act("gomoku:ready")}>准备</button>{/if}
      {#if mySeat && myReady && !drawingFirst}<button disabled>等待对方</button>{/if}
    </div>
  {/if}
  {#if gameState?.ended && mySeat && room.phase === "result"}
    <button class="primary gomoku-restart-button" disabled={busy} onclick={() => act("gomoku:restart")}>再来一局</button>
  {/if}
  {#if undoRequest}
    <div class={`gomoku-request-card ${undoToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
        <p class="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
      </div>
      {#if undoToMe}
        <div class="gomoku-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("gomoku:undoRespond", { accept: true })}>同意悔棋</button>
          <button class="soft-button" disabled={busy} onclick={() => act("gomoku:undoRespond", { accept: false })}>拒绝</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if resignRequest}
    <div class={`gomoku-request-card ${resignToMe ? "needs-action" : ""}`}>
      <div>
        <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
        <p class="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
      </div>
      {#if resignToMe}
        <div class="gomoku-request-actions">
          <button class="primary" disabled={busy} onclick={() => act("gomoku:resignRespond", { accept: true })}>同意认输</button>
          <button class="soft-button" disabled={busy} onclick={() => act("gomoku:resignRespond", { accept: false })}>拒绝，继续下棋</button>
        </div>
      {/if}
    </div>
  {/if}
  {#if showGiveawayControl || canRequestUndo || canRequestResign}
    <div class="gomoku-risk-actions">
      {#if canRequestUndo}
        <button class="soft-button gomoku-undo-button" disabled={busy} onclick={() => act("gomoku:undoRequest")}>
          申请悔棋<br />本局剩 {undoLimit - (gameState?.undoCount?.[mySeat as SeatKey] ?? 0)} 次
        </button>
      {/if}
      {#if showGiveawayControl}
        <div class={`gomoku-giveaway-card ${giveawayArmed ? "armed" : ""}`}>
          <div>
            <strong>{giveawayArmed ? "本手已白给" : "选择本手白给"}</strong>
            <p class="hint">
              {#if giveawayArmed}
                {gameState?.giveawayForcedByMasterName ? `${gameState.giveawayForcedByMasterName} 强制本手白给，请选择落点。` : "你已选择本手白给，请选择落点。"}
              {:else if !isMyTurn}轮到你时，可在落子前选择本手白给。
              {:else}本手所选位置将会落为对方棋子。{/if}
            </p>
          </div>
          <button class="giveaway-move-button" disabled={busy || !canChooseGiveaway || giveawayArmed} onclick={() => act("gomoku:giveaway")}>
            🫴 {giveawayArmed ? "已白给" : "白给"}
          </button>
        </div>
      {/if}
      {#if canRequestResign}
        <button class="soft-button danger-soft gomoku-resign-button" disabled={busy} onclick={() => act("gomoku:resignRequest")}>申请认输</button>
      {/if}
    </div>
  {/if}
  <GameClockBar {room} state={gameState} moveSeconds={room.settings.gomokuMoveSeconds} gameMinutes={room.settings.gomokuGameMinutes} />
  <div class="gomoku-board" role="grid" aria-label="五子棋棋盘" style={boardTheme}>
    {#each board as row, rowIndex (rowIndex)}
      {#each row as cell, colIndex (colIndex)}
        {@const key = `${rowIndex}-${colIndex}`}
        {@const isLast = Boolean(lastMove && lastMove.row === rowIndex && lastMove.col === colIndex)}
        {@const isWinning = winningKeys.has(key)}
        {@const isPending = isTouchDevice && pendingCell?.row === rowIndex && pendingCell?.col === colIndex}
        <div class="gomoku-cell-wrap">
          <button
            type="button"
            class={`gomoku-cell ${isWinning ? "winning" : ""}`}
            disabled={!isMyTurn || Boolean(cell)}
            onclick={() => placeStone(rowIndex, colIndex)}
            aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列${isPending ? "，再次点击确认落子" : ""}`}
          ></button>
          {#if cell}<span class={`gomoku-stone ${cell} ${isLast ? "last-move" : ""}`}></span>{/if}
          {#if !cell && isPending}<span class="gomoku-crosshair" aria-hidden="true"></span>{/if}
        </div>
      {/each}
    {/each}
  </div>
</div>
