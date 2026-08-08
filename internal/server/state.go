package server

import (
	"database/sql"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/geoip"
	"github.com/doumiao/newRPS/internal/types"
)

const (
	lobbyChannel = "lobby"
	// lobbySuggestionChannel 现为「大厅聊天」实时频道：大厅视图与房间内的「大厅」tab
	// 都加入此频道以收 chat:new 增量推送（历史与首屏走 chat:load RPC，读 SQLite）。
	lobbySuggestionChannel = "lobby:suggestions"
	// petbondChannel：宠物乐园面板订阅频道。仅订阅者收到 petbond:update 个性化推送，
	// 避免上线/关系变更时对全体在线玩家 O(O·P) 构建状态。
	petbondChannel       = "petbond"
	chatPageSize         = 100
	chatLoadMorePageSize = 50
	roomHistoryPageSize  = 10
	// roomHistoryMaxKeep：内存中每房保留的对局历史条数上限（快照仍只下发最近 pageSize 条）。
	roomHistoryMaxKeep    = 100
	giveawayBoardDuration = 12 * time.Hour
	broadcastMetricWindow = 60 * time.Second
)

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

// LiarsDiceDisconnectForfeit：大话骰不进 Seats/SeatKey 体系，赢家由"入席顺序上家"
// 决定，不是固定座位——所以不能复用 DisconnectForfeit（它的字段是 SeatKey 类型的）。
type LiarsDiceDisconnectForfeit struct {
	LoserID        string
	LoserName      string
	WinnerID       string
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
	CooldownMs int64
}

type rateBucket struct {
	Hits          []int64
	CooldownUntil int64
}

type rateLimitBucket struct {
	ResetAt int64
	Count   int
}

// maxPlayerSecrets 是一个身份最多允许"记住"的设备数；超出时挤掉最早添加的一条。
const maxPlayerSecrets = 3

type PlayerState struct {
	types.PublicPlayer
	SocketID    string
	Token       string
	IPAddress   string
	Fingerprint string // 浏览器指纹 visitorId
	DeviceKey   string // sha256(ip||fingerprint)，防多开维度
	PlayerID    string // long-term identity (not public)
	// PlayerSecrets：一台设备一条，明文存储（认领/迁移方案已确认不哈希）。
	// 最多 maxPlayerSecrets 条，超出后挤掉最早的一条（见 addPlayerSecret）。
	// 服务端新签发用 randomPlayerSecret（16 字节）；前端注册可用 UUID；旧短 secret 仍有效。
	PlayerSecrets []string
	// ClaimKey：认领密钥，明文、单值、一次性（randomClaimKey，12 字节 base64url）——
	// 用于把身份"分享"给另一台设备。认领成功后立即轮换；也可手动刷新作废旧值。
	// 库内 playerId 仍为 UUID；前端展示认领码时把 id 压成 base64url。
	ClaimKey string
	// ActiveSecret：当前这条活跃 socket 是用 PlayerSecrets 里哪一条验证通过的——
	// 挤人时要避开它，不能把正在用的这条自己挤掉自己。
	ActiveSecret string
	// 推送偏好（Level 2 Web Push 用；私有字段，不进 PublicPlayer，其他玩家看不到）。
	// nil 视为未开启——和其它 *bool 偏好字段（GiveawayEnabled 等）保持同一约定。
	PushMentionEnabled *bool // 聊天 @ 我
	PushTurnEnabled    *bool // 轮到我出招/落子
	PushSeatEnabled    *bool // 我的房间参战席被坐满
	PushBondEnabled    *bool // 我的主/宠上线
	// GiveawayVotedTargets：本玩家已投过票的自救板，key 为目标玩家 ID，value 为投票时
	// 目标那条 GiveawayBoardSubmittedAt（即"板子版本"）。防止同一条 giveaway:vote 请求
	// 被抓包重放对同一条自救内容反复计分——目标重新上板（新版本）后可以再投一次。
	// 纯内存态，不落库，与其它每小时计数器（GiveawayVoteLikesThisHour 等）同一约定。
	GiveawayVotedTargets map[string]int64
	Persistent           bool
	CurrentSID           string
	CreatedAt            int64
	LastSeenAt           int64
	// disconnect timers (invalidated via generation)
	graceGen   int
	timerGen   int
	graceTimer *time.Timer
	discTimer  *time.Timer
}

// addPlayerSecret 把一条新设备凭据加进列表；超过 maxPlayerSecrets 时挤掉最早的一条
// （但绝不挤掉 ActiveSecret——不能让当前正连着的这条会话把自己顶下线），
// 返回被挤掉的那条（""表示没有挤掉任何一条），供调用方决定要不要通知那台设备。
func (p *PlayerState) addPlayerSecret(secret string) (evicted string) {
	if secret == "" {
		return ""
	}
	for _, s := range p.PlayerSecrets {
		if s == secret {
			return ""
		}
	}
	p.PlayerSecrets = append(p.PlayerSecrets, secret)
	if len(p.PlayerSecrets) > maxPlayerSecrets {
		evictIdx := 0
		if p.ActiveSecret != "" && p.PlayerSecrets[0] == p.ActiveSecret {
			evictIdx = 1 // 最早那条恰好是当前活跃会话，改挤第二早的
		}
		evicted = p.PlayerSecrets[evictIdx]
		p.PlayerSecrets = append(p.PlayerSecrets[:evictIdx], p.PlayerSecrets[evictIdx+1:]...)
	}
	return evicted
}

// hasPlayerSecret 判断某个明文 secret 是否在这个身份当前有效的设备凭据列表里。
func (p *PlayerState) hasPlayerSecret(secret string) bool {
	if secret == "" {
		return false
	}
	for _, s := range p.PlayerSecrets {
		if s == secret {
			return true
		}
	}
	return false
}

// removePlayerSecret 撤销某一条设备凭据（登出时用）。
func (p *PlayerState) removePlayerSecret(secret string) {
	out := p.PlayerSecrets[:0]
	for _, s := range p.PlayerSecrets {
		if s != secret {
			out = append(out, s)
		}
	}
	p.PlayerSecrets = out
}

// SeatOccupant is a human player snapshot in a battle seat (nil = empty).
type SeatOccupant interface {
	GetID() string
}

// HumanSeat wraps a public player snapshot in a seat.
type HumanSeat struct {
	Player types.PublicPlayer
}

func (h *HumanSeat) GetID() string { return h.Player.ID }

type RoomState struct {
	ID string

	UpdatedAt       int64
	Settings        types.RoomSettings
	Status          string
	Phase           types.GamePhase
	Seats           map[types.SeatKey]SeatOccupant
	SpectatorIDs    []string
	Ready           map[types.SeatKey]bool
	Choices         map[types.SeatKey]types.Move
	RevealedChoices map[types.SeatKey]types.Move
	Othello         *types.OthelloState
	TicTacToe       *types.TicTacToeState
	LiarsDice       *types.LiarsDiceState
	Gomoku          *types.GomokuState
	Jungle          *types.JungleState
	// LiarsDiceHands：私有骰子，playerId -> 点数列表；绝不进 roomSnapshot/广播，
	// 只通过 emitToClient 单播给玩家自己（保密性来自"只发给这一个 socket"，不需要加密）。
	LiarsDiceHands              map[string][]int
	LiarsDiceDisconnectForfeits map[string]LiarsDiceDisconnectForfeit
	ResultText                  string
	PunishedPlayerIDs           []string
	Proofs                      []types.PunishmentProof
	Score                       map[types.SeatKey]int
	SeatedScore                 map[types.SeatKey]int
	SeatStats                   map[types.SeatKey]types.SeatStats
	RoundHistory                []types.RoundHistoryItem
	OwnerID                     string
	// CreatorName：创建者的展示名快照（创建时的 playerShortName），供房间关闭时写入
	// rooms 表用；不能在关闭时现取，届时创建者可能已改名甚至不在房间里了。
	CreatorName      string
	LockedSeatIDs    map[string]struct{}
	ForgiveAdvantage *forgiveAdvantage
	// ForcedGiveawayBySeat 仅属于当前房间/当前局：seat -> 主人昵称。
	// RPS 会立即覆盖该座位本轮出拳；回合制游戏在该座位本手落子时消费。
	// GiveawayBoostedBySeat 记录五子棋本手主动白给是否已在点击按钮时加过白给值，避免主人随后强制时重复增加。
	GiveawayBoostedBySeat map[types.SeatKey]bool
	ForcedGiveawayBySeat  map[types.SeatKey]string
	DisconnectForfeits    map[string]DisconnectForfeit
	CreatedAt             int64
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
	id      string
	conn    *websocket.Conn
	writeMu sync.Mutex
	// 每连接独立发送队列 + 写协程：广播/应答在持锁状态下只做非阻塞入队，
	// 真正的网络写在 writeLoop 里完成，避免慢客户端在 s.mu 下卡住全局。
	sendCh     chan []byte
	done       chan struct{}
	closeOnce  sync.Once
	sid        string
	token      string
	sessionExp int64
	ipAddress  string
	// fingerprint 为浏览器 visitorId；deviceKey = sha256(ip||fp)，用于防多开
	fingerprint string
	deviceKey   string
	playerID    string
	rooms       map[string]struct{}
	closed      bool
	// replaced：同 SID 被新连接顶替时置位，断线清理不再动玩家 Connected 状态
	replaced  bool
	userAgent string
	host      string
	origin    string
	// connectedAt/compression：连接建立时的快照，断连时一起写进 connection_events 一行
	// （见 activitylog.go），不再在 connect 时单独落盘。
	connectedAt int64
	// onlineCreditedAt：上次把在线时长记入玩家 TotalOnlineMs 的时刻。
	// 初始化为 connectedAt；定期 checkpoint / 断线累加只计 (now - onlineCreditedAt)，
	// 避免 kill -9 等非优雅退出把整段长会话全部丢掉。connectedAt 本身不变，
	// connection_events 仍记录完整连接起止。
	onlineCreditedAt int64
	compression      string
	// connectionLogged：优雅关停路径已写过 connection_events 时置 true，
	// 避免随后 onClientDisconnect 再插一条重复记录。
	connectionLogged bool
	// ana*：分析用的派生维度，握手时算一次并缓存，之后 analytics:collect 每批只做字段取值，
	// 不在持 s.mu 时做任何字符串解析或 IP 查表。
	anaVisitor string
	anaBrowser string
	anaOS      string
	anaDevice  string
	anaGeo     geoip.Region
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
	// roomClients：逻辑频道/房间名 → 已 join 的 client.id 集合（与 Client.rooms 双向维护）。
	// emitWireToRoom 直接扫此索引，避免对全服 C 个连接做 inRoom 过滤。
	roomClients map[string]map[string]struct{}

	othelloSettlementTimers map[string]*time.Timer
	ticTacToeGiveawayTimers map[string]*time.Timer
	liarsDiceStartTimers    map[string]*time.Timer
	gomokuUndoTimers        map[string]*time.Timer
	othelloClockTimers      map[string]*time.Timer
	gomokuClockTimers       map[string]*time.Timer
	jungleClockTimers       map[string]*time.Timer

	// deviceCreateAttempts：按 deviceKey 记录 10 分钟内新建玩家时间戳
	deviceCreateAttempts map[string][]int64
	// ipCreateAttempts：纯 IP 维度（不含指纹）记录 10 分钟内新建玩家时间戳，
	// 作为 deviceCreateAttempts 的兜底——防止客户端伪造指纹绕过按设备计算的限制。
	ipCreateAttempts map[string][]int64
	// db：应用共享 SQLite 连接（database.db），chatDB/eventDB 都是它的薄封装
	db *sql.DB
	// chatDB：房间/大厅聊天的 SQLite 持久化存储（重启不丢）
	chatDB *chatStore
	// eventDB：房间生命周期 + 惩罚任务/证明事件的 SQLite 持久化存储（重启不丢）
	eventDB *eventStore
	// pushDB：Web Push 订阅存储；vapid：VAPID 密钥对（work/vapid.json 或环境变量）
	pushDB   *pushStore
	playerDB *playerStore
	// petBondDB / petBonds / petBondRequests：认主关系与待办申请（内存权威 + SQLite 写穿）
	petBondDB       *petBondStore
	petBonds        map[string]*petBond        // bondKey(master,pet) -> bond
	petBondRequests map[string]*petBondRequest // id -> pending request
	// activityDB：玩家审计事件（改名/模式开关）+ 连接生命周期事件的 SQLite 持久化存储，
	// system/error 两张活动日志表不在此列，仍走 work/logs/*.log（见 activitylog.go）
	activityDB *activityStore
	// analyticsDB：分析写路径（走 s.db）；analyticsRO 为只读聚合连接；
	// analyticsSnap 无锁发布预算结果；analyticsKick 容量 1，手动催一次重算。
	analyticsDB   *analyticsStore
	analyticsRO   *sql.DB
	analyticsSnap atomic.Pointer[analyticsSnapshot]
	analyticsKick chan struct{}
	// analyticsEnabled / analyticsGeoEnabled / analyticsTZOffsetMin / analyticsRawRetentionDays：
	// 启动时从环境变量解析，运行期只读。
	analyticsEnabled         bool
	analyticsGeoEnabled      bool
	analyticsTZOffsetMin     int
	analyticsRawRetentionDays int
	analyticsSalt            []byte
	vapid                    vapidKeys
	adminClientIDs map[string]struct{}
	sidToClientID  map[string]string
	clientIDToSID  map[string]string
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

	// dirtyPlayerIDs：待写盘的持久化玩家（s.mu 保护）。writeSnapshot 只 UPSERT 这些 ID；
	// 若集合为空则退回全量序列化，避免漏 mark 丢档。
	dirtyPlayerIDs map[string]struct{}

	serverStats types.ServerStats

	isProduction        bool
	sessionSecret       []byte
	sessionTtlMs        int64
	maxSocketsPerDevice int

	host string
	port int

	uploadsDir       string
	proofUploadsDir  string
	adminUploadsDir  string
	avatarUploadsDir string
	dataDir          string
	playersFile      string // players.json 路径，仅用作 SQLite 写失败时的降级保底（见 persist.go writePlayersJSONFallback）
	distDir          string

	persistMu          sync.Mutex
	persistDirty       bool
	persistScheduled   bool
	immediateScheduled bool
	// 活动日志异步落盘队列（Run 启动消费者）
	logCh chan activityLogEntry

	startedAt int64
}
