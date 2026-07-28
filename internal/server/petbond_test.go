package server

import (
	"strings"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestBuildPetBondChainsSplitLong(t *testing.T) {
	// a→b→c→d 四级：不再拆窗，应整链展示为 a-b-c-d，且不重复展示其子区间（a-b-c、b-c-d、a-b 等）。
	edges := []types.PetBondEdge{
		{MasterID: "a", PetID: "b"},
		{MasterID: "b", PetID: "c"},
		{MasterID: "c", PetID: "d"},
	}
	chains := buildPetBondChains(edges)
	if len(chains) != 1 {
		t.Fatalf("expected exactly 1 full chain, got %v", chains)
	}
	if strings.Join(chains[0], ">") != "a>b>c>d" {
		t.Fatalf("expected a>b>c>d, got %v", chains)
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

// TestClearOfflinePetBondRequests 认主/认宠/解除关系申请均要求双方在同意前保持在线：
// 任一方离线应作废该申请，三种类型规则统一。
func TestClearOfflinePetBondRequests(t *testing.T) {
	s := &Server{
		players:  map[string]*PlayerState{},
		petBonds: map[string]*petBond{},
		petBondRequests: map[string]*petBondRequest{
			"seekAsFrom": {ID: "seekAsFrom", Kind: petBondKindSeekMaster, FromID: "leaver", ToID: "other", Status: petBondStatusPending, Approvals: map[string]bool{}},
			"seekAsTo":   {ID: "seekAsTo", Kind: petBondKindSeekPet, FromID: "other", ToID: "leaver", Status: petBondStatusPending, Approvals: map[string]bool{}},
			"unrelated":  {ID: "unrelated", Kind: petBondKindSeekMaster, FromID: "x", ToID: "y", Status: petBondStatusPending, Approvals: map[string]bool{}},
			"release":    {ID: "release", Kind: petBondKindRelease, FromID: "leaver", ToID: "other", MasterID: "leaver", PetID: "other", Status: petBondStatusPending, Approvals: map[string]bool{}},
		},
	}
	if changed := s.clearOfflinePetBondRequests("leaver"); !changed {
		t.Fatal("expected changed=true")
	}
	if _, ok := s.petBondRequests["seekAsFrom"]; ok {
		t.Fatal("seekAsFrom should have been cancelled")
	}
	if _, ok := s.petBondRequests["seekAsTo"]; ok {
		t.Fatal("seekAsTo should have been cancelled")
	}
	if _, ok := s.petBondRequests["unrelated"]; !ok {
		t.Fatal("unrelated request should be untouched")
	}
	if _, ok := s.petBondRequests["release"]; ok {
		t.Fatal("release request should have been cancelled by going offline")
	}
	if changed := s.clearOfflinePetBondRequests("nobody"); changed {
		t.Fatal("expected changed=false for unrelated player")
	}
}
