package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// seedV19DBWithWrongAnalyticsGeo 建一份停在 v19 的库，analytics_sessions/
// analytics_visitors/analytics_daily 里塞的是旧版 parseRegion 字段错位会写出的
// 典型错误值（province 存了城市名、isp 存了 iso-alpha2 国家码），country 保持不受影响
// 的正确值，用来验证 v20 只清错误列、不动 country。
func seedV19DBWithWrongAnalyticsGeo(t *testing.T) string {
	t.Helper()
	dirPath := t.TempDir()
	path := dirPath + "/database.db"

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE analytics_visitors (
			visitor        TEXT PRIMARY KEY,
			first_at       INTEGER NOT NULL,
			first_day      INTEGER NOT NULL,
			last_at        INTEGER NOT NULL,
			sessions       INTEGER NOT NULL DEFAULT 0,
			first_referrer TEXT NOT NULL DEFAULT '',
			first_province TEXT NOT NULL DEFAULT ''
		);
		CREATE TABLE analytics_sessions (
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
		CREATE TABLE analytics_daily (
			day    INTEGER NOT NULL,
			metric TEXT NOT NULL,
			key    TEXT NOT NULL DEFAULT '',
			value  INTEGER NOT NULL DEFAULT 0,
			sealed INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (day, metric, key)
		);
		INSERT INTO analytics_sessions
			(id, visitor, started_at, last_at, day, country, province, city, isp)
		VALUES
			('s1', 'v1', 1000, 2000, 19000, '中国', '南京市', '', 'CN');
		INSERT INTO analytics_visitors (visitor, first_at, first_day, last_at, first_province)
		VALUES ('v1', 1000, 19000, 2000, '南京市');
		INSERT INTO analytics_daily (day, metric, key, value, sealed) VALUES
			(19000, 'province', '南京市', 1, 1),
			(19000, 'isp', 'CN', 1, 1),
			(19000, 'city', '', 1, 1),
			(19000, 'dau', '', 5, 1);

		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (19);
	`); err != nil {
		t.Fatalf("seed v19 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}
	return dirPath
}

// TestSchemaMigrationV20ClearsWrongAnalyticsGeo 验证 v20 清空
// analytics_sessions.province/city/isp 与 analytics_visitors.first_province，
// 保留不受污染的 country，并删掉 analytics_daily 里错误聚合出的 province/city/isp
// 行（其它 metric，如 dau，不受影响）。
func TestSchemaMigrationV20ClearsWrongAnalyticsGeo(t *testing.T) {
	dir := seedV19DBWithWrongAnalyticsGeo(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v20 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	var country, province, city, isp string
	if err := db.QueryRow(`SELECT country, province, city, isp FROM analytics_sessions WHERE id='s1'`).
		Scan(&country, &province, &city, &isp); err != nil {
		t.Fatalf("query analytics_sessions: %v", err)
	}
	if country != "中国" {
		t.Fatalf("country should be untouched, got %q", country)
	}
	if province != "" || city != "" || isp != "" {
		t.Fatalf("expected province/city/isp cleared, got province=%q city=%q isp=%q", province, city, isp)
	}

	var firstProvince string
	if err := db.QueryRow(`SELECT first_province FROM analytics_visitors WHERE visitor='v1'`).Scan(&firstProvince); err != nil {
		t.Fatalf("query analytics_visitors: %v", err)
	}
	if firstProvince != "" {
		t.Fatalf("expected first_province cleared, got %q", firstProvince)
	}

	var geoRowCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM analytics_daily WHERE metric IN ('province','city','isp')`).Scan(&geoRowCount); err != nil {
		t.Fatalf("query analytics_daily: %v", err)
	}
	if geoRowCount != 0 {
		t.Fatalf("expected all province/city/isp analytics_daily rows deleted, got %d left", geoRowCount)
	}

	var dauValue int
	if err := db.QueryRow(`SELECT value FROM analytics_daily WHERE metric='dau' AND day=19000`).Scan(&dauValue); err != nil {
		t.Fatalf("expected unrelated dau row to survive: %v", err)
	}
	if dauValue != 5 {
		t.Fatalf("dau row value should be untouched, got %d", dauValue)
	}
}
