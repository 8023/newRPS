package server

import (
	"database/sql"
	"encoding/json"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func TestMigrationV43PreservesLegacyTaskSeriesAndVersions(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migration.db")+"?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaVersionSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE schema_version SET version=40`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyPunishmentPoolSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyContributionEnvelopeSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(punishmentEventSchema); err != nil {
		t.Fatal(err)
	}

	pendingTask := `{"variants":[{"text":"待审任务","factionIds":["f1"]}],"inRandomPool":true,"order":10,"tagIds":[],"backgroundImage":"","backgroundOpacity":0.22}`
	if _, err := db.Exec(`INSERT INTO contribution_submissions
		(id,kind,submitter_player_id,submitter_name_snapshot,status,active_version,created_at,updated_at)
		VALUES ('task-pending','task','p1','投稿者','pending',1,10,20)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contribution_versions
		(submission_id,version,content,created_by,created_at) VALUES ('task-pending',1,?,'p1',10)`, pendingTask); err != nil {
		t.Fatal(err)
	}

	seriesV1 := `{"name":"旧系列","targetFactionIds":["f1"],"steps":[{"variants":[{"text":"第一版","factionIds":["f1"]}],"inRandomPool":true,"order":20,"tagIds":[],"backgroundImage":"","backgroundOpacity":0.22}]}`
	seriesV2 := `{"name":"旧系列","targetFactionIds":["f1"],"steps":[{"variants":[{"text":"正式第二版","factionIds":["f1"]}],"inRandomPool":true,"order":30,"tagIds":[],"backgroundImage":"","backgroundOpacity":0.22}]}`
	if _, err := db.Exec(`INSERT INTO contribution_submissions
		(id,kind,submitter_player_id,submitter_name_snapshot,status,published_target_id,published_version,active_version,created_at,updated_at,reviewed_at,reviewed_by)
		VALUES ('series-sub','series','p2','系列作者','approved','series-live',2,2,30,50,50,'admin')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contribution_versions
		(submission_id,version,content,created_by,created_at,reviewed_content,reviewed_at,reviewed_by,review_result)
		VALUES ('series-sub',1,?,'p2',30,'',0,'',''), ('series-sub',2,?,'p2',40,?,50,'admin','approved')`, seriesV1, seriesV2, seriesV2); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO punishment_tasks
		(id,text,tag_ids,faction_ids,difficulty_order,background_images,background_opacity,sort_index,
		 contributor_player_id,content_version,submission_id,task_group_id,created_at)
		VALUES ('task-row','正式第二版','[]','["f1"]',30,'[]',0.22,0,'p2',2,'series-sub','step-live',40)`); err != nil {
		t.Fatal(err)
	}
	steps, _ := json.Marshal([]map[string]any{{"taskIds": []string{"task-row"}}})
	if _, err := db.Exec(`INSERT INTO punishment_series
		(id,name,room_name_pool,room_background_images,steps,sort_index,contributor_player_id,content_version,
		 submission_id,target_faction_ids,created_at)
		VALUES ('series-live','旧系列','{}','[]',?,0,'p2',2,'series-sub','["f1"]',30)`, string(steps)); err != nil {
		t.Fatal(err)
	}

	if _, err := db.Exec(`INSERT INTO punishment_events
		(id,room_id,task_source,approver_id,performer_id,task_at,status,formal_task_id,formal_task_version)
		VALUES ('event-v2','room','system','reviewer','performer',1,'approved','step-live',2)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO contribution_votes
		(round_id,punishment_event_id,voter_player_id,target_kind,target_id,target_version,vote,created_at,updated_at)
		VALUES ('round','event-v2','performer','task','step-live',2,1,1,1)`); err != nil {
		t.Fatal(err)
	}

	if err := ensureSchema(db); err != nil {
		t.Fatal(err)
	}
	if version, err := readSchemaVersion(db); err != nil || version != currentSchemaVersion {
		t.Fatalf("schema version=%d err=%v", version, err)
	}
	for _, table := range []string{"contribution_submissions", "contribution_versions", "punishment_tasks", "punishment_series"} {
		if ok, err := tableExists(db, table); err != nil || ok {
			t.Fatalf("legacy table %s still exists: ok=%v err=%v", table, ok, err)
		}
	}

	// contributor_name 已被 v48 迁移 DROP（昵称不做快照，统一按 contributor_player_id
	// 现查 s.players），这里改为断言迁移正确保留了贡献者 ID。
	var status, contributorID string
	if err := db.QueryRow(`SELECT status, contributor_player_id FROM sub_tasks WHERE id='task-pending' AND version=1`).Scan(&status, &contributorID); err != nil {
		t.Fatal(err)
	}
	if status != "pending" || contributorID != "p1" {
		t.Fatalf("pending task status/contributor=%q/%q", status, contributorID)
	}
	var seriesVersions int
	if err := db.QueryRow(`SELECT COUNT(*) FROM series WHERE id='series-live'`).Scan(&seriesVersions); err != nil {
		t.Fatal(err)
	}
	if seriesVersions != 2 {
		t.Fatalf("series versions=%d, want exact legacy versions 1 and 2", seriesVersions)
	}
	if err := db.QueryRow(`SELECT status, contributor_player_id FROM series WHERE id='series-live' AND version=2`).Scan(&status, &contributorID); err != nil {
		t.Fatal(err)
	}
	if status != "approved" || contributorID != "p2" {
		t.Fatalf("series v2 status/contributor=%q/%q", status, contributorID)
	}
	var variants string
	if err := db.QueryRow(`SELECT variants FROM sub_tasks WHERE id='step-live' AND version=2`).Scan(&variants); err != nil {
		t.Fatal(err)
	}
	if variants == "" || variants == "[]" {
		t.Fatalf("published step v2 was not preserved: %q", variants)
	}
	var active int
	if err := db.QueryRow(`SELECT active FROM sub_tasks WHERE id='step-live' AND version=2`).Scan(&active); err != nil || active != 1 {
		t.Fatalf("step active=%d err=%v", active, err)
	}
	var likes, performerVote int
	if err := db.QueryRow(`SELECT like_count FROM sub_tasks WHERE id='step-live' AND version=2`).Scan(&likes); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT performer_vote FROM punishment_events WHERE id='event-v2'`).Scan(&performerVote); err != nil {
		t.Fatal(err)
	}
	if likes != 1 || performerVote != 1 {
		t.Fatalf("legacy vote migration likes=%d performerVote=%d", likes, performerVote)
	}
}

func TestEnsureSchemaDoesNotRecreateLegacyPunishmentTables(t *testing.T) {
	dataDir := t.TempDir()
	db, err := openDatabase(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	db, err = openDatabase(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, table := range []string{"contribution_submissions", "contribution_versions", "punishment_tasks", "punishment_series"} {
		if ok, err := tableExists(db, table); err != nil || ok {
			t.Fatalf("legacy table %s was recreated after restart: ok=%v err=%v", table, ok, err)
		}
	}
}

func TestMigrationV45DropsLegacyScaffoldsRecreatedByEarlierBuild(t *testing.T) {
	db, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "migration-v45.db")+"?_foreign_keys=1")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(schemaVersionSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE schema_version SET version=44`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyPunishmentPoolSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(legacyContributionEnvelopeSchema); err != nil {
		t.Fatal(err)
	}
	if err := ensureSchema(db); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"contribution_submissions", "contribution_versions", "punishment_tasks", "punishment_series"} {
		if ok, err := tableExists(db, table); err != nil || ok {
			t.Fatalf("legacy table %s survived v45 cleanup: ok=%v err=%v", table, ok, err)
		}
	}
}
