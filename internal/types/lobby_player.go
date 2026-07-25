package types

// LobbyPlayer 大厅/全局广播用的精简玩家视图。
// 完整 PublicPlayer 通过 player:get / 进房快照 / 个人 me 下发。
// 投票额度（GiveawayVote*）也在此列出：前端自身的 me.player 是通过与
// lobby.players 中自己那条记录的 diff 合并来保持更新的（见 web/src/App.tsx），
// 如果这里裁掉这些字段，投票后广播回来的自身记录会把额度字段抹成 undefined，
// 导致前端一直显示满额度、且刷新倒计时永远不出现。
type LobbyPlayer struct {
	ID                  string       `json:"id"`
	Name                string       `json:"name"`
	GenderID            string       `json:"genderId"`
	GenderLabel         string       `json:"genderLabel"`
	FactionID           string       `json:"factionId"`
	FactionLabel        string       `json:"factionLabel"`
	FactionColors       GenderColors `json:"factionColors"`
	DisplayName         string       `json:"displayName"`
	AvatarURL           string       `json:"avatarUrl,omitempty"`
	Connected           bool         `json:"connected"`
	DisconnectedAt      *int64       `json:"disconnectedAt,omitempty"`
	DisconnectExpiresAt *int64       `json:"disconnectExpiresAt,omitempty"`

	NameWarEnabled              *bool  `json:"nameWarEnabled,omitempty"`
	NameWarPenaltyName          string `json:"nameWarPenaltyName,omitempty"`
	NameWarPunished             *bool  `json:"nameWarPunished,omitempty"`
	NameWarAllowRename          *bool  `json:"nameWarAllowRename,omitempty"`
	NameWarRenameProtectedUntil *int64 `json:"nameWarRenameProtectedUntil,omitempty"`
	NameWarRenamedByName        string `json:"nameWarRenamedByName,omitempty"`

	GiveawayEnabled              *bool    `json:"giveawayEnabled,omitempty"`
	GiveawayValue                *float64 `json:"giveawayValue,omitempty"`
	GiveawayBoardText            string   `json:"giveawayBoardText,omitempty"`
	GiveawayBoardExpiresAt       *int64   `json:"giveawayBoardExpiresAt,omitempty"`
	GiveawayBoardLikes           *int     `json:"giveawayBoardLikes,omitempty"`
	GiveawayBoardDislikes        *int     `json:"giveawayBoardDislikes,omitempty"`
	GiveawayVoteWindowStartedAt  *int64   `json:"giveawayVoteWindowStartedAt,omitempty"`
	GiveawayVoteLikesThisHour    *int     `json:"giveawayVoteLikesThisHour,omitempty"`
	GiveawayVoteDislikesThisHour *int     `json:"giveawayVoteDislikesThisHour,omitempty"`

	RankMultiplierUnlocked      *bool  `json:"rankMultiplierUnlocked,omitempty"`
	ExtremeModeEnabled          *bool  `json:"extremeModeEnabled,omitempty"`
	ExtremeWinStreak            *int   `json:"extremeWinStreak,omitempty"`
	ExtremeForceClosed          *bool  `json:"extremeForceClosed,omitempty"`
	ExtremeForceClosedAt        *int64 `json:"extremeForceClosedAt,omitempty"`
	ExtremeRenameProtectedUntil *int64 `json:"extremeRenameProtectedUntil,omitempty"`
	ExtremeRenamedByName        string `json:"extremeRenamedByName,omitempty"`

	RoomID    string     `json:"roomId,omitempty"`
	Stats     LobbyStats `json:"stats"`
	GameStats GameStats  `json:"gameStats"`
}

// LobbyStats 大厅展示所需战绩字段。
type LobbyStats struct {
	Wins             int    `json:"wins"`
	Losses           int    `json:"losses"`
	Draws            int    `json:"draws"`
	Punishments      int    `json:"punishments"`
	RankedPoints     int    `json:"rankedPoints"`
	HighestScore     int    `json:"highestScore"`
	LowestScore      int    `json:"lowestScore"`
	SortRankedPoints int    `json:"sortRankedPoints"`
	SortHighestScore int    `json:"sortHighestScore"`
	SortLowestScore  int    `json:"sortLowestScore"`
	Title            string `json:"title"`
	TitleCustom      bool   `json:"titleCustom,omitempty"`
	// TotalOnlineMs：累计在线时长（毫秒）；与 PublicStats 同语义。
	TotalOnlineMs int64 `json:"totalOnlineMs,omitempty"`
}

// ToLobbyPlayer 从完整公开资料裁剪。
func ToLobbyPlayer(p PublicPlayer) LobbyPlayer {
	return LobbyPlayer{
		ID:                           p.ID,
		Name:                         p.Name,
		GenderID:                     p.GenderID,
		GenderLabel:                  p.GenderLabel,
		FactionID:                    p.FactionID,
		FactionLabel:                 p.FactionLabel,
		FactionColors:                p.FactionColors,
		DisplayName:                  p.DisplayName,
		AvatarURL:                    p.AvatarURL,
		Connected:                    p.Connected,
		DisconnectedAt:               p.DisconnectedAt,
		DisconnectExpiresAt:          p.DisconnectExpiresAt,
		NameWarEnabled:               p.NameWarEnabled,
		NameWarPenaltyName:           p.NameWarPenaltyName,
		NameWarPunished:              p.NameWarPunished,
		NameWarAllowRename:           p.NameWarAllowRename,
		NameWarRenameProtectedUntil:  p.NameWarRenameProtectedUntil,
		NameWarRenamedByName:         p.NameWarRenamedByName,
		GiveawayEnabled:              p.GiveawayEnabled,
		GiveawayValue:                p.GiveawayValue,
		GiveawayBoardText:            p.GiveawayBoardText,
		GiveawayBoardExpiresAt:       p.GiveawayBoardExpiresAt,
		GiveawayBoardLikes:           p.GiveawayBoardLikes,
		GiveawayBoardDislikes:        p.GiveawayBoardDislikes,
		GiveawayVoteWindowStartedAt:  p.GiveawayVoteWindowStartedAt,
		GiveawayVoteLikesThisHour:    p.GiveawayVoteLikesThisHour,
		GiveawayVoteDislikesThisHour: p.GiveawayVoteDislikesThisHour,
		RankMultiplierUnlocked:       p.RankMultiplierUnlocked,
		ExtremeModeEnabled:           p.ExtremeModeEnabled,
		ExtremeWinStreak:             p.ExtremeWinStreak,
		ExtremeForceClosed:           p.ExtremeForceClosed,
		ExtremeForceClosedAt:         p.ExtremeForceClosedAt,
		ExtremeRenameProtectedUntil:  p.ExtremeRenameProtectedUntil,
		ExtremeRenamedByName:         p.ExtremeRenamedByName,
		RoomID:                       p.RoomID,
		Stats: LobbyStats{
			Wins: p.Stats.Wins, Losses: p.Stats.Losses, Draws: p.Stats.Draws,
			Punishments: p.Stats.Punishments, RankedPoints: p.Stats.RankedPoints, Title: p.Stats.Title,
			HighestScore: p.Stats.HighestScore, LowestScore: p.Stats.LowestScore,
			SortRankedPoints: p.Stats.SortRankedPoints, SortHighestScore: p.Stats.SortHighestScore,
			SortLowestScore: p.Stats.SortLowestScore, TitleCustom: p.Stats.TitleCustom,
			TotalOnlineMs: p.Stats.TotalOnlineMs,
		},
		GameStats: p.GameStats,
	}
}

// AsPublicPlayer 将大厅视图提升为完整结构（缺省字段为零值），便于前端复用组件。
func (p LobbyPlayer) AsPublicPlayer() PublicPlayer {
	return PublicPlayer{
		ID: p.ID, Name: p.Name, GenderID: p.GenderID, GenderLabel: p.GenderLabel,
		FactionID: p.FactionID, FactionLabel: p.FactionLabel, FactionColors: p.FactionColors,
		DisplayName: p.DisplayName, AvatarURL: p.AvatarURL, Connected: p.Connected,
		DisconnectedAt: p.DisconnectedAt, DisconnectExpiresAt: p.DisconnectExpiresAt,
		NameWarEnabled: p.NameWarEnabled, NameWarPenaltyName: p.NameWarPenaltyName,
		NameWarPunished: p.NameWarPunished, NameWarAllowRename: p.NameWarAllowRename,
		NameWarRenameProtectedUntil: p.NameWarRenameProtectedUntil, NameWarRenamedByName: p.NameWarRenamedByName,
		GiveawayEnabled: p.GiveawayEnabled, GiveawayValue: p.GiveawayValue,
		GiveawayBoardText: p.GiveawayBoardText, GiveawayBoardExpiresAt: p.GiveawayBoardExpiresAt,
		GiveawayBoardLikes: p.GiveawayBoardLikes, GiveawayBoardDislikes: p.GiveawayBoardDislikes,
		GiveawayVoteWindowStartedAt: p.GiveawayVoteWindowStartedAt,
		GiveawayVoteLikesThisHour:   p.GiveawayVoteLikesThisHour, GiveawayVoteDislikesThisHour: p.GiveawayVoteDislikesThisHour,
		RankMultiplierUnlocked: p.RankMultiplierUnlocked, ExtremeModeEnabled: p.ExtremeModeEnabled,
		ExtremeWinStreak: p.ExtremeWinStreak, ExtremeForceClosed: p.ExtremeForceClosed,
		ExtremeForceClosedAt: p.ExtremeForceClosedAt, ExtremeRenameProtectedUntil: p.ExtremeRenameProtectedUntil,
		ExtremeRenamedByName: p.ExtremeRenamedByName, RoomID: p.RoomID,
		Stats: PublicStats{
			Wins: p.Stats.Wins, Losses: p.Stats.Losses, Draws: p.Stats.Draws,
			Punishments: p.Stats.Punishments, RankedPoints: p.Stats.RankedPoints, Title: p.Stats.Title,
			HighestScore: p.Stats.HighestScore, LowestScore: p.Stats.LowestScore,
			SortRankedPoints: p.Stats.SortRankedPoints, SortHighestScore: p.Stats.SortHighestScore,
			SortLowestScore: p.Stats.SortLowestScore, TitleCustom: p.Stats.TitleCustom,
			TotalOnlineMs: p.Stats.TotalOnlineMs,
		},
		GameStats: p.GameStats,
	}
}
