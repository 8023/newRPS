package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestMigrateSeriesVariantsToTasks 验证系列变体打散后的任务池 + steps.taskIds
// 引用能被 pickSeriesTaskForPlayer 按阵营正确选取；表非空时 migrate 幂等。
func TestMigrateSeriesVariantsToTasks(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)

	// 模拟 migrate 写入结果：原 tasks + 系列变体打散任务 + steps 引用
	tasks := []types.PunishmentTaskConfig{
		{ID: "t1", Text: "原任务", TagIDs: []string{"truth"}, Order: 50, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
		{ID: "s1_s0_v0", Text: "女变体", TagIDs: []string{"series:s1"}, FactionIDs: []string{"female_faction"}, Order: 50, BackgroundImages: []string{}, BackgroundOpacity: 0.3},
		{ID: "s1_s0_v1", Text: "通用变体", TagIDs: []string{"series:s1"}, FactionIDs: []string{}, Order: 50, BackgroundImages: []string{}, BackgroundOpacity: 0.22},
	}
	series := []types.PunishmentSeriesTaskConfig{{
		ID: "s1", Name: "试炼",
		RoomNamePool: &types.RoomNamePool{Subjects: []string{"b"}, RoomWords: []string{"c"}},
		Steps:        []types.PunishmentSeriesStep{{TaskIDs: []string{"s1_s0_v0", "s1_s0_v1"}}},
	}}
	if err := ps.replaceTasks(tasks); err != nil {
		t.Fatal(err)
	}
	if err := ps.replaceSeries(series); err != nil {
		t.Fatal(err)
	}

	s := &Server{punishmentStore: ps}
	s.reloadPunishmentCaches()

	// 迁移产物保留 series:<id> 内部标签和普通难度 50；只要阵营精确命中，
	// 它也应进入普通随机池，而不是被隐式当成系列专用任务。
	randomPool := candidateTasksForFaction(s.punishmentTasksCache, "female_faction")
	randomPool = candidateTasksForTags(randomPool, nil, nil)
	randomPool = candidateTasksForRandomDifficulty(randomPool)
	foundMigratedVariant := false
	for _, task := range randomPool {
		if task.ID == "s1_s0_v0" {
			foundMigratedVariant = true
			break
		}
	}
	if !foundMigratedVariant {
		t.Fatalf("migrated series variant should remain eligible for random mode: %#v", randomPool)
	}

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	s.players = map[string]*PlayerState{player.ID: player}
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}
	r := s.pickSeriesTaskForPlayer(room, player, "s1", "胜")
	if r == nil || r.TaskText != "女变体" {
		t.Fatalf("pick after migrate: %#v", r)
	}

	// 表非空 → migrate 幂等（不覆盖）
	s.migratePunishmentPoolFromJSONIfNeeded()
	got, err := ps.listTasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 3 {
		t.Fatalf("idempotent count=%d want 3", len(got))
	}
}

func TestStampLegacyPunishmentAttribution_when_importRunsAfterSchemaMigrations_then_fillsMetadata(t *testing.T) {
	tasks := []types.PunishmentTaskConfig{{ID: "legacy_task"}}
	series := []types.PunishmentSeriesTaskConfig{{ID: "legacy_series"}}
	stampLegacyPunishmentAttribution(tasks, series)
	if tasks[0].ContributorPlayerID != legacyPunishmentContributorID || tasks[0].VoteVersion != 1 || tasks[0].ContentVersion != 1 || tasks[0].TaskGroupID != tasks[0].ID {
		t.Fatalf("legacy task metadata incomplete: %+v", tasks[0])
	}
	if series[0].ContributorPlayerID != legacyPunishmentContributorID || series[0].VoteVersion != 1 || series[0].ContentVersion != 1 {
		t.Fatalf("legacy series metadata incomplete: %+v", series[0])
	}
}
