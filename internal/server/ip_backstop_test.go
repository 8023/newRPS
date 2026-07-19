package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// 这些测试覆盖"攻击脚本每次换指纹/换 sid 绕过按设备限流"的加固：纯 IP 维度的兜底桶。

func newAccessControlTestServer(t *testing.T) *Server {
	t.Helper()
	s := newTestServer(t)
	s.cfg.AccessControl.MaxOnlinePerIP = 3
	s.cfg.AccessControl.MaxCreatesPer10Min = 5
	s.cfg.AccessControl.IPBackstopMultiplier = 3
	s.cfg.AccessControl.IPBackstopMinLimit = 10
	s.cfg.AccessControl.MaxSessionIssuePerIP = 30
	s.cfg.AccessControl.MaxOnlinePerIPTotal = 2
	s.cfg.AccessControl.MaxCreatesPerIP = 3
	s.cfg.AccessControl.MaxActiveRoomsPerOwner = 2
	s.cfg.AccessControl.MaxProofUploadsPerPlayer = 8
	return s
}

func TestCheckIPEventBackstopBlocksHighVolumeFromOneIP(t *testing.T) {
	s := newAccessControlTestServer(t)
	opts := RateLimitOptions{Limit: 5, WindowMs: 10_000, CooldownMs: 10_000}
	// multiplier=3 -> 15，高于 minLimit=10，因此桶容量应为 15。
	for i := 0; i < 15; i++ {
		if !s.checkIPEventBackstop("room:move", "9.9.9.9", opts) {
			t.Fatalf("request %d should still pass (limit=15)", i)
		}
	}
	if s.checkIPEventBackstop("room:move", "9.9.9.9", opts) {
		t.Fatal("16th request from same IP should be blocked by the backstop bucket")
	}
	// 换一个不同 sid 的调用方（不同 key 维度）无法绕过这层纯 IP 桶：同一 event+ip 仍被卡住。
	if s.checkIPEventBackstop("room:move", "9.9.9.9", opts) {
		t.Fatal("backstop bucket must stay blocked regardless of caller identity")
	}
	// 换一个不同 IP 则完全独立，不受影响。
	if !s.checkIPEventBackstop("room:move", "1.1.1.1", opts) {
		t.Fatal("a different IP must have its own independent bucket")
	}
}

func TestCheckIPEventBackstopRespectsMinLimit(t *testing.T) {
	s := newAccessControlTestServer(t)
	// Limit=1 * multiplier(3) = 3，低于 minLimit=10，应取 10。
	opts := RateLimitOptions{Limit: 1, WindowMs: 10_000, CooldownMs: 10_000}
	for i := 0; i < 10; i++ {
		if !s.checkIPEventBackstop("room:create", "2.2.2.2", opts) {
			t.Fatalf("request %d should pass under the min-limit floor of 10", i)
		}
	}
	if s.checkIPEventBackstop("room:create", "2.2.2.2", opts) {
		t.Fatal("11th request should be blocked once the min-limit floor is reached")
	}
}

func TestOnlineAndCreateFromIPIgnoresFingerprintRotation(t *testing.T) {
	s := newAccessControlTestServer(t)
	ip := "5.5.5.5"
	s.players["p1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Connected: true}, IPAddress: ip}
	s.players["p2"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p2", Connected: true}, IPAddress: ip}
	// 即使这两个玩家用的是完全不同的 deviceKey（不同指纹），纯 IP 计数仍应看到 2 个在线。
	if got := s.onlinePlayersFromIP(ip, ""); got != 2 {
		t.Fatalf("onlinePlayersFromIP = %d, want 2", got)
	}
	if got := s.onlinePlayersFromIP(ip, "p1"); got != 1 {
		t.Fatalf("onlinePlayersFromIP excluding p1 = %d, want 1", got)
	}

	// MaxCreatesPerIP = 3：同一 IP 无论 fingerprint 怎么换，第 4 次新建应被拒绝。
	for i := 0; i < 3; i++ {
		if !s.canCreateFromIP(ip) {
			t.Fatalf("create attempt %d should be allowed", i)
		}
	}
	if s.canCreateFromIP(ip) {
		t.Fatal("4th create attempt from the same IP should be rejected regardless of fingerprint")
	}
	// 另一个 IP 完全独立。
	if !s.canCreateFromIP("6.6.6.6") {
		t.Fatal("a different IP must have its own independent create-attempt counter")
	}
}
