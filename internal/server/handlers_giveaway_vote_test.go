package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

// setupGiveawayVoteServer 造一个 actor + target 玩家，target 已上白给自救板，
// actor 通过 sock-actor 连接。调用方按需再改字段（比如调小配额触发上限分支）。
func setupGiveawayVoteServer(t *testing.T) (*Server, *Client) {
	t.Helper()
	s := newTestServer(t)
	s.cfg.Giveaway.LikeVoteLimitPerHour = 3
	s.cfg.Giveaway.LikeVoteValue = 1
	s.cfg.Giveaway.DislikeVoteLimitPerHour = 10
	s.cfg.Giveaway.DislikeVoteValue = 0.1

	now := nowMs()
	target := &PlayerState{PublicPlayer: types.PublicPlayer{
		ID:                     "target1",
		Name:                   "自救玩家",
		GiveawayEnabled:        boolPtr(true),
		GiveawayValue:          floatPtr(50),
		GiveawayBoardText:      "求放过",
		GiveawayBoardExpiresAt: int64Ptr(now + 3_600_000),
	}}
	target.GiveawayBoardSubmittedAt = int64Ptr(now)
	actor := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "actor1", Name: "投票玩家"}, SocketID: "sock-actor"}
	s.players["target1"] = target
	s.players["actor1"] = actor

	client := &Client{id: "sock-actor", playerID: "actor1", sendCh: make(chan []byte, 8)}
	return s, client
}

// TestOnGiveawayVoteRejectsReplayOnSameBoard 复现"抓包重放"刷赞/刷踩：同一个 actor 对
// 同一条自救板重复发送完全相同的 giveaway:vote 请求，第二次起必须被拒绝，且不能重复扣减/
// 增加目标的白给值——这正是本次修复要堵上的漏洞（此前只按"每小时总票数"限速，同一目标可以
// 在配额耗尽前被反复投票)。
func TestOnGiveawayVoteRejectsReplayOnSameBoard(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	env := wsEnvelope{E: "giveaway:vote", ID: 1, D: map[string]any{"targetId": "target1", "vote": "like"}}

	s.onGiveawayVote(client, env)
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("首次投票应成功，got %#v", data)
	}
	target := s.players["target1"]
	if got := ptrFloat(target.GiveawayValue); got != 49 {
		t.Fatalf("首次点赞应扣减 1%%，got %v", got)
	}
	if got := ptrInt(target.GiveawayBoardLikes); got != 1 {
		t.Fatalf("点赞计数应为 1，got %d", got)
	}

	// 重放同一条请求（同一个 target、同一次上板）
	s.onGiveawayVote(client, env)
	errMsg := lastReplyError(t, client)
	if errMsg != "你已经对这条自救内容投过票了" {
		t.Fatalf("重放请求应被拒绝，got err=%q", errMsg)
	}
	if got := ptrFloat(target.GiveawayValue); got != 49 {
		t.Fatalf("重放不应再次扣减白给值，got %v", got)
	}
	if got := ptrInt(target.GiveawayBoardLikes); got != 1 {
		t.Fatalf("重放不应再次增加点赞计数，got %d", got)
	}
}

// TestOnGiveawayVoteAllowsVoteAfterReboard 目标重新上板（新的一次自救宣言，新的
// GiveawayBoardSubmittedAt）后，之前投过票的 actor 应该能再投一次——确认修复没有把
// actor 对 target 的限制做成永久性的"终身一票"。
func TestOnGiveawayVoteAllowsVoteAfterReboard(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	env := wsEnvelope{E: "giveaway:vote", ID: 1, D: map[string]any{"targetId": "target1", "vote": "like"}}
	s.onGiveawayVote(client, env)
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("首次投票应成功，got %#v", data)
	}

	target := s.players["target1"]
	target.GiveawayBoardSubmittedAt = int64Ptr(*target.GiveawayBoardSubmittedAt + 1)
	target.GiveawayBoardExpiresAt = int64Ptr(nowMs() + 3_600_000)
	target.GiveawayValue = floatPtr(50)

	s.onGiveawayVote(client, wsEnvelope{E: "giveaway:vote", ID: 2, D: map[string]any{"targetId": "target1", "vote": "like"}})
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("重新上板后应允许再次投票，got %#v", data)
	}
	if got := ptrFloat(target.GiveawayValue); got != 49 {
		t.Fatalf("新一轮投票应正常扣减白给值，got %v", got)
	}
}

func lastReplyError(t *testing.T, client *Client) string {
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
