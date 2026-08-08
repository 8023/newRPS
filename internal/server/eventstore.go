package server

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
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

// room_events：房间生命周期的统一事件日志，一个事件一行，纯 append-only，取代旧版
// rooms（一房间一行的生命周期表）+ room_join_events（可重复发生的加入事件表）两表——
// 存量历史数据已由 schema_migrations.go 的 v14 迁移无损转换进本表，旧两表随之整体
// 删除，不再保留。action 取值 "create"/"join"/"close"；
// role（战斗席 A/战斗席 B/观战）只在 action="join" 时有意义；reason
// （admin_close/empty_cleanup/server_shutdown）只在 action="close" 时有意义；
// password_hash（无盐 SHA256，房间无密码则为空字符串——不对空串取哈希，否则所有无
// 密码房间会得到同一个哈希值）只在 action="create" 时有意义，房间密码整局不变，没必要
// 等到关闭才记录。user_id/user_name 是这个事件的发起者：create 是创建者、join 是
// 加入者、close 视原因而定——admin_close 是点击关闭的管理员本人（不是创建者），
// empty_cleanup/server_shutdown 没有具体操作者，沿用创建者信息占位。
const roomEventLogSchema = `
CREATE TABLE IF NOT EXISTS room_events (
	seq           INTEGER PRIMARY KEY AUTOINCREMENT,
	at            INTEGER NOT NULL,
	room_id       TEXT NOT NULL,
	room_name     TEXT,
	game_id       TEXT,
	user_id       TEXT,
	user_name     TEXT,
	action        TEXT NOT NULL,
	role          TEXT,
	reason        TEXT,
	password_hash TEXT,
	ip            TEXT
);
CREATE INDEX IF NOT EXISTS idx_room_events_room ON room_events(room_id, at);
CREATE INDEX IF NOT EXISTS idx_room_events_user ON room_events(user_id, at);
CREATE INDEX IF NOT EXISTS idx_room_events_at ON room_events(at, action);
`

// hashRoomPassword 用无盐 SHA256 对房间密码取哈希；空密码返回空字符串（不哈希空串）。
// 房间密码只是随手设置的进门口令、不保护敏感信息，合规留痕不需要 bcrypt 之类的慢哈希。
func hashRoomPassword(password string) string {
	if password == "" {
		return ""
	}
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

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
CREATE INDEX IF NOT EXISTS idx_punishment_events_at    ON punishment_events(task_at);
CREATE INDEX IF NOT EXISTS idx_punishment_events_proof ON punishment_events(proof_at);
`

func newEventStore(db *sql.DB) *eventStore {
	return &eventStore{db: db}
}

// roomEventInput 是 insertRoomEvent 的入参；字段按 action 分组使用（见 room_events
// 表注释），用具名结构体而不是一长串同类型 string 位置参数，避免调用点传错顺序。
type roomEventInput struct {
	At           int64
	RoomID       string
	RoomName     string
	GameID       string
	UserID       string
	UserName     string
	Action       string // create / join / close
	Role         string // 战斗席 A / 战斗席 B / 观战，仅 join 有意义
	Reason       string // admin_close / empty_cleanup / server_shutdown，仅 close 有意义
	PasswordHash string // 仅 create 有意义
	IP           string
}

// insertRoomEvent 写入 room_events 的一行；对应房间生命周期里的一个事件（创建/加入/关闭）。
func (e *eventStore) insertRoomEvent(in roomEventInput) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(
		`INSERT INTO room_events (at, room_id, room_name, game_id, user_id, user_name, action, role, reason, password_hash, ip)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		in.At, in.RoomID, in.RoomName, in.GameID, in.UserID, in.UserName, in.Action, in.Role, in.Reason, in.PasswordHash, in.IP,
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
