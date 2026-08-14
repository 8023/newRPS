package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) onPetBondGetState(client *Client, env wsEnvelope) {
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func forcedGiveawayMasterName(room *RoomState, seat types.SeatKey) string {
	if room == nil || room.ForcedGiveawayBySeat == nil {
		return ""
	}
	return room.ForcedGiveawayBySeat[seat]
}

func setForcedGiveaway(room *RoomState, seat types.SeatKey, masterName string) {
	if room.ForcedGiveawayBySeat == nil {
		room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	}
	room.ForcedGiveawayBySeat[seat] = masterName
}

func takeForcedGiveaway(room *RoomState, seat types.SeatKey) string {
	masterName := forcedGiveawayMasterName(room, seat)
	if room != nil && room.ForcedGiveawayBySeat != nil {
		delete(room.ForcedGiveawayBySeat, seat)
	}
	return masterName
}

// onPetBondForceGiveaway 把宠物当前局的本轮选择直接改成白给。RPS 立即覆盖已经锁定的出拳；
// 回合制游戏若正轮到宠物，则立即进入/结算本手白给，否则在宠物本局下一手开始时生效。
func (s *Server) onPetBondForceGiveaway(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameRPS && room.Settings.GameID != types.GameOthello &&
		room.Settings.GameID != types.GameTicTacToe && room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前游戏不支持强制白给")
		return
	}
	if room.Settings.GameID == types.GameOthello && !room.Settings.EnableRanked {
		client.reply(env.ID, nil, "黑白棋只有排位房才支持白给结算")
		return
	}
	masterSeat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以强制白给")
		return
	}
	petSeat := oppositeSeat(masterSeat)
	petOcc := room.Seats[petSeat]
	if petOcc == nil {
		client.reply(env.ID, nil, "对方不在战斗席")
		return
	}
	pet := s.players[petOcc.GetID()]
	targetID := strings.TrimSpace(p.TargetID)
	if pet == nil || pet.ID != targetID {
		client.reply(env.ID, nil, "目标玩家不正确")
		return
	}
	if s.getBond(player.ID, pet.ID) == nil {
		client.reply(env.ID, nil, "对方不是你的宠物")
		return
	}
	if !ptrBool(pet.GiveawayEnabled) {
		client.reply(env.ID, nil, "对方未开启白给模式")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚阶段不能强制白给")
		return
	}

	masterName := playerShortName(player)
	petName := playerShortName(pet)
	switch room.Settings.GameID {
	case types.GameRPS:
		if room.Phase == types.PhaseResult {
			s.prepareNextChoice(room)
		}
		if room.Phase != types.PhaseChoosing {
			client.reply(env.ID, nil, "当前不能强制白给")
			return
		}
		if forcedGiveawayMasterName(room, petSeat) != "" {
			client.reply(env.ID, nil, "已经强制过一次，正在等待本轮结算")
			return
		}
		if room.Choices == nil {
			room.Choices = map[types.SeatKey]types.Move{}
		}
		previous := room.Choices[petSeat]
		setForcedGiveaway(room, petSeat, masterName)
		room.Choices[petSeat] = types.MoveGiveaway
		if previous != types.MoveGiveaway {
			s.addGiveawayValue(pet, s.cfg.Giveaway.ActiveBoostValue)
		}
		s.roomNotice(room, fmt.Sprintf("主人（%s）强制（%s）白给，本轮选择已改为白给。", masterName, petName))
		oldStatus := room.Status
		s.finishRoundIfReady(room)
		client.reply(env.ID, map[string]any{"ok": true}, "")
		s.broadcastRoom(room.ID, oldStatus != room.Status)
		return

	case types.GameOthello:
		if room.Phase != types.PhaseChoosing || room.Othello == nil || room.Othello.Ended {
			client.reply(env.ID, nil, "当前黑白棋对局不能强制白给")
			return
		}
		if pending := room.Othello.PendingSettlement; pending != nil && pending.Seat == petSeat {
			if pending.ForcedByMasterName != "" {
				client.reply(env.ID, nil, "已经强制过一次，正在结算")
				return
			}
			pending.Forced = "giveaway"
			pending.ForcedByMasterName = masterName
			if ok, errMsg := s.settleOthelloPendingMoveClean(room, "giveaway", "masterForce"); !ok {
				client.reply(env.ID, nil, errMsg)
				return
			}
			client.reply(env.ID, map[string]any{"ok": true}, "")
			s.broadcastRoom(room.ID, true)
			return
		}

	case types.GameTicTacToe:
		if room.Phase != types.PhaseChoosing || room.TicTacToe == nil || room.TicTacToe.Ended {
			client.reply(env.ID, nil, "当前井字棋对局不能强制白给")
			return
		}
		if room.TicTacToe.Turn == petSeat {
			s.clearTicTacToeGiveawayTimer(room.ID)
			takeForcedGiveaway(room, petSeat)
			ok, row, col, errMsg := s.applyTicTacToeRandomMove(room, petSeat, "forcedGiveaway")
			if !ok {
				client.reply(env.ID, nil, errMsg)
				return
			}
			text := fmt.Sprintf("主人（%s）强制（%s）白给，系统随机落在第 %d 行第 %d 列。", masterName, petName, row+1, col+1)
			if room.TicTacToe != nil && !room.TicTacToe.Ended {
				room.ResultText = text
			}
			s.roomNotice(room, text)
			client.reply(env.ID, map[string]any{"ok": true}, "")
			s.broadcastRoom(room.ID, true)
			return
		}

	case types.GameGomoku:
		if room.Phase != types.PhaseChoosing || room.Gomoku == nil || room.Gomoku.Ended {
			client.reply(env.ID, nil, "当前五子棋对局不能强制白给")
			return
		}
		if room.Gomoku.UndoRequest != nil || room.Gomoku.ResignRequest != nil {
			client.reply(env.ID, nil, "请求处理期间不能强制白给")
			return
		}
		if room.Gomoku.Turn == petSeat && room.Gomoku.GiveawaySeat == petSeat {
			if room.Gomoku.GiveawayForcedByMasterName != "" {
				client.reply(env.ID, nil, "已经强制过一次，正在等待宠物落子")
				return
			}
			// 情况 4：宠物已经主动点了白给，主人随后强制不会改变其已知状态。
			room.Gomoku.GiveawayForcedByMasterName = masterName
			room.ResultText = fmt.Sprintf("主人（%s）强制（%s）本手白给，请选择落点。", masterName, petName)
			s.roomNotice(room, room.ResultText)
			client.reply(env.ID, map[string]any{"ok": true}, "")
			s.broadcastRoom(room.ID, true)
			return
		}
	}

	if forcedGiveawayMasterName(room, petSeat) != "" {
		message := "已经强制过一次，正在等待宠物本局下一手"
		if room.Settings.GameID == types.GameGomoku && room.Gomoku != nil && room.Gomoku.Turn == petSeat {
			message = "已经强制过一次，正在等待宠物落子"
		}
		client.reply(env.ID, nil, message)
		return
	}
	setForcedGiveaway(room, petSeat, masterName)
	client.reply(env.ID, map[string]any{"ok": true}, "")
	if room.Settings.GameID != types.GameGomoku {
		s.roomNotice(room, fmt.Sprintf("主人（%s）强制（%s）白给，将在宠物本局下一手生效。", masterName, petName))
		s.broadcastRoom(room.ID, true)
	}
}

func (s *Server) onPetBondSeekMaster(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if !ptrBool(player.BondMasterEnabled) {
		client.reply(env.ID, nil, "请先在个人设置开启认主")
		return
	}
	targetID := strings.TrimSpace(p.TargetID)
	target := s.players[targetID]
	if target == nil || !target.Connected {
		client.reply(env.ID, nil, "目标玩家不在线")
		return
	}
	if !ptrBool(target.BondPetEnabled) {
		client.reply(env.ID, nil, "对方未开启认宠")
		return
	}
	if targetID == player.ID {
		client.reply(env.ID, nil, "不能认自己为主")
		return
	}
	// target 成为主人，player 成为宠物
	if s.getBond(targetID, player.ID) != nil {
		client.reply(env.ID, nil, "已经是对方的宠物")
		return
	}
	if s.isDirectMaster(player.ID, targetID) {
		client.reply(env.ID, nil, "不能与直系主人互相认主（关系链长度须大于 2）")
		return
	}
	if len(s.mastersOf(player.ID)) >= s.maxMasters() {
		client.reply(env.ID, nil, "你的主人数量已达上限")
		return
	}
	if len(s.petsOf(targetID)) >= s.maxPets() {
		client.reply(env.ID, nil, "对方宠物数量已达上限")
		return
	}
	if s.findPendingRequest(petBondKindSeekMaster, player.ID, targetID, "", "") != nil {
		client.reply(env.ID, nil, "已有进行中的申请")
		return
	}
	// 取消反向同向冲突申请（可选：保留）
	req := &petBondRequest{
		ID: randomID(), Kind: petBondKindSeekMaster,
		FromID: player.ID, ToID: targetID,
		Status: petBondStatusPending, CreatedAt: nowMs(),
		Approvals: map[string]bool{},
	}
	if err := s.addPetBondRequest(req); err != nil {
		client.reply(env.ID, nil, "申请失败，请稍后重试")
		return
	}
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func (s *Server) onPetBondSeekPet(client *Client, env wsEnvelope) {
	var p struct {
		TargetID string `json:"targetId"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	if !ptrBool(player.BondPetEnabled) {
		client.reply(env.ID, nil, "请先在个人设置开启认宠")
		return
	}
	targetID := strings.TrimSpace(p.TargetID)
	target := s.players[targetID]
	if target == nil || !target.Connected {
		client.reply(env.ID, nil, "目标玩家不在线")
		return
	}
	if !ptrBool(target.BondMasterEnabled) {
		client.reply(env.ID, nil, "对方未开启认主")
		return
	}
	if targetID == player.ID {
		client.reply(env.ID, nil, "不能认自己为宠物")
		return
	}
	// player 成为主人，target 成为宠物
	if s.getBond(player.ID, targetID) != nil {
		client.reply(env.ID, nil, "已经是对方的主人")
		return
	}
	if s.isDirectMaster(targetID, player.ID) {
		client.reply(env.ID, nil, "不能与直系宠物互相认主（关系链长度须大于 2）")
		return
	}
	if len(s.petsOf(player.ID)) >= s.maxPets() {
		client.reply(env.ID, nil, "你的宠物数量已达上限")
		return
	}
	if len(s.mastersOf(targetID)) >= s.maxMasters() {
		client.reply(env.ID, nil, "对方主人数量已达上限")
		return
	}
	if s.findPendingRequest(petBondKindSeekPet, player.ID, targetID, "", "") != nil {
		client.reply(env.ID, nil, "已有进行中的申请")
		return
	}
	req := &petBondRequest{
		ID: randomID(), Kind: petBondKindSeekPet,
		FromID: player.ID, ToID: targetID,
		Status: petBondStatusPending, CreatedAt: nowMs(),
		Approvals: map[string]bool{},
	}
	if err := s.addPetBondRequest(req); err != nil {
		client.reply(env.ID, nil, "申请失败，请稍后重试")
		return
	}
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func (s *Server) onPetBondApprove(client *Client, env wsEnvelope) {
	var p struct {
		RequestID string `json:"requestId"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	req := s.petBondRequests[strings.TrimSpace(p.RequestID)]
	if req == nil || req.Status != petBondStatusPending {
		client.reply(env.ID, nil, "申请不存在或已处理")
		return
	}
	if err := s.approvePetBondRequest(req, player.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	// 关系/申请变化会影响候选列表与公开关系图，推送给全部在线用户。
	if pet := s.players[req.FromID]; pet != nil {
		s.refreshPlayerSnapshots(pet)
	}
	if pet := s.players[req.ToID]; pet != nil {
		s.refreshPlayerSnapshots(pet)
	}
	if pet := s.players[req.PetID]; pet != nil {
		s.refreshPlayerSnapshots(pet)
	}
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

// onPetBondCancel 发起人撤销自己的待办申请（认主/认宠/解除均可）。
func (s *Server) onPetBondCancel(client *Client, env wsEnvelope) {
	var p struct {
		RequestID string `json:"requestId"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	req := s.petBondRequests[strings.TrimSpace(p.RequestID)]
	if req == nil || req.Status != petBondStatusPending {
		client.reply(env.ID, nil, "申请不存在或已处理")
		return
	}
	if req.FromID != player.ID {
		client.reply(env.ID, nil, "只能撤销自己发起的申请")
		return
	}
	s.cancelRequest(req, petBondStatusCancelled)
	s.notifyAllOnlinePetBondStates()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func (s *Server) onPetBondRequestRelease(client *Client, env wsEnvelope) {
	var p struct {
		MasterID string `json:"masterId"`
		PetID    string `json:"petId"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	masterID, petID := strings.TrimSpace(p.MasterID), strings.TrimSpace(p.PetID)
	if s.getBond(masterID, petID) == nil {
		client.reply(env.ID, nil, "关系不存在")
		return
	}
	if player.ID != masterID && player.ID != petID {
		client.reply(env.ID, nil, "只能解除自己的关系")
		return
	}
	if existing := s.findPendingRequest(petBondKindRelease, "", "", masterID, petID); existing != nil {
		client.reply(env.ID, nil, "已有进行中的解除申请")
		return
	}
	req := &petBondRequest{
		ID: randomID(), Kind: petBondKindRelease,
		FromID: player.ID, ToID: "",
		MasterID: masterID, PetID: petID,
		Status: petBondStatusPending, CreatedAt: nowMs(),
		Approvals: map[string]bool{},
	}
	// ToID 填对方，便于展示
	if player.ID == masterID {
		req.ToID = petID
	} else {
		req.ToID = masterID
	}
	if err := s.addPetBondRequest(req); err != nil {
		client.reply(env.ID, nil, "申请失败，请稍后重试")
		return
	}
	s.notifyAllOnlinePetBondStates()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}

func (s *Server) onPetBondSetTitle(client *Client, env wsEnvelope) {
	var p struct {
		PetID string `json:"petId"`
		Title string `json:"title"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	petID := strings.TrimSpace(p.PetID)
	b := s.getBond(player.ID, petID)
	if b == nil {
		client.reply(env.ID, nil, "只能为自己的直系宠物设置称号")
		return
	}
	maxLen := s.petBondCfg().MaxTitleLength
	title := cleanText(p.Title, maxLen)
	b.PetTitle = title
	b.TitleUpdatedAt = nowMs()
	s.persistBond(b)
	if pet := s.players[petID]; pet != nil {
		pet.DisplayName = s.formatDisplayName(pet)
		s.refreshPlayerSnapshots(pet)
	}
	s.notifyAllOnlinePetBondStates()
	s.broadcastLobby()
	client.reply(env.ID, s.buildPetBondState(player.ID), "")
}
