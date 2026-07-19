package server

import (
	"testing"

	"google.golang.org/protobuf/proto"

	"github.com/doumiao/newRPS/internal/wire"
)

func newRegistrationTestClient() *Client {
	return &Client{id: "c1", sid: "sid-1", ipAddress: "8.8.8.8", sendCh: make(chan []byte, 4)}
}

func lastReplyErr(t *testing.T, client *Client) string {
	t.Helper()
	select {
	case data := <-client.sendCh:
		var env wire.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			t.Fatalf("failed to decode reply: %v", err)
		}
		return env.Err
	default:
		t.Fatal("expected a reply on sendCh, got none")
		return ""
	}
}

// 攻击脚本批量注册时，管理员可以在后台一键勾选"禁止新用户注册"止血；
// 这里验证勾选后新玩家确实会被拒绝，且不会消耗设备/IP 的建号计数额度
// （避免误伤——开关关闭后这些额度应该还是满的，不会因为攻击期间的大量
// 被拒尝试而被提前用光）。
func TestOnPlayerJoinRejectsNewPlayerWhenRegistrationDisabled(t *testing.T) {
	s := newTestServer(t)
	s.cfg.AccessControl.MaxOnlinePerIP = 3
	s.cfg.AccessControl.MaxOnlinePerIPTotal = 10
	s.cfg.AccessControl.MaxCreatesPer10Min = 5
	s.cfg.AccessControl.MaxCreatesPerIP = 15
	s.cfg.AccessControl.RegistrationDisabled = true
	s.tokenToPlayer = map[string]string{}
	s.sidToPlayerID = map[string]string{}
	s.playerIdToID = map[string]string{}

	client := newRegistrationTestClient()
	s.onPlayerJoin(client, wsEnvelope{E: "player:join", ID: 1, D: map[string]any{"name": "新玩家"}})

	if got := lastReplyErr(t, client); got != "当前暂停新用户注册，请使用已有账号登录" {
		t.Fatalf("unexpected reply error: %q", got)
	}
	if len(s.players) != 0 {
		t.Fatalf("no player should have been created, got %d", len(s.players))
	}
	if len(s.deviceCreateAttempts) != 0 || len(s.ipCreateAttempts) != 0 {
		t.Fatal("rejected registration attempts must not consume the device/IP create-rate budget")
	}
}

func TestOnPlayerJoinAllowsNewPlayerWhenRegistrationEnabled(t *testing.T) {
	s := newTestServer(t)
	s.cfg.AccessControl.MaxOnlinePerIP = 3
	s.cfg.AccessControl.MaxOnlinePerIPTotal = 10
	s.cfg.AccessControl.MaxCreatesPer10Min = 5
	s.cfg.AccessControl.MaxCreatesPerIP = 15
	s.cfg.AccessControl.RegistrationDisabled = false
	s.tokenToPlayer = map[string]string{}
	s.sidToPlayerID = map[string]string{}
	s.playerIdToID = map[string]string{}

	client := newRegistrationTestClient()
	s.onPlayerJoin(client, wsEnvelope{E: "player:join", ID: 1, D: map[string]any{"name": "新玩家"}})

	if len(s.players) != 1 {
		t.Fatalf("expected exactly 1 player to be created, got %d", len(s.players))
	}
}
