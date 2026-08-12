package server

import (
	"fmt"
	"strings"
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

	const tz = 480
	today := analyticsDay(nowMs(), tz)
	offsetMs := int64(tz) * 60_000
	day0Start := today*86_400_000 - offsetMs
	createdAt := day0Start + 3_600_000

	// 用户 cohort：当日新建 player（id 与 analytics/connection 里的 player_id 一致）
	if _, err := db.Exec(
		`INSERT INTO players (id, player_id, name, created_at, last_seen_at) VALUES ('p1', 'pub-p1', '测试', ?, ?)`,
		createdAt, createdAt,
	); err != nil {
		t.Fatalf("insert player: %v", err)
	}

	// D0：桌面端会话（visitor v-pc）
	if err := store.writeBatch(
		analyticsVisitorRow{Visitor: "v-pc", FirstAt: createdAt, FirstDay: today, LastAt: createdAt, SessionsDelta: 1},
		true,
		analyticsSessionRow{
			ID: "s1", Visitor: "v-pc", StartedAt: createdAt, LastAt: createdAt + 1000,
			Day: today, PlayerID: "p1", IsNew: 1,
		},
		nil,
	); err != nil {
		t.Fatalf("write first session: %v", err)
	}
	s := &Server{analyticsDB: store, analyticsRO: ro}
	if err := s.backfillRetention(today, today, tz); err != nil {
		t.Fatalf("backfill first retention: %v", err)
	}
	var d0 int64
	if err := db.QueryRow(
		`SELECT value FROM analytics_daily WHERE day=? AND metric=? AND key=?`,
		today, metricRetention, fmt.Sprintf("%d:0", today),
	).Scan(&d0); err != nil {
		t.Fatalf("query D0 retention: %v", err)
	}
	if d0 != 1 {
		t.Fatalf("D0 retention = %d, want 1", d0)
	}

	// D1：换手机端（另一 visitor）以同一 player_id 登录 —— 设备口径会算流失，用户口径应记为回访
	day1At := day0Start + 86_400_000 + 3_600_000
	if err := store.writeBatch(
		analyticsVisitorRow{Visitor: "v-phone", FirstAt: day1At, FirstDay: today + 1, LastAt: day1At, SessionsDelta: 1},
		true,
		analyticsSessionRow{
			ID: "s2", Visitor: "v-phone", StartedAt: day1At, LastAt: day1At + 1000,
			Day: today + 1, PlayerID: "p1",
		},
		nil,
	); err != nil {
		t.Fatalf("write returning session: %v", err)
	}
	if err := s.backfillRetention(today, today+1, tz); err != nil {
		t.Fatalf("backfill refreshed retention: %v", err)
	}
	var retained int64
	if err := db.QueryRow(
		`SELECT value FROM analytics_daily WHERE day=? AND metric=? AND key=?`,
		today, metricRetention, fmt.Sprintf("%d:1", today),
	).Scan(&retained); err != nil {
		t.Fatalf("query D1 retention: %v", err)
	}
	if retained != 1 {
		t.Fatalf("D1 retention = %d, want 1 (same player across devices)", retained)
	}
}

// TestBackfillRetentionIgnoresDeviceOnlyVisitors 纯访客（未注册用户）不进入用户留存分母。
func TestBackfillRetentionIgnoresDeviceOnlyVisitors(t *testing.T) {
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

	const tz = 480
	today := analyticsDay(nowMs(), tz)
	if err := store.writeBatch(
		analyticsVisitorRow{Visitor: "bounce", FirstAt: 1000, FirstDay: today, LastAt: 1000, SessionsDelta: 1},
		true,
		analyticsSessionRow{ID: "sb", Visitor: "bounce", StartedAt: 1000, LastAt: 2000, Day: today, IsNew: 1},
		nil,
	); err != nil {
		t.Fatalf("write bounce session: %v", err)
	}
	s := &Server{analyticsDB: store, analyticsRO: ro}
	if err := s.backfillRetention(today, today, tz); err != nil {
		t.Fatalf("backfill retention: %v", err)
	}
	var n int
	if err := db.QueryRow(
		`SELECT COUNT(*) FROM analytics_daily WHERE day=? AND metric=?`, today, metricRetention,
	).Scan(&n); err != nil {
		t.Fatalf("count retention rows: %v", err)
	}
	if n != 0 {
		t.Fatalf("device-only visitors must not create account retention rows, got %d", n)
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

// TestRebuildDayPunishmentDoneUsesApproved 回归：惩罚「完成」必须按 status=approved
// 计数（eventstore 审核通过写的就是 approved），误用 done 会导致完成数恒为 0。
func TestRebuildDayPunishmentDoneUsesApproved(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newAnalyticsStore(db)
	events := newEventStore(db)
	tz := 480
	day := analyticsDay(nowMs(), tz)
	offsetMs := int64(tz) * 60_000
	// 落在本地「今天」中段，避免边界抖动
	taskAt := day*86_400_000 - offsetMs + 12*3_600_000

	// 发布 4 条：1 approved + 1 rejected + 1 pending + 1 assigned
	if err := events.insertPunishmentTask("t-approved", taskAt, "system", "r1", "", "", "p1", "甲", "任务A"); err != nil {
		t.Fatalf("insert approved task: %v", err)
	}
	if err := events.updatePunishmentProof("t-approved", taskAt+1000, "证明", "", "approved"); err != nil {
		t.Fatalf("proof approved: %v", err)
	}
	if err := events.insertPunishmentTask("t-rejected", taskAt+1, "player", "r1", "w", "胜", "p2", "乙", "任务B"); err != nil {
		t.Fatalf("insert rejected task: %v", err)
	}
	if err := events.updatePunishmentProof("t-rejected", taskAt+2000, "证明", "", "pending"); err != nil {
		t.Fatalf("proof pending: %v", err)
	}
	if err := events.markPunishmentRedo("t-rejected", "t-redo"); err != nil {
		t.Fatalf("mark redo: %v", err)
	}
	if err := events.insertPunishmentTask("t-pending", taskAt+2, "system", "r1", "", "", "p3", "丙", "任务C"); err != nil {
		t.Fatalf("insert pending task: %v", err)
	}
	if err := events.updatePunishmentProof("t-pending", taskAt+3000, "证明", "", "pending"); err != nil {
		t.Fatalf("proof pending: %v", err)
	}
	if err := events.insertPunishmentTask("t-assigned", taskAt+3, "system", "r1", "", "", "p4", "丁", "任务D"); err != nil {
		t.Fatalf("insert assigned task: %v", err)
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
		t.Fatalf("rebuildDay: %v", err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		switch r.Metric {
		case metricPunishPublish, metricPunishDone, metricPunishReject:
			got[r.Metric] = r.Value
		}
	}
	if got[metricPunishPublish] != 4 {
		t.Fatalf("publish = %d, want 4", got[metricPunishPublish])
	}
	if got[metricPunishDone] != 1 {
		t.Fatalf("done = %d, want 1 (status=approved only)", got[metricPunishDone])
	}
	if got[metricPunishReject] != 1 {
		t.Fatalf("reject = %d, want 1", got[metricPunishReject])
	}
}

// TestRebuildDayFunnelMetrics 转化漏斗五层均为设备 UV（player→visitor 映射）。
func TestRebuildDayFunnelMetrics(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newAnalyticsStore(db)
	events := newEventStore(db)
	tz := 480
	day := analyticsDay(nowMs(), tz)
	offsetMs := int64(tz) * 60_000
	at := day*86_400_000 - offsetMs + 12*3_600_000

	// 3 个访问设备；dev1/dev2 登录 p1/p2 并进大厅；dev3 只看登录页
	for _, row := range []struct {
		vis, sid, pid, view string
	}{
		{"dev1", "s1", "p1", "lobby"},
		{"dev2", "s2", "p2", "lobby"},
		{"dev3", "s3", "", "login"},
	} {
		if err := store.writeBatch(
			analyticsVisitorRow{Visitor: row.vis, FirstAt: at, LastAt: at, FirstDay: day, SessionsDelta: 1},
			true,
			analyticsSessionRow{
				ID: row.sid, Visitor: row.vis, StartedAt: at, LastAt: at, Day: day,
				PlayerID: row.pid, PageviewsDelta: 1, EventsDelta: 1,
			},
			[]analyticsEventRow{
				{At: at, Day: day, Source: 0, SessionID: row.sid, Visitor: row.vis, PlayerID: row.pid, Name: "pageview", View: row.view},
			},
		); err != nil {
			t.Fatalf("session %s: %v", row.sid, err)
		}
	}

	// p1 建房 + p2 进房 → 设备 UV = 2（dev1, dev2）
	if err := events.insertRoomEvent(roomEventInput{
		At: at, RoomID: "r1", RoomName: "房", GameID: "rps",
		UserID: "p1", UserName: "甲", Action: "create",
	}); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := events.insertRoomEvent(roomEventInput{
		At: at + 2, RoomID: "r1", RoomName: "房", GameID: "rps",
		UserID: "p2", UserName: "乙", Action: "join", Role: "战斗席 B",
	}); err != nil {
		t.Fatalf("join: %v", err)
	}

	// 开局/结算：双方各一条；finish 副记录 value=0 仍计入设备 UV
	if err := store.insertEvents([]analyticsEventRow{
		{At: at, Day: day, Source: 1, Name: "game_start", Detail: "rps", Value: 1, View: "r1", PlayerID: "p1"},
		{At: at, Day: day, Source: 1, Name: "game_start", Detail: "rps", Value: 1, View: "r1", PlayerID: "p2"},
		{At: at + 2, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", Value: 1, View: "r1", PlayerID: "p1"},
		{At: at + 2, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", Value: 0, View: "r1", PlayerID: "p2"},
	}); err != nil {
		t.Fatalf("game events: %v", err)
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
		t.Fatalf("rebuildDay: %v", err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		if r.Metric == metricFunnel {
			got[r.Key] = r.Value
		}
	}
	if got["visit"] != 3 {
		t.Fatalf("visit = %d, want 3 devices", got["visit"])
	}
	if got["lobby"] != 2 {
		t.Fatalf("lobby = %d, want 2", got["lobby"])
	}
	if got["room"] != 2 {
		t.Fatalf("room = %d, want 2 device UV", got["room"])
	}
	if got["round"] != 2 {
		t.Fatalf("round(开局) = %d, want 2 device UV", got["round"])
	}
	if got["finish"] != 2 {
		t.Fatalf("finish(完成对局) = %d, want 2 device UV", got["finish"])
	}
	if _, ok := got["login"]; ok {
		t.Fatal("legacy funnel key login should not be written")
	}
	if _, ok := got["punish_done"]; ok {
		t.Fatal("legacy funnel key punish_done should not be written")
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

func TestMergeBucketAliasProofMsLegacy(t *testing.T) {
	// 旧 20min / 20min+ 应并入 10min+，再与现有 10min+ 累加。
	merged := mergeBucketAlias([]analyticsBucket{
		{Key: "10min", Value: 3},
		{Key: "20min", Value: 2},
		{Key: "20min+", Value: 1},
		{Key: "10min+", Value: 4},
	}, map[string]string{"20min": "10min+", "20min+": "10min+"})
	got := orderedBuckets(merged, []string{"10s", "30s", "1min", "5min", "10min", "10min+"})
	want := map[string]int64{"10min": 3, "10min+": 7}
	for _, b := range got {
		if b.Value != want[b.Key] {
			t.Fatalf("key %s = %d, want %d (full=%+v)", b.Key, b.Value, want[b.Key], got)
		}
	}
}

func TestRebuildDayRoomRoundsAndChatSpeakers(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	store := newAnalyticsStore(db)
	tz := 480
	now := nowMs()
	day := analyticsDay(now, tz)
	// roomA 3 局，roomB 1 局 → max=3, avg=2；无 view 的旧事件不计入。
	_ = store.insertEvents([]analyticsEventRow{
		{At: now, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", View: "roomA", Value: 1},
		{At: now, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", View: "roomA", Value: 1},
		{At: now, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", View: "roomA", Value: 1},
		{At: now, Day: day, Source: 1, Name: "game_round", Detail: "othello:win", View: "roomB", Value: 1},
		{At: now, Day: day, Source: 1, Name: "game_round", Detail: "rps:win", View: "", Value: 1},
	})

	// 大厅 2 人发言，房间 3 人发言（其中 1 人跨两房间，去重后仍 3）
	if _, err := db.Exec(`INSERT INTO lobby_messages (id, player_id, author, author_role, text, mentions, at) VALUES
		('m1','p1','a','','hi','',?),
		('m2','p2','b','','yo','',?),
		('m3','p1','a','','again','',?)`, now, now, now); err != nil {
		t.Fatalf("seed lobby: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO room_messages (room_id, id, player_id, author, author_role, text, mentions, at) VALUES
		('r1','rm1','p1','a','','hi','',?),
		('r1','rm2','p3','c','','yo','',?),
		('r2','rm3','p4','d','','zz','',?),
		('r2','rm4','p1','a','','again','',?)`, now, now, now, now); err != nil {
		t.Fatalf("seed room chat: %v", err)
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
		t.Fatalf("rebuildDay: %v", err)
	}
	got := map[string]int64{}
	for _, r := range rows {
		if r.Key == "" {
			got[r.Metric] = r.Value
		}
	}
	if got[metricRoomRoundsMax] != 3 {
		t.Fatalf("room_rounds_max = %d, want 3", got[metricRoomRoundsMax])
	}
	if got[metricRoomRoundsAvg] != 2 {
		t.Fatalf("room_rounds_avg = %d, want 2", got[metricRoomRoundsAvg])
	}
	if got[metricChatSpeakers] != 2 {
		t.Fatalf("chat_speakers (lobby) = %d, want 2", got[metricChatSpeakers])
	}
	if got[metricChatRoomSpeakers] != 3 {
		t.Fatalf("chat_room_speakers = %d, want 3", got[metricChatRoomSpeakers])
	}
}

func TestSanitizeAnalyticsRoomID(t *testing.T) {
	if got := sanitizeAnalyticsRoomID("ab-cd_EF09"); got != "ab-cd_EF09" {
		t.Fatalf("valid id rejected: %q", got)
	}
	if got := sanitizeAnalyticsRoomID("bad id!"); got != "" {
		t.Fatalf("invalid id accepted: %q", got)
	}
	if got := sanitizeAnalyticsRoomID(strings.Repeat("a", 65)); got != "" {
		t.Fatalf("too long id accepted")
	}
}
