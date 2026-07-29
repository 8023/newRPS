export type GenderColors = {
  textColor: string;
  backgroundColor: string;
  borderColor: string;
};
export type GenderOption = { id: string; label: string; factionId: string };
// taskGroup 决定系统任务/称号取哪份阵营专属文案，固定三选一：
// "male"（生理男：顺性别男、男跨女）/ "female"（生理女：顺性别女、女跨男）/ "default"（默认兜底：无性别）。
export type GenderFaction = GenderColors & {
  id: string;
  label: string;
  taskGroup: string;
};
export type Move = "rock" | "scissors" | "paper" | "giveaway" | "forfeit" | "noMove";
export type RoundResult = "A" | "B" | "draw" | "doubleLoss";
export type GamePhase = "waiting" | "ready" | "choosing" | "result" | "punishment";
export type SeatKey = "A" | "B";
export type GameId = "rps" | "othello" | "tictactoe" | "liarsdice" | "gomoku" | "jungle";
export type RankStake = 1 | 2 | 5 | 10 | 20;
export type OthelloCell = "black" | "white" | null;
export type TicTacToeCell = "X" | "O" | null;
export type TicTacToeState = {
  board: TicTacToeCell[][];
  turn: SeatKey;
  xSeat: SeatKey;
  moveCount: number;
  giveawayPrompt?: {
    seat: SeatKey;
    forced?: boolean;
    startedAt: number;
    expiresAt: number;
  };
  winningLine?: Array<{ row: number; col: number }>;
  rankedDelta?: Record<SeatKey, number>;
  ended?: boolean;
  winner?: RoundResult;
};
export type OthelloState = {
  board: OthelloCell[][];
  turn: SeatKey;
  blackSeat: SeatKey;
  legalMoves: Array<{ row: number; col: number }>;
  passCount: number;
  blackCount: number;
  whiteCount: number;
  rankedDelta?: Record<SeatKey, number>;
  settlementEvents?: string[];
  pendingSettlement?: {
    id: string;
    seat: SeatKey;
    opponentSeat: SeatKey;
    flips: number;
    stake: number;
    nextTurn: SeatKey;
    expiresAt: number;
    forced?: "giveaway" | "tribute";
    forcedByMasterName?: string;
    resolvedAs?: "normal" | "giveaway" | "tribute";
  };
  surrenderRequest?: {
    fromSeat: SeatKey;
    toSeat: SeatKey;
    createdAt: number;
  };
  // 计时：0/缺省表示对应计时器未启用或当前无人在计时（如白给/上贡结算等待窗口期间）。
  // clockRemaining 是双方总时长剩余（毫秒）：非当前落子方为冻结静态值，当前落子方需用
  // clockDeadlineAt 在前端本地倒算实时剩余，不依赖服务器逐秒推送。
  moveDeadlineAt?: number;
  clockDeadlineAt?: number;
  clockRemaining?: Record<SeatKey, number>;
  ended?: boolean;
  winner?: RoundResult;
};
export type GomokuCell = "black" | "white" | null;
export type GomokuState = {
  board: GomokuCell[][];
  turn: SeatKey;
  blackSeat: SeatKey;
  moveCount: number;
  moves: Array<{ row: number; col: number }>;
  winningLine?: Array<{ row: number; col: number }>;
  rankedDelta?: Record<SeatKey, number>;
  undoCount?: Record<SeatKey, number>;
  undoRequest?: {
    fromSeat: SeatKey;
    toSeat: SeatKey;
    createdAt: number;
    expiresAt: number;
  };
  resignRequest?: {
    fromSeat: SeatKey;
    toSeat: SeatKey;
    createdAt: number;
  };
  giveawaySeat?: SeatKey;
  giveawayForcedByMasterName?: string;
  ended?: boolean;
  winner?: RoundResult;
  moveDeadlineAt?: number;
  clockDeadlineAt?: number;
  clockRemaining?: Record<SeatKey, number>;
};
/** 斗兽棋棋子编码："A:rat" / "B:elephant" */
export type JungleCell = string | null;
export type JungleAnimal = "rat" | "cat" | "dog" | "wolf" | "leopard" | "tiger" | "lion" | "elephant";
export type JungleState = {
  board: JungleCell[][];
  turn: SeatKey;
  moveCount: number;
  lastFrom?: { row: number; col: number } | null;
  lastTo?: { row: number; col: number } | null;
  rankedDelta?: Record<SeatKey, number>;
  resignRequest?: {
    fromSeat: SeatKey;
    toSeat: SeatKey;
    createdAt: number;
  };
  ended?: boolean;
  winner?: RoundResult;
  moveDeadlineAt?: number;
  clockDeadlineAt?: number;
  clockRemaining?: Record<SeatKey, number>;
};
export type RankMultiplier = 1 | 2 | 5 | 10;

export type RoomNamePool = {
  adjectives: string[];
  subjects: string[];
  roomWords: string[];
};

export type RoomInfoTagStyle = GenderColors & {
  label: string;
};

export type PunishmentTaskConfig = {
  id: string;
  name: string;
  variants: Record<string, string>;
  backgroundImages?: string[];
  backgroundOpacity?: number;
};

export type PublicStats = {
  wins: number;
  losses: number;
  draws: number;
  punishments: number;
  // 展示值（服务端已按 rankedScore 封顶）；排序请用 sort* 真实分。
  rankedPoints: number;
  highestScore: number;
  lowestScore: number;
  sortRankedPoints: number;
  sortHighestScore: number;
  sortLowestScore: number;
  title: string;
  titleSegmentId?: string;
  titleCustom?: boolean;
  /** 玩家自设称号；展示优先级：管理员自定义 > 主人设置的宠物称号 > selfTitle > 系统自动称号。 */
  selfTitle?: string;
  /** 当前 title 的赋予来源，供称号标签按来源着色（见 AppConfig.titleTagStyles）。 */
  titleSource?: "system" | "self" | "master" | "admin";
  /** 服务端已按 titleSource 从 AppConfig.titleTagStyles 解析好的配色，直接当行内样式用。 */
  titleColors?: GenderColors;
  /** 累计在线时长（毫秒）；在线时服务端会加上本会话已持续时长。 */
  totalOnlineMs?: number;
};

export type GameWLD = {
  wins: number;
  losses: number;
  draws: number;
};

export type GameStats = {
  rps: GameWLD;
  othello: GameWLD;
  tictactoe: GameWLD;
  gomoku: GameWLD;
  liarsdice: GameWLD;
  jungle: GameWLD;
};

export type PublicPlayer = {
  id: string;
  name: string;
  genderId: string;
  genderLabel: string;
  factionId: string;
  factionLabel: string;
  factionColors: GenderColors;
  displayName: string;
  avatarUrl?: string;
  connected: boolean;
  disconnectedAt?: number;
  disconnectExpiresAt?: number;
  profileUpdatedAt?: number;
  nameWarEnabled?: boolean;
  nameWarToggledAt?: number;
  nameWarOriginalName?: string;
  nameWarPenaltyName?: string;
  nameWarPunished?: boolean;
  nameWarAllowRename?: boolean;
  nameWarRenameProtectedUntil?: number;
  nameWarRenamedBy?: string;
  nameWarRenamedByName?: string;
  nameWarRenameWindowStartedAt?: number;
  nameWarRenameCount?: number;
  giveawayEnabled?: boolean;
  giveawayValue?: number;
  giveawayClicks?: number;
  giveawayBoardText?: string;
  giveawayBoardSubmittedAt?: number;
  giveawayBoardExpiresAt?: number;
  giveawayBoardLikes?: number;
  giveawayBoardDislikes?: number;
  giveawayBoardLikeWindowStartedAt?: number;
  giveawayBoardLikesThisHour?: number;
  giveawayVoteWindowStartedAt?: number;
  giveawayVoteCount?: number;
  giveawayVoteLikesThisHour?: number;
  giveawayVoteDislikesThisHour?: number;
  rankMultiplierUnlocked?: boolean;
  extremeModeEnabled?: boolean;
  extremeModeToggledAt?: number;
  extremeModeCooldownUntil?: number;
  extremeWinStreak?: number;
  extremeLastDecayHour?: number;
  extremeForceClosed?: boolean;
  extremeForceClosedAt?: number;
  extremeRenameProtectedUntil?: number;
  extremeRenamedBy?: string;
  extremeRenamedByName?: string;
  /** 开启认主：可申请认他人为主人 */
  bondMasterEnabled?: boolean;
  /** 开启认宠：可被申请 / 主动收宠 */
  bondPetEnabled?: boolean;
  /** 公开展示：出现在大厅关系图谱 */
  bondPublicDisplay?: boolean;
  roomId?: string;
  isAdmin?: boolean;
  stats: PublicStats;
  gameStats: GameStats;
};

export type PetBondConfig = {
  panelTitle: string;
  maxPetsPerMaster: number;
  maxMastersPerPet: number;
  maxTitleLength: number;
};

export type PetBondEdge = {
  masterId: string;
  petId: string;
  petTitle?: string;
};

export type PetBondMember = {
  playerId: string;
  name: string;
  displayName: string;
  avatarUrl?: string;
  connected: boolean;
  petTitle?: string;
  releasePending?: boolean;
  releaseIncoming?: boolean;
  releaseRequestId?: string;
  /** 现有主人视角：该宠物正在申请认新主 */
  newMasterPendingId?: string;
  newMasterPendingName?: string;
};

export type PetBondCandidate = {
  playerId: string;
  name: string;
  displayName: string;
  avatarUrl?: string;
  connected: boolean;
  status: "none" | "pending" | "already" | string;
  requestId?: string;
  incoming?: boolean;
  incomingId?: string;
  incomingLabel?: string;
};

export type PetBondRequest = {
  id: string;
  kind: string;
  fromId: string;
  toId: string;
  masterId?: string;
  petId?: string;
  fromName: string;
  toName: string;
  createdAt: number;
  requiredIds: string[];
  approvedIds: string[];
  canApprove: boolean;
  label: string;
  pinTop?: boolean;
};

export type PetBondChain = {
  playerIds: string[];
  playerNames: string[];
};

export type PetBondState = {
  masters: PetBondMember[];
  pets: PetBondMember[];
  masterCandidates: PetBondCandidate[];
  petCandidates: PetBondCandidate[];
  incoming: PetBondRequest[];
  outgoing: PetBondRequest[];
  chains: PetBondChain[];
  config: PetBondConfig;
};

export type SeatOccupant = PublicPlayer | null;

export type ChatMessage = {
  id: string;
  roomId?: string;
  playerId: string;
  author: string;
  authorRole?: string;
  text: string;
  at: number;
  system?: boolean;
  transient?: boolean;
  expiresAt?: number;
  // 被 @ 的玩家 public id 列表；me.id 命中时气泡高亮。
  mentions?: string[];
  // SQLite 自增序号，做瀑布流分页游标；系统消息无此字段。
  seq?: number;
};

export type Suggestion = {
  id: string;
  playerId: string;
  author: string;
  authorPlayer?: PublicPlayer;
  text: string;
  at: number;
};

export type RoomSettings = {
  name: string;
  password?: string;
  gameId: GameId;
  enablePunishment: boolean;
  punishmentSource?: "system" | "player";
  punishmentId?: string;
  punishmentIds?: string[];
  roomBackgroundImage?: string;
  enableTags?: boolean;
  tags?: string[];
  allowProofImage?: boolean;
  tieDoublePunish: boolean;
  requireOpponentConfirm: boolean;
  enableRanked: boolean;
  stake: RankStake;
  enableRankMultiplier?: boolean;
  rankMultiplier?: RankMultiplier;
  enableExtremeRanked?: boolean;
  othelloBoardTheme?: "classic" | "pastel" | "midnight" | "wood" | "neon";
  tictactoeBoardTheme?: "paper" | "mint" | "midnight" | "candy" | "arcade";
  gomokuBoardTheme?: "classic" | "pastel" | "midnight" | "wood" | "neon";
  jungleBoardTheme?: "forest" | "bamboo" | "river" | "dusk" | "night";
  gomokuUndoLimit?: 0 | 1 | 3 | 10;
  liarsDiceMinPlayers?: number;
  liarsDiceMaxPlayers?: number;
  // 计时设置：0/缺省表示不限时
  othelloMoveSeconds?: number;
  othelloGameMinutes?: number;
  gomokuMoveSeconds?: number;
  gomokuGameMinutes?: number;
  jungleMoveSeconds?: number;
  jungleGameMinutes?: number;
};

export type LiarsDiceBid = {
  playerId: string;
  count: number;
  face: number;
  at: number;
};

export type LiarsDiceState = {
  participantIds: string[];
  readyPlayerIds: string[];
  diceCounts: Record<string, number>;
  currentTurn?: string;
  currentBid?: LiarsDiceBid | null;
  bidHistory: LiarsDiceBid[];
  onesWildDisabled?: boolean;
  roundNumber: number;
  ended?: boolean;
  winnerId?: string;
  loserId?: string;
  revealedHands?: Record<string, number[]>;
  actualCount?: number;
  minPlayers?: number;
  maxPlayers?: number;
};

export type PunishmentProof = {
  playerId: string;
  text: string;
  imageUrl?: string;
  taskText?: string;
  status?: "pending" | "approved" | "rejected";
  confirmedBy?: string;
  reviewedBy?: string;
  reviewedAt?: number;
  rejectReason?: string;
  redoTaskText?: string;
  submittedAt: number;
};

export type SeatStats = {
  wins: number;
  losses: number;
  draws: number;
  punishments: number;
};

export type RoundHistoryItem = {
  id: string;
  round: number;
  at: number;
  playerA: string;
  playerB: string;
  moveA: Move;
  moveB: Move;
  result: RoundResult;
  resultLabel: string;
  resultText: string;
  gameId?: GameId;
  othelloScore?: { black: number; white: number };
  othelloBlackSeat?: SeatKey;
  tictactoeXSeat?: SeatKey;
  tictactoeLine?: Array<{ row: number; col: number }>;
  gomokuBlackSeat?: SeatKey;
  gomokuLine?: Array<{ row: number; col: number }>;
  liarsDiceWinnerId?: string;
  liarsDiceLoserId?: string;
  liarsDiceBidCount?: number;
  liarsDiceBidFace?: number;
  liarsDiceActualCount?: number;
  liarsDiceHands?: Record<string, number[]>;
  liarsDiceHandOrder?: string[];
  liarsDiceNames?: Record<string, string>;
  ranked: boolean;
  stake?: RankStake;
  rankMultiplier?: RankMultiplier;
  effectiveStake?: number;
  extremeRanked?: boolean;
  punishmentName?: string;
  punishmentDescription?: string;
  punishmentTasks: Array<{
    playerId: string;
    playerName: string;
    factionId: string;
    factionLabel: string;
    taskText: string;
    backgroundImage?: string;
    backgroundOpacity?: number;
    assignedBy?: string;
    assignedByName?: string;
  }>;
  punishedNames: string[];
  proofs: Array<{
    playerId: string;
    playerName: string;
    text: string;
    imageUrl?: string;
    taskText?: string;
    status?: "pending" | "approved" | "rejected";
    reviewedBy?: string;
    reviewedAt?: number;
    rejectReason?: string;
    redoTaskText?: string;
    submittedAt: number;
  }>;
};

export type RoomSnapshot = {
  id: string;
  updatedAt: number;
  settings: RoomSettings;
  status: "waiting" | "playing" | "punishment";
  phase: GamePhase;
  seats: Record<SeatKey, SeatOccupant>;
  spectators: PublicPlayer[];
  ready: Record<SeatKey, boolean>;
  choices: Partial<Record<SeatKey, Move | "hidden">>;
  revealedChoices?: Partial<Record<SeatKey, Move>>;
  othello?: OthelloState;
  tictactoe?: TicTacToeState;
  liarsDice?: LiarsDiceState;
  gomoku?: GomokuState;
  jungle?: JungleState;
  resultText?: string;
  punishedPlayerIds: string[];
  proofs: PunishmentProof[];
  score: Record<SeatKey, number>;
  seatedScore: Record<SeatKey, number>;
  seatStats: Record<SeatKey, SeatStats>;
  roundHistory: RoundHistoryItem[];
  roundHistoryTotal: number;
  chat: ChatMessage[];
  forgiveAdvantageTargetId?: string;
  forgiveAdvantageBeneficiaryId?: string;
};

export type ServerStats = {
  startedAt: number;
  roomBroadcasts: number;
  lobbyBroadcasts: number;
  disconnects: number;
  reconnects: number;
  lastRoomSnapshotBytes: number;
  lastLobbySnapshotBytes: number;
  recentRoomBroadcasts: number;
  recentLobbyBroadcasts: number;
  averageRoomSnapshotBytes: number;
  averageLobbySnapshotBytes: number;
};

export type LobbySnapshot = {
  config?: AppConfig;
  onlineCount: number;
  players: PublicPlayer[];
  rooms: Array<{
    id: string;
    gameId: GameId;
    name: string;
    hasPassword: boolean;
    players: number;
    spectators: number;
    versus: {
      A: { player: PublicPlayer } | null;
      B: { player: PublicPlayer } | null;
    };
    status: RoomSnapshot["status"];
    roomBackgroundImage?: string;
    enablePunishment: boolean;
    punishmentIds?: string[];
    punishmentId?: string;
    tieDoublePunish: boolean;
    requireOpponentConfirm: boolean;
    enableRanked: boolean;
    stake: RankStake;
    enableRankMultiplier?: boolean;
    rankMultiplier?: RankMultiplier;
    enableExtremeRanked?: boolean;
    tags?: string[];
    liarsDiceMinPlayers?: number;
    liarsDiceMaxPlayers?: number;
    othelloMoveSeconds?: number;
    othelloGameMinutes?: number;
    gomokuMoveSeconds?: number;
    gomokuGameMinutes?: number;
    gomokuUndoLimit?: number;
    jungleMoveSeconds?: number;
    jungleGameMinutes?: number;
  }>;
  normalLeaderboard: PublicPlayer[];
  rankedLeaderboard: PublicPlayer[];
  suggestions: Suggestion[];
  lobbyChat: ChatMessage[];
  serverStats: ServerStats;
  petBonds?: PetBondEdge[];
};

export type AppConfig = {
  site: {
    name: string;
    description: string;
    adminPassword: string;
  };
  announcementBoard: {
    enabled?: boolean;
    title: string;
    content: string;
  };
  securityDisclaimer: {
    enabled?: boolean;
  };
  genders: GenderOption[];
  genderFactions: GenderFaction[];
  titles: Array<{
    id: string;
    minPercent: number;
    maxPercent: number;
    names: string[];
    factionNames?: Record<string, string[]>;
  }>;
  punishments: Array<{
    id: string;
    name: string;
    description: string;
    variants?: Record<string, string>;
    tasks?: PunishmentTaskConfig[];
    cardImageUrl?: string;
    cardImageOpacity?: number;
    roomBackgroundImages?: string[];
    roomNamePool?: RoomNamePool;
  }>;
  playerPunishmentRoomNamePool?: RoomNamePool;
  roomTags: string[];
  roomInfoTags: Record<string, RoomInfoTagStyle>;
  /** 称号标签按赋予来源（system/self/master/admin）的配色，形状与 roomInfoTags 一致。 */
  titleTagStyles: Record<string, RoomInfoTagStyle>;
  accessControl: {
    maxOnlinePerIp: number;
    maxCreatesPer10Min: number;
    ipBackstopMultiplier: number;
    ipBackstopMinLimit: number;
    maxSessionIssuePerIp: number;
    maxOnlinePerIpTotal: number;
    maxCreatesPerIp: number;
    maxActiveRoomsPerOwner: number;
    maxProofUploadsPerPlayer: number;
    registrationDisabled: boolean;
  };
  nameWar: {
    penaltyPrefix: string;
    loserPanelTitle: string;
    escapeTitle: string;
    renamePanelTitle?: string;
    nameWarLoserLabel?: string;
    extremeForceClosedLabel?: string;
    /** 真实排位分 ≤ 此值时名争失格（默认 -4999） */
    penaltyThreshold: number;
  };
  giveaway: {
    panelTitle: string;
    panelDescription: string;
    submitPlaceholder: string;
    emptyText: string;
    /** 主动白给（手动选白给 / 点击白给按钮 / RPS 概率强制白给）每次增加的白给值百分比 */
    activeBoostValue: number;
    /** 白给已开启的玩家每赢一局（含断线判负），白给值减少的百分比 */
    winPenaltyValue: number;
    /** 自救板点赞每小时可投次数上限、每次降低的百分比 */
    likeVoteLimitPerHour: number;
    likeVoteValue: number;
    /** 自救板倒赞每小时可投次数上限、每次增加的百分比 */
    dislikeVoteLimitPerHour: number;
    dislikeVoteValue: number;
  };
  petBond: PetBondConfig;
  extremeMode: {
    label: string;
    emoji: string;
    cooldownHours: number;
    positiveLossRates: Record<string, number>;
    negativeWinRates: Record<string, number>;
    hourlyDecay: Record<string, number>;
    winStreakThreshold: number;
    winStreakCrashChance: number;
    crashTargetPoints: number;
    forceCloseWarning?: string;
    forceRenameMinPoints?: number;
    forceRenameProtectHours?: number;
  };
  // rankedScore：排位分「展示」上下限与每日衰减比例。存储分数本身不设限，
  // 仅在下发给客户端时按这里的 max/min/nameWarMin 封顶展示。
  rankedScore: {
    max: number;
    min: number;
    nameWarMin: number;
    dailyDecayRatio: number;
  };
  games: Array<{
    id: GameId;
    name: string;
    description: string;
  }>;
  messages: Record<string, string>;
};

export type ClientError = { message: string };
