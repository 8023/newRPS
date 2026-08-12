package pbconv

import (
	"encoding/json"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
)

func TestFillPlayerDefaults_BoolAndNumberZero(t *testing.T) {
	// wire 形态：false/0 被 EmitUnpopulated:false 连 key 一起丢掉，只剩其它字段
	in := map[string]any{
		"id":           "p1",
		"connected":    true,
		"isAdmin_todo": true, // 无关字段，确认不会被误伤
	}
	out, ok := fillPlayerDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["nameWarEnabled"] != false {
		t.Fatalf("nameWarEnabled want false, got %#v", out["nameWarEnabled"])
	}
	if out["isAdmin"] != false {
		t.Fatalf("isAdmin want false, got %#v", out["isAdmin"])
	}
	if out["disconnectedAt"] != float64(0) {
		t.Fatalf("disconnectedAt want 0, got %#v", out["disconnectedAt"])
	}
	if out["extremeWinStreak"] != float64(0) {
		t.Fatalf("extremeWinStreak want 0, got %#v", out["extremeWinStreak"])
	}
}

func TestFillPlayerDefaults_KeepsRealValues(t *testing.T) {
	in := map[string]any{
		"disconnectedAt": float64(123),
		"nameWarEnabled": true,
	}
	out, ok := fillPlayerDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["disconnectedAt"] != float64(123) {
		t.Fatalf("disconnectedAt=%#v", out["disconnectedAt"])
	}
	if out["nameWarEnabled"] != true {
		t.Fatalf("nameWarEnabled=%#v", out["nameWarEnabled"])
	}
}

func TestFillRoomSettingsDefaults_AllowProofImageMissingIsFalse(t *testing.T) {
	// handlers_room.go 建房时已把 nil 归一化成具体值，线上房间不会真正"未设置"；
	// false 是它的零值，缺省应还原为 false（true 是非零值，protojson 永远不会丢）。
	in := map[string]any{"name": "room", "enableRanked": true}
	out, ok := fillRoomSettingsDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["allowProofImage"] != false {
		t.Fatalf("allowProofImage want false when missing, got %#v", out["allowProofImage"])
	}
}

func TestFillRoomSettingsDefaults_KeepsExplicitTrue(t *testing.T) {
	in := map[string]any{"allowProofImage": true}
	out, ok := fillRoomSettingsDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["allowProofImage"] != true {
		t.Fatalf("allowProofImage want true, got %#v", out["allowProofImage"])
	}
}

func TestFillRoundHistoryItemDefaults_ZeroAndNestedProofs(t *testing.T) {
	in := map[string]any{
		"id":     "h1",
		"ranked": false,
		"proofs": []any{
			map[string]any{"playerId": "p1"},
		},
		"punishmentTasks": []any{
			map[string]any{"playerId": "p1"},
		},
	}
	out, ok := fillRoundHistoryItemDefaults(in).(map[string]any)
	if !ok {
		t.Fatal("not map")
	}
	if out["stake"] != float64(0) || out["rankMultiplier"] != float64(0) || out["effectiveStake"] != float64(0) {
		t.Fatalf("stake/rankMultiplier/effectiveStake want 0, got %#v", out)
	}
	proofs := out["proofs"].([]any)
	pm := proofs[0].(map[string]any)
	if pm["reviewedAt"] != float64(0) {
		t.Fatalf("nested proof reviewedAt want 0, got %#v", pm)
	}
	tasks := out["punishmentTasks"].([]any)
	tm := tasks[0].(map[string]any)
	if tm["backgroundOpacity"] != float64(0) {
		t.Fatalf("nested task backgroundOpacity want 0, got %#v", tm)
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

func TestRoomProtoToFront_ReadyScoreAndSettings(t *testing.T) {
	room := &wire.RoomSnapshot{
		Id: "r1",
		Settings: &wire.RoomSettings{
			Name:            "t",
			AllowProofImage: false,
			EnableRanked:    true,
			Stake:           1,
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
	if settings["allowProofImage"] != false {
		t.Fatalf("allowProofImage want false, got %#v", settings["allowProofImage"])
	}
	ready, _ := front["ready"].(map[string]any)
	if ready["A"] != false || ready["B"] != true {
		t.Fatalf("ready=%#v", ready)
	}
	score, _ := front["score"].(map[string]any)
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
	r := rooms[0].(map[string]any)
	if r["hasPassword"] != true {
		t.Fatalf("hasPassword want true, got %#v", r["hasPassword"])
	}
}

func TestLobbyPunishmentSelectionSurvivesWire(t *testing.T) {
	snap := types.LobbySnapshot{
		Players: map[string]types.LobbyPlayer{},
		Rooms: map[string]types.LobbyRoomInfo{
			"r1": {
				ID: "r1", EnablePunishment: true, PunishmentSource: "series",
				PunishmentTagsIncluded: []string{"truth"},
				PunishmentTagsExcluded: []string{"water"}, PunishmentSeriesID: "s1",
				Versus: map[types.SeatKey]any{types.SeatA: nil, types.SeatB: nil}, Tags: []string{},
			},
		},
		NormalLeaderboard: []types.LobbyPlayer{}, RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions: []types.Suggestion{}, LobbyChat: []types.ChatMessage{},
	}
	doc, _, _, err := MarshalStateLobby(snap)
	if err != nil {
		t.Fatal(err)
	}
	tree, err := StateDocToFront(doc)
	if err != nil {
		t.Fatal(err)
	}
	rooms := tree.(map[string]any)["rooms"].([]any)
	r := rooms[0].(map[string]any)
	if r["punishmentSource"] != "series" || r["punishmentSeriesId"] != "s1" {
		t.Fatalf("selection lost: %#v", r)
	}
	included, _ := r["punishmentTagsIncluded"].([]any)
	excluded, _ := r["punishmentTagsExcluded"].([]any)
	if len(included) != 1 || included[0] != "truth" || len(excluded) != 1 || excluded[0] != "water" {
		t.Fatalf("tag selection lost: included=%#v excluded=%#v", included, excluded)
	}
}
