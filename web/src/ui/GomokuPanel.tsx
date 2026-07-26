import { useEffect, useState, type CSSProperties } from "react";
import type { PublicPlayer, RoomSnapshot, RoomSettings, SeatKey } from "../shared/types";
import { gomokuBoardThemes } from "../lib/constants";
import { ask } from "../lib/rpc";
import { GameClockBar, occupantDisplay } from "./AppViews";

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

export function gomokuThemeStyle(themeId?: RoomSettings["gomokuBoardTheme"]): CSSProperties {
  const theme = gomokuBoardThemes.find((item) => item.id === themeId) || gomokuBoardThemes.find((item) => item.id === "wood") || gomokuBoardThemes[0];
  return {
    "--gomoku-board": theme.board,
    "--gomoku-cell": theme.cell,
    "--gomoku-line": theme.line,
    "--gomoku-hover": theme.hover,
    "--gomoku-border": theme.border,
    "--gomoku-black-disc": theme.blackDisc,
    "--gomoku-white-disc": theme.whiteDisc,
    "--gomoku-black-ring": theme.blackRing,
    "--gomoku-white-ring": theme.whiteRing
  } as CSSProperties;
}

function gomokuDeltaText(state: NonNullable<RoomSnapshot["gomoku"]>, seat: SeatKey) {
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function GomokuScore({ room }: { room: RoomSnapshot }) {
  const state = room.gomoku;
  if (!state) {
    const bothReady = room.ready.A && room.ready.B;
    return <p className="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>;
  }
  const whiteSeat: SeatKey = state.blackSeat === "A" ? "B" : "A";
  return (
    <div className="gomoku-score-mini">
      <span>⚫ {state.blackSeat}</span>
      <span>⚪ {whiteSeat}</span>
      {room.settings.enableRanked && state.rankedDelta && <span className="gomoku-live-rank">黑 {gomokuDeltaText(state, state.blackSeat)} / 白 {gomokuDeltaText(state, whiteSeat)}</span>}
      <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${state.blackSeat === state.turn ? "黑" : "白"}`}</strong>
    </div>
  );
}

export function GomokuPanel({ room, me, now, onError }: { room: RoomSnapshot; me: PublicPlayer; now: number; onError: (message: string) => void }) {
  const state = room.gomoku;
  const boardTheme = gomokuThemeStyle(room.settings.gomokuBoardTheme);
  const [busy, setBusy] = useState(false);
  const [pendingCell, setPendingCell] = useState<{ row: number; col: number } | null>(null);
  const mySeat = room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null;

  useEffect(() => {
    setPendingCell(null);
  }, [state?.moveCount, room.phase]);

  async function act(event: string, payload: unknown = {}) {
    if (busy) return;
    setBusy(true);
    try {
      await ask(event, payload);
    } catch (error) {
      onError(error instanceof Error ? error.message : "操作失败");
    } finally {
      setBusy(false);
    }
  }

  function placeStone(rowIndex: number, colIndex: number) {
    if (isTouchDevice) {
      if (pendingCell && pendingCell.row === rowIndex && pendingCell.col === colIndex) {
        setPendingCell(null);
        act("gomoku:move", { row: rowIndex, col: colIndex });
      } else {
        setPendingCell({ row: rowIndex, col: colIndex });
      }
      return;
    }
    act("gomoku:move", { row: rowIndex, col: colIndex });
  }

  const isMyTurn = Boolean(mySeat && state && state.turn === mySeat && room.phase === "choosing" && !state.ended && !state.undoRequest && !state.resignRequest);
  const giveawayArmed = Boolean(mySeat && state?.giveawaySeat === mySeat);
  const showGiveawayControl = Boolean(mySeat && state && me.giveawayEnabled && room.seats.A && room.seats.B && room.phase === "choosing" && !state.ended);
  const canChooseGiveaway = Boolean(showGiveawayControl && isMyTurn);
  const turnName = state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const waitingForReady = room.phase === "ready" && Boolean(room.seats.A && room.seats.B);
  const drawingFirst = waitingForReady && room.ready.A && room.ready.B;
  const myReady = mySeat ? room.ready[mySeat] : false;
  const blackSeat = state?.blackSeat;
  const whiteSeat = blackSeat ? (blackSeat === "A" ? "B" : "A") : null;
  const lastMove = state?.moves.length ? state.moves[state.moves.length - 1] : null;
  const winningKeys = new Set((state?.winningLine || []).map((p) => `${p.row}-${p.col}`));

  // 缺省按 0（禁止悔棋）而不是 3：wire.ts/pbconv 的 fillRoomSettingsDefaults 已经把
  // "键缺失"统一按真值 0 补齐（建房时必然是 0/1/3/10 之一，缺失只可能是零值被 protojson
  // 省略），这里若退回 3 会跟服务端 game_gomoku.go 的实际校验（用的是真实值，不做兜底）
  // 不一致——显示"还能悔棋"，点了却被服务端拒绝。
  const undoLimit = room.settings.gomokuUndoLimit ?? 0;
  const canRequestUndo = Boolean(
    state && mySeat && room.phase === "choosing" && !state.ended && !state.undoRequest && !state.resignRequest && !giveawayArmed &&
    state.turn === mySeat && state.moveCount >= 2 && (state.undoCount?.[mySeat] ?? 0) < undoLimit
  );
  const canRequestResign = Boolean(state && mySeat && room.phase === "choosing" && !state.ended && !state.undoRequest && !state.resignRequest && !giveawayArmed);

  const undoRequest = state?.undoRequest;
  const undoToMe = Boolean(mySeat && undoRequest?.toSeat === mySeat);
  const undoFromMe = Boolean(mySeat && undoRequest?.fromSeat === mySeat);
  const undoFromName = undoRequest ? occupantDisplay(room.seats[undoRequest.fromSeat]) : "";
  const undoSecondsLeft = undoRequest ? Math.max(0, Math.ceil((undoRequest.expiresAt - now) / 1000)) : 0;

  const resignRequest = state?.resignRequest;
  const resignToMe = Boolean(mySeat && resignRequest?.toSeat === mySeat);
  const resignFromMe = Boolean(mySeat && resignRequest?.fromSeat === mySeat);
  const resignFromName = resignRequest ? occupantDisplay(room.seats[resignRequest.fromSeat]) : "";

  return (
    <div className="gomoku-panel">
      <div className="gomoku-head">
        <div>
          <h3>●○ 五子棋</h3>
          <p className="hint">
            {!room.seats.A || !room.seats.B
              ? "等待两个战斗席坐满。"
              : drawingFirst
                ? "正在随机执黑先手..."
                : waitingForReady
                  ? "双方准备后随机决定谁执黑先手。"
                  : state?.ended
                    ? room.resultText || "对局结束"
                    : giveawayArmed ? "本手已白给，请选择落点；落子将视为对方棋子。" : isMyTurn ? "轮到你落子。" : `轮到 ${turnName} 落子。`}
          </p>
        </div>
        {state && (
          <div className="gomoku-turn-card">
            <span>{state.blackSeat === state.turn ? "⚫ 黑棋" : "⚪ 白棋"}</span>
            {room.settings.enableRanked && state.rankedDelta && <small>本局排位：黑 {gomokuDeltaText(state, blackSeat as SeatKey)} / 白 {whiteSeat ? gomokuDeltaText(state, whiteSeat) : ""}</small>}
          </div>
        )}
      </div>
      {waitingForReady && (
        <div className={`gomoku-ready-card ${drawingFirst ? "drawing" : ""}`}>
          <div className="gomoku-draw-animation" aria-hidden="true">
            <span>⚫</span>
            <span>⚪</span>
            <span>⚫</span>
          </div>
          <div>
            <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
            <p className="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
          </div>
          {mySeat && !myReady && !drawingFirst && <button className="primary" disabled={busy} onClick={() => act("gomoku:ready")}>准备</button>}
          {mySeat && myReady && !drawingFirst && <button disabled>等待对方</button>}
        </div>
      )}
      {state?.ended && mySeat && room.phase === "result" && <button className="primary gomoku-restart-button" disabled={busy} onClick={() => act("gomoku:restart")}>再来一局</button>}
      {undoRequest && (
        <div className={`gomoku-request-card ${undoToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
            <p className="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
          </div>
          {undoToMe && (
            <div className="gomoku-request-actions">
              <button className="primary" disabled={busy} onClick={() => act("gomoku:undoRespond", { accept: true })}>同意悔棋</button>
              <button className="soft-button" disabled={busy} onClick={() => act("gomoku:undoRespond", { accept: false })}>拒绝</button>
            </div>
          )}
        </div>
      )}
      {resignRequest && (
        <div className={`gomoku-request-card ${resignToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
            <p className="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
          </div>
          {resignToMe && (
            <div className="gomoku-request-actions">
              <button className="primary" disabled={busy} onClick={() => act("gomoku:resignRespond", { accept: true })}>同意认输</button>
              <button className="soft-button" disabled={busy} onClick={() => act("gomoku:resignRespond", { accept: false })}>拒绝，继续下棋</button>
            </div>
          )}
        </div>
      )}
      {(showGiveawayControl || canRequestUndo || canRequestResign) && (
        <div className="gomoku-risk-actions">
          {canRequestUndo && (
            <button className="soft-button gomoku-undo-button" disabled={busy} onClick={() => act("gomoku:undoRequest")}>
              申请悔棋<br />本局剩 {undoLimit - (state?.undoCount?.[mySeat as SeatKey] ?? 0)} 次
            </button>
          )}
          {showGiveawayControl && (
            <div className={`gomoku-giveaway-card ${giveawayArmed ? "armed" : ""}`}>
              <div>
                <strong>{giveawayArmed ? "本手已白给" : "选择本手白给"}</strong>
                <p className="hint">
                  {giveawayArmed
                    ? state?.giveawayForcedByMasterName
                      ? `${state.giveawayForcedByMasterName} 强制本手白给，请选择落点。`
                      : "你已选择本手白给，请选择落点。"
                    : !isMyTurn
                      ? "轮到你时，可在落子前选择本手白给。"
                      : "本手所选位置将会落为对方棋子。"}
                </p>
              </div>
              <button className="giveaway-move-button" disabled={busy || !canChooseGiveaway || giveawayArmed} onClick={() => act("gomoku:giveaway")}>
                🫴 {giveawayArmed ? "已白给" : "白给"}
              </button>
            </div>
          )}
          {canRequestResign && (
            <button className="soft-button danger-soft gomoku-resign-button" disabled={busy} onClick={() => act("gomoku:resignRequest")}>
              申请认输
            </button>
          )}
        </div>
      )}
      <GameClockBar room={room} state={state} moveSeconds={room.settings.gomokuMoveSeconds} gameMinutes={room.settings.gomokuGameMinutes} now={now} />
      <div className="gomoku-board" role="grid" aria-label="五子棋棋盘" style={boardTheme}>
        {(state?.board || Array.from({ length: 15 }, () => Array.from({ length: 15 }, () => null))).map((row, rowIndex) => row.map((cell, colIndex) => {
          const key = `${rowIndex}-${colIndex}`;
          const isLast = Boolean(lastMove && lastMove.row === rowIndex && lastMove.col === colIndex);
          const isWinning = winningKeys.has(key);
          const isPending = isTouchDevice && pendingCell?.row === rowIndex && pendingCell?.col === colIndex;
          return (
            // 棋子挪到 button 外层作为兄弟节点（同 othello-cell-wrap）：button 的
            // disabled 变暗（opacity:.6）只应影响格子自身，不应连带压暗已经落下的棋子。
            <div className="gomoku-cell-wrap" key={key}>
              <button
                type="button"
                className={`gomoku-cell ${isWinning ? "winning" : ""}`}
                disabled={!isMyTurn || Boolean(cell)}
                onClick={() => placeStone(rowIndex, colIndex)}
                aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列${isPending ? "，再次点击确认落子" : ""}`}
              />
              {cell && <span className={`gomoku-stone ${cell} ${isLast ? "last-move" : ""}`} />}
              {!cell && isPending && <span className="gomoku-crosshair" aria-hidden="true" />}
            </div>
          );
        }))}
      </div>
      <div className="gomoku-legend">
        <span>⚫ 黑棋：{blackSeat ? occupantDisplay(room.seats[blackSeat]) : "准备后随机"}</span>
        <span>⚪ 白棋：{whiteSeat ? occupantDisplay(room.seats[whiteSeat]) : "准备后随机"}</span>
        <span>{mySeat ? `你在战斗席 ${mySeat}` : "你正在观战"}</span>
      </div>
    </div>
  );
}
