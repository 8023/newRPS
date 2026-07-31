import { useEffect, useMemo, useState, type CSSProperties } from "react";
import type { JungleAnimal, PublicPlayer, RoomSnapshot, RoomSettings, SeatKey } from "../shared/types";
import { jungleBoardThemes } from "../lib/constants";
import { ask } from "../lib/rpc";
import { GameClockBar, occupantDisplay } from "./AppViews";

const ROWS = 9;
const COLS = 7;

const ANIMAL_LABEL: Record<JungleAnimal, string> = {
  rat: "鼠", cat: "猫", dog: "狗", wolf: "狼",
  leopard: "豹", tiger: "虎", lion: "狮", elephant: "象"
};
const ANIMAL_EMOJI: Record<JungleAnimal, string> = {
  rat: "🐀", cat: "🐈", dog: "🐕", wolf: "🐺",
  leopard: "🐆", tiger: "🐅", lion: "🦁", elephant: "🐘"
};
const ANIMAL_RANK: Record<JungleAnimal, number> = {
  rat: 1, cat: 2, dog: 3, wolf: 4, leopard: 5, tiger: 6, lion: 7, elephant: 8
};

function parseCell(cell: string | null | undefined): { side: SeatKey; animal: JungleAnimal } | null {
  if (!cell) return null;
  const [side, animal] = cell.split(":");
  if ((side !== "A" && side !== "B") || !(animal in ANIMAL_RANK)) return null;
  return { side: side as SeatKey, animal: animal as JungleAnimal };
}

function isWater(row: number, col: number) {
  if (row < 3 || row > 5) return false;
  return (col >= 1 && col <= 2) || (col >= 4 && col <= 5);
}
function isDen(row: number, col: number, side: SeatKey) {
  return col === 3 && (side === "A" ? row === 8 : row === 0);
}
function isTrap(row: number, col: number, side: SeatKey) {
  if (side === "A") return (row === 8 && (col === 2 || col === 4)) || (row === 7 && col === 3);
  return (row === 0 && (col === 2 || col === 4)) || (row === 1 && col === 3);
}
function trapOwner(row: number, col: number): SeatKey | null {
  if (isTrap(row, col, "A")) return "A";
  if (isTrap(row, col, "B")) return "B";
  return null;
}

function canCapture(attackerSide: SeatKey, attacker: JungleAnimal, fromR: number, fromC: number, toR: number, toC: number, defender: JungleAnimal) {
  if (trapOwner(toR, toC) === attackerSide) return true;
  if (attacker === "rat" && defender === "elephant") {
    if (isWater(fromR, fromC) && !isWater(toR, toC)) return false;
    return true;
  }
  if (attacker === "elephant" && defender === "rat") return false;
  return ANIMAL_RANK[attacker] >= ANIMAL_RANK[defender];
}

function jumpDest(board: Array<Array<string | null>>, fromR: number, fromC: number, dr: number, dc: number) {
  let r = fromR + dr;
  let c = fromC + dc;
  if (r < 0 || r >= ROWS || c < 0 || c >= COLS || !isWater(r, c)) return null;
  while (r >= 0 && r < ROWS && c >= 0 && c < COLS && isWater(r, c)) {
    if (board[r][c]) return null;
    r += dr;
    c += dc;
  }
  if (r < 0 || r >= ROWS || c < 0 || c >= COLS) return null;
  return { row: r, col: c };
}

function canMoveTo(board: Array<Array<string | null>>, seat: SeatKey, animal: JungleAnimal, fromR: number, fromC: number, toR: number, toC: number, isJump: boolean) {
  if (toR < 0 || toR >= ROWS || toC < 0 || toC >= COLS) return false;
  if (isDen(toR, toC, seat)) return false;
  if (!isJump && isWater(toR, toC) && animal !== "rat") return false;
  const target = board[toR][toC];
  if (!target) return true;
  const def = parseCell(target);
  if (!def || def.side === seat) return false;
  return canCapture(seat, animal, fromR, fromC, toR, toC, def.animal);
}

function legalDests(board: Array<Array<string | null>>, seat: SeatKey, fromR: number, fromC: number) {
  const src = parseCell(board[fromR]?.[fromC]);
  if (!src || src.side !== seat) return [] as Array<{ row: number; col: number }>;
  const dests: Array<{ row: number; col: number }> = [];
  for (const [dr, dc] of [[-1, 0], [1, 0], [0, -1], [0, 1]] as const) {
    const r = fromR + dr;
    const c = fromC + dc;
    if (canMoveTo(board, seat, src.animal, fromR, fromC, r, c, false)) dests.push({ row: r, col: c });
  }
  if (src.animal === "lion" || src.animal === "tiger") {
    for (const [dr, dc] of [[-1, 0], [1, 0], [0, -1], [0, 1]] as const) {
      const land = jumpDest(board, fromR, fromC, dr, dc);
      if (land && canMoveTo(board, seat, src.animal, fromR, fromC, land.row, land.col, true)) dests.push(land);
    }
  }
  return dests;
}

export function jungleThemeStyle(themeId?: RoomSettings["jungleBoardTheme"]): CSSProperties {
  const theme = jungleBoardThemes.find((item) => item.id === themeId) || jungleBoardThemes[0];
  return {
    "--jungle-board": theme.board,
    "--jungle-land": theme.land,
    "--jungle-water": theme.water,
    "--jungle-trap": theme.trap,
    "--jungle-den": theme.den,
    "--jungle-border": theme.border,
    "--jungle-a-piece": theme.aPiece,
    "--jungle-b-piece": theme.bPiece,
    "--jungle-a-text": theme.aText,
    "--jungle-b-text": theme.bText,
    "--jungle-hover": theme.hover
  } as CSSProperties;
}

function jungleDeltaText(state: NonNullable<RoomSnapshot["jungle"]>, seat: SeatKey) {
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

/** 阵地 + 棋色：A 下方白棋、B 上方黑棋（棋子 CSS 对应浅/深底）。 */
export function jungleSideLabel(seat: SeatKey): string {
  return seat === "A" ? "下方⬇ 白棋⚪" : "上方⬆ 黑棋⚫";
}

/** 对战比分区简写（完整文案在左右座位卡）：⬇ ⚪ / ⬆ ⚫ */
export function jungleSideBrief(seat: SeatKey): string {
  return seat === "A" ? "⬇ ⚪" : "⬆ ⚫";
}

export function JungleScore({ room }: { room: RoomSnapshot }) {
  const state = room.jungle;
  if (!state) {
    const bothReady = room.ready.A && room.ready.B;
    return <p className="settlement-placeholder">{bothReady ? "正在随机先手" : "等待准备"}</p>;
  }
  return (
    <div className="jungle-score-mini">
      <span className="jungle-score-side">
        <em>玩家 A</em>
        <b>{jungleSideBrief("A")}</b>
      </span>
      <span className="jungle-score-side">
        <em>玩家 B</em>
        <b>{jungleSideBrief("B")}</b>
      </span>
      {room.settings.enableRanked && state.rankedDelta && (
        <span className="jungle-live-rank">白 {jungleDeltaText(state, "A")} / 黑 {jungleDeltaText(state, "B")}</span>
      )}
      <strong>{state.ended ? room.resultText || "对局结束" : `轮到 ${jungleSideLabel(state.turn)}`}</strong>
    </div>
  );
}

export function JunglePanel({ room, me, onError }: { room: RoomSnapshot; me: PublicPlayer; onError: (message: string) => void }) {
  const state = room.jungle;
  const boardTheme = jungleThemeStyle(room.settings.jungleBoardTheme);
  const [busy, setBusy] = useState(false);
  const [selected, setSelected] = useState<{ row: number; col: number } | null>(null);
  const mySeat = room.seats.A?.id === me.id ? "A" : room.seats.B?.id === me.id ? "B" : null;

  useEffect(() => {
    setSelected(null);
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
    () => state?.board || Array.from({ length: ROWS }, () => Array.from({ length: COLS }, () => null as string | null)),
    [state?.board]
  );

  const isMyTurn = Boolean(mySeat && state && state.turn === mySeat && room.phase === "choosing" && !state.ended && !state.resignRequest);
  const destKeys = useMemo(() => {
    if (!isMyTurn || !selected || !mySeat) return new Set<string>();
    return new Set(legalDests(board, mySeat, selected.row, selected.col).map((p) => `${p.row}-${p.col}`));
  }, [board, isMyTurn, selected, mySeat]);

  function onCellClick(row: number, col: number) {
    if (!isMyTurn || !mySeat || busy) return;
    const key = `${row}-${col}`;
    if (selected && destKeys.has(key)) {
      const from = selected;
      setSelected(null);
      act("jungle:move", { fromRow: from.row, fromCol: from.col, toRow: row, toCol: col });
      return;
    }
    const piece = parseCell(board[row][col]);
    if (piece && piece.side === mySeat) {
      if (selected && selected.row === row && selected.col === col) {
        setSelected(null);
      } else {
        setSelected({ row, col });
      }
      return;
    }
    setSelected(null);
  }

  const turnName = state?.turn === "A" ? occupantDisplay(room.seats.A) : occupantDisplay(room.seats.B);
  const waitingForReady = room.phase === "ready" && Boolean(room.seats.A && room.seats.B);
  const drawingFirst = waitingForReady && room.ready.A && room.ready.B;
  const myReady = mySeat ? room.ready[mySeat] : false;
  const resignRequest = state?.resignRequest;
  const resignToMe = Boolean(mySeat && resignRequest?.toSeat === mySeat);
  const resignFromMe = Boolean(mySeat && resignRequest?.fromSeat === mySeat);
  const resignFromName = resignRequest ? occupantDisplay(room.seats[resignRequest.fromSeat]) : "";
  const canRequestResign = Boolean(state && mySeat && room.phase === "choosing" && !state.ended && !state.resignRequest);

  let statusHint: string;
  if (!room.seats.A || !room.seats.B) {
    statusHint = "等待两个战斗席坐满。";
  } else if (drawingFirst) {
    statusHint = "正在随机先手...";
  } else if (waitingForReady) {
    statusHint = "双方准备后随机决定谁先手；A 执下方白棋⚪，B 执上方黑棋⚫。";
  } else if (state?.ended) {
    statusHint = room.resultText || "对局结束";
  } else if (isMyTurn) {
    statusHint = selected ? "点击目标格走子。" : "点击想要移动的棋子。";
  } else if (mySeat) {
    statusHint = `轮到对方（ ${turnName} ）走子。`;
  } else {
    statusHint = `轮到 ${turnName} 走子。`;
  }

  return (
    <div className="jungle-panel">
      <div className="jungle-head">
        <div>
          <h3>🦁 斗兽棋</h3>
          <p className="hint">{statusHint}</p>
        </div>
        {state && (
          <div className="jungle-turn-card">
            <span>轮到 {jungleSideLabel(state.turn)}</span>
            {room.settings.enableRanked && state.rankedDelta && (
              <small>本局排位：白 {jungleDeltaText(state, "A")} / 黑 {jungleDeltaText(state, "B")}</small>
            )}
          </div>
        )}
      </div>
      {waitingForReady && (
        <div className={`jungle-ready-card ${drawingFirst ? "drawing" : ""}`}>
          <div className="jungle-draw-animation" aria-hidden="true">
            <span>🦁</span>
            <span>🐘</span>
            <span>🐯</span>
          </div>
          <div>
            <strong>{drawingFirst ? "抽签中..." : myReady ? "你已准备" : "准备开始"}</strong>
            <p className="hint">A：{room.ready.A ? "已准备" : "未准备"} · B：{room.ready.B ? "已准备" : "未准备"}</p>
          </div>
          {mySeat && !myReady && !drawingFirst && <button className="primary" disabled={busy} onClick={() => act("jungle:ready")}>准备</button>}
          {mySeat && myReady && !drawingFirst && <button disabled>等待对方</button>}
        </div>
      )}
      {state?.ended && mySeat && room.phase === "result" && (
        <button className="primary jungle-restart-button" disabled={busy} onClick={() => act("jungle:restart")}>再来一局</button>
      )}
      {resignRequest && (
        <div className={`jungle-request-card ${resignToMe ? "needs-action" : ""}`}>
          <div>
            <strong>{resignFromMe ? "已申请认输" : `${resignFromName} 申请认输`}</strong>
            <p className="hint">{resignFromMe ? "等待对方确认；对局状态会保持不变。" : "你可以同意结束本局，或拒绝后继续下棋。"}</p>
          </div>
          {resignToMe && (
            <div className="jungle-request-actions">
              <button className="primary" disabled={busy} onClick={() => act("jungle:resignRespond", { accept: true })}>同意认输</button>
              <button className="soft-button" disabled={busy} onClick={() => act("jungle:resignRespond", { accept: false })}>拒绝，继续下棋</button>
            </div>
          )}
        </div>
      )}
      {canRequestResign && (
        <div className="jungle-risk-actions">
          <button className="soft-button danger-soft" disabled={busy} onClick={() => act("jungle:resignRequest")}>申请认输</button>
        </div>
      )}
      <GameClockBar
        room={room}
        state={state ? { turn: state.turn, blackSeat: "A", ended: state.ended, moveDeadlineAt: state.moveDeadlineAt, clockDeadlineAt: state.clockDeadlineAt, clockRemaining: state.clockRemaining } : undefined}
        moveSeconds={room.settings.jungleMoveSeconds}
        gameMinutes={room.settings.jungleGameMinutes}
        labels={{ primary: "下方⬇ 白", secondary: "上方⬆ 黑" }}
      />
      <div className="jungle-board" role="grid" aria-label="斗兽棋棋盘" style={boardTheme}>
        {board.map((row, rowIndex) => row.map((cell, colIndex) => {
          const key = `${rowIndex}-${colIndex}`;
          const piece = parseCell(cell);
          const water = isWater(rowIndex, colIndex);
          const denA = isDen(rowIndex, colIndex, "A");
          const denB = isDen(rowIndex, colIndex, "B");
          const trapA = isTrap(rowIndex, colIndex, "A");
          const trapB = isTrap(rowIndex, colIndex, "B");
          const isSelected = Boolean(selected && selected.row === rowIndex && selected.col === colIndex);
          const isDest = destKeys.has(key);
          const isLastFrom = Boolean(state?.lastFrom && state.lastFrom.row === rowIndex && state.lastFrom.col === colIndex);
          const isLastTo = Boolean(state?.lastTo && state.lastTo.row === rowIndex && state.lastTo.col === colIndex);
          const terrain = water ? "water" : denA || denB ? "den" : trapA || trapB ? "trap" : "land";
          return (
            <button
              type="button"
              key={key}
              className={[
                "jungle-cell",
                terrain,
                isSelected ? "selected" : "",
                isDest ? "legal" : "",
                isLastFrom ? "last-from" : "",
                isLastTo ? "last-to" : ""
              ].filter(Boolean).join(" ")}
              disabled={!isMyTurn}
              onClick={() => onCellClick(rowIndex, colIndex)}
              aria-label={`第 ${rowIndex + 1} 行第 ${colIndex + 1} 列${piece ? `，${piece.side}方${ANIMAL_LABEL[piece.animal]}` : ""}`}
            >
              {(denA || denB) && <span className="jungle-terrain-mark" aria-hidden="true">穴</span>}
              {(trapA || trapB) && !piece && <span className="jungle-terrain-mark trap-mark" aria-hidden="true">陷</span>}
              {water && !piece && <span className="jungle-terrain-mark water-mark" aria-hidden="true">～</span>}
              {piece && (
                <span className={`jungle-piece side-${piece.side}`} title={ANIMAL_LABEL[piece.animal]} aria-hidden="true">
                  {ANIMAL_EMOJI[piece.animal]}
                </span>
              )}
            </button>
          );
        }))}
      </div>
      <div className="jungle-legend">
        <span>{jungleSideLabel("A")}：{occupantDisplay(room.seats.A)}</span>
        <span>{jungleSideLabel("B")}：{occupantDisplay(room.seats.B)}</span>
        <span>{mySeat ? `你在战斗席 ${mySeat}（${jungleSideLabel(mySeat)}）` : "你正在观战"}</span>
      </div>
      <div className="jungle-rules-hint hint">
        <p className="jungle-rank-line" aria-label="等级：象大于狮大于虎大于豹大于狼大于狗大于猫大于鼠">
          <span className="jungle-rank-label">等级</span>
          {(["elephant", "lion", "tiger", "leopard", "wolf", "dog", "cat", "rat"] as const).map((animal, index) => (
            <span key={animal} className="jungle-rank-item">
              {index > 0 && <span className="jungle-rank-gt" aria-hidden="true">&gt;</span>}
              <span className="jungle-rank-icon" title={ANIMAL_LABEL[animal]}>{ANIMAL_EMOJI[animal]}</span>
            </span>
          ))}
        </p>
        <p>
          🐀 可吃 🐘（不能从水中上岸吃），🐘 不能吃 🐀；仅 🐀 可入水；🦁🐅 可跳河（河中有 🐀 则不能跳）；先入对方兽穴或令对方无子可走者胜。
        </p>
      </div>
    </div>
  );
}
