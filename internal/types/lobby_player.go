package types

// LobbyPlayer 大厅/全局广播用的精简玩家视图（不含投票窗口等私有过程字段）。
// 完整 PublicPlayer 通过 player:get / 进房快照 / 个人 me 下发。
type LobbyPlayer struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	GenderID            string       `json:"genderId"`
	GenderLabel         string       `json:"genderLabel"`
	FactionID           string       `json:"factionId"`
	FactionLabel        string       `json:"factionLabel"`
	FactionColors       GenderColors `json:"factionColors"`
	DisplayName         string       `json:"displayName"`
	Connected           bool         `json:"connected"`
	DisconnectedAt      *int64       `json:"disconnectedAt,omitempty"`
	DisconnectExpiresAt *int64       `json:"disconnectExpiresAt,omitempty"`

	NameWarEnabled              *bool  `json:"nameWarEnabled,omitempty"`
	NameWarPenaltyName          string `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished             *bool  `json:"nameWarPunished,omitempty"`
	NameWarAllowRename          *bool  `json:"nameWarAllowRename,omitempty"`
	NameWarRenameProtectedUntil *int64 `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedByName        string `json:"nameWarRenamedByName,omitempty"`

	GiveawayEnabled          *bool    `json:"giveawayEnabled,omitempty"`
	GiveawayValue            *float64 `json:"giveawayValue,omitempty"`
	GiveawayBoardText        string   `json:"giveawayBoardText,omitempty"`
	GiveawayBoardExpiresAt   *int64   `json:"giveawayBoardExpiresAt,omitempty"`
	GiveawayBoardLikes       *int     `json:"giveawayBoardLikes,omitempty"`
	GiveawayBoardDislikes    *int     `json:"giveawayBoardDislikes,omitempty"`
	// 投票额度仅影响操作者自己，大厅列表不需要每人的 voteWindow

	RankMultiplierUnlocked *bool `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled     *bool `json:"extremeModeEnabled,omitempty"`
	ExtremeWinStreak       *int  `json:"extremeWinStreak,omitempty"`
	ExtremeForceClosed     *bool `json:"extremeForceClosed,omitempty"`
	ExtremeForceClosedAt   *int64 `json:"extremeForceClosedAt,omitempty"`
	ExtremeRenameProtectedUntil *int64 `json:"extremeRenameProtectedUntil,omitempty"`
	ExtremeRenamedByName   string `json:"extremeRenamedByName,omitempty"`

	RoomID string      `json:"roomId,omitempty"`
	Stats  LobbyStats  `json:"stats"`
	OthelloStats OthelloStats `json:"othelloStats"`
}

// LobbyStats 大厅展示所需战绩字段。
type LobbyStats struct {
	Wins         int    `json:"wins"`
	Losses       int    `json:"losses"`
	Draws        int    `json:"draws"`
	Punishments  int    `json:"punishments"`
	RankedPoints int    `json:"rankedPoints"`
	Title        string `json:"title"`
}

// ToLobbyPlayer 从完整公开资料裁剪。
func ToLobbyPlayer(p PublicPlayer) LobbyPlayer {
	return LobbyPlayer{
		ID:                  p.ID,
		Name:                p.Name,
		GenderID:            p.GenderID,
		GenderLabel:         p.GenderLabel,
		FactionID:           p.FactionID,
		FactionLabel:        p.FactionLabel,
		FactionColors:       p.FactionColors,
		DisplayName:         p.DisplayName,
		Connected:           p.Connected,
		DisconnectedAt:      p.DisconnectedAt,
		DisconnectExpiresAt: p.DisconnectExpiresAt,
		NameWarEnabled:      p.NameWarEnabled,
		NameWarPenaltyName:  p.NameWarPenaltyName,
		NameWarPunished:     p.NameWarPunished,
		NameWarAllowRename:  p.NameWarAllowRename,
		NameWarRenameProtectedUntil: p.NameWarRenameProtectedUntil,
		NameWarRenamedByName:        p.NameWarRenamedByName,
		GiveawayEnabled:        p.GiveawayEnabled,
		GiveawayValue:          p.GiveawayValue,
		GiveawayBoardText:      p.GiveawayBoardText,
		GiveawayBoardExpiresAt: p.GiveawayBoardExpiresAt,
		GiveawayBoardLikes:     p.GiveawayBoardLikes,
		GiveawayBoardDislikes:  p.GiveawayBoardDislikes,
		RankMultiplierUnlocked: p.RankMultiplierUnlocked,
		ExtremeModeEnabled:     p.ExtremeModeEnabled,
		ExtremeWinStreak:       p.ExtremeWinStreak,
		ExtremeForceClosed:     p.ExtremeForceClosed,
		ExtremeForceClosedAt:   p.ExtremeForceClosedAt,
		ExtremeRenameProtectedUntil: p.ExtremeRenameProtectedUntil,
		ExtremeRenamedByName:   p.ExtremeRenamedByName,
		RoomID: p.RoomID,
		Stats: LobbyStats{
			Wins: p.Stats.Wins, Losses: p.Stats.Losses, Draws: p.Stats.Draws,
			Punishments: p.Stats.Punishments, RankedPoints: p.Stats.RankedPoints, Title: p.Stats.Title,
		},
		OthelloStats: p.OthelloStats,
	}
}

// AsPublicPlayer 将大厅视图提升为完整结构（缺省字段为零值），便于前端复用组件。
func (p LobbyPlayer) AsPublicPlayer() PublicPlayer {
	return PublicPlayer{
		ID: p.ID, Name: p.Name, GenderID: p.GenderID, GenderLabel: p.GenderLabel,
		FactionID: p.FactionID, FactionLabel: p.FactionLabel, FactionColors: p.FactionColors,
		DisplayName: p.DisplayName, Connected: p.Connected,
		DisconnectedAt: p.DisconnectedAt, DisconnectExpiresAt: p.DisconnectExpiresAt,
		NameWarEnabled: p.NameWarEnabled, NameWarPenaltyName: p.NameWarPenaltyName,
		NameWarPunished: p.NameWarPunished, NameWarAllowRename: p.NameWarAllowRename,
		NameWarRenameProtectedUntil: p.NameWarRenameProtectedUntil, NameWarRenamedByName: p.NameWarRenamedByName,
		GiveawayEnabled: p.GiveawayEnabled, GiveawayValue: p.GiveawayValue,
		GiveawayBoardText: p.GiveawayBoardText, GiveawayBoardExpiresAt: p.GiveawayBoardExpiresAt,
		GiveawayBoardLikes: p.GiveawayBoardLikes, GiveawayBoardDislikes: p.GiveawayBoardDislikes,
		RankMultiplierUnlocked: p.RankMultiplierUnlocked, ExtremeModeEnabled: p.ExtremeModeEnabled,
		ExtremeWinStreak: p.ExtremeWinStreak, ExtremeForceClosed: p.ExtremeForceClosed,
		ExtremeForceClosedAt: p.ExtremeForceClosedAt, ExtremeRenameProtectedUntil: p.ExtremeRenameProtectedUntil,
		ExtremeRenamedByName: p.ExtremeRenamedByName, RoomID: p.RoomID,
		Stats: PublicStats{
			Wins: p.Stats.Wins, Losses: p.Stats.Losses, Draws: p.Stats.Draws,
			Punishments: p.Stats.Punishments, RankedPoints: p.Stats.RankedPoints, Title: p.Stats.Title,
		},
		OthelloStats: p.OthelloStats,
	}
}
