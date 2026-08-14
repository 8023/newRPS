package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV28ChatModeration 验证真实的 v27 旧聊天表升级到当前版本时：
// 历史消息保留、软删除/房间名字段及索引补齐，随后 v30 清理已废弃的标签偏好表。
func TestSchemaMigrationV28ChatModeration(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/database.db"
	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (27);
		CREATE TABLE lobby_messages (
			seq INTEGER PRIMARY KEY AUTOINCREMENT, id TEXT NOT NULL, player_id TEXT,
			author TEXT, author_role TEXT, text TEXT, mentions TEXT, at INTEGER
		);
		CREATE TABLE room_messages (
			seq INTEGER PRIMARY KEY AUTOINCREMENT, room_id TEXT NOT NULL, id TEXT NOT NULL,
			player_id TEXT, author TEXT, author_role TEXT, text TEXT, mentions TEXT, at INTEGER
		);
		CREATE TABLE player_punishment_tag_prefs (
			player_id TEXT NOT NULL, tag_id TEXT NOT NULL, state TEXT NOT NULL,
			PRIMARY KEY (player_id, tag_id)
		);
		INSERT INTO lobby_messages (id, player_id, author, text, mentions, at)
			VALUES ('l1', 'p1', '甲', '旧大厅消息', '[]', 100);
		INSERT INTO room_messages (room_id, id, player_id, author, text, mentions, at)
			VALUES ('r1', 'rmsg1', 'p2', '乙', '旧房间消息', '[]', 200);
		INSERT INTO player_punishment_tag_prefs VALUES ('p1', 'tag1', 'include');
	`); err != nil {
		seed.Close()
		t.Fatalf("seed schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase migrate: %v", err)
	}
	defer db.Close()

	var version int
	if err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != currentSchemaVersion {
		t.Fatalf("schema_version=%d err=%v want=%d", version, err, currentSchemaVersion)
	}
	for table, columns := range map[string][]string{
		"lobby_messages": {"deleted", "deleted_at"},
		"room_messages":  {"deleted", "deleted_at", "room_name"},
	} {
		for _, column := range columns {
			ok, err := tableHasColumn(db, table, column)
			if err != nil || !ok {
				t.Fatalf("%s.%s missing: ok=%v err=%v", table, column, ok, err)
			}
		}
	}
	for _, index := range []string{
		"idx_lobby_messages_deleted", "idx_room_messages_deleted",
		"idx_lobby_messages_id", "idx_room_messages_id",
	} {
		var count int
		if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name=?`, index).Scan(&count); err != nil || count != 1 {
			t.Fatalf("index %s count=%d err=%v", index, count, err)
		}
	}
	var lobbyDeleted, roomDeleted int
	var roomName string
	if err := db.QueryRow(`SELECT deleted FROM lobby_messages WHERE id='l1'`).Scan(&lobbyDeleted); err != nil {
		t.Fatalf("old lobby row: %v", err)
	}
	if err := db.QueryRow(`SELECT deleted, room_name FROM room_messages WHERE id='rmsg1'`).Scan(&roomDeleted, &roomName); err != nil {
		t.Fatalf("old room row: %v", err)
	}
	if lobbyDeleted != 0 || roomDeleted != 0 || roomName != "" {
		t.Fatalf("old row defaults: lobbyDeleted=%d roomDeleted=%d roomName=%q", lobbyDeleted, roomDeleted, roomName)
	}
	var tagPrefsTable int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='player_punishment_tag_prefs'`).Scan(&tagPrefsTable); err != nil {
		t.Fatalf("check tag prefs table: %v", err)
	}
	if tagPrefsTable != 0 {
		t.Fatalf("player_punishment_tag_prefs should be dropped by v30")
	}
}
