package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestShouldPush(t *testing.T) {
	on := boolPtr(true)
	off := boolPtr(false)
	connected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: true}}
	disconnected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: false}}

	cases := []struct {
		name   string
		player *PlayerState
		pref   *bool
		want   bool
	}{
		{"nil player", nil, on, false},
		{"connected, pref on", connected, on, true},
		{"disconnected, pref on", disconnected, on, true},
		{"disconnected, pref off", disconnected, off, false},
		{"disconnected, pref nil", disconnected, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldPush(c.player, c.pref); got != c.want {
				t.Errorf("shouldPush() = %v, want %v", got, c.want)
			}
		})
	}
}

// TestIsAllowedPushEndpoint 校验 push:subscribe 的 SSRF 防护：只有已知推送网关域名的 https
// 地址才被接受，内网/回环地址、非 https、以及貌似合法实则不匹配后缀的域名都要拒绝。
func TestIsAllowedPushEndpoint(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"fcm https", "https://fcm.googleapis.com/fcm/send/abc123", true},
		{"android googleapis https", "https://android.googleapis.com/gcm/send/abc123", true},
		{"mozilla https", "https://updates.push.services.mozilla.com/wpush/v2/abc123", true},
		{"apple https", "https://web.push.apple.com/abc123", true},
		{"windows notify https", "https://sn1.notify.windows.com/abc123", true},
		{"samsung osp https", "https://push.samsungosp.com/spp/v1/send/abc123", true},
		{"samsung osp subdomain", "https://eu.push.samsungosp.com/spp/v1/send/x", true},
		{"non-https fcm", "http://fcm.googleapis.com/fcm/send/abc123", false},
		{"loopback", "https://127.0.0.1/steal", false},
		{"loopback host name", "https://localhost:9988/api/session", false},
		{"private ip", "https://10.0.0.5/internal", false},
		{"link-local metadata", "https://169.254.169.254/latest/meta-data", false},
		{"lookalike suffix without dot", "https://evilgoogleapis.com/x", false},
		{"lookalike suffix appended as subdomain of attacker domain", "https://googleapis.com.evil.com/x", false},
		{"empty", "", false},
		{"garbage", "not a url", false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := isAllowedPushEndpoint(c.url); got != c.want {
				t.Errorf("isAllowedPushEndpoint(%q) = %v, want %v", c.url, got, c.want)
			}
		})
	}
}

// TestOnPushSubscribeRejectsSSRFEndpoint push:subscribe 直接把内网地址当 endpoint 提交时
// 必须在写库之前就被拒绝——否则后续 deliverPush/push:test 会对着这个地址发起真实出网请求，
// 等于把服务端变成一个可控的 SSRF 探测器。
func TestOnPushSubscribeRejectsSSRFEndpoint(t *testing.T) {
	s := newTestServer(t)
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1"}, SocketID: "sock1"}
	s.players["p1"] = player
	client := &Client{id: "sock1", playerID: "p1", sendCh: make(chan []byte, 4)}

	env := wsEnvelope{E: "push:subscribe", ID: 1, D: map[string]any{
		"endpoint": "https://127.0.0.1:9988/api/session",
		"keys":     map[string]any{"p256dh": "key1", "auth": "auth1"},
	}}
	s.onPushSubscribe(client, env)
	errMsg := lastReplyError(t, client)
	if errMsg != "不支持的推送服务地址" {
		t.Fatalf("SSRF endpoint should be rejected, got err=%q", errMsg)
	}
}

func TestLoadOrGenerateVAPIDKeysReportsPersistenceFailure(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	root := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(root, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocking file: %v", err)
	}

	keys, err := loadOrGenerateVAPIDKeys(root)
	if err == nil {
		t.Fatalf("expected persistence error, got keys public=%t private=%t", keys.PublicKey != "", keys.PrivateKey != "")
	}
}

func TestLoadOrGenerateVAPIDKeysRejectsPartialEnvironment(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "public-only")
	t.Setenv("VAPID_PRIVATE_KEY", "")

	if _, err := loadOrGenerateVAPIDKeys(t.TempDir()); err == nil {
		t.Fatal("expected an error for a partial VAPID environment configuration")
	}
}

func TestLoadOrGenerateVAPIDKeysRejectsCorruptPersistedKeys(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create work directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "vapid.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt key file: %v", err)
	}

	if _, err := loadOrGenerateVAPIDKeys(root); err == nil {
		t.Fatal("expected an error for corrupt persisted VAPID keys")
	}
}

func TestHandlePushVapidKeyReportsProtocolVersion(t *testing.T) {
	s := &Server{vapid: vapidKeys{PublicKey: "test-public-key"}}
	req := httptest.NewRequest(http.MethodGet, "/api/push/vapid-key", nil)
	rec := httptest.NewRecorder()

	s.handlePushVapidKey(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusOK)
	}
	var body struct {
		PublicKey       string `json:"publicKey"`
		ProtocolVersion int    `json:"protocolVersion"`
	}
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.PublicKey != "test-public-key" || body.ProtocolVersion != pushProtocolVersion {
		t.Fatalf("unexpected response: %+v", body)
	}
	if got := rec.Header().Get("Cache-Control"); got != "no-store" {
		t.Fatalf("Cache-Control = %q, want no-store", got)
	}
}

func TestPushEventsAreRegistered(t *testing.T) {
	s := &Server{}
	for _, event := range []string{
		"push:subscribe", "push:unsubscribe", "push:getPreferences", "push:updatePreferences", "push:test",
	} {
		t.Run(event, func(t *testing.T) {
			if _, handler := s.eventHandler(event); handler == nil {
				t.Fatalf("event %q is not registered", event)
			}
		})
	}
}

func TestRemoveSubscriptionForPlayerRequiresOwnership(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	store := newPushStore(db)
	if err := store.upsertSubscription("player-a", "https://push.example/a", "k1", "a1", 1); err != nil {
		t.Fatalf("upsert: %v", err)
	}
	if err := store.upsertSubscription("player-b", "https://push.example/b", "k2", "a2", 2); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	// 玩家 B 不能删掉玩家 A 的 endpoint。
	if err := store.removeSubscriptionForPlayer("player-b", "https://push.example/a"); err != nil {
		t.Fatalf("removeSubscriptionForPlayer: %v", err)
	}
	subsA, err := store.subscriptionsForPlayer("player-a")
	if err != nil {
		t.Fatalf("subscriptionsForPlayer A: %v", err)
	}
	if len(subsA) != 1 {
		t.Fatalf("player-a subscriptions = %d, want 1 (ownership must block delete)", len(subsA))
	}

	// 玩家 A 只能删自己的。
	if err := store.removeSubscriptionForPlayer("player-a", "https://push.example/a"); err != nil {
		t.Fatalf("remove own subscription: %v", err)
	}
	subsA, err = store.subscriptionsForPlayer("player-a")
	if err != nil {
		t.Fatalf("subscriptionsForPlayer A after delete: %v", err)
	}
	if len(subsA) != 0 {
		t.Fatalf("player-a subscriptions = %d, want 0", len(subsA))
	}
	subsB, err := store.subscriptionsForPlayer("player-b")
	if err != nil {
		t.Fatalf("subscriptionsForPlayer B: %v", err)
	}
	if len(subsB) != 1 {
		t.Fatalf("player-b subscriptions = %d, want 1", len(subsB))
	}

	// 无归属的失效清理仍可按 endpoint 删除。
	if err := store.removeSubscription("https://push.example/b"); err != nil {
		t.Fatalf("removeSubscription: %v", err)
	}
	subsB, err = store.subscriptionsForPlayer("player-b")
	if err != nil {
		t.Fatalf("subscriptionsForPlayer B after gateway cleanup: %v", err)
	}
	if len(subsB) != 0 {
		t.Fatalf("player-b subscriptions = %d, want 0", len(subsB))
	}
}

func TestPushDeliveryClientError(t *testing.T) {
	cases := []struct {
		name   string
		result pushDeliveryResult
		want   string
	}{
		{"accepted", pushDeliveryResult{Accepted: 1}, ""},
		{"expired", pushDeliveryResult{Failed: 1, Expired: 1}, "浏览器推送订阅已经失效，请重新订阅这台设备"},
		{"missing roots", pushDeliveryResult{Failed: 1, LastError: "x509: certificate signed by unknown authority"}, "服务器无法验证推送网关的 TLS 证书（缺少 CA 根证书）"},
		{"rejected", pushDeliveryResult{Failed: 1, LastError: "HTTP 403"}, "浏览器推送网关拒绝了通知（HTTP 403）"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := pushDeliveryClientError(c.result); got != c.want {
				t.Fatalf("pushDeliveryClientError() = %q, want %q", got, c.want)
			}
		})
	}
}
