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
	defaultJungleRoomName    = "新的斗兽棋房间"
	defaultChessRoomName     = "新的国际象棋房间"
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

// occupantShortName 返回座位玩家的短名（不含性别/称号，名争惩罚态下为惩罚名），
// 与 playerShortName 对 *PlayerState 的口径一致——供不应嵌入称号的场合使用
// （如 seatWinLabel 的「玩家A/B「昵称」胜利」）；occupantName 仍保留完整的
// 性别-称号-昵称，供 resultText 等既有场合使用。
func occupantShortName(occupant SeatOccupant) string {
	if occupant == nil {
		return "空位"
	}
	p := occupant.(*HumanSeat).Player
	if ptrBool(p.NameWarPunished) && p.NameWarPenaltyName != "" {
		return p.NameWarPenaltyName
	}
	return p.Name
}

// seatWinLabel 组装「玩家A/B「昵称」胜利」格式的简讯，五个双座位游戏
// （RPS/黑白棋/井字棋/五子棋/斗兽棋）的 resultLabel 共用——刻意不含性别/称号，
// 那些完整信息留给 resultText。
func seatWinLabel(room *RoomState, winnerSeat types.SeatKey) string {
	seatText := "玩家A"
	if winnerSeat == types.SeatB {
		seatText = "玩家B"
	}
	return fmt.Sprintf("%s「%s」胜利", seatText, occupantShortName(room.Seats[winnerSeat]))
}

func (s *Server) shouldCloseRoom(room *RoomState) bool {
	if room.Settings.GameID == types.GameLiarsDice {
		return len(room.SpectatorIDs) == 0 && (room.LiarsDice == nil || len(room.LiarsDice.ParticipantIDs) == 0)
	}
	return room.Seats[types.SeatA] == nil &&
		room.Seats[types.SeatB] == nil &&
		len(room.SpectatorIDs) == 0
}

// dropProofImageRoomEntries 清掉 s.proofImageRooms 里指向这个房间的表项。不清也不影响
// 正确性（serveStatic 靠 s.rooms[roomID] != nil 判活，房间一删这些表项自然就会判定为
// "房间已销毁"），纯粹是防止这张表随房间生命周期只增不减、长期运行下无限膨胀。调用方必须
// 已持有 s.mu。
func (s *Server) dropProofImageRoomEntries(roomID string) {
	for filename, rid := range s.proofImageRooms {
		if rid == roomID {
			delete(s.proofImageRooms, filename)
		}
	}
}

func (s *Server) cleanupRoomIfEmpty(room *RoomState) bool {
	if !s.shouldCloseRoom(room) {
		return false
	}
	s.clearOthelloSettlementTimer(room.ID)
	s.clearTicTacToeGiveawayTimer(room.ID)
	s.clearGomokuUndoTimer(room.ID)
	s.clearTurnBasedUndoTimer(room.ID)
	s.clearLiarsDiceStartTimer(room.ID)
	s.clearTurnBasedClockTimer(room.ID)
	s.clearRoomBroadcastTimer(room.ID)
	s.dropSyncChannel(channelRoom(room.ID))
	delete(s.rooms, room.ID)
	s.dropProofImageRoomEntries(room.ID)
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
		Jungle:            room.Jungle,
		Chess:             room.Chess,
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

// filterValidTagIDs 保留配置中存在的标签 ID，去重，顺序按后台标签列表固定顺序。
func (s *Server) filterValidTagIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	want := map[string]struct{}{}
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			want[id] = struct{}{}
		}
	}
	var out []string
	for _, tag := range s.cfg.PunishmentTags {
		if _, ok := want[tag.ID]; ok {
			out = append(out, tag.ID)
		}
	}
	return out
}

// firstTagWithRoomNamePool 按后台标签固定顺序遍历已选标签，返回第一个房名词库非空的。
func (s *Server) firstTagWithRoomNamePool(included []string) *types.PunishmentTagConfig {
	want := map[string]struct{}{}
	for _, id := range included {
		want[id] = struct{}{}
	}
	for i := range s.cfg.PunishmentTags {
		tag := &s.cfg.PunishmentTags[i]
		if _, ok := want[tag.ID]; !ok {
			continue
		}
		if tag.RoomNamePool != nil && len(tag.RoomNamePool.Subjects) > 0 && len(tag.RoomNamePool.RoomWords) > 0 {
			return tag
		}
	}
	return nil
}

// firstTagWithBackground 按后台标签固定顺序遍历已选标签，返回第一个背景图库非空的。
func (s *Server) firstTagWithBackground(included []string) *types.PunishmentTagConfig {
	want := map[string]struct{}{}
	for _, id := range included {
		want[id] = struct{}{}
	}
	for i := range s.cfg.PunishmentTags {
		tag := &s.cfg.PunishmentTags[i]
		if _, ok := want[tag.ID]; !ok {
			continue
		}
		if len(tag.RoomBackgroundImages) > 0 {
			return tag
		}
	}
	return nil
}

// findSeriesByID 只返回"可用"的系列（seriesIsUsable：至少要有一步）。没有步骤的
// 系列即便已保存到库里，对外也视为不存在：建房面板选不到，房间设置校验拒绝，
// 运行时也不会产出任务。阵营覆盖不全不再拦截（运行时走兜底文案）。
func (s *Server) findSeriesByID(id string) *types.PunishmentSeriesTaskConfig {
	id = strings.TrimSpace(id)
	if id == "" {
		return nil
	}
	for i := range s.punishmentSeriesCache {
		if s.punishmentSeriesCache[i].ID == id {
			series := &s.punishmentSeriesCache[i]
			if !s.seriesIsUsable(*series) {
				return nil
			}
			return series
		}
	}
	return nil
}

func (s *Server) roomNamePoolForSettings(settings types.RoomSettings) *types.RoomNamePool {
	if !settings.EnablePunishment {
		return nil
	}
	src := normalizePunishmentSource(settings.PunishmentSource)
	if src == "player" {
		return s.cfg.PlayerPunishmentRoomNamePool
	}
	if src == "series" {
		if series := s.findSeriesByID(settings.PunishmentSeriesID); series != nil {
			return series.RoomNamePool
		}
		return nil
	}
	// random：遍历已选标签，取第一个有房名词库的。
	if tag := s.firstTagWithRoomNamePool(settings.PunishmentTagsIncluded); tag != nil {
		return tag.RoomNamePool
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
	if settings.GameID == types.GameJungle && !settings.EnablePunishment {
		return s.uniqueRoomName(defaultJungleRoomName)
	}
	if settings.GameID == types.GameChess && !settings.EnablePunishment {
		return s.uniqueRoomName(defaultChessRoomName)
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
	if cleanName == "" || cleanName == defaultRoomName || cleanName == defaultOthelloRoomName || cleanName == defaultTicTacToeRoomName || cleanName == defaultGomokuRoomName || cleanName == defaultJungleRoomName || cleanName == defaultChessRoomName {
		return s.generatedRoomName(settings)
	}
	return cleanName
}

func (s *Server) randomRoomBackground(settings types.RoomSettings) string {
	if !settings.EnablePunishment {
		return ""
	}
	src := normalizePunishmentSource(settings.PunishmentSource)
	if src == "player" {
		return ""
	}
	if src == "series" {
		if series := s.findSeriesByID(settings.PunishmentSeriesID); series != nil && len(series.RoomBackgroundImages) > 0 {
			return randomFromF(series.RoomBackgroundImages)
		}
		return ""
	}
	if tag := s.firstTagWithBackground(settings.PunishmentTagsIncluded); tag != nil {
		return randomFromF(tag.RoomBackgroundImages)
	}
	return ""
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
	if room.Settings.GameID == types.GameJungle && room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "斗兽棋对局进行中不能离开战斗席，请等待对局结束"}
		}
	}
	if room.Settings.GameID == types.GameChess && room.Phase == types.PhaseChoosing {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "国际象棋对局进行中不能离开战斗席，请等待对局结束"}
		}
	}
	if room.Settings.GameID == types.GameLiarsDice && room.Phase == types.PhaseChoosing {
		if room.LiarsDice != nil && containsString(room.LiarsDice.ParticipantIDs, player.ID) && isProtected {
			return LeaveResult{OK: false, Error: "大话骰对局进行中不能离开参战席，请等待对局结束"}
		}
	}
	// RPS 出拳阶段禁止手动离席/换房/观战：否则会留下对手已出拳，新人入座被秒结算或卡死。
	// 但若双方都还没出拳（room.Choices 为空），则不存在"留下对手已出拳"的风险，允许随意离席。
	if (room.Settings.GameID == types.GameRPS || room.Settings.GameID == "") && room.Phase == types.PhaseChoosing && len(room.Choices) > 0 {
		if _, ok := s.seatOf(room, player.ID); ok && isProtected {
			return LeaveResult{OK: false, Error: "出拳进行中不能离开战斗席，请等待本局结算或断线超时"}
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
	if room.ForcedGiveawayBySeat != nil {
		delete(room.ForcedGiveawayBySeat, seat)
	}
	room.Score[seat] = 0
	if room.GiveawayBoostedBySeat != nil {
		delete(room.GiveawayBoostedBySeat, seat)
	}
	room.SeatedScore[seat] = 0
	room.SeatStats[seat] = emptySeatStats()
	if room.ForgiveAdvantage != nil && leavingID != "" {
		if room.ForgiveAdvantage.BeneficiaryID == leavingID || room.ForgiveAdvantage.TargetID == leavingID {
			room.ForgiveAdvantage = nil
		}
	}
	// RPS 出拳中有人离席（断线超时/踢人等）：整桌收回出拳与 forfeit，回到 waiting，避免陈年 Choices。
	if (room.Settings.GameID == types.GameRPS || room.Settings.GameID == "") && room.Phase == types.PhaseChoosing {
		room.Phase = types.PhaseReady
		room.Status = "waiting"
		room.Choices = map[types.SeatKey]types.Move{}
		room.RevealedChoices = nil
		room.DisconnectForfeits = map[string]DisconnectForfeit{}
		room.ResultText = ""
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
	if room.Settings.GameID == types.GameJungle && room.Phase != types.PhaseResult && room.Phase != types.PhaseChoosing {
		s.resetJungleRoom(room)
	}
	if room.Settings.GameID == types.GameChess && room.Phase != types.PhaseResult && room.Phase != types.PhaseChoosing {
		s.resetChessRoom(room)
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
	if reason == LeaveAdminKick && room.Phase == types.PhaseChoosing {
		if room.Settings.GameID == types.GameOthello || room.Settings.GameID == types.GameTicTacToe || room.Settings.GameID == types.GameGomoku ||
			room.Settings.GameID == types.GameJungle || room.Settings.GameID == types.GameChess || room.Settings.GameID == types.GameRPS || room.Settings.GameID == "" {
			if _, ok := s.seatOf(room, player.ID); ok {
				s.createDisconnectForfeit(room, player)
				s.applyDisconnectForfeit(room, player)
			}
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
	if room.Settings.GameID == types.GameJungle && room.Jungle == nil {
		return
	}
	if room.Settings.GameID == types.GameChess && room.Chess == nil {
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
		LoserName:      occupantName(room.Seats[loserSeat]),
		WinnerID:       winner.GetID(),
		WinnerSeat:     winnerSeat,
		WinnerName:     occupantName(winner),
		Stake:          stake,
		BaseStake:      room.Settings.Stake,
		RankMultiplier: rankMultiplierFor(room.Settings),
	}
}
