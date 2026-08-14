package server

import (
	"sort"
	"time"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
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

// lobbyLivePlayerWindowMs：离线玩家仍出现在大厅实时快照的最长窗口。
// 与前端 isRenameTargetVisible / isNameWarLoserVisible（30 分钟）对齐。
const lobbyLivePlayerWindowMs int64 = 30 * 60 * 1000

// lobbyPlayerInLiveSnapshot 判断玩家是否应进入大厅实时通道：
// 在线全收；离线仅保留「近期可被点名/改名/名争展示」或「白给上板未过期」的。
// 全站历史档案不再塞进 lobby FULL/DELTA（流量与 CPU 根因 #1）。
func lobbyPlayerInLiveSnapshot(player *PlayerState, now int64) bool {
	if player == nil {
		return false
	}
	if player.Connected {
		return true
	}
	if player.DisconnectExpiresAt != nil && *player.DisconnectExpiresAt > now {
		return true
	}
	if player.DisconnectedAt != nil && now-*player.DisconnectedAt <= lobbyLivePlayerWindowMs {
		return true
	}
	if player.GiveawayBoardText != "" && player.GiveawayBoardExpiresAt != nil && *player.GiveawayBoardExpiresAt > now {
		return true
	}
	return false
}

func (s *Server) lobbySnapshot(includeConfig, includeSuggestions bool) types.LobbySnapshot {
	humanPlayers := make(map[string]types.LobbyPlayer)
	online := 0
	now := nowMs()
	for _, player := range s.players {
		s.refreshGiveawayBoard(player, now)
		if s.refreshNameWarState(player, now) {
			s.refreshPlayerSnapshots(player)
		}
		if player.Connected {
			online++
		}
		if !lobbyPlayerInLiveSnapshot(player, now) {
			continue
		}
		// 实时通道用轻量视图（无分游戏 GameStats），降低 FULL/DELTA 体积。
		lp := types.ToLiveLobbyPlayer(s.publicPlayer(player))
		humanPlayers[lp.ID] = lp
	}

	rooms := make(map[string]types.LobbyRoomInfo, len(s.rooms))
	for _, room := range s.rooms {
		playersCount := 0
		if room.Seats[types.SeatA] != nil {
			playersCount++
		}
		if room.Seats[types.SeatB] != nil {
			playersCount++
		}
		spectatorCount := len(room.SpectatorIDs)
		// 大话骰不进 Seats 体系：参战人数看 ParticipantIDs；参战玩家仍存在 SpectatorIDs 里
		// （为了让快照能按 id 查到完整资料），大厅观战计数要把他们扣掉，避免重复计入。
		if room.Settings.GameID == types.GameLiarsDice && room.LiarsDice != nil {
			playersCount = len(room.LiarsDice.ParticipantIDs)
			spectatorCount = len(room.SpectatorIDs) - playersCount
			if spectatorCount < 0 {
				spectatorCount = 0
			}
		}
		info := types.LobbyRoomInfo{
			ID: room.ID, GameID: room.Settings.GameID, Name: room.Settings.Name,
			HasPassword: room.Settings.Password != "", Players: playersCount, Spectators: spectatorCount,
			Versus: map[types.SeatKey]any{
				types.SeatA: s.lobbySeatSummary(room.Seats[types.SeatA]),
				types.SeatB: s.lobbySeatSummary(room.Seats[types.SeatB]),
			},
			Status: room.Status, RoomBackgroundImage: room.Settings.RoomBackgroundImage,
			EnablePunishment: room.Settings.EnablePunishment, PunishmentIDs: room.Settings.PunishmentIDs,
			PunishmentID: room.Settings.PunishmentID, PunishmentSource: room.Settings.PunishmentSource,
			PunishmentTagsIncluded: room.Settings.PunishmentTagsIncluded,
			PunishmentTagsExcluded: room.Settings.PunishmentTagsExcluded,
			PunishmentSeriesID:     room.Settings.PunishmentSeriesID, TieDoublePunish: room.Settings.TieDoublePunish,
			RequireOpponentConfirm: room.Settings.RequireOpponentConfirm, EnableRanked: room.Settings.EnableRanked,
			Stake: room.Settings.Stake, EnableRankMultiplier: room.Settings.EnableRankMultiplier,
			RankMultiplier: rankMultiplierFor(room.Settings), EnableExtremeRanked: room.Settings.EnableExtremeRanked,
			LiarsDiceMinPlayers: room.Settings.LiarsDiceMinPlayers, LiarsDiceMaxPlayers: room.Settings.LiarsDiceMaxPlayers,
			OthelloMoveSeconds: room.Settings.OthelloMoveSeconds, OthelloGameMinutes: room.Settings.OthelloGameMinutes,
			GomokuMoveSeconds: room.Settings.GomokuMoveSeconds, GomokuGameMinutes: room.Settings.GomokuGameMinutes,
			GomokuUndoLimit:   room.Settings.GomokuUndoLimit,
			JungleMoveSeconds: room.Settings.JungleMoveSeconds, JungleGameMinutes: room.Settings.JungleGameMinutes,
		}
		if room.Settings.EnableTags {
			info.Tags = room.Settings.Tags
		} else {
			info.Tags = []string{}
		}
		rooms[room.ID] = info
	}

	// 大厅聊天（原留言板）已迁到 SQLite，走 chat:load / chat:new，不再随快照下发。
	_ = includeSuggestions
	snap := types.LobbySnapshot{
		OnlineCount: online, Players: humanPlayers, Rooms: rooms,
		NormalLeaderboard: []types.LobbyPlayer{}, RankedLeaderboard: []types.LobbyPlayer{},
		Suggestions: []types.Suggestion{}, LobbyChat: []types.ChatMessage{}, ServerStats: s.serverStats,
		PetBonds: s.publicPetBondEdges(),
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
	// 大厅对阵只嵌精简玩家
	p := occupant.(*HumanSeat).Player
	return map[string]any{"player": types.ToLobbyPlayer(p)}
}

func (s *Server) emitLobbyUpdate() {
	s.lobbyBroadcastTimer = nil
	snapshot := s.lobbySnapshot(false, true)
	env, nbytes, err := s.buildStateEnvelope("lobby:update", channelLobby(), snapshot)
	if err != nil || env == nil {
		return
	}
	s.serverStats.LobbyBroadcasts++
	s.serverStats.LastLobbySnapshotBytes = nbytes
	s.recordBroadcast("lobby", nbytes)
	s.emitWireToRoom(lobbyChannel, env)
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

// forceBroadcastLobby 立即全量刷新大厅（房间删除等结构性变化，避免防抖/增量漏删）。
func (s *Server) forceBroadcastLobby() {
	if s.lobbyBroadcastTimer != nil {
		s.lobbyBroadcastTimer.Stop()
		s.lobbyBroadcastTimer = nil
	}
	// 清空同步基线，强制下一包 FULL（避免增量漏删幽灵房间）
	s.resetSyncChannel(channelLobby())
	s.emitLobbyUpdate()
}

func (s *Server) emitRoomUpdate(roomID string) {
	pending := s.roomBroadcastTimers[roomID]
	delete(s.roomBroadcastTimers, roomID)
	room := s.rooms[roomID]
	if room == nil {
		return
	}
	room.UpdatedAt = nowMs()
	// 必须带上 recent history：惩罚任务文本 / 证明会写在 latest history 里。
	// 若 includeHistory=false，客户端只能靠本地旧 history，任务发布后需手动刷新才能看见。
	// chat 仍走 chat:append 增量，不在此重复全量下发。
	snapshot := s.roomSnapshot(room, false, true)
	env, nbytes, err := s.buildStateEnvelope("room:update", channelRoom(roomID), snapshot)
	if err != nil || env == nil {
		return
	}
	s.serverStats.RoomBroadcasts++
	s.serverStats.LastRoomSnapshotBytes = nbytes
	s.recordBroadcast("room", nbytes)
	s.emitWireToRoom(roomID, env)
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

// broadcastRoomImmediate 取消 debounce 并立刻推送房间状态（惩罚证明提交等时效场景）。
func (s *Server) broadcastRoomImmediate(roomID string, updateLobby bool) {
	pending := s.roomBroadcastTimers[roomID]
	if pending != nil {
		pending.timer.Stop()
		if pending.updateLobby {
			updateLobby = true
		}
		delete(s.roomBroadcastTimers, roomID)
	}
	// 借 pending 结构把 updateLobby 传给 emitRoomUpdate
	s.roomBroadcastTimers[roomID] = &roomBroadcastPending{updateLobby: updateLobby}
	s.emitRoomUpdate(roomID)
}

func (s *Server) clearRoomBroadcastTimer(roomID string) {
	pending := s.roomBroadcastTimers[roomID]
	if pending == nil {
		return
	}
	if pending.timer != nil {
		pending.timer.Stop()
	}
	delete(s.roomBroadcastTimers, roomID)
}

// broadcastPlayerUpdate 100ms 聚合后批量推送精简玩家视图。
// 注意：必须用 RAW 推送「本批变更列表」，不能走状态通道 FULL/DELTA——
// 否则通道文档会被替换成「仅本批玩家」，Diff 会错误地 remove 其它玩家。
func (s *Server) broadcastPlayerUpdate(player *PlayerState) {
	if s.pendingPlayerUpdates == nil {
		s.pendingPlayerUpdates = map[string]*PlayerState{}
	}
	s.pendingPlayerUpdates[player.ID] = player
	if s.playerUpdateTimer != nil {
		return
	}
	s.playerUpdateTimer = timeAfterFunc(100*time.Millisecond, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		s.flushPlayerUpdates()
	})
}

func (s *Server) flushPlayerUpdates() {
	s.playerUpdateTimer = nil
	if len(s.pendingPlayerUpdates) == 0 {
		return
	}
	list := make([]types.LobbyPlayer, 0, len(s.pendingPlayerUpdates))
	for _, p := range s.pendingPlayerUpdates {
		list = append(list, types.ToLobbyPlayer(s.publicPlayer(p)))
	}
	s.pendingPlayerUpdates = map[string]*PlayerState{}
	sort.Slice(list, func(i, j int) bool { return list[i].ID < list[j].ID })
	s.emitVolatileAll("player:batch", list)
}

// chatLiveChannel 返回某条聊天实时推送的目标频道：房间聊天推给房间频道，
// 大厅聊天推给 lobbySuggestionChannel（大厅视图与房间内「大厅」tab 都在此频道）。
func chatLiveChannel(roomID string) string {
	if roomID == "" {
		return lobbySuggestionChannel
	}
	return roomID
}

// deliverChat 持久化（可选）+ 实时推送一条聊天。persist=false 用于系统提示等瞬时消息。
// 走 chat:new（动态 RAW），携带 mentions/seq；房间/大厅由 message.RoomID 区分。
// 房间聊天顺带把当前房间名当作快照落盘（room_messages.room_name），供后台聊天管理按
// 房间名检索——房间之后改名/关闭都不影响历史消息里记录的这个名字。
func (s *Server) deliverChat(message types.ChatMessage, persist bool) {
	if persist && s.chatDB != nil {
		roomName := ""
		if message.RoomID != "" {
			if room := s.rooms[message.RoomID]; room != nil {
				roomName = room.Settings.Name
			}
		}
		if seq, err := s.chatDB.append(message.RoomID, roomName, message); err != nil {
			s.errorLog("chat_persist_failed", err.Error())
		} else {
			message.Seq = seq
		}
	}
	s.emitToRoom(chatLiveChannel(message.RoomID), "chat:new", message)
}

// broadcastChatDeleted 按消息所属 scope（大厅 RoomID=="" / 各房间）分组推送 chat:deleted，
// 通知在线客户端从本地视图立即摘除这些消息 id——配合 chatStore.older/recent 已经过滤掉
// deleted=1 的行，做到新老访客都看不到被管理员删除的聊天。只在软删除（非恢复）时调用：
// 恢复不需要主动推送，客户端本就已经看不到该消息，管理员刷新检索列表即可核实。
func (s *Server) broadcastChatDeleted(refs []chatMessageRef) {
	byScope := map[string][]string{}
	for _, ref := range refs {
		byScope[ref.RoomID] = append(byScope[ref.RoomID], ref.ID)
	}
	for roomID, ids := range byScope {
		s.emitToRoom(chatLiveChannel(roomID), "chat:deleted", map[string]any{"roomId": roomID, "ids": ids})
	}
}

func (s *Server) emitWireToRoom(room string, env *wire.Envelope) {
	data, err := proto.Marshal(env)
	if err != nil {
		return
	}
	// 走 roomClients 反向索引：O(订阅者) 而非 O(全服连接)
	for id := range s.roomClients[room] {
		if c := s.clients[id]; c != nil {
			_ = c.writeBinary(data)
		}
	}
}

func (s *Server) emitWireAll(env *wire.Envelope) {
	data, err := proto.Marshal(env)
	if err != nil {
		return
	}
	for _, c := range s.clients {
		_ = c.writeBinary(data)
	}
}

func (s *Server) emitWireClient(c *Client, env *wire.Envelope) {
	if c == nil {
		return
	}
	data, err := proto.Marshal(env)
	if err != nil {
		return
	}
	_ = c.writeBinary(data)
}

// 兼容旧 JSON 推送路径（chat 等即时消息仍可用 raw envelope）
func (s *Server) emitToRoom(room string, event string, data any) {
	env, err := s.buildRawEnvelope(event, 0, data, "")
	if err != nil {
		return
	}
	s.emitWireToRoom(room, env)
}

func (s *Server) emitVolatileAll(event string, data any) {
	env, err := s.buildRawEnvelope(event, 0, data, "")
	if err != nil {
		return
	}
	s.emitWireAll(env)
}

func (s *Server) emitToClient(clientID string, event string, data any) {
	c := s.clients[clientID]
	if c == nil {
		return
	}
	env, err := s.buildRawEnvelope(event, 0, data, "")
	if err != nil {
		return
	}
	s.emitWireClient(c, env)
}

// emitConfigUpdate 配置变更用状态通道（可增量）。
func (s *Server) emitConfigUpdate() {
	cfg := s.publicConfig()
	env, _, err := s.buildStateEnvelope("config:update", channelConfig(), cfg)
	if err != nil || env == nil {
		return
	}
	s.emitWireAll(env)
}

func (s *Server) dropSyncChannel(name string) {
	if s.syncChans == nil {
		return
	}
	delete(s.syncChans, name)
}

// force full resync helper for one client
func (s *Server) sendFullChannel(c *Client, channel string) {
	switch {
	case channel == channelLobby():
		snap := s.lobbySnapshot(false, true)
		env, _, err := s.buildFullEnvelope("lobby:update", channel, snap)
		if err == nil {
			s.emitWireClient(c, env)
		}
	case channel == channelConfig():
		env, _, err := s.buildFullEnvelope("config:update", channel, s.publicConfig())
		if err == nil {
			s.emitWireClient(c, env)
		}
	// channel "players" 已废弃：全站档案走鉴权+分页+限流的 players:roster，
	// 禁止 sync:full 旁路一次 dump 全站（含 GameStats）成为免鉴权 O(P) 放大器。
	case len(channel) > 5 && channel[:5] == "room:":
		roomID := channel[5:]
		room := s.rooms[roomID]
		if room == nil {
			return
		}
		// 必须已在房内（进房过密码/加入过），禁止用 sync:full 旁观任意房间。
		if c == nil || !c.inRoom(roomID) {
			return
		}
		snap := s.roomSnapshot(room, true, true)
		env, _, err := s.buildFullEnvelope("room:update", channel, snap)
		if err == nil {
			s.emitWireClient(c, env)
		}
	}
}

func (s *Server) roomNotice(room *RoomState, text string) {
	exp := nowMs() + 5_000
	// 瞬时系统提示：不入库，房间广播默认不带 chat，必须单独推送。
	s.deliverChat(types.ChatMessage{
		ID: randomID(), RoomID: room.ID, PlayerID: "system", Author: "系统",
		Text: text, At: nowMs(), System: true, Transient: true, ExpiresAt: &exp,
	}, false)
}
