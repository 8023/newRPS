package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestRecordGameOutcomeSyncsTotals(t *testing.T) {
	p := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "A"}}
	recordGameOutcome(p, types.GameRPS, "win")
	recordGameOutcome(p, types.GameOthello, "loss")
	recordGameOutcome(p, types.GameGomoku, "draw")
	if p.GameStats.RPS.Wins != 1 || p.GameStats.Othello.Losses != 1 || p.GameStats.Gomoku.Draws != 1 {
		t.Fatalf("game stats wrong: %+v", p.GameStats)
	}
	if p.Stats.Wins != 1 || p.Stats.Losses != 1 || p.Stats.Draws != 1 {
		t.Fatalf("totals wrong: %+v", p.Stats)
	}
}

func TestMigrateGameStatsFromLegacyOthello(t *testing.T) {
	item := persistedPlayer{
		Stats: types.PublicStats{Wins: 10, Losses: 4, Draws: 2},
		LegacyOthelloStats: &struct {
			Wins     int `json:"wins"`
			Losses   int `json:"losses"`
			Draws    int `json:"draws"`
			Games    int `json:"games"`
			Captured int `json:"captured"`
			Lost     int `json:"lost"`
		}{Wins: 3, Losses: 1, Draws: 0, Games: 4, Captured: 99, Lost: 10},
	}
	gs := migrateGameStats(item)
	if gs.Othello.Wins != 3 || gs.Othello.Losses != 1 {
		t.Fatalf("othello migrate: %+v", gs.Othello)
	}
	if gs.RPS.Wins != 7 || gs.RPS.Losses != 3 || gs.RPS.Draws != 2 {
		t.Fatalf("rps residual migrate: %+v", gs.RPS)
	}
	// 不再保留吃子字段
	w, l, d := gs.Total()
	if w != 10 || l != 4 || d != 2 {
		t.Fatalf("total after migrate = %d/%d/%d", w, l, d)
	}
}
