package server

import (
	"fmt"
	"sort"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// 申请类型：
//   seek_master — 宠物申请认某人为主人（from=宠物 to=新主人）
//   seek_pet    — 主人申请认某人为宠物（from=主人 to=宠物）
//   release     — 任一方申请解除已有关系（from=发起人，master_id/pet_id 标明边）
const (
	petBondKindSeekMaster = "seek_master"
	petBondKindSeekPet    = "seek_pet"
	petBondKindRelease    = "release"
	petBondStatusPending  = "pending"
	petBondStatusDone     = "done"
	petBondStatusCancelled = "cancelled"
)

// petBond 内存中的一条主人→宠物边。
type petBond struct {
	MasterID       string
	PetID          string
	PetTitle       string
	TitleUpdatedAt int64
	CreatedAt      int64
}

// petBondRequest 内存中的一条待办申请。
type petBondRequest struct {
	ID        string
	Kind      string
	FromID    string
	ToID      string
	MasterID  string
	PetID     string
	Status    string
	CreatedAt int64
	// Approvals：已点同意的玩家 id 集合。
	Approvals map[string]bool
}

func bondKey(masterID, petID string) string {
	return masterID + "\x00" + petID
}

func (s *Server) loadPetBondsFromDisk() {
	if s.petBondDB == nil {
		s.petBonds = map[string]*petBond{}
		s.petBondRequests = map[string]*petBondRequest{}
		return
	}
	bonds, err := s.petBondDB.loadAllBonds()
	if err != nil {
		s.errorLog("petbond_load_failed", err.Error())
		s.petBonds = map[string]*petBond{}
	} else {
		s.petBonds = make(map[string]*petBond, len(bonds))
		for _, r := range bonds {
			s.petBonds[bondKey(r.MasterID, r.PetID)] = &petBond{
				MasterID: r.MasterID, PetID: r.PetID, PetTitle: r.PetTitle,
				TitleUpdatedAt: r.TitleUpdatedAt, CreatedAt: r.CreatedAt,
			}
		}
	}
	reqs, err := s.petBondDB.loadPendingRequests()
	if err != nil {
		s.errorLog("petbond_requests_load_failed", err.Error())
		s.petBondRequests = map[string]*petBondRequest{}
		return
	}
	s.petBondRequests = make(map[string]*petBondRequest, len(reqs))
	for _, r := range reqs {
		approvals := map[string]bool{}
		for _, pid := range r.Approvals {
			approvals[pid] = true
		}
		s.petBondRequests[r.ID] = &petBondRequest{
			ID: r.ID, Kind: r.Kind, FromID: r.FromID, ToID: r.ToID,
			MasterID: r.MasterID, PetID: r.PetID, Status: r.Status,
			CreatedAt: r.CreatedAt, Approvals: approvals,
		}
	}
}

func (s *Server) petBondCfg() types.PetBondConfig {
	cfg := s.cfg.PetBond
	if cfg.MaxPetsPerMaster < 1 {
		cfg.MaxPetsPerMaster = 3
	}
	if cfg.MaxMastersPerPet < 1 {
		cfg.MaxMastersPerPet = 3
	}
	if cfg.MaxTitleLength < 1 {
		cfg.MaxTitleLength = 12
	}
	if strings.TrimSpace(cfg.PanelTitle) == "" {
		cfg.PanelTitle = "宠物乐园"
	}
	return cfg
}

func (s *Server) getBond(masterID, petID string) *petBond {
	return s.petBonds[bondKey(masterID, petID)]
}

func (s *Server) mastersOf(petID string) []*petBond {
	var out []*petBond
	for _, b := range s.petBonds {
		if b.PetID == petID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

func (s *Server) petsOf(masterID string) []*petBond {
	var out []*petBond
	for _, b := range s.petBonds {
		if b.MasterID == masterID {
			out = append(out, b)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].CreatedAt < out[j].CreatedAt })
	return out
}

// isDirectMaster：A 是否是 B 的直系主人（链长=2 的反向禁止用）。
func (s *Server) isDirectMaster(masterID, petID string) bool {
	return s.getBond(masterID, petID) != nil
}

func (s *Server) maxPets() int { return s.petBondCfg().MaxPetsPerMaster }
func (s *Server) maxMasters() int { return s.petBondCfg().MaxMastersPerPet }

func (s *Server) persistBond(b *petBond) {
	if s.petBondDB == nil || b == nil {
		return
	}
	if err := s.petBondDB.upsertBond(petBondRow{
		MasterID: b.MasterID, PetID: b.PetID, PetTitle: b.PetTitle,
		TitleUpdatedAt: b.TitleUpdatedAt, CreatedAt: b.CreatedAt,
	}); err != nil {
		s.errorLog("petbond_upsert_failed", err.Error())
	}
}

func (s *Server) removeBond(masterID, petID string) {
	delete(s.petBonds, bondKey(masterID, petID))
	if s.petBondDB != nil {
		if err := s.petBondDB.deleteBond(masterID, petID); err != nil {
			s.errorLog("petbond_delete_failed", err.Error())
		}
	}
	// 称号可能变回积分档，刷新宠物展示。
	if pet := s.players[petID]; pet != nil {
		s.refreshPlayerSnapshots(pet)
	}
}

func (s *Server) createBond(masterID, petID string) error {
	if masterID == "" || petID == "" || masterID == petID {
		return fmt.Errorf("无效的认主关系")
	}
	if s.getBond(masterID, petID) != nil {
		return fmt.Errorf("已经是主人/宠物关系")
	}
	// 直系反向（链长=2）禁止：对方已是自己的直系主人时不能反过来。
	if s.isDirectMaster(petID, masterID) {
		return fmt.Errorf("不能与直系主人互相认主（关系链长度须大于 2）")
	}
	if len(s.petsOf(masterID)) >= s.maxPets() {
		return fmt.Errorf("该主人的宠物数量已达上限（%d）", s.maxPets())
	}
	if len(s.mastersOf(petID)) >= s.maxMasters() {
		return fmt.Errorf("该宠物的主人数量已达上限（%d）", s.maxMasters())
	}
	now := nowMs()
	b := &petBond{MasterID: masterID, PetID: petID, CreatedAt: now}
	s.petBonds[bondKey(masterID, petID)] = b
	s.persistBond(b)
	return nil
}

// bestPetTitle 取宠物所有主人设定的称号中最近更新的非空值。
func (s *Server) bestPetTitle(petID string) string {
	var best string
	var bestAt int64 = -1
	for _, b := range s.mastersOf(petID) {
		title := strings.TrimSpace(b.PetTitle)
		if title == "" {
			continue
		}
		at := b.TitleUpdatedAt
		if at >= bestAt {
			bestAt = at
			best = title
		}
	}
	return best
}

// applyDisplayTitle 在 publicPlayer 出站时：管理员自定义 > 宠物称号 > 积分档称号。
// 名争惩罚态由调用方 / 展示层处理（不在此改 Title）。
func (s *Server) applyDisplayTitle(player *PlayerState, p *types.PublicPlayer) {
	if player == nil || p == nil {
		return
	}
	if p.Stats.TitleCustom {
		return
	}
	if ptrBool(p.NameWarPunished) {
		return
	}
	if title := s.bestPetTitle(player.ID); title != "" {
		p.Stats.Title = title
	}
}

// publicPetBondEdges 大厅可公开展示的边：双方在线且都开了公开展示。
func (s *Server) publicPetBondEdges() []types.PetBondEdge {
	var edges []types.PetBondEdge
	for _, b := range s.petBonds {
		master := s.players[b.MasterID]
		pet := s.players[b.PetID]
		if master == nil || pet == nil {
			continue
		}
		if !master.Connected || !pet.Connected {
			continue
		}
		if !ptrBool(master.BondPublicDisplay) || !ptrBool(pet.BondPublicDisplay) {
			continue
		}
		edges = append(edges, types.PetBondEdge{
			MasterID: b.MasterID, PetID: b.PetID, PetTitle: b.PetTitle,
		})
	}
	sort.Slice(edges, func(i, j int) bool {
		if edges[i].MasterID != edges[j].MasterID {
			return edges[i].MasterID < edges[j].MasterID
		}
		return edges[i].PetID < edges[j].PetID
	})
	return edges
}

// buildPetBondChains 把公开边拆成 2~3 级关系链（超过 3 级滑动窗口拆分）。
func buildPetBondChains(edges []types.PetBondEdge) [][]string {
	if len(edges) == 0 {
		return nil
	}
	// adj: master -> pets
	adj := map[string][]string{}
	inDeg := map[string]int{}
	nodes := map[string]bool{}
	for _, e := range edges {
		adj[e.MasterID] = append(adj[e.MasterID], e.PetID)
		inDeg[e.PetID]++
		nodes[e.MasterID] = true
		nodes[e.PetID] = true
		if _, ok := inDeg[e.MasterID]; !ok {
			inDeg[e.MasterID] = 0
		}
	}
	for m := range adj {
		sort.Strings(adj[m])
	}
	// 从入度为 0 的节点出发 DFS 找极大简单路径；对环上节点也兜底扫一遍。
	var paths [][]string
	var dfs func(cur string, path []string, onPath map[string]bool)
	dfs = func(cur string, path []string, onPath map[string]bool) {
		children := adj[cur]
		extended := false
		for _, next := range children {
			if onPath[next] {
				continue
			}
			extended = true
			onPath[next] = true
			dfs(next, append(path, next), onPath)
			delete(onPath, next)
		}
		if !extended && len(path) >= 2 {
			cp := append([]string{}, path...)
			paths = append(paths, cp)
		}
	}
	starts := make([]string, 0, len(nodes))
	for id := range nodes {
		if inDeg[id] == 0 {
			starts = append(starts, id)
		}
	}
	sort.Strings(starts)
	visitedStart := map[string]bool{}
	for _, st := range starts {
		visitedStart[st] = true
		onPath := map[string]bool{st: true}
		dfs(st, []string{st}, onPath)
	}
	// 纯环 / 未覆盖边：对每个未作为路径起点的节点再尝试
	for id := range nodes {
		if visitedStart[id] {
			continue
		}
		onPath := map[string]bool{id: true}
		dfs(id, []string{id}, onPath)
	}
	// 拆成 2~3 级窗口
	seen := map[string]bool{}
	var chains [][]string
	add := func(c []string) {
		if len(c) < 2 || len(c) > 3 {
			return
		}
		key := strings.Join(c, ">")
		if seen[key] {
			return
		}
		seen[key] = true
		chains = append(chains, c)
	}
	for _, path := range paths {
		n := len(path)
		if n <= 3 {
			add(path)
			continue
		}
		for i := 0; i+2 < n; i++ {
			add(path[i : i+3])
		}
		// 末尾若只剩 2 人边且未出现在任何 3 人窗里，也补上（一般已覆盖）
		add(path[n-2 : n])
	}
	// 若某条边没被任何路径覆盖（孤立边），补 2 人链
	for _, e := range edges {
		add([]string{e.MasterID, e.PetID})
	}
	sort.Slice(chains, func(i, j int) bool {
		return strings.Join(chains[i], ">") < strings.Join(chains[j], ">")
	})
	return chains
}

func (s *Server) findPendingRequest(kind, fromID, toID, masterID, petID string) *petBondRequest {
	for _, r := range s.petBondRequests {
		if r.Status != petBondStatusPending {
			continue
		}
		if r.Kind != kind {
			continue
		}
		if kind == petBondKindRelease {
			if r.MasterID == masterID && r.PetID == petID {
				return r
			}
			continue
		}
		if r.FromID == fromID && r.ToID == toID {
			return r
		}
	}
	return nil
}

// requiredApprovers 返回该申请还需要谁点同意。
func (s *Server) requiredApprovers(r *petBondRequest) []string {
	if r == nil {
		return nil
	}
	switch r.Kind {
	case petBondKindSeekMaster:
		// 新主人 + 宠物的全体现有主人
		set := map[string]bool{r.ToID: true}
		for _, b := range s.mastersOf(r.FromID) {
			set[b.MasterID] = true
		}
		out := make([]string, 0, len(set))
		for id := range set {
			out = append(out, id)
		}
		sort.Strings(out)
		return out
	case petBondKindSeekPet:
		// 仅宠物本人（UI「同意认宠请求」）；不要求其现有主人同意
		return []string{r.ToID}
	case petBondKindRelease:
		// 对方同意即可
		other := r.MasterID
		if r.FromID == r.MasterID {
			other = r.PetID
		}
		return []string{other}
	default:
		return nil
	}
}

func (s *Server) requestFullyApproved(r *petBondRequest) bool {
	for _, id := range s.requiredApprovers(r) {
		if !r.Approvals[id] {
			return false
		}
	}
	return true
}

func (s *Server) cancelRequest(r *petBondRequest, status string) {
	if r == nil {
		return
	}
	r.Status = status
	delete(s.petBondRequests, r.ID)
	if s.petBondDB != nil {
		_ = s.petBondDB.setRequestStatus(r.ID, status)
		_ = s.petBondDB.deleteApprovals(r.ID)
	}
}

func (s *Server) addPetBondRequest(r *petBondRequest) error {
	if s.petBondRequests == nil {
		s.petBondRequests = map[string]*petBondRequest{}
	}
	if r.Approvals == nil {
		r.Approvals = map[string]bool{}
	}
	s.petBondRequests[r.ID] = r
	if s.petBondDB != nil {
		if err := s.petBondDB.insertRequest(petBondRequestRow{
			ID: r.ID, Kind: r.Kind, FromID: r.FromID, ToID: r.ToID,
			MasterID: r.MasterID, PetID: r.PetID, Status: r.Status, CreatedAt: r.CreatedAt,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) approvePetBondRequest(r *petBondRequest, playerID string) error {
	if r == nil || r.Status != petBondStatusPending {
		return fmt.Errorf("申请不存在或已处理")
	}
	needed := false
	for _, id := range s.requiredApprovers(r) {
		if id == playerID {
			needed = true
			break
		}
	}
	if !needed {
		return fmt.Errorf("你无权处理该申请")
	}
	if r.Approvals == nil {
		r.Approvals = map[string]bool{}
	}
	r.Approvals[playerID] = true
	if s.petBondDB != nil {
		_ = s.petBondDB.addApproval(r.ID, playerID, nowMs())
	}
	if !s.requestFullyApproved(r) {
		return nil
	}
	// 全部同意 → 执行
	var err error
	switch r.Kind {
	case petBondKindSeekMaster:
		err = s.createBond(r.ToID, r.FromID)
	case petBondKindSeekPet:
		err = s.createBond(r.FromID, r.ToID)
	case petBondKindRelease:
		s.removeBond(r.MasterID, r.PetID)
	}
	if err != nil {
		// 执行失败时撤回本票同意（内存 + 落盘），保留申请让人重试/取消
		delete(r.Approvals, playerID)
		if s.petBondDB != nil {
			_ = s.petBondDB.removeApproval(r.ID, playerID)
		}
		return err
	}
	s.cancelRequest(r, petBondStatusDone)
	return nil
}

// ── 出站视图 ──────────────────────────────────────────

type petBondMemberView struct {
	PlayerID         string `json:"playerId"`
	Name             string `json:"name"`
	DisplayName      string `json:"displayName"`
	AvatarURL        string `json:"avatarUrl,omitempty"`
	Connected        bool   `json:"connected"`
	PetTitle         string `json:"petTitle,omitempty"`
	ReleasePending   bool   `json:"releasePending,omitempty"`
	ReleaseIncoming  bool   `json:"releaseIncoming,omitempty"` // 对方申请解除，我可同意
	ReleaseRequestID string `json:"releaseRequestId,omitempty"`
	// 现有主人视角：该宠物正在申请认新主，需要我同意
	NewMasterPendingID   string `json:"newMasterPendingId,omitempty"`
	NewMasterPendingName string `json:"newMasterPendingName,omitempty"`
}

type petBondRequestView struct {
	ID              string   `json:"id"`
	Kind            string   `json:"kind"`
	FromID          string   `json:"fromId"`
	ToID            string   `json:"toId"`
	MasterID        string   `json:"masterId,omitempty"`
	PetID           string   `json:"petId,omitempty"`
	FromName        string   `json:"fromName"`
	ToName          string   `json:"toName"`
	CreatedAt       int64    `json:"createdAt"`
	RequiredIDs     []string `json:"requiredIds"`
	ApprovedIDs     []string `json:"approvedIds"`
	CanApprove      bool     `json:"canApprove"`
	Label           string   `json:"label"` // 按钮文案
	PinTop          bool     `json:"pinTop,omitempty"`
}

type petBondCandidateView struct {
	PlayerID   string `json:"playerId"`
	Name       string `json:"name"`
	DisplayName string `json:"displayName"`
	AvatarURL  string `json:"avatarUrl,omitempty"`
	Connected  bool   `json:"connected"`
	// pending / none / already
	Status    string `json:"status"`
	RequestID string `json:"requestId,omitempty"`
	// 对方发来、我可同意
	Incoming    bool   `json:"incoming,omitempty"`
	IncomingID  string `json:"incomingId,omitempty"`
	IncomingLabel string `json:"incomingLabel,omitempty"`
}

type petBondChainView struct {
	PlayerIDs   []string `json:"playerIds"`
	PlayerNames []string `json:"playerNames"`
}

type petBondStateView struct {
	Masters     []petBondMemberView    `json:"masters"`
	Pets        []petBondMemberView    `json:"pets"`
	// 认主侧可见的可申请列表（开启认宠的在线玩家）
	MasterCandidates []petBondCandidateView `json:"masterCandidates"`
	// 认宠侧可见的可申请列表（开启认主的在线玩家）
	PetCandidates []petBondCandidateView `json:"petCandidates"`
	// 需要我处理的申请（置顶用）
	Incoming []petBondRequestView `json:"incoming"`
	// 我发出的待处理申请
	Outgoing []petBondRequestView `json:"outgoing"`
	Chains   []petBondChainView   `json:"chains"`
	Config   types.PetBondConfig  `json:"config"`
}

func (s *Server) playerDisplayName(id string) string {
	p := s.players[id]
	if p == nil {
		return "未知玩家"
	}
	if p.DisplayName != "" {
		return p.DisplayName
	}
	return p.Name
}

func (s *Server) playerName(id string) string {
	p := s.players[id]
	if p == nil {
		return "未知玩家"
	}
	return p.Name
}

func (s *Server) buildPetBondState(viewerID string) petBondStateView {
	cfg := s.petBondCfg()
	view := petBondStateView{
		Masters:          []petBondMemberView{},
		Pets:             []petBondMemberView{},
		MasterCandidates: []petBondCandidateView{},
		PetCandidates:    []petBondCandidateView{},
		Incoming:         []petBondRequestView{},
		Outgoing:         []petBondRequestView{},
		Chains:           []petBondChainView{},
		Config:           cfg,
	}
	viewer := s.players[viewerID]
	if viewer == nil {
		return view
	}

	// 自己的主人/宠物列表
	for _, b := range s.mastersOf(viewerID) {
		p := s.players[b.MasterID]
		m := petBondMemberView{
			PlayerID: b.MasterID, PetTitle: b.PetTitle,
		}
		if p != nil {
			m.Name, m.DisplayName, m.AvatarURL, m.Connected = p.Name, p.DisplayName, p.AvatarURL, p.Connected
		} else {
			m.Name, m.DisplayName = "已注销", "已注销"
		}
		if r := s.findPendingRequest(petBondKindRelease, "", "", b.MasterID, b.PetID); r != nil {
			m.ReleaseRequestID = r.ID
			if r.FromID == viewerID {
				m.ReleasePending = true
			} else {
				m.ReleaseIncoming = true
			}
		}
		view.Masters = append(view.Masters, m)
	}
	for _, b := range s.petsOf(viewerID) {
		p := s.players[b.PetID]
		m := petBondMemberView{
			PlayerID: b.PetID, PetTitle: b.PetTitle,
		}
		if p != nil {
			m.Name, m.DisplayName, m.AvatarURL, m.Connected = p.Name, p.DisplayName, p.AvatarURL, p.Connected
		} else {
			m.Name, m.DisplayName = "已注销", "已注销"
		}
		if r := s.findPendingRequest(petBondKindRelease, "", "", b.MasterID, b.PetID); r != nil {
			m.ReleaseRequestID = r.ID
			if r.FromID == viewerID {
				m.ReleasePending = true
			} else {
				m.ReleaseIncoming = true
			}
		}
		// 该宠物申请认新主，且我是现有主人审批人之一
		for _, r := range s.petBondRequests {
			if r.Status != petBondStatusPending || r.Kind != petBondKindSeekMaster || r.FromID != b.PetID {
				continue
			}
			if r.ToID == viewerID {
				continue // 新主人侧走候选列表「同意认主请求」
			}
			if r.Approvals[viewerID] {
				continue
			}
			for _, need := range s.requiredApprovers(r) {
				if need == viewerID {
					m.NewMasterPendingID = r.ID
					m.NewMasterPendingName = s.playerName(r.ToID)
					break
				}
			}
		}
		view.Pets = append(view.Pets, m)
	}

	// 申请视图
	for _, r := range s.petBondRequests {
		if r.Status != petBondStatusPending {
			continue
		}
		rv := s.requestToView(r, viewerID)
		isParty := r.FromID == viewerID || r.ToID == viewerID || r.MasterID == viewerID || r.PetID == viewerID
		if !isParty {
			// 现有主人也可能是 seek_master 的审批人
			for _, id := range s.requiredApprovers(r) {
				if id == viewerID {
					isParty = true
					break
				}
			}
		}
		if !isParty {
			continue
		}
		if r.FromID == viewerID {
			view.Outgoing = append(view.Outgoing, rv)
		}
		if rv.CanApprove {
			rv.PinTop = true
			view.Incoming = append(view.Incoming, rv)
		}
	}
	sort.Slice(view.Incoming, func(i, j int) bool { return view.Incoming[i].CreatedAt > view.Incoming[j].CreatedAt })
	sort.Slice(view.Outgoing, func(i, j int) bool { return view.Outgoing[i].CreatedAt > view.Outgoing[j].CreatedAt })

	// 认主侧候选：开启认宠的在线玩家 + 向我发起认宠申请的人（置顶）；已是关系不展示
	if ptrBool(viewer.BondMasterEnabled) {
		seen := map[string]bool{}
		addMasterCandidate := func(p *PlayerState) *petBondCandidateView {
			if p == nil || p.ID == viewerID || seen[p.ID] {
				return nil
			}
			if s.getBond(p.ID, viewerID) != nil {
				return nil
			}
			seen[p.ID] = true
			c := petBondCandidateView{
				PlayerID: p.ID, Name: p.Name, DisplayName: p.DisplayName,
				AvatarURL: p.AvatarURL, Connected: p.Connected, Status: "none",
			}
			if r := s.findPendingRequest(petBondKindSeekMaster, viewerID, p.ID, "", ""); r != nil {
				c.Status = "pending"
				c.RequestID = r.ID
			}
			if r := s.findPendingRequest(petBondKindSeekPet, p.ID, viewerID, "", ""); r != nil {
				c.Incoming = true
				c.IncomingID = r.ID
				c.IncomingLabel = "同意认宠请求"
			}
			view.MasterCandidates = append(view.MasterCandidates, c)
			return &view.MasterCandidates[len(view.MasterCandidates)-1]
		}
		// 先放 incoming（置顶）
		for _, r := range s.petBondRequests {
			if r.Status != petBondStatusPending || r.Kind != petBondKindSeekPet || r.ToID != viewerID {
				continue
			}
			addMasterCandidate(s.players[r.FromID])
		}
		for _, p := range s.players {
			if !p.Connected || !ptrBool(p.BondPetEnabled) {
				continue
			}
			addMasterCandidate(p)
		}
		sort.SliceStable(view.MasterCandidates, func(i, j int) bool {
			if view.MasterCandidates[i].Incoming != view.MasterCandidates[j].Incoming {
				return view.MasterCandidates[i].Incoming
			}
			return view.MasterCandidates[i].Name < view.MasterCandidates[j].Name
		})
	}
	// 认宠侧候选：开启认主的在线玩家 + 向我发起认主申请的人（置顶）；已是关系不展示
	if ptrBool(viewer.BondPetEnabled) {
		seen := map[string]bool{}
		addPetCandidate := func(p *PlayerState) *petBondCandidateView {
			if p == nil || p.ID == viewerID || seen[p.ID] {
				return nil
			}
			if s.getBond(viewerID, p.ID) != nil {
				return nil
			}
			seen[p.ID] = true
			c := petBondCandidateView{
				PlayerID: p.ID, Name: p.Name, DisplayName: p.DisplayName,
				AvatarURL: p.AvatarURL, Connected: p.Connected, Status: "none",
			}
			if r := s.findPendingRequest(petBondKindSeekPet, viewerID, p.ID, "", ""); r != nil {
				c.Status = "pending"
				c.RequestID = r.ID
			}
			if r := s.findPendingRequest(petBondKindSeekMaster, p.ID, viewerID, "", ""); r != nil {
				c.Incoming = true
				c.IncomingID = r.ID
				c.IncomingLabel = "同意认主请求"
			}
			view.PetCandidates = append(view.PetCandidates, c)
			return &view.PetCandidates[len(view.PetCandidates)-1]
		}
		for _, r := range s.petBondRequests {
			if r.Status != petBondStatusPending || r.Kind != petBondKindSeekMaster || r.ToID != viewerID {
				continue
			}
			addPetCandidate(s.players[r.FromID])
		}
		for _, p := range s.players {
			if !p.Connected || !ptrBool(p.BondMasterEnabled) {
				continue
			}
			addPetCandidate(p)
		}
		sort.SliceStable(view.PetCandidates, func(i, j int) bool {
			if view.PetCandidates[i].Incoming != view.PetCandidates[j].Incoming {
				return view.PetCandidates[i].Incoming
			}
			return view.PetCandidates[i].Name < view.PetCandidates[j].Name
		})
	}

	// 关系链
	for _, chain := range buildPetBondChains(s.publicPetBondEdges()) {
		names := make([]string, len(chain))
		for i, id := range chain {
			names[i] = s.playerDisplayName(id)
		}
		view.Chains = append(view.Chains, petBondChainView{PlayerIDs: chain, PlayerNames: names})
	}
	return view
}

func (s *Server) requestToView(r *petBondRequest, viewerID string) petBondRequestView {
	reqIDs := s.requiredApprovers(r)
	approved := make([]string, 0, len(r.Approvals))
	for id := range r.Approvals {
		approved = append(approved, id)
	}
	sort.Strings(approved)
	can := false
	for _, id := range reqIDs {
		if id == viewerID && !r.Approvals[viewerID] {
			can = true
			break
		}
	}
	label := "同意"
	switch r.Kind {
	case petBondKindSeekMaster:
		if r.ToID == viewerID {
			label = "同意认主请求"
		} else {
			label = "同意宠物认新主"
		}
	case petBondKindSeekPet:
		label = "同意认宠请求"
	case petBondKindRelease:
		label = "同意解除关系"
	}
	return petBondRequestView{
		ID: r.ID, Kind: r.Kind, FromID: r.FromID, ToID: r.ToID,
		MasterID: r.MasterID, PetID: r.PetID,
		FromName: s.playerName(r.FromID), ToName: s.playerName(r.ToID),
		CreatedAt: r.CreatedAt, RequiredIDs: reqIDs, ApprovedIDs: approved,
		CanApprove: can, Label: label,
	}
}

// notifyAllOnlinePetBondStates 向所有在线玩家推送宠物乐园状态。
// 当有人开关认主/认宠/公开展示，或关系变更时调用，保证他人列表实时刷新。
func (s *Server) notifyAllOnlinePetBondStates() {
	for _, p := range s.players {
		if p == nil || !p.Connected || p.SocketID == "" {
			continue
		}
		if s.clients[p.SocketID] == nil {
			continue
		}
		s.emitToClient(p.SocketID, "petbond:update", s.buildPetBondState(p.ID))
	}
}
