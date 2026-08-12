// Package config 从 config/json/ 下按功能拆分的 JSON 文件原地读写配置。
// 不在代码中维护第二套默认文案；文件缺失或校验失败直接返回 error。
//
// 文件一览（相对 config/json/）：
//
//	site.json, announcement-board.json, security-disclaimer.json, genders.json, gender-factions.json,
//	titles.json, punishments.json, player-punishment-room-name-pool.json, room-tags.json,
//	room-info-tags.json, title-tag-styles.json, access-control.json, name-war.json, giveaway.json,
//	pet-bond.json, extreme-mode.json, ranked-score.json, games.json, messages.json
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
	"title-tag-styles.json",
	"access-control.json",
	"name-war.json",
	"giveaway.json",
	"pet-bond.json",
	"extreme-mode.json",
	"ranked-score.json",
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
	cfg := filepath.Join(abs, "config", "json")
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

// SetRootDirForTest 仅测试用：临时改 root，调用方须在 Cleanup 里还原。
func SetRootDirForTest(dir string) {
	rootDir = dir
}

func configDir() string {
	return filepath.Join(rootDir, "config", "json")
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

// NormalizePunishmentTasks 清洗任务池列表：trim 文案/ID，任务难度夹紧到 1-99，
// 标签/阵营去重去空（不在此处校验标签 ID 是否存在——那是调用方的事）。
// -1 是"不参与随机抽取，仅供系列任务按 ID 引用"的显式标记，唯一合法取值即 -1，
// 其余负数一律归一化为 -1（不保留具体负值，避免被误解为有额外含义）；
// 0 及正数才是真实难度，夹紧到 1-99。
// 导出给 server 包的 punishmentTasksSave 复用。
func NormalizePunishmentTasks(tasks []types.PunishmentTaskConfig) []types.PunishmentTaskConfig {
	out := make([]types.PunishmentTaskConfig, 0, len(tasks))
	for i, task := range tasks {
		id := strings.TrimSpace(task.ID)
		if id == "" {
			id = fmt.Sprintf("task%d", i+1)
		}
		order := task.Order
		if order < 0 {
			order = -1
		} else {
			order = clampNumber(float64(order), 1, 99)
		}
		out = append(out, types.PunishmentTaskConfig{
			ID:                id,
			Name:              strings.TrimSpace(task.Name),
			Text:              strings.TrimSpace(task.Text),
			TagIDs:            dedupStrings(cleanLines(task.TagIDs)),
			FactionIDs:        dedupStrings(cleanLines(task.FactionIDs)),
			Order:             order,
			BackgroundImages:  cleanLines(task.BackgroundImages),
			BackgroundOpacity: clampOpacity(task.BackgroundOpacity),
		})
	}
	return out
}

// NormalizePunishmentSeriesTasks 清洗系列任务列表：trim、房名/背景规范化，
// 每步 TaskIDs 去重去空（不校验任务 ID 是否存在——允许悬空引用）。
// 导出给 server 包的 punishmentSeriesSave 复用。
func NormalizePunishmentSeriesTasks(series []types.PunishmentSeriesTaskConfig) []types.PunishmentSeriesTaskConfig {
	out := make([]types.PunishmentSeriesTaskConfig, 0, len(series))
	for _, s := range series {
		s.ID = strings.TrimSpace(s.ID)
		s.Name = strings.TrimSpace(s.Name)
		s.RoomBackgroundImages = cleanLines(s.RoomBackgroundImages)
		s.RoomNamePool = normalizeRoomNamePool(s.RoomNamePool)
		steps := make([]types.PunishmentSeriesStep, 0, len(s.Steps))
		for _, step := range s.Steps {
			steps = append(steps, types.PunishmentSeriesStep{
				TaskIDs: dedupStrings(cleanLines(step.TaskIDs)),
			})
		}
		s.Steps = steps
		out = append(out, s)
	}
	return out
}

func normalizePunishmentTags(tags []types.PunishmentTagConfig) []types.PunishmentTagConfig {
	out := make([]types.PunishmentTagConfig, 0, len(tags))
	for _, tag := range tags {
		tag.ID = strings.TrimSpace(tag.ID)
		tag.Name = strings.TrimSpace(tag.Name)
		tag.RoomBackgroundImages = cleanLines(tag.RoomBackgroundImages)
		tag.RoomNamePool = normalizeRoomNamePool(tag.RoomNamePool)
		out = append(out, tag)
	}
	return out
}

func normalizePunishmentRandomSettings(rs types.PunishmentRandomSettings) types.PunishmentRandomSettings {
	if rs.OrderStep < 0 {
		rs.OrderStep = 0
	}
	if rs.MaxDifficultyOvershoot < 0 {
		rs.MaxDifficultyOvershoot = 0
	}
	return rs
}

// punishmentsFileData 是 punishments.json 磁盘形态（仅标签 + 全局难度参数）。
// 任务池/系列任务已迁到 SQLite；旧文件里的 tasks/seriesTasks 字段由标准
// encoding/json 忽略（unknown field 默认丢弃），并由 server 一次性导入。
type punishmentsFileData struct {
	Tags                   []types.PunishmentTagConfig `json:"tags"`
	OrderStep              float64                     `json:"orderStep"`
	MaxDifficultyOvershoot float64                     `json:"maxDifficultyOvershoot"`
}

// punishmentsLegacyDiskData 仅用于磁盘迁移路径：旧 punishments.json 顶层数组拍平、
// 以及 server 一次性导入前从磁盘读出 tasks/seriesTasks。
// seriesTasks 用宽松 JSON 承接旧 subtasks/variants 结构，不绑新领域类型。
type punishmentsLegacyDiskData struct {
	Tags                   []types.PunishmentTagConfig  `json:"tags"`
	Tasks                  []types.PunishmentTaskConfig `json:"tasks"`
	SeriesTasks            json.RawMessage              `json:"seriesTasks"`
	OrderStep              float64                      `json:"orderStep"`
	MaxDifficultyOvershoot float64                      `json:"maxDifficultyOvershoot"`
}

func punishmentsFileFromConfig(cfg types.AppConfig) punishmentsFileData {
	return punishmentsFileData{
		Tags:                   cfg.PunishmentTags,
		OrderStep:              cfg.PunishmentRandomSettings.OrderStep,
		MaxDifficultyOvershoot: cfg.PunishmentRandomSettings.MaxDifficultyOvershoot,
	}
}

// punishmentsFileWriteData 是实际写盘用的形态：在 punishmentsFileData 之外，原样带上
// 磁盘上尚未被一次性迁移消费掉的旧 tasks/seriesTasks 字段（若有），避免任何一次
// config:save/config:reset（包括与惩罚任务毫无关系的字段改动）把这两个字段连带整份
// 文件一起冲掉——迁移是否已跑完全看 SQLite 里 punishment_tasks 表是否有行，不看
// 磁盘文件；写盘时无条件保留能读到的旧字段，多写不多余（迁移完成后就是纯死数据，
// 但比"抢在迁移之前被写没"安全）。
type punishmentsFileWriteData struct {
	Tags                   []types.PunishmentTagConfig `json:"tags"`
	OrderStep              float64                     `json:"orderStep"`
	MaxDifficultyOvershoot float64                     `json:"maxDifficultyOvershoot"`
	Tasks                  json.RawMessage             `json:"tasks,omitempty"`
	SeriesTasks            json.RawMessage             `json:"seriesTasks,omitempty"`
}

// preserveLegacyPunishmentPoolJSON 从磁盘现有 punishments.json 原样读出 tasks/seriesTasks
// 两个字段（未知/缺失/解析失败时返回 nil，不阻断保存）。
func preserveLegacyPunishmentPoolJSON(path string) (tasks, seriesTasks json.RawMessage) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil
	}
	var probe struct {
		Tasks       json.RawMessage `json:"tasks"`
		SeriesTasks json.RawMessage `json:"seriesTasks"`
	}
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, nil
	}
	return probe.Tasks, probe.SeriesTasks
}

func applyPunishmentsFile(cfg *types.AppConfig, file punishmentsFileData) {
	cfg.PunishmentTags = file.Tags
	cfg.PunishmentRandomSettings = types.PunishmentRandomSettings{
		OrderStep:              file.OrderStep,
		MaxDifficultyOvershoot: file.MaxDifficultyOvershoot,
	}
	// 清空旧字段，避免下发/写回时混入。
	cfg.Punishments = nil
	// 系列摘要由 server.publicConfig 现算，不从磁盘读。
	cfg.PunishmentSeriesSummaries = nil
}

// ReadLegacyPunishmentPoolFromDisk 从磁盘 punishments.json 原始读取已废弃的
// tasks/seriesTasks 字段，供 server 一次性导入 SQLite。
// seriesTasks 保持 RawMessage，因旧结构是 subtasks/variants，新结构是 steps/taskIds，
// 解析留给 server 的迁移逻辑。
func ReadLegacyPunishmentPoolFromDisk() (tasks []types.PunishmentTaskConfig, seriesRaw json.RawMessage, err error) {
	path := configPath("punishments.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil, nil
		}
		return nil, nil, err
	}
	var file punishmentsLegacyDiskData
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, nil, fmt.Errorf("解析惩罚配置失败 %s: %w", path, err)
	}
	return file.Tasks, file.SeriesTasks, nil
}

// dedupStrings 按出现顺序去重（用于任务的阵营标签多选，避免后台重复勾选被存两遍）。
func dedupStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
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

	punishmentTags := normalizePunishmentTags(input.PunishmentTags)
	punishmentRandom := normalizePunishmentRandomSettings(input.PunishmentRandomSettings)

	adminPass := strings.TrimSpace(input.Site.AdminPassword)
	if env := strings.TrimSpace(os.Getenv("ADMIN_PASSWORD")); env != "" {
		adminPass = env
	}

	ab := input.AnnouncementBoard
	ab.Title = sliceRunes(ab.Title, 32)
	ab.Content = sliceRunes(ab.Content, 800)

	games := make([]types.GameConfig, 0, len(input.Games))
	for _, g := range input.Games {
		// 与 types.GameID / 建房列表白名单保持一致；漏写会导致 config.games 静默丢掉该玩法。
		switch g.ID {
		case "rps", "othello", "tictactoe", "liarsdice", "gomoku", "jungle":
		default:
			continue
		}
		g.Name = sliceRunes(g.Name, 18)
		g.Description = sliceRunes(g.Description, 120)
		games = append(games, g)
	}

	roomInfoTags := map[string]types.RoomInfoTagStyle{}
	for key, tag := range input.RoomInfoTags {
		tag.Label = sliceRunes(tag.Label, 16)
		tag.TextColor = strings.TrimSpace(tag.TextColor)
		tag.BackgroundColor = strings.TrimSpace(tag.BackgroundColor)
		tag.BorderColor = strings.TrimSpace(tag.BorderColor)
		roomInfoTags[key] = tag
	}

	titleTagStyles := map[string]types.RoomInfoTagStyle{}
	for key, tag := range input.TitleTagStyles {
		tag.Label = sliceRunes(tag.Label, 16)
		tag.TextColor = strings.TrimSpace(tag.TextColor)
		tag.BackgroundColor = strings.TrimSpace(tag.BackgroundColor)
		tag.BorderColor = strings.TrimSpace(tag.BorderColor)
		titleTagStyles[key] = tag
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
	out.PunishmentTags = punishmentTags
	out.PunishmentRandomSettings = punishmentRandom
	out.PunishmentSeriesSummaries = nil // 不从配置加载；由 server.publicConfig 现算
	out.Punishments = nil
	out.RoomTags = cleanLines(input.RoomTags)
	out.Games = games
	out.RoomInfoTags = roomInfoTags
	out.TitleTagStyles = titleTagStyles
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
	out.PetBond.PanelTitle = sliceRunes(input.PetBond.PanelTitle, 24)
	if out.PetBond.PanelTitle == "" {
		out.PetBond.PanelTitle = "宠物乐园"
	}
	if out.PetBond.MaxPetsPerMaster < 1 {
		out.PetBond.MaxPetsPerMaster = 3
	}
	if out.PetBond.MaxMastersPerPet < 1 {
		out.PetBond.MaxMastersPerPet = 3
	}
	if out.PetBond.MaxTitleLength < 1 {
		out.PetBond.MaxTitleLength = 12
	}
	if out.PetBond.MaxTitleLength > 24 {
		out.PetBond.MaxTitleLength = 24
	}
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
	for _, g := range input.Genders {
		if g.Label == "" {
			return input, fmt.Errorf("性别显示文字不能为空")
		}
		// 显示文字允许重复：同一文案（如 "SUB"）可以同时挂在多个阵营下，各自是独立的
		// GenderOption（不同 ID + 不同 FactionID）。唯一性由上面的性别 ID 保证，
		// 前端 gendersForFaction 按 factionId 过滤下拉池，同一阵营内不会出现重复文案的选项。
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
	if len(input.PunishmentTags) == 0 {
		return input, fmt.Errorf("至少需要一个惩罚标签")
	}
	tagIDs := make([]string, len(input.PunishmentTags))
	for i, tag := range input.PunishmentTags {
		tagIDs[i] = tag.ID
	}
	if err := assertUnique(tagIDs, "惩罚标签 ID"); err != nil {
		return input, err
	}
	for _, tag := range input.PunishmentTags {
		if tag.Name == "" {
			return input, fmt.Errorf("惩罚标签 %s 的名称不能为空", tag.ID)
		}
		if tag.RoomBackgroundImages == nil {
			return input, fmt.Errorf("%s 的房间背景图库格式不正确", tag.Name)
		}
		if tag.RoomNamePool == nil || len(tag.RoomNamePool.Subjects) == 0 || len(tag.RoomNamePool.RoomWords) == 0 {
			return input, fmt.Errorf("%s 的随机房名至少需要名词/动词和房间词", tag.Name)
		}
	}
	// 任务池 / 系列任务已迁到 SQLite，不再随 AppConfig 校验；见 server 的
	// punishmentTasksSave / punishmentSeriesSave。
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
	for key, tag := range input.TitleTagStyles {
		if strings.TrimSpace(tag.Label) == "" {
			return input, fmt.Errorf("称号标签 %s 的名字不能为空", key)
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
	if input.NameWar.RenameMinPoints < 1 {
		return input, fmt.Errorf("名字争夺战改名最低分至少为 1")
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
	if input.Giveaway.PetLikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("对宠物点赞每小时次数上限至少为 1")
	}
	if input.Giveaway.PetLikeVoteValue <= 0 || input.Giveaway.PetLikeVoteValue > 100 {
		return input, fmt.Errorf("对宠物点赞降低值必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.PetDislikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("对宠物倒赞每小时次数上限至少为 1")
	}
	if input.Giveaway.PetDislikeVoteValue <= 0 || input.Giveaway.PetDislikeVoteValue > 100 {
		return input, fmt.Errorf("对宠物倒赞增加值必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.MasterLikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("对主人点赞每小时次数上限至少为 1")
	}
	if input.Giveaway.MasterLikeVoteValue <= 0 || input.Giveaway.MasterLikeVoteValue > 100 {
		return input, fmt.Errorf("对主人点赞降低值必须在 0（不含）到 100 之间")
	}
	if input.Giveaway.MasterDislikeVoteLimitPerHour < 1 {
		return input, fmt.Errorf("对主人倒赞每小时次数上限至少为 1")
	}
	if input.Giveaway.MasterDislikeVoteValue <= 0 || input.Giveaway.MasterDislikeVoteValue > 100 {
		return input, fmt.Errorf("对主人倒赞增加值必须在 0（不含）到 100 之间")
	}
	if strings.TrimSpace(input.PetBond.PanelTitle) == "" {
		return input, fmt.Errorf("宠物乐园面板标题不能为空")
	}
	if input.PetBond.MaxPetsPerMaster < 1 || input.PetBond.MaxPetsPerMaster > 20 {
		return input, fmt.Errorf("每名主人最多宠物数须在 1～20")
	}
	if input.PetBond.MaxMastersPerPet < 1 || input.PetBond.MaxMastersPerPet > 20 {
		return input, fmt.Errorf("每名宠物最多主人数须在 1～20")
	}
	if input.PetBond.MaxTitleLength < 1 || input.PetBond.MaxTitleLength > 24 {
		return input, fmt.Errorf("宠物称号长度上限须在 1～24")
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
	if err := ensureTitleTagStylesFile(); err != nil {
		return types.AppConfig{}, err
	}
	if err := migratePunishmentsFile(); err != nil {
		return types.AppConfig{}, err
	}
	cfg, err := readSplitConfig()
	if err != nil {
		return types.AppConfig{}, err
	}
	cfg = fixAnnouncementBoardEnabled(configPath("announcement-board.json"), cfg)
	cfg = fixGiveawayVoteLimits(cfg)
	valid, err := ValidateConfig(cfg)
	if err != nil {
		return types.AppConfig{}, fmt.Errorf("配置校验失败: %w", err)
	}
	return valid, nil
}

// punishmentTaskMigrationRecord 是新旧两种任务格式的联合读取结构。旧版任务把文案放在
// variants[group]，新版任务则把文案放在 text 并用 factionIds 指定适用阵营；这里故意用
// 原始 JSON 结构承接旧字段，避免先反序列化到新版类型后把旧文案静默丢掉。
type punishmentTaskMigrationRecord struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	Text              string            `json:"text"`
	FactionIDs        []string          `json:"factionIds"`
	Order             int               `json:"order"`
	Variants          map[string]string `json:"variants"`
	BackgroundImages  []string          `json:"backgroundImages"`
	BackgroundOpacity float64           `json:"backgroundOpacity"`
}

type punishmentMigrationRecord struct {
	ID                     string                          `json:"id"`
	Name                   string                          `json:"name"`
	Description            string                          `json:"description"`
	Variants               map[string]string               `json:"variants"`
	Tasks                  []punishmentTaskMigrationRecord `json:"tasks"`
	CardImageURL           string                          `json:"cardImageUrl"`
	CardImageOpacity       float64                         `json:"cardImageOpacity"`
	RoomBackgroundImages   []string                        `json:"roomBackgroundImages"`
	RoomNamePool           *types.RoomNamePool             `json:"roomNamePool"`
	OrderStep              float64                         `json:"orderStep"`
	MaxDifficultyOvershoot float64                         `json:"maxDifficultyOvershoot"`
}

func hasLegacyPunishments(records []punishmentMigrationRecord) bool {
	for _, punishment := range records {
		if strings.TrimSpace(punishment.Description) != "" || len(punishment.Variants) > 0 {
			return true
		}
		for _, task := range punishment.Tasks {
			if len(task.Variants) > 0 {
				return true
			}
		}
	}
	return false
}

func convertPunishmentRecords(records []punishmentMigrationRecord, factionIDsByGroup map[string][]string) []types.PunishmentConfig {
	converted := make([]types.PunishmentConfig, 0, len(records))
	for _, punishment := range records {
		out := types.PunishmentConfig{
			ID:                     strings.TrimSpace(punishment.ID),
			Name:                   strings.TrimSpace(punishment.Name),
			CardImageURL:           strings.TrimSpace(punishment.CardImageURL),
			CardImageOpacity:       punishment.CardImageOpacity,
			RoomBackgroundImages:   cleanLines(punishment.RoomBackgroundImages),
			RoomNamePool:           punishment.RoomNamePool,
			OrderStep:              punishment.OrderStep,
			MaxDifficultyOvershoot: punishment.MaxDifficultyOvershoot,
			Tasks:                  make([]types.PunishmentTaskConfig, 0),
		}
		for taskIndex, task := range punishment.Tasks {
			order := task.Order
			if order <= 0 {
				order = 50
			}
			if len(task.Variants) == 0 && strings.TrimSpace(task.Text) != "" {
				out.Tasks = append(out.Tasks, types.PunishmentTaskConfig{
					ID:                strings.TrimSpace(task.ID),
					Name:              strings.TrimSpace(task.Name),
					Text:              strings.TrimSpace(task.Text),
					FactionIDs:        cleanLines(task.FactionIDs),
					Order:             order,
					BackgroundImages:  cleanLines(task.BackgroundImages),
					BackgroundOpacity: task.BackgroundOpacity,
				})
				continue
			}

			baseID := strings.TrimSpace(task.ID)
			if baseID == "" {
				baseID = fmt.Sprintf("task%d", taskIndex+1)
			}
			baseName := strings.TrimSpace(task.Name)
			if baseName == "" {
				baseName = fmt.Sprintf("任务 %d", taskIndex+1)
			}
			created := 0
			for _, group := range []string{"default", "male", "female"} {
				text := strings.TrimSpace(task.Variants[group])
				if text == "" {
					text = strings.TrimSpace(punishment.Variants[group])
				}
				if text == "" {
					text = strings.TrimSpace(punishment.Description)
				}
				if text == "" {
					continue
				}
				factionIDs := append([]string(nil), factionIDsByGroup[group]...)
				if group != "default" && len(factionIDs) == 0 {
					// 当前配置没有该任务分组时，旧文案没有可对应的玩家阵营。
					continue
				}
				id := baseID + "_" + group
				if created == 0 && len(task.Variants) == 0 && len(punishment.Variants) == 0 {
					id = baseID
				}
				out.Tasks = append(out.Tasks, types.PunishmentTaskConfig{
					ID:                id,
					Name:              baseName,
					Text:              text,
					FactionIDs:        factionIDs,
					Order:             order,
					BackgroundImages:  cleanLines(task.BackgroundImages),
					BackgroundOpacity: task.BackgroundOpacity,
				})
				created++
			}
			if created == 0 {
				out.Tasks = append(out.Tasks, types.PunishmentTaskConfig{
					ID: baseID, Name: baseName, Text: "请完成本局惩罚。", FactionIDs: []string{}, Order: order,
					BackgroundImages: cleanLines(task.BackgroundImages), BackgroundOpacity: task.BackgroundOpacity,
				})
			}
		}
		if len(out.Tasks) == 0 {
			out.Tasks = []types.PunishmentTaskConfig{{
				ID: "task1", Name: "默认任务", Text: "请完成本局惩罚。", FactionIDs: []string{}, Order: 50,
				BackgroundImages: []string{}, BackgroundOpacity: 0.22,
			}}
		}
		converted = append(converted, out)
	}
	return converted
}

// upgradeLegacyPunishmentsForSave：管理端若仍提交旧「任务类型」数组（Punishments 非空且
// 新标签字段为空），现场拍平成 tags（任务池不再进 AppConfig，由磁盘迁移 + SQLite 导入承接）。
func upgradeLegacyPunishmentsForSave(input types.AppConfig) (types.AppConfig, error) {
	if len(input.PunishmentTags) > 0 {
		input.Punishments = nil
		return input, nil
	}
	if len(input.Punishments) == 0 {
		return input, nil
	}
	data, err := json.Marshal(input.Punishments)
	if err != nil {
		return input, fmt.Errorf("读取旧惩罚配置失败: %w", err)
	}
	var records []punishmentMigrationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return input, fmt.Errorf("读取旧惩罚配置失败: %w", err)
	}
	if hasLegacyPunishments(records) {
		factionIDsByGroup := map[string][]string{}
		for _, faction := range input.GenderFactions {
			group := strings.TrimSpace(faction.TaskGroup)
			if group == "" {
				group = "default"
			}
			if id := strings.TrimSpace(faction.ID); id != "" {
				factionIDsByGroup[group] = append(factionIDsByGroup[group], id)
			}
		}
		converted := convertPunishmentRecords(records, factionIDsByGroup)
		raw, err := json.Marshal(converted)
		if err != nil {
			return input, err
		}
		if err := json.Unmarshal(raw, &records); err != nil {
			return input, err
		}
	}
	file := flattenPunishmentTypesToFile(records)
	// 只把标签/难度写回 AppConfig；tasks 仍写在磁盘（migratePunishmentsFile 路径），
	// 由 server 一次性导入 SQLite。这里 Save 路径只保留标签。
	applyPunishmentsFile(&input, punishmentsFileData{
		Tags:                   file.Tags,
		OrderStep:              file.OrderStep,
		MaxDifficultyOvershoot: file.MaxDifficultyOvershoot,
	})
	return input, nil
}

// migratePunishmentsFile 将旧版 punishments.json 迁到「标签 + 拍平任务 + 系列」结构：
//  1. 若仍是顶层数组（旧任务类型容器），先展开 variants→factionIds，再拍平成 tags/tasks；
//  2. 若已是对象但缺少 tags 字段，同样触发拍平；
//  3. 已是新结构（顶层含 tags）则跳过。
// 迁移后写回磁盘；旧文件保留为 punishments.json.bak（若尚无备份）。
func migratePunishmentsFile() error {
	path := configPath("punishments.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return fmt.Errorf("惩罚配置为空 %s", path)
	}

	// 已是新结构：顶层对象且含 tags。
	if trimmed[0] == '{' {
		var probe map[string]json.RawMessage
		if err := json.Unmarshal(data, &probe); err != nil {
			return fmt.Errorf("解析惩罚配置失败 %s: %w", path, err)
		}
		if _, ok := probe["tags"]; ok {
			return nil
		}
		// 对象但无 tags：视为异常/半迁移，尝试按旧数组字段继续不适用，直接报错提示。
		return fmt.Errorf("惩罚配置 %s 缺少 tags 字段，请检查或从备份恢复", path)
	}

	// 顶层数组：旧「任务类型」列表。
	var records []punishmentMigrationRecord
	if err := json.Unmarshal(data, &records); err != nil {
		return fmt.Errorf("解析惩罚配置失败 %s: %w", path, err)
	}

	// 先做 variants → factionIds 展开（若有），再拍平成 tags/tasks。
	if hasLegacyPunishments(records) {
		factionIDsByGroup, err := punishmentFactionIDsByTaskGroup()
		if err != nil {
			return err
		}
		convertedTypes := convertPunishmentRecords(records, factionIDsByGroup)
		// 把转换结果再编码成 migration record 形态，便于统一拍平。
		raw, err := json.Marshal(convertedTypes)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(raw, &records); err != nil {
			return err
		}
	}

	file := flattenPunishmentTypesToFile(records)
	backupPath := path + ".bak"
	if _, err := os.Stat(backupPath); err == nil {
		// 备份已存在时直接覆盖写新结构（幂等），不再二次备份。
		if err := writeJSONFile(path, file); err != nil {
			return fmt.Errorf("写入迁移后的惩罚配置失败: %w", err)
		}
		return nil
	}
	if err := os.Rename(path, backupPath); err != nil {
		return fmt.Errorf("备份旧惩罚配置失败: %w", err)
	}
	if err := writeJSONFile(path, file); err != nil {
		_ = os.Rename(backupPath, path)
		return fmt.Errorf("写入迁移后的惩罚配置失败: %w", err)
	}
	return nil
}

// flattenPunishmentTypesToFile 把旧「任务类型」列表拍平成标签 + 任务（磁盘迁移用）。
// 全局难度取第一个类型上的非零值，否则默认 2/5；系列任务留空。
// 返回 legacy 形态以便 migratePunishmentsFile 把 tasks 写进磁盘，供 server 导入 SQLite。
func flattenPunishmentTypesToFile(records []punishmentMigrationRecord) punishmentsLegacyDiskData {
	file := punishmentsLegacyDiskData{
		Tags:                   make([]types.PunishmentTagConfig, 0, len(records)),
		Tasks:                  make([]types.PunishmentTaskConfig, 0),
		SeriesTasks:            json.RawMessage("[]"),
		OrderStep:              2,
		MaxDifficultyOvershoot: 5,
	}
	stepSet, overSet := false, false
	for _, rec := range records {
		tagID := strings.TrimSpace(rec.ID)
		if tagID == "" {
			continue
		}
		file.Tags = append(file.Tags, types.PunishmentTagConfig{
			ID:                   tagID,
			Name:                 strings.TrimSpace(rec.Name),
			RoomNamePool:         rec.RoomNamePool,
			RoomBackgroundImages: cleanLines(rec.RoomBackgroundImages),
		})
		if !stepSet && rec.OrderStep > 0 {
			file.OrderStep = rec.OrderStep
			stepSet = true
		}
		if !overSet && rec.MaxDifficultyOvershoot > 0 {
			file.MaxDifficultyOvershoot = rec.MaxDifficultyOvershoot
			overSet = true
		}
		for ti, task := range rec.Tasks {
			order := task.Order
			if order <= 0 {
				order = 50
			}
			id := strings.TrimSpace(task.ID)
			if id == "" {
				id = fmt.Sprintf("%s_task%d", tagID, ti+1)
			} else if !strings.HasPrefix(id, tagID+"_") {
				// 避免不同旧类型下同名 task_default 撞 ID。
				id = tagID + "_" + id
			}
			file.Tasks = append(file.Tasks, types.PunishmentTaskConfig{
				ID:                id,
				Name:              strings.TrimSpace(task.Name),
				Text:              strings.TrimSpace(task.Text),
				TagIDs:            []string{tagID},
				FactionIDs:        cleanLines(task.FactionIDs),
				Order:             order,
				BackgroundImages:  cleanLines(task.BackgroundImages),
				BackgroundOpacity: task.BackgroundOpacity,
			})
		}
	}
	return file
}

func punishmentFactionIDsByTaskGroup() (map[string][]string, error) {
	result := map[string][]string{}
	path := configPath("gender-factions.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return nil, fmt.Errorf("读取阵营配置失败 %s: %w", path, err)
	}
	var factions []types.GenderFaction
	if err := json.Unmarshal(data, &factions); err != nil {
		return nil, fmt.Errorf("解析阵营配置失败 %s: %w", path, err)
	}
	for _, faction := range factions {
		group := strings.TrimSpace(faction.TaskGroup)
		if group == "" {
			group = "default"
		}
		id := strings.TrimSpace(faction.ID)
		if id == "" {
			continue
		}
		result[group] = append(result[group], id)
	}
	return result, nil
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

// ensureTitleTagStylesFile：本版本新增的称号标签按来源着色配置，老部署目录里没有对应旧文件可迁移，
// 直接补写默认四色（系统默认/自定义/主人赋予/管理员赋予），否则老实例升级后会因缺文件启动失败。
func ensureTitleTagStylesFile() error {
	path := configPath("title-tag-styles.json")
	if _, err := os.Stat(path); err == nil {
		return nil
	}
	return writeJSONFile(path, map[string]types.RoomInfoTagStyle{
		"system": {Label: "系统默认", GenderColors: types.GenderColors{TextColor: "#1f6f45", BackgroundColor: "#e3f7ea", BorderColor: "#96d9b1"}},
		"self":   {Label: "自定义", GenderColors: types.GenderColors{TextColor: "#225c8d", BackgroundColor: "#e1f2ff", BorderColor: "#8fcaf0"}},
		"master": {Label: "主人赋予", GenderColors: types.GenderColors{TextColor: "#5c3b82", BackgroundColor: "#f0e7ff", BorderColor: "#c7a8f5"}},
		"admin":  {Label: "管理员赋予", GenderColors: types.GenderColors{TextColor: "#7c5900", BackgroundColor: "#fff1c4", BorderColor: "#ffd875"}},
	})
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
		return fmt.Errorf("缺少配置目录文件（需要 config/json/site.json 等，或旧版 default.json/active.json）: %w", err)
	}
	var cfg types.AppConfig
	if err := readJSONFile(mono, &cfg); err != nil {
		return err
	}
	// 旧单体可能只有 punishments 数组，没有 tags；现场拍平后再拆分写盘。
	// 任务池已不进 AppConfig，但拍平后的 tasks 仍需写入 punishments.json，
	// 供 server 一次性导入 SQLite。
	var monothlicPool *punishmentsLegacyDiskData
	if len(cfg.PunishmentTags) == 0 && len(cfg.Punishments) > 0 {
		data, err := json.Marshal(cfg.Punishments)
		if err != nil {
			return fmt.Errorf("迁移单体惩罚配置失败: %w", err)
		}
		var records []punishmentMigrationRecord
		if err := json.Unmarshal(data, &records); err != nil {
			return fmt.Errorf("迁移单体惩罚配置失败: %w", err)
		}
		if hasLegacyPunishments(records) {
			factionIDsByGroup := map[string][]string{}
			for _, faction := range cfg.GenderFactions {
				group := strings.TrimSpace(faction.TaskGroup)
				if group == "" {
					group = "default"
				}
				if id := strings.TrimSpace(faction.ID); id != "" {
					factionIDsByGroup[group] = append(factionIDsByGroup[group], id)
				}
			}
			converted := convertPunishmentRecords(records, factionIDsByGroup)
			raw, err := json.Marshal(converted)
			if err != nil {
				return fmt.Errorf("迁移单体惩罚配置失败: %w", err)
			}
			if err := json.Unmarshal(raw, &records); err != nil {
				return fmt.Errorf("迁移单体惩罚配置失败: %w", err)
			}
		}
		file := flattenPunishmentTypesToFile(records)
		monothlicPool = &file
		applyPunishmentsFile(&cfg, punishmentsFileData{
			Tags:                   file.Tags,
			OrderStep:              file.OrderStep,
			MaxDifficultyOvershoot: file.MaxDifficultyOvershoot,
		})
	}
	if err := writeSplitConfig(cfg); err != nil {
		return fmt.Errorf("迁移单体配置到拆分文件失败: %w", err)
	}
	// 单体拍平产生的 tasks 写回 punishments.json（含 tasks 字段），供 SQLite 导入。
	if monothlicPool != nil {
		if err := writeJSONFile(configPath("punishments.json"), *monothlicPool); err != nil {
			return fmt.Errorf("写入迁移后的惩罚任务池失败: %w", err)
		}
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
	var punishmentsFile punishmentsFileData
	parts := []part{
		{"site.json", &cfg.Site},
		{"announcement-board.json", &cfg.AnnouncementBoard},
		{"security-disclaimer.json", &cfg.SecurityDisclaimer},
		{"genders.json", &cfg.Genders},
		{"gender-factions.json", &cfg.GenderFactions},
		{"titles.json", &cfg.Titles},
		{"punishments.json", &punishmentsFile},
		{"player-punishment-room-name-pool.json", &cfg.PlayerPunishmentRoomNamePool},
		{"room-tags.json", &cfg.RoomTags},
		{"room-info-tags.json", &cfg.RoomInfoTags},
		{"title-tag-styles.json", &cfg.TitleTagStyles},
		{"access-control.json", &cfg.AccessControl},
		{"name-war.json", &cfg.NameWar},
		{"giveaway.json", &cfg.Giveaway},
		{"pet-bond.json", &cfg.PetBond},
		{"extreme-mode.json", &cfg.ExtremeMode},
		{"ranked-score.json", &cfg.RankedScore},
		{"games.json", &cfg.Games},
		{"messages.json", &cfg.Messages},
	}
	for _, p := range parts {
		if err := readJSONFile(configPath(p.file), p.dest); err != nil {
			return types.AppConfig{}, err
		}
	}
	applyPunishmentsFile(&cfg, punishmentsFile)
	return cfg, nil
}

func writeSplitConfig(cfg types.AppConfig) error {
	// 指针池字段：nil 时写空结构，避免 null 文件
	pool := cfg.PlayerPunishmentRoomNamePool
	if pool == nil {
		pool = &types.RoomNamePool{}
	}
	punishmentsFile := punishmentsFileFromConfig(cfg)
	if punishmentsFile.Tags == nil {
		punishmentsFile.Tags = []types.PunishmentTagConfig{}
	}
	legacyTasks, legacySeries := preserveLegacyPunishmentPoolJSON(configPath("punishments.json"))
	punishmentsOut := punishmentsFileWriteData{
		Tags:                   punishmentsFile.Tags,
		OrderStep:              punishmentsFile.OrderStep,
		MaxDifficultyOvershoot: punishmentsFile.MaxDifficultyOvershoot,
		Tasks:                  legacyTasks,
		SeriesTasks:            legacySeries,
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
		{"punishments.json", punishmentsOut},
		{"player-punishment-room-name-pool.json", pool},
		{"room-tags.json", cfg.RoomTags},
		{"room-info-tags.json", cfg.RoomInfoTags},
		{"title-tag-styles.json", cfg.TitleTagStyles},
		{"access-control.json", cfg.AccessControl},
		{"name-war.json", cfg.NameWar},
		{"giveaway.json", cfg.Giveaway},
		{"pet-bond.json", cfg.PetBond},
		{"extreme-mode.json", cfg.ExtremeMode},
		{"ranked-score.json", cfg.RankedScore},
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

// fixGiveawayVoteLimits：本版本把自救板投票额度从"全体共用一档"拆成"普通/宠物/主人"
// 三档（次数上限 + 升/降值百分比共 8 个字段），老部署的 giveaway.json 里没有这些新字段，
// json.Unmarshal 会把它们留成 0——Validate 要求这些字段 >0（次数上限 >=1），缺字段就会导致
// 老实例升级后直接启动失败（与 migrateAnnouncementBoardFile 等其它自愈函数同一诱因）。
// 0 在这些字段上本就不是合法值，缺失或显式填 0 都按新增宠物/主人档回退到与普通档相同的数值。
func fixGiveawayVoteLimits(cfg types.AppConfig) types.AppConfig {
	if cfg.Giveaway.PetLikeVoteLimitPerHour == 0 {
		cfg.Giveaway.PetLikeVoteLimitPerHour = cfg.Giveaway.LikeVoteLimitPerHour
	}
	if cfg.Giveaway.PetLikeVoteValue == 0 {
		cfg.Giveaway.PetLikeVoteValue = cfg.Giveaway.LikeVoteValue
	}
	if cfg.Giveaway.PetDislikeVoteLimitPerHour == 0 {
		cfg.Giveaway.PetDislikeVoteLimitPerHour = cfg.Giveaway.DislikeVoteLimitPerHour
	}
	if cfg.Giveaway.PetDislikeVoteValue == 0 {
		cfg.Giveaway.PetDislikeVoteValue = cfg.Giveaway.DislikeVoteValue
	}
	if cfg.Giveaway.MasterLikeVoteLimitPerHour == 0 {
		cfg.Giveaway.MasterLikeVoteLimitPerHour = cfg.Giveaway.LikeVoteLimitPerHour
	}
	if cfg.Giveaway.MasterLikeVoteValue == 0 {
		cfg.Giveaway.MasterLikeVoteValue = cfg.Giveaway.LikeVoteValue
	}
	if cfg.Giveaway.MasterDislikeVoteLimitPerHour == 0 {
		cfg.Giveaway.MasterDislikeVoteLimitPerHour = cfg.Giveaway.DislikeVoteLimitPerHour
	}
	if cfg.Giveaway.MasterDislikeVoteValue == 0 {
		cfg.Giveaway.MasterDislikeVoteValue = cfg.Giveaway.DislikeVoteValue
	}
	return cfg
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
	// 旧版管理端提交的配置没有新增的宠物/主人投票字段；与 LoadConfig 保持同一
	// 兼容策略，缺省字段回退到普通档，否则升级后保存任意其它配置都会被 Validate 拒绝。
	next = fixGiveawayVoteLimits(next)
	var err error
	if next, err = upgradeLegacyPunishmentsForSave(next); err != nil {
		return types.AppConfig{}, err
	}
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

// EnsureConfigPermissions 将 config/json/*.json 设为仅属主可读写（0600）、bin/server 设为可执行（部署后可选调用）。
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
