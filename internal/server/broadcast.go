package server

import (
	"encoding/json"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) recordBroadcast(typ string, bytes int) {
	now := nowMs()
	s.recentBroadcasts = append(s.recentBroadcasts, broadcastMetric{Type: typ, Bytes: bytes, At: now})
	cutoff := now - int64(broadcastMetricWindow/time.Millisecond)
	i := 0
	for i < len(s.recentBroadcasts) && s.recentBroadcasts[i].At <= cutoff {
		i++
	}
	if i > 0 {
		s.recentBroadcasts = append([]broadcastMetric{}, s.recentBroadcasts[i:]...)
	}
	var roomSum, lobbySum, roomN, lobbyN int
	for _, item := range s.recentBroadcasts {
		if item.Type == "room" {
			roomSum += item.Bytes
			roomN++
		} else {
			lobbySum += item.Bytes
			lobbyN++
		}
	}
	s.serverStats.RecentRoomBroadcasts = roomN
	s.serverStats.RecentLobbyBroadcasts = lobbyN
	if roomN > 0 {
		s.serverStats.AverageRoomSnapshotBytes = roomSum / roomN
	} else {
		s.serverStats.AverageRoomSnapshotBytes = 0
	}
	if lobbyN > 0 {
		s.serverStats.AverageLobbySnapshotBytes = lobbySum / lobbyN
	} else {
		s.serverStats.AverageLobbySnapshotBytes = 0
	}
}

func (s *Server) lobbySnapshot(includeConfig, includeSuggestions bool) types.LobbySnapshot {
	for _, player := range s.players {
		s.refreshGiveawayBoard(player, nowMs())
		if s.refreshNameWarState(player, nowMs()) {
			s.refreshPlayerSnapshots(player)
		}
	}
	humanPlayers := make([]types.PublicPlayer, 0, len(s.players))
	online := 0
	for _, player := range s.players {
		humanPlayers = append(humanPlayers, s.publicPlayer(player))
		if player.Connected {
			online++
		}
	}
	rooms := make([]types.LobbyRoomInfo, 0, len(s.rooms))
	for _, room := range s.rooms {
		playersCount := 0
		if room.Seats[types.SeatA] != nil {
			playersCount++
		}
		if room.Seats[types.SeatB] != nil {
			playersCount++
		}
		info := types.LobbyRoomInfo{
			ID:                     room.ID,
			GameID:                 room.Settings.GameID,
			Code:                   room.Code,
			Name:                   room.Settings.Name,
			HasPassword:            room.Settings.Password != "",
			Players:                playersCount,
			Spectators:             len(room.SpectatorIDs),
			Versus: map[types.SeatKey]any{
				types.SeatA: s.lobbySeatSummary(room.Seats[types.SeatA]),
				types.SeatB: s.lobbySeatSummary(room.Seats[types.SeatB]),
			},
			Status:                 room.Status,
			RoomBackgroundImage:    room.Settings.RoomBackgroundImage,
			EnableBot:              room.Settings.EnableBot,
			BotDifficulty:          room.Settings.BotDifficulty,
			EnablePunishment:       room.Settings.EnablePunishment,
			PunishmentIDs:          room.Settings.PunishmentIDs,
			PunishmentID:           room.Settings.PunishmentID,
			TieDoublePunish:        room.Settings.TieDoublePunish,
			RequireOpponentConfirm: room.Settings.RequireOpponentConfirm,
			EnableRanked:           room.Settings.EnableRanked,
			Stake:                  room.Settings.Stake,
			EnableRankMultiplier:   room.Settings.EnableRankMultiplier,
			RankMultiplier:         rankMultiplierFor(room.Settings),
			EnableExtremeRanked:    room.Settings.EnableExtremeRanked,
		}
		if room.Settings.EnableTags {
			info.Tags = room.Settings.Tags
		} else {
			info.Tags = []string{}
		}
		rooms = append(rooms, info)
	}
	suggestions := []types.Suggestion{}
	if includeSuggestions {
		limit := 50
		if len(s.suggestions) < limit {
			limit = len(s.suggestions)
		}
		suggestions = append(suggestions, s.suggestions[:limit]...)
	}
	snap := types.LobbySnapshot{
		OnlineCount:       online,
		Players:           humanPlayers,
		Rooms:             rooms,
		NormalLeaderboard: []types.PublicPlayer{},
		RankedLeaderboard: []types.PublicPlayer{},
		Suggestions:       suggestions,
		LobbyChat:         []types.ChatMessage{},
		ServerStats:       s.serverStats,
	}
	if includeConfig {
		cfg := sanitizePublicConfig(s.publicConfig())
		snap.Config = &cfg
	}
	return sanitizeLobbySnapshot(snap)
}

func (s *Server) lobbySeatSummary(occupant SeatOccupant) any {
	if occupant == nil {
		return nil
	}
	if occupant.IsBot() {
		bot := occupant.(*BotSeat).Bot
		return map[string]any{"name": bot.Name, "isBot": true}
	}
	return map[string]any{"player": occupant.(*HumanSeat).Player}
}

func (s *Server) emitLobbyUpdate() {
	s.lobbyBroadcastTimer = nil
	snapshot := s.lobbySnapshot(false, true)
	s.serverStats.LobbyBroadcasts++
	data, _ := json.Marshal(snapshot)
	s.serverStats.LastLobbySnapshotBytes = len(data)
	s.recordBroadcast("lobby", len(data))
	s.emitToRoom(lobbyChannel, "lobby:update", snapshot)
}

func (s *Server) broadcastLobby() {
	if s.lobbyBroadcastTimer != nil {
		return
	}
	s.lobbyBroadcastTimer = timeAfterFunc(s.lobbyBroadcastDelay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.emitLobbyUpdate()
	})
}

func (s *Server) emitRoomUpdate(roomID string) {
	pending := s.roomBroadcastTimers[roomID]
	delete(s.roomBroadcastTimers, roomID)
	room := s.rooms[roomID]
	if room == nil {
		return
	}
	room.UpdatedAt = nowMs()
	snapshot := s.roomSnapshot(room, false, false)
	s.serverStats.RoomBroadcasts++
	data, _ := json.Marshal(snapshot)
	s.serverStats.LastRoomSnapshotBytes = len(data)
	s.recordBroadcast("room", len(data))
	s.emitToRoom(roomID, "room:update", snapshot)
	if pending != nil && pending.updateLobby {
		s.broadcastLobby()
	}
}

func (s *Server) broadcastRoom(roomID string, updateLobby bool) {
	pending := s.roomBroadcastTimers[roomID]
	if pending != nil {
		if updateLobby {
			pending.updateLobby = true
		}
		return
	}
	timer := timeAfterFunc(s.roomBroadcastDelay, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.emitRoomUpdate(roomID)
	})
	s.roomBroadcastTimers[roomID] = &roomBroadcastPending{timer: timer, updateLobby: updateLobby}
}

func (s *Server) clearRoomBroadcastTimer(roomID string) {
	pending := s.roomBroadcastTimers[roomID]
	if pending == nil {
		return
	}
	pending.timer.Stop()
	delete(s.roomBroadcastTimers, roomID)
}

func (s *Server) broadcastPlayerUpdate(player *PlayerState) {
	s.emitVolatileAll("player:update", s.publicPlayer(player))
}

func (s *Server) appendRoomChat(room *RoomState, message types.ChatMessage) {
	room.Chat = append(room.Chat, message)
	if len(room.Chat) > maxRoomChatMessages {
		room.Chat = room.Chat[len(room.Chat)-maxRoomChatMessages:]
	}
}

func (s *Server) appendLobbyChat(message types.ChatMessage) {
	s.lobbyChat = append(s.lobbyChat, message)
	if len(s.lobbyChat) > maxLobbyMessages {
		s.lobbyChat = s.lobbyChat[len(s.lobbyChat)-maxLobbyMessages:]
	}
}

func (s *Server) emitLobbyChatAppend(message types.ChatMessage) {
	s.emitToRoom(lobbyChannel, "chat:append", message)
}

func (s *Server) appendSuggestion(suggestion types.Suggestion) {
	s.suggestions = append([]types.Suggestion{suggestion}, s.suggestions...)
	if len(s.suggestions) > maxLobbyMessages {
		s.suggestions = s.suggestions[:maxLobbyMessages]
	}
}

func (s *Server) systemChat(text string, roomID string) {
	message := types.ChatMessage{
		ID:       randomID(),
		RoomID:   roomID,
		PlayerID: "system",
		Author:   "系统",
		Text:     text,
		At:       nowMs(),
		System:   true,
	}
	if roomID != "" {
		room := s.rooms[roomID]
		if room == nil {
			return
		}
		s.appendRoomChat(room, message)
		s.emitToRoom(roomID, "chat:append", message)
	} else {
		s.appendLobbyChat(message)
		s.emitLobbyChatAppend(message)
	}
}

func (s *Server) roomNotice(room *RoomState, text string) {
	exp := nowMs() + 5_000
	s.appendRoomChat(room, types.ChatMessage{
		ID:        randomID(),
		RoomID:    room.ID,
		PlayerID:  "system",
		Author:    "系统",
		Text:      text,
		At:        nowMs(),
		System:    true,
		Transient: true,
		ExpiresAt: &exp,
	})
}
