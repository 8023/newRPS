package server

import "testing"

func TestActiveRoomsOwnedByCountsOnlyLiveRoomsForThatOwner(t *testing.T) {
	s := &Server{rooms: map[string]*RoomState{
		"r1": {ID: "r1", OwnerID: "alice"},
		"r2": {ID: "r2", OwnerID: "alice"},
		"r3": {ID: "r3", OwnerID: "bob"},
	}}
	if got := s.activeRoomsOwnedBy("alice"); got != 2 {
		t.Fatalf("activeRoomsOwnedBy(alice) = %d, want 2", got)
	}
	if got := s.activeRoomsOwnedBy("bob"); got != 1 {
		t.Fatalf("activeRoomsOwnedBy(bob) = %d, want 1", got)
	}
	if got := s.activeRoomsOwnedBy("carol"); got != 0 {
		t.Fatalf("activeRoomsOwnedBy(carol) = %d, want 0", got)
	}
	// 房间关闭后会从 s.rooms 删除；模拟关闭后计数应相应下降。
	delete(s.rooms, "r1")
	if got := s.activeRoomsOwnedBy("alice"); got != 1 {
		t.Fatalf("after closing r1, activeRoomsOwnedBy(alice) = %d, want 1", got)
	}
}
