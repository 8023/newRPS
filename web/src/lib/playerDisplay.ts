// 与玩家展示相关的纯函数：徽标文案/配色、性别阵营查表、排行榜排序键、抢名/白给判定。
// 源：ui/AppViews.tsx（原 React 版散落在文件各处，这里按主题收拢到一处，供各 Svelte
// 组件共用；本文件不含任何框架依赖）。
import type { AppConfig, GenderColors, GenderFaction, PublicPlayer, RoomSettings, RoomSnapshot } from "../shared/types";

export function mentionLabel(player?: PublicPlayer): string {
  if (!player) return "";
  return player.nameWarPunished && player.nameWarPenaltyName ? player.nameWarPenaltyName : player.name;
}

export function displayPlayerName(player: PublicPlayer) {
  if (player.nameWarPunished && player.nameWarPenaltyName) return `${player.extremeModeEnabled ? "极 " : ""}${player.nameWarPenaltyName}`;
  return player.name;
}

export function shouldShowGiveawayValue(player: PublicPlayer) {
  return Boolean(player.giveawayEnabled || (player.giveawayValue || 0) > 0);
}

export function formatGiveawayValue(value: number) {
  if (!Number.isFinite(value)) return "0";
  return value.toFixed(2).replace(/0+$/, "").replace(/\.$/, "");
}

// 按阵营过滤可选性别：性别预设未设归属阵营（factionId 为空）时视为不限阵营都能选，
// 与后端 validGenderSubmission 的兼容口径保持一致。
export function gendersForFaction(config: AppConfig, factionId: string) {
  return config.genders.filter((gender) => !gender.factionId || gender.factionId === factionId);
}

export function firstFactionId(config: AppConfig) {
  return config.genderFactions[0]?.id || "";
}

export function firstGenderId(config: AppConfig, factionId: string) {
  return gendersForFaction(config, factionId)[0]?.id || "";
}

// 切换阵营时性别是否需要跟着重置：当前性别只有在还落在新阵营的可选池内才保留，
// 否则退回新阵营的第一个预设。
export function nextGenderIdForFaction(config: AppConfig, factionId: string, currentGenderId: string) {
  if (!currentGenderId) return currentGenderId;
  const pool = gendersForFaction(config, factionId);
  if (pool.some((gender) => gender.id === currentGenderId)) return currentGenderId;
  return pool[0]?.id || "";
}

export function genderStyle(player: PublicPlayer): Record<string, string> {
  return {
    color: player.factionColors.textColor,
    backgroundColor: player.factionColors.backgroundColor,
    borderColor: player.factionColors.borderColor
  };
}

// 称号标签底色按赋予来源着色，服务端已解析好（见 PublicStats.titleColors），
// 这里兜底一份中性灰以防旧缓存数据缺该字段。
export const DEFAULT_TITLE_COLORS: GenderColors = { textColor: "#4d5c6f", backgroundColor: "#eef3f8", borderColor: "#c9d6e4" };

export function titleStyle(colors: GenderColors | null | undefined): Record<string, string> {
  const c = colors || DEFAULT_TITLE_COLORS;
  return { color: c.textColor, backgroundColor: c.backgroundColor, borderColor: c.borderColor };
}

export function factionStyle(faction: GenderFaction): Record<string, string> {
  return { color: faction.textColor, backgroundColor: faction.backgroundColor, borderColor: faction.borderColor };
}

// 与后端 validGenderSubmission 保持一致：genderId 必须命中某个预设，阵营由该预设的
// factionId 查表决定，不再单独提交/校验。返回空字符串表示校验通过，否则返回给用户看的原因。
export function genderChoiceError(config: AppConfig, genderId: string) {
  if (!config.genders.some((gender) => gender.id === genderId)) return "所选性别不存在";
  return "";
}

export function isValidGenderChoice(config: AppConfig, genderId: string) {
  return genderChoiceError(config, genderId) === "";
}

/** 排行榜排序键（有 sort* 用 sort*，否则退回展示分）。普通连接下发的 sort* 已与展示分一致，
 * 不携带真实分——真实分只有管理员后台可见，见 server.publicPlayer/publicPlayerAdmin。 */
export function sortRankedPointsOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortRankedPoints)) ? Number(s.sortRankedPoints) : Number(s.rankedPoints) || 0;
}
export function sortHighestScoreOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortHighestScore)) ? Number(s.sortHighestScore) : Number(s.highestScore) || 0;
}
export function sortLowestScoreOf(player: PublicPlayer) {
  const s = player?.stats;
  if (!s) return 0;
  return Number.isFinite(Number(s.sortLowestScore)) ? Number(s.sortLowestScore) : Number(s.lowestScore) || 0;
}

export function safePlayerStats(player: PublicPlayer | null | undefined) {
  const s = player?.stats || ({} as PublicPlayer["stats"]);
  const title = typeof s.title === "string" ? s.title.trim() : "";
  const rankedPoints = Number.isFinite(Number(s.rankedPoints)) ? Number(s.rankedPoints) : 0;
  const highestScore = Number.isFinite(Number(s.highestScore)) ? Number(s.highestScore) : 0;
  const lowestScore = Number.isFinite(Number(s.lowestScore)) ? Number(s.lowestScore) : 0;
  return {
    wins: Number.isFinite(Number(s.wins)) ? Number(s.wins) : 0,
    losses: Number.isFinite(Number(s.losses)) ? Number(s.losses) : 0,
    draws: Number.isFinite(Number(s.draws)) ? Number(s.draws) : 0,
    punishments: Number.isFinite(Number(s.punishments)) ? Number(s.punishments) : 0,
    rankedPoints,
    highestScore,
    lowestScore,
    sortRankedPoints: Number.isFinite(Number(s.sortRankedPoints)) ? Number(s.sortRankedPoints) : rankedPoints,
    sortHighestScore: Number.isFinite(Number(s.sortHighestScore)) ? Number(s.sortHighestScore) : highestScore,
    sortLowestScore: Number.isFinite(Number(s.sortLowestScore)) ? Number(s.sortLowestScore) : lowestScore,
    title: title || "暂无称号",
    titleCustom: !!s.titleCustom,
    titleColors: s.titleColors && typeof s.titleColors === "object" ? s.titleColors : DEFAULT_TITLE_COLORS,
    totalOnlineMs: Number.isFinite(Number(s.totalOnlineMs)) ? Number(s.totalOnlineMs) : 0,
    contributionApprovedCount: Number.isFinite(Number(s.contributionApprovedCount)) ? Number(s.contributionApprovedCount) : 0
  };
}

/** 排行榜展示用累计在线时长（毫秒）。 */
export function totalOnlineMsOf(player: PublicPlayer) {
  return safePlayerStats(player).totalOnlineMs || 0;
}

/** 排行榜展示用共建投稿审批通过数（随机任务 + 系列每个子任务各算一条）。 */
export function contributionApprovedCountOf(player: PublicPlayer) {
  return safePlayerStats(player).contributionApprovedCount || 0;
}

export function winRateText(player: PublicPlayer) {
  const stats = safePlayerStats(player);
  const decisive = stats.wins + stats.losses;
  return `${decisive === 0 ? 0 : Math.round((stats.wins / decisive) * 100)}%`;
}

export function isNameWarLoser(player: PublicPlayer) {
  // 失格目标以 nameWarPunished 为准；分值线由服务端按真实分 + penaltyThreshold 判定。
  return Boolean(player.nameWarEnabled && player.nameWarAllowRename && player.nameWarPunished);
}

export function isNameWarLoserVisible(player: PublicPlayer, now = Date.now()) {
  if (!isNameWarLoser(player)) return false;
  if (player.connected) return true;
  return Boolean(player.disconnectedAt && now - player.disconnectedAt <= 1_800_000);
}

export function isExtremeRenameTarget(player: PublicPlayer) {
  return Boolean(player.extremeForceClosed);
}

export function isRenameTarget(player: PublicPlayer) {
  return isNameWarLoser(player) || isExtremeRenameTarget(player);
}

export function isRenameTargetVisible(player: PublicPlayer, now = Date.now()) {
  if (!isRenameTarget(player)) return false;
  if (player.connected) return true;
  return Boolean(player.disconnectedAt && now - player.disconnectedAt <= 1_800_000);
}

export function nameWarRenameQuotaLeft(player: PublicPlayer, now = Date.now()) {
  if (!player.nameWarRenameWindowStartedAt || now - player.nameWarRenameWindowStartedAt >= 10_800_000) return 3;
  return Math.max(0, 3 - (player.nameWarRenameCount || 0));
}

export function rankMultiplierForSettings(settings: Pick<RoomSettings, "enableRanked" | "enableRankMultiplier" | "rankMultiplier">) {
  if (!settings.enableRanked || !settings.enableRankMultiplier) return 1;
  return ([2, 5, 10] as const).includes(settings.rankMultiplier as 2 | 5 | 10) ? settings.rankMultiplier || 1 : 1;
}

export function phaseText(phase: RoomSnapshot["phase"], gameId?: RoomSettings["gameId"]) {
  if (phase === "ready") return "🪑 等待坐满";
  if (phase === "choosing") {
    if (gameId === "liarsdice") return "🎲 叫点中";
    if (gameId === "coinflip") return "🪙 可以抛硬币";
    if (gameId === "othello" || gameId === "tictactoe" || gameId === "gomoku" || gameId === "jungle" || gameId === "chess") return "♟ 对局中";
    return "🤜 出拳中";
  }
  if (phase === "result") return "✨ 结果展示";
  if (phase === "punishment") return "🎲 惩罚阶段";
  return "⏳ 等待中";
}

export function roomStatusText(status: string) {
  if (status === "playing") return "对战中";
  if (status === "punishment") return "惩罚中";
  return "等待中";
}

export function connectionStateText(state: "connected" | "connecting" | "disconnected") {
  if (state === "connected") return "已连接";
  if (state === "connecting") return "连接中";
  return "重连中";
}
