package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV46DropsGenderNormalizedLabel verifies that a pre-v46 database
// (gender_factions/gender_options still carrying the redundant normalized_label column,
// gender_options still constrained by UNIQUE(faction_id, normalized_label)) is rebuilt
// into the new UNIQUE(faction_id, label) structure without losing existing rows, and that
// the new store-level constraint only rejects literal duplicate labels (see
// schema_migrations.go v46 comment).
func TestSchemaMigrationV46DropsGenderNormalizedLabel(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE gender_factions (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL DEFAULT '',
			normalized_label TEXT NOT NULL DEFAULT '',
			text_color TEXT NOT NULL DEFAULT '',
			background_color TEXT NOT NULL DEFAULT '',
			border_color TEXT NOT NULL DEFAULT '',
			task_group TEXT NOT NULL DEFAULT 'default',
			sort_index INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE gender_options (
			id TEXT PRIMARY KEY,
			label TEXT NOT NULL DEFAULT '',
			normalized_label TEXT NOT NULL DEFAULT '',
			faction_id TEXT NOT NULL,
			sort_index INTEGER NOT NULL DEFAULT 0,
			created_at INTEGER NOT NULL DEFAULT 0,
			updated_at INTEGER NOT NULL DEFAULT 0,
			FOREIGN KEY (faction_id) REFERENCES gender_factions(id),
			UNIQUE (faction_id, normalized_label)
		);
		INSERT INTO gender_factions (id, label, normalized_label, text_color, background_color, border_color, task_group, sort_index, created_at, updated_at)
		VALUES ('f1', '阵营一', 'f1norm', '#111111', '#222222', '#333333', 'default', 0, 100, 100);
		INSERT INTO gender_options (id, label, normalized_label, faction_id, sort_index, created_at, updated_at)
		VALUES ('g1', '猫', 'g1norm', 'f1', 0, 200, 200);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version VALUES (45);
	`); err != nil {
		t.Fatalf("seed v45 db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("run v46 migration: %v", err)
	}
	defer db.Close()

	if has, err := tableHasColumn(db, "gender_factions", "normalized_label"); err != nil || has {
		t.Fatalf("gender_factions.normalized_label should be dropped: has=%v err=%v", has, err)
	}
	if has, err := tableHasColumn(db, "gender_options", "normalized_label"); err != nil || has {
		t.Fatalf("gender_options.normalized_label should be dropped: has=%v err=%v", has, err)
	}

	var factionLabel string
	if err := db.QueryRow(`SELECT label FROM gender_factions WHERE id = 'f1'`).Scan(&factionLabel); err != nil {
		t.Fatalf("query migrated faction: %v", err)
	}
	if factionLabel != "阵营一" {
		t.Fatalf("faction row corrupted by migration: label=%q", factionLabel)
	}

	var genderLabel, genderFaction string
	if err := db.QueryRow(`SELECT label, faction_id FROM gender_options WHERE id = 'g1'`).Scan(&genderLabel, &genderFaction); err != nil {
		t.Fatalf("query migrated gender: %v", err)
	}
	if genderLabel != "猫" || genderFaction != "f1" {
		t.Fatalf("gender row corrupted by migration: label=%q faction=%q", genderLabel, genderFaction)
	}

	// 新约束是字面量 UNIQUE(faction_id, label)：完全相同的字符串仍会被拒绝。
	if _, err := db.Exec(`INSERT INTO gender_options (id, label, faction_id) VALUES ('g2', '猫', 'f1')`); err == nil {
		t.Fatal("literal duplicate label in same faction must still fail after migration")
	}
	// 但只在大小写/空白上不同的近似重复不再由 DB 约束拦截（应用层职责，见 adminSaveGenders）。
	if _, err := db.Exec(`INSERT INTO gender_options (id, label, faction_id) VALUES ('g2', ' 猫 ', 'f1')`); err != nil {
		t.Fatalf("near-duplicate (whitespace) label should be allowed by the store-level constraint: %v", err)
	}
}
