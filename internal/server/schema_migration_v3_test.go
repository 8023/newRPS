package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV3AddsRankedScoreColumnsAndBackfills 模拟一个刚好停在 v2 的旧库
// （没有 highest_score/lowest_score/ranked_last_decay_day 三列），验证升级到 v3 后：
//  1. 三列被正确加出来；
//  2. 已有的 ranked_points 按正负号回填 highest_score/lowest_score（>0 记最高，<0 记最低）；
//  3. ranked_last_decay_day 保持 NULL（交给 ingestPersistedPlayer 加载时按需兜底）。
func TestSchemaMigrationV3AddsRankedScoreColumnsAndBackfills(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path+"?_foreign_keys=on")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	// 手工建出「v2 时代」的 players 表：没有本次新增的三列。
	if _, err := seed.Exec(`
		CREATE TABLE players (
			id TEXT PRIMARY KEY,
			player_id TEXT NOT NULL UNIQUE,
			ranked_points INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (2);
		INSERT INTO players (id, player_id, ranked_points) VALUES
			('p-pos', 'ident-pos', 777),
			('p-neg', 'ident-neg', -333),
			('p-zero', 'ident-zero', 0);
	`); err != nil {
		t.Fatalf("seed v2 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v3 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	rows, err := db.Query(`SELECT id, ranked_points, highest_score, lowest_score, ranked_last_decay_day FROM players ORDER BY id`)
	if err != nil {
		t.Fatalf("query migrated columns: %v", err)
	}
	defer rows.Close()

	got := map[string][3]int{}
	nullDecay := map[string]bool{}
	for rows.Next() {
		var id string
		var rankedPoints, highest, lowest int
		var decayDay sql.NullInt64
		if err := rows.Scan(&id, &rankedPoints, &highest, &lowest, &decayDay); err != nil {
			t.Fatalf("scan: %v", err)
		}
		got[id] = [3]int{rankedPoints, highest, lowest}
		nullDecay[id] = !decayDay.Valid
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows err: %v", err)
	}

	want := map[string][3]int{
		"p-pos":  {777, 777, 0},
		"p-neg":  {-333, 0, -333},
		"p-zero": {0, 0, 0},
	}
	for id, w := range want {
		if got[id] != w {
			t.Fatalf("player %s: got (ranked,highest,lowest)=%v, want %v", id, got[id], w)
		}
		if !nullDecay[id] {
			t.Fatalf("player %s: ranked_last_decay_day should stay NULL after migration", id)
		}
	}
}

// TestSchemaMigrationFromLegacyV0AddsMissingPlayerColumns 模拟生产 v2.1.28 风格库：
// 无 schema_version、players 表缺 highest_score 等新列。确保 openDatabase 会幂等加列
// 并把版本推到 currentSchemaVersion，而不是跳过迁移却标成最新。
func TestSchemaMigrationFromLegacyV0AddsMissingPlayerColumns(t *testing.T) {
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
			claim_key TEXT NOT NULL DEFAULT '',
			player_secret_hash TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			gender_id TEXT NOT NULL DEFAULT '',
			avatar_url TEXT NOT NULL DEFAULT '',
			ranked_points INTEGER NOT NULL DEFAULT 0,
			title TEXT NOT NULL DEFAULT '',
			title_segment_id TEXT NOT NULL DEFAULT '',
			wins INTEGER NOT NULL DEFAULT 0,
			losses INTEGER NOT NULL DEFAULT 0,
			draws INTEGER NOT NULL DEFAULT 0,
			punishments INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE INDEX idx_players_player_id ON players(player_id);
		INSERT INTO players (id, player_id, name, ranked_points) VALUES ('p1','id1','Alice',100);
	`); err != nil {
		t.Fatalf("seed legacy players: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase on legacy v0: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	var highest, ranked int
	if err := db.QueryRow(`SELECT highest_score, ranked_points FROM players WHERE id='p1'`).Scan(&highest, &ranked); err != nil {
		t.Fatalf("query highest_score (must exist after v0 migration): %v", err)
	}
	if ranked != 100 {
		t.Fatalf("ranked_points = %d, want 100", ranked)
	}
	if highest != 100 {
		t.Fatalf("highest_score backfill = %d, want 100", highest)
	}

	var hashColCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('players') WHERE name = 'player_secret_hash'`).Scan(&hashColCount); err != nil {
		t.Fatalf("query players columns: %v", err)
	}
	if hashColCount != 0 {
		t.Fatalf("expected legacy player_secret_hash column to be dropped by v6 migration, still present")
	}
}
