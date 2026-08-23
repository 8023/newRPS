package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// 大话骰：刻意不碰 Seats/SeatKey（那是 RPS/黑白棋/井字棋共用的两人定长模型），
// 参战名单、叫点顺序、结算全部走 RoomState.LiarsDice / LiarsDiceHands 这两个专属字段。
// 私有骰子只通过 emitToClient 单播给玩家自己，保密性来自"只发给这一个 socket"，不需要加密。

const liarsDiceDicePerPlayer = 5
const liarsDiceReadyStartDelay = 5 * time.Second

func indexOfString(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

func removeString(list []string, s string) []string {
	out := list[:0]
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

func liarsDiceNextTurn(order []string, current string) string {
	idx := indexOfString(order, current)
	if idx < 0 || len(order) == 0 {
		return ""
	}
	return order[(idx+1)%len(order)]
}

// liarsDicePredecessor：当前叫点顺序里的"上家"（开局会随机重排 ParticipantIDs），
// 掉线判负规则用；与"谁最后叫过点"无关，任何时刻都有明确定义（只要还在名单里）。
func liarsDicePredecessor(order []string, playerID string) string {
	idx := indexOfString(order, playerID)
	if idx < 0 || len(order) < 2 {
		return ""
	}
	return order[(idx-1+len(order))%len(order)]
}

func liarsDiceValidRaise(prev *types.LiarsDiceBid, count, face int) bool {
	if face < 1 || face > 6 || count < 1 {
		return false
	}
	if prev == nil {
		return true
	}
	if count > prev.Count {
		return true
	}
	if count == prev.Count && face > prev.Face {
		return true
	}
	return false
}

func (s *Server) clearLiarsDiceStartTimer(roomID string) {
	if t := s.liarsDiceStartTimers[roomID]; t != nil {
		t.Stop()
		delete(s.liarsDiceStartTimers, roomID)
	}
}

// scheduleLiarsDiceReadyStart：参战名单或准备状态一变就重置这个 5 秒防抖计时器；
// 到点时用"当下"的房间状态重新校验一遍，天然保证"名单 5 秒无变动"（照抄
// othelloSettlementTimers/ticTacToeGiveawayTimers 的每房间单计时器模式）。
func (s *Server) scheduleLiarsDiceReadyStart(room *RoomState) {
	s.clearLiarsDiceStartTimer(room.ID)
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		return
	}
	if room.Phase != types.PhaseReady {
		return
	}
	minPlayers := room.LiarsDice.MinPlayers
	if minPlayers < 2 {
		minPlayers = 2
	}
	participants := room.LiarsDice.ParticipantIDs
	if len(participants) < minPlayers || !allLiarsDiceReady(room.LiarsDice) {
		return
	}
	roomID := room.ID
	timer := timeAfterFunc(liarsDiceReadyStartDelay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.liarsDiceStartTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameLiarsDice || current.LiarsDice == nil {
			return
		}
		if current.Phase != types.PhaseReady {
			return
		}
		mp := current.LiarsDice.MinPlayers
		if mp < 2 {
			mp = 2
		}
		if len(current.LiarsDice.ParticipantIDs) < mp || !allLiarsDiceReady(current.LiarsDice) {
			return
		}
		s.startLiarsDiceRound(current)
		s.roomNotice(current, "大话骰开局，已随机叫点顺序并为每位参战玩家摇好骰子。")
		s.broadcastRoom(current.ID, true)
	})
	s.liarsDiceStartTimers[room.ID] = timer
}

func allLiarsDiceReady(state *types.LiarsDiceState) bool {
	if len(state.ParticipantIDs) == 0 {
		return false
	}
	for _, id := range state.ParticipantIDs {
		if !containsString(state.ReadyPlayerIDs, id) {
			return false
		}
	}
	return true
}

// shuffleStrings 返回 list 的随机排列副本（不改动入参）。
func shuffleStrings(list []string) []string {
	out := append([]string{}, list...)
	rand.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}

// startLiarsDiceRound：重新摇骰（每局独立，不延续上局骰子数量/不淘汰）；
// 开局随机重排参战名单作为叫点顺序（防止固定上下家 / 按入席顺序操纵），并随机选首个叫点者。
func (s *Server) startLiarsDiceRound(room *RoomState) {
	if room.LiarsDice == nil || len(room.LiarsDice.ParticipantIDs) < 2 {
		return
	}
	// 随机叫点顺序，并写回 ParticipantIDs，前后端上下家关系一致。
	participants := shuffleStrings(room.LiarsDice.ParticipantIDs)
	room.LiarsDice.ParticipantIDs = participants
	room.LiarsDiceHands = map[string][]int{}
	diceCounts := map[string]int{}
	for _, id := range participants {
		dice := make([]int, liarsDiceDicePerPlayer)
		for i := range dice {
			dice[i] = rand.Intn(6) + 1
		}
		room.LiarsDiceHands[id] = dice
		diceCounts[id] = liarsDiceDicePerPlayer
	}
	first := participants[rand.Intn(len(participants))]
	room.LiarsDice.DiceCounts = diceCounts
	room.LiarsDice.CurrentBid = nil
	room.LiarsDice.BidHistory = []types.LiarsDiceBid{}
	room.LiarsDice.OnesWildDisabled = false
	room.LiarsDice.RoundNumber++
	room.LiarsDice.Ended = false
	room.LiarsDice.WinnerID = ""
	room.LiarsDice.LoserID = ""
	room.LiarsDice.RevealedHands = nil
	room.LiarsDice.ActualCount = 0
	room.LiarsDice.CurrentTurn = first
	room.LiarsDice.ReadyPlayerIDs = []string{}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.ResultText = ""
	room.Proofs = []types.PunishmentProof{}
	room.PunishedPlayerIDs = []string{}
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.LiarsDiceDisconnectForfeits = map[string]LiarsDiceDisconnectForfeit{}
	s.logGameStart(room)
	for _, id := range participants {
		if player := s.players[id]; player != nil {
			s.emitToClient(player.SocketID, "liarsdice:hand", map[string]any{
				"roomId": room.ID,
				"dice":   room.LiarsDiceHands[id],
			})
		}
	}
	if firstPlayer := s.players[first]; firstPlayer != nil && shouldPush(firstPlayer, firstPlayer.PushTurnEnabled) {
		s.sendPush(firstPlayer, "轮到你了", "大话骰新的一局开始了，轮到你叫点。", "turn-"+room.ID)
	}
}

// liarsDiceRoster：房间设置里 min/max 参战人数（含默认值/边界钳制）。
func liarsDiceRosterBounds(settings types.RoomSettings) (min, max int) {
	min, max = settings.LiarsDiceMinPlayers, settings.LiarsDiceMaxPlayers
	if max < 2 {
		max = 3
	}
	if max > 8 {
		max = 8
	}
	if min < 2 {
		min = 3
	}
	if min > max {
		min = max
	}
	return min, max
}

func (s *Server) onLiarsDiceJoinRoster(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if room.Phase == types.PhaseChoosing || room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "当前不能加入参战席")
		return
	}
	if text := s.rankedSeatRestrictionText(room, player); text != "" {
		client.reply(env.ID, nil, text)
		return
	}
	if containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		client.reply(env.ID, nil, "你已经在参战席了")
		return
	}
	_, max := liarsDiceRosterBounds(room.Settings)
	if len(room.LiarsDice.ParticipantIDs) >= max {
		client.reply(env.ID, nil, "参战席已满")
		return
	}
	// 参战席不从 SpectatorIDs 里挪走——房间快照里玩家资料（头像/称号）只在 Spectators
	// 这份 PublicPlayer 列表里有，ParticipantIDs 只是 id 字符串；留在 spectators 里
	// 前端才能按 id 查到完整资料。ParticipantIDs 单独标记"谁在参战"。
	room.LiarsDice.ParticipantIDs = append(room.LiarsDice.ParticipantIDs, player.ID)
	s.roomNotice(room, playerShortName(player)+" 加入了大话骰参战席。")
	s.scheduleLiarsDiceReadyStart(room)
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLiarsDiceLeaveRoster(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if room.Phase == types.PhaseChoosing || room.Phase == types.PhasePunishment {
		client.reply(env.ID, nil, "对局进行中不能离开参战席")
		return
	}
	if !containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		client.reply(env.ID, nil, "你不在参战席")
		return
	}
	s.leaveLiarsDiceRoster(room, player.ID)
	s.roomNotice(room, playerShortName(player)+" 离开了大话骰参战席。")
	s.scheduleLiarsDiceReadyStart(room)
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// leaveLiarsDiceRoster 只清角色状态（名单/已准备/私有骰子），不动 SpectatorIDs——
// 调用方决定离开后落到观战席还是彻底离开房间。
func (s *Server) leaveLiarsDiceRoster(room *RoomState, playerID string) {
	if room.LiarsDice == nil {
		return
	}
	room.LiarsDice.ParticipantIDs = removeString(room.LiarsDice.ParticipantIDs, playerID)
	room.LiarsDice.ReadyPlayerIDs = removeString(room.LiarsDice.ReadyPlayerIDs, playerID)
	delete(room.LiarsDiceHands, playerID)
	delete(room.LiarsDiceDisconnectForfeits, playerID)
}

func (s *Server) onLiarsDiceReady(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if !containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		client.reply(env.ID, nil, "只有参战席玩家可以准备")
		return
	}
	if room.Phase != types.PhaseReady {
		client.reply(env.ID, nil, "当前不能准备")
		return
	}
	if !containsString(room.LiarsDice.ReadyPlayerIDs, player.ID) {
		room.LiarsDice.ReadyPlayerIDs = append(room.LiarsDice.ReadyPlayerIDs, player.ID)
	}
	s.roomNotice(room, playerShortName(player)+" 已准备大话骰。")
	client.reply(env.ID, map[string]any{"ok": true}, "")
	s.scheduleLiarsDiceReadyStart(room)
	s.broadcastRoom(room.ID, true)
}

func (s *Server) onLiarsDiceBid(client *Client, env wsEnvelope) {
	var p struct {
		Count int `json:"count"`
		Face  int `json:"face"`
	}
	_ = decodeD(env, &p)
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if room.Phase != types.PhaseChoosing {
		client.reply(env.ID, nil, "现在还不能叫点")
		return
	}
	if room.LiarsDice.CurrentTurn != player.ID {
		client.reply(env.ID, nil, "还没轮到你叫点")
		return
	}
	if room.LiarsDice.CurrentBid == nil {
		minCount := len(room.LiarsDice.ParticipantIDs) + 1
		if p.Count < minCount || p.Face < 1 || p.Face > 6 {
			client.reply(env.ID, nil, fmt.Sprintf("第一个叫点至少要喊 %d 个", minCount))
			return
		}
	} else if !liarsDiceValidRaise(room.LiarsDice.CurrentBid, p.Count, p.Face) {
		client.reply(env.ID, nil, "叫点无效，必须比上家颗数更多，或颗数不变但点数更大")
		return
	}
	bid := types.LiarsDiceBid{PlayerID: player.ID, Count: p.Count, Face: p.Face, At: nowMs()}
	room.LiarsDice.CurrentBid = &bid
	room.LiarsDice.BidHistory = append(room.LiarsDice.BidHistory, bid)
	if p.Face == 1 {
		room.LiarsDice.OnesWildDisabled = true
	}
	next := liarsDiceNextTurn(room.LiarsDice.ParticipantIDs, player.ID)
	room.LiarsDice.CurrentTurn = next
	s.roomNotice(room, fmt.Sprintf("%s 叫点：%d 个 %d。", playerShortName(player), p.Count, p.Face))
	if nextPlayer := s.players[next]; nextPlayer != nil && shouldPush(nextPlayer, nextPlayer.PushTurnEnabled) {
		s.sendPush(nextPlayer, "轮到你了", "轮到你在大话骰中叫点或开牌了。", "turn-"+room.ID)
	}
	s.broadcastRoom(room.ID, false)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onLiarsDiceChallenge(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if room.Phase != types.PhaseChoosing || room.LiarsDice.CurrentBid == nil {
		client.reply(env.ID, nil, "现在还不能开牌")
		return
	}
	if !containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		client.reply(env.ID, nil, "只有参战玩家可以开牌")
		return
	}
	// 不能开自己刚叫的点：否则赢家=输家=自己，既无法给自己发布/审核惩罚任务，也会卡死退出房间的流程
	// （canLeaveRoom 里"受罚未完成不能离房"与 canReviewPlayer 里"不能审核自己"两条规则都会连带触发）。
	if room.LiarsDice.CurrentBid.PlayerID == player.ID {
		client.reply(env.ID, nil, "不能开自己叫的点，请等待其他玩家开牌")
		return
	}
	s.resolveLiarsDiceChallenge(room, player.ID)
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// resolveLiarsDiceChallenge：揭晓全体参战玩家的骰子，按叫点面值计数（1 是否万能取决于
// OnesWildDisabled），叫点者与开牌者一胜一负，其余参战玩家本局"平"，不计分不受罚。
func (s *Server) resolveLiarsDiceChallenge(room *RoomState, challengerID string) {
	bid := *room.LiarsDice.CurrentBid
	hands := map[string][]int{}
	actual := 0
	for _, id := range room.LiarsDice.ParticipantIDs {
		dice := append([]int{}, room.LiarsDiceHands[id]...)
		hands[id] = dice
		for _, d := range dice {
			if d == bid.Face {
				actual++
			} else if d == 1 && bid.Face != 1 && !room.LiarsDice.OnesWildDisabled {
				actual++
			}
		}
	}
	winnerID, loserID := bid.PlayerID, challengerID
	if actual < bid.Count {
		winnerID, loserID = challengerID, bid.PlayerID
	}
	winner := s.players[winnerID]
	loser := s.players[loserID]
	names := map[string]string{}
	for _, id := range room.LiarsDice.ParticipantIDs {
		if p := s.players[id]; p != nil {
			names[id] = playerShortName(p)
		}
	}
	room.LiarsDice.Ended = true
	room.LiarsDice.WinnerID = winnerID
	room.LiarsDice.LoserID = loserID
	room.LiarsDice.RevealedHands = hands
	room.LiarsDice.ActualCount = actual
	room.Phase = types.PhaseResult
	room.Status = "playing"

	rankedText := ""
	streakText := ""
	s.recordGameOutcome(winner, types.GameLiarsDice, "win")
	s.recordGameOutcome(loser, types.GameLiarsDice, "loss")
	// 其余参战玩家本局记平（不计排位分、不受罚）。
	for _, id := range room.LiarsDice.ParticipantIDs {
		if id == winnerID || id == loserID {
			continue
		}
		if p := s.players[id]; p != nil {
			s.recordGameOutcome(p, types.GameLiarsDice, "draw")
		}
	}
	s.logGameRoundForPlayers(room, string(types.GameLiarsDice)+":win", roomHumanPlayerIDs(room))
	s.applyGiveawayWinPenalty(winner)
	if room.Settings.EnableRanked {
		wD, lD := s.applyRankedStake(winner, loser, effectiveRankedStake(room.Settings))
		s.resetExtremeWinStreak(loser)
		streakText = s.applyExtremeWinStreakRisk(room, winner)
		rankedText = fmt.Sprintf("（%s %s，%s %d）", names[winnerID], formatSigned(wD), names[loserID], lD)
	}
	verdict := "叫点成立"
	if winnerID == challengerID {
		verdict = "叫点不成立"
	}
	room.ResultText = fmt.Sprintf("%s 开牌：叫点 %d 个 %d，实际 %d 个，%s，%s 胜利%s%s",
		names[challengerID], bid.Count, bid.Face, actual, verdict, names[winnerID], rankedText, streakText)
	for _, id := range room.LiarsDice.ParticipantIDs {
		if p := s.players[id]; p != nil {
			s.refreshPlayerSnapshots(p)
		}
	}

	var punishedPlayers []*PlayerState
	var punishedNames []string
	if room.Settings.EnablePunishment && loser != nil {
		punishedPlayers = []*PlayerState{loser}
		punishedNames = []string{names[loserID]}
	}
	punishmentTasks := s.buildLiarsDicePunishmentTasks(room, punishedPlayers, winner)

	handOrder := append([]string{}, room.LiarsDice.ParticipantIDs...)
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: names[winnerID], PlayerB: names[loserID],
		MoveA: types.MoveNoMove, MoveB: types.MoveNoMove,
		Result: types.ResultA, ResultLabel: names[winnerID] + " 胜利", ResultText: room.ResultText,
		GameID: types.GameLiarsDice, Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
		LiarsDiceWinnerID: winnerID, LiarsDiceLoserID: loserID,
		LiarsDiceBidCount: bid.Count, LiarsDiceBidFace: bid.Face, LiarsDiceActualCount: actual,
		LiarsDiceHands: hands, LiarsDiceHandOrder: handOrder, LiarsDiceNames: names,
	}
	if room.Settings.EnableRanked {
		st := room.Settings.Stake
		item.Stake = &st
		rm := rankMultiplierFor(room.Settings)
		item.RankMultiplier = &rm
		es := effectiveRankedStake(room.Settings)
		item.EffectiveStake = &es
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentRoundLabel(room, punishmentTasks)
	}
	s.roomNotice(room, room.ResultText)
	s.addRoundHistory(room, item)
	s.setupPunishmentForPlayers(room, punishedIDs(punishedPlayers))
}

func punishedIDs(players []*PlayerState) []string {
	out := make([]string, 0, len(players))
	for _, p := range players {
		out = append(out, p.ID)
	}
	return out
}

// buildLiarsDicePunishmentTasks：与 buildPunishmentTasksWithWinnerName 同构，但赢家/受罚者都是
// 直接传入的 *PlayerState（不走 RoundHistory[0] 反查）——此时本局 item 尚未写入
// room.RoundHistory，liarsDicePunishmentReviewer 那套按历史反查赢家的逻辑在这里还用不了，
// 直接用调用方传入的 winner 充当 approver（大话骰的 approver 恒定是本局赢家，见
// eventstore.go punishment_events 表注释）。
func (s *Server) buildLiarsDicePunishmentTasks(room *RoomState, punishedPlayers []*PlayerState, winner *PlayerState) []types.PunishmentTask {
	out := make([]types.PunishmentTask, 0, len(punishedPlayers))
	winnerName := ""
	if winner != nil {
		winnerName = playerShortName(winner)
	}
	src := normalizePunishmentSource(room.Settings.PunishmentSource)
	for _, player := range punishedPlayers {
		var systemTask *punishmentTaskResult
		if src == "series" {
			systemTask = s.pickSeriesTaskForPlayer(room, player, room.Settings.PunishmentSeriesID, winnerName)
		} else if src != "player" {
			systemTask = s.pickSystemTaskForPlayerAdvancing(room, player, winnerName)
		}
		task := types.PunishmentTask{
			PlayerID: player.ID, PlayerName: playerShortName(player),
			FactionID: player.FactionID, FactionLabel: player.FactionLabel,
		}
		if systemTask != nil {
			task.TaskText = systemTask.TaskText
			task.TypeName = systemTask.TypeName
			task.BackgroundImage = systemTask.BackgroundImage
			if systemTask.BackgroundOpacity != nil {
				task.BackgroundOpacity = systemTask.BackgroundOpacity
			}
		}
		// AssignedBy 只表示玩家临时任务的发布者；系统/系列任务的审批人虽仍是赢家，
		// 但不能在 UI 上显示成“由赢家发布”。
		if src == "player" && winner != nil {
			task.AssignedBy = winner.ID
			task.AssignedByName = winner.Name
		}
		// player 来源不在这里建事件行：此时玩家还没输入任务文案，事件要等
		// handlers_room.go 里赢家真正提交自定义任务文案时才创建。
		if s.eventDB != nil && systemTask != nil {
			eventID := randomID()
			approverID, approverName := "", ""
			if winner != nil {
				approverID, approverName = winner.ID, playerShortName(winner)
			}
			meta := systemTask.EventMeta
			meta.Trigger = "round_end"
			if err := s.eventDB.insertPunishmentTask(eventID, nowMs(), room.ID, approverID, approverName, player.ID, task.PlayerName, task.TaskText, meta); err != nil {
				s.errorLog("punishment_event_insert_failed", err.Error())
			} else {
				task.EventID = eventID
			}
		}
		out = append(out, task)
	}
	return out
}

// prepareNextLiarsDiceRound：惩罚流程走完后由 resetForNextRound 调用——统一回到 Ready，
// 参战名单保留但清掉"已准备"，给玩家一个"对局未开始"的窗口自由加入/离席，再重新准备开下一局。
// （不复用 RPS「上一局出拳即开下一局」的即时续局：大话骰是 N 人动态名单，续局前必须留调整窗口。）
func (s *Server) prepareNextLiarsDiceRound(room *RoomState) {
	if room.LiarsDice == nil {
		return
	}
	s.returnLiarsDiceToReady(room)
	s.scheduleLiarsDiceReadyStart(room)
}

// returnLiarsDiceToReady 把房间收束回 Ready（清掉上一局的叫点/骰子/开牌状态与"已准备"标记，
// 保留参战名单）。上一局开牌结果仍可在对局记录里回看。
func (s *Server) returnLiarsDiceToReady(room *RoomState) {
	if room.LiarsDice == nil {
		return
	}
	s.clearLiarsDiceStartTimer(room.ID)
	room.Phase = types.PhaseReady
	room.Status = "waiting"
	room.ResultText = ""
	ld := room.LiarsDice
	ld.ReadyPlayerIDs = []string{}
	ld.DiceCounts = map[string]int{}
	ld.CurrentBid = nil
	ld.CurrentTurn = ""
	ld.BidHistory = []types.LiarsDiceBid{}
	ld.OnesWildDisabled = false
	ld.Ended = false
	ld.WinnerID = ""
	ld.LoserID = ""
	ld.RevealedHands = nil
	ld.ActualCount = 0
	room.LiarsDiceHands = map[string][]int{}
	room.PunishedPlayerIDs = []string{}
	room.Proofs = []types.PunishmentProof{}
	room.LockedSeatIDs = map[string]struct{}{}
}

// liarsDicePunishmentReviewer 返回大话骰某个受罚玩家的审核人=本局赢家，
// 以最近一条对局记录里的 winner/loser 为准（结算/断线判负时写入）。大话骰不进 Seats 体系，
// 所以惩罚系统里所有原本用 seatOf 判断"对手"的地方都要改走这里（见 punishment.go）。
// 赢家若已离开参战席（roster 里查不到），视为无审核人——与座位制游戏里对手座位被清空后
// humanPlayerFromSeat 返回 nil 是同一语义，否则赢家掉线离场后，证明会永远卡在待审核
// （s.players[id] 只要玩家是持久身份就一直存在，不能拿它判断"是否还在这局里"）。
func (s *Server) liarsDicePunishmentReviewer(room *RoomState, punishedID string) *PlayerState {
	if len(room.RoundHistory) == 0 {
		return nil
	}
	latest := room.RoundHistory[0]
	if latest.LiarsDiceWinnerID == "" || latest.LiarsDiceLoserID != punishedID {
		return nil
	}
	if room.LiarsDice == nil || !containsString(room.LiarsDice.ParticipantIDs, latest.LiarsDiceWinnerID) {
		return nil
	}
	return s.players[latest.LiarsDiceWinnerID]
}

// onLiarsDiceNextRound：本局结算展示（Result）后，任一参战玩家点"再来一局"把房间收回 Ready。
// 惩罚模式下走的是 Punishment→惩罚完成→resetForNextRound 自动回 Ready，不经过这里。
func (s *Server) onLiarsDiceNextRound(client *Client, env wsEnvelope) {
	player, room, ok := s.requireRoomPlayer(client, env)
	if !ok {
		return
	}
	if room.Settings.GameID != types.GameLiarsDice || room.LiarsDice == nil {
		client.reply(env.ID, nil, "当前房间不是大话骰")
		return
	}
	if room.Phase != types.PhaseResult {
		client.reply(env.ID, nil, "本局还没有结束")
		return
	}
	if !containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		client.reply(env.ID, nil, "只有参战玩家可以再来一局")
		return
	}
	s.returnLiarsDiceToReady(room)
	s.roomNotice(room, playerShortName(player)+" 结束了本局，准备再来一局。")
	s.broadcastRoom(room.ID, true)
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

// ── 断线判负：断线的人判负，"上家"（入席顺序里的前一位）判胜 ──

func (s *Server) createLiarsDiceDisconnectForfeit(room *RoomState, player *PlayerState) {
	if room.Phase != types.PhaseChoosing || room.LiarsDice == nil {
		return
	}
	if !containsString(room.LiarsDice.ParticipantIDs, player.ID) {
		return
	}
	winnerID := liarsDicePredecessor(room.LiarsDice.ParticipantIDs, player.ID)
	if winnerID == "" || winnerID == player.ID {
		return
	}
	winner := s.players[winnerID]
	if winner == nil {
		return
	}
	if room.LiarsDiceDisconnectForfeits == nil {
		room.LiarsDiceDisconnectForfeits = map[string]LiarsDiceDisconnectForfeit{}
	}
	room.LiarsDiceDisconnectForfeits[player.ID] = LiarsDiceDisconnectForfeit{
		LoserID: player.ID, LoserName: playerShortName(player),
		WinnerID: winnerID, WinnerName: playerShortName(winner),
		Stake: effectiveRankedStake(room.Settings), BaseStake: room.Settings.Stake,
		RankMultiplier: rankMultiplierFor(room.Settings),
	}
}

func (s *Server) applyLiarsDiceDisconnectForfeit(room *RoomState, player *PlayerState) bool {
	forfeit, ok := room.LiarsDiceDisconnectForfeits[player.ID]
	if !ok {
		return false
	}
	delete(room.LiarsDiceDisconnectForfeits, player.ID)
	if room.Phase == types.PhaseResult || (room.LiarsDice != nil && room.LiarsDice.Ended) {
		return true
	}
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	rankedText := ""
	streakText := ""
	s.recordGameOutcome(winner, types.GameLiarsDice, "win")
	s.recordGameOutcome(loser, types.GameLiarsDice, "loss")
	// 其余参战玩家本局记平（与正常开牌结算一致）。
	if room.LiarsDice != nil {
		for _, id := range room.LiarsDice.ParticipantIDs {
			if id == forfeit.WinnerID || id == forfeit.LoserID {
				continue
			}
			if p := s.players[id]; p != nil {
				s.recordGameOutcome(p, types.GameLiarsDice, "draw")
			}
		}
	}
	s.logGameRoundForPlayers(room, string(types.GameLiarsDice)+":win", roomHumanPlayerIDs(room))
	s.applyGiveawayWinPenalty(winner)
	if room.Settings.EnableRanked {
		wD, lD := s.applyRankedStake(winner, loser, forfeit.Stake)
		s.resetExtremeWinStreak(loser)
		streakText = s.applyExtremeWinStreakRisk(room, winner)
		rankedText = fmt.Sprintf("（%s %s，%s %d）", forfeit.WinnerName, formatSigned(wD), forfeit.LoserName, lD)
	}
	room.Phase = types.PhaseResult
	room.Status = "playing"
	if room.LiarsDice != nil {
		room.LiarsDice.Ended = true
		room.LiarsDice.WinnerID = forfeit.WinnerID
		room.LiarsDice.LoserID = forfeit.LoserID
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，上家 %s 大话骰胜利%s%s", forfeit.LoserName, forfeit.WinnerName, rankedText, streakText)
	if room.LiarsDice != nil {
		for _, id := range room.LiarsDice.ParticipantIDs {
			if p := s.players[id]; p != nil {
				s.refreshPlayerSnapshots(p)
			}
		}
	} else {
		if winner != nil {
			s.refreshPlayerSnapshots(winner)
		}
		if loser != nil {
			s.refreshPlayerSnapshots(loser)
		}
	}
	var punishedPlayers []*PlayerState
	var punishedNames []string
	if room.Settings.EnablePunishment && loser != nil {
		punishedPlayers = []*PlayerState{loser}
		punishedNames = []string{forfeit.LoserName}
	}
	punishmentTasks := s.buildLiarsDicePunishmentTasks(room, punishedPlayers, winner)
	baseStake := forfeit.BaseStake
	rm := forfeit.RankMultiplier
	es := forfeit.Stake
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: forfeit.WinnerName, PlayerB: forfeit.LoserName,
		MoveA: types.MoveNoMove, MoveB: types.MoveForfeit,
		Result: types.ResultA, ResultLabel: forfeit.WinnerName + " 胜利", ResultText: room.ResultText,
		GameID: types.GameLiarsDice, Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
		LiarsDiceWinnerID: forfeit.WinnerID, LiarsDiceLoserID: forfeit.LoserID,
	}
	if room.Settings.EnableRanked {
		item.Stake = &baseStake
		item.RankMultiplier = &rm
		item.EffectiveStake = &es
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentRoundLabel(room, punishmentTasks)
	}
	s.roomNotice(room, room.ResultText)
	s.addRoundHistory(room, item)
	s.setupPunishmentForPlayers(room, punishedIDs(punishedPlayers))
	return true
}
