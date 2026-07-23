package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func boolPtrVal(v bool) *bool { return &v }

func TestMatchAdminPlayerFilters(t *testing.T) {
	now := int64(1_000_000_000_000)
	p := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID: "a",
			Stats: types.PublicStats{RankedPoints: 10},
		},
		LastSeenAt: now - 1000,
		Persistent: true,
	}

	if !matchAdminPlayerFilters(p, adminPlayerListQuery{}, now) {
		t.Fatal("empty query should match")
	}
	p.Connected = false
	if matchAdminPlayerFilters(p, adminPlayerListQuery{Online: true}, now) {
		t.Fatal("online filter should reject offline")
	}
	p.Connected = true
	if !matchAdminPlayerFilters(p, adminPlayerListQuery{Online: true}, now) {
		t.Fatal("online filter should accept connected")
	}
	if matchAdminPlayerFilters(p, adminPlayerListQuery{NameWar: true}, now) {
		t.Fatal("name war filter should reject when disabled")
	}
	p.NameWarEnabled = boolPtrVal(true)
	if !matchAdminPlayerFilters(p, adminPlayerListQuery{NameWar: true}, now) {
		t.Fatal("name war filter should accept when enabled")
	}

	p.Stats.RankedPoints = 0
	if matchAdminPlayerFilters(p, adminPlayerListQuery{RankedNonZero: true}, now) {
		t.Fatal("ranked non-zero should reject 0")
	}
	p.Stats.RankedPoints = 1
	if !matchAdminPlayerFilters(p, adminPlayerListQuery{RankedNonZero: true}, now) {
		t.Fatal("ranked non-zero should accept non-zero")
	}

	// 7 天外离线应被拒
	p.Connected = false
	p.LastSeenAt = now - adminPlayerRecentLoginMs - 1
	if matchAdminPlayerFilters(p, adminPlayerListQuery{RecentLogin7d: true}, now) {
		t.Fatal("recent login filter should reject stale offline")
	}
	// 当前在线即使 lastSeen 旧也通过
	p.Connected = true
	if !matchAdminPlayerFilters(p, adminPlayerListQuery{RecentLogin7d: true}, now) {
		t.Fatal("recent login filter should accept connected")
	}
	// 7 天内离线通过
	p.Connected = false
	p.LastSeenAt = now - 1000
	if !matchAdminPlayerFilters(p, adminPlayerListQuery{RecentLogin7d: true}, now) {
		t.Fatal("recent login filter should accept recent offline")
	}
}

func TestBuildAdminPlayerListSortAndLimit(t *testing.T) {
	s := newTestServer(t)
	// publicPlayer 展示分封顶需要合理的 RankedScore 区间
	s.cfg.RankedScore.Min = -9999
	s.cfg.RankedScore.Max = 9999
	s.cfg.RankedScore.NameWarMin = -9999
	// 积分 5 / 1 / 3，白给 0 / 9 / 2
	mk := func(id string, ranked int, giveaway float64, connected bool) {
		var gv *float64
		var ge *bool
		if giveaway > 0 {
			gv = &giveaway
			ge = boolPtrVal(true)
		}
		s.players[id] = &PlayerState{
			PublicPlayer: types.PublicPlayer{
				ID: id, Name: id, Connected: connected,
				Stats:           types.PublicStats{RankedPoints: ranked},
				GiveawayEnabled: ge,
				GiveawayValue:   gv,
			},
			LastSeenAt: 1000 + int64(ranked),
			Persistent: true,
			PlayerID:   "pid-" + id,
		}
	}
	mk("p5", 5, 0, true)
	mk("p1", 1, 9, false)
	mk("p3", 3, 2, true)

	// 默认：积分升序，不过滤
	res := s.buildAdminPlayerList(adminPlayerListQuery{})
	if res.Total != 3 || res.OnlineCount != 2 || res.OfflineCount != 1 {
		t.Fatalf("counts: total=%d online=%d offline=%d", res.Total, res.OnlineCount, res.OfflineCount)
	}
	if res.Players[0].ID != "p1" || res.Players[1].ID != "p3" || res.Players[2].ID != "p5" {
		t.Fatalf("default ranked asc order: %v %v %v", res.Players[0].ID, res.Players[1].ID, res.Players[2].ID)
	}

	// 积分降序
	res = s.buildAdminPlayerList(adminPlayerListQuery{SortRankedDesc: true})
	if res.Players[0].ID != "p5" || res.Players[2].ID != "p1" {
		t.Fatalf("ranked desc: %v ... %v", res.Players[0].ID, res.Players[2].ID)
	}

	// 白给降序优先
	res = s.buildAdminPlayerList(adminPlayerListQuery{SortGiveawayDesc: true})
	if res.Players[0].ID != "p1" || res.Players[1].ID != "p3" || res.Players[2].ID != "p5" {
		t.Fatalf("giveaway desc: %v %v %v", res.Players[0].ID, res.Players[1].ID, res.Players[2].ID)
	}

	// 仅在线
	res = s.buildAdminPlayerList(adminPlayerListQuery{Online: true})
	if res.Total != 2 || len(res.Players) != 2 {
		t.Fatalf("online only total=%d len=%d", res.Total, len(res.Players))
	}
	// online/offline 计数在「仅在线」之外的过滤器上统计
	if res.OnlineCount != 2 || res.OfflineCount != 1 {
		t.Fatalf("split counts online=%d offline=%d", res.OnlineCount, res.OfflineCount)
	}

	// 在线过滤器 + 其它条件：离线玩家被挡在 total 外，但仍进 offlineCount
	res = s.buildAdminPlayerList(adminPlayerListQuery{Online: true, RankedNonZero: true})
	if res.Total != 2 {
		t.Fatalf("online+nonzero total=%d", res.Total)
	}

	// 截断
	res = s.buildAdminPlayerList(adminPlayerListQuery{Limit: 1})
	if !res.Truncated || res.Total != 3 || len(res.Players) != 1 {
		t.Fatalf("limit: truncated=%v total=%d len=%d", res.Truncated, res.Total, len(res.Players))
	}
}

func TestClampAdminPlayerListLimit(t *testing.T) {
	if clampAdminPlayerListLimit(0) != adminPlayerListDefaultLimit {
		t.Fatal("default")
	}
	if clampAdminPlayerListLimit(9999) != adminPlayerListMaxLimit {
		t.Fatal("max")
	}
	if clampAdminPlayerListLimit(50) != 50 {
		t.Fatal("passthrough")
	}
}
