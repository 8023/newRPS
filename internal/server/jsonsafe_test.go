package server

import (
	"encoding/json"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestSanitizeRoomSnapshotNeverNullArrays(t *testing.T) {
	snap := sanitizeRoomSnapshot(types.RoomSnapshot{
		ID:   "r1",
		Code: "DM-TEST",
		Seats: map[types.SeatKey]any{
			types.SeatA: nil,
			// B missing on purpose
		},
		// all slices left nil
	})
	raw, err := json.Marshal(snap)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"punishedPlayerIds", "proofs", "roundHistory", "chat", "spectators"} {
		v, ok := m[key]
		if !ok {
			t.Fatalf("missing key %s", key)
		}
		if v == nil {
			t.Fatalf("%s must not be null", key)
		}
		arr, ok := v.([]any)
		if !ok {
			t.Fatalf("%s must be array, got %T", key, v)
		}
		if arr == nil {
			t.Fatalf("%s array must not be nil", key)
		}
	}
	seats := m["seats"].(map[string]any)
	if _, ok := seats["A"]; !ok {
		t.Fatal("seats.A missing")
	}
	if _, ok := seats["B"]; !ok {
		t.Fatal("seats.B missing")
	}
	choices := m["choices"]
	if choices == nil {
		t.Fatal("choices must not be null")
	}
}

func TestSanitizeRoundHistoryItem(t *testing.T) {
	item := sanitizeRoundHistoryItem(types.RoundHistoryItem{ID: "h1", Round: 1})
	raw, _ := json.Marshal(item)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, key := range []string{"punishmentTasks", "punishedNames", "proofs"} {
		if m[key] == nil {
			t.Fatalf("%s null", key)
		}
	}
}

func TestSanitizeLobbySnapshot(t *testing.T) {
	snap := sanitizeLobbySnapshot(types.LobbySnapshot{})
	raw, _ := json.Marshal(snap)
	var m map[string]any
	_ = json.Unmarshal(raw, &m)
	for _, key := range []string{"players", "rooms", "suggestions", "lobbyChat", "normalLeaderboard", "rankedLeaderboard"} {
		if m[key] == nil {
			t.Fatalf("%s null", key)
		}
	}
}
