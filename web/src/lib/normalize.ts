/**
 * 状态归一化（B5）：
 * - 服务端 jsonsafe 已保证出站 slice/map 非 null。
 * - 此处负责：DELTA 合并后的薄兜底、lobby chat 本地保留、history 按 id 合并、配置缺字段补全。
 */
import type { AppConfig, LobbySnapshot, PublicPlayer, RoomSnapshot, SeatKey } from "../shared/types";

export function playerSyncKey(player: PublicPlayer) {
  return [
    player.name,
    player.displayName,
    player.avatarUrl || "",
    player.genderId,
    player.genderLabel,
    player.factionId,
    player.connected ? "1" : "0",
    player.disconnectedAt || 0,
    player.disconnectExpiresAt || 0,
    player.stats.wins,
    player.stats.losses,
    player.stats.draws,
    player.stats.punishments,
    player.stats.rankedPoints,
    player.stats.title,
    player.gameStats?.rps?.wins || 0,
    player.gameStats?.rps?.losses || 0,
    player.gameStats?.rps?.draws || 0,
    player.gameStats?.othello?.wins || 0,
    player.gameStats?.othello?.losses || 0,
    player.gameStats?.othello?.draws || 0,
    player.gameStats?.tictactoe?.wins || 0,
    player.gameStats?.tictactoe?.losses || 0,
    player.gameStats?.tictactoe?.draws || 0,
    player.gameStats?.gomoku?.wins || 0,
    player.gameStats?.gomoku?.losses || 0,
    player.gameStats?.gomoku?.draws || 0,
    player.gameStats?.liarsdice?.wins || 0,
    player.gameStats?.liarsdice?.losses || 0,
    player.gameStats?.liarsdice?.draws || 0,
    player.nameWarEnabled ? "1" : "0",
    player.nameWarPunished ? "1" : "0",
    player.nameWarPenaltyName || "",
    player.nameWarAllowRename ? "1" : "0",
    player.nameWarRenameProtectedUntil || 0,
    player.nameWarRenamedBy || "",
    player.nameWarRenamedByName || "",
    player.nameWarRenameWindowStartedAt || 0,
    player.nameWarRenameCount || 0,
    player.giveawayEnabled ? "1" : "0",
    player.giveawayValue || 0,
    player.giveawayClicks || 0,
    player.giveawayBoardText || "",
    player.giveawayBoardExpiresAt || 0,
    player.giveawayBoardLikes || 0,
    player.giveawayBoardDislikes || 0,
    player.giveawayVoteWindowStartedAt || 0,
    player.giveawayVoteCount || 0,
    player.giveawayVoteLikesThisHour || 0,
    player.giveawayVoteDislikesThisHour || 0,
    player.rankMultiplierUnlocked ? "1" : "0",
    player.extremeModeEnabled ? "1" : "0",
    player.extremeModeToggledAt || 0,
    player.extremeModeCooldownUntil || 0,
    player.extremeWinStreak || 0,
    player.extremeLastDecayHour || 0,
    player.extremeForceClosed ? "1" : "0",
    player.extremeForceClosedAt || 0,
    player.extremeRenameProtectedUntil || 0,
    player.extremeRenamedBy || "",
    player.extremeRenamedByName || ""
  ].join("|");
}

export function appendHistoryPage(oldItems: RoomSnapshot["roundHistory"], newItems: RoomSnapshot["roundHistory"], currentItems: RoomSnapshot["roundHistory"]) {
  const seen = new Set([...currentItems, ...oldItems].map((item) => item.id));
  return [...oldItems, ...newItems.filter((item) => {
    if (seen.has(item.id)) return false;
    seen.add(item.id);
    return true;
  })];
}

/** 战绩数字兜底：proto 省略 0 时避免 undefined/NaN */
export function statNum(value: unknown, fallback = 0): number {
  const n = Number(value);
  return Number.isFinite(n) ? n : fallback;
}

export function normalizePublicStats(stats: PublicPlayer["stats"] | null | undefined): PublicPlayer["stats"] {
  const s = stats || ({} as PublicPlayer["stats"]);
  const title = typeof s.title === "string" ? s.title.trim() : "";
  return {
    wins: statNum(s.wins),
    losses: statNum(s.losses),
    draws: statNum(s.draws),
    punishments: statNum(s.punishments),
    rankedPoints: statNum(s.rankedPoints),
    highestScore: statNum(s.highestScore),
    lowestScore: statNum(s.lowestScore),
    sortRankedPoints: statNum(s.sortRankedPoints, statNum(s.rankedPoints)),
    sortHighestScore: statNum(s.sortHighestScore, statNum(s.highestScore)),
    sortLowestScore: statNum(s.sortLowestScore, statNum(s.lowestScore)),
    title: title || "暂无称号",
    ...(s.titleSegmentId ? { titleSegmentId: s.titleSegmentId } : {}),
    ...(s.titleCustom ? { titleCustom: true } : {})
  };
}

export function emptyGameWLD(): PublicPlayer["gameStats"]["rps"] {
  return { wins: 0, losses: 0, draws: 0 };
}

export function normalizeGameWLD(stats: PublicPlayer["gameStats"]["rps"] | null | undefined) {
  const s = stats || emptyGameWLD();
  return { wins: statNum(s.wins), losses: statNum(s.losses), draws: statNum(s.draws) };
}

export function normalizeGameStats(stats: PublicPlayer["gameStats"] | null | undefined): PublicPlayer["gameStats"] {
  const s = stats || ({} as PublicPlayer["gameStats"]);
  return {
    rps: normalizeGameWLD(s.rps),
    othello: normalizeGameWLD(s.othello),
    tictactoe: normalizeGameWLD(s.tictactoe),
    gomoku: normalizeGameWLD(s.gomoku),
    liarsdice: normalizeGameWLD(s.liarsdice)
  };
}

export function totalFromGameStats(gs: PublicPlayer["gameStats"] | null | undefined) {
  const g = normalizeGameStats(gs);
  const parts = [g.rps, g.othello, g.tictactoe, g.gomoku, g.liarsdice];
  return {
    wins: parts.reduce((n, p) => n + p.wins, 0),
    losses: parts.reduce((n, p) => n + p.losses, 0),
    draws: parts.reduce((n, p) => n + p.draws, 0)
  };
}

export function normalizePublicPlayer(player: PublicPlayer): PublicPlayer {
  if (!player) return player;
  const gameStats = normalizeGameStats(player.gameStats);
  const totals = totalFromGameStats(gameStats);
  const stats = normalizePublicStats(player.stats);
  // 展示用总榜优先用分项相加，避免旧数据/广播缺字段时与分项不一致
  return {
    ...player,
    gameStats,
    stats: { ...stats, wins: totals.wins, losses: totals.losses, draws: totals.draws }
  };
}

export function playerWinRate(player: PublicPlayer) {
  const stats = normalizePublicStats(player?.stats);
  const decisive = stats.wins + stats.losses;
  return decisive === 0 ? 0 : stats.wins / decisive;
}

export function rankedPlayers(players: PublicPlayer[]) {
  return players
    .filter((player) => player.connected)
    .map(normalizePublicPlayer)
    .sort((a, b) => b.stats.rankedPoints - a.stats.rankedPoints || b.stats.wins - a.stats.wins);
}

export function normalWinRatePlayers(players: PublicPlayer[]) {
  return players
    .map(normalizePublicPlayer)
    .filter((player) => player.stats.wins + player.stats.losses + player.stats.draws >= 5)
    .sort((a, b) => playerWinRate(b) - playerWinRate(a) || b.stats.wins - a.stats.wins)
    .slice(0, 10);
}

export function replacePlayerInRoom(room: RoomSnapshot, player: PublicPlayer) {
  let changed = false;
  const seats = { ...(room.seats || { A: null, B: null }) };
  for (const seat of ["A", "B"] as SeatKey[]) {
    const occupant = seats[seat];
    if (occupant?.id === player.id && !("isBot" in occupant)) {
      seats[seat] = player;
      changed = true;
    }
  }
  const spectators = (room.spectators || []).map((spectator) => {
    if (spectator.id !== player.id) return spectator;
    changed = true;
    return player;
  });
  return changed ? { ...room, seats, spectators } : room;
}

export function replaceLobbyVersusPlayer(versus: LobbySnapshot["rooms"][number]["versus"], player: PublicPlayer) {
  return {
    A: versus.A && !("isBot" in versus.A) && versus.A.player.id === player.id ? { player } : versus.A,
    B: versus.B && !("isBot" in versus.B) && versus.B.player.id === player.id ? { player } : versus.B
  };
}

export function replacePlayerInLobby(lobby: LobbySnapshot, player: PublicPlayer) {
  const prev = lobby.players || [];
  const exists = prev.some((item) => item.id === player.id);
  // player:batch 可能带来新上线玩家；旧实现只 map 不 insert，导致在线列表/人数不涨。
  const players = exists
    ? prev.map((item) => (item.id === player.id ? player : item))
    : [...prev, player];
  return {
    ...lobby,
    players,
    onlineCount: players.filter((p) => p.connected).length,
    normalLeaderboard: normalWinRatePlayers(players),
    rankedLeaderboard: rankedPlayers(players),
    rooms: (lobby.rooms || []).map((roomInfo) => ({
      ...roomInfo,
      versus: replaceLobbyVersusPlayer(roomInfo.versus || { A: null, B: null }, player)
    }))
  };
}

/** 展示用在线人数：以玩家列表 connected 为准（与 player:batch / lobby 增量保持一致）。 */
export function lobbyOnlineCount(lobby: LobbySnapshot | null | undefined) {
  if (!lobby) return 0;
  const players = lobby.players || [];
  const connected = players.filter((p) => p.connected).length;
  if (connected > 0) return connected;
  return Number(lobby.onlineCount) || 0;
}

export function coerceLobbyPlayers(raw: LobbySnapshot["players"] | Record<string, PublicPlayer> | PublicPlayer[] | null | undefined): PublicPlayer[] {
  if (!raw) return [];
  if (Array.isArray(raw)) return raw as PublicPlayer[];
  return Object.values(raw as Record<string, PublicPlayer>);
}

export function coerceLobbyRooms(raw: LobbySnapshot["rooms"] | Record<string, LobbySnapshot["rooms"][number]> | null | undefined): LobbySnapshot["rooms"] {
  if (!raw) return [] as any;
  if (Array.isArray(raw)) return raw;
  return Object.values(raw as Record<string, any>) as any;
}

export function normalizeLobbySnapshot(snapshot: LobbySnapshot, old?: LobbySnapshot | null) {
  const players = coerceLobbyPlayers(snapshot.players as any).map(normalizePublicPlayer);
  const rooms = coerceLobbyRooms(snapshot.rooms as any).map((room) => ({
    ...room,
    versus: {
      A: room.versus?.A && !("isBot" in room.versus.A) && room.versus.A.player
        ? { player: normalizePublicPlayer(room.versus.A.player) }
        : room.versus?.A ?? null,
      B: room.versus?.B && !("isBot" in room.versus.B) && room.versus.B.player
        ? { player: normalizePublicPlayer(room.versus.B.player) }
        : room.versus?.B ?? null
    }
  }));
  const lobbyChat = (!snapshot.lobbyChat || snapshot.lobbyChat.length === 0) && old ? old.lobbyChat : (snapshot.lobbyChat || []);
  const connected = players.filter((p) => p.connected).length;
  // 服务端 onlineCount 与 players 列表偶发短暂不一致时，以列表 connected 为准。
  const onlineCount = connected > 0 ? connected : (Number(snapshot.onlineCount) || 0);
  return {
    ...snapshot,
    players,
    rooms,
    onlineCount,
    suggestions: snapshot.suggestions || [],
    lobbyChat,
    normalLeaderboard: normalWinRatePlayers(players),
    rankedLeaderboard: rankedPlayers(players)
  } as LobbySnapshot;
}

/** 兜底：Go nil slice/map → JSON null 时，避免前端 .includes/.map 白屏 */
export function normalizeRoundHistoryItem(item: RoomSnapshot["roundHistory"][number]): RoomSnapshot["roundHistory"][number] {
  return {
    ...item,
    punishmentTasks: item.punishmentTasks || [],
    punishedNames: item.punishedNames || [],
    proofs: item.proofs || [],
    tictactoeLine: item.tictactoeLine || undefined,
    gomokuLine: item.gomokuLine || undefined
  };
}

/** 合并对局记录：服务端 recent history 覆盖同 id 项（任务/证明更新），并保留本地已加载的更早页。 */
export function mergeRoundHistory(
  oldItems: RoomSnapshot["roundHistory"] | undefined,
  nextItems: RoomSnapshot["roundHistory"] | undefined
): RoomSnapshot["roundHistory"] {
  const old = oldItems || [];
  const next = (nextItems || []).map(normalizeRoundHistoryItem);
  if (!next.length) return old;
  const nextIds = new Set(next.map((item) => item.id));
  const rest = old.filter((item) => !nextIds.has(item.id));
  return [...next, ...rest];
}

function fixPosList(list: Array<{ row?: number; col?: number }> | undefined | null): Array<{ row: number; col: number }> {
  if (!list?.length) return [];
  return list.map((p) => {
    const row = Number(p?.row);
    const col = Number(p?.col);
    return {
      row: Number.isFinite(row) ? row : 0,
      col: Number.isFinite(col) ? col : 0
    };
  });
}

function padBoard<T>(board: (T | null)[][] | undefined | null, n: number): (T | null)[][] {
  const out: (T | null)[][] = [];
  for (let r = 0; r < n; r++) {
    const src = board?.[r] || [];
    const row: (T | null)[] = [];
    for (let c = 0; c < n; c++) row.push(c < src.length ? (src[c] ?? null) : null);
    out.push(row);
  }
  return out;
}

export function normalizeOthello(state: RoomSnapshot["othello"]): RoomSnapshot["othello"] {
  if (!state) return state;
  return {
    ...state,
    // protobufjs 可能丢掉 int 0，合法手/棋盘必须归一
    legalMoves: fixPosList(state.legalMoves),
    settlementEvents: state.settlementEvents || [],
    rankedDelta: state.rankedDelta || { A: 0, B: 0 },
    blackCount: Number(state.blackCount) || 0,
    whiteCount: Number(state.whiteCount) || 0,
    passCount: Number(state.passCount) || 0,
    board: padBoard(state.board, 8),
    moveDeadlineAt: Number(state.moveDeadlineAt) || 0,
    clockDeadlineAt: Number(state.clockDeadlineAt) || 0,
    clockRemaining: state.clockRemaining || { A: 0, B: 0 }
  };
}

export function normalizeTicTacToe(state: RoomSnapshot["tictactoe"]): RoomSnapshot["tictactoe"] {
  if (!state) return state;
  return {
    ...state,
    winningLine: state.winningLine?.length ? fixPosList(state.winningLine) : undefined,
    rankedDelta: state.rankedDelta || { A: 0, B: 0 },
    moveCount: Number(state.moveCount) || 0,
    board: padBoard(state.board, 3)
  };
}

export function normalizeGomoku(state: RoomSnapshot["gomoku"]): RoomSnapshot["gomoku"] {
  if (!state) return state;
  return {
    ...state,
    moves: fixPosList(state.moves),
    winningLine: state.winningLine?.length ? fixPosList(state.winningLine) : undefined,
    rankedDelta: state.rankedDelta || { A: 0, B: 0 },
    undoCount: state.undoCount || { A: 0, B: 0 },
    moveCount: Number(state.moveCount) || 0,
    board: padBoard(state.board, 15),
    moveDeadlineAt: Number(state.moveDeadlineAt) || 0,
    clockDeadlineAt: Number(state.clockDeadlineAt) || 0,
    clockRemaining: state.clockRemaining || { A: 0, B: 0 }
  };
}

export function normalizeLiarsDice(state: RoomSnapshot["liarsDice"]): RoomSnapshot["liarsDice"] {
  if (!state) return state;
  return {
    ...state,
    participantIds: state.participantIds || [],
    readyPlayerIds: state.readyPlayerIds || [],
    diceCounts: state.diceCounts || {},
    bidHistory: state.bidHistory || [],
    roundNumber: Number(state.roundNumber) || 0,
    actualCount: Number(state.actualCount) || 0,
    currentBid: state.currentBid || null
  };
}

export function normalizeRoomSnapshot(room: RoomSnapshot): RoomSnapshot {
  const seats = room.seats || { A: null, B: null };
  return {
    ...room,
    settings: {
      ...room.settings,
      tags: room.settings?.tags || [],
      punishmentIds: room.settings?.punishmentIds || []
    },
    spectators: room.spectators || [],
    punishedPlayerIds: room.punishedPlayerIds || [],
    proofs: room.proofs || [],
    roundHistory: (room.roundHistory || []).map(normalizeRoundHistoryItem),
    chat: room.chat || [],
    choices: room.choices || {},
    ready: room.ready || { A: false, B: false },
    score: room.score || { A: 0, B: 0 },
    seatedScore: room.seatedScore || { A: 0, B: 0 },
    seatStats: room.seatStats || {
      A: { wins: 0, losses: 0, draws: 0, punishments: 0 },
      B: { wins: 0, losses: 0, draws: 0, punishments: 0 }
    },
    seats: { A: seats.A ?? null, B: seats.B ?? null },
    othello: normalizeOthello(room.othello),
    tictactoe: normalizeTicTacToe(room.tictactoe),
    liarsDice: normalizeLiarsDice(room.liarsDice),
    gomoku: normalizeGomoku(room.gomoku)
  };
}

/** 与 config/ranked-score.json 默认值一致；proto 漏字段 / 旧包时补齐，避免 config.rankedScore 为 undefined。 */
export const DEFAULT_RANKED_SCORE: AppConfig["rankedScore"] = {
  max: 4999,
  min: -4999,
  nameWarMin: -9999,
  dailyDecayRatio: 0.98
};

/** 与 config/name-war.json 的 penaltyThreshold 默认一致。 */
export const DEFAULT_NAME_WAR_PENALTY_THRESHOLD = -4999;

/** 与 config/access-control.json 默认值一致（后台编辑时也用同一份兜底）。 */
export const DEFAULT_ACCESS_CONTROL: AppConfig["accessControl"] = {
  maxOnlinePerIp: 3,
  maxCreatesPer10Min: 5,
  ipBackstopMultiplier: 6,
  ipBackstopMinLimit: 30,
  maxSessionIssuePerIp: 30,
  maxOnlinePerIpTotal: 10,
  maxCreatesPerIp: 15,
  maxActiveRoomsPerOwner: 3,
  maxProofUploadsPerPlayer: 8,
  registrationDisabled: false
};

/** 合并排位分展示配置；缺对象或子字段时用默认值（负分用 ?? 不能用 ||）。 */
export function withRankedScoreDefaults(rs?: Partial<AppConfig["rankedScore"]> | null): AppConfig["rankedScore"] {
  return {
    max: rs?.max ?? DEFAULT_RANKED_SCORE.max,
    min: rs?.min ?? DEFAULT_RANKED_SCORE.min,
    nameWarMin: rs?.nameWarMin ?? DEFAULT_RANKED_SCORE.nameWarMin,
    dailyDecayRatio: rs?.dailyDecayRatio ?? DEFAULT_RANKED_SCORE.dailyDecayRatio
  };
}

export function withAccessControlDefaults(ac?: Partial<AppConfig["accessControl"]> | null): AppConfig["accessControl"] {
  return { ...DEFAULT_ACCESS_CONTROL, ...ac };
}

/** 与 config/giveaway.json 默认值一致（后台编辑时也用同一份兜底）。 */
export const DEFAULT_GIVEAWAY: AppConfig["giveaway"] = {
  panelTitle: "白给自救板",
  panelDescription: "提交一点自我惩罚宣言，等待其他玩家点赞帮你降低白给值。",
  submitPlaceholder: "写下你的自我惩罚宣言...",
  emptyText: "还没有人在白给自救板上。",
  activeBoostValue: 2,
  winPenaltyValue: 1,
  likeVoteLimitPerHour: 3,
  likeVoteValue: 1,
  dislikeVoteLimitPerHour: 10,
  dislikeVoteValue: 0.1
};

export function withGiveawayDefaults(g?: Partial<AppConfig["giveaway"]> | null): AppConfig["giveaway"] {
  return { ...DEFAULT_GIVEAWAY, ...g };
}

export function normalizeConfig(config: AppConfig): AppConfig {
  return {
    ...config,
    genders: config.genders || [],
    genderFactions: (config.genderFactions || []).map((faction) => ({
      ...faction,
      taskGroup: faction.taskGroup || "default"
    })),
    titles: (config.titles || []).map((segment) => ({
      ...segment,
      // 百分比分段：旧绝对分字段或漏字段时至少保证数字存在，避免后台/展示读 undefined。
      minPercent: segment.minPercent ?? 0,
      maxPercent: segment.maxPercent ?? 0,
      names: segment.names || [],
      factionNames: segment.factionNames || {}
    })),
    punishments: (config.punishments || []).map((punishment) => ({
      ...punishment,
      variants: punishment.variants || {},
      roomBackgroundImages: punishment.roomBackgroundImages || [],
      tasks: (punishment.tasks || []).map((task) => ({
        ...task,
        variants: task.variants || {},
        backgroundImages: task.backgroundImages || []
      }))
    })),
    roomTags: config.roomTags || [],
    roomInfoTags: config.roomInfoTags || {},
    accessControl: withAccessControlDefaults(config.accessControl),
    nameWar: {
      ...config.nameWar,
      penaltyThreshold: config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD
    },
    giveaway: withGiveawayDefaults(config.giveaway),
    bots: {
      names: config.bots?.names || [],
      difficulties: config.bots?.difficulties || []
    },
    games: config.games || [],
    messages: config.messages || {},
    // 与 extremeMode 同模式：整段对象 + 内部 map 缺省，保证业务代码可直接点字段。
    extremeMode: {
      ...config.extremeMode,
      positiveLossRates: config.extremeMode?.positiveLossRates || {},
      negativeWinRates: config.extremeMode?.negativeWinRates || {},
      hourlyDecay: config.extremeMode?.hourlyDecay || {}
    },
    rankedScore: withRankedScoreDefaults(config.rankedScore)
  };
}
