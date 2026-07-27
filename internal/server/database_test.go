package server

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// ensureSchema 必须能在任意一张表带着"更早版本遗留、缺了后来才加的列"的旧结构时
// 自动把该表改名让路，而不是让 CREATE INDEX 因为列不存在报错、拖垮整个 openDatabase
// （进而让聊天/房间/玩家/推送等全部 SQLite 持久化一起静默退化为纯内存模式）。
// punishment_events 隔离后会被 convertLegacyPunishmentEvents（见 v4 迁移）尽量转换进
// 新表并整体丢弃 _legacy 表，因此不再放进本表驱动用例里断言"旧数据留在 _legacy 表"，
// 单独见 TestOpenDatabaseConvertsLegacyPunishmentEvents。
func TestOpenDatabaseMigratesLegacyTables(t *testing.T) {
	cases := []struct {
		name         string
		table        string
		newCol       string // 新结构里才有的列，用来确认迁移后新表是"新结构"而不是复用了旧表
		legacyDDL    string
		legacyInsert string
		checkCol     string
		wantVal      string
	}{
		{
			// 直接对应用户点名的"用户连接"场景：模拟一张缺 connected_at/player_id 等新列的旧版
			// connection_events 表，证明这个兜底不是只认 punishment_events 这一个表名。
			name:   "connection_events",
			table:  "connection_events",
			newCol: "connected_at",
			legacyDDL: `CREATE TABLE connection_events (
				seq INTEGER PRIMARY KEY AUTOINCREMENT,
				socket_id TEXT NOT NULL,
				ip TEXT,
				note TEXT
			)`,
			legacyInsert: `INSERT INTO connection_events (socket_id, ip, note) VALUES ('s1', '127.0.0.1', 'old row')`,
			checkCol:     "note",
			wantVal:      "old row",
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "database.db")

			seed, err := sql.Open("sqlite3", path)
			if err != nil {
				t.Fatalf("seed legacy db: %v", err)
			}
			if _, err := seed.Exec(c.legacyDDL); err != nil {
				t.Fatalf("create legacy table: %v", err)
			}
			if _, err := seed.Exec(c.legacyInsert); err != nil {
				t.Fatalf("seed row: %v", err)
			}
			if err := seed.Close(); err != nil {
				t.Fatalf("close seed: %v", err)
			}

			db, err := openDatabase(dir)
			if err != nil {
				t.Fatalf("openDatabase should succeed against a legacy %s schema, got: %v", c.table, err)
			}
			defer db.Close()

			// 新表必须真的可用：新增列 + 索引都建成功了（不是复用了旧表）。
			var newColCount int
			if err := db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info(?) WHERE name = ?`, c.table, c.newCol).Scan(&newColCount); err != nil {
				t.Fatalf("query migrated table: %v", err)
			}
			if newColCount != 1 {
				t.Fatalf("migrated %s table missing expected new column %s", c.table, c.newCol)
			}

			// 旧数据没有丢：改名后的表里还能查到。
			var oldVal string
			row := db.QueryRow(`SELECT ` + c.checkCol + ` FROM ` + c.table + `_legacy LIMIT 1`)
			if err := row.Scan(&oldVal); err != nil {
				t.Fatalf("legacy row should survive under renamed table: %v", err)
			}
			if oldVal != c.wantVal {
				t.Fatalf("legacy row %s = %q, want %q", c.checkCol, oldVal, c.wantVal)
			}
		})
	}
}

// 端到端确认：迁移后真的能往新表里正常写业务数据（不只是"表存在"）。
func TestOpenDatabaseMigratedPunishmentEventsAcceptsInserts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("seed legacy db: %v", err)
	}
	if _, err := seed.Exec(`CREATE TABLE punishment_events (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		at INTEGER NOT NULL,
		kind TEXT NOT NULL,
		source TEXT,
		room_id TEXT NOT NULL,
		player_id TEXT,
		player_name TEXT,
		target_id TEXT,
		task_text TEXT,
		status TEXT,
		proof_text TEXT,
		image_file TEXT
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newEventStore(db)
	if err := store.insertPunishmentTask("id1", 100, "src", "r1", "pub", "发布者", "t1", "目标", "任务文本"); err != nil {
		t.Fatalf("insert into migrated punishment_events: %v", err)
	}
}

// TestOpenDatabaseConvertsLegacyPunishmentEvents 验证 convertLegacyPunishmentEvents
// 的实际拼接效果：一对能配上的 task+proof、一条没有 proof 的 task、一条找不到 task 的
// 孤立 proof，三种真实生产数据里都出现过的情况都要落到新表的正确行为上，且
// _legacy 表本身要被丢弃（不再有旧结构残留）。
func TestOpenDatabaseConvertsLegacyPunishmentEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("seed legacy db: %v", err)
	}
	if _, err := seed.Exec(`CREATE TABLE punishment_events (
		seq INTEGER PRIMARY KEY AUTOINCREMENT,
		at INTEGER NOT NULL,
		kind TEXT NOT NULL,
		source TEXT,
		room_id TEXT NOT NULL,
		player_id TEXT,
		player_name TEXT,
		target_id TEXT,
		task_text TEXT,
		status TEXT,
		proof_text TEXT,
		image_file TEXT
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	seedRows := []string{
		// 配对成功：发布 -> 提交证明。
		`INSERT INTO punishment_events (at, kind, source, room_id, player_id, player_name, target_id, task_text, status) VALUES (1, 'task', 'player', 'r1', 'pub1', '发布者', 'tgt1', '讲个笑话', '')`,
		`INSERT INTO punishment_events (at, kind, room_id, player_id, player_name, task_text, status, proof_text) VALUES (2, 'proof', 'r1', 'tgt1', '目标', '讲个笑话', 'approved', '哈哈哈')`,
		// 只有发布，没人提交证明：仍应处于 assigned。
		`INSERT INTO punishment_events (at, kind, source, room_id, player_id, player_name, target_id, task_text, status) VALUES (3, 'task', 'system', 'r1', '', '', 'tgt2', '唱首歌', '')`,
		// 孤立 proof：找不到对应 task 行。
		`INSERT INTO punishment_events (at, kind, room_id, player_id, player_name, task_text, status, proof_text) VALUES (4, 'proof', 'r2', 'tgt3', '孤立目标', '对方已离开', 'pending', '已完成')`,
	}
	for _, s := range seedRows {
		if _, err := seed.Exec(s); err != nil {
			t.Fatalf("seed row: %v", err)
		}
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should succeed against a legacy punishment_events schema, got: %v", err)
	}
	defer db.Close()

	var legacyCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name LIKE 'punishment_events_legacy%'`).Scan(&legacyCount); err != nil {
		t.Fatalf("query sqlite_master: %v", err)
	}
	if legacyCount != 0 {
		t.Fatalf("expected legacy punishment_events table to be dropped after conversion, still found %d", legacyCount)
	}

	var total int
	if err := db.QueryRow(`SELECT COUNT(*) FROM punishment_events`).Scan(&total); err != nil {
		t.Fatalf("count punishment_events: %v", err)
	}
	if total != 3 {
		t.Fatalf("expected 3 converted rows (1 matched task+proof, 1 open task, 1 orphan proof), got %d", total)
	}

	var publisherID, targetName, status, proofText string
	if err := db.QueryRow(`SELECT publisher_id, target_name, status, proof_text FROM punishment_events WHERE task_text = '讲个笑话'`).
		Scan(&publisherID, &targetName, &status, &proofText); err != nil {
		t.Fatalf("query matched row: %v", err)
	}
	if publisherID != "pub1" || targetName != "目标" || status != "approved" || proofText != "哈哈哈" {
		t.Fatalf("matched task+proof row not merged correctly: publisher_id=%q target_name=%q status=%q proof_text=%q",
			publisherID, targetName, status, proofText)
	}

	var openStatus string
	var openProofText sql.NullString
	if err := db.QueryRow(`SELECT status, proof_text FROM punishment_events WHERE task_text = '唱首歌'`).
		Scan(&openStatus, &openProofText); err != nil {
		t.Fatalf("query open task row: %v", err)
	}
	if openStatus != "assigned" || openProofText.Valid {
		t.Fatalf("task without a matching proof should stay assigned with no proof, got status=%q proof_text=%v", openStatus, openProofText)
	}

	var orphanTargetID, orphanPublisherID string
	if err := db.QueryRow(`SELECT target_id, publisher_id FROM punishment_events WHERE task_text = '对方已离开'`).
		Scan(&orphanTargetID, &orphanPublisherID); err != nil {
		t.Fatalf("query orphan proof row: %v", err)
	}
	if orphanTargetID != "tgt3" || orphanPublisherID != "" {
		t.Fatalf("orphan proof row should synthesize target from submitter with empty publisher, got target_id=%q publisher_id=%q", orphanTargetID, orphanPublisherID)
	}
}

// TestOpenDatabaseMigratesLegacyRoomsIntoRoomEvents 验证 v14 迁移把旧版 rooms（含
// 更早版本遗留的 code 列，先经 v5 丢弃）+ room_join_events 两表的历史数据无损转换进
// room_events，然后把两张旧表彻底删除——覆盖 create/close/join 三种事件、以及
// join 行按 room_id 反查 room_name/game_id 的场景。
func TestOpenDatabaseMigratesLegacyRoomsIntoRoomEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("seed legacy db: %v", err)
	}
	if _, err := seed.Exec(`CREATE TABLE rooms (
		room_id      TEXT PRIMARY KEY,
		code         TEXT,
		room_name    TEXT,
		game_id      TEXT,
		creator_id   TEXT,
		creator_name TEXT,
		created_at   INTEGER NOT NULL,
		closed_at    INTEGER,
		close_reason TEXT
	)`); err != nil {
		t.Fatalf("create legacy rooms table: %v", err)
	}
	if _, err := seed.Exec(
		`INSERT INTO rooms (room_id, code, room_name, game_id, creator_id, creator_name, created_at, closed_at, close_reason)
		 VALUES ('r1', 'DM-1234', '房间一', 'rps', 'p1', '玩家一', 1, 100, 'empty_cleanup'),
		        ('r2', NULL, '房间二', 'othello', 'p2', '玩家二', 2, NULL, NULL)`,
	); err != nil {
		t.Fatalf("seed rooms row: %v", err)
	}
	if _, err := seed.Exec(`CREATE TABLE room_join_events (
		seq         INTEGER PRIMARY KEY AUTOINCREMENT,
		at          INTEGER NOT NULL,
		room_id     TEXT NOT NULL,
		player_id   TEXT,
		player_name TEXT,
		role        TEXT,
		ip          TEXT
	)`); err != nil {
		t.Fatalf("create legacy room_join_events table: %v", err)
	}
	if _, err := seed.Exec(
		`INSERT INTO room_join_events (at, room_id, player_id, player_name, role, ip)
		 VALUES (50, 'r1', 'p3', '玩家三', '观战', '127.0.0.1')`,
	); err != nil {
		t.Fatalf("seed room_join_events row: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	for _, table := range []string{"rooms", "room_join_events"} {
		var name string
		err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
		if err != sql.ErrNoRows {
			t.Fatalf("expected legacy table %s to be dropped, err=%v", table, err)
		}
	}

	var roomName, userName string
	if err := db.QueryRow(`SELECT room_name, user_name FROM room_events WHERE room_id = 'r1' AND action = 'create'`).
		Scan(&roomName, &userName); err != nil {
		t.Fatalf("query migrated create event: %v", err)
	}
	if roomName != "房间一" || userName != "玩家一" {
		t.Fatalf("create event = (%q, %q), want (房间一, 玩家一)", roomName, userName)
	}

	var closeReason string
	if err := db.QueryRow(`SELECT reason FROM room_events WHERE room_id = 'r1' AND action = 'close'`).Scan(&closeReason); err != nil {
		t.Fatalf("query migrated close event: %v", err)
	}
	if closeReason != "empty_cleanup" {
		t.Fatalf("close event reason = %q, want empty_cleanup", closeReason)
	}

	var closeCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM room_events WHERE room_id = 'r2' AND action = 'close'`).Scan(&closeCount); err != nil {
		t.Fatalf("count r2 close events: %v", err)
	}
	if closeCount != 0 {
		t.Fatalf("room r2 never closed (closed_at NULL) should not synthesize a close event, got %d", closeCount)
	}

	var joinRoomName, joinUserName string
	if err := db.QueryRow(`SELECT room_name, user_name FROM room_events WHERE room_id = 'r1' AND action = 'join'`).
		Scan(&joinRoomName, &joinUserName); err != nil {
		t.Fatalf("query migrated join event: %v", err)
	}
	if joinRoomName != "房间一" || joinUserName != "玩家三" {
		t.Fatalf("join event = (%q, %q), want (房间一, 玩家三)", joinRoomName, joinUserName)
	}
}
