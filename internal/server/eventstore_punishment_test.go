package server

import "testing"

func TestPunishmentEventInsertUpdateRedo(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newEventStore(db)

	// 任务发布：insert 一行，status 应为 assigned，proof 相关字段为空。
	taskID := randomID()
	if err := store.insertPunishmentTask(taskID, 1000, "system", "room1", "", "", "p1", "小明", "去阳台大喊三声"); err != nil {
		t.Fatalf("insertPunishmentTask: %v", err)
	}
	row := db.QueryRow(`SELECT task_text, status, proof_text, proof_at, redo_id FROM punishment_events WHERE id = ?`, taskID)
	var taskText, status string
	var proofText *string
	var proofAt *int64
	var redoID *string
	if err := row.Scan(&taskText, &status, &proofText, &proofAt, &redoID); err != nil {
		t.Fatalf("scan after insert: %v", err)
	}
	if taskText != "去阳台大喊三声" || status != "assigned" || proofText != nil || proofAt != nil || redoID != nil {
		t.Fatalf("unexpected row after insert: task=%q status=%q proofText=%v proofAt=%v redoID=%v", taskText, status, proofText, proofAt, redoID)
	}

	// 证明提交：按 id UPDATE 回填，不新增行。
	if err := store.updatePunishmentProof(taskID, 2000, "已经喊了", "img.jpg", "pending"); err != nil {
		t.Fatalf("updatePunishmentProof: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM punishment_events`).Scan(&count); err != nil {
		t.Fatalf("count after proof: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count after proof submit = %d, want 1 (update in place, not a new row)", count)
	}
	row = db.QueryRow(`SELECT status, proof_text, image_file, proof_at FROM punishment_events WHERE id = ?`, taskID)
	var proofTextVal, imageFile string
	var proofAtVal int64
	if err := row.Scan(&status, &proofTextVal, &imageFile, &proofAtVal); err != nil {
		t.Fatalf("scan after proof: %v", err)
	}
	if status != "pending" || proofTextVal != "已经喊了" || imageFile != "img.jpg" || proofAtVal != 2000 {
		t.Fatalf("unexpected row after proof: status=%q proof=%q image=%q at=%d", status, proofTextVal, imageFile, proofAtVal)
	}

	// 驳回重做：旧行 status=rejected 且 redo_id 指向新行；新行独立 insert，status=assigned。
	newTaskID := randomID()
	if err := store.markPunishmentRedo(taskID, newTaskID); err != nil {
		t.Fatalf("markPunishmentRedo: %v", err)
	}
	if err := store.insertPunishmentTask(newTaskID, 3000, "player", "room1", "reviewer1", "审核员", "p1", "小明", "重新去阳台大喊十声"); err != nil {
		t.Fatalf("insertPunishmentTask redo: %v", err)
	}
	row = db.QueryRow(`SELECT status, redo_id FROM punishment_events WHERE id = ?`, taskID)
	var oldStatus string
	var oldRedoID *string
	if err := row.Scan(&oldStatus, &oldRedoID); err != nil {
		t.Fatalf("scan old row after redo: %v", err)
	}
	if oldStatus != "rejected" || oldRedoID == nil || *oldRedoID != newTaskID {
		t.Fatalf("old row after redo: status=%q redoID=%v, want rejected -> %s", oldStatus, oldRedoID, newTaskID)
	}
	row = db.QueryRow(`SELECT task_text, status, publisher_id, target_name FROM punishment_events WHERE id = ?`, newTaskID)
	var newTaskText, newStatus, publisherID, targetName string
	if err := row.Scan(&newTaskText, &newStatus, &publisherID, &targetName); err != nil {
		t.Fatalf("scan new row after redo: %v", err)
	}
	if newTaskText != "重新去阳台大喊十声" || newStatus != "assigned" || publisherID != "reviewer1" || targetName != "小明" {
		t.Fatalf("unexpected new row: text=%q status=%q publisher=%q target=%q", newTaskText, newStatus, publisherID, targetName)
	}

	// 审核通过：只改 status，不动其他字段。
	if err := store.updatePunishmentStatus(newTaskID, "approved"); err != nil {
		t.Fatalf("updatePunishmentStatus: %v", err)
	}
	var finalStatus, finalTaskText string
	if err := db.QueryRow(`SELECT status, task_text FROM punishment_events WHERE id = ?`, newTaskID).Scan(&finalStatus, &finalTaskText); err != nil {
		t.Fatalf("scan after approve: %v", err)
	}
	if finalStatus != "approved" || finalTaskText != "重新去阳台大喊十声" {
		t.Fatalf("unexpected row after approve: status=%q text=%q", finalStatus, finalTaskText)
	}

	if err := db.QueryRow(`SELECT COUNT(*) FROM punishment_events`).Scan(&count); err != nil {
		t.Fatalf("final count: %v", err)
	}
	if count != 2 {
		t.Fatalf("final row count = %d, want 2 (original rejected row + one redo row)", count)
	}
}
