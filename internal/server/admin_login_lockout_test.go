package server

import "testing"

func newAdminLockoutTestServer(t *testing.T, password string) *Server {
	t.Helper()
	s := newTestServer(t)
	s.cfg.Site.AdminPassword = password
	return s
}

func TestAdminPasswordMatchesSucceedsWithCorrectPassword(t *testing.T) {
	s := newAdminLockoutTestServer(t, "correct-horse")
	if !s.adminPasswordMatches("correct-horse", "1.2.3.4") {
		t.Fatal("correct password should be accepted")
	}
}

// TestAdminPasswordMatchesLocksOutAfterRepeatedFailures 连续猜错超过 adminLoginFailurePerIPLimit
// 次后（checkRateLimit 语义：第 Limit+1 次才真正触发冷却），同一 IP 即便随后发来正确口令也应该
// 被锁定拒绝，直到冷却期结束。
func TestAdminPasswordMatchesLocksOutAfterRepeatedFailures(t *testing.T) {
	s := newAdminLockoutTestServer(t, "correct-horse")
	ip := "1.2.3.4"
	for i := 0; i < adminLoginFailurePerIPLimit+1; i++ {
		if s.adminPasswordMatches("wrong", ip) {
			t.Fatalf("wrong password should never be accepted (attempt %d)", i)
		}
	}
	if s.adminPasswordMatches("correct-horse", ip) {
		t.Fatal("correct password should be rejected once the IP is locked out")
	}
}

// TestAdminPasswordMatchesCorrectAttemptsDontConsumeQuota 管理员在同一会话里反复携带正确口令
// 调用 config:save 等接口是正常使用场景，不应该被失败配额限制——只有猜错才计数。
func TestAdminPasswordMatchesCorrectAttemptsDontConsumeQuota(t *testing.T) {
	s := newAdminLockoutTestServer(t, "correct-horse")
	ip := "5.6.7.8"
	for i := 0; i < adminLoginFailurePerIPLimit*3; i++ {
		if !s.adminPasswordMatches("correct-horse", ip) {
			t.Fatalf("repeated correct password should keep succeeding (attempt %d)", i)
		}
	}
}

// TestAdminPasswordMatchesLockoutIsPerIP 一个 IP 被锁定不应该影响另一个 IP 的正常尝试
// （直到触发全局桶为止）。
func TestAdminPasswordMatchesLockoutIsPerIP(t *testing.T) {
	s := newAdminLockoutTestServer(t, "correct-horse")
	attacker := "9.9.9.9"
	for i := 0; i < adminLoginFailurePerIPLimit; i++ {
		s.adminPasswordMatches("wrong", attacker)
	}
	if !s.adminPasswordMatches("correct-horse", "10.10.10.10") {
		t.Fatal("a different IP should not be affected by another IP's lockout")
	}
}

// TestAdminPasswordMatchesGlobalLockout 分散在多个 IP 上的撞库尝试累计触发全局桶后，
// 即便是从未失败过的新 IP 也应该被临时锁定，防止绕开单 IP 限制的分布式爆破。
func TestAdminPasswordMatchesGlobalLockout(t *testing.T) {
	s := newAdminLockoutTestServer(t, "correct-horse")
	attemptsPerIP := adminLoginFailurePerIPLimit - 1
	ipCount := (adminLoginFailureGlobalLimit / attemptsPerIP) + 2
	for i := 0; i < ipCount; i++ {
		ip := "ip-" + string(rune('a'+i))
		for j := 0; j < attemptsPerIP; j++ {
			s.adminPasswordMatches("wrong", ip)
		}
	}
	if s.adminPasswordMatches("correct-horse", "brand-new-ip") {
		t.Fatal("global failure budget exhausted should lock out even a fresh IP")
	}
}
