package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestAccumulateClientOnlineMs(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Stats.TotalOnlineMs = 1000
	s.players["p1"] = p
	c := &Client{id: "sock1", playerID: "p1", connectedAt: 10_000, onlineCreditedAt: 10_000}
	s.accumulateClientOnlineMs(c, 15_000)
	if p.Stats.TotalOnlineMs != 6000 {
		t.Fatalf("TotalOnlineMs=%d want 6000", p.Stats.TotalOnlineMs)
	}
	// no player id: no change
	c2 := &Client{id: "sock2", connectedAt: 1}
	s.accumulateClientOnlineMs(c2, 999)
	if p.Stats.TotalOnlineMs != 6000 {
		t.Fatalf("guest should not change: %d", p.Stats.TotalOnlineMs)
	}
	s.persistMu.Lock()
	dirty := s.persistDirty
	s.persistMu.Unlock()
	if !dirty {
		t.Fatalf("accumulate should mark persist dirty")
	}
}

// TestCheckpointOnlineMsNoDoubleCount 定期 checkpoint 后断线只计剩余段落。
func TestCheckpointOnlineMsNoDoubleCount(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Stats.TotalOnlineMs = 0
	s.players["p1"] = p
	// 固定时钟不好注入，用 connectedAt 远在过去 + 先手动设 onlineCreditedAt。
	c := &Client{id: "sock1", playerID: "p1", connectedAt: 1_000, onlineCreditedAt: 1_000}
	s.clients["sock1"] = c

	// 模拟 checkpoint 已把 1_000→11_000 的 10s 记入（手工对齐字段）。
	p.Stats.TotalOnlineMs = 10_000
	c.onlineCreditedAt = 11_000

	s.accumulateClientOnlineMs(c, 16_000)
	if p.Stats.TotalOnlineMs != 15_000 {
		t.Fatalf("after disconnect TotalOnlineMs=%d want 15000 (10000 + 5000 remaining)", p.Stats.TotalOnlineMs)
	}
	// 再累加同一连接应无增量
	s.accumulateClientOnlineMs(c, 16_000)
	if p.Stats.TotalOnlineMs != 15_000 {
		t.Fatalf("double accumulate: %d", p.Stats.TotalOnlineMs)
	}
}

func TestCheckpointOnlineMsAdvancesCreditedAt(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	s.players["p1"] = p
	// connected ~5s ago
	start := nowMs() - 5_000
	c := &Client{id: "sock1", playerID: "p1", connectedAt: start, onlineCreditedAt: start}
	s.clients["sock1"] = c

	if !s.checkpointOnlineMs() {
		t.Fatal("expected checkpoint to touch online ms")
	}
	if p.Stats.TotalOnlineMs < 4_000 || p.Stats.TotalOnlineMs > 8_000 {
		t.Fatalf("checkpoint TotalOnlineMs=%d want ~5000", p.Stats.TotalOnlineMs)
	}
	if c.onlineCreditedAt < start+4_000 {
		t.Fatalf("onlineCreditedAt not advanced: %d", c.onlineCreditedAt)
	}
	// 立即再 checkpoint 应几乎无增量（或零）
	before := p.Stats.TotalOnlineMs
	s.checkpointOnlineMs()
	if p.Stats.TotalOnlineMs-before > 50 {
		t.Fatalf("second checkpoint added too much: %d -> %d", before, p.Stats.TotalOnlineMs)
	}
}

// TestIngestPersistedPlayerLoadsTotalOnlineMs 回归：加载档案必须带上 TotalOnlineMs，
// 否则重启后内存从 0 起算并写回覆盖库。
func TestIngestPersistedPlayerLoadsTotalOnlineMs(t *testing.T) {
	s := &Server{
		players:      map[string]*PlayerState{},
		playerIdToID: map[string]string{},
		tokenToPlayer: map[string]string{},
		cfg:          types.AppConfig{},
	}
	ok := s.ingestPersistedPlayer(persistedPlayer{
		ID: "pub1", PlayerID: "ident1", ClaimKey: "claim", Name: "Alice",
		GenderID: "male", FactionID: "male",
		Stats: types.PublicStats{TotalOnlineMs: 9_876_543, RankedPoints: 10},
		CreatedAt: 1, LastSeenAt: 2,
	})
	if !ok {
		t.Fatal("ingest failed")
	}
	p := s.players["pub1"]
	if p == nil {
		t.Fatal("player missing")
	}
	if p.Stats.TotalOnlineMs != 9_876_543 {
		t.Fatalf("TotalOnlineMs=%d want 9876543 (must load from disk)", p.Stats.TotalOnlineMs)
	}
}

// TestCloseLiveStateAccumulatesOnlineMsAndMarksDirty 回归：关停累加后必须置 dirty，
// 否则 Run 末尾 flushPersist 会因 dirty==false 跳过写盘。
func TestCloseLiveStateAccumulatesOnlineMsAndMarksDirty(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1"}, SocketID: "sock1"}
	p.Stats.TotalOnlineMs = 1000
	s.players["p1"] = p
	// connectedAt 设为「接近现在」的固定偏移，避免依赖真实时钟精度。
	s.clients["sock1"] = &Client{
		id: "sock1", playerID: "p1", connectedAt: nowMs() - 10_000, onlineCreditedAt: nowMs() - 10_000,
	}

	s.closeLiveStateOnShutdown()

	if p.Stats.TotalOnlineMs < 10_000 {
		t.Fatalf("TotalOnlineMs=%d want >=11000 (1000 base + ~10s session)", p.Stats.TotalOnlineMs)
	}
	s.persistMu.Lock()
	dirty := s.persistDirty
	s.persistMu.Unlock()
	if !dirty {
		t.Fatalf("persistDirty=false after shutdown accumulation")
	}
	// 幂等：再次关停不应重复累加（connectionLogged 已置位）
	before := p.Stats.TotalOnlineMs
	s.closeLiveStateOnShutdown()
	if p.Stats.TotalOnlineMs != before {
		t.Fatalf("double shutdown re-accumulated: %d -> %d", before, p.Stats.TotalOnlineMs)
	}
}

func TestEffectiveOnlineMsIncludesCurrentSession(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Connected = true
	p.SocketID = "sock1"
	p.Stats.TotalOnlineMs = 1000
	s.players["p1"] = p
	// connectedAt 1 hour ago — but we use fixed math via stubbing is hard; just check base when offline
	p.Connected = false
	if got := s.effectiveOnlineMs(p); got != 1000 {
		t.Fatalf("offline effective=%d want 1000", got)
	}
	p.Connected = true
	s.clients["sock1"] = &Client{id: "sock1", connectedAt: nowMs() - 5000, onlineCreditedAt: nowMs() - 5000}
	got := s.effectiveOnlineMs(p)
	if got < 5000 || got > 7000 {
		t.Fatalf("online effective=%d want ~6000", got)
	}
}

func TestEffectiveOnlineMsAfterCheckpoint(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Connected = true
	p.SocketID = "sock1"
	// 已 checkpoint 的 10s + 当前未记账 2s
	p.Stats.TotalOnlineMs = 10_000
	s.players["p1"] = p
	s.clients["sock1"] = &Client{
		id: "sock1", connectedAt: nowMs() - 12_000, onlineCreditedAt: nowMs() - 2_000,
	}
	got := s.effectiveOnlineMs(p)
	if got < 11_000 || got > 14_000 {
		t.Fatalf("effective after checkpoint=%d want ~12000", got)
	}
}
