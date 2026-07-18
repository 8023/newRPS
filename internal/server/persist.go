package server

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// persistedPlayer 是磁盘上的玩家档案形状：
// - 写路径：内存 → SQLite（players + player_secrets）
// - 读路径：SQLite；若仍存在旧版 players.json 则自动迁移进库（上线环境也会走此逻辑）
type persistedPlayer struct {
	ID       string `json:"id"`
	PlayerID string `json:"playerId"`
	// PlayerSecretHash：旧版单值哈希字段，仅用于迁移期兼容读取，新写入不再产生。
	// 一个月后（见 README）应整体删除，届时这个字段与相关兼容分支一起清理。
	PlayerSecretHash             string   `json:"playerSecretHash,omitempty"`
	PlayerSecrets                []string `json:"playerSecrets,omitempty"`
	ClaimKey                     string   `json:"claimKey,omitempty"`
	Name                         string   `json:"name"`
	GenderID                     string   `json:"genderId"`
	AvatarURL                    string   `json:"avatarUrl,omitempty"`
	NameWarEnabled               *bool    `json:"nameWarEnabled,omitempty"`
	NameWarAllowRename           *bool    `json:"nameWarAllowRename,omitempty"`
	NameWarToggledAt             *int64   `json:"nameWarToggledAt,omitempty"`
	NameWarOriginalName          string   `json:"nameWarOriginalName,omitempty"`
	NameWarPenaltyName           string   `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished              *bool    `json:"nameWarPunished,omitempty"`
	NameWarRenameProtectedUntil  *int64   `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedBy             string   `json:"nameWarRenamedBy,omitempty"`
	NameWarRenamedByName         string   `json:"nameWarRenamedByName,omitempty"`
	NameWarRenameWindowStartedAt *int64   `json:"nameWarRenameWindowStartedAt,omitempty"`
	NameWarRenameCount           *int     `json:"nameWarRenameCount,omitempty"`
	GiveawayEnabled              *bool    `json:"giveawayEnabled,omitempty"`
	GiveawayValue                *float64 `json:"giveawayValue,omitempty"`
	GiveawayClicks               *int     `json:"giveawayClicks,omitempty"`
	RankMultiplierUnlocked       *bool    `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled           *bool    `json:"extremeModeEnabled,omitempty"`
	ExtremeModeToggledAt         *int64   `json:"extremeModeToggledAt,omitempty"`
	ExtremeModeCooldownUntil      *int64   `json:"extremeModeCooldownUntil,omitempty"`
	ExtremeWinStreak             *int     `json:"extremeWinStreak,omitempty"`
	ExtremeLastDecayHour         *int64   `json:"extremeLastDecayHour,omitempty"`
	PushMentionEnabled           *bool    `json:"pushMentionEnabled,omitempty"`
	PushTurnEnabled              *bool    `json:"pushTurnEnabled,omitempty"`
	PushSeatEnabled              *bool    `json:"pushSeatEnabled,omitempty"`
	Stats                        types.PublicStats `json:"stats"`
	GameStats                    types.GameStats   `json:"gameStats"`
	// OthelloStats：仅用于加载旧 JSON 存档迁移，新写入不再产生。
	LegacyOthelloStats *struct {
		Wins     int `json:"wins"`
		Losses   int `json:"losses"`
		Draws    int `json:"draws"`
		Games    int `json:"games"`
		Captured int `json:"captured"`
		Lost     int `json:"lost"`
	} `json:"othelloStats,omitempty"`
	CreatedAt  int64 `json:"createdAt,omitempty"`
	LastSeenAt int64 `json:"lastSeenAt,omitempty"`
}

func (s *Server) serializePlayers() []persistedPlayer {
	var out []persistedPlayer
	for _, p := range s.players {
		if !p.Persistent || p.PlayerID == "" || (len(p.PlayerSecrets) == 0 && p.PlayerSecretHash == "") {
			continue
		}
		p.SyncTotalsFromGameStats()
		out = append(out, persistedPlayer{
			ID: p.ID, PlayerID: p.PlayerID, PlayerSecretHash: p.PlayerSecretHash,
			PlayerSecrets: p.PlayerSecrets, ClaimKey: p.ClaimKey,
			Name: p.Name, GenderID: p.GenderID, AvatarURL: p.AvatarURL,
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
			PushMentionEnabled: p.PushMentionEnabled, PushTurnEnabled: p.PushTurnEnabled, PushSeatEnabled: p.PushSeatEnabled,
			Stats: p.Stats, GameStats: p.GameStats,
			CreatedAt: p.CreatedAt, LastSeenAt: p.LastSeenAt,
		})
	}
	return out
}

// migrateGameStats 从旧 othelloStats + 总榜残差迁到五游戏分项。
func migrateGameStats(item persistedPlayer) types.GameStats {
	gs := item.GameStats
	hasNew := gs.RPS.Wins+gs.RPS.Losses+gs.RPS.Draws+
		gs.Othello.Wins+gs.Othello.Losses+gs.Othello.Draws+
		gs.TicTacToe.Wins+gs.TicTacToe.Losses+gs.TicTacToe.Draws+
		gs.Gomoku.Wins+gs.Gomoku.Losses+gs.Gomoku.Draws+
		gs.LiarsDice.Wins+gs.LiarsDice.Losses+gs.LiarsDice.Draws > 0
	if hasNew {
		return gs
	}
	if item.LegacyOthelloStats != nil {
		gs.Othello = types.GameWLD{
			Wins: item.LegacyOthelloStats.Wins, Losses: item.LegacyOthelloStats.Losses, Draws: item.LegacyOthelloStats.Draws,
		}
	}
	// 旧总榜减去黑白棋分项后的残差归入锤子剪刀布（无法再拆历史其它游戏）
	resW := item.Stats.Wins - gs.Othello.Wins
	resL := item.Stats.Losses - gs.Othello.Losses
	resD := item.Stats.Draws - gs.Othello.Draws
	if resW < 0 {
		resW = 0
	}
	if resL < 0 {
		resL = 0
	}
	if resD < 0 {
		resD = 0
	}
	gs.RPS = types.GameWLD{Wins: resW, Losses: resL, Draws: resD}
	return gs
}

// loadPlayersFromDisk：优先 SQLite；若仍有旧 players.json 则幂等导入缺失玩家并改名备份。
// 上线环境只要带上旧 JSON，启动时同样会自动迁库。
func (s *Server) loadPlayersFromDisk() {
	if s.playerDB != nil {
		if err := s.loadPlayersFromSQLite(); err != nil {
			s.errorLog("players_sqlite_load_failed", err.Error())
		}
	}
	// 无论库是否已有数据，都尝试导入 JSON 中尚未入库的身份（幂等）。
	if n, err := s.migratePlayersJSONIfNeeded(); err != nil {
		s.errorLog("players_json_migrate_failed", err.Error())
	} else if n > 0 {
		s.activityLog("system", []string{"event", "detail"}, []string{
			time.Now().Format(time.RFC3339), "players_json_migrated", fmt.Sprintf("%d", n),
		})
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

// migratePlayersJSONIfNeeded 读取 data/players.json，把库中尚不存在的玩家写入 SQLite 与内存。
// 成功处理（含「文件在但无需新增」）后将 JSON 改名为 players.json.migrated，避免每次启动重复扫。
// 返回新迁入人数。
func (s *Server) migratePlayersJSONIfNeeded() (int, error) {
	data, err := os.ReadFile(s.playersFile)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, err
	}
	var list []persistedPlayer
	if err := json.Unmarshal(data, &list); err != nil {
		return 0, fmt.Errorf("parse players.json: %w", err)
	}
	imported := 0
	var toUpsert []persistedPlayer
	for _, item := range list {
		if item.ID == "" || item.PlayerID == "" || (len(item.PlayerSecrets) == 0 && item.PlayerSecretHash == "") {
			continue
		}
		if s.players[item.ID] != nil || s.playerIdToID[item.PlayerID] != "" {
			continue
		}
		item.GameStats = migrateGameStats(item)
		totalW, totalL, totalD := item.GameStats.Total()
		item.Stats.Wins, item.Stats.Losses, item.Stats.Draws = totalW, totalL, totalD
		if !s.ingestPersistedPlayer(item) {
			continue
		}
		toUpsert = append(toUpsert, item)
		imported++
	}
	if s.playerDB != nil && len(toUpsert) > 0 {
		if err := s.playerDB.upsertMany(toUpsert); err != nil {
			return imported, fmt.Errorf("upsert migrated players: %w", err)
		}
	}
	// 已成功解析并处理完毕：改名保留备份，便于回滚；下次启动不再重复读。
	// 即便 imported==0（库里已有全部人），也改名，避免反复扫大 JSON。
	bak := s.playersFile + ".migrated"
	if err := os.Rename(s.playersFile, bak); err != nil {
		// 改名失败不阻断：下次启动会因 id 已在库中而跳过，仍幂等。
		s.errorLog("players_json_rename_failed", err.Error())
	} else {
		// 附带清理旧的临时写盘文件
		_ = os.Remove(s.playersFile + ".tmp")
	}
	return imported, nil
}

// ingestPersistedPlayer 把磁盘档案灌进内存索引；已存在则跳过。返回是否新灌入。
func (s *Server) ingestPersistedPlayer(item persistedPlayer) bool {
	if item.ID == "" || item.PlayerID == "" || (len(item.PlayerSecrets) == 0 && item.PlayerSecretHash == "") {
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
	if genderID == "" {
		genderID = "male"
	}
	gender := s.genderInfo(genderID)
	title := item.Stats.Title
	if title == "" {
		title = s.randomTitleFromSegment(s.titleSegmentFor(0), gender.FactionID)
	}
	gameStats := migrateGameStats(item)
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
			FactionColors: gender.FactionColors,
			AvatarURL:    item.AvatarURL,
			Connected:    false,
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
				Wins: totalW, Losses: totalL, Draws: totalD,
				Punishments: item.Stats.Punishments, RankedPoints: item.Stats.RankedPoints,
				Title: title, TitleSegmentID: item.Stats.TitleSegmentID,
			},
			GameStats: gameStats,
		},
		Token:              randomID(),
		Persistent:         true,
		PlayerID:           item.PlayerID,
		PlayerSecretHash:   item.PlayerSecretHash,
		PlayerSecrets:      append([]string{}, item.PlayerSecrets...),
		ClaimKey:           item.ClaimKey,
		PushMentionEnabled: item.PushMentionEnabled,
		PushTurnEnabled:    item.PushTurnEnabled,
		PushSeatEnabled:    item.PushSeatEnabled,
		CreatedAt:          createdAt,
		LastSeenAt:         lastSeenAt,
	}
	if player.ClaimKey == "" {
		player.ClaimKey = randomID()
	}
	player.DisplayName = formatDisplayName(player)
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

