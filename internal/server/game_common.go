package server

import (
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// 轮流制棋类（黑白棋 / 井字棋 / 五子棋 / 斗兽棋 / 国际象棋）共用：复位字段、座位胜负统计、历史+惩罚收尾。
// RPS 结算路径不同（按手即时结算），不走此处。

// clampMoveSeconds/clampGameMinutes：黑白棋/五子棋/斗兽棋/国际象棋建房计时下拉框合法值校验，0 表示不限时。
func clampMoveSeconds(v int) int {
	switch v {
	case 0, 30, 45, 60, 90, 120, 180:
		return v
	default:
		return 0
	}
}

func clampGameMinutes(v int) int {
	switch v {
	case 0, 5, 10, 15, 20, 30, 45, 60:
		return v
	default:
		return 0
	}
}

// clampUndoLimit：黑白棋/五子棋/斗兽棋/国际象棋悔棋次数合法值，0 表示禁止悔棋。
func clampUndoLimit(v int) int {
	switch v {
	case 0, 1, 3, 10:
		return v
	default:
		return 0
	}
}

// undoRequestWindow：悔棋请求等待对方回应的窗口，超时自动拒绝（与五子棋一致）。
const undoRequestWindow = 30 * time.Second

// clearTurnBasedUndoTimer：黑白棋/斗兽棋/国际象棋共用一张按房间索引的悔棋超时表
// （一个房间同一时刻只运行一种游戏，与 turnBasedClockTimers 同理）。
func (s *Server) clearTurnBasedUndoTimer(roomID string) {
	if t := s.turnBasedUndoTimers[roomID]; t != nil {
		t.Stop()
		delete(s.turnBasedUndoTimers, roomID)
	}
}

// scheduleTurnBasedUndoTimeout：黑白棋/斗兽棋/国际象棋共用的悔棋超时调度。
// createdAt 为本次请求的创建时间戳（用于识别请求是否已被替换）；
// lookup 返回当前请求的创建时间戳（0 表示已无请求）；expire 执行置空请求、恢复计时与公告。
func (s *Server) scheduleTurnBasedUndoTimeout(roomID string, createdAt int64, lookup func(*RoomState) int64, expire func(*RoomState)) {
	s.clearTurnBasedUndoTimer(roomID)
	if createdAt == 0 {
		return
	}
	timer := timeAfterFunc(undoRequestWindow, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.turnBasedUndoTimers, roomID)
		current := s.rooms[roomID]
		if current == nil {
			return
		}
		if lookup(current) != createdAt {
			return
		}
		expire(current)
		s.broadcastRoom(current.ID, true)
	})
	s.turnBasedUndoTimers[roomID] = timer
}

// freshUndoCount：开局时给双方初始化 0 次悔棋计数。
func freshUndoCount() map[types.SeatKey]int {
	return map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
}

// earliestPositiveDeadline 取两个 epoch 毫秒时间戳中较早的一个；<=0 视为「未启用」并忽略。
func earliestPositiveDeadline(a, b int64) int64 {
	if a <= 0 {
		return b
	}
	if b <= 0 {
		return a
	}
	if a < b {
		return a
	}
	return b
}

// resetTurnBasedRoom 将房间回到「双方准备」公共字段（调用方再清各自的 othello/tictactoe 状态）。
func (s *Server) resetTurnBasedRoom(room *RoomState) {
	room.Phase = types.PhaseReady
	room.Status = "waiting"
	room.ResultText = ""
	room.RevealedChoices = nil
	room.Choices = map[types.SeatKey]types.Move{}
	room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.Ready = map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false}
}

// startTurnBasedPlaying 双方已坐、开局时的公共字段（调用方再写游戏状态）。
func (s *Server) startTurnBasedPlaying(room *RoomState) {
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.ResultText = ""
	room.Choices = map[types.SeatKey]types.Move{}
	room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	room.RevealedChoices = nil
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.Ready = map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false}
	s.logGameStart(room)
}

// roomHumanPlayerIDs 当前对局相关的人类玩家 ID（座位制取 A/B；大话骰取参战名单）。
func roomHumanPlayerIDs(room *RoomState) []string {
	if room == nil {
		return nil
	}
	if room.Settings.GameID == types.GameLiarsDice && room.LiarsDice != nil {
		out := make([]string, 0, len(room.LiarsDice.ParticipantIDs))
		seen := map[string]struct{}{}
		for _, id := range room.LiarsDice.ParticipantIDs {
			if id == "" {
				continue
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			out = append(out, id)
		}
		return out
	}
	var out []string
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if occ := room.Seats[seat]; occ != nil {
			if id := occ.GetID(); id != "" {
				out = append(out, id)
			}
		}
	}
	return out
}

// logGameStart 为每位参战玩家记一条开局事件（供漏斗按 player→visitor 折设备 UV）。
func (s *Server) logGameStart(room *RoomState) {
	if room == nil {
		return
	}
	gameID := room.Settings.GameID
	if gameID == "" {
		gameID = types.GameRPS
	}
	ids := roomHumanPlayerIDs(room)
	if len(ids) == 0 {
		s.logAnalyticsServerEvent("game_start", string(gameID), 1, "", room.ID)
		return
	}
	for _, pid := range ids {
		s.logAnalyticsServerEvent("game_start", string(gameID), 1, pid, room.ID)
	}
}

// logGameRoundForPlayers 对局结算埋点：每位参战玩家一条。
// value=1 仅写在第一条（对局计数/结果分布/单房局数仍按「一局一次」）；其余 value=0 只参与漏斗设备 UV 映射。
func (s *Server) logGameRoundForPlayers(room *RoomState, detail string, playerIDs []string) {
	if room == nil {
		return
	}
	if len(playerIDs) == 0 {
		s.logAnalyticsServerEvent("game_round", detail, 1, "", room.ID)
		return
	}
	seen := map[string]struct{}{}
	first := true
	for _, pid := range playerIDs {
		if pid == "" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		val := int64(0)
		if first {
			val = 1
			first = false
		}
		s.logAnalyticsServerEvent("game_round", detail, val, pid, room.ID)
	}
	if first {
		// 全是空 id
		s.logAnalyticsServerEvent("game_round", detail, 1, "", room.ID)
	}
}

// applySeatOutcome 按 result 更新人类玩家分游戏战绩、座位分与 seatStats；不处理排位分。
func (s *Server) applySeatOutcome(room *RoomState, result types.RoundResult) (playerA, playerB *PlayerState) {
	playerA = s.humanPlayerFromSeat(room, types.SeatA)
	playerB = s.humanPlayerFromSeat(room, types.SeatB)
	gameID := room.Settings.GameID
	if gameID == "" {
		gameID = types.GameRPS
	}
	// 分析：服务端侧对局事件（前端看不见结果）。
	detail := string(gameID)
	switch result {
	case types.ResultDraw:
		detail = string(gameID) + ":draw"
	case types.ResultDoubleLoss:
		detail = string(gameID) + ":doubleloss"
	case types.ResultA, types.ResultB:
		detail = string(gameID) + ":win"
	}
	var pids []string
	if playerA != nil {
		pids = append(pids, playerA.ID)
	}
	if playerB != nil {
		pids = append(pids, playerB.ID)
	}
	s.logGameRoundForPlayers(room, detail, pids)

	if result == types.ResultDraw {
		s.recordGameOutcome(playerA, gameID, "draw")
		s.recordGameOutcome(playerB, gameID, "draw")
		ssA := room.SeatStats[types.SeatA]
		ssA.Draws++
		room.SeatStats[types.SeatA] = ssA
		ssB := room.SeatStats[types.SeatB]
		ssB.Draws++
		room.SeatStats[types.SeatB] = ssB
		return
	}
	if result == types.ResultDoubleLoss {
		s.recordGameOutcome(playerA, gameID, "loss")
		s.recordGameOutcome(playerB, gameID, "loss")
		ssA := room.SeatStats[types.SeatA]
		ssA.Losses++
		room.SeatStats[types.SeatA] = ssA
		ssB := room.SeatStats[types.SeatB]
		ssB.Losses++
		room.SeatStats[types.SeatB] = ssB
		return
	}
	if result != types.ResultA && result != types.ResultB {
		return
	}
	loserSeat := oppositeSeat(types.SeatKey(result))
	winner := s.humanPlayerFromSeat(room, types.SeatKey(result))
	loser := s.humanPlayerFromSeat(room, loserSeat)
	s.recordGameOutcome(winner, gameID, "win")
	s.recordGameOutcome(loser, gameID, "loss")
	s.applyGiveawayWinPenalty(winner)
	room.Score[types.SeatKey(result)]++
	room.SeatedScore[types.SeatKey(result)]++
	ssW := room.SeatStats[types.SeatKey(result)]
	ssW.Wins++
	room.SeatStats[types.SeatKey(result)] = ssW
	ssL := room.SeatStats[loserSeat]
	ssL.Losses++
	room.SeatStats[loserSeat] = ssL
	return
}

// buildMatchHistoryShell 组装终局 history 公共字段（惩罚任务 / 名字 / 排位元数据可选填）。
func (s *Server) buildMatchHistoryShell(room *RoomState, result types.RoundResult, gameID types.GameID, resultLabel, resultText, punishmentTrigger string) types.RoundHistoryItem {
	punishedPlayers := s.punishmentPlayersForResult(room, result)
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, result, punishmentTrigger)
	item := types.RoundHistoryItem{
		ID:              randomID(),
		Round:           len(room.RoundHistory) + 1,
		At:              nowMs(),
		PlayerA:         occupantName(room.Seats[types.SeatA]),
		PlayerB:         occupantName(room.Seats[types.SeatB]),
		MoveA:           types.MoveNoMove,
		MoveB:           types.MoveNoMove,
		Result:          result,
		ResultLabel:     resultLabel,
		ResultText:      resultText,
		GameID:          gameID,
		Ranked:          room.Settings.EnableRanked,
		PunishmentTasks: punishmentTasks,
		PunishedNames:   punishedNames,
		Proofs:          []types.HistoryProof{},
	}
	if room.Settings.EnableRanked {
		st := room.Settings.Stake
		item.Stake = &st
		rm := rankMultiplierFor(room.Settings)
		item.RankMultiplier = &rm
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentRoundLabel(room, punishmentTasks)
	}
	return item
}

// finalizeMatch 写入 history 并进入惩罚或下一局。
func (s *Server) finalizeMatch(room *RoomState, result types.RoundResult, item types.RoundHistoryItem) {
	s.addRoundHistory(room, item)
	if room.skipEndPunishment {
		room.skipEndPunishment = false
		return
	}
	s.setupPunishmentOrNext(room, result)
}

// refreshHumans 刷新双方快照（出站用）。
func (s *Server) refreshHumans(playerA, playerB *PlayerState) {
	if playerA != nil {
		s.refreshPlayerSnapshots(playerA)
	}
	if playerB != nil {
		s.refreshPlayerSnapshots(playerB)
	}
}
