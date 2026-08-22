package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaMigrationV26CreatesPunishmentPoolTables(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/database.db"
	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (25);
	`); err != nil {
		t.Fatal(err)
	}
	if err := seed.Close(); err != nil {
		t.Fatal(err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatal(err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version=%d want %d", v, currentSchemaVersion)
	}
	// v26 本身会建出 punishment_tasks/punishment_series（旧版"整表替换式"任务池），但
	// 紧随其后的 v43 会把它们连同其它旧共建表一起 DROP、改用 sub_tasks/series——
	// 从 v25 升到当前版本走完整条链后，旧表不应再存在，新表应该已经建好。
	for _, table := range []string{"punishment_tasks", "punishment_series"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != sql.ErrNoRows {
			t.Fatalf("legacy table %s should have been dropped by v43, err=%v", table, err)
		}
	}
	for _, table := range []string{"sub_tasks", "series"} {
		var name string
		if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name); err != nil {
			t.Fatalf("table %s missing: %v", table, err)
		}
	}
}
