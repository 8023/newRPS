package server

import (
	"fmt"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

const (
	gomokuBoardSize  = 15
	gomokuWinLength  = 5
	gomokuUndoWindow = 30 * time.Second
)

func initialGomokuBoard() [][]*types.GomokuCell {
	board := make([][]*types.GomokuCell, gomokuBoardSize)
	for i := 0; i < gomokuBoardSize; i++ {
		board[i] = make([]*types.GomokuCell, gomokuBoardSize)
	}
	return board
}

func freshGomokuState(blackSeat types.SeatKey) *types.GomokuState {
	return &types.GomokuState{
		Board:       initialGomokuBoard(),
		Turn:        blackSeat,
		BlackSeat:   blackSeat,
		MoveCount:   0,
		Moves:       []types.Pos{},
		RankedDelta: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		UndoCount:   map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
	}
}

func gomokuMarkForSeat(state *types.GomokuState, seat types.SeatKey) types.GomokuCell {
	if state.BlackSeat == seat {
		return types.GomokuBlack
	}
	return types.GomokuWhite
}

func gomokuSeatForMark(state *types.GomokuState, mark types.GomokuCell) types.SeatKey {
	if mark == types.GomokuBlack {
		return state.BlackSeat
	}
	return oppositeSeat(state.BlackSeat)
}

func gomokuMarkName(mark types.GomokuCell) string {
	if mark == types.GomokuBlack {
		return "黑"
	}
	return "白"
}

func (s *Server) chooseGomokuGiveaway(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Gomoku == nil || room.Phase != types.PhaseChoosing || room.Gomoku.Ended {
		return false, "当前不能选择白给"
	}
	if room.Gomoku.Turn != seat {
		return false, "还没轮到你选择白给"
	}
	if room.Gomoku.UndoRequest != nil || room.Gomoku.ResignRequest != nil {
		return false, "请求处理完成前不能选择白给"
	}
	player := s.humanPlayerFromSeat(room, seat)
	if player == nil || !ptrBool(player.GiveawayEnabled) || !s.isFullHumanRoom(room) {
		return false, "请先开启白给模式并等待双方入座"
	}
	if room.Gomoku.GiveawaySeat == seat {
		return false, "本手已经进入白给状态"
	}
	room.Gomoku.GiveawaySeat = seat
	// 主人先强制、宠物随后主动点击白给：直到这次点击才公开本手状态，
	// 并同时消费隐藏的“本步必白给”标记，避免残留到宠物下一手。
	room.Gomoku.GiveawayForcedByMasterName = takeForcedGiveaway(room, seat)
	player.GiveawayClicks = intPtr(ptrInt(player.GiveawayClicks) + 1)
	s.addGiveawayValue(player, s.cfg.Giveaway.ActiveBoostValue)
	s.refreshPlayerSnapshots(player)
	if room.GiveawayBoostedBySeat == nil {
		room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	}
	room.GiveawayBoostedBySeat[seat] = true
	if masterName := room.Gomoku.GiveawayForcedByMasterName; masterName != "" {
		room.ResultText = fmt.Sprintf("主人（%s）强制（%s）本手白给，请选择落点。", masterName, occupantName(room.Seats[seat]))
	} else {
		room.ResultText = occupantName(room.Seats[seat]) + " 选择本手白给，请选择落点。"
	}
	return true, ""
}

var gomokuDirections = [4][2]int{{0, 1}, {1, 0}, {1, 1}, {1, -1}}

// gomokuWinningLine 从刚落子的位置沿四个方向扫描，找到 >=5 连子即返回整条连线。
func gomokuWinningLine(board [][]*types.GomokuCell, row, col int) []types.Pos {
	cell := board[row][col]
	if cell == nil {
		return nil
	}
	for _, dir := range gomokuDirections {
		var forward []types.Pos
		r, c := row+dir[0], col+dir[1]
		for r >= 0 && r < gomokuBoardSize && c >= 0 && c < gomokuBoardSize && board[r][c] != nil && *board[r][c] == *cell {
			forward = append(forward, types.Pos{Row: r, Col: c})
			r += dir[0]
			c += dir[1]
		}
		var backward []types.Pos
		r, c = row-dir[0], col-dir[1]
		for r >= 0 && r < gomokuBoardSize && c >= 0 && c < gomokuBoardSize && board[r][c] != nil && *board[r][c] == *cell {
			backward = append(backward, types.Pos{Row: r, Col: c})
			r -= dir[0]
			c -= dir[1]
		}
		total := len(forward) + len(backward) + 1
		if total >= gomokuWinLength {
			line := make([]types.Pos, 0, total)
			for i := len(backward) - 1; i >= 0; i-- {
				line = append(line, backward[i])
			}
			line = append(line, types.Pos{Row: row, Col: col})
			line = append(line, forward...)
			return line
		}
	}
	return nil
}

func (s *Server) clearGomokuUndoTimer(roomID string) {
	if t := s.gomokuUndoTimers[roomID]; t != nil {
		t.Stop()
		delete(s.gomokuUndoTimers, roomID)
	}
}

func (s *Server) resetGomokuRoom(room *RoomState) {
	s.clearGomokuUndoTimer(room.ID)
	s.clearGomokuClockTimer(room.ID)
	room.Gomoku = nil
	s.resetTurnBasedRoom(room)
}

func (s *Server) startGomokuRoom(room *RoomState) {
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	s.clearGomokuUndoTimer(room.ID)
	blackSeat := randomSeat()
	s.startTurnBasedPlaying(room)
	room.Gomoku = freshGomokuState(blackSeat)
	s.armGomokuTimers(room, blackSeat)
}

// ── 每子时长 / 每局时长（超时判负）──────────────────────

func (s *Server) clearGomokuClockTimer(roomID string) {
	if t := s.gomokuClockTimers[roomID]; t != nil {
		t.Stop()
		delete(s.gomokuClockTimers, roomID)
	}
}

// freezeGomokuClock 把 seat 当前正在走动的总时长时钟冻结为静态剩余值（毫秒）。
func (s *Server) freezeGomokuClock(room *RoomState, seat types.SeatKey) {
	if room.Gomoku == nil || room.Gomoku.ClockRemaining == nil {
		return
	}
	if room.Gomoku.ClockDeadlineAt > 0 {
		remaining := room.Gomoku.ClockDeadlineAt - nowMs()
		if remaining < 0 {
			remaining = 0
		}
		room.Gomoku.ClockRemaining[seat] = remaining
	}
	room.Gomoku.ClockDeadlineAt = 0
}

// pauseGomokuTimers：悔棋/认输请求处理完成前不能落子，期间暂停两个计时器，避免用等待对方
// 回应的时间把自己计时计没了；respondGomokuUndo/respondGomokuResign 处理完后重新计时。
func (s *Server) pauseGomokuTimers(room *RoomState) {
	if room.Gomoku == nil {
		return
	}
	s.freezeGomokuClock(room, room.Gomoku.Turn)
	room.Gomoku.MoveDeadlineAt = 0
	s.clearGomokuClockTimer(room.ID)
}

func (s *Server) resumeGomokuTimers(room *RoomState) {
	if room.Gomoku == nil || room.Gomoku.Ended {
		return
	}
	s.armGomokuTimers(room, room.Gomoku.Turn)
}

// armGomokuTimers 在真正轮到 seat 落子时重新起算每子倒计时/总时长时钟，并重排服务端超时检测。
func (s *Server) armGomokuTimers(room *RoomState, seat types.SeatKey) {
	if room.Gomoku == nil {
		return
	}
	if room.Settings.GomokuMoveSeconds > 0 {
		room.Gomoku.MoveDeadlineAt = nowMs() + int64(room.Settings.GomokuMoveSeconds)*1000
	} else {
		room.Gomoku.MoveDeadlineAt = 0
	}
	if room.Settings.GomokuGameMinutes > 0 {
		if room.Gomoku.ClockRemaining == nil {
			total := int64(room.Settings.GomokuGameMinutes) * 60_000
			room.Gomoku.ClockRemaining = map[types.SeatKey]int64{types.SeatA: total, types.SeatB: total}
		}
		room.Gomoku.ClockDeadlineAt = nowMs() + room.Gomoku.ClockRemaining[seat]
	} else {
		room.Gomoku.ClockDeadlineAt = 0
	}
	s.scheduleGomokuClockTimer(room)
}

func (s *Server) scheduleGomokuClockTimer(room *RoomState) {
	s.clearGomokuClockTimer(room.ID)
	if room.Gomoku == nil || room.Gomoku.Ended {
		return
	}
	deadline := earliestPositiveDeadline(room.Gomoku.MoveDeadlineAt, room.Gomoku.ClockDeadlineAt)
	if deadline == 0 {
		return
	}
	delay := deadline - nowMs()
	if delay < 0 {
		delay = 0
	}
	seat := room.Gomoku.Turn
	roomID := room.ID
	timer := timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.gomokuClockTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.Gomoku == nil || current.Gomoku.Ended {
			return
		}
		if current.Gomoku.Turn != seat || current.Gomoku.UndoRequest != nil || current.Gomoku.ResignRequest != nil {
			return
		}
		now := nowMs()
		moveExpired := current.Gomoku.MoveDeadlineAt > 0 && now >= current.Gomoku.MoveDeadlineAt
		clockExpired := current.Gomoku.ClockDeadlineAt > 0 && now >= current.Gomoku.ClockDeadlineAt
		if !moveExpired && !clockExpired {
			s.scheduleGomokuClockTimer(current)
			return
		}
		winnerSeat := oppositeSeat(seat)
		loserName := occupantName(current.Seats[seat])
		s.roomNotice(current, loserName+" 用时已到，本局判负。")
		s.finishGomokuGame(current, types.RoundResult(winnerSeat), nil, "超时判负")
		s.broadcastRoom(current.ID, true)
	})
	s.gomokuClockTimers[room.ID] = timer
}

func (s *Server) scheduleGomokuReadyStart(room *RoomState) {
	if room.Settings.GameID != types.GameGomoku {
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
	if room.ResultText == "正在随机五子棋先手..." {
		return
	}
	room.ResultText = "正在随机五子棋先手..."
	s.broadcastRoom(room.ID, true)
	roomID := room.ID
	timeAfterFunc(1200*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameGomoku {
			return
		}
		if current.Phase != types.PhaseReady || current.Seats[types.SeatA] == nil || current.Seats[types.SeatB] == nil ||
			!current.Ready[types.SeatA] || !current.Ready[types.SeatB] {
			return
		}
		s.startGomokuRoom(current)
		blackSeat := types.SeatA
		if current.Gomoku != nil {
			blackSeat = current.Gomoku.BlackSeat
		}
		s.roomNotice(current, fmt.Sprintf("随机完成：%s 执黑先手。", occupantName(current.Seats[blackSeat])))
		s.broadcastRoom(current.ID, true)
	})
}

// finishGomokuGame 结束对局；note 非空时会作为「（认输）」这类后缀追加到结果文案与历史记录。
func (s *Server) finishGomokuGame(room *RoomState, result types.RoundResult, winningLine []types.Pos, note string) {
	if room.Gomoku == nil {
		return
	}
	s.clearGomokuUndoTimer(room.ID)
	s.clearGomokuClockTimer(room.ID)
	room.Gomoku.UndoRequest = nil
	room.Gomoku.GiveawaySeat = ""
	room.Gomoku.GiveawayForcedByMasterName = ""
	room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
	room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
	room.Gomoku.ResignRequest = nil
	rankedDelta := room.Gomoku.RankedDelta
	if rankedDelta == nil {
		rankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	rankedText := ""
	streakText := ""
	playerA, playerB := s.applySeatOutcome(room, result)
	if result == types.ResultDraw {
		if room.Settings.EnableRanked && room.Settings.TieDoublePunish {
			dA, dB := s.applyRankedDrawPenaltyStake(playerA, playerB, effectiveRankedStake(room.Settings))
			rankedDelta[types.SeatA] += dA
			rankedDelta[types.SeatB] += dB
			rankedText = fmt.Sprintf("（平局双扣：A %d，B %d）", dA, dB)
		}
	} else if result == types.ResultA || result == types.ResultB {
		if room.Settings.EnableRanked {
			loserSeat := oppositeSeat(types.SeatKey(result))
			winner := s.humanPlayerFromSeat(room, types.SeatKey(result))
			loser := s.humanPlayerFromSeat(room, loserSeat)
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
	room.Gomoku.RankedDelta = rankedDelta
	room.Gomoku.WinningLine = winningLine
	room.Gomoku.Ended = true
	room.Gomoku.Winner = result
	room.Phase = types.PhaseResult
	room.Status = "playing"
	noteSuffix := ""
	if note != "" {
		noteSuffix = "（" + note + "）"
	}
	if result == types.ResultDraw {
		room.ResultText = "五子棋平局" + rankedText + noteSuffix
	} else {
		room.ResultText = occupantName(room.Seats[types.SeatKey(result)]) + " 五子棋胜利" + rankedText + streakText + noteSuffix
	}
	s.refreshHumans(playerA, playerB)
	resultLabel := "五子棋平局"
	if result != types.ResultDraw {
		resultLabel = occupantName(room.Seats[types.SeatKey(result)]) + "胜利"
	}
	item := s.buildMatchHistoryShell(room, result, types.GameGomoku, resultLabel, room.ResultText)
	item.GomokuBlackSeat = room.Gomoku.BlackSeat
	item.GomokuLine = winningLine
	item.ExtremeRanked = room.Settings.EnableExtremeRanked
	if room.Settings.EnableRanked {
		es := effectiveRankedStake(room.Settings)
		item.EffectiveStake = &es
	}
	s.roomNotice(room, room.ResultText)
	s.finalizeMatch(room, result, item)
}

func (s *Server) applyGomokuMove(room *RoomState, seat types.SeatKey, row, col int) (bool, string) {
	if room.Gomoku == nil {
		return false, "五子棋还没有开始"
	}
	if room.Phase != types.PhaseChoosing {
		return false, "当前不能落子"
	}
	if room.Gomoku.Ended {
		return false, "当前五子棋对局已经结束"
	}
	if room.Gomoku.Turn != seat {
		return false, "还没轮到你落子"
	}
	if room.Gomoku.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能落子"
	}
	if room.Gomoku.ResignRequest != nil {
		return false, "认输请求处理完成前不能落子"
	}
	if row < 0 || row >= gomokuBoardSize || col < 0 || col >= gomokuBoardSize {
		return false, "这个位置不能落子"
	}
	if room.Gomoku.Board[row][col] != nil {
		return false, "这个位置已经有棋子"
	}

	player := s.humanPlayerFromSeat(room, seat)
	probability := 0.0
	if player != nil {
		probability = ptrFloat(player.GiveawayValue)
	}
	armed := room.Gomoku.GiveawaySeat == seat
	forcedByMaster := room.Gomoku.GiveawayForcedByMasterName
	if forcedByMaster == "" {
		// 情况 5：主人强制但宠物没有点白给。标记只在落子时消费，
		// 因而落子前不会向宠物暴露，本步则必定按白给结算。
		forcedByMaster = takeForcedGiveaway(room, seat)
	}
	naturalGiveaway := !armed && forcedByMaster == "" && s.shouldTriggerGiveaway(player)
	giveaway := armed || forcedByMaster != "" || naturalGiveaway
	mark := gomokuMarkForSeat(room.Gomoku, seat)
	if giveaway {
		mark = gomokuMarkForSeat(room.Gomoku, oppositeSeat(seat))
	}

	boosted := room.GiveawayBoostedBySeat != nil && room.GiveawayBoostedBySeat[seat]
	if giveaway && !boosted && player != nil {
		s.addGiveawayValue(player, s.cfg.Giveaway.ActiveBoostValue)
		s.refreshPlayerSnapshots(player)
	}
	if room.GiveawayBoostedBySeat != nil {
		delete(room.GiveawayBoostedBySeat, seat)
	}
	room.Gomoku.GiveawaySeat = ""
	room.Gomoku.GiveawayForcedByMasterName = ""

	giveawayText := ""
	if forcedByMaster != "" {
		giveawayText = fmt.Sprintf("主人（%s）强制（%s）白给，第 %d 行第 %d 列落为%s棋。", forcedByMaster, occupantName(room.Seats[seat]), row+1, col+1, gomokuMarkName(mark))
	} else if armed {
		giveawayText = fmt.Sprintf("%s 本手白给，第 %d 行第 %d 列落为对方的%s棋。", occupantName(room.Seats[seat]), row+1, col+1, gomokuMarkName(mark))
	} else if naturalGiveaway {
		giveawayText = fmt.Sprintf("%s 按 %.1f%% 白给值触发白给，第 %d 行第 %d 列变为对方的%s棋。", occupantName(room.Seats[seat]), probability, row+1, col+1, gomokuMarkName(mark))
	}

	s.pauseGomokuTimers(room)
	board := cloneGomokuBoard(room.Gomoku.Board)
	m := mark
	board[row][col] = &m
	moveCount := room.Gomoku.MoveCount + 1
	winningLine := gomokuWinningLine(board, row, col)
	room.Gomoku.Board = board
	room.Gomoku.MoveCount = moveCount
	room.Gomoku.Moves = append(room.Gomoku.Moves, types.Pos{Row: row, Col: col})
	room.Gomoku.Turn = oppositeSeat(seat)
	if winningLine != nil {
		s.finishGomokuGame(room, types.RoundResult(gomokuSeatForMark(room.Gomoku, mark)), winningLine, giveawayText)
	} else if moveCount >= gomokuBoardSize*gomokuBoardSize {
		s.finishGomokuGame(room, types.ResultDraw, nil, giveawayText)
	} else {
		room.ResultText = giveawayText
		if giveawayText != "" {
			s.roomNotice(room, giveawayText)
		}
		s.armGomokuTimers(room, room.Gomoku.Turn)
		s.notifyOpponentTurn(room, room.Gomoku.Turn)
	}
	return true, ""
}

func cloneGomokuBoard(board [][]*types.GomokuCell) [][]*types.GomokuCell {
	out := make([][]*types.GomokuCell, len(board))
	for i, row := range board {
		out[i] = make([]*types.GomokuCell, len(row))
		for j, cell := range row {
			if cell != nil {
				c := *cell
				out[i][j] = &c
			}
		}
	}
	return out
}

// requestGomokuUndo：只能由「当前轮到落子的一方」发起，悔回自己上一手 + 对方紧接着的应手，
// 每局的悔棋次数上限由 room.Settings.GomokuUndoLimit 决定（建房时校验为 0/1/3/10），30 秒无响应自动拒绝。
func (s *Server) requestGomokuUndo(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Gomoku == nil || room.Phase != types.PhaseChoosing || room.Gomoku.Ended {
		return false, "当前不能悔棋"
	}
	if room.Gomoku.Turn != seat {
		return false, "只能在轮到你落子时申请悔棋"
	}
	if room.Gomoku.MoveCount < 2 {
		return false, "棋局刚开始，还不能悔棋"
	}
	if room.Gomoku.UndoRequest != nil {
		return false, "当前已有悔棋请求，请先处理"
	}
	if room.Gomoku.ResignRequest != nil {
		return false, "认输请求处理完成前不能悔棋"
	}
	if room.Gomoku.UndoCount[seat] >= room.Settings.GomokuUndoLimit {
		return false, "你本局的悔棋次数已经用完"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能悔棋"
	}
	startedAt := nowMs()
	room.Gomoku.UndoRequest = &types.GomokuUndoRequest{
		FromSeat: seat, ToSeat: toSeat, CreatedAt: startedAt, ExpiresAt: startedAt + gomokuUndoWindow.Milliseconds(),
	}
	s.pauseGomokuTimers(room)
	s.scheduleGomokuUndoTimeout(room)
	return true, ""
}

func (s *Server) scheduleGomokuUndoTimeout(room *RoomState) {
	s.clearGomokuUndoTimer(room.ID)
	if room.Gomoku == nil || room.Gomoku.UndoRequest == nil {
		return
	}
	request := *room.Gomoku.UndoRequest
	roomID := room.ID
	timer := timeAfterFunc(gomokuUndoWindow, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.gomokuUndoTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.Gomoku == nil || current.Gomoku.UndoRequest == nil {
			return
		}
		if current.Gomoku.UndoRequest.CreatedAt != request.CreatedAt {
			return
		}
		current.Gomoku.UndoRequest = nil
		s.resumeGomokuTimers(current)
		s.roomNotice(current, fmt.Sprintf("%s 未在 30 秒内回应悔棋请求，已自动拒绝。", occupantName(current.Seats[request.ToSeat])))
		s.broadcastRoom(current.ID, true)
	})
	s.gomokuUndoTimers[room.ID] = timer
}

// respondGomokuUndo：accept 时回退最后 2 手（请求方自己的上一手 + 对方紧接着的应手），
// 落子权回到请求方；轮次校验依赖 undo 只能在请求方回合发起，回退 2 手后奇偶不变，轮次天然保持一致。
func (s *Server) respondGomokuUndo(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Gomoku == nil || room.Gomoku.UndoRequest == nil {
		return false, "当前没有悔棋请求"
	}
	request := room.Gomoku.UndoRequest
	if request.ToSeat != seat {
		return false, "这个悔棋请求不是发给你的"
	}
	s.clearGomokuUndoTimer(room.ID)
	fromSeat := request.FromSeat
	room.Gomoku.UndoRequest = nil
	if !accept {
		s.resumeGomokuTimers(room)
		s.roomNotice(room, occupantName(room.Seats[seat])+" 拒绝悔棋，对局继续。")
		return true, ""
	}
	moves := room.Gomoku.Moves
	if len(moves) < 2 {
		s.resumeGomokuTimers(room)
		return false, "当前棋局状态不支持悔棋"
	}
	board := cloneGomokuBoard(room.Gomoku.Board)
	for i := 0; i < 2; i++ {
		last := moves[len(moves)-1]
		moves = moves[:len(moves)-1]
		board[last.Row][last.Col] = nil
	}
	room.Gomoku.Board = board
	room.Gomoku.Moves = moves
	room.Gomoku.MoveCount = len(moves)
	room.Gomoku.Turn = fromSeat
	room.Gomoku.UndoCount[fromSeat]++
	s.resumeGomokuTimers(room)
	s.roomNotice(room, occupantName(room.Seats[seat])+" 同意悔棋，棋局回退 2 手。")
	return true, ""
}

// requestGomokuResign / respondGomokuResign：认输需要对方确认才结束对局（与黑白棋认输一致）。
func (s *Server) requestGomokuResign(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Gomoku == nil || room.Phase != types.PhaseChoosing || room.Gomoku.Ended {
		return false, "当前不能申请认输"
	}
	if room.Gomoku.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能申请认输"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能申请认输"
	}
	if room.Gomoku.ResignRequest != nil && room.Gomoku.ResignRequest.FromSeat == seat {
		return false, "你已经申请认输，正在等待对方确认"
	}
	if room.Gomoku.ResignRequest != nil {
		return false, "当前已有认输请求，请先处理"
	}
	room.Gomoku.ResignRequest = &types.GomokuResignRequest{FromSeat: seat, ToSeat: toSeat, CreatedAt: nowMs()}
	s.pauseGomokuTimers(room)
	return true, ""
}

func (s *Server) respondGomokuResign(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Gomoku == nil || room.Gomoku.ResignRequest == nil {
		return false, "当前没有认输请求"
	}
	request := room.Gomoku.ResignRequest
	if request.ToSeat != seat {
		return false, "这个认输请求不是发给你的"
	}
	loserSeat := request.FromSeat
	winnerSeat := request.ToSeat
	loserName := occupantName(room.Seats[loserSeat])
	winnerName := occupantName(room.Seats[winnerSeat])
	if !accept {
		room.Gomoku.ResignRequest = nil
		s.resumeGomokuTimers(room)
		s.roomNotice(room, winnerName+" 拒绝认输，对局继续。")
		return true, ""
	}
	s.roomNotice(room, fmt.Sprintf("%s 同意 %s 认输，本局结束。", winnerName, loserName))
	s.finishGomokuGame(room, types.RoundResult(winnerSeat), nil, "认输")
	return true, ""
}

func (s *Server) applyGomokuDisconnectForfeit(room *RoomState, forfeit DisconnectForfeit) bool {
	if room.Phase == types.PhaseResult || (room.Gomoku != nil && room.Gomoku.Ended) {
		return true
	}
	s.clearGomokuClockTimer(room.ID)
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	rankedDelta := map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	if room.Gomoku != nil && room.Gomoku.RankedDelta != nil {
		rankedDelta = room.Gomoku.RankedDelta
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
	recordGameOutcome(winner, types.GameGomoku, "win")
	recordGameOutcome(loser, types.GameGomoku, "loss")
	s.applyGiveawayWinPenalty(winner)
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
	if room.Gomoku != nil {
		s.clearGomokuUndoTimer(room.ID)
		room.Gomoku.UndoRequest = nil
		room.Gomoku.ResignRequest = nil
		room.Gomoku.GiveawaySeat = ""
		room.Gomoku.GiveawayForcedByMasterName = ""
		room.ForcedGiveawayBySeat = map[types.SeatKey]string{}
		room.GiveawayBoostedBySeat = map[types.SeatKey]bool{}
		room.Gomoku.RankedDelta = rankedDelta
		room.Gomoku.Ended = true
		room.Gomoku.Winner = types.RoundResult(forfeit.WinnerSeat)
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s 五子棋胜利%s%s", forfeit.LoserName, forfeit.WinnerName, rankedText, streakText)
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
		ResultText: room.ResultText + "（断线判负）", GameID: types.GameGomoku,
		Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
	}
	if room.Gomoku != nil {
		item.GomokuBlackSeat = room.Gomoku.BlackSeat
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
