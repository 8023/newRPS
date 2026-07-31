package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestRecordGameOutcomeSyncsTotals(t *testing.T) {
	s := &Server{}
	p := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "A"}}
	s.recordGameOutcome(p, types.GameRPS, "win")
	s.recordGameOutcome(p, types.GameOthello, "loss")
	s.recordGameOutcome(p, types.GameGomoku, "draw")
	if p.GameStats.RPS.Wins != 1 || p.GameStats.Othello.Losses != 1 || p.GameStats.Gomoku.Draws != 1 {
		t.Fatalf("game stats wrong: %+v", p.GameStats)
	}
	if p.Stats.Wins != 1 || p.Stats.Losses != 1 || p.Stats.Draws != 1 {
		t.Fatalf("totals wrong: %+v", p.Stats)
	}
}
