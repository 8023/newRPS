package server

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// 应用共享 SQLite 数据库：聊天、房间/惩罚事件、Web Push、玩家档案（players + player_secrets）
// 共用同一份文件/连接（单连接，写已在各 store 内部用 mutex 串行化，符合 SQLite 单写者限制）。
//
// 早期版本文件名为 chat.db，只存聊天；现在改名 database.db 并入房间/惩罚事件/玩家表。
// 首次启动时若 database.db 不存在但旧的 chat.db 存在，会自动重命名迁移，不丢历史聊天记录。
// 旧版玩家档案 players.json：启动时 loadPlayersFromDisk 会幂等导入后改名为 players.json.migrated。
// WAL 模式下未 checkpoint 的最新数据在 -wal 边车文件里，必须连 -shm/-wal 一起搬，
// 否则只搬主文件会把最近一批还没落盘的聊天记录留在旧文件名下，等于丢数据。
//
// 建表/建索引/结构升级都交给 ensureSchema（见 schema_migrations.go）：数据库里有一张
// schema_version 表记录当前结构版本，和代码里的 currentSchemaVersion 比对，版本落后
// 就顺序跑 migrations 里显式写好的升级步骤（加列/改列/搬数据），而不是靠"建索引时
// 报错"去反推该怎么改。
func openDatabase(dataDir string) (*sql.DB, error) {
	path := filepath.Join(dataDir, "database.db")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		oldPath := filepath.Join(dataDir, "chat.db")
		if _, err := os.Stat(oldPath); err == nil {
			_ = os.Rename(oldPath, path)
			for _, suffix := range []string{"-wal", "-shm", "-journal"} {
				if _, err := os.Stat(oldPath + suffix); err == nil {
					_ = os.Rename(oldPath+suffix, path+suffix)
				}
			}
		}
	}
	// WAL + synchronous=NORMAL：写吞吐更高；代价是进程崩溃那一刻的最后一次事务可能丢失
	// （不会损坏数据库），对聊天/事件这类数据可以接受这个权衡。busy_timeout 应对偶发排队。
	dsn := path + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL&_foreign_keys=1"
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	}
	if err := ensureSchema(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database schema: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}

// allSchemas 是每个 store 各自维护的一段建表 DDL（CREATE TABLE/INDEX IF NOT EXISTS），
// 顺序无关紧要——彼此互不引用。始终反映"当前代码期望的最新结构"；已有数据库从旧结构
// 升到这个结构靠 schema_migrations.go 里的 migrations，不靠改这里的 DDL 文本本身。
var allSchemas = []string{
	chatSchema, roomEventSchema, roomEventLogSchema, punishmentEventSchema, pushSubscriptionSchema, playerSchema, activityEventSchema,
}
