package types

// 共享类型：与前端/原 Node 项目 JSON 契约保持一致。

type GenderColors struct {
	TextColor       string `json:"textColor"`
	BackgroundColor string `json:"backgroundColor"`
	BorderColor     string `json:"borderColor"`
}

type GenderOption struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	FactionID string `json:"factionId"`
}

type GenderFaction struct {
	GenderColors
	ID      string         `json:"id"`
	Label   string         `json:"label"`
	Genders []GenderOption `json:"genders"`
}

type Move string

const (
	MoveRock     Move = "rock"
	MoveScissors Move = "scissors"
	MovePaper    Move = "paper"
	MoveGiveaway Move = "giveaway"
	MoveForfeit  Move = "forfeit"
	MoveNoMove   Move = "noMove"
)

type RoundResult string

const (
	ResultA          RoundResult = "A"
	ResultB          RoundResult = "B"
	ResultDraw       RoundResult = "draw"
	ResultDoubleLoss RoundResult = "doubleLoss"
)

type GamePhase string

const (
	PhaseWaiting    GamePhase = "waiting"
	PhaseReady      GamePhase = "ready"
	PhaseChoosing   GamePhase = "choosing"
	PhaseResult     GamePhase = "result"
	PhasePunishment GamePhase = "punishment"
)

type SeatKey string

const (
	SeatA SeatKey = "A"
	SeatB SeatKey = "B"
)

type GameID string

const (
	GameRPS       GameID = "rps"
	GameOthello   GameID = "othello"
	GameTicTacToe GameID = "tictactoe"
	GameLiarsDice GameID = "liarsdice"
)

type RankStake int
type RankMultiplier int
type BotDifficulty string
type BotStrategy string

type RoomNamePool struct {
	Adjectives []string `json:"adjectives"`
	Subjects   []string `json:"subjects"`
	RoomWords  []string `json:"roomWords"`
}

type RoomInfoTagStyle struct {
	GenderColors
	Label string `json:"label"`
}

type PunishmentTaskConfig struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Variants          map[string]string `json:"variants"`
	BackgroundImages  []string          `json:"backgroundImages,omitempty"`
	BackgroundOpacity float64           `json:"backgroundOpacity,omitempty"`
}

type PublicStats struct {
	Wins           int    `json:"wins"`
	Losses         int    `json:"losses"`
	Draws          int    `json:"draws"`
	Punishments    int    `json:"punishments"`
	RankedPoints   int    `json:"rankedPoints"`
	Title          string `json:"title"`
	TitleSegmentID string `json:"titleSegmentId,omitempty"`
}

type OthelloStats struct {
	Wins     int `json:"wins"`
	Losses   int `json:"losses"`
	Draws    int `json:"draws"`
	Games    int `json:"games"`
	Captured int `json:"captured"`
	Lost     int `json:"lost"`
}

type PublicPlayer struct {
	ID                               string       `json:"id"`
	Name                             string       `json:"name"`
	GenderID                         string       `json:"genderId"`
	GenderLabel                      string       `json:"genderLabel"`
	FactionID                        string       `json:"factionId"`
	FactionLabel                     string       `json:"factionLabel"`
	FactionColors                    GenderColors `json:"factionColors"`
	DisplayName                      string       `json:"displayName"`
	Connected                        bool         `json:"connected"`
	DisconnectedAt                   *int64       `json:"disconnectedAt,omitempty"`
	DisconnectExpiresAt              *int64       `json:"disconnectExpiresAt,omitempty"`
	ProfileUpdatedAt                 *int64       `json:"profileUpdatedAt,omitempty"`
	NameWarEnabled                   *bool        `json:"nameWarEnabled,omitempty"`
	NameWarToggledAt                 *int64       `json:"nameWarToggledAt,omitempty"`
	NameWarOriginalName              string       `json:"nameWarOriginalName,omitempty"`
	NameWarPenaltyName               string       `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished                  *bool        `json:"nameWarPunished,omitempty"`
	NameWarAllowRename               *bool        `json:"nameWarAllowRename,omitempty"`
	NameWarRenameProtectedUntil      *int64       `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedBy                 string       `json:"nameWarRenamedBy,omitempty"`
	NameWarRenamedByName             string       `json:"nameWarRenamedByName,omitempty"`
	NameWarRenameWindowStartedAt     *int64       `json:"nameWarRenameWindowStartedAt,omitempty"`
	NameWarRenameCount               *int         `json:"nameWarRenameCount,omitempty"`
	GiveawayEnabled                  *bool        `json:"giveawayEnabled,omitempty"`
	GiveawayValue                    *float64     `json:"giveawayValue,omitempty"`
	GiveawayClicks                   *int         `json:"giveawayClicks,omitempty"`
	GiveawayBoardText                string       `json:"giveawayBoardText,omitempty"`
	GiveawayBoardSubmittedAt         *int64       `json:"giveawayBoardSubmittedAt,omitempty"`
	GiveawayBoardExpiresAt           *int64       `json:"giveawayBoardExpiresAt,omitempty"`
	GiveawayBoardLikes               *int         `json:"giveawayBoardLikes,omitempty"`
	GiveawayBoardDislikes            *int         `json:"giveawayBoardDislikes,omitempty"`
	GiveawayBoardLikesThisHour       *int         `json:"giveawayBoardLikesThisHour,omitempty"`
	GiveawayBoardLikeWindowStartedAt *int64       `json:"giveawayBoardLikeWindowStartedAt,omitempty"`
	GiveawayVoteWindowStartedAt      *int64       `json:"giveawayVoteWindowStartedAt,omitempty"`
	GiveawayVoteCount                *int         `json:"giveawayVoteCount,omitempty"`
	GiveawayVoteLikesThisHour        *int         `json:"giveawayVoteLikesThisHour,omitempty"`
	GiveawayVoteDislikesThisHour     *int         `json:"giveawayVoteDislikesThisHour,omitempty"`
	RankMultiplierUnlocked           *bool        `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled               *bool        `json:"extremeModeEnabled,omitempty"`
	ExtremeModeToggledAt             *int64       `json:"extremeModeToggledAt,omitempty"`
	ExtremeModeCooldownUntil         *int64       `json:"extremeModeCooldownUntil,omitempty"`
	ExtremeWinStreak                 *int         `json:"extremeWinStreak,omitempty"`
	ExtremeLastDecayHour             *int64       `json:"extremeLastDecayHour,omitempty"`
	ExtremeForceClosed               *bool        `json:"extremeForceClosed,omitempty"`
	ExtremeForceClosedAt             *int64       `json:"extremeForceClosedAt,omitempty"`
	ExtremeRenameProtectedUntil      *int64       `json:"extremeRenameProtectedUntil,omitempty"`
	ExtremeRenamedBy                 string       `json:"extremeRenamedBy,omitempty"`
	ExtremeRenamedByName             string       `json:"extremeRenamedByName,omitempty"`
	RoomID                           string       `json:"roomId,omitempty"`
	IsAdmin                          *bool        `json:"isAdmin,omitempty"`
	Stats                            PublicStats  `json:"stats"`
	OthelloStats                     OthelloStats `json:"othelloStats"`
}

type BotPlayer struct {
	ID         string        `json:"id"`
	Name       string        `json:"name"`
	Difficulty BotDifficulty `json:"difficulty"`
	IsBot      bool          `json:"isBot"`
}

// SeatOccupant is either *PublicPlayer, *BotPlayer or null (JSON).
// Encoded as json.RawMessage / any in snapshots.

type ChatMessage struct {
	ID           string        `json:"id"`
	RoomID       string        `json:"roomId,omitempty"`
	PlayerID     string        `json:"playerId"`
	Author       string        `json:"author"`
	AuthorPlayer *PublicPlayer `json:"authorPlayer,omitempty"`
	AuthorRole   string        `json:"authorRole,omitempty"`
	Text         string        `json:"text"`
	At           int64         `json:"at"`
	System       bool          `json:"system,omitempty"`
	Transient    bool          `json:"transient,omitempty"`
	ExpiresAt    *int64        `json:"expiresAt,omitempty"`
	// Mentions：被 @ 的玩家 public id 列表；前端据此在 me.id 命中时高亮气泡。
	Mentions []string `json:"mentions,omitempty"`
	// Seq：SQLite 自增序号，仅出站聊天用作分页游标；系统消息为 0。
	Seq int64 `json:"seq,omitempty"`
}

type Suggestion struct {
	ID           string        `json:"id"`
	PlayerID     string        `json:"playerId"`
	Author       string        `json:"author"`
	AuthorPlayer *PublicPlayer `json:"authorPlayer,omitempty"`
	Text         string        `json:"text"`
	At           int64         `json:"at"`
}

type RoomSettings struct {
	Name             string        `json:"name"`
	Password         string        `json:"password,omitempty"`
	GameID           GameID        `json:"gameId"`
	EnableBot        bool          `json:"enableBot"`
	BotDifficulty    BotDifficulty `json:"botDifficulty"`
	EnablePunishment bool          `json:"enablePunishment"`
	PunishmentSource string        `json:"punishmentSource,omitempty"`
	PunishmentID     string        `json:"punishmentId,omitempty"`
	// 数组字段不用 omitempty：空切片会变成 null 或字段缺失，前端 .map/.includes 会挂
	PunishmentIDs          []string       `json:"punishmentIds"`
	RoomBackgroundImage    string         `json:"roomBackgroundImage,omitempty"`
	EnableTags             bool           `json:"enableTags,omitempty"`
	Tags                   []string       `json:"tags"`
	AllowProofImage        *bool          `json:"allowProofImage,omitempty"`
	TieDoublePunish        bool           `json:"tieDoublePunish"`
	RequireOpponentConfirm bool           `json:"requireOpponentConfirm"`
	EnableRanked           bool           `json:"enableRanked"`
	Stake                  RankStake      `json:"stake"`
	EnableRankMultiplier   bool           `json:"enableRankMultiplier,omitempty"`
	RankMultiplier         RankMultiplier `json:"rankMultiplier,omitempty"`
	EnableExtremeRanked    bool           `json:"enableExtremeRanked,omitempty"`
	OthelloBoardTheme      string         `json:"othelloBoardTheme,omitempty"`
	TicTacToeBoardTheme    string         `json:"tictactoeBoardTheme,omitempty"`
	LiarsDiceMinPlayers    int            `json:"liarsDiceMinPlayers,omitempty"`
	LiarsDiceMaxPlayers    int            `json:"liarsDiceMaxPlayers,omitempty"`
}

type PunishmentProof struct {
	PlayerID     string `json:"playerId"`
	Text         string `json:"text"`
	ImageURL     string `json:"imageUrl,omitempty"`
	TaskText     string `json:"taskText,omitempty"`
	Status       string `json:"status,omitempty"`
	ConfirmedBy  string `json:"confirmedBy,omitempty"`
	ReviewedBy   string `json:"reviewedBy,omitempty"`
	ReviewedAt   *int64 `json:"reviewedAt,omitempty"`
	RejectReason string `json:"rejectReason,omitempty"`
	RedoTaskText string `json:"redoTaskText,omitempty"`
	SubmittedAt  int64  `json:"submittedAt"`
}

type SeatStats struct {
	Wins        int `json:"wins"`
	Losses      int `json:"losses"`
	Draws       int `json:"draws"`
	Punishments int `json:"punishments"`
}

type Pos struct {
	Row int `json:"row"`
	Col int `json:"col"`
}

type RoundHistoryItem struct {
	ID                    string           `json:"id"`
	Round                 int              `json:"round"`
	At                    int64            `json:"at"`
	PlayerA               string           `json:"playerA"`
	PlayerB               string           `json:"playerB"`
	MoveA                 Move             `json:"moveA"`
	MoveB                 Move             `json:"moveB"`
	Result                RoundResult      `json:"result"`
	ResultLabel           string           `json:"resultLabel"`
	ResultText            string           `json:"resultText"`
	GameID                GameID           `json:"gameId,omitempty"`
	OthelloScore          *OthelloScore    `json:"othelloScore,omitempty"`
	OthelloBlackSeat      SeatKey          `json:"othelloBlackSeat,omitempty"`
	TicTacToeXSeat        SeatKey          `json:"tictactoeXSeat,omitempty"`
	TicTacToeLine         []Pos            `json:"tictactoeLine,omitempty"`
	Ranked                bool             `json:"ranked"`
	Stake                 *RankStake       `json:"stake,omitempty"`
	RankMultiplier        *RankMultiplier  `json:"rankMultiplier,omitempty"`
	EffectiveStake        *int             `json:"effectiveStake,omitempty"`
	ExtremeRanked         bool             `json:"extremeRanked,omitempty"`
	PunishmentName        string           `json:"punishmentName,omitempty"`
	PunishmentDescription string           `json:"punishmentDescription,omitempty"`
	PunishmentTasks       []PunishmentTask `json:"punishmentTasks"`
	PunishedNames         []string         `json:"punishedNames"`
	Proofs                []HistoryProof   `json:"proofs"`
	LiarsDiceWinnerID     string           `json:"liarsDiceWinnerId,omitempty"`
	LiarsDiceLoserID      string           `json:"liarsDiceLoserId,omitempty"`
	LiarsDiceBidCount     int              `json:"liarsDiceBidCount,omitempty"`
	LiarsDiceBidFace      int              `json:"liarsDiceBidFace,omitempty"`
	LiarsDiceActualCount  int              `json:"liarsDiceActualCount,omitempty"`
	// LiarsDiceHands/LiarsDiceNames：结算后全员开牌，playerId -> 骰子/展示名。
	// pbconv.historyToProto 里手动映射成 pair 列表，不走通用 JSONCamelToProto。
	LiarsDiceHands     map[string][]int  `json:"liarsDiceHands,omitempty"`
	LiarsDiceHandOrder []string          `json:"liarsDiceHandOrder,omitempty"`
	LiarsDiceNames     map[string]string `json:"liarsDiceNames,omitempty"`
}

type OthelloScore struct {
	Black int `json:"black"`
	White int `json:"white"`
}

type PunishmentTask struct {
	PlayerID          string   `json:"playerId"`
	PlayerName        string   `json:"playerName"`
	FactionID         string   `json:"factionId"`
	FactionLabel      string   `json:"factionLabel"`
	TaskText          string   `json:"taskText"`
	BackgroundImage   string   `json:"backgroundImage,omitempty"`
	BackgroundOpacity *float64 `json:"backgroundOpacity,omitempty"`
	AssignedBy        string   `json:"assignedBy,omitempty"`
	AssignedByName    string   `json:"assignedByName,omitempty"`
}

type HistoryProof struct {
	PlayerID     string `json:"playerId"`
	PlayerName   string `json:"playerName"`
	Text         string `json:"text"`
	ImageURL     string `json:"imageUrl,omitempty"`
	TaskText     string `json:"taskText,omitempty"`
	Status       string `json:"status,omitempty"`
	ReviewedBy   string `json:"reviewedBy,omitempty"`
	ReviewedAt   *int64 `json:"reviewedAt,omitempty"`
	RejectReason string `json:"rejectReason,omitempty"`
	RedoTaskText string `json:"redoTaskText,omitempty"`
	SubmittedAt  int64  `json:"submittedAt"`
}

type OthelloCell string

const (
	CellBlack OthelloCell = "black"
	CellWhite OthelloCell = "white"
	// empty cells are null in JSON
)

type OthelloPendingSettlement struct {
	ID           string  `json:"id"`
	Seat         SeatKey `json:"seat"`
	OpponentSeat SeatKey `json:"opponentSeat"`
	Flips        int     `json:"flips"`
	Stake        int     `json:"stake"`
	NextTurn     SeatKey `json:"nextTurn"`
	ExpiresAt    int64   `json:"expiresAt"`
	Forced       string  `json:"forced,omitempty"` // giveaway | tribute
	ResolvedAs   string  `json:"resolvedAs,omitempty"`
}

type OthelloSurrenderRequest struct {
	FromSeat  SeatKey `json:"fromSeat"`
	ToSeat    SeatKey `json:"toSeat"`
	CreatedAt int64   `json:"createdAt"`
}

type OthelloState struct {
	Board             [][]*OthelloCell          `json:"board"`
	Turn              SeatKey                   `json:"turn"`
	BlackSeat         SeatKey                   `json:"blackSeat"`
	LegalMoves        []Pos                     `json:"legalMoves"`
	PassCount         int                       `json:"passCount"`
	BlackCount        int                       `json:"blackCount"`
	WhiteCount        int                       `json:"whiteCount"`
	RankedDelta       map[SeatKey]int           `json:"rankedDelta"`
	SettlementEvents  []string                  `json:"settlementEvents"`
	PendingSettlement *OthelloPendingSettlement `json:"pendingSettlement,omitempty"`
	SurrenderRequest  *OthelloSurrenderRequest  `json:"surrenderRequest,omitempty"`
	Ended             bool                      `json:"ended,omitempty"`
	Winner            RoundResult               `json:"winner,omitempty"`
}

type TicTacToeCell string

const (
	CellX TicTacToeCell = "X"
	CellO TicTacToeCell = "O"
)

type TicTacToeGiveawayPrompt struct {
	Seat      SeatKey `json:"seat"`
	Forced    bool    `json:"forced,omitempty"`
	StartedAt int64   `json:"startedAt"`
	ExpiresAt int64   `json:"expiresAt"`
}

type TicTacToeState struct {
	Board          [][]*TicTacToeCell       `json:"board"`
	Turn           SeatKey                  `json:"turn"`
	XSeat          SeatKey                  `json:"xSeat"`
	MoveCount      int                      `json:"moveCount"`
	GiveawayPrompt *TicTacToeGiveawayPrompt `json:"giveawayPrompt,omitempty"`
	WinningLine    []Pos                    `json:"winningLine"`
	RankedDelta    map[SeatKey]int          `json:"rankedDelta"`
	Ended          bool                     `json:"ended,omitempty"`
	Winner         RoundResult              `json:"winner,omitempty"`
}

type LiarsDiceBid struct {
	PlayerID string `json:"playerId"`
	Count    int    `json:"count"`
	Face     int    `json:"face"`
	At       int64  `json:"at"`
}

type LiarsDiceState struct {
	ParticipantIDs   []string         `json:"participantIds"`
	ReadyPlayerIDs   []string         `json:"readyPlayerIds"`
	DiceCounts       map[string]int   `json:"diceCounts"`
	CurrentTurn      string           `json:"currentTurn,omitempty"`
	CurrentBid       *LiarsDiceBid    `json:"currentBid,omitempty"`
	BidHistory       []LiarsDiceBid   `json:"bidHistory"`
	OnesWildDisabled bool             `json:"onesWildDisabled,omitempty"`
	RoundNumber      int              `json:"roundNumber"`
	Ended            bool             `json:"ended,omitempty"`
	WinnerID         string           `json:"winnerId,omitempty"`
	LoserID          string           `json:"loserId,omitempty"`
	RevealedHands    map[string][]int `json:"revealedHands,omitempty"`
	ActualCount      int              `json:"actualCount,omitempty"`
	MinPlayers       int              `json:"minPlayers,omitempty"`
	MaxPlayers       int              `json:"maxPlayers,omitempty"`
}

type RoomSnapshot struct {
	ID                string                `json:"id"`
	Code              string                `json:"code"`
	UpdatedAt         int64                 `json:"updatedAt"`
	Settings          RoomSettings          `json:"settings"`
	Status            string                `json:"status"`
	Phase             GamePhase             `json:"phase"`
	Seats             map[SeatKey]any       `json:"seats"`
	Spectators        []PublicPlayer        `json:"spectators"`
	Ready             map[SeatKey]bool      `json:"ready"`
	Choices           map[SeatKey]any       `json:"choices"`
	RevealedChoices   map[SeatKey]Move      `json:"revealedChoices,omitempty"`
	Othello           *OthelloState         `json:"othello,omitempty"`
	TicTacToe         *TicTacToeState       `json:"tictactoe,omitempty"`
	LiarsDice         *LiarsDiceState       `json:"liarsDice,omitempty"`
	ResultText        string                `json:"resultText,omitempty"`
	PunishedPlayerIDs []string              `json:"punishedPlayerIds"`
	Proofs            []PunishmentProof     `json:"proofs"`
	Score             map[SeatKey]int       `json:"score"`
	SeatedScore       map[SeatKey]int       `json:"seatedScore"`
	SeatStats         map[SeatKey]SeatStats `json:"seatStats"`
	RoundHistory      []RoundHistoryItem    `json:"roundHistory"`
	RoundHistoryTotal int                   `json:"roundHistoryTotal"`
	Chat              []ChatMessage         `json:"chat"`
}

type ServerStats struct {
	StartedAt                 int64 `json:"startedAt"`
	RoomBroadcasts            int   `json:"roomBroadcasts"`
	LobbyBroadcasts           int   `json:"lobbyBroadcasts"`
	Disconnects               int   `json:"disconnects"`
	Reconnects                int   `json:"reconnects"`
	LastRoomSnapshotBytes     int   `json:"lastRoomSnapshotBytes"`
	LastLobbySnapshotBytes    int   `json:"lastLobbySnapshotBytes"`
	RecentRoomBroadcasts      int   `json:"recentRoomBroadcasts"`
	RecentLobbyBroadcasts     int   `json:"recentLobbyBroadcasts"`
	AverageRoomSnapshotBytes  int   `json:"averageRoomSnapshotBytes"`
	AverageLobbySnapshotBytes int   `json:"averageLobbySnapshotBytes"`
}

type LobbyRoomInfo struct {
	ID                     string          `json:"id"`
	GameID                 GameID          `json:"gameId"`
	Code                   string          `json:"code"`
	Name                   string          `json:"name"`
	HasPassword            bool            `json:"hasPassword"`
	Players                int             `json:"players"`
	Spectators             int             `json:"spectators"`
	Versus                 map[SeatKey]any `json:"versus"`
	Status                 string          `json:"status"`
	RoomBackgroundImage    string          `json:"roomBackgroundImage,omitempty"`
	EnableBot              bool            `json:"enableBot"`
	BotDifficulty          BotDifficulty   `json:"botDifficulty"`
	EnablePunishment       bool            `json:"enablePunishment"`
	PunishmentIDs          []string        `json:"punishmentIds"`
	PunishmentID           string          `json:"punishmentId,omitempty"`
	TieDoublePunish        bool            `json:"tieDoublePunish"`
	RequireOpponentConfirm bool            `json:"requireOpponentConfirm"`
	EnableRanked           bool            `json:"enableRanked"`
	Stake                  RankStake       `json:"stake"`
	EnableRankMultiplier   bool            `json:"enableRankMultiplier,omitempty"`
	RankMultiplier         RankMultiplier  `json:"rankMultiplier,omitempty"`
	EnableExtremeRanked    bool            `json:"enableExtremeRanked,omitempty"`
	Tags                   []string        `json:"tags"`
}

type LobbySnapshot struct {
	Config      *AppConfig `json:"config,omitempty"`
	OnlineCount int        `json:"onlineCount"`
	// Players 按 id 索引，便于增量补丁；完整资料见 player:get / 房间快照 / me
	Players map[string]LobbyPlayer `json:"players"`
	// Rooms 按 id 索引
	Rooms             map[string]LobbyRoomInfo `json:"rooms"`
	NormalLeaderboard []LobbyPlayer            `json:"normalLeaderboard"`
	RankedLeaderboard []LobbyPlayer            `json:"rankedLeaderboard"`
	Suggestions       []Suggestion             `json:"suggestions"`
	LobbyChat         []ChatMessage            `json:"lobbyChat"`
	ServerStats       ServerStats              `json:"serverStats"`
}

type TitleSegment struct {
	ID           string              `json:"id"`
	Min          int                 `json:"min"`
	Max          int                 `json:"max"`
	Names        []string            `json:"names"`
	FactionNames map[string][]string `json:"factionNames,omitempty"`
}

type PunishmentConfig struct {
	ID                   string                 `json:"id"`
	Name                 string                 `json:"name"`
	Description          string                 `json:"description"`
	Variants             map[string]string      `json:"variants,omitempty"`
	Tasks                []PunishmentTaskConfig `json:"tasks,omitempty"`
	CardImageURL         string                 `json:"cardImageUrl,omitempty"`
	CardImageOpacity     float64                `json:"cardImageOpacity,omitempty"`
	RoomBackgroundImages []string               `json:"roomBackgroundImages,omitempty"`
	RoomNamePool         *RoomNamePool          `json:"roomNamePool,omitempty"`
}

type BotDifficultyConfig struct {
	ID          BotDifficulty `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Emoji       string        `json:"emoji,omitempty"`
	Level       int           `json:"level,omitempty"`
	Strategy    BotStrategy   `json:"strategy,omitempty"`
	CardColor   string        `json:"cardColor,omitempty"`
}

type GameConfig struct {
	ID          GameID `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type DailyAnnouncement struct {
	Enabled    bool   `json:"enabled"`
	Title      string `json:"title"`
	Content    string `json:"content"`
	ButtonText string `json:"buttonText"`
	Version    string `json:"version"`
}

type ExtremeModeConfig struct {
	Label                   string             `json:"label"`
	Emoji                   string             `json:"emoji"`
	CooldownHours           int                `json:"cooldownHours"`
	PositiveLossRates       map[string]float64 `json:"positiveLossRates"`
	NegativeWinRates        map[string]float64 `json:"negativeWinRates"`
	HourlyDecay             map[string]float64 `json:"hourlyDecay"`
	WinStreakThreshold      int                `json:"winStreakThreshold"`
	WinStreakCrashChance    float64            `json:"winStreakCrashChance"`
	CrashTargetPoints       int                `json:"crashTargetPoints"`
	ForceCloseWarning       string             `json:"forceCloseWarning,omitempty"`
	ForceRenameMinPoints    int                `json:"forceRenameMinPoints,omitempty"`
	ForceRenameProtectHours int                `json:"forceRenameProtectHours,omitempty"`
}

type AppConfig struct {
	Site struct {
		Name          string `json:"name"`
		Description   string `json:"description"`
		AdminPassword string `json:"adminPassword"`
	} `json:"site"`
	DailyAnnouncement            DailyAnnouncement           `json:"dailyAnnouncement"`
	Genders                      []GenderOption              `json:"genders"`
	GenderFactions               []GenderFaction             `json:"genderFactions"`
	Titles                       []TitleSegment              `json:"titles"`
	Punishments                  []PunishmentConfig          `json:"punishments"`
	PlayerPunishmentRoomNamePool *RoomNamePool               `json:"playerPunishmentRoomNamePool,omitempty"`
	RoomTags                     []string                    `json:"roomTags"`
	RoomInfoTags                 map[string]RoomInfoTagStyle `json:"roomInfoTags"`
	AccessControl                struct {
		MaxOnlinePerIP     int `json:"maxOnlinePerIp"`
		MaxCreatesPer10Min int `json:"maxCreatesPer10Min"`
	} `json:"accessControl"`
	NameWar struct {
		PenaltyPrefix           string `json:"penaltyPrefix"`
		LoserPanelTitle         string `json:"loserPanelTitle"`
		EscapeTitle             string `json:"escapeTitle"`
		RenamePanelTitle        string `json:"renamePanelTitle,omitempty"`
		NameWarLoserLabel       string `json:"nameWarLoserLabel,omitempty"`
		ExtremeForceClosedLabel string `json:"extremeForceClosedLabel,omitempty"`
	} `json:"nameWar"`
	Giveaway struct {
		PanelTitle        string `json:"panelTitle"`
		PanelDescription  string `json:"panelDescription"`
		SubmitPlaceholder string `json:"submitPlaceholder"`
		EmptyText         string `json:"emptyText"`
	} `json:"giveaway"`
	ExtremeMode ExtremeModeConfig `json:"extremeMode"`
	Bots        struct {
		Names        []string              `json:"names"`
		Difficulties []BotDifficultyConfig `json:"difficulties"`
	} `json:"bots"`
	Games    []GameConfig      `json:"games"`
	Messages map[string]string `json:"messages"`
}
