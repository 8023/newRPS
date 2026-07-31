package server

import "testing"

func TestLobbyPlayerInLiveSnapshot(t *testing.T) {
	now := int64(1_000_000_000_000)

	online := &PlayerState{}
	online.Connected = true
	if !lobbyPlayerInLiveSnapshot(online, now) {
		t.Fatal("online player must be in live snapshot")
	}

	recent := &PlayerState{}
	d := now - lobbyLivePlayerWindowMs/2
	recent.DisconnectedAt = &d
	if !lobbyPlayerInLiveSnapshot(recent, now) {
		t.Fatal("recently disconnected player must remain visible")
	}

	stale := &PlayerState{}
	old := now - lobbyLivePlayerWindowMs - 1
	stale.DisconnectedAt = &old
	if lobbyPlayerInLiveSnapshot(stale, now) {
		t.Fatal("long-offline player must not enter live lobby snapshot")
	}

	board := &PlayerState{}
	board.GiveawayBoardText = "hello"
	exp := now + 60_000
	board.GiveawayBoardExpiresAt = &exp
	if !lobbyPlayerInLiveSnapshot(board, now) {
		t.Fatal("active giveaway board should stay in live snapshot")
	}

	expiredBoard := &PlayerState{}
	expiredBoard.GiveawayBoardText = "bye"
	past := now - 1
	expiredBoard.GiveawayBoardExpiresAt = &past
	if lobbyPlayerInLiveSnapshot(expiredBoard, now) {
		t.Fatal("expired giveaway board alone should not keep offline player in snapshot")
	}
}
