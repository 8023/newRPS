package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestPickSeriesTaskProgressAndClamp 覆盖「每个玩家在本房间内各自独立计数」——房间内其他
// 玩家挨罚不影响自己的步数，换房间（哪怕系列相同）从 0 重来，房内切系列对该玩家也重来。
func TestPickSeriesTaskProgressAndClamp(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "t_step1", Text: "第一步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50, SeriesID: "s1", StepIndex: 0},
			{ID: "t_step2", Text: "第二步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50, SeriesID: "s1", StepIndex: 1},
			{ID: "t_step3", Text: "第三步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50, SeriesID: "s1", StepIndex: 2},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{ID: "s1", Name: "试炼"}},
	}
	s.rebuildPunishmentCacheIndexes()
	pa := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "pa", Name: "小甲", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	pb := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "pb", Name: "小乙", FactionID: "female_faction"}, PlayerID: "pid2", Persistent: true}
	s.players[pa.ID] = pa
	s.players[pb.ID] = pb
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}

	// a 自己输两次，走自己的第 1、2 步。
	r1 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r1 == nil || r1.TaskText != "第一步 小甲" {
		t.Fatalf("pa step1: %#v", r1)
	}
	r2 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r2 == nil || r2.TaskText != "第二步 小甲" {
		t.Fatalf("pa step2: %#v", r2)
	}

	// 期间 b 输掉一次：b 是自己第一次抽这个系列，从第 1 步开始，不受 a 进度影响。
	r3 := s.pickSeriesTaskForPlayer(room, pb, "s1", "胜者")
	if r3 == nil || r3.TaskText != "第一步 小乙" {
		t.Fatalf("pb should start fresh at her own step1 regardless of pa: %#v", r3)
	}

	// a 再输一次：是 a 自己第 3 次轮到，走第 3 步——跟房间里发生过多少次别人的事件无关。
	r4 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r4 == nil || r4.TaskText != "第三步 小甲" {
		t.Fatalf("pa's own 3rd turn should be step3: %#v", r4)
	}

	// 越界 clamp 到最后一条反复执行（a 自己的进度）。
	r5 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r5 == nil || r5.TaskText != "第三步 小甲" {
		t.Fatalf("clamp last: %#v", r5)
	}

	// 同一玩家换个房间（哪怕用的是同一个系列）：进度从头开始，与老房间互不影响。
	room2 := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}
	r6 := s.pickSeriesTaskForPlayer(room2, pa, "s1", "胜者")
	if r6 == nil || r6.TaskText != "第一步 小甲" {
		t.Fatalf("new room should restart from step1: %#v", r6)
	}

	// 老房间进度不受新房间影响：a 在老房间仍卡在（clamp 到）第三步——等价于「退房再进同一
	// 房间/去观众席再回来」进度保持，因为进度就挂在同一个 *RoomState 上，与座位状态无关。
	r7 := s.pickSeriesTaskForPlayer(room, pa, "s1", "胜者")
	if r7 == nil || r7.TaskText != "第三步 小甲" {
		t.Fatalf("old room keeps its own progress: %#v", r7)
	}

	// b 也不受 a 影响，继续她自己的第 2 步。
	r8 := s.pickSeriesTaskForPlayer(room, pb, "s1", "胜者")
	if r8 == nil || r8.TaskText != "第二步 小乙" {
		t.Fatalf("pb continues at her own step2: %#v", r8)
	}

	// 房间换成另一个系列：a 对新系列的计数器重新从 0 开始。
	s.punishmentTasksCache = append(s.punishmentTasksCache, types.PunishmentTaskConfig{
		ID: "t2_step2", Text: "第二步 {loser}", FactionIDs: []string{"female_faction"}, Order: 50, SeriesID: "s2", StepIndex: 0,
	})
	s.punishmentSeriesCache = append(s.punishmentSeriesCache, types.PunishmentSeriesTaskConfig{ID: "s2", Name: "另一条"})
	s.rebuildPunishmentCacheIndexes()
	room.Settings.PunishmentSeriesID = "s2"
	r9 := s.pickSeriesTaskForPlayer(room, pa, "s2", "胜者")
	if r9 == nil || r9.TaskText != "第二步 小甲" {
		t.Fatalf("switch series should reset pa's counter: %#v", r9)
	}
	prog := room.PunishmentSeriesPlayerProgress[pa.ID]
	if prog == nil || prog.SeriesID != "s2" || prog.Step != 1 {
		t.Fatalf("pa's progress on s2 should be step=1, got %#v", prog)
	}
}

// TestPickSeriesStepTaskIgnoresNegativeOrder：系列抽取只看 FactionIDs 精确匹配 + StepIndex，
// 不看 Order；难度为负数（"仅供系列引用"标记）的任务照常可被系列步骤命中。
func TestPickSeriesStepTaskIgnoresNegativeOrder(t *testing.T) {
	variants := []*types.PunishmentTaskConfig{
		{ID: "series_only", Text: "仅系列", FactionIDs: []string{"female_faction"}, Order: -1, StepIndex: 0},
	}
	tsk := pickSeriesStepTask(variants, 0, "female_faction")
	if tsk == nil || tsk.Text != "仅系列" {
		t.Fatalf("negative-order task should still be pickable via series step, got %#v", tsk)
	}
}

func TestPickSeriesStepTaskExactFactionOnly(t *testing.T) {
	variants := []*types.PunishmentTaskConfig{
		{ID: "male", Text: "男", FactionIDs: []string{"male_faction"}, StepIndex: 0},
		{ID: "dead", Text: "未勾选阵营", FactionIDs: []string{}, StepIndex: 0},
		{ID: "female", Text: "女", FactionIDs: []string{"female_faction"}, StepIndex: 0},
	}
	// 精确命中阵营
	if tsk := pickSeriesStepTask(variants, 0, "female_faction"); tsk == nil || tsk.Text != "女" {
		t.Fatalf("exact faction match: %#v", tsk)
	}
	// 未勾选任何阵营的任务永不参与匹配，不再作为通用兜底
	if tsk := pickSeriesStepTask(variants, 0, "unknown_faction"); tsk != nil {
		t.Fatalf("no exact match should return nil (no generic fallback), got %#v", tsk)
	}
	deadOnly := []*types.PunishmentTaskConfig{{ID: "dead", Text: "未勾选阵营", FactionIDs: []string{}, StepIndex: 0}}
	if tsk := pickSeriesStepTask(deadOnly, 0, "female_faction"); tsk != nil {
		t.Fatalf("faction-less task should never match, got %#v", tsk)
	}
	// 步骤不匹配：同样的候选，查询别的 StepIndex 应该返回 nil。
	if tsk := pickSeriesStepTask(variants, 1, "female_faction"); tsk != nil {
		t.Fatalf("step index mismatch should return nil: %#v", tsk)
	}
}

// TestPickSeriesStepTaskRandomAmongMatches：同一阵营有多个候选任务时随机抽取，
// 而不是固定取列表顺序里的第一个。
func TestPickSeriesStepTaskRandomAmongMatches(t *testing.T) {
	variants := []*types.PunishmentTaskConfig{
		{ID: "f1", Text: "女A", FactionIDs: []string{"female_faction"}, StepIndex: 0},
		{ID: "f2", Text: "女B", FactionIDs: []string{"female_faction"}, StepIndex: 0},
	}
	seen := map[string]bool{}
	for i := 0; i < 200; i++ {
		tsk := pickSeriesStepTask(variants, 0, "female_faction")
		if tsk == nil || (tsk.Text != "女A" && tsk.Text != "女B") {
			t.Fatalf("unexpected pick: %#v", tsk)
		}
		seen[tsk.Text] = true
	}
	if len(seen) != 2 {
		t.Fatalf("expected both candidates to appear over repeated picks, got %v", seen)
	}
}

// TestSeriesUsableRequiresSteps 覆盖运行时对历史异常数据的防御：审批后的新系列会保证
// 声明的目标阵营逐步覆盖；但缓存里若出现没有目标声明的旧系列，只要仍有步骤就可运行，
// 未命中玩家阵营时由随机池替补（见 TestSeriesFactionReplacement）。
func TestSeriesUsableRequiresSteps(t *testing.T) {
	s := &Server{
		cfg: types.AppConfig{
			GenderFactions: []types.GenderFaction{
				{ID: "male_faction", Label: "顺性别男"},
				{ID: "female_faction", Label: "顺性别女"},
			},
		},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "male_only", Text: "男", FactionIDs: []string{"male_faction"}, SeriesID: "incomplete", StepIndex: 0},
			{ID: "male_split", Text: "男", FactionIDs: []string{"male_faction"}, SeriesID: "complete_split", StepIndex: 0},
			{ID: "female_split", Text: "女", FactionIDs: []string{"female_faction"}, SeriesID: "complete_split", StepIndex: 0},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{
			{ID: "incomplete", Name: "覆盖不全"}, // 缺 female_faction，但依旧生效
			{ID: "complete_split", Name: "分开覆盖"},
			{ID: "no_steps", Name: "没有步骤"}, // 没有任何 sub_tasks 引用它
		},
	}
	s.rebuildPunishmentCacheIndexes()
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

// TestSeriesFactionReplacement：受罚者阵营没有被当前步候选任务覆盖时，从随机池抽替补，
// 进度照常推进；票记到替补任务自己的 ID，不记到系列头上。
func TestSeriesFactionReplacement(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		cfg: types.AppConfig{
			PunishmentRandomSettings: types.PunishmentRandomSettings{OrderStep: 2, MaxDifficultyOvershoot: 5},
			PunishmentTags:           []types.PunishmentTagConfig{{ID: "truth", Name: "真心话"}},
		},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "male_only", Text: "男向第一步", FactionIDs: []string{"male_faction"}, Order: 50, TagIDs: []string{"truth"}, SeriesID: "s1", StepIndex: 0},
			{ID: "female_only", Text: "女向第二步", FactionIDs: []string{"female_faction"}, Order: 50, SeriesID: "s1", StepIndex: 1},
			{ID: "random_f", Text: "随机替补 {loser}", FactionIDs: []string{"female_faction"}, Order: 40, ContributorPlayerID: "c1"},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{
			ID: "s1", Name: "试炼", ContributorPlayerID: "series_author", Version: 1,
		}},
	}
	s.rebuildPunishmentCacheIndexes()
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	s.players[player.ID] = player
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: "s1"}}

	r1 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r1 == nil || r1.TaskText != "随机替补 小败" {
		t.Fatalf("step1 should pick random replacement: %#v", r1)
	}
	if r1.EventMeta.FormalTaskID != "random_f" {
		t.Fatalf("replacement vote should use its own task id: %#v", r1.EventMeta)
	}
	prog := room.PunishmentSeriesPlayerProgress[player.ID]
	if prog == nil || prog.Step != 1 {
		t.Fatalf("player progress after replacement = %#v want step=1", prog)
	}

	r2 := s.pickSeriesTaskForPlayer(room, player, "s1", "胜者")
	if r2 == nil || r2.TaskText != "女向第二步" {
		t.Fatalf("step2 should resume series task: %#v", r2)
	}
}

func TestSeriesTagTriState(t *testing.T) {
	s := &Server{
		cfg: types.AppConfig{
			PunishmentTags: []types.PunishmentTagConfig{
				{ID: "truth", Name: "真心话"},
				{ID: "dare", Name: "冒险"},
				{ID: "sing", Name: "唱歌"},
			},
		},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "a", TagIDs: []string{"truth", "dare"}, SeriesID: "s1", StepIndex: 0},
			{ID: "b", TagIDs: []string{"truth"}, SeriesID: "s1", StepIndex: 0},
			{ID: "c", TagIDs: []string{"dare"}, SeriesID: "s1", StepIndex: 1},
		},
		punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{ID: "s1", Name: "试炼"}},
	}
	s.rebuildPunishmentCacheIndexes()
	series := s.punishmentSeriesByID["s1"]
	incl, excl := s.seriesTagTriState(*series)
	if len(incl) != 1 || (incl[0] != "dare" && incl[0] != "truth") {
		t.Fatalf("included=%v", incl)
	}
	// truth 出现 2 次，dare 出现 2 次，并列按 ID 字典序取第一个：dare < truth，所以 dare。
	if incl[0] != "dare" {
		t.Fatalf("tie should pick lexicographically first, got %v", incl)
	}
	if len(excl) != 1 || excl[0] != "sing" {
		t.Fatalf("excluded=%v", excl)
	}
}

// TestRecordSeriesRunProgressOnCloseAveragesPercent 覆盖「完成率」的房间销毁结算：只统计
// 至少走完 1 步的玩家，样本值是各自「已走步数/系列总步数」的百分比，最终落库为这些样本的
// 算术平均，而不是「是否走到最后一步」的二元判定。
func TestRecordSeriesRunProgressOnCloseAveragesPercent(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)
	s := &Server{punishmentStore: ps, punishmentTasksCache: []types.PunishmentTaskConfig{
		{ID: "s1_0", SeriesID: "s1", StepIndex: 0},
		{ID: "s1_1", SeriesID: "s1", StepIndex: 1},
		{ID: "s1_2", SeriesID: "s1", StepIndex: 2},
		{ID: "s1_3", SeriesID: "s1", StepIndex: 3},
	}, punishmentSeriesCache: []types.PunishmentSeriesTaskConfig{{ID: "s1", Version: 1}}}
	s.rebuildPunishmentCacheIndexes()

	room := &RoomState{PunishmentSeriesPlayerProgress: map[string]*seriesPlayerProgress{
		"pa": {SeriesID: "s1", Step: 3}, // 3/4 = 75%
		"pb": {SeriesID: "s1", Step: 1}, // 1/4 = 25%
		"pc": {SeriesID: "s1", Step: 0}, // 一步没走，不计入样本
	}}
	s.recordSeriesRunProgressOnClose(room)

	participants, percentSum, err := ps.seriesRunStats("s1", 1)
	if err != nil {
		t.Fatalf("seriesRunStats: %v", err)
	}
	if participants != 2 || percentSum != 100 {
		t.Fatalf("got participants=%d percentSum=%d, want 2/100 (75+25)", participants, percentSum)
	}
	if avg := float64(percentSum) / float64(participants); avg != 50 {
		t.Fatalf("average completion rate = %v, want 50", avg)
	}
}
