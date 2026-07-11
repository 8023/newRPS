package server

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func initialTicTacToeBoard() [][]*types.TicTacToeCell {
	board := make([][]*types.TicTacToeCell, 3)
	for i := 0; i < 3; i++ {
		board[i] = make([]*types.TicTacToeCell, 3)
	}
	return board
}

func freshTicTacToeState(xSeat types.SeatKey) *types.TicTacToeState {
	return &types.TicTacToeState{
		Board:       initialTicTacToeBoard(),
		Turn:        xSeat,
		XSeat:       xSeat,
		MoveCount:   0,
		RankedDelta: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
	}
}

func tictactoeMarkForSeat(state *types.TicTacToeState, seat types.SeatKey) types.TicTacToeCell {
	if state.XSeat == seat {
		return types.CellX
	}
	return types.CellO
}

func tictactoeSeatForMark(state *types.TicTacToeState, mark types.TicTacToeCell) types.SeatKey {
	if mark == types.CellX {
		return state.XSeat
	}
	return oppositeSeat(state.XSeat)
}

var tictactoeLines = [][]types.Pos{
	{{Row: 0, Col: 0}, {Row: 0, Col: 1}, {Row: 0, Col: 2}},
	{{Row: 1, Col: 0}, {Row: 1, Col: 1}, {Row: 1, Col: 2}},
	{{Row: 2, Col: 0}, {Row: 2, Col: 1}, {Row: 2, Col: 2}},
	{{Row: 0, Col: 0}, {Row: 1, Col: 0}, {Row: 2, Col: 0}},
	{{Row: 0, Col: 1}, {Row: 1, Col: 1}, {Row: 2, Col: 1}},
	{{Row: 0, Col: 2}, {Row: 1, Col: 2}, {Row: 2, Col: 2}},
	{{Row: 0, Col: 0}, {Row: 1, Col: 1}, {Row: 2, Col: 2}},
	{{Row: 0, Col: 2}, {Row: 1, Col: 1}, {Row: 2, Col: 0}},
}

func tictactoeWinningLine(board [][]*types.TicTacToeCell) []types.Pos {
	for _, line := range tictactoeLines {
		first := board[line[0].Row][line[0].Col]
		if first == nil {
			continue
		}
		if board[line[1].Row][line[1].Col] != nil && *board[line[1].Row][line[1].Col] == *first &&
			board[line[2].Row][line[2].Col] != nil && *board[line[2].Row][line[2].Col] == *first {
			out := make([]types.Pos, len(line))
			copy(out, line)
			return out
		}
	}
	return nil
}

func tictactoeEmptyCells(board [][]*types.TicTacToeCell) []types.Pos {
	var cells []types.Pos
	for row := 0; row < 3; row++ {
		for col := 0; col < 3; col++ {
			if board[row][col] == nil {
				cells = append(cells, types.Pos{Row: row, Col: col})
			}
		}
	}
	return cells
}

func (s *Server) ticTacToeTurnPlayer(room *RoomState) *PlayerState {
	if room.TicTacToe == nil {
		return nil
	}
	return s.humanPlayerFromSeat(room, room.TicTacToe.Turn)
}

func (s *Server) clearTicTacToeGiveawayTimer(roomID string) {
	if t := s.ticTacToeGiveawayTimers[roomID]; t != nil {
		t.Stop()
		delete(s.ticTacToeGiveawayTimers, roomID)
	}
}

func (s *Server) resetTicTacToeRoom(room *RoomState) {
	s.clearTicTacToeGiveawayTimer(room.ID)
	room.TicTacToe = nil
	room.Phase = types.PhaseReady
	room.Status = "waiting"
	room.ResultText = ""
	room.RevealedChoices = nil
	room.Choices = map[types.SeatKey]types.Move{}
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.Ready = map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false}
}

func (s *Server) startTicTacToeRoom(room *RoomState) {
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	s.clearTicTacToeGiveawayTimer(room.ID)
	xSeat := randomSeat()
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	room.ResultText = ""
	room.Choices = map[types.SeatKey]types.Move{}
	room.RevealedChoices = nil
	room.DisconnectForfeits = map[string]DisconnectForfeit{}
	room.TicTacToe = freshTicTacToeState(xSeat)
	room.Ready = map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false}
}

func (s *Server) scheduleTicTacToeReadyStart(room *RoomState) {
	if room.Settings.GameID != types.GameTicTacToe {
		return
	}
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	if room.Phase != types.PhaseReady {
		return
	}
	if !room.Ready[types.SeatA] || !room.Ready[types.SeatB] {
		return
	}
	if room.ResultText == "正在随机井字棋先手..." {
		return
	}
	room.ResultText = "正在随机井字棋先手..."
	s.broadcastRoom(room.ID, true)
	roomID := room.ID
	timeAfterFunc(1200*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameTicTacToe {
			return
		}
		if current.Phase != types.PhaseReady || current.Seats[types.SeatA] == nil || current.Seats[types.SeatB] == nil ||
			!current.Ready[types.SeatA] || !current.Ready[types.SeatB] {
			return
		}
		s.startTicTacToeRoom(current)
		s.prepareTicTacToeGiveawayPrompt(current)
		xSeat := types.SeatA
		if current.TicTacToe != nil {
			xSeat = current.TicTacToe.XSeat
		}
		s.roomNotice(current, fmt.Sprintf("随机完成：%s 执 X 先手。", occupantName(current.Seats[xSeat])))
		s.broadcastRoom(current.ID, true)
	})
}

func tictactoeRankedText(state *types.TicTacToeState) string {
	if state == nil || state.RankedDelta == nil {
		return ""
	}
	xDelta := state.RankedDelta[state.XSeat]
	oDelta := state.RankedDelta[oppositeSeat(state.XSeat)]
	return fmt.Sprintf("X %s，O %s", formatSigned(xDelta), formatSigned(oDelta))
}

func (s *Server) finishTicTacToeGame(room *RoomState, result types.RoundResult, winningLine []types.Pos) {
	if room.TicTacToe == nil {
		return
	}
	s.clearTicTacToeGiveawayTimer(room.ID)
	playerA := s.humanPlayerFromSeat(room, types.SeatA)
	playerB := s.humanPlayerFromSeat(room, types.SeatB)
	rankedDelta := room.TicTacToe.RankedDelta
	if rankedDelta == nil {
		rankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	rankedText := ""
	streakText := ""
	if result == types.ResultDraw {
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
		if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, effectiveRankedStake(room.Settings))
			rankedDelta[types.SeatA] += dA
			rankedDelta[types.SeatB] += dB
			rankedText = fmt.Sprintf("（平局双扣：A %d，B %d）", dA, dB)
		}
	} else if result == types.ResultA || result == types.ResultB {
		loserSeat := oppositeSeat(types.SeatKey(result))
		winner := s.humanPlayerFromSeat(room, types.SeatKey(result))
		loser := s.humanPlayerFromSeat(room, loserSeat)
		if winner != nil {
			winner.Stats.Wins++
		}
		if loser != nil {
			loser.Stats.Losses++
		}
		room.Score[types.SeatKey(result)]++
		room.SeatedScore[types.SeatKey(result)]++
		ssW := room.SeatStats[types.SeatKey(result)]
		ssW.Wins++
		room.SeatStats[types.SeatKey(result)] = ssW
		ssL := room.SeatStats[loserSeat]
		ssL.Losses++
		room.SeatStats[loserSeat] = ssL
		if room.Settings.EnableRanked {
			wD, lD := s.applyRankedStake(winner, loser, effectiveRankedStake(room.Settings))
			rankedDelta[types.SeatKey(result)] += wD
			rankedDelta[loserSeat] += lD
			resetExtremeWinStreak(loser)
			streakText = s.applyExtremeWinStreakRisk(room, winner)
			rankedText = fmt.Sprintf("（%s %s，%s %d）",
				occupantName(room.Seats[types.SeatKey(result)]), formatSigned(wD),
				occupantName(room.Seats[loserSeat]), lD)
		}
	}
	room.TicTacToe.RankedDelta = rankedDelta
	room.TicTacToe.WinningLine = winningLine
	room.TicTacToe.Ended = true
	room.TicTacToe.Winner = result
	room.Phase = types.PhaseResult
	room.Status = "playing"
	if result == types.ResultDraw {
		room.ResultText = "井字棋平局" + rankedText
	} else {
		room.ResultText = occupantName(room.Seats[types.SeatKey(result)]) + " 井字棋胜利" + rankedText + streakText
	}
	punishedPlayers := s.punishmentPlayersForResult(room, result)
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, result, punishment)
	if playerA != nil {
		s.refreshPlayerSnapshots(playerA)
	}
	if playerB != nil {
		s.refreshPlayerSnapshots(playerB)
	}
	resultLabel := "井字棋平局"
	if result != types.ResultDraw {
		resultLabel = occupantName(room.Seats[types.SeatKey(result)]) + "胜利"
	}
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: occupantName(room.Seats[types.SeatA]), PlayerB: occupantName(room.Seats[types.SeatB]),
		MoveA: types.MoveNoMove, MoveB: types.MoveNoMove, Result: result, ResultLabel: resultLabel,
		ResultText: room.ResultText, GameID: types.GameTicTacToe, TicTacToeXSeat: room.TicTacToe.XSeat,
		TicTacToeLine: winningLine, Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
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
		item.PunishmentName = s.punishmentNameForRoom(room, punishment)
		if room.Settings.PunishmentSource != "player" && punishment != nil {
			item.PunishmentDescription = punishment.Description
		}
	}
	s.addRoundHistory(room, item)
	s.roomNotice(room, room.ResultText)
	s.setupPunishmentOrNext(room, result)
}

func (s *Server) scheduleTicTacToeGiveawayPrompt(room *RoomState) {
	s.clearTicTacToeGiveawayTimer(room.ID)
	if room.TicTacToe == nil || room.TicTacToe.GiveawayPrompt == nil {
		return
	}
	prompt := *room.TicTacToe.GiveawayPrompt
	delay := prompt.ExpiresAt - nowMs()
	if delay < 250 {
		delay = 250
	}
	roomID := room.ID
	startedAt := prompt.StartedAt
	timer := timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.ticTacToeGiveawayTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.TicTacToe == nil || current.TicTacToe.GiveawayPrompt == nil {
			return
		}
		currentPrompt := current.TicTacToe.GiveawayPrompt
		if currentPrompt.StartedAt != startedAt {
			return
		}
		if current.Phase != types.PhaseChoosing || current.TicTacToe.Ended {
			return
		}
		if current.TicTacToe.Turn != currentPrompt.Seat {
			return
		}
		if currentPrompt.Forced {
			forcedPlayer := s.ticTacToeTurnPlayer(current)
			ok, row, col, _ := s.applyTicTacToeRandomMove(current, currentPrompt.Seat, "forcedGiveaway")
			if ok && forcedPlayer != nil {
				s.roomNotice(current, fmt.Sprintf("%s 触发强制白给，系统随机落在第 %d 行第 %d 列。", playerShortName(forcedPlayer), row+1, col+1))
			}
		} else {
			current.TicTacToe.GiveawayPrompt = nil
			current.ResultText = fmt.Sprintf("%s 10 秒未选择，自动不白给。", occupantName(current.Seats[currentPrompt.Seat]))
		}
		s.broadcastRoom(current.ID, true)
	})
	s.ticTacToeGiveawayTimers[room.ID] = timer
}

func (s *Server) prepareTicTacToeGiveawayPrompt(room *RoomState) {
	if room.TicTacToe == nil || room.TicTacToe.Ended || room.Phase != types.PhaseChoosing {
		return
	}
	player := s.ticTacToeTurnPlayer(room)
	seat := room.TicTacToe.Turn
	if player == nil || !ptrBool(player.GiveawayEnabled) || len(tictactoeEmptyCells(room.TicTacToe.Board)) == 0 {
		s.clearTicTacToeGiveawayTimer(room.ID)
		room.TicTacToe.GiveawayPrompt = nil
		return
	}
	forced := s.shouldTriggerGiveaway(player)
	startedAt := nowMs()
	expires := startedAt + 10_000
	if forced {
		expires = startedAt + 2400
	}
	room.TicTacToe.GiveawayPrompt = &types.TicTacToeGiveawayPrompt{
		Seat: seat, Forced: forced, StartedAt: startedAt, ExpiresAt: expires,
	}
	s.scheduleTicTacToeGiveawayPrompt(room)
}

func (s *Server) applyTicTacToeRandomMove(room *RoomState, seat types.SeatKey, mode string) (ok bool, row, col int, errMsg string) {
	if room.TicTacToe == nil {
		return false, 0, 0, "井字棋还没有开始"
	}
	cells := tictactoeEmptyCells(room.TicTacToe.Board)
	if len(cells) == 0 {
		return false, 0, 0, "已经没有空格可以落子"
	}
	cell := cells[rand.Intn(len(cells))]
	player := s.ticTacToeTurnPlayer(room)
	ok2, err2 := s.applyTicTacToeMove(room, seat, cell.Row, cell.Col, mode)
	if ok2 && player != nil {
		s.addGiveawayValue(player, 0.3)
	}
	if !ok2 {
		if err2 == "" {
			err2 = "井字棋白给落子失败"
		}
		return false, 0, 0, err2
	}
	return true, cell.Row, cell.Col, ""
}

func (s *Server) applyTicTacToeMove(room *RoomState, seat types.SeatKey, row, col int, mode string) (bool, string) {
	if room.TicTacToe == nil {
		return false, "井字棋还没有开始"
	}
	if room.Phase != types.PhaseChoosing {
		return false, "当前不能落子"
	}
	if room.TicTacToe.Ended {
		return false, "当前井字棋对局已经结束"
	}
	if room.TicTacToe.Turn != seat {
		return false, "还没轮到你落子"
	}
	prompt := room.TicTacToe.GiveawayPrompt
	if prompt != nil && prompt.Seat == seat {
		if prompt.Forced && mode != "forcedGiveaway" {
			return false, "强制白给中，系统正在随机落子"
		}
		if !prompt.Forced && mode == "normal" {
			return false, "请先选择不白给或白给落子"
		}
		if !prompt.Forced && mode != "normal" && mode != "giveaway" {
			return false, "井字棋白给状态不正确"
		}
	}
	if row < 0 || row >= 3 || col < 0 || col >= 3 {
		return false, "这个位置不能落子"
	}
	if room.TicTacToe.Board[row][col] != nil {
		return false, "这个位置已经有棋子"
	}
	s.clearTicTacToeGiveawayTimer(room.ID)
	mark := tictactoeMarkForSeat(room.TicTacToe, seat)
	board := cloneTicBoard(room.TicTacToe.Board)
	m := mark
	board[row][col] = &m
	moveCount := room.TicTacToe.MoveCount + 1
	winningLine := tictactoeWinningLine(board)
	room.TicTacToe.Board = board
	room.TicTacToe.MoveCount = moveCount
	room.TicTacToe.Turn = oppositeSeat(seat)
	room.TicTacToe.GiveawayPrompt = nil
	if winningLine != nil {
		s.finishTicTacToeGame(room, types.RoundResult(tictactoeSeatForMark(room.TicTacToe, mark)), winningLine)
	} else if moveCount >= 9 {
		s.finishTicTacToeGame(room, types.ResultDraw, nil)
	} else {
		playerName := occupantName(room.Seats[seat])
		if mode == "giveaway" {
			room.ResultText = fmt.Sprintf("%s 选择白给落子，系统随机落在第 %d 行第 %d 列。", playerName, row+1, col+1)
		} else if mode == "forcedGiveaway" {
			room.ResultText = fmt.Sprintf("%s 触发强制白给，系统随机落在第 %d 行第 %d 列。", playerName, row+1, col+1)
		} else {
			room.ResultText = ""
		}
		s.prepareTicTacToeGiveawayPrompt(room)
	}
	return true, ""
}

func cloneTicBoard(board [][]*types.TicTacToeCell) [][]*types.TicTacToeCell {
	out := make([][]*types.TicTacToeCell, len(board))
	for i, row := range board {
		out[i] = make([]*types.TicTacToeCell, len(row))
		for j, cell := range row {
			if cell != nil {
				c := *cell
				out[i][j] = &c
			}
		}
	}
	return out
}

func (s *Server) applyTicTacToeDisconnectForfeit(room *RoomState, forfeit DisconnectForfeit) bool {
	if room.Phase == types.PhaseResult || (room.TicTacToe != nil && room.TicTacToe.Ended) {
		return true
	}
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	rankedDelta := map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	if room.TicTacToe != nil && room.TicTacToe.RankedDelta != nil {
		rankedDelta = room.TicTacToe.RankedDelta
	}
	rankedText := ""
	streakText := ""
	if room.Settings.EnableRanked {
		wD, lD := s.applyRankedStake(winner, loser, forfeit.Stake)
		rankedDelta[forfeit.WinnerSeat] += wD
		rankedDelta[forfeit.LoserSeat] += lD
		resetExtremeWinStreak(loser)
		streakText = s.applyExtremeWinStreakRisk(room, winner)
		rankedText = fmt.Sprintf("，排位 %d 分已结算（%s %s，%s %d）", forfeit.Stake, forfeit.WinnerName, formatSigned(wD), forfeit.LoserName, lD)
	}
	if winner != nil {
		winner.Stats.Wins++
	}
	if loser != nil {
		loser.Stats.Losses++
	}
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
	if room.TicTacToe != nil {
		room.TicTacToe.RankedDelta = rankedDelta
		room.TicTacToe.Ended = true
		room.TicTacToe.Winner = types.RoundResult(forfeit.WinnerSeat)
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s 井字棋胜利%s%s", forfeit.LoserName, forfeit.WinnerName, rankedText, streakText)
	punishedPlayers := s.punishmentPlayersForResult(room, types.RoundResult(forfeit.WinnerSeat))
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, types.RoundResult(forfeit.WinnerSeat), punishment)
	if winner != nil {
		s.refreshPlayerSnapshots(winner)
	}
	if loser != nil {
		s.refreshPlayerSnapshots(loser)
	}
	playerAName, playerBName := forfeit.WinnerName, forfeit.LoserName
	moveA, moveB := types.MoveNoMove, types.MoveNoMove
	if forfeit.LoserSeat == types.SeatA {
		playerAName, playerBName = forfeit.LoserName, forfeit.WinnerName
		moveA = types.MoveForfeit
	} else {
		moveB = types.MoveForfeit
	}
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: playerAName, PlayerB: playerBName, MoveA: moveA, MoveB: moveB,
		Result: types.RoundResult(forfeit.WinnerSeat), ResultLabel: forfeit.WinnerName + "胜利",
		ResultText: room.ResultText + "（断线判负）", GameID: types.GameTicTacToe,
		Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
	}
	if room.TicTacToe != nil {
		item.TicTacToeXSeat = room.TicTacToe.XSeat
	}
	if room.Settings.EnableRanked {
		st := forfeit.BaseStake
		item.Stake = &st
		rm := forfeit.RankMultiplier
		item.RankMultiplier = &rm
		es := forfeit.Stake
		item.EffectiveStake = &es
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentNameForRoom(room, punishment)
		if room.Settings.PunishmentSource != "player" && punishment != nil {
			item.PunishmentDescription = punishment.Description
		}
	}
	s.addRoundHistory(room, item)
	s.roomNotice(room, room.ResultText)
	s.setupPunishmentOrNext(room, types.RoundResult(forfeit.WinnerSeat))
	return true
}
