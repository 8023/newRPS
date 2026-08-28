// 房间信息小标签（大厅列表 + 房间头部共用）：游戏/阶段/排位/惩罚来源/计时/悔棋等一排
// 药丸标签，文案与配色优先取后台 config.roomInfoTags，缺失则退回内置默认色。
// 源：ui/AppViews.tsx 3298-3491 与 5201-5229（两处分散定义，这里按主题收拢到一处）。
import type { AppConfig, LobbySnapshot, RoomInfoTagStyle, RoomSettings, RoomSnapshot } from "../shared/types";
import { rankMultiplierForSettings } from "./playerDisplay";

export type RoomInfoTagView = { key: string; text: string; style: RoomInfoTagStyle };

export const roomInfoTagOrder = [
  { key: "gameRps", label: "锤子剪刀布" },
  { key: "gameOthello", label: "黑白棋" },
  { key: "gameTicTacToe", label: "井字棋" },
  { key: "gameGomoku", label: "五子棋" },
  { key: "gameJungle", label: "斗兽棋" },
  { key: "gameChess", label: "国际象棋" },
  { key: "gameLiarsDice", label: "大话骰" },
  { key: "gameCoinFlip", label: "猜硬币" },
  { key: "phaseReady", label: "等待坐满" },
  { key: "phaseChoosing", label: "出拳中" },
  { key: "phaseChoosingLiarsDice", label: "叫点中" },
  { key: "phaseChoosingCoinFlip", label: "抛硬币中" },
  { key: "phaseResult", label: "结算中" },
  { key: "phasePunishment", label: "惩罚阶段" },
  { key: "normal", label: "普通局" },
  { key: "ranked", label: "排位" },
  { key: "extremeRanked", label: "极限排位" },
  { key: "punishment", label: "惩罚开启" },
  { key: "noPunishment", label: "无惩罚" },
  { key: "tieDoublePunish", label: "平局双罚" },
  { key: "requireOpponentConfirm", label: "需要对手确认" },
  { key: "allowProofImage", label: "允许图片证明" },
  { key: "textProofOnly", label: "仅文字证明" }
];

export function defaultRoomInfoTagStyle(label: string): RoomInfoTagStyle {
  return { label, textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };
}

export function roomInfoTag(config: AppConfig, key: string, extra = "", prefix = ""): RoomInfoTagView {
  const fromConfig = config.roomInfoTags?.[key];
  const fromOrder = roomInfoTagOrder.find((item) => item.key === key);
  // 配置缺失时用中文默认名+默认色，避免直接露出 gameRps 等内部 key
  const fallback: RoomInfoTagStyle = defaultRoomInfoTagStyle(fromOrder?.label || key);
  const style: RoomInfoTagStyle = {
    label: fromConfig?.label || fallback.label,
    textColor: fromConfig?.textColor || fallback.textColor,
    backgroundColor: fromConfig?.backgroundColor || fallback.backgroundColor,
    borderColor: fromConfig?.borderColor || fallback.borderColor
  };
  return { key: `${key}-${extra}`, text: `${prefix}${style.label}${extra}`, style };
}

export function gameInfoTag(config: AppConfig, gameId: RoomSettings["gameId"]) {
  return gameId === "othello"
    ? roomInfoTag(config, "gameOthello", "", "⚫⚪ ")
    : gameId === "tictactoe"
      ? roomInfoTag(config, "gameTicTacToe", "", "❌⭕ ")
      : gameId === "liarsdice"
        ? roomInfoTag(config, "gameLiarsDice", "", "🎲 ")
        : gameId === "gomoku"
          ? roomInfoTag(config, "gameGomoku", "", "●○ ")
          : gameId === "jungle"
            ? roomInfoTag(config, "gameJungle", "", "🦁 ")
            : gameId === "chess"
              ? roomInfoTag(config, "gameChess", "", "♔ ")
              : gameId === "coinflip"
                ? roomInfoTag(config, "gameCoinFlip", "", "🪙 ")
                : roomInfoTag(config, "gameRps");
}

/** 惩罚来源标签正文：自定义惩罚直接写"自定义惩罚"，随机任务写"#标签"（空格分隔；
 * maxTags 限制大厅列表最多显示 3 个，房间内头部不传即不限），系列任务直接写系列名——
 * 都不带"惩罚开启："这层前缀。大厅列表（lobbyPunishmentTagText）与房间头部
 * （punishmentInfoTag）共用这份措辞，避免同一房间进出显示两种文案。 */
export function punishmentSourceTagText(config: AppConfig, settings: Pick<RoomSettings, "punishmentSource" | "punishmentTagsIncluded" | "punishmentSeriesId" | "punishmentId" | "punishmentIds">, maxTags?: number) {
  const src = settings.punishmentSource === "system" ? "random" : (settings.punishmentSource || "random");
  if (src === "player") return "自定义惩罚";
  if (src === "series") {
    const series = (config.punishmentSeriesSummaries || []).find((s) => s.id === settings.punishmentSeriesId);
    return series ? series.name : "系列任务";
  }
  let names = (settings.punishmentTagsIncluded || [])
    .map((id) => (config.punishmentTags || []).find((t) => t.id === id)?.name)
    .filter((name): name is string => Boolean(name));
  if (maxTags) names = names.slice(0, maxTags);
  if (!names.length) return "随机任务";
  return names.map((name) => `#${name}`).join(" ");
}

export function lobbyPunishmentTagText(config: AppConfig, settings: Pick<RoomSettings, "punishmentSource" | "punishmentTagsIncluded" | "punishmentSeriesId" | "punishmentId" | "punishmentIds">) {
  return punishmentSourceTagText(config, settings, 3);
}

export function punishmentInfoTag(config: AppConfig, room: RoomSnapshot) {
  if (!room.settings.enablePunishment) return roomInfoTag(config, "noPunishment");
  // 房内标签统一成与大厅列表一致的措辞（punishmentSourceTagText），不再套「惩罚开启：」
  // 前缀——大厅、房间内看到的应当是同一句话；房间内标签数量不设上限（大厅列表限 3 个）。
  const style = roomInfoTag(config, "punishment").style;
  return { key: `punishment-${room.id}-${room.settings.punishmentSource || "random"}`, text: punishmentSourceTagText(config, room.settings), style };
}

export function rankedInfoExtra(stake: number, multiplier = 1, gameId: RoomSettings["gameId"] = "rps") {
  if (gameId === "othello") return multiplier > 1 ? ` ${stake} 分/子 ×${multiplier}` : ` ${stake} 分/子`;
  return multiplier > 1 ? ` ${stake} 分 ×${multiplier}` : ` ${stake} 分`;
}

const plainInfoTagStyle: RoomInfoTagStyle = { label: "", textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };

/** 大话骰参战人数区间标签，如「3-7人」；人数区间未知（如 0）时不显示。 */
function liarsDiceRosterTag(roomId: string, min?: number, max?: number): RoomInfoTagView | null {
  if (!min || !max) return null;
  return { key: `liarsdice-roster-${roomId}`, text: `${min}-${max}人`, style: plainInfoTagStyle };
}

/** 黑白棋/五子棋/斗兽棋/国际象棋计时标签，如「计时：30秒/20分钟」；两个计时器都未启用时不显示。 */
function gameTimerTag(roomId: string, moveSeconds?: number, gameMinutes?: number): RoomInfoTagView | null {
  if (!moveSeconds && !gameMinutes) return null;
  const parts: string[] = [];
  if (moveSeconds) parts.push(`${moveSeconds}秒`);
  if (gameMinutes) parts.push(`${gameMinutes}分钟`);
  return { key: `game-timer-${roomId}`, text: `计时：${parts.join("/")}`, style: plainInfoTagStyle };
}

/** 棋类悔棋次数标签，如「禁止悔棋」「允许悔棋1次」。 */
function gameUndoTag(roomId: string, gameId: RoomSettings["gameId"], undoLimit?: number): RoomInfoTagView {
  return {
    key: `${gameId}-undo-${roomId}`,
    text: undoLimit ? `允许悔棋${undoLimit}次` : "禁止悔棋",
    style: plainInfoTagStyle
  };
}

export function roomInfoTags(config: AppConfig, room: RoomSnapshot): RoomInfoTagView[] {
  const phaseKey = room.phase === "ready" ? "phaseReady"
    : room.phase === "choosing" ? (room.settings.gameId === "liarsdice" ? "phaseChoosingLiarsDice" : room.settings.gameId === "coinflip" ? "phaseChoosingCoinFlip" : "phaseChoosing")
      : room.phase === "result" ? "phaseResult"
        : room.phase === "punishment" ? "phasePunishment"
          : "phaseReady";
  const multiplier = rankMultiplierForSettings(room.settings);
  const tags: RoomInfoTagView[] = [
    gameInfoTag(config, room.settings.gameId),
    roomInfoTag(config, phaseKey),
    room.settings.enableRanked ? roomInfoTag(config, "ranked", rankedInfoExtra(room.settings.stake, multiplier, room.settings.gameId)) : roomInfoTag(config, "normal"),
    punishmentInfoTag(config, room)
  ];
  if (room.settings.enableExtremeRanked) tags.push(roomInfoTag(config, "extremeRanked"));
  if (room.settings.enablePunishment) {
    if (room.settings.tieDoublePunish && room.settings.gameId !== "liarsdice") tags.push(roomInfoTag(config, "tieDoublePunish"));
    if (room.settings.requireOpponentConfirm) tags.push(roomInfoTag(config, "requireOpponentConfirm"));
    tags.push(roomInfoTag(config, room.settings.allowProofImage === false ? "textProofOnly" : "allowProofImage"));
  }
  if (room.settings.gameId === "liarsdice") {
    const t = liarsDiceRosterTag(room.id, room.settings.liarsDiceMinPlayers, room.settings.liarsDiceMaxPlayers);
    if (t) tags.push(t);
  } else if (room.settings.gameId === "othello") {
    const t = gameTimerTag(room.id, room.settings.othelloMoveSeconds, room.settings.othelloGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.settings.gameId, room.settings.othelloUndoLimit));
  } else if (room.settings.gameId === "gomoku") {
    const t = gameTimerTag(room.id, room.settings.gomokuMoveSeconds, room.settings.gomokuGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.settings.gameId, room.settings.gomokuUndoLimit));
  } else if (room.settings.gameId === "jungle") {
    const t = gameTimerTag(room.id, room.settings.jungleMoveSeconds, room.settings.jungleGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.settings.gameId, room.settings.jungleUndoLimit));
  } else if (room.settings.gameId === "chess") {
    const t = gameTimerTag(room.id, room.settings.chessMoveSeconds, room.settings.chessGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.settings.gameId, room.settings.chessUndoLimit));
  }
  return tags;
}

export function lobbyRoomInfoTags(config: AppConfig, room: LobbySnapshot["rooms"][number]): RoomInfoTagView[] {
  const multiplier = room.rankMultiplier || 1;
  const tags: RoomInfoTagView[] = [
    gameInfoTag(config, room.gameId),
    room.enableRanked ? roomInfoTag(config, "ranked", rankedInfoExtra(room.stake, multiplier, room.gameId)) : roomInfoTag(config, "normal")
  ];
  // 大厅列表的惩罚标签只显示玩家能感知的具体内容（自定义惩罚 / #标签 / 系列任务名），
  // 不再套「惩罚开启：」这层前缀；未开启惩罚则完全不展示这个标签。
  if (room.enablePunishment) {
    const punishmentStyle = roomInfoTag(config, "punishment").style;
    tags.push({ key: `punishment-${room.id}`, text: lobbyPunishmentTagText(config, room), style: punishmentStyle });
  }
  if (room.enableExtremeRanked) tags.push(roomInfoTag(config, "extremeRanked"));
  if (room.enablePunishment) {
    if (room.tieDoublePunish && room.gameId !== "liarsdice") tags.push(roomInfoTag(config, "tieDoublePunish"));
    if (room.requireOpponentConfirm) tags.push(roomInfoTag(config, "requireOpponentConfirm"));
  }
  if (room.gameId === "liarsdice") {
    const t = liarsDiceRosterTag(room.id, room.liarsDiceMinPlayers, room.liarsDiceMaxPlayers);
    if (t) tags.push(t);
  } else if (room.gameId === "othello") {
    const t = gameTimerTag(room.id, room.othelloMoveSeconds, room.othelloGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.gameId, room.othelloUndoLimit));
  } else if (room.gameId === "gomoku") {
    const t = gameTimerTag(room.id, room.gomokuMoveSeconds, room.gomokuGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.gameId, room.gomokuUndoLimit));
  } else if (room.gameId === "jungle") {
    const t = gameTimerTag(room.id, room.jungleMoveSeconds, room.jungleGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.gameId, room.jungleUndoLimit));
  } else if (room.gameId === "chess") {
    const t = gameTimerTag(room.id, room.chessMoveSeconds, room.chessGameMinutes);
    if (t) tags.push(t);
    tags.push(gameUndoTag(room.id, room.gameId, room.chessUndoLimit));
  }
  tags.push({ key: `opponent-${room.id}`, text: "在线对战", style: plainInfoTagStyle });
  return tags;
}

export function roomInfoTagStyle(style: RoomInfoTagStyle): Record<string, string> {
  return {
    "--room-info-text": style.textColor,
    "--room-info-bg": style.backgroundColor,
    "--room-info-border": style.borderColor
  };
}

export function roomStatusText(status: RoomSnapshot["status"]) {
  if (status === "playing") return "对战中";
  if (status === "punishment") return "惩罚中";
  return "等待中";
}
