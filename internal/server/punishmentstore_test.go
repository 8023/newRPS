package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestSeriesOnlyOrderMarkerExcludedFromRandomPoolButAvailableToSeries 直接构造
// punishmentTasksCache（不再经过已下线的后台任务池编辑器）验证：order=-1（"仅供系列引用，
// 不参与随机抽取"）的任务能正确被难度筛选层挡在随机池外，但依然能被系列任务按
// (SeriesID, StepIndex) 选中。
func TestSeriesOnlyOrderMarkerExcludedFromRandomPoolButAvailableToSeries(t *testing.T) {
	s := &Server{
		cfg: types.AppConfig{GenderFactions: []types.GenderFaction{{ID: "female_faction"}}},
		punishmentTasksCache: []types.PunishmentTaskConfig{{
			ID: "series_only", Text: "系列专用任务",
			TagIDs: []string{}, FactionIDs: []string{"female_faction"}, Order: -1,
			SeriesID: "s1", StepIndex: 0,
			BackgroundImage: "", BackgroundOpacity: 0.22,
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
	variants := []*types.PunishmentTaskConfig{&s.punishmentTasksCache[0]}
	if got := pickSeriesStepTask(variants, 0, "female_faction"); got == nil {
		t.Fatal("series-only task should remain available to series selection")
	}
	if got := pickSeriesStepTask(variants, 0, "male_faction"); got != nil {
		t.Fatal("faction mismatch should not match")
	}
	if got := pickSeriesStepTask(variants, 1, "female_faction"); got != nil {
		t.Fatal("step index mismatch should not match")
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
