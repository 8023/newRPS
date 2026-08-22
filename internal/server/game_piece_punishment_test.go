package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func emptyJungleBoardForTest() [][]*types.JungleCell {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	return board
}

func setupJunglePieceRoom(t *testing.T, perPiece bool) (*Server, *RoomState, *PlayerState, *PlayerState) {
	t.Helper()
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	playerA := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "playerA", Name: "甲", DisplayName: "甲", RoomID: "jungle-room"}}
	playerB := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "playerB", Name: "乙", DisplayName: "乙", RoomID: "jungle-room"}}
	s.players["playerA"] = playerA
	s.players["playerB"] = playerB
	board := emptyJungleBoardForTest()
	placeJungle(board, 4, 0, types.SeatA, types.JungleElephant)
	placeJungle(board, 5, 0, types.SeatB, types.JungleWolf)
	placeJungle(board, 8, 6, types.SeatA, types.JungleLion)
	placeJungle(board, 0, 6, types.SeatB, types.JungleLion)
	room := &RoomState{
		ID:    "jungle-room",
		Phase: types.PhaseChoosing,
		Settings: types.RoomSettings{
			GameID:                   types.GameJungle,
			EnablePunishment:         true,
			EnablePerPiecePunishment: perPiece,
			PunishmentSource:         "random",
			RequireOpponentConfirm:   false,
		},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "playerA", Name: "甲", DisplayName: "甲"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "playerB", Name: "乙", DisplayName: "乙"}},
		},
		Score:       map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatedScore: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatStats:   map[types.SeatKey]types.SeatStats{types.SeatA: {}, types.SeatB: {}},
		Jungle: &types.JungleState{
			Board:       board,
			Turn:        types.SeatA,
			RankedDelta: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
			UndoCount:   freshUndoCount(),
		},
	}
	s.rooms[room.ID] = room
	return s, room, playerA, playerB
}

func approveMidGamePunishment(t *testing.T, s *Server, room *RoomState, playerID string) {
	t.Helper()
	room.Proofs = []types.PunishmentProof{{
		PlayerID: playerID, Status: "approved", ConfirmedBy: "test",
	}}
	if !s.finishPunishmentIfComplete(room) {
		t.Fatal("mid-game punishment should complete")
	}
}

func TestJungleCaptureStartsMidGamePunishmentWithoutScore(t *testing.T) {
	s, room, _, loser := setupJunglePieceRoom(t, true)
	if ok, errMsg := s.applyJungleMove(room, types.SeatA, 4, 0, 5, 0); !ok {
		t.Fatalf("capture move: %s", errMsg)
	}
	if room.Phase != types.PhasePunishment {
		t.Fatalf("capture should enter punishment, got %s", room.Phase)
	}
	if room.Score[types.SeatA] != 0 || room.Score[types.SeatB] != 0 {
		t.Fatalf("per-piece punishment must not change score, got %+v", room.Score)
	}
	if room.Jungle == nil || room.Jungle.Ended {
		t.Fatal("game must stay in progress during per-piece punishment")
	}
	if !containsString(room.PunishedPlayerIDs, loser.ID) {
		t.Fatalf("captured side should be punished, got %v", room.PunishedPlayerIDs)
	}
}

func TestJungleMidGamePunishmentResumeKeepsBoard(t *testing.T) {
	s, room, _, loser := setupJunglePieceRoom(t, true)
	if ok, errMsg := s.applyJungleMove(room, types.SeatA, 4, 0, 5, 0); !ok {
		t.Fatalf("capture move: %s", errMsg)
	}
	approveMidGamePunishment(t, s, room, loser.ID)
	if room.Phase != types.PhaseChoosing {
		t.Fatalf("after approval should resume choosing, got %s", room.Phase)
	}
	if room.Jungle == nil || room.Jungle.Ended {
		t.Fatal("board must not reset after mid-game punishment")
	}
	if room.Jungle.Turn != types.SeatB {
		t.Fatalf("after capture it should be B's turn, got %s", room.Jungle.Turn)
	}
	cell := room.Jungle.Board[5][0]
	if cell == nil || *cell != jungleCellOf(types.SeatA, types.JungleElephant) {
		t.Fatalf("captured square should keep attacker, got %v", cell)
	}
}

// TestJungleMidGamePunishmentConfirmRPCResumesGameNotNewRound 复现真实反馈：棋类「每子惩罚」
// 走 onPunishmentConfirm/onPunishmentReview/onPunishmentSubmit 这些真实 RPC 入口审批通过时，
// 之前直接 `if s.punishmentComplete(room) { s.resetForNextRound(room) }`，完全没检查
// room.midGamePunishment，导致每子惩罚一结算就把进行中的对局强行开新局。
// 之前的测试（approveMidGamePunishment）都是直接调用 finishPunishmentIfComplete，绕过了这几个
// RPC handler 里的重复代码，所以没能测出这个问题——这里改为走真实的 onPunishmentConfirm 入口。
func TestJungleMidGamePunishmentConfirmRPCResumesGameNotNewRound(t *testing.T) {
	s, room, winner, loser := setupJunglePieceRoom(t, true)
	if ok, errMsg := s.applyJungleMove(room, types.SeatA, 4, 0, 5, 0); !ok {
		t.Fatalf("capture move: %s", errMsg)
	}
	if room.Phase != types.PhasePunishment {
		t.Fatalf("capture should enter punishment, got %s", room.Phase)
	}
	room.Proofs = []types.PunishmentProof{{PlayerID: loser.ID, Status: "pending", TaskText: "任务"}}
	winner.SocketID, winner.RoomID = "sock-w", room.ID
	loser.SocketID, loser.RoomID = "sock-l", room.ID
	s.clients = map[string]*Client{
		"sock-w": {id: "sock-w", playerID: winner.ID, sendCh: make(chan []byte, 8), done: make(chan struct{})},
		"sock-l": {id: "sock-l", playerID: loser.ID, sendCh: make(chan []byte, 8), done: make(chan struct{})},
	}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}

	env := wsEnvelope{ID: 1, D: map[string]any{"playerId": loser.ID}}
	s.onPunishmentConfirm(s.clients["sock-w"], env)

	if room.Jungle == nil || room.Jungle.Ended {
		t.Fatal("mid-game punishment approval via RPC must resume the same game, not end it")
	}
	if room.Phase != types.PhaseChoosing {
		t.Fatalf("after RPC approval should resume choosing, got %s", room.Phase)
	}
	cell := room.Jungle.Board[5][0]
	if cell == nil || *cell != jungleCellOf(types.SeatA, types.JungleElephant) {
		t.Fatalf("captured square should keep attacker (board must not reset), got %v", cell)
	}
}

func TestJungleCaptureEndingSkipsSecondPunishment(t *testing.T) {
	s, room, _, loser := setupJunglePieceRoom(t, true)
	room.Jungle.Board[0][6] = nil
	if ok, errMsg := s.applyJungleMove(room, types.SeatA, 4, 0, 5, 0); !ok {
		t.Fatalf("last-piece capture: %s", errMsg)
	}
	if room.Phase != types.PhasePunishment || room.Jungle.Ended {
		t.Fatalf("last capture should punish first, phase=%s ended=%v", room.Phase, room.Jungle != nil && room.Jungle.Ended)
	}
	approveMidGamePunishment(t, s, room, loser.ID)
	if room.Jungle == nil || !room.Jungle.Ended {
		t.Fatal("after last-capture punishment the game should end")
	}
	if room.Phase == types.PhasePunishment {
		t.Fatal("game end after per-piece must not start a second punishment")
	}
	if room.Score[types.SeatA] != 1 {
		t.Fatalf("ending after last capture should still score the win, got %d", room.Score[types.SeatA])
	}
}

func TestJungleDenEntryWithoutCaptureStillPunishes(t *testing.T) {
	s, room, _, loser := setupJunglePieceRoom(t, true)
	room.Jungle.Board = emptyJungleBoardForTest()
	placeJungle(room.Jungle.Board, 1, 3, types.SeatA, types.JungleRat)
	placeJungle(room.Jungle.Board, 8, 6, types.SeatA, types.JungleLion)
	placeJungle(room.Jungle.Board, 0, 6, types.SeatB, types.JungleLion)
	if ok, errMsg := s.applyJungleMove(room, types.SeatA, 1, 3, 0, 3); !ok {
		t.Fatalf("den entry: %s", errMsg)
	}
	if room.Jungle == nil || !room.Jungle.Ended {
		t.Fatal("den entry should end the game")
	}
	if room.Phase != types.PhasePunishment {
		t.Fatalf("den entry without capture should still use end-game punishment, got %s", room.Phase)
	}
	if !containsString(room.PunishedPlayerIDs, loser.ID) {
		t.Fatalf("loser should be punished, got %v", room.PunishedPlayerIDs)
	}
}

func TestJungleGiveawayChanceIsHalfValue(t *testing.T) {
	if got := jungleGiveawayChance(80); got != 40 {
		t.Fatalf("jungleGiveawayChance(80)=%v, want 40", got)
	}
	if got := jungleGiveawayChance(0); got != 0 {
		t.Fatalf("jungleGiveawayChance(0)=%v, want 0", got)
	}
}

func TestJungleTurnSkipHandsTurnToOpponent(t *testing.T) {
	_, room, _, _ := setupJunglePieceRoom(t, false)
	room.Jungle.Turn = types.SeatA
	skipJungleTurn(room, types.SeatA)
	if room.Jungle.Turn != types.SeatB {
		t.Fatalf("skip should hand turn to B, got %s", room.Jungle.Turn)
	}
}

func TestJungleForcedGiveawaySkipsCurrentTurn(t *testing.T) {
	s, room, _, _ := setupJunglePieceRoom(t, false)
	room.Jungle.Turn = types.SeatA
	setForcedGiveaway(room, types.SeatA, "主人甲")
	s.maybeJungleGiveawaySkip(room)
	if room.Jungle.Turn != types.SeatB {
		t.Fatalf("forced giveaway should skip A, got %s", room.Jungle.Turn)
	}
	if forcedGiveawayMasterName(room, types.SeatA) != "" {
		t.Fatal("forced giveaway should be consumed")
	}
}

// 国际象棋每子惩罚在对局中途插入一条 history，前端靠 ChessWhiteSeat 判断哪一边执白；
// 该字段留空时两个座位都会被判成黑棋，对局记录会显示成"黑棋子被黑棋子吃掉"。
func TestChessCapturePunishmentHistoryKeepsWhiteSeat(t *testing.T) {
	s := newTestServer(t)
	s.rooms = map[string]*RoomState{}
	s.roomBroadcastTimers = map[string]*roomBroadcastPending{}
	s.turnBasedClockTimers = map[string]*turnBasedClockTimer{}
	s.players["playerA"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "playerA", Name: "甲", DisplayName: "甲", RoomID: "chess-piece"}}
	s.players["playerB"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "playerB", Name: "乙", DisplayName: "乙", RoomID: "chess-piece"}}
	room := &RoomState{
		ID:     "chess-piece",
		Phase:  types.PhaseReady,
		Status: "waiting",
		Settings: types.RoomSettings{
			GameID:                   types.GameChess,
			EnablePunishment:         true,
			EnablePerPiecePunishment: true,
			PunishmentSource:         "random",
		},
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "playerA", Name: "甲", DisplayName: "甲"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "playerB", Name: "乙", DisplayName: "乙"}},
		},
		Ready:       map[types.SeatKey]bool{types.SeatA: true, types.SeatB: true},
		Score:       map[types.SeatKey]int{},
		SeatedScore: map[types.SeatKey]int{},
		SeatStats:   map[types.SeatKey]types.SeatStats{},
	}
	s.rooms[room.ID] = room
	s.startChessRoom(room)
	white := room.Chess.WhiteSeat
	black := oppositeSeat(white)
	// 1. e4 d5 2. exd5 —— 白吃黑兵，触发每子惩罚
	if ok, msg := s.applyChessMove(room, white, 6, 4, 4, 4, ""); !ok {
		t.Fatalf("e4: %s", msg)
	}
	if ok, msg := s.applyChessMove(room, black, 1, 3, 3, 3, ""); !ok {
		t.Fatalf("d5: %s", msg)
	}
	if ok, msg := s.applyChessMove(room, white, 4, 4, 3, 3, ""); !ok {
		t.Fatalf("exd5: %s", msg)
	}
	if room.Phase != types.PhasePunishment {
		t.Fatalf("吃子后应进入惩罚阶段，got %s", room.Phase)
	}
	if len(room.RoundHistory) == 0 {
		t.Fatal("吃子惩罚应写入一条对局记录")
	}
	item := room.RoundHistory[0]
	if item.ResultLabel != "吃子惩罚" {
		t.Fatalf("resultLabel=%q, want 吃子惩罚", item.ResultLabel)
	}
	if item.ChessWhiteSeat != white {
		t.Fatalf("history.ChessWhiteSeat=%q, want %q（留空会让前端把两边都显示成黑棋）", item.ChessWhiteSeat, white)
	}
}
