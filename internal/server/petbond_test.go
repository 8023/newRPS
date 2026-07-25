package server

import (
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestBuildPetBondChainsSplitLong(t *testing.T) {
	// a→b→c→d 四级：应拆成 a-b-c 与 b-c-d（以及可能的 2 人边）
	edges := []types.PetBondEdge{
		{MasterID: "a", PetID: "b"},
		{MasterID: "b", PetID: "c"},
		{MasterID: "c", PetID: "d"},
	}
	chains := buildPetBondChains(edges)
	joined := map[string]bool{}
	for _, c := range chains {
		if len(c) < 2 || len(c) > 3 {
			t.Fatalf("chain length out of range: %v", c)
		}
		joined[strings.Join(c, ">")] = true
	}
	if !joined["a>b>c"] {
		t.Fatalf("expected a>b>c, got %v", chains)
	}
	if !joined["b>c>d"] {
		t.Fatalf("expected b>c>d, got %v", chains)
	}
}

func TestBuildPetBondChainsPair(t *testing.T) {
	edges := []types.PetBondEdge{{MasterID: "x", PetID: "y"}}
	chains := buildPetBondChains(edges)
	if len(chains) != 1 || strings.Join(chains[0], ">") != "x>y" {
		t.Fatalf("unexpected chains: %v", chains)
	}
}

func TestDirectReverseForbidden(t *testing.T) {
	s := &Server{
		players:  map[string]*PlayerState{},
		petBonds: map[string]*petBond{},
	}
	s.cfg.PetBond = types.PetBondConfig{MaxPetsPerMaster: 3, MaxMastersPerPet: 3, MaxTitleLength: 12}
	if err := s.createBond("a", "b"); err != nil {
		t.Fatal(err)
	}
	// b 不能再认 a 为主（直系反向）
	if err := s.createBond("b", "a"); err == nil {
		t.Fatal("expected direct reverse to fail")
	}
	// c 在 a→b→c 之后可以认 a（形成环长 3）
	if err := s.createBond("b", "c"); err != nil {
		t.Fatal(err)
	}
	if err := s.createBond("c", "a"); err != nil {
		t.Fatalf("cycle of 3 should be allowed: %v", err)
	}
}
