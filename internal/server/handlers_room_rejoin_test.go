package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestOnRoomJoinAllowsRejoiningSameRoomWhilePunished 复现"惩罚未完成时刷新页面重新进入房间"：
// 玩家仍在受罚且未离开房间（player.RoomID 依然是这个房间），前端却又发了一次 room:join（例如
// 刷新后状态还没恢复完、又点了大厅里同一个房间）。此前 onRoomJoin 无条件调用
// leaveRoom(player, LeaveSwitchRoom) 把这当成"换房"处理，命中惩罚未完成禁止离开的检查，
// 直接把玩家挡在门外——但玩家根本没打算换房，只是想回到自己已经在的房间。
func TestOnRoomJoinAllowsRejoiningSameRoomWhilePunished(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}

	loser := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "l", Name: "输家", RoomID: "room1"}, SocketID: "sock-l"}
	s.players["l"] = loser

	room := &RoomState{
		ID:       "room1",
		Phase:    types.PhasePunishment,
		Settings: types.RoomSettings{GameID: types.GameRPS, EnablePunishment: true},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "w"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "l"}},
		},
		PunishedPlayerIDs: []string{"l"},
	}
	s.rooms["room1"] = room

	client := &Client{id: "sock-l", playerID: "l", sendCh: make(chan []byte, 64)}

	env := wsEnvelope{E: "room:join", ID: 1, D: map[string]any{"roomId": "room1"}}
	s.onRoomJoin(client, env)

	data := lastReplyData(t, client)
	if data["room"] == nil {
		t.Fatalf("受罚玩家重新进入自己已在的房间应成功回传房间快照，got %#v", data)
	}
	if loser.RoomID != "room1" {
		t.Fatalf("玩家应仍在房间内，got RoomID=%q", loser.RoomID)
	}
	if len(room.PunishedPlayerIDs) != 1 || room.PunishedPlayerIDs[0] != "l" {
		t.Fatalf("重新进入不应清空未完成的惩罚状态，got %#v", room.PunishedPlayerIDs)
	}
}
