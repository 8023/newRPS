package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) onRoomCreate(client *Client, env wsEnvelope) {
	var p struct {
		Settings types.RoomSettings `json:"settings"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayer(client, env)
	if !ok {
		return
	}
	settings := p.Settings
	gameID := settings.GameID
	if gameID != types.GameOthello && gameID != types.GameTicTacToe {
		gameID = types.GameRPS
	}
	settings.GameID = gameID
	if settings.Stake == 0 {
		settings.Stake = 5
	}
	if settings.AllowProofImage == nil {
		settings.AllowProofImage = boolPtr(true)
	}
	if settings.PunishmentSource == "" {
		settings.PunishmentSource = "system"
	}
	settings.EnableRankMultiplier = settings.EnableRanked && settings.EnableRankMultiplier
	if settings.EnableRankMultiplier {
		switch settings.RankMultiplier {
		case 2, 5, 10:
		default:
			settings.RankMultiplier = 1
		}
	} else {
		settings.RankMultiplier = 1
	}
	settings.EnableExtremeRanked = settings.EnableRanked && settings.EnableExtremeRanked
	if settings.GameID == types.GameOthello {
		switch settings.Stake {
		case 1, 2, 5, 10:
		default:
			settings.Stake = 5
		}
		switch settings.OthelloBoardTheme {
		case "classic", "pastel", "midnight", "wood", "neon":
		default:
			settings.OthelloBoardTheme = "classic"
		}
		settings.EnableBot = false
	} else if settings.GameID == types.GameTicTacToe {
		switch settings.Stake {
		case 5, 10, 20:
		default:
			settings.Stake = 5
		}
		switch settings.TicTacToeBoardTheme {
		case "paper", "mint", "midnight", "candy", "arcade":
		default:
			settings.TicTacToeBoardTheme = "paper"
		}
		settings.EnableBot = false
	} else {
		switch settings.Stake {
		case 5, 10, 20:
		default:
			settings.Stake = 5
		}
	}
	if !settings.EnableRanked {
		settings.EnableRankMultiplier = false
		settings.RankMultiplier = 1
		settings.EnableExtremeRanked = false
	}
	if settings.EnableExtremeRanked {
		settings.EnableRankMultiplier = false
		settings.RankMultiplier = 1
	}
	if settings.EnablePunishment && settings.PunishmentSource != "player" {
		settings.PunishmentIDs = s.selectedPunishmentIDs(settings)
		if len(settings.PunishmentIDs) > 0 {
			settings.PunishmentID = settings.PunishmentIDs[len(settings.PunishmentIDs)-1]
		}
	}
	if settings.PunishmentSource == "player" {
		settings.PunishmentIDs = []string{}
		settings.PunishmentID = ""
	}
	settings.Tags = s.normalizeRoomTags(settings)
	settings.EnableTags = len(settings.Tags) > 0
	settings.Name = s.normalizeRoomName(settings)
	settings.RoomBackgroundImage = s.randomRoomBackground(settings)
	if settings.EnableRanked && settings.EnableBot {
		client.reply(env.ID, nil, "排位战不能开启 Bot")
		return
	}
	if settings.EnableRanked && ptrBool(player.ExtremeModeEnabled) != settings.EnableExtremeRanked {
		if ptrBool(player.ExtremeModeEnabled) {
			client.reply(env.ID, nil, "极限模式玩家只能创建极限排位房")
		} else {
			client.reply(env.ID, nil, "只有极限模式玩家可以创建极限排位房")
		}
		return
	}
	if settings.EnableExtremeRanked && settings.EnableBot {
		client.reply(env.ID, nil, "极限排位不能开启 Bot")
		return
	}
	if settings.EnableExtremeRanked && settings.EnableRankMultiplier {
		client.reply(env.ID, nil, "极限排位不能开启倍率模式")
		return
	}
	if settings.EnableRankMultiplier && !ptrBool(player.RankMultiplierUnlocked) {
		client.reply(env.ID, nil, "请先提交 200 排位积分解锁倍率模式")
		return
	}
	if settings.EnablePunishment && settings.PunishmentSource == "player" && settings.EnableBot {
		client.reply(env.ID, nil, "玩家发布任务模式不能开启 Bot")
		return
	}
	leaveResult := s.leaveRoom(player, LeaveSwitchRoom)
	if !leaveResult.OK {
		client.reply(env.ID, nil, leaveResult.Error)
		return
	}
	roomID := randomID()
	status, phase := "waiting", types.PhaseReady
	seats := map[types.SeatKey]SeatOccupant{types.SeatA: &HumanSeat{Player: s.publicPlayer(player)}, types.SeatB: nil}
	if settings.EnableBot {
		status, phase = "playing", types.PhaseChoosing
		seats[types.SeatB] = s.makeBot(settings.BotDifficulty)
	}
	room := &RoomState{
		ID: roomID, Code: s.roomCode(), OwnerID: player.ID, Settings: settings,
		Status: status, UpdatedAt: nowMs(), Phase: phase, Seats: seats,
		SpectatorIDs: []string{}, Ready: map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false},
		Choices: map[types.SeatKey]types.Move{}, PunishedPlayerIDs: []string{}, Proofs: []types.PunishmentProof{},
		Score: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatedScore: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatStats: map[types.SeatKey]types.SeatStats{types.SeatA: emptySeatStats(), types.SeatB: emptySeatStats()},
		RoundHistory: []types.RoundHistoryItem{}, Chat: []types.ChatMessage{}, LockedSeatIDs: map[string]struct{}{},
		DisconnectForfeits: map[string]DisconnectForfeit{}, CreatedAt: nowMs(),
	}
	s.rooms[roomID] = room
	player.RoomID = roomID
	client.leaveRoom(lobbyChannel)
	client.joinRoom(roomID)
	s.securityLog("room_created", map[string]any{"sid": player.ID, "ip": player.IPAddress, "roomId": roomID, "event": "room:create", "userAgent": client.userAgent})
	s.roomNotice(room, playerShortName(player)+" 进入房间，坐在战斗席 A。")
	client.reply(env.ID, map[string]any{"room": s.roomSnapshot(room, true, true)}, "")
	s.broadcastRoom(roomID, true)
}

func (s *Server) onRoomJoin(client *Client, env wsEnvelope) {
	var p struct {
		RoomID   string `json:"roomId"`
		Password string `json:"password"`
	}
	_ = decodeD(env, &p)
	player := s.requirePlayerQuiet(client)
	room := s.rooms[p.RoomID]
	if player == nil || room == nil {
		client.reply(env.ID, nil, "房间不存在")
		return
	}
	if room.Settings.Password != "" && room.Settings.Password != p.Password {
		msg := "密码错误"
		if s.cfg.Messages != nil && s.cfg.Messages["passwordWrong"] != "" {
			msg = s.cfg.Messages["passwordWrong"]
		}
		client.reply(env.ID, nil, msg)
		return
	}
	leaveResult := s.leaveRoom(player, LeaveSwitchRoom)
	if !leaveResult.OK {
		client.reply(env.ID, nil, leaveResult.Error)
		return
	}
	player.RoomID = room.ID
	joinRole := "观战"
	if s.canAutoSeatOnJoin(room, player) && room.Seats[types.SeatA] == nil {
		room.Seats[types.SeatA] = &HumanSeat{Player: s.publicPlayer(player)}
		joinRole = "战斗席 A"
	} else if s.canAutoSeatOnJoin(room, player) && room.Seats[types.SeatB] == nil {
		room.Seats[types.SeatB] = &HumanSeat{Player: s.publicPlayer(player)}
		joinRole = "战斗席 B"
	} else {
		room.SpectatorIDs = append(room.SpectatorIDs, player.ID)
	}
	client.leaveRoom(lobbyChannel)
	client.joinRoom(room.ID)
	s.securityLog("room_joined", map[string]any{"sid": player.ID, "ip": player.IPAddress, "roomId": room.ID, "event": "room:join", "userAgent": client.userAgent})
	s.roomNotice(room, fmt.Sprintf("%s 进入房间，位置：%s。", playerShortName(player), joinRole))
	s.maybeStartChoosing(room)
	client.reply(env.ID, map[string]any{"room": s.roomSnapshot(room, true, true)}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onRoomLeave(client *Client, env wsEnvelope) {
	player := s.requirePlayerQuiet(client)
	if player == nil {
		return
	}
	leaveResult := s.leaveRoom(player, LeaveManual)
	if !leaveResult.OK {
		client.reply(env.ID, nil, leaveResult.Error)
		return
	}
	client.joinRoom(lobbyChannel)
	client.joinRoom(lobbySuggestionChannel)
	// 离房后补一次大厅全量，避免在房期间退订 lobby 后本地列表过期。
	s.sendFullChannel(client, channelLobby())
	s.securityLog("room_left", map[string]any{"sid": player.ID, "ip": player.IPAddress, "event": "room:leave", "userAgent": client.userAgent})
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onRoomHistory(client *Client, env wsEnvelope) {
	var p struct {
		RoomID string `json:"roomId"`
		Offset int    `json:"offset"`
		Limit  int    `json:"limit"`
	}
	_ = decodeD(env, &p)
	player := s.requirePlayerQuiet(client)
	room := s.rooms[p.RoomID]
	if player == nil || room == nil || player.RoomID != room.ID {
		client.reply(env.ID, nil, "你不在这个房间里")
		return
	}
	safeOffset := p.Offset
	if safeOffset < 0 {
		safeOffset = 0
	}
	safeLimit := p.Limit
	if safeLimit <= 0 {
		safeLimit = roomHistoryPageSize
	}
	if safeLimit > 100 {
		safeLimit = 100
	}
	end := safeOffset + safeLimit
	if end > len(room.RoundHistory) {
		end = len(room.RoundHistory)
	}
	items := []types.RoundHistoryItem{}
	if safeOffset < len(room.RoundHistory) {
		raw := room.RoundHistory[safeOffset:end]
		items = make([]types.RoundHistoryItem, len(raw))
		for i := range raw {
			items[i] = sanitizeRoundHistoryItem(raw[i])
		}
	}
	client.reply(env.ID, map[string]any{"items": items, "total": len(room.RoundHistory)}, "")
}

func (s *Server) onRoomSit(client *Client, env wsEnvelope) {
	var p struct {
		Seat types.SeatKey `json:"seat"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚完成前不能切换座位")
		return
	}
	if text := s.rankedSeatRestrictionText(room, player); text != "" {
		client.reply(env.ID, nil, text)
		return
	}
	if room.Seats[p.Seat] != nil {
		client.reply(env.ID, nil, "这个战斗席已经有人了")
		return
	}
	oldSeat, hasOld := s.seatOf(room, player.ID)
	if hasOld && room.Settings.GameID == types.GameOthello && room.Phase == types.PhaseChoosing {
		client.reply(env.ID, nil, "黑白棋对局进行中不能换座")
		return
	}
	if hasOld && room.Settings.GameID == types.GameTicTacToe && room.Phase == types.PhaseChoosing {
		client.reply(env.ID, nil, "井字棋对局进行中不能换座")
		return
	}
	if hasOld && room.Phase == types.PhaseChoosing && (room.Choices[types.SeatA] != "" || room.Choices[types.SeatB] != "") {
		client.reply(env.ID, nil, "本局已经有人出拳，暂时不能换座")
		return
	}
	if hasOld {
		s.clearSeatForPlayer(room, oldSeat)
	}
	filtered := room.SpectatorIDs[:0]
	for _, id := range room.SpectatorIDs {
		if id != player.ID {
			filtered = append(filtered, id)
		}
	}
	room.SpectatorIDs = filtered
	room.Seats[p.Seat] = &HumanSeat{Player: s.publicPlayer(player)}
	room.Ready[p.Seat] = false
	delete(room.Choices, p.Seat)
	room.SeatedScore[p.Seat] = 0
	room.SeatStats[p.Seat] = emptySeatStats()
	s.roomNotice(room, fmt.Sprintf("%s 坐到战斗席 %s。", playerShortName(player), p.Seat))
	if room.Settings.GameID == types.GameOthello && room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil && room.Phase != types.PhaseChoosing {
		s.resetOthelloRoom(room)
	} else if room.Settings.GameID == types.GameTicTacToe && room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil && room.Phase != types.PhaseChoosing {
		s.resetTicTacToeRoom(room)
	} else {
		s.maybeStartChoosing(room)
	}
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onRoomSpectate(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	leaveCheck := s.canLeaveRoom(player, LeaveSpectate)
	if !leaveCheck.OK {
		client.reply(env.ID, nil, leaveCheck.Error)
		return
	}
	oldSeat, hasOld := s.seatOf(room, player.ID)
	if room.Phase != types.PhasePunishment && hasOld && room.Choices[oldSeat] != "" {
		client.reply(env.ID, nil, "你已经出拳，本局暂时不能离开座位")
		return
	}
	s.handlePunishmentDeparture(room, player, LeaveSpectate)
	if hasOld {
		s.clearSeatForPlayer(room, oldSeat)
	}
	if !containsString(room.SpectatorIDs, player.ID) {
		room.SpectatorIDs = append(room.SpectatorIDs, player.ID)
	}
	s.roomNotice(room, playerShortName(player)+" 进入观战席。")
	if !s.cleanupRoomIfEmpty(room) {
		s.broadcastRoom(room.ID, true)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onRoomMove(client *Client, env wsEnvelope) {
	var p struct {
		Move types.Move `json:"move"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameRPS {
		client.reply(env.ID, nil, "当前玩法不能出拳")
		return
	}
	if p.Move != types.MoveRock && p.Move != types.MoveScissors && p.Move != types.MovePaper && p.Move != types.MoveGiveaway {
		client.reply(env.ID, nil, "出拳无效")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以出拳")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能出拳")
		return
	}
	if p.Move == types.MoveGiveaway && (!ptrBool(player.GiveawayEnabled) || !s.isHumanVsHumanRoom(room)) {
		client.reply(env.ID, nil, "白给只在真人对战并开启白给模式后可用")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚完成前不能出拳")
		return
	}
	if room.Phase == types.PhaseResult {
		s.prepareNextChoice(room)
	}
	if room.Phase != types.PhaseChoosing {
		client.reply(env.ID, nil, "现在还不能出拳")
		return
	}
	if room.Choices[seat] != "" {
		client.reply(env.ID, nil, "你已经出拳，不能修改")
		return
	}
	room.Choices[seat] = p.Move
	if p.Move == types.MoveGiveaway {
		player.GiveawayClicks = intPtr(ptrInt(player.GiveawayClicks) + 1)
		s.addGiveawayValue(player, 2)
	}
	s.maybeBotAct(room)
	oldStatus := room.Status
	s.finishRoundIfReady(room)
	s.broadcastRoom(room.ID, oldStatus != room.Status)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onOthelloReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以准备")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能准备")
		return
	}
	if room.Phase != types.PhaseReady {
		client.reply(env.ID, nil, "当前不能准备")
		return
	}
	room.Ready[seat] = true
	s.roomNotice(room, playerShortName(player)+" 已准备黑白棋。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleOthelloReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloMove(client *Client, env wsEnvelope) {
	var p struct {
		Row int `json:"row"`
		Col int `json:"col"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以落子")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能开始")
		return
	}
	ok2, errMsg := s.applyOthelloMove(room, seat, p.Row, p.Col)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onOthelloSettleMove(client *Client, env wsEnvelope) {
	var p struct {
		Mode string `json:"mode"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	if p.Mode != "normal" && p.Mode != "giveaway" && p.Mode != "tribute" {
		client.reply(env.ID, nil, "结算选择无效")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以结算本手")
		return
	}
	pending := room.Othello
	if pending == nil || pending.PendingSettlement == nil {
		client.reply(env.ID, nil, "当前没有待结算落子")
		return
	}
	if pending.PendingSettlement.Seat != seat {
		client.reply(env.ID, nil, "只能由本手落子玩家选择")
		return
	}
	if pending.PendingSettlement.Forced != "" {
		client.reply(env.ID, nil, "本手已触发强制结算，不能改选")
		return
	}
	ok2, errMsg := s.settleOthelloPendingMove(room, p.Mode, "choice")
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloRequestSurrender(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	fromSeat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请认输")
		return
	}
	if room.Othello == nil || room.Phase != types.PhaseChoosing || room.Othello.Ended {
		client.reply(env.ID, nil, "当前不能申请认输")
		return
	}
	if room.Othello.PendingSettlement != nil {
		client.reply(env.ID, nil, "本手白给/上贡结算完成前不能申请认输")
		return
	}
	toSeat := oppositeSeat(fromSeat)
	if room.Seats[toSeat] == nil {
		client.reply(env.ID, nil, "对手不在战斗席，不能申请认输")
		return
	}
	if room.Othello.SurrenderRequest != nil && room.Othello.SurrenderRequest.FromSeat == fromSeat {
		client.reply(env.ID, nil, "你已经申请认输，正在等待对方确认")
		return
	}
	if room.Othello.SurrenderRequest != nil {
		client.reply(env.ID, nil, "当前已有认输请求，请先处理")
		return
	}
	room.Othello.SurrenderRequest = &types.OthelloSurrenderRequest{
		FromSeat: fromSeat, ToSeat: toSeat, CreatedAt: nowMs(),
	}
	s.roomNotice(room, playerShortName(player)+" 申请认输，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloRespondSurrender(client *Client, env wsEnvelope) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理认输请求")
		return
	}
	if room.Othello == nil || room.Phase != types.PhaseChoosing || room.Othello.Ended {
		client.reply(env.ID, nil, "当前不能处理认输请求")
		return
	}
	request := room.Othello.SurrenderRequest
	if request == nil {
		client.reply(env.ID, nil, "当前没有认输请求")
		return
	}
	if request.ToSeat != seat {
		client.reply(env.ID, nil, "这个认输请求不是发给你的")
		return
	}
	loserSeat := request.FromSeat
	winnerSeat := request.ToSeat
	loserName := occupantName(room.Seats[loserSeat])
	winnerName := occupantName(room.Seats[winnerSeat])
	accept := p.Accept != nil && *p.Accept
	if !accept {
		room.Othello.SurrenderRequest = nil
		s.roomNotice(room, winnerName+" 拒绝认输，对局继续。")
		client.reply(env.ID, map[string]any{"ok": true}, "")
		s.broadcastRoom(room.ID, true)
		return
	}
	ok2, errMsg := s.forceEndOthelloGame(room, types.RoundResult(winnerSeat), forceOthelloOpts{
		Label: fmt.Sprintf("%s认输，%s胜利", loserName, winnerName),
		HistoryNote: "认输",
		Notice: fmt.Sprintf("%s 同意 %s 认输，本局结束。", winnerName, loserName),
		ForfeitRankedFloor: true,
	})
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloEscape(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	loserSeat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以逃跑")
		return
	}
	if room.Othello == nil || room.Phase != types.PhaseChoosing || room.Othello.Ended {
		client.reply(env.ID, nil, "当前不能逃跑")
		return
	}
	if room.Othello.PendingSettlement != nil {
		client.reply(env.ID, nil, "本手白给/上贡结算完成前不能逃跑")
		return
	}
	winnerSeat := oppositeSeat(loserSeat)
	if room.Seats[winnerSeat] == nil {
		client.reply(env.ID, nil, "对手不在战斗席，不能逃跑")
		return
	}
	loserName := playerShortName(player)
	winnerName := occupantName(room.Seats[winnerSeat])
	ok2, errMsg := s.forceEndOthelloGame(room, types.RoundResult(winnerSeat), forceOthelloOpts{
		Label: fmt.Sprintf("%s逃跑，%s胜利", loserName, winnerName),
		HistoryNote: "逃跑",
		Notice: loserName + " 选择逃跑，本局立即判负。",
		ForfeitRankedFloor: true,
		EscapePenaltyRatio: 0.5,
		EscapePenaltyLabel: "逃跑",
	})
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloRestart(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameOthello {
		client.reply(env.ID, nil, "当前房间不是黑白棋")
		return
	}
	if _, ok := s.seatOf(room, player.ID); !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以重新开始")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能重新开始")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚完成前不能重新开始")
		return
	}
	s.resetOthelloRoom(room)
	s.roomNotice(room, playerShortName(player)+" 发起黑白棋再来一局，请双方准备。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onTicTacToeReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameTicTacToe {
		client.reply(env.ID, nil, "当前房间不是井字棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以准备")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能准备")
		return
	}
	if room.Phase != types.PhaseReady {
		client.reply(env.ID, nil, "当前不能准备")
		return
	}
	room.Ready[seat] = true
	s.roomNotice(room, playerShortName(player)+" 已准备井字棋。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleTicTacToeReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onTicTacToeMove(client *Client, env wsEnvelope) {
	var p struct {
		Row int `json:"row"`
		Col int `json:"col"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameTicTacToe {
		client.reply(env.ID, nil, "当前房间不是井字棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以落子")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能开始")
		return
	}
	ok2, errMsg := s.applyTicTacToeMove(room, seat, p.Row, p.Col, "normal")
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onTicTacToeGiveawayChoice(client *Client, env wsEnvelope) {
	var p struct {
		Mode string `json:"mode"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameTicTacToe {
		client.reply(env.ID, nil, "当前房间不是井字棋")
		return
	}
	if room.TicTacToe == nil {
		client.reply(env.ID, nil, "井字棋还没有开始")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以选择白给")
		return
	}
	if room.Phase != types.PhaseChoosing || room.TicTacToe.Ended {
		client.reply(env.ID, nil, "当前不能选择白给")
		return
	}
	if room.TicTacToe.Turn != seat {
		client.reply(env.ID, nil, "还没轮到你落子")
		return
	}
	prompt := room.TicTacToe.GiveawayPrompt
	if prompt == nil || prompt.Seat != seat {
		client.reply(env.ID, nil, "当前没有井字棋白给选择")
		return
	}
	if prompt.Forced {
		client.reply(env.ID, nil, "强制白给中，系统正在随机落子")
		return
	}
	if p.Mode != "normal" && p.Mode != "giveaway" {
		client.reply(env.ID, nil, "白给选择不正确")
		return
	}
	if p.Mode == "normal" {
		s.clearTicTacToeGiveawayTimer(room.ID)
		room.TicTacToe.GiveawayPrompt = nil
		room.ResultText = playerShortName(player) + " 选择不白给，请正常落子。"
		client.reply(env.ID, map[string]any{"ok": true}, "")
		s.broadcastRoom(room.ID, true)
		return
	}
	ok2, row, col, errMsg := s.applyTicTacToeRandomMove(room, seat, "giveaway")
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, fmt.Sprintf("%s 选择白给落子，系统随机落在第 %d 行第 %d 列。", playerShortName(player), row+1, col+1))
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onTicTacToeRestart(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameTicTacToe {
		client.reply(env.ID, nil, "当前房间不是井字棋")
		return
	}
	if _, ok := s.seatOf(room, player.ID); !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以重新开始")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能重新开始")
		return
	}
	if room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "惩罚完成前不能重新开始")
		return
	}
	s.resetTicTacToeRoom(room)
	s.roomNotice(room, playerShortName(player)+" 发起井字棋再来一局，请双方准备。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentSubmit(client *Client, env wsEnvelope) {
	var p struct {
		Text     string `json:"text"`
		ImageURL string `json:"imageUrl"`
	}
	_ = decodeD(env, &p)
	player := s.requirePlayerQuiet(client)
	if player == nil || player.RoomID == "" {
		client.reply(env.ID, nil, "你当前不需要提交惩罚")
		return
	}
	room := s.rooms[player.RoomID]
	if room == nil || !containsString(room.PunishedPlayerIDs, player.ID) {
		client.reply(env.ID, nil, "你当前不需要提交惩罚")
		return
	}
	cleanProofText := cleanText(p.Text, 500)
	cleanImageURL := safeUploadURL(p.ImageURL)
	if cleanProofText == "" {
		client.reply(env.ID, nil, "请填写文字证明")
		return
	}
	if p.ImageURL != "" && cleanImageURL == "" {
		client.reply(env.ID, nil, "图片地址无效")
		return
	}
	if p.ImageURL != "" && room.Settings.AllowProofImage != nil && !*room.Settings.AllowProofImage {
		client.reply(env.ID, nil, "本房间已关闭图片证明")
		return
	}
	var taskText string
	if len(room.RoundHistory) > 0 {
		for _, t := range room.RoundHistory[0].PunishmentTasks {
			if t.PlayerID == player.ID {
				taskText = t.TaskText
				break
			}
		}
	}
	for _, pr := range room.Proofs {
		if pr.PlayerID == player.ID && pr.RedoTaskText != "" {
			taskText = pr.RedoTaskText
		}
	}
	if room.Settings.PunishmentSource == "player" && strings.TrimSpace(taskText) == "" {
		client.reply(env.ID, nil, "等待对方发布惩罚任务")
		return
	}
	// 人人对战且开启「需对手确认」时保持 pending，由胜方审批（系统任务与自定义任务一致）。
	// 仅 Bot 对战 / 无人类对手 / 房间关闭确认 时系统自动通过。
	approvedBySystem := s.opponentIsBot(room, player.ID) || s.humanOpponent(room, player.ID) == nil || !room.Settings.RequireOpponentConfirm
	submittedAt := nowMs()
	filtered := room.Proofs[:0]
	for _, pr := range room.Proofs {
		if pr.PlayerID != player.ID {
			filtered = append(filtered, pr)
		}
	}
	status := "pending"
	confirmedBy, reviewedBy := "", ""
	var reviewedAt *int64
	if approvedBySystem {
		status = "approved"
		confirmedBy = "system-auto-confirm"
		reviewedBy = "system-auto-confirm"
		reviewedAt = &submittedAt
	}
	room.Proofs = append(filtered, types.PunishmentProof{
		PlayerID: player.ID, Text: cleanProofText, ImageURL: cleanImageURL, TaskText: taskText,
		Status: status, ConfirmedBy: confirmedBy, ReviewedBy: reviewedBy, ReviewedAt: reviewedAt, SubmittedAt: submittedAt,
	})
	s.attachProofToLatestHistory(room, types.HistoryProof{
		PlayerID: player.ID, PlayerName: playerShortName(player), Text: cleanProofText, ImageURL: cleanImageURL,
		TaskText: taskText, Status: status, ReviewedBy: reviewedBy, ReviewedAt: reviewedAt, SubmittedAt: submittedAt,
	})
	// 先应答再广播，避免广播路径异常时客户端收不到 ack
	client.reply(env.ID, map[string]any{"ok": true}, "")
	// 提交后广播（含 history 中的证明），胜方无需刷新即可审核。
	if s.punishmentComplete(room) {
		s.resetForNextRound(room)
	} else {
		s.broadcastRoom(room.ID, false)
	}
}

func (s *Server) onPunishmentAssignTask(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID string `json:"playerId"`
		TaskText string `json:"taskText"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Phase != types.PhasePunishment || room.Settings.PunishmentSource != "player" {
		client.reply(env.ID, nil, "当前不是玩家发布任务模式")
		return
	}
	if !containsString(room.PunishedPlayerIDs, p.PlayerID) {
		client.reply(env.ID, nil, "这个玩家当前不需要惩罚")
		return
	}
	if player.ID == p.PlayerID {
		client.reply(env.ID, nil, "不能给自己发布任务")
		return
	}
	reviewerSeat, ok1 := s.seatOf(room, player.ID)
	targetSeat, ok2 := s.seatOf(room, p.PlayerID)
	if !ok1 || !ok2 || reviewerSeat == targetSeat {
		client.reply(env.ID, nil, "只能给对手发布任务")
		return
	}
	var task *types.PunishmentTask
	if len(room.RoundHistory) > 0 {
		for i := range room.RoundHistory[0].PunishmentTasks {
			if room.RoundHistory[0].PunishmentTasks[i].PlayerID == p.PlayerID {
				task = &room.RoundHistory[0].PunishmentTasks[i]
				break
			}
		}
	}
	expectedAssigner := s.taskAssigner(room, p.PlayerID)
	assignedBy := ""
	if task != nil {
		assignedBy = task.AssignedBy
	}
	if assignedBy == "" && expectedAssigner != nil {
		assignedBy = expectedAssigner.ID
	}
	if task == nil || assignedBy != player.ID {
		client.reply(env.ID, nil, "这条任务不由你发布")
		return
	}
	if strings.TrimSpace(task.TaskText) != "" {
		client.reply(env.ID, nil, "任务已经发布")
		return
	}
	cleanTask := cleanText(p.TaskText, 300)
	if cleanTask == "" {
		client.reply(env.ID, nil, "请填写惩罚任务")
		return
	}
	// 玩家发布任务也支持 {winner}/{loser}（胜者=发布方，败者=受罚方）
	loserName := ""
	if target := s.players[p.PlayerID]; target != nil {
		loserName = playerShortName(target)
	}
	cleanTask = applyPunishmentPlaceholders(cleanTask, loserName, playerShortName(player))
	s.updatePunishmentTask(room, p.PlayerID, cleanTask, player)
	s.broadcastRoom(room.ID, false)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentReview(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID     string `json:"playerId"`
		Action       string `json:"action"`
		RedoTaskText string `json:"redoTaskText"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if _, ok := s.seatOf(room, player.ID); !ok {
		client.reply(env.ID, nil, "只有对战玩家可以确认")
		return
	}
	if player.ID == p.PlayerID {
		client.reply(env.ID, nil, "不能审核自己的证明")
		return
	}
	targetSeat, ok := s.seatOf(room, p.PlayerID)
	reviewerSeat, _ := s.seatOf(room, player.ID)
	if !ok || targetSeat == reviewerSeat {
		client.reply(env.ID, nil, "只能审核对手的证明")
		return
	}
	var proof *types.PunishmentProof
	for i := range room.Proofs {
		if room.Proofs[i].PlayerID == p.PlayerID {
			proof = &room.Proofs[i]
			break
		}
	}
	if proof == nil {
		client.reply(env.ID, nil, "对方还没有提交证明")
		return
	}
	reviewedAt := nowMs()
	if p.Action == "reject" {
		cleanTask := cleanText(p.RedoTaskText, 300)
		if cleanTask == "" {
			client.reply(env.ID, nil, "请填写新的惩罚任务")
			return
		}
		proof.Status = "rejected"
		proof.ReviewedBy = player.ID
		proof.ReviewedAt = &reviewedAt
		proof.RejectReason = "需要重做"
		proof.RedoTaskText = cleanTask
		proof.ConfirmedBy = ""
		s.updatePunishmentTask(room, p.PlayerID, cleanTask, nil)
		s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
			Status: "rejected", ReviewedBy: player.ID, ReviewedAt: &reviewedAt,
			RejectReason: "需要重做", RedoTaskText: cleanTask,
		})
		s.broadcastRoom(room.ID, false)
		client.reply(env.ID, map[string]any{"ok": true}, "")
		return
	}
	var reviewMessage string
	if p.Action == "forgive" {
		reviewMessage = s.applyForgiveReview(room, player.ID, p.PlayerID)
	}
	proof.Status = "approved"
	proof.ConfirmedBy = player.ID
	proof.ReviewedBy = player.ID
	proof.ReviewedAt = &reviewedAt
	proof.RejectReason = reviewMessage
	s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
		Status: "approved", ReviewedBy: player.ID, ReviewedAt: &reviewedAt, RejectReason: reviewMessage,
	})
	if s.punishmentComplete(room) {
		s.resetForNextRound(room)
	} else {
		s.broadcastRoom(room.ID, false)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentConfirm(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID string `json:"playerId"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if _, ok := s.seatOf(room, player.ID); !ok {
		client.reply(env.ID, nil, "只有对战玩家可以确认")
		return
	}
	if player.ID == p.PlayerID {
		client.reply(env.ID, nil, "不能审核自己的证明")
		return
	}
	targetSeat, ok := s.seatOf(room, p.PlayerID)
	reviewerSeat, _ := s.seatOf(room, player.ID)
	if !ok || targetSeat == reviewerSeat {
		client.reply(env.ID, nil, "只能审核对手的证明")
		return
	}
	var proof *types.PunishmentProof
	for i := range room.Proofs {
		if room.Proofs[i].PlayerID == p.PlayerID {
			proof = &room.Proofs[i]
			break
		}
	}
	if proof == nil {
		client.reply(env.ID, nil, "对方还没有提交证明")
		return
	}
	reviewedAt := nowMs()
	proof.Status = "approved"
	proof.ConfirmedBy = player.ID
	proof.ReviewedBy = player.ID
	proof.ReviewedAt = &reviewedAt
	s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
		Status: "approved", ReviewedBy: player.ID, ReviewedAt: &reviewedAt,
	})
	if s.punishmentComplete(room) {
		s.resetForNextRound(room)
	} else {
		s.broadcastRoom(room.ID, false)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onChatSend(client *Client, env wsEnvelope) {
	var p struct {
		RoomID string `json:"roomId"`
		Text   string `json:"text"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	cleanMessageText := cleanText(p.Text, 300)
	if cleanMessageText == "" {
		client.reply(env.ID, nil, "请输入聊天内容")
		return
	}
	s.refreshNameWarState(player, nowMs())
	s.refreshPlayerSnapshots(player)
	pub := s.publicPlayer(player)
	message := types.ChatMessage{
		ID: randomID(), RoomID: p.RoomID, PlayerID: player.ID,
		Author: player.DisplayName, AuthorPlayer: &pub, Text: cleanMessageText, At: nowMs(),
	}
	if p.RoomID != "" {
		room := s.rooms[p.RoomID]
		if room == nil || player.RoomID != p.RoomID {
			client.reply(env.ID, nil, "你不在这个房间里")
			return
		}
		message.AuthorRole = s.roomRole(room, player.ID)
		s.appendRoomChat(room, message)
		s.emitToRoom(p.RoomID, "chat:append", message)
	} else {
		s.appendLobbyChat(message)
		s.emitLobbyChatAppend(message)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onSuggestionAdd(client *Client, env wsEnvelope) {
	var p struct {
		Text string `json:"text"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	cleanSuggestionText := cleanText(p.Text, 500)
	if cleanSuggestionText == "" {
		client.reply(env.ID, nil, "请输入留言内容")
		return
	}
	pub := s.publicPlayer(player)
	suggestion := types.Suggestion{
		ID: randomID(), PlayerID: player.ID, Author: player.DisplayName,
		AuthorPlayer: &pub, Text: cleanSuggestionText, At: nowMs(),
	}
	s.appendSuggestion(suggestion)
	s.emitToRoom(lobbySuggestionChannel, "suggestion:append", suggestion)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onAdminAction(client *Client, env wsEnvelope) {
	var p struct {
		Action          string             `json:"action"`
		RoomID          string             `json:"roomId"`
		PlayerID        string             `json:"playerId"`
		Name            string             `json:"name"`
		RankedPoints    *float64           `json:"rankedPoints"`
		Title           string             `json:"title"`
		Message         string             `json:"message"`
		DurationSeconds *float64           `json:"durationSeconds"`
		OthelloResult   types.RoundResult  `json:"othelloResult"`
	}
	_ = decodeD(env, &p)
	admin := s.getPlayerByClientID(client.id)
	_, isAdminSocket := s.adminClientIDs[client.id]
	if (admin == nil || !ptrBool(admin.IsAdmin)) && !isAdminSocket {
		client.reply(env.ID, nil, "需要管理员权限")
		return
	}
	roomDeleted := false
	changedPlayerRoomID := ""
	if p.Action == "clearSuggestions" {
		s.suggestions = nil
	}
	if p.Action == "clearLobbyChat" {
		s.lobbyChat = nil
	}
	if p.Action == "broadcastAnnouncement" {
		cleanMessage := cleanText(p.Message, 200)
		if cleanMessage == "" {
			client.reply(env.ID, nil, "公告内容不能为空")
			return
		}
		secs := 8
		if p.DurationSeconds != nil {
			secs = int(*p.DurationSeconds + 0.5)
		}
		secs = clamp(secs, 3, 60)
		s.emitVolatileAll("announcement:show", map[string]any{
			"id": randomID(), "message": cleanMessage, "durationMs": secs * 1000, "createdAt": nowMs(),
		})
	}
	if p.Action == "closeRoom" && p.RoomID != "" {
		if room := s.rooms[p.RoomID]; room != nil {
			s.emitToRoom(p.RoomID, "room:closed", map[string]any{"message": fmt.Sprintf("房间 %s 已被管理员关闭。", room.Settings.Name)})
			ids := append([]string{}, room.SpectatorIDs...)
			if room.Seats[types.SeatA] != nil {
				ids = append(ids, room.Seats[types.SeatA].GetID())
			}
			if room.Seats[types.SeatB] != nil {
				ids = append(ids, room.Seats[types.SeatB].GetID())
			}
			for _, id := range ids {
				if pl := s.players[id]; pl != nil {
					pl.RoomID = ""
					s.clearDisconnectHold(pl)
				}
			}
			for _, c := range s.clients {
				c.leaveRoom(p.RoomID)
			}
			if t := s.botTimers[p.RoomID]; t != nil {
				t.Stop()
				delete(s.botTimers, p.RoomID)
			}
			s.clearOthelloSettlementTimer(p.RoomID)
			s.clearTicTacToeGiveawayTimer(p.RoomID)
			s.clearRoomBroadcastTimer(p.RoomID)
			s.dropSyncChannel(channelRoom(p.RoomID))
			delete(s.rooms, p.RoomID)
			roomDeleted = true
		}
	}
	if p.Action == "clearRoomChat" && p.RoomID != "" {
		if room := s.rooms[p.RoomID]; room != nil {
			room.Chat = []types.ChatMessage{}
		}
	}
	if p.Action == "forceNext" && p.RoomID != "" {
		if room := s.rooms[p.RoomID]; room != nil {
			s.resetForNextRound(room)
		}
	}
	if p.Action == "forceOthelloRestart" && p.RoomID != "" {
		room := s.rooms[p.RoomID]
		if room == nil || room.Settings.GameID != types.GameOthello {
			client.reply(env.ID, nil, "当前房间不是黑白棋房间")
			return
		}
		s.flushOthelloPendingSettlement(room)
		s.resetOthelloRoom(room)
		s.roomNotice(room, "管理员已重开黑白棋对局。")
	}
	if p.Action == "forceOthelloEnd" && p.RoomID != "" {
		room := s.rooms[p.RoomID]
		if room == nil || room.Settings.GameID != types.GameOthello {
			client.reply(env.ID, nil, "当前房间不是黑白棋房间")
			return
		}
		if p.OthelloResult != types.ResultA && p.OthelloResult != types.ResultB && p.OthelloResult != types.ResultDraw {
			client.reply(env.ID, nil, "请选择黑方胜、白方胜或平局")
			return
		}
		forcedResult := p.OthelloResult
		if p.OthelloResult != types.ResultDraw && room.Othello != nil {
			if p.OthelloResult == types.ResultA {
				forcedResult = types.RoundResult(room.Othello.BlackSeat)
			} else {
				forcedResult = types.RoundResult(oppositeSeat(room.Othello.BlackSeat))
			}
		}
		ok, errMsg := s.forceEndOthelloGame(room, forcedResult, forceOthelloOpts{})
		if !ok {
			client.reply(env.ID, nil, errMsg)
			return
		}
	}
	if p.Action == "kick" && p.PlayerID != "" {
		if pl := s.players[p.PlayerID]; pl != nil {
			s.leaveRoom(pl, LeaveAdminKick)
			s.clearDisconnectHold(pl)
			delete(s.players, pl.ID)
			delete(s.tokenToPlayer, pl.Token)
			if pl.PlayerID != "" {
				delete(s.playerIdToID, pl.PlayerID)
			}
			if pl.CurrentSID != "" && s.sidToPlayerID[pl.CurrentSID] == pl.ID {
				delete(s.sidToPlayerID, pl.CurrentSID)
			}
			if pl.Persistent {
				s.requestPersist("important")
			}
			if pl.SocketID != "" {
				s.emitToClient(pl.SocketID, "player:kicked", map[string]any{})
			}
		}
	}
	if p.Action == "editPlayer" && p.PlayerID != "" {
		pl := s.players[p.PlayerID]
		if pl == nil {
			client.reply(env.ID, nil, "玩家不存在")
			return
		}
		cleanName := cleanText(p.Name, 12)
		cleanTitle := cleanText(p.Title, 18)
		if len([]rune(cleanName)) < 2 {
			client.reply(env.ID, nil, "名字至少需要 2 个字")
			return
		}
		if p.RankedPoints == nil {
			client.reply(env.ID, nil, "积分格式不正确")
			return
		}
		pl.Name = cleanName
		pl.NameWarOriginalName = cleanName
		s.setRankedPointsByAdmin(pl, int(*p.RankedPoints))
		if cleanTitle != "" {
			pl.Stats.Title = cleanTitle
		}
		pl.DisplayName = formatDisplayName(pl)
		s.refreshPlayerSnapshots(pl)
		changedPlayerRoomID = pl.RoomID
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastLobby()
	if p.RoomID != "" && !roomDeleted {
		s.broadcastRoom(p.RoomID, false)
	}
	if changedPlayerRoomID != "" && changedPlayerRoomID != p.RoomID {
		s.broadcastRoom(changedPlayerRoomID, false)
	}
}
