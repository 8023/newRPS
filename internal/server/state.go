package server

import (
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/types"
)

const (
	lobbyChannel           = "lobby"
	// lobbySuggestionChannel 现为「大厅聊天」实时频道：大厅视图与房间内的「大厅」tab
	// 都加入此频道以收 chat:new 增量推送（历史与首屏走 chat:load RPC，读 SQLite）。
	lobbySuggestionChannel = "lobby:suggestions"
	chatPageSize           = 100
	roomHistoryPageSize    = 20
	giveawayBoardDuration  = 12 * time.Hour
	broadcastMetricWindow  = 60 * time.Second
)

type RpsMove string

type DisconnectForfeit struct {
	LoserID        string
	LoserSeat      types.SeatKey
	LoserName      string
	WinnerID       string
	WinnerSeat     types.SeatKey
	WinnerName     string
	Stake          int
	BaseStake      types.RankStake
	RankMultiplier types.RankMultiplier
}

type LeaveReason string

const (
	LeaveManual            LeaveReason = "manual"
	LeaveSwitchRoom        LeaveReason = "switchRoom"
	LeaveSpectate          LeaveReason = "spectate"
	LeaveDisconnectTimeout LeaveReason = "disconnectTimeout"
	LeaveAdminKick         LeaveReason = "adminKick"
)

type LeaveResult struct {
	OK    bool
	Error string
}

type SessionPayload struct {
	SID string
	Exp int64
}

type RateLimitOptions struct {
	Limit      int
	WindowMs   int64
	CooldownMs  int64
}

type rateBucket struct {
	Hits          []int64
	CooldownUntil int64
}

type rateLimitBucket struct {
	ResetAt int64
	Count   int
}

type PlayerState struct {
	types.PublicPlayer
	SocketID         string
	Token            string
	IPAddress        string
	Fingerprint      string // 浏览器指纹 visitorId
	DeviceKey        string // sha256(ip||fingerprint)，防多开维度
	RecentMoves      []RpsMove
	PlayerID         string // long-term identity (not public)
	PlayerSecretHash string
	Persistent       bool
	CurrentSID       string
	CreatedAt        int64
	LastSeenAt       int64
	// disconnect timers (invalidated via generation)
	graceGen  int
	timerGen  int
	graceTimer *time.Timer
	discTimer  *time.Timer
}

type SeatOccupant interface {
	GetID() string
	IsBot() bool
}

// HumanSeat wraps a public player snapshot in a seat.
type HumanSeat struct {
	Player types.PublicPlayer
}

func (h *HumanSeat) GetID() string { return h.Player.ID }
func (h *HumanSeat) IsBot() bool   { return false }

// BotSeat wraps a bot.
type BotSeat struct {
	Bot types.BotPlayer
}

func (b *BotSeat) GetID() string { return b.Bot.ID }
func (b *BotSeat) IsBot() bool   { return true }

type RoomState struct {
	ID                  string
	Code                string
	UpdatedAt           int64
	Settings            types.RoomSettings
	Status              string
	Phase               types.GamePhase
	Seats               map[types.SeatKey]SeatOccupant
	SpectatorIDs        []string
	Ready               map[types.SeatKey]bool
	Choices             map[types.SeatKey]types.Move
	RevealedChoices     map[types.SeatKey]types.Move
	Othello             *types.OthelloState
	TicTacToe           *types.TicTacToeState
	ResultText          string
	PunishedPlayerIDs   []string
	Proofs              []types.PunishmentProof
	Score               map[types.SeatKey]int
	SeatedScore         map[types.SeatKey]int
	SeatStats           map[types.SeatKey]types.SeatStats
	RoundHistory        []types.RoundHistoryItem
	OwnerID             string
	LockedSeatIDs       map[string]struct{}
	ForgiveAdvantage    *forgiveAdvantage
	DisconnectForfeits  map[string]DisconnectForfeit
	CreatedAt           int64
}

type forgiveAdvantage struct {
	BeneficiaryID string
	TargetID      string
}

type broadcastMetric struct {
	Type  string // room | lobby
	Bytes int
	At    int64
}

type roomBroadcastPending struct {
	timer       *time.Timer
	updateLobby bool
}

// Client is a connected WebSocket peer.
type Client struct {
	id            string
	conn          *websocket.Conn
	writeMu       sync.Mutex
	// 每连接独立发送队列 + 写协程：广播/应答在持锁状态下只做非阻塞入队，
	// 真正的网络写在 writeLoop 里完成，避免慢客户端在 s.mu 下卡住全局。
	sendCh        chan []byte
	done          chan struct{}
	closeOnce     sync.Once
	sid           string
	token         string
	sessionExp    int64
	ipAddress     string
	// fingerprint 为浏览器 visitorId；deviceKey = sha256(ip||fp)，用于防多开
	fingerprint   string
	deviceKey      string
	playerID      string
	rooms         map[string]struct{}
	closed        bool
	// replaced：同 SID 被新连接顶替时置位，断线清理不再动玩家 Connected 状态
	replaced      bool
	userAgent     string
	host          string
	origin        string
}

// Server holds all game state.
type Server struct {
	mu sync.Mutex

	cfg types.AppConfig

	players        map[string]*PlayerState
	tokenToPlayer  map[string]string
	playerIdToID   map[string]string // long-term playerId -> public id
	sidToPlayerID  map[string]string
	rooms          map[string]*RoomState
	clients        map[string]*Client // client.id -> client
	socketToClient map[string]*Client // same as clients by id

	botTimers              map[string]*time.Timer
	othelloSettlementTimers map[string]*time.Timer
	ticTacToeGiveawayTimers map[string]*time.Timer

	// deviceCreateAttempts：按 deviceKey 记录 10 分钟内新建玩家时间戳
	deviceCreateAttempts map[string][]int64
	// chatDB：房间/大厅聊天的 SQLite 持久化存储（重启不丢）
	chatDB               *chatStore
	adminClientIDs       map[string]struct{}
	sidToClientID        map[string]string
	clientIDToSID        map[string]string
	// clientIDsByDevice：deviceKey → 当前套接字集合（同指纹限连）
	clientIDsByDevice map[string]map[string]struct{}
	rateBuckets       map[string]*rateBucket
	rateLimitBuckets  map[string]*rateLimitBucket

	recentBroadcasts    []broadcastMetric
	lobbyBroadcastDelay time.Duration
	roomBroadcastDelay  time.Duration
	lobbyBroadcastTimer *time.Timer
	roomBroadcastTimers map[string]*roomBroadcastPending

	// 状态同步通道（增量）
	syncChans map[string]*syncChannel

	// 玩家更新 100ms 聚合
	pendingPlayerUpdates map[string]*PlayerState
	playerUpdateTimer    *time.Timer

	serverStats types.ServerStats

	isProduction       bool
	sessionSecret      []byte
	sessionTtlMs       int64
	maxSocketsPerDevice int

	host string
	port int

	uploadsDir      string
	proofUploadsDir string
	adminUploadsDir string
	dataDir         string
	playersFile     string
	distDir         string

	persistMu        sync.Mutex
	persistDirty     bool
	persistScheduled bool
	immediateScheduled bool
	// 活动日志异步落盘队列（Run 启动消费者）
	logCh chan activityLogEntry

	startedAt int64
}
