package pbconv

import (
	"encoding/json"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestTitleColorsAlwaysPresentInFrontTree 守住大厅 CRC 不因空 titleColors 分叉：
// lobbyStatsToProto 必须恒下发非 nil TitleColors，否则 Go 哈希树缺键、前端 materialize
// 为 null/{}，hash 永不相等 → 无限 sync:full。
func TestTitleColorsAlwaysPresentInFrontTree(t *testing.T) {
	src := types.LobbyPlayer{
		ID: "p1", Name: "x",
		Stats: types.LobbyStats{Wins: 1, Title: "x", TitleSource: "system"},
	}
	pb, err := lobbyPlayerToProto(src)
	if err != nil {
		t.Fatal(err)
	}
	if pb.Stats == nil || pb.Stats.TitleColors == nil {
		t.Fatalf("TitleColors must be non-nil empty message, stats=%+v", pb.Stats)
	}

	snap := types.LobbySnapshot{Players: map[string]types.LobbyPlayer{"p1": src}}
	lpb, err := LobbyToProto(snap)
	if err != nil {
		t.Fatal(err)
	}
	front, err := LobbyProtoToFront(lpb)
	if err != nil {
		t.Fatal(err)
	}
	players, _ := front["players"].([]any)
	if len(players) != 1 {
		t.Fatalf("players: %#v", front["players"])
	}
	p, _ := players[0].(map[string]any)
	stats, _ := p["stats"].(map[string]any)
	raw, _ := json.Marshal(stats)
	if _, ok := stats["titleColors"]; !ok {
		t.Fatalf("titleColors key MISSING from front tree — CRC fork; stats=%s", raw)
	}
}

func TestTitleColorsWithValuesRoundTrip(t *testing.T) {
	src := types.LobbyPlayer{
		ID: "p2",
		Stats: types.LobbyStats{
			Title: "主人", TitleSource: "master",
			TitleColors: types.GenderColors{
				TextColor: "#fff", BackgroundColor: "#000", BorderColor: "#111",
			},
		},
	}
	pb, err := lobbyPlayerToProto(src)
	if err != nil {
		t.Fatal(err)
	}
	tc := pb.Stats.TitleColors
	if tc == nil || tc.TextColor != "#fff" || tc.BackgroundColor != "#000" || tc.BorderColor != "#111" {
		t.Fatalf("titleColors: %+v", tc)
	}
}
