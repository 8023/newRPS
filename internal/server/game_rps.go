package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func isRpsMove(move types.Move) bool {
	return move == types.MoveRock || move == types.MoveScissors || move == types.MovePaper
}

// maxRecentMoves 限制每个玩家保留的历史出拳数（Bot 反制策略只看最近一手）。
const maxRecentMoves = 20

func appendRecentMove(moves []RpsMove, move RpsMove) []RpsMove {
	moves = append(moves, move)
	if len(moves) > maxRecentMoves {
		moves = moves[len(moves)-maxRecentMoves:]
	}
	return moves
}

func judge(a, b RpsMove) types.RoundResult {
	if a == b {
		return types.ResultDraw
	}
	if (a == "rock" && b == "scissors") || (a == "scissors" && b == "paper") || (a == "paper" && b == "rock") {
		return types.ResultA
	}
	return types.ResultB
}

func (s *Server) applyForgiveAdvantage(room *RoomState, result types.RoundResult) types.RoundResult {
	advantage := room.ForgiveAdvantage
	if advantage == nil {
		return result
	}
	room.ForgiveAdvantage = nil
	if result == types.ResultDoubleLoss {
		return result
	}
	beneficiarySeat, ok1 := s.seatOf(room, advantage.BeneficiaryID)
	targetSeat, ok2 := s.seatOf(room, advantage.TargetID)
	if !ok1 || !ok2 || beneficiarySeat == targetSeat {
		return result
	}
	if rand.Float64() < 0.66 {
		return types.RoundResult(beneficiarySeat)
	}
	return result
}

func (s *Server) giveawayForcedSeats(room *RoomState) []types.SeatKey {
	if !s.isHumanVsHumanRoom(room) {
		return nil
	}
	var seats []types.SeatKey
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if room.Choices[seat] == types.MoveGiveaway {
			continue
		}
		occ := room.Seats[seat]
		if occ == nil || occ.IsBot() {
			continue
		}
		player := s.players[occ.GetID()]
		if player != nil && s.shouldTriggerGiveaway(player) {
			seats = append(seats, seat)
		}
	}
	return seats
}

func (s *Server) shouldTriggerGiveaway(player *PlayerState) bool {
	return ptrBool(player.GiveawayEnabled) && ptrFloat(player.GiveawayValue) > 0 &&
		rand.Float64()*100 < ptrFloat(player.GiveawayValue)
}

func (s *Server) resultWithGiveaway(room *RoomState, baseResult types.RoundResult, finalChoices map[types.SeatKey]types.Move) types.RoundResult {
	var giveawaySeats []types.SeatKey
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if finalChoices[seat] == types.MoveGiveaway {
			giveawaySeats = append(giveawaySeats, seat)
		}
	}
	if len(giveawaySeats) == 1 {
		if giveawaySeats[0] == types.SeatA {
			return types.ResultB
		}
		return types.ResultA
	}
	if len(giveawaySeats) >= 2 {
		return types.ResultDoubleLoss
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

func winningMoveAgainst(move RpsMove) RpsMove {
	switch move {
	case "rock":
		return "paper"
	case "paper":
		return "scissors"
	default:
		return "rock"
	}
}

func losingMoveAgainst(move RpsMove) RpsMove {
	switch move {
	case "rock":
		return "scissors"
	case "paper":
		return "rock"
	default:
		return "paper"
	}
}

func (s *Server) botMove(room *RoomState, bot types.BotPlayer) types.Move {
	moves := []RpsMove{"rock", "scissors", "paper"}
	var opponent *PlayerState
	if room.Seats[types.SeatA] != nil && !room.Seats[types.SeatA].IsBot() {
		opponent = s.players[room.Seats[types.SeatA].GetID()]
	}
	strategy := types.BotStrategy("random")
	for _, d := range s.cfg.Bots.Difficulties {
		if d.ID == bot.Difficulty {
			if d.Strategy != "" {
				strategy = d.Strategy
			}
			break
		}
	}
	if strategy == "" {
		switch bot.Difficulty {
		case "normal":
			strategy = "counter"
		case "chaos":
			strategy = "chaos"
		default:
			strategy = "random"
		}
	}
	var currentOpponentMove RpsMove
	hasCurrent := false
	if opponent != nil {
		if seat, ok := s.seatOf(room, opponent.ID); ok {
			if m, ok2 := room.Choices[seat]; ok2 && isRpsMove(m) {
				currentOpponentMove = RpsMove(m)
				hasCurrent = true
			}
		}
	}
	if strategy == "win" && hasCurrent {
		return types.Move(winningMoveAgainst(currentOpponentMove))
	}
	if strategy == "throw" && hasCurrent {
		return types.Move(losingMoveAgainst(currentOpponentMove))
	}
	if strategy == "chaos" && rand.Float64() < 0.35 && room.RevealedChoices != nil {
		if m, ok := room.RevealedChoices[types.SeatB]; ok && isRpsMove(m) {
			return m
		}
	}
	if strategy == "counter" && opponent != nil && len(opponent.RecentMoves) > 0 {
		last := opponent.RecentMoves[len(opponent.RecentMoves)-1]
		return types.Move(winningMoveAgainst(last))
	}
	return types.Move(moves[rand.Intn(len(moves))])
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
	if room.Phase == types.PhaseChoosing && (room.Choices[types.SeatA] != "" || room.Choices[types.SeatB] != "") {
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.Choices = map[types.SeatKey]types.Move{}
	room.RevealedChoices = nil
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
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		room.Phase = types.PhaseReady
		room.Status = "waiting"
		return
	}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.Choices = map[types.SeatKey]types.Move{}
	room.RevealedChoices = nil
	room.ResultText = ""
	room.Proofs = []types.PunishmentProof{}
	room.PunishedPlayerIDs = []string{}
}

func (s *Server) maybeBotAct(room *RoomState) {
	var botSeat types.SeatKey
	found := false
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if room.Seats[seat] != nil && room.Seats[seat].IsBot() {
			botSeat = seat
			found = true
			break
		}
	}
	if !found || room.Phase != types.PhaseChoosing {
		return
	}
	if room.Choices[botSeat] != "" {
		return
	}
	if s.botTimers[room.ID] != nil {
		return
	}
	delay := time.Duration(600+rand.Intn(700)) * time.Millisecond
	roomID := room.ID
	timer := timeAfterFunc(delay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.botTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.Phase != types.PhaseChoosing {
			return
		}
		if current.Choices[botSeat] != "" {
			return
		}
		botOcc := current.Seats[botSeat]
		if botOcc == nil || !botOcc.IsBot() {
			return
		}
		current.Choices[botSeat] = s.botMove(current, botOcc.(*BotSeat).Bot)
		oldStatus := current.Status
		s.finishRoundIfReady(current)
		s.broadcastRoom(current.ID, oldStatus != current.Status)
	})
	s.botTimers[room.ID] = timer
}

func (s *Server) finishRoundIfReady(room *RoomState) {
	if room.Choices[types.SeatA] == "" || room.Choices[types.SeatB] == "" {
		return
	}
	choiceA := room.Choices[types.SeatA]
	choiceB := room.Choices[types.SeatB]
	giveawaySeats := s.giveawayForcedSeats(room)
	finalChoices := map[types.SeatKey]types.Move{
		types.SeatA: choiceA,
		types.SeatB: choiceB,
	}
	for _, seat := range giveawaySeats {
		finalChoices[seat] = types.MoveGiveaway
	}
	var baseResult types.RoundResult
	if finalChoices[types.SeatA] == types.MoveGiveaway || finalChoices[types.SeatB] == types.MoveGiveaway {
		baseResult = types.ResultDraw
	} else {
		baseResult = judge(RpsMove(finalChoices[types.SeatA]), RpsMove(finalChoices[types.SeatB]))
	}
	result := s.resultWithGiveaway(room, baseResult, finalChoices)
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
	if playerA != nil && isRpsMove(choiceA) {
		playerA.RecentMoves = appendRecentMove(playerA.RecentMoves, RpsMove(choiceA))
	}
	if playerB != nil && isRpsMove(choiceB) {
		playerB.RecentMoves = appendRecentMove(playerB.RecentMoves, RpsMove(choiceB))
	}
	for _, seat := range giveawaySeats {
		var player *PlayerState
		if seat == types.SeatA {
			player = playerA
		} else {
			player = playerB
		}
		if player != nil {
			s.addGiveawayValue(player, 2)
		}
	}
	var giveawayResultSeats []types.SeatKey
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if finalChoices[seat] == types.MoveGiveaway {
			giveawayResultSeats = append(giveawayResultSeats, seat)
		}
	}
	giveawayText := ""
	if len(giveawayResultSeats) >= 2 {
		giveawayText = "双方白给"
	} else if len(giveawayResultSeats) == 1 {
		giveawayText = occupantName(room.Seats[giveawayResultSeats[0]]) + " 白给"
	}

	if result == types.ResultDoubleLoss {
		if playerA != nil {
			playerA.Stats.Losses++
		}
		if playerB != nil {
			playerB.Stats.Losses++
		}
		ssA := room.SeatStats[types.SeatA]
		ssA.Losses++
		room.SeatStats[types.SeatA] = ssA
		ssB := room.SeatStats[types.SeatB]
		ssB.Losses++
		room.SeatStats[types.SeatB] = ssB
		resetExtremeWinStreak(playerA)
		resetExtremeWinStreak(playerB)
		if room.Settings.EnableRanked {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
			room.ResultText = fmt.Sprintf("双方白给，双输：A %d 分，B %d 分", dA, dB)
		} else {
			room.ResultText = "双方白给，双输"
		}
	} else if result == types.ResultDraw {
		if playerA != nil {
			playerA.Stats.Draws++
		}
		if playerB != nil {
			playerB.Stats.Draws++
		}
		ssA := room.SeatStats[types.SeatA]
		ssA.Draws++
		room.SeatStats[types.SeatA] = ssA
		ssB := room.SeatStats[types.SeatB]
		ssB.Draws++
		room.SeatStats[types.SeatB] = ssB
		resetExtremeWinStreak(playerA)
		resetExtremeWinStreak(playerB)
		if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, rankedStake)
			room.ResultText = fmt.Sprintf("平局双罚：双方都出了 %s，A %d 分，B %d 分", moveText(finalChoices[types.SeatA]), dA, dB)
		} else {
			room.ResultText = fmt.Sprintf("平局：双方都出了 %s", moveText(finalChoices[types.SeatA]))
		}
		if giveawayText != "" {
			room.ResultText = giveawayText + "：" + room.ResultText
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
		if winner != nil {
			winner.Stats.Wins++
		}
		if loser != nil {
			loser.Stats.Losses++
		}
		room.Score[winnerSeat]++
		room.SeatedScore[winnerSeat]++
		ssW := room.SeatStats[winnerSeat]
		ssW.Wins++
		room.SeatStats[winnerSeat] = ssW
		ssL := room.SeatStats[loserSeat]
		ssL.Losses++
		room.SeatStats[loserSeat] = ssL
		streakText := ""
		rankedText := ""
		if room.Settings.EnableRanked {
			wD, lD := s.applyRankedStake(winner, loser, rankedStake)
			resetExtremeWinStreak(loser)
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
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	wD, lD := s.applyRankedStake(winner, loser, forfeit.Stake)
	if winner != nil {
		winner.Stats.Wins++
	}
	if loser != nil {
		loser.Stats.Losses++
	}
	resetExtremeWinStreak(loser)
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
		ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: []types.PunishmentTask{}, PunishedNames: []string{}, Proofs: []types.HistoryProof{},
	})
	s.roomNotice(room, room.ResultText)
	return true
}
