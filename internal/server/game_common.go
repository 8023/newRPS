package server

import "github.com/doumiao/newRPS/internal/types"

// 轮流制棋类（黑白棋 / 井字棋）共用：复位字段、座位胜负统计、历史+惩罚收尾。
// RPS 结算路径不同（按手即时结算），不走此处。

// clampMoveSeconds/clampGameMinutes：黑白棋/五子棋建房计时下拉框合法值校验，0 表示不限时。
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
}

// applySeatOutcome 按 result 更新人类玩家分游戏战绩、座位分与 seatStats；不处理排位分。
func (s *Server) applySeatOutcome(room *RoomState, result types.RoundResult) (playerA, playerB *PlayerState) {
	playerA = s.humanPlayerFromSeat(room, types.SeatA)
	playerB = s.humanPlayerFromSeat(room, types.SeatB)
	gameID := room.Settings.GameID
	if gameID == "" {
		gameID = types.GameRPS
	}
	if result == types.ResultDraw {
		recordGameOutcome(playerA, gameID, "draw")
		recordGameOutcome(playerB, gameID, "draw")
		ssA := room.SeatStats[types.SeatA]
		ssA.Draws++
		room.SeatStats[types.SeatA] = ssA
		ssB := room.SeatStats[types.SeatB]
		ssB.Draws++
		room.SeatStats[types.SeatB] = ssB
		return
	}
	if result == types.ResultDoubleLoss {
		recordGameOutcome(playerA, gameID, "loss")
		recordGameOutcome(playerB, gameID, "loss")
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
	recordGameOutcome(winner, gameID, "win")
	recordGameOutcome(loser, gameID, "loss")
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
func (s *Server) buildMatchHistoryShell(room *RoomState, result types.RoundResult, gameID types.GameID, resultLabel, resultText string) types.RoundHistoryItem {
	punishedPlayers := s.punishmentPlayersForResult(room, result)
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, result, punishment)
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
		item.PunishmentName = s.punishmentNameForRoom(room, punishment)
		if room.Settings.PunishmentSource != "player" && punishment != nil {
			item.PunishmentDescription = punishment.Description
		}
	}
	return item
}

// finalizeMatch 写入 history 并进入惩罚或下一局。
func (s *Server) finalizeMatch(room *RoomState, result types.RoundResult, item types.RoundHistoryItem) {
	s.addRoundHistory(room, item)
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
