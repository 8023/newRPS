package server

import (
	"fmt"
	"math/rand"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func judge(a, b types.Move) types.RoundResult {
	if a == b {
		return types.ResultDraw
	}
	if (a == types.MoveRock && b == types.MoveScissors) ||
		(a == types.MoveScissors && b == types.MovePaper) ||
		(a == types.MovePaper && b == types.MoveRock) {
		return types.ResultA
	}
	return types.ResultB
}

// applyForgiveAdvantage 处理"放过对方"后的命运安排：非平局/非双白给时，
// 有 66% 概率无视双方实际出拳，直接判恩惠方获胜。第二个返回值仅在真正
// 改写了结果时非空，供结算文案说明原因。
func (s *Server) applyForgiveAdvantage(room *RoomState, result types.RoundResult) (types.RoundResult, *forgiveAdvantage) {
	advantage := room.ForgiveAdvantage
	if advantage == nil {
		return result, nil
	}
	room.ForgiveAdvantage = nil
	if result == types.ResultDoubleLoss || result == types.ResultDraw {
		return result, nil
	}
	beneficiarySeat, ok1 := s.seatOf(room, advantage.BeneficiaryID)
	targetSeat, ok2 := s.seatOf(room, advantage.TargetID)
	if !ok1 || !ok2 || beneficiarySeat == targetSeat {
		return result, nil
	}
	if rand.Float64() < 0.66 {
		return types.RoundResult(beneficiarySeat), advantage
	}
	return result, nil
}

// giveawayForcedSeats 返回本回合按白给值概率触发白给的座位。
// 主人强制会在点击时直接把 room.Choices 覆盖成 MoveGiveaway，不走概率分支。
func (s *Server) giveawayForcedSeats(room *RoomState) map[types.SeatKey]string {
	if !s.isFullHumanRoom(room) {
		return nil
	}
	seats := map[types.SeatKey]string{}
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if room.Choices[seat] == types.MoveGiveaway {
			continue
		}
		occ := room.Seats[seat]
		if occ == nil {
			continue
		}
		player := s.players[occ.GetID()]
		if player != nil && s.shouldTriggerGiveaway(player) {
			seats[seat] = ""
		}
	}
	return seats
}

// shouldTriggerGiveaway 只处理玩家自己的白给值概率；主人强制由房间内本局状态处理。
func (s *Server) shouldTriggerGiveaway(player *PlayerState) bool {
	return player != nil && ptrBool(player.GiveawayEnabled) && ptrFloat(player.GiveawayValue) > 0 &&
		rand.Float64()*100 < ptrFloat(player.GiveawayValue)
}

func (s *Server) resultWithGiveaway(room *RoomState, baseResult types.RoundResult, finalChoices map[types.SeatKey]types.Move) (types.RoundResult, *forgiveAdvantage) {
	var giveawaySeats []types.SeatKey
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if finalChoices[seat] == types.MoveGiveaway {
			giveawaySeats = append(giveawaySeats, seat)
		}
	}
	if len(giveawaySeats) == 1 {
		// 白给回合的结果与出拳无关，命运安排的名额不应因此被延后到未来某局生效。
		room.ForgiveAdvantage = nil
		if giveawaySeats[0] == types.SeatA {
			return types.ResultB, nil
		}
		return types.ResultA, nil
	}
	if len(giveawaySeats) >= 2 {
		room.ForgiveAdvantage = nil
		return types.ResultDraw, nil
	}
	return s.applyForgiveAdvantage(room, baseResult)
}

func moveText(move types.Move) string {
	switch move {
	case types.MoveNoMove:
		return "未出拳"
	case types.MoveForfeit:
		return "断线判负"
	case types.MoveGiveaway:
		return "白给"
	case types.MoveRock:
		return "石头"
	case types.MoveScissors:
		return "剪刀"
	case types.MovePaper:
		return "布"
	default:
		return string(move)
	}
}

func (s *Server) roundResultLabel(room *RoomState, result types.RoundResult) string {
	if result == types.ResultDoubleLoss {
		return "双方白给，双输"
	}
	if result == types.ResultDraw {
		if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			return fmt.Sprintf("平局双扣 -%d", effectiveRankedStake(room.Settings))
		}
		return "平局"
	}
	return occupantName(room.Seats[types.SeatKey(result)]) + "胜利"
}

func (s *Server) maybeStartChoosing(room *RoomState) {
	if room.Phase == types.PhasePunishment || room.Phase == types.PhaseResult {
		return
	}
	if room.Settings.GameID == types.GameOthello {
		if room.Phase == types.PhaseReady || room.Phase == types.PhaseChoosing {
			return
		}
		if room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil {
			s.resetOthelloRoom(room)
		}
		return
	}
	if room.Settings.GameID == types.GameTicTacToe {
		if room.Phase == types.PhaseReady || room.Phase == types.PhaseChoosing {
			return
		}
		if room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil {
			s.resetTicTacToeRoom(room)
		}
		return
	}
	if room.Settings.GameID == types.GameGomoku {
		if room.Phase == types.PhaseReady || room.Phase == types.PhaseChoosing {
			return
		}
		if room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil {
			s.resetGomokuRoom(room)
		}
		return
	}
	if room.Settings.GameID == types.GameJungle {
		if room.Phase == types.PhaseReady || room.Phase == types.PhaseChoosing {
			return
		}
		if room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil {
			s.resetJungleRoom(room)
		}
		return
	}
	if room.Phase == types.PhaseChoosing && (room.Choices[types.SeatA] != "" || room.Choices[types.SeatB] != "") {
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.Choices = map[types.SeatKey]types.Move{}
	room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	room.RevealedChoices = nil
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.ResultText = ""
	room.Proofs = []types.PunishmentProof{}
	room.PunishedPlayerIDs = []string{}
}

func (s *Server) prepareNextChoice(room *RoomState) {
	if room.Settings.GameID == types.GameOthello {
		s.resetOthelloRoom(room)
		return
	}
	if room.Settings.GameID == types.GameTicTacToe {
		s.resetTicTacToeRoom(room)
		return
	}
	if room.Settings.GameID == types.GameLiarsDice {
		s.prepareNextLiarsDiceRound(room)
		return
	}
	if room.Settings.GameID == types.GameGomoku {
		s.resetGomokuRoom(room)
		return
	}
	if room.Settings.GameID == types.GameJungle {
		s.resetJungleRoom(room)
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		room.Phase = types.PhaseReady
		room.Status = "waiting"
		room.Choices = map[types.SeatKey]types.Move{}
		room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
		room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
		room.RevealedChoices = nil
		room.DisconnectForfeits = map[string]DisconnectForfeit{}
		return
	}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.Choices = map[types.SeatKey]types.Move{}
	room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	room.RevealedChoices = nil
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.ResultText = ""
	room.Proofs = []types.PunishmentProof{}
	room.PunishedPlayerIDs = []string{}
}

// forceEndRpsRound 管理员强制判定当前 RPS 出拳阶段的胜负；仅在 PhaseChoosing 时允许，
// 避免和 finishRoundIfReady 的自然结算竞争，也避免对已出结果的对局重复结算
// （RPS 没有像其它三个游戏那样常驻的 Ended 状态对象可供二次校验，改用 Phase 把关）。
func (s *Server) forceEndRpsRound(room *RoomState, result types.RoundResult) (bool, string) {
	if room.Phase != types.PhaseChoosing {
		return false, "当前不在出拳阶段，无法强制判定"
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return false, "双方座位未坐满"
	}
	// 与自然结算一致：强制判定后作废未消费的断线判负条目，避免超时回调二次结算。
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	rankedStake := effectiveRankedStake(room.Settings)
	rankedText := ""
	streakText := ""
	playerA, playerB := s.applySeatOutcome(room, result)
	if result == types.ResultDraw {
		if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
			rankedText = fmt.Sprintf("（平局双扣：A %d，B %d）", dA, dB)
		}
	} else if result == types.ResultA || result == types.ResultB {
		if room.Settings.EnableRanked {
			loserSeat := oppositeSeat(types.SeatKey(result))
			winner := s.humanPlayerFromSeat(room, types.SeatKey(result))
			loser := s.humanPlayerFromSeat(room, loserSeat)
			wD, lD := s.applyRankedStake(winner, loser, rankedStake)
			s.resetExtremeWinStreak(loser)
			streakText = s.applyExtremeWinStreakRisk(room, winner)
			rankedText = fmt.Sprintf("（%s %s，%s %d）",
				occupantName(room.Seats[types.SeatKey(result)]), formatSigned(wD),
				occupantName(room.Seats[loserSeat]), lD)
		}
	}
	room.Phase = types.PhaseResult
	room.Status = "playing"
	room.RevealedChoices = nil
	if result == types.ResultDraw {
		room.ResultText = "管理员判定：平局" + rankedText
	} else {
		room.ResultText = "管理员判定：" + occupantName(room.Seats[types.SeatKey(result)]) + " 胜利" + rankedText + streakText
	}
	s.refreshHumans(playerA, playerB)
	resultLabel := s.roundResultLabel(room, result)
	item := s.buildMatchHistoryShell(room, result, types.GameRPS, resultLabel, room.ResultText)
	item.ExtremeRanked = room.Settings.EnableExtremeRanked
	if room.Settings.EnableRanked {
		es := rankedStake
		item.EffectiveStake = &es
	}
	s.roomNotice(room, room.ResultText)
	s.finalizeMatch(room, result, item)
	return true, ""
}

func (s *Server) finishRoundIfReady(room *RoomState) {
	if room.Choices[types.SeatA] == "" || room.Choices[types.SeatB] == "" {
		return
	}
	// 本局已自然结算：丢弃断线判负条目，防止 discTimer 再走 applyDisconnectForfeit 双重计分。
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	choiceA := room.Choices[types.SeatA]
	choiceB := room.Choices[types.SeatB]
	giveawaySeats := s.giveawayForcedSeats(room)
	finalChoices := map[types.SeatKey]types.Move{
		types.SeatA: choiceA,
		types.SeatB: choiceB,
	}
	for seat := range giveawaySeats {
		finalChoices[seat] = types.MoveGiveaway
	}
	var baseResult types.RoundResult
	if finalChoices[types.SeatA] == types.MoveGiveaway || finalChoices[types.SeatB] == types.MoveGiveaway {
		baseResult = types.ResultDraw
	} else {
		baseResult = judge(finalChoices[types.SeatA], finalChoices[types.SeatB])
	}
	result, forgiveOutcome := s.resultWithGiveaway(room, baseResult, finalChoices)
	punishedPlayers := s.punishmentPlayersForResult(room, result)
	punishedNames := make([]string, 0, len(punishedPlayers))
	for _, p := range punishedPlayers {
		punishedNames = append(punishedNames, playerShortName(p))
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, result, punishment)
	room.Phase = types.PhaseResult
	room.RevealedChoices = finalChoices

	playerA := s.humanPlayerFromSeat(room, types.SeatA)
	playerB := s.humanPlayerFromSeat(room, types.SeatB)
	rankedMultiplier := rankMultiplierFor(room.Settings)
	rankedStake := effectiveRankedStake(room.Settings)
	for seat := range giveawaySeats {
		var player *PlayerState
		if seat == types.SeatA {
			player = playerA
		} else {
			player = playerB
		}
		if player != nil {
			s.addGiveawayValue(player, s.cfg.Giveaway.ActiveBoostValue)
		}
	}
	var giveawayResultSeats []types.SeatKey
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if finalChoices[seat] == types.MoveGiveaway {
			giveawayResultSeats = append(giveawayResultSeats, seat)
		}
	}
	var giveawayReasons []string
	for _, seat := range giveawayResultSeats {
		if masterName := forcedGiveawayMasterName(room, seat); masterName != "" {
			giveawayReasons = append(giveawayReasons, fmt.Sprintf("主人（%s）强制（%s）白给", masterName, occupantName(room.Seats[seat])))
		} else {
			giveawayReasons = append(giveawayReasons, occupantName(room.Seats[seat])+" 白给")
		}
		if room.ForcedGiveawayBySeat != nil {
			delete(room.ForcedGiveawayBySeat, seat)
		}
	}
	giveawayText := strings.Join(giveawayReasons, "；")

	if result == types.ResultDoubleLoss {
		s.applySeatOutcome(room, result)
		s.resetExtremeWinStreak(playerA)
		s.resetExtremeWinStreak(playerB)
		if room.Settings.EnableRanked {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
			room.ResultText = fmt.Sprintf("双方白给，双输：A %d 分，B %d 分", dA, dB)
		} else {
			room.ResultText = "双方白给，双输"
		}
	} else if result == types.ResultDraw {
		s.applySeatOutcome(room, result)
		s.resetExtremeWinStreak(playerA)
		s.resetExtremeWinStreak(playerB)
		if len(giveawayResultSeats) >= 2 {
			room.ResultText = "双方白给，平局"
			if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
				dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
				room.ResultText = fmt.Sprintf("双方白给，平局双罚：A %d 分，B %d 分", dA, dB)
			}
			if giveawayText != "" {
				room.ResultText += "（" + giveawayText + "）"
			}
		} else if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
			room.ResultText = fmt.Sprintf("平局双罚：双方都出了 %s，A %d 分，B %d 分", moveText(finalChoices[types.SeatA]), dA, dB)
		} else {
			room.ResultText = fmt.Sprintf("平局：双方都出了 %s", moveText(finalChoices[types.SeatA]))
		}
	} else {
		winnerSeat := types.SeatKey(result)
		loserSeat := oppositeSeat(winnerSeat)
		var winner, loser *PlayerState
		if winnerSeat == types.SeatA {
			winner, loser = playerA, playerB
		} else {
			winner, loser = playerB, playerA
		}
		s.applySeatOutcome(room, result)
		streakText := ""
		rankedText := ""
		if room.Settings.EnableRanked {
			wD, lD := s.applyRankedStake(winner, loser, rankedStake)
			s.resetExtremeWinStreak(loser)
			streakText = s.applyExtremeWinStreakRisk(room, winner)
			rankedText = fmt.Sprintf("（%s %s，%s %d）",
				occupantName(room.Seats[winnerSeat]), formatSigned(wD),
				occupantName(room.Seats[loserSeat]), lD)
		}
		if giveawayText != "" {
			room.ResultText = fmt.Sprintf("%s，%s胜利%s%s", giveawayText, occupantName(room.Seats[winnerSeat]), rankedText, streakText)
		} else {
			room.ResultText = fmt.Sprintf("%s胜利%s%s", occupantName(room.Seats[winnerSeat]), rankedText, streakText)
		}
		if forgiveOutcome != nil {
			// 命中命运安排时 winnerSeat 必是 forgiveOutcome 的受益方座位，loserSeat 是被放过方座位。
			room.ResultText += fmt.Sprintf("（%s 上局放过了 %s，本局已受到「命运的干预」）",
				occupantName(room.Seats[winnerSeat]), occupantName(room.Seats[loserSeat]))
		}
	}

	item := types.RoundHistoryItem{
		ID:              randomID(),
		Round:           len(room.RoundHistory) + 1,
		At:              nowMs(),
		PlayerA:         occupantName(room.Seats[types.SeatA]),
		PlayerB:         occupantName(room.Seats[types.SeatB]),
		MoveA:           finalChoices[types.SeatA],
		MoveB:           finalChoices[types.SeatB],
		Result:          result,
		ResultLabel:     s.roundResultLabel(room, result),
		ResultText:      room.ResultText,
		Ranked:          room.Settings.EnableRanked,
		ExtremeRanked:   room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks,
		PunishedNames:   punishedNames,
		Proofs:          []types.HistoryProof{},
	}
	if room.Settings.EnableRanked {
		st := room.Settings.Stake
		item.Stake = &st
		rm := rankedMultiplier
		item.RankMultiplier = &rm
		es := rankedStake
		item.EffectiveStake = &es
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentNameForRoom(room, punishment)
		if room.Settings.PunishmentSource != "player" && punishment != nil {
			item.PunishmentDescription = punishment.Description
		}
	}
	s.addRoundHistory(room, item)
	s.setupPunishmentOrNext(room, result)
}

func (s *Server) applyDisconnectForfeit(room *RoomState, player *PlayerState) bool {
	if room.Settings.GameID == types.GameLiarsDice {
		return s.applyLiarsDiceDisconnectForfeit(room, player)
	}
	forfeit, ok := room.DisconnectForfeits[player.ID]
	if !ok {
		return false
	}
	delete(room.DisconnectForfeits, player.ID)
	if room.Settings.GameID == types.GameOthello {
		return s.applyOthelloDisconnectForfeit(room, forfeit)
	}
	if room.Settings.GameID == types.GameTicTacToe {
		return s.applyTicTacToeDisconnectForfeit(room, forfeit)
	}
	if room.Settings.GameID == types.GameGomoku {
		return s.applyGomokuDisconnectForfeit(room, forfeit)
	}
	if room.Settings.GameID == types.GameJungle {
		return s.applyJungleDisconnectForfeit(room, forfeit)
	}
	// RPS：仅出拳阶段可判负；已进入 result/punishment/ready 说明本局已结算或未在对局中。
	if room.Phase != types.PhaseChoosing {
		return true
	}
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	wD, lD := s.applyRankedStake(winner, loser, forfeit.Stake)
	s.recordGameOutcome(winner, types.GameRPS, "win")
	s.recordGameOutcome(loser, types.GameRPS, "loss")
	s.applyGiveawayWinPenalty(winner)
	s.resetExtremeWinStreak(loser)
	streakText := s.applyExtremeWinStreakRisk(room, winner)
	room.Score[forfeit.WinnerSeat]++
	room.SeatedScore[forfeit.WinnerSeat]++
	ssW := room.SeatStats[forfeit.WinnerSeat]
	ssW.Wins++
	room.SeatStats[forfeit.WinnerSeat] = ssW
	ssL := room.SeatStats[forfeit.LoserSeat]
	ssL.Losses++
	room.SeatStats[forfeit.LoserSeat] = ssL
	room.Phase = types.PhaseResult
	room.Status = "playing"
	room.RevealedChoices = nil
	stakeText := fmt.Sprintf("%d 分", forfeit.Stake)
	if forfeit.RankMultiplier > 1 {
		stakeText = fmt.Sprintf("%d 分 ×%d = %d 分", forfeit.BaseStake, forfeit.RankMultiplier, forfeit.Stake)
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s胜利，排位 %s 已结算（%s %s，%s %d）%s",
		forfeit.LoserName, forfeit.WinnerName, stakeText,
		forfeit.WinnerName, formatSigned(wD), forfeit.LoserName, lD, streakText)
	moveA, moveB := types.MoveNoMove, types.MoveNoMove
	if forfeit.LoserSeat == types.SeatA {
		moveA = types.MoveForfeit
		if m, ok := room.Choices[types.SeatB]; ok {
			moveB = m
		}
	} else {
		moveB = types.MoveForfeit
		if m, ok := room.Choices[types.SeatA]; ok {
			moveA = m
		}
	}
	playerAName, playerBName := forfeit.WinnerName, forfeit.LoserName
	if forfeit.LoserSeat == types.SeatA {
		playerAName, playerBName = forfeit.LoserName, forfeit.WinnerName
	}
	baseStake := forfeit.BaseStake
	rm := forfeit.RankMultiplier
	es := forfeit.Stake
	s.addRoundHistory(room, types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: playerAName, PlayerB: playerBName,
		MoveA: moveA, MoveB: moveB, Result: types.RoundResult(forfeit.WinnerSeat),
		ResultLabel: forfeit.WinnerName + "胜利", ResultText: room.ResultText,
		Ranked: true, Stake: &baseStake, RankMultiplier: &rm, EffectiveStake: &es,
		ExtremeRanked:   room.Settings.EnableExtremeRanked,
		PunishmentTasks: []types.PunishmentTask{}, PunishedNames: []string{}, Proofs: []types.HistoryProof{},
	})
	// 消费掉本局 forfeit 后清空 map，避免另一侧同时断线再结算一次。
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	s.roomNotice(room, room.ResultText)
	return true
}
