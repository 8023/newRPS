package pbconv

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

// TestLobbyPlayerGiveawayVoteFieldsRoundTrip 守住大厅精简玩家上的投票额度字段。
// 领域类型 types.LobbyPlayer 已有这些字段（前端 me.player 与 lobby.players 合并依赖它们），
// 若 proto LobbyPlayer 漏掉，大厅广播会把额度抹掉，表现为“永远满额度”。
func TestLobbyPlayerGiveawayVoteFieldsRoundTrip(t *testing.T) {
	started := int64(1_700_000_000_000)
	likes, dislikes := 2, 5
	src := types.LobbyPlayer{
		ID:                           "p1",
		Name:                         "测试",
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

	// 经 Lobby 快照 front 形态也应保留（大厅 DELTA/FULL 路径）。
	snap := types.LobbySnapshot{
		Players: map[string]types.LobbyPlayer{"p1": src},
	}
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
	if int(asFloat(p["giveawayVoteLikesThisHour"])) != 2 {
		t.Fatalf("front likes=%#v", p["giveawayVoteLikesThisHour"])
	}
	if int(asFloat(p["giveawayVoteDislikesThisHour"])) != 5 {
		t.Fatalf("front dislikes=%#v", p["giveawayVoteDislikesThisHour"])
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
