package server

import (
	"fmt"
	mrand "math/rand"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

const (
	defaultRoomName          = "新的锤子剪刀布房间"
	defaultOthelloRoomName   = "新的黑白棋房间"
	defaultTicTacToeRoomName = "新的井字棋房间"
	defaultGomokuRoomName    = "新的五子棋房间"
)

// activeRoomsOwnedBy 统计某玩家当前存活（未关闭，s.rooms 里还在）的房间数，
// 用于限制单玩家同时开着的房间数量——房间创建频率限流本身不限制"攒了多少个不关"。
func (s *Server) activeRoomsOwnedBy(playerID string) int {
	n := 0
	for _, room := range s.rooms {
		if room.OwnerID == playerID {
			n++
		}
	}
	return n
}

func (s *Server) seatOf(room *RoomState, playerID string) (types.SeatKey, bool) {
	if room.Seats[types.SeatA] != nil && room.Seats[types.SeatA].GetID() == playerID {
		return types.SeatA, true
	}
	if room.Seats[types.SeatB] != nil && room.Seats[types.SeatB].GetID() == playerID {
		return types.SeatB, true
	}
	return "", false
}

func (s *Server) roomHasPlayer(room *RoomState, playerID string) bool {
	if _, ok := s.seatOf(room, playerID); ok {
		return true
	}
	return containsString(room.SpectatorIDs, playerID)
}

func occupantName(occupant SeatOccupant) string {
	if occupant == nil {
		return "空位"
	}
	return occupant.(*HumanSeat).Player.DisplayName
}

func (s *Server) shouldCloseRoom(room *RoomState) bool {
	if room.Settings.GameID == types.GameLiarsDice {
		return len(room.SpectatorIDs) == 0 && (room.LiarsDice == nil || len(room.LiarsDice.ParticipantIDs) == 0)
	}
	return room.Seats[types.SeatA] == nil &&
		room.Seats[types.SeatB] == nil &&
		len(room.SpectatorIDs) == 0
}

func (s *Server) cleanupRoomIfEmpty(room *RoomState) bool {
	if !s.shouldCloseRoom(room) {
		return false
	}
	s.clearOthelloSettlementTimer(room.ID)
	s.clearTicTacToeGiveawayTimer(room.ID)
	s.clearGomokuUndoTimer(room.ID)
	s.clearLiarsDiceStartTimer(room.ID)
	s.clearOthelloClockTimer(room.ID)
	s.clearGomokuClockTimer(room.ID)
	s.clearRoomBroadcastTimer(room.ID)
	s.dropSyncChannel(channelRoom(room.ID))
	delete(s.rooms, room.ID)
	if s.eventDB != nil {
		if err := s.eventDB.insertRoomEvent(roomEventInput{
			At: nowMs(), RoomID: room.ID, RoomName: room.Settings.Name, GameID: string(room.Settings.GameID),
			UserID: room.OwnerID, UserName: room.CreatorName, Action: "close", Reason: "empty_cleanup",
		}); err != nil {
			s.errorLog("room_event_close_failed", err.Error())
		}
	}
	// 房间从大厅列表移除属于结构性变化，立即全量推大厅，避免他人列表残留幽灵房间。
	s.forceBroadcastLobby()
	return true
}

func publicRoomSettings(settings types.RoomSettings) types.RoomSettings {
	out := settings
	out.Password = ""
	return sanitizeRoomSettings(out)
}

func (s *Server) hideOpponentChoices(room *RoomState) map[types.SeatKey]any {
	if room.Phase == types.PhaseResult || room.Phase == types.PhasePunishment {
		out := map[types.SeatKey]any{}
		for k, v := range room.RevealedChoices {
			out[k] = v
		}
		return out
	}
	hidden := map[types.SeatKey]any{}
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if _, ok := room.Choices[seat]; ok && room.Choices[seat] != "" {
			hidden[seat] = "hidden"
		}
	}
	return hidden
}

func (s *Server) roomSnapshot(room *RoomState, includeChat, includeHistory bool) types.RoomSnapshot {
	for _, id := range append([]string{}, room.SpectatorIDs...) {
		if player := s.players[id]; player != nil {
			if s.refreshNameWarState(player, nowMs()) {
				s.refreshPlayerSnapshots(player)
			}
		}
	}
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		if occ := room.Seats[seat]; occ != nil {
			if player := s.players[occ.GetID()]; player != nil {
				if s.refreshNameWarState(player, nowMs()) {
					s.refreshPlayerSnapshots(player)
				}
			}
		}
	}
	spectators := make([]types.PublicPlayer, 0, len(room.SpectatorIDs))
	for _, id := range room.SpectatorIDs {
		if player := s.players[id]; player != nil {
			spectators = append(spectators, s.publicPlayer(player))
		}
	}
	seats := map[types.SeatKey]any{
		types.SeatA: nil,
		types.SeatB: nil,
	}
	for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
		occ := room.Seats[seat]
		if occ == nil {
			seats[seat] = nil
		} else {
			seats[seat] = occ.(*HumanSeat).Player
		}
	}
	ready := map[types.SeatKey]bool{
		types.SeatA: room.Ready[types.SeatA],
		types.SeatB: room.Ready[types.SeatB],
	}
	score := map[types.SeatKey]int{types.SeatA: room.Score[types.SeatA], types.SeatB: room.Score[types.SeatB]}
	seatedScore := map[types.SeatKey]int{types.SeatA: room.SeatedScore[types.SeatA], types.SeatB: room.SeatedScore[types.SeatB]}
	seatStats := map[types.SeatKey]types.SeatStats{
		types.SeatA: room.SeatStats[types.SeatA],
		types.SeatB: room.SeatStats[types.SeatB],
	}
	history := []types.RoundHistoryItem{}
	if includeHistory {
		limit := roomHistoryPageSize
		if len(room.RoundHistory) < limit {
			limit = len(room.RoundHistory)
		}
		history = append(history, room.RoundHistory[:limit]...)
	}
	// 房间聊天已迁到 SQLite，走 chat:load / chat:new，不再随房间快照下发。
	_ = includeChat
	choices := s.hideOpponentChoices(room)
	snap := types.RoomSnapshot{
		ID:                room.ID,
		UpdatedAt:         room.UpdatedAt,
		Settings:          publicRoomSettings(room.Settings),
		Status:            room.Status,
		Phase:             room.Phase,
		Seats:             seats,
		Spectators:        spectators,
		Ready:             ready,
		Choices:           choices,
		Othello:           room.Othello,
		TicTacToe:         room.TicTacToe,
		LiarsDice:         room.LiarsDice,
		Gomoku:            room.Gomoku,
		ResultText:        room.ResultText,
		PunishedPlayerIDs: room.PunishedPlayerIDs,
		Proofs:            room.Proofs,
		Score:             score,
		SeatedScore:       seatedScore,
		SeatStats:         seatStats,
		RoundHistory:      history,
		RoundHistoryTotal: len(room.RoundHistory),
		Chat:              []types.ChatMessage{},
	}
	if room.Phase == types.PhaseResult || room.Phase == types.PhasePunishment {
		if room.RevealedChoices != nil {
			rc := map[types.SeatKey]types.Move{}
			for k, v := range room.RevealedChoices {
				rc[k] = v
			}
			snap.RevealedChoices = rc
		}
	}
	if room.ForgiveAdvantage != nil {
		snap.ForgiveAdvantageTargetID = room.ForgiveAdvantage.TargetID
		snap.ForgiveAdvantageBeneficiaryID = room.ForgiveAdvantage.BeneficiaryID
	}
	return sanitizeRoomSnapshot(snap)
}

func (s *Server) roomRole(room *RoomState, playerID string) string {
	if seat, ok := s.seatOf(room, playerID); ok {
		return "战斗席 " + string(seat)
	}
	if containsString(room.SpectatorIDs, playerID) {
		return "观战"
	}
	return "房间"
}

func (s *Server) canUseBattleSeat(room *RoomState, player *PlayerState) bool {
	if !room.Settings.EnableRanked {
		return true
	}
	return ptrBool(player.ExtremeModeEnabled) == room.Settings.EnableExtremeRanked
}

func (s *Server) rankedSeatRestrictionText(room *RoomState, player *PlayerState) string {
	if !room.Settings.EnableRanked || s.canUseBattleSeat(room, player) {
		return ""
	}
	if room.Settings.EnableExtremeRanked {
		return "非极限模式玩家只能观战极限排位房"
	}
	return "极限模式玩家只能观战普通排位房"
}

func (s *Server) canAutoSeatOnJoin(room *RoomState, player *PlayerState) bool {
	if !s.canUseBattleSeat(room, player) {
		return false
	}
	if room.Phase == types.PhasePunishment || room.Phase == types.PhaseResult {
		return false
	}
	if room.Phase == types.PhaseChoosing && (room.Choices[types.SeatA] != "" || room.Choices[types.SeatB] != "") {
		return false
	}
	return true
}

func (s *Server) normalizeRoomTags(settings types.RoomSettings) []string {
	if !settings.EnableTags {
		return []string{}
	}
	allowed := map[string]struct{}{}
	for _, t := range s.cfg.RoomTags {
		allowed[t] = struct{}{}
	}
	var out []string
	seen := map[string]struct{}{}
	for _, tag := range settings.Tags {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			continue
		}
		if _, ok := allowed[tag]; !ok {
			continue
		}
		if _, ok := seen[tag]; ok {
			continue
		}
		seen[tag] = struct{}{}
		out = append(out, tag)
		if len(out) >= 5 {
			break
		}
	}
	return out
}

func (s *Server) selectedPunishmentIDs(settings types.RoomSettings) []string {
	rawIDs := settings.PunishmentIDs
	if len(rawIDs) == 0 && settings.PunishmentID != "" {
		rawIDs = []string{settings.PunishmentID}
	}
	var valid []string
	seen := map[string]struct{}{}
	for _, id := range rawIDs {
		if _, ok := seen[id]; ok {
			continue
		}
		found := false
		for _, p := range s.cfg.Punishments {
			if p.ID == id {
				found = true
				break
			}
		}
		if found {
			seen[id] = struct{}{}
			valid = append(valid, id)
		}
	}
	if len(valid) > 0 {
		return valid
	}
	if len(s.cfg.Punishments) > 0 {
		return []string{s.cfg.Punishments[0].ID}
	}
	return nil
}

func (s *Server) selectedPunishments(settings types.RoomSettings) []types.PunishmentConfig {
	ids := s.selectedPunishmentIDs(settings)
	var out []types.PunishmentConfig
	for _, id := range ids {
		for _, p := range s.cfg.Punishments {
			if p.ID == id {
				out = append(out, p)
				break
			}
		}
	}
	return out
}

func (s *Server) primaryPunishmentForSettings(settings types.RoomSettings) *types.PunishmentConfig {
	ids := s.selectedPunishmentIDs(settings)
	if len(ids) == 0 {
		return nil
	}
	lastID := ids[len(ids)-1]
	for i := range s.cfg.Punishments {
		if s.cfg.Punishments[i].ID == lastID {
			return &s.cfg.Punishments[i]
		}
	}
	return nil
}

func (s *Server) roomNamePoolForSettings(settings types.RoomSettings) *types.RoomNamePool {
	if settings.EnablePunishment && settings.PunishmentSource == "player" {
		return s.cfg.PlayerPunishmentRoomNamePool
	}
	if settings.EnablePunishment {
		p := s.primaryPunishmentForSettings(settings)
		if p != nil {
			return p.RoomNamePool
		}
	}
	return nil
}

func (s *Server) uniqueRoomName(baseName string) string {
	name := baseName
	if name == "" {
		name = defaultRoomName
	}
	counter := 2
	for {
		exists := false
		for _, room := range s.rooms {
			if room.Settings.Name == name {
				exists = true
				break
			}
		}
		if !exists {
			return name
		}
		name = fmt.Sprintf("%s %d", baseName, counter)
		counter++
	}
}

func (s *Server) generatedRoomName(settings types.RoomSettings) string {
	if settings.GameID == types.GameOthello && !settings.EnablePunishment {
		return s.uniqueRoomName(defaultOthelloRoomName)
	}
	if settings.GameID == types.GameTicTacToe && !settings.EnablePunishment {
		return s.uniqueRoomName(defaultTicTacToeRoomName)
	}
	if settings.GameID == types.GameGomoku && !settings.EnablePunishment {
		return s.uniqueRoomName(defaultGomokuRoomName)
	}
	pool := s.roomNamePoolForSettings(settings)
	if pool == nil {
		if strings.TrimSpace(settings.Name) != "" {
			return settings.Name
		}
		return defaultRoomName
	}
	subject := randomFromF(pool.Subjects)
	roomWord := randomFromF(pool.RoomWords)
	adjective := ""
	if len(pool.Adjectives) > 0 && randFloat() < 0.75 {
		adjective = randomFromF(pool.Adjectives)
	}
	base := adjective + subject + roomWord
	return s.uniqueRoomName(base)
}

func randFloat() float64 {
	return mrand.Float64()
}

func (s *Server) normalizeRoomName(settings types.RoomSettings) string {
	cleanName := strings.TrimSpace(settings.Name)
	runes := []rune(cleanName)
	if len(runes) > 24 {
		cleanName = string(runes[:24])
	}
	if cleanName == "" || cleanName == defaultRoomName || cleanName == defaultOthelloRoomName || cleanName == defaultTicTacToeRoomName || cleanName == defaultGomokuRoomName {
		return s.generatedRoomName(settings)
	}
	return cleanName
}

func (s *Server) randomRoomBackground(settings types.RoomSettings) string {
	if !settings.EnablePunishment || settings.PunishmentSource == "player" {
		return ""
	}
	p := s.primaryPunishmentForSettings(settings)
	if p == nil || len(p.RoomBackgroundImages) == 0 {
		return ""
	}
	return randomFromF(p.RoomBackgroundImages)
}

func (s *Server) humanPlayerFromSeat(room *RoomState, seat types.SeatKey) *PlayerState {
	occ := room.Seats[seat]
	if occ == nil {
		return nil
	}
	return s.players[occ.GetID()]
}

// isFullHumanRoom 双方战斗席都有真人（平台仅支持真人在线对局）。
func (s *Server) isFullHumanRoom(room *RoomState) bool {
	return room.Seats[types.SeatA] != nil && room.Seats[types.SeatB] != nil
}

func (s *Server) canLeaveRoom(player *PlayerState, reason LeaveReason) LeaveResult {
	if player.RoomID == "" {
		return LeaveResult{OK: true}
	}
	room := s.rooms[player.RoomID]
	if room == nil {
		return LeaveResult{OK: true}
	}
	isProtected := reason == LeaveManual || reason == LeaveSwitchRoom || reason == LeaveSpectate
	if room.Settings.GameID == types.GameOthello && room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "黑白棋对局进行中不能离开战斗席，可以申请认输、逃跑或等待对局结束"}
		}
	}
	if room.Settings.GameID == types.GameTicTacToe && room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "井字棋对局进行中不能离开战斗席，请等待对局结束"}
		}
	}
	if room.Settings.GameID == types.GameGomoku && room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "五子棋对局进行中不能离开战斗席，请等待对局结束"}
		}
	}
	if room.Settings.GameID == types.GameLiarsDice && room.Phase == types.PhaseChoosing {
		if room.LiarsDice != nil && containsString(room.LiarsDice.ParticipantIDs, player.ID) && isProtected {
			return LeaveResult{OK: false, Error: "大话骰对局进行中不能离开参战席，请等待对局结束"}
		}
	}
	if room.Phase != types.PhasePunishment {
		return LeaveResult{OK: true}
	}
	isPunished := containsString(room.PunishedPlayerIDs, player.ID)
	if isPunished && isProtected {
		return LeaveResult{OK: false, Error: "惩罚完成前不能离开房间"}
	}
	return LeaveResult{OK: true}
}

func (s *Server) clearSeatForPlayer(room *RoomState, seat types.SeatKey) {
	var leavingID string
	if occ := room.Seats[seat]; occ != nil {
		leavingID = occ.GetID()
		delete(room.DisconnectForfeits, leavingID)
	}
	room.Seats[seat] = nil
	room.Ready[seat] = false
	delete(room.Choices, seat)
	room.Score[seat] = 0
	room.SeatedScore[seat] = 0
	room.SeatStats[seat] = emptySeatStats()
	if room.ForgiveAdvantage != nil && leavingID != "" {
		if room.ForgiveAdvantage.BeneficiaryID == leavingID || room.ForgiveAdvantage.TargetID == leavingID {
			room.ForgiveAdvantage = nil
		}
	}
	if room.Settings.GameID == types.GameOthello && room.Phase != types.PhaseResult && room.Phase != types.PhaseChoosing {
		s.resetOthelloRoom(room)
	}
	if room.Settings.GameID == types.GameTicTacToe && room.Phase != types.PhaseResult && room.Phase != types.PhaseChoosing {
		s.resetTicTacToeRoom(room)
	}
	if room.Settings.GameID == types.GameGomoku && room.Phase != types.PhaseResult && room.Phase != types.PhaseChoosing {
		s.resetGomokuRoom(room)
	}
}

func (s *Server) leaveRoom(player *PlayerState, reason LeaveReason) LeaveResult {
	if player.RoomID == "" {
		return LeaveResult{OK: true}
	}
	leaveCheck := s.canLeaveRoom(player, reason)
	if !leaveCheck.OK {
		return leaveCheck
	}
	room := s.rooms[player.RoomID]
	if room == nil {
		if c := s.clients[player.SocketID]; c != nil {
			s.clientLeaveRoom(c, player.RoomID)
		}
		player.RoomID = ""
		return LeaveResult{OK: true}
	}
	s.handlePunishmentDeparture(room, player, reason)
	if reason == LeaveManual || reason == LeaveSwitchRoom {
		s.roomNotice(room, playerShortName(player)+" 离开了房间。")
	}
	if reason == LeaveAdminKick {
		s.roomNotice(room, playerShortName(player)+" 被管理员移出房间。")
	}
	if reason == LeaveAdminKick && (room.Settings.GameID == types.GameOthello || room.Settings.GameID == types.GameTicTacToe || room.Settings.GameID == types.GameGomoku) &&
		room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok {
			s.createDisconnectForfeit(room, player)
			s.applyDisconnectForfeit(room, player)
		}
	}
	if reason == LeaveAdminKick && room.Settings.GameID == types.GameLiarsDice && room.Phase == types.PhaseChoosing {
		if room.LiarsDice != nil && containsString(room.LiarsDice.ParticipantIDs, player.ID) {
			s.createDisconnectForfeit(room, player)
			s.applyDisconnectForfeit(room, player)
		}
	}
	if seat, ok := s.seatOf(room, player.ID); ok {
		s.clearSeatForPlayer(room, seat)
	}
	if room.Settings.GameID == types.GameLiarsDice {
		s.leaveLiarsDiceRoster(room, player.ID)
		s.scheduleLiarsDiceReadyStart(room)
	}
	if c := s.clients[player.SocketID]; c != nil {
		s.clientLeaveRoom(c, room.ID)
	}
	filtered := room.SpectatorIDs[:0]
	for _, id := range room.SpectatorIDs {
		if id != player.ID {
			filtered = append(filtered, id)
		}
	}
	room.SpectatorIDs = filtered
	player.RoomID = ""
	roomDeleted := s.cleanupRoomIfEmpty(room)
	if !roomDeleted {
		s.broadcastRoom(room.ID, false)
		s.broadcastLobby()
	}
	return LeaveResult{OK: true}
}

func (s *Server) clearDisconnectForfeit(player *PlayerState) {
	if player.RoomID == "" {
		return
	}
	room := s.rooms[player.RoomID]
	if room != nil {
		delete(room.DisconnectForfeits, player.ID)
		// 大话骰断线判负走独立的 map，重连时同样要撤销，否则残留的旧判负条目
		// 可能在后续某次强制结算里被错误套用（见 game_liarsdice.go）。
		delete(room.LiarsDiceDisconnectForfeits, player.ID)
	}
}

func (s *Server) createDisconnectForfeit(room *RoomState, player *PlayerState) {
	if room.Settings.GameID == types.GameLiarsDice {
		s.createLiarsDiceDisconnectForfeit(room, player)
		return
	}
	if room.Phase != types.PhaseChoosing {
		return
	}
	if room.Settings.GameID == types.GameRPS && !room.Settings.EnableRanked {
		return
	}
	if room.Settings.GameID == types.GameOthello && room.Othello == nil {
		return
	}
	if room.Settings.GameID == types.GameTicTacToe && room.TicTacToe == nil {
		return
	}
	if room.Settings.GameID == types.GameGomoku && room.Gomoku == nil {
		return
	}
	loserSeat, ok := s.seatOf(room, player.ID)
	if !ok {
		return
	}
	winnerSeat := oppositeSeat(loserSeat)
	winner := room.Seats[winnerSeat]
	if winner == nil {
		return
	}
	stake := effectiveRankedStake(room.Settings)
	if room.Settings.GameID == types.GameOthello {
		stake = 0
	}
	if room.DisconnectForfeits == nil {
		room.DisconnectForfeits = map[string]DisconnectForfeit{}
	}
	room.DisconnectForfeits[player.ID] = DisconnectForfeit{
		LoserID:        player.ID,
		LoserSeat:      loserSeat,
		LoserName:      playerShortName(player),
		WinnerID:       winner.GetID(),
		WinnerSeat:     winnerSeat,
		WinnerName:     occupantName(winner),
		Stake:          stake,
		BaseStake:      room.Settings.Stake,
		RankMultiplier: rankMultiplierFor(room.Settings),
	}
}
