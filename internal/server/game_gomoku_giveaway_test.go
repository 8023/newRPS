package server

import (
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func setupGomokuGiveawayTest(t *testing.T, value float64) (*Server, *RoomState, *PlayerState) {
	t.Helper()
	s := newTestServer(t)
	s.cfg.Giveaway.ActiveBoostValue = 2
	s.rooms = map[string]*RoomState{}
	playerA := &PlayerState{PublicPlayer: types.PublicPlayer{
		ID: "playerA", Name: "甲", DisplayName: "甲", RoomID: "gomoku-room",
		GiveawayEnabled: boolPtr(true), GiveawayValue: floatPtr(value),
	}}
	playerB := &PlayerState{PublicPlayer: types.PublicPlayer{
		ID: "playerB", Name: "乙", DisplayName: "乙", RoomID: "gomoku-room",
		GiveawayEnabled: boolPtr(true), GiveawayValue: floatPtr(0),
	}}
	s.players["playerA"] = playerA
	s.players["playerB"] = playerB
	room := &RoomState{
		ID: "gomoku-room", Phase: types.PhaseChoosing,
		Settings: types.RoomSettings{GameID: types.GameGomoku},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "playerA", Name: "甲", DisplayName: "甲"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "playerB", Name: "乙", DisplayName: "乙"}},
		},
		Gomoku: freshGomokuState(types.SeatA),
	}
	s.rooms[room.ID] = room
	return s, room, playerA
}

func TestGomokuManualGiveawayArmsOnlyCurrentHand(t *testing.T) {
	s, room, player := setupGomokuGiveawayTest(t, 10)
	if ok, errMsg := s.chooseGomokuGiveaway(room, types.SeatA); !ok {
		t.Fatalf("choose giveaway: %s", errMsg)
	}
	if room.Gomoku.GiveawaySeat != types.SeatA {
		t.Fatalf("白给按钮应武装当前手，got %q", room.Gomoku.GiveawaySeat)
	}
	if ok, errMsg := s.applyGomokuMove(room, types.SeatA, 7, 7); !ok {
		t.Fatalf("apply giveaway move: %s", errMsg)
	}
	cell := room.Gomoku.Board[7][7]
	if cell == nil || *cell != types.GomokuWhite {
		t.Fatalf("黑方本手白给应落成白棋，got %#v", cell)
	}
	if room.Gomoku.GiveawaySeat != "" || room.Gomoku.GiveawayForcedByMasterName != "" {
		t.Fatal("白给状态应在本手落子后清除，不能延续到后续手")
	}
	if got := ptrFloat(player.GiveawayValue); got != 12 {
		t.Fatalf("主动白给应只增加一次白给值，got %.1f", got)
	}
}

func TestGomokuNaturalGiveawayChangesJustPlacedStone(t *testing.T) {
	s, room, _ := setupGomokuGiveawayTest(t, 100)
	if room.ResultText != "" {
		t.Fatalf("概率白给在落子前不应有提示，got %q", room.ResultText)
	}
	if ok, errMsg := s.applyGomokuMove(room, types.SeatA, 3, 4); !ok {
		t.Fatalf("apply natural giveaway move: %s", errMsg)
	}
	cell := room.Gomoku.Board[3][4]
	if cell == nil || *cell != types.GomokuWhite {
		t.Fatalf("100%% 白给值应把黑方刚落的子变成白棋，got %#v", cell)
	}
	if !strings.Contains(room.ResultText, "按 100.0% 白给值触发白给") {
		t.Fatalf("自然触发提示缺少概率原因：%q", room.ResultText)
	}
}

func TestGomokuQueuedMasterForceIsCertainOnPetMove(t *testing.T) {
	s, room, player := setupGomokuGiveawayTest(t, 0)
	setForcedGiveaway(room, types.SeatA, "主人甲")
	if room.Gomoku.GiveawaySeat != "" || room.Gomoku.GiveawayForcedByMasterName != "" || room.ResultText != "" {
		t.Fatalf("主人强制在宠物落子前必须保持隐藏，got seat=%q master=%q text=%q", room.Gomoku.GiveawaySeat, room.Gomoku.GiveawayForcedByMasterName, room.ResultText)
	}
	if ok, errMsg := s.applyGomokuMove(room, types.SeatA, 5, 6); !ok {
		t.Fatalf("apply forced giveaway move: %s", errMsg)
	}
	cell := room.Gomoku.Board[5][6]
	if cell == nil || *cell != types.GomokuWhite {
		t.Fatalf("主人强制应无视 0%% 白给值，100%% 落成对方白棋，got %#v", cell)
	}
	if !strings.Contains(room.ResultText, "主人（主人甲）强制（甲）白给") {
		t.Fatalf("主人强制提示缺少明确原因：%q", room.ResultText)
	}
	if got := ptrFloat(player.GiveawayValue); got != 2 {
		t.Fatalf("强制白给应只增加一次主动白给值，got %.1f", got)
	}
}
