package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestApplyPunishmentPlaceholders(t *testing.T) {
	got := applyPunishmentPlaceholders("{loser} 需要拥抱 {winner}", "小败", "大胜")
	want := "小败 需要拥抱 大胜"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 多次出现
	got = applyPunishmentPlaceholders("{loser} 对 {winner} 说：我是 {loser}", "A", "B")
	want = "A 对 B 说：我是 A"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	// 无占位符
	got = applyPunishmentPlaceholders("喝一杯水", "A", "B")
	if got != "喝一杯水" {
		t.Fatalf("got %q", got)
	}
	// 无胜者（平局双罚）时 winner 置空
	got = applyPunishmentPlaceholders("{loser} 请向 {winner} 道歉", "A", "")
	want = "A 请向  道歉"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestWinnerNameForResult(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}}
	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w1", Name: "胜者甲"}}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l1", Name: "败者乙"}}
	s.players[winner.ID] = winner
	s.players[loser.ID] = loser
	room := &RoomState{
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: winner.PublicPlayer},
			types.SeatB: &HumanSeat{Player: loser.PublicPlayer},
		},
	}
	if got := s.winnerNameForResult(room, types.ResultA); got != "胜者甲" {
		t.Fatalf("ResultA winner=%q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultB); got != "败者乙" {
		t.Fatalf("ResultB winner=%q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultDraw); got != "" {
		t.Fatalf("draw winner should be empty, got %q", got)
	}
	if got := s.winnerNameForResult(room, types.ResultDoubleLoss); got != "" {
		t.Fatalf("doubleLoss winner should be empty, got %q", got)
	}
}

func TestPickSystemTaskForPlayerPlaceholders(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}, cfg: types.AppConfig{
		GenderFactions: []types.GenderFaction{
			{ID: "male_faction", Label: "顺性别男", TaskGroup: "male"},
			{ID: "female_faction", Label: "顺性别女", TaskGroup: "female"},
		},
		PunishmentTags:           []types.PunishmentTagConfig{{ID: "hug", Name: "拥抱"}},
		PunishmentRandomSettings: types.PunishmentRandomSettings{OrderStep: 2, MaxDifficultyOvershoot: 5},
	}, punishmentTasksCache: []types.PunishmentTaskConfig{
		{ID: "t1", Text: "{loser} 需要拥抱 {winner}", TagIDs: []string{"hug"}, FactionIDs: []string{"female_faction"}, Order: 50},
	}}
	winner := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "w1", Name: "Alice", FactionID: "male_faction", FactionLabel: "顺性别男"}}
	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l1", Name: "Bob", FactionID: "female_faction", FactionLabel: "顺性别女"}}
	s.players[winner.ID] = winner
	s.players[loser.ID] = loser
	room := &RoomState{
		Settings: types.RoomSettings{EnablePunishment: true},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: winner.PublicPlayer},
			types.SeatB: &HumanSeat{Player: loser.PublicPlayer},
		},
	}
	// B 败（ResultA 表示 A 胜），Bob 是女性阵营，唯一候选任务就限定给女性阵营。
	res, advanced := s.pickSystemTaskForPlayer(room, loser, "Alice", 0)
	if res == nil || res.TaskText != "Bob 需要拥抱 Alice" {
		t.Fatalf("task=%#v", res)
	}
	if !advanced {
		t.Fatalf("expected advanced=true for a real (non-fallback) pick")
	}
}

func TestFormalPunishmentDoesNotMislabelReviewerAsPublisher(t *testing.T) {
	reviewer := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "winner", Name: "胜者", FactionID: "f1"}}
	performer := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "loser", Name: "败者", FactionID: "f1"}}
	s := &Server{
		players: map[string]*PlayerState{reviewer.ID: reviewer, performer.ID: performer},
		punishmentTasksCache: []types.PunishmentTaskConfig{
			{ID: "formal", Version: 1, Text: "正式任务", FactionIDs: []string{"f1"}, Order: 50},
		},
	}
	room := &RoomState{
		Settings: types.RoomSettings{PunishmentSource: "random"},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: reviewer.PublicPlayer},
			types.SeatB: &HumanSeat{Player: performer.PublicPlayer},
		},
	}
	tasks := s.buildPunishmentTasksWithWinnerName(room, []*PlayerState{performer}, "胜者", "round_end")
	if len(tasks) != 1 || tasks[0].TaskText != "正式任务" {
		t.Fatalf("unexpected formal punishment: %+v", tasks)
	}
	if tasks[0].AssignedBy != "" || tasks[0].AssignedByName != "" {
		t.Fatalf("formal task reviewer must not be exposed as publisher: %+v", tasks[0])
	}
}

// TestPickSystemTaskForPlayerExcludesNegativeOrder：难度为负数的任务是"仅供系列任务引用"的
// 显式标记，随机模式绝不应抽到；候选池只剩负数任务时应退化为通用兜底话术（advanced=false）。
func TestPickSystemTaskForPlayerExcludesNegativeOrder(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}, cfg: types.AppConfig{
		GenderFactions: []types.GenderFaction{
			{ID: "female_faction", Label: "顺性别女", TaskGroup: "female"},
		},
		PunishmentTags:           []types.PunishmentTagConfig{{ID: "hug", Name: "拥抱"}},
		PunishmentRandomSettings: types.PunishmentRandomSettings{OrderStep: 2, MaxDifficultyOvershoot: 5},
	}, punishmentTasksCache: []types.PunishmentTaskConfig{
		{ID: "series_only", Text: "系列专用任务", TagIDs: []string{"hug"}, FactionIDs: []string{"female_faction"}, Order: -1},
	}}
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}}
	s.players[player.ID] = player
	room := &RoomState{Settings: types.RoomSettings{EnablePunishment: true}}

	res, advanced := s.pickSystemTaskForPlayer(room, player, "胜者", 0)
	if res == nil || res.TaskText != "请完成本局惩罚。" {
		t.Fatalf("expected fallback text when only negative-order task exists, got %#v", res)
	}
	if advanced {
		t.Fatalf("fallback pick must not advance room difficulty progress")
	}

	// 混入一条正常正数难度任务后，随机模式应只可能抽到它，负数任务始终不出现。
	s.punishmentTasksCache = append(s.punishmentTasksCache, types.PunishmentTaskConfig{
		ID: "normal", Text: "普通任务", TagIDs: []string{"hug"}, FactionIDs: []string{"female_faction"}, Order: 50,
	})
	for i := 0; i < 50; i++ {
		res, advanced = s.pickSystemTaskForPlayer(room, player, "胜者", 0)
		if res == nil || res.TaskText != "普通任务" {
			t.Fatalf("random pick must never surface negative-order task, got %#v", res)
		}
		if !advanced {
			t.Fatalf("real pick should advance progress")
		}
	}
}

// TestWeightedDifficultyPickHardCap：倒伽马下绝不能抽到 难度 >= 目标+overshoot 的题（不含硬顶档）。
func TestWeightedDifficultyPickHardCap(t *testing.T) {
	// target = (count+1)*step；count=24, step=2 → target=50；硬顶 55（不含）。
	candidates := []types.PunishmentTaskConfig{
		{ID: "easy", Text: "e", Order: 10},
		{ID: "near", Text: "n", Order: 50},
		{ID: "edge", Text: "e54", Order: 54},
		{ID: "cap", Text: "c", Order: 55},
		{ID: "over", Text: "o", Order: 56},
		{ID: "hard", Text: "h", Order: 90},
	}
	seen := map[string]int{}
	for i := 0; i < 500; i++ {
		got := weightedDifficultyPick(candidates, 24, 2, 5)
		seen[got.ID]++
		if got.Order >= 55 {
			t.Fatalf("drew at/over exclusive hard cap: order=%d id=%s", got.Order, got.ID)
		}
	}
	if seen["cap"] != 0 || seen["over"] != 0 || seen["hard"] != 0 {
		t.Fatalf("hard-cap boundary and beyond must never be drawn: %v", seen)
	}
	if seen["near"] == 0 {
		t.Fatalf("near-target task should be drawn, got %v", seen)
	}
	// 更简单的题不受硬顶剔除：当池里只剩"简单 + 超顶难题"时，必中简单题。
	onlyEasyVsOver := []types.PunishmentTaskConfig{
		{ID: "easy", Text: "e", Order: 10},
		{ID: "over", Text: "o", Order: 90},
	}
	for i := 0; i < 50; i++ {
		got := weightedDifficultyPick(onlyEasyVsOver, 24, 2, 5)
		if got.ID != "easy" {
			t.Fatalf("easier task must remain eligible when harder side is capped out, got %s", got.ID)
		}
	}
	// overshoot<=0 时走默认 5，行为与显式 5 一致（>=55 仍不可中）。
	for i := 0; i < 100; i++ {
		got := weightedDifficultyPick(candidates, 24, 2, 0)
		if got.Order >= 55 {
			t.Fatalf("default overshoot must still exclude >=55, got order=%d", got.Order)
		}
	}
}

// TestWeightedDifficultyPickHardCapEarlyRoom：房间目标还很低时，绝不能抽到 >= 目标+overshoot。
func TestWeightedDifficultyPickHardCapEarlyRoom(t *testing.T) {
	// count=0, step=2 → target=2，硬顶 7（不含）。
	candidates := []types.PunishmentTaskConfig{
		{ID: "t2", Text: "2", Order: 2},
		{ID: "t5", Text: "5", Order: 5},
		{ID: "t6", Text: "6", Order: 6},
		{ID: "t7", Text: "7", Order: 7},
		{ID: "t8", Text: "8", Order: 8},
		{ID: "t50", Text: "50", Order: 50},
	}
	for i := 0; i < 400; i++ {
		got := weightedDifficultyPick(candidates, 0, 2, 5)
		if got.Order >= 7 {
			t.Fatalf("early room drew at/over exclusive hard cap: order=%d id=%s", got.Order, got.ID)
		}
	}
}

// TestPunishmentTaskProgressIsPerPlayer：系统任务难度进度按玩家在房间内各自独立计数——
// 谁挨罚就推进谁自己的计数器，房间里其他人挨罚不影响自己；与具体任务类型无关。
func TestPunishmentTaskProgressIsPerPlayer(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}, cfg: types.AppConfig{
		GenderFactions: []types.GenderFaction{
			{ID: "male_faction", Label: "顺性别男", TaskGroup: "male"},
			{ID: "female_faction", Label: "顺性别女", TaskGroup: "female"},
		},
		PunishmentTags: []types.PunishmentTagConfig{
			{ID: "type-a", Name: "A 类"},
			{ID: "type-b", Name: "B 类"},
		},
		PunishmentRandomSettings: types.PunishmentRandomSettings{OrderStep: 2, MaxDifficultyOvershoot: 5},
	}, punishmentTasksCache: []types.PunishmentTaskConfig{
		// 阵营需覆盖 p1（male_faction）与 p2（female_faction）——未勾选阵营的任务永远不参与匹配。
		{ID: "a1", Text: "A1", TagIDs: []string{"type-a"}, FactionIDs: []string{"male_faction", "female_faction"}, Order: 10},
		{ID: "a2", Text: "A2", TagIDs: []string{"type-a"}, FactionIDs: []string{"male_faction", "female_faction"}, Order: 50},
		{ID: "b1", Text: "B1", TagIDs: []string{"type-b"}, FactionIDs: []string{"male_faction", "female_faction"}, Order: 20},
		{ID: "b2", Text: "B2", TagIDs: []string{"type-b"}, FactionIDs: []string{"male_faction", "female_faction"}, Order: 80},
	}}
	p1 := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "甲", FactionID: "male_faction"}}
	p2 := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p2", Name: "乙", FactionID: "female_faction"}}
	s.players[p1.ID] = p1
	s.players[p2.ID] = p2
	room := &RoomState{
		Settings: types.RoomSettings{
			EnablePunishment: true,
			PunishmentSource: "random",
			// 包含集合为空 = 全池随机（只按拒绝集合排除）；确认进度不按标签分桶。
			PunishmentTagsIncluded: []string{},
		},
	}
	if len(room.PunishmentTaskProgress) != 0 {
		t.Fatalf("new room progress should be empty, got %#v", room.PunishmentTaskProgress)
	}
	// p1 单人受罚的一局：只推进 p1 自己的计数，p2 不受影响。
	if tasks := s.buildPunishmentTasksWithWinnerName(room, []*PlayerState{p1}, "胜", "round_end"); len(tasks) != 1 || tasks[0].TaskText == "" {
		t.Fatalf("first round tasks=%#v", tasks)
	}
	if got := room.PunishmentTaskProgress["p1"]; got != 1 {
		t.Fatalf("after 1 round p1 progress=%d want 1", got)
	}
	if got := room.PunishmentTaskProgress["p2"]; got != 0 {
		t.Fatalf("p2 progress should stay 0 while only p1 is punished, got %d", got)
	}
	// 换 p2 受罚的一局：p2 从自己的 0 起算 +1，p1 的进度不受影响。
	if tasks := s.buildPunishmentTasksWithWinnerName(room, []*PlayerState{p2}, "胜", "round_end"); len(tasks) != 1 || tasks[0].TaskText == "" {
		t.Fatalf("second round tasks=%#v", tasks)
	}
	if got := room.PunishmentTaskProgress["p2"]; got != 1 {
		t.Fatalf("after 1 round p2 progress=%d want 1 (own counter starts fresh)", got)
	}
	if got := room.PunishmentTaskProgress["p1"]; got != 1 {
		t.Fatalf("p1 progress should stay at 1 while p2 is punished, got %d", got)
	}
	// 平局双罚/双败：一局内同时惩罚两名玩家，二人各自的计数都 +1。
	if tasks := s.buildPunishmentTasksWithWinnerName(room, []*PlayerState{p1, p2}, "", "round_end"); len(tasks) != 2 || tasks[0].TaskText == "" || tasks[1].TaskText == "" {
		t.Fatalf("double-punish round tasks=%#v", tasks)
	}
	if got := room.PunishmentTaskProgress["p1"]; got != 2 {
		t.Fatalf("after double-punish round p1 progress=%d want 2", got)
	}
	if got := room.PunishmentTaskProgress["p2"]; got != 2 {
		t.Fatalf("after double-punish round p2 progress=%d want 2", got)
	}
	// 再抽几局只罚 p1，确认 p2 的进度维持不变（各自独立，不共用房间级计数）。
	for i := 0; i < 3; i++ {
		if tasks := s.buildPunishmentTasksWithWinnerName(room, []*PlayerState{p1}, "胜", "round_end"); len(tasks) != 1 {
			t.Fatalf("round %d failed", i+4)
		}
	}
	if got := room.PunishmentTaskProgress["p1"]; got != 5 {
		t.Fatalf("after 5 rounds total p1 progress=%d want 5", got)
	}
	if got := room.PunishmentTaskProgress["p2"]; got != 2 {
		t.Fatalf("p2 progress should still be 2 (untouched by p1-only rounds), got %d", got)
	}
}
