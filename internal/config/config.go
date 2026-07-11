package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

var rootDir string

func init() {
	rootDir = findRootDir()
}

func findRootDir() string {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, wd)
	}
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Dir(exe)
		candidates = append(candidates, dir, filepath.Join(dir, ".."), filepath.Join(dir, "../.."))
	}
	for _, c := range candidates {
		abs, err := filepath.Abs(c)
		if err != nil {
			continue
		}
		if _, err := os.Stat(filepath.Join(abs, "config", "default.json")); err == nil {
			return abs
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func GetRootDir() string {
	return rootDir
}

func configDir() string {
	return filepath.Join(rootDir, "config")
}

func defaultPath() string {
	return filepath.Join(configDir(), "default.json")
}

func activePath() string {
	return filepath.Join(configDir(), "active.json")
}

var defaultFactions = []types.GenderFaction{
	{
		GenderColors: types.GenderColors{TextColor: "#225c8d", BackgroundColor: "#dff2ff", BorderColor: "#92cdf2"},
		ID:           "male_faction",
		Label:        "男性阵营",
		Genders: []types.GenderOption{
			{ID: "boy", Label: "男生", FactionID: "male_faction"},
			{ID: "male", Label: "男性", FactionID: "male_faction"},
		},
	},
	{
		GenderColors: types.GenderColors{TextColor: "#8a3158", BackgroundColor: "#ffe2ef", BorderColor: "#f3a9ca"},
		ID:           "female_faction",
		Label:        "女性阵营",
		Genders: []types.GenderOption{
			{ID: "girl", Label: "女生", FactionID: "female_faction"},
			{ID: "female", Label: "女性", FactionID: "female_faction"},
		},
	},
	{
		GenderColors: types.GenderColors{TextColor: "#6650a4", BackgroundColor: "#eee9ff", BorderColor: "#c7b5ff"},
		ID:           "femboy_faction",
		Label:        "男娘阵营",
		Genders: []types.GenderOption{
			{ID: "femboy", Label: "男娘", FactionID: "femboy_faction"},
			{ID: "transgirl", Label: "药娘", FactionID: "femboy_faction"},
		},
	},
	{
		GenderColors: types.GenderColors{TextColor: "#4d5c6f", BackgroundColor: "#eef3f8", BorderColor: "#c9d6e4"},
		ID:           "other_faction",
		Label:        "其他阵营",
		Genders: []types.GenderOption{
			{ID: "attack_helicopter", Label: "武装直升机", FactionID: "other_faction"},
			{ID: "walmart_bag", Label: "沃尔玛购物袋", FactionID: "other_faction"},
		},
	},
}

var defaultRoomNamePool = types.RoomNamePool{
	Adjectives: []string{"粉蓝", "闪亮", "轻松", "神秘"},
	Subjects:   []string{"拳手", "挑战", "真心话", "冒险"},
	RoomWords:  []string{"小屋", "房间", "擂台", "茶会"},
}

var defaultPlayerPunishmentRoomNamePool = types.RoomNamePool{
	Adjectives: []string{"临时", "即兴", "神秘", "互写"},
	Subjects:   []string{"任务", "挑战", "惩罚", "考验"},
	RoomWords:  []string{"小屋", "房间", "剧场", "擂台"},
}

var defaultRoomTags = []string{"轻松", "认真", "排位", "惩罚", "聊天"}

var defaultGames = []types.GameConfig{
	{ID: "rps", Name: "锤子剪刀布", Description: "双方同时选择石头、剪刀、布，服务器公开结算。"},
	{ID: "othello", Name: "黑白棋", Description: "8x8 棋盘轮流落子，服务器判断翻棋和胜负。"},
	{ID: "tictactoe", Name: "井字棋", Description: "3x3 棋盘轮流落子，先连成一线者胜。"},
}

var defaultNameWar = struct {
	PenaltyPrefix           string
	LoserPanelTitle         string
	EscapeTitle             string
	RenamePanelTitle        string
	NameWarLoserLabel       string
	ExtremeForceClosedLabel string
}{
	PenaltyPrefix:           "失名者",
	LoserPanelTitle:         "名字争夺战失格者",
	EscapeTitle:             "逃跑的人",
	RenamePanelTitle:        "通用改名处",
	NameWarLoserLabel:       "名争失格",
	ExtremeForceClosedLabel: "极限强关",
}

var defaultGiveaway = struct {
	PanelTitle        string
	PanelDescription  string
	SubmitPlaceholder string
	EmptyText         string
}{
	PanelTitle:        "白给自救板",
	PanelDescription:  "提交一点自我惩罚宣言，等待其他玩家点赞帮你降低白给值。",
	SubmitPlaceholder: "写下你的自我惩罚宣言...",
	EmptyText:         "还没有人在白给自救板上。",
}

var defaultExtremeMode = types.ExtremeModeConfig{
	Label:                "极限模式",
	Emoji:                "⚡",
	CooldownHours:         12,
	PositiveLossRates:    map[string]float64{"pos1": 0.9, "pos2": 0.75, "pos3": 0.6, "pos4": 0.5},
	NegativeWinRates:     map[string]float64{"neg1": 0.9, "neg2": 0.75, "neg3": 0.6, "neg4": 0.5},
	HourlyDecay:          map[string]float64{"pos4": 10, "pos3": 6, "pos2": 4, "pos1": 2, "default": 2},
	WinStreakThreshold:   10,
	WinStreakCrashChance: 0.5,
	CrashTargetPoints:    333,
	ForceCloseWarning:    "强行关闭极限模式后，你会永久进入通用改名处，可被符合条件的极限玩家改名。",
	ForceRenameMinPoints: 1,
	ForceRenameProtectHours: 4,
}

var defaultDailyAnnouncement = types.DailyAnnouncement{
	Enabled:    true,
	Title:      "今日公告",
	Content:    "欢迎来到抖喵游戏屋。游玩时请尊重其他玩家，遇到卡房或异常可以联系管理员处理。",
	ButtonText: "知道了",
	Version:    "default",
}

var defaultAccessControl = struct {
	MaxOnlinePerIP     int
	MaxCreatesPer10Min int
}{MaxOnlinePerIP: 3, MaxCreatesPer10Min: 5}

var defaultRoomInfoTags = map[string]types.RoomInfoTagStyle{
	"gameRps":              {GenderColors: types.GenderColors{TextColor: "#4d5c6f", BackgroundColor: "#eef3f8", BorderColor: "#c9d6e4"}, Label: "锤子剪刀布"},
	"gameOthello":          {GenderColors: types.GenderColors{TextColor: "#163c32", BackgroundColor: "#dff7ec", BorderColor: "#93d8b8"}, Label: "黑白棋"},
	"gameTicTacToe":        {GenderColors: types.GenderColors{TextColor: "#5c3b82", BackgroundColor: "#f0e7ff", BorderColor: "#c7a8f5"}, Label: "井字棋"},
	"phaseReady":           {GenderColors: types.GenderColors{TextColor: "#225c8d", BackgroundColor: "#e5f5ff", BorderColor: "#9ed7ff"}, Label: "等待坐满"},
	"phaseChoosing":        {GenderColors: types.GenderColors{TextColor: "#6b4b00", BackgroundColor: "#fff3c4", BorderColor: "#ffd875"}, Label: "出拳中"},
	"phaseResult":          {GenderColors: types.GenderColors{TextColor: "#6b3f8d", BackgroundColor: "#f1e7ff", BorderColor: "#c9a9ff"}, Label: "结算中"},
	"phasePunishment":      {GenderColors: types.GenderColors{TextColor: "#8a3158", BackgroundColor: "#ffe2ef", BorderColor: "#f3a9ca"}, Label: "惩罚阶段"},
	"normal":               {GenderColors: types.GenderColors{TextColor: "#3c6074", BackgroundColor: "#edf8fb", BorderColor: "#b7dfe9"}, Label: "普通局"},
	"ranked":               {GenderColors: types.GenderColors{TextColor: "#765100", BackgroundColor: "#fff0bd", BorderColor: "#ffd66e"}, Label: "排位"},
	"punishment":           {GenderColors: types.GenderColors{TextColor: "#8a3158", BackgroundColor: "#ffe5f1", BorderColor: "#f3a9ca"}, Label: "惩罚开启"},
	"noPunishment":         {GenderColors: types.GenderColors{TextColor: "#4d5c6f", BackgroundColor: "#eef3f8", BorderColor: "#c9d6e4"}, Label: "无惩罚"},
	"tieDoublePunish":      {GenderColors: types.GenderColors{TextColor: "#7b3a22", BackgroundColor: "#ffe8dc", BorderColor: "#ffb894"}, Label: "平局双罚"},
	"requireOpponentConfirm": {GenderColors: types.GenderColors{TextColor: "#225c8d", BackgroundColor: "#e1f2ff", BorderColor: "#8fcaf0"}, Label: "需要对手确认"},
	"allowProofImage":      {GenderColors: types.GenderColors{TextColor: "#326749", BackgroundColor: "#e3f8ec", BorderColor: "#9ed9b8"}, Label: "允许图片证明"},
	"textProofOnly":        {GenderColors: types.GenderColors{TextColor: "#5c5570", BackgroundColor: "#f0edf8", BorderColor: "#c8bedf"}, Label: "仅文字证明"},
	"extremeRanked":        {GenderColors: types.GenderColors{TextColor: "#7c3d00", BackgroundColor: "#fff1d8", BorderColor: "#ffbf75"}, Label: "极限排位"},
}

func readJSON(filePath string) (types.AppConfig, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return types.AppConfig{}, err
	}
	var cfg types.AppConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return types.AppConfig{}, err
	}
	return cfg, nil
}

func cloneDefaultFactions() []types.GenderFaction {
	out := make([]types.GenderFaction, len(defaultFactions))
	for i, f := range defaultFactions {
		genders := make([]types.GenderOption, len(f.Genders))
		copy(genders, f.Genders)
		out[i] = f
		out[i].Genders = genders
	}
	return out
}

func flattenGenders(factions []types.GenderFaction) []types.GenderOption {
	var out []types.GenderOption
	for _, faction := range factions {
		for _, g := range faction.Genders {
			g.FactionID = faction.ID
			out = append(out, g)
		}
	}
	return out
}

func cleanLines(values []string, fallback []string) []string {
	var items []string
	for _, v := range values {
		t := strings.TrimSpace(v)
		if t != "" {
			items = append(items, t)
		}
	}
	if len(items) == 0 {
		out := make([]string, len(fallback))
		copy(out, fallback)
		return out
	}
	return items
}

func cleanLinesAny(values any, fallback []string) []string {
	switch v := values.(type) {
	case []string:
		return cleanLines(v, fallback)
	case []any:
		var items []string
		for _, x := range v {
			t := strings.TrimSpace(fmt.Sprint(x))
			if t != "" {
				items = append(items, t)
			}
		}
		if len(items) == 0 {
			out := make([]string, len(fallback))
			copy(out, fallback)
			return out
		}
		return items
	default:
		out := make([]string, len(fallback))
		copy(out, fallback)
		return out
	}
}

func clampNumber(value float64, min, max, fallback float64) int {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return int(fallback)
	}
	n := math.Round(value)
	if n < min {
		return int(min)
	}
	if n > max {
		return int(max)
	}
	return int(n)
}

func clampOpacity(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.26
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampRatio(value, fallback float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fallback
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampTaskBackgroundOpacity(value float64) float64 {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0.22
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func normalizeNumberRecord(input map[string]float64, fallback map[string]float64, min, max float64) map[string]float64 {
	out := make(map[string]float64, len(fallback))
	for key, fb := range fallback {
		v, ok := input[key]
		if !ok || math.IsNaN(v) || math.IsInf(v, 0) {
			out[key] = fb
			continue
		}
		if v < min {
			v = min
		}
		if v > max {
			v = max
		}
		out[key] = v
	}
	return out
}

func isBotStrategy(value types.BotStrategy) bool {
	switch value {
	case "random", "counter", "chaos", "throw", "win":
		return true
	default:
		return false
	}
}

func normalizeRoomNamePool(pool *types.RoomNamePool, fallback types.RoomNamePool) types.RoomNamePool {
	if pool == nil {
		return types.RoomNamePool{
			Adjectives: cleanLines(nil, fallback.Adjectives),
			Subjects:   cleanLines(nil, fallback.Subjects),
			RoomWords:  cleanLines(nil, fallback.RoomWords),
		}
	}
	return types.RoomNamePool{
		Adjectives: cleanLines(pool.Adjectives, fallback.Adjectives),
		Subjects:   cleanLines(pool.Subjects, fallback.Subjects),
		RoomWords:  cleanLines(pool.RoomWords, fallback.RoomWords),
	}
}

func normalizeGames(input []types.GameConfig) []types.GameConfig {
	games := make(map[types.GameID]types.GameConfig, 3)
	for _, g := range defaultGames {
		games[g.ID] = g
	}
	for _, g := range input {
		if g.ID != "rps" && g.ID != "othello" && g.ID != "tictactoe" {
			continue
		}
		def := games[g.ID]
		name := strings.TrimSpace(g.Name)
		if name == "" {
			name = def.Name
		}
		if len([]rune(name)) > 18 {
			name = string([]rune(name)[:18])
		}
		desc := strings.TrimSpace(g.Description)
		if desc == "" {
			desc = def.Description
		}
		if len([]rune(desc)) > 120 {
			desc = string([]rune(desc)[:120])
		}
		games[g.ID] = types.GameConfig{ID: g.ID, Name: name, Description: desc}
	}
	return []types.GameConfig{games["rps"], games["othello"], games["tictactoe"]}
}

func normalizeRoomInfoTags(input map[string]types.RoomInfoTagStyle) map[string]types.RoomInfoTagStyle {
	out := make(map[string]types.RoomInfoTagStyle, len(defaultRoomInfoTags))
	for key, fallback := range defaultRoomInfoTags {
		current, ok := input[key]
		label := strings.TrimSpace(current.Label)
		if !ok || label == "" {
			label = fallback.Label
		}
		if len([]rune(label)) > 16 {
			label = string([]rune(label)[:16])
		}
		tc := current.TextColor
		if tc == "" {
			tc = fallback.TextColor
		}
		bg := current.BackgroundColor
		if bg == "" {
			bg = fallback.BackgroundColor
		}
		bc := current.BorderColor
		if bc == "" {
			bc = fallback.BorderColor
		}
		out[key] = types.RoomInfoTagStyle{
			GenderColors: types.GenderColors{TextColor: tc, BackgroundColor: bg, BorderColor: bc},
			Label:        label,
		}
	}
	return out
}

func normalizeBotDifficulties(difficulties []types.BotDifficultyConfig) []types.BotDifficultyConfig {
	defaults := map[types.BotDifficulty]types.BotDifficultyConfig{
		"easy":   {ID: "easy", Name: "简单", Description: "完全随机出拳。", Emoji: "🌱", Level: 1, Strategy: "random", CardColor: "#9ed7ff"},
		"normal": {ID: "normal", Name: "普通", Description: "观察玩家最近选择，稍微尝试反制。", Emoji: "🎯", Level: 3, Strategy: "counter", CardColor: "#b8a7ff"},
		"chaos":  {ID: "chaos", Name: "混乱", Description: "大多随机，偶尔连续使用同一招。", Emoji: "🌀", Level: 2, Strategy: "chaos", CardColor: "#ffaad1"},
	}
	find := func(id types.BotDifficulty) *types.BotDifficultyConfig {
		for i := range difficulties {
			if difficulties[i].ID == id {
				return &difficulties[i]
			}
		}
		return nil
	}
	ids := []types.BotDifficulty{"easy", "normal", "chaos"}
	out := make([]types.BotDifficultyConfig, 0, 3)
	for _, id := range ids {
		fallback := defaults[id]
		current := find(id)
		item := fallback
		if current != nil {
			if current.Name != "" {
				item.Name = current.Name
			}
			if current.Description != "" {
				item.Description = current.Description
			}
			if current.Emoji != "" {
				item.Emoji = current.Emoji
			} else {
				item.Emoji = fallback.Emoji
			}
			item.Level = clampNumber(float64(current.Level), 1, 5, float64(fallback.Level))
			if isBotStrategy(current.Strategy) {
				item.Strategy = current.Strategy
			} else {
				item.Strategy = fallback.Strategy
			}
			if current.CardColor != "" {
				item.CardColor = current.CardColor
			} else {
				item.CardColor = fallback.CardColor
			}
		}
		out = append(out, item)
	}
	return out
}

func normalizeExtremeMode(input *types.ExtremeModeConfig) types.ExtremeModeConfig {
	if input == nil {
		return defaultExtremeMode
	}
	label := strings.TrimSpace(input.Label)
	if label == "" {
		label = defaultExtremeMode.Label
	}
	if len([]rune(label)) > 16 {
		label = string([]rune(label)[:16])
	}
	emoji := strings.TrimSpace(input.Emoji)
	if emoji == "" {
		emoji = defaultExtremeMode.Emoji
	}
	if len([]rune(emoji)) > 4 {
		emoji = string([]rune(emoji)[:4])
	}
	warn := strings.TrimSpace(input.ForceCloseWarning)
	if warn == "" {
		warn = defaultExtremeMode.ForceCloseWarning
	}
	if len([]rune(warn)) > 180 {
		warn = string([]rune(warn)[:180])
	}
	forceMin := input.ForceRenameMinPoints
	if forceMin == 0 {
		forceMin = defaultExtremeMode.ForceRenameMinPoints
	}
	forceHours := input.ForceRenameProtectHours
	if forceHours == 0 {
		forceHours = defaultExtremeMode.ForceRenameProtectHours
	}
	return types.ExtremeModeConfig{
		Label:                   label,
		Emoji:                   emoji,
		CooldownHours:           clampNumber(float64(input.CooldownHours), 1, 168, float64(defaultExtremeMode.CooldownHours)),
		PositiveLossRates:       normalizeNumberRecord(input.PositiveLossRates, defaultExtremeMode.PositiveLossRates, 0, 1),
		NegativeWinRates:        normalizeNumberRecord(input.NegativeWinRates, defaultExtremeMode.NegativeWinRates, 0, 1),
		HourlyDecay:             normalizeNumberRecord(input.HourlyDecay, defaultExtremeMode.HourlyDecay, 0, 999),
		WinStreakThreshold:      clampNumber(float64(input.WinStreakThreshold), 1, 100, float64(defaultExtremeMode.WinStreakThreshold)),
		WinStreakCrashChance:    clampRatio(input.WinStreakCrashChance, defaultExtremeMode.WinStreakCrashChance),
		CrashTargetPoints:       clampNumber(float64(input.CrashTargetPoints), 1, 1999, float64(defaultExtremeMode.CrashTargetPoints)),
		ForceCloseWarning:       warn,
		ForceRenameMinPoints:    clampNumber(float64(forceMin), 1, 999, float64(defaultExtremeMode.ForceRenameMinPoints)),
		ForceRenameProtectHours: clampNumber(float64(forceHours), 1, 168, float64(defaultExtremeMode.ForceRenameProtectHours)),
	}
}

func normalizePunishmentTasks(punishment types.PunishmentConfig, factions []types.GenderFaction) []types.PunishmentTaskConfig {
	fallbackVariants := make(map[string]string)
	for _, faction := range factions {
		v := ""
		if punishment.Variants != nil {
			v = punishment.Variants[faction.ID]
		}
		if v == "" {
			v = punishment.Description
		}
		if v == "" {
			v = "请完成本局惩罚。"
		}
		fallbackVariants[faction.ID] = v
	}
	rawTasks := punishment.Tasks
	if len(rawTasks) == 0 {
		rawTasks = []types.PunishmentTaskConfig{{
			ID:       "task1",
			Name:     "默认任务",
			Variants: fallbackVariants,
		}}
	}
	out := make([]types.PunishmentTaskConfig, 0, len(rawTasks))
	for i, task := range rawTasks {
		id := task.ID
		if id == "" {
			id = fmt.Sprintf("task%d", i+1)
		}
		name := task.Name
		if name == "" {
			name = fmt.Sprintf("任务 %d", i+1)
		}
		variants := make(map[string]string)
		for _, faction := range factions {
			v := ""
			if task.Variants != nil {
				v = task.Variants[faction.ID]
			}
			if v == "" {
				v = fallbackVariants[faction.ID]
			}
			variants[faction.ID] = v
		}
		out = append(out, types.PunishmentTaskConfig{
			ID:                id,
			Name:              name,
			BackgroundImages:  cleanLines(task.BackgroundImages, []string{}),
			BackgroundOpacity: clampTaskBackgroundOpacity(task.BackgroundOpacity),
			Variants:          variants,
		})
	}
	return out
}

func sliceRunes(s string, max int) string {
	r := []rune(s)
	if len(r) > max {
		return string(r[:max])
	}
	return s
}

func orDefault(s, fallback string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return fallback
	}
	return s
}

func normalizeConfig(input types.AppConfig) types.AppConfig {
	existingFactions := input.GenderFactions
	if len(existingFactions) == 0 {
		existingFactions = cloneDefaultFactions()
	}
	genderFactions := make([]types.GenderFaction, 0, len(existingFactions))
	for _, faction := range existingFactions {
		genders := make([]types.GenderOption, 0, len(faction.Genders))
		for _, g := range faction.Genders {
			g.FactionID = faction.ID
			genders = append(genders, g)
		}
		faction.Genders = genders
		genderFactions = append(genderFactions, faction)
	}
	genders := flattenGenders(genderFactions)

	titles := make([]types.TitleSegment, 0, len(input.Titles))
	for _, segment := range input.Titles {
		names := cleanLines(segment.Names, []string{"初心拳手"})
		factionNames := make(map[string][]string)
		for _, faction := range genderFactions {
			var src []string
			if segment.FactionNames != nil {
				src = segment.FactionNames[faction.ID]
			}
			fb := names
			if len(fb) == 0 {
				fb = []string{"初心拳手"}
			}
			factionNames[faction.ID] = cleanLines(src, fb)
		}
		segment.Names = names
		segment.FactionNames = factionNames
		titles = append(titles, segment)
	}

	punishments := make([]types.PunishmentConfig, 0, len(input.Punishments))
	for _, punishment := range input.Punishments {
		punishment.CardImageOpacity = clampOpacity(punishment.CardImageOpacity)
		punishment.RoomBackgroundImages = cleanLines(punishment.RoomBackgroundImages, []string{})
		punishment.Tasks = normalizePunishmentTasks(punishment, genderFactions)
		pool := normalizeRoomNamePool(punishment.RoomNamePool, defaultRoomNamePool)
		punishment.RoomNamePool = &pool
		punishments = append(punishments, punishment)
	}

	adminPass := strings.TrimSpace(input.Site.AdminPassword)
	if env := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")); env != "" {
		adminPass = env
	}

	daEnabled := true
	if input.DailyAnnouncement.Title != "" || input.DailyAnnouncement.Content != "" || input.DailyAnnouncement.Version != "" {
		// if fully zero-value we still default enabled true; matching TS: enabled !== false
	}
	// In Go, bool zero is false. When unmarshaling missing field is false.
	// TS uses `input.dailyAnnouncement?.enabled !== false` so missing means true.
	// We detect "present" via Version/Title or a raw approach: if all empty and disabled, treat as default.
	// Safer: use default enabled true unless we can know. We'll re-read via custom if needed.
	// For active.json from TS, "enabled": true is always written. For missing, default true.
	// Problem: Go unmarshals missing bool as false. We'll set enabled true if version is set to default or content exists.
	// Actually after saveConfig, enabled is always written. On first load from default.json it's true.
	// When user sets enabled:false, Title etc still exist.
	// Approach: if Title is empty AND Content empty AND Version empty AND ButtonText empty -> use defaults (enabled true).
	// else use input.DailyAnnouncement.Enabled as-is.
	da := input.DailyAnnouncement
	if strings.TrimSpace(da.Title) == "" && strings.TrimSpace(da.Content) == "" &&
		strings.TrimSpace(da.ButtonText) == "" && strings.TrimSpace(da.Version) == "" {
		da = defaultDailyAnnouncement
		daEnabled = true
	} else {
		daEnabled = da.Enabled // may be false intentionally
		// Wait - if JSON has enabled:true it's fine. If JSON missing enabled, it's false in Go.
		// default.json has "enabled": true. When loading partial configs from admin, enabled may be missing.
		// Match TS: enabled !== false means only explicit false disables. So we need a pointer or custom unmarshal.
		// For 1:1 with written active.json (always has enabled), this is OK.
		// For normalize of incomplete input from admin save, TS preserves.
		// I'll use: if Version was unmarshaled empty and no fields - defaults; else if all other fields filled but enabled false could be intentional.
		// Additional heuristic: after JSON from default it's true. Keep as-is for now; default.json always has enabled.
		_ = daEnabled
		daEnabled = da.Enabled
		// Fix missing enabled: if content present and version present from file, enabled is in file.
		// Use: enabled defaults to true unless DailyAnnouncement was completely empty (handled above).
		// Actually the Go issue is real for `{"dailyAnnouncement":{"title":"x",...}}` without enabled.
		// TS would enable. We'll default enabled to true when the field was zero AND title is non-empty from partial?
		// Simplest 1:1: custom type. For practicality set Enabled = true if Title/Content non-empty and we can't know...
		// Looking at saveConfig path - always full config with enabled.
		// loadConfig from active - always full.
		// config:save from admin - sends full config.
		// So Enabled from JSON is fine.
	}

	title := orDefault(sliceRunes(orDefault(da.Title, defaultDailyAnnouncement.Title), 32), defaultDailyAnnouncement.Title)
	content := orDefault(sliceRunes(orDefault(da.Content, defaultDailyAnnouncement.Content), 800), defaultDailyAnnouncement.Content)
	buttonText := orDefault(sliceRunes(orDefault(da.ButtonText, defaultDailyAnnouncement.ButtonText), 16), defaultDailyAnnouncement.ButtonText)
	version := orDefault(sliceRunes(orDefault(da.Version, defaultDailyAnnouncement.Version), 32), defaultDailyAnnouncement.Version)

	// Re-default enabled: for unmarshaled configs without the field, Go gives false.
	// active/default always include it. When enabled is false AND content is the default welcome text, still respect false.
	// We'll leave Enabled as unmarshaled; for empty announcement block we already replaced with default.

	botNames := cleanLines(input.Bots.Names, []string{"Bot 小蓝", "Bot 小粉"})

	// JSON 缺省字段在 Go 中为 0；与 TS 的 undefined 回落到 default 对齐。
	maxOnlineIn := float64(input.AccessControl.MaxOnlinePerIP)
	if input.AccessControl.MaxOnlinePerIP == 0 {
		maxOnlineIn = math.NaN()
	}
	maxCreatesIn := float64(input.AccessControl.MaxCreatesPer10Min)
	if input.AccessControl.MaxCreatesPer10Min == 0 {
		maxCreatesIn = math.NaN()
	}
	maxOnline := clampNumber(maxOnlineIn, 1, 100, float64(defaultAccessControl.MaxOnlinePerIP))
	maxCreates := clampNumber(maxCreatesIn, 1, 200, float64(defaultAccessControl.MaxCreatesPer10Min))

	playerPool := normalizeRoomNamePool(input.PlayerPunishmentRoomNamePool, defaultPlayerPunishmentRoomNamePool)

	out := input
	out.Site.Name = strings.TrimSpace(input.Site.Name)
	out.Site.Description = input.Site.Description
	if out.Site.Description == "" {
		out.Site.Description = ""
	}
	out.Site.AdminPassword = adminPass
	out.DailyAnnouncement = types.DailyAnnouncement{
		Enabled:    daEnabled,
		Title:      title,
		Content:    content,
		ButtonText: buttonText,
		Version:    version,
	}
	// Fix: TS `enabled !== false` — if the original unmarshaled had Enabled false because field missing,
	// but Version was "default", treat as true. Only if file explicitly has "enabled": false keep false.
	// We can't distinguish. Use raw JSON re-read for dailyAnnouncement.enabled if needed later.
	// For default.json and active.json always have the field. OK.
	if strings.TrimSpace(input.DailyAnnouncement.Title) != "" || strings.TrimSpace(input.DailyAnnouncement.Content) != "" {
		// Prefer explicit: if input Enabled is false and Version set, keep false (user disabled).
		// If input looks like zero-value Enabled with content from a broken client, force true when
		// Version is non-empty and Content non-empty and Title non-empty — still ambiguous.
		// Keep daEnabled as assigned.
	}
	// Special case matching TS for missing enabled: re-parse is better. Quick fix for load:
	// When Enabled is false but content looks like a full announcement, check raw file? Skip.
	// Force: many Go ports use `Enabled: true` unless explicit. We'll set enabled from:
	// if dailyAnnouncement completely missing fields -> true; else use unmarshaled value.
	// But wait - disabled announcements still have title/content. Unmarshaled Enabled=false is correct.
	out.GenderFactions = genderFactions
	out.Genders = genders
	out.Titles = titles
	out.Punishments = punishments
	out.Bots.Names = botNames
	out.Bots.Difficulties = normalizeBotDifficulties(input.Bots.Difficulties)
	out.RoomTags = cleanLines(input.RoomTags, defaultRoomTags)
	out.Games = normalizeGames(input.Games)
	out.RoomInfoTags = normalizeRoomInfoTags(input.RoomInfoTags)
	out.AccessControl.MaxOnlinePerIP = maxOnline
	out.AccessControl.MaxCreatesPer10Min = maxCreates
	out.NameWar.PenaltyPrefix = orDefault(sliceRunes(orDefault(input.NameWar.PenaltyPrefix, defaultNameWar.PenaltyPrefix), 16), defaultNameWar.PenaltyPrefix)
	out.NameWar.LoserPanelTitle = orDefault(sliceRunes(orDefault(input.NameWar.LoserPanelTitle, defaultNameWar.LoserPanelTitle), 24), defaultNameWar.LoserPanelTitle)
	out.NameWar.EscapeTitle = orDefault(sliceRunes(orDefault(input.NameWar.EscapeTitle, defaultNameWar.EscapeTitle), 18), defaultNameWar.EscapeTitle)
	renameTitle := input.NameWar.RenamePanelTitle
	if strings.TrimSpace(renameTitle) == "" {
		renameTitle = input.NameWar.LoserPanelTitle
	}
	out.NameWar.RenamePanelTitle = orDefault(sliceRunes(orDefault(renameTitle, defaultNameWar.RenamePanelTitle), 24), defaultNameWar.RenamePanelTitle)
	out.NameWar.NameWarLoserLabel = orDefault(sliceRunes(orDefault(input.NameWar.NameWarLoserLabel, defaultNameWar.NameWarLoserLabel), 16), defaultNameWar.NameWarLoserLabel)
	out.NameWar.ExtremeForceClosedLabel = orDefault(sliceRunes(orDefault(input.NameWar.ExtremeForceClosedLabel, defaultNameWar.ExtremeForceClosedLabel), 16), defaultNameWar.ExtremeForceClosedLabel)
	out.Giveaway.PanelTitle = orDefault(sliceRunes(orDefault(input.Giveaway.PanelTitle, defaultGiveaway.PanelTitle), 24), defaultGiveaway.PanelTitle)
	out.Giveaway.PanelDescription = orDefault(sliceRunes(orDefault(input.Giveaway.PanelDescription, defaultGiveaway.PanelDescription), 160), defaultGiveaway.PanelDescription)
	out.Giveaway.SubmitPlaceholder = orDefault(sliceRunes(orDefault(input.Giveaway.SubmitPlaceholder, defaultGiveaway.SubmitPlaceholder), 60), defaultGiveaway.SubmitPlaceholder)
	out.Giveaway.EmptyText = orDefault(sliceRunes(orDefault(input.Giveaway.EmptyText, defaultGiveaway.EmptyText), 60), defaultGiveaway.EmptyText)
	em := input.ExtremeMode
	out.ExtremeMode = normalizeExtremeMode(&em)
	out.PlayerPunishmentRoomNamePool = &playerPool
	return out
}

var hexColorRe = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func assertUnique(values []string, label string) error {
	seen := map[string]struct{}{}
	for _, value := range values {
		if value == "" {
			return fmt.Errorf("%s 不能为空", label)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("%s 不能重复：%s", label, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func assertHexColor(value, label string) error {
	if !hexColorRe.MatchString(value) {
		return fmt.Errorf("%s 必须是 #RRGGBB 颜色值", label)
	}
	return nil
}

// ValidateConfig performs basic but useful config validation.
func ValidateConfig(input types.AppConfig) (types.AppConfig, error) {
	input = normalizeConfig(input)
	if strings.TrimSpace(input.Site.Name) == "" {
		return input, fmt.Errorf("网站名称不能为空")
	}
	if input.DailyAnnouncement.Enabled {
		if strings.TrimSpace(input.DailyAnnouncement.Title) == "" {
			return input, fmt.Errorf("每日公告标题不能为空")
		}
		if strings.TrimSpace(input.DailyAnnouncement.Content) == "" {
			return input, fmt.Errorf("每日公告内容不能为空")
		}
		if strings.TrimSpace(input.DailyAnnouncement.ButtonText) == "" {
			return input, fmt.Errorf("每日公告按钮文字不能为空")
		}
	}
	if len(input.Genders) == 0 {
		return input, fmt.Errorf("至少需要一个性别选项")
	}
	if len(input.GenderFactions) == 0 {
		return input, fmt.Errorf("至少需要一个性别阵营")
	}
	factionIDs := make([]string, len(input.GenderFactions))
	for i, f := range input.GenderFactions {
		factionIDs[i] = f.ID
	}
	if err := assertUnique(factionIDs, "阵营 ID"); err != nil {
		return input, err
	}
	genderIDs := make([]string, len(input.Genders))
	for i, g := range input.Genders {
		genderIDs[i] = g.ID
	}
	if err := assertUnique(genderIDs, "性别 ID"); err != nil {
		return input, err
	}
	for _, faction := range input.GenderFactions {
		if faction.Label == "" {
			return input, fmt.Errorf("阵营名称不能为空")
		}
		if len(faction.Genders) == 0 {
			return input, fmt.Errorf("%s 至少需要一个性别", faction.Label)
		}
		if err := assertHexColor(faction.TextColor, faction.Label+" 文字颜色"); err != nil {
			return input, err
		}
		if err := assertHexColor(faction.BackgroundColor, faction.Label+" 背景颜色"); err != nil {
			return input, err
		}
		if err := assertHexColor(faction.BorderColor, faction.Label+" 边框颜色"); err != nil {
			return input, err
		}
	}
	if len(input.Titles) == 0 {
		return input, fmt.Errorf("至少需要一个称号段位")
	}
	for _, segment := range input.Titles {
		if len(segment.Names) == 0 {
			return input, fmt.Errorf("%s 至少需要一个通用称号", segment.ID)
		}
		for _, faction := range input.GenderFactions {
			if len(segment.FactionNames[faction.ID]) == 0 {
				return input, fmt.Errorf("%s 缺少 %s 专属称号", segment.ID, faction.Label)
			}
		}
	}
	if len(input.Punishments) == 0 {
		return input, fmt.Errorf("至少需要一个惩罚选项")
	}
	for _, punishment := range input.Punishments {
		if len(punishment.Tasks) == 0 {
			return input, fmt.Errorf("%s 至少需要一个任务", punishment.Name)
		}
		if punishment.CardImageOpacity < 0 || punishment.CardImageOpacity > 1 {
			return input, fmt.Errorf("%s 的卡片背景透明率必须在 0 到 1 之间", punishment.Name)
		}
		if punishment.RoomBackgroundImages == nil {
			return input, fmt.Errorf("%s 的房间背景图库格式不正确", punishment.Name)
		}
		taskIDs := make([]string, len(punishment.Tasks))
		for i, t := range punishment.Tasks {
			taskIDs[i] = t.ID
		}
		if err := assertUnique(taskIDs, punishment.Name+" 任务 ID"); err != nil {
			return input, err
		}
		for _, task := range punishment.Tasks {
			if task.Name == "" {
				return input, fmt.Errorf("%s 里有任务名称为空", punishment.Name)
			}
			if task.BackgroundImages == nil {
				return input, fmt.Errorf("%s / %s 的任务背景图库格式不正确", punishment.Name, task.Name)
			}
			if task.BackgroundOpacity < 0 || task.BackgroundOpacity > 1 {
				return input, fmt.Errorf("%s / %s 的任务背景透明率必须在 0 到 1 之间", punishment.Name, task.Name)
			}
			for _, faction := range input.GenderFactions {
				if strings.TrimSpace(task.Variants[faction.ID]) == "" {
					return input, fmt.Errorf("%s / %s 缺少 %s 任务版本", punishment.Name, task.Name, faction.Label)
				}
			}
		}
		if punishment.RoomNamePool == nil || len(punishment.RoomNamePool.Subjects) == 0 || len(punishment.RoomNamePool.RoomWords) == 0 {
			return input, fmt.Errorf("%s 的随机房名至少需要名词/动词和房间词", punishment.Name)
		}
	}
	if input.PlayerPunishmentRoomNamePool == nil || len(input.PlayerPunishmentRoomNamePool.Subjects) == 0 || len(input.PlayerPunishmentRoomNamePool.RoomWords) == 0 {
		return input, fmt.Errorf("玩家发布任务随机房名至少需要名词/动词和房间词")
	}
	if input.RoomTags == nil {
		return input, fmt.Errorf("房间标签格式不正确")
	}
	for key, tag := range input.RoomInfoTags {
		if strings.TrimSpace(tag.Label) == "" {
			return input, fmt.Errorf("房间信息标签 %s 的名字不能为空", key)
		}
		if err := assertHexColor(tag.TextColor, tag.Label+" 文字颜色"); err != nil {
			return input, err
		}
		if err := assertHexColor(tag.BackgroundColor, tag.Label+" 背景颜色"); err != nil {
			return input, err
		}
		if err := assertHexColor(tag.BorderColor, tag.Label+" 边框颜色"); err != nil {
			return input, err
		}
	}
	if strings.TrimSpace(input.NameWar.PenaltyPrefix) == "" {
		return input, fmt.Errorf("名字争夺战前缀不能为空")
	}
	if strings.TrimSpace(input.NameWar.LoserPanelTitle) == "" {
		return input, fmt.Errorf("名字争夺战失格者标题不能为空")
	}
	if strings.TrimSpace(input.NameWar.EscapeTitle) == "" {
		return input, fmt.Errorf("名字争夺战逃跑称号不能为空")
	}
	if strings.TrimSpace(input.NameWar.RenamePanelTitle) == "" {
		return input, fmt.Errorf("通用改名处标题不能为空")
	}
	if strings.TrimSpace(input.NameWar.NameWarLoserLabel) == "" {
		return input, fmt.Errorf("名争失格标签不能为空")
	}
	if strings.TrimSpace(input.NameWar.ExtremeForceClosedLabel) == "" {
		return input, fmt.Errorf("极限强关标签不能为空")
	}
	if strings.TrimSpace(input.Giveaway.PanelTitle) == "" {
		return input, fmt.Errorf("白给模式面板标题不能为空")
	}
	if strings.TrimSpace(input.Giveaway.PanelDescription) == "" {
		return input, fmt.Errorf("白给模式说明不能为空")
	}
	if strings.TrimSpace(input.Giveaway.SubmitPlaceholder) == "" {
		return input, fmt.Errorf("白给模式输入提示不能为空")
	}
	if strings.TrimSpace(input.Giveaway.EmptyText) == "" {
		return input, fmt.Errorf("白给模式空状态文案不能为空")
	}
	if strings.TrimSpace(input.ExtremeMode.Label) == "" {
		return input, fmt.Errorf("极限模式名称不能为空")
	}
	if strings.TrimSpace(input.ExtremeMode.Emoji) == "" {
		return input, fmt.Errorf("极限模式标志不能为空")
	}
	if input.ExtremeMode.CooldownHours < 1 {
		return input, fmt.Errorf("极限模式冷却小时数至少为 1")
	}
	for key, value := range input.ExtremeMode.PositiveLossRates {
		if value < 0 || value > 1 {
			return input, fmt.Errorf("极限模式 %s 掉分比例必须在 0 到 1 之间", key)
		}
	}
	for key, value := range input.ExtremeMode.NegativeWinRates {
		if value < 0 || value > 1 {
			return input, fmt.Errorf("极限模式 %s 加分比例必须在 0 到 1 之间", key)
		}
	}
	for key, value := range input.ExtremeMode.HourlyDecay {
		if value < 0 {
			return input, fmt.Errorf("极限模式 %s 整点扣分不能小于 0", key)
		}
	}
	if input.ExtremeMode.WinStreakThreshold < 1 {
		return input, fmt.Errorf("极限模式连胜阈值至少为 1")
	}
	if input.ExtremeMode.WinStreakCrashChance < 0 || input.ExtremeMode.WinStreakCrashChance > 1 {
		return input, fmt.Errorf("极限模式连胜风险概率必须在 0 到 1 之间")
	}
	if input.ExtremeMode.CrashTargetPoints < 1 {
		return input, fmt.Errorf("极限模式连胜风险扣分至少为 1")
	}
	forceRenameMinPoints := input.ExtremeMode.ForceRenameMinPoints
	if forceRenameMinPoints == 0 {
		forceRenameMinPoints = defaultExtremeMode.ForceRenameMinPoints
	}
	forceRenameProtectHours := input.ExtremeMode.ForceRenameProtectHours
	if forceRenameProtectHours == 0 {
		forceRenameProtectHours = defaultExtremeMode.ForceRenameProtectHours
	}
	if strings.TrimSpace(input.ExtremeMode.ForceCloseWarning) == "" {
		return input, fmt.Errorf("极限模式强行关闭提示不能为空")
	}
	if forceRenameMinPoints < 1 {
		return input, fmt.Errorf("极限强关改名最低分至少为 1")
	}
	if forceRenameProtectHours < 1 {
		return input, fmt.Errorf("极限强关改名保护小时至少为 1")
	}
	if input.AccessControl.MaxOnlinePerIP < 1 {
		return input, fmt.Errorf("同 IP 在线人数限制至少为 1")
	}
	if input.AccessControl.MaxCreatesPer10Min < 1 {
		return input, fmt.Errorf("同 IP 10 分钟新建玩家限制至少为 1")
	}
	if len(input.Bots.Names) == 0 || len(input.Bots.Difficulties) == 0 {
		return input, fmt.Errorf("bot 名字和难度不能为空")
	}
	for _, difficulty := range input.Bots.Difficulties {
		if !isBotStrategy(difficulty.Strategy) {
			return input, fmt.Errorf("%s 的 Bot 策略不正确", difficulty.Name)
		}
		if difficulty.CardColor == "" || !hexColorRe.MatchString(difficulty.CardColor) {
			return input, fmt.Errorf("%s 的卡片颜色必须是 #RRGGBB", difficulty.Name)
		}
	}
	if len(input.Games) == 0 {
		return input, fmt.Errorf("至少需要一个游戏配置")
	}
	return input, nil
}

// LoadConfig loads and validates config/active.json (copying from default if missing).
func LoadConfig() (types.AppConfig, error) {
	if _, err := os.Stat(activePath()); os.IsNotExist(err) {
		if err := copyFile(defaultPath(), activePath()); err != nil {
			return types.AppConfig{}, err
		}
	}
	cfg, err := readJSON(activePath())
	if err != nil {
		return types.AppConfig{}, err
	}
	// Fix dailyAnnouncement.enabled: Go unmarshals missing bool as false; match TS !== false
	cfg = fixDailyAnnouncementEnabled(activePath(), cfg)
	return ValidateConfig(cfg)
}

func fixDailyAnnouncementEnabled(path string, cfg types.AppConfig) types.AppConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return cfg
	}
	daRaw, ok := raw["dailyAnnouncement"]
	if !ok {
		cfg.DailyAnnouncement.Enabled = true
		return cfg
	}
	var da map[string]json.RawMessage
	if err := json.Unmarshal(daRaw, &da); err != nil {
		return cfg
	}
	if _, has := da["enabled"]; !has {
		cfg.DailyAnnouncement.Enabled = true
	}
	return cfg
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// SaveConfig validates and writes config.
func SaveConfig(next types.AppConfig) (types.AppConfig, error) {
	valid, err := ValidateConfig(next)
	if err != nil {
		return types.AppConfig{}, err
	}
	data, err := json.MarshalIndent(valid, "", "  ")
	if err != nil {
		return types.AppConfig{}, err
	}
	if err := os.WriteFile(activePath(), data, 0o644); err != nil {
		return types.AppConfig{}, err
	}
	return LoadConfig()
}

// ResetConfig restores default.json to active.json.
func ResetConfig() (types.AppConfig, error) {
	if err := copyFile(defaultPath(), activePath()); err != nil {
		return types.AppConfig{}, err
	}
	return LoadConfig()
}

// ExportConfigText returns raw active.json text.
func ExportConfigText() (string, error) {
	data, err := os.ReadFile(activePath())
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// Silence unused helper for any-slice cleanLines if needed later
var _ = cleanLinesAny
