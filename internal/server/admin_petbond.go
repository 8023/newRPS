package server

import (
	"sort"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// 管理员「主宠关系」面板：力导向图的节点/边，以及直接增删关系的入口。
// 与玩家自助的 petbond:* 流程完全独立——管理员绕开在线要求/双方审批，直接改写 s.petBonds，
// 仍复用 createBond/removeBond 保证自环/直系反向/数量上限等结构性约束不被绕过。

// 节点直接下发完整 PublicPlayer（而不是精简字段）：前端头像、选中气泡里的性别/称号/
// 白给/名争标签都要用同一份 PlayerBadge 组件渲染，字段必须跟大厅/后台用户列表一致。
type adminPetBondEdge struct {
	MasterID  string `json:"masterId"`
	PetID     string `json:"petId"`
	PetTitle  string `json:"petTitle,omitempty"`
	CreatedAt int64  `json:"createdAt"`
}

type adminPetBondGraph struct {
	Nodes []types.PublicPlayer `json:"nodes"`
	Edges []adminPetBondEdge   `json:"edges"`
}

// buildAdminPetBondGraph 节点只包含当前至少有一条边的玩家（新增关系时另外走
// admin:listPlayers 搜索，不需要把全体玩家都塞进图里）。
func (s *Server) buildAdminPetBondGraph() adminPetBondGraph {
	nodeIDs := map[string]bool{}
	edges := make([]adminPetBondEdge, 0, len(s.petBonds))
	for _, b := range s.petBonds {
		nodeIDs[b.MasterID] = true
		nodeIDs[b.PetID] = true
		edges = append(edges, adminPetBondEdge{
			MasterID: b.MasterID, PetID: b.PetID, PetTitle: b.PetTitle, CreatedAt: b.CreatedAt,
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].MasterID != edges[j].MasterID {
			return edges[i].MasterID < edges[j].MasterID
		}
		return edges[i].PetID < edges[j].PetID
	})
	ids := make([]string, 0, len(nodeIDs))
	for id := range nodeIDs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	nodes := make([]types.PublicPlayer, 0, len(ids))
	for _, id := range ids {
		if p := s.players[id]; p != nil {
			nodes = append(nodes, s.publicPlayer(p))
		} else {
			// 目前没有账号注销路径，这里只是防御性兜底，避免边引用的玩家 id 意外查不到时前端崩掉。
			nodes = append(nodes, types.PublicPlayer{ID: id, Name: "已注销", DisplayName: "已注销"})
		}
	}
	return adminPetBondGraph{Nodes: nodes, Edges: edges}
}

func (s *Server) refreshPlayerSnapshotsForBond(masterID, petID string) {
	if p := s.players[masterID]; p != nil {
		s.refreshPlayerSnapshots(p)
	}
	if p := s.players[petID]; p != nil {
		s.refreshPlayerSnapshots(p)
	}
}

func (s *Server) requireAdminSocket(client *Client, env wsEnvelope) bool {
	if _, ok := s.adminClientIDs[client.id]; ok {
		return true
	}
	client.reply(env.ID, nil, "需要管理员权限")
	return false
}

func (s *Server) onAdminPetBondGraph(client *Client, env wsEnvelope) {
	if !s.requireAdminSocket(client, env) {
		return
	}
	client.reply(env.ID, s.buildAdminPetBondGraph(), "")
}

func (s *Server) onAdminPetBondAdd(client *Client, env wsEnvelope) {
	if !s.requireAdminSocket(client, env) {
		return
	}
	var p struct {
		MasterID string `json:"masterId"`
		PetID    string `json:"petId"`
	}
	_ = decodeD(env, &p)
	masterID, petID := strings.TrimSpace(p.MasterID), strings.TrimSpace(p.PetID)
	if s.players[masterID] == nil || s.players[petID] == nil {
		client.reply(env.ID, nil, "玩家不存在")
		return
	}
	if err := s.createBond(masterID, petID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	s.refreshPlayerSnapshotsForBond(masterID, petID)
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildAdminPetBondGraph(), "")
}

func (s *Server) onAdminPetBondRemove(client *Client, env wsEnvelope) {
	if !s.requireAdminSocket(client, env) {
		return
	}
	var p struct {
		MasterID string `json:"masterId"`
		PetID    string `json:"petId"`
	}
	_ = decodeD(env, &p)
	masterID, petID := strings.TrimSpace(p.MasterID), strings.TrimSpace(p.PetID)
	if s.getBond(masterID, petID) == nil {
		client.reply(env.ID, nil, "关系不存在")
		return
	}
	s.removeBond(masterID, petID)
	// 关系没了，挂在这条边上的解除申请（如果有）也一并作废，避免物是人非的申请留着。
	for _, r := range s.petBondRequests {
		if r.Status == petBondStatusPending && r.Kind == petBondKindRelease &&
			r.MasterID == masterID && r.PetID == petID {
			s.cancelRequest(r, petBondStatusCancelled)
		}
	}
	s.refreshPlayerSnapshotsForBond(masterID, petID)
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildAdminPetBondGraph(), "")
}
