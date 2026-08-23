package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaMigrationV49BackfillsFirstApprovalAndRecomputesPoolGrowth(t *testing.T) {
	t.Setenv("ANALYTICS_TZ_OFFSET_MIN", "480")
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatal(err)
	}
	for _, schema := range allSchemas {
		if _, err := db.Exec(schema); err != nil {
			t.Fatalf("seed schema: %v", err)
		}
	}
	if _, err := db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL); INSERT INTO schema_version VALUES (48)`); err != nil {
		t.Fatal(err)
	}

	day := int64(20_000)
	dayStart := day*86_400_000 - 480*60_000
	if _, err := db.Exec(`
		INSERT INTO sub_tasks (id, version, variants, status, difficulty_order, reviewed_at)
		VALUES ('task1', 1, '[]', 'approved', 10, ?),
		       ('task1', 2, '[]', 'approved', 10, ?),
		       ('legacy-task', 1, '[]', 'approved', 20, 0);
		INSERT INTO series (id, version, name, status, reviewed_at)
		VALUES ('series1', 1, '系列', 'approved', ?);
		INSERT INTO analytics_daily (day, metric, key, value, sealed)
		VALUES (?, 'dau', '', 7, 1),
		       (?, ?, '', 99, 1),
		       (?, ?, '', 99, 1),
		       (?, ?, '', 99, 1),
		       (?, ?, '', 99, 1);
	`, dayStart+1_000, dayStart+2_000, dayStart+3_000,
		day,
		day, metricPunishTaskPoolNew,
		day, metricPunishTaskPoolTotal,
		day, metricPunishSeriesPoolNew,
		day, metricPunishSeriesPoolTotal,
	); err != nil {
		t.Fatalf("seed rows: %v", err)
	}
	// 模拟真正的 v48 表：DDL 由当前 allSchemas 创建后，移除只有 v49 才存在的列。
	if _, err := db.Exec(`ALTER TABLE sub_tasks DROP COLUMN first_approved_at; ALTER TABLE series DROP COLUMN first_approved_at`); err != nil {
		t.Fatalf("shape v48 tables: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("migrate v48 to v49: %v", err)
	}
	defer db.Close()
	var version int
	if err := db.QueryRow(`SELECT version FROM schema_version`).Scan(&version); err != nil || version != 49 {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
	for table, want := range map[string]int64{"sub_tasks": dayStart + 1_000, "series": dayStart + 3_000} {
		var got int64
		if err := db.QueryRow(`SELECT MIN(first_approved_at) FROM `+table+` WHERE id=?`, map[string]string{"sub_tasks": "task1", "series": "series1"}[table]).Scan(&got); err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s first approval=%d want=%d", table, got, want)
		}
	}
	var legacyFirst int64
	if err := db.QueryRow(`SELECT first_approved_at FROM sub_tasks WHERE id='legacy-task'`).Scan(&legacyFirst); err != nil || legacyFirst != 0 {
		t.Fatalf("unrecoverable legacy approval must remain 0: got=%d err=%v", legacyFirst, err)
	}

	wantMetrics := map[string]int64{
		metricPunishTaskPoolNew:     1,
		metricPunishTaskPoolTotal:   2,
		metricPunishSeriesPoolNew:   1,
		metricPunishSeriesPoolTotal: 1,
	}
	for metric, want := range wantMetrics {
		var got int64
		var sealed int
		if err := db.QueryRow(`SELECT value, sealed FROM analytics_daily WHERE day=? AND metric=? AND key=''`, day, metric).Scan(&got, &sealed); err != nil {
			t.Fatalf("read %s: %v", metric, err)
		}
		if got != want || sealed != 1 {
			t.Fatalf("%s value/sealed=%d/%d want=%d/1", metric, got, sealed, want)
		}
	}
}
