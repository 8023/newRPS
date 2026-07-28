package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV15MigratesCustomGenderPlayers 模拟停在 v14 的旧库（players 表带
// custom_gender_label 列，几行历史数据分别落在不同阵营、选的都是自定义性别）。验证升级
// 后：每行按其 faction_id 改写成该阵营配置的默认预设性别，faction_id 未知的行兜底成
// male_faction/boy，custom_gender_label 列被彻底删除。
func TestSchemaMigrationV15MigratesCustomGenderPlayers(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE players (
			id TEXT PRIMARY KEY,
			player_id TEXT NOT NULL UNIQUE,
			name TEXT NOT NULL DEFAULT '',
			gender_id TEXT NOT NULL DEFAULT '',
			custom_gender_label TEXT NOT NULL DEFAULT '',
			faction_id TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			ranked_points INTEGER NOT NULL DEFAULT 0,
			highest_score INTEGER NOT NULL DEFAULT 0,
			lowest_score INTEGER NOT NULL DEFAULT 0,
			title_custom INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (14);
		INSERT INTO players (id, player_id, name, gender_id, custom_gender_label, faction_id) VALUES
			('p1', 'ident1', 'Alice', '', '赛博人', 'trans_male_faction'),
			('p2', 'ident2', 'Bob', '', '沃尔玛购物袋2号', 'other_faction'),
			('p3', 'ident3', 'Carol', '', '未知阵营自定义', 'no_such_faction'),
			('p4', 'ident4', 'Dave', 'femboy', '', 'femboy_faction');
	`); err != nil {
		t.Fatalf("seed v14 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v15 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	cases := []struct {
		id, wantGenderID, wantFactionID string
	}{
		{"p1", "ftm", "trans_male_faction"},
		{"p2", "attack_helicopter", "other_faction"},
		{"p3", "boy", "male_faction"},
		{"p4", "femboy", "femboy_faction"},
	}
	for _, c := range cases {
		var genderID, factionID string
		if err := db.QueryRow(`SELECT gender_id, faction_id FROM players WHERE id=?`, c.id).Scan(&genderID, &factionID); err != nil {
			t.Fatalf("query %s: %v", c.id, err)
		}
		if genderID != c.wantGenderID || factionID != c.wantFactionID {
			t.Fatalf("%s: gender_id/faction_id = %q/%q, want %q/%q", c.id, genderID, factionID, c.wantGenderID, c.wantFactionID)
		}
	}

	if has, err := tableHasColumn(db, "players", "custom_gender_label"); err != nil {
		t.Fatalf("tableHasColumn: %v", err)
	} else if has {
		t.Fatal("custom_gender_label column should be dropped after v15 migration")
	}
	if has, err := tableHasColumn(db, "players", "self_title"); err != nil {
		t.Fatalf("tableHasColumn: %v", err)
	} else if !has {
		t.Fatal("self_title column should exist after v15 migration")
	}
}
