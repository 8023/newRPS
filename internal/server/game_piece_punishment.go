package server

import (
	"fmt"
	"math/rand"

	"github.com/doumiao/newRPS/internal/types"
)

type pendingPieceEnd struct {
	result types.RoundResult
	note   string
}

func perPiecePunishmentEnabled(room *RoomState) bool {
	return room.Settings.EnablePunishment && room.Settings.EnablePerPiecePunishment &&
		(room.Settings.GameID == types.GameChess || room.Settings.GameID == types.GameJungle)
}

func (s *Server) beginMidGamePiecePunishment(room *RoomState, loserSeat, winnerSeat types.SeatKey, note string) {
	if s.humanPlayerFromSeat(room, loserSeat) == nil {
		if pending := room.pendingPieceEnd; pending != nil {
			room.pendingPieceEnd = nil
			s.finishPendingPieceGame(room, pending)
		}
		return
	}
	result := types.RoundResult(winnerSeat)
	text := occupantName(room.Seats[loserSeat]) + " 的棋子被吃，触发每子惩罚"
	if note != "" {
		text += "（" + note + "）"
	}
	item := s.buildMatchHistoryShell(room, result, room.Settings.GameID, "吃子惩罚", text)
	s.addRoundHistory(room, item)
	s.setupPunishmentOrNext(room, result)
	room.midGamePunishment = true
	s.roomNotice(room, text)
}

func (s *Server) finishPendingPieceGame(room *RoomState, pending *pendingPieceEnd) {
	room.skipEndPunishment = true
	switch room.Settings.GameID {
	case types.GameChess:
		s.finishChessGame(room, pending.result, pending.note)
	case types.GameJungle:
		s.finishJungleGame(room, pending.result, pending.note)
	}
}

func (s *Server) resumeAfterMidGamePunishment(room *RoomState) {
	room.midGamePunishment = false
	room.PunishedPlayerIDs = []string{}
	room.Proofs = []types.PunishmentProof{}
	room.LockedSeatIDs = map[string]struct{}{}
	if pending := room.pendingPieceEnd; pending != nil {
		room.pendingPieceEnd = nil
		s.finishPendingPieceGame(room, pending)
		s.broadcastRoom(room.ID, true)
		return
	}
	room.Phase = types.PhaseChoosing
	room.Status = "playing"
	switch room.Settings.GameID {
	case types.GameChess:
		if room.Chess != nil && !room.Chess.Ended {
			s.armChessTimers(room, room.Chess.Turn)
			s.notifyOpponentTurn(room, room.Chess.Turn)
		}
	case types.GameJungle:
		if room.Jungle != nil && !room.Jungle.Ended {
			s.maybeJungleGiveawaySkip(room)
		}
	}
	s.broadcastRoom(room.ID, true)
}

func jungleGiveawayChance(value float64) float64 {
	return value / 2
}

func skipJungleTurn(room *RoomState, seat types.SeatKey) {
	if room.Jungle != nil && room.Jungle.Turn == seat {
		room.Jungle.Turn = oppositeSeat(seat)
	}
}

func (s *Server) shouldTriggerJungleGiveaway(player *PlayerState) bool {
	return player != nil && ptrBool(player.GiveawayEnabled) && ptrFloat(player.GiveawayValue) > 0 &&
		rand.Float64()*100 < jungleGiveawayChance(ptrFloat(player.GiveawayValue))
}

func (s *Server) maybeJungleGiveawaySkip(room *RoomState) {
	if room.Jungle == nil || room.Jungle.Ended || room.Phase != types.PhaseChoosing {
		return
	}
	for hops := 0; hops < 4; hops++ {
		seat := room.Jungle.Turn
		forcedBy := takeForcedGiveaway(room, seat)
		player := s.humanPlayerFromSeat(room, seat)
		if forcedBy == "" && !s.shouldTriggerJungleGiveaway(player) {
			break
		}
		skipJungleTurn(room, seat)
		name := occupantName(room.Seats[seat])
		var text string
		if forcedBy != "" {
			if player != nil {
				s.addGiveawayValue(player, s.cfg.Giveaway.ActiveBoostValue)
			}
			text = fmt.Sprintf("主人（%s）强制（%s）白给，跳过本手。", forcedBy, name)
		} else {
			chance := 0.0
			if player != nil {
				chance = jungleGiveawayChance(ptrFloat(player.GiveawayValue))
			}
			text = fmt.Sprintf("%s 按 %.1f%% 白给值触发白给，跳过本手。", name, chance)
		}
		room.ResultText = text
		s.roomNotice(room, text)
	}
	if room.Jungle != nil && !room.Jungle.Ended {
		s.armJungleTimers(room, room.Jungle.Turn)
		s.notifyOpponentTurn(room, room.Jungle.Turn)
	}
}
