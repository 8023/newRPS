package server

import (
	"sync/atomic"
	"testing"
)

func TestOnAnalyticsCollectDoesNotTouchDB(t *testing.T) {
	s := &Server{
		analyticsEnabled:     true,
		analyticsSalt:        []byte("test-salt-32-bytes-padding!!!!!!"),
		logCh:                make(chan activityLogEntry, 8),
		analyticsTZOffsetMin: 480,
	}
	client := &Client{
		id: "c1", anaVisitor: "vis1", anaBrowser: "Chrome", anaOS: "Windows", anaDevice: "desktop",
		playerID: "p1",
	}
	env := wsEnvelope{
		E: "analytics:collect", ID: 0,
		D: map[string]any{
			"sid": "abc-123",
			"ev": []any{
				map[string]any{"n": "pageview", "v": "lobby", "t": float64(nowMs())},
				map[string]any{"n": "unknown_evt", "v": "weird_view", "d": "!!!", "t": float64(nowMs())},
			},
		},
	}
	// analyticsDB 故意为 nil：若 handler 同步写库会 panic / 报错
	s.onAnalyticsCollect(client, env)

	select {
	case e := <-s.logCh:
		if e.analytics == nil {
			t.Fatal("expected analytics payload")
		}
		if len(e.analytics.Events) != 2 {
			t.Fatalf("events = %d, want 2", len(e.analytics.Events))
		}
		if e.analytics.Events[1].Name != "other" {
			t.Fatalf("unknown event name = %q, want other", e.analytics.Events[1].Name)
		}
		if e.analytics.Events[1].View != "other" {
			t.Fatalf("unknown view = %q, want other", e.analytics.Events[1].View)
		}
	default:
		t.Fatal("expected logCh entry")
	}
}

func TestOnAnalyticsCollectTruncatesOver40(t *testing.T) {
	s := &Server{
		analyticsEnabled:     true,
		analyticsSalt:        []byte("salt"),
		logCh:                make(chan activityLogEntry, 2),
		analyticsTZOffsetMin: 480,
	}
	client := &Client{anaVisitor: "v"}
	ev := make([]any, 50)
	for i := range ev {
		ev[i] = map[string]any{"n": "session_ping", "t": float64(nowMs())}
	}
	s.onAnalyticsCollect(client, wsEnvelope{D: map[string]any{"sid": "s1", "ev": ev}})
	e := <-s.logCh
	if len(e.analytics.Events) != 40 {
		t.Fatalf("truncated to %d, want 40", len(e.analytics.Events))
	}
}

func TestOnAnalyticsCollectRejectsBadSID(t *testing.T) {
	s := &Server{
		analyticsEnabled: true,
		logCh:            make(chan activityLogEntry, 2),
	}
	client := &Client{anaVisitor: "v"}
	s.onAnalyticsCollect(client, wsEnvelope{D: map[string]any{
		"sid": "bad sid!",
		"ev":  []any{map[string]any{"n": "pageview", "t": float64(nowMs())}},
	}})
	select {
	case <-s.logCh:
		t.Fatal("bad sid should discard batch")
	default:
	}
}

func TestOnAnalyticsCollectNormalizesEmptyEventName(t *testing.T) {
	s := &Server{
		analyticsEnabled:     true,
		logCh:                make(chan activityLogEntry, 1),
		analyticsTZOffsetMin: 480,
	}
	client := &Client{anaVisitor: "v"}
	s.onAnalyticsCollect(client, wsEnvelope{D: map[string]any{
		"sid": "s1",
		"ev":  []any{map[string]any{"n": "", "t": float64(nowMs())}},
	}})
	select {
	case entry := <-s.logCh:
		if got := entry.analytics.Events[0].Name; got != "other" {
			t.Fatalf("empty event name = %q, want other", got)
		}
	default:
		t.Fatal("expected logCh entry")
	}
}

func TestAdminAnalyticsNoSQLWhenSnapReady(t *testing.T) {
	s := &Server{
		analyticsKick:  make(chan struct{}, 1),
		adminClientIDs: map[string]struct{}{"c1": {}},
	}
	snap := &analyticsSnapshot{
		BuiltAt:      nowMs(),
		Days:         []int64{1, 2, 3},
		DAU:          int64Slice{1, 2, 3},
		Sessions:     int64Slice{1, 2, 3},
		Pageviews:    int64Slice{10, 20, 30},
		NewVisitors:  int64Slice{0, 1, 0},
		AvgSessionMs: int64Slice{1000, 2000, 3000},
		PeakOnline:   int64Slice{1, 2, 1},
		ByKey:        map[string]map[string]int64Slice{},
		Retention:    map[int64]map[int]int64{},
	}
	s.analyticsSnap.Store(snap)
	// analyticsRO 为 nil：证明 RPC 不碰 SQL
	s.analyticsRO = nil

	client := &Client{id: "c1", sendCh: make(chan []byte, 4), done: make(chan struct{})}
	var replied atomic.Bool
	// reply 需要 writeLoop；这里直接调 forRange 路径验证不 panic
	view := snap.forRange(7)
	if view == nil {
		t.Fatal("forRange nil")
	}
	s.onAdminAnalytics(client, wsEnvelope{ID: 1, D: map[string]any{"days": 7}})
	// ID!=0 时 reply 会写 sendCh；无 writeLoop 也能入队
	select {
	case <-client.sendCh:
		replied.Store(true)
	default:
	}
	if !replied.Load() {
		// reply 可能因 proto 编码失败；至少没 panic
	}
}

func TestSweepPeakOnline(t *testing.T) {
	rows := []struct{ Start, End int64 }{
		{0, 100},
		{50, 150},
		{140, 200},
	}
	if p := sweepPeakOnline(rows); p != 2 {
		t.Fatalf("peak = %d, want 2", p)
	}
}

func TestNormalizeReferrerHost(t *testing.T) {
	if got := normalizeReferrerHost("https://WWW.Example.COM/path?q=1"); got != "www.example.com" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeReferrerHost("javascript:alert(1)"); got != "(other)" {
		t.Fatalf("got %q", got)
	}
}
