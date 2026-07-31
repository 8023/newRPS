package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func setupChatMentionServer(t *testing.T) (*Server, *Client) {
	t.Helper()
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}

	sender := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "sender", Name: "发送者", RoomID: "room1"}, SocketID: "sock-sender"}
	inRoomTarget := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "inroom", Name: "房间里的人"}, PushMentionEnabled: boolPtr(true)}
	strangerTarget := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "stranger", Name: "陌生人"}, PushMentionEnabled: boolPtr(true)}
	s.players["sender"] = sender
	s.players["inroom"] = inRoomTarget
	s.players["stranger"] = strangerTarget

	room := &RoomState{
		ID:           "room1",
		Settings:     types.RoomSettings{GameID: types.GameRPS},
		Seats:        map[types.SeatKey]SeatOccupant{types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "sender"}}, types.SeatB: nil},
		SpectatorIDs: []string{"inroom"},
		Choices:      map[types.SeatKey]types.Move{},
	}
	s.rooms["room1"] = room

	client := &Client{id: "sock-sender", playerID: "sender", sendCh: make(chan []byte, 8)}
	return s, client
}

// TestOnChatSendRoomMentionsFilterToNonMembers 房间聊天 @ 一个不在这个房间里的陌生玩家，
// 不应该触发针对 Ta 的推送——之前的实现不做任何房间成员校验，任何人都能把大厅里随便一个
// 玩家的 ID 塞进 mentions 触发一条与该房间无关的骚扰推送。
func TestOnChatSendRoomMentionsFilterToNonMembers(t *testing.T) {
	s, client := setupChatMentionServer(t)
	env := wsEnvelope{E: "chat:send", ID: 1, D: map[string]any{
		"roomId": "room1", "text": "hi", "mentions": []string{"inroom", "stranger"},
	}}
	s.onChatSend(client, env)
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("chat send should succeed, got %#v", data)
	}
	// 房间里的目标：mention push 的限流桶应该被创建（说明推送逻辑真正尝试过）。
	if s.rateBuckets["mention_push:sender:inroom"] == nil {
		t.Fatal("in-room mention should reach the push-throttle gate")
	}
	// 不在房间里的陌生人：应该在过滤阶段就被剔除，push 逻辑完全不会碰到它。
	if s.rateBuckets["mention_push:sender:stranger"] != nil {
		t.Fatal("mentioning a player outside the room should not trigger a push attempt")
	}
}

// TestOnChatSendMentionPushCooldownThrottlesRepeatedTargeting 同一发送者反复群发消息 @
// 同一个人，应该只有第一条真正触发推送节流通过，第二条起被节流拦下——避免抓包重放/脚本
// 群发消息对固定目标刷屏骚扰。
func TestOnChatSendMentionPushCooldownThrottlesRepeatedTargeting(t *testing.T) {
	s, client := setupChatMentionServer(t)
	env := func(id int64) wsEnvelope {
		return wsEnvelope{E: "chat:send", ID: id, D: map[string]any{
			"roomId": "room1", "text": "hi again", "mentions": []string{"inroom"},
		}}
	}
	s.onChatSend(client, env(1))
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("first chat send should succeed, got %#v", data)
	}
	s.onChatSend(client, env(2))
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("second chat send should still succeed (message itself is not blocked), got %#v", data)
	}
	bucket := s.rateBuckets["mention_push:sender:inroom"]
	if bucket == nil || bucket.CooldownUntil <= nowMs() {
		t.Fatal("repeated mention pushes to the same target should trip the per-pair cooldown")
	}
}
