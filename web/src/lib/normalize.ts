/**
 * 状态归一化（B5）：
 * - 服务端 jsonsafe 已保证出站 slice/map 非 null。
 * - 此处负责：DELTA 合并后的薄兜底、lobby chat 本地保留、history 按 id 合并、配置缺字段补全。
 */
import type { AppConfig, GameId, LobbySnapshot, PublicPlayer, RoomSnapshot, SeatKey } from "../shared/types";

/** 游戏缺省排位档位（与服务端 types.DefaultStakeTiers 镜像）：黑白棋按每子，其余整局。 */
export function defaultStakeTiers(gameId: GameId): number[] {
  return gameId === "othello" ? [1, 2, 5, 10] : [5, 10, 20];
}

/** 取某游戏可选排位档位；旧后端 config.games 无 stakes 字段时回退默认表。 */
export function stakeTiersFor(config: Pick<AppConfig, "games">, gameId: GameId): number[] {
  const tiers = config.games?.find((game) => game.id === gameId)?.stakes;
  return tiers?.length ? tiers : defaultStakeTiers(gameId);
}

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
    player.gameStats?.jungle?.wins || 0,
    player.gameStats?.jungle?.losses || 0,
    player.gameStats?.jungle?.draws || 0,
    player.gameStats?.chess?.wins || 0,
    player.gameStats?.chess?.losses || 0,
    player.gameStats?.chess?.draws || 0,
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
    player.extremeRenamedByName || "",
    player.bondMasterEnabled ? "1" : "0",
    player.bondPetEnabled ? "1" : "0",
    player.bondPublicDisplay ? "1" : "0"
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
    selfTitle: typeof s.selfTitle === "string" ? s.selfTitle : "",
    totalOnlineMs: statNum(s.totalOnlineMs),
    contributionApprovedCount: statNum(s.contributionApprovedCount),
    titleSource: s.titleSource || "system",
    titleColors: s.titleColors,
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
    liarsdice: normalizeGameWLD(s.liarsdice),
    jungle: normalizeGameWLD(s.jungle),
    chess: normalizeGameWLD(s.chess)
  };
}

export function totalFromGameStats(gs: PublicPlayer["gameStats"] | null | undefined) {
  const g = normalizeGameStats(gs);
  const parts = [g.rps, g.othello, g.tictactoe, g.gomoku, g.liarsdice, g.jungle, g.chess];
  return {
    wins: parts.reduce((n, p) => n + p.wins, 0),
    losses: parts.reduce((n, p) => n + p.losses, 0),
    draws: parts.reduce((n, p) => n + p.draws, 0)
  };
}

/**
 * 将大厅玩家视图合并回登录时取得的完整玩家资料。
 * 实时大厅快照会省略 GameStats；protobuf 物化后无法再区分“省略”与“全零”，
 * 因此当补丁没有任何分游戏记录时，保留当前完整资料里的分项战绩。
 */
export function mergeLobbyPlayerIntoFullPlayer(current: PublicPlayer, incoming: PublicPlayer): PublicPlayer {
  const totals = totalFromGameStats(incoming.gameStats);
  const incomingHasGameStats = totals.wins + totals.losses + totals.draws > 0;
  return {
    ...current,
    ...incoming,
    gameStats: incomingHasGameStats ? incoming.gameStats : current.gameStats
  };
}

export function normalizePublicPlayer(player: PublicPlayer): PublicPlayer {
  if (!player) return player;
  const gameStats = normalizeGameStats(player.gameStats);
  const totals = totalFromGameStats(gameStats);
  const stats = normalizePublicStats(player.stats);
  // 大厅实时通道会省略 GameStats（ToLiveLobbyPlayer）；此时保留服务端 Stats 合计，
  // 避免用全零分项把 wins/losses/draws 抹成 0。有分项数据时仍以分项之和为准。
  const hasGameData = totals.wins + totals.losses + totals.draws > 0;
  return {
    ...player,
    gameStats,
    stats: hasGameData
      ? { ...stats, wins: totals.wins, losses: totals.losses, draws: totals.draws }
      : stats
  };
}

export function playerWinRate(player: PublicPlayer) {
  const stats = normalizePublicStats(player?.stats);
  const decisive = stats.wins + stats.losses;
  return decisive === 0 ? 0 : stats.wins / decisive;
}

/** 在线积分榜：调用方应已 normalize 过玩家；此处不再对每人做一遍 normalizePublicPlayer。 */
export function rankedPlayers(players: PublicPlayer[]) {
  return players
    .filter((player) => player.connected)
    .slice()
    .sort((a, b) => {
      const ar = a.stats?.rankedPoints || 0;
      const br = b.stats?.rankedPoints || 0;
      if (br !== ar) return br - ar;
      return (b.stats?.wins || 0) - (a.stats?.wins || 0);
    });
}

/** 普通胜率榜 Top10：同样假定 players 已规范化。 */
export function normalWinRatePlayers(players: PublicPlayer[]) {
  return players
    .filter((player) => {
      const s = player.stats;
      return (s?.wins || 0) + (s?.losses || 0) + (s?.draws || 0) >= 5;
    })
    .slice()
    .sort((a, b) => playerWinRate(b) - playerWinRate(a) || (b.stats?.wins || 0) - (a.stats?.wins || 0))
    .slice(0, 10);
}

export function replacePlayerInRoom(room: RoomSnapshot, player: PublicPlayer) {
  return replacePlayersInRoom(room, [player]);
}

/** 批量把补丁合进房间座位/观战列表；单次遍历 seats + spectators。 */
export function replacePlayersInRoom(room: RoomSnapshot, patches: PublicPlayer[]): RoomSnapshot {
  if (!patches?.length) return room;
  const byId = new Map(patches.map((p) => [p.id, p]));
  let changed = false;
  const seats = { ...(room.seats || { A: null, B: null }) };
  for (const seat of ["A", "B"] as SeatKey[]) {
    const occupant = seats[seat];
    if (occupant?.id && byId.has(occupant.id)) {
      seats[seat] = byId.get(occupant.id)!;
      changed = true;
    }
  }
  const spectators = (room.spectators || []).map((spectator) => {
    const next = byId.get(spectator.id);
    if (!next) return spectator;
    changed = true;
    return next;
  });
  return changed ? { ...room, seats, spectators } : room;
}

/**
 * 批量合并 player:batch 到大厅。
 * 旧路径对每个 patch 各做一次 O(P log P) 双榜重排 + O(R) versus 扫描；
 * 这里合并一次列表、扫一遍房间、重排一次双榜：O(P + R + k + P log P)。
 * patches 应由调用方 normalize（App 的 applyPlayerPatches 已做）。
 */
export function replacePlayersInLobby(lobby: LobbySnapshot, patches: PublicPlayer[]): LobbySnapshot {
  if (!patches?.length) return lobby;
  const byId = new Map<string, PublicPlayer>();
  for (const raw of patches) {
    if (!raw?.id) continue;
    byId.set(raw.id, raw);
  }
  if (!byId.size) return lobby;

  const prev = lobby.players || [];
  const seen = new Set<string>();
  const players: PublicPlayer[] = new Array(prev.length);
  let write = 0;
  for (const item of prev) {
    const next = byId.get(item.id);
    if (next) {
      players[write++] = next;
      seen.add(item.id);
    } else {
      players[write++] = item;
    }
  }
  players.length = write;
  for (const [id, p] of byId) {
    if (!seen.has(id)) players.push(p);
  }

  let onlineCount = 0;
  for (const p of players) {
    if (p.connected) onlineCount++;
  }

  const rooms = (lobby.rooms || []).map((roomInfo) => {
    const versus = roomInfo.versus || { A: null, B: null };
    const aId = versus.A?.player?.id;
    const bId = versus.B?.player?.id;
    const aNext = aId ? byId.get(aId) : undefined;
    const bNext = bId ? byId.get(bId) : undefined;
    if (!aNext && !bNext) return roomInfo;
    return {
      ...roomInfo,
      versus: {
        A: aNext ? { player: aNext } : versus.A,
        B: bNext ? { player: bNext } : versus.B
      }
    };
  });

  return {
    ...lobby,
    players,
    rooms,
    onlineCount,
    normalLeaderboard: normalWinRatePlayers(players),
    rankedLeaderboard: rankedPlayers(players)
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
    // 最终展示边界再做一次有限数字兜底，兼容旧服务端或已在浏览器中缓存的
    // 缺字段快照，避免任何大厅/后台房间卡片出现 undefined/NaN。
    players: statNum(room.players),
    spectators: statNum(room.spectators),
    punishmentSource: room.punishmentSource === "system" ? "random" : (room.punishmentSource || "random"),
    punishmentTagsIncluded: room.punishmentTagsIncluded || [],
    punishmentTagsExcluded: room.punishmentTagsExcluded || [],
    punishmentSeriesId: room.punishmentSeriesId || "",
    versus: {
      A: room.versus?.A?.player
        ? { player: normalizePublicPlayer(room.versus.A.player) }
        : null,
      B: room.versus?.B?.player
        ? { player: normalizePublicPlayer(room.versus.B.player) }
        : null
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
  return padBoardRect(board, n, n);
}

function padBoardRect<T>(board: (T | null)[][] | undefined | null, rows: number, cols: number): (T | null)[][] {
  const out: (T | null)[][] = [];
  for (let r = 0; r < rows; r++) {
    const src = board?.[r] || [];
    const row: (T | null)[] = [];
    for (let c = 0; c < cols; c++) row.push(c < src.length ? (src[c] ?? null) : null);
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
    undoCount: state.undoCount || { A: 0, B: 0 },
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

export function normalizeJungle(state: RoomSnapshot["jungle"]): RoomSnapshot["jungle"] {
  if (!state) return state;
  return {
    ...state,
    rankedDelta: state.rankedDelta || { A: 0, B: 0 },
    undoCount: state.undoCount || { A: 0, B: 0 },
    moveCount: Number(state.moveCount) || 0,
    board: padBoardRect(state.board, 9, 7),
    moveDeadlineAt: Number(state.moveDeadlineAt) || 0,
    clockDeadlineAt: Number(state.clockDeadlineAt) || 0,
    clockRemaining: state.clockRemaining || { A: 0, B: 0 }
  };
}

export function normalizeChess(state: RoomSnapshot["chess"]): RoomSnapshot["chess"] {
  if (!state) return state;
  return {
    ...state,
    rankedDelta: state.rankedDelta || { A: 0, B: 0 },
    undoCount: state.undoCount || { A: 0, B: 0 },
    moveCount: Number(state.moveCount) || 0,
    board: padBoardRect(state.board, 8, 8),
    moveDeadlineAt: Number(state.moveDeadlineAt) || 0,
    clockDeadlineAt: Number(state.clockDeadlineAt) || 0,
    clockRemaining: state.clockRemaining || { A: 0, B: 0 },
    halfmoveClock: Number(state.halfmoveClock) || 0,
    legalMoves: Array.isArray(state.legalMoves) ? state.legalMoves : []
  };
}

export function normalizeCoinFlip(state: RoomSnapshot["coinFlip"]): RoomSnapshot["coinFlip"] {
  if (!state) return state;
  return {
    guess: state.guess || "",
    result: state.result || "",
    correct: Boolean(state.correct),
    settledAt: Number(state.settledAt) || 0
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
      punishmentIds: room.settings?.punishmentIds || [],
      punishmentTagsIncluded: room.settings?.punishmentTagsIncluded || [],
      punishmentTagsExcluded: room.settings?.punishmentTagsExcluded || [],
      punishmentSeriesId: room.settings?.punishmentSeriesId || "",
      punishmentSource: room.settings?.punishmentSource === "system"
        ? "random"
        : (room.settings?.punishmentSource || "random")
    },
    spectators: room.spectators || [],
    punishedPlayerIds: room.punishedPlayerIds || [],
    proofs: room.proofs || [],
    roundHistory: (room.roundHistory || []).map(normalizeRoundHistoryItem),
    roundHistoryTotal: room.roundHistoryTotal || 0,
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
    gomoku: normalizeGomoku(room.gomoku),
    jungle: normalizeJungle(room.jungle),
    chess: normalizeChess(room.chess),
    coinFlip: normalizeCoinFlip(room.coinFlip)
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

/** 与 config/name-war.json 的 renameMinPoints 默认一致。 */
export const DEFAULT_NAME_WAR_RENAME_MIN_POINTS = 500;

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
  dislikeVoteValue: 0.1,
  petLikeVoteLimitPerHour: 5,
  petLikeVoteValue: 1,
  petDislikeVoteLimitPerHour: 15,
  petDislikeVoteValue: 0.1,
  masterLikeVoteLimitPerHour: 5,
  masterLikeVoteValue: 1,
  masterDislikeVoteLimitPerHour: 15,
  masterDislikeVoteValue: 0.1
};

export function withGiveawayDefaults(g?: Partial<AppConfig["giveaway"]> | null): AppConfig["giveaway"] {
  const base = { ...DEFAULT_GIVEAWAY, ...(g || {}) };
  // 0/NaN 不是合法配置（服务端 Validate 拒绝），多为 proto 漏字段或 defaults:false 后的零值，回退默认。
  if (!(base.activeBoostValue > 0)) base.activeBoostValue = DEFAULT_GIVEAWAY.activeBoostValue;
  if (!(base.winPenaltyValue > 0)) base.winPenaltyValue = DEFAULT_GIVEAWAY.winPenaltyValue;
  if (!(base.likeVoteLimitPerHour >= 1)) base.likeVoteLimitPerHour = DEFAULT_GIVEAWAY.likeVoteLimitPerHour;
  if (!(base.likeVoteValue > 0)) base.likeVoteValue = DEFAULT_GIVEAWAY.likeVoteValue;
  if (!(base.dislikeVoteLimitPerHour >= 1)) base.dislikeVoteLimitPerHour = DEFAULT_GIVEAWAY.dislikeVoteLimitPerHour;
  if (!(base.dislikeVoteValue > 0)) base.dislikeVoteValue = DEFAULT_GIVEAWAY.dislikeVoteValue;
  if (!(base.petLikeVoteLimitPerHour >= 1)) base.petLikeVoteLimitPerHour = DEFAULT_GIVEAWAY.petLikeVoteLimitPerHour;
  if (!(base.petLikeVoteValue > 0)) base.petLikeVoteValue = DEFAULT_GIVEAWAY.petLikeVoteValue;
  if (!(base.petDislikeVoteLimitPerHour >= 1)) base.petDislikeVoteLimitPerHour = DEFAULT_GIVEAWAY.petDislikeVoteLimitPerHour;
  if (!(base.petDislikeVoteValue > 0)) base.petDislikeVoteValue = DEFAULT_GIVEAWAY.petDislikeVoteValue;
  if (!(base.masterLikeVoteLimitPerHour >= 1)) base.masterLikeVoteLimitPerHour = DEFAULT_GIVEAWAY.masterLikeVoteLimitPerHour;
  if (!(base.masterLikeVoteValue > 0)) base.masterLikeVoteValue = DEFAULT_GIVEAWAY.masterLikeVoteValue;
  if (!(base.masterDislikeVoteLimitPerHour >= 1)) base.masterDislikeVoteLimitPerHour = DEFAULT_GIVEAWAY.masterDislikeVoteLimitPerHour;
  if (!(base.masterDislikeVoteValue > 0)) base.masterDislikeVoteValue = DEFAULT_GIVEAWAY.masterDislikeVoteValue;
  return base;
}

export const DEFAULT_PET_BOND: AppConfig["petBond"] = {
  panelTitle: "宠物乐园",
  maxPetsPerMaster: 3,
  maxMastersPerPet: 3,
  maxTitleLength: 12
};

export function withPetBondDefaults(p?: Partial<AppConfig["petBond"]> | null): AppConfig["petBond"] {
  const base = { ...DEFAULT_PET_BOND, ...(p || {}) };
  if (!(base.maxPetsPerMaster >= 1)) base.maxPetsPerMaster = DEFAULT_PET_BOND.maxPetsPerMaster;
  if (!(base.maxMastersPerPet >= 1)) base.maxMastersPerPet = DEFAULT_PET_BOND.maxMastersPerPet;
  if (!(base.maxTitleLength >= 1)) base.maxTitleLength = DEFAULT_PET_BOND.maxTitleLength;
  if (!base.panelTitle?.trim()) base.panelTitle = DEFAULT_PET_BOND.panelTitle;
  return base;
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
    punishments: [],
    punishmentTags: (config.punishmentTags || []).map((tag) => ({
      ...tag,
      roomBackgroundImages: tag.roomBackgroundImages || []
    })),
    punishmentSeriesSummaries: (config.punishmentSeriesSummaries || []).map((series) => ({
      ...series,
      roomBackgroundImages: series.roomBackgroundImages || [],
      stepCount: series.stepCount ?? 0,
      targetFactionIds: series.targetFactionIds || []
    })),
    punishmentRandomSettings: {
      orderStep: config.punishmentRandomSettings?.orderStep ?? 2,
      maxDifficultyOvershoot: config.punishmentRandomSettings?.maxDifficultyOvershoot ?? 5,
      minSeriesSteps: (config.punishmentRandomSettings?.minSeriesSteps ?? 5) > 0 ? (config.punishmentRandomSettings?.minSeriesSteps ?? 5) : 5,
      maxSeriesSteps: (config.punishmentRandomSettings?.maxSeriesSteps ?? 20) > 0 ? (config.punishmentRandomSettings?.maxSeriesSteps ?? 20) : 20
    },
    roomTags: config.roomTags || [],
    roomInfoTags: config.roomInfoTags || {},
    titleTagStyles: config.titleTagStyles || {},
    accessControl: withAccessControlDefaults(config.accessControl),
    nameWar: {
      ...config.nameWar,
      penaltyThreshold: config.nameWar?.penaltyThreshold ?? DEFAULT_NAME_WAR_PENALTY_THRESHOLD,
      renameMinPoints: config.nameWar?.renameMinPoints ?? DEFAULT_NAME_WAR_RENAME_MIN_POINTS
    },
    giveaway: withGiveawayDefaults(config.giveaway),
    petBond: withPetBondDefaults(config.petBond),
    games: (config.games || []).map((game) => ({ ...game, stakes: game.stakes?.length ? game.stakes : defaultStakeTiers(game.id) })),
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
