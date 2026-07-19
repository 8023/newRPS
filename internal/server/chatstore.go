package server

import (
	"database/sql"
	"encoding/json"
	"sync"

	"github.com/doumiao/newRPS/internal/types"
)

// chatStore 是聊天记录的持久化存储（SQLite，CGO 驱动 mattn/go-sqlite3，连接由
// openDatabase 统一打开并与房间/惩罚事件表共用，见 database.go）。两张表：
//   - lobby_messages：大厅（留言板）聊天，全局一份。
//   - room_messages ：房间聊天，按 room_id 区分。
//
// 只持久化玩家发的真实聊天；系统提示（roomNotice 等）是瞬时消息，不入库。
// seq 为自增主键，作为稳定排序与分页游标（比毫秒时间戳更可靠，不会因同毫秒并列而漏读）。
//
// ⚠️ mattn/go-sqlite3 需要 CGO：CGO_ENABLED=0 编译出的二进制仍能启动，但驱动是一个
// stub——openDatabase 会失败（已记录到 error.csv 的 database_open_failed），聊天/
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
	at            INTEGER
);
CREATE INDEX IF NOT EXISTS idx_lobby_seq ON lobby_messages(seq);
CREATE TABLE IF NOT EXISTS room_messages (
	seq           INTEGER PRIMARY KEY AUTOINCREMENT,
	room_id       TEXT NOT NULL,
	id            TEXT NOT NULL,
	player_id     TEXT,
	author        TEXT,
	author_role   TEXT,
	text          TEXT,
	mentions      TEXT,
	at            INTEGER
);
CREATE INDEX IF NOT EXISTS idx_room_seq ON room_messages(room_id, seq);
`

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

// append 写入一条聊天并回填 seq；roomID 为空写大厅表，否则写房间表。
func (c *chatStore) append(roomID string, msg types.ChatMessage) (int64, error) {
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
			`INSERT INTO room_messages (room_id, id, player_id, author, author_role, text, mentions, at)
			 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			roomID, msg.ID, msg.PlayerID, msg.Author, msg.AuthorRole, msg.Text,
			encodeMentions(msg.Mentions), msg.At,
		)
	}
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// recent 返回最近 limit 条（按 seq 升序，即从旧到新，方便前端直接追加渲染），
// 以及是否还有更早的历史（hasMore）。
func (c *chatStore) recent(roomID string, limit int) ([]types.ChatMessage, bool, error) {
	return c.older(roomID, 0, limit)
}

// older 返回 seq < beforeSeq 的最近 limit 条（升序）+ hasMore。beforeSeq<=0 表示取最新一页。
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
				 FROM lobby_messages WHERE seq < ? ORDER BY seq DESC LIMIT ?`, beforeSeq, fetch)
		} else {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM lobby_messages ORDER BY seq DESC LIMIT ?`, fetch)
		}
	} else {
		if beforeSeq > 0 {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM room_messages WHERE room_id = ? AND seq < ? ORDER BY seq DESC LIMIT ?`, roomID, beforeSeq, fetch)
		} else {
			rows, err = c.db.Query(
				`SELECT seq, id, player_id, author, author_role, text, mentions, at
				 FROM room_messages WHERE room_id = ? ORDER BY seq DESC LIMIT ?`, roomID, fetch)
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

func (c *chatStore) clearLobby() error {
	if c == nil || c.db == nil {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.db.Exec(`DELETE FROM lobby_messages`)
	return err
}

func (c *chatStore) clearRoom(roomID string) error {
	if c == nil || c.db == nil || roomID == "" {
		return nil
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	_, err := c.db.Exec(`DELETE FROM room_messages WHERE room_id = ?`, roomID)
	return err
}
