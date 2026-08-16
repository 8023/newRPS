package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestPickSeriesTaskProgressAndClamp(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "t_step1", Text: "第一步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50},
			{ID: "t_step2", Text: "第二步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{
			ID:   "s1",
			Name: "试炼",
			Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"t_step1"}},
				{TaskIDs: []string{"t_step2"}},
			},
		}},
	}
	pa := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "pa", Name: "小甲", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	pb := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "pb", Name: "小乙", FactionID: "female_faction"}, PlayerID: "pid2", Persistent: true}
	s.players[pa.ID] = pa
	s.players[pb.ID] = pb
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}

	// 房间内全员共享：小甲做完第 1 步，小乙接着从第 2 步开始。
	r1 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r1 == nil || r1.TaskText != "第一步 小甲" {
		t.Fatalf("pa step1: %#v", r1)
	}
	r2 := s.pickSeriesTaskForPlayer(room, pb, "s1", "胜者")
	if r2 == nil || r2.TaskText != "第二步 小乙" {
		t.Fatalf("pb should continue at step2 after pa did step1: %#v", r2)
	}
	if room.PunishmentSeriesProgress != 2 || room.PunishmentSeriesProgressID != "s1" {
		t.Fatalf("room progress = %d (%s) want 2 (s1)", room.PunishmentSeriesProgress, room.PunishmentSeriesProgressID)
	}

	// 越界 clamp 到最后一条反复执行
	r3 := s.pickSeriesTaskForPlayer(room, pb, "s1", "胜者")
	if r3 == nil || r3.TaskText != "第二步 小乙" {
		t.Fatalf("clamp last: %#v", r3)
	}

	// 同一玩家换个房间：进度从头开始，与老房间互不影响。
	room2 := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}
	r4 := s.pickSeriesTaskForPlayer(room2, pa, "s1", "胜者")
	if r4 == nil || r4.TaskText != "第一步 小甲" {
		t.Fatalf("new room should restart from step1: %#v", r4)
	}

	// 老房间进度不被新房间影响
	r5 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r5 == nil || r5.TaskText != "第二步 小甲" {
		t.Fatalf("old room keeps its progress: %#v", r5)
	}

	// 房间换成另一个系列：计数器重新从 0 开始。
	s.punishmentSeriesCache = append(s.punishmentSeriesCache, types.PunishmentSeriesTaskConfig{
		ID: "s2", Name: "另一条", Steps: []types.PunishmentSeriesStep{{TaskIDs: []string{"t_step2"}}},
	})
	room.Settings.PunishmentSeriesID = "s2"
	r6 := s.pickSeriesTaskForPlayer(room, pa, "s2", "胜者")
	if r6 == nil || r6.TaskText != "第二步 小甲" || room.PunishmentSeriesProgressID != "s2" || room.PunishmentSeriesProgress != 1 {
		t.Fatalf("switch series should reset counter: %#v progress=%d(%s)", r6, room.PunishmentSeriesProgress, room.PunishmentSeriesProgressID)
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

// TestSeriesUsableRequiresSteps：系列只要有步骤就可用——阵营覆盖不全不再拦截，
// findSeriesByID / buildPunishmentSeriesSummaries 都会放行，运行时未覆盖阵营
// 拿兜底文案（见 TestSeriesFactionFallback）。
func TestSeriesUsableRequiresSteps(t *testing.T) {
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
				{TaskIDs: []string{"male_only"}}, // 缺 female_faction，但依旧生效
			}},
			{ID: "complete_split", Name: "分开覆盖", Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"male_only", "female_only"}},
			}},
			{ID: "no_steps", Name: "没有步骤", Steps: nil},
		},
	}
	if s.findSeriesByID("incomplete") == nil {
		t.Fatal("incomplete series should still be usable")
	}
	if s.findSeriesByID("no_steps") != nil {
		t.Fatal("step-less series should be treated as not found")
	}
	summaries := s.buildPunishmentSeriesSummaries()
	if len(summaries) != 2 {
		t.Fatalf("expected 2 usable series in summary, got %d: %#v", len(summaries), summaries)
	}
	for _, sum := range summaries {
		if sum.ID == "no_steps" {
			t.Fatalf("step-less series must not appear in summaries: %#v", summaries)
		}
	}
}

// TestSeriesFactionFallback：受罚者阵营没有被当前步候选任务覆盖时，系列照常生效——
// 下发可配置的兜底文案（未配置用内置默认），且房间共享进度照常推进到下一步。
func TestSeriesFactionFallback(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		cfg: types.AppConfig{
			PunishmentRandomSettings: types.PunishmentRandomSettings{
				OrderStep: 2, MaxDifficultyOvershoot: 5, SeriesFactionFallbackText: "这一步没准备 {loser} 的任务",
			},
		},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "male_only", Text: "男向第一步", FactionIDs: []string{"male_faction"}, Order: 50},
			{ID: "female_only", Text: "女向第二步", FactionIDs: []string{"female_faction"}, Order: 50},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{
			ID:   "s1",
			Name: "试炼",
			Steps: []types.PunishmentSeriesStep{
				{TaskIDs: []string{"male_only"}}, // 女玩家无命中
				{TaskIDs: []string{"female_only"}},
			},
		}},
	}
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	s.players[player.ID] = player
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}

	r1 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r1 == nil || r1.TaskText != "这一步没准备 小败 的任务" {
		t.Fatalf("step1 fallback with configured text: %#v", r1)
	}
	if r1.TypeName != "试炼" {
		t.Fatalf("fallback should carry series name as TypeName: %#v", r1)
	}
	if room.PunishmentSeriesProgress != 1 {
		t.Fatalf("room progress after fallback step = %d want 1", room.PunishmentSeriesProgress)
	}

	r2 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r2 == nil || r2.TaskText != "女向第二步" {
		t.Fatalf("step2 should resume normal task: %#v", r2)
	}

	// 未配置兜底文案时回落到内置默认。
	s.cfg.PunishmentRandomSettings.SeriesFactionFallbackText = ""
	room2 := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}
	r3 := s.pickSeriesTaskForPlayer(room2, player, "s1", "胜者")
	if r3 == nil || r3.TaskText != defaultSeriesFactionFallbackText {
		t.Fatalf("builtin fallback text: %#v", r3)
	}
}
