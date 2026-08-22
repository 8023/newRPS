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
//
// performer_id/performer_name：执行这个任务的人（原 target_id/target_name）。
// approver_id/approver_name：审批这个任务证明的人（原 publisher_id/publisher_name）——
// 不管任务来源是随机/系列/玩家发布都会有值（座位制游戏固定是对手，大话骰固定是本局赢家，
// 见 canReviewPlayer/liarsDicePunishmentReviewer），不像旧版 publisher 只有玩家发布任务
// 时才有值。performer_vote/approver_vote 是这两人各自的评价（0 未投/1 赞/-1 踩），
// 直接作为持久化审计记录——防重复投票就靠这两列本身（非 0 即视为已投过，见
// contribution_vote_rpc.go 的 castContributionVote），不需要另外的内存态或投票表。
// trigger 区分这次惩罚是终局触发（round_end）还是国际象棋/斗兽棋每子惩罚触发
// （piece_capture）。tie_double_punish 标记这一行是不是"平局双罚"产生的——这种场景下
// approver_id 存的是"这局里另一个被罚的人"而不是字面意义的赢家（双罚本就没有真正的
// 赢家，但审批关系不看输赢，两人依然互审对方证明，见 canReviewPlayer），这一列纯粹是
// 审计用的显式说明，不参与任何业务判断。
// task_source/formal_series_id/formal_series_version/contributor_player_id/
// contributor_name_snapshot/contributor_anonymous 六列在 v48 迁移已 DROP：是否玩家自定义
// 任务恒等于 formal_task_id 是否为空，系列归属/贡献者/匿名与否/昵称都现查 sub_tasks +
// s.players，不必在事件行上再存一份（见 punishmentEventMeta 顶部注释）。仍留在这份 DDL
// 里，是因为更早的迁移（execSchemaQuarantiningLegacyTables 的隔离重建、
// punishment_migrate_v43.go 的存量搬迁）按列名显式写过它们，DDL 提前少列会让那些老库
// 升级路径直接报错，只能等它们跑完后再由 v48 删掉。
const punishmentEventSchema = `
CREATE TABLE IF NOT EXISTS punishment_events (
	id             TEXT PRIMARY KEY,
	room_id        TEXT NOT NULL,
	task_source    TEXT,
	approver_id    TEXT,
	approver_name  TEXT,
	performer_id   TEXT NOT NULL,
	performer_name TEXT,
	task_text      TEXT,
	task_at        INTEGER NOT NULL,
	proof_text     TEXT,
	image_file     TEXT,
	proof_at       INTEGER,
	status         TEXT NOT NULL,
	redo_id        TEXT,
	round_id       TEXT NOT NULL DEFAULT '',
	formal_task_id TEXT NOT NULL DEFAULT '',
	formal_task_version INTEGER NOT NULL DEFAULT 0,
	formal_series_id TEXT NOT NULL DEFAULT '',
	formal_series_version INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	contributor_name_snapshot TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	performer_vote INTEGER NOT NULL DEFAULT 0,
	approver_vote  INTEGER NOT NULL DEFAULT 0,
	trigger        TEXT NOT NULL DEFAULT 'round_end',
	tie_double_punish INTEGER NOT NULL DEFAULT 0,
	completed_at INTEGER
);
CREATE INDEX IF NOT EXISTS idx_punishment_events_room   ON punishment_events(room_id, task_at);
CREATE INDEX IF NOT EXISTS idx_punishment_events_at    ON punishment_events(task_at);
CREATE INDEX IF NOT EXISTS idx_punishment_events_proof ON punishment_events(proof_at);
-- idx_punishment_events_target（performer_id 上的索引）不放在这里：这段 DDL 在每次启动时
-- 无条件先于 migrations 执行，老库此时这张表还叫 target_id（改名要等 v43 迁移跑完），
-- 提前对着不存在的列建索引会直接报错。改名后的索引改在 v43 迁移里显式建
-- （punishment_migrate_v43.go），不受这个先后顺序问题影响。
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
//
// task_source/formal_series_id/formal_series_version/contributor_player_id/
// contributor_name_snapshot/contributor_anonymous 六列仍留在建表 DDL 里（老库升级路径
// 里更早的迁移——execSchemaQuarantiningLegacyTables 的隔离重建、punishment_migrate_v43.go
// 的存量搬迁——按列名显式读写过它们，DDL 提前少一列会直接报错），但现在的业务代码不再
// 读写：是否玩家自定义任务恒等于 formal_task_id 是否为空（贡献者、系列归属、匿名与否都
// 现查 sub_tasks，见 contribution_vote_rpc.go 的 atVersion；昵称统一现查 s.players，不做
// 快照），v48 迁移在所有历史迁移跑完后把这六列物理 DROP 掉。
type punishmentEventMeta struct {
	RoundID           string
	FormalTaskID      string
	FormalTaskVersion int
	Trigger           string // "" 时落库按 round_end 兜底（见建表 DEFAULT）
	TieDoublePunish   bool
}

func (e *eventStore) insertPunishmentTask(id string, taskAt int64, roomID, approverID, approverName, performerID, performerName, taskText string, meta punishmentEventMeta) error {
	if e == nil || e.db == nil {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	trigger := meta.Trigger
	if trigger == "" {
		trigger = "round_end"
	}
	_, err := e.db.Exec(
		`INSERT INTO punishment_events (
			id, room_id, approver_id, approver_name, performer_id, performer_name, task_text, task_at, status,
			round_id, formal_task_id, formal_task_version, trigger, tie_double_punish
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, roomID, approverID, approverName, performerID, performerName, taskText, taskAt, "assigned",
		meta.RoundID, meta.FormalTaskID, meta.FormalTaskVersion, trigger, boolInt(meta.TieDoublePunish),
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
	// 只有 approved 才写 completed_at（且用 COALESCE 避免重复批准覆盖首次完成时间）；
	// 非 approved 状态不带 completed 参数，避免把 completed_at 从 NULL 误固化成 0。
	if status == "approved" {
		_, err := e.db.Exec(`UPDATE punishment_events SET status = ?, completed_at = COALESCE(completed_at, ?) WHERE id = ?`, status, nowMs(), id)
		return err
	}
	_, err := e.db.Exec(`UPDATE punishment_events SET status = ? WHERE id = ?`, status, id)
	return err
}

func (e *eventStore) bindEventRound(id, roundID string) error {
	if e == nil || e.db == nil || id == "" || roundID == "" {
		return nil
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	_, err := e.db.Exec(`UPDATE punishment_events SET round_id = ? WHERE id = ?`, roundID, id)
	return err
}

func (e *eventStore) getPunishmentEvent(id string) (punishmentEventRow, error) {
	var row punishmentEventRow
	if e == nil || e.db == nil {
		return row, sql.ErrNoRows
	}
	err := e.db.QueryRow(`
		SELECT id, room_id, status, redo_id, round_id, formal_task_id, formal_task_version,
			performer_id, approver_id, performer_vote, approver_vote, completed_at
		FROM punishment_events WHERE id = ?`, id).Scan(
		&row.ID, &row.RoomID, &row.Status, &row.RedoID, &row.RoundID,
		&row.FormalTaskID, &row.FormalTaskVersion,
		&row.PerformerID, &row.ApproverID, &row.PerformerVote, &row.ApproverVote, &row.CompletedAt,
	)
	return row, err
}

type punishmentEventRow struct {
	ID                string
	RoomID            string
	Status            string
	RedoID            sql.NullString
	RoundID           string
	FormalTaskID      string
	FormalTaskVersion int
	PerformerID       string
	ApproverID        string
	PerformerVote     int
	ApproverVote      int
	CompletedAt       sql.NullInt64
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

// castPunishmentEventVoteAndAggregate 在同一个事务里写入事件侧防重列，并把票累加到
// 玩家实际体验的 formal_task_id + formal_task_version。任一步失败都回滚，避免“资格已消耗
// 但统计未增加”，也避免投稿后来修订后把旧事件的票误记到新版本。
func (e *eventStore) castPunishmentEventVoteAndAggregate(id string, asPerformer bool, vote int, taskID string, taskVersion int) (bool, error) {
	if e == nil || e.db == nil {
		return false, sql.ErrNoRows
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	tx, err := e.db.Begin()
	if err != nil {
		return false, err
	}
	defer func() { _ = tx.Rollback() }()
	col := "approver_vote"
	if asPerformer {
		col = "performer_vote"
	}
	res, err := tx.Exec(`UPDATE punishment_events SET `+col+` = ? WHERE id = ? AND `+col+` = 0`, vote, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil || n == 0 {
		return false, err
	}
	voteCol := "down_count"
	if vote == 1 {
		voteCol = "like_count"
	}
	res, err = tx.Exec(`UPDATE sub_tasks SET `+voteCol+` = `+voteCol+` + 1, updated_at = ?
		WHERE id = ? AND version = ?`, nowMs(), taskID, taskVersion)
	if err != nil {
		return false, err
	}
	if n, err = res.RowsAffected(); err != nil {
		return false, err
	} else if n == 0 {
		return false, errContributionNotFound
	}
	if err := tx.Commit(); err != nil {
		return false, err
	}
	return true, nil
}
