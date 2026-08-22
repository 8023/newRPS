package server

import (
	"fmt"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

const (
	jungleRows = 9
	jungleCols = 7
)

// 等级：鼠1 < 猫2 < 狗3 < 狼4 < 豹5 < 虎6 < 狮7 < 象8
var jungleRank = map[types.JungleAnimal]int{
	types.JungleRat: 1, types.JungleCat: 2, types.JungleDog: 3, types.JungleWolf: 4,
	types.JungleLeopard: 5, types.JungleTiger: 6, types.JungleLion: 7, types.JungleElephant: 8,
}

func jungleCellOf(side types.SeatKey, animal types.JungleAnimal) types.JungleCell {
	return types.JungleCell(string(side) + ":" + string(animal))
}

func parseJungleCell(cell types.JungleCell) (side types.SeatKey, animal types.JungleAnimal, ok bool) {
	parts := strings.SplitN(string(cell), ":", 2)
	if len(parts) != 2 {
		return "", "", false
	}
	side = types.SeatKey(parts[0])
	if side != types.SeatA && side != types.SeatB {
		return "", "", false
	}
	animal = types.JungleAnimal(parts[1])
	if _, exists := jungleRank[animal]; !exists {
		return "", "", false
	}
	return side, animal, true
}

func jungleInBounds(row, col int) bool {
	return row >= 0 && row < jungleRows && col >= 0 && col < jungleCols
}

// jungleIsWater：两片 3×2 水域，行 3–5，列 1–2 与 4–5。
func jungleIsWater(row, col int) bool {
	if row < 3 || row > 5 {
		return false
	}
	return (col >= 1 && col <= 2) || (col >= 4 && col <= 5)
}

// jungleIsDen：A 兽穴 (8,3)，B 兽穴 (0,3)。
func jungleIsDen(row, col int, side types.SeatKey) bool {
	if col != 3 {
		return false
	}
	if side == types.SeatA {
		return row == 8
	}
	return row == 0
}

// jungleIsTrap：各方兽穴前/左/右三格陷阱。
func jungleIsTrap(row, col int, side types.SeatKey) bool {
	if side == types.SeatA {
		return (row == 8 && (col == 2 || col == 4)) || (row == 7 && col == 3)
	}
	return (row == 0 && (col == 2 || col == 4)) || (row == 1 && col == 3)
}

func jungleDenOwner(row, col int) (types.SeatKey, bool) {
	if jungleIsDen(row, col, types.SeatA) {
		return types.SeatA, true
	}
	if jungleIsDen(row, col, types.SeatB) {
		return types.SeatB, true
	}
	return "", false
}

func jungleTrapOwner(row, col int) (types.SeatKey, bool) {
	if jungleIsTrap(row, col, types.SeatA) {
		return types.SeatA, true
	}
	if jungleIsTrap(row, col, types.SeatB) {
		return types.SeatB, true
	}
	return "", false
}

func placeJungle(board [][]*types.JungleCell, row, col int, side types.SeatKey, animal types.JungleAnimal) {
	c := jungleCellOf(side, animal)
	board[row][col] = &c
}

func initialJungleBoard() [][]*types.JungleCell {
	board := make([][]*types.JungleCell, jungleRows)
	for r := 0; r < jungleRows; r++ {
		board[r] = make([]*types.JungleCell, jungleCols)
	}
	// 座位 A 下方（行 6–8），座位 B 上方（行 0–2）；双方按各自视角 180° 旋转对称摆放。
	// B (top)
	placeJungle(board, 0, 0, types.SeatB, types.JungleLion)
	placeJungle(board, 0, 6, types.SeatB, types.JungleTiger)
	placeJungle(board, 1, 1, types.SeatB, types.JungleDog)
	placeJungle(board, 1, 5, types.SeatB, types.JungleCat)
	placeJungle(board, 2, 0, types.SeatB, types.JungleRat)
	placeJungle(board, 2, 2, types.SeatB, types.JungleLeopard)
	placeJungle(board, 2, 4, types.SeatB, types.JungleWolf)
	placeJungle(board, 2, 6, types.SeatB, types.JungleElephant)
	// A (bottom)
	placeJungle(board, 6, 0, types.SeatA, types.JungleElephant)
	placeJungle(board, 6, 2, types.SeatA, types.JungleWolf)
	placeJungle(board, 6, 4, types.SeatA, types.JungleLeopard)
	placeJungle(board, 6, 6, types.SeatA, types.JungleRat)
	placeJungle(board, 7, 1, types.SeatA, types.JungleCat)
	placeJungle(board, 7, 5, types.SeatA, types.JungleDog)
	placeJungle(board, 8, 0, types.SeatA, types.JungleTiger)
	placeJungle(board, 8, 6, types.SeatA, types.JungleLion)
	return board
}

func freshJungleState(first types.SeatKey) *types.JungleState {
	return &types.JungleState{
		Board:       initialJungleBoard(),
		Turn:        first,
		MoveCount:   0,
		RankedDelta: map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		UndoCount:   freshUndoCount(),
	}
}

// jungleMoveRecord：一步已走棋的回退信息。Captured 为落点被吃的子（nil=空格）；
// 斗兽棋棋子身份不变，回退时把落点上的子放回 From、恢复 Captured 即可。
type jungleMoveRecord struct {
	From     types.Pos
	To       types.Pos
	Captured *types.JungleCell
}

func cloneJungleBoard(board [][]*types.JungleCell) [][]*types.JungleCell {
	out := make([][]*types.JungleCell, len(board))
	for i, row := range board {
		out[i] = make([]*types.JungleCell, len(row))
		for j, cell := range row {
			if cell != nil {
				c := *cell
				out[i][j] = &c
			}
		}
	}
	return out
}

// jungleCanCapture：攻击方能否吃掉目标格上的防御方。
// 规则：等级 ≥ 对方；鼠可吃象、象不可吃鼠；鼠从水中不能上岸吃象；
// 防御方若在攻击方己方陷阱中，视为无等级，可被任意棋子吃掉。
func jungleCanCapture(attackerSide types.SeatKey, attacker types.JungleAnimal, fromRow, fromCol, toRow, toCol int, defender types.JungleAnimal) bool {
	if trapOwner, ok := jungleTrapOwner(toRow, toCol); ok && trapOwner == attackerSide {
		// 防御方陷在攻击方的陷阱里，任意己方子可吃。
		return true
	}
	if attacker == types.JungleRat && defender == types.JungleElephant {
		// 鼠不能从水中上岸吃象。
		if jungleIsWater(fromRow, fromCol) && !jungleIsWater(toRow, toCol) {
			return false
		}
		return true
	}
	if attacker == types.JungleElephant && defender == types.JungleRat {
		return false
	}
	return jungleRank[attacker] >= jungleRank[defender]
}

func jungleStepDeltas() [][2]int {
	return [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}}
}

// jungleLegalDests 返回 seat 在 from 的棋子所有合法目标格。
func jungleLegalDests(board [][]*types.JungleCell, seat types.SeatKey, fromRow, fromCol int) []types.Pos {
	if !jungleInBounds(fromRow, fromCol) || board[fromRow][fromCol] == nil {
		return nil
	}
	side, animal, ok := parseJungleCell(*board[fromRow][fromCol])
	if !ok || side != seat {
		return nil
	}
	var dests []types.Pos
	// 一步正交
	for _, d := range jungleStepDeltas() {
		r, c := fromRow+d[0], fromCol+d[1]
		if !jungleInBounds(r, c) {
			continue
		}
		if jungleCanMoveTo(board, seat, animal, fromRow, fromCol, r, c, false) {
			dests = append(dests, types.Pos{Row: r, Col: c})
		}
	}
	// 狮虎跳河
	if animal == types.JungleLion || animal == types.JungleTiger {
		for _, d := range jungleStepDeltas() {
			if dest, ok := jungleJumpDest(board, fromRow, fromCol, d[0], d[1]); ok {
				if jungleCanMoveTo(board, seat, animal, fromRow, fromCol, dest.Row, dest.Col, true) {
					dests = append(dests, dest)
				}
			}
		}
	}
	return dests
}

// jungleJumpDest：从 from 沿 dir 越过连续水域后的第一块陆地；途中水域有鼠则不可跳。
func jungleJumpDest(board [][]*types.JungleCell, fromRow, fromCol, dr, dc int) (types.Pos, bool) {
	r, c := fromRow+dr, fromCol+dc
	if !jungleInBounds(r, c) || !jungleIsWater(r, c) {
		return types.Pos{}, false
	}
	for jungleInBounds(r, c) && jungleIsWater(r, c) {
		if board[r][c] != nil {
			// 河中有子（通常是鼠）阻挡跳跃。
			return types.Pos{}, false
		}
		r += dr
		c += dc
	}
	if !jungleInBounds(r, c) {
		return types.Pos{}, false
	}
	return types.Pos{Row: r, Col: c}, true
}

func jungleCanMoveTo(board [][]*types.JungleCell, seat types.SeatKey, animal types.JungleAnimal, fromRow, fromCol, toRow, toCol int, isJump bool) bool {
	if !jungleInBounds(toRow, toCol) {
		return false
	}
	// 不可进入己方兽穴。
	if jungleIsDen(toRow, toCol, seat) {
		return false
	}
	// 只有鼠能进水（跳跃落地必是陆地，无需再检）。
	if !isJump && jungleIsWater(toRow, toCol) && animal != types.JungleRat {
		return false
	}
	target := board[toRow][toCol]
	if target == nil {
		return true
	}
	defSide, defAnimal, ok := parseJungleCell(*target)
	if !ok || defSide == seat {
		return false
	}
	return jungleCanCapture(seat, animal, fromRow, fromCol, toRow, toCol, defAnimal)
}

func jungleHasAnyLegalMove(board [][]*types.JungleCell, seat types.SeatKey) bool {
	for r := 0; r < jungleRows; r++ {
		for c := 0; c < jungleCols; c++ {
			if board[r][c] == nil {
				continue
			}
			side, _, ok := parseJungleCell(*board[r][c])
			if !ok || side != seat {
				continue
			}
			if len(jungleLegalDests(board, seat, r, c)) > 0 {
				return true
			}
		}
	}
	return false
}

func (s *Server) resetJungleRoom(room *RoomState) {
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	room.Jungle = nil
	room.jungleMoves = nil
	s.resetTurnBasedRoom(room)
}

func (s *Server) startJungleRoom(room *RoomState) {
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	first := randomSeat()
	s.startTurnBasedPlaying(room)
		room.Jungle = freshJungleState(first)
		room.jungleMoves = nil
		s.maybeJungleGiveawaySkip(room)
	}

func (s *Server) pauseJungleTimers(room *RoomState) {
	if room.Jungle != nil {
		s.pauseTurnBasedClock(room, room.Jungle.Turn, s.jungleClockHooks())
	}
}

func (s *Server) resumeJungleTimers(room *RoomState) {
	s.resumeTurnBasedClock(room, s.jungleClockHooks())
}

func (s *Server) armJungleTimers(room *RoomState, seat types.SeatKey) {
	s.armTurnBasedClock(room, seat, s.jungleClockHooks())
}

func (s *Server) jungleClockHooks() turnBasedClockHooks {
	return turnBasedClockHooks{
		state: func(room *RoomState) (turnBasedClockState, bool) {
			if room.Jungle == nil {
				return turnBasedClockState{}, false
			}
			return turnBasedClockState{
				turn: room.Jungle.Turn, ended: room.Jungle.Ended,
					blocked:        room.Phase == types.PhasePunishment || room.Jungle.ResignRequest != nil || room.Jungle.UndoRequest != nil,
				moveDeadlineAt: &room.Jungle.MoveDeadlineAt, clockDeadlineAt: &room.Jungle.ClockDeadlineAt,
				clockRemaining: &room.Jungle.ClockRemaining,
			}, true
		},
		settings: func(room *RoomState) (int, int) {
			return room.Settings.JungleMoveSeconds, room.Settings.JungleGameMinutes
		},
		onTimeout: func(room *RoomState, seat types.SeatKey) {
			winnerSeat := oppositeSeat(seat)
			loserName := occupantName(room.Seats[seat])
			s.roomNotice(room, loserName+" 用时已到，本局判负。")
			s.finishJungleGame(room, types.RoundResult(winnerSeat), "超时判负")
		},
	}
}

func (s *Server) scheduleJungleReadyStart(room *RoomState) {
	if room.Settings.GameID != types.GameJungle {
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
	if room.ResultText == "正在随机斗兽棋先手..." {
		return
	}
	room.ResultText = "正在随机斗兽棋先手..."
	s.broadcastRoom(room.ID, true)
	roomID := room.ID
	timeAfterFunc(1200*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameJungle {
			return
		}
		if current.Phase != types.PhaseReady || current.Seats[types.SeatA] == nil || current.Seats[types.SeatB] == nil ||
			!current.Ready[types.SeatA] || !current.Ready[types.SeatB] {
			return
		}
		s.startJungleRoom(current)
		first := types.SeatA
		if current.Jungle != nil {
			first = current.Jungle.Turn
		}
		s.roomNotice(current, fmt.Sprintf("随机完成：%s 先手。", occupantName(current.Seats[first])))
		s.broadcastRoom(current.ID, true)
	})
}

func (s *Server) finishJungleGame(room *RoomState, result types.RoundResult, note string) {
	if room.Jungle == nil {
		return
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	room.Jungle.ResignRequest = nil
	room.Jungle.UndoRequest = nil
	rankedDelta := room.Jungle.RankedDelta
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
			s.resetExtremeWinStreak(loser)
			streakText = s.applyExtremeWinStreakRisk(room, winner)
			rankedText = fmt.Sprintf("（%s %s，%s %d）",
				occupantName(room.Seats[types.SeatKey(result)]), formatSigned(wD),
				occupantName(room.Seats[loserSeat]), lD)
		}
	}
	room.Jungle.RankedDelta = rankedDelta
	room.Jungle.Ended = true
	room.Jungle.Winner = result
	room.Phase = types.PhaseResult
	room.Status = "playing"
	noteSuffix := ""
	if note != "" {
		noteSuffix = "（" + note + "）"
	}
	if result == types.ResultDraw {
		room.ResultText = "斗兽棋平局" + rankedText + noteSuffix
	} else {
		room.ResultText = occupantName(room.Seats[types.SeatKey(result)]) + " 斗兽棋胜利" + rankedText + streakText + noteSuffix
	}
	s.refreshHumans(playerA, playerB)
	resultLabel := "斗兽棋平局"
	if result != types.ResultDraw {
		resultLabel = seatWinLabel(room, types.SeatKey(result))
	}
	item := s.buildMatchHistoryShell(room, result, types.GameJungle, resultLabel, room.ResultText)
	item.ExtremeRanked = room.Settings.EnableExtremeRanked
	if room.Settings.EnableRanked {
		es := effectiveRankedStake(room.Settings)
		item.EffectiveStake = &es
	}
	s.roomNotice(room, room.ResultText)
	s.finalizeMatch(room, result, item)
}

func (s *Server) applyJungleMove(room *RoomState, seat types.SeatKey, fromRow, fromCol, toRow, toCol int) (bool, string) {
	if room.Jungle == nil {
		return false, "斗兽棋还没有开始"
	}
	if room.Phase != types.PhaseChoosing {
		return false, "当前不能走子"
	}
	if room.Jungle.Ended {
		return false, "当前斗兽棋对局已经结束"
	}
	if room.Jungle.Turn != seat {
		return false, "还没轮到你走子"
	}
	if room.Jungle.ResignRequest != nil {
		return false, "认输请求处理完成前不能走子"
	}
	if room.Jungle.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能走子"
	}
	if !jungleInBounds(fromRow, fromCol) || !jungleInBounds(toRow, toCol) {
		return false, "这个位置不能走"
	}
	src := room.Jungle.Board[fromRow][fromCol]
	if src == nil {
		return false, "起点没有棋子"
	}
	side, _, ok := parseJungleCell(*src)
	if !ok || side != seat {
		return false, "只能移动自己的棋子"
	}
	legal := false
	for _, p := range jungleLegalDests(room.Jungle.Board, seat, fromRow, fromCol) {
		if p.Row == toRow && p.Col == toCol {
			legal = true
			break
		}
	}
	if !legal {
		return false, "这步棋不合法"
	}

	s.pauseJungleTimers(room)
	board := cloneJungleBoard(room.Jungle.Board)
	// 执行移动（吃子直接覆盖）；悔棋历史记录落点原内容
	var captured *types.JungleCell
	if target := board[toRow][toCol]; target != nil {
		c := *target
		captured = &c
	}
	room.jungleMoves = append(room.jungleMoves, jungleMoveRecord{
		From: types.Pos{Row: fromRow, Col: fromCol}, To: types.Pos{Row: toRow, Col: toCol}, Captured: captured,
	})
	board[toRow][toCol] = board[fromRow][fromCol]
	board[fromRow][fromCol] = nil
	room.Jungle.Board = board
	room.Jungle.MoveCount++
	from := types.Pos{Row: fromRow, Col: fromCol}
	to := types.Pos{Row: toRow, Col: toCol}
	room.Jungle.LastFrom = &from
	room.Jungle.LastTo = &to
	room.Jungle.ResignRequest = nil

		ending := ""
		if denOwner, ok := jungleDenOwner(toRow, toCol); ok && denOwner != seat {
			ending = "攻入兽穴"
		} else {
			next := oppositeSeat(seat)
			room.Jungle.Turn = next
			if !jungleHasAnyLegalMove(board, next) {
				ending = "对方无子可走"
			}
		}
		if captured != nil && perPiecePunishmentEnabled(room) {
			if ending != "" {
				room.pendingPieceEnd = &pendingPieceEnd{result: types.RoundResult(seat), note: ending}
			}
			s.beginMidGamePiecePunishment(room, oppositeSeat(seat), seat, "棋子被吃")
			return true, ""
		}
		if ending != "" {
			s.finishJungleGame(room, types.RoundResult(seat), ending)
			return true, ""
		}
		room.ResultText = ""
		s.maybeJungleGiveawaySkip(room)
		return true, ""
	}

func (s *Server) requestJungleResign(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Jungle == nil || room.Phase != types.PhaseChoosing || room.Jungle.Ended {
		return false, "当前不能申请认输"
	}
	if room.Jungle.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能申请认输"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能申请认输"
	}
	if room.Jungle.ResignRequest != nil && room.Jungle.ResignRequest.FromSeat == seat {
		return false, "你已经申请认输，正在等待对方确认"
	}
	if room.Jungle.ResignRequest != nil {
		return false, "当前已有认输请求，请先处理"
	}
	room.Jungle.ResignRequest = &types.JungleResignRequest{FromSeat: seat, ToSeat: toSeat, CreatedAt: nowMs()}
	s.pauseJungleTimers(room)
	return true, ""
}

func (s *Server) respondJungleResign(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Jungle == nil || room.Jungle.ResignRequest == nil {
		return false, "当前没有认输请求"
	}
	request := room.Jungle.ResignRequest
	if request.ToSeat != seat {
		return false, "这个认输请求不是发给你的"
	}
	loserSeat := request.FromSeat
	winnerSeat := request.ToSeat
	loserName := occupantName(room.Seats[loserSeat])
	winnerName := occupantName(room.Seats[winnerSeat])
	if !accept {
		room.Jungle.ResignRequest = nil
		s.resumeJungleTimers(room)
		s.roomNotice(room, winnerName+" 拒绝认输，对局继续。")
		return true, ""
	}
	s.roomNotice(room, fmt.Sprintf("%s 同意 %s 认输，本局结束。", winnerName, loserName))
	s.finishJungleGame(room, types.RoundResult(winnerSeat), "认输")
	return true, ""
}

// requestJungleUndo：只能由「当前轮到走子的一方」发起，回退最后 2 手（自己上一手 +
// 对方应手），次数上限由 room.Settings.JungleUndoLimit 决定，30 秒无响应自动拒绝。
func (s *Server) requestJungleUndo(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Jungle == nil || room.Phase != types.PhaseChoosing || room.Jungle.Ended {
		return false, "当前不能悔棋"
	}
	if room.Jungle.Turn != seat {
		return false, "只能在轮到你走子时申请悔棋"
	}
	if room.Jungle.MoveCount < 2 {
		return false, "棋局刚开始，还不能悔棋"
	}
	if room.Jungle.UndoRequest != nil {
		return false, "当前已有悔棋请求，请先处理"
	}
	if room.Jungle.ResignRequest != nil {
		return false, "认输请求处理完成前不能悔棋"
	}
	if room.Jungle.UndoCount[seat] >= room.Settings.JungleUndoLimit {
		return false, "你本局的悔棋次数已经用完"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能悔棋"
	}
	startedAt := nowMs()
	room.Jungle.UndoRequest = &types.JungleUndoRequest{
		FromSeat: seat, ToSeat: toSeat, CreatedAt: startedAt, ExpiresAt: startedAt + undoRequestWindow.Milliseconds(),
	}
	s.pauseJungleTimers(room)
	s.scheduleJungleUndoTimeout(room)
	return true, ""
}

func (s *Server) scheduleJungleUndoTimeout(room *RoomState) {
	request := room.Jungle.UndoRequest
	if request == nil {
		return
	}
	roomID := room.ID
	createdAt := request.CreatedAt
	s.scheduleTurnBasedUndoTimeout(roomID, createdAt,
		func(current *RoomState) int64 {
			if current.Jungle == nil || current.Jungle.UndoRequest == nil {
				return 0
			}
			return current.Jungle.UndoRequest.CreatedAt
		},
		func(current *RoomState) {
			current.Jungle.UndoRequest = nil
			s.resumeJungleTimers(current)
			s.roomNotice(current, fmt.Sprintf("%s 未在 30 秒内回应悔棋请求，已自动拒绝。", occupantName(current.Seats[request.ToSeat])))
		})
}

// respondJungleUndo：accept 时按走子历史逆序回退最后 2 手，落子权回到请求方。
func (s *Server) respondJungleUndo(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Jungle == nil || room.Jungle.UndoRequest == nil {
		return false, "当前没有悔棋请求"
	}
	request := room.Jungle.UndoRequest
	if request.ToSeat != seat {
		return false, "这个悔棋请求不是发给你的"
	}
	s.clearTurnBasedUndoTimer(room.ID)
	fromSeat := request.FromSeat
	room.Jungle.UndoRequest = nil
	if !accept {
		s.resumeJungleTimers(room)
		s.roomNotice(room, occupantName(room.Seats[seat])+" 拒绝悔棋，对局继续。")
		return true, ""
	}
	if len(room.jungleMoves) < 2 {
		s.resumeJungleTimers(room)
		return false, "当前棋局状态不支持悔棋"
	}
	board := cloneJungleBoard(room.Jungle.Board)
	moves := room.jungleMoves[:len(room.jungleMoves)-2]
	for i := len(room.jungleMoves) - 1; i >= len(moves); i-- {
		rec := room.jungleMoves[i]
		mover := board[rec.To.Row][rec.To.Col]
		board[rec.To.Row][rec.To.Col] = rec.Captured
		board[rec.From.Row][rec.From.Col] = mover
	}
	room.Jungle.Board = board
	room.jungleMoves = moves
	room.Jungle.MoveCount -= 2
	if len(moves) > 0 {
		from := types.Pos{Row: moves[len(moves)-1].From.Row, Col: moves[len(moves)-1].From.Col}
		to := types.Pos{Row: moves[len(moves)-1].To.Row, Col: moves[len(moves)-1].To.Col}
		room.Jungle.LastFrom = &from
		room.Jungle.LastTo = &to
	} else {
		room.Jungle.LastFrom = nil
		room.Jungle.LastTo = nil
	}
	room.Jungle.Turn = fromSeat
	room.Jungle.UndoCount[fromSeat]++
	s.resumeJungleTimers(room)
	s.roomNotice(room, occupantName(room.Seats[seat])+" 同意悔棋，棋局回退 2 手。")
	return true, ""
}

func (s *Server) applyJungleDisconnectForfeit(room *RoomState, forfeit DisconnectForfeit) bool {
	if room.Phase == types.PhaseResult || (room.Jungle != nil && room.Jungle.Ended) {
		return true
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	rankedDelta := map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	if room.Jungle != nil && room.Jungle.RankedDelta != nil {
		rankedDelta = room.Jungle.RankedDelta
	}
	rankedText := ""
	streakText := ""
	if room.Settings.EnableRanked {
		wD, lD := s.applyRankedStake(winner, loser, forfeit.Stake)
		rankedDelta[forfeit.WinnerSeat] += wD
		rankedDelta[forfeit.LoserSeat] += lD
		s.resetExtremeWinStreak(loser)
		streakText = s.applyExtremeWinStreakRisk(room, winner)
		rankedText = fmt.Sprintf("，排位 %d 分已结算（%s %s，%s %d）", forfeit.Stake, forfeit.WinnerName, formatSigned(wD), forfeit.LoserName, lD)
	}
	s.recordGameOutcome(winner, types.GameJungle, "win")
	s.recordGameOutcome(loser, types.GameJungle, "loss")
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
	if room.Jungle != nil {
		room.Jungle.ResignRequest = nil
		room.Jungle.UndoRequest = nil
		room.Jungle.RankedDelta = rankedDelta
		room.Jungle.Ended = true
		room.Jungle.Winner = types.RoundResult(forfeit.WinnerSeat)
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s 斗兽棋胜利%s%s", forfeit.LoserName, forfeit.WinnerName, rankedText, streakText)
	punishedPlayers := s.punishmentPlayersForResult(room, types.RoundResult(forfeit.WinnerSeat))
	punishedNames := make([]string, len(punishedPlayers))
	for i, p := range punishedPlayers {
		punishedNames[i] = playerShortName(p)
	}
	punishmentTasks := s.buildPunishmentTasks(room, punishedPlayers, types.RoundResult(forfeit.WinnerSeat), "round_end")
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
		Result: types.RoundResult(forfeit.WinnerSeat), ResultLabel: seatWinLabel(room, forfeit.WinnerSeat),
		ResultText: room.ResultText + "（断线判负）", GameID: types.GameJungle,
		Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
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
		item.PunishmentName = s.punishmentRoundLabel(room, punishmentTasks)
	}
	s.addRoundHistory(room, item)
	s.roomNotice(room, room.ResultText)
	s.setupPunishmentOrNext(room, types.RoundResult(forfeit.WinnerSeat))
	return true
}
