package server

import (
	"database/sql"
	"sync"
)

// activityStore 承接原先写 work/logs/*.csv 的 players / connections 两张活动日志，改为
// SQLite 持久化，便于后台按玩家查询改名/模式开关历史、按玩家或连接排查登录记录。
// system / error 两张表触发频率或波动性不适合塞进这个共享连接，仍走 activitylog.go 的 CSV 路径。
//
// player_activity_events：低频、user-initiated 事件（改名/头像/性别阵营/大话骰名战/送礼/
// 极限模式开关等），调用点都在 s.mu 持锁期间，写入量级和 eventStore 的房间/惩罚事件相当，
// 沿用其同步加锁写模式。new_value/old_value 是泛化后的"变更后/变更前"值，具体含义随 action
// 变化：rename 存昵称，avatar_change/avatar_clear 存头像 URL，gender_change 存 genderId；
// 不适用新旧值对比的 action（如 nameWar_enable、giveaway_board_submit）只填 new_value 或都留空。
//
// connection_events：一条连接一行，但只在连接结束（正常断连 / 进程优雅关停）时才一次性
// insertConnectionEvent 写入完整一行——connect 时只把 connectedAt/compression 记在内存里
// 的 Client 结构体上（见 state.go），不落盘。这样每条连接只有一次写库，不再需要"未关闭行"
// 这种中间态、也不需要事后 UPDATE 回填；"当前谁在线"直接看内存 s.clients 即可。
// 代价：如果进程被 kill -9（而非优雅关停）而不是正常断连，当时仍在线的连接这次访问会
// 完全没有落盘记录（不是留下一条不完整的行，而是压根没有这条记录）——这是有意接受的权衡，
// 详见与用户的讨论：这是审计日志而非关键业务数据，用一次写换取更简单的数据模型。
const activityEventSchema = `
CREATE TABLE IF NOT EXISTS player_activity_events (
	seq         INTEGER PRIMARY KEY AUTOINCREMENT,
	at          INTEGER NOT NULL,
	action      TEXT NOT NULL,
	player_id   TEXT NOT NULL,
	new_value   TEXT,
	old_value   TEXT,
	ip          TEXT,
	device      TEXT,
	fingerprint TEXT,
	text        TEXT
);
CREATE INDEX IF NOT EXISTS idx_player_activity_player ON player_activity_events(player_id, at);

CREATE TABLE IF NOT EXISTS connection_events (
	seq             INTEGER PRIMARY KEY AUTOINCREMENT,
	socket_id       TEXT NOT NULL,
	connected_at    INTEGER NOT NULL,
	session_sid     TEXT,
	ip              TEXT,
	device          TEXT,
	fingerprint     TEXT,
	user_agent      TEXT,
	compression     TEXT,
	player_id       TEXT,
	disconnected_at INTEGER NOT NULL,
	close_reason    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_connection_events_player ON connection_events(player_id, connected_at);
`

type activityStore struct {
	db *sql.DB
	mu sync.Mutex
}

func newActivityStore(db *sql.DB) *activityStore {
	return &activityStore{db: db}
}

// insertPlayerActivityEvent 记录一条玩家审计事件；action 如 "create"/"rename"/
// "avatar_change"/"avatar_clear"/"gender_change"/"nameWar_enable"/"giveaway_board_submit"
// 等，见 handlers.go/http.go 各调用点。newValue/oldValue 含义随 action 变化，见上方表注释。
func (a *activityStore) insertPlayerActivityEvent(at int64, action, playerID, newValue, oldValue, ip, device, fingerprint, text string) error {
	if a == nil || a.db == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(
		`INSERT INTO player_activity_events (at, action, player_id, new_value, old_value, ip, device, fingerprint, text)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at, action, playerID, newValue, oldValue, ip, device, fingerprint, text,
	)
	return err
}

// insertConnectionEvent 在连接结束（正常断连 / 进程优雅关停批量收尾）时一次性写入完整一行；
// connectedAt 来自内存里的 Client.connectedAt 快照，playerID 传空字符串代表这条连接自始至终
// 没有关联到任何玩家（比如连上就断的游客）；closeReason 如 "disconnect"/"server_shutdown"。
func (a *activityStore) insertConnectionEvent(socketID string, connectedAt, disconnectedAt int64, sessionSID, ip, device, fingerprint, userAgent, compression, playerID, closeReason string) error {
	if a == nil || a.db == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(
		`INSERT INTO connection_events (socket_id, connected_at, session_sid, ip, device, fingerprint, user_agent, compression, player_id, disconnected_at, close_reason)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		socketID, connectedAt, sessionSID, ip, device, fingerprint, userAgent, compression, playerID, disconnectedAt, closeReason,
	)
	return err
}
