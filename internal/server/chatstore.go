package server

import (
	"database/sql"
	"encoding/json"
	"strings"
	"sync"

	"github.com/doumiao/newRPS/internal/types"
)

// chatStore 是聊天记录的持久化存储（SQLite，CGO 驱动 mattn/go-sqlite3，连接由
// openDatabase 统一打开并与房间/惩罚事件表共用，见 database.go）。两张表：
//   - lobby_messages：大厅（留言板）聊天，全局一份。
//   - room_messages ：房间聊天，按 room_id 区分；room_name 是发送时房间名的快照
//     （不随房间改名/关闭变化），供后台按房间名检索历史消息，即便房间早已不存在。
//
// 只持久化玩家发的真实聊天；系统提示（roomNotice 等）是瞬时消息，不入库。
// seq 为自增主键，作为稳定排序与分页游标（比毫秒时间戳更可靠，不会因同毫秒并列而漏读）。
//
// deleted/deleted_at：后台聊天管理走软删除（管理员只能针对违规留言单条/批量删除，不再有
// 一键清空整表的入口）——面向普通玩家的 recent/older 一律过滤掉 deleted=1 的行，新老访客
// 都读不到；后台 search 默认同样过滤，仅 IncludeDeleted=true 时才连同已删除的一并返回，
// 供核查/恢复用。删除后 deliverChat 之外另有 admin 侧的 chat:deleted 推送，通知在线客户端
// 立即从本地视图摘除，不必等下次刷新历史。
//
// ⚠️ mattn/go-sqlite3 需要 CGO：CGO_ENABLED=0 编译出的二进制仍能启动，但驱动是一个
// stub——openDatabase 会失败（已记录到 error.log 的 database_open_failed），聊天/
// 房间/惩罚事件从此完全不落盘且没有任何显式报错，非常隐蔽。发布构建（npm run build:server）已固定
// 加 CGO_ENABLED=1，本地/CI 交叉编译或换构建机时也必须保证该环境装有可用的 C 工具链
// （gcc + libc 头文件），否则请改回 modernc.org/sqlite（纯 Go，无此限制）。
//
// 另外 CGO 意味着编译产物动态链接构建机的 glibc：运行环境（Docker 基础镜像）的 glibc
// 版本必须 >= 构建机，否则二进制可能直接无法运行（"GLIBC_x.xx not found"）。
// docker-compose.yml 因此用 debian:trixie-slim（glibc 2.41）而非 bookworm（2.36）——
// 常见构建机（如 Ubuntu 24.04, glibc 2.39）编译的产物在 bookworm 上曾经能跑只是碰巧没
// 用到 2.36 以上的符号，并非必然成立；trixie 的 glibc 版本更新，留有更多安全余量。
// 已实测：Ubuntu 24.04 宿主机编译的真实 server 二进制在 debian:trixie-slim 容器内
// 正常启动，且聊天写入/重启/读取全链路数据不丢。宿主机大版本升级后仍建议重新验证；
// 最稳妥的长期方案是在与运行环境一致的容器内构建（Dockerfile 多阶段构建）。
type chatStore struct {
	db *sql.DB
	mu sync.Mutex // SQLite 单写者：串行化写入，避免 "database is locked"
}

const chatSchema = `
CREATE TABLE IF NOT EXISTS lobby_messages (
	seq           INTEGER PRIMARY KEY AUTOINCREMENT,
	id            TEXT NOT NULL,
	player_id     TEXT,
	author        TEXT,
	author_role   TEXT,
	text          TEXT,
	mentions      TEXT,
	at            INTEGER,
	deleted       INTEGER NOT NULL DEFAULT 0,
	deleted_at    INTEGER
);
CREATE INDEX IF NOT EXISTS idx_lobby_seq ON lobby_messages(seq);
CREATE INDEX IF NOT EXISTS idx_lobby_messages_at ON lobby_messages(at);
CREATE INDEX IF NOT EXISTS idx_lobby_messages_id ON lobby_messages(id);
CREATE TABLE IF NOT EXISTS room_messages (
	seq           INTEGER PRIMARY KEY AUTOINCREMENT,
	room_id       TEXT NOT NULL,
	room_name     TEXT NOT NULL DEFAULT '',
	id            TEXT NOT NULL,
	player_id     TEXT,
	author        TEXT,
	author_role   TEXT,
	text          TEXT,
	mentions      TEXT,
	at            INTEGER,
	deleted       INTEGER NOT NULL DEFAULT 0,
	deleted_at    INTEGER
);
CREATE INDEX IF NOT EXISTS idx_room_seq ON room_messages(room_id, seq);
CREATE INDEX IF NOT EXISTS idx_room_messages_at ON room_messages(at);
CREATE INDEX IF NOT EXISTS idx_room_messages_id ON room_messages(room_id, id);
`

// idx_lobby_messages_deleted / idx_room_messages_deleted 由 schema_migrations.go 的
// v28 迁移创建，不放进上面的 chatSchema 常量——ensureSchema 对已存在的库会先无条件
// db.Exec(chatSchema) 再跑迁移，若这里提前引用 deleted 列，未升级到 v28 的旧库会在
// CREATE INDEX 那一步直接报 "no such column" 而无法启动。

func newChatStore(db *sql.DB) *chatStore {
	return &chatStore{db: db}
}

// openChatStore 单独打开 dataDir 下的共享数据库并返回聊天存储视图。生产环境走
// server.go 里的 openDatabase + newChatStore（一次打开，聊天/房间/惩罚事件共用同一个
// *sql.DB）；这个函数只在测试里用来独立开关一份数据库。
func openChatStore(dataDir string) (*chatStore, error) {
	db, err := openDatabase(dataDir)
	if err != nil {
		return nil, err
	}
	return newChatStore(db), nil
}

// Close 关闭底层数据库连接。仅供 openChatStore 独立打开的场景（如测试）配对使用；
// 生产环境的共享连接由 server.go 在关停时统一关闭。
func (c *chatStore) Close() error {
	if c == nil || c.db == nil {
		return nil
	}
	return c.db.Close()
}

func encodeMentions(mentions []string) string {
	if len(mentions) == 0 {
		return ""
	}
	b, err := json.Marshal(mentions)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeMentions(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}

// append 写入一条聊天并回填 seq；roomID 为空写大厅表，否则写房间表。roomName 是发送时刻
// 的房间名快照，仅对房间聊天有意义（大厅聊天忽略该参数），供后台按房间名检索历史消息。
func (c *chatStore) append(roomID, roomName string, msg types.ChatMessage) (int64, error) {
	if c == nil || c.db == nil {
		return 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	var (
		res sql.Result
		err error
	)
	if roomID == "" {
		res, err = c.db.Exec(
			`INSERT INTO lobby_messages (id, player_id, author, author_role, text, mentions, at)
			 VALUES (?, ?, ?, ?, ?, ?, ?)`,
			msg.ID, msg.PlayerID, msg.Author, msg.AuthorRole, msg.Text,
			encodeMentions(msg.Mentions), msg.At,
		)
	} else {
		res, err = c.db.Exec(
			`INSERT INTO room_messages (room_id, room_name, id, player_id, author, author_role, text, mentions, at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			roomID, roomName, msg.ID, msg.PlayerID, msg.Author, msg.AuthorRole, msg.Text,
			encodeMentions(msg.Mentions), msg.At,
		)
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// recent 返回最近 limit 条（按 seq 升序，即从旧到新，方便前端直接追加渲染），
// 以及是否还有更早的历史（hasMore）。已被软删除的消息不会出现在这里。
func (c *chatStore) recent(roomID string, limit int) ([]types.ChatMessage, bool, error) {
	return c.older(roomID, 0, limit)
}

// older 返回 seq < beforeSeq 的最近 limit 条（升序）+ hasMore。beforeSeq<=0 表示取最新一页。
// 已被软删除的消息（deleted=1）不会出现在这里——面向普通玩家的历史加载，新老访客都读不到
// 已删除的内容；后台核查已删除消息请用 search(IncludeDeleted: true)。
func (c *chatStore) older(roomID string, beforeSeq int64, limit int) ([]types.ChatMessage, bool, error) {
	if c == nil || c.db == nil {
		return []types.ChatMessage{}, false, nil
	}
	if limit <= 0 {
		limit = 100
	}
	// 多取一条用于判断是否还有更早历史。
	fetch := limit + 1
	var (
		rows *sql.Rows
		err  error
	)
	// 先按 seq 降序取「最新的 limit+1 条（可带 before 游标）」，再反转成升序。
	if roomID == "" {
		if beforeSeq > 0 {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM lobby_messages WHERE seq < ? AND (deleted IS NULL OR deleted = 0) ORDER BY seq DESC LIMIT ?`, beforeSeq, fetch)
		} else {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM lobby_messages WHERE (deleted IS NULL OR deleted = 0) ORDER BY seq DESC LIMIT ?`, fetch)
		}
	} else {
		if beforeSeq > 0 {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM room_messages WHERE room_id = ? AND seq < ? AND (deleted IS NULL OR deleted = 0) ORDER BY seq DESC LIMIT ?`, roomID, beforeSeq, fetch)
		} else {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM room_messages WHERE room_id = ? AND (deleted IS NULL OR deleted = 0) ORDER BY seq DESC LIMIT ?`, roomID, fetch)
		}
	}
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var desc []types.ChatMessage
	for rows.Next() {
		var (
			m        types.ChatMessage
			mentions string
		)
		if err := rows.Scan(&m.Seq, &m.ID, &m.PlayerID, &m.Author, &m.AuthorRole, &m.Text, &mentions, &m.At); err != nil {
			return nil, false, err
		}
		m.RoomID = roomID
		m.Mentions = decodeMentions(mentions)
		desc = append(desc, m)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(desc) > limit
	if hasMore {
		desc = desc[:limit]
	}
	// 反转为升序（旧 → 新）
	out := make([]types.ChatMessage, len(desc))
	for i := range desc {
		out[len(desc)-1-i] = desc[i]
	}
	return out, hasMore, nil
}

// chatMessageRef 定位一条聊天消息：RoomID=="" 表示大厅表，否则是房间表里 room_id=RoomID 的行。
// id 由 randomID() 生成、实践中全局唯一，但仍带上 RoomID 做二次限定，避免误删同 id 撞车
// （理论概率极低，双重限定几乎零成本）。
type chatMessageRef struct {
	RoomID string `json:"roomId"`
	ID     string `json:"id"`
}

func normalizeChatMessageRefs(refs []chatMessageRef) []chatMessageRef {
	out := make([]chatMessageRef, 0, len(refs))
	seen := make(map[chatMessageRef]struct{}, len(refs))
	for _, ref := range refs {
		ref.RoomID = strings.TrimSpace(ref.RoomID)
		ref.ID = strings.TrimSpace(ref.ID)
		if ref.ID == "" {
			continue
		}
		if _, ok := seen[ref]; ok {
			continue
		}
		seen[ref] = struct{}{}
		out = append(out, ref)
	}
	return out
}

// adminChatMessage 是后台聊天检索/管理专用的返回结构：在玩家可见的 ChatMessage 之上叠加
// RoomName（发送时快照，供列表展示"在哪个房间"）与软删除状态，不进入普通聊天协议。
type adminChatMessage struct {
	types.ChatMessage
	RoomName  string `json:"roomName,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
	DeletedAt int64  `json:"deletedAt,omitempty"`
}

// chatSearchQuery 是后台"聊天管理"检索面板的筛选参数：用户名（仅昵称）/聊天内容子串匹配 +
// 房间名子串匹配（或 LobbyOnly 只看大厅）。三者可任意组合，均为空则返回全部（按时间倒序分页）。
type chatSearchQuery struct {
	Author         string `json:"author"`
	Text           string `json:"text"`
	Room           string `json:"room"`
	LobbyOnly      bool   `json:"lobbyOnly"`
	IncludeDeleted bool   `json:"includeDeleted"`
	Limit          int    `json:"limit"`
	Offset         int    `json:"offset"`
}

const (
	chatSearchDefaultLimit = 50
	chatSearchMaxLimit     = 200
	chatBulkMaxRefs        = 1000
)

func clampChatSearchLimit(limit int) int {
	if limit <= 0 {
		return chatSearchDefaultLimit
	}
	if limit > chatSearchMaxLimit {
		return chatSearchMaxLimit
	}
	return limit
}

func normalizeChatSearchQuery(q chatSearchQuery) chatSearchQuery {
	q.Author = cleanText(q.Author, 64)
	q.Text = cleanText(q.Text, 200)
	q.Room = cleanText(q.Room, 64)
	q.Limit = clampChatSearchLimit(q.Limit)
	if q.Offset < 0 {
		q.Offset = 0
	}
	return q
}

// likeContainsPattern 把用户输入编成 LIKE 子串模式，并转义 \ % _，调用方必须带 ESCAPE '\'。
func likeContainsPattern(raw string) string {
	escaped := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(raw)
	return "%" + escaped + "%"
}

// chatAuthorNameMatchSQL 只匹配昵称/用户名，不匹配落在 author 里的性别、称号。
// author 运行时存的是 formatDisplayName（"性别 - 称号 - 昵称"，名争惩罚中则是惩罚名本身）：
//  1. 现档 players.name / name_war_penalty_name（改名后仍能按当前昵称搜到旧留言）
//  2. 不含 " - " 的快照（惩罚名、早期纯昵称）
//  3. 标准三段式的最后一段（历史快照里的当时昵称）
//
// table 必须是 lobby_messages / room_messages 的限定名：子查询里 players 也有
// player_id 列，不写表名会被内层吃掉，EXISTS 会对所有留言成立。
func chatAuthorNameMatchSQL(table string) string {
	author := table + ".author"
	playerID := table + ".player_id"
	return `(` +
		`EXISTS (SELECT 1 FROM players p WHERE p.id = ` + playerID + ` AND (` +
		`p.name LIKE ? ESCAPE '\' OR IFNULL(p.name_war_penalty_name,'') LIKE ? ESCAPE '\'` +
		`))` +
		` OR (instr(` + author + `, ' - ') = 0 AND ` + author + ` LIKE ? ESCAPE '\')` +
		` OR ` + author + ` LIKE '% - % - ' || ? ESCAPE '\'` +
		`)`
}

func chatAuthorNameMatchArgs(pattern string) []any {
	return []any{pattern, pattern, pattern, pattern}
}

// chatSearchConditions 为 search 组装单张表的 WHERE 子句。forRoom=true 时额外应用房间名
// 过滤（大厅表没有房间名列，房间名筛选对它没有意义）。
func chatSearchConditions(q chatSearchQuery, forRoom bool) (string, []any) {
	table := "lobby_messages"
	if forRoom {
		table = "room_messages"
	}
	var conds []string
	var args []any
	if a := strings.TrimSpace(q.Author); a != "" {
		conds = append(conds, chatAuthorNameMatchSQL(table))
		args = append(args, chatAuthorNameMatchArgs(likeContainsPattern(a))...)
	}
	if t := strings.TrimSpace(q.Text); t != "" {
		conds = append(conds, table+".text LIKE ? ESCAPE '\\'")
		args = append(args, likeContainsPattern(t))
	}
	if forRoom {
		if r := strings.TrimSpace(q.Room); r != "" {
			conds = append(conds, table+".room_name LIKE ? ESCAPE '\\'")
			args = append(args, likeContainsPattern(r))
		}
	}
	if !q.IncludeDeleted {
		conds = append(conds, "("+table+".deleted IS NULL OR "+table+".deleted = 0)")
	}
	if len(conds) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(conds, " AND "), args
}

// search 跨大厅/房间表按用户名/内容/房间名检索，供后台聊天管理列表使用。默认按时间倒序，
// 分页用 Limit/Offset（不是 seq 游标——检索结果本就是跨表混合排序，游标语义无法复用）。
// Room 非空且 LobbyOnly=false 时只搜房间（房间名子串匹配），不含大厅；其余情况下大厅与
// 房间一并检索。
func (c *chatStore) search(q chatSearchQuery) ([]adminChatMessage, bool, error) {
	if c == nil || c.db == nil {
		return []adminChatMessage{}, false, nil
	}
	q = normalizeChatSearchQuery(q)
	limit := q.Limit
	offset := q.Offset

	includeLobby := q.LobbyOnly || strings.TrimSpace(q.Room) == ""
	includeRooms := !q.LobbyOnly

	var parts []string
	var args []any
	if includeLobby {
		cond, condArgs := chatSearchConditions(q, false)
		parts = append(parts, `SELECT seq, id, '' AS room_id, '' AS room_name, player_id, author, author_role, text, mentions, at, deleted, deleted_at FROM lobby_messages`+cond)
		args = append(args, condArgs...)
	}
	if includeRooms {
		cond, condArgs := chatSearchConditions(q, true)
		parts = append(parts, `SELECT seq, id, room_id, room_name, player_id, author, author_role, text, mentions, at, deleted, deleted_at FROM room_messages`+cond)
		args = append(args, condArgs...)
	}
	if len(parts) == 0 {
		return []adminChatMessage{}, false, nil
	}

	// 多取一条用于判断是否还有更多。
	fetch := limit + 1
	query := strings.Join(parts, " UNION ALL ") + ` ORDER BY at DESC, seq DESC LIMIT ? OFFSET ?`
	args = append(args, fetch, offset)

	rows, err := c.db.Query(query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	var out []adminChatMessage
	for rows.Next() {
		var (
			row       adminChatMessage
			mentions  string
			deleted   int
			deletedAt sql.NullInt64
		)
		if err := rows.Scan(&row.Seq, &row.ID, &row.RoomID, &row.RoomName, &row.PlayerID,
			&row.Author, &row.AuthorRole, &row.Text, &mentions, &row.At, &deleted, &deletedAt); err != nil {
			return nil, false, err
		}
		row.Mentions = decodeMentions(mentions)
		row.Deleted = deleted != 0
		if deletedAt.Valid {
			row.DeletedAt = deletedAt.Int64
		}
		out = append(out, row)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	hasMore := len(out) > limit
	if hasMore {
		out = out[:limit]
	}
	return out, hasMore, nil
}

// setDeleted 批量软删除（deleted=true）或恢复（deleted=false）指定消息，按 chatMessageRef
// 定位。整批在同一事务内执行，任一条报错即整体回滚；返回实际命中（RowsAffected>0）的条数
// ——引用了不存在的 id 会被静默跳过，不视为失败。
func (c *chatStore) setDeleted(refs []chatMessageRef, deleted bool, at int64) (int, error) {
	refs = normalizeChatMessageRefs(refs)
	if c == nil || c.db == nil || len(refs) == 0 {
		return 0, nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	tx, err := c.db.Begin()
	if err != nil {
		return 0, err
	}
	var deletedAt any
	if deleted {
		deletedAt = at
	}
	affected := 0
	for _, ref := range refs {
		var (
			res sql.Result
			err error
		)
		if ref.RoomID == "" {
			res, err = tx.Exec(`UPDATE lobby_messages SET deleted = ?, deleted_at = ? WHERE id = ?`,
				boolToInt(deleted), deletedAt, ref.ID)
		} else {
			res, err = tx.Exec(`UPDATE room_messages SET deleted = ?, deleted_at = ? WHERE id = ? AND room_id = ?`,
				boolToInt(deleted), deletedAt, ref.ID, ref.RoomID)
		}
		if err != nil {
			_ = tx.Rollback()
			return 0, err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			affected += int(n)
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return affected, nil
}
