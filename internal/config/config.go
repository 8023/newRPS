// Package config 从 config/ 下按功能拆分的 JSON 文件原地读写配置。
// 不在代码中维护第二套默认文案；文件缺失或校验失败直接返回 error。
//
// 文件一览（相对 config/）：
//
//	site.json, announcement-board.json, security-disclaimer.json, genders.json, gender-factions.json,
//	titles.json, punishments.json, player-punishment-room-name-pool.json, room-tags.json,
//	room-info-tags.json, access-control.json, name-war.json, giveaway.json,
//	extreme-mode.json, ranked-score.json, bots.json, games.json, messages.json
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

// 配置文件权限：仅属主可读写（含明文管理员口令，禁止同机其他用户读取）。
const configFileMode = 0o600

// 拆分后的功能配置文件名（缺一即加载失败，除非仍存在旧单体文件可迁移）。
var splitConfigFiles = []string{
	"site.json",
	"announcement-board.json",
	"security-disclaimer.json",
	"genders.json",
	"gender-factions.json",
	"titles.json",
	"punishments.json",
	"player-punishment-room-name-pool.json",
	"room-tags.json",
	"room-info-tags.json",
	"access-control.json",
	"name-war.json",
	"giveaway.json",
	"extreme-mode.json",
	"ranked-score.json",
	"bots.json",
	"games.json",
	"messages.json",
}

var rootDir string

func init() {
	rootDir = findRootDir()
}

func findRootDir() string {
	candidates := []string{}
	if wd, err := os.Getwd(); err == nil {
		// 从 cwd 向上查找含 config/ 的仓库根（兼容 go test 在子包目录执行）
		dir := wd
		for i := 0; i < 8; i++ {
			candidates = append(candidates, dir)
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
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
		if isConfigRoot(abs) {
			return abs
		}
	}
	if wd, err := os.Getwd(); err == nil {
		return wd
	}
	return "."
}

func isConfigRoot(abs string) bool {
	cfg := filepath.Join(abs, "config")
	if _, err := os.Stat(filepath.Join(cfg, "site.json")); err == nil {
		return true
	}
	// 兼容尚未迁移的旧单体
	if _, err := os.Stat(filepath.Join(cfg, "active.json")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(cfg, "default.json")); err == nil {
		return true
	}
	return false
}

func GetRootDir() string {
	return rootDir
}

func configDir() string {
	return filepath.Join(rootDir, "config")
}

func configPath(name string) string {
	return filepath.Join(configDir(), name)
}

func readJSONFile(filePath string, dest any) error {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("读取配置失败 %s: %w", filePath, err)
	}
	if err := json.Unmarshal(data, dest); err != nil {
		return fmt.Errorf("解析配置失败 %s: %w", filePath, err)
	}
	return nil
}

func writeJSONFile(filePath string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filePath, data, configFileMode); err != nil {
		return fmt.Errorf("写入配置失败 %s: %w", filePath, err)
	}
	// umask 可能收紧权限，显式保证可写
	_ = os.Chmod(filePath, configFileMode)
	return nil
}

func cleanLines(values []string) []string {
	items := make([]string, 0, len(values))
	for _, v := range values {
		t := strings.TrimSpace(v)
		if t != "" {
			items = append(items, t)
		}
	}
	return items
}

func clampNumber(value float64, min, max float64) int {
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return int(min)
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
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 1 {
		return 1
	}
	return value
}

func clampRatio(value float64) float64 {
	return clampOpacity(value)
}

func isBotStrategy(value types.BotStrategy) bool {
	switch value {
	case "random", "counter", "chaos", "throw", "win":
		return true
	default:
		return false
	}
}

func sliceRunes(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > max {
		return string(r[:max])
	}
	return string(r)
}

func normalizeRoomNamePool(pool *types.RoomNamePool) *types.RoomNamePool {
	if pool == nil {
		return &types.RoomNamePool{Adjectives: []string{}, Subjects: []string{}, RoomWords: []string{}}
	}
	return &types.RoomNamePool{
		Adjectives: cleanLines(pool.Adjectives),
		Subjects:   cleanLines(pool.Subjects),
		RoomWords:  cleanLines(pool.RoomWords),
	}
}

func normalizeNumberRecord(input map[string]float64, min, max float64) map[string]float64 {
	if input == nil {
		return map[string]float64{}
	}
	out := make(map[string]float64, len(input))
	for key, v := range input {
		if math.IsNaN(v) || math.IsInf(v, 0) {
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

// distinctTaskGroups 按出现顺序去重收集当前配置里用到的任务分组（生理男/生理女/默认兜底）。
func distinctTaskGroups(factions []types.GenderFaction) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, faction := range factions {
		group := faction.TaskGroup
		if group == "" {
			continue
		}
		if _, ok := seen[group]; ok {
			continue
		}
		seen[group] = struct{}{}
		out = append(out, group)
	}
	return out
}

func normalizePunishmentTasks(punishment types.PunishmentConfig, groups []string) []types.PunishmentTaskConfig {
	out := make([]types.PunishmentTaskConfig, 0, len(punishment.Tasks))
	for i, task := range punishment.Tasks {
		id := strings.TrimSpace(task.ID)
		if id == "" {
			id = fmt.Sprintf("task%d", i+1)
		}
		name := strings.TrimSpace(task.Name)
		variants := make(map[string]string)
		for _, group := range groups {
			v := ""
			if task.Variants != nil {
				v = strings.TrimSpace(task.Variants[group])
			}
			variants[group] = v
		}
		out = append(out, types.PunishmentTaskConfig{
			ID:                id,
			Name:              name,
			BackgroundImages:  cleanLines(task.BackgroundImages),
			BackgroundOpacity: clampOpacity(task.BackgroundOpacity),
			Variants:          variants,
		})
	}
	return out
}

// normalizeConfig 只做清洗（trim、空切片、数值夹紧），不注入代码内默认文案。
func normalizeConfig(input types.AppConfig) types.AppConfig {
	genders := make([]types.GenderOption, 0, len(input.Genders))
	for _, g := range input.Genders {
		g.ID = strings.TrimSpace(g.ID)
		g.Label = strings.TrimSpace(g.Label)
		g.FactionID = strings.TrimSpace(g.FactionID)
		genders = append(genders, g)
	}

	genderFactions := make([]types.GenderFaction, 0, len(input.GenderFactions))
	for _, faction := range input.GenderFactions {
		faction.ID = strings.TrimSpace(faction.ID)
		faction.Label = strings.TrimSpace(faction.Label)
		faction.TaskGroup = strings.TrimSpace(faction.TaskGroup)
		if faction.TaskGroup == "" {
			faction.TaskGroup = "default"
		}
		genderFactions = append(genderFactions, faction)
	}

	groups := distinctTaskGroups(genderFactions)

	titles := make([]types.TitleSegment, 0, len(input.Titles))
	for _, segment := range input.Titles {
		segment.ID = strings.TrimSpace(segment.ID)
		segment.Names = cleanLines(segment.Names)
		factionNames := make(map[string][]string)
		for _, group := range groups {
			var src []string
			if segment.FactionNames != nil {
				src = segment.FactionNames[group]
			}
			factionNames[group] = cleanLines(src)
		}
		segment.FactionNames = factionNames
		titles = append(titles, segment)
	}

	punishments := make([]types.PunishmentConfig, 0, len(input.Punishments))
	for _, punishment := range input.Punishments {
		punishment.ID = strings.TrimSpace(punishment.ID)
		punishment.Name = strings.TrimSpace(punishment.Name)
		punishment.Description = strings.TrimSpace(punishment.Description)
		punishment.CardImageOpacity = clampOpacity(punishment.CardImageOpacity)
		punishment.RoomBackgroundImages = cleanLines(punishment.RoomBackgroundImages)
		punishment.Tasks = normalizePunishmentTasks(punishment, groups)
		punishment.RoomNamePool = normalizeRoomNamePool(punishment.RoomNamePool)
		punishments = append(punishments, punishment)
	}

	adminPass := strings.TrimSpace(input.Site.AdminPassword)
	if env := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")); env != "" {
		adminPass = env
	}

	ab := input.AnnouncementBoard
	ab.Title = sliceRunes(ab.Title, 32)
	ab.Content = sliceRunes(ab.Content, 800)

	games := make([]types.GameConfig, 0, len(input.Games))
	for _, g := range input.Games {
		if g.ID != "rps" && g.ID != "othello" && g.ID != "tictactoe" && g.ID != "liarsdice" && g.ID != "gomoku" {
			continue
		}
		g.Name = sliceRunes(g.Name, 18)
		g.Description = sliceRunes(g.Description, 120)
		games = append(games, g)
	}

	botDifficulties := make([]types.BotDifficultyConfig, 0, len(input.Bots.Difficulties))
	for _, d := range input.Bots.Difficulties {
		d.ID = types.BotDifficulty(strings.TrimSpace(string(d.ID)))
		d.Name = strings.TrimSpace(d.Name)
		d.Description = strings.TrimSpace(d.Description)
		d.Emoji = sliceRunes(d.Emoji, 4)
		d.CardColor = strings.TrimSpace(d.CardColor)
		if d.Level < 1 {
			d.Level = 1
		}
		if d.Level > 5 {
			d.Level = 5
		}
		botDifficulties = append(botDifficulties, d)
	}

	roomInfoTags := map[string]types.RoomInfoTagStyle{}
	for key, tag := range input.RoomInfoTags {
		tag.Label = sliceRunes(tag.Label, 16)
		tag.TextColor = strings.TrimSpace(tag.TextColor)
		tag.BackgroundColor = strings.TrimSpace(tag.BackgroundColor)
		tag.BorderColor = strings.TrimSpace(tag.BorderColor)
		roomInfoTags[key] = tag
	}

	em := input.ExtremeMode
	em.Label = sliceRunes(em.Label, 16)
	em.Emoji = sliceRunes(em.Emoji, 4)
	em.ForceCloseWarning = sliceRunes(em.ForceCloseWarning, 180)
	em.PositiveLossRates = normalizeNumberRecord(em.PositiveLossRates, 0, 1)
	em.NegativeWinRates = normalizeNumberRecord(em.NegativeWinRates, 0, 1)
	em.HourlyDecay = normalizeNumberRecord(em.HourlyDecay, 0, 999)
	if em.CooldownHours > 0 {
		em.CooldownHours = clampNumber(float64(em.CooldownHours), 1, 168)
	}
	if em.WinStreakThreshold > 0 {
		em.WinStreakThreshold = clampNumber(float64(em.WinStreakThreshold), 1, 100)
	}
	em.WinStreakCrashChance = clampRatio(em.WinStreakCrashChance)
	if em.CrashTargetPoints > 0 {
		em.CrashTargetPoints = clampNumber(float64(em.CrashTargetPoints), 1, 1999)
	}
	if em.ForceRenameMinPoints > 0 {
		em.ForceRenameMinPoints = clampNumber(float64(em.ForceRenameMinPoints), 1, 999)
	}
	if em.ForceRenameProtectHours > 0 {
		em.ForceRenameProtectHours = clampNumber(float64(em.ForceRenameProtectHours), 1, 168)
	}

	out := input
	out.Site.Name = strings.TrimSpace(input.Site.Name)
	out.Site.Description = strings.TrimSpace(input.Site.Description)
	out.Site.AdminPassword = adminPass
	out.AnnouncementBoard = ab
	out.GenderFactions = genderFactions
	out.Genders = genders
	out.Titles = titles
	out.Punishments = punishments
	out.Bots.Names = cleanLines(input.Bots.Names)
	out.Bots.Difficulties = botDifficulties
	out.RoomTags = cleanLines(input.RoomTags)
	out.Games = games
	out.RoomInfoTags = roomInfoTags
	out.ExtremeMode = em
	out.PlayerPunishmentRoomNamePool = normalizeRoomNamePool(input.PlayerPunishmentRoomNamePool)
	out.NameWar.PenaltyPrefix = sliceRunes(input.NameWar.PenaltyPrefix, 16)
	out.NameWar.LoserPanelTitle = sliceRunes(input.NameWar.LoserPanelTitle, 24)
	out.NameWar.EscapeTitle = sliceRunes(input.NameWar.EscapeTitle, 18)
	out.NameWar.RenamePanelTitle = sliceRunes(input.NameWar.RenamePanelTitle, 24)
	if out.NameWar.RenamePanelTitle == "" {
		out.NameWar.RenamePanelTitle = out.NameWar.LoserPanelTitle
	}
	out.NameWar.NameWarLoserLabel = sliceRunes(input.NameWar.NameWarLoserLabel, 16)
	out.NameWar.ExtremeForceClosedLabel = sliceRunes(input.NameWar.ExtremeForceClosedLabel, 16)
	out.Giveaway.PanelTitle = sliceRunes(input.Giveaway.PanelTitle, 24)
	out.Giveaway.PanelDescription = sliceRunes(input.Giveaway.PanelDescription, 160)
	out.Giveaway.SubmitPlaceholder = sliceRunes(input.Giveaway.SubmitPlaceholder, 60)
	out.Giveaway.EmptyText = sliceRunes(input.Giveaway.EmptyText, 60)
	// AccessControl：0 表示非法，留给 Validate 报错，不做静默默认
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

// ValidateConfig 清洗并校验；不完整配置直接 error，不注入代码默认内容。
func ValidateConfig(input types.AppConfig) (types.AppConfig, error) {
	input = normalizeConfig(input)
	if strings.TrimSpace(input.Site.Name) == "" {
		return input, fmt.Errorf("网站名称不能为空")
	}
	if input.AnnouncementBoard.Enabled {
		if strings.TrimSpace(input.AnnouncementBoard.Title) == "" {
			return input, fmt.Errorf("公告板标题不能为空")
		}
		if strings.TrimSpace(input.AnnouncementBoard.Content) == "" {
			return input, fmt.Errorf("公告板内容不能为空")
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
	factionIDSet := make(map[string]struct{}, len(factionIDs))
	for _, id := range factionIDs {
		factionIDSet[id] = struct{}{}
	}
	genderLabels := make(map[string]struct{}, len(input.Genders))
	for _, g := range input.Genders {
		if g.Label == "" {
			return input, fmt.Errorf("性别显示文字不能为空")
		}
		if _, dup := genderLabels[g.Label]; dup {
			return input, fmt.Errorf("性别显示文字 %s 重复", g.Label)
		}
		genderLabels[g.Label] = struct{}{}
		if g.FactionID == "" {
			return input, fmt.Errorf("性别 %s 必须归属一个阵营", g.Label)
		}
		if _, ok := factionIDSet[g.FactionID]; !ok {
			return input, fmt.Errorf("性别 %s 所属阵营 %s 不存在", g.Label, g.FactionID)
		}
	}
	for _, faction := range input.GenderFactions {
		if faction.Label == "" {
			return input, fmt.Errorf("阵营名称不能为空")
		}
		if faction.TaskGroup != "male" && faction.TaskGroup != "female" && faction.TaskGroup != "default" {
			return input, fmt.Errorf("%s 的任务分组必须是生理男/生理女/默认兜底之一", faction.Label)
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
	taskGroups := distinctTaskGroups(input.GenderFactions)
	for _, segment := range input.Titles {
		if segment.MinPercent < -100 || segment.MaxPercent > 100 {
			return input, fmt.Errorf("%s 的百分比范围必须在 -100 到 100 之间", segment.ID)
		}
		if segment.MinPercent > segment.MaxPercent {
			return input, fmt.Errorf("%s 的最小百分比不能大于最大百分比", segment.ID)
		}
		if len(segment.Names) == 0 {
			return input, fmt.Errorf("%s 至少需要一个通用称号", segment.ID)
		}
		for _, group := range taskGroups {
			if len(segment.FactionNames[group]) == 0 {
				return input, fmt.Errorf("%s 缺少 %s 分组专属称号", segment.ID, group)
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
			for _, group := range taskGroups {
				if strings.TrimSpace(task.Variants[group]) == "" {
					return input, fmt.Errorf("%s / %s 缺少 %s 分组任务版本", punishment.Name, task.Name, group)
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
	if input.NameWar.PenaltyThreshold >= 0 {
		return input, fmt.Errorf("名字争夺战失格分阈值必须为负数")
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
	if input.Giveaway.ActiveBoostValue <= 0 || input.Giveaway.ActiveBoostValue > 100 {
		return input, fmt.Errorf("主动白给增量必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.WinPenaltyValue <= 0 || input.Giveaway.WinPenaltyValue > 100 {
		return input, fmt.Errorf("胜利白给扣减值必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.LikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("点赞每小时次数上限至少为 1")
	}
	if input.Giveaway.LikeVoteValue <= 0 || input.Giveaway.LikeVoteValue > 100 {
		return input, fmt.Errorf("点赞降低值必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.DislikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("倒赞每小时次数上限至少为 1")
	}
	if input.Giveaway.DislikeVoteValue <= 0 || input.Giveaway.DislikeVoteValue > 100 {
		return input, fmt.Errorf("倒赞增加值必须在 0（不含）到 100 之间")
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
	if strings.TrimSpace(input.ExtremeMode.ForceCloseWarning) == "" {
		return input, fmt.Errorf("极限模式强行关闭提示不能为空")
	}
	if input.ExtremeMode.ForceRenameMinPoints < 1 {
		return input, fmt.Errorf("极限强关改名最低分至少为 1")
	}
	if input.ExtremeMode.ForceRenameProtectHours < 1 {
		return input, fmt.Errorf("极限强关改名保护小时至少为 1")
	}
	if input.RankedScore.Max < 1 {
		return input, fmt.Errorf("排位分显示上限必须大于 0")
	}
	if input.RankedScore.Min >= 0 || input.RankedScore.Min >= input.RankedScore.Max {
		return input, fmt.Errorf("排位分普通玩家显示下限必须为负且小于上限")
	}
	if input.RankedScore.NameWarMin > input.RankedScore.Min {
		return input, fmt.Errorf("抢名大战显示下限不能高于普通玩家显示下限")
	}
	if input.RankedScore.DailyDecayRatio <= 0 || input.RankedScore.DailyDecayRatio > 1 {
		return input, fmt.Errorf("每日衰减比例必须在 0（不含）到 1（含）之间")
	}
	if input.AccessControl.MaxOnlinePerIP < 1 {
		return input, fmt.Errorf("同设备在线人数限制至少为 1")
	}
	if input.AccessControl.MaxCreatesPer10Min < 1 {
		return input, fmt.Errorf("同设备 10 分钟新建玩家限制至少为 1")
	}
	if input.AccessControl.IPBackstopMultiplier < 1 {
		return input, fmt.Errorf("IP 兜底限流倍数至少为 1")
	}
	if input.AccessControl.IPBackstopMinLimit < 1 {
		return input, fmt.Errorf("IP 兜底限流最低下限至少为 1")
	}
	if input.AccessControl.MaxSessionIssuePerIP < 1 {
		return input, fmt.Errorf("同 IP 10 分钟签发会话上限至少为 1")
	}
	if input.AccessControl.MaxOnlinePerIPTotal < 1 {
		return input, fmt.Errorf("同 IP 在线人数上限至少为 1")
	}
	if input.AccessControl.MaxCreatesPerIP < 1 {
		return input, fmt.Errorf("同 IP 10 分钟新建玩家上限至少为 1")
	}
	if input.AccessControl.MaxActiveRoomsPerOwner < 1 {
		return input, fmt.Errorf("单玩家同时开房数量上限至少为 1")
	}
	if input.AccessControl.MaxProofUploadsPerPlayer < 1 {
		return input, fmt.Errorf("同玩家 10 分钟证明图上传上限至少为 1")
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

// LoadConfig 读取 config/ 下拆分的 JSON；若仅有旧版单体 default/active 则自动迁移。
func LoadConfig() (types.AppConfig, error) {
	if err := ensureSplitConfig(); err != nil {
		return types.AppConfig{}, err
	}
	// 官方升级流程明确要求整目录保留旧 config/（见 README「升级服务器且不丢玩家数据」），
	// 不会被新版本覆盖；本版本新增/改名的拆分文件必须能从旧目录状态自愈，否则所有已部署
	// 实例升级后会因缺文件直接启动失败。
	if err := migrateAnnouncementBoardFile(); err != nil {
		return types.AppConfig{}, err
	}
	if err := ensureSecurityDisclaimerFile(); err != nil {
		return types.AppConfig{}, err
	}
	if err := migrateGendersFile(); err != nil {
		return types.AppConfig{}, err
	}
	cfg, err := readSplitConfig()
	if err != nil {
		return types.AppConfig{}, err
	}
	cfg = fixAnnouncementBoardEnabled(configPath("announcement-board.json"), cfg)
	valid, err := ValidateConfig(cfg)
	if err != nil {
		return types.AppConfig{}, fmt.Errorf("配置校验失败: %w", err)
	}
	return valid, nil
}

// migrateAnnouncementBoardFile：v2.1.28 把 daily-announcement.json 改名为
// announcement-board.json（并丢弃了 buttonText/version 字段）。announcement-board.json
// 缺失但旧文件还在时，从旧文件迁出 enabled/title/content，旧文件改名为 .bak；
// 两个文件都不存在则什么也不做，交给 readSplitConfig 报出清晰的缺文件错误。
func migrateAnnouncementBoardFile() error {
	newPath := configPath("announcement-board.json")
	if _, err := os.Stat(newPath); err == nil {
		return nil
	}
	oldPath := configPath("daily-announcement.json")
	data, err := os.ReadFile(oldPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return fmt.Errorf("解析旧配置失败 %s: %w", oldPath, err)
	}
	board := types.AnnouncementBoard{Enabled: true}
	if v, ok := raw["enabled"]; ok {
		_ = json.Unmarshal(v, &board.Enabled)
	}
	if v, ok := raw["title"]; ok {
		_ = json.Unmarshal(v, &board.Title)
	}
	if v, ok := raw["content"]; ok {
		_ = json.Unmarshal(v, &board.Content)
	}
	if err := writeJSONFile(newPath, board); err != nil {
		return fmt.Errorf("迁移公告板配置失败: %w", err)
	}
	_ = os.Rename(oldPath, oldPath+".bak")
	return nil
}

// ensureSecurityDisclaimerFile：v2.1.28 新增的配置项，旧部署的 config/ 目录里没有对应
// 文件、也没有旧格式可迁移；缺失时按默认开启写入，避免升级后因缺文件启动失败。
func ensureSecurityDisclaimerFile() error {
	path := configPath("security-disclaimer.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeJSONFile(path, types.SecurityDisclaimerConfig{Enabled: true})
}

// legacyFactionTaskGroups 仅用于旧库升级时的一次性回填：性别/阵营解耦前，只有这 4 个默认
// 阵营 ID，升级后它们的任务分组按当时的语义映射（femboy_faction→生理男、other_faction→默认兜底）。
// 管理员之后自建的阵营不在这张表里，一律归入 default，需要管理员自己在后台调整任务分组。
var legacyFactionTaskGroups = map[string]string{
	"male_faction":   "male",
	"femboy_faction": "male",
	"female_faction": "female",
	"other_faction":  "default",
}

// migrateGendersFile 把解耦前"性别嵌套在阵营里"的旧版 gender-factions.json 拆成新版
// genders.json（扁平性别预设）+ gender-factions.json（阵营不再带 genders，改带 taskGroup）。
// genders.json 已存在时视为已完成迁移（含全新安装：仓库自带的默认 genders.json），直接跳过。
func migrateGendersFile() error {
	gendersPath := configPath("genders.json")
	if _, err := os.Stat(gendersPath); err == nil {
		return nil
	}
	factionsPath := configPath("gender-factions.json")
	data, err := os.ReadFile(factionsPath)
	if err != nil {
		if os.IsNotExist(err) {
			// gender-factions.json 也不存在：交给 readSplitConfig 报出统一的缺文件错误。
			return nil
		}
		return err
	}
	type legacyGenderOption struct {
		ID    string `json:"id"`
		Label string `json:"label"`
	}
	type legacyGenderFaction struct {
		types.GenderColors
		ID      string               `json:"id"`
		Label   string               `json:"label"`
		Genders []legacyGenderOption `json:"genders"`
	}
	var legacy []legacyGenderFaction
	if err := json.Unmarshal(data, &legacy); err != nil {
		return fmt.Errorf("解析旧版性别阵营配置失败 %s: %w", factionsPath, err)
	}
	hasNested := false
	for _, f := range legacy {
		if len(f.Genders) > 0 {
			hasNested = true
			break
		}
	}
	if !hasNested {
		// 已经是新格式（没有嵌套 genders），只是 genders.json 意外丢失：写空文件，
		// 交给 ValidateConfig 报"至少需要一个性别选项"，而不是在这里静默造数据。
		return writeJSONFile(gendersPath, []types.GenderOption{})
	}
	var genders []types.GenderOption
	seen := map[string]struct{}{}
	factions := make([]types.GenderFaction, 0, len(legacy))
	for _, f := range legacy {
		for _, g := range f.Genders {
			if _, ok := seen[g.ID]; ok {
				continue
			}
			seen[g.ID] = struct{}{}
			genders = append(genders, types.GenderOption{ID: g.ID, Label: g.Label, FactionID: f.ID})
		}
		group := legacyFactionTaskGroups[f.ID]
		if group == "" {
			group = "default"
		}
		factions = append(factions, types.GenderFaction{
			GenderColors: f.GenderColors,
			ID:           f.ID,
			Label:        f.Label,
			TaskGroup:    group,
		})
	}
	if err := writeJSONFile(gendersPath, genders); err != nil {
		return fmt.Errorf("迁移性别预设配置失败: %w", err)
	}
	if err := writeJSONFile(factionsPath, factions); err != nil {
		return fmt.Errorf("迁移性别阵营配置失败: %w", err)
	}
	return nil
}

func ensureSplitConfig() error {
	if _, err := os.Stat(configPath("site.json")); err == nil {
		return nil
	}
	// 从旧单体迁移
	mono := configPath("active.json")
	if _, err := os.Stat(mono); err != nil {
		mono = configPath("default.json")
	}
	if _, err := os.Stat(mono); err != nil {
		return fmt.Errorf("缺少配置目录文件（需要 config/site.json 等，或旧版 default.json/active.json）: %w", err)
	}
	var cfg types.AppConfig
	if err := readJSONFile(mono, &cfg); err != nil {
		return err
	}
	if err := writeSplitConfig(cfg); err != nil {
		return fmt.Errorf("迁移单体配置到拆分文件失败: %w", err)
	}
	// 旧文件改名为 .bak，避免与拆分文件混淆
	for _, name := range []string{"active.json", "default.json"} {
		p := configPath(name)
		if _, err := os.Stat(p); err == nil {
			_ = os.Rename(p, p+".bak")
		}
	}
	return nil
}

func readSplitConfig() (types.AppConfig, error) {
	var cfg types.AppConfig
	type part struct {
		file string
		dest any
	}
	parts := []part{
		{"site.json", &cfg.Site},
		{"announcement-board.json", &cfg.AnnouncementBoard},
		{"security-disclaimer.json", &cfg.SecurityDisclaimer},
		{"genders.json", &cfg.Genders},
		{"gender-factions.json", &cfg.GenderFactions},
		{"titles.json", &cfg.Titles},
		{"punishments.json", &cfg.Punishments},
		{"player-punishment-room-name-pool.json", &cfg.PlayerPunishmentRoomNamePool},
		{"room-tags.json", &cfg.RoomTags},
		{"room-info-tags.json", &cfg.RoomInfoTags},
		{"access-control.json", &cfg.AccessControl},
		{"name-war.json", &cfg.NameWar},
		{"giveaway.json", &cfg.Giveaway},
		{"extreme-mode.json", &cfg.ExtremeMode},
		{"ranked-score.json", &cfg.RankedScore},
		{"bots.json", &cfg.Bots},
		{"games.json", &cfg.Games},
		{"messages.json", &cfg.Messages},
	}
	for _, p := range parts {
		if err := readJSONFile(configPath(p.file), p.dest); err != nil {
			return types.AppConfig{}, err
		}
	}
	return cfg, nil
}

func writeSplitConfig(cfg types.AppConfig) error {
	// 指针池字段：nil 时写空结构，避免 null 文件
	pool := cfg.PlayerPunishmentRoomNamePool
	if pool == nil {
		pool = &types.RoomNamePool{}
	}
	parts := []struct {
		file  string
		value any
	}{
		{"site.json", cfg.Site},
		{"announcement-board.json", cfg.AnnouncementBoard},
		{"security-disclaimer.json", cfg.SecurityDisclaimer},
		{"genders.json", cfg.Genders},
		{"gender-factions.json", cfg.GenderFactions},
		{"titles.json", cfg.Titles},
		{"punishments.json", cfg.Punishments},
		{"player-punishment-room-name-pool.json", pool},
		{"room-tags.json", cfg.RoomTags},
		{"room-info-tags.json", cfg.RoomInfoTags},
		{"access-control.json", cfg.AccessControl},
		{"name-war.json", cfg.NameWar},
		{"giveaway.json", cfg.Giveaway},
		{"extreme-mode.json", cfg.ExtremeMode},
		{"ranked-score.json", cfg.RankedScore},
		{"bots.json", cfg.Bots},
		{"games.json", cfg.Games},
		{"messages.json", cfg.Messages},
	}
	for _, p := range parts {
		if err := writeJSONFile(configPath(p.file), p.value); err != nil {
			return err
		}
	}
	return nil
}

// fixAnnouncementBoardEnabled：JSON 缺省 bool 在 Go 为 false；缺 enabled 字段时视为 true。
// path 可为 announcement-board.json 或旧单体整文件（历史字段名 dailyAnnouncement）。
func fixAnnouncementBoardEnabled(path string, cfg types.AppConfig) types.AppConfig {
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg
	}
	// 拆分文件：整文件即 announcementBoard
	var ab map[string]json.RawMessage
	if err := json.Unmarshal(data, &ab); err != nil {
		return cfg
	}
	if _, hasLegacy := ab["dailyAnnouncement"]; hasLegacy {
		// 旧单体：再挖一层
		var nested map[string]json.RawMessage
		if err := json.Unmarshal(ab["dailyAnnouncement"], &nested); err != nil {
			return cfg
		}
		if _, has := nested["enabled"]; !has {
			cfg.AnnouncementBoard.Enabled = true
		}
		return cfg
	}
	if _, has := ab["enabled"]; !has {
		// 若像 site 一样不是公告文件，有 title/content 才认定
		if _, ok := ab["title"]; ok {
			cfg.AnnouncementBoard.Enabled = true
		}
	}
	return cfg
}

// SaveConfig 校验后按功能文件原地写回 config/*.json。
func SaveConfig(next types.AppConfig) (types.AppConfig, error) {
	valid, err := ValidateConfig(next)
	if err != nil {
		return types.AppConfig{}, err
	}
	if err := writeSplitConfig(valid); err != nil {
		return types.AppConfig{}, err
	}
	return LoadConfig()
}

// ResetConfig 从磁盘重新加载配置（已无 default/active 双轨；不覆盖用户文件）。
func ResetConfig() (types.AppConfig, error) {
	return LoadConfig()
}

// ExportConfigText 导出合并后的完整配置 JSON（便于备份/迁移）。
func ExportConfigText() (string, error) {
	cfg, err := LoadConfig()
	if err != nil {
		return "", fmt.Errorf("导出配置失败: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return "", fmt.Errorf("导出配置失败: %w", err)
	}
	return string(data) + "\n", nil
}

// EnsureConfigPermissions 将 config/*.json 设为仅属主可读写（0600）、bin/server 设为可执行（部署后可选调用）。
func EnsureConfigPermissions() {
	dir := configDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		_ = os.Chmod(filepath.Join(dir, e.Name()), configFileMode)
	}
	serverBin := filepath.Join(rootDir, "bin", "server")
	if st, err := os.Stat(serverBin); err == nil && !st.IsDir() {
		_ = os.Chmod(serverBin, 0o755)
	}
}
