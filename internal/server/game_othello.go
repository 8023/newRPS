package server

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

var othelloDirections = [][2]int{
	{-1, -1}, {-1, 0}, {-1, 1},
	{0, -1}, {0, 1},
	{1, -1}, {1, 0}, {1, 1},
}

func initialOthelloBoard() [][]*types.OthelloCell {
	board := make([][]*types.OthelloCell, 8)
	for i := 0; i < 8; i++ {
		board[i] = make([]*types.OthelloCell, 8)
	}
	w, b := types.CellWhite, types.CellBlack
	board[3][3] = &w
	board[3][4] = &b
	board[4][3] = &b
	board[4][4] = &w
	return board
}

func othelloColorForSeat(state *types.OthelloState, seat types.SeatKey) types.OthelloCell {
	if state.BlackSeat == seat {
		return types.CellBlack
	}
	return types.CellWhite
}

func oppositeOthelloColor(color types.OthelloCell) types.OthelloCell {
	if color == types.CellBlack {
		return types.CellWhite
	}
	return types.CellBlack
}

func othelloSeatForColor(state *types.OthelloState, color types.OthelloCell) types.SeatKey {
	if color == types.CellBlack {
		return state.BlackSeat
	}
	return oppositeSeat(state.BlackSeat)
}

func inOthelloBoard(row, col int) bool {
	return row >= 0 && row < 8 && col >= 0 && col < 8
}

func othelloFlips(board [][]*types.OthelloCell, row, col int, color types.OthelloCell) []types.Pos {
	if !inOthelloBoard(row, col) || board[row][col] != nil {
		return nil
	}
	opponent := oppositeOthelloColor(color)
	var flips []types.Pos
	for _, d := range othelloDirections {
		var line []types.Pos
		r, c := row+d[0], col+d[1]
		for inOthelloBoard(r, c) && board[r][c] != nil && *board[r][c] == opponent {
			line = append(line, types.Pos{Row: r, Col: c})
			r += d[0]
			c += d[1]
		}
		if len(line) > 0 && inOthelloBoard(r, c) && board[r][c] != nil && *board[r][c] == color {
			flips = append(flips, line...)
		}
	}
	return flips
}

func othelloLegalMoves(board [][]*types.OthelloCell, color types.OthelloCell) []types.Pos {
	var moves []types.Pos
	for row := 0; row < 8; row++ {
		for col := 0; col < 8; col++ {
			if len(othelloFlips(board, row, col, color)) > 0 {
				moves = append(moves, types.Pos{Row: row, Col: col})
			}
		}
	}
	return moves
}

func othelloCounts(board [][]*types.OthelloCell) (blackCount, whiteCount int) {
	for _, row := range board {
		for _, cell := range row {
			if cell == nil {
				continue
			}
			if *cell == types.CellBlack {
				blackCount++
			}
			if *cell == types.CellWhite {
				whiteCount++
			}
		}
	}
	return
}

func othelloRankedText(state *types.OthelloState) string {
	if state == nil || state.RankedDelta == nil {
		return ""
	}
	blackDelta := state.RankedDelta[state.BlackSeat]
	whiteDelta := state.RankedDelta[oppositeSeat(state.BlackSeat)]
	return fmt.Sprintf("黑棋 %s，白棋 %s", formatSigned(blackDelta), formatSigned(whiteDelta))
}

// othelloSettlementSummary 终局只展示统计摘要，不再把每一手明细拼进 ResultText。
func othelloSettlementSummary(state *types.OthelloState) string {
	if state == nil || len(state.SettlementEvents) == 0 {
		return ""
	}
	var giveawayHands, giveawayFlips, tributeHands, forcedHands int
	for _, e := range state.SettlementEvents {
		switch {
		case strings.Contains(e, "白给"):
			giveawayHands++
			giveawayFlips += extractLeadingIntBefore(e, "子")
			if strings.Contains(e, "强制") {
				forcedHands++
			}
		case strings.Contains(e, "上贡"):
			tributeHands++
		}
	}
	if giveawayHands == 0 && tributeHands == 0 {
		return ""
	}
	var parts []string
	if giveawayHands > 0 {
		if giveawayFlips > 0 {
			parts = append(parts, fmt.Sprintf("白给 %d 手（共 %d 子不计分）", giveawayHands, giveawayFlips))
		} else {
			parts = append(parts, fmt.Sprintf("白给 %d 手", giveawayHands))
		}
	}
	if tributeHands > 0 {
		parts = append(parts, fmt.Sprintf("上贡 %d 手", tributeHands))
	}
	if forcedHands > 0 {
		parts = append(parts, fmt.Sprintf("其中强制 %d 手", forcedHands))
	}
	return "；本局白给/上贡：" + joinChinese(parts)
}

// extractLeadingIntBefore 从「…，N 子…」一类文案里取子数，解析失败返回 0。
func extractLeadingIntBefore(text, marker string) int {
	idx := strings.Index(text, marker)
	if idx <= 0 {
		return 0
	}
	// 向前扫描数字
	end := idx
	for end > 0 && text[end-1] == ' ' {
		end--
	}
	start := end
	for start > 0 && text[start-1] >= '0' && text[start-1] <= '9' {
		start--
	}
	if start == end {
		return 0
	}
	n := 0
	for i := start; i < end; i++ {
		n = n*10 + int(text[i]-'0')
	}
	return n
}

func joinChinese(parts []string) string {
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "；"
		}
		out += p
	}
	return out
}

func (s *Server) freshOthelloState(blackSeat types.SeatKey) *types.OthelloState {
	board := initialOthelloBoard()
	bc, wc := othelloCounts(board)
	return &types.OthelloState{
		Board:            board,
		Turn:             blackSeat,
		BlackSeat:        blackSeat,
		LegalMoves:       othelloLegalMoves(board, types.CellBlack),
		PassCount:        0,
		BlackCount:       bc,
		WhiteCount:       wc,
		RankedDelta:      map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		SettlementEvents: []string{},
	}
}

func (s *Server) clearOthelloSettlementTimer(roomID string) {
	if t := s.othelloSettlementTimers[roomID]; t != nil {
		t.Stop()
		delete(s.othelloSettlementTimers, roomID)
	}
}

func (s *Server) resetOthelloRoom(room *RoomState) {
	// 和 forceEndOthelloGame/applyOthelloDisconnectForfeit 一样，重置前必须先把还没定
	// 型的白给/上贡结算 flush 掉，否则那笔待结算的排位分/回合记录会被下面的置空直接吞掉。
	s.flushOthelloPendingSettlement(room)
	s.clearOthelloSettlementTimer(room.ID)
	s.clearOthelloClockTimer(room.ID)
	room.Othello = nil
	s.resetTurnBasedRoom(room)
}

func (s *Server) startOthelloRoom(room *RoomState) {
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	blackSeat := randomSeat()
	s.startTurnBasedPlaying(room)
	room.Othello = s.freshOthelloState(blackSeat)
	s.armOthelloTimers(room, blackSeat)
}

// ── 每子时长 / 每局时长（超时判负）──────────────────────

func (s *Server) clearOthelloClockTimer(roomID string) {
	if t := s.othelloClockTimers[roomID]; t != nil {
		t.Stop()
		delete(s.othelloClockTimers, roomID)
	}
}

// freezeOthelloClock 把 seat 当前正在走动的总时长时钟冻结为静态剩余值（毫秒），供离场/换手时调用。
func (s *Server) freezeOthelloClock(room *RoomState, seat types.SeatKey) {
	if room.Othello == nil || room.Othello.ClockRemaining == nil {
		return
	}
	if room.Othello.ClockDeadlineAt > 0 {
		remaining := room.Othello.ClockDeadlineAt - nowMs()
		if remaining < 0 {
			remaining = 0
		}
		room.Othello.ClockRemaining[seat] = remaining
	}
	room.Othello.ClockDeadlineAt = 0
}

// pauseOthelloTimers：进入白给/上贡结算等待窗口时暂停两个计时器——该窗口已有自己独立的超时机制
// （scheduleOthelloSettlement），避免和每子/每局计时器重复判负。
func (s *Server) pauseOthelloTimers(room *RoomState, seat types.SeatKey) {
	if room.Othello == nil {
		return
	}
	s.freezeOthelloClock(room, seat)
	room.Othello.MoveDeadlineAt = 0
	s.clearOthelloClockTimer(room.ID)
}

// armOthelloTimers 在真正轮到 seat 落子时重新起算每子倒计时/总时长时钟，并重排服务端超时检测。
func (s *Server) armOthelloTimers(room *RoomState, seat types.SeatKey) {
	if room.Othello == nil {
		return
	}
	if room.Settings.OthelloMoveSeconds > 0 {
		room.Othello.MoveDeadlineAt = nowMs() + int64(room.Settings.OthelloMoveSeconds)*1000
	} else {
		room.Othello.MoveDeadlineAt = 0
	}
	if room.Settings.OthelloGameMinutes > 0 {
		if room.Othello.ClockRemaining == nil {
			total := int64(room.Settings.OthelloGameMinutes) * 60_000
			room.Othello.ClockRemaining = map[types.SeatKey]int64{types.SeatA: total, types.SeatB: total}
		}
		room.Othello.ClockDeadlineAt = nowMs() + room.Othello.ClockRemaining[seat]
	} else {
		room.Othello.ClockDeadlineAt = 0
	}
	s.scheduleOthelloClockTimer(room)
}

func (s *Server) scheduleOthelloClockTimer(room *RoomState) {
	s.clearOthelloClockTimer(room.ID)
	if room.Othello == nil || room.Othello.Ended {
		return
	}
	deadline := earliestPositiveDeadline(room.Othello.MoveDeadlineAt, room.Othello.ClockDeadlineAt)
	if deadline == 0 {
		return
	}
	delay := deadline - nowMs()
	if delay < 0 {
		delay = 0
	}
	seat := room.Othello.Turn
	roomID := room.ID
	timer := timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		delete(s.othelloClockTimers, roomID)
		current := s.rooms[roomID]
		if current == nil || current.Othello == nil || current.Othello.Ended {
			return
		}
		if current.Othello.Turn != seat || current.Othello.PendingSettlement != nil {
			return
		}
		now := nowMs()
		moveExpired := current.Othello.MoveDeadlineAt > 0 && now >= current.Othello.MoveDeadlineAt
		clockExpired := current.Othello.ClockDeadlineAt > 0 && now >= current.Othello.ClockDeadlineAt
		if !moveExpired && !clockExpired {
			s.scheduleOthelloClockTimer(current)
			return
		}
		s.forfeitOthelloTimeout(current, seat)
		s.broadcastRoom(current.ID, true)
	})
	s.othelloClockTimers[room.ID] = timer
}

func (s *Server) forfeitOthelloTimeout(room *RoomState, loserSeat types.SeatKey) {
	winnerSeat := oppositeSeat(loserSeat)
	loserName := occupantName(room.Seats[loserSeat])
	winnerName := occupantName(room.Seats[winnerSeat])
	s.forceEndOthelloGame(room, types.RoundResult(winnerSeat), forceOthelloOpts{
		Label:              fmt.Sprintf("%s超时判负，%s胜利", loserName, winnerName),
		HistoryNote:        "超时判负",
		Notice:             fmt.Sprintf("%s 用时已到，本局判负。", loserName),
		ForfeitRankedFloor: true,
	})
}

func (s *Server) scheduleOthelloReadyStart(room *RoomState) {
	if room.Settings.GameID != types.GameOthello {
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
	if room.ResultText == "正在随机执黑先手..." {
		return
	}
	room.ResultText = "正在随机执黑先手..."
	s.broadcastRoom(room.ID, true)
	roomID := room.ID
	timeAfterFunc(1400*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameOthello {
			return
		}
		if current.Phase != types.PhaseReady || current.Seats[types.SeatA] == nil || current.Seats[types.SeatB] == nil ||
			!current.Ready[types.SeatA] || !current.Ready[types.SeatB] {
			return
		}
		s.startOthelloRoom(current)
		blackSeat := types.SeatA
		if current.Othello != nil {
			blackSeat = current.Othello.BlackSeat
		}
		s.roomNotice(current, fmt.Sprintf("随机完成：%s 执黑先手。", occupantName(current.Seats[blackSeat])))
		s.broadcastRoom(current.ID, true)
	})
}

func (s *Server) scheduleOthelloSettlement(room *RoomState) {
	s.clearOthelloSettlementTimer(room.ID)
	if room.Othello == nil || room.Othello.PendingSettlement == nil {
		return
	}
	pending := room.Othello.PendingSettlement
	delay := pending.ExpiresAt - nowMs()
	if delay < 250 {
		delay = 250
	}
	roomID := room.ID
	pendingID := pending.ID
	timer := timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Othello == nil || current.Othello.PendingSettlement == nil {
			return
		}
		if current.Othello.PendingSettlement.ID != pendingID {
			return
		}
		mode := current.Othello.PendingSettlement.Forced
		if mode == "" {
			mode = "normal"
		}
		s.settleOthelloPendingMove(current, mode, "timeout")
		s.broadcastRoom(current.ID, true)
	})
	s.othelloSettlementTimers[room.ID] = timer
}

func (s *Server) settleOthelloPendingMove(room *RoomState, mode, reason string) (bool, string) {
	return s.settleOthelloPendingMoveClean(room, mode, reason)
}

func (s *Server) settleOthelloPendingMoveClean(room *RoomState, mode, reason string) (bool, string) {
	if room.Othello == nil || room.Othello.PendingSettlement == nil {
		return false, "当前没有待结算落子"
	}
	pending := *room.Othello.PendingSettlement
	finalMode := mode
	if pending.Forced != "" {
		finalMode = pending.Forced
	}
	s.clearOthelloSettlementTimer(room.ID)
	player := s.humanPlayerFromSeat(room, pending.Seat)
	opponent := s.humanPlayerFromSeat(room, pending.OpponentSeat)
	rankedDelta := room.Othello.RankedDelta
	if rankedDelta == nil {
		rankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	resultText := ""
	switch finalMode {
	case "normal":
		wD, lD := s.applyRankedStake(player, opponent, pending.Stake)
		rankedDelta[pending.Seat] += wD
		rankedDelta[pending.OpponentSeat] += lD
		resultText = fmt.Sprintf("%s 本手正常结算：%s", occupantName(room.Seats[pending.Seat]), formatSigned(wD))
	case "giveaway":
		if player != nil {
			s.addGiveawayValue(player, float64(pending.Flips)*0.1)
		}
		resultText = fmt.Sprintf("%s 本手白给，%d 子不结算排位分", occupantName(room.Seats[pending.Seat]), pending.Flips)
	default: // tribute
		wD, lD := s.applyRankedStake(opponent, player, pending.Stake)
		rankedDelta[pending.OpponentSeat] += wD
		rankedDelta[pending.Seat] += lD
		if player != nil {
			s.addGiveawayValue(player, float64(pending.Flips)*0.2)
		}
		resultText = fmt.Sprintf("%s 本手上贡，%s 获得 %s", occupantName(room.Seats[pending.Seat]), occupantName(room.Seats[pending.OpponentSeat]), formatSigned(wD))
	}
	if reason == "timeout" && finalMode == "normal" {
		resultText += "（10 秒未选择，自动不白给）"
	}
	if pending.Forced == "giveaway" {
		if pending.ForcedByMasterName != "" {
			resultText += fmt.Sprintf("（主人（%s）强制（%s）白给）", pending.ForcedByMasterName, occupantName(room.Seats[pending.Seat]))
		} else {
			resultText += "（强制白给）"
		}
	}
	if pending.Forced == "tribute" {
		resultText += "（强制上贡）"
	}
	events := append([]string{}, room.Othello.SettlementEvents...)
	events = append(events, resultText)
	if len(events) > 30 {
		events = events[len(events)-30:]
	}
	if player != nil {
		s.refreshPlayerSnapshots(player)
	}
	if opponent != nil {
		s.refreshPlayerSnapshots(opponent)
	}
	room.Othello.RankedDelta = rankedDelta
	room.Othello.SettlementEvents = events
	room.Othello.PendingSettlement = nil
	room.Othello.SurrenderRequest = nil
	room.ResultText = resultText
	s.roomNotice(room, resultText)
	s.advanceOthelloTurn(room, pending.NextTurn, 0)
	return true, ""
}

func (s *Server) flushOthelloPendingSettlement(room *RoomState) {
	if room.Othello == nil || room.Othello.PendingSettlement == nil {
		return
	}
	s.settleOthelloPendingMoveClean(room, "normal", "cleanup")
}

func (s *Server) applyOthelloForfeitRankedFloor(room *RoomState, winnerSeat, loserSeat types.SeatKey) string {
	if !room.Settings.EnableRanked || room.Othello == nil || room.Othello.RankedDelta == nil {
		return ""
	}
	winner := s.humanPlayerFromSeat(room, winnerSeat)
	loser := s.humanPlayerFromSeat(room, loserSeat)
	minimumWin := int(room.Settings.Stake) * int(rankMultiplierFor(room.Settings))
	currentWinnerDelta := room.Othello.RankedDelta[winnerSeat]
	currentLoserDelta := room.Othello.RankedDelta[loserSeat]
	targetWinnerDelta := currentWinnerDelta
	if targetWinnerDelta < minimumWin {
		targetWinnerDelta = minimumWin
	}
	targetLoserDelta := currentLoserDelta
	if targetLoserDelta > -minimumWin {
		targetLoserDelta = -minimumWin
	}
	winnerAdjustment := targetWinnerDelta - currentWinnerDelta
	loserAdjustment := targetLoserDelta - currentLoserDelta
	if winner != nil && winnerAdjustment != 0 {
		s.updateRankedPoints(winner, winnerAdjustment)
	}
	if loser != nil && loserAdjustment != 0 {
		s.updateRankedPoints(loser, loserAdjustment)
	}
	room.Othello.RankedDelta[winnerSeat] = targetWinnerDelta
	room.Othello.RankedDelta[loserSeat] = targetLoserDelta
	if winnerAdjustment == 0 && loserAdjustment == 0 {
		return ""
	}
	return fmt.Sprintf("；判负兜底：赢家至少 +%d，输家至少 -%d", minimumWin, minimumWin)
}

func (s *Server) applyOthelloEscapeRankedPenalty(room *RoomState, winnerSeat, loserSeat types.SeatKey, ratio float64, label string) string {
	if !room.Settings.EnableRanked || room.Othello == nil || room.Othello.RankedDelta == nil {
		return ""
	}
	winner := s.humanPlayerFromSeat(room, winnerSeat)
	loser := s.humanPlayerFromSeat(room, loserSeat)
	unit := int(room.Settings.Stake) * int(rankMultiplierFor(room.Settings))
	remainingSpaces := 0
	for _, row := range room.Othello.Board {
		for _, cell := range row {
			if cell == nil {
				remainingSpaces++
			}
		}
	}
	penalty := int(math.Round(float64(remainingSpaces) * float64(unit) * ratio))
	if penalty < unit {
		penalty = unit
	}
	if winner != nil {
		s.updateRankedPoints(winner, penalty)
	}
	if loser != nil {
		s.updateRankedPoints(loser, -penalty)
	}
	room.Othello.RankedDelta[winnerSeat] += penalty
	room.Othello.RankedDelta[loserSeat] -= penalty
	ratioText := fmt.Sprintf("%v", ratio)
	if ratio == 0.5 {
		ratioText = "1/2"
	}
	return fmt.Sprintf("；%s追加：剩余 %d 格 × %d × %s = %d", label, remainingSpaces, unit, ratioText, penalty)
}

func (s *Server) advanceOthelloTurn(room *RoomState, nextTurn types.SeatKey, passCount int) {
	if room.Othello == nil {
		return
	}
	nextColor := othelloColorForSeat(room.Othello, nextTurn)
	legalMoves := othelloLegalMoves(room.Othello.Board, nextColor)
	if len(legalMoves) > 0 {
		bc, wc := othelloCounts(room.Othello.Board)
		room.Othello.Turn = nextTurn
		room.Othello.LegalMoves = legalMoves
		room.Othello.PassCount = passCount
		room.Othello.BlackCount = bc
		room.Othello.WhiteCount = wc
		room.ResultText = ""
		s.armOthelloTimers(room, nextTurn)
		s.notifyOpponentTurn(room, nextTurn)
		return
	}
	if passCount+1 >= 2 {
		s.finishOthelloGame(room)
		return
	}
	skippedName := occupantName(room.Seats[nextTurn])
	fallbackTurn := oppositeSeat(nextTurn)
	fallbackColor := othelloColorForSeat(room.Othello, fallbackTurn)
	fallbackLegalMoves := othelloLegalMoves(room.Othello.Board, fallbackColor)
	if len(fallbackLegalMoves) == 0 {
		s.finishOthelloGame(room)
		return
	}
	s.roomNotice(room, skippedName+" 没有合法落子，系统自动跳过。")
	bc, wc := othelloCounts(room.Othello.Board)
	room.Othello.Turn = fallbackTurn
	room.Othello.LegalMoves = fallbackLegalMoves
	room.Othello.PassCount = passCount + 1
	room.Othello.BlackCount = bc
	room.Othello.WhiteCount = wc
	s.armOthelloTimers(room, fallbackTurn)
	s.notifyOpponentTurn(room, fallbackTurn)
}

func (s *Server) applyOthelloMove(room *RoomState, seat types.SeatKey, row, col int) (bool, string) {
	if room.Othello == nil {
		return false, "黑白棋还没有开始"
	}
	if room.Phase != types.PhaseChoosing {
		return false, "当前不能落子"
	}
	if room.Othello.PendingSettlement != nil {
		return false, "上一手还在等待白给/上贡结算"
	}
	if room.Othello.Turn != seat {
		return false, "还没轮到你落子"
	}
	color := othelloColorForSeat(room.Othello, seat)
	flips := othelloFlips(room.Othello.Board, row, col, color)
	if len(flips) == 0 {
		return false, "这个位置不能落子"
	}
	s.pauseOthelloTimers(room, seat)
	board := cloneOthelloBoard(room.Othello.Board)
	c := color
	board[row][col] = &c
	for _, item := range flips {
		cc := color
		board[item.Row][item.Col] = &cc
	}
	rankedDelta := room.Othello.RankedDelta
	if rankedDelta == nil {
		rankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	opponentSeat := oppositeSeat(seat)
	player := s.humanPlayerFromSeat(room, seat)
	opponent := s.humanPlayerFromSeat(room, opponentSeat)
	liveStake := len(flips) * int(room.Settings.Stake) * int(rankMultiplierFor(room.Settings))
	useGiveawaySettlement := room.Settings.EnableRanked && player != nil && ptrBool(player.GiveawayEnabled) && s.isFullHumanRoom(room)
	nextTurn := othelloSeatForColor(room.Othello, oppositeOthelloColor(color))
	if useGiveawaySettlement {
		// 主人强制只作用于当前房间本局，并保证本手判成 giveaway；自然概率仍可能进入上贡分支。
		forcedByMaster := takeForcedGiveaway(room, seat)
		forcedGiveaway := forcedByMaster != "" || s.shouldTriggerGiveaway(player)
		var forced string
		if forcedGiveaway {
			if forcedByMaster != "" {
				forced = "giveaway"
			} else if ptrFloat(player.GiveawayValue) >= 75 && rand.Float64() < 0.5 {
				forced = "tribute"
			} else {
				forced = "giveaway"
			}
		}
		expires := nowMs() + 10_000
		if forced != "" {
			expires = nowMs() + 2400
		}
		pending := &types.OthelloPendingSettlement{
			ID: randomID(), Seat: seat, OpponentSeat: opponentSeat,
			Flips: len(flips), Stake: liveStake, NextTurn: nextTurn,
			ExpiresAt: expires, Forced: forced, ForcedByMasterName: forcedByMaster,
		}
		bc, wc := othelloCounts(board)
		room.Othello.Board = board
		room.Othello.PassCount = 0
		room.Othello.RankedDelta = rankedDelta
		room.Othello.PendingSettlement = pending
		room.Othello.SurrenderRequest = nil
		room.Othello.LegalMoves = []types.Pos{}
		room.Othello.BlackCount = bc
		room.Othello.WhiteCount = wc
		if forced == "tribute" {
			room.ResultText = occupantName(room.Seats[seat]) + " 触发强制上贡，正在结算..."
		} else if forced == "giveaway" {
			if forcedByMaster != "" {
				room.ResultText = fmt.Sprintf("主人（%s）强制（%s）白给，正在结算...", forcedByMaster, occupantName(room.Seats[seat]))
			} else {
				room.ResultText = occupantName(room.Seats[seat]) + " 触发强制白给，正在结算..."
			}
		} else {
			room.ResultText = occupantName(room.Seats[seat]) + " 请在 10 秒内选择：不白给 / 白给 / 上贡。"
		}
		if player != nil {
			s.refreshPlayerSnapshots(player)
		}
		if opponent != nil {
			s.refreshPlayerSnapshots(opponent)
		}
		s.scheduleOthelloSettlement(room)
		return true, ""
	}
	if room.Settings.EnableRanked {
		wD, lD := s.applyRankedStake(player, opponent, liveStake)
		rankedDelta[seat] += wD
		rankedDelta[opponentSeat] += lD
	}
	if player != nil {
		s.refreshPlayerSnapshots(player)
	}
	if opponent != nil {
		s.refreshPlayerSnapshots(opponent)
	}
	bc, wc := othelloCounts(board)
	room.Othello.Board = board
	room.Othello.PassCount = 0
	room.Othello.RankedDelta = rankedDelta
	room.Othello.SurrenderRequest = nil
	room.Othello.BlackCount = bc
	room.Othello.WhiteCount = wc
	s.advanceOthelloTurn(room, nextTurn, 0)
	return true, ""
}

func cloneOthelloBoard(board [][]*types.OthelloCell) [][]*types.OthelloCell {
	out := make([][]*types.OthelloCell, len(board))
	for i, row := range board {
		out[i] = make([]*types.OthelloCell, len(row))
		for j, cell := range row {
			if cell != nil {
				c := *cell
				out[i][j] = &c
			}
		}
	}
	return out
}

func (s *Server) finishOthelloGame(room *RoomState) {
	if room.Othello == nil {
		return
	}
	s.clearOthelloSettlementTimer(room.ID)
	s.clearOthelloClockTimer(room.ID)
	blackCount, whiteCount := othelloCounts(room.Othello.Board)
	blackSeat := room.Othello.BlackSeat
	whiteSeat := oppositeSeat(blackSeat)
	var result types.RoundResult
	if blackCount == whiteCount {
		result = types.ResultDraw
	} else if blackCount > whiteCount {
		result = types.RoundResult(blackSeat)
	} else {
		result = types.RoundResult(whiteSeat)
	}
	room.Othello.BlackCount = blackCount
	room.Othello.WhiteCount = whiteCount
	room.Othello.Ended = true
	room.Othello.Winner = result
	room.Othello.LegalMoves = []types.Pos{}
	room.Phase = types.PhaseResult
	room.Status = "playing"
	if result == types.ResultDraw {
		room.ResultText = fmt.Sprintf("黑白棋平局：黑 %d，白 %d", blackCount, whiteCount)
	} else {
		room.ResultText = fmt.Sprintf("%s胜利：黑 %d，白 %d", occupantName(room.Seats[types.SeatKey(result)]), blackCount, whiteCount)
	}
	rankedDelta := room.Othello.RankedDelta
	if rankedDelta == nil {
		rankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	if room.Settings.EnableRanked {
		room.ResultText += "（实时结算：" + othelloRankedText(room.Othello) + "）"
	}
	playerA, playerB := s.applySeatOutcome(room, result)
	room.ResultText += othelloSettlementSummary(room.Othello)
	s.refreshHumans(playerA, playerB)
	resultLabel := "黑白棋平局"
	if result != types.ResultDraw {
		resultLabel = occupantName(room.Seats[types.SeatKey(result)]) + "胜利"
	}
	item := s.buildMatchHistoryShell(room, result, types.GameOthello, resultLabel, room.ResultText)
	item.OthelloScore = &types.OthelloScore{Black: blackCount, White: whiteCount}
	item.OthelloBlackSeat = blackSeat
	if room.Settings.EnableRanked {
		es := absMax(rankedDelta[types.SeatA], rankedDelta[types.SeatB])
		item.EffectiveStake = &es
	}
	s.finalizeMatch(room, result, item)
}

func absMax(a, b int) int {
	if a < 0 {
		a = -a
	}
	if b < 0 {
		b = -b
	}
	if a > b {
		return a
	}
	return b
}

func (s *Server) forceEndOthelloGame(room *RoomState, result types.RoundResult, opts forceOthelloOpts) (bool, string) {
	if room.Othello == nil {
		return false, "黑白棋还没有开始"
	}
	if room.Othello.Ended || room.Phase == types.PhaseResult {
		return false, "当前黑白棋对局已经结束"
	}
	s.flushOthelloPendingSettlement(room)
	if room.Othello == nil || room.Othello.Ended {
		return false, "当前黑白棋对局已经结束"
	}
	s.clearOthelloSettlementTimer(room.ID)
	s.clearOthelloClockTimer(room.ID)
	blackCount, whiteCount := othelloCounts(room.Othello.Board)
	punishedPlayers := s.punishmentPlayersForResult(room, result)
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, result, punishment)
	playerA := s.humanPlayerFromSeat(room, types.SeatA)
	playerB := s.humanPlayerFromSeat(room, types.SeatB)
	room.Othello.BlackCount = blackCount
	room.Othello.WhiteCount = whiteCount
	room.Othello.Ended = true
	room.Othello.Winner = result
	room.Othello.LegalMoves = []types.Pos{}
	room.Othello.SurrenderRequest = nil
	room.Phase = types.PhaseResult
	room.Status = "playing"
	blackSeat := room.Othello.BlackSeat
	label := opts.Label
	if label == "" {
		if result == types.ResultDraw {
			label = "管理员判定平局"
		} else if types.SeatKey(result) == blackSeat {
			label = "管理员判定黑方胜利"
		} else {
			label = "管理员判定白方胜利"
		}
	}
	rankedFloorText := ""
	if opts.ForfeitRankedFloor && (result == types.ResultA || result == types.ResultB) {
		rankedFloorText = s.applyOthelloForfeitRankedFloor(room, types.SeatKey(result), oppositeSeat(types.SeatKey(result)))
	}
	escapePenaltyText := ""
	if opts.EscapePenaltyRatio > 0 && (result == types.ResultA || result == types.ResultB) {
		escapeLabel := opts.EscapePenaltyLabel
		if escapeLabel == "" {
			escapeLabel = "逃跑"
		}
		escapePenaltyText = s.applyOthelloEscapeRankedPenalty(room, types.SeatKey(result), oppositeSeat(types.SeatKey(result)), opts.EscapePenaltyRatio, escapeLabel)
	}
	// 与 applySeatOutcome 一致：按游戏记分项并同步总榜
	s.applySeatOutcome(room, result)
	rankedText := ""
	if room.Settings.EnableRanked {
		rankedText = "；实时结算：" + othelloRankedText(room.Othello) + rankedFloorText + escapePenaltyText
	}
	room.ResultText = fmt.Sprintf("%s：黑 %d，白 %d%s%s", label, blackCount, whiteCount, rankedText, othelloSettlementSummary(room.Othello))
	if playerA != nil {
		s.refreshPlayerSnapshots(playerA)
	}
	if playerB != nil {
		s.refreshPlayerSnapshots(playerB)
	}
	finalRankedDelta := room.Othello.RankedDelta
	if finalRankedDelta == nil {
		finalRankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	historyNote := "管理员处理"
	if opts.HistoryNote != "" {
		historyNote = opts.HistoryNote
	}
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: occupantName(room.Seats[types.SeatA]), PlayerB: occupantName(room.Seats[types.SeatB]),
		MoveA: types.MoveNoMove, MoveB: types.MoveNoMove, Result: result, ResultLabel: label,
		ResultText: room.ResultText + "（" + historyNote + "）", GameID: types.GameOthello,
		OthelloScore: &types.OthelloScore{Black: blackCount, White: whiteCount}, OthelloBlackSeat: blackSeat,
		Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
	}
	if room.Settings.EnableRanked {
		st := room.Settings.Stake
		item.Stake = &st
		rm := rankMultiplierFor(room.Settings)
		item.RankMultiplier = &rm
		es := absMax(finalRankedDelta[types.SeatA], finalRankedDelta[types.SeatB])
		item.EffectiveStake = &es
	}
	if len(punishedNames) > 0 {
		item.PunishmentName = s.punishmentNameForRoom(room, punishment)
		if room.Settings.PunishmentSource != "player" && punishment != nil {
			item.PunishmentDescription = punishment.Description
		}
	}
	s.addRoundHistory(room, item)
	notice := opts.Notice
	if notice == "" {
		notice = label + "，本局已结束。"
	}
	s.roomNotice(room, notice)
	s.setupPunishmentOrNext(room, result)
	return true, ""
}

type forceOthelloOpts struct {
	Label              string
	HistoryNote        string
	Notice             string
	ForfeitRankedFloor bool
	EscapePenaltyRatio float64
	EscapePenaltyLabel string
}

func (s *Server) applyOthelloDisconnectForfeit(room *RoomState, forfeit DisconnectForfeit) bool {
	s.flushOthelloPendingSettlement(room)
	if room.Phase == types.PhaseResult || (room.Othello != nil && room.Othello.Ended) {
		return true
	}
	s.clearOthelloClockTimer(room.ID)
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	blackCount, whiteCount := 0, 0
	if room.Othello != nil {
		blackCount, whiteCount = othelloCounts(room.Othello.Board)
	}
	recordGameOutcome(winner, types.GameOthello, "win")
	recordGameOutcome(loser, types.GameOthello, "loss")
	s.applyGiveawayWinPenalty(winner)
	rankedFloorText := s.applyOthelloForfeitRankedFloor(room, forfeit.WinnerSeat, forfeit.LoserSeat)
	fullForfeitText := s.applyOthelloEscapeRankedPenalty(room, forfeit.WinnerSeat, forfeit.LoserSeat, 1, "断线全输")
	rankedDelta := map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	if room.Othello != nil && room.Othello.RankedDelta != nil {
		rankedDelta = room.Othello.RankedDelta
	}
	punishedPlayers := s.punishmentPlayersForResult(room, types.RoundResult(forfeit.WinnerSeat))
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishment := s.currentPunishment(room)
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, types.RoundResult(forfeit.WinnerSeat), punishment)
	resetExtremeWinStreak(loser)
	streakText := ""
	if room.Settings.EnableRanked {
		streakText = s.applyExtremeWinStreakRisk(room, winner)
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
	if room.Othello != nil {
		room.Othello.BlackCount = blackCount
		room.Othello.WhiteCount = whiteCount
		room.Othello.Ended = true
		room.Othello.Winner = types.RoundResult(forfeit.WinnerSeat)
		room.Othello.LegalMoves = []types.Pos{}
		room.Othello.SurrenderRequest = nil
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s胜利（黑 %d，白 %d；实时结算：%s%s%s）%s%s",
		forfeit.LoserName, forfeit.WinnerName, blackCount, whiteCount,
		othelloRankedText(room.Othello), rankedFloorText, fullForfeitText, streakText, othelloSettlementSummary(room.Othello))
	if winner != nil {
		s.refreshPlayerSnapshots(winner)
	}
	if loser != nil {
		s.refreshPlayerSnapshots(loser)
	}
	playerAName, playerBName := forfeit.WinnerName, forfeit.LoserName
	if forfeit.LoserSeat == types.SeatA {
		playerAName, playerBName = forfeit.LoserName, forfeit.WinnerName
	}
	item := types.RoundHistoryItem{
		ID: randomID(), Round: len(room.RoundHistory) + 1, At: nowMs(),
		PlayerA: playerAName, PlayerB: playerBName,
		MoveA: types.MoveNoMove, MoveB: types.MoveNoMove,
		Result: types.RoundResult(forfeit.WinnerSeat), ResultLabel: forfeit.WinnerName + "胜利",
		ResultText: room.ResultText + "（断线判负）", GameID: types.GameOthello,
		OthelloScore: &types.OthelloScore{Black: blackCount, White: whiteCount},
		Ranked:       room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
	}
	if room.Othello != nil {
		item.OthelloBlackSeat = room.Othello.BlackSeat
	}
	if forfeit.LoserSeat == types.SeatA {
		item.MoveA = types.MoveForfeit
	} else {
		item.MoveB = types.MoveForfeit
	}
	if room.Settings.EnableRanked {
		st := forfeit.BaseStake
		item.Stake = &st
		rm := forfeit.RankMultiplier
		item.RankMultiplier = &rm
		es := absMax(rankedDelta[types.SeatA], rankedDelta[types.SeatB])
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
