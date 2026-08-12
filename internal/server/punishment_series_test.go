package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestPickSeriesTaskProgressAndClamp(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newPlayerStore(db)
	if err := store.upsert(persistedPlayer{ID: "p1", PlayerID: "pid1", Name: "小败", FactionID: "female_faction"}); err != nil {
		t.Fatalf("seed player: %v", err)
	}

	s := &Server{
		playerDB: store,
		players:  map[string]*PlayerState{},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			// 阵营为空的任务永远不参与匹配，此处直接勾选受罚玩家所在阵营。
			{ID: "t_step1", Text: "第一步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50},
			{ID: "t_female", Text: "女向第二步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50},
			{ID: "t_dead", Text: "未勾选阵营，永不命中", FactionIDs: []string{}, Order: 50},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{
			ID:   "s1",
			Name: "试炼",
			Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"t_step1"}},
				{TaskIDs: []string{"t_female", "t_dead"}},
			},
		}},
	}
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	s.players[player.ID] = player
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}

	r1 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r1 == nil || r1.TaskText != "第一步 小败" {
		t.Fatalf("step1: %#v", r1)
	}
	n, _ := store.getSeriesProgress("p1", "s1")
	if n != 1 {
		t.Fatalf("progress after step1 = %d want 1", n)
	}

	r2 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r2 == nil || r2.TaskText != "女向第二步 小败" {
		t.Fatalf("step2 female variant: %#v", r2)
	}
	n, _ = store.getSeriesProgress("p1", "s1")
	if n != 2 {
		t.Fatalf("progress after step2 = %d want 2", n)
	}

	// 越界 clamp 到最后一条反复执行
	r3 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r3 == nil || r3.TaskText != "女向第二步 小败" {
		t.Fatalf("clamp last: %#v", r3)
	}

	// 换对手/跨房间：进度仍在
	room2 := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}
	r4 := s.pickSeriesTaskForPlayer(room2, player, "s1", "另一胜")
	if r4 == nil || r4.TaskText != "女向第二步 小败" {
		t.Fatalf("cross-room still last: %#v", r4)
	}

	// 重置
	if err := store.resetSeriesProgress("s1"); err != nil {
		t.Fatal(err)
	}
	n, _ = store.getSeriesProgress("p1", "s1")
	if n != 0 {
		t.Fatalf("after reset progress=%d", n)
	}
}

// TestPickSeriesStepTaskIgnoresNegativeOrder：系列抽取只看 FactionIDs 精确匹配，
// 不看 Order；难度为负数（"仅供系列引用"标记）的任务照常可被系列步骤命中。
func TestPickSeriesStepTaskIgnoresNegativeOrder(t *testing.T) {
	taskByID := map[string]*types.PunishmentTaskConfig{
		"series_only": {ID: "series_only", Text: "仅系列", FactionIDs: []string{"female_faction"}, Order: -1},
	}
	tsk := pickSeriesStepTask([]string{"series_only"}, "female_faction", taskByID)
	if tsk == nil || tsk.Text != "仅系列" {
		t.Fatalf("negative-order task should still be pickable via series step, got %#v", tsk)
	}
}

func TestPickSeriesStepTaskExactFactionOnly(t *testing.T) {
	taskByID := map[string]*types.PunishmentTaskConfig{
		"male":   {ID: "male", Text: "男", FactionIDs: []string{"male_faction"}},
		"dead":   {ID: "dead", Text: "未勾选阵营", FactionIDs: []string{}},
		"female": {ID: "female", Text: "女", FactionIDs: []string{"female_faction"}},
	}
	// 精确命中阵营
	if tsk := pickSeriesStepTask([]string{"male", "dead", "female"}, "female_faction", taskByID); tsk == nil || tsk.Text != "女" {
		t.Fatalf("exact faction match: %#v", tsk)
	}
	// 未勾选任何阵营的任务永不参与匹配，不再作为通用兜底
	if tsk := pickSeriesStepTask([]string{"male", "dead", "female"}, "unknown_faction", taskByID); tsk != nil {
		t.Fatalf("no exact match should return nil (no generic fallback), got %#v", tsk)
	}
	if tsk := pickSeriesStepTask([]string{"dead"}, "female_faction", taskByID); tsk != nil {
		t.Fatalf("faction-less task should never match, got %#v", tsk)
	}
	// 悬空引用跳过
	if tsk := pickSeriesStepTask([]string{"missing", "also_gone"}, "female_faction", taskByID); tsk != nil {
		t.Fatalf("all dangling should return nil: %#v", tsk)
	}
	if tsk := pickSeriesStepTask([]string{"missing", "female"}, "female_faction", taskByID); tsk == nil || tsk.Text != "女" {
		t.Fatalf("dangling then exact match: %#v", tsk)
	}
}

// TestPickSeriesStepTaskRandomAmongMatches：同一阵营有多个候选任务时随机抽取，
// 而不是固定取列表顺序里的第一个。
func TestPickSeriesStepTaskRandomAmongMatches(t *testing.T) {
	taskByID := map[string]*types.PunishmentTaskConfig{
		"f1": {ID: "f1", Text: "女A", FactionIDs: []string{"female_faction"}},
		"f2": {ID: "f2", Text: "女B", FactionIDs: []string{"female_faction"}},
	}
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		tsk := pickSeriesStepTask([]string{"f1", "f2"}, "female_faction", taskByID)
		if tsk == nil || (tsk.Text != "女A" && tsk.Text != "女B") {
			t.Fatalf("unexpected pick: %#v", tsk)
		}
		seen[tsk.Text] = true
	}
	if len(seen) != 2 {
		t.Fatalf("expected both candidates to appear over repeated picks, got %v", seen)
	}
}

// TestSeriesIsUsableRequiresFullFactionCoverage：系列每一步的候选任务合起来必须覆盖
// 全部已定义阵营，否则整个系列判定为不可用——findSeriesByID 返回 nil，
// buildPunishmentSeriesSummaries 也不会把它列进建房面板目录。
func TestSeriesIsUsableRequiresFullFactionCoverage(t *testing.T) {
	s := &Server{
		cfg: types.AppConfig{
			GenderFactions: []types.GenderFaction{
				{ID: "male_faction", Label: "顺性别男"},
				{ID: "female_faction", Label: "顺性别女"},
			},
		},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "male_only", Text: "男", FactionIDs: []string{"male_faction"}},
			{ID: "female_only", Text: "女", FactionIDs: []string{"female_faction"}},
			{ID: "both", Text: "通用", FactionIDs: []string{"male_faction", "female_faction"}},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{
			{ID: "incomplete", Name: "覆盖不全", Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"male_only"}}, // 缺 female_faction
			}},
			{ID: "complete_split", Name: "分开覆盖", Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"male_only", "female_only"}},
			}},
			{ID: "complete_single", Name: "单任务通用", Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"both"}},
			}},
		},
	}
	if s.findSeriesByID("incomplete") != nil {
		t.Fatal("incomplete series should be treated as not found")
	}
	if s.findSeriesByID("complete_split") == nil {
		t.Fatal("complete_split series should be usable")
	}
	if s.findSeriesByID("complete_single") == nil {
		t.Fatal("complete_single series should be usable")
	}
	summaries := s.buildPunishmentSeriesSummaries()
	if len(summaries) != 2 {
		t.Fatalf("expected 2 usable series in summary, got %d: %#v", len(summaries), summaries)
	}
	for _, sum := range summaries {
		if sum.ID == "incomplete" {
			t.Fatalf("incomplete series must not appear in summaries: %#v", summaries)
		}
	}
}
