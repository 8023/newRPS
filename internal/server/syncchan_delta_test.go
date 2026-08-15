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

func TestLobbyRoomCountsDeltaPreservesZero(t *testing.T) {
	s := &Server{}
	lobby := types.LobbySnapshot{
		Players: map[string]types.LobbyPlayer{},
		Rooms: map[string]types.LobbyRoomInfo{
			"r1": {
				ID: "r1", GameID: types.GameRPS, Name: "人数归零",
				Players: 1, Spectators: 2,
				Versus: map[types.SeatKey]any{types.SeatA: nil, types.SeatB: nil},
				Tags:   []string{},
			},
		},
		NormalLeaderboard: []types.LobbyPlayer{},
		RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions:       []types.Suggestion{},
		LobbyChat:         []types.ChatMessage{},
	}
	if env, _, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby); err != nil || env == nil {
		t.Fatalf("initial full: env=%+v err=%v", env, err)
	}

	room := lobby.Rooms["r1"]
	room.Players = 0
	room.Spectators = 0
	lobby.Rooms = map[string]types.LobbyRoomInfo{"r1": room}
	env, _, err := s.buildStateEnvelope("lobby:update", channelLobby(), lobby)
	if err != nil {
		t.Fatal(err)
	}
	if env == nil || env.Kind != wire.PayloadKind_KIND_DELTA || env.Delta == nil {
		t.Fatalf("expected count delta, got %+v", env)
	}

	want := map[string]bool{"/rooms/0/players": false, "/rooms/0/spectators": false}
	for _, op := range env.Delta.Ops {
		if _, ok := want[op.Path]; !ok {
			continue
		}
		if op.Remove || op.Value == nil || op.Value.AsInterface() != float64(0) {
			t.Fatalf("%s must be an explicit numeric zero, got remove=%v value=%v", op.Path, op.Remove, op.Value)
		}
		want[op.Path] = true
	}
	for path, found := range want {
		if !found {
			t.Fatalf("missing zero-value delta for %s: %+v", path, env.Delta.Ops)
		}
	}
}

func TestChessMoveDeadlineChangesReachStateChannel(t *testing.T) {
	s := &Server{}
	state := freshChessState(types.SeatA)
	state.MoveDeadlineAt = 1_000
	room := types.RoomSnapshot{
		ID: "chess-clock", Settings: types.RoomSettings{GameID: types.GameChess, ChessMoveSeconds: 30},
		Seats: map[types.SeatKey]any{}, Ready: map[types.SeatKey]bool{}, Choices: map[types.SeatKey]any{},
		Score: map[types.SeatKey]int{}, SeatedScore: map[types.SeatKey]int{}, SeatStats: map[types.SeatKey]types.SeatStats{},
		Spectators: []types.PublicPlayer{}, PunishedPlayerIDs: []string{}, Proofs: []types.PunishmentProof{},
		RoundHistory: []types.RoundHistoryItem{}, Chat: []types.ChatMessage{}, Chess: state,
	}
	first, _, err := s.buildStateEnvelope("room:update", channelRoom(room.ID), room)
	if err != nil || first == nil {
		t.Fatalf("initial chess state: env=%+v err=%v", first, err)
	}

	nextState := *state
	nextState.Turn = types.SeatB
	nextState.MoveCount = 1
	nextState.MoveDeadlineAt = 31_000
	room.Chess = &nextState
	second, _, err := s.buildStateEnvelope("room:update", channelRoom(room.ID), room)
	if err != nil || second == nil {
		t.Fatalf("changed chess state: env=%+v err=%v", second, err)
	}
	if second.Hash == first.Hash {
		t.Fatal("turn/deadline change must alter the room state hash")
	}
	_, tree, _, err := buildStateTreeAndFull(channelRoom(room.ID), room)
	if err != nil {
		t.Fatal(err)
	}
	front, ok := tree.(map[string]any)
	if !ok {
		t.Fatalf("front tree type=%T", tree)
	}
	chess, ok := front["chess"].(map[string]any)
	if !ok {
		t.Fatalf("chess tree=%T", front["chess"])
	}
	if got := chess["moveDeadlineAt"]; got != float64(31_000) {
		t.Fatalf("moveDeadlineAt=%v, want 31000; chess=%#v", got, chess)
	}
}
