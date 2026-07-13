package pbconv

import (
	"encoding/json"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
)

func TestResolvePresence_AllowProofImageFalse(t *testing.T) {
	// protojson 省略 false 后只剩 hasAllowProofImage
	in := map[string]any{
		"name":                "room",
		"hasAllowProofImage":  true,
		"enableRanked":        true,
	}
	out, ok := resolvePresenceFlags(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if _, has := out["hasAllowProofImage"]; has {
		t.Fatalf("presence flag should be stripped: %#v", out)
	}
	if out["allowProofImage"] != false {
		t.Fatalf("allowProofImage want false, got %#v", out["allowProofImage"])
	}
}

func TestResolvePresence_KeepHasPassword(t *testing.T) {
	in := map[string]any{
		"id":          "r1",
		"hasPassword": true,
		"name":        "密房",
	}
	out, ok := resolvePresenceFlags(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["hasPassword"] != true {
		t.Fatalf("hasPassword must stay true, got %#v", out)
	}
	if _, has := out["password"]; has {
		t.Fatalf("must not invent password field: %#v", out)
	}
}

func TestResolvePresence_StripWhenCompanionPresent(t *testing.T) {
	in := map[string]any{
		"disconnectedAt":    float64(123),
		"hasDisconnectedAt": true,
		"nameWarEnabled":    true,
		"hasNameWarEnabled": true,
	}
	out, ok := resolvePresenceFlags(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if _, has := out["hasDisconnectedAt"]; has {
		t.Fatal("hasDisconnectedAt should be stripped")
	}
	if _, has := out["hasNameWarEnabled"]; has {
		t.Fatal("hasNameWarEnabled should be stripped")
	}
	if out["disconnectedAt"] != float64(123) {
		t.Fatalf("disconnectedAt=%#v", out["disconnectedAt"])
	}
	if out["nameWarEnabled"] != true {
		t.Fatalf("nameWarEnabled=%#v", out["nameWarEnabled"])
	}
}

func TestResolvePresence_RestoreNumericZero(t *testing.T) {
	in := map[string]any{
		"hasGiveawayValue":    true,
		"hasExtremeWinStreak": true,
	}
	out, ok := resolvePresenceFlags(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["giveawayValue"] != float64(0) {
		t.Fatalf("giveawayValue=%#v", out["giveawayValue"])
	}
	if out["extremeWinStreak"] != float64(0) {
		t.Fatalf("extremeWinStreak=%#v", out["extremeWinStreak"])
	}
}

func TestPairsMaps_OmittedFalseAndZero(t *testing.T) {
	// wire 形态：false/0 被省略
	readyIn := []any{
		map[string]any{"key": "A"},
		map[string]any{"key": "B", "value": true},
	}
	ready := pairsToBoolMap(readyIn)
	if ready["A"] != false {
		t.Fatalf("ready.A want false, got %#v", ready["A"])
	}
	if ready["B"] != true {
		t.Fatalf("ready.B want true, got %#v", ready["B"])
	}

	scoreIn := []any{
		map[string]any{"key": "A"},
		map[string]any{"key": "B", "value": float64(1)},
	}
	score := pairsToIntMap(scoreIn)
	if score["A"] != float64(0) {
		t.Fatalf("score.A want 0, got %#v", score["A"])
	}
	if score["B"] != float64(1) {
		t.Fatalf("score.B want 1, got %#v", score["B"])
	}
}

func TestRoomProtoToFront_ReadyScoreAndProofSetting(t *testing.T) {
	room := &wire.RoomSnapshot{
		Id: "r1",
		Settings: &wire.RoomSettings{
			Name:               "t",
			AllowProofImage:    false,
			HasAllowProofImage: true,
			EnableBot:          false,
			EnableRanked:       true,
			Stake:              1,
		},
		Ready: []*wire.BoolPair{{Key: "A", Value: false}, {Key: "B", Value: true}},
		Score: []*wire.IntPair{{Key: "A", Value: 0}, {Key: "B", Value: 1}},
	}
	front, err := RoomProtoToFront(room)
	if err != nil {
		t.Fatal(err)
	}
	settings, _ := front["settings"].(map[string]any)
	if settings == nil {
		t.Fatalf("no settings: %#v", front)
	}
	if _, has := settings["hasAllowProofImage"]; has {
		t.Fatalf("hasAllowProofImage should be stripped: %#v", settings)
	}
	if settings["allowProofImage"] != false {
		t.Fatalf("allowProofImage want false, got %#v", settings["allowProofImage"])
	}
	ready, _ := front["ready"].(map[string]any)
	if ready["A"] != false || ready["B"] != true {
		t.Fatalf("ready=%#v", ready)
	}
	score, _ := front["score"].(map[string]any)
	// JSON number
	if score["A"] != float64(0) && score["A"] != 0 {
		t.Fatalf("score.A=%#v", score["A"])
	}
	if score["B"] != float64(1) && score["B"] != 1 {
		t.Fatalf("score.B=%#v", score["B"])
	}
	b, _ := json.Marshal(front)
	t.Logf("front=%s", b)
}

func TestLobbyHasPasswordStillPresent(t *testing.T) {
	snap := types.LobbySnapshot{
		OnlineCount: 0,
		Players:     map[string]types.LobbyPlayer{},
		Rooms: map[string]types.LobbyRoomInfo{
			"r1": {
				ID: "r1", Code: "ABC", Name: "密房", HasPassword: true,
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
	r := rooms[0].(map[string]any)
	if r["hasPassword"] != true {
		t.Fatalf("hasPassword want true, got %#v", r["hasPassword"])
	}
}
