package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestFillAdminChatRoomNames(t *testing.T) {
	s := &Server{
		rooms: map[string]*RoomState{
			"live": {Settings: types.RoomSettings{Name: "现开房间"}},
		},
	}
	msgs := []adminChatMessage{
		{ChatMessage: types.ChatMessage{ID: "1", RoomID: "snap"}, RoomName: "快照名"},
		{ChatMessage: types.ChatMessage{ID: "2", RoomID: "snap"}},
		{ChatMessage: types.ChatMessage{ID: "3", RoomID: "live"}},
		{ChatMessage: types.ChatMessage{ID: "4", RoomID: "gone"}},
		{ChatMessage: types.ChatMessage{ID: "5"}},
	}
	s.fillAdminChatRoomNames(msgs)
	if msgs[0].RoomName != "快照名" {
		t.Fatalf("kept snapshot = %q", msgs[0].RoomName)
	}
	if msgs[1].RoomName != "快照名" {
		t.Fatalf("same-page fill = %q, want 快照名", msgs[1].RoomName)
	}
	if msgs[2].RoomName != "现开房间" {
		t.Fatalf("live room fill = %q, want 现开房间", msgs[2].RoomName)
	}
	if msgs[3].RoomName != "" {
		t.Fatalf("closed room should stay empty, got %q", msgs[3].RoomName)
	}
	if msgs[4].RoomName != "" {
		t.Fatalf("lobby should stay empty, got %q", msgs[4].RoomName)
	}
}
