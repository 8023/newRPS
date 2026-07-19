package server

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// ensureSchema 必须能在任意一张表带着"更早版本遗留、缺了后来才加的列"的旧结构时
// 自动把该表改名让路，而不是让 CREATE INDEX 因为列不存在报错、拖垮整个 openDatabase
// （进而让聊天/房间/玩家/推送等全部 SQLite 持久化一起静默退化为纯内存模式）。
// 用两张不同的表验证这不是针对 punishment_events 硬编码的一次性补丁，而是通用机制。
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
			name:   "punishment_events",
			table:  "punishment_events",
			newCol: "task_at",
			legacyDDL: `CREATE TABLE punishment_events (
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
			)`,
			legacyInsert: `INSERT INTO punishment_events (at, kind, room_id, target_id, task_text, status) VALUES (1, 'x', 'r1', 't1', 'old task', 'assigned')`,
			checkCol:     "task_text",
			wantVal:      "old task",
		},
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
		room_id TEXT NOT NULL,
		target_id TEXT,
		task_text TEXT,
		status TEXT
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
