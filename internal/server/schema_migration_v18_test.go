package server

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/geoip"
	_ "github.com/mattn/go-sqlite3"
)

// repoXdbPathForTest 从当前包目录向上找仓库根下的 config/xdb/ip2region_v4.xdb；
// xdb 不入 git，本地未 fetch 时跳过（不能假设 CI/开发机一定有这个文件）。
func repoXdbPathForTest(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "config", "xdb", "ip2region_v4.xdb")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Skip("config/xdb/ip2region_v4.xdb not fetched locally, run npm run fetch-geoip")
	return ""
}

// seedV17DBWithConnectionEvents 建一份停在 v17（含 province/isp 空列）的库，塞几行
// connection_events：一条公网 IP（待回填）、一条内网 IP（geoip 会解析成"本地"，无省份/ISP，
// 不应被回填）、一条空 IP（不该被查表）。
func seedV17DBWithConnectionEvents(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

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
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip)
		VALUES ('public', 1000, 2000, 'disconnect', '114.114.114.114');
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip)
		VALUES ('private', 1000, 2000, 'disconnect', '192.168.1.1');
		INSERT INTO connection_events (socket_id, connected_at, disconnected_at, close_reason, ip)
		VALUES ('noip', 1000, 2000, 'disconnect', '');

		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (17);
	`); err != nil {
		t.Fatalf("seed v17 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}
	return dir
}

// provinceISPFor 读取一行的 province/isp。ALTER TABLE ADD COLUMN 补的历史行在未被回填时
// 是 SQL NULL（不是空字符串），所以扫进 sql.NullString 再摊平成 ""，不能直接 Scan 进 string。
func provinceISPFor(t *testing.T, db *sql.DB, socketID string) (province, isp string) {
	t.Helper()
	var p, i sql.NullString
	if err := db.QueryRow(`SELECT province, isp FROM connection_events WHERE socket_id = ?`, socketID).
		Scan(&p, &i); err != nil {
		t.Fatalf("query connection_events %s: %v", socketID, err)
	}
	return p.String, i.String
}

// TestSchemaMigrationV18BackfillsConnectionEventsGeo 验证 geoip 已启用时，v18 迁移会用
// connection_events 历史行的 ip 列回填 province/isp，且只回填一次（幂等门控见 schema_version）。
func TestSchemaMigrationV18BackfillsConnectionEventsGeo(t *testing.T) {
	xdbPath := repoXdbPathForTest(t)
	if err := geoip.Init(xdbPath); err != nil {
		t.Fatalf("geoip.Init: %v", err)
	}
	defer geoip.Disable()

	dir := seedV17DBWithConnectionEvents(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v18 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	if province, isp := provinceISPFor(t, db, "public"); province == "" && isp == "" {
		t.Fatalf("expected public IP row to be backfilled with province/isp, got province=%q isp=%q", province, isp)
	}
	if province, isp := provinceISPFor(t, db, "private"); province != "" || isp != "" {
		t.Fatalf("private IP row should stay empty (resolves to 本地, no province/isp), got province=%q isp=%q", province, isp)
	}
	if province, isp := provinceISPFor(t, db, "noip"); province != "" || isp != "" {
		t.Fatalf("empty-ip row should stay empty, got province=%q isp=%q", province, isp)
	}
}

// TestSchemaMigrationV18SkipsWhenGeoDisabled 验证 geoip 未启用（ANALYTICS_GEO_ENABLED=0
// 或 xdb 未加载）时，迁移直接跳过、不报错、不产生任何写入——版本号仍然照常升到最新。
func TestSchemaMigrationV18SkipsWhenGeoDisabled(t *testing.T) {
	geoip.Disable() // 确保没有前一个测试留下的全局状态

	dir := seedV17DBWithConnectionEvents(t)
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v18 migration cleanly even with geoip disabled: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	if province, isp := provinceISPFor(t, db, "public"); province != "" || isp != "" {
		t.Fatalf("geoip disabled: expected no backfill, got province=%q isp=%q", province, isp)
	}
}
