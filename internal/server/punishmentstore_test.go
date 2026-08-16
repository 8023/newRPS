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
		{ID: "t1", Name: "一", Text: "文案1", TagIDs: []string{"truth"}, FactionIDs: []string{}, Order: 10, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
		{ID: "t2", Name: "二", Text: "文案2", TagIDs: []string{"truth", "sing"}, FactionIDs: []string{"female_faction"}, Order: 80, BackgroundImages: []string{"a.png"}, BackgroundOpacity: 0.5},
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
		{ID: "only", Name: "仅", Text: "x", TagIDs: []string{"truth"}, FactionIDs: []string{}, Order: 50, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
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
}

func TestSavePunishmentTasksAllowsSeriesOnlyMarkers(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	s := &Server{
		cfg:             types.AppConfig{GenderFactions: []types.GenderFaction{{ID: "female_faction"}}},
		punishmentStore: newPunishmentStore(db),
	}
	if err := s.savePunishmentTasks([]types.PunishmentTaskConfig{{
		ID: "series_only", Name: "仅系列", Text: "系列专用任务",
		TagIDs: []string{}, FactionIDs: []string{"female_faction"}, Order: -1,
		BackgroundImages: []string{}, BackgroundOpacity: 0.22,
	}}); err != nil {
		t.Fatalf("save series-only task: %v", err)
	}
	if len(s.punishmentTasksCache) != 1 || s.punishmentTasksCache[0].Order != -1 || len(s.punishmentTasksCache[0].TagIDs) != 0 {
		t.Fatalf("saved task changed series-only markers: %#v", s.punishmentTasksCache)
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
