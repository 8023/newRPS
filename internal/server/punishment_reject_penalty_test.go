package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestProofRejectionPenaltyPoints(t *testing.T) {
	cases := map[int]int{
		0: 0, 1: 0, 2: 0,
		3: 100, 4: 200, 5: 300, 6: 400,
	}
	for rejectCount, want := range cases {
		if got := proofRejectionPenaltyPoints(rejectCount); got != want {
			t.Fatalf("proofRejectionPenaltyPoints(%d) = %d, want %d", rejectCount, got, want)
		}
	}
}

func TestApplyProofRejectionPenaltyEscalatesFromThirdReject(t *testing.T) {
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "loser"}}
	s := &Server{players: map[string]*PlayerState{"loser": player}}
	room := &RoomState{
		Settings: types.RoomSettings{EnablePunishment: true},
		RoundHistory: []types.RoundHistoryItem{{
			PunishmentTasks: []types.PunishmentTask{{PlayerID: "loser", TaskText: "task"}},
		}},
	}

	// 前两次不通过：不扣分。
	s.applyProofRejectionPenalty(room, "loser")
	s.applyProofRejectionPenalty(room, "loser")
	if player.Stats.RankedPoints != 0 {
		t.Fatalf("expected no penalty within first 2 rejects, got %d", player.Stats.RankedPoints)
	}

	// 第 3 次起按 100/200/300... 累加扣分。
	s.applyProofRejectionPenalty(room, "loser")
	if player.Stats.RankedPoints != -100 {
		t.Fatalf("expected -100 after 3rd reject, got %d", player.Stats.RankedPoints)
	}
	s.applyProofRejectionPenalty(room, "loser")
	if player.Stats.RankedPoints != -300 {
		t.Fatalf("expected -300 after 4th reject, got %d", player.Stats.RankedPoints)
	}
	s.applyProofRejectionPenalty(room, "loser")
	if player.Stats.RankedPoints != -600 {
		t.Fatalf("expected -600 after 5th reject, got %d", player.Stats.RankedPoints)
	}
}

func TestApplyProofRejectionPenaltyResetsPerRound(t *testing.T) {
	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "loser"}}
	s := &Server{players: map[string]*PlayerState{"loser": player}}
	room := &RoomState{
		Settings: types.RoomSettings{EnablePunishment: true},
		RoundHistory: []types.RoundHistoryItem{{
			PunishmentTasks: []types.PunishmentTask{{PlayerID: "loser", TaskText: "task"}},
		}},
	}
	for i := 0; i < 3; i++ {
		s.applyProofRejectionPenalty(room, "loser")
	}
	if player.Stats.RankedPoints != -100 {
		t.Fatalf("expected -100 after 3 rejects in round 1, got %d", player.Stats.RankedPoints)
	}

	// 新一局：buildPunishmentTasks 会生成全新的 PunishmentTask，RejectCount 天然清零。
	room.RoundHistory = []types.RoundHistoryItem{{
		PunishmentTasks: []types.PunishmentTask{{PlayerID: "loser", TaskText: "task2"}},
	}}
	s.applyProofRejectionPenalty(room, "loser")
	s.applyProofRejectionPenalty(room, "loser")
	if player.Stats.RankedPoints != -100 {
		t.Fatalf("expected no additional penalty within first 2 rejects of new round, got %d", player.Stats.RankedPoints)
	}
}
