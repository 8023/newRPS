package server

import (
	"crypto/subtle"
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
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
	if s.activeRoomsOwnedBy(player.ID) >= s.cfg.AccessControl.MaxActiveRoomsPerOwner {
		client.reply(env.ID, nil, fmt.Sprintf("同时最多只能开 %d 个房间，请先关闭其他房间", s.cfg.AccessControl.MaxActiveRoomsPerOwner))
		return
	}
	settings := p.Settings
	gameID := settings.GameID
	if gameID != types.GameOthello && gameID != types.GameTicTacToe && gameID != types.GameLiarsDice && gameID != types.GameGomoku && gameID != types.GameJungle && gameID != types.GameChess && gameID != types.GameCoinFlip {
		gameID = types.GameRPS
	}
	settings.GameID = gameID
	if gameID == types.GameCoinFlip {
		// 猜硬币没有真人对手：结构性关闭排位/倍率/极限/平局双罚/需对手确认/每子惩罚，
		// 并强制开启惩罚——这是它唯一的玩法内容，不靠调用点自觉。玩家发布任务模式
		// 在下面的 PunishmentSource 校验里单独拒绝（没有对手可以发布任务）。
		settings.EnableRanked = false
		settings.EnableRankMultiplier = false
		settings.RankMultiplier = 1
		settings.EnableExtremeRanked = false
		settings.TieDoublePunish = false
		settings.RequireOpponentConfirm = false
		settings.EnablePerPiecePunishment = false
		settings.EnablePunishment = true
	}
	// 排位档位按游戏从 games.json 读取（后台可调），不在档位内回退默认档。
	stakeTiers := s.stakeTiersFor(gameID)
	if !containsInt(stakeTiers, int(settings.Stake)) {
		settings.Stake = types.RankStake(stakeTiers[0])
	}
	if settings.AllowProofImage == nil {
		settings.AllowProofImage = boolPtr(true)
	}
	settings.PunishmentSource = normalizePunishmentSource(settings.PunishmentSource)
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
		switch settings.OthelloBoardTheme {
		case "classic", "pastel", "midnight", "wood", "neon":
		default:
			settings.OthelloBoardTheme = "classic"
		}
		settings.OthelloUndoLimit = clampUndoLimit(settings.OthelloUndoLimit)
		settings.OthelloMoveSeconds = clampMoveSeconds(settings.OthelloMoveSeconds)
		settings.OthelloGameMinutes = clampGameMinutes(settings.OthelloGameMinutes)
	} else if settings.GameID == types.GameTicTacToe {
		switch settings.TicTacToeBoardTheme {
		case "paper", "mint", "midnight", "candy", "arcade":
		default:
			settings.TicTacToeBoardTheme = "paper"
		}
	} else if settings.GameID == types.GameLiarsDice {
		minP, maxP := liarsDiceRosterBounds(settings)
		settings.LiarsDiceMinPlayers = minP
		settings.LiarsDiceMaxPlayers = maxP
		// 大话骰每局永远 1 胜 1 负，旁观者记平但不触发「平局双罚」；强制关闭以免歧义。
		settings.TieDoublePunish = false
	} else if settings.GameID == types.GameGomoku {
		switch settings.GomokuBoardTheme {
		case "classic", "pastel", "midnight", "wood", "neon":
		default:
			settings.GomokuBoardTheme = "wood"
		}
		settings.GomokuUndoLimit = clampUndoLimit(settings.GomokuUndoLimit)
		settings.GomokuMoveSeconds = clampMoveSeconds(settings.GomokuMoveSeconds)
		settings.GomokuGameMinutes = clampGameMinutes(settings.GomokuGameMinutes)
	} else if settings.GameID == types.GameJungle {
		switch settings.JungleBoardTheme {
		case "forest", "bamboo", "river", "dusk", "night":
		default:
			settings.JungleBoardTheme = "forest"
		}
		settings.JungleUndoLimit = clampUndoLimit(settings.JungleUndoLimit)
		settings.JungleMoveSeconds = clampMoveSeconds(settings.JungleMoveSeconds)
		settings.JungleGameMinutes = clampGameMinutes(settings.JungleGameMinutes)
	} else if settings.GameID == types.GameChess {
		switch settings.ChessBoardTheme {
		case "classic", "wood", "midnight", "marble", "green":
		default:
			settings.ChessBoardTheme = "classic"
		}
		settings.ChessUndoLimit = clampUndoLimit(settings.ChessUndoLimit)
		settings.ChessMoveSeconds = clampMoveSeconds(settings.ChessMoveSeconds)
		settings.ChessGameMinutes = clampGameMinutes(settings.ChessGameMinutes)
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
	if settings.GameID == types.GameCoinFlip && settings.PunishmentSource == "player" {
		client.reply(env.ID, nil, "猜硬币没有真人对手，不能使用玩家发布任务")
		return
	}
	// 废弃的旧任务类型字段清空；按来源校验并归一新字段。
	settings.PunishmentIDs = []string{}
	settings.PunishmentID = ""
	if settings.EnablePunishment {
		switch settings.PunishmentSource {
		case "player":
			settings.PunishmentTagsIncluded = []string{}
			settings.PunishmentTagsExcluded = []string{}
			settings.PunishmentSeriesID = ""
		case "series":
			settings.PunishmentTagsIncluded = []string{}
			settings.PunishmentTagsExcluded = []string{}
			if s.findSeriesByID(settings.PunishmentSeriesID) == nil {
				client.reply(env.ID, nil, "请选择一个有效的系列任务")
				return
			}
		default: // random
			settings.PunishmentSeriesID = ""
			settings.PunishmentTagsIncluded = s.filterValidTagIDs(settings.PunishmentTagsIncluded)
			settings.PunishmentTagsExcluded = s.filterValidTagIDs(settings.PunishmentTagsExcluded)
			// 同一标签不能既选中又拒绝；拒绝优先剔除。
			if len(settings.PunishmentTagsExcluded) > 0 {
				excl := map[string]struct{}{}
				for _, id := range settings.PunishmentTagsExcluded {
					excl[id] = struct{}{}
				}
				var incl []string
				for _, id := range settings.PunishmentTagsIncluded {
					if _, bad := excl[id]; !bad {
						incl = append(incl, id)
					}
				}
				settings.PunishmentTagsIncluded = incl
			}
		}
	} else {
		settings.PunishmentTagsIncluded = []string{}
		settings.PunishmentTagsExcluded = []string{}
		settings.PunishmentSeriesID = ""
	}
	if !settings.EnablePunishment || (settings.GameID != types.GameChess && settings.GameID != types.GameJungle) {
		settings.EnablePerPiecePunishment = false
	}
	settings.Tags = s.normalizeRoomTags(settings)
	settings.EnableTags = len(settings.Tags) > 0
	settings.Name = s.normalizeRoomName(settings)
	settings.RoomBackgroundImage = s.randomRoomBackground(settings)
	if settings.EnableRanked && ptrBool(player.ExtremeModeEnabled) != settings.EnableExtremeRanked {
		if ptrBool(player.ExtremeModeEnabled) {
			client.reply(env.ID, nil, "极限模式玩家只能创建极限排位房")
		} else {
			client.reply(env.ID, nil, "只有极限模式玩家可以创建极限排位房")
		}
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
	leaveResult := s.leaveRoom(player, LeaveSwitchRoom)
	if !leaveResult.OK {
		client.reply(env.ID, nil, leaveResult.Error)
		return
	}
	roomID := randomID()
	status, phase := "waiting", types.PhaseReady
	seats := map[types.SeatKey]SeatOccupant{types.SeatA: &HumanSeat{Player: s.publicPlayer(player)}, types.SeatB: nil}
	isLiarsDice := settings.GameID == types.GameLiarsDice
	if isLiarsDice {
		seats = map[types.SeatKey]SeatOccupant{types.SeatA: nil, types.SeatB: nil}
	}
	room := &RoomState{
		ID: roomID, OwnerID: player.ID, CreatorName: playerShortName(player), Settings: settings,
		Status: status, UpdatedAt: nowMs(), Phase: phase, Seats: seats,
		SpectatorIDs: []string{}, Ready: map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false},
		Choices: map[types.SeatKey]types.Move{}, PunishedPlayerIDs: []string{}, Proofs: []types.PunishmentProof{},
		Score:        map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatedScore:  map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SeatStats:    map[types.SeatKey]types.SeatStats{types.SeatA: emptySeatStats(), types.SeatB: emptySeatStats()},
		RoundHistory: []types.RoundHistoryItem{}, LockedSeatIDs: map[string]struct{}{},
		DisconnectForfeits: map[string]DisconnectForfeit{}, CreatedAt: nowMs(),
		// PunishmentTaskProgress 零值即 nil map，按玩家 ID 懒初始化，随房间存活期累积，销毁即释放。
	}
	if isLiarsDice {
		minP, maxP := liarsDiceRosterBounds(settings)
		room.LiarsDice = &types.LiarsDiceState{
			ParticipantIDs: []string{}, ReadyPlayerIDs: []string{}, DiceCounts: map[string]int{},
			BidHistory: []types.LiarsDiceBid{}, MinPlayers: minP, MaxPlayers: maxP,
		}
		room.LiarsDiceHands = map[string][]int{}
		room.LiarsDiceDisconnectForfeits = map[string]LiarsDiceDisconnectForfeit{}
		room.SpectatorIDs = []string{player.ID}
	}
	s.rooms[roomID] = room
	player.RoomID = roomID
	// 随机任务模式下，建房成功后给数据分析记一笔标签选中/排除（标签三态偏好本身现在
	// 只存浏览器本地，不再落玩家档案）；系列任务模式记一笔系列选中。
	if settings.EnablePunishment && settings.PunishmentSource == "random" {
		for _, id := range settings.PunishmentTagsIncluded {
			s.logAnalyticsServerEvent("punish_tag_include", id, 1, player.ID, roomID)
		}
		for _, id := range settings.PunishmentTagsExcluded {
			s.logAnalyticsServerEvent("punish_tag_exclude", id, 1, player.ID, roomID)
		}
	} else if settings.EnablePunishment && settings.PunishmentSource == "series" && settings.PunishmentSeriesID != "" {
		s.logAnalyticsServerEvent("punish_series_select", settings.PunishmentSeriesID, 1, player.ID, roomID)
	}
	s.clientLeaveRoom(client, lobbyChannel)
	s.clientJoinRoom(client, roomID)
	if s.eventDB != nil {
		if err := s.eventDB.insertRoomEvent(roomEventInput{
			At: nowMs(), RoomID: roomID, RoomName: settings.Name, GameID: string(settings.GameID),
			UserID: player.ID, UserName: playerShortName(player), Action: "create",
			PasswordHash: hashRoomPassword(settings.Password), IP: player.IPAddress,
		}); err != nil {
			s.errorLog("room_event_insert_failed", err.Error())
		}
	}
	if isLiarsDice {
		s.roomNotice(room, playerShortName(player)+" 创建了大话骰房间，当前为观战。")
	} else if settings.GameID == types.GameCoinFlip {
		s.roomNotice(room, playerShortName(player)+" 创建了猜硬币房间，坐在参战席。")
		// 猜硬币没有"双方坐满才开局"的等待逻辑，座位 A 一有人就直接可抛。
		s.maybeStartChoosing(room)
	} else {
		s.roomNotice(room, playerShortName(player)+" 进入房间，坐在战斗席 A。")
	}
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
	if player.RoomID == room.ID {
		// 已经在这个房间里（典型场景：惩罚未完成时刷新页面，前端状态还没恢复完就又发了
		// 一次 room:join）：不当成“换房”处理，跳过密码校验与 leaveRoom，避免被惩罚阶段
		// 的禁止离开检查挡在门外，直接重新同步频道并回传当前房间快照即可。
		s.clientLeaveRoom(client, lobbyChannel)
		s.clientJoinRoom(client, room.ID)
		client.reply(env.ID, map[string]any{"room": s.roomSnapshot(room, true, true)}, "")
		return
	}
	if room.Settings.Password != "" {
		// 只在比对失败时扣配额（与 admin 登录失败限流一致）：猜对不该消耗爆破计数，
		// 否则 NAT 后多人共享同一 IP 桶时，正常进房也会被 5 次/分钟卡住。
		if subtle.ConstantTimeCompare([]byte(room.Settings.Password), []byte(p.Password)) != 1 {
			if !s.checkRateLimit("room_password:"+room.ID+":"+client.ipAddress, RateLimitOptions{Limit: 5, WindowMs: 60_000, CooldownMs: 300_000}) {
				s.securityLog("room_password_bruteforce", map[string]any{"sid": client.sid, "ip": client.ipAddress, "roomId": room.ID})
				client.reply(env.ID, nil, "密码错误次数过多，请稍后再试")
				return
			}
			msg := "密码错误"
			if s.cfg.Messages != nil && s.cfg.Messages["passwordWrong"] != "" {
				msg = s.cfg.Messages["passwordWrong"]
			}
			client.reply(env.ID, nil, msg)
			return
		}
	}
	leaveResult := s.leaveRoom(player, LeaveSwitchRoom)
	if !leaveResult.OK {
		client.reply(env.ID, nil, leaveResult.Error)
		return
	}
	player.RoomID = room.ID
	joinRole := "观战"
	if room.Settings.GameID == types.GameLiarsDice {
		room.SpectatorIDs = append(room.SpectatorIDs, player.ID)
	} else if s.canAutoSeatOnJoin(room, player) && room.Seats[types.SeatA] == nil {
		room.Seats[types.SeatA] = &HumanSeat{Player: s.publicPlayer(player)}
		joinRole = "战斗席 A"
		s.notifySeatFilled(room, types.SeatA)
	} else if room.Settings.GameID != types.GameCoinFlip && s.canAutoSeatOnJoin(room, player) && room.Seats[types.SeatB] == nil {
		// 猜硬币没有战斗席 B（永远锁死），进房自动入座只尝试座位 A，坐不上就落观战。
		room.Seats[types.SeatB] = &HumanSeat{Player: s.publicPlayer(player)}
		joinRole = "战斗席 B"
		s.notifySeatFilled(room, types.SeatB)
	} else {
		room.SpectatorIDs = append(room.SpectatorIDs, player.ID)
	}
	s.clientLeaveRoom(client, lobbyChannel)
	s.clientJoinRoom(client, room.ID)
	if s.eventDB != nil {
		if err := s.eventDB.insertRoomEvent(roomEventInput{
			At: nowMs(), RoomID: room.ID, RoomName: room.Settings.Name, GameID: string(room.Settings.GameID),
			UserID: player.ID, UserName: playerShortName(player), Action: "join", Role: joinRole, IP: player.IPAddress,
		}); err != nil {
			s.errorLog("room_event_insert_failed", err.Error())
		}
	}
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
	s.clientJoinRoom(client, lobbyChannel)
	s.clientJoinRoom(client, lobbySuggestionChannel)
	// 离房后补一次大厅全量，避免在房期间退订 lobby 后本地列表过期。
	s.sendFullChannel(client, channelLobby())
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
	// 大话骰不用 A/B 座位（用参战名单），拦掉座位 RPC 以免观战者往 Seats 里注入脏数据。
	if room.Settings.GameID == types.GameLiarsDice {
		client.reply(env.ID, nil, "大话骰请使用「加入参战席」")
		return
	}
	// 猜硬币没有战斗席 B（永远锁死，只用座位 A）。
	if room.Settings.GameID == types.GameCoinFlip && p.Seat != types.SeatA {
		client.reply(env.ID, nil, "猜硬币没有战斗席 B，另一位只能观战")
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
	if hasOld && room.Settings.GameID == types.GameGomoku && room.Phase == types.PhaseChoosing {
		client.reply(env.ID, nil, "五子棋对局进行中不能换座")
		return
	}
	if hasOld && room.Settings.GameID == types.GameJungle && room.Phase == types.PhaseChoosing {
		client.reply(env.ID, nil, "斗兽棋对局进行中不能换座")
		return
	}
	if hasOld && room.Settings.GameID == types.GameChess && room.Phase == types.PhaseChoosing {
		client.reply(env.ID, nil, "国际象棋对局进行中不能换座")
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
	s.notifySeatFilled(room, p.Seat)
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
	// 大话骰参战玩家去观战：从参战名单移除（否则名单里残留一个"其实在观战"的人）。
	if room.Settings.GameID == types.GameLiarsDice {
		s.leaveLiarsDiceRoster(room, player.ID)
		s.scheduleLiarsDiceReadyStart(room)
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
	if p.Move == types.MoveGiveaway && (!ptrBool(player.GiveawayEnabled) || !s.isFullHumanRoom(room)) {
		client.reply(env.ID, nil, "白给只在双方都入座并开启白给模式后可用")
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
	// 白给点击数/白给值加成统一挪到 finishRoundIfReady 真正结算的那一刻才发放（见该函数
	// 内的说明）：选择本身不再有任何即时副作用，允许下面 room:move:withdraw 安全撤回。
	// 对手还没出拳：提醒 Ta 该出拳了；双方都出了就直接进结算，不用再提醒。
	if room.Choices[oppositeSeat(seat)] == "" {
		s.notifyOpponentTurn(room, oppositeSeat(seat))
	}
	oldStatus := room.Status
	s.finishRoundIfReady(room)
	s.broadcastRoom(room.ID, oldStatus != room.Status)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// onRoomMoveWithdraw 撤回自己在本局已经出的拳（仅锤子剪刀布，仅出拳阶段、对局还没结算
// 时）：解决"一方出拳后另一方挂机不出，出拳方彻底卡在房间里退不出去"的问题——挂机不会
// 触发断线判负（那套计时器只在 socket 真断开时才走），撤回是唯一的自救手段。
// 白给可以撤回（选择本身不再有即时副作用，见 onRoomMove 的注释），但被主人强制的白给
// 不行——那是"身不由己"的强制机制，允许撤回就名存实亡了。
func (s *Server) onRoomMoveWithdraw(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameRPS {
		client.reply(env.ID, nil, "当前玩法不支持撤回出拳")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以撤回出拳")
		return
	}
	if room.Phase != types.PhaseChoosing {
		client.reply(env.ID, nil, "现在不能撤回出拳")
		return
	}
	if room.Choices[seat] == "" {
		client.reply(env.ID, nil, "你还没有出拳")
		return
	}
	if forcedGiveawayMasterName(room, seat) != "" {
		client.reply(env.ID, nil, "已被主人强制白给，无法撤回")
		return
	}
	delete(room.Choices, seat)
	s.roomNotice(room, playerShortName(player)+" 撤回了出拳。")
	s.broadcastRoom(room.ID, false)
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

type turnBasedUndoHandler struct {
	gameID   types.GameID
	gameName string
	request  func(*RoomState, types.SeatKey) (bool, string)
	respond  func(*RoomState, types.SeatKey, bool) (bool, string)
}

func (s *Server) onTurnBasedUndoRequest(client *Client, env wsEnvelope, handler turnBasedUndoHandler) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != handler.gameID {
		client.reply(env.ID, nil, "当前房间不是"+handler.gameName)
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请悔棋")
		return
	}
	ok, errMsg := handler.request(room, seat)
	if !ok {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, playerShortName(player)+" 申请悔棋，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onTurnBasedUndoRespond(client *Client, env wsEnvelope, handler turnBasedUndoHandler) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != handler.gameID {
		client.reply(env.ID, nil, "当前房间不是"+handler.gameName)
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理悔棋请求")
		return
	}
	accept := p.Accept != nil && *p.Accept
	ok, errMsg := handler.respond(room, seat, accept)
	if !ok {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onOthelloUndoRequest(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRequest(client, env, turnBasedUndoHandler{
		gameID: types.GameOthello, gameName: "黑白棋", request: s.requestOthelloUndo, respond: s.respondOthelloUndo,
	})
}

func (s *Server) onOthelloUndoRespond(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRespond(client, env, turnBasedUndoHandler{
		gameID: types.GameOthello, gameName: "黑白棋", request: s.requestOthelloUndo, respond: s.respondOthelloUndo,
	})
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
	if room.Othello.UndoRequest != nil {
		client.reply(env.ID, nil, "悔棋请求处理完成前不能申请认输")
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
		Label:              fmt.Sprintf("%s认输，%s胜利", loserName, winnerName),
		HistoryNote:        "认输",
		Notice:             fmt.Sprintf("%s 同意 %s 认输，本局结束。", winnerName, loserName),
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
		Label:              fmt.Sprintf("%s逃跑，%s胜利", loserName, winnerName),
		HistoryNote:        "逃跑",
		Notice:             loserName + " 选择逃跑，本局立即判负。",
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

func (s *Server) onGomokuReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
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
	s.roomNotice(room, playerShortName(player)+" 已准备五子棋。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleGomokuReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuGiveaway(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以选择白给")
		return
	}
	ok2, errMsg := s.chooseGomokuGiveaway(room, seat)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, room.ResultText)
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuMove(client *Client, env wsEnvelope) {
	var p struct {
		Row int `json:"row"`
		Col int `json:"col"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
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
	ok2, errMsg := s.applyGomokuMove(room, seat, p.Row, p.Col)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuUndoRequest(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请悔棋")
		return
	}
	ok2, errMsg := s.requestGomokuUndo(room, seat)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, playerShortName(player)+" 申请悔棋，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuUndoRespond(client *Client, env wsEnvelope) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理悔棋请求")
		return
	}
	accept := p.Accept != nil && *p.Accept
	ok2, errMsg := s.respondGomokuUndo(room, seat, accept)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuResignRequest(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请认输")
		return
	}
	ok2, errMsg := s.requestGomokuResign(room, seat)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, playerShortName(player)+" 申请认输，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuResignRespond(client *Client, env wsEnvelope) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理认输请求")
		return
	}
	accept := p.Accept != nil && *p.Accept
	ok2, errMsg := s.respondGomokuResign(room, seat, accept)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onGomokuRestart(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameGomoku {
		client.reply(env.ID, nil, "当前房间不是五子棋")
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
	s.resetGomokuRoom(room)
	s.roomNotice(room, playerShortName(player)+" 发起五子棋再来一局，请双方准备。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onJungleReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameJungle {
		client.reply(env.ID, nil, "当前房间不是斗兽棋")
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
	s.roomNotice(room, playerShortName(player)+" 已准备斗兽棋。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleJungleReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onJungleMove(client *Client, env wsEnvelope) {
	var p struct {
		FromRow int `json:"fromRow"`
		FromCol int `json:"fromCol"`
		ToRow   int `json:"toRow"`
		ToCol   int `json:"toCol"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameJungle {
		client.reply(env.ID, nil, "当前房间不是斗兽棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以走子")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能开始")
		return
	}
	ok2, errMsg := s.applyJungleMove(room, seat, p.FromRow, p.FromCol, p.ToRow, p.ToCol)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onJungleUndoRequest(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRequest(client, env, turnBasedUndoHandler{
		gameID: types.GameJungle, gameName: "斗兽棋", request: s.requestJungleUndo, respond: s.respondJungleUndo,
	})
}

func (s *Server) onJungleUndoRespond(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRespond(client, env, turnBasedUndoHandler{
		gameID: types.GameJungle, gameName: "斗兽棋", request: s.requestJungleUndo, respond: s.respondJungleUndo,
	})
}

func (s *Server) onJungleResignRequest(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameJungle {
		client.reply(env.ID, nil, "当前房间不是斗兽棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请认输")
		return
	}
	ok2, errMsg := s.requestJungleResign(room, seat)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, playerShortName(player)+" 申请认输，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onJungleResignRespond(client *Client, env wsEnvelope) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameJungle {
		client.reply(env.ID, nil, "当前房间不是斗兽棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理认输请求")
		return
	}
	accept := p.Accept != nil && *p.Accept
	ok2, errMsg := s.respondJungleResign(room, seat, accept)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onJungleRestart(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameJungle {
		client.reply(env.ID, nil, "当前房间不是斗兽棋")
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
	s.resetJungleRoom(room)
	s.roomNotice(room, playerShortName(player)+" 发起斗兽棋再来一局，请双方准备。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onChessReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameChess {
		client.reply(env.ID, nil, "当前房间不是国际象棋")
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
	s.roomNotice(room, playerShortName(player)+" 已准备国际象棋。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleChessReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onChessMove(client *Client, env wsEnvelope) {
	var p struct {
		FromRow int    `json:"fromRow"`
		FromCol int    `json:"fromCol"`
		ToRow   int    `json:"toRow"`
		ToCol   int    `json:"toCol"`
		Promote string `json:"promote"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameChess {
		client.reply(env.ID, nil, "当前房间不是国际象棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以走子")
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		client.reply(env.ID, nil, "需要双方都坐下才能开始")
		return
	}
	ok2, errMsg := s.applyChessMove(room, seat, p.FromRow, p.FromCol, p.ToRow, p.ToCol, p.Promote)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onChessUndoRequest(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRequest(client, env, turnBasedUndoHandler{
		gameID: types.GameChess, gameName: "国际象棋", request: s.requestChessUndo, respond: s.respondChessUndo,
	})
}

func (s *Server) onChessUndoRespond(client *Client, env wsEnvelope) {
	s.onTurnBasedUndoRespond(client, env, turnBasedUndoHandler{
		gameID: types.GameChess, gameName: "国际象棋", request: s.requestChessUndo, respond: s.respondChessUndo,
	})
}

func (s *Server) onChessResignRequest(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameChess {
		client.reply(env.ID, nil, "当前房间不是国际象棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以申请认输")
		return
	}
	ok2, errMsg := s.requestChessResign(room, seat)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	s.roomNotice(room, playerShortName(player)+" 申请认输，等待对方确认。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onChessResignRespond(client *Client, env wsEnvelope) {
	var p struct {
		Accept *bool `json:"accept"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameChess {
		client.reply(env.ID, nil, "当前房间不是国际象棋")
		return
	}
	seat, ok := s.seatOf(room, player.ID)
	if !ok {
		client.reply(env.ID, nil, "只有战斗席玩家可以处理认输请求")
		return
	}
	accept := p.Accept != nil && *p.Accept
	ok2, errMsg := s.respondChessResign(room, seat, accept)
	if !ok2 {
		client.reply(env.ID, nil, errMsg)
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onChessRestart(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameChess {
		client.reply(env.ID, nil, "当前房间不是国际象棋")
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
	s.resetChessRoom(room)
	s.roomNotice(room, playerShortName(player)+" 发起国际象棋再来一局，请双方准备。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentSubmit(client *Client, env wsEnvelope) {
	var p struct {
		Text     string `json:"text"`
		ImageURL string `json:"imageUrl"`
	}
	if err := decodeD(env, &p); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
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
	var taskText, taskEventID string
	if task := latestPunishmentTask(room, player.ID); task != nil {
		taskText = task.TaskText
		taskEventID = task.EventID
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
	// 无人类对手 / 房间关闭确认 时系统自动通过。
	approvedBySystem := s.punishmentReviewer(room, player.ID) == nil || !room.Settings.RequireOpponentConfirm
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
	// 证明提交后立即推送（不走 debounce），胜方尽快看到图片/文字。
	// 用 finishPunishmentIfComplete 而不是直接 resetForNextRound：棋类「每子惩罚」需要在惩罚完成后
	// 恢复原对局（resumeAfterMidGamePunishment）而不是强行开新局。
	if !s.finishPunishmentIfComplete(room) {
		s.broadcastRoomImmediate(room.ID, false)
	}
	imageFile := ""
	if cleanImageURL != "" {
		imageFile = filepath.Base(cleanImageURL)
	}
	if s.eventDB != nil && taskEventID != "" {
		if err := s.eventDB.updatePunishmentProof(taskEventID, submittedAt, cleanProofText, imageFile, status); err != nil {
			s.errorLog("punishment_event_update_failed", err.Error())
		}
	}
}

func (s *Server) onPunishmentAssignTask(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID string `json:"playerId"`
		TaskText string `json:"taskText"`
	}
	if err := decodeD(env, &p); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	p.PlayerID = strings.TrimSpace(p.PlayerID)
	if p.PlayerID == "" || len([]rune(p.PlayerID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "玩家 ID 无效")
		return
	}
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
	if !s.canReviewPlayer(room, player.ID, p.PlayerID) {
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
	// 玩家临时任务：直接使用原文，不替换 {winner}/{loser}（占位符仅用于系统任务配置）
	s.updatePunishmentTask(room, p.PlayerID, cleanTask, player)
	if s.eventDB != nil {
		if updated := latestPunishmentTask(room, p.PlayerID); updated != nil {
			eventID := randomID()
			if err := s.eventDB.insertPunishmentTask(eventID, nowMs(), room.ID, player.ID, playerShortName(player), p.PlayerID, updated.PlayerName, cleanTask, punishmentEventMeta{}); err != nil {
				s.errorLog("punishment_event_insert_failed", err.Error())
			} else {
				updated.EventID = eventID
			}
		}
	}
	// EventID 成功落库后再广播，保证客户端第一次看见任务时评价入口已经可用。
	s.broadcastRoomImmediate(room.ID, false)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentReview(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID     string `json:"playerId"`
		Action       string `json:"action"`
		RedoTaskText string `json:"redoTaskText"`
	}
	if err := decodeD(env, &p); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	p.PlayerID = strings.TrimSpace(p.PlayerID)
	p.Action = strings.TrimSpace(p.Action)
	if p.PlayerID == "" || len([]rune(p.PlayerID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "玩家 ID 无效")
		return
	}
	if p.Action != "approve" && p.Action != "reject" && p.Action != "forgive" {
		client.reply(env.ID, nil, "审核动作无效")
		return
	}
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if player.ID == p.PlayerID {
		client.reply(env.ID, nil, "不能审核自己的证明")
		return
	}
	if !s.canReviewPlayer(room, player.ID, p.PlayerID) {
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
	// proof 只有处于 pending（待审核）时才能被审核一次；已经 rejected/approved 的证明
	// 必须等玩家重新提交（onPunishmentSubmit 会生成一条全新的 pending proof）才能再次审核。
	// 否则同一条已驳回的证明可以被反复调用 reject 无限次，RejectCount 无限递增导致扣分失控。
	if proof.Status != "pending" {
		client.reply(env.ID, nil, "这条证明已经审核过了")
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
		oldEventID := ""
		if oldTask := latestPunishmentTask(room, p.PlayerID); oldTask != nil {
			oldEventID = oldTask.EventID
		}
		s.updatePunishmentTask(room, p.PlayerID, cleanTask, nil)
		if s.eventDB != nil && oldEventID != "" {
			newTask := latestPunishmentTask(room, p.PlayerID)
			newID := randomID()
			if newTask != nil {
				// 旧事件已被驳回；若新事件落库失败，不能继续把旧 EventID 暴露成当前任务。
				newTask.EventID = ""
			}
			targetName := ""
			if newTask != nil {
				targetName = newTask.PlayerName
			}
			if err := s.eventDB.redoPunishmentTask(oldEventID, newID, reviewedAt, room.ID, player.ID, playerShortName(player), p.PlayerID, targetName, cleanTask, punishmentEventMeta{}); err != nil {
				s.errorLog("punishment_event_insert_failed", err.Error())
			} else if newTask != nil {
				newTask.EventID = newID
			}
		}
		s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
			Status: "rejected", ReviewedBy: player.ID, ReviewedAt: &reviewedAt,
			RejectReason: "需要重做", RedoTaskText: cleanTask,
		})
		s.applyProofRejectionPenalty(room, p.PlayerID)
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
	if s.eventDB != nil {
		if task := latestPunishmentTask(room, p.PlayerID); task != nil && task.EventID != "" {
			if err := s.eventDB.updatePunishmentStatus(task.EventID, "approved"); err != nil {
				s.errorLog("punishment_event_update_failed", err.Error())
			}
		}
	}
	s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
		Status: "approved", ReviewedBy: player.ID, ReviewedAt: &reviewedAt, RejectReason: reviewMessage,
	})
	// 用 finishPunishmentIfComplete 而不是直接 resetForNextRound：棋类「每子惩罚」需要恢复原对局而不是开新局。
	if !s.finishPunishmentIfComplete(room) {
		s.broadcastRoom(room.ID, false)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onPunishmentConfirm(client *Client, env wsEnvelope) {
	var p struct {
		PlayerID string `json:"playerId"`
	}
	if err := decodeD(env, &p); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	p.PlayerID = strings.TrimSpace(p.PlayerID)
	if p.PlayerID == "" || len([]rune(p.PlayerID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "玩家 ID 无效")
		return
	}
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if player.ID == p.PlayerID {
		client.reply(env.ID, nil, "不能审核自己的证明")
		return
	}
	if !s.canReviewPlayer(room, player.ID, p.PlayerID) {
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
	if proof.Status != "pending" {
		client.reply(env.ID, nil, "这条证明已经审核过了")
		return
	}
	reviewedAt := nowMs()
	proof.Status = "approved"
	proof.ConfirmedBy = player.ID
	proof.ReviewedBy = player.ID
	proof.ReviewedAt = &reviewedAt
	if s.eventDB != nil {
		if task := latestPunishmentTask(room, p.PlayerID); task != nil && task.EventID != "" {
			if err := s.eventDB.updatePunishmentStatus(task.EventID, "approved"); err != nil {
				s.errorLog("punishment_event_update_failed", err.Error())
			}
		}
	}
	s.updateProofInLatestHistory(room, p.PlayerID, types.HistoryProof{
		Status: "approved", ReviewedBy: player.ID, ReviewedAt: &reviewedAt,
	})
	// 用 finishPunishmentIfComplete 而不是直接 resetForNextRound：棋类「每子惩罚」需要恢复原对局而不是开新局。
	if !s.finishPunishmentIfComplete(room) {
		s.broadcastRoom(room.ID, false)
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onChatSend(client *Client, env wsEnvelope) {
	var p struct {
		RoomID   string   `json:"roomId"`
		Text     string   `json:"text"`
		Mentions []string `json:"mentions"`
	}
	_ = decodeD(env, &p)
	player, ok := s.requirePlayerInGame(client, env)
	if !ok {
		return
	}
	// 大厅聊天（roomId==""）允许更长文本，保留原留言板体验；房间聊天 300。
	maxLen := 300
	if p.RoomID == "" {
		maxLen = 500
	}
	cleanMessageText := cleanText(p.Text, maxLen)
	if cleanMessageText == "" {
		client.reply(env.ID, nil, "请输入聊天内容")
		return
	}
	s.refreshNameWarState(player, nowMs())
	s.refreshPlayerSnapshots(player)
	var room *RoomState
	if p.RoomID != "" {
		room = s.rooms[p.RoomID]
		if room == nil || player.RoomID != p.RoomID {
			client.reply(env.ID, nil, "你不在这个房间里")
			return
		}
	}
	mentions := normalizeMentions(p.Mentions)
	if room != nil {
		// 房间聊天的 @ 只对同房间内的人有意义；否则任何人都能把陌生玩家 ID 塞进 mentions
		// 触发一条与该房间毫无关系的推送通知——大厅聊天没有"房间"概念，不受此限制。
		filtered := mentions[:0]
		for _, id := range mentions {
			if s.roomHasPlayer(room, id) {
				filtered = append(filtered, id)
			}
		}
		mentions = filtered
	}
	message := types.ChatMessage{
		ID: randomID(), RoomID: p.RoomID, PlayerID: player.ID,
		Author: player.DisplayName, Text: cleanMessageText, At: nowMs(),
		Mentions: mentions,
	}
	if room != nil {
		message.AuthorRole = s.roomRole(room, player.ID)
	}
	// 持久化 + 实时推送（房间/大厅由 RoomID 区分）。
	s.deliverChat(message, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
	for _, mentionedID := range message.Mentions {
		mentioned := s.players[mentionedID]
		if mentioned == nil || !shouldPush(mentioned, mentioned.PushMentionEnabled) {
			continue
		}
		// 同一发送者对同一被 @ 玩家的推送节流：避免靠反复群发消息对固定目标刷屏骚扰
		// （聊天消息本身仍正常送达，这里只限制会触达设备通知栏的 Web Push）。
		if !s.checkRateLimit("mention_push:"+player.ID+":"+mentionedID, RateLimitOptions{Limit: 1, WindowMs: 600_000, CooldownMs: 600_000}) {
			continue
		}
		s.sendPush(mentioned, "有人 @ 了你", fmt.Sprintf("%s：%s", player.DisplayName, cleanMessageText), "mention")
	}
}

// normalizeMentions 去重并限制被 @ 玩家 id 数量上限，防止刷屏 / 过大 payload。
func normalizeMentions(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	const maxMentions = 20
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
		if len(out) >= maxMentions {
			break
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// onChatLoad 返回某聊天频道最近一页（默认 100 条，升序）+ 是否还有更早历史。
func (s *Server) onChatLoad(client *Client, env wsEnvelope) {
	var p struct {
		RoomID string `json:"roomId"`
	}
	_ = decodeD(env, &p)
	if p.RoomID != "" {
		// 房间聊天：必须在该房间内才能拉取
		player := s.getPlayerByClient(client)
		if player == nil || player.RoomID != p.RoomID {
			client.reply(env.ID, nil, "你不在这个房间里")
			return
		}
	}
	messages, hasMore, err := s.loadChatPage(p.RoomID, 0, chatPageSize)
	if err != nil {
		client.reply(env.ID, nil, "聊天记录加载失败")
		return
	}
	client.reply(env.ID, map[string]any{
		"roomId": p.RoomID, "messages": messages, "hasMore": hasMore,
		"authors": s.chatAuthorsFor(messages),
	}, "")
}

// onChatLoadOlder 瀑布流：返回 seq < beforeSeq 的更早一页 + hasMore。
func (s *Server) onChatLoadOlder(client *Client, env wsEnvelope) {
	var p struct {
		RoomID    string `json:"roomId"`
		BeforeSeq int64  `json:"beforeSeq"`
	}
	_ = decodeD(env, &p)
	if p.RoomID != "" {
		player := s.getPlayerByClient(client)
		if player == nil || player.RoomID != p.RoomID {
			client.reply(env.ID, nil, "你不在这个房间里")
			return
		}
	}
	if p.BeforeSeq <= 0 {
		client.reply(env.ID, nil, "游标无效")
		return
	}
	messages, hasMore, err := s.loadChatPage(p.RoomID, p.BeforeSeq, chatLoadMorePageSize)
	if err != nil {
		client.reply(env.ID, nil, "聊天记录加载失败")
		return
	}
	client.reply(env.ID, map[string]any{
		"roomId": p.RoomID, "messages": messages, "hasMore": hasMore,
		"authors": s.chatAuthorsFor(messages),
	}, "")
}

func (s *Server) loadChatPage(roomID string, beforeSeq int64, limit int) ([]types.ChatMessage, bool, error) {
	if s.chatDB == nil {
		return []types.ChatMessage{}, false, nil
	}
	return s.chatDB.older(roomID, beforeSeq, limit)
}

// chatAuthorsFor 按消息里的 playerId 从内存 s.players 取**当前**公开资料（含离线玩家）。
// 持久化玩家启动时全部 load 进内存，不依赖"是否在大厅/房间在线名单"。
// 返回 map 便于前端按 id 索引；已删档/不存在的 id 不会出现在 map 里，前端退回 message.author 文本。
func (s *Server) chatAuthorsFor(messages []types.ChatMessage) map[string]types.PublicPlayer {
	out := make(map[string]types.PublicPlayer)
	for _, m := range messages {
		if m.PlayerID == "" {
			continue
		}
		if _, ok := out[m.PlayerID]; ok {
			continue
		}
		if p := s.players[m.PlayerID]; p != nil {
			out[m.PlayerID] = s.publicPlayer(p)
		}
	}
	return out
}

func (s *Server) onAdminAction(client *Client, env wsEnvelope) {
	var p struct {
		Action             string            `json:"action"`
		RoomID             string            `json:"roomId"`
		PlayerID           string            `json:"playerId"`
		Name               string            `json:"name"`
		RankedPoints       *float64          `json:"rankedPoints"`
		Title              *string           `json:"title"`
		GenderID           *string           `json:"genderId"`
		GiveawayValueInput *string           `json:"giveawayValueInput"`
		Message            string            `json:"message"`
		DurationSeconds    *float64          `json:"durationSeconds"`
		Result             types.RoundResult `json:"result"`
		Refs               []chatMessageRef  `json:"refs"`
	}
	if err := decodeD(env, &p); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	admin := s.getPlayerByClient(client)
	_, isAdminSocket := s.adminClientIDs[client.id]
	// 只认本 socket 的 admin:login 会话，不认粘滞的 player.IsAdmin。
	if !isAdminSocket {
		client.reply(env.ID, nil, "需要管理员权限")
		return
	}
	if strings.HasPrefix(p.Action, "contribution") || p.Action == "gendersGet" || p.Action == "gendersSave" {
		raw := map[string]any{}
		_ = decodeD(env, &raw)
		raw["adminId"] = "admin"
		if admin != nil {
			raw["adminId"] = admin.ID
		}
		out, err := s.handleAdminContributionAction(p.Action, raw)
		if err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
		if out != nil {
			client.reply(env.ID, out, "")
		} else {
			client.reply(env.ID, map[string]any{"ok": true}, "")
		}
		return
	}
	// 任务池/系列任务不再支持管理员在后台直接增删改：内容一律走玩家投稿 + 管理员审批
	// （见 contribution* 系列 admin action），管理员只保留审批/批注/下架等审核类操作。
	// 聊天管理走软删除，不再有"一键清空大厅/房间聊天"——只能针对检索出的具体消息
	// （见 admin:chatSearch）单条/批量删除或恢复。删除即时持久化（面向普通玩家的
	// recent/older 从此过滤掉这些行，新老访客都读不到）并推送 chat:deleted 通知在线
	// 客户端摘除本地视图；恢复只影响持久化状态，不主动推回（客户端本就已看不到该
	// 消息，管理员刷新检索列表即可核实）。这两个分支各自 reply 后直接 return，跳过
	// 函数末尾统一的房间广播——聊天删除与房间/玩家结构状态无关。
	if p.Action == "chatSoftDelete" || p.Action == "chatRestore" {
		refs := normalizeChatMessageRefs(p.Refs)
		if len(refs) == 0 {
			client.reply(env.ID, nil, "未选择聊天消息")
			return
		}
		if len(refs) > chatBulkMaxRefs {
			client.reply(env.ID, nil, fmt.Sprintf("单次最多操作 %d 条聊天消息", chatBulkMaxRefs))
			return
		}
		if s.chatDB == nil {
			client.reply(env.ID, nil, "聊天存储不可用")
			return
		}
		deleted := p.Action == "chatSoftDelete"
		n, err := s.setAdminChatDeletedWithoutServerLock(refs, deleted, nowMs())
		if err != nil {
			s.errorLog("chat_set_deleted_failed", err.Error())
			client.reply(env.ID, nil, "操作失败")
			return
		}
		if deleted {
			s.broadcastChatDeleted(refs)
		}
		client.reply(env.ID, map[string]any{"ok": true, "affected": n}, "")
		return
	}
	roomDeleted := false
	changedPlayerRoomID := ""
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
				s.clientLeaveRoom(c, p.RoomID)
			}
			s.clearOthelloSettlementTimer(p.RoomID)
			s.clearTicTacToeGiveawayTimer(p.RoomID)
			s.clearGomokuUndoTimer(p.RoomID)
			s.clearTurnBasedUndoTimer(p.RoomID)
			s.clearLiarsDiceStartTimer(p.RoomID)
			s.clearTurnBasedClockTimer(p.RoomID)
			s.clearRoomBroadcastTimer(p.RoomID)
			s.dropSyncChannel(channelRoom(p.RoomID))
			delete(s.rooms, p.RoomID)
			s.dropProofImageRoomEntries(p.RoomID)
			s.recordSeriesRunProgressOnClose(room)
			if s.eventDB != nil {
				closerID, closerName := "", "管理员"
				if admin != nil {
					closerID, closerName = admin.ID, playerShortName(admin)
				}
				if err := s.eventDB.insertRoomEvent(roomEventInput{
					At: nowMs(), RoomID: room.ID, RoomName: room.Settings.Name, GameID: string(room.Settings.GameID),
					UserID: closerID, UserName: closerName, Action: "close", Reason: "admin_close", IP: client.ipAddress,
				}); err != nil {
					s.errorLog("room_event_close_failed", err.Error())
				}
			}
			roomDeleted = true
		}
	}
	if p.Action == "forceNext" && p.RoomID != "" {
		if room := s.rooms[p.RoomID]; room != nil {
			s.resetForNextRound(room)
		}
	}
	if p.Action == "forceSeatOutcome" && p.RoomID != "" {
		room := s.rooms[p.RoomID]
		if room == nil {
			client.reply(env.ID, nil, "房间不存在")
			return
		}
		if p.Result != types.ResultA && p.Result != types.ResultB && p.Result != types.ResultDraw {
			client.reply(env.ID, nil, "请选择 A 方胜、B 方胜或平局")
			return
		}
		switch room.Settings.GameID {
		case types.GameOthello:
			forcedResult := p.Result
			if p.Result != types.ResultDraw && room.Othello != nil {
				if p.Result == types.ResultA {
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
		case types.GameTicTacToe:
			if room.TicTacToe == nil || room.TicTacToe.Ended {
				client.reply(env.ID, nil, "对局尚未开始或已经结束")
				return
			}
			s.finishTicTacToeGame(room, p.Result, nil, "管理员判定")
		case types.GameGomoku:
			if room.Gomoku == nil || room.Gomoku.Ended {
				client.reply(env.ID, nil, "对局尚未开始或已经结束")
				return
			}
			s.finishGomokuGame(room, p.Result, nil, "管理员判定")
		case types.GameJungle:
			if room.Jungle == nil || room.Jungle.Ended {
				client.reply(env.ID, nil, "对局尚未开始或已经结束")
				return
			}
			s.finishJungleGame(room, p.Result, "管理员判定")
		case types.GameChess:
			if room.Chess == nil || room.Chess.Ended {
				client.reply(env.ID, nil, "对局尚未开始或已经结束")
				return
			}
			s.finishChessGame(room, p.Result, "管理员判定")
		case types.GameRPS:
			ok, errMsg := s.forceEndRpsRound(room, p.Result)
			if !ok {
				client.reply(env.ID, nil, errMsg)
				return
			}
		default:
			client.reply(env.ID, nil, "当前游戏不支持强制判定胜负")
			return
		}
	}
	if p.Action == "altAccounts" && p.PlayerID != "" {
		// "小号查询"：BFS 传递展开同设备关联账号，见 altAccountCandidates 顶部注释
		// （含防死循环、遍历上限说明）。
		if s.players[p.PlayerID] == nil {
			client.reply(env.ID, nil, "玩家不存在")
			return
		}
		accounts, truncated, err := s.altAccountCandidatesForAdmin(p.PlayerID)
		if _, stillAdmin := s.adminClientIDs[client.id]; !stillAdmin {
			client.reply(env.ID, nil, "需要管理员权限")
			return
		}
		if err != nil {
			if errors.Is(err, errAltAccountsStoreUnavailable) {
				client.reply(env.ID, nil, err.Error())
				return
			}
			s.errorLog("alt_accounts_query_failed", err.Error())
			client.reply(env.ID, nil, "查询失败")
			return
		}
		client.reply(env.ID, map[string]any{"players": accounts, "truncated": truncated}, "")
		return
	}
	if p.Action == "showClaimKey" && p.PlayerID != "" {
		// 管理员协助玩家找回账号：返回与个人资料「显示认领密钥」相同的 playerId.claimKey。
		// 只回给当前管理员 socket，不广播；非持久身份没有认领体系。后台改成单个"点击即显示"
		// 输入框后，每次调用都轮换一把新密钥（旧密钥立即作废），语义对齐 identity:refreshClaimKey。
		pl := s.players[p.PlayerID]
		if pl == nil {
			client.reply(env.ID, nil, "玩家不存在")
			return
		}
		if !pl.Persistent || pl.PlayerID == "" {
			client.reply(env.ID, nil, "该玩家身份不支持认领密钥")
			return
		}
		pl.ClaimKey = randomClaimKey()
		s.markPlayerDirty(pl)
		s.requestPersist("lazy")
		s.securityLog("admin_show_claim_key", map[string]any{"ip": client.ipAddress, "targetPlayerId": pl.ID})
		client.reply(env.ID, map[string]any{"claimKey": pl.ClaimKey, "playerId": pl.PlayerID}, "")
		return
	}
	if p.Action == "kick" && p.PlayerID != "" {
		if pl := s.players[p.PlayerID]; pl != nil {
			socketID := pl.SocketID
			s.leaveRoom(pl, LeaveAdminKick)
			s.clearDisconnectHold(pl)
			if socketID != "" {
				s.emitToClient(socketID, "player:kicked", map[string]any{})
			}
			if pl.CurrentSID != "" && s.sidToPlayerID[pl.CurrentSID] == pl.ID {
				delete(s.sidToPlayerID, pl.CurrentSID)
			}
			if pl.Persistent {
				// 持久玩家只下线、保留存档；serializePlayers 只写内存现存玩家，
				// 直接 delete 会把该玩家积分/战绩从 SQLite 里永久抹掉。
				now := nowMs()
				pl.Connected = false
				pl.SocketID = ""
				pl.DisconnectedAt = &now
				pl.DisconnectExpiresAt = nil
				pl.LastSeenAt = now
				s.refreshPlayerSnapshots(pl)
				s.markPlayerDirty(pl)
				s.requestPersist("lazy")
			} else {
				delete(s.players, pl.ID)
				delete(s.tokenToPlayer, pl.Token)
				if pl.PlayerID != "" {
					delete(s.playerIdToID, pl.PlayerID)
				}
			}
			// 认主/认宠申请要求双方在线，管理员踢人视同离线，作废其相关申请。
			s.clearOfflinePetBondRequests(pl.ID)
			// 宠物乐园候选/关系图依赖在线状态，即便没有待处理申请，该玩家的公开关系链也要摘除。
			s.notifyAllOnlinePetBondStates()
		}
	}
	if p.Action == "editPlayer" && p.PlayerID != "" {
		pl := s.players[p.PlayerID]
		if pl == nil {
			client.reply(env.ID, nil, "玩家不存在")
			return
		}
		cleanName := cleanText(p.Name, 12)
		if len([]rune(cleanName)) < 2 {
			client.reply(env.ID, nil, "名字至少需要 2 个字")
			return
		}
		pl.Name = cleanName
		pl.NameWarOriginalName = cleanName
		// RankedPoints 为 nil 表示管理员没有改动这一栏——前端现在展示/编辑的是真实存储分
		// （sortRankedPoints），不传就保持原值。
		if p.RankedPoints != nil {
			s.setRankedPointsByAdmin(pl, int(*p.RankedPoints))
		}
		// Title 为 nil 表示管理员没有改动这一栏，保持原值不动。非 nil 时：非空文本 → 管理员
		// 自定义，之后不随排位分档位变化自动改写（见 syncTitleForRankSegment 里的 TitleCustom
		// 判断）；清空则视为主动清除自定义，立即按当前排位分重新计算回自动称号。
		if p.Title != nil {
			cleanTitle := cleanText(*p.Title, 18)
			if cleanTitle != "" {
				pl.Stats.Title = cleanTitle
				pl.Stats.TitleCustom = true
			} else if pl.Stats.TitleCustom {
				pl.Stats.TitleCustom = false
				s.syncTitleForRankSegment(pl, true)
			}
		}
		// GenderID 为 nil 表示管理员没有改动这一栏；非 nil 时按查表法解析（genderId 命中预设
		// 才合法，阵营由预设的 factionId 决定，不再单独提交）。
		if p.GenderID != nil {
			if ok, reason := s.validGenderSubmission(*p.GenderID); !ok {
				client.reply(env.ID, nil, reason)
				return
			}
			s.applyGender(pl, *p.GenderID)
		}
		// GiveawayValueInput 为 nil 表示管理员没有改动这一栏；否则按数字语义解析（0-100，1 位小数）：
		// 大于 0 直接设为白给值并开启；0 或任何解析失败/超范围的内容都视为清空白给值并关闭白给
		// （不再接受"关闭"这个文本哨兵值，前端已经改成纯数字输入框）。
		if p.GiveawayValueInput != nil {
			trimmed := strings.TrimSpace(*p.GiveawayValueInput)
			if v, err := strconv.ParseFloat(trimmed, 64); err == nil && v > 0 && v <= 100 {
				pl.GiveawayEnabled = boolPtr(true)
				pl.GiveawayValue = floatPtr(clampGiveawayValue(v))
			} else {
				pl.GiveawayEnabled = boolPtr(false)
				pl.GiveawayValue = floatPtr(0)
				pl.GiveawayBoardText = ""
				pl.GiveawayBoardSubmittedAt = nil
				pl.GiveawayBoardExpiresAt = nil
			}
		}
		pl.DisplayName = s.formatDisplayName(pl)
		s.refreshPlayerSnapshots(pl)
		changedPlayerRoomID = pl.RoomID
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
	if roomDeleted {
		// 房间删除属结构性变化，立即全量推大厅，避免他人列表残留幽灵房间。
		s.forceBroadcastLobby()
	} else {
		s.broadcastLobby()
	}
	if p.RoomID != "" && !roomDeleted {
		s.broadcastRoom(p.RoomID, false)
	}
	if changedPlayerRoomID != "" && changedPlayerRoomID != p.RoomID {
		s.broadcastRoom(changedPlayerRoomID, false)
	}
}
