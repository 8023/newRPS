import { useEffect, useMemo, useState, type CSSProperties } from "react";
import { ChessKnight } from "lucide-react";
import type { ChessPiece, PublicPlayer, RoomSnapshot, RoomSettings, SeatKey } from "../shared/types";
import { chessBoardThemes } from "../lib/constants";
import { ask } from "../lib/rpc";
import { GameClockBar, occupantDisplay, useNow } from "./AppViews";

const SIZE = 8;
const FILES = ["a", "b", "c", "d", "e", "f", "g", "h"];
// 双方都用实心字形，颜色交给 CSS，避免空心白子在浅格上几乎看不见。
const PIECE_GLYPH: Record<string, string> = {
  king: "♚",
  queen: "♛",
  rook: "♜",
  bishop: "♝",
  knight: "♞",
  pawn: "♟"
};
const PIECE_LABEL: Record<string, string> = {
  king: "王", queen: "后", rook: "车", bishop: "象", knight: "马", pawn: "兵"
};
const PROMOTE_OPTIONS: Array<{ id: ChessPiece; label: string }> = [
  { id: "queen", label: "后" },
  { id: "rook", label: "车" },
  { id: "bishop", label: "象" },
  { id: "knight", label: "马" }
];

type RGB = { r: number; g: number; b: number };

// 沿用网站的蓝 / 粉 / 暖金语义，但为不同深浅的棋盘格准备足够宽的明度范围。
const CHESS_HINT_PALETTES = {
  move: ["#082f49", "#075985", "#0369a1", "#0ea5e9", "#7dd3fc", "#bae6fd"],
  capture: ["#4c0519", "#9f1239", "#be123c", "#e11d48", "#fda4af", "#fecdd3"],
  last: ["#422006", "#854d0e", "#a16207", "#eab308", "#fde047", "#fef08a"]
} as const;

function parseHexColor(color: string): RGB | null {
  const hex = color.trim().replace(/^#/, "");
  const expanded = hex.length === 3 ? hex.split("").map((part) => `${part}${part}`).join("") : hex;
  if (!/^[\da-f]{6}$/i.test(expanded)) return null;
  return {
    r: Number.parseInt(expanded.slice(0, 2), 16),
    g: Number.parseInt(expanded.slice(2, 4), 16),
    b: Number.parseInt(expanded.slice(4, 6), 16)
  };
}

function relativeLuminance(color: RGB): number {
  const channel = (value: number) => {
    const normalized = value / 255;
    return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
  };
  return 0.2126 * channel(color.r) + 0.7152 * channel(color.g) + 0.0722 * channel(color.b);
}

function contrastRatio(a: RGB, b: RGB): number {
  const lighter = Math.max(relativeLuminance(a), relativeLuminance(b));
  const darker = Math.min(relativeLuminance(a), relativeLuminance(b));
  return (lighter + 0.05) / (darker + 0.05);
}

function highestContrastColor(background: string, palette: readonly string[]): string {
  const backgroundRGB = parseHexColor(background);
  if (!backgroundRGB) return palette[0];
  return palette.reduce((best, candidate) => {
    const bestRGB = parseHexColor(best);
    const candidateRGB = parseHexColor(candidate);
    if (!bestRGB || !candidateRGB) return best;
    return contrastRatio(backgroundRGB, candidateRGB) > contrastRatio(backgroundRGB, bestRGB) ? candidate : best;
  });
}

function chessSquareHintStyle(tone: "light" | "dark", background: string): Record<string, string> {
  return {
    [`--chess-${tone}-move`]: highestContrastColor(background, CHESS_HINT_PALETTES.move),
    [`--chess-${tone}-capture`]: highestContrastColor(background, CHESS_HINT_PALETTES.capture),
    [`--chess-${tone}-last`]: highestContrastColor(background, CHESS_HINT_PALETTES.last)
  };
}

function parseCell(cell: string | null | undefined): { color: "white" | "black"; piece: string } | null {
  if (!cell) return null;
  const [color, piece] = cell.split(":");
  if ((color !== "white" && color !== "black") || !piece) return null;
  return { color, piece };
}

export function chessThemeStyle(themeId?: RoomSettings["chessBoardTheme"]): CSSProperties {
  const theme = chessBoardThemes.find((item) => item.id === themeId) || chessBoardThemes[0];
  return {
    "--chess-board": theme.board,
    "--chess-light": theme.light,
    "--chess-dark": theme.dark,
    "--chess-border": theme.border,
    "--chess-hover": theme.hover,
    "--chess-check": theme.check,
    ...chessSquareHintStyle("light", theme.light),
    ...chessSquareHintStyle("dark", theme.dark)
  } as CSSProperties;
}

function chessDeltaText(state: NonNullable<RoomSnapshot["chess"]>, seat: SeatKey) {
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function chessSideLabel(state: RoomSnapshot["chess"] | undefined, seat: SeatKey): string {
  if (!state) return "随机后显示白/黑";
  return state.whiteSeat === seat ? "⚪ 白棋" : "⚫ 黑棋";
}

export function ChessScore({ room }: { room: RoomSnapshot }) {
  const state = room.chess;
  if (!state) {
    const bothReady = room.ready.A && room.ready.B;
    return <p className="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>;
  }
  return (
    <div className="chess-score-mini">
      <span className="chess-score-side">
        <em>玩家 A</em>
        <b>{chessSideLabel(state, "A")}</b>
      </span>
      <span className="chess-score-side">
        <em>玩家 B</em>
        <b>{chessSideLabel(state, "B")}</b>
      </span>
      {room.settings.enableRanked && state.rankedDelta && (
        <span className="chess-live-rank">A {chessDeltaText(state, "A")} / B {chessDeltaText(state, "B")}</span>
      )}
      <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${chessSideLabel(state, state.turn)}`}</strong>
    </div>
  );
}

export function ChessPanel({ room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void }) {
  const state = room.chess;
  const boardTheme = chessThemeStyle(room.settings.chessBoardTheme);
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<{ row: number; col: number } | null>(null);
  const [pendingPromote, setPendingPromote] = useState<{ from: { row: number; col: number }; to: { row: number; col: number } } | null>(null);
  const mySeat = room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null;
  const myColor = state && mySeat ? (state.whiteSeat === mySeat ? "white" : "black") : null;
  useEffect(() => {
    setSelected(null);
    setPendingPromote(null);
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

  const board = useMemo(
    () => state?.board || Array.from({ length: SIZE }, () => Array.from({ length: SIZE }, () => null as string | null)),
    [state?.board]
  );

  const isMyTurn = Boolean(mySeat && state && state.turn === mySeat && room.phase === "choosing" && !state.ended && !state.resignRequest && !state.undoRequest);
  const dests = useMemo(() => {
    if (!isMyTurn || !selected || !state) return [] as Array<{ row: number; col: number; capture: boolean; promote?: string }>;
    return (state.legalMoves || [])
      .filter((m) => Number(m.from?.row) === selected.row && Number(m.from?.col) === selected.col)
      .map((m) => {
        const row = Number(m.to?.row);
        const col = Number(m.to?.col);
        const fromPiece = parseCell(board[selected.row]?.[selected.col]);
        return {
          row,
          col,
          promote: m.promote,
          capture: Boolean(parseCell(board[row]?.[col])) || (fromPiece?.piece === "pawn" && col !== selected.col)
        };
      });
  }, [isMyTurn, selected, state, board]);
  const destKeys = useMemo(() => new Set(dests.map((p) => `${p.row}-${p.col}`)), [dests]);
  const captureKeys = useMemo(() => new Set(dests.filter((p) => p.capture).map((p) => `${p.row}-${p.col}`)), [dests]);

  function onCellClick(row: number, col: number) {
    if (!isMyTurn || !mySeat || busy) return;
    const key = `${row}-${col}`;
    if (selected && destKeys.has(key)) {
      const options = dests.filter((d) => d.row === row && d.col === col);
      const needsPromote = options.some((d) => d.promote);
      if (needsPromote) {
        setPendingPromote({ from: selected, to: { row, col } });
        return;
      }
      const from = selected;
      setSelected(null);
      act("chess:move", { fromRow: from.row, fromCol: from.col, toRow: row, toCol: col });
      return;
    }
    const piece = parseCell(board[row][col]);
    if (piece && piece.color === myColor) {
      if (selected && selected.row === row && selected.col === col) {
        setSelected(null);
      } else {
        setSelected({ row, col });
      }
      return;
    }
    setSelected(null);
  }

  function confirmPromote(promote: ChessPiece) {
    if (!pendingPromote) return;
    const { from, to } = pendingPromote;
    setPendingPromote(null);
    setSelected(null);
    act("chess:move", { fromRow: from.row, fromCol: from.col, toRow: to.row, toCol: to.col, promote });
  }

  const turnName = state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const waitingForReady = room.phase === "ready" && Boolean(room.seats.A && room.seats.B);
  const drawingFirst = waitingForReady && room.ready.A && room.ready.B;
  const myReady = mySeat ? room.ready[mySeat] : false;
  const undoLimit = room.settings.chessUndoLimit ?? 0;
  const undoRequest = state?.undoRequest;
  const undoToMe = Boolean(mySeat && undoRequest?.toSeat === mySeat);
  const undoFromMe = Boolean(mySeat && undoRequest?.fromSeat === mySeat);
  const undoFromName = undoRequest ? occupantDisplay(room.seats[undoRequest.fromSeat]) : "";
  const now = useNow(1000, Boolean(undoRequest));
  const undoSecondsLeft = undoRequest ? Math.max(0, Math.ceil((undoRequest.expiresAt - now) / 1000)) : 0;
  const myUndoCount = mySeat ? (state?.undoCount?.[mySeat] ?? 0) : 0;
  const canRequestUndo = Boolean(
    state && mySeat && room.phase === "choosing" && !state.ended && !state.undoRequest && !state.resignRequest &&
    state.turn === mySeat && state.moveCount >= 2 && myUndoCount < undoLimit
  );
  const resignRequest = state?.resignRequest;
  const resignToMe = Boolean(mySeat && resignRequest?.toSeat === mySeat);
  const resignFromMe = Boolean(mySeat && resignRequest?.fromSeat === mySeat);
  const resignFromName = resignRequest ? occupantDisplay(room.seats[resignRequest.fromSeat]) : "";
  const canRequestResign = Boolean(state && mySeat && room.phase === "choosing" && !state.ended && !state.undoRequest && !state.resignRequest);
  const kingInCheck = Boolean(state?.inCheck && state.turn);
  const checkSquare = useMemo(() => {
    if (!state?.inCheck) return null;
    const color = state.whiteSeat === state.turn ? "white" : "black";
    for (let r = 0; r < SIZE; r++) {
      for (let c = 0; c < SIZE; c++) {
        const p = parseCell(board[r][c]);
        if (p && p.color === color && p.piece === "king") return { row: r, col: c };
      }
    }
    return null;
  }, [state?.inCheck, state?.turn, state?.whiteSeat, board]);

  let statusHint: string;
  if (!room.seats.A || !room.seats.B) {
    statusHint = "等待两个战斗席坐满。";
  } else if (drawingFirst) {
    statusHint = "正在随机谁执白先手...";
  } else if (waitingForReady) {
    statusHint = "双方准备后随机决定谁执白先走。";
  } else if (state?.ended) {
    statusHint = room.resultText || "对局结束";
  } else if (isMyTurn) {
    statusHint = pendingPromote ? "选择升变成什么棋子。" : selected ? "点击目标格走子。" : "点击想要移动的棋子。";
  } else if (mySeat) {
    statusHint = `轮到对方（ ${turnName} ）走子。`;
  } else {
    statusHint = `轮到 ${turnName} 走子。`;
  }

  const blackSeat: SeatKey = state ? (state.whiteSeat === "A" ? "B" : "A") : "B";

  return (
    <div className="chess-panel">
      <div className="chess-head">
        <div>
          <h3 className="chess-title"><ChessKnight size={19} strokeWidth={2.1} aria-hidden="true" />国际象棋</h3>
          <p className="hint">{statusHint}{kingInCheck && !state?.ended ? (isMyTurn ? " 你已被将军。" : mySeat ? " 对方已被将军。" : " 当前行棋方已被将军。") : ""}</p>
        </div>
        {state && (
          <div className="chess-turn-card">
            <span>轮到 {chessSideLabel(state, state.turn)}</span>
            {room.settings.enableRanked && state.rankedDelta && (
              <small>本局排位：A {chessDeltaText(state, "A")} / B {chessDeltaText(state, "B")}</small>
            )}
          </div>
        )}
      </div>
      {waitingForReady && (
        <div className={`chess-ready-card ${drawingFirst ? "drawing" : ""}`}>
          <div className="chess-draw-animation" aria-hidden="true">
            <span>♔</span>
            <span>♕</span>
            <span>♞</span>
          </div>
          <div>
            <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
            <p className="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
          </div>
          {mySeat && !myReady && !drawingFirst && <button className="primary" disabled={busy} onClick={() => act("chess:ready")}>准备</button>}
          {mySeat && myReady && !drawingFirst && <button disabled>等待对方</button>}
        </div>
      )}
      {state?.ended && mySeat && room.phase === "result" && (
        <button className="primary chess-restart-button" disabled={busy} onClick={() => act("chess:restart")}>再来一局</button>
      )}
      {undoRequest && (
        <div className={`chess-request-card ${undoToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{undoFromMe ? "已申请悔棋" : `${undoFromName} 申请悔棋`}</strong>
            <p className="hint">{undoFromMe ? `等待对方确认，${undoSecondsLeft} 秒后自动拒绝。` : `同意后棋局回退 2 手，${undoSecondsLeft} 秒后自动拒绝。`}</p>
          </div>
          {undoToMe && (
            <div className="chess-request-actions">
              <button className="primary" disabled={busy} onClick={() => act("chess:undoRespond", { accept: true })}>同意悔棋</button>
              <button className="soft-button" disabled={busy} onClick={() => act("chess:undoRespond", { accept: false })}>拒绝</button>
            </div>
          )}
        </div>
      )}
      {resignRequest && (
        <div className={`chess-request-card ${resignToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
            <p className="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
          </div>
          {resignToMe && (
            <div className="chess-request-actions">
              <button className="primary" disabled={busy} onClick={() => act("chess:resignRespond", { accept: true })}>同意认输</button>
              <button className="soft-button" disabled={busy} onClick={() => act("chess:resignRespond", { accept: false })}>拒绝，继续下棋</button>
            </div>
          )}
        </div>
      )}
      {(canRequestUndo || canRequestResign) && (
        <div className="chess-risk-actions">
          {canRequestUndo && (
            <button className="soft-button" disabled={busy} onClick={() => act("chess:undoRequest")}>申请悔棋<br />本局剩 {undoLimit - myUndoCount} 次</button>
          )}
          {canRequestResign && <button className="soft-button danger-soft" disabled={busy} onClick={() => act("chess:resignRequest")}>申请认输</button>}
        </div>
      )}
      {pendingPromote && (
        <div className="chess-request-card needs-action">
          <div>
            <strong>兵升变</strong>
            <p className="hint">选择把兵变成哪种棋子。</p>
          </div>
          <div className="chess-request-actions">
            {PROMOTE_OPTIONS.map((opt) => (
              <button key={opt.id} className="primary" disabled={busy} onClick={() => confirmPromote(opt.id)}>{opt.label}</button>
            ))}
            <button className="soft-button" disabled={busy} onClick={() => setPendingPromote(null)}>取消</button>
          </div>
        </div>
      )}
      <GameClockBar
        room={room}
        state={state ? { turn: state.turn, blackSeat, ended: state.ended, moveDeadlineAt: state.moveDeadlineAt, clockDeadlineAt: state.clockDeadlineAt, clockRemaining: state.clockRemaining } : undefined}
        moveSeconds={room.settings.chessMoveSeconds}
        gameMinutes={room.settings.chessGameMinutes}
        labels={{ primary: "⚫ 黑", secondary: "⚪ 白" }}
      />
      <div className="chess-board-wrap" style={boardTheme}>
        <div className="chess-ranks" aria-hidden="true">
          {Array.from({ length: SIZE }, (_, i) => (
            <span key={i}>{SIZE - i}</span>
          ))}
        </div>
        <div className="chess-board" role="grid" aria-label="国际象棋棋盘">
          {Array.from({ length: SIZE }, (_, row) => (
            Array.from({ length: SIZE }, (_, col) => {
              const key = `${row}-${col}`;
              const cell = board[row]?.[col];
              const piece = parseCell(cell);
              const isLight = (row + col) % 2 === 0;
              const isSelected = Boolean(selected && selected.row === row && selected.col === col);
              const isDest = destKeys.has(key);
              const isCapture = captureKeys.has(key);
              const isLastFrom = Boolean(state?.lastFrom && state.lastFrom.row === row && state.lastFrom.col === col);
              const isLastTo = Boolean(state?.lastTo && state.lastTo.row === row && state.lastTo.col === col);
              const isCheck = Boolean(checkSquare && checkSquare.row === row && checkSquare.col === col);
              return (
                <button
                  type="button"
                  key={key}
                  className={[
                    "chess-cell",
                    isLight ? "light" : "dark",
                    isSelected ? "selected" : "",
                    isDest ? "legal" : "",
                    isCapture ? "capture" : "",
                    isLastFrom ? "last-from" : "",
                    isLastTo ? "last-to" : "",
                    isCheck ? "in-check" : ""
                  ].filter(Boolean).join(" ")}
                  disabled={!isMyTurn}
                  onClick={() => onCellClick(row, col)}
                  aria-label={`${FILES[col]}${8 - row}${piece ? `，${piece.color === "white" ? "白" : "黑"}${PIECE_LABEL[piece.piece] || piece.piece}` : ""}`}
                >
                  {piece && (
                    <span className={`chess-piece color-${piece.color}`} aria-hidden="true">
                      {PIECE_GLYPH[piece.piece] || "?"}
                    </span>
                  )}
                </button>
              );
            })
          ))}
        </div>
        <div className="chess-files" aria-hidden="true">
          {Array.from({ length: SIZE }, (_, i) => (
            <span key={i}>{FILES[i]}</span>
          ))}
        </div>
      </div>
      <div className="chess-legend">
        <span>{chessSideLabel(state, "A")}：{occupantDisplay(room.seats.A)}</span>
        <span>{chessSideLabel(state, "B")}：{occupantDisplay(room.seats.B)}</span>
        <span>{mySeat ? `你在战斗席 ${mySeat}（${chessSideLabel(state, mySeat)}）` : "你正在观战"}</span>
      </div>
      <div className="chess-rules-hint hint">
        <p>白方先走。王车易位、吃过路兵、兵到底线升变都按常规规则。将死对方的王即胜；逼和、子力不足、五十步或三次重复局面算平局。</p>
      </div>
    </div>
  );
}
