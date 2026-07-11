package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

type persistedPlayer struct {
	ID                           string      `json:"id"`
	PlayerID                     string      `json:"playerId"`
	PlayerSecretHash             string      `json:"playerSecretHash"`
	Name                         string      `json:"name"`
	GenderID                     string      `json:"genderId"`
	NameWarEnabled               *bool       `json:"nameWarEnabled,omitempty"`
	NameWarAllowRename           *bool       `json:"nameWarAllowRename,omitempty"`
	NameWarToggledAt             *int64      `json:"nameWarToggledAt,omitempty"`
	NameWarOriginalName          string      `json:"nameWarOriginalName,omitempty"`
	NameWarPenaltyName           string      `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished              *bool       `json:"nameWarPunished,omitempty"`
	NameWarRenameProtectedUntil  *int64      `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedBy             string      `json:"nameWarRenamedBy,omitempty"`
	NameWarRenamedByName         string      `json:"nameWarRenamedByName,omitempty"`
	NameWarRenameWindowStartedAt *int64      `json:"nameWarRenameWindowStartedAt,omitempty"`
	NameWarRenameCount           *int        `json:"nameWarRenameCount,omitempty"`
	GiveawayEnabled              *bool       `json:"giveawayEnabled,omitempty"`
	GiveawayValue                *float64    `json:"giveawayValue,omitempty"`
	GiveawayClicks               *int        `json:"giveawayClicks,omitempty"`
	RankMultiplierUnlocked       *bool       `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled           *bool       `json:"extremeModeEnabled,omitempty"`
	ExtremeModeToggledAt         *int64      `json:"extremeModeToggledAt,omitempty"`
	ExtremeModeCooldownUntil      *int64      `json:"extremeModeCooldownUntil,omitempty"`
	ExtremeWinStreak             *int        `json:"extremeWinStreak,omitempty"`
	ExtremeLastDecayHour         *int64      `json:"extremeLastDecayHour,omitempty"`
	Stats                        types.PublicStats  `json:"stats"`
	OthelloStats                 types.OthelloStats `json:"othelloStats"`
	CreatedAt                    int64       `json:"createdAt,omitempty"`
	LastSeenAt                   int64       `json:"lastSeenAt,omitempty"`
}

func (s *Server) serializePlayers() []persistedPlayer {
	var out []persistedPlayer
	for _, p := range s.players {
		if !p.Persistent || p.PlayerID == "" || p.PlayerSecretHash == "" {
			continue
		}
		out = append(out, persistedPlayer{
			ID: p.ID, PlayerID: p.PlayerID, PlayerSecretHash: p.PlayerSecretHash,
			Name: p.Name, GenderID: p.GenderID,
			NameWarEnabled: p.NameWarEnabled, NameWarAllowRename: p.NameWarAllowRename,
			NameWarToggledAt: p.NameWarToggledAt, NameWarOriginalName: p.NameWarOriginalName,
			NameWarPenaltyName: p.NameWarPenaltyName, NameWarPunished: p.NameWarPunished,
			NameWarRenameProtectedUntil: p.NameWarRenameProtectedUntil,
			NameWarRenamedBy: p.NameWarRenamedBy, NameWarRenamedByName: p.NameWarRenamedByName,
			NameWarRenameWindowStartedAt: p.NameWarRenameWindowStartedAt,
			NameWarRenameCount: p.NameWarRenameCount,
			GiveawayEnabled: p.GiveawayEnabled, GiveawayValue: p.GiveawayValue, GiveawayClicks: p.GiveawayClicks,
			RankMultiplierUnlocked: p.RankMultiplierUnlocked,
			ExtremeModeEnabled: p.ExtremeModeEnabled, ExtremeModeToggledAt: p.ExtremeModeToggledAt,
			ExtremeModeCooldownUntil: p.ExtremeModeCooldownUntil, ExtremeWinStreak: p.ExtremeWinStreak,
			ExtremeLastDecayHour: p.ExtremeLastDecayHour,
			Stats: p.Stats, OthelloStats: p.OthelloStats,
			CreatedAt: p.CreatedAt, LastSeenAt: p.LastSeenAt,
		})
	}
	return out
}

func (s *Server) loadPlayersFromDisk() {
	data, err := os.ReadFile(s.playersFile)
	if err != nil {
		return
	}
	var list []persistedPlayer
	if err := json.Unmarshal(data, &list); err != nil {
		fmt.Println("[players] load failed:", err)
		return
	}
	now := nowMs()
	for _, item := range list {
		if item.ID == "" || item.PlayerID == "" || item.PlayerSecretHash == "" {
			continue
		}
		if s.players[item.ID] != nil || s.playerIdToID[item.PlayerID] != "" {
			continue
		}
		name := item.Name
		if name == "" {
			name = "玩家"
		}
		genderID := item.GenderID
		if genderID == "" {
			genderID = "male"
		}
		gender := s.genderInfo(genderID)
		title := item.Stats.Title
		if title == "" {
			title = s.randomTitleFromSegment(s.titleSegmentFor(0), gender.FactionID)
		}
		othello := item.OthelloStats
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
				FactionColors: gender.FactionColors,
				Connected: false,
				NameWarOriginalName: origName,
				NameWarEnabled: item.NameWarEnabled,
				NameWarAllowRename: item.NameWarAllowRename,
				NameWarToggledAt: item.NameWarToggledAt,
				NameWarPenaltyName: item.NameWarPenaltyName,
				NameWarPunished: item.NameWarPunished,
				NameWarRenameProtectedUntil: item.NameWarRenameProtectedUntil,
				NameWarRenamedBy: item.NameWarRenamedBy,
				NameWarRenamedByName: item.NameWarRenamedByName,
				NameWarRenameWindowStartedAt: item.NameWarRenameWindowStartedAt,
				NameWarRenameCount: item.NameWarRenameCount,
				GiveawayEnabled: item.GiveawayEnabled,
				GiveawayValue: orFloat(item.GiveawayValue, 0),
				GiveawayClicks: orInt(item.GiveawayClicks, 0),
				GiveawayBoardLikes: intPtr(0),
				GiveawayBoardDislikes: intPtr(0),
				GiveawayVoteLikesThisHour: intPtr(0),
				GiveawayVoteDislikesThisHour: intPtr(0),
				RankMultiplierUnlocked: item.RankMultiplierUnlocked,
				ExtremeModeEnabled: item.ExtremeModeEnabled,
				ExtremeModeToggledAt: item.ExtremeModeToggledAt,
				ExtremeModeCooldownUntil: item.ExtremeModeCooldownUntil,
				ExtremeWinStreak: orInt(item.ExtremeWinStreak, 0),
				ExtremeLastDecayHour: orInt64(item.ExtremeLastDecayHour, currentExtremeDecayHour(now)),
				Stats: types.PublicStats{
					Wins: item.Stats.Wins, Losses: item.Stats.Losses, Draws: item.Stats.Draws,
					Punishments: item.Stats.Punishments, RankedPoints: item.Stats.RankedPoints,
					Title: title, TitleSegmentID: item.Stats.TitleSegmentID,
				},
				OthelloStats: othello,
			},
			Token:            randomID(),
			Persistent:       true,
			PlayerID:         item.PlayerID,
			PlayerSecretHash: item.PlayerSecretHash,
			CreatedAt:        createdAt,
			LastSeenAt:       lastSeenAt,
		}
		player.DisplayName = formatDisplayName(player)
		s.players[player.ID] = player
		s.playerIdToID[player.PlayerID] = player.ID
		s.tokenToPlayer[player.Token] = player.ID
	}
	fmt.Printf("[players] loaded %d players\n", len(s.players))
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

func (s *Server) writeSnapshot() {
	snapshot := s.serializePlayers()
	data, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		fmt.Println("[players] persist failed:", err)
		s.persistDirty = true
		return
	}
	if err := os.MkdirAll(s.dataDir, 0o755); err != nil {
		fmt.Println("[players] persist failed:", err)
		s.persistDirty = true
		return
	}
	tmp := s.playersFile + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		fmt.Println("[players] persist failed:", err)
		s.persistDirty = true
		return
	}
	if err := os.Rename(tmp, s.playersFile); err != nil {
		fmt.Println("[players] persist failed:", err)
		s.persistDirty = true
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
				s.mu.Lock()
				s.writeSnapshot()
				s.mu.Unlock()
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
			s.mu.Lock()
			s.writeSnapshot()
			s.mu.Unlock()
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
		s.mu.Lock()
		s.writeSnapshot()
		s.mu.Unlock()
	}
}
