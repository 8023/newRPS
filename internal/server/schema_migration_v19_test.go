package server

import (
	"database/sql"
	"testing"

	"github.com/doumiao/newRPS/internal/geoip"
	_ "github.com/mattn/go-sqlite3"
)

// seedV18DBWithWrongGeo 建一份停在 v18 的库，province/isp 列里塞的是 v18 那版
// parseRegion 字段错位会实际写出的错误值（把「城市」当「省份」、把 iso-alpha2-code
// 当「ISP」），用来验证 v19 会无条件覆盖修正，而不是被"已有值非空"挡住。
func seedV18DBWithWrongGeo(t *testing.T) string {
	t.Helper()
	dirPath := t.TempDir()
	path := dirPath + "/database.db"

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE connection_events (
			seq             INTEGER PRIMARY KEY AUTOINCREMENT,
			socket_id       TEXT NOT NULL,
			connected_at    INTEGER NOT NULL,
			session_sid     TEXT,
			ip              TEXT,
			device          TEXT,
			fingerprint     TEXT,
			user_agent      TEXT,
			compression     TEXT,
			player_id       TEXT,
			disconnected_at INTEGER NOT NULL,
			close_reason    TEXT NOT NULL,
			province        TEXT,
			isp             TEXT
		);
		-- 114.114.114.114 真实字段是 中国|江苏省|南京市|0|CN：v18 错误地写成
		-- province=南京市（本应是城市）、isp=CN（本应是 iso 码，真实 ISP 字段是占位符 0）。
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip, province, isp)
		VALUES ('dns114', 1000, 2000, 'disconnect', '114.114.114.114', '南京市', 'CN');
		-- 202.96.134.133 真实字段是 中国|广东省|深圳市|电信|CN：v18 错误地写成
		-- province=深圳市、isp=CN，真实值应为 province=广东省、isp=电信。
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip, province, isp)
		VALUES ('telecom', 1000, 2000, 'disconnect', '202.96.134.133', '深圳市', 'CN');
		-- 内网 IP：v18/v19 都应保持 province/isp 为空。
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip, province, isp)
		VALUES ('private', 1000, 2000, 'disconnect', '192.168.1.1', '', '');

		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (18);
	`); err != nil {
		t.Fatalf("seed v18 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}
	return dirPath
}

// TestSchemaMigrationV19FixesWrongGeoFromV18 验证 v19 迁移会无条件覆盖 v18 写入的
// 错误 province/isp（旧值非空，不会被"仍为空"的判断选中，必须无条件重新解析覆盖）。
func TestSchemaMigrationV19FixesWrongGeoFromV18(t *testing.T) {
	xdbPath := repoXdbPathForTest(t)
	if err := geoip.Init(xdbPath); err != nil {
		t.Fatalf("geoip.Init: %v", err)
	}
	defer geoip.Disable()

	dir := seedV18DBWithWrongGeo(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v19 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	if province, isp := provinceISPFor(t, db, "dns114"); province != "江苏省" || isp != "" {
		t.Fatalf("expected dns114 row fixed to province=江苏省 isp=(empty), got province=%q isp=%q", province, isp)
	}
	if province, isp := provinceISPFor(t, db, "telecom"); province != "广东省" || isp != "电信" {
		t.Fatalf("expected telecom row fixed to province=广东省 isp=电信, got province=%q isp=%q", province, isp)
	}
	if province, isp := provinceISPFor(t, db, "private"); province != "" || isp != "" {
		t.Fatalf("private IP row should stay empty, got province=%q isp=%q", province, isp)
	}
}

// TestSchemaMigrationV19SkipsWhenGeoDisabled 验证 geoip 未启用时 v19 直接跳过、
// 不动已有数据、不报错，版本号仍照常升到最新。
func TestSchemaMigrationV19SkipsWhenGeoDisabled(t *testing.T) {
	geoip.Disable()

	dir := seedV18DBWithWrongGeo(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v19 migration cleanly even with geoip disabled: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	// geoip 禁用时不重新解析，旧（错误）值原样保留。
	if province, isp := provinceISPFor(t, db, "dns114"); province != "南京市" || isp != "CN" {
		t.Fatalf("geoip disabled: expected untouched old value province=南京市 isp=CN, got province=%q isp=%q", province, isp)
	}
}
