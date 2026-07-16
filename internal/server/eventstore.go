package server

import (
	"database/sql"
	"sync"
)

// eventStore 持久化房间生命周期与惩罚任务/证明事件，与聊天记录共用同一个 SQLite
// 连接（见 database.go）。这几类事件在当前规模下（约 90 在线 / 20 房间，多数观战）
// 触发频率很低（开房/进房几分钟一次量级，惩罚任务/证明每局才一次），沿用聊天记录
// 已验证过的同步写入模式（加锁 + db.Exec），不需要额外的异步队列。
type eventStore struct {
	db *sql.DB
	mu sync.Mutex
}

// rooms：房间生命周期实体表，一房间一行；创建时 insertRoom，关闭时 closeRoom 补写
// closed_at/close_reason。
//
// room_join_events：可重复发生的加入事件（战斗席/观战都会触发），一次进房一行，
// 不适合塞进 rooms 单行。
const roomEventSchema = `
CREATE TABLE IF NOT EXISTS rooms (
	room_id      TEXT PRIMARY KEY,
	code         TEXT,
	room_name    TEXT,
	game_id      TEXT,
	creator_id   TEXT,
	creator_name TEXT,
	created_at   INTEGER NOT NULL,
	closed_at    INTEGER,
	close_reason TEXT
);
CREATE TABLE IF NOT EXISTS room_join_events (
	seq         INTEGER PRIMARY KEY AUTOINCREMENT,
	at          INTEGER NOT NULL,
	room_id     TEXT NOT NULL,
	player_id   TEXT,
	player_name TEXT,
	role        TEXT,
	ip          TEXT
);
CREATE INDEX IF NOT EXISTS idx_room_join_room   ON room_join_events(room_id, at);
CREATE INDEX IF NOT EXISTS idx_room_join_player ON room_join_events(player_id, at);
`

// punishment_events：任务发布（系统/玩家）与证明提交，三个业务动作并列的事件日志，
// kind/source 区分具体类型；player_id/target_id 的语义随 kind 而不同（task 时
// player_id 是发布者、target_id 是被罚玩家；proof 时 player_id 是提交者、target_id 为空）。
const punishmentEventSchema = `
CREATE TABLE IF NOT EXISTS punishment_events (
	seq         INTEGER PRIMARY KEY AUTOINCREMENT,
	at          INTEGER NOT NULL,
	kind        TEXT NOT NULL,
	source      TEXT,
	room_id     TEXT NOT NULL,
	player_id   TEXT,
	player_name TEXT,
	target_id   TEXT,
	task_text   TEXT,
	status      TEXT,
	proof_text  TEXT,
	image_file  TEXT
);
CREATE INDEX IF NOT EXISTS idx_punishment_events_room   ON punishment_events(room_id, at);
CREATE INDEX IF NOT EXISTS idx_punishment_events_player ON punishment_events(player_id, at);
`

func newEventStore(db *sql.DB) *eventStore {
	return &eventStore{db: db}
}

// insertRoom 在房间创建时写入一行；createdAt 为 unix 毫秒。
func (e *eventStore) insertRoom(roomID, code, roomName, gameID, creatorID, creatorName string, createdAt int64) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`INSERT INTO rooms (room_id, code, room_name, game_id, creator_id, creator_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		roomID, code, roomName, gameID, creatorID, creatorName, createdAt,
	)
	return err
}

// closeRoom 在房间关闭（自动清理空房 / 管理员关闭）时补写 closed_at/close_reason。
// 只更新尚未关闭的行，避免同一房间被重复关闭时覆盖第一次的关闭原因。
func (e *eventStore) closeRoom(roomID string, closedAt int64, reason string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`UPDATE rooms SET closed_at = ?, close_reason = ? WHERE room_id = ? AND closed_at IS NULL`,
		closedAt, reason, roomID,
	)
	return err
}

// insertRoomJoinEvent 记录一次进房（战斗席或观战）；role 建议传 "战斗席A"/"战斗席B"/"观战"。
func (e *eventStore) insertRoomJoinEvent(at int64, roomID, playerID, playerName, role, ip string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`INSERT INTO room_join_events (at, room_id, player_id, player_name, role, ip)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		at, roomID, playerID, playerName, role, ip,
	)
	return err
}

// insertPunishmentEvent 记录一次任务发布（kind="task"）或证明提交（kind="proof"）。
func (e *eventStore) insertPunishmentEvent(at int64, kind, source, roomID, playerID, playerName, targetID, taskText, status, proofText, imageFile string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`INSERT INTO punishment_events (at, kind, source, room_id, player_id, player_name, target_id, task_text, status, proof_text, image_file)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		at, kind, source, roomID, playerID, playerName, targetID, taskText, status, proofText, imageFile,
	)
	return err
}
