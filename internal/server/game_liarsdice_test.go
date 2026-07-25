package server

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func newLiarsDiceTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	return &Server{
		players:              map[string]*PlayerState{},
		rooms:                map[string]*RoomState{},
		clients:              map[string]*Client{},
		socketToClient:       map[string]*Client{},
		liarsDiceStartTimers: map[string]*time.Timer{},
		dataDir:              dir,
		playersFile:          filepath.Join(dir, "players.json"),
	}
}

func newLiarsDiceTestPlayer(id, name string) *PlayerState {
	return &PlayerState{
		PublicPlayer: types.PublicPlayer{ID: id, Name: name, DisplayName: name},
		PlayerID:     id,
	}
}

func newLiarsDiceTestRoom(id string, minP, maxP int) *RoomState {
	return &RoomState{
		ID: id,
		Settings: types.RoomSettings{GameID: types.GameLiarsDice, Stake: 5, LiarsDiceMinPlayers: minP, LiarsDiceMaxPlayers: maxP},
		Phase:    types.PhaseReady, Status: "waiting",
		Seats:              map[types.SeatKey]SeatOccupant{types.SeatA: nil, types.SeatB: nil},
		SpectatorIDs:       []string{},
		Ready:              map[types.SeatKey]bool{},
		Choices:            map[types.SeatKey]types.Move{},
		Score:              map[types.SeatKey]int{},
		SeatedScore:        map[types.SeatKey]int{},
		SeatStats:          map[types.SeatKey]types.SeatStats{},
		LockedSeatIDs:      map[string]struct{}{},
		DisconnectForfeits: map[string]DisconnectForfeit{},
		LiarsDice: &types.LiarsDiceState{
			ParticipantIDs: []string{}, ReadyPlayerIDs: []string{}, DiceCounts: map[string]int{},
			BidHistory: []types.LiarsDiceBid{}, MinPlayers: minP, MaxPlayers: maxP,
		},
		LiarsDiceHands:              map[string][]int{},
		LiarsDiceDisconnectForfeits: map[string]LiarsDiceDisconnectForfeit{},
	}
}

func TestLiarsDiceValidRaise(t *testing.T) {
	cases := []struct {
		name        string
		prev        *types.LiarsDiceBid
		count, face int
		want        bool
	}{
		{"first bid any face", nil, 4, 3, true},
		{"bigger count same face", &types.LiarsDiceBid{Count: 4, Face: 3}, 5, 3, true},
		{"same count bigger face", &types.LiarsDiceBid{Count: 4, Face: 3}, 4, 4, true},
		{"same count same face rejected", &types.LiarsDiceBid{Count: 4, Face: 3}, 4, 3, false},
		{"same count smaller face rejected", &types.LiarsDiceBid{Count: 4, Face: 3}, 4, 2, false},
		{"smaller count rejected even with bigger face", &types.LiarsDiceBid{Count: 4, Face: 3}, 3, 6, false},
		{"face out of range rejected", &types.LiarsDiceBid{Count: 4, Face: 3}, 5, 7, false},
		{"bigger count smaller face allowed", &types.LiarsDiceBid{Count: 4, Face: 5}, 5, 1, true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := liarsDiceValidRaise(c.prev, c.count, c.face); got != c.want {
				t.Fatalf("liarsDiceValidRaise(%v,%d,%d) = %v, want %v", c.prev, c.count, c.face, got, c.want)
			}
		})
	}
}

func TestLiarsDicePredecessor(t *testing.T) {
	order := []string{"a", "b", "c"}
	if got := liarsDicePredecessor(order, "a"); got != "c" {
		t.Fatalf("predecessor(a) = %q, want c", got)
	}
	if got := liarsDicePredecessor(order, "b"); got != "a" {
		t.Fatalf("predecessor(b) = %q, want a", got)
	}
	if got := liarsDicePredecessor(order, "c"); got != "b" {
		t.Fatalf("predecessor(c) = %q, want b", got)
	}
	if got := liarsDicePredecessor([]string{"solo"}, "solo"); got != "" {
		t.Fatalf("predecessor with 1 participant should be empty, got %q", got)
	}
	if got := liarsDicePredecessor(order, "missing"); got != "" {
		t.Fatalf("predecessor of absent player should be empty, got %q", got)
	}
}

func TestLiarsDiceNextTurn(t *testing.T) {
	order := []string{"a", "b", "c"}
	if got := liarsDiceNextTurn(order, "a"); got != "b" {
		t.Fatalf("next(a) = %q, want b", got)
	}
	if got := liarsDiceNextTurn(order, "c"); got != "a" {
		t.Fatalf("next(c) should wrap to a, got %q", got)
	}
}

func TestStartLiarsDiceRoundRollsFreshHandsAndPicksTurn(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	a, b, c := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob"), newLiarsDiceTestPlayer("c", "Carol")
	s.players[a.ID], s.players[b.ID], s.players[c.ID] = a, b, c
	room.LiarsDice.ParticipantIDs = []string{"a", "b", "c"}
	s.rooms[room.ID] = room

	s.startLiarsDiceRound(room)

	if room.Phase != types.PhaseChoosing {
		t.Fatalf("phase = %v, want choosing", room.Phase)
	}
	for _, id := range room.LiarsDice.ParticipantIDs {
		dice := room.LiarsDiceHands[id]
		if len(dice) != liarsDiceDicePerPlayer {
			t.Fatalf("player %s got %d dice, want %d", id, len(dice), liarsDiceDicePerPlayer)
		}
		for _, d := range dice {
			if d < 1 || d > 6 {
				t.Fatalf("die value %d out of range", d)
			}
		}
		if room.LiarsDice.DiceCounts[id] != liarsDiceDicePerPlayer {
			t.Fatalf("diceCounts[%s] = %d, want %d", id, room.LiarsDice.DiceCounts[id], liarsDiceDicePerPlayer)
		}
	}
	if indexOfString(room.LiarsDice.ParticipantIDs, room.LiarsDice.CurrentTurn) < 0 {
		t.Fatalf("currentTurn %q not among participants", room.LiarsDice.CurrentTurn)
	}
	if room.LiarsDice.CurrentBid != nil {
		t.Fatalf("fresh round should start with no current bid")
	}
}

func TestShuffleStringsIsPermutation(t *testing.T) {
	in := []string{"a", "b", "c", "d"}
	out := shuffleStrings(in)
	if len(out) != len(in) {
		t.Fatalf("len = %d, want %d", len(out), len(in))
	}
	// 入参不被修改
	if stringsJoin(in) != "a,b,c,d" {
		t.Fatalf("input mutated: %v", in)
	}
	seen := map[string]int{}
	for _, id := range out {
		seen[id]++
	}
	for _, id := range in {
		if seen[id] != 1 {
			t.Fatalf("shuffle result %v is not a permutation of %v", out, in)
		}
	}
}

func TestStartLiarsDiceRoundRandomizesOrder(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	// 足够多的玩家与重复次数，几乎必会观察到至少一次与初始顺序不同的排列
	ids := []string{"a", "b", "c", "d", "e"}
	for _, id := range ids {
		s.players[id] = newLiarsDiceTestPlayer(id, id)
	}
	original := append([]string{}, ids...)
	sawDifferent := false
	for i := 0; i < 40; i++ {
		room := newLiarsDiceTestRoom("r1", 2, 8)
		room.LiarsDice.ParticipantIDs = append([]string{}, original...)
		s.rooms[room.ID] = room
		s.startLiarsDiceRound(room)
		got := room.LiarsDice.ParticipantIDs
		if len(got) != len(original) {
			t.Fatalf("len participants = %d, want %d", len(got), len(original))
		}
		// 集合不变
		seen := map[string]bool{}
		for _, id := range got {
			seen[id] = true
		}
		for _, id := range original {
			if !seen[id] {
				t.Fatalf("missing participant %s after shuffle: %v", id, got)
			}
		}
		if stringsJoin(got) != stringsJoin(original) {
			sawDifferent = true
			break
		}
	}
	if !sawDifferent {
		t.Fatal("expected at least one shuffled order different from join order")
	}
}

func stringsJoin(list []string) string {
	if len(list) == 0 {
		return ""
	}
	out := list[0]
	for _, s := range list[1:] {
		out += "," + s
	}
	return out
}

// TestResolveLiarsDiceChallenge_BidderWins：手动摆好骰子让实际数量 >= 叫点数，验证叫点者赢、
// 挑战者输，排位分正确结算，历史记录字段完整。
func TestResolveLiarsDiceChallenge_BidderWins(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	room.Settings.EnableRanked = true
	a, b, c := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob"), newLiarsDiceTestPlayer("c", "Carol")
	s.players[a.ID], s.players[b.ID], s.players[c.ID] = a, b, c
	room.LiarsDice.ParticipantIDs = []string{"a", "b", "c"}
	s.rooms[room.ID] = room

	// 手摆骰子：面值 4 总共出现 5 次（a:2 + b:2 + c:1），叫点 5 个 4 应该成立。
	room.LiarsDiceHands = map[string][]int{
		"a": {4, 4, 1, 2, 3},
		"b": {4, 4, 5, 6, 2},
		"c": {4, 1, 1, 3, 6},
	}
	room.LiarsDice.DiceCounts = map[string]int{"a": 5, "b": 5, "c": 5}
	room.Phase = types.PhaseChoosing
	bid := types.LiarsDiceBid{PlayerID: "b", Count: 5, Face: 4, At: 1}
	room.LiarsDice.CurrentBid = &bid
	room.LiarsDice.CurrentTurn = "c"

	s.resolveLiarsDiceChallenge(room, "c")

	if !room.LiarsDice.Ended {
		t.Fatalf("round should be marked ended")
	}
	if room.LiarsDice.WinnerID != "b" || room.LiarsDice.LoserID != "c" {
		t.Fatalf("winner/loser = %s/%s, want b/c", room.LiarsDice.WinnerID, room.LiarsDice.LoserID)
	}
	// 4 出现 5 次 + 1 出现 3 次（1 未被喊过，仍是万能点）= 8 >= 5
	if room.LiarsDice.ActualCount != 8 {
		t.Fatalf("actualCount = %d, want 8", room.LiarsDice.ActualCount)
	}
	if room.Phase != types.PhaseResult {
		t.Fatalf("phase = %v, want result", room.Phase)
	}
	if b.Stats.Wins != 1 || c.Stats.Losses != 1 {
		t.Fatalf("stats not updated: b.Wins=%d c.Losses=%d", b.Stats.Wins, c.Stats.Losses)
	}
	// 第三人 a 未参与叫点/开牌，应记平；胜/负不应误记。
	if a.Stats.Draws != 1 || a.Stats.Wins != 0 || a.Stats.Losses != 0 {
		t.Fatalf("draw participant stats: a wins=%d losses=%d draws=%d, want 0/0/1", a.Stats.Wins, a.Stats.Losses, a.Stats.Draws)
	}
	if a.GameStats.LiarsDice.Draws != 1 {
		t.Fatalf("a.GameStats.LiarsDice.Draws=%d, want 1", a.GameStats.LiarsDice.Draws)
	}
	if b.Stats.RankedPoints <= 0 || c.Stats.RankedPoints >= 0 {
		t.Fatalf("ranked points not applied: b=%d c=%d", b.Stats.RankedPoints, c.Stats.RankedPoints)
	}
	// 平局玩家不计排位分。
	if a.Stats.RankedPoints != 0 {
		t.Fatalf("draw participant should not get ranked points: a=%d", a.Stats.RankedPoints)
	}
	if len(room.RoundHistory) != 1 {
		t.Fatalf("expected 1 history item, got %d", len(room.RoundHistory))
	}
	item := room.RoundHistory[0]
	if item.LiarsDiceWinnerID != "b" || item.LiarsDiceLoserID != "c" || item.LiarsDiceActualCount != 8 {
		t.Fatalf("history item liarsDice fields wrong: %+v", item)
	}
	if len(item.LiarsDiceHands) != 3 {
		t.Fatalf("history should reveal all 3 hands, got %d", len(item.LiarsDiceHands))
	}
}

// TestResolveLiarsDiceChallenge_OnesWildDisabled：一旦本局喊过面值 1，1 就不再算万能点。
func TestResolveLiarsDiceChallenge_OnesWildDisabled(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	a, b := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob")
	s.players[a.ID], s.players[b.ID] = a, b
	room.LiarsDice.ParticipantIDs = []string{"a", "b"}
	s.rooms[room.ID] = room

	// 全是 1，如果 1 仍是万能点，喊 "4 个 4" 会成立（1 算进 4 的统计）；
	// 但本局已经喊过 1（OnesWildDisabled=true），所以 1 不再算进 4 的统计，实际 0 个 4，叫点不成立。
	room.LiarsDiceHands = map[string][]int{
		"a": {1, 1, 1, 1, 1},
		"b": {1, 1, 1, 1, 1},
	}
	room.LiarsDice.DiceCounts = map[string]int{"a": 5, "b": 5}
	room.LiarsDice.OnesWildDisabled = true
	room.Phase = types.PhaseChoosing
	bid := types.LiarsDiceBid{PlayerID: "a", Count: 4, Face: 4, At: 1}
	room.LiarsDice.CurrentBid = &bid

	s.resolveLiarsDiceChallenge(room, "b")

	if room.LiarsDice.ActualCount != 0 {
		t.Fatalf("actualCount = %d, want 0 (ones wild disabled)", room.LiarsDice.ActualCount)
	}
	if room.LiarsDice.WinnerID != "b" || room.LiarsDice.LoserID != "a" {
		t.Fatalf("challenger should win when bid doesn't hold: winner=%s loser=%s", room.LiarsDice.WinnerID, room.LiarsDice.LoserID)
	}
}

// TestResolveLiarsDiceChallenge_PunishmentSkipsResultPhase：开启惩罚时，resolveLiarsDiceChallenge
// 先把 room.Phase 设成 PhaseResult，紧接着 setupPunishmentForPlayers 会在同一次调用里把它覆盖成
// PhasePunishment——client 广播里永远看不到 "result" 这个中间态。前端的开牌揭晓面板必须同时兼容
// "result" 和 "punishment" 两种 phase（RPS 的 Settlement 组件就是这么处理的），否则开惩罚的房间
// 结算时骰子揭晓面板会被跳过，直接看到惩罚阶段，见 LiarsDicePanel.tsx 的复现 bug。
func TestResolveLiarsDiceChallenge_PunishmentSkipsResultPhase(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	room.Settings.EnablePunishment = true
	a, b := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob")
	s.players[a.ID], s.players[b.ID] = a, b
	room.LiarsDice.ParticipantIDs = []string{"a", "b"}
	s.rooms[room.ID] = room

	room.LiarsDiceHands = map[string][]int{
		"a": {4, 4, 1, 2, 3},
		"b": {6, 5, 2, 2, 6},
	}
	room.LiarsDice.DiceCounts = map[string]int{"a": 5, "b": 5}
	room.Phase = types.PhaseChoosing
	bid := types.LiarsDiceBid{PlayerID: "a", Count: 5, Face: 4, At: 1}
	room.LiarsDice.CurrentBid = &bid
	room.LiarsDice.CurrentTurn = "b"

	s.resolveLiarsDiceChallenge(room, "b")

	if room.Phase != types.PhasePunishment {
		t.Fatalf("phase = %v, want punishment (loser must be punished)", room.Phase)
	}
	if !room.LiarsDice.Ended {
		t.Fatalf("round should still be marked ended even though phase skipped straight to punishment")
	}
	if room.LiarsDice.RevealedHands == nil {
		t.Fatalf("revealedHands must still be populated so the reveal panel can render during punishment phase")
	}
	if room.LiarsDice.WinnerID == "" || room.LiarsDice.LoserID == "" {
		t.Fatalf("winner/loser must be set: winner=%q loser=%q", room.LiarsDice.WinnerID, room.LiarsDice.LoserID)
	}
}

// TestLiarsDiceDisconnectForfeit：断线玩家判负，入席顺序上的"上家"判胜。
func TestLiarsDiceDisconnectForfeit(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	room.Settings.EnableRanked = true
	a, b, c := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob"), newLiarsDiceTestPlayer("c", "Carol")
	s.players[a.ID], s.players[b.ID], s.players[c.ID] = a, b, c
	room.LiarsDice.ParticipantIDs = []string{"a", "b", "c"}
	room.Phase = types.PhaseChoosing
	s.rooms[room.ID] = room

	// b 掉线：上家是 a（入席顺序里 b 前一位），a 应该判胜。
	s.createLiarsDiceDisconnectForfeit(room, b)
	ok := s.applyLiarsDiceDisconnectForfeit(room, b)
	if !ok {
		t.Fatalf("applyLiarsDiceDisconnectForfeit returned false")
	}
	if room.LiarsDice.WinnerID != "a" || room.LiarsDice.LoserID != "b" {
		t.Fatalf("winner/loser = %s/%s, want a/b (a is b's predecessor)", room.LiarsDice.WinnerID, room.LiarsDice.LoserID)
	}
	if a.Stats.Wins != 1 || b.Stats.Losses != 1 {
		t.Fatalf("stats not applied: a.Wins=%d b.Losses=%d", a.Stats.Wins, b.Stats.Losses)
	}
	// 第三人 c 未成为胜/负方，应记平。
	if c.Stats.Draws != 1 || c.Stats.Wins != 0 || c.Stats.Losses != 0 {
		t.Fatalf("draw participant stats: c wins=%d losses=%d draws=%d, want 0/0/1", c.Stats.Wins, c.Stats.Losses, c.Stats.Draws)
	}
	if room.Phase != types.PhaseResult {
		t.Fatalf("phase = %v, want result", room.Phase)
	}
}

// TestLiarsDicePunishmentReviewerResolvesWinner：大话骰不进 Seats，惩罚审核人必须由
// 最近一条对局记录里的 winner/loser 解析，而不是 seatOf。
func TestLiarsDicePunishmentReviewerResolvesWinner(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	winner := newLiarsDiceTestPlayer("w", "Winner")
	loser := newLiarsDiceTestPlayer("l", "Loser")
	other := newLiarsDiceTestPlayer("o", "Other")
	s.players["w"], s.players["l"], s.players["o"] = winner, loser, other
	room.RoundHistory = []types.RoundHistoryItem{{
		GameID: types.GameLiarsDice, LiarsDiceWinnerID: "w", LiarsDiceLoserID: "l",
	}}
	s.rooms[room.ID] = room

	if got := s.liarsDicePunishmentReviewer(room, "l"); got == nil || got.ID != "w" {
		t.Fatalf("reviewer for loser = %v, want winner w", got)
	}
	if got := s.punishmentReviewer(room, "l"); got == nil || got.ID != "w" {
		t.Fatalf("punishmentReviewer(loser) = %v, want w", got)
	}
	if got := s.taskAssigner(room, "l"); got == nil || got.ID != "w" {
		t.Fatalf("taskAssigner(loser) = %v, want w", got)
	}
	// 赢家可以审核输家；第三方 / 自己 / 反向都不行。
	if !s.canReviewPlayer(room, "w", "l") {
		t.Fatalf("winner should be able to review loser")
	}
	if s.canReviewPlayer(room, "o", "l") {
		t.Fatalf("third participant must not review the loser")
	}
	if s.canReviewPlayer(room, "l", "w") {
		t.Fatalf("loser must not review the winner")
	}
	if s.canReviewPlayer(room, "w", "w") {
		t.Fatalf("nobody reviews themselves")
	}
}

// TestReturnLiarsDiceToReadyClearsRoundState：一局结束后收回 Ready，清掉叫点/骰子/开牌与
// 已准备标记，但保留参战名单，让玩家能调整名单并重新准备。
func TestReturnLiarsDiceToReadyClearsRoundState(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	a, b := newLiarsDiceTestPlayer("a", "Alice"), newLiarsDiceTestPlayer("b", "Bob")
	s.players["a"], s.players["b"] = a, b
	room.LiarsDice.ParticipantIDs = []string{"a", "b"}
	room.LiarsDice.ReadyPlayerIDs = []string{"a", "b"}
	s.rooms[room.ID] = room
	s.startLiarsDiceRound(room)
	// 走一次开牌，进入 Result
	room.LiarsDice.CurrentBid = &types.LiarsDiceBid{PlayerID: "a", Count: 3, Face: 5}
	room.LiarsDice.CurrentTurn = "b"
	s.resolveLiarsDiceChallenge(room, "b")
	if room.Phase != types.PhaseResult {
		t.Fatalf("phase should be result before reset, got %v", room.Phase)
	}

	s.returnLiarsDiceToReady(room)

	if room.Phase != types.PhaseReady {
		t.Fatalf("phase = %v, want ready", room.Phase)
	}
	if len(room.LiarsDice.ReadyPlayerIDs) != 0 {
		t.Fatalf("ready flags should be cleared, got %v", room.LiarsDice.ReadyPlayerIDs)
	}
	if room.LiarsDice.CurrentBid != nil || room.LiarsDice.Ended || room.LiarsDice.RevealedHands != nil {
		t.Fatalf("round state not cleared: bid=%v ended=%v reveal=%v", room.LiarsDice.CurrentBid, room.LiarsDice.Ended, room.LiarsDice.RevealedHands)
	}
	if len(room.LiarsDiceHands) != 0 {
		t.Fatalf("private hands should be cleared, got %v", room.LiarsDiceHands)
	}
	if len(room.LiarsDice.ParticipantIDs) != 2 {
		t.Fatalf("roster must be preserved, got %v", room.LiarsDice.ParticipantIDs)
	}
}

func TestSetupPunishmentForPlayersIsGenericOverIDs(t *testing.T) {
	s := newLiarsDiceTestServer(t)
	room := newLiarsDiceTestRoom("r1", 2, 8)
	room.Settings.EnablePunishment = true
	loser := newLiarsDiceTestPlayer("loser", "Loser")
	s.players[loser.ID] = loser

	s.setupPunishmentForPlayers(room, []string{"loser"})

	if room.Phase != types.PhasePunishment {
		t.Fatalf("phase = %v, want punishment", room.Phase)
	}
	if len(room.PunishedPlayerIDs) != 1 || room.PunishedPlayerIDs[0] != "loser" {
		t.Fatalf("PunishedPlayerIDs = %v", room.PunishedPlayerIDs)
	}
	if loser.Stats.Punishments != 1 {
		t.Fatalf("loser punishments = %d, want 1", loser.Stats.Punishments)
	}
}
