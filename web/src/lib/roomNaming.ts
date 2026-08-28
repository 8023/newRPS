// 建房自动起名：按已选惩罚来源（随机任务标签 / 系列任务 / 玩家发布）从对应词库随机拼一个
// 房名；没有惩罚或词库为空时退回各游戏的固定默认房名。源：ui/AppViews.tsx:1376-1401。
import type { AppConfig, RoomSettings } from "../shared/types";
import {
  defaultChessRoomName, defaultCoinFlipRoomName, defaultGomokuRoomName, defaultJungleRoomName,
  defaultLiarsDiceRoomName, defaultOthelloRoomName, defaultRoomName, defaultTicTacToeRoomName
} from "./constants";

export function defaultRoomNameForGame(gameId: RoomSettings["gameId"]): string {
  switch (gameId) {
    case "othello": return defaultOthelloRoomName;
    case "tictactoe": return defaultTicTacToeRoomName;
    case "gomoku": return defaultGomokuRoomName;
    case "jungle": return defaultJungleRoomName;
    case "chess": return defaultChessRoomName;
    case "liarsdice": return defaultLiarsDiceRoomName;
    case "coinflip": return defaultCoinFlipRoomName;
    default: return defaultRoomName;
  }
}

export function randomItem(items: string[]) {
  return items[Math.floor(Math.random() * items.length)] || "";
}

/** 按后台标签固定顺序取第一个有房名词库的已选标签。 */
export function roomNamePoolForSettings(config: AppConfig, settings: RoomSettings) {
  const src = settings.punishmentSource === "system" ? "random" : (settings.punishmentSource || "random");
  if (src === "player") return config.playerPunishmentRoomNamePool;
  if (src === "series") {
    const series = (config.punishmentSeriesSummaries || []).find((s) => s.id === settings.punishmentSeriesId);
    return series?.roomNamePool;
  }
  const included = new Set(settings.punishmentTagsIncluded || []);
  for (const tag of config.punishmentTags || []) {
    if (!included.has(tag.id)) continue;
    if (tag.roomNamePool?.subjects?.length && tag.roomNamePool?.roomWords?.length) {
      return tag.roomNamePool;
    }
  }
  return undefined;
}

export function generateRoomName(config: AppConfig, settings: RoomSettings) {
  if (settings.gameId === "othello" && !settings.enablePunishment) return defaultOthelloRoomName;
  if (settings.gameId === "tictactoe" && !settings.enablePunishment) return defaultTicTacToeRoomName;
  if (settings.gameId === "gomoku" && !settings.enablePunishment) return defaultGomokuRoomName;
  if (settings.gameId === "jungle" && !settings.enablePunishment) return defaultJungleRoomName;
  if (settings.gameId === "chess" && !settings.enablePunishment) return defaultChessRoomName;
  if (settings.gameId === "liarsdice" && !settings.enablePunishment) return defaultLiarsDiceRoomName;
  const pool = roomNamePoolForSettings(config, settings);
  if (!pool) return defaultRoomNameForGame(settings.gameId);
  const subject = randomItem(pool.subjects);
  const roomWord = randomItem(pool.roomWords);
  const adjective = pool.adjectives.length && Math.random() < 0.75 ? randomItem(pool.adjectives) : "";
  return `${adjective}${subject}${roomWord}`;
}

export function filterTagIds(config: AppConfig, ids: string[]) {
  const valid = new Set((config.punishmentTags || []).map((t) => t.id));
  return ids.filter((id, index) => ids.indexOf(id) === index && valid.has(id));
}

export function sameStringArray(left: string[], right: string[]) {
  return left.length === right.length && left.every((item, index) => item === right[index]);
}
