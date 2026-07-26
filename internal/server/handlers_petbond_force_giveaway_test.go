package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// setupForceGiveawayRoom 造一个 A=master、B=pet 的 RPS 房间，master 已经是 pet 的直系主人，
// pet 开启了白给模式。返回 (server, masterClient)，调用方按需再改字段（比如换成 othello/tictactoe，
// 或关掉 pet 的白给）。
func setupForceGiveawayRoom(t *testing.T, gameID types.GameID) (*Server, *Client) {
	t.Helper()
	s := newTestServer(t)
	s.petBonds = map[string]*petBond{}
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}

	master := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "master1", Name: "师傅", RoomID: "room1"}, SocketID: "sock-master"}
	pet := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "pet1", Name: "徒弟", RoomID: "room1", GiveawayEnabled: boolPtr(true), GiveawayValue: floatPtr(1)}}
	s.players["master1"] = master
	s.players["pet1"] = pet
	if err := s.createBond("master1", "pet1"); err != nil {
		t.Fatalf("seed bond: %v", err)
	}

	room := &RoomState{
		ID:       "room1",
		Settings: types.RoomSettings{GameID: gameID, EnableRanked: true},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "master1"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "pet1"}},
		},
	}
	switch gameID {
	case types.GameRPS:
		room.Phase = types.PhaseChoosing
		room.Choices = map[types.SeatKey]types.Move{types.SeatB: types.MovePaper}
	case types.GameGomoku:
		room.Phase = types.PhaseChoosing
		room.Gomoku = freshGomokuState(types.SeatB)
	}
	s.rooms["room1"] = room

	client := &Client{id: "sock-master", playerID: "master1", sendCh: make(chan []byte, 4)}
	return s, client
}

func TestOnPetBondForceGiveawaySuccess(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameRPS)
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", data)
	}
	room := s.rooms["room1"]
	if got := room.Choices[types.SeatB]; got != types.MoveGiveaway {
		t.Fatalf("主人强制应立即覆盖宠物已选出拳，got %q", got)
	}
	if got := forcedGiveawayMasterName(room, types.SeatB); got != "师傅" {
		t.Fatalf("expected room-scoped force reason 师傅, got %q", got)
	}
}

func TestOnPetBondForceGiveawayKeepsCurrentGomokuForceHidden(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameGomoku)
	room := s.rooms["room1"]
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("expected ok=true, got %#v", data)
	}
	if room.Gomoku.GiveawaySeat != "" || room.Gomoku.GiveawayForcedByMasterName != "" || room.ResultText != "" {
		t.Fatalf("宠物直接落子前不得看到主人强制，got seat=%q master=%q text=%q", room.Gomoku.GiveawaySeat, room.Gomoku.GiveawayForcedByMasterName, room.ResultText)
	}
	if got := forcedGiveawayMasterName(room, types.SeatB); got != "师傅" {
		t.Fatalf("隐藏强制标记应保存在房间内，got %q", got)
	}
}

func TestOnPetBondForceGiveawayRevealsWhenPetThenClicksGiveaway(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameGomoku)
	room := s.rooms["room1"]
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	lastReplyData(t, client)
	if ok, errMsg := s.chooseGomokuGiveaway(room, types.SeatB); !ok {
		t.Fatalf("pet choose giveaway: %s", errMsg)
	}
	if room.Gomoku.GiveawaySeat != types.SeatB || room.Gomoku.GiveawayForcedByMasterName != "师傅" {
		t.Fatalf("宠物主动点击后应公开本手白给，got seat=%q master=%q", room.Gomoku.GiveawaySeat, room.Gomoku.GiveawayForcedByMasterName)
	}
	if got := forcedGiveawayMasterName(room, types.SeatB); got != "" {
		t.Fatalf("公开后应消费隐藏标记，got %q", got)
	}
}

func TestOnPetBondForceGiveawayRejectsNonRankedOthello(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameOthello)
	s.rooms["room1"].Settings.EnableRanked = false
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	if got := lastReplyErr(t, client); got != "黑白棋只有排位房才支持白给结算" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestOnPetBondForceGiveawayRejectsNotBonded(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameRPS)
	s.petBonds = map[string]*petBond{} // 撤掉刚建立的认主关系
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	if got := lastReplyErr(t, client); got != "对方不是你的宠物" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestOnPetBondForceGiveawayRejectsGiveawayDisabled(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameRPS)
	s.players["pet1"].GiveawayEnabled = boolPtr(false)
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	if got := lastReplyErr(t, client); got != "对方未开启白给模式" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestOnPetBondForceGiveawayRejectsDoubleTrigger(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameRPS)
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	lastReplyData(t, client) // drain first reply
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 2, D: map[string]any{"targetId": "pet1"}})
	if got := lastReplyErr(t, client); got != "已经强制过一次，正在等待本轮结算" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestOnPetBondForceGiveawayRejectsWrongDirection(t *testing.T) {
	// pet 不能对 master 使用强制白给：反过来发起应该被拒绝（对方不是自己的宠物）。
	s, _ := setupForceGiveawayRoom(t, types.GameRPS)
	petClient := &Client{id: "sock-pet", playerID: "pet1", sendCh: make(chan []byte, 4)}
	s.players["pet1"].SocketID = "sock-pet"
	s.onPetBondForceGiveaway(petClient, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "master1"}})
	if got := lastReplyErr(t, petClient); got != "对方不是你的宠物" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestOnPetBondForceGiveawayAfterPetArmedDoesNotDoubleBoost(t *testing.T) {
	s, client := setupForceGiveawayRoom(t, types.GameGomoku)
	s.cfg.Giveaway.ActiveBoostValue = 2
	room := s.rooms["room1"]
	pet := s.players["pet1"]
	if ok, errMsg := s.chooseGomokuGiveaway(room, types.SeatB); !ok {
		t.Fatalf("pet choose giveaway: %s", errMsg)
	}
	if got := ptrFloat(pet.GiveawayValue); got != 3 {
		t.Fatalf("宠物主动白给后白给值应为 3，got %.1f", got)
	}
	s.onPetBondForceGiveaway(client, wsEnvelope{E: "petbond:forceGiveaway", ID: 1, D: map[string]any{"targetId": "pet1"}})
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("expected force ok, got %#v", data)
	}
	if room.Gomoku.GiveawayForcedByMasterName != "师傅" {
		t.Fatalf("主人随后强制应沿用同一个本手白给状态，got %q", room.Gomoku.GiveawayForcedByMasterName)
	}
	if ok, errMsg := s.applyGomokuMove(room, types.SeatB, 7, 7); !ok {
		t.Fatalf("apply forced move: %s", errMsg)
	}
	cell := room.Gomoku.Board[7][7]
	if cell == nil || *cell != types.GomokuWhite {
		t.Fatalf("宠物执黑时被强制白给应落成白棋，got %#v", cell)
	}
	if got := ptrFloat(pet.GiveawayValue); got != 3 {
		t.Fatalf("主动后再被主人强制不应重复增加白给值，got %.1f", got)
	}
}
