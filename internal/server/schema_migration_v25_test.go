package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV25CreatesPunishmentTables 验证从 v24 升级到当前版本时：
// player_punishment_series_progress（系列任务进度）仍在；player_punishment_tag_prefs
// 由 v25 建出、又被 v30 废弃（标签三态偏好改为纯浏览器本地存储），升级后不应残留。
func TestSchemaMigrationV25CreatesPunishmentTables(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/database.db"
	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (24);
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
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, "player_punishment_series_progress").Scan(&name); err != nil {
		t.Fatalf("table player_punishment_series_progress missing: %v", err)
	}
	err = db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, "player_punishment_tag_prefs").Scan(&name)
	if err != sql.ErrNoRows {
		t.Fatalf("table player_punishment_tag_prefs should have been dropped by v30, got err=%v", err)
	}
}
