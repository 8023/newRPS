package server

import (
	"crypto/subtle"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// defaultNameWarPenaltyThreshold 与 config/name-war.json 的 penaltyThreshold 默认值一致，
// 供 nameWarPenaltyThreshold 在配置异常（非负）时兜底；ValidateConfig 已在启动/保存时拒绝
// 非负值，正常运行时这条分支理论上走不到。
const defaultNameWarPenaltyThreshold = -4999

func (s *Server) publicConfig() types.AppConfig {
	cfg := s.cfg
	cfg.Site.AdminPassword = ""
	return sanitizePublicConfig(cfg)
}

func (s *Server) adminPasswordMatches(password string) bool {
	expected := s.cfg.Site.AdminPassword
	if expected == "" {
		return false
	}
	// 恒定时间比较，避免按字符早退泄露口令长度/前缀。
	return subtle.ConstantTimeCompare([]byte(password), []byte(expected)) == 1
}

// rankedScorePercent 把真实排位分映射到相对展示上下限的百分比（可超出 ±100）。
// 正分：points/Max*100；负分：points/|Min|*100；0 → 0。
func (s *Server) rankedScorePercent(points int) float64 {
	if points == 0 {
		return 0
	}
	if points > 0 {
		max := s.cfg.RankedScore.Max
		if max <= 0 {
			return 0
		}
		return float64(points) / float64(max) * 100
	}
	min := s.cfg.RankedScore.Min
	if min >= 0 {
		return 0
	}
	return float64(points) / float64(-min) * 100
}

func (s *Server) titleSegmentFor(points int) *types.TitleSegment {
	if len(s.cfg.Titles) == 0 {
		return nil
	}
	percent := s.rankedScorePercent(points)
	var nearest *types.TitleSegment
	nearestDist := math.MaxFloat64
	for i := range s.cfg.Titles {
		item := &s.cfg.Titles[i]
		if percent >= item.MinPercent && percent <= item.MaxPercent {
			return item
		}
		// 落在这一段之外（含分段之间的空隙、或超出首尾边界）：记录到该段最近边界的距离，
		// 最终夹到距离最近的分段，而不是无条件夹到最高档——分段表若配置成中间留空隙，
		// 落在空隙里的百分比应该就近取邻近段，不能因为循环顺序只跟最高档比较。
		dist := item.MinPercent - percent
		if percent > item.MaxPercent {
			dist = percent - item.MaxPercent
		}
		if dist < nearestDist {
			nearestDist = dist
			nearest = item
		}
	}
	return nearest
}

func (s *Server) titleNamesForSegment(segment *types.TitleSegment, factionID string) []string {
	if segment == nil {
		return []string{"初心拳手"}
	}
	if factionID != "" && segment.FactionNames != nil {
		if names := segment.FactionNames[factionID]; len(names) > 0 {
			return names
		}
	}
	if len(segment.Names) > 0 {
		return segment.Names
	}
	return []string{"初心拳手"}
}

func (s *Server) randomTitleFromSegment(segment *types.TitleSegment, factionID string) string {
	names := s.titleNamesForSegment(segment, factionID)
	if len(names) == 0 {
		return "初心拳手"
	}
	return names[rand.Intn(len(names))]
}

func (s *Server) syncTitleForRankSegment(player *PlayerState, force bool) {
	segment := s.titleSegmentFor(player.Stats.RankedPoints)
	if segment == nil {
		return
	}
	if !force && player.Stats.TitleSegmentID == "" && player.Stats.Title != "" {
		player.Stats.TitleSegmentID = segment.ID
		return
	}
	if force || player.Stats.TitleSegmentID != segment.ID || player.Stats.Title == "" {
		player.Stats.Title = s.randomTitleFromSegment(segment, player.FactionID)
	}
	player.Stats.TitleSegmentID = segment.ID
}

type genderInfoResult struct {
	GenderID      string
	GenderLabel   string
	FactionID     string
	FactionLabel  string
	FactionColors types.GenderColors
}

func (s *Server) genderInfo(genderID string) genderInfoResult {
	fallbackFaction := types.GenderFaction{
		ID: "unknown_faction", Label: "未知阵营",
		GenderColors: types.GenderColors{TextColor: "#4d5c6f", BackgroundColor: "#eef3f8", BorderColor: "#c9d6e4"},
	}
	if len(s.cfg.GenderFactions) > 0 {
		fallbackFaction = s.cfg.GenderFactions[0]
	}
	var gender *types.GenderOption
	for i := range s.cfg.Genders {
		if s.cfg.Genders[i].ID == genderID {
			gender = &s.cfg.Genders[i]
			break
		}
	}
	if gender == nil && len(s.cfg.Genders) > 0 {
		gender = &s.cfg.Genders[0]
	}
	faction := fallbackFaction
	if gender != nil {
		for i := range s.cfg.GenderFactions {
			if s.cfg.GenderFactions[i].ID == gender.FactionID {
				faction = s.cfg.GenderFactions[i]
				break
			}
		}
	}
	gid, glabel := genderID, genderID
	if gender != nil {
		gid, glabel = gender.ID, gender.Label
	}
	return genderInfoResult{
		GenderID: gid, GenderLabel: glabel,
		FactionID: faction.ID, FactionLabel: faction.Label,
		FactionColors: types.GenderColors{
			TextColor: faction.TextColor, BackgroundColor: faction.BackgroundColor, BorderColor: faction.BorderColor,
		},
	}
}

func (s *Server) nameWarCode() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (s *Server) generateNameWarPenaltyName() string {
	prefix := strings.TrimSpace(s.cfg.NameWar.PenaltyPrefix)
	if prefix == "" {
		prefix = "失名者"
	}
	return prefix + "-" + s.nameWarCode()
}

func formatDisplayName(player *PlayerState) string {
	if ptrBool(player.NameWarPunished) && player.NameWarPenaltyName != "" {
		return player.NameWarPenaltyName
	}
	return player.GenderLabel + " - " + player.Stats.Title + " - " + player.Name
}

func playerShortName(player *PlayerState) string {
	if ptrBool(player.NameWarPunished) && player.NameWarPenaltyName != "" {
		return player.NameWarPenaltyName
	}
	return player.Name
}

func playerShortNamePublic(p types.PublicPlayer) string {
	if ptrBool(p.NameWarPunished) && p.NameWarPenaltyName != "" {
		return p.NameWarPenaltyName
	}
	return p.Name
}

func (s *Server) refreshGiveawayBoard(player *PlayerState, now int64) {
	if player.GiveawayBoardExpiresAt == nil || *player.GiveawayBoardExpiresAt > now {
		return
	}
	player.GiveawayBoardText = ""
	player.GiveawayBoardSubmittedAt = nil
	player.GiveawayBoardExpiresAt = nil
	player.GiveawayBoardLikes = intPtr(0)
	player.GiveawayBoardDislikes = intPtr(0)
	player.GiveawayBoardLikesThisHour = intPtr(0)
	player.GiveawayBoardLikeWindowStartedAt = nil
}

func (s *Server) addGiveawayValue(player *PlayerState, delta float64) {
	v := ptrFloat(player.GiveawayValue) + delta
	player.GiveawayValue = floatPtr(clampGiveawayValue(v))
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
}

func (s *Server) refreshNameWarState(player *PlayerState, now int64) bool {
	before := fmt.Sprintf("%s|%s|%v|%s|%v|%s|%s",
		player.Stats.Title, player.DisplayName, ptrBool(player.NameWarPunished),
		player.NameWarPenaltyName, player.NameWarRenameProtectedUntil,
		player.NameWarRenamedBy, player.NameWarRenamedByName)

	if !ptrBool(player.NameWarEnabled) {
		player.NameWarPunished = boolPtr(false)
		player.NameWarPenaltyName = ""
		player.NameWarRenameProtectedUntil = nil
		player.NameWarRenamedBy = ""
		player.NameWarRenamedByName = ""
		s.syncTitleForRankSegment(player, false)
	} else {
		protectedActive := player.NameWarRenameProtectedUntil != nil && *player.NameWarRenameProtectedUntil > now
		if protectedActive && player.NameWarPenaltyName != "" {
			player.NameWarPunished = boolPtr(true)
		} else if player.Stats.RankedPoints <= s.nameWarPenaltyThreshold() {
			player.NameWarPunished = boolPtr(true)
			if player.NameWarPenaltyName == "" {
				player.NameWarPenaltyName = s.generateNameWarPenaltyName()
			}
			if player.NameWarRenameProtectedUntil != nil && *player.NameWarRenameProtectedUntil <= now {
				player.NameWarRenameProtectedUntil = nil
			}
		} else {
			player.NameWarPunished = boolPtr(false)
			player.NameWarPenaltyName = ""
			player.NameWarRenameProtectedUntil = nil
			player.NameWarRenamedBy = ""
			player.NameWarRenamedByName = ""
			s.syncTitleForRankSegment(player, false)
		}
	}
	player.DisplayName = formatDisplayName(player)
	after := fmt.Sprintf("%s|%s|%v|%s|%v|%s|%s",
		player.Stats.Title, player.DisplayName, ptrBool(player.NameWarPunished),
		player.NameWarPenaltyName, player.NameWarRenameProtectedUntil,
		player.NameWarRenamedBy, player.NameWarRenamedByName)
	return before != after
}

func (s *Server) applyGender(player *PlayerState, genderID string) {
	oldFactionID := player.FactionID
	next := s.genderInfo(genderID)
	player.GenderID = next.GenderID
	player.GenderLabel = next.GenderLabel
	player.FactionID = next.FactionID
	player.FactionLabel = next.FactionLabel
	player.FactionColors = next.FactionColors
	if oldFactionID != "" && oldFactionID != next.FactionID && !ptrBool(player.NameWarPunished) {
		s.syncTitleForRankSegment(player, true)
	}
	player.DisplayName = formatDisplayName(player)
}

func (s *Server) publicPlayer(player *PlayerState) types.PublicPlayer {
	p := player.PublicPlayer
	p.SyncTotalsFromGameStats()
	// 真实分留给排序；展示字段按后台配置封顶（含历史最高/最低）。
	p.Stats.SortRankedPoints = player.Stats.RankedPoints
	p.Stats.SortHighestScore = player.Stats.HighestScore
	p.Stats.SortLowestScore = player.Stats.LowestScore
	p.Stats.RankedPoints = s.displayClampScore(player, player.Stats.RankedPoints)
	p.Stats.HighestScore = s.displayClampScore(player, player.Stats.HighestScore)
	p.Stats.LowestScore = s.displayClampScore(player, player.Stats.LowestScore)
	return p
}

// displayClampScore 仅用于下发展示：真实存储永不改动；按 RankedScore 的 max/min
// （名争玩家用 nameWarMin）夹紧后的副本。排位分、历史最高、历史最低共用。
func (s *Server) displayClampScore(player *PlayerState, score int) int {
	min := s.cfg.RankedScore.Min
	if ptrBool(player.NameWarEnabled) {
		min = s.cfg.RankedScore.NameWarMin
	}
	return clamp(score, min, s.cfg.RankedScore.Max)
}

func (s *Server) refreshPlayerSnapshots(player *PlayerState) {
	pub := s.publicPlayer(player)
	for _, room := range s.rooms {
		for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
			occ := room.Seats[seat]
			if occ == nil || occ.IsBot() {
				continue
			}
			if occ.GetID() == player.ID {
				room.Seats[seat] = &HumanSeat{Player: pub}
			}
		}
	}
}

func (s *Server) refreshAllPlayersForConfig() {
	for _, player := range s.players {
		s.applyGender(player, player.GenderID)
		s.refreshNameWarState(player, nowMs())
		s.refreshPlayerSnapshots(player)
	}
}

// onlinePlayersFromDevice 统计同一 deviceKey（IP+指纹）下已连接玩家数。
func (s *Server) onlinePlayersFromDevice(deviceKey string, exceptPlayerID string) int {
	if deviceKey == "" {
		return 0
	}
	return s.onlinePlayersMatching(exceptPlayerID, func(p *PlayerState) bool { return p.DeviceKey == deviceKey })
}

// canCreateFromDevice 同 deviceKey 在 10 分钟内新建玩家次数上限。
func (s *Server) canCreateFromDevice(deviceKey string) bool {
	if deviceKey == "" {
		deviceKey = "unknown"
	}
	return s.canCreateFromKey(s.deviceCreateAttempts, deviceKey, s.cfg.AccessControl.MaxCreatesPer10Min)
}

// onlinePlayersFromIP 统计同一出口 IP（不看指纹）下已连接玩家数，是
// onlinePlayersFromDevice 的兜底——指纹由客户端上报、未做真实性校验，
// 攻击脚本每次随机化指纹即可让 deviceKey 各不相同，绕过按设备计算的限制。
func (s *Server) onlinePlayersFromIP(ipAddress string, exceptPlayerID string) int {
	if ipAddress == "" {
		return 0
	}
	return s.onlinePlayersMatching(exceptPlayerID, func(p *PlayerState) bool { return p.IPAddress == ipAddress })
}

// onlinePlayersMatching 是 onlinePlayersFromDevice/onlinePlayersFromIP 共用的计数逻辑：
// 统计满足 match 的已连接玩家数（exceptPlayerID 排除自身，用于"算上我自己以外还有几个"）。
func (s *Server) onlinePlayersMatching(exceptPlayerID string, match func(*PlayerState) bool) int {
	n := 0
	for _, player := range s.players {
		if player.Connected && player.ID != exceptPlayerID && match(player) {
			n++
		}
	}
	return n
}

// canCreateFromIP 同 canCreateFromDevice，但按纯 IP 维度计算，用作兜底。
func (s *Server) canCreateFromIP(ipAddress string) bool {
	if ipAddress == "" {
		ipAddress = "unknown-ip"
	}
	return s.canCreateFromKey(s.ipCreateAttempts, ipAddress, s.cfg.AccessControl.MaxCreatesPerIP)
}

// canCreateFromKey 是 canCreateFromDevice/canCreateFromIP 共用的滑动窗口计数器：
// key 在 10 分钟窗口内的新建次数达到 limit 后拒绝，否则记一次并放行。
func (s *Server) canCreateFromKey(attempts map[string][]int64, key string, limit int) bool {
	now := nowMs()
	windowMs := int64(10 * 60 * 1000)
	filtered := attempts[key][:0]
	for _, t := range attempts[key] {
		if now-t < windowMs {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= limit {
		attempts[key] = filtered
		return false
	}
	attempts[key] = append(filtered, now)
	return true
}

func (s *Server) createPlayer(name, genderID, token string, identityPlayerID, identitySecret string) *PlayerState {
	session := s.verifySessionToken(token)
	persistent := identityPlayerID != "" && identitySecret != ""
	id := randomID()
	if persistent {
		id = generatePublicID()
	} else if session != nil {
		id = session.SID
	}
	gender := s.genderInfo(genderID)
	titleSegment := s.titleSegmentFor(0)
	title := s.randomTitleFromSegment(titleSegment, gender.FactionID)
	titleSegID := ""
	if titleSegment != nil {
		titleSegID = titleSegment.ID
	}
	now := nowMs()
	player := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID:                  id,
			Name:                name,
			GenderID:            gender.GenderID,
			GenderLabel:         gender.GenderLabel,
			FactionID:           gender.FactionID,
			FactionLabel:        gender.FactionLabel,
			FactionColors:       gender.FactionColors,
			DisplayName:         gender.GenderLabel + " - " + title + " - " + name,
			Connected:           true,
			NameWarOriginalName: name,
			GiveawayEnabled:     boolPtr(false),
			GiveawayValue:       floatPtr(0),
			GiveawayClicks:      intPtr(0),
			GiveawayBoardLikes:  intPtr(0),
			GiveawayBoardDislikes: intPtr(0),
			GiveawayVoteLikesThisHour:    intPtr(0),
			GiveawayVoteDislikesThisHour: intPtr(0),
			RankMultiplierUnlocked: boolPtr(false),
			ExtremeModeEnabled:     boolPtr(false),
			ExtremeWinStreak:       intPtr(0),
			ExtremeLastDecayHour:   int64Ptr(currentExtremeDecayHour(now)),
			ExtremeForceClosed:     boolPtr(false),
			Stats: types.PublicStats{
				RankedPoints:   0,
				Title:          title,
				TitleSegmentID: titleSegID,
			},
			GameStats: freshGameStats(),
		},
		Token:      token,
		RecentMoves: nil,
		Persistent: persistent,
		PlayerID:   identityPlayerID,
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if token == "" {
		player.Token = randomID()
	}
	if persistent {
		player.PlayerSecrets = []string{identitySecret}
		player.ClaimKey = randomID()
	}
	if session != nil {
		player.CurrentSID = session.SID
	}
	s.players[player.ID] = player
	s.tokenToPlayer[player.Token] = player.ID
	if player.PlayerID != "" {
		s.playerIdToID[player.PlayerID] = player.ID
	}
	if session != nil {
		s.sidToPlayerID[session.SID] = player.ID
	}
	if persistent {
		s.requestPersist("lazy")
	}
	return player
}

func (s *Server) getPlayerByClientID(clientID string) *PlayerState {
	for _, player := range s.players {
		if player.SocketID == clientID {
			return player
		}
	}
	return nil
}

func (s *Server) clearDisconnectHold(player *PlayerState) {
	if player.graceTimer != nil {
		player.graceTimer.Stop()
		player.graceTimer = nil
	}
	if player.discTimer != nil {
		player.discTimer.Stop()
		player.discTimer = nil
	}
	player.graceGen++
	player.timerGen++
	player.DisconnectExpiresAt = nil
}

// recordRankedExtremes 更新历史最高/最低排位分记录；这两个字段永不设上下限、永不回退。
func recordRankedExtremes(player *PlayerState) {
	if player.Stats.RankedPoints > player.Stats.HighestScore {
		player.Stats.HighestScore = player.Stats.RankedPoints
	}
	if player.Stats.RankedPoints < player.Stats.LowestScore {
		player.Stats.LowestScore = player.Stats.RankedPoints
	}
}

// updateRankedPoints 按增量结算排位分。真实存储值不设上下限——RankedScore 配置的
// 上下限只在 publicPlayer()/displayRankedPoints() 下发展示时生效。
func (s *Server) updateRankedPoints(player *PlayerState, delta int) {
	player.Stats.RankedPoints += delta
	recordRankedExtremes(player)
	s.refreshNameWarState(player, nowMs())
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	if player.Persistent {
		s.requestPersist("important")
	}
}

func (s *Server) setRankedPointsByAdmin(player *PlayerState, points int) {
	player.Stats.RankedPoints = int(math.Round(float64(points)))
	recordRankedExtremes(player)
	s.refreshNameWarState(player, nowMs())
	if player.Persistent {
		s.requestPersist("important")
	}
}

// extremeSegmentID 与称号分段同一套百分比刻度（相对 RankedScore.Max/|Min|），
// 供极限模式的正分输分/负分赢分/整点扣分表按 pos1~4 / neg1~4 取系数。
func (s *Server) extremeSegmentID(points int) string {
	percent := s.rankedScorePercent(points)
	if percent >= 75 {
		return "pos4"
	}
	if percent >= 50 {
		return "pos3"
	}
	if percent >= 25 {
		return "pos2"
	}
	if percent > 0 {
		return "pos1"
	}
	if percent <= -75 {
		return "neg4"
	}
	if percent <= -50 {
		return "neg3"
	}
	if percent <= -25 {
		return "neg2"
	}
	if percent < 0 {
		return "neg1"
	}
	return "pos0"
}

func (s *Server) adjustedRankedDelta(player *PlayerState, delta float64) int {
	if player == nil || !ptrBool(player.ExtremeModeEnabled) || delta == 0 {
		return int(math.Round(delta))
	}
	segment := s.extremeSegmentID(player.Stats.RankedPoints)
	if delta < 0 && player.Stats.RankedPoints > 0 {
		rate := s.cfg.ExtremeMode.PositiveLossRates[segment]
		if rate == 0 && segment != "pos0" {
			// missing key -> 1
			if _, ok := s.cfg.ExtremeMode.PositiveLossRates[segment]; !ok {
				rate = 1
			}
		} else if _, ok := s.cfg.ExtremeMode.PositiveLossRates[segment]; !ok {
			rate = 1
		}
		return -int(math.Round(math.Abs(delta) * rate))
	}
	if delta > 0 && player.Stats.RankedPoints < 0 {
		rate, ok := s.cfg.ExtremeMode.NegativeWinRates[segment]
		if !ok {
			if segment == "neg4" {
				rate = 0.5
			} else {
				rate = 1
			}
		}
		return int(math.Round(delta * rate))
	}
	return int(math.Round(delta))
}

func (s *Server) applyRankedStake(winner, loser *PlayerState, stake int) (winnerDelta, loserDelta int) {
	winnerDelta = s.adjustedRankedDelta(winner, float64(stake))
	loserDelta = s.adjustedRankedDelta(loser, float64(-stake))
	if winner != nil {
		s.updateRankedPoints(winner, winnerDelta)
	}
	if loser != nil {
		s.updateRankedPoints(loser, loserDelta)
	}
	return
}

func (s *Server) applyRankedDrawPenaltyStake(playerA, playerB *PlayerState, stake int) (deltaA, deltaB int) {
	deltaA = s.adjustedRankedDelta(playerA, float64(-stake))
	deltaB = s.adjustedRankedDelta(playerB, float64(-stake))
	if playerA != nil {
		s.updateRankedPoints(playerA, deltaA)
	}
	if playerB != nil {
		s.updateRankedPoints(playerB, deltaB)
	}
	return
}

func rankMultiplierFor(settings types.RoomSettings) types.RankMultiplier {
	if !settings.EnableRanked || !settings.EnableRankMultiplier {
		return 1
	}
	switch settings.RankMultiplier {
	case 2, 5, 10:
		return settings.RankMultiplier
	default:
		return 1
	}
}

func effectiveRankedStake(settings types.RoomSettings) int {
	return int(settings.Stake) * int(rankMultiplierFor(settings))
}

func (s *Server) extremeHourlyDecayAmount(player *PlayerState) int {
	segment := s.extremeSegmentID(player.Stats.RankedPoints)
	amount, ok := s.cfg.ExtremeMode.HourlyDecay[segment]
	if !ok {
		amount = s.cfg.ExtremeMode.HourlyDecay["default"]
		if amount == 0 {
			amount = 2
		}
	}
	if amount < 0 {
		amount = 0
	}
	return int(math.Round(amount))
}

func (s *Server) applyExtremeHourlyDecay() {
	now := nowMs()
	hour := currentExtremeDecayHour(now)
	changedRoomIDs := map[string]struct{}{}
	changed := false
	for _, player := range s.players {
		if !ptrBool(player.ExtremeModeEnabled) {
			continue
		}
		if player.ExtremeLastDecayHour != nil && *player.ExtremeLastDecayHour == hour {
			continue
		}
		player.ExtremeLastDecayHour = int64Ptr(hour)
		amount := s.extremeHourlyDecayAmount(player)
		if amount <= 0 {
			continue
		}
		s.updateRankedPoints(player, -amount)
		changed = true
		if player.RoomID != "" {
			changedRoomIDs[player.RoomID] = struct{}{}
		}
	}
	if !changed {
		return
	}
	for roomID := range changedRoomIDs {
		s.broadcastRoom(roomID, false)
	}
}

func (s *Server) scheduleExtremeHourlyDecay() {
	s.scheduleBoundaryAlignedDecay(3_600_000, currentExtremeDecayHour, time.Hour, s.applyExtremeHourlyDecay)
}

// scheduleBoundaryAlignedDecay 是 scheduleExtremeHourlyDecay/scheduleRankedDailyDecay 共用的
// 调度逻辑：先精确对齐到下一个周期边界（整点/每日零点）触发一次，之后按 period 周期性触发；
// 每次触发都在 s.mu 内跑 apply，与其它状态变更保持串行。periodIndex 把毫秒时间戳换算成
// "第几个周期"（如 currentExtremeDecayHour/currentRankedDecayDay），periodMs 是一个周期的
// 毫秒数（用来算下一个边界），period 是等间隔 ticker 的间隔，应与 periodMs 一致。
func (s *Server) scheduleBoundaryAlignedDecay(periodMs int64, periodIndex func(nowMs int64) int64, period time.Duration, apply func()) {
	now := nowMs()
	nextBoundary := (periodIndex(now) + 1) * periodMs
	delay := nextBoundary - now + 500
	if delay < 1000 {
		delay = 1000
	}
	timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		apply()
		s.mu.Unlock()
		ticker := time.NewTicker(period)
		go func() {
			for range ticker.C {
				s.mu.Lock()
				apply()
				s.mu.Unlock()
			}
		}()
	})
}

// rankedDailyDecayAmount 按配置的比例算出衰减后的目标分值（向 0 截断小数部分）。
// 与极限模式的整点衰减是两套独立机制：这里对所有玩家生效，不要求开启极限模式。
func (s *Server) rankedDailyDecayAmount(player *PlayerState) int {
	ratio := s.cfg.RankedScore.DailyDecayRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 1
	}
	return int(float64(player.Stats.RankedPoints) * ratio)
}

func (s *Server) applyRankedDailyDecay() {
	now := nowMs()
	day := currentRankedDecayDay(now)
	changedRoomIDs := map[string]struct{}{}
	changed := false
	for _, player := range s.players {
		if player.RankedLastDecayDay != nil && *player.RankedLastDecayDay == day {
			continue
		}
		player.RankedLastDecayDay = int64Ptr(day)
		if player.Stats.RankedPoints == 0 {
			continue
		}
		next := s.rankedDailyDecayAmount(player)
		delta := next - player.Stats.RankedPoints
		if delta == 0 {
			continue
		}
		s.updateRankedPoints(player, delta)
		changed = true
		if player.RoomID != "" {
			changedRoomIDs[player.RoomID] = struct{}{}
		}
	}
	if !changed {
		return
	}
	for roomID := range changedRoomIDs {
		s.broadcastRoom(roomID, false)
	}
}

func (s *Server) scheduleRankedDailyDecay() {
	s.scheduleBoundaryAlignedDecay(86_400_000, currentRankedDecayDay, 24*time.Hour, s.applyRankedDailyDecay)
}

func resetExtremeWinStreak(player *PlayerState) {
	if player != nil && ptrBool(player.ExtremeModeEnabled) {
		player.ExtremeWinStreak = intPtr(0)
	}
}

func (s *Server) applyExtremeWinStreakRisk(room *RoomState, winner *PlayerState) string {
	if winner == nil || !ptrBool(winner.ExtremeModeEnabled) || !room.Settings.EnableRanked {
		return ""
	}
	streak := ptrInt(winner.ExtremeWinStreak) + 1
	winner.ExtremeWinStreak = intPtr(streak)
	threshold := s.cfg.ExtremeMode.WinStreakThreshold
	if streak < threshold {
		return ""
	}
	if rand.Float64() >= s.cfg.ExtremeMode.WinStreakCrashChance {
		return ""
	}
	penalty := s.cfg.ExtremeMode.CrashTargetPoints
	if penalty < 1 {
		penalty = 1
	}
	s.updateRankedPoints(winner, -penalty)
	s.refreshPlayerSnapshots(winner)
	return fmt.Sprintf("；%s 极限连胜触发风险，额外扣 %d 分", playerShortName(winner), penalty)
}

func (s *Server) nameWarRenameQuota(player *PlayerState, now int64) int {
	if player.NameWarRenameWindowStartedAt == nil || now-*player.NameWarRenameWindowStartedAt >= 10_800_000 {
		player.NameWarRenameWindowStartedAt = int64Ptr(now)
		player.NameWarRenameCount = intPtr(0)
	}
	return 3 - ptrInt(player.NameWarRenameCount)
}

func (s *Server) nameWarPenaltyThreshold() int {
	th := s.cfg.NameWar.PenaltyThreshold
	if th >= 0 {
		// 配置异常时的安全兜底：名争失格必须是负分线。
		return defaultNameWarPenaltyThreshold
	}
	return th
}

func (s *Server) isNameWarRenameTarget(player types.PublicPlayer) bool {
	// 用真实分（SortRankedPoints）判定，不用展示封顶后的 RankedPoints。
	return ptrBool(player.NameWarEnabled) && ptrBool(player.NameWarAllowRename) &&
		ptrBool(player.NameWarPunished) && player.Stats.SortRankedPoints <= s.nameWarPenaltyThreshold()
}
