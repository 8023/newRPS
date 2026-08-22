package server

import (
	"fmt"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func freshChessState(whiteSeat types.SeatKey) *types.ChessState {
	state := &types.ChessState{
		Board:          initialChessBoard(),
		Turn:           whiteSeat,
		WhiteSeat:      whiteSeat,
		MoveCount:      0,
		RankedDelta:    map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0},
		UndoCount:      freshUndoCount(),
		CastlingWhiteK: true,
		CastlingWhiteQ: true,
		CastlingBlackK: true,
		CastlingBlackQ: true,
	}
	state.LegalMoves = chessLegalMoves(state)
	return state
}

// chessUndoSnapshot：走子前的完整局面快照。国际象棋存在王车易位（一手动两个子）、
// 吃过路兵（被吃兵不在落点上）、兵升变（棋子身份改变），逆操作难以可靠还原，
// 因此悔棋走「恢复快照」而非「回放逆推」；计时字段刻意不入快照（回退后由 resume 重建，
// 避免把走子耗时一并退回去）。
type chessUndoSnapshot struct {
	board         [][]*types.ChessCell
	turn          types.SeatKey
	lastFrom      *types.Pos
	lastTo        *types.Pos
	castlingWK    bool
	castlingWQ    bool
	castlingBK    bool
	castlingBQ    bool
	enPassant     *types.Pos
	halfmoveClock int
	inCheck       bool
	legalMoves    []types.ChessMove
	repetitionLen int
}

func snapshotChessState(state *types.ChessState, repetitionLen int) *chessUndoSnapshot {
	snap := &chessUndoSnapshot{
		board:         cloneChessBoard(state.Board),
		turn:          state.Turn,
		lastFrom:      cloneChessPos(state.LastFrom),
		lastTo:        cloneChessPos(state.LastTo),
		castlingWK:    state.CastlingWhiteK,
		castlingWQ:    state.CastlingWhiteQ,
		castlingBK:    state.CastlingBlackK,
		castlingBQ:    state.CastlingBlackQ,
		enPassant:     cloneChessPos(state.EnPassant),
		halfmoveClock: state.HalfmoveClock,
		inCheck:       state.InCheck,
		legalMoves:    append([]types.ChessMove(nil), state.LegalMoves...),
		repetitionLen: repetitionLen,
	}
	return snap
}

func restoreChessSnapshot(room *RoomState, snap *chessUndoSnapshot) {
	state := room.Chess
	state.Board = cloneChessBoard(snap.board)
	state.Turn = snap.turn
	state.LastFrom = cloneChessPos(snap.lastFrom)
	state.LastTo = cloneChessPos(snap.lastTo)
	state.CastlingWhiteK = snap.castlingWK
	state.CastlingWhiteQ = snap.castlingWQ
	state.CastlingBlackK = snap.castlingBK
	state.CastlingBlackQ = snap.castlingBQ
	state.EnPassant = cloneChessPos(snap.enPassant)
	state.HalfmoveClock = snap.halfmoveClock
	state.InCheck = snap.inCheck
	state.LegalMoves = append([]types.ChessMove(nil), snap.legalMoves...)
	room.chessRepetition = room.chessRepetition[:snap.repetitionLen]
}

func (s *Server) resetChessRoom(room *RoomState) {
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	room.Chess = nil
	room.chessRepetition = nil
	room.chessUndoStack = nil
	s.resetTurnBasedRoom(room)
}

func (s *Server) startChessRoom(room *RoomState) {
	if room.Seats[types.SeatA] == nil || room.Seats[types.SeatB] == nil {
		return
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	s.startTurnBasedPlaying(room)
	whiteSeat := randomSeat()
	room.Chess = freshChessState(whiteSeat)
	room.chessRepetition = []string{chessPositionKey(room.Chess)}
	room.chessUndoStack = nil
	s.armChessTimers(room, whiteSeat)
}

func (s *Server) chessClockHooks() turnBasedClockHooks {
	return turnBasedClockHooks{
		state: func(room *RoomState) (turnBasedClockState, bool) {
			if room.Chess == nil {
				return turnBasedClockState{}, false
			}
			return turnBasedClockState{
				turn: room.Chess.Turn, ended: room.Chess.Ended,
					blocked:        room.Phase == types.PhasePunishment || room.Chess.ResignRequest != nil || room.Chess.UndoRequest != nil,
				moveDeadlineAt: &room.Chess.MoveDeadlineAt, clockDeadlineAt: &room.Chess.ClockDeadlineAt,
				clockRemaining: &room.Chess.ClockRemaining,
			}, true
		},
		settings: func(room *RoomState) (int, int) {
			return room.Settings.ChessMoveSeconds, room.Settings.ChessGameMinutes
		},
		onTimeout: func(room *RoomState, seat types.SeatKey) {
			winnerSeat := oppositeSeat(seat)
			loserName := occupantName(room.Seats[seat])
			winnerColor := chessColorForSeat(room.Chess, winnerSeat)
			if !chessCanPossiblyMate(room.Chess.Board, winnerColor) {
				s.roomNotice(room, loserName+" 用时已到，但对方无法将死，本局和棋。")
				s.finishChessGame(room, types.ResultDraw, "超时但对方子力不足")
				return
			}
			s.roomNotice(room, loserName+" 用时已到，本局判负。")
			s.finishChessGame(room, types.RoundResult(winnerSeat), "超时判负")
		},
	}
}

func (s *Server) pauseChessTimers(room *RoomState) {
	if room.Chess != nil {
		s.pauseTurnBasedClock(room, room.Chess.Turn, s.chessClockHooks())
	}
}

func (s *Server) resumeChessTimers(room *RoomState) {
	s.resumeTurnBasedClock(room, s.chessClockHooks())
}

func (s *Server) armChessTimers(room *RoomState, seat types.SeatKey) {
	s.armTurnBasedClock(room, seat, s.chessClockHooks())
}

func (s *Server) scheduleChessClockTimer(room *RoomState) {
	s.scheduleTurnBasedClockTimer(room, s.chessClockHooks())
}

func (s *Server) scheduleChessReadyStart(room *RoomState) {
	if room.Settings.GameID != types.GameChess {
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
	if room.ResultText == "正在随机国际象棋先手..." {
		return
	}
	room.ResultText = "正在随机国际象棋先手..."
	s.broadcastRoom(room.ID, true)
	roomID := room.ID
	timeAfterFunc(1200*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.rooms[roomID]
		if current == nil || current.Settings.GameID != types.GameChess {
			return
		}
		if current.Phase != types.PhaseReady || current.Seats[types.SeatA] == nil || current.Seats[types.SeatB] == nil ||
			!current.Ready[types.SeatA] || !current.Ready[types.SeatB] {
			return
		}
		s.startChessRoom(current)
		whiteSeat := types.SeatA
		if current.Chess != nil {
			whiteSeat = current.Chess.WhiteSeat
		}
		s.roomNotice(current, fmt.Sprintf("随机完成：%s 执白先手。", occupantName(current.Seats[whiteSeat])))
		s.broadcastRoom(current.ID, true)
	})
}

func (s *Server) finishChessGame(room *RoomState, result types.RoundResult, note string) {
	if room.Chess == nil {
		return
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	room.Chess.ResignRequest = nil
	room.Chess.UndoRequest = nil
	room.Chess.LegalMoves = nil
	rankedDelta := room.Chess.RankedDelta
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
	room.Chess.RankedDelta = rankedDelta
	room.Chess.Ended = true
	room.Chess.Winner = result
	room.Phase = types.PhaseResult
	room.Status = "playing"
	noteSuffix := ""
	if note != "" {
		noteSuffix = "（" + note + "）"
	}
	if result == types.ResultDraw {
		room.ResultText = "国际象棋平局" + rankedText + noteSuffix
	} else {
		room.ResultText = occupantName(room.Seats[types.SeatKey(result)]) + " 国际象棋胜利" + rankedText + streakText + noteSuffix
	}
	s.refreshHumans(playerA, playerB)
	resultLabel := "国际象棋平局"
	if result != types.ResultDraw {
		resultLabel = seatWinLabel(room, types.SeatKey(result))
	}
	item := s.buildMatchHistoryShell(room, result, types.GameChess, resultLabel, room.ResultText)
	item.ChessWhiteSeat = room.Chess.WhiteSeat
	item.ExtremeRanked = room.Settings.EnableExtremeRanked
	if room.Settings.EnableRanked {
		es := effectiveRankedStake(room.Settings)
		item.EffectiveStake = &es
	}
	s.roomNotice(room, room.ResultText)
	s.finalizeMatch(room, result, item)
}

func (s *Server) applyChessMove(room *RoomState, seat types.SeatKey, fromRow, fromCol, toRow, toCol int, promote string) (bool, string) {
	if room.Chess == nil {
		return false, "国际象棋还没有开始"
	}
	if room.Phase != types.PhaseChoosing {
		return false, "当前不能走子"
	}
	if room.Chess.Ended {
		return false, "当前国际象棋对局已经结束"
	}
	if room.Chess.Turn != seat {
		return false, "还没轮到你走子"
	}
	if room.Chess.ResignRequest != nil {
		return false, "认输请求处理完成前不能走子"
	}
	if room.Chess.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能走子"
	}
	if !chessInBounds(fromRow, fromCol) || !chessInBounds(toRow, toCol) {
		return false, "这个位置不能走"
	}
	src := room.Chess.Board[fromRow][fromCol]
	if src == nil {
		return false, "起点没有棋子"
	}
	color, piece, ok := parseChessCell(*src)
	if !ok || color != chessColorForSeat(room.Chess, seat) {
		return false, "只能移动自己的棋子"
	}
	promote = normalizeChessPromote(promote)
	lastRow := 0
	if color == types.ChessBlack {
		lastRow = 7
	}
	if piece == types.ChessPawn && toRow == lastRow && promote == "" {
		return false, "兵到底线必须选择升变"
	}
	if piece != types.ChessPawn || toRow != lastRow {
		promote = ""
	}
	move := types.ChessMove{From: types.Pos{Row: fromRow, Col: fromCol}, To: types.Pos{Row: toRow, Col: toCol}, Promote: promote}
	if _, ok := chessFindLegal(room.Chess, move); !ok {
		return false, "这步棋不合法"
	}

		captured := chessMoveCaptures(room.Chess, move)
		s.pauseChessTimers(room)
		room.chessUndoStack = append(room.chessUndoStack, snapshotChessState(room.Chess, len(room.chessRepetition)))
		chessApplyMoveToState(room.Chess, move)
		room.Chess.ResignRequest = nil
		room.chessRepetition = append(room.chessRepetition, chessPositionKey(room.Chess))

		outcome := chessEvaluateOutcome(room.Chess, room.chessRepetition)
		note := chessOutcomeNote(outcome)
		if captured && perPiecePunishmentEnabled(room) {
			switch outcome {
			case chessCheckmate:
				room.pendingPieceEnd = &pendingPieceEnd{result: types.RoundResult(seat), note: note}
			case chessStalemate, chessFiftyMove, chessInsufficient, chessRepetition:
				room.pendingPieceEnd = &pendingPieceEnd{result: types.ResultDraw, note: note}
			}
			s.beginMidGamePiecePunishment(room, oppositeSeat(seat), seat, "棋子被吃")
			return true, ""
		}
		switch outcome {
		case chessCheckmate:
			s.finishChessGame(room, types.RoundResult(seat), note)
		case chessStalemate, chessFiftyMove, chessInsufficient, chessRepetition:
			s.finishChessGame(room, types.ResultDraw, note)
		default:
			room.ResultText = ""
			s.armChessTimers(room, room.Chess.Turn)
			s.notifyOpponentTurn(room, room.Chess.Turn)
		}
		return true, ""
	}

func (s *Server) requestChessResign(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Chess == nil || room.Phase != types.PhaseChoosing || room.Chess.Ended {
		return false, "当前不能申请认输"
	}
	if room.Chess.UndoRequest != nil {
		return false, "悔棋请求处理完成前不能申请认输"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能申请认输"
	}
	if room.Chess.ResignRequest != nil && room.Chess.ResignRequest.FromSeat == seat {
		return false, "你已经申请认输，正在等待对方确认"
	}
	if room.Chess.ResignRequest != nil {
		return false, "当前已有认输请求，请先处理"
	}
	room.Chess.ResignRequest = &types.ChessResignRequest{FromSeat: seat, ToSeat: toSeat, CreatedAt: nowMs()}
	s.pauseChessTimers(room)
	return true, ""
}

func (s *Server) respondChessResign(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Chess == nil || room.Chess.ResignRequest == nil {
		return false, "当前没有认输请求"
	}
	request := room.Chess.ResignRequest
	if request.ToSeat != seat {
		return false, "这个认输请求不是发给你的"
	}
	loserSeat := request.FromSeat
	winnerSeat := request.ToSeat
	loserName := occupantName(room.Seats[loserSeat])
	winnerName := occupantName(room.Seats[winnerSeat])
	if !accept {
		room.Chess.ResignRequest = nil
		s.resumeChessTimers(room)
		s.roomNotice(room, winnerName+" 拒绝认输，对局继续。")
		return true, ""
	}
	s.roomNotice(room, fmt.Sprintf("%s 同意 %s 认输，本局结束。", winnerName, loserName))
	s.finishChessGame(room, types.RoundResult(winnerSeat), "认输")
	return true, ""
}

// requestChessUndo：只能由「当前轮到走子的一方」发起，回退最后 2 手（自己上一手 +
// 对方应手），次数上限由 room.Settings.ChessUndoLimit 决定，30 秒无响应自动拒绝。
func (s *Server) requestChessUndo(room *RoomState, seat types.SeatKey) (bool, string) {
	if room.Chess == nil || room.Phase != types.PhaseChoosing || room.Chess.Ended {
		return false, "当前不能悔棋"
	}
	if room.Chess.Turn != seat {
		return false, "只能在轮到你走子时申请悔棋"
	}
	if room.Chess.MoveCount < 2 {
		return false, "棋局刚开始，还不能悔棋"
	}
	if room.Chess.UndoRequest != nil {
		return false, "当前已有悔棋请求，请先处理"
	}
	if room.Chess.ResignRequest != nil {
		return false, "认输请求处理完成前不能悔棋"
	}
	if room.Chess.UndoCount[seat] >= room.Settings.ChessUndoLimit {
		return false, "你本局的悔棋次数已经用完"
	}
	toSeat := oppositeSeat(seat)
	if room.Seats[toSeat] == nil {
		return false, "对手不在战斗席，不能悔棋"
	}
	startedAt := nowMs()
	room.Chess.UndoRequest = &types.ChessUndoRequest{
		FromSeat: seat, ToSeat: toSeat, CreatedAt: startedAt, ExpiresAt: startedAt + undoRequestWindow.Milliseconds(),
	}
	s.pauseChessTimers(room)
	s.scheduleChessUndoTimeout(room)
	return true, ""
}

func (s *Server) scheduleChessUndoTimeout(room *RoomState) {
	request := room.Chess.UndoRequest
	if request == nil {
		return
	}
	roomID := room.ID
	createdAt := request.CreatedAt
	s.scheduleTurnBasedUndoTimeout(roomID, createdAt,
		func(current *RoomState) int64 {
			if current.Chess == nil || current.Chess.UndoRequest == nil {
				return 0
			}
			return current.Chess.UndoRequest.CreatedAt
		},
		func(current *RoomState) {
			current.Chess.UndoRequest = nil
			s.resumeChessTimers(current)
			s.roomNotice(current, fmt.Sprintf("%s 未在 30 秒内回应悔棋请求，已自动拒绝。", occupantName(current.Seats[request.ToSeat])))
		})
}

// respondChessUndo：accept 时弹出 2 份走子前快照并恢复较早的一份，落子权回到请求方。
func (s *Server) respondChessUndo(room *RoomState, seat types.SeatKey, accept bool) (bool, string) {
	if room.Chess == nil || room.Chess.UndoRequest == nil {
		return false, "当前没有悔棋请求"
	}
	request := room.Chess.UndoRequest
	if request.ToSeat != seat {
		return false, "这个悔棋请求不是发给你的"
	}
	s.clearTurnBasedUndoTimer(room.ID)
	fromSeat := request.FromSeat
	room.Chess.UndoRequest = nil
	if !accept {
		s.resumeChessTimers(room)
		s.roomNotice(room, occupantName(room.Seats[seat])+" 拒绝悔棋，对局继续。")
		return true, ""
	}
	if len(room.chessUndoStack) < 2 {
		s.resumeChessTimers(room)
		return false, "当前棋局状态不支持悔棋"
	}
	restore := room.chessUndoStack[len(room.chessUndoStack)-2]
	snapshots := room.chessUndoStack[:len(room.chessUndoStack)-2]
	restoreChessSnapshot(room, restore)
	room.chessUndoStack = snapshots
	room.Chess.MoveCount -= 2
	room.Chess.Turn = fromSeat
	room.Chess.UndoCount[fromSeat]++
	s.resumeChessTimers(room)
	s.roomNotice(room, occupantName(room.Seats[seat])+" 同意悔棋，棋局回退 2 手。")
	return true, ""
}

func (s *Server) applyChessDisconnectForfeit(room *RoomState, forfeit DisconnectForfeit) bool {
	if room.Phase == types.PhaseResult || (room.Chess != nil && room.Chess.Ended) {
		return true
	}
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	winner := s.players[forfeit.WinnerID]
	loser := s.players[forfeit.LoserID]
	rankedDelta := map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	if room.Chess != nil && room.Chess.RankedDelta != nil {
		rankedDelta = room.Chess.RankedDelta
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
	s.recordGameOutcome(winner, types.GameChess, "win")
	s.recordGameOutcome(loser, types.GameChess, "loss")
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
	if room.Chess != nil {
		room.Chess.ResignRequest = nil
		room.Chess.UndoRequest = nil
		room.Chess.LegalMoves = nil
		room.Chess.RankedDelta = rankedDelta
		room.Chess.Ended = true
		room.Chess.Winner = types.RoundResult(forfeit.WinnerSeat)
	}
	room.ResultText = fmt.Sprintf("%s 断线超时判负，%s 国际象棋胜利%s%s", forfeit.LoserName, forfeit.WinnerName, rankedText, streakText)
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
		ResultText: room.ResultText + "（断线判负）", GameID: types.GameChess,
		Ranked: room.Settings.EnableRanked, ExtremeRanked: room.Settings.EnableExtremeRanked,
		PunishmentTasks: punishmentTasks, PunishedNames: punishedNames, Proofs: []types.HistoryProof{},
	}
	if room.Chess != nil {
		item.ChessWhiteSeat = room.Chess.WhiteSeat
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
