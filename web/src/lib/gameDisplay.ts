// 座位制游戏（RPS/黑白棋/井字棋/五子棋/斗兽棋/国际象棋）通用的纯展示函数：出拳文案、
// 房间玩家名单、惩罚审批/发布任务权限判定、对局记录文案、棋盘主题样式。
// 源：ui/AppViews.tsx 散落各处（Room/RoundHistoryCard/SeatView 等组件之间共用）。
import type { ChessPiece, CoinFace, Move, PublicPlayer, RoomSettings, RoomSnapshot, SeatKey, SeatOccupant } from "../shared/types";
import {
  chessBoardThemes, gomokuBoardThemes, jungleBoardThemes, othelloBoardThemes, tictactoeBoardThemes
} from "./constants";
import { displayPlayerName } from "./playerDisplay";

export function occupantDisplay(occupant: SeatOccupant) {
  if (!occupant) return "空位";
  return displayPlayerName(occupant);
}

export function choiceText(choice: Move | "hidden") {
  if (choice === "hidden") return "🔒 已出拳";
  if (choice === "noMove") return "⏳ 未出拳";
  if (choice === "forfeit") return "📴 断线判负";
  if (choice === "giveaway") return "🫴 白给";
  return choice === "rock" ? "✊ 锤子" : choice === "scissors" ? "✌️ 剪刀" : "🖐️ 布";
}

// 出拳按钮用：与 choiceText 分开是因为按钮里 emoji 和文字分两个节点渲染（撤回按钮要接
// 着复用同一个 emoji），choiceText 是拼成一整句提示文案，两者用途不同不合并。
export function moveButtonIcon(move: Move | "hidden") {
  switch (move) {
    case "rock": return "✊";
    case "scissors": return "✌️";
    case "paper": return "🖐️";
    case "giveaway": return "🫴";
    default: return "❔";
  }
}

export function moveButtonLabel(move: Move | "hidden") {
  switch (move) {
    case "rock": return "锤子";
    case "scissors": return "剪刀";
    case "paper": return "布";
    case "giveaway": return "白给";
    default: return "";
  }
}

export function historyResultText(result: RoomSnapshot["roundHistory"][number]["result"]) {
  if (result === "doubleLoss") return "双输";
  if (result === "draw") return "平局";
  return `${result} 胜`;
}

export function roomPlayerList(room: RoomSnapshot) {
  const result: Array<{ player: PublicPlayer; role: string }> = [];
  for (const seat of ["A", "B"] as SeatKey[]) {
    const occupant = room.seats[seat];
    if (occupant) result.push({ player: occupant, role: "战斗席" });
  }
  for (const player of room.spectators) result.push({ player, role: "观战" });
  return result;
}

export function roomPlayerById(room: RoomSnapshot, playerId: string) {
  return roomPlayerList(room).map((item) => item.player).find((player) => player.id === playerId);
}

export function punishedPlayerName(room: RoomSnapshot, playerId: string) {
  const player = roomPlayerById(room, playerId);
  return player ? displayPlayerName(player) : playerId;
}

export function punishedPlayerNames(room: RoomSnapshot) {
  const players = roomPlayerList(room).map((item) => item.player);
  return (room.punishedPlayerIds || []).map((id) => {
    const player = players.find((item) => item.id === id);
    return player ? displayPlayerName(player) : id;
  });
}

export function escapeRegExp(value: string) {
  return value.replace(/[.*+?^${}()|[\]\\]/g, "\\$&");
}

export function taskTextOnly(taskText: string, factionLabel?: string) {
  const labels = [factionLabel, "男性阵营", "女性阵营", "男娘阵营", "其他阵营"].filter(Boolean) as string[];
  let result = taskText;
  for (const label of labels) {
    result = result.replace(new RegExp(`^${escapeRegExp(label)}[：:]\\s*`), "");
  }
  return result.trim();
}

// canReviewPunishmentProof 与后端 canReviewPlayer 对齐：座位制=不同座位的对手；
// 大话骰=本局赢家审核本局输家（以最近一条对局记录为准）。
export function canReviewPunishmentProof(room: RoomSnapshot, reviewerId: string, targetId: string) {
  if (!reviewerId || reviewerId === targetId) return false;
  if (room.settings.gameId === "liarsdice") {
    const latest = room.roundHistory[0];
    return Boolean(latest && latest.liarsDiceWinnerId === reviewerId && latest.liarsDiceLoserId === targetId);
  }
  const reviewerSeat = room.seats.A?.id === reviewerId ? "A" : room.seats.B?.id === reviewerId ? "B" : null;
  const targetSeat = room.seats.A?.id === targetId ? "A" : room.seats.B?.id === targetId ? "B" : null;
  return Boolean(reviewerSeat && targetSeat && reviewerSeat !== targetSeat);
}

export function canAssignPunishmentTask(room: RoomSnapshot, currentPlayerId: string, punishedPlayerId: string, assignedBy?: string) {
  if (assignedBy) return assignedBy === currentPlayerId;
  if (room.settings.gameId === "liarsdice") {
    const winnerId = room.roundHistory[0]?.liarsDiceWinnerId;
    return Boolean(winnerId && winnerId === currentPlayerId && winnerId !== punishedPlayerId);
  }
  const punishedSeat = room.seats.A?.id === punishedPlayerId ? "A" : room.seats.B?.id === punishedPlayerId ? "B" : null;
  if (!punishedSeat) return false;
  const opponent = punishedSeat === "A" ? room.seats.B : room.seats.A;
  return Boolean(opponent && opponent.id === currentPlayerId);
}

export function historyProofStatusLabel(proof: { status?: string; confirmedBy?: string; rejectReason?: string }) {
  if (proof.status === "approved" || proof.confirmedBy) {
    if (proof.rejectReason === "对方选择放过你" || proof.rejectReason === "双方互相放过，下一局正常开始。") return "已放过";
    return "已通过";
  }
  if (proof.status === "rejected") return "需重做";
  if (proof.status === "pending") return "待审核";
  return proof.status || "已提交";
}

export function historySeatLabel(item: RoomSnapshot["roundHistory"][number], seat: SeatKey, coinFaceLabel: (face: CoinFace | undefined) => string) {
  if (item.gameId === "othello") return item.othelloBlackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋";
  if (item.gameId === "tictactoe") return item.tictactoeXSeat === seat ? "❌ X" : "⭕ O";
  if (item.gameId === "gomoku") return item.gomokuBlackSeat === seat ? "⚫ 黑棋" : "⚪ 白棋";
  if (item.gameId === "jungle") return jungleSideLabel(seat);
  if (item.gameId === "chess") return item.chessWhiteSeat === seat ? "⚪ 白棋" : "⚫ 黑棋";
  if (item.gameId === "liarsdice") {
    // 大话骰对局记录里 playerA 固定是本局赢家、playerB 固定是输家（见后端 game_liarsdice.go）。
    return seat === "A" ? "🏆 胜" : "💤 负";
  }
  if (item.gameId === "coinflip") {
    return seat === "A" ? `🪙 猜${coinFaceLabel(item.coinFlipGuess)}` : `开${coinFaceLabel(item.coinFlipResult)}`;
  }
  return choiceText(seat === "A" ? item.moveA : item.moveB);
}

export function formatClockMs(ms: number): string {
  const totalSeconds = Math.max(0, Math.ceil(ms / 1000));
  const m = Math.floor(totalSeconds / 60);
  const s = totalSeconds % 60;
  return `${m}:${s.toString().padStart(2, "0")}`;
}

export function othelloDeltaText(state: NonNullable<RoomSnapshot["othello"]>, color: "black" | "white") {
  const seat = color === "black" ? state.blackSeat : state.blackSeat === "A" ? "B" : "A";
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function tictactoeDeltaText(state: NonNullable<RoomSnapshot["tictactoe"]>, mark: "X" | "O") {
  const seat = mark === "X" ? state.xSeat : state.xSeat === "A" ? "B" : "A";
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function gomokuDeltaText(state: NonNullable<RoomSnapshot["gomoku"]>, seat: SeatKey) {
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function jungleDeltaText(state: NonNullable<RoomSnapshot["jungle"]>, seat: SeatKey) {
  const delta = state.rankedDelta?.[seat] || 0;
  return `${delta >= 0 ? "+" : ""}${delta}`;
}

export function chessDeltaText(state: NonNullable<RoomSnapshot["chess"]>, seat: SeatKey) {
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

export function chessSideLabel(state: RoomSnapshot["chess"] | undefined, seat: SeatKey): string {
  if (!state) return "随机后显示白/黑";
  return state.whiteSeat === seat ? "⚪ 白棋" : "⚫ 黑棋";
}

export function othelloThemeStyle(themeId?: RoomSettings["othelloBoardTheme"]): Record<string, string> {
  const theme = othelloBoardThemes.find((item) => item.id === themeId) || othelloBoardThemes[0];
  return {
    "--othello-board": theme.board, "--othello-cell": theme.cell, "--othello-line": theme.line, "--othello-hover": theme.hover,
    "--othello-border": theme.border, "--othello-black-disc": theme.blackDisc, "--othello-white-disc": theme.whiteDisc,
    "--othello-black-ring": theme.blackRing, "--othello-white-ring": theme.whiteRing
  };
}

export function tictactoeThemeStyle(themeId?: RoomSettings["tictactoeBoardTheme"]): Record<string, string> {
  const theme = tictactoeBoardThemes.find((item) => item.id === themeId) || tictactoeBoardThemes[0];
  return {
    "--ttt-board": theme.board, "--ttt-cell": theme.cell, "--ttt-line": theme.line, "--ttt-hover": theme.hover,
    "--ttt-border": theme.border, "--ttt-x": theme.x, "--ttt-o": theme.o, "--ttt-win": theme.win
  };
}

export function gomokuThemeStyle(themeId?: RoomSettings["gomokuBoardTheme"]): Record<string, string> {
  const theme = gomokuBoardThemes.find((item) => item.id === themeId) || gomokuBoardThemes.find((item) => item.id === "wood") || gomokuBoardThemes[0];
  return {
    "--gomoku-board": theme.board, "--gomoku-cell": theme.cell, "--gomoku-line": theme.line, "--gomoku-hover": theme.hover,
    "--gomoku-border": theme.border, "--gomoku-black-disc": theme.blackDisc, "--gomoku-white-disc": theme.whiteDisc,
    "--gomoku-black-ring": theme.blackRing, "--gomoku-white-ring": theme.whiteRing
  };
}

export function jungleThemeStyle(themeId?: RoomSettings["jungleBoardTheme"]): Record<string, string> {
  const theme = jungleBoardThemes.find((item) => item.id === themeId) || jungleBoardThemes[0];
  return {
    "--jungle-board": theme.board, "--jungle-land": theme.land, "--jungle-water": theme.water, "--jungle-trap": theme.trap,
    "--jungle-den": theme.den, "--jungle-border": theme.border, "--jungle-a-piece": theme.aPiece, "--jungle-b-piece": theme.bPiece,
    "--jungle-a-text": theme.aText, "--jungle-b-text": theme.bText, "--jungle-hover": theme.hover
  };
}

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
  return { r: Number.parseInt(expanded.slice(0, 2), 16), g: Number.parseInt(expanded.slice(2, 4), 16), b: Number.parseInt(expanded.slice(4, 6), 16) };
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

export function chessThemeStyle(themeId?: RoomSettings["chessBoardTheme"]): Record<string, string> {
  const theme = chessBoardThemes.find((item) => item.id === themeId) || chessBoardThemes[0];
  return {
    "--chess-board": theme.board, "--chess-light": theme.light, "--chess-dark": theme.dark, "--chess-border": theme.border,
    "--chess-hover": theme.hover, "--chess-check": theme.check,
    ...chessSquareHintStyle("light", theme.light), ...chessSquareHintStyle("dark", theme.dark)
  };
}

export function parseChessCell(cell: string | null | undefined): { color: "white" | "black"; piece: string } | null {
  if (!cell) return null;
  const [color, piece] = cell.split(":");
  if ((color !== "white" && color !== "black") || !piece) return null;
  return { color, piece };
}

export type { ChessPiece };
