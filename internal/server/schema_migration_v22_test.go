package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// seedV21DBWithWrongPunishDone 建一份停在 v21 的库：punishment_events 里有
// approved/rejected/pending，analytics_daily 里对应日的 punish_done / funnel.punish_done
// 却是错误的 0（模拟旧聚合 status='done' 的结果），publish/reject 保持正确。
func seedV21DBWithWrongPunishDone(t *testing.T) string {
	t.Helper()
	dirPath := t.TempDir()
	path := dirPath + "/database.db"

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	// 固定本地日：UTC+8 下 day=20000 对应 task_at 落在该日中段。
	// dayStart = 20000*86400000 - 480*60000
	const day int64 = 20000
	const tzMin = 480
	taskAt := day*86_400_000 - int64(tzMin)*60_000 + 12*3_600_000

	if _, err := seed.Exec(`
		CREATE TABLE punishment_events (
			id             TEXT PRIMARY KEY,
			room_id        TEXT NOT NULL,
			task_source    TEXT,
			publisher_id   TEXT,
			publisher_name TEXT,
			target_id      TEXT NOT NULL,
			target_name    TEXT,
			task_text      TEXT,
			task_at        INTEGER NOT NULL,
			proof_text     TEXT,
			image_file     TEXT,
			proof_at       INTEGER,
			status         TEXT NOT NULL,
			redo_id        TEXT
		);
		CREATE TABLE analytics_daily (
			day    INTEGER NOT NULL,
			metric TEXT NOT NULL,
			key    TEXT NOT NULL DEFAULT '',
			value  INTEGER NOT NULL DEFAULT 0,
			sealed INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (day, metric, key)
		);
		INSERT INTO punishment_events
			(id, room_id, target_id, task_text, task_at, status)
		VALUES
			('a', 'r1', 'p1', '任务A', ?, 'approved'),
			('b', 'r1', 'p2', '任务B', ?, 'approved'),
			('c', 'r1', 'p3', '任务C', ?, 'rejected'),
			('d', 'r1', 'p4', '任务D', ?, 'pending');
		INSERT INTO analytics_daily (day, metric, key, value, sealed) VALUES
			(?, 'punish_publish', '', 4, 1),
			(?, 'punish_done', '', 0, 1),
			(?, 'punish_reject', '', 1, 1),
			(?, 'funnel', 'punish_done', 0, 1),
			(?, 'dau', '', 10, 1);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (21);
	`, taskAt, taskAt+1, taskAt+2, taskAt+3, day, day, day, day, day); err != nil {
		t.Fatalf("seed v21 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}
	return dirPath
}

// TestSchemaMigrationV22FixesPunishmentDone 验证 v22 把 sealed 的错误 punish_done
// 从 0 纠正为 approved 计数，不动 publish/reject/dau。
func TestSchemaMigrationV22FixesPunishmentDone(t *testing.T) {
	t.Setenv("ANALYTICS_TZ_OFFSET_MIN", "480")
	dir := seedV21DBWithWrongPunishDone(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v22 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	const day int64 = 20000
	assertDaily := func(metric, key string, want int64) {
		t.Helper()
		var got int64
		if err := db.QueryRow(
			`SELECT value FROM analytics_daily WHERE day=? AND metric=? AND key=?`,
			day, metric, key,
		).Scan(&got); err != nil {
			t.Fatalf("query %s/%s: %v", metric, key, err)
		}
		if got != want {
			t.Fatalf("%s/%s = %d, want %d", metric, key, got, want)
		}
	}
	assertDaily(metricPunishDone, "", 2)
	assertDaily(metricPunishPublish, "", 4) // 本就正确，应保持
	assertDaily(metricPunishReject, "", 1)
	assertDaily(metricDAU, "", 10) // 无关 metric 不动
	// v23 起 funnel 不再使用 punish_done 键；v22 写过的旧键会被 v23 清掉并换成新口径。
	var funnelPunish sql.NullInt64
	if err := db.QueryRow(
		`SELECT value FROM analytics_daily WHERE day=? AND metric=? AND key=?`,
		day, metricFunnel, "punish_done",
	).Scan(&funnelPunish); err == nil {
		t.Fatalf("legacy funnel punish_done should be removed by v23, still = %d", funnelPunish.Int64)
	} else if err != sql.ErrNoRows {
		t.Fatalf("query funnel punish_done: %v", err)
	}
}
