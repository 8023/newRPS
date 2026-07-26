package server

import (
	"path/filepath"
	"testing"
	"time"
)

// newTestServer 返回一个字段齐全、可以安全触发 requestPersist 的 Server（写盘路径落在
// t.TempDir()，测试结束自动清理，不会污染工作目录）。
// lobby/room 广播防抖设为 1 小时：测试同步调用 handler 后立刻读写 s 的字段，若沿用生产环境
// 的短延迟，broadcastLobby/broadcastRoom 起的计时器协程会在测试主 goroutine 还没返回时就
// 触发、并发读写同一个 Server，在 -race 下报数据竞争（各 handler 本身持锁没问题，但测试代码
// 不持锁）；测试不需要真的等广播落地，拉长延迟等同于禁用。
func newTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	return &Server{
		players:              map[string]*PlayerState{},
		dataDir:              dir,
		playersFile:          filepath.Join(dir, "players.json"),
		deviceCreateAttempts: map[string][]int64{},
		ipCreateAttempts:     map[string][]int64{},
		rateBuckets:          map[string]*rateBucket{},
		lobbyBroadcastDelay:  time.Hour,
		roomBroadcastDelay:   time.Hour,
	}
}

func TestIsSameDevice(t *testing.T) {
	cases := []struct {
		recorded, incoming string
		want               bool
	}{
		{"deviceA", "deviceA", true},
		{"deviceA", "deviceB", false},
		{"", "", false}, // 空值不算"同设备"，宁可多确认一次也不要静默放行
		{"", "deviceB", false},
		{"deviceA", "", false},
	}
	for _, c := range cases {
		if got := isSameDevice(c.recorded, c.incoming); got != c.want {
			t.Errorf("isSameDevice(%q, %q) = %v, want %v", c.recorded, c.incoming, got, c.want)
		}
	}
}

func TestNeedsKickConfirm(t *testing.T) {
	cases := []struct {
		sameDevice, forceKick, want bool
	}{
		{sameDevice: true, forceKick: false, want: false}, // 同设备刷新：永不确认
		{sameDevice: true, forceKick: true, want: false},  // 同设备 + forceKick：依然不需要（forceKick 无害）
		{sameDevice: false, forceKick: false, want: true}, // 换设备、未确认：必须先确认
		{sameDevice: false, forceKick: true, want: false}, // 换设备、已确认：放行
	}
	for _, c := range cases {
		if got := needsKickConfirm(c.sameDevice, c.forceKick); got != c.want {
			t.Errorf("needsKickConfirm(%v, %v) = %v, want %v", c.sameDevice, c.forceKick, got, c.want)
		}
	}
}

func TestAddPlayerSecretSkipsActiveOldest(t *testing.T) {
	p := &PlayerState{}
	p.addPlayerSecret("a")
	p.addPlayerSecret("b")
	p.addPlayerSecret("c")
	p.ActiveSecret = "a" // a 是最早的，但正在使用，不能被挤
	ev := p.addPlayerSecret("d")
	if ev != "b" {
		t.Fatalf("want evict b (second oldest, since a is protected), got %q", ev)
	}
	if !p.hasPlayerSecret("a") || !p.hasPlayerSecret("c") || !p.hasPlayerSecret("d") {
		t.Fatalf("unexpected remaining secrets: %#v", p.PlayerSecrets)
	}
	if p.hasPlayerSecret("b") {
		t.Fatalf("b should have been evicted")
	}
	if len(p.PlayerSecrets) != maxPlayerSecrets {
		t.Fatalf("expected %d secrets, got %d", maxPlayerSecrets, len(p.PlayerSecrets))
	}
}

func TestAddPlayerSecretDedup(t *testing.T) {
	p := &PlayerState{}
	p.addPlayerSecret("a")
	ev := p.addPlayerSecret("a")
	if ev != "" {
		t.Fatalf("re-adding existing secret should not evict anything, got %q", ev)
	}
	if len(p.PlayerSecrets) != 1 {
		t.Fatalf("duplicate secret should not be appended twice: %#v", p.PlayerSecrets)
	}
}

func TestRemovePlayerSecret(t *testing.T) {
	p := &PlayerState{}
	p.addPlayerSecret("a")
	p.addPlayerSecret("b")
	p.removePlayerSecret("a")
	if p.hasPlayerSecret("a") {
		t.Fatalf("a should have been removed")
	}
	if !p.hasPlayerSecret("b") {
		t.Fatalf("b should remain")
	}
}

func TestVerifyPlayerSecretModernList(t *testing.T) {
	s := newTestServer(t)
	p := &PlayerState{}
	p.addPlayerSecret("modern-secret")
	if !s.verifyPlayerSecret(p, "modern-secret") {
		t.Fatalf("should verify against modern list")
	}
	if s.verifyPlayerSecret(p, "wrong") {
		t.Fatalf("should reject wrong secret")
	}
}

func TestVerifyPlayerSecretRejectsGarbage(t *testing.T) {
	s := newTestServer(t)
	p := &PlayerState{}
	p.addPlayerSecret("real")
	if s.verifyPlayerSecret(p, "fake") {
		t.Fatalf("should reject non-matching secret")
	}
	if s.verifyPlayerSecret(p, "") {
		t.Fatalf("should reject empty secret")
	}
}
