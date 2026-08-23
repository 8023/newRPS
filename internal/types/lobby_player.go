package types

// LobbyPlayer 大厅/全局广播用的精简玩家视图。
// 完整 PublicPlayer 通过 player:get / 进房快照 / 个人 me 下发。
// 旧版大厅客户端依赖的聚合投票字段仍保留并随自己的玩家记录广播；新客户端的精确额度
// 走 giveaway:voteQuotas RPC 单播，不把按目标 map 放进大厅广播。
// 前端自身的 me.player 是通过与 lobby.players 中自己那条记录的 diff 合并来保持更新的
// （见 web/src/App.tsx）——凡是要放进这个结构体的字段都得遵守这条：漏了就会被广播抹成
// undefined。
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

	BondMasterEnabled *bool `json:"bondMasterEnabled,omitempty"`
	BondPetEnabled    *bool `json:"bondPetEnabled,omitempty"`
	BondPublicDisplay *bool `json:"bondPublicDisplay,omitempty"`

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
	// TitleSource：与 PublicStats.TitleSource 同语义，供称号标签按来源着色。
	TitleSource string `json:"titleSource,omitempty"`
	// TitleColors：与 PublicStats.TitleColors 同语义。
	TitleColors GenderColors `json:"titleColors"`
	// TotalOnlineMs：累计在线时长（毫秒）；与 PublicStats 同语义。
	TotalOnlineMs int64 `json:"totalOnlineMs,omitempty"`
	// SelfTitle：与 PublicStats.SelfTitle 同语义。这里必须带上，否则 player:batch 广播
	// （大厅刷新、他人上下线等很频繁）合并进 App.tsx 的 me.player 时会把已设置的自设称号
	// 覆盖抹掉——AppViews.tsx 的 ProfilePanel 重新打开时读到的就是被抹空后的值。
	SelfTitle string `json:"selfTitle,omitempty"`
	// ContributionApprovedCount：该玩家提交并审批通过的共建投稿任务条数（随机任务 +
	// 系列任务的每个子任务各算一条），仅供「共建」排行榜用。不是内存态战绩字段——只由
	// onPlayersRoster 按需查库后手动填充，不参与 lobbyStatsFromPublic/AsPublicPlayer 的
	// 常规转换，也不随大厅实时广播下发。
	ContributionApprovedCount int `json:"contributionApprovedCount,omitempty"`
}

// ToLobbyPlayer 从完整公开资料裁剪（含分游戏战绩，供 player:batch / roster 用）。
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
		BondMasterEnabled:            p.BondMasterEnabled,
		BondPetEnabled:               p.BondPetEnabled,
		BondPublicDisplay:            p.BondPublicDisplay,
		RoomID:                       p.RoomID,
		Stats:                        lobbyStatsFromPublic(p.Stats),
		GameStats:                    p.GameStats,
	}
}

// ToLiveLobbyPlayer 大厅实时通道用的更轻视图：去掉分游戏 GameStats。
// 大厅侧榜只用 Stats 合计；分项战绩由 players:roster 按需拉取。
// 前端 normalizePublicPlayer 在 gameStats 全零时保留 Stats.wins/losses/draws。
func ToLiveLobbyPlayer(p PublicPlayer) LobbyPlayer {
	lp := ToLobbyPlayer(p)
	lp.GameStats = GameStats{}
	return lp
}

func lobbyStatsFromPublic(s PublicStats) LobbyStats {
	return LobbyStats{
		Wins: s.Wins, Losses: s.Losses, Draws: s.Draws,
		Punishments: s.Punishments, RankedPoints: s.RankedPoints, Title: s.Title,
		HighestScore: s.HighestScore, LowestScore: s.LowestScore,
		SortRankedPoints: s.SortRankedPoints, SortHighestScore: s.SortHighestScore,
		SortLowestScore: s.SortLowestScore, TitleCustom: s.TitleCustom,
		TotalOnlineMs: s.TotalOnlineMs, SelfTitle: s.SelfTitle,
		TitleSource: s.TitleSource, TitleColors: s.TitleColors,
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
		GiveawayVoteWindowStartedAt:  p.GiveawayVoteWindowStartedAt,
		GiveawayVoteLikesThisHour:    p.GiveawayVoteLikesThisHour,
		GiveawayVoteDislikesThisHour: p.GiveawayVoteDislikesThisHour,
		GiveawayBoardLikes:           p.GiveawayBoardLikes, GiveawayBoardDislikes: p.GiveawayBoardDislikes,
		RankMultiplierUnlocked: p.RankMultiplierUnlocked, ExtremeModeEnabled: p.ExtremeModeEnabled,
		ExtremeWinStreak: p.ExtremeWinStreak, ExtremeForceClosed: p.ExtremeForceClosed,
		ExtremeForceClosedAt: p.ExtremeForceClosedAt, ExtremeRenameProtectedUntil: p.ExtremeRenameProtectedUntil,
		ExtremeRenamedByName: p.ExtremeRenamedByName,
		BondMasterEnabled:    p.BondMasterEnabled, BondPetEnabled: p.BondPetEnabled, BondPublicDisplay: p.BondPublicDisplay,
		RoomID: p.RoomID,
		Stats: PublicStats{
			Wins: p.Stats.Wins, Losses: p.Stats.Losses, Draws: p.Stats.Draws,
			Punishments: p.Stats.Punishments, RankedPoints: p.Stats.RankedPoints, Title: p.Stats.Title,
			HighestScore: p.Stats.HighestScore, LowestScore: p.Stats.LowestScore,
			SortRankedPoints: p.Stats.SortRankedPoints, SortHighestScore: p.Stats.SortHighestScore,
			SortLowestScore: p.Stats.SortLowestScore, TitleCustom: p.Stats.TitleCustom,
			TotalOnlineMs: p.Stats.TotalOnlineMs, SelfTitle: p.Stats.SelfTitle,
			TitleSource: p.Stats.TitleSource, TitleColors: p.Stats.TitleColors,
		},
		GameStats: p.GameStats,
	}
}
