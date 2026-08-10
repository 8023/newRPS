package pbconv

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestLobbyPlayerHandMappedFields 守住 lobbyPlayerToProto 里手写映射的标量/子消息字段
// （不走通用 JSONCamelToProto，漏映射不会在编译期报错，只能靠测试兜底）。
func TestLobbyPlayerHandMappedFields(t *testing.T) {
	likes, streak := 3, 7
	val := 12.5
	src := types.LobbyPlayer{
		ID: "p2", Name: "手写映射", GenderID: "g1", GenderLabel: "女",
		FactionID: "f1", FactionLabel: "阵营", DisplayName: "展示名",
		Connected: true, AvatarURL: "https://x/a.webp",
		GiveawayEnabled: boolPtr(true), GiveawayValue: &val,
		GiveawayBoardLikes: &likes, ExtremeWinStreak: &streak,
		BondMasterEnabled: boolPtr(true),
		Stats: types.LobbyStats{
			Wins: 1, Losses: 2, Draws: 3, RankedPoints: 100, Title: "称号",
			SelfTitle: "自设", TitleSource: "self",
		},
		GameStats: types.GameStats{
			RPS: types.GameWLD{Wins: 5, Losses: 1, Draws: 0},
		},
	}
	pb, err := lobbyPlayerToProto(src)
	if err != nil {
		t.Fatal(err)
	}
	if pb.Id != "p2" || pb.Name != "手写映射" || !pb.Connected {
		t.Fatalf("identity: %+v", pb)
	}
	if pb.GiveawayValue != 12.5 || pb.GiveawayBoardLikes != 3 || pb.ExtremeWinStreak != 7 {
		t.Fatalf("scalars: value=%v likes=%d streak=%d", pb.GiveawayValue, pb.GiveawayBoardLikes, pb.ExtremeWinStreak)
	}
	if pb.Stats == nil || pb.Stats.Wins != 1 || pb.Stats.SelfTitle != "自设" {
		t.Fatalf("stats: %+v", pb.Stats)
	}
	if pb.GameStats == nil || pb.GameStats.Rps == nil || pb.GameStats.Rps.Wins != 5 {
		t.Fatalf("gameStats: %+v", pb.GameStats)
	}
	if !pb.BondMasterEnabled {
		t.Fatal("bond master")
	}
	// 空 GameStats 不应编码子消息
	empty, err := lobbyPlayerToProto(types.LobbyPlayer{ID: "e", Stats: types.LobbyStats{Wins: 9}})
	if err != nil {
		t.Fatal(err)
	}
	if empty.GameStats != nil {
		t.Fatalf("empty gameStats should be nil, got %+v", empty.GameStats)
	}
	if empty.Stats == nil || empty.Stats.Wins != 9 {
		t.Fatalf("stats wins: %+v", empty.Stats)
	}
}

func boolPtr(v bool) *bool { return &v }
func TestLobbyPlayerGiveawayVoteFieldsRoundTrip(t *testing.T) {
	started := int64(1_700_000_000_000)
	likes, dislikes := 2, 5
	src := types.LobbyPlayer{
		ID: "p1", Name: "测试",
		GiveawayVoteWindowStartedAt:  &started,
		GiveawayVoteLikesThisHour:    &likes,
		GiveawayVoteDislikesThisHour: &dislikes,
	}
	pb, err := lobbyPlayerToProto(src)
	if err != nil {
		t.Fatal(err)
	}
	if pb.GiveawayVoteWindowStartedAt != started {
		t.Fatalf("window started: got %d want %d", pb.GiveawayVoteWindowStartedAt, started)
	}
	if pb.GiveawayVoteLikesThisHour != 2 {
		t.Fatalf("likes this hour: got %d", pb.GiveawayVoteLikesThisHour)
	}
	if pb.GiveawayVoteDislikesThisHour != 5 {
		t.Fatalf("dislikes this hour: got %d", pb.GiveawayVoteDislikesThisHour)
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
		t.Fatalf("players: %#v", players)
	}
	p, _ := players[0].(map[string]any)
	if p == nil {
		t.Fatalf("player not map: %#v", players[0])
	}
	if int(asFloat(p["giveawayVoteLikesThisHour"])) != 2 || int(asFloat(p["giveawayVoteDislikesThisHour"])) != 5 {
		t.Fatalf("front vote fields: %#v", p)
	}
	if asFloat(p["giveawayVoteWindowStartedAt"]) != float64(started) {
		t.Fatalf("front window=%#v", p["giveawayVoteWindowStartedAt"])

	}
}

// TestChatMessageMentionsAndSeqOnTypedPath 确保 typed ChatMessage（chat:append /
// lobbyChat 等）不再丢掉 mentions/seq。chat:new 走动态 RAW 另有测试覆盖。
func TestChatMessageMentionsAndSeqOnTypedPath(t *testing.T) {
	msg := types.ChatMessage{
		ID: "m1", PlayerID: "p1", Author: "阿明", Text: "hi @小红",
		At: 123, Mentions: []string{"pHong"}, Seq: 42,
	}
	body, err := BuildRawBody("chat:append", msg)
	if err != nil {
		t.Fatal(err)
	}
	if body.GetChat() == nil {
		t.Fatal("expected typed chat body")
	}
	if body.GetChat().Seq != 42 {
		t.Fatalf("seq=%d", body.GetChat().Seq)
	}
	if len(body.GetChat().Mentions) != 1 || body.GetChat().Mentions[0] != "pHong" {
		t.Fatalf("mentions=%v", body.GetChat().Mentions)
	}
	front, err := RawBodyToFront(body)
	if err != nil {
		t.Fatal(err)
	}
	m, ok := front.(map[string]any)
	if !ok {
		// ProtoToCamelMap may return map from chat message
		t.Fatalf("front type %T", front)
	}
	mentions, _ := m["mentions"].([]any)
	if len(mentions) != 1 || mentions[0] != "pHong" {
		t.Fatalf("front mentions=%#v full=%#v", m["mentions"], m)
	}
	if int(asFloat(m["seq"])) != 42 {
		t.Fatalf("front seq=%#v", m["seq"])
	}
}
