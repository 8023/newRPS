import type { AppConfig, PublicPlayer, RoomSnapshot } from "../shared/types";

/** 房间开了系列惩罚且我的阵营不在目标集合内时，返回阻塞确认文案。 */
export function seriesFactionWarning(
  room: Pick<RoomSnapshot, "settings">,
  config: Pick<AppConfig, "punishmentSeriesSummaries" | "genderFactions">,
  me: Pick<PublicPlayer, "factionId">,
): string | null {
  if (room.settings.punishmentSource !== "series") return null;
  const seriesId = room.settings.punishmentSeriesId;
  if (!seriesId) return null;
  const series = (config.punishmentSeriesSummaries || []).find((item) => item.id === seriesId);
  if (!series) return null;
  const targets = series.targetFactionIds || [];
  if (targets.length === 0) return null;
  if (targets.includes(me.factionId)) return null;
  const labels = targets.map((id) => config.genderFactions.find((f) => f.id === id)?.label || id);
  return `这个系列是给「${labels.join("、")}」准备的，你的阵营不在其中——轮到你受罚时会临时抽一条随机任务顶上，剧情会断。仍然要坐下吗？`;
}
