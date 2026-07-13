package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/delta"
	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
)

func TestBuildStateEnvelopeDeltaAndHash(t *testing.T) {
	s := &Server{}
	// 两帧大厅：仅 onlineCount / 玩家 connected 变化
	lobby1 := types.LobbySnapshot{
		OnlineCount: 1,
		Players: map[string]types.LobbyPlayer{
			"p1": {ID: "p1", Name: "甲", Connected: true, DisplayName: "甲"},
		},
		Rooms:             map[string]types.LobbyRoomInfo{},
		NormalLeaderboard: []types.LobbyPlayer{},
		RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions:       []types.Suggestion{},
		LobbyChat:         []types.ChatMessage{},
	}
	env1, _, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby1)
	if err != nil {
		t.Fatal(err)
	}
	if env1 == nil || env1.Kind != wire.PayloadKind_KIND_FULL {
		t.Fatalf("first should be FULL, got %+v", env1)
	}
	if env1.Hash == "" || env1.FullState == nil {
		t.Fatal("missing hash or full state")
	}

	lobby2 := lobby1
	lobby2.OnlineCount = 0
	p := lobby2.Players["p1"]
	p.Connected = false
	lobby2.Players = map[string]types.LobbyPlayer{"p1": p}

	env2, _, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby2)
	if err != nil {
		t.Fatal(err)
	}
	if env2 == nil {
		t.Fatal("expected second envelope")
	}
	if env2.Kind != wire.PayloadKind_KIND_DELTA {
		t.Fatalf("expected DELTA, got %v full=%v", env2.Kind, env2.FullState != nil)
	}
	if env2.Delta == nil || len(env2.Delta.Ops) == 0 {
		t.Fatal("expected delta ops")
	}
	if env2.Hash == "" || env2.Hash == env1.Hash {
		t.Fatalf("hash should change: %s vs %s", env1.Hash, env2.Hash)
	}

	// 验证：从 lobby1 树 apply 到 lobby2 后哈希与 env2 一致
	_, tree1, h1, err := buildStateTreeAndFull(channelLobby(), lobby1)
	if err != nil {
		t.Fatal(err)
	}
	_, tree2, h2, err := buildStateTreeAndFull(channelLobby(), lobby2)
	if err != nil {
		t.Fatal(err)
	}
	ops, err := delta.Diff(tree1, tree2)
	if err != nil {
		t.Fatal(err)
	}
	got, err := delta.Apply(tree1, ops)
	if err != nil {
		t.Fatal(err)
	}
	hg, err := delta.Hash(got)
	if err != nil {
		t.Fatal(err)
	}
	if hg != h2 {
		t.Fatalf("apply hash %s want %s (h1=%s)", hg, h2, h1)
	}
	// 服务端发出的 hash 应对齐 tree2
	if env2.Hash != h2 {
		t.Fatalf("env hash %s want tree2 %s", env2.Hash, h2)
	}
}

func TestBuildStateEnvelopeNoChange(t *testing.T) {
	s := &Server{}
	lobby := types.LobbySnapshot{
		OnlineCount:       0,
		Players:           map[string]types.LobbyPlayer{},
		Rooms:             map[string]types.LobbyRoomInfo{},
		NormalLeaderboard: []types.LobbyPlayer{},
		RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions:       []types.Suggestion{},
		LobbyChat:         []types.ChatMessage{},
	}
	env1, _, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby)
	if err != nil || env1 == nil {
		t.Fatalf("full: %v %v", env1, err)
	}
	env2, n, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby)
	if err != nil {
		t.Fatal(err)
	}
	if env2 != nil || n != 0 {
		t.Fatalf("expected no broadcast on identical state, got %+v n=%d", env2, n)
	}
}
