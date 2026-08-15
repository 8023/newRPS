package server

import (
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func TestTurnBasedGamesShareClockLifecycle(t *testing.T) {
	s := &Server{turnBasedClockTimers: map[string]*turnBasedClockTimer{}}
	type clockFields struct {
		moveDeadlineAt  int64
		clockDeadlineAt int64
		remaining       map[types.SeatKey]int64
	}
	tests := []struct {
		name   string
		room   *RoomState
		arm    func(*RoomState)
		pause  func(*RoomState)
		resume func(*RoomState)
		fields func(*RoomState) clockFields
	}{
		{
			name:   "othello",
			room:   &RoomState{ID: "clock-othello", Settings: types.RoomSettings{OthelloMoveSeconds: 30, OthelloGameMinutes: 1}, Othello: &types.OthelloState{Turn: types.SeatA}},
			arm:    func(room *RoomState) { s.armOthelloTimers(room, types.SeatA) },
			pause:  func(room *RoomState) { s.pauseOthelloTimers(room, types.SeatA) },
			resume: func(room *RoomState) { s.armOthelloTimers(room, room.Othello.Turn) },
			fields: func(room *RoomState) clockFields {
				return clockFields{room.Othello.MoveDeadlineAt, room.Othello.ClockDeadlineAt, room.Othello.ClockRemaining}
			},
		},
		{
			name:  "gomoku",
			room:  &RoomState{ID: "clock-gomoku", Settings: types.RoomSettings{GomokuMoveSeconds: 30, GomokuGameMinutes: 1}, Gomoku: &types.GomokuState{Turn: types.SeatA}},
			arm:   func(room *RoomState) { s.armGomokuTimers(room, types.SeatA) },
			pause: s.pauseGomokuTimers, resume: s.resumeGomokuTimers,
			fields: func(room *RoomState) clockFields {
				return clockFields{room.Gomoku.MoveDeadlineAt, room.Gomoku.ClockDeadlineAt, room.Gomoku.ClockRemaining}
			},
		},
		{
			name:  "jungle",
			room:  &RoomState{ID: "clock-jungle", Settings: types.RoomSettings{JungleMoveSeconds: 30, JungleGameMinutes: 1}, Jungle: &types.JungleState{Turn: types.SeatA}},
			arm:   func(room *RoomState) { s.armJungleTimers(room, types.SeatA) },
			pause: s.pauseJungleTimers, resume: s.resumeJungleTimers,
			fields: func(room *RoomState) clockFields {
				return clockFields{room.Jungle.MoveDeadlineAt, room.Jungle.ClockDeadlineAt, room.Jungle.ClockRemaining}
			},
		},
		{
			name:  "chess",
			room:  &RoomState{ID: "clock-chess", Settings: types.RoomSettings{ChessMoveSeconds: 30, ChessGameMinutes: 1}, Chess: &types.ChessState{Turn: types.SeatA}},
			arm:   func(room *RoomState) { s.armChessTimers(room, types.SeatA) },
			pause: s.pauseChessTimers, resume: s.resumeChessTimers,
			fields: func(room *RoomState) clockFields {
				return clockFields{room.Chess.MoveDeadlineAt, room.Chess.ClockDeadlineAt, room.Chess.ClockRemaining}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.arm(test.room)
			armed := test.fields(test.room)
			if armed.moveDeadlineAt <= nowMs() || armed.clockDeadlineAt <= nowMs() {
				t.Fatalf("clock not armed: %+v", armed)
			}
			if armed.remaining[types.SeatA] != 60_000 || armed.remaining[types.SeatB] != 60_000 {
				t.Fatalf("initial total clock=%v", armed.remaining)
			}
			if s.turnBasedClockTimers[test.room.ID] == nil {
				t.Fatal("shared server timer was not scheduled")
			}

			time.Sleep(10 * time.Millisecond)
			test.pause(test.room)
			paused := test.fields(test.room)
			if paused.moveDeadlineAt != 0 || paused.clockDeadlineAt != 0 {
				t.Fatalf("clock not paused: %+v", paused)
			}
			if got := paused.remaining[types.SeatA]; got <= 0 || got >= 60_000 {
				t.Fatalf("elapsed time not frozen, remaining=%d", got)
			}
			if s.turnBasedClockTimers[test.room.ID] != nil {
				t.Fatal("paused clock still has a server timer")
			}

			test.resume(test.room)
			resumed := test.fields(test.room)
			if resumed.moveDeadlineAt <= nowMs() || resumed.clockDeadlineAt <= nowMs() {
				t.Fatalf("clock not resumed: %+v", resumed)
			}
			s.clearTurnBasedClockTimer(test.room.ID)
		})
	}
}
