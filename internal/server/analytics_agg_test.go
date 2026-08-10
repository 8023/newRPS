package server

import (
	"fmt"
	"testing"
)

func TestAnalyticsDay(t *testing.T) {
	// 本地日边界：以 UTC+8 的 2020-01-01 00:00（= 2019-12-31 16:00 UTC）为日起点。
	// 2019-12-31 16:00:00 UTC ms:
	dayStartUTC := int64(1577808000000) // 2020-01-01 00:00:00 CST
	d1 := analyticsDay(dayStartUTC, 480)
	d2 := analyticsDay(dayStartUTC+86_400_000-1, 480)
	if d1 != d2 {
		t.Fatalf("same local day: %d vs %d", d1, d2)
	}
	d3 := analyticsDay(dayStartUTC+86_400_000, 480)
	if d3 != d1+1 {
		t.Fatalf("next day: %d want %d", d3, d1+1)
	}
}

func TestRebuildDayFromSeed(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newAnalyticsStore(db)
	tz := 480
	day := analyticsDay(nowMs(), tz)
	// seed sessions
	_ = store.writeBatch(
		analyticsVisitorRow{Visitor: "v1", FirstAt: nowMs(), FirstDay: day, LastAt: nowMs(), SessionsDelta: 1},
		true,
		analyticsSessionRow{
			ID: "s1", Visitor: "v1", StartedAt: nowMs() - 60_000, LastAt: nowMs(), Day: day,
			Browser: "Chrome", OS: "Windows", DeviceType: "desktop", Province: "广东省",
			PageviewsDelta: 3, EventsDelta: 3, IsNew: 1,
		},
		[]analyticsEventRow{
			{At: nowMs(), Day: day, Source: 0, SessionID: "s1", Visitor: "v1", Name: "pageview", View: "lobby"},
			{At: nowMs(), Day: day, Source: 1, Name: "game_round", Detail: "rps:win", Value: 1},
		},
	)

	ro, err := openAnalyticsReadOnlyDB(dir)
	if err != nil {
		t.Fatalf("ro: %v", err)
	}
	defer ro.Close()

	s := &Server{
		db: db, analyticsDB: store, analyticsRO: ro,
		analyticsTZOffsetMin: tz, analyticsEnabled: true,
		analyticsSalt: []byte("x"),
	}
	rows, err := s.rebuildDay(day, tz)
	if err != nil {
		t.Fatalf("rebuildDay: %v", err)
	}
	foundDAU := false
	foundGame := false
	for _, r := range rows {
		if r.Metric == metricDAU && r.Value >= 1 {
			foundDAU = true
		}
		if r.Metric == metricGameRound && r.Key == "rps" && r.Value >= 1 {
			foundGame = true
		}
	}
	if !foundDAU {
		t.Fatal("expected dau >= 1")
	}
	if !foundGame {
		t.Fatal("expected game_round metric")
	}
}

// TestRebuildDayNonIntegerAvgSessionMs 回归测试：SQLite 的 AVG(duration_ms)
// 结果是 REAL，一旦跨会话平均值带小数（如 1.5ms），rebuildDay 必须仍能正常
// 出结果，而不是在 Scan 阶段整体报错——历史上这里错用 sql.NullInt64 接
// float64 列，导致平均值非整数的那一天全部维度（含来源/省份/ISP）都聚合不出来。
func TestRebuildDayNonIntegerAvgSessionMs(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newAnalyticsStore(db)
	tz := 480
	day := analyticsDay(nowMs(), tz)
	base := nowMs() - 10_000
	// 两个会话时长分别为 1ms / 2ms，平均值 1.5ms 是非整数。
	for i, dur := range []int64{1, 2} {
		v := fmt.Sprintf("v%d", i)
		_ = store.writeBatch(
			analyticsVisitorRow{Visitor: v, FirstAt: base, FirstDay: day, LastAt: base + dur, SessionsDelta: 1},
			true,
			analyticsSessionRow{
				ID: fmt.Sprintf("s%d", i), Visitor: v, StartedAt: base, LastAt: base + dur, Day: day,
				Browser: "Chrome", OS: "Windows", DeviceType: "desktop",
				PageviewsDelta: 1, EventsDelta: 1, IsNew: 1,
			},
			nil,
		)
	}

	ro, err := openAnalyticsReadOnlyDB(dir)
	if err != nil {
		t.Fatalf("ro: %v", err)
	}
	defer ro.Close()

	s := &Server{
		db: db, analyticsDB: store, analyticsRO: ro,
		analyticsTZOffsetMin: tz, analyticsEnabled: true,
		analyticsSalt: []byte("x"),
	}
	rows, err := s.rebuildDay(day, tz)
	if err != nil {
		t.Fatalf("rebuildDay should tolerate non-integer AVG(duration_ms): %v", err)
	}
	found := false
	for _, r := range rows {
		if r.Metric == metricAvgSessionMs {
			found = true
			if r.Value != 1 { // int64(1.5)
				t.Fatalf("avg session ms = %d, want 1", r.Value)
			}
		}
	}
	if !found {
		t.Fatal("expected avg_session_ms row")
	}
}

func TestBackfillRetentionRefreshesRecentCohort(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newAnalyticsStore(db)
	ro, err := openAnalyticsReadOnlyDB(dir)
	if err != nil {
		t.Fatalf("openAnalyticsReadOnlyDB: %v", err)
	}
	defer ro.Close()

	today := analyticsDay(nowMs(), 480)
	if err := store.writeBatch(
		analyticsVisitorRow{Visitor: "v1", FirstAt: 1000, FirstDay: today, LastAt: 1000, SessionsDelta: 1},
		true,
		analyticsSessionRow{ID: "s1", Visitor: "v1", StartedAt: 1000, LastAt: 1000, Day: today, IsNew: 1},
		nil,
	); err != nil {
		t.Fatalf("write first session: %v", err)
	}
	s := &Server{analyticsDB: store, analyticsRO: ro}
	if err := s.backfillRetention(today, today, 480); err != nil {
		t.Fatalf("backfill first retention: %v", err)
	}

	if err := store.writeBatch(
		analyticsVisitorRow{Visitor: "v1", FirstAt: 1000, FirstDay: today, LastAt: 86_401_000, SessionsDelta: 1},
		true,
		analyticsSessionRow{ID: "s2", Visitor: "v1", StartedAt: 86_400_000, LastAt: 86_401_000, Day: today + 1},
		nil,
	); err != nil {
		t.Fatalf("write returning session: %v", err)
	}
	if err := s.backfillRetention(today, today+1, 480); err != nil {
		t.Fatalf("backfill refreshed retention: %v", err)
	}
	var retained int64
	if err := db.QueryRow(`SELECT value FROM analytics_daily WHERE day=? AND metric=? AND key=?`, today, metricRetention, fmt.Sprintf("%d:1", today)).Scan(&retained); err != nil {
		t.Fatalf("query D1 retention: %v", err)
	}
	if retained != 1 {
		t.Fatalf("D1 retention = %d, want 1", retained)
	}
}

func TestForRangeKPI(t *testing.T) {
	snap := &analyticsSnapshot{
		BuiltAt:      1,
		Days:         []int64{10, 11, 12, 13, 14, 15, 16},
		DAU:          int64Slice{1, 2, 3, 4, 5, 6, 7},
		Sessions:     int64Slice{10, 10, 10, 10, 10, 10, 10},
		Pageviews:    int64Slice{100, 100, 100, 100, 100, 100, 100},
		NewVisitors:  int64Slice{1, 0, 1, 0, 1, 0, 1},
		AvgSessionMs: int64Slice{1000, 1000, 1000, 1000, 1000, 1000, 1000},
		PeakOnline:   int64Slice{1, 2, 3, 2, 1, 2, 3},
		ByKey: map[string]map[string]int64Slice{
			metricDevice: {"desktop": {1, 1, 1, 1, 1, 1, 1}},
		},
		Retention: map[int64]map[int]int64{},
	}
	view := snap.forRange(7)
	if view.KPI.SessionsV != 70 {
		t.Fatalf("sessions sum = %d, want 70", view.KPI.SessionsV)
	}
	if len(view.Series.DAU) != 7 {
		t.Fatalf("series len = %d", len(view.Series.DAU))
	}
	if len(view.Devices) != 1 || view.Devices[0].Key != "desktop" {
		t.Fatalf("devices = %+v", view.Devices)
	}
}

func TestBuildSnapshotGameRoundByKey(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newAnalyticsStore(db)
	day := analyticsDay(nowMs(), 480)
	if err := store.upsertDaily([]analyticsDailyRow{
		{Day: day, Metric: metricGameRound, Key: "rps", Value: 5},
	}); err != nil {
		t.Fatalf("upsert daily: %v", err)
	}
	ro, err := openAnalyticsReadOnlyDB(dir)
	if err != nil {
		t.Fatalf("openAnalyticsReadOnlyDB: %v", err)
	}
	defer ro.Close()

	snap, err := (&Server{analyticsRO: ro}).buildSnapshot(480, 0, 0, 0)
	if err != nil {
		t.Fatalf("buildSnapshot: %v", err)
	}
	view := snap.forRange(7)
	if len(view.GameRounds) != 1 || view.GameRounds[0].Key != "rps" {
		t.Fatalf("game rounds = %+v", view.GameRounds)
	}
	if len(view.GameRounds[0].Values) != 1 || view.GameRounds[0].Values[0] != 5 {
		t.Fatalf("game round values = %+v, want [5]", view.GameRounds[0].Values)
	}
}

func TestOrderedBucketsDropsLegacyKeys(t *testing.T) {
	got := orderedBuckets([]analyticsBucket{{Key: "1-5m", Value: 4}}, []string{"0-1m", "1-2m"})
	if len(got) != 2 || got[0].Key != "0-1m" || got[0].Value != 0 || got[1].Key != "1-2m" || got[1].Value != 0 {
		t.Fatalf("legacy bucket should have been dropped, not merged into current order: %+v", got)
	}
}
