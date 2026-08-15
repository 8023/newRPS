package server

import (
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func TestPrepareNextChoiceResetsChessRoom(t *testing.T) {
	s := &Server{}
	room := &RoomState{
		ID:       "chess-1",
		Settings: types.RoomSettings{GameID: types.GameChess},
		Phase:    types.PhaseResult,
		Status:   "playing",
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "a", Name: "甲"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "b", Name: "乙"}},
		},
		Ready: map[types.SeatKey]bool{types.SeatA: true, types.SeatB: true},
		Chess: freshChessState(types.SeatA),
	}
	room.Chess.Ended = true
	room.chessRepetition = []string{"x"}
	s.prepareNextChoice(room)
	if room.Phase != types.PhaseReady {
		t.Fatalf("phase=%s, want ready", room.Phase)
	}
	if room.Chess != nil {
		t.Fatal("chess state should be cleared after punishment/next-round reset")
	}
	if room.chessRepetition != nil {
		t.Fatal("repetition history should be cleared")
	}
	if room.Ready[types.SeatA] || room.Ready[types.SeatB] {
		t.Fatal("both seats should need to ready again")
	}
}

func testChessRoom(t *testing.T, moveSeconds int) (*Server, *RoomState) {
	t.Helper()
	s := &Server{
		rooms:                map[string]*RoomState{},
		players:              map[string]*PlayerState{},
		turnBasedClockTimers: map[string]*turnBasedClockTimer{},
		roomBroadcastTimers:  map[string]*roomBroadcastPending{},
	}
	room := &RoomState{
		ID: "chess-timer",
		Settings: types.RoomSettings{
			GameID:           types.GameChess,
			ChessMoveSeconds: moveSeconds,
		},
		Phase:  types.PhaseReady,
		Status: "waiting",
		Seats: map[types.SeatKey]SeatOccupant{
			types.SeatA: &HumanSeat{Player: types.PublicPlayer{ID: "a", Name: "甲"}},
			types.SeatB: &HumanSeat{Player: types.PublicPlayer{ID: "b", Name: "乙"}},
		},
		Ready:       map[types.SeatKey]bool{types.SeatA: true, types.SeatB: true},
		Score:       map[types.SeatKey]int{},
		SeatedScore: map[types.SeatKey]int{},
		SeatStats:   map[types.SeatKey]types.SeatStats{},
	}
	s.rooms[room.ID] = room
	s.startChessRoom(room)
	if room.Chess == nil {
		t.Fatal("expected chess state")
	}
	return s, room
}

func TestApplyChessMoveResetsMoveDeadline(t *testing.T) {
	s, room := testChessRoom(t, 30)
	first := room.Chess.MoveDeadlineAt
	if first <= 0 {
		t.Fatal("start should arm per-move clock")
	}
	time.Sleep(15 * time.Millisecond)
	ok, errMsg := s.applyChessMove(room, room.Chess.WhiteSeat, 6, 4, 4, 4, "")
	if !ok {
		t.Fatalf("e2-e4: %s", errMsg)
	}
	if room.Chess.MoveDeadlineAt <= first {
		t.Fatalf("move deadline should reset, before=%d after=%d", first, room.Chess.MoveDeadlineAt)
	}
	if max := nowMs() + 30_000; room.Chess.MoveDeadlineAt > max {
		t.Fatalf("move deadline must not exceed configured turn time, deadline=%d max=%d", room.Chess.MoveDeadlineAt, max)
	}
	if s.turnBasedClockTimers[room.ID] == nil {
		t.Fatal("server clock timer should be rescheduled after a move")
	}
}

func TestChessGameClockNeverGainsTimeAcrossMoves(t *testing.T) {
	s, room := testChessRoom(t, 30)
	room.Settings.ChessGameMinutes = 1
	// testChessRoom 已按仅每子计时启动；重新挂钟以覆盖双方总时长。
	s.armChessTimers(room, room.Chess.Turn)
	whiteSeat := room.Chess.WhiteSeat
	time.Sleep(15 * time.Millisecond)
	if ok, errMsg := s.applyChessMove(room, whiteSeat, 6, 4, 4, 4, ""); !ok {
		t.Fatalf("e2-e4: %s", errMsg)
	}
	whiteRemaining := room.Chess.ClockRemaining[whiteSeat]
	if whiteRemaining <= 0 || whiteRemaining >= 60_000 {
		t.Fatalf("white remaining=%d, want elapsed time deducted from 60000", whiteRemaining)
	}

	blackSeat := oppositeSeat(whiteSeat)
	time.Sleep(15 * time.Millisecond)
	if ok, errMsg := s.applyChessMove(room, blackSeat, 1, 4, 3, 4, ""); !ok {
		t.Fatalf("e7-e5: %s", errMsg)
	}
	blackRemaining := room.Chess.ClockRemaining[blackSeat]
	if blackRemaining <= 0 || blackRemaining >= 60_000 {
		t.Fatalf("black remaining=%d, want elapsed time deducted from 60000", blackRemaining)
	}
	if got := room.Chess.ClockDeadlineAt - nowMs(); got > whiteRemaining || got < whiteRemaining-100 {
		t.Fatalf("resumed white clock=%dms, frozen remaining=%dms", got, whiteRemaining)
	}
}

func TestChessMoveTimeoutForfeits(t *testing.T) {
	s, room := testChessRoom(t, 30)
	room.Chess.MoveDeadlineAt = nowMs() - 1
	s.scheduleChessClockTimer(room)
	deadline := time.Now().Add(400 * time.Millisecond)
	for time.Now().Before(deadline) {
		s.mu.Lock()
		ended := room.Chess != nil && room.Chess.Ended
		s.mu.Unlock()
		if ended {
			if room.Chess.Winner == types.ResultDraw {
				t.Fatal("opening position timeout should be a win, not a draw")
			}
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("expired move clock should finish the game")
}
