package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV8AddsFactionAndCustomGenderColumns 模拟性别/阵营解耦前停在 v7 的
// 旧库（players 表没有 faction_id/custom_gender_label 两列，阵营完全靠 gender_id 反推）。
// 验证升级后两列被正确加出来、默认空字符串，且旧行的 gender_id 原样保留（faction_id 为空
// 时 resolveGender/findFaction 会退回第一个已配置阵营，这条 SQL 迁移本身不做回填）。
func TestSchemaMigrationV8AddsFactionAndCustomGenderColumns(t *testing.T) {
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
			avatar_url TEXT NOT NULL DEFAULT '',
			ranked_points INTEGER NOT NULL DEFAULT 0,
			highest_score INTEGER NOT NULL DEFAULT 0,
			lowest_score INTEGER NOT NULL DEFAULT 0,
			title_custom INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			last_seen_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (7);
		INSERT INTO players (id, player_id, name, gender_id) VALUES ('p1', 'ident1', 'Alice', 'femboy');
	`); err != nil {
		t.Fatalf("seed v7 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v8 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	var genderID, factionID, customLabel string
	if err := db.QueryRow(`SELECT gender_id, faction_id, custom_gender_label FROM players WHERE id='p1'`).Scan(&genderID, &factionID, &customLabel); err != nil {
		t.Fatalf("query migrated columns (must exist after v8 migration): %v", err)
	}
	if genderID != "femboy" {
		t.Fatalf("gender_id = %q, want femboy (untouched by v8 migration)", genderID)
	}
	if factionID != "" || customLabel != "" {
		t.Fatalf("faction_id/custom_gender_label should default to empty on migrated rows, got %q/%q", factionID, customLabel)
	}
}

// TestIngestPersistedPlayerKeepsCustomGenderLabel 验证自定义性别（GenderID 为空、
// CustomGenderLabel 是真实文本）在重新加载时原样保留，不会被误判成"极旧数据缺字段"
// 从而被兜底成默认的 male 预设。
func TestIngestPersistedPlayerKeepsCustomGenderLabel(t *testing.T) {
	s := &Server{
		players:       map[string]*PlayerState{},
		playerIdToID:  map[string]string{},
		tokenToPlayer: map[string]string{},
		cfg: types.AppConfig{
			Genders: []types.GenderOption{
				{ID: "male", Label: "男性"},
			},
			GenderFactions: []types.GenderFaction{
				{ID: "trans_male_faction", Label: "女跨男", TaskGroup: "female"},
			},
		},
	}
	ok := s.ingestPersistedPlayer(persistedPlayer{
		ID: "p1", PlayerID: "ident1", Name: "Bob",
		GenderID: "", CustomGenderLabel: "赛博人", FactionID: "trans_male_faction",
	})
	if !ok {
		t.Fatal("ingestPersistedPlayer should succeed")
	}
	p := s.players["p1"]
	if p == nil {
		t.Fatal("player not loaded")
	}
	if p.GenderID != "" || p.GenderLabel != "赛博人" {
		t.Fatalf("custom gender not preserved: GenderID=%q GenderLabel=%q", p.GenderID, p.GenderLabel)
	}
	if p.FactionID != "trans_male_faction" {
		t.Fatalf("FactionID = %q, want trans_male_faction", p.FactionID)
	}
}
