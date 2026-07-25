package server

import (
	"testing"
)

func TestAccumulateClientOnlineMs(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Stats.TotalOnlineMs = 1000
	s.players["p1"] = p
	c := &Client{id: "sock1", playerID: "p1", connectedAt: 10_000}
	s.accumulateClientOnlineMs(c, 15_000)
	if p.Stats.TotalOnlineMs != 6000 {
		t.Fatalf("TotalOnlineMs=%d want 6000", p.Stats.TotalOnlineMs)
	}
	// no player id: no change
	c2 := &Client{id: "sock2", connectedAt: 1}
	s.accumulateClientOnlineMs(c2, 999)
	if p.Stats.TotalOnlineMs != 6000 {
		t.Fatalf("guest should not change: %d", p.Stats.TotalOnlineMs)
	}
}

func TestEffectiveOnlineMsIncludesCurrentSession(t *testing.T) {
	s := &Server{
		players: map[string]*PlayerState{},
		clients: map[string]*Client{},
	}
	p := &PlayerState{}
	p.ID = "p1"
	p.Connected = true
	p.SocketID = "sock1"
	p.Stats.TotalOnlineMs = 1000
	s.players["p1"] = p
	// connectedAt 1 hour ago — but we use fixed math via stubbing is hard; just check base when offline
	p.Connected = false
	if got := s.effectiveOnlineMs(p); got != 1000 {
		t.Fatalf("offline effective=%d want 1000", got)
	}
	p.Connected = true
	s.clients["sock1"] = &Client{id: "sock1", connectedAt: nowMs() - 5000}
	got := s.effectiveOnlineMs(p)
	if got < 5000 || got > 7000 {
		t.Fatalf("online effective=%d want ~6000", got)
	}
}
