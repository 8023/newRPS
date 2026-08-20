package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

// lastReplyEnvelope 与 lastReplyData 不同：错误回复不 t.Fatal，而是把 err 文案一并返回，
// 用来断言"应该失败，且失败原因是 XXX"这类用例。
func lastReplyEnvelope(t *testing.T, client *Client) (any, string) {
	t.Helper()
	select {
	case data := <-client.sendCh:
		var env wire.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			t.Fatalf("failed to decode reply: %v", err)
		}
		if env.Err != "" {
			return nil, env.Err
		}
		front, err := pbconv.RawBodyToFront(env.RawBody)
		if err != nil {
			t.Fatalf("decode raw body: %v", err)
		}
		return front, ""
	default:
		t.Fatal("expected a reply on sendCh, got none")
		return nil, ""
	}
}

// TestOnRoomCreateRejectsUnknownSeriesID 覆盖后端对 punishmentSource=series 的兜底校验：
// 前端下拉菜单本身已经不会提交不存在的系列 ID（无匹配即禁用下拉+禁用创建按钮，见
// AppViews.tsx CreateRoom），但 WS RPC 是可以绕过前端直接调用的，后端必须独立拒绝一次
// 提交了悬空/不存在 punishmentSeriesId 的建房请求，而不是信任客户端传来的 ID。
func TestOnRoomCreateRejectsUnknownSeriesID(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.cfg.AccessControl.MaxActiveRoomsPerOwner = 2
	s.punishmentSeriesCache = []types.PunishmentSeriesTaskConfig{
		{ID: "real-series", Name: "真实系列", Steps: []types.PunishmentSeriesStep{{TaskIDs: []string{"t1"}}}},
	}

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "玩家"}, SocketID: "sock-1"}
	s.players["p1"] = player
	client := &Client{id: "sock-1", playerID: "p1", sendCh: make(chan []byte, 64)}

	env := wsEnvelope{E: "room:create", ID: 1, D: map[string]any{
		"settings": map[string]any{
			"gameId":             "rps",
			"enablePunishment":   true,
			"punishmentSource":   "series",
			"punishmentSeriesId": "does-not-exist",
		},
	}}
	s.onRoomCreate(client, env)

	_, errMsg := lastReplyEnvelope(t, client)
	if errMsg != "请选择一个有效的系列任务" {
		t.Fatalf("expected 请选择一个有效的系列任务, got %q", errMsg)
	}
	if len(s.rooms) != 0 {
		t.Fatalf("校验失败不应创建房间，got %d rooms", len(s.rooms))
	}
}

// TestOnRoomCreateAcceptsKnownSeriesID 正向用例：真实存在的系列 ID 能正常建房。
func TestOnRoomCreateAcceptsKnownSeriesID(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.cfg.AccessControl.MaxActiveRoomsPerOwner = 2
	s.punishmentSeriesCache = []types.PunishmentSeriesTaskConfig{
		{ID: "real-series", Name: "真实系列", Steps: []types.PunishmentSeriesStep{{TaskIDs: []string{"t1"}}}},
	}

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "玩家"}, SocketID: "sock-1"}
	s.players["p1"] = player
	client := &Client{id: "sock-1", playerID: "p1", sendCh: make(chan []byte, 64)}

	env := wsEnvelope{E: "room:create", ID: 1, D: map[string]any{
		"settings": map[string]any{
			"gameId":             "rps",
			"enablePunishment":   true,
			"punishmentSource":   "series",
			"punishmentSeriesId": "real-series",
		},
	}}
	s.onRoomCreate(client, env)

	_, errMsg := lastReplyEnvelope(t, client)
	if errMsg != "" {
		t.Fatalf("提交存在的系列 ID 应建房成功，got error: %q", errMsg)
	}
	if len(s.rooms) != 1 {
		t.Fatalf("expected 1 room created, got %d", len(s.rooms))
	}
}
