package server

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) publicConfig() types.AppConfig {
	cfg := s.cfg
	cfg.Site.AdminPassword = ""
	return sanitizePublicConfig(cfg)
}

func (s *Server) adminPasswordMatches(password string) bool {
	return s.cfg.Site.AdminPassword != "" && password == s.cfg.Site.AdminPassword
}

func (s *Server) titleSegmentFor(points int) *types.TitleSegment {
	for i := range s.cfg.Titles {
		item := &s.cfg.Titles[i]
		if points >= item.Min && points <= item.Max {
			return item
		}
	}
	if len(s.cfg.Titles) > 0 {
		return &s.cfg.Titles[0]
	}
	return nil
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
	// Match TS: Math.max(player.stats.rankedPoints, -999)
	ep := player.Stats.RankedPoints
	if ep < -999 {
		ep = -999
	}
	segment := s.titleSegmentFor(ep)
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
		} else if player.Stats.RankedPoints <= -1000 {
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
	if player.OthelloStats == (types.OthelloStats{}) && player.OthelloStats.Games == 0 {
		// always ensure othelloStats exists (zero value is fine)
	}
	p := player.PublicPlayer
	return p
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

func (s *Server) onlinePlayersFromIP(ipAddress string, exceptPlayerID string) int {
	n := 0
	for _, player := range s.players {
		if player.Connected && player.IPAddress == ipAddress && player.ID != exceptPlayerID {
			n++
		}
	}
	return n
}

func (s *Server) canCreateFromIP(ipAddress string) bool {
	now := nowMs()
	windowMs := int64(10 * 60 * 1000)
	attempts := s.ipCreateAttempts[ipAddress]
	filtered := attempts[:0]
	for _, t := range attempts {
		if now-t < windowMs {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= s.cfg.AccessControl.MaxCreatesPer10Min {
		s.ipCreateAttempts[ipAddress] = filtered
		return false
	}
	filtered = append(filtered, now)
	s.ipCreateAttempts[ipAddress] = filtered
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
			OthelloStats: freshOthelloStats(),
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
		player.PlayerSecretHash = hashSecret(identitySecret)
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

func (s *Server) updateRankedPoints(player *PlayerState, delta int) {
	minPoints := -999
	if ptrBool(player.NameWarEnabled) {
		minPoints = -1999
	}
	player.Stats.RankedPoints = clamp(player.Stats.RankedPoints+delta, minPoints, 999)
	s.refreshNameWarState(player, nowMs())
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	if player.Persistent {
		s.requestPersist("important")
	}
}

func (s *Server) setRankedPointsByAdmin(player *PlayerState, points int) {
	minPoints := -999
	if ptrBool(player.NameWarEnabled) {
		minPoints = -1999
	}
	player.Stats.RankedPoints = clamp(int(math.Round(float64(points))), minPoints, 999)
	s.refreshNameWarState(player, nowMs())
	if player.Persistent {
		s.requestPersist("important")
	}
}

func (s *Server) extremeSegmentID(points int) string {
	if points >= 750 {
		return "pos4"
	}
	if points >= 500 {
		return "pos3"
	}
	if points >= 250 {
		return "pos2"
	}
	if points >= 1 {
		return "pos1"
	}
	if points <= -750 {
		return "neg4"
	}
	if points <= -500 {
		return "neg3"
	}
	if points <= -250 {
		return "neg2"
	}
	if points <= -1 {
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
	now := nowMs()
	nextHour := (currentExtremeDecayHour(now) + 1) * 3_600_000
	delay := nextHour - now + 500
	if delay < 1000 {
		delay = 1000
	}
	timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		s.applyExtremeHourlyDecay()
		s.mu.Unlock()
		ticker := time.NewTicker(time.Hour)
		go func() {
			for range ticker.C {
				s.mu.Lock()
				s.applyExtremeHourlyDecay()
				s.mu.Unlock()
			}
		}()
	})
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

func isNameWarRenameTarget(player types.PublicPlayer) bool {
	return ptrBool(player.NameWarEnabled) && ptrBool(player.NameWarAllowRename) &&
		ptrBool(player.NameWarPunished) && player.Stats.RankedPoints <= -1000
}
