package server

import "github.com/doumiao/newRPS/internal/types"

// JSON 空值约定（前后端协议，B5）：
//   - 出站：nil slice/map 一律洗成 [] / {}，禁止 JSON null（否则前端 .map/.includes 白屏）。
//   - 前端 normalize* 只做 DELTA 合并后的薄兜底，不假设服务端会再发 null。
// 本文件是出站清洗的唯一入口；新增字段若含 slice/map，须在 sanitize* 里接上。

func emptyStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func emptyPlayers(s []types.PublicPlayer) []types.PublicPlayer {
	if s == nil {
		return []types.PublicPlayer{}
	}
	return s
}

func emptyLobbyPlayers(s []types.LobbyPlayer) []types.LobbyPlayer {
	if s == nil {
		return []types.LobbyPlayer{}
	}
	return s
}

func emptyLobbyPlayerMap(m map[string]types.LobbyPlayer) map[string]types.LobbyPlayer {
	if m == nil {
		return map[string]types.LobbyPlayer{}
	}
	return m
}

func emptyLobbyRoomMap(m map[string]types.LobbyRoomInfo) map[string]types.LobbyRoomInfo {
	if m == nil {
		return map[string]types.LobbyRoomInfo{}
	}
	return m
}

func emptyChat(s []types.ChatMessage) []types.ChatMessage {
	if s == nil {
		return []types.ChatMessage{}
	}
	return s
}

func emptySuggestions(s []types.Suggestion) []types.Suggestion {
	if s == nil {
		return []types.Suggestion{}
	}
	return s
}

func emptyProofs(s []types.PunishmentProof) []types.PunishmentProof {
	if s == nil {
		return []types.PunishmentProof{}
	}
	return s
}

func emptyHistoryProofs(s []types.HistoryProof) []types.HistoryProof {
	if s == nil {
		return []types.HistoryProof{}
	}
	return s
}

func emptyHistory(s []types.RoundHistoryItem) []types.RoundHistoryItem {
	if s == nil {
		return []types.RoundHistoryItem{}
	}
	return s
}

func emptyPunishmentTasks(s []types.PunishmentTask) []types.PunishmentTask {
	if s == nil {
		return []types.PunishmentTask{}
	}
	return s
}

func emptyPos(s []types.Pos) []types.Pos {
	if s == nil {
		return []types.Pos{}
	}
	return s
}

func seatBoolMap(m map[types.SeatKey]bool) map[types.SeatKey]bool {
	if m == nil {
		return map[types.SeatKey]bool{types.SeatA: false, types.SeatB: false}
	}
	out := map[types.SeatKey]bool{
		types.SeatA: m[types.SeatA],
		types.SeatB: m[types.SeatB],
	}
	return out
}

func seatIntMap(m map[types.SeatKey]int) map[types.SeatKey]int {
	if m == nil {
		return map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	return map[types.SeatKey]int{
		types.SeatA: m[types.SeatA],
		types.SeatB: m[types.SeatB],
	}
}

func seatStatsMap(m map[types.SeatKey]types.SeatStats) map[types.SeatKey]types.SeatStats {
	if m == nil {
		return map[types.SeatKey]types.SeatStats{types.SeatA: {}, types.SeatB: {}}
	}
	return map[types.SeatKey]types.SeatStats{
		types.SeatA: m[types.SeatA],
		types.SeatB: m[types.SeatB],
	}
}

func seatAnyMap(m map[types.SeatKey]any) map[types.SeatKey]any {
	if m == nil {
		return map[types.SeatKey]any{}
	}
	return m
}

func sanitizeRoomSettings(settings types.RoomSettings) types.RoomSettings {
	settings.PunishmentIDs = emptyStrings(settings.PunishmentIDs)
	settings.Tags = emptyStrings(settings.Tags)
	return settings
}

func sanitizeRoundHistoryItem(item types.RoundHistoryItem) types.RoundHistoryItem {
	item.PunishmentTasks = emptyPunishmentTasks(item.PunishmentTasks)
	item.PunishedNames = emptyStrings(item.PunishedNames)
	item.Proofs = emptyHistoryProofs(item.Proofs)
	item.TicTacToeLine = emptyPos(item.TicTacToeLine)
	item.GomokuLine = emptyPos(item.GomokuLine)
	if item.LiarsDiceHands == nil {
		item.LiarsDiceHands = map[string][]int{}
	}
	if item.LiarsDiceHandOrder == nil {
		item.LiarsDiceHandOrder = []string{}
	}
	if item.LiarsDiceNames == nil {
		item.LiarsDiceNames = map[string]string{}
	}
	return item
}

func sanitizeOthello(state *types.OthelloState) *types.OthelloState {
	if state == nil {
		return nil
	}
	out := *state
	out.LegalMoves = emptyPos(out.LegalMoves)
	out.SettlementEvents = emptyStrings(out.SettlementEvents)
	if out.RankedDelta == nil {
		out.RankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	// board 内 null 格子是合法的；保证外层 8 行存在
	if out.Board == nil {
		out.Board = make([][]*types.OthelloCell, 8)
		for i := range out.Board {
			out.Board[i] = make([]*types.OthelloCell, 8)
		}
	}
	return &out
}

func sanitizeTicTacToe(state *types.TicTacToeState) *types.TicTacToeState {
	if state == nil {
		return nil
	}
	out := *state
	out.WinningLine = emptyPos(out.WinningLine)
	if out.RankedDelta == nil {
		out.RankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	if out.Board == nil {
		out.Board = make([][]*types.TicTacToeCell, 3)
		for i := range out.Board {
			out.Board[i] = make([]*types.TicTacToeCell, 3)
		}
	}
	return &out
}

func sanitizeGomoku(state *types.GomokuState) *types.GomokuState {
	if state == nil {
		return nil
	}
	out := *state
	out.Moves = emptyPos(out.Moves)
	out.WinningLine = emptyPos(out.WinningLine)
	if out.RankedDelta == nil {
		out.RankedDelta = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	if out.UndoCount == nil {
		out.UndoCount = map[types.SeatKey]int{types.SeatA: 0, types.SeatB: 0}
	}
	if out.Board == nil {
		out.Board = make([][]*types.GomokuCell, gomokuBoardSize)
		for i := range out.Board {
			out.Board[i] = make([]*types.GomokuCell, gomokuBoardSize)
		}
	}
	return &out
}

func sanitizeLiarsDice(state *types.LiarsDiceState) *types.LiarsDiceState {
	if state == nil {
		return nil
	}
	out := *state
	if out.ParticipantIDs == nil {
		out.ParticipantIDs = []string{}
	}
	if out.ReadyPlayerIDs == nil {
		out.ReadyPlayerIDs = []string{}
	}
	if out.DiceCounts == nil {
		out.DiceCounts = map[string]int{}
	}
	if out.BidHistory == nil {
		out.BidHistory = []types.LiarsDiceBid{}
	}
	return &out
}

func sanitizeRoomSnapshot(snap types.RoomSnapshot) types.RoomSnapshot {
	snap.Settings = sanitizeRoomSettings(snap.Settings)
	snap.Spectators = emptyPlayers(snap.Spectators)
	snap.Ready = seatBoolMap(snap.Ready)
	snap.Choices = seatAnyMap(snap.Choices)
	if snap.Seats == nil {
		snap.Seats = map[types.SeatKey]any{types.SeatA: nil, types.SeatB: nil}
	}
	// 保证 A/B 键存在
	if _, ok := snap.Seats[types.SeatA]; !ok {
		snap.Seats[types.SeatA] = nil
	}
	if _, ok := snap.Seats[types.SeatB]; !ok {
		snap.Seats[types.SeatB] = nil
	}
	snap.PunishedPlayerIDs = emptyStrings(snap.PunishedPlayerIDs)
	snap.Proofs = emptyProofs(snap.Proofs)
	snap.Score = seatIntMap(snap.Score)
	snap.SeatedScore = seatIntMap(snap.SeatedScore)
	snap.SeatStats = seatStatsMap(snap.SeatStats)
	history := emptyHistory(snap.RoundHistory)
	for i := range history {
		history[i] = sanitizeRoundHistoryItem(history[i])
	}
	snap.RoundHistory = history
	snap.Chat = emptyChat(snap.Chat)
	snap.Othello = sanitizeOthello(snap.Othello)
	snap.TicTacToe = sanitizeTicTacToe(snap.TicTacToe)
	snap.LiarsDice = sanitizeLiarsDice(snap.LiarsDice)
	snap.Gomoku = sanitizeGomoku(snap.Gomoku)
	return snap
}

func sanitizeLobbyRoom(info types.LobbyRoomInfo) types.LobbyRoomInfo {
	info.PunishmentIDs = emptyStrings(info.PunishmentIDs)
	info.Tags = emptyStrings(info.Tags)
	if info.Versus == nil {
		info.Versus = map[types.SeatKey]any{types.SeatA: nil, types.SeatB: nil}
	}
	if _, ok := info.Versus[types.SeatA]; !ok {
		info.Versus[types.SeatA] = nil
	}
	if _, ok := info.Versus[types.SeatB]; !ok {
		info.Versus[types.SeatB] = nil
	}
	return info
}

func sanitizeLobbySnapshot(snap types.LobbySnapshot) types.LobbySnapshot {
	snap.Players = emptyLobbyPlayerMap(snap.Players)
	rooms := emptyLobbyRoomMap(snap.Rooms)
	for id, room := range rooms {
		rooms[id] = sanitizeLobbyRoom(room)
	}
	snap.Rooms = rooms
	snap.NormalLeaderboard = emptyLobbyPlayers(snap.NormalLeaderboard)
	snap.RankedLeaderboard = emptyLobbyPlayers(snap.RankedLeaderboard)
	snap.Suggestions = emptySuggestions(snap.Suggestions)
	snap.LobbyChat = emptyChat(snap.LobbyChat)
	return snap
}

// sanitizePublicConfig 保证配置里数组字段不为 null（后台/大厅 config:update）。
func sanitizePublicConfig(cfg types.AppConfig) types.AppConfig {
	cfg.Genders = emptyIfNilGender(cfg.Genders)
	if cfg.GenderFactions == nil {
		cfg.GenderFactions = []types.GenderFaction{}
	}
	for i := range cfg.GenderFactions {
		if cfg.GenderFactions[i].Genders == nil {
			cfg.GenderFactions[i].Genders = []types.GenderOption{}
		}
	}
	if cfg.Titles == nil {
		cfg.Titles = []types.TitleSegment{}
	}
	for i := range cfg.Titles {
		cfg.Titles[i].Names = emptyStrings(cfg.Titles[i].Names)
		if cfg.Titles[i].FactionNames == nil {
			cfg.Titles[i].FactionNames = map[string][]string{}
		}
		for k, v := range cfg.Titles[i].FactionNames {
			cfg.Titles[i].FactionNames[k] = emptyStrings(v)
		}
	}
	if cfg.Punishments == nil {
		cfg.Punishments = []types.PunishmentConfig{}
	}
	for i := range cfg.Punishments {
		p := &cfg.Punishments[i]
		if p.Variants == nil {
			p.Variants = map[string]string{}
		}
		p.RoomBackgroundImages = emptyStrings(p.RoomBackgroundImages)
		if p.Tasks == nil {
			p.Tasks = []types.PunishmentTaskConfig{}
		}
		for j := range p.Tasks {
			t := &p.Tasks[j]
			if t.Variants == nil {
				t.Variants = map[string]string{}
			}
			t.BackgroundImages = emptyStrings(t.BackgroundImages)
		}
		if p.RoomNamePool != nil {
			p.RoomNamePool.Adjectives = emptyStrings(p.RoomNamePool.Adjectives)
			p.RoomNamePool.Subjects = emptyStrings(p.RoomNamePool.Subjects)
			p.RoomNamePool.RoomWords = emptyStrings(p.RoomNamePool.RoomWords)
		}
	}
	if cfg.PlayerPunishmentRoomNamePool != nil {
		cfg.PlayerPunishmentRoomNamePool.Adjectives = emptyStrings(cfg.PlayerPunishmentRoomNamePool.Adjectives)
		cfg.PlayerPunishmentRoomNamePool.Subjects = emptyStrings(cfg.PlayerPunishmentRoomNamePool.Subjects)
		cfg.PlayerPunishmentRoomNamePool.RoomWords = emptyStrings(cfg.PlayerPunishmentRoomNamePool.RoomWords)
	}
	cfg.RoomTags = emptyStrings(cfg.RoomTags)
	if cfg.RoomInfoTags == nil {
		cfg.RoomInfoTags = map[string]types.RoomInfoTagStyle{}
	}
	cfg.Bots.Names = emptyStrings(cfg.Bots.Names)
	if cfg.Bots.Difficulties == nil {
		cfg.Bots.Difficulties = []types.BotDifficultyConfig{}
	}
	if cfg.Games == nil {
		cfg.Games = []types.GameConfig{}
	}
	if cfg.Messages == nil {
		cfg.Messages = map[string]string{}
	}
	if cfg.ExtremeMode.PositiveLossRates == nil {
		cfg.ExtremeMode.PositiveLossRates = map[string]float64{}
	}
	if cfg.ExtremeMode.NegativeWinRates == nil {
		cfg.ExtremeMode.NegativeWinRates = map[string]float64{}
	}
	if cfg.ExtremeMode.HourlyDecay == nil {
		cfg.ExtremeMode.HourlyDecay = map[string]float64{}
	}
	return cfg
}

func emptyIfNilGender(s []types.GenderOption) []types.GenderOption {
	if s == nil {
		return []types.GenderOption{}
	}
	return s
}
