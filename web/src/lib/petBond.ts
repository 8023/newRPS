// 认主认宠展示辅助：候选/成员行的兜底用户信息拼装。源：ui/AppViews.tsx:4288-4358。
import type { PetBondBadgeFields, PetBondState, PublicPlayer } from "../shared/types";
import { withPetBondDefaults } from "./normalize";

export const emptyPetBondState = (): PetBondState => ({
  masters: [],
  pets: [],
  masterCandidates: [],
  petCandidates: [],
  incoming: [],
  outgoing: [],
  chains: [],
  config: withPetBondDefaults(null)
});

/** 从大厅玩家列表中解析完整用户信息，供徽章展示。 */
export function resolveLobbyPlayer(players: PublicPlayer[], id: string, fallback?: Partial<PublicPlayer>): PublicPlayer {
  const found = players.find((p) => p.id === id);
  if (found) return found;
  return {
    id,
    name: fallback?.name || "未知玩家",
    genderId: fallback?.genderId || "",
    genderLabel: fallback?.genderLabel || "",
    factionId: fallback?.factionId || "",
    factionLabel: fallback?.factionLabel || "",
    factionColors: fallback?.factionColors || { textColor: "#243447", backgroundColor: "#eef6fc", borderColor: "#b9dcf4" },
    displayName: fallback?.displayName || fallback?.name || "未知玩家",
    avatarUrl: fallback?.avatarUrl,
    connected: Boolean(fallback?.connected),
    stats: fallback?.stats || {
      wins: 0, losses: 0, draws: 0, punishments: 0,
      rankedPoints: 0, highestScore: 0, lowestScore: 0,
      sortRankedPoints: 0, sortHighestScore: 0, sortLowestScore: 0,
      title: "暂无称号"
    },
    gameStats: fallback?.gameStats || {
      rps: { wins: 0, losses: 0, draws: 0 },
      othello: { wins: 0, losses: 0, draws: 0 },
      tictactoe: { wins: 0, losses: 0, draws: 0 },
      gomoku: { wins: 0, losses: 0, draws: 0 },
      liarsdice: { wins: 0, losses: 0, draws: 0 },
      jungle: { wins: 0, losses: 0, draws: 0 },
      chess: { wins: 0, losses: 0, draws: 0 }
    }
  };
}

/** 认主/认宠成员/候选兜底信息转换为 resolveLobbyPlayer 的 fallback：即便对方当前离线，
 * 性别/称号/⚡极限-⚔️名争模式/白给徽标也要用后端随成员一起下发的快照值，不能只留名字/头像。 */
export function petBondBadgeFallback(m: { name: string; displayName: string; avatarUrl?: string; connected: boolean } & PetBondBadgeFields): Partial<PublicPlayer> {
  return {
    name: m.name,
    displayName: m.displayName,
    avatarUrl: m.avatarUrl,
    connected: m.connected,
    genderId: m.genderId,
    genderLabel: m.genderLabel,
    factionLabel: m.factionLabel,
    factionColors: m.factionColors,
    extremeModeEnabled: m.extremeModeEnabled,
    nameWarEnabled: m.nameWarEnabled,
    nameWarPunished: m.nameWarPunished,
    nameWarPenaltyName: m.nameWarPenaltyName,
    giveawayEnabled: m.giveawayEnabled,
    giveawayValue: m.giveawayValue,
    stats: {
      wins: 0, losses: 0, draws: 0, punishments: 0,
      rankedPoints: 0, highestScore: 0, lowestScore: 0,
      sortRankedPoints: 0, sortHighestScore: 0, sortLowestScore: 0,
      title: m.title || "暂无称号",
      titleColors: m.titleColors
    }
  };
}
