package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV10V11FromV9 模拟停在 v9 的旧库（无认主开关列、无斗兽棋战绩列、
// 无 pet_bonds 表）。openDatabase 应：
//  1. CREATE IF NOT EXISTS 建出 pet_bonds 等表；
//  2. 跑 v10/v11 迁移给 players 补齐列；
//  3. 旧行可被 SELECT（新列为 NULL/0，登录加载不炸）。
func TestSchemaMigrationV10V11FromV9(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	// 刻意用「缺 v10/v11 列」的精简 players 表，模拟线上 v9 结构。
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
			total_online_ms INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE player_secrets (
			player_id TEXT NOT NULL,
			secret TEXT NOT NULL,
			PRIMARY KEY (player_id, secret)
		);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (9);
		INSERT INTO players (id, player_id, claim_key, name) VALUES ('p1', 'ident-old', 'oldclaimkey', '老用户');
		INSERT INTO player_secrets (player_id, secret) VALUES ('p1', 'legacy-device-secret');
	`); err != nil {
		t.Fatalf("seed v9 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should migrate v9→v11 cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	// 新列存在且旧行可读（NULL 偏好 / 0 战绩）
	var jW, jL, jD int
	var bondM, bondP, bondPub sql.NullInt64
	if err := db.QueryRow(`
		SELECT jungle_wins, jungle_losses, jungle_draws,
		       bond_master_enabled, bond_pet_enabled, bond_public_display
		FROM players WHERE id='p1'`).Scan(&jW, &jL, &jD, &bondM, &bondP, &bondPub); err != nil {
		t.Fatalf("query migrated player columns: %v", err)
	}
	if jW != 0 || jL != 0 || jD != 0 {
		t.Fatalf("jungle WLD should default 0, got %d/%d/%d", jW, jL, jD)
	}
	if bondM.Valid || bondP.Valid || bondPub.Valid {
		t.Fatalf("bond flags should be NULL for old rows, got valid=%v/%v/%v", bondM.Valid, bondP.Valid, bondPub.Valid)
	}

	// 旧 secret 仍在，登录凭据无痛
	var secret string
	if err := db.QueryRow(`SELECT secret FROM player_secrets WHERE player_id='p1'`).Scan(&secret); err != nil {
		t.Fatalf("old secret must survive migration: %v", err)
	}
	if secret != "legacy-device-secret" {
		t.Fatalf("secret = %q, want legacy-device-secret", secret)
	}

	// pet_bonds 表已建出
	for _, table := range []string{"pet_bonds", "pet_bond_requests", "pet_bond_request_approvals"} {
		ok, err := tableExists(db, table)
		if err != nil || !ok {
			t.Fatalf("table %s should exist after migration, ok=%v err=%v", table, ok, err)
		}
	}
}
