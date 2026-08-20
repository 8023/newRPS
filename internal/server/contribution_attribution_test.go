package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestMetaForFormalTaskAlwaysVotesOnTaskGroup 确认投票目标恒定落在 TaskGroupID 这一级——
// 不管任务是独立投稿、还是系列某一步引用的任务，只要有贡献者就必须拿到一个 FormalTaskID，
// 且必须优先取 TaskGroupID（同一步/同一份投稿的所有阵营变体共享，但系列不同步骤各不相同），
// 只有 TaskGroupID 为空（历史遗留数据）时才退回 task.ID。
func TestMetaForFormalTaskAlwaysVotesOnTaskGroup(t *testing.T) {
	s := newTestServer(t)
	task := types.PunishmentTaskConfig{
		ID: "ct_1", SubmissionID: "sub_1", TaskGroupID: "tg_1", ContributorPlayerID: "contributor1", ContributorAnonymous: false, VoteVersion: 3,
	}
	series := &types.PunishmentSeriesTaskConfig{ID: "cs_1", ContributorPlayerID: "", VoteVersion: 0}

	meta := s.metaForFormalTask(task, series)

	if meta.FormalSeriesID != "" {
		t.Fatalf("series without its own contributor must not claim the vote, got FormalSeriesID=%q", meta.FormalSeriesID)
	}
	if meta.FormalTaskID != task.TaskGroupID || meta.FormalTaskVersion != task.VoteVersion {
		t.Fatalf("should attribute to the task group, got FormalTaskID=%q FormalTaskVersion=%d", meta.FormalTaskID, meta.FormalTaskVersion)
	}
	if meta.ContributorPlayerID != task.ContributorPlayerID {
		t.Fatalf("contributor should stay the task's own contributor, got %q", meta.ContributorPlayerID)
	}

	legacy := types.PunishmentTaskConfig{ID: "ct_legacy", ContributorPlayerID: "contributor1", VoteVersion: 1}
	legacyMeta := s.metaForFormalTask(legacy, series)
	if legacyMeta.FormalTaskID != "ct_legacy" {
		t.Fatalf("empty TaskGroupID should fall back to task ID, got %q", legacyMeta.FormalTaskID)
	}
}

// TestMetaForFormalTaskSeriesStepsVoteIndependently 复现用户报的设计缺陷并确认修复：系列的
// 每一步各自独立计票，不能像旧实现那样全系列共用一个 target（系列头上仍然记 FormalSeriesID
// 供溯源/完成率使用，但那不是投票目标）。
func TestMetaForFormalTaskSeriesStepsVoteIndependently(t *testing.T) {
	s := newTestServer(t)
	series := &types.PunishmentSeriesTaskConfig{
		ID: "cs_2", ContributorPlayerID: "contributor2", ContributorAnonymous: true, VoteVersion: 5,
	}
	step1 := types.PunishmentTaskConfig{ID: "ct_2a", TaskGroupID: "tg_step1", ContributorPlayerID: "contributor2", VoteVersion: 5}
	step2 := types.PunishmentTaskConfig{ID: "ct_2b", TaskGroupID: "tg_step2", ContributorPlayerID: "contributor2", VoteVersion: 5}

	meta1 := s.metaForFormalTask(step1, series)
	meta2 := s.metaForFormalTask(step2, series)

	if meta1.FormalSeriesID != series.ID || meta2.FormalSeriesID != series.ID {
		t.Fatalf("both steps should still record which series they belong to, got %q / %q", meta1.FormalSeriesID, meta2.FormalSeriesID)
	}
	if meta1.FormalTaskID != "tg_step1" || meta2.FormalTaskID != "tg_step2" {
		t.Fatalf("each step must vote on its own task group, got %q / %q", meta1.FormalTaskID, meta2.FormalTaskID)
	}
	if meta1.FormalTaskID == meta2.FormalTaskID {
		t.Fatalf("two different series steps must not share one vote target")
	}
	if meta1.ContributorPlayerID != series.ContributorPlayerID || !meta1.ContributorAnonymous {
		t.Fatalf("contributor should come from the series, got %q anon=%v", meta1.ContributorPlayerID, meta1.ContributorAnonymous)
	}
}

// TestUpsertTaskTxAppendsSortIndexInsteadOfHardcodingZero 复现审阅发现的排序问题：每批准一条
// 投稿都把新任务插到 sort_index=0，会把管理员手工排好的顺序打乱到最前面。
func TestUpsertTaskTxAppendsSortIndexInsteadOfHardcodingZero(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)
	existing := []types.PunishmentTaskConfig{
		{ID: "sys_1", Text: "第一条", Order: 10, BackgroundImages: []string{}},
		{ID: "sys_2", Text: "第二条", Order: 20, BackgroundImages: []string{}},
	}
	if err := ps.replaceTasks(existing); err != nil {
		t.Fatal(err)
	}

	tx, err := db.Begin()
	if err != nil {
		t.Fatal(err)
	}
	newTask := types.PunishmentTaskConfig{
		ID: "ct_new", Text: "投稿的任务", Order: 30,
		ContributorPlayerID: "p1", ContentVersion: 1, VoteVersion: 1,
	}
	if err := upsertTaskTx(tx, newTask); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(); err != nil {
		t.Fatal(err)
	}

	list, err := ps.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 3 || list[0].ID != "sys_1" || list[1].ID != "sys_2" {
		t.Fatalf("newly published task must not jump ahead of the curated order, got %v", taskIDs(list))
	}
	if list[2].ID != "ct_new" {
		t.Fatalf("newly published task should append at the end, got %v", taskIDs(list))
	}
}

func taskIDs(list []types.PunishmentTaskConfig) []string {
	out := make([]string, len(list))
	for i, t := range list {
		out[i] = t.ID
	}
	return out
}

// TestMigrateContributionV33BackfillsLegacyContributor 确认 v33 迁移会把上线前管理员直接在
// 后台添加、从未走过投稿流程的旧任务/系列统一归到 8023 本人名下（非匿名、已审批），且不会
// 碰到已经走过投稿流程的新数据。
func TestMigrateContributionV33BackfillsLegacyContributor(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)
	legacyTask := types.PunishmentTaskConfig{ID: "legacy_task", Text: "旧文案", Order: 15, BackgroundImages: []string{}}
	contributedTask := types.PunishmentTaskConfig{
		ID: "contrib_task", Text: "新文案", Order: 25, BackgroundImages: []string{},
		ContributorPlayerID: "someone_else", ContentVersion: 1, VoteVersion: 1,
	}
	if err := ps.replaceTasks([]types.PunishmentTaskConfig{legacyTask, contributedTask}); err != nil {
		t.Fatal(err)
	}
	legacySeries := types.PunishmentSeriesTaskConfig{ID: "legacy_series", Name: "旧系列", RoomBackgroundImages: []string{}, Steps: []types.PunishmentSeriesStep{{TaskIDs: []string{"legacy_task"}}}}
	if err := ps.replaceSeries([]types.PunishmentSeriesTaskConfig{legacySeries}); err != nil {
		t.Fatal(err)
	}

	if err := migrateContributionV33(db); err != nil {
		t.Fatal(err)
	}

	tasks, err := ps.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	byID := map[string]types.PunishmentTaskConfig{}
	for _, tt := range tasks {
		byID[tt.ID] = tt
	}
	legacy := byID["legacy_task"]
	if legacy.ContributorPlayerID != "QEuBIrirIxpd" || legacy.ContributorAnonymous || legacy.VoteVersion != 1 {
		t.Fatalf("legacy task should be backfilled to 8023, got %+v", legacy)
	}
	contrib := byID["contrib_task"]
	if contrib.ContributorPlayerID != "someone_else" {
		t.Fatalf("already-contributed task must not be overwritten, got %+v", contrib)
	}

	series, err := ps.listSeries()
	if err != nil {
		t.Fatal(err)
	}
	if len(series) != 1 || series[0].ContributorPlayerID != "QEuBIrirIxpd" || series[0].ContributorAnonymous || series[0].VoteVersion != 1 {
		t.Fatalf("legacy series should be backfilled to 8023, got %+v", series)
	}
}
