package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestSchemaMigrationV48DropsRedundantSnapshotsWithoutLosingRows(t *testing.T) {
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
	if _, err := db.Exec(`
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version VALUES (47);
		INSERT INTO sub_tasks (
			id, version, variants, status, contributor_player_id, contributor_name, contributor_anonymous
		) VALUES ('task1', 1, '[{"text":"任务"}]', 'approved', 'p1', '旧昵称', 1);
		INSERT INTO series (
			id, version, name, status, contributor_player_id, contributor_name, contributor_anonymous
		) VALUES ('series1', 1, '系列', 'approved', 'p1', '旧昵称', 1);
		INSERT INTO punishment_events (
			id, room_id, task_source, performer_id, task_at, status,
			formal_task_id, formal_task_version, formal_series_id, formal_series_version,
			contributor_player_id, contributor_name_snapshot, contributor_anonymous
		) VALUES ('event1', 'room1', 'series', 'p2', 123, 'assigned',
			'task1', 1, 'series1', 1, 'p1', '旧昵称', 1);
	`); err != nil {
		t.Fatalf("seed v47 rows: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("migrate v47 to v48: %v", err)
	}
	defer db.Close()
	for table, cols := range map[string][]string{
		"punishment_events": {"task_source", "formal_series_id", "formal_series_version", "contributor_player_id", "contributor_name_snapshot", "contributor_anonymous"},
		"sub_tasks":         {"contributor_name"},
		"series":            {"contributor_name"},
	} {
		for _, col := range cols {
			if has, err := tableHasColumn(db, table, col); err != nil || has {
				t.Fatalf("%s.%s should be dropped: has=%v err=%v", table, col, has, err)
			}
		}
	}
	var taskID string
	var taskVersion int
	if err := db.QueryRow(`SELECT formal_task_id, formal_task_version FROM punishment_events WHERE id='event1'`).Scan(&taskID, &taskVersion); err != nil {
		t.Fatal(err)
	}
	if taskID != "task1" || taskVersion != 1 {
		t.Fatalf("event attribution corrupted: id=%q version=%d", taskID, taskVersion)
	}
	for table, id := range map[string]string{"sub_tasks": "task1", "series": "series1", "punishment_events": "event1"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM `+table+` WHERE id=?`, id).Scan(&n); err != nil || n != 1 {
			t.Fatalf("%s row lost: count=%d err=%v", table, n, err)
		}
	}
}
