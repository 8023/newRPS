package server

import (
	"database/sql"
	"fmt"
	"strings"
	"sync"
)

// analyticsSchema：用户分析四张表。
//   - analytics_visitors / analytics_sessions / analytics_events 是"原料"
//   - analytics_daily 是"半成品"，后台面板只读这一张
// 详见 plan.md 与 CLAUDE.md「用户分析」说明。
const analyticsSchema = `
CREATE TABLE IF NOT EXISTS analytics_visitors (
	visitor        TEXT PRIMARY KEY,
	first_at       INTEGER NOT NULL,
	first_day      INTEGER NOT NULL,
	last_at        INTEGER NOT NULL,
	sessions       INTEGER NOT NULL DEFAULT 0,
	first_referrer TEXT NOT NULL DEFAULT '',
	first_province TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_analytics_visitors_first_day ON analytics_visitors(first_day);
CREATE INDEX IF NOT EXISTS idx_analytics_visitors_last ON analytics_visitors(last_at);

CREATE TABLE IF NOT EXISTS analytics_sessions (
	id          TEXT PRIMARY KEY,
	visitor     TEXT NOT NULL,
	started_at  INTEGER NOT NULL,
	last_at     INTEGER NOT NULL,
	day         INTEGER NOT NULL,
	player_id   TEXT NOT NULL DEFAULT '',
	is_new      INTEGER NOT NULL DEFAULT 0,
	browser     TEXT NOT NULL DEFAULT '',
	os          TEXT NOT NULL DEFAULT '',
	device_type TEXT NOT NULL DEFAULT '',
	referrer    TEXT NOT NULL DEFAULT '',
	landing     TEXT NOT NULL DEFAULT '',
	country     TEXT NOT NULL DEFAULT '',
	province    TEXT NOT NULL DEFAULT '',
	city        TEXT NOT NULL DEFAULT '',
	isp         TEXT NOT NULL DEFAULT '',
	pageviews   INTEGER NOT NULL DEFAULT 0,
	events      INTEGER NOT NULL DEFAULT 0,
	duration_ms INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_analytics_sessions_day ON analytics_sessions(day);
CREATE INDEX IF NOT EXISTS idx_analytics_sessions_visitor ON analytics_sessions(visitor, started_at);
CREATE INDEX IF NOT EXISTS idx_analytics_sessions_last ON analytics_sessions(last_at);

CREATE TABLE IF NOT EXISTS analytics_events (
	seq        INTEGER PRIMARY KEY AUTOINCREMENT,
	at         INTEGER NOT NULL,
	day        INTEGER NOT NULL,
	source     INTEGER NOT NULL DEFAULT 0,
	session_id TEXT NOT NULL DEFAULT '',
	visitor    TEXT NOT NULL DEFAULT '',
	player_id  TEXT NOT NULL DEFAULT '',
	name       TEXT NOT NULL,
	view       TEXT NOT NULL DEFAULT '',
	detail     TEXT NOT NULL DEFAULT '',
	value      INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_analytics_events_day ON analytics_events(day, name);
CREATE INDEX IF NOT EXISTS idx_analytics_events_at ON analytics_events(at);
CREATE INDEX IF NOT EXISTS idx_analytics_events_session ON analytics_events(session_id, at);

CREATE TABLE IF NOT EXISTS analytics_daily (
	day    INTEGER NOT NULL,
	metric TEXT NOT NULL,
	key    TEXT NOT NULL DEFAULT '',
	value  INTEGER NOT NULL DEFAULT 0,
	sealed INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (day, metric, key)
);
CREATE INDEX IF NOT EXISTS idx_analytics_daily_metric ON analytics_daily(metric, day);
`

type analyticsStore struct {
	db *sql.DB
	mu sync.Mutex
}

func newAnalyticsStore(db *sql.DB) *analyticsStore {
	return &analyticsStore{db: db}
}

// analyticsVisitorRow / analyticsSessionRow / analyticsEventRow 是写路径入参。
type analyticsVisitorRow struct {
	Visitor       string
	FirstAt       int64
	FirstDay      int64
	LastAt        int64
	SessionsDelta int // 本次要加的会话数（通常 0 或 1）
	FirstReferrer string
	FirstProvince string
}

type analyticsSessionRow struct {
	ID             string
	Visitor        string
	StartedAt      int64
	LastAt         int64
	Day            int64
	PlayerID       string
	IsNew          int
	Browser        string
	OS             string
	DeviceType     string
	Referrer       string
	Landing        string
	Country        string
	Province       string
	City           string
	ISP            string
	PageviewsDelta int
	EventsDelta    int
}

type analyticsEventRow struct {
	At        int64
	Day       int64
	Source    int // 0=前端 1=服务端
	SessionID string
	Visitor   string
	PlayerID  string
	Name      string
	View      string
	Detail    string
	Value     int64
}

// insertEvents 一条多值 INSERT 写入原始事件。
func (a *analyticsStore) insertEvents(rows []analyticsEventRow) error {
	if a == nil || a.db == nil || len(rows) == 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var b strings.Builder
	b.WriteString(`INSERT INTO analytics_events (at, day, source, session_id, visitor, player_id, name, view, detail, value) VALUES `)
	args := make([]any, 0, len(rows)*10)
	for i, r := range rows {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`(?,?,?,?,?,?,?,?,?,?)`)
		args = append(args, r.At, r.Day, r.Source, r.SessionID, r.Visitor, r.PlayerID, r.Name, r.View, r.Detail, r.Value)
	}
	_, err := a.db.Exec(b.String(), args...)
	return err
}

// writeBatch 在一个事务里完成访客/会话/事件写入（后台消费者调用）。
func (a *analyticsStore) writeBatch(visitor analyticsVisitorRow, sessionNew bool, session analyticsSessionRow, events []analyticsEventRow) error {
	if a == nil || a.db == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// 访客：先尝试插入；已存在则只更新 last_at + sessions
	res, err := tx.Exec(
		`INSERT INTO analytics_visitors (visitor, first_at, first_day, last_at, sessions, first_referrer, first_province)
		 VALUES (?, ?, ?, ?, ?, ?, ?)
		 ON CONFLICT(visitor) DO UPDATE SET
			last_at = MAX(analytics_visitors.last_at, excluded.last_at),
			sessions = analytics_visitors.sessions + excluded.sessions`,
		visitor.Visitor, visitor.FirstAt, visitor.FirstDay, visitor.LastAt, visitor.SessionsDelta,
		visitor.FirstReferrer, visitor.FirstProvince,
	)
	if err != nil {
		return fmt.Errorf("visitor: %w", err)
	}
	// 若是新建访客且会话标记 is_new 尚未决定，由调用方在事务前判定；
	// 这里仅用 RowsAffected 信息——调用方已把 is_new 写进 session。
	_ = res

	// 会话：若本批是新会话首次出现，SessionsDelta 已在 visitor 里 +1
	_, err = tx.Exec(
		`INSERT INTO analytics_sessions (
			id, visitor, started_at, last_at, day, player_id, is_new,
			browser, os, device_type, referrer, landing,
			country, province, city, isp, pageviews, events, duration_ms
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, MAX(0, ? - ?))
		ON CONFLICT(id) DO UPDATE SET
			last_at = MAX(analytics_sessions.last_at, excluded.last_at),
			duration_ms = MAX(0, MAX(analytics_sessions.last_at, excluded.last_at) - analytics_sessions.started_at),
			pageviews = analytics_sessions.pageviews + excluded.pageviews,
			events = analytics_sessions.events + excluded.events,
			player_id = CASE WHEN analytics_sessions.player_id = '' THEN excluded.player_id ELSE analytics_sessions.player_id END`,
		session.ID, session.Visitor, session.StartedAt, session.LastAt, session.Day, session.PlayerID, session.IsNew,
		session.Browser, session.OS, session.DeviceType, session.Referrer, session.Landing,
		session.Country, session.Province, session.City, session.ISP,
		session.PageviewsDelta, session.EventsDelta, session.LastAt, session.StartedAt,
	)
	if err != nil {
		return fmt.Errorf("session: %w", err)
	}
	_ = sessionNew

	if len(events) > 0 {
		var b strings.Builder
		b.WriteString(`INSERT INTO analytics_events (at, day, source, session_id, visitor, player_id, name, view, detail, value) VALUES `)
		args := make([]any, 0, len(events)*10)
		for i, r := range events {
			if i > 0 {
				b.WriteByte(',')
			}
			b.WriteString(`(?,?,?,?,?,?,?,?,?,?)`)
			args = append(args, r.At, r.Day, r.Source, r.SessionID, r.Visitor, r.PlayerID, r.Name, r.View, r.Detail, r.Value)
		}
		if _, err := tx.Exec(b.String(), args...); err != nil {
			return fmt.Errorf("events: %w", err)
		}
	}
	return tx.Commit()
}

// deleteDailyMetricDays 删除某 metric 在 [startDay, endDay]（含）内的全部 EAV 行。
// 用于口径切换（如设备留存 → 用户留存）时清掉缺失 offset 上的陈旧格子。
func (a *analyticsStore) deleteDailyMetricDays(metric string, startDay, endDay int64) error {
	if a == nil || a.db == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(
		`DELETE FROM analytics_daily WHERE metric = ? AND day BETWEEN ? AND ?`,
		metric, startDay, endDay,
	)
	return err
}

// upsertDaily 批量 UPSERT 日聚合行（同一事务）。
func (a *analyticsStore) upsertDaily(rows []analyticsDailyRow) error {
	if a == nil || a.db == nil || len(rows) == 0 {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	tx, err := a.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(
		`INSERT INTO analytics_daily (day, metric, key, value, sealed)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(day, metric, key) DO UPDATE SET
			value = excluded.value,
			sealed = CASE WHEN analytics_daily.sealed = 1 THEN 1 ELSE excluded.sealed END`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, r := range rows {
		if _, err := stmt.Exec(r.Day, r.Metric, r.Key, r.Value, r.Sealed); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type analyticsDailyRow struct {
	Day    int64
	Metric string
	Key    string
	Value  int64
	Sealed int
}

// sealDays 把 day < beforeDay 且 sealed=0 的行定稿。
func (a *analyticsStore) sealDays(beforeDay int64) error {
	if a == nil || a.db == nil {
		return nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	_, err := a.db.Exec(`UPDATE analytics_daily SET sealed = 1 WHERE day < ? AND sealed = 0`, beforeDay)
	return err
}

// pruneRaw 分批删除过期原始事件/会话。返回本批删除总行数。
func (a *analyticsStore) pruneRaw(eventsBeforeDay, sessionsBeforeDay int64, limit int) (int64, error) {
	if a == nil || a.db == nil || limit <= 0 {
		return 0, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var total int64
	res, err := a.db.Exec(
		`DELETE FROM analytics_events WHERE seq IN (
			SELECT seq FROM analytics_events WHERE day < ? LIMIT ?
		)`, eventsBeforeDay, limit,
	)
	if err != nil {
		return 0, err
	}
	n, _ := res.RowsAffected()
	total += n
	res, err = a.db.Exec(
		`DELETE FROM analytics_sessions WHERE rowid IN (
			SELECT rowid FROM analytics_sessions WHERE day < ? LIMIT ?
		)`, sessionsBeforeDay, limit,
	)
	if err != nil {
		return total, err
	}
	n, _ = res.RowsAffected()
	total += n
	return total, nil
}

// sessionExists 判断会话是否已落库（用于判定是否新会话以累加 visitor.sessions）。
func (a *analyticsStore) sessionExists(id string) (bool, error) {
	if a == nil || a.db == nil || id == "" {
		return false, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var n int
	err := a.db.QueryRow(`SELECT 1 FROM analytics_sessions WHERE id = ? LIMIT 1`, id).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

// visitorExists 判断访客是否已存在。
func (a *analyticsStore) visitorExists(visitor string) (bool, error) {
	if a == nil || a.db == nil || visitor == "" {
		return false, nil
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	var n int
	err := a.db.QueryRow(`SELECT 1 FROM analytics_visitors WHERE visitor = ? LIMIT 1`, visitor).Scan(&n)
	if err == sql.ErrNoRows {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}
