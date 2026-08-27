package server

import (
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func newRankedScoreTestServer(t *testing.T) *Server {
	t.Helper()
	s := &Server{
		players:           map[string]*PlayerState{},
		rooms:             map[string]*RoomState{},
		playerUpdateDelay: time.Hour,
		cfg: types.AppConfig{
			RankedScore: types.RankedScoreConfig{
				Max: 4999, Min: -4999, NameWarMin: -9999, DailyDecayRatio: 0.98,
			},
			Titles: []types.TitleSegment{
				{ID: "neg4", MinPercent: -100, MaxPercent: -75, Names: []string{"反向传奇"}},
				{ID: "neg1", MinPercent: -25, MaxPercent: -0.01, Names: []string{"新手保护期"}},
				{ID: "pos0", MinPercent: 0, MaxPercent: 0, Names: []string{"初心拳手"}},
				{ID: "pos4", MinPercent: 75, MaxPercent: 100, Names: []string{"常胜拳王"}},
			},
		},
	}
	s.cfg.NameWar.PenaltyThreshold = -4999
	return s
}

func newRankedScoreTestPlayer(id string) *PlayerState {
	return &PlayerState{
		PublicPlayer: types.PublicPlayer{ID: id, Name: id, DisplayName: id},
		PlayerID:     id,
	}
}

func TestUpdateRankedPointsIsUnbounded(t *testing.T) {
	s := newRankedScoreTestServer(t)
	p := newRankedScoreTestPlayer("p1")
	s.players[p.ID] = p

	s.updateRankedPoints(p, 8000)
	if p.Stats.RankedPoints != 8000 {
		t.Fatalf("want unbounded 8000, got %d", p.Stats.RankedPoints)
	}
	if p.Stats.HighestScore != 8000 {
		t.Fatalf("highest score not recorded: %d", p.Stats.HighestScore)
	}

	s.updateRankedPoints(p, -20000)
	if p.Stats.RankedPoints != -12000 {
		t.Fatalf("want unbounded -12000, got %d", p.Stats.RankedPoints)
	}
	if p.Stats.LowestScore != -12000 {
		t.Fatalf("lowest score not recorded: %d", p.Stats.LowestScore)
	}
	// 历史最高分不应因为分数回落而回退。
	if p.Stats.HighestScore != 8000 {
		t.Fatalf("highest score should not regress: %d", p.Stats.HighestScore)
	}
}

func TestSetRankedPointsByAdminIsUnbounded(t *testing.T) {
	s := newRankedScoreTestServer(t)
	p := newRankedScoreTestPlayer("p2")
	s.players[p.ID] = p

	s.setRankedPointsByAdmin(p, 15000)
	if p.Stats.RankedPoints != 15000 {
		t.Fatalf("want unbounded 15000, got %d", p.Stats.RankedPoints)
	}
	if p.Stats.HighestScore != 15000 {
		t.Fatalf("highest score not recorded: %d", p.Stats.HighestScore)
	}
}

func TestPublicPlayerClampsDisplayIncludingExtremes(t *testing.T) {
	s := newRankedScoreTestServer(t)
	p := newRankedScoreTestPlayer("p3")
	s.players[p.ID] = p
	s.updateRankedPoints(p, 8000)

	pub := s.publicPlayer(p)
	if pub.Stats.RankedPoints != 4999 {
		t.Fatalf("display should clamp to max 4999, got %d", pub.Stats.RankedPoints)
	}
	if pub.Stats.HighestScore != 4999 {
		t.Fatalf("historical highest display should also clamp to 4999, got %d", pub.Stats.HighestScore)
	}
	// publicPlayer() 下发给普通连接：Sort* 不得带出真实分，必须与展示值一致，
	// 否则任何玩家都能靠 player:get/players:roster 等报文读到真实积分。
	if pub.Stats.SortRankedPoints != 4999 || pub.Stats.SortHighestScore != 4999 {
		t.Fatalf("public sort keys must not leak real scores: ranked=%d highest=%d", pub.Stats.SortRankedPoints, pub.Stats.SortHighestScore)
	}
	if p.Stats.RankedPoints != 8000 {
		t.Fatalf("underlying stored value must stay unbounded, got %d", p.Stats.RankedPoints)
	}

	// publicPlayerAdmin() 仅供管理员后台单播使用：Sort* 换回真实存储分。
	admin := s.publicPlayerAdmin(p)
	if admin.Stats.RankedPoints != 4999 {
		t.Fatalf("admin display should still clamp to max 4999, got %d", admin.Stats.RankedPoints)
	}
	if admin.Stats.SortRankedPoints != 8000 || admin.Stats.SortHighestScore != 8000 {
		t.Fatalf("admin sort keys must keep real scores: ranked=%d highest=%d", admin.Stats.SortRankedPoints, admin.Stats.SortHighestScore)
	}

	// NameWar 开启时下限更低。
	p.NameWarEnabled = boolPtr(true)
	s.updateRankedPoints(p, -20000)
	pub = s.publicPlayer(p)
	if pub.Stats.RankedPoints != -9999 {
		t.Fatalf("display should clamp to nameWarMin -9999, got %d", pub.Stats.RankedPoints)
	}
	if pub.Stats.LowestScore != -9999 {
		t.Fatalf("historical lowest display should clamp to nameWarMin, got %d", pub.Stats.LowestScore)
	}
	if pub.Stats.SortLowestScore != -9999 {
		t.Fatalf("public sort lowest must not leak real score, got %d", pub.Stats.SortLowestScore)
	}
	admin = s.publicPlayerAdmin(p)
	if admin.Stats.SortLowestScore != -12000 {
		t.Fatalf("admin sort lowest must stay real -12000, got %d", admin.Stats.SortLowestScore)
	}
}

func TestTitleSegmentForUsesPercentOfDisplayRange(t *testing.T) {
	s := newRankedScoreTestServer(t)

	// 80% of max 4999 ≈ 3999.2 → pos4
	high := s.titleSegmentFor(4000)
	if high == nil || high.ID != "pos4" {
		t.Fatalf("high score should map to pos4 by percent, got %+v", high)
	}
	// far beyond display max still clamps to highest segment
	ultra := s.titleSegmentFor(50000)
	if ultra == nil || ultra.ID != "pos4" {
		t.Fatalf("very high score should map to highest segment, got %+v", ultra)
	}
	low := s.titleSegmentFor(-50000)
	if low == nil || low.ID != "neg4" {
		t.Fatalf("very low score should map to lowest segment, got %+v", low)
	}
	mid := s.titleSegmentFor(0)
	if mid == nil || mid.ID != "pos0" {
		t.Fatalf("zero should match pos0, got %+v", mid)
	}
}

func TestNameWarPenaltyThresholdUsesRealScore(t *testing.T) {
	s := newRankedScoreTestServer(t)
	p := newRankedScoreTestPlayer("nw")
	p.NameWarEnabled = boolPtr(true)
	p.NameWarAllowRename = boolPtr(true)
	p.Stats.RankedPoints = -5000
	s.refreshNameWarState(p, nowMs())
	if !ptrBool(p.NameWarPunished) {
		t.Fatal("real -5000 should trigger punishment at threshold -4999")
	}
	if !s.isNameWarRenameTarget(p) {
		t.Fatal("rename target should use real stored score")
	}
}

func TestSelfTitleOutranksSystemTitleWhileNameWarPunished(t *testing.T) {
	s := newRankedScoreTestServer(t)
	p := newRankedScoreTestPlayer("nw2")
	p.NameWarEnabled = boolPtr(true)
	p.NameWarAllowRename = boolPtr(true)
	p.Stats.RankedPoints = -5444
	p.Stats.SelfTitle = "abcd"
	s.refreshNameWarState(p, nowMs())
	if !ptrBool(p.NameWarPunished) {
		t.Fatal("real -5444 should trigger name-war punishment at threshold -4999")
	}
	pub := s.publicPlayer(p)
	if pub.Stats.Title != "abcd" {
		t.Fatalf("self-set title must outrank the system rank-tier title even while name-war punished, got %q", pub.Stats.Title)
	}
	if pub.Stats.TitleSource != "self" {
		t.Fatalf("title source should be self, got %q", pub.Stats.TitleSource)
	}
}

func TestRankedDailyDecayTruncatesTowardZeroAndIsIdempotentPerDay(t *testing.T) {
	s := newRankedScoreTestServer(t)
	pos := newRankedScoreTestPlayer("pos")
	neg := newRankedScoreTestPlayer("neg")
	s.players[pos.ID] = pos
	s.players[neg.ID] = neg
	pos.Stats.RankedPoints = 101
	neg.Stats.RankedPoints = -101

	s.applyRankedDailyDecay()
	if pos.Stats.RankedPoints != 98 { // 101*0.98 = 98.98 -> truncate to 98
		t.Fatalf("positive decay truncation: got %d", pos.Stats.RankedPoints)
	}
	if neg.Stats.RankedPoints != -98 { // -101*0.98 = -98.98 -> truncate toward zero to -98
		t.Fatalf("negative decay truncation: got %d", neg.Stats.RankedPoints)
	}

	// 同一天内再次调用不应重复衰减。
	s.applyRankedDailyDecay()
	if pos.Stats.RankedPoints != 98 || neg.Stats.RankedPoints != -98 {
		t.Fatalf("decay should not double-apply within the same day: pos=%d neg=%d", pos.Stats.RankedPoints, neg.Stats.RankedPoints)
	}
}
