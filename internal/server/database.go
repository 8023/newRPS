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
	if _, err := db.Exec(chatSchema + roomEventSchema + punishmentEventSchema + pushSubscriptionSchema + playerSchema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("database schema: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}
