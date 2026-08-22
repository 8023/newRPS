package server

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// newContributionImageTestDB 打开一对写/只读连接指向同一份测试数据库，供
// contributionImageAccessible 用的 s.activityRO 查询——它要求库先以读写方式初始化过一次
// （建表/开 WAL），只读连接才能打开成功，见 openAnalyticsReadOnlyDB 的注释。
func newContributionImageTestDB(t *testing.T, dir string) (*sql.DB, *sql.DB) {
	t.Helper()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	ro, err := openAnalyticsReadOnlyDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = ro.Close() })
	return db, ro
}

// insertSubTaskWithImage 直接插入一行 sub_tasks（跳过完整的投稿/审核流程），只用来准备
// contributionImageAccessible 测试要用的 background_image 归属状态。
func insertSubTaskWithImage(t *testing.T, db *sql.DB, id string, version int, image string) {
	t.Helper()
	if _, err := db.Exec(`INSERT INTO sub_tasks (
		id, version, variants, status, background_image, contributor_player_id
	) VALUES (?, ?, '[]', 'draft', ?, 'p1')`, id, version, image); err != nil {
		t.Fatal(err)
	}
}

// TestRefererOrigin 验证从完整 Referer URL 提取 "scheme://host" 的行为，
// 与浏览器 Origin 请求头同构，供 isAllowedOrigin 复用。
func TestRefererOrigin(t *testing.T) {
	cases := []struct {
		in   string
		want string
	}{
		{"", ""},
		{"https://rps.rbq.io/room/abc?x=1", "https://rps.rbq.io"},
		{"http://127.0.0.1:5173/", "http://127.0.0.1:5173"},
		{"not a url", ""},
		{"about:blank", ""},
	}
	for _, c := range cases {
		if got := refererOrigin(c.in); got != c.want {
			t.Errorf("refererOrigin(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

// TestServeStaticUploadsRequiresAllowedReferer 覆盖 1.2 号发现的加固措施：证明图等上传目录
// 现在要求请求带着来自本站页面的 Referer，缺失或跨站 Referer 一律 404（与"文件不存在"同一
// 响应，不额外泄露信息），仅同源 Referer 才能实际读到文件内容。
func TestServeStaticUploadsRequiresAllowedReferer(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "secret.webp"), []byte("fake-image-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Server{host: "rps.rbq.io", uploadsDir: dir}

	// 无 Referer：拒绝。
	req := httptest.NewRequest("GET", "/uploads/secret.webp", nil)
	req.Host = "rps.rbq.io"
	rec := httptest.NewRecorder()
	s.serveStatic(rec, req)
	if rec.Code != 404 {
		t.Fatalf("missing referer: got status %d, want 404", rec.Code)
	}

	// 跨站 Referer：拒绝。
	req = httptest.NewRequest("GET", "/uploads/secret.webp", nil)
	req.Host = "rps.rbq.io"
	req.Header.Set("Referer", "https://evil.example/hotlink")
	rec = httptest.NewRecorder()
	s.serveStatic(rec, req)
	if rec.Code != 404 {
		t.Fatalf("cross-site referer: got status %d, want 404", rec.Code)
	}

	// 同站 Referer：放行，能读到内容。
	req = httptest.NewRequest("GET", "/uploads/secret.webp", nil)
	req.Host = "rps.rbq.io"
	req.Header.Set("Referer", "https://rps.rbq.io/room/abc")
	rec = httptest.NewRecorder()
	s.serveStatic(rec, req)
	if rec.Code != 200 {
		t.Fatalf("same-site referer: got status %d, want 200", rec.Code)
	}
	if rec.Body.String() != "fake-image-bytes" {
		t.Fatalf("unexpected body: %q", rec.Body.String())
	}
	// 非盈利小站优先省流量：Cache-Control 仍是 public（让 CDN/反代等共享缓存扛下大部分
	// 请求），但 max-age 从最初的 30 天收窄到 6 小时（多数房间的存活时长量级），把"链接
	// 泄露后经共享缓存仍可访问"的窗口大幅收窄；证明图另有 room-liveness 校验兜底
	// （见 TestServeStaticProofImageRequiresLiveRoom），两者互补，不是互相替代。
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "public") || !strings.Contains(cc, "max-age=21600") {
		t.Fatalf("Cache-Control = %q, want public with max-age=21600", cc)
	}
}

func TestServeStaticContributionCacheIsThirtyDays(t *testing.T) {
	dir := t.TempDir()
	contrib := filepath.Join(dir, "contributions")
	if err := os.MkdirAll(contrib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contrib, "bg.webp"), []byte("bg"), 0o600); err != nil {
		t.Fatal(err)
	}
	// sub_tasks 里必须有一行仍以这张图作为它最新版本的 background_image，否则「孤儿图不可
	// 访问」那道新增校验（见 contributionImageAccessible/serveStatic）会先于这里想测的
	// Referer/Cache-Control 逻辑把请求 404 掉。
	db, ro := newContributionImageTestDB(t, dir)
	insertSubTaskWithImage(t, db, "st_1", 1, "/uploads/contributions/bg.webp")
	s := &Server{host: "rps.rbq.io", uploadsDir: dir, activityRO: ro, contributionStore: newContributionStore(db)}
	for _, referer := range []string{"", "https://evil.example/hotlink"} {
		req := httptest.NewRequest("GET", "/uploads/contributions/bg.webp", nil)
		req.Host = "rps.rbq.io"
		if referer != "" {
			req.Header.Set("Referer", referer)
		}
		rec := httptest.NewRecorder()
		s.serveStatic(rec, req)
		if rec.Code != http.StatusNotFound {
			t.Fatalf("referer %q: status %d, want 404", referer, rec.Code)
		}
	}
	req := httptest.NewRequest("GET", "/uploads/contributions/bg.webp", nil)
	req.Host = "rps.rbq.io"
	req.Header.Set("Referer", "https://rps.rbq.io/")
	rec := httptest.NewRecorder()
	s.serveStatic(rec, req)
	if rec.Code != 200 {
		t.Fatalf("status %d", rec.Code)
	}
	if cc := rec.Header().Get("Cache-Control"); !strings.Contains(cc, "public") || !strings.Contains(cc, "max-age=2592000") || !strings.Contains(cc, "immutable") {
		t.Fatalf("Cache-Control = %q, want public max-age=2592000 immutable", cc)
	}
}

// TestServeStaticProofImageRequiresLiveRoom 覆盖"证明图所属房间已销毁则一律拒绝访问"这道
// 额外限制：即使 Referer 合法，s.proofImageRooms 记录的房间必须还在 s.rooms 里才放行；房间
// 被清理（cleanupRoomIfEmpty/管理员关房/优雅关停）或压根没有映射（如进程重启后，房间本就
// 不落盘）都视为"房间已销毁"，一律 404。只作用于 /uploads/proofs/，不影响 avatars/admin。
func TestServeStaticProofImageRequiresLiveRoom(t *testing.T) {
	dir := t.TempDir()
	proofsDir := filepath.Join(dir, "proofs")
	if err := os.MkdirAll(proofsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(proofsDir, "proof.webp"), []byte("proof-bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	s := &Server{
		host:            "rps.rbq.io",
		uploadsDir:      dir,
		rooms:           map[string]*RoomState{"room-1": {ID: "room-1"}},
		proofImageRooms: map[string]string{"proof.webp": "room-1"},
	}

	req := func() *http.Request {
		r := httptest.NewRequest("GET", "/uploads/proofs/proof.webp", nil)
		r.Host = "rps.rbq.io"
		r.Header.Set("Referer", "https://rps.rbq.io/room/room-1")
		return r
	}

	// 房间仍存活：放行。
	rec := httptest.NewRecorder()
	s.serveStatic(rec, req())
	if rec.Code != 200 {
		t.Fatalf("live room: got status %d, want 200", rec.Code)
	}

	// 房间已销毁（从 s.rooms 移除）：即使 Referer 合法也拒绝。
	delete(s.rooms, "room-1")
	rec = httptest.NewRecorder()
	s.serveStatic(rec, req())
	if rec.Code != 404 {
		t.Fatalf("destroyed room: got status %d, want 404", rec.Code)
	}

	// 压根没有映射（例如进程重启后 proofImageRooms 是空表）：同样拒绝。
	s2 := &Server{host: "rps.rbq.io", uploadsDir: dir, rooms: map[string]*RoomState{}, proofImageRooms: map[string]string{}}
	rec = httptest.NewRecorder()
	s2.serveStatic(rec, req())
	if rec.Code != 404 {
		t.Fatalf("no mapping: got status %d, want 404", rec.Code)
	}
}

// TestServeStaticContributionImageRequiresActiveRow 覆盖"共建封面图只在仍是它所属 id 最新
// 版本引用的图时才能访问，不看投稿审核状态"这道限制（见 CLAUDE.md 共建投稿一节）：只有
// 被重新上传覆盖替换掉、不再是任何 id 最新版本引用的孤儿图才 404，图片文件本身不因此被删除。
func TestServeStaticContributionImageRequiresActiveRow(t *testing.T) {
	dir := t.TempDir()
	contrib := filepath.Join(dir, "contributions")
	if err := os.MkdirAll(contrib, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(contrib, "bg.webp"), []byte("bg"), 0o600); err != nil {
		t.Fatal(err)
	}
	req := func() *http.Request {
		r := httptest.NewRequest("GET", "/uploads/contributions/bg.webp", nil)
		r.Host = "rps.rbq.io"
		r.Header.Set("Referer", "https://rps.rbq.io/")
		return r
	}
	db, ro := newContributionImageTestDB(t, dir)
	path := "/uploads/contributions/bg.webp"

	// 仍是这个 id 最新版本引用的图：放行——不管这份投稿是草稿、待审、已通过还是被
	// 驳回/撤回，状态完全不影响。
	insertSubTaskWithImage(t, db, "st_1", 1, path)
	s := &Server{host: "rps.rbq.io", uploadsDir: dir, activityRO: ro, contributionStore: newContributionStore(db)}
	rec := httptest.NewRecorder()
	s.serveStatic(rec, req())
	if rec.Code != 200 {
		t.Fatalf("active image: got status %d, want 200", rec.Code)
	}

	// 重新上传另一张图覆盖（新版本行不再引用旧图）：旧图变成孤儿，即使 Referer 合法也
	// 拒绝，文件本身仍在磁盘上。
	insertSubTaskWithImage(t, db, "st_1", 2, "/uploads/contributions/other.webp")
	rec = httptest.NewRecorder()
	s.serveStatic(rec, req())
	if rec.Code != 404 {
		t.Fatalf("orphaned image: got status %d, want 404", rec.Code)
	}
	if _, err := os.Stat(filepath.Join(contrib, "bg.webp")); err != nil {
		t.Fatalf("file must not be deleted: %v", err)
	}

	// 完全没有这行记录（未知路径）：同样拒绝。
	req2 := func() *http.Request {
		r := httptest.NewRequest("GET", "/uploads/contributions/unknown.webp", nil)
		r.Host = "rps.rbq.io"
		r.Header.Set("Referer", "https://rps.rbq.io/")
		return r
	}
	rec = httptest.NewRecorder()
	s.serveStatic(rec, req2())
	if rec.Code != 404 {
		t.Fatalf("unknown path: got status %d, want 404", rec.Code)
	}

	// activityRO 压根没打开（例如进程刚起步的边缘情况）：同样拒绝，不是放行。
	s3 := &Server{host: "rps.rbq.io", uploadsDir: dir}
	rec = httptest.NewRecorder()
	s3.serveStatic(rec, req())
	if rec.Code != 404 {
		t.Fatalf("no activityRO: got status %d, want 404", rec.Code)
	}
}
