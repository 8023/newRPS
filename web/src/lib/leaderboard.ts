// 全站排行榜（GlobalLeaderboardPanel 用）：分类排序规则 + 全站档案分页拉取与短 TTL 缓存。
// 源：ui/AppViews.tsx 3694-3966（rosterCache 相关）与 3907-3966（leaderboardPlayers）。
import type { PublicPlayer } from "../shared/types";
import { ask } from "./rpc";
import { normalizePublicPlayer } from "./normalize";
import {
  contributionApprovedCountOf, safePlayerStats, sortHighestScoreOf, sortLowestScoreOf, sortRankedPointsOf, totalOnlineMsOf
} from "./playerDisplay";

export type GlobalLeaderboardTab =
  | "positive" | "negative" | "historyPositive" | "historyNegative" | "extremePositive" | "extremeNegative" | "nameWar" | "giveaway"
  | "totalWins" | "onlineTime" | "contribution"
  | "rps" | "othello" | "tictactoe" | "gomoku" | "liarsdice" | "jungle" | "chess";

export const GAME_LEADERBOARD_TABS: Array<{ id: GlobalLeaderboardTab; label: string; title: string }> = [
  { id: "rps", label: "猜拳", title: "锤子剪刀布胜场榜" },
  { id: "othello", label: "黑白棋", title: "黑白棋胜场榜" },
  { id: "tictactoe", label: "井字棋", title: "井字棋胜场榜" },
  { id: "gomoku", label: "五子棋", title: "五子棋胜场榜" },
  { id: "jungle", label: "斗兽棋", title: "斗兽棋胜场榜" },
  { id: "chess", label: "国象", title: "国际象棋胜场榜" },
  { id: "liarsdice", label: "大话骰", title: "大话骰胜场榜" }
];

export function isGameLeaderboardTab(tab: GlobalLeaderboardTab) {
  return tab === "rps" || tab === "othello" || tab === "tictactoe" || tab === "gomoku" || tab === "jungle" || tab === "chess" || tab === "liarsdice";
}

export function gameWLDOf(player: PublicPlayer, tab: GlobalLeaderboardTab) {
  const gs = player.gameStats || { rps: {}, othello: {}, tictactoe: {}, gomoku: {}, liarsdice: {}, jungle: {}, chess: {} } as PublicPlayer["gameStats"];
  const raw = tab === "rps" ? gs.rps
    : tab === "othello" ? gs.othello
      : tab === "tictactoe" ? gs.tictactoe
        : tab === "gomoku" ? gs.gomoku
          : tab === "jungle" ? gs.jungle
            : tab === "chess" ? gs.chess
              : tab === "liarsdice" ? gs.liarsdice
                : undefined;
  return {
    wins: Number(raw?.wins) || 0,
    losses: Number(raw?.losses) || 0,
    draws: Number(raw?.draws) || 0
  };
}

export function leaderboardPlayers(players: PublicPlayer[], tab: GlobalLeaderboardTab) {
  const copy = [...players];
  // 过滤可用展示分（已封顶）；排序一律用 sort*（服务端下发时已与展示值一致，不带真实分，
  // 真实分只有管理员后台可见——见 server.publicPlayer/publicPlayerAdmin），未封顶时才等于真实分。
  if (tab === "positive") return copy.filter((player) => sortRankedPointsOf(player) > 0).sort((a, b) => sortRankedPointsOf(b) - sortRankedPointsOf(a) || b.stats.wins - a.stats.wins);
  if (tab === "negative") return copy.filter((player) => sortRankedPointsOf(player) < 0).sort((a, b) => sortRankedPointsOf(a) - sortRankedPointsOf(b) || b.stats.losses - a.stats.losses);
  if (tab === "historyPositive") return copy.filter((player) => sortHighestScoreOf(player) > 0).sort((a, b) => sortHighestScoreOf(b) - sortHighestScoreOf(a) || b.stats.wins - a.stats.wins);
  if (tab === "historyNegative") return copy.filter((player) => sortLowestScoreOf(player) < 0).sort((a, b) => sortLowestScoreOf(a) - sortLowestScoreOf(b) || b.stats.losses - a.stats.losses);
  if (tab === "extremePositive") return copy.filter((player) => player.extremeModeEnabled && sortRankedPointsOf(player) > 0).sort((a, b) => sortRankedPointsOf(b) - sortRankedPointsOf(a) || (b.extremeWinStreak || 0) - (a.extremeWinStreak || 0));
  if (tab === "extremeNegative") return copy.filter((player) => player.extremeModeEnabled && sortRankedPointsOf(player) < 0).sort((a, b) => sortRankedPointsOf(a) - sortRankedPointsOf(b) || (b.extremeWinStreak || 0) - (a.extremeWinStreak || 0));
  if (tab === "nameWar") {
    return copy
      .filter((player) => player.nameWarEnabled || player.nameWarPunished)
      .sort((a, b) => Number(Boolean(b.nameWarPunished)) - Number(Boolean(a.nameWarPunished)) || sortRankedPointsOf(a) - sortRankedPointsOf(b));
  }
  // 总局数：按全部游戏合计胜场排序（与 stats.wins 一致）。
  if (tab === "totalWins") {
    return copy
      .filter((player) => {
        const s = safePlayerStats(player);
        return s.wins + s.losses + s.draws > 0;
      })
      .sort((a, b) => {
        const sa = safePlayerStats(a);
        const sb = safePlayerStats(b);
        return sb.wins - sa.wins || sa.losses - sb.losses || sb.draws - sa.draws;
      });
  }
  // 在线时长：累计在线毫秒从长到短。
  if (tab === "onlineTime") {
    return copy
      .filter((player) => totalOnlineMsOf(player) > 0)
      .sort((a, b) => totalOnlineMsOf(b) - totalOnlineMsOf(a) || sortRankedPointsOf(b) - sortRankedPointsOf(a));
  }
  // 共建：提交并审批通过的投稿任务条数（随机任务 + 系列每个子任务各算一条）从多到少。
  if (tab === "contribution") {
    return copy
      .filter((player) => contributionApprovedCountOf(player) > 0)
      .sort((a, b) => contributionApprovedCountOf(b) - contributionApprovedCountOf(a) || sortRankedPointsOf(b) - sortRankedPointsOf(a));
  }
  if (isGameLeaderboardTab(tab)) {
    return copy
      .filter((player) => {
        const g = gameWLDOf(player, tab);
        return g.wins + g.losses + g.draws > 0;
      })
      .sort((a, b) => {
        const ga = gameWLDOf(a, tab);
        const gb = gameWLDOf(b, tab);
        return gb.wins - ga.wins || ga.losses - gb.losses || gb.draws - ga.draws;
      });
  }
  return copy
    .filter((player) => player.giveawayEnabled || (player.giveawayValue || 0) > 0)
    .sort((a, b) => (b.giveawayValue || 0) - (a.giveawayValue || 0) || sortRankedPointsOf(b) - sortRankedPointsOf(a));
}

// 全站榜单模块级短 TTL 缓存：同一浏览器 tab 内反复点开「排行榜」不必每次都重新拉全量分页——
// 缓存新鲜则直接复用；过期则先展示旧数据、后台悄悄重新拉一遍再原地刷新，避免空转等待。
export let rosterCache: { data: PublicPlayer[]; fetchedAt: number } | null = null;
export const ROSTER_CACHE_TTL_MS = 20_000;

export function setRosterCache(data: PublicPlayer[]) {
  rosterCache = { data, fetchedAt: Date.now() };
}

// 按 players:roster 分页拉取全站玩家档案。先顺序拉第 1 页拿到 total，再把剩余页一次性并发
// 发出（服务端每页只是纯内存转换，互不依赖），把原来 N 次顺序往返的墙钟耗时压到约 2 次。
export async function fetchRosterPages(alive: () => boolean): Promise<PublicPlayer[] | null> {
  // 与服务端 rosterMaxLimit 对齐；maxPages 对应最多 5000 人，防止异常库无限拉。
  const pageSize = 500;
  const maxPages = 10;
  const first = await ask<{
    players?: PublicPlayer[];
    hasMore?: boolean;
    total?: number;
  }>("players:roster", { offset: 0, limit: pageSize });
  if (!alive()) return null;
  const chunks: PublicPlayer[][] = [(first?.players || []).map(normalizePublicPlayer)];
  const total = first?.total || 0;
  if (first?.hasMore) {
    const offsets: number[] = [];
    for (let offset = pageSize; offset < total && offsets.length < maxPages - 1; offset += pageSize) {
      offsets.push(offset);
    }
    const rest = await Promise.all(
      offsets.map((offset) =>
        ask<{ players?: PublicPlayer[] }>("players:roster", { offset, limit: pageSize }).then((res) =>
          (res?.players || []).map(normalizePublicPlayer)
        )
      )
    );
    if (!alive()) return null;
    chunks.push(...rest);
  }
  return chunks.flat();
}
