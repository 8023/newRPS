// 斗兽棋前端合法走子计算（仅用于高亮可走格子，真正的规则裁定在服务端）。
// 源：ui/JunglePanel.tsx:1-96。
import type { JungleAnimal, SeatKey } from "../shared/types";

export const ROWS = 9;
export const COLS = 7;

export const ANIMAL_LABEL: Record<JungleAnimal, string> = {
  rat: "鼠", cat: "猫", dog: "狗", wolf: "狼", leopard: "豹", tiger: "虎", lion: "狮", elephant: "象"
};
export const ANIMAL_EMOJI: Record<JungleAnimal, string> = {
  rat: "🐀", cat: "🐈", dog: "🐕", wolf: "🐺", leopard: "🐆", tiger: "🐅", lion: "🦁", elephant: "🐘"
};
export const ANIMAL_RANK: Record<JungleAnimal, number> = {
  rat: 1, cat: 2, dog: 3, wolf: 4, leopard: 5, tiger: 6, lion: 7, elephant: 8
};

export function parseCell(cell: string | null | undefined): { side: SeatKey; animal: JungleAnimal } | null {
  if (!cell) return null;
  const [side, animal] = cell.split(":");
  if ((side !== "A" && side !== "B") || !(animal in ANIMAL_RANK)) return null;
  return { side: side as SeatKey, animal: animal as JungleAnimal };
}

export function isWater(row: number, col: number) {
  if (row < 3 || row > 5) return false;
  return (col >= 1 && col <= 2) || (col >= 4 && col <= 5);
}
export function isDen(row: number, col: number, side: SeatKey) {
  return col === 3 && (side === "A" ? row === 8 : row === 0);
}
export function isTrap(row: number, col: number, side: SeatKey) {
  if (side === "A") return (row === 8 && (col === 2 || col === 4)) || (row === 7 && col === 3);
  return (row === 0 && (col === 2 || col === 4)) || (row === 1 && col === 3);
}
export function trapOwner(row: number, col: number): SeatKey | null {
  if (isTrap(row, col, "A")) return "A";
  if (isTrap(row, col, "B")) return "B";
  return null;
}

export function canCapture(attackerSide: SeatKey, attacker: JungleAnimal, fromR: number, fromC: number, toR: number, toC: number, defender: JungleAnimal) {
  if (trapOwner(toR, toC) === attackerSide) return true;
  if (attacker === "rat" && defender === "elephant") {
    if (isWater(fromR, fromC) && !isWater(toR, toC)) return false;
    return true;
  }
  if (attacker === "elephant" && defender === "rat") return false;
  return ANIMAL_RANK[attacker] >= ANIMAL_RANK[defender];
}

export function jumpDest(board: Array<Array<string | null>>, fromR: number, fromC: number, dr: number, dc: number) {
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

export function canMoveTo(board: Array<Array<string | null>>, seat: SeatKey, animal: JungleAnimal, fromR: number, fromC: number, toR: number, toC: number, isJump: boolean) {
  if (toR < 0 || toR >= ROWS || toC < 0 || toC >= COLS) return false;
  if (isDen(toR, toC, seat)) return false;
  if (!isJump && isWater(toR, toC) && animal !== "rat") return false;
  const target = board[toR][toC];
  if (!target) return true;
  const def = parseCell(target);
  if (!def || def.side === seat) return false;
  return canCapture(seat, animal, fromR, fromC, toR, toC, def.animal);
}

export function legalDests(board: Array<Array<string | null>>, seat: SeatKey, fromR: number, fromC: number) {
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
