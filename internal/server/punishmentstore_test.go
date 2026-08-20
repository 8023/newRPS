package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestPunishmentStoreReplaceAndList(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)

	tasks := []types.PunishmentTaskConfig{
		{ID: "t1", Text: "文案1", TagIDs: []string{"truth"}, FactionIDs: []string{}, Order: 10, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
		{ID: "t2", Text: "文案2", TagIDs: []string{"truth", "sing"}, FactionIDs: []string{"female_faction"}, Order: 80, BackgroundImages: []string{"a.png"}, BackgroundOpacity: 0.5},
	}
	if err := ps.replaceTasks(tasks); err != nil {
		t.Fatalf("replaceTasks: %v", err)
	}
	got, err := ps.listTasks()
	if err != nil {
		t.Fatalf("listTasks: %v", err)
	}
	if len(got) != 2 || got[0].ID != "t1" || got[1].ID != "t2" {
		t.Fatalf("listTasks=%+v", got)
	}
	if got[1].Order != 80 || len(got[1].TagIDs) != 2 || got[1].BackgroundImages[0] != "a.png" {
		t.Fatalf("task2 fields: %+v", got[1])
	}

	// 全量覆盖
	if err := ps.replaceTasks([]types.PunishmentTaskConfig{
		{ID: "only", Text: "x", TagIDs: []string{"truth"}, FactionIDs: []string{}, Order: 50, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
	}); err != nil {
		t.Fatal(err)
	}
	got, _ = ps.listTasks()
	if len(got) != 1 || got[0].ID != "only" {
		t.Fatalf("after replace list=%+v", got)
	}

	series := []types.PunishmentSeriesTaskConfig{{
		ID: "s1", Name: "系列一",
		RoomNamePool:         &types.RoomNamePool{Adjectives: []string{"a"}, Subjects: []string{"b"}, RoomWords: []string{"c"}},
		RoomBackgroundImages: []string{"bg.png"},
		TargetFactionIDs:     []string{"female_faction"},
		Steps: []types.PunishmentSeriesStep{
			{TaskIDs: []string{"only", "missing"}},
			{TaskIDs: []string{"only"}},
		},
	}}
	if err := ps.replaceSeries(series); err != nil {
		t.Fatalf("replaceSeries: %v", err)
	}
	gotS, err := ps.listSeries()
	if err != nil {
		t.Fatalf("listSeries: %v", err)
	}
	if len(gotS) != 1 || gotS[0].ID != "s1" || len(gotS[0].Steps) != 2 {
		t.Fatalf("listSeries=%+v", gotS)
	}
	if len(gotS[0].Steps[0].TaskIDs) != 2 || gotS[0].Steps[0].TaskIDs[1] != "missing" {
		t.Fatalf("step0 taskIds=%v", gotS[0].Steps[0].TaskIDs)
	}
	if gotS[0].RoomNamePool == nil || gotS[0].RoomNamePool.Subjects[0] != "b" {
		t.Fatalf("roomNamePool=%+v", gotS[0].RoomNamePool)
	}
	if len(gotS[0].TargetFactionIDs) != 1 || gotS[0].TargetFactionIDs[0] != "female_faction" {
		t.Fatalf("targetFactionIds=%v", gotS[0].TargetFactionIDs)
	}
}

// TestSeriesOnlyOrderMarkerExcludedFromRandomPoolButAvailableToSeries 直接构造
// punishmentTasksCache（不再经过已下线的后台任务池编辑器 savePunishmentTasks）验证：
// order=-1（"仅供系列引用，不参与随机抽取"）的任务能正确被难度筛选层挡在随机池外，
// 但依然能被系列任务按 ID 选中。
func TestSeriesOnlyOrderMarkerExcludedFromRandomPoolButAvailableToSeries(t *testing.T) {
	s := &Server{
		cfg: types.AppConfig{GenderFactions: []types.GenderFaction{{ID: "female_faction"}}},
		punishmentTasksCache: []types.PunishmentTaskConfig{{
			ID: "series_only", Text: "系列专用任务",
			TagIDs: []string{}, FactionIDs: []string{"female_faction"}, Order: -1,
			BackgroundImages: []string{}, BackgroundOpacity: 0.22,
		}},
	}
	// 无标签任务现在会通过标签筛选（无视 S/R），把 order=-1 的系列专用任务挡在
	// 随机池外的是难度过滤这一层。
	pool := candidateTasksForTags(s.punishmentTasksCache, nil, nil)
	if len(pool) != 1 {
		t.Fatalf("tagless task should pass tag filter: %#v", pool)
	}
	if got := candidateTasksForRandomDifficulty(pool); len(got) != 0 {
		t.Fatalf("series-only task entered random pool: %#v", got)
	}
	taskByID := map[string]*types.PunishmentTaskConfig{"series_only": &s.punishmentTasksCache[0]}
	if got := pickSeriesStepTask([]string{"series_only"}, "female_faction", taskByID); got == nil {
		t.Fatal("series-only task should remain available to series selection")
	}
}

// TestSeriesRunStatsRecordAndRead 覆盖系列完成率样本的存储层：分桶按 series_id+series_version，
// 每次记录各自独立累加参与数与百分比之和，未命中的桶查询返回 0 而不是报错。
func TestSeriesRunStatsRecordAndRead(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)

	if participants, percentSum, err := ps.seriesRunStats("s1", 1); err != nil || participants != 0 || percentSum != 0 {
		t.Fatalf("unseen bucket should read as 0/0, got %d/%d err=%v", participants, percentSum, err)
	}

	for _, percent := range []int{100, 75, 50} {
		if err := ps.recordSeriesRunProgress("s1", 1, percent); err != nil {
			t.Fatalf("recordSeriesRunProgress: %v", err)
		}
	}
	participants, percentSum, err := ps.seriesRunStats("s1", 1)
	if err != nil || participants != 3 || percentSum != 225 {
		t.Fatalf("s1/v1 = %d/%d err=%v, want 3/225", participants, percentSum, err)
	}

	// 改版（version 变化）另起一桶，不与旧版本混算。
	if err := ps.recordSeriesRunProgress("s1", 2, 60); err != nil {
		t.Fatalf("recordSeriesRunProgress v2: %v", err)
	}
	if participants, percentSum, err := ps.seriesRunStats("s1", 2); err != nil || participants != 1 || percentSum != 60 {
		t.Fatalf("s1/v2 = %d/%d err=%v, want 1/60", participants, percentSum, err)
	}
	if participants, percentSum, err := ps.seriesRunStats("s1", 1); err != nil || participants != 3 || percentSum != 225 {
		t.Fatalf("s1/v1 should be unaffected by v2, got %d/%d err=%v", participants, percentSum, err)
	}
}
