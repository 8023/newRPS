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

// punishment_events：一个任务从发布到完成算一行，id 由调用方在任务发布时生成
// （randomID()），发布时 insert，证明提交/审核通过时按 id UPDATE 回填，而不是另开一行。
// task_at/proof_at 分别是任务发布、证明提交的时间，二者不共用一列。
// 驳回重做不会复用旧行：旧行 status 置为 rejected、redo_id 指向新插入的任务行 id，
// 新一轮任务重新走一次 insert，这样每次尝试都留痕，不会被覆盖。
const punishmentEventSchema = `
CREATE TABLE IF NOT EXISTS punishment_events (
	id             TEXT PRIMARY KEY,
	room_id        TEXT NOT NULL,
	task_source    TEXT,
	publisher_id   TEXT,
	publisher_name TEXT,
	target_id      TEXT NOT NULL,
	target_name    TEXT,
	task_text      TEXT,
	task_at        INTEGER NOT NULL,
	proof_text     TEXT,
	image_file     TEXT,
	proof_at       INTEGER,
	status         TEXT NOT NULL,
	redo_id        TEXT
);
CREATE INDEX IF NOT EXISTS idx_punishment_events_room   ON punishment_events(room_id, task_at);
CREATE INDEX IF NOT EXISTS idx_punishment_events_target ON punishment_events(target_id, task_at);
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

// insertPunishmentTask 在任务发布时插入一行；id 由调用方生成（randomID()），后续
// updatePunishmentProof/updatePunishmentStatus/markPunishmentRedo 都按这个 id 定位同一行。
func (e *eventStore) insertPunishmentTask(id string, taskAt int64, source, roomID, publisherID, publisherName, targetID, targetName, taskText string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`INSERT INTO punishment_events (id, room_id, task_source, publisher_id, publisher_name, target_id, target_name, task_text, task_at, status)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, roomID, source, publisherID, publisherName, targetID, targetName, taskText, taskAt, "assigned",
	)
	return err
}

// updatePunishmentProof 证明提交时回填证明字段；status 传 "pending"（待审核）或
// "approved"（系统/规则自动通过），与 types.PunishmentProof.Status 的取值保持一致。
func (e *eventStore) updatePunishmentProof(id string, proofAt int64, proofText, imageFile, status string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`UPDATE punishment_events SET proof_text = ?, image_file = ?, proof_at = ?, status = ? WHERE id = ?`,
		proofText, imageFile, proofAt, status, id,
	)
	return err
}

// updatePunishmentStatus 审核通过时把状态改为 approved，任务/证明字段不变。
func (e *eventStore) updatePunishmentStatus(id, status string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(`UPDATE punishment_events SET status = ? WHERE id = ?`, status, id)
	return err
}

// markPunishmentRedo 驳回重做时收尾旧行：status 置为 rejected，redo_id 指向新插入的
// 任务行 id，新行本身由调用方另外调用 insertPunishmentTask 写入。
func (e *eventStore) markPunishmentRedo(id, redoID string) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(`UPDATE punishment_events SET status = 'rejected', redo_id = ? WHERE id = ?`, redoID, id)
	return err
}
