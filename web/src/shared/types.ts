export type GenderColors = {
  textColor: string;
  backgroundColor: string;
  borderColor: string;
};
export type GenderOption = { id: string; label: string; factionId: string };
export type GenderFaction = GenderColors & {
  id: string;
  label: string;
  genders: GenderOption[];
};
export type Move = "rock" | "scissors" | "paper" | "giveaway" | "forfeit" | "noMove";
export type RoundResult = "A" | "B" | "draw" | "doubleLoss";
export type GamePhase = "waiting" | "ready" | "choosing" | "result" | "punishment";
export type SeatKey = "A" | "B";
export type GameId = "rps" | "othello" | "tictactoe" | "liarsdice" | "gomoku";
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
  ended?: boolean;
  winner?: RoundResult;
  moveDeadlineAt?: number;
  clockDeadlineAt?: number;
  clockRemaining?: Record<SeatKey, number>;
};
export type RankMultiplier = 1 | 2 | 5 | 10;
export type BotDifficulty = "easy" | "normal" | "chaos";
export type BotStrategy = "random" | "counter" | "chaos" | "throw" | "win";

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
  roomId?: string;
  isAdmin?: boolean;
  stats: PublicStats;
  gameStats: GameStats;
};

export type BotPlayer = {
  id: string;
  name: string;
  difficulty: BotDifficulty;
  isBot: true;
};

export type SeatOccupant = PublicPlayer | BotPlayer | null;

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
  enableBot: boolean;
  botDifficulty: BotDifficulty;
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
  gomokuUndoLimit?: 0 | 1 | 3 | 10;
  liarsDiceMinPlayers?: number;
  liarsDiceMaxPlayers?: number;
  // 计时设置：0/缺省表示不限时
  othelloMoveSeconds?: number;
  othelloGameMinutes?: number;
  gomokuMoveSeconds?: number;
  gomokuGameMinutes?: number;
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
      A: { name: string; isBot: true } | { player: PublicPlayer } | null;
      B: { name: string; isBot: true } | { player: PublicPlayer } | null;
    };
    status: RoomSnapshot["status"];
    roomBackgroundImage?: string;
    enableBot: boolean;
    botDifficulty: BotDifficulty;
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
  }>;
  normalLeaderboard: PublicPlayer[];
  rankedLeaderboard: PublicPlayer[];
  suggestions: Suggestion[];
  lobbyChat: ChatMessage[];
  serverStats: ServerStats;
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
  };
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
  bots: {
    names: string[];
    difficulties: Array<{
      id: BotDifficulty;
      name: string;
      description: string;
      emoji?: string;
      level?: number;
      strategy?: BotStrategy;
      cardColor?: string;
    }>;
  };
  games: Array<{
    id: GameId;
    name: string;
    description: string;
  }>;
  messages: Record<string, string>;
};

export type ClientError = { message: string };
