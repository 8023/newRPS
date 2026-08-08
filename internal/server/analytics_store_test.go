package server

import (
	"testing"
)

func TestAnalyticsStoreUpsertIdempotent(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newAnalyticsStore(db)

	visitor := analyticsVisitorRow{
		Visitor: "abc123", FirstAt: 1000, FirstDay: 1, LastAt: 1000,
		SessionsDelta: 1, FirstReferrer: "example.com", FirstProvince: "广东省",
	}
	session := analyticsSessionRow{
		ID: "sess-1", Visitor: "abc123", StartedAt: 1000, LastAt: 2000, Day: 1,
		Browser: "Chrome", OS: "Windows", DeviceType: "desktop",
		PageviewsDelta: 2, EventsDelta: 3,
	}
	events := []analyticsEventRow{
		{At: 1100, Day: 1, Source: 0, SessionID: "sess-1", Visitor: "abc123", Name: "pageview", View: "lobby"},
		{At: 1200, Day: 1, Source: 0, SessionID: "sess-1", Visitor: "abc123", Name: "pageview", View: "room"},
	}
	if err := store.writeBatch(visitor, true, session, events); err != nil {
		t.Fatalf("writeBatch: %v", err)
	}
	// 同会话再写一批：pageviews/events 累加，访客 sessions 不再 +1
	session2 := session
	session2.LastAt = 3000
	session2.PageviewsDelta = 1
	session2.EventsDelta = 1
	visitor2 := visitor
	visitor2.SessionsDelta = 0
	visitor2.LastAt = 3000
	if err := store.writeBatch(visitor2, false, session2, []analyticsEventRow{
		{At: 2500, Day: 1, Source: 0, SessionID: "sess-1", Visitor: "abc123", Name: "session_ping"},
	}); err != nil {
		t.Fatalf("writeBatch 2: %v", err)
	}

	var pv, ev, sess int
	if err := db.QueryRow(`SELECT pageviews, events FROM analytics_sessions WHERE id='sess-1'`).Scan(&pv, &ev); err != nil {
		t.Fatalf("query session: %v", err)
	}
	if pv != 3 || ev != 4 {
		t.Fatalf("pageviews/events = %d/%d, want 3/4", pv, ev)
	}
	if err := db.QueryRow(`SELECT sessions FROM analytics_visitors WHERE visitor='abc123'`).Scan(&sess); err != nil {
		t.Fatalf("query visitor: %v", err)
	}
	if sess != 1 {
		t.Fatalf("visitor sessions = %d, want 1", sess)
	}
	var ec int
	if err := db.QueryRow(`SELECT COUNT(*) FROM analytics_events`).Scan(&ec); err != nil {
		t.Fatalf("count events: %v", err)
	}
	if ec != 3 {
		t.Fatalf("events count = %d, want 3", ec)
	}
}

func TestAnalyticsDailyUpsert(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newAnalyticsStore(db)
	rows := []analyticsDailyRow{
		{Day: 100, Metric: "dau", Value: 10},
		{Day: 100, Metric: "dau", Value: 12}, // upsert 覆盖
	}
	if err := store.upsertDaily(rows[:1]); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := store.upsertDaily(rows[1:]); err != nil {
		t.Fatalf("upsert 2: %v", err)
	}
	var v int64
	if err := db.QueryRow(`SELECT value FROM analytics_daily WHERE day=100 AND metric='dau'`).Scan(&v); err != nil {
		t.Fatalf("query: %v", err)
	}
	if v != 12 {
		t.Fatalf("value = %d, want 12", v)
	}
}
