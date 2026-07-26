package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV12BackfillOnlineMs 验证 v12 用 connection_events 时长之和抬升
// 偏低的 total_online_ms，且不会压低已更大的值。
func TestSchemaMigrationV12BackfillOnlineMs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open seed: %v", err)
	}
	// 停在 v11：已有 total_online_ms 列，但关停漏写导致严重偏低。
	if _, err := seed.Exec(`
		CREATE TABLE players (
			id TEXT PRIMARY KEY,
			player_id TEXT NOT NULL UNIQUE,
			claim_key TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			gender_id TEXT NOT NULL DEFAULT '',
			custom_gender_label TEXT NOT NULL DEFAULT '',
			faction_id TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			wins INTEGER NOT NULL DEFAULT 0,
			losses INTEGER NOT NULL DEFAULT 0,
			draws INTEGER NOT NULL DEFAULT 0,
			punishments INTEGER NOT NULL DEFAULT 0,
			ranked_points INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			title_segment_id TEXT NOT NULL DEFAULT '',
			title_custom INTEGER NOT NULL DEFAULT 0,
			highest_score INTEGER NOT NULL DEFAULT 0,
			lowest_score INTEGER NOT NULL DEFAULT 0,
			rps_wins INTEGER NOT NULL DEFAULT 0,
			rps_losses INTEGER NOT NULL DEFAULT 0,
			rps_draws INTEGER NOT NULL DEFAULT 0,
			othello_wins INTEGER NOT NULL DEFAULT 0,
			othello_losses INTEGER NOT NULL DEFAULT 0,
			othello_draws INTEGER NOT NULL DEFAULT 0,
			tictactoe_wins INTEGER NOT NULL DEFAULT 0,
			tictactoe_losses INTEGER NOT NULL DEFAULT 0,
			tictactoe_draws INTEGER NOT NULL DEFAULT 0,
			gomoku_wins INTEGER NOT NULL DEFAULT 0,
			gomoku_losses INTEGER NOT NULL DEFAULT 0,
			gomoku_draws INTEGER NOT NULL DEFAULT 0,
			liarsdice_wins INTEGER NOT NULL DEFAULT 0,
			liarsdice_losses INTEGER NOT NULL DEFAULT 0,
			liarsdice_draws INTEGER NOT NULL DEFAULT 0,
			jungle_wins INTEGER NOT NULL DEFAULT 0,
			jungle_losses INTEGER NOT NULL DEFAULT 0,
			jungle_draws INTEGER NOT NULL DEFAULT 0,
			total_online_ms INTEGER NOT NULL DEFAULT 0,
			bond_master_enabled INTEGER,
			bond_pet_enabled INTEGER,
			bond_public_display INTEGER,
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE player_secrets (
			player_id TEXT NOT NULL,
			secret TEXT NOT NULL,
			PRIMARY KEY (player_id, secret)
		);
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
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (11);
		INSERT INTO players (id, player_id, claim_key, name, total_online_ms)
			VALUES ('ok1', 'ident-ok', 'claim', 'ok', 46673);
		INSERT INTO players (id, player_id, claim_key, name, total_online_ms)
			VALUES ('hi1', 'ident-hi', 'claim2', 'hi', 9_999_999);
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, player_id, close_reason)
			VALUES
			('s1', 1000, 1000+3600_000, 'ok1', 'server_shutdown'),
			('s2', 2000, 2000+46_673, 'ok1', 'disconnect'),
			('s3', 3000, 3000+1000, 'hi1', 'disconnect');
	`); err != nil {
		t.Fatalf("seed v11: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase migrate: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	var okMs, hiMs int64
	if err := db.QueryRow(`SELECT total_online_ms FROM players WHERE id='ok1'`).Scan(&okMs); err != nil {
		t.Fatalf("scan ok: %v", err)
	}
	// 3600000 + 46673
	if okMs != 3_646_673 {
		t.Fatalf("ok total_online_ms=%d, want 3646673 (backfilled from events)", okMs)
	}
	if err := db.QueryRow(`SELECT total_online_ms FROM players WHERE id='hi1'`).Scan(&hiMs); err != nil {
		t.Fatalf("scan hi: %v", err)
	}
	// 已大于 events 之和(1000)，不应被压低
	if hiMs != 9_999_999 {
		t.Fatalf("hi total_online_ms=%d, want 9999999 (unchanged)", hiMs)
	}
}
