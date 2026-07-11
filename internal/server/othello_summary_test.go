package server

import (
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestOthelloSettlementSummaryShort(t *testing.T) {
	state := &types.OthelloState{
		SettlementEvents: []string{
			"甲 本手白给，2 子不结算排位分",
			"乙 本手正常结算：+5",
			"甲 本手白给，1 子不结算排位分（强制白给）",
			"乙 本手上贡，甲 获得 +3",
		},
	}
	got := othelloSettlementSummary(state)
	if !strings.Contains(got, "白给") || !strings.Contains(got, "上贡") {
		t.Fatalf("expected summary, got %q", got)
	}
	if strings.Contains(got, "本手白给，2") || strings.Contains(got, "本手正常结算") {
		t.Fatalf("summary should not include per-move detail: %q", got)
	}
	if strings.Count(got, "；") > 3 {
		t.Fatalf("summary too long: %q", got)
	}
}

func TestPunishmentCompletePending(t *testing.T) {
	s := &Server{players: map[string]*PlayerState{}}
	room := &RoomState{
		Settings:          types.RoomSettings{EnablePunishment: true, RequireOpponentConfirm: true, PunishmentSource: "system"},
		PunishedPlayerIDs: []string{"p1"},
		Proofs: []types.PunishmentProof{
			{PlayerID: "p1", Status: "pending", Text: "done"},
		},
		RoundHistory: []types.RoundHistoryItem{{
			PunishmentTasks: []types.PunishmentTask{{PlayerID: "p1", TaskText: "task"}},
		}},
		Seats: map[types.SeatKey]SeatOccupant{},
	}
	if s.punishmentComplete(room) {
		t.Fatal("pending proof must not complete punishment")
	}
	room.Proofs[0].Status = "approved"
	room.Proofs[0].ConfirmedBy = "p2"
	if !s.punishmentComplete(room) {
		t.Fatal("approved proof should complete")
	}
}
