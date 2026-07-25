package server

import (
	"encoding/json"
	"os"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// persistedPlayer 是磁盘上的玩家档案形状（写路径：内存 → SQLite players + player_secrets；
// 读路径：SQLite）。
type persistedPlayer struct {
	ID                           string            `json:"id"`
	PlayerID                     string            `json:"playerId"`
	PlayerSecrets                []string          `json:"playerSecrets,omitempty"`
	ClaimKey                     string            `json:"claimKey,omitempty"`
	Name                         string            `json:"name"`
	GenderID                     string            `json:"genderId"`
	CustomGenderLabel            string            `json:"customGenderLabel,omitempty"`
	FactionID                    string            `json:"factionId"`
	AvatarURL                    string            `json:"avatarUrl,omitempty"`
	NameWarEnabled               *bool             `json:"nameWarEnabled,omitempty"`
	NameWarAllowRename           *bool             `json:"nameWarAllowRename,omitempty"`
	NameWarToggledAt             *int64            `json:"nameWarToggledAt,omitempty"`
	NameWarOriginalName          string            `json:"nameWarOriginalName,omitempty"`
	NameWarPenaltyName           string            `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished              *bool             `json:"nameWarPunished,omitempty"`
	NameWarRenameProtectedUntil  *int64            `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedBy             string            `json:"nameWarRenamedBy,omitempty"`
	NameWarRenamedByName         string            `json:"nameWarRenamedByName,omitempty"`
	NameWarRenameWindowStartedAt *int64            `json:"nameWarRenameWindowStartedAt,omitempty"`
	NameWarRenameCount           *int              `json:"nameWarRenameCount,omitempty"`
	GiveawayEnabled              *bool             `json:"giveawayEnabled,omitempty"`
	GiveawayValue                *float64          `json:"giveawayValue,omitempty"`
	GiveawayClicks               *int              `json:"giveawayClicks,omitempty"`
	RankMultiplierUnlocked       *bool             `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled           *bool             `json:"extremeModeEnabled,omitempty"`
	ExtremeModeToggledAt         *int64            `json:"extremeModeToggledAt,omitempty"`
	ExtremeModeCooldownUntil     *int64            `json:"extremeModeCooldownUntil,omitempty"`
	ExtremeWinStreak             *int              `json:"extremeWinStreak,omitempty"`
	ExtremeLastDecayHour         *int64            `json:"extremeLastDecayHour,omitempty"`
	RankedLastDecayDay           *int64            `json:"rankedLastDecayDay,omitempty"`
	PushMentionEnabled           *bool             `json:"pushMentionEnabled,omitempty"`
	PushTurnEnabled              *bool             `json:"pushTurnEnabled,omitempty"`
	PushSeatEnabled              *bool             `json:"pushSeatEnabled,omitempty"`
	BondMasterEnabled            *bool             `json:"bondMasterEnabled,omitempty"`
	BondPetEnabled               *bool             `json:"bondPetEnabled,omitempty"`
	BondPublicDisplay            *bool             `json:"bondPublicDisplay,omitempty"`
	Stats                        types.PublicStats `json:"stats"`
	GameStats                    types.GameStats   `json:"gameStats"`
	CreatedAt                    int64             `json:"createdAt,omitempty"`
	LastSeenAt                   int64             `json:"lastSeenAt,omitempty"`
}

func (s *Server) serializePlayers() []persistedPlayer {
	var out []persistedPlayer
	for _, p := range s.players {
		if !p.Persistent || p.PlayerID == "" {
			continue
		}
		p.SyncTotalsFromGameStats()
		customGenderLabel := ""
		if p.GenderID == "" {
			customGenderLabel = p.GenderLabel
		}
		out = append(out, persistedPlayer{
			ID: p.ID, PlayerID: p.PlayerID,
			PlayerSecrets: p.PlayerSecrets, ClaimKey: p.ClaimKey,
			Name: p.Name, GenderID: p.GenderID, CustomGenderLabel: customGenderLabel, FactionID: p.FactionID, AvatarURL: p.AvatarURL,
			NameWarEnabled: p.NameWarEnabled, NameWarAllowRename: p.NameWarAllowRename,
			NameWarToggledAt: p.NameWarToggledAt, NameWarOriginalName: p.NameWarOriginalName,
			NameWarPenaltyName: p.NameWarPenaltyName, NameWarPunished: p.NameWarPunished,
			NameWarRenameProtectedUntil: p.NameWarRenameProtectedUntil,
			NameWarRenamedBy:            p.NameWarRenamedBy, NameWarRenamedByName: p.NameWarRenamedByName,
			NameWarRenameWindowStartedAt: p.NameWarRenameWindowStartedAt,
			NameWarRenameCount:           p.NameWarRenameCount,
			GiveawayEnabled:              p.GiveawayEnabled, GiveawayValue: p.GiveawayValue, GiveawayClicks: p.GiveawayClicks,
			RankMultiplierUnlocked: p.RankMultiplierUnlocked,
			ExtremeModeEnabled:     p.ExtremeModeEnabled, ExtremeModeToggledAt: p.ExtremeModeToggledAt,
			ExtremeModeCooldownUntil: p.ExtremeModeCooldownUntil, ExtremeWinStreak: p.ExtremeWinStreak,
			ExtremeLastDecayHour: p.ExtremeLastDecayHour,
			RankedLastDecayDay:   p.RankedLastDecayDay,
			PushMentionEnabled:   p.PushMentionEnabled, PushTurnEnabled: p.PushTurnEnabled, PushSeatEnabled: p.PushSeatEnabled,
			BondMasterEnabled: p.BondMasterEnabled, BondPetEnabled: p.BondPetEnabled, BondPublicDisplay: p.BondPublicDisplay,
			Stats: p.Stats, GameStats: p.GameStats,
			CreatedAt: p.CreatedAt, LastSeenAt: p.LastSeenAt,
		})
	}
	return out
}

// loadPlayersFromDisk 从 SQLite 加载全部玩家档案进内存。
func (s *Server) loadPlayersFromDisk() {
	if s.playerDB != nil {
		if err := s.loadPlayersFromSQLite(); err != nil {
			s.errorLog("players_sqlite_load_failed", err.Error())
		}
	}
}

func (s *Server) loadPlayersFromSQLite() error {
	rows, err := s.playerDB.loadAll()
	if err != nil {
		return err
	}
	for _, row := range rows {
		s.ingestPersistedPlayer(row.item)
	}
	return nil
}

// ingestPersistedPlayer 把磁盘档案灌进内存索引；已存在则跳过。返回是否新灌入。
func (s *Server) ingestPersistedPlayer(item persistedPlayer) bool {
	if item.ID == "" || item.PlayerID == "" {
		return false
	}
	if s.players[item.ID] != nil || s.playerIdToID[item.PlayerID] != "" {
		return false
	}
	now := nowMs()
	name := item.Name
	if name == "" {
		name = "玩家"
	}
	genderID := item.GenderID
	if genderID == "" && item.CustomGenderLabel == "" {
		genderID = "male"
	}
	gender := s.resolveGender(genderID, item.CustomGenderLabel, item.FactionID)
	title := item.Stats.Title
	if title == "" {
		title = s.randomTitleFromSegment(s.titleSegmentFor(0), gender.FactionID)
	}
	gameStats := item.GameStats
	totalW, totalL, totalD := gameStats.Total()
	createdAt := item.CreatedAt
	if createdAt == 0 {
		createdAt = now
	}
	lastSeenAt := item.LastSeenAt
	if lastSeenAt == 0 {
		lastSeenAt = now
	}
	origName := item.NameWarOriginalName
	if origName == "" {
		origName = name
	}
	player := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID: idOr(item.ID), Name: name,
			GenderID: gender.GenderID, GenderLabel: gender.GenderLabel,
			FactionID: gender.FactionID, FactionLabel: gender.FactionLabel,
			FactionColors:                gender.FactionColors,
			AvatarURL:                    item.AvatarURL,
			Connected:                    false,
			NameWarOriginalName:          origName,
			NameWarEnabled:               item.NameWarEnabled,
			NameWarAllowRename:           item.NameWarAllowRename,
			NameWarToggledAt:             item.NameWarToggledAt,
			NameWarPenaltyName:           item.NameWarPenaltyName,
			NameWarPunished:              item.NameWarPunished,
			NameWarRenameProtectedUntil:  item.NameWarRenameProtectedUntil,
			NameWarRenamedBy:             item.NameWarRenamedBy,
			NameWarRenamedByName:         item.NameWarRenamedByName,
			NameWarRenameWindowStartedAt: item.NameWarRenameWindowStartedAt,
			NameWarRenameCount:           item.NameWarRenameCount,
			GiveawayEnabled:              item.GiveawayEnabled,
			GiveawayValue:                orFloat(item.GiveawayValue, 0),
			GiveawayClicks:               orInt(item.GiveawayClicks, 0),
			GiveawayBoardLikes:           intPtr(0),
			GiveawayBoardDislikes:        intPtr(0),
			GiveawayVoteLikesThisHour:    intPtr(0),
			GiveawayVoteDislikesThisHour: intPtr(0),
			RankMultiplierUnlocked:       item.RankMultiplierUnlocked,
			ExtremeModeEnabled:           item.ExtremeModeEnabled,
			ExtremeModeToggledAt:         item.ExtremeModeToggledAt,
			ExtremeModeCooldownUntil:     item.ExtremeModeCooldownUntil,
			ExtremeWinStreak:             orInt(item.ExtremeWinStreak, 0),
			ExtremeLastDecayHour:         orInt64(item.ExtremeLastDecayHour, currentExtremeDecayHour(now)),
			RankedLastDecayDay:           orInt64(item.RankedLastDecayDay, currentRankedDecayDay(now)),
			BondMasterEnabled:            item.BondMasterEnabled,
			BondPetEnabled:               item.BondPetEnabled,
			BondPublicDisplay:            item.BondPublicDisplay,
			Stats: types.PublicStats{
				Wins: totalW, Losses: totalL, Draws: totalD,
				Punishments: item.Stats.Punishments, RankedPoints: item.Stats.RankedPoints,
				HighestScore: item.Stats.HighestScore, LowestScore: item.Stats.LowestScore,
				Title: title, TitleSegmentID: item.Stats.TitleSegmentID, TitleCustom: item.Stats.TitleCustom,
			},
			GameStats: gameStats,
		},
		Token:              randomID(),
		Persistent:         true,
		PlayerID:           item.PlayerID,
		PlayerSecrets:      append([]string{}, item.PlayerSecrets...),
		ClaimKey:           item.ClaimKey,
		PushMentionEnabled: item.PushMentionEnabled,
		PushTurnEnabled:    item.PushTurnEnabled,
		PushSeatEnabled:    item.PushSeatEnabled,
		CreatedAt:          createdAt,
		LastSeenAt:         lastSeenAt,
	}
	if player.ClaimKey == "" {
		player.ClaimKey = randomClaimKey()
	}
	player.DisplayName = formatDisplayName(player)
	// 旧版 players.json 迁移进库时没有历史最高/最低分字段：以当前排位分兜底，
	// 保证展示上"历史最高"不会低于"当前"（SQLite 直接加载的行已在迁移 SQL 里回填过，这里是幂等的）。
	recordRankedExtremes(player)
	s.players[player.ID] = player
	s.playerIdToID[player.PlayerID] = player.ID
	s.tokenToPlayer[player.Token] = player.ID
	return true
}

func idOr(s string) string { return s }

func orFloat(p *float64, d float64) *float64 {
	if p == nil {
		return floatPtr(d)
	}
	return p
}
func orInt(p *int, d int) *int {
	if p == nil {
		return intPtr(d)
	}
	return p
}
func orInt64(p *int64, d int64) *int64 {
	if p == nil {
		return int64Ptr(d)
	}
	return p
}

func (s *Server) markPersistDirty() {
	s.persistMu.Lock()
	s.persistDirty = true
	s.persistMu.Unlock()
}

func (s *Server) writeSnapshot() {
	// 仅在序列化期间持 s.mu；磁盘 I/O 放到锁外，避免拖慢全服。
	s.mu.Lock()
	snapshot := s.serializePlayers()
	s.mu.Unlock()

	if s.playerDB == nil {
		// 库不可用时降级：仍写 JSON，避免测试环境/库损坏时彻底丢档。
		s.writePlayersJSONFallback(snapshot)
		return
	}
	if err := s.playerDB.upsertMany(snapshot); err != nil {
		s.errorLog("players_sqlite_persist_failed", err.Error())
		s.markPersistDirty()
		// 二次降级写 JSON，尽量保住数据
		s.writePlayersJSONFallback(snapshot)
	}
}

// writePlayersJSONFallback：SQLite 不可用或写失败时的保底路径（保留旧文件形态）。
func (s *Server) writePlayersJSONFallback(snapshot []persistedPlayer) {
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		s.errorLog("players_persist_failed", err.Error())
		s.markPersistDirty()
		return
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		s.errorLog("players_persist_failed", err.Error())
		s.markPersistDirty()
		return
	}
	tmp := s.playersFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		s.errorLog("players_persist_failed", err.Error())
		s.markPersistDirty()
		return
	}
	if err := os.Rename(tmp, s.playersFile); err != nil {
		s.errorLog("players_persist_failed", err.Error())
		s.markPersistDirty()
	}
}

func (s *Server) requestPersist(mode string) {
	s.persistMu.Lock()
	defer s.persistMu.Unlock()
	s.persistDirty = true
	if mode == "important" {
		if s.immediateScheduled {
			return
		}
		s.immediateScheduled = true
		go func() {
			s.persistMu.Lock()
			s.immediateScheduled = false
			if s.persistDirty {
				s.persistDirty = false
				s.persistMu.Unlock()
				s.writeSnapshot()
				return
			}
			s.persistMu.Unlock()
		}()
		return
	}
	if s.persistScheduled {
		return
	}
	s.persistScheduled = true
	go func() {
		time.Sleep(3 * time.Second)
		s.persistMu.Lock()
		s.persistScheduled = false
		if s.persistDirty {
			s.persistDirty = false
			s.persistMu.Unlock()
			s.writeSnapshot()
			return
		}
		s.persistMu.Unlock()
	}()
}

func (s *Server) flushPersist() {
	s.persistMu.Lock()
	dirty := s.persistDirty
	s.persistDirty = false
	s.persistMu.Unlock()
	if dirty {
		s.writeSnapshot()
	}
}
