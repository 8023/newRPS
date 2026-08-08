package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV17AnalyticsIndexes 模拟停在 v16 的旧库，升级后应有：
// 时间维度索引、connection_events.province/isp 列、analytics_* 四张表、旧数据不丢。
func TestSchemaMigrationV17AnalyticsIndexes(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE connection_events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			socket_id TEXT NOT NULL,
			connected_at INTEGER NOT NULL,
			session_sid TEXT,
			ip TEXT,
			device TEXT,
			fingerprint TEXT,
			user_agent TEXT,
			compression TEXT,
			player_id TEXT,
			disconnected_at INTEGER NOT NULL,
			close_reason TEXT NOT NULL
		);
		CREATE INDEX IF NOT EXISTS idx_connection_events_player ON connection_events(player_id, connected_at);
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason)
		VALUES ('s1', 1000, 2000, 'disconnect');

		CREATE TABLE player_activity_events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			at INTEGER NOT NULL,
			action TEXT NOT NULL,
			player_id TEXT NOT NULL,
			new_value TEXT,
			old_value TEXT,
			ip TEXT,
			device TEXT,
			fingerprint TEXT,
			text TEXT
		);
		CREATE TABLE room_events (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			at INTEGER NOT NULL,
			room_id TEXT NOT NULL,
			room_name TEXT,
			game_id TEXT,
			user_id TEXT,
			user_name TEXT,
			action TEXT NOT NULL,
			role TEXT,
			reason TEXT,
			password_hash TEXT,
			ip TEXT
		);
		CREATE TABLE punishment_events (
			id TEXT PRIMARY KEY,
			room_id TEXT NOT NULL,
			task_source TEXT,
			publisher_id TEXT,
			publisher_name TEXT,
			target_id TEXT NOT NULL,
			target_name TEXT,
			task_text TEXT,
			task_at INTEGER NOT NULL,
			proof_text TEXT,
			image_file TEXT,
			proof_at INTEGER,
			status TEXT NOT NULL,
			redo_id TEXT
		);
		CREATE TABLE lobby_messages (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			id TEXT NOT NULL,
			player_id TEXT,
			author TEXT,
			author_role TEXT,
			text TEXT,
			mentions TEXT,
			at INTEGER
		);
		CREATE TABLE room_messages (
			seq INTEGER PRIMARY KEY AUTOINCREMENT,
			room_id TEXT NOT NULL,
			id TEXT NOT NULL,
			player_id TEXT,
			author TEXT,
			author_role TEXT,
			text TEXT,
			mentions TEXT,
			at INTEGER
		);
		CREATE TABLE pet_bonds (
			master_id TEXT NOT NULL,
			pet_id TEXT NOT NULL,
			pet_title TEXT NOT NULL DEFAULT '',
			title_updated_at INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (master_id, pet_id)
		);
		CREATE TABLE players (
			id TEXT PRIMARY KEY,
			player_id TEXT NOT NULL UNIQUE,
			claim_key TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			gender_id TEXT NOT NULL DEFAULT '',
			faction_id TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (16);
	`); err != nil {
		t.Fatalf("seed v16 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v17 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	// 旧数据还在
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM connection_events`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("connection_events count = %d err=%v, want 1", n, err)
	}

	// 新列
	ok, err := tableHasColumn(db, "connection_events", "province")
	if err != nil || !ok {
		t.Fatalf("connection_events.province missing: ok=%v err=%v", ok, err)
	}
	ok, err = tableHasColumn(db, "connection_events", "isp")
	if err != nil || !ok {
		t.Fatalf("connection_events.isp missing: ok=%v err=%v", ok, err)
	}

	// 索引
	for _, name := range []string{
		"idx_connection_events_at",
		"idx_player_activity_at",
		"idx_room_events_at",
		"idx_punishment_events_at",
		"idx_lobby_messages_at",
		"idx_players_created_at",
	} {
		if !indexExists(db, name) {
			t.Fatalf("expected index %s", name)
		}
	}

	// analytics 表
	for _, table := range []string{
		"analytics_visitors", "analytics_sessions", "analytics_events", "analytics_daily",
	} {
		ok, err := tableExists(db, table)
		if err != nil || !ok {
			t.Fatalf("table %s should exist: ok=%v err=%v", table, ok, err)
		}
	}
}

func indexExists(db *sql.DB, name string) bool {
	var n int
	err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, name).Scan(&n)
	return err == nil && n > 0
}
