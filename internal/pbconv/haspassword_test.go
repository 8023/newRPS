package pbconv

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestLobbyHasPasswordInFrontTree(t *testing.T) {
	snap := types.LobbySnapshot{
		OnlineCount: 0,
		Players:     map[string]types.LobbyPlayer{},
		Rooms: map[string]types.LobbyRoomInfo{
			"r1": {
				ID: "r1", Name: "密房", HasPassword: true,
				Versus: map[types.SeatKey]any{types.SeatA: nil, types.SeatB: nil},
				Tags:   []string{},
			},
		},
		NormalLeaderboard: []types.LobbyPlayer{},
		RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions:       []types.Suggestion{},
		LobbyChat:         []types.ChatMessage{},
	}
	doc, _, _, err := MarshalStateLobby(snap)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := StateDocToFront(doc)
	if err != nil {
		t.Fatal(err)
	}
	m := tree.(map[string]any)
	rooms := m["rooms"].([]any)
	if len(rooms) != 1 {
		t.Fatalf("rooms=%v", rooms)
	}
	r := rooms[0].(map[string]any)
	t.Logf("room keys=%v hasPassword=%v (%T)", keysOf(r), r["hasPassword"], r["hasPassword"])
	if r["hasPassword"] != true {
		t.Fatalf("hasPassword want true, got %#v", r["hasPassword"])
	}
	if r["players"] != float64(0) || r["spectators"] != float64(0) {
		t.Fatalf("zero room counts must survive front-tree conversion: players=%#v spectators=%#v", r["players"], r["spectators"])
	}
}
func keysOf(m map[string]any) []string {
	ks := make([]string, 0, len(m))
	for k := range m {
		ks = append(ks, k)
	}
	return ks
}
