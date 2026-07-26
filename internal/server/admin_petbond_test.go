package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

func newAdminPetBondTestServer(t *testing.T) *Server {
	s := newTestServer(t)
	s.petBonds = map[string]*petBond{}
	s.petBondRequests = map[string]*petBondRequest{}
	s.adminClientIDs = map[string]struct{}{}
	s.cfg.PetBond = types.PetBondConfig{MaxPetsPerMaster: 3, MaxMastersPerPet: 3, MaxTitleLength: 12}
	return s
}

func lastReplyData(t *testing.T, client *Client) map[string]any {
	t.Helper()
	select {
	case data := <-client.sendCh:
		var env wire.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			t.Fatalf("failed to decode reply: %v", err)
		}
		if env.Err != "" {
			t.Fatalf("expected success reply, got error: %s", env.Err)
		}
		front, err := pbconv.RawBodyToFront(env.RawBody)
		if err != nil {
			t.Fatalf("decode raw body: %v", err)
		}
		m, ok := front.(map[string]any)
		if !ok {
			t.Fatalf("expected map reply, got %T (%#v)", front, front)
		}
		return m
	default:
		t.Fatal("expected a reply on sendCh, got none")
		return nil
	}
}

func TestAdminPetBondGraphRequiresAdminSocket(t *testing.T) {
	s := newAdminPetBondTestServer(t)
	client := newRegistrationTestClient()
	s.onAdminPetBondGraph(client, wsEnvelope{E: "admin:petBondGraph", ID: 1})
	if got := lastReplyErr(t, client); got != "需要管理员权限" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestAdminPetBondAddAndRemove(t *testing.T) {
	s := newAdminPetBondTestServer(t)
	s.players["m1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "m1", Name: "主人"}}
	s.players["p1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "宠物"}}

	client := newRegistrationTestClient()
	s.adminClientIDs[client.id] = struct{}{}

	s.onAdminPetBondAdd(client, wsEnvelope{E: "admin:petBondAdd", ID: 1, D: map[string]any{"masterId": "m1", "petId": "p1"}})
	graph := lastReplyData(t, client)
	edges, _ := graph["edges"].([]any)
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge after add, got %#v", graph["edges"])
	}
	nodes, _ := graph["nodes"].([]any)
	if len(nodes) != 2 {
		t.Fatalf("expected 2 nodes (full PublicPlayer each) after add, got %#v", graph["nodes"])
	}
	names := map[string]bool{}
	for _, n := range nodes {
		nm, _ := n.(map[string]any)
		if nm == nil {
			t.Fatalf("expected node to decode as a PublicPlayer map, got %#v", n)
		}
		if _, ok := nm["genderLabel"]; !ok {
			t.Fatalf("expected node to carry full PublicPlayer fields (genderLabel), got %#v", nm)
		}
		names[nm["name"].(string)] = true
	}
	if !names["主人"] || !names["宠物"] {
		t.Fatalf("expected both player names present, got %#v", names)
	}
	if s.getBond("m1", "p1") == nil {
		t.Fatal("expected bond to exist in memory after admin add")
	}

	// 挂一条针对这条边的待办解除申请，验证管理员强制移除时会一并作废，
	// 不会留下指向已不存在关系的僵尸申请。
	s.petBondRequests["r1"] = &petBondRequest{
		ID: "r1", Kind: petBondKindRelease, FromID: "m1", ToID: "p1",
		MasterID: "m1", PetID: "p1", Status: petBondStatusPending, Approvals: map[string]bool{},
	}

	s.onAdminPetBondRemove(client, wsEnvelope{E: "admin:petBondRemove", ID: 2, D: map[string]any{"masterId": "m1", "petId": "p1"}})
	graph = lastReplyData(t, client)
	edges, _ = graph["edges"].([]any)
	if len(edges) != 0 {
		t.Fatalf("expected 0 edges after remove, got %#v", graph["edges"])
	}
	if s.getBond("m1", "p1") != nil {
		t.Fatal("expected bond to be removed after admin remove")
	}
	if _, ok := s.petBondRequests["r1"]; ok {
		t.Fatal("stale release request tied to the removed edge should have been cancelled")
	}
}

func TestAdminPetBondAddRejectsUnknownPlayer(t *testing.T) {
	s := newAdminPetBondTestServer(t)
	client := newRegistrationTestClient()
	s.adminClientIDs[client.id] = struct{}{}
	s.onAdminPetBondAdd(client, wsEnvelope{E: "admin:petBondAdd", ID: 1, D: map[string]any{"masterId": "ghost", "petId": "also-ghost"}})
	if got := lastReplyErr(t, client); got != "玩家不存在" {
		t.Fatalf("unexpected reply: %q", got)
	}
}

func TestAdminPetBondAddRejectsOverLimit(t *testing.T) {
	s := newAdminPetBondTestServer(t)
	s.cfg.PetBond.MaxPetsPerMaster = 1
	s.players["m1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "m1", Name: "主人"}}
	s.players["p1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "宠物一"}}
	s.players["p2"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p2", Name: "宠物二"}}
	if err := s.createBond("m1", "p1"); err != nil {
		t.Fatalf("seed bond: %v", err)
	}

	client := newRegistrationTestClient()
	s.adminClientIDs[client.id] = struct{}{}
	s.onAdminPetBondAdd(client, wsEnvelope{E: "admin:petBondAdd", ID: 1, D: map[string]any{"masterId": "m1", "petId": "p2"}})
	// 管理员新增关系仍复用 createBond，不绕过数量上限——超限时应报错而不是静默突破。
	if got := lastReplyErr(t, client); got == "" {
		t.Fatal("expected error when exceeding maxPetsPerMaster, got success")
	}
}
