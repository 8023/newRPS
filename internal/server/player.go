package server

import (
	"crypto/subtle"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// defaultNameWarPenaltyThreshold 与 config/name-war.json 的 penaltyThreshold 默认值一致，
// 供 nameWarPenaltyThreshold 在配置异常（非负）时兜底；ValidateConfig 已在启动/保存时拒绝
// 非负值，正常运行时这条分支理论上走不到。
const defaultNameWarPenaltyThreshold = -4999

// defaultNameWarRenameMinPoints 与 config/name-war.json 的 renameMinPoints 默认值一致，
// 供 nameWarRenameMinPoints 在配置异常（小于 1）时兜底；ValidateConfig 已在启动/保存时拒绝
// 小于 1 的值，正常运行时这条分支理论上走不到。
const defaultNameWarRenameMinPoints = 500

func (s *Server) publicConfig() types.AppConfig {
	cfg := s.cfg
	cfg.Site.AdminPassword = ""
	cfg.PunishmentSeriesSummaries = s.buildPunishmentSeriesSummaries()
	return sanitizePublicConfig(cfg)
}

// buildPunishmentSeriesSummaries 从系列缓存生成建房用公开目录（无任务文案/taskIds）。
// 阵营覆盖不全（seriesIsUsable==false）的系列不出现在目录里，等同不可选。
func (s *Server) buildPunishmentSeriesSummaries() []types.PunishmentSeriesSummary {
	out := make([]types.PunishmentSeriesSummary, 0, len(s.punishmentSeriesCache))
	taskByID := make(map[string]*types.PunishmentTaskConfig, len(s.punishmentTasksCache))
	for i := range s.punishmentTasksCache {
		task := &s.punishmentTasksCache[i]
		taskByID[task.ID] = task
	}
	for _, series := range s.punishmentSeriesCache {
		if !s.seriesIsUsableWithTaskMap(series, taskByID) {
			continue
		}
		out = append(out, types.PunishmentSeriesSummary{
			ID:                   series.ID,
			Name:                 series.Name,
			RoomNamePool:         series.RoomNamePool,
			RoomBackgroundImages: series.RoomBackgroundImages,
			StepCount:            len(series.Steps),
		})
	}
	return out
}

// reloadPunishmentCaches 从 SQLite 刷新内存缓存（启动 / admin save 后调用）。
// 调用方须持有 s.mu，或在尚无并发访问的启动路径调用。
func (s *Server) reloadPunishmentCaches() {
	if s.punishmentStore == nil {
		s.punishmentTasksCache = []types.PunishmentTaskConfig{}
		s.punishmentSeriesCache = []types.PunishmentSeriesTaskConfig{}
		return
	}
	if tasks, err := s.punishmentStore.listTasks(); err != nil {
		s.errorLog("punishment_tasks_load_failed", err.Error())
		s.punishmentTasksCache = []types.PunishmentTaskConfig{}
	} else {
		s.punishmentTasksCache = tasks
	}
	if series, err := s.punishmentStore.listSeries(); err != nil {
		s.errorLog("punishment_series_load_failed", err.Error())
		s.punishmentSeriesCache = []types.PunishmentSeriesTaskConfig{}
	} else {
		s.punishmentSeriesCache = series
	}
}

// savePunishmentTasks 清洗/校验后全量替换任务池，并刷新缓存。调用方须持有 s.mu。
func (s *Server) savePunishmentTasks(tasks []types.PunishmentTaskConfig) error {
	tasks = config.NormalizePunishmentTasks(tasks)
	if len(tasks) == 0 {
		return fmt.Errorf("至少需要一条任务")
	}
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	if err := assertUniqueIDs(ids, "任务 ID"); err != nil {
		return err
	}
	tagSet := map[string]struct{}{}
	for _, tag := range s.cfg.PunishmentTags {
		tagSet[tag.ID] = struct{}{}
	}
	factionSet := map[string]struct{}{}
	for _, f := range s.cfg.GenderFactions {
		factionSet[f.ID] = struct{}{}
	}
	for _, t := range tasks {
		label := t.Name
		if label == "" {
			label = t.ID
		}
		if strings.TrimSpace(t.Text) == "" {
			return fmt.Errorf("随机任务 %s 的文案不能为空", label)
		}
		for _, tid := range t.TagIDs {
			// series:<id> 内部标签（迁移产物）不在标签管理里，放行；其余须存在。
			if strings.HasPrefix(tid, "series:") {
				continue
			}
			if _, ok := tagSet[tid]; !ok {
				return fmt.Errorf("随机任务 %s 引用了不存在的标签 %s", label, tid)
			}
		}
		if t.Order != -1 && (t.Order < 1 || t.Order > 99) {
			return fmt.Errorf("随机任务 %s 的任务难度必须是 -1（不参与随机抽取）或在 1 到 99 之间", label)
		}
		for _, fid := range t.FactionIDs {
			if _, ok := factionSet[fid]; !ok {
				return fmt.Errorf("随机任务 %s 勾选了不存在的阵营 %s", label, fid)
			}
		}
		if t.BackgroundImages == nil {
			return fmt.Errorf("随机任务 %s 的背景图库格式不正确", label)
		}
		if t.BackgroundOpacity < 0 || t.BackgroundOpacity > 1 {
			return fmt.Errorf("随机任务 %s 的背景透明率必须在 0 到 1 之间", label)
		}
	}
	if s.punishmentStore == nil {
		return fmt.Errorf("任务池存储不可用")
	}
	if err := s.punishmentStore.replaceTasks(tasks); err != nil {
		s.errorLog("punishment_tasks_save_failed", err.Error())
		return fmt.Errorf("保存任务池失败")
	}
	s.punishmentTasksCache = tasks
	return nil
}

// savePunishmentSeries 清洗/校验后全量替换系列任务，并刷新缓存。调用方须持有 s.mu。
// 不校验 TaskIDs 是否指向存在任务（允许悬空引用，运行时跳过）。
func (s *Server) savePunishmentSeries(series []types.PunishmentSeriesTaskConfig) error {
	series = config.NormalizePunishmentSeriesTasks(series)
	ids := make([]string, len(series))
	for i, ser := range series {
		ids[i] = ser.ID
	}
	if err := assertUniqueIDs(ids, "系列任务 ID"); err != nil {
		return err
	}
	for _, ser := range series {
		if ser.Name == "" {
			return fmt.Errorf("系列任务 %s 的名称不能为空", ser.ID)
		}
		if ser.RoomBackgroundImages == nil {
			return fmt.Errorf("%s 的房间背景图库格式不正确", ser.Name)
		}
		if ser.RoomNamePool == nil || len(ser.RoomNamePool.Subjects) == 0 || len(ser.RoomNamePool.RoomWords) == 0 {
			return fmt.Errorf("%s 的随机房名至少需要名词/动词和房间词", ser.Name)
		}
		if len(ser.Steps) == 0 {
			return fmt.Errorf("%s 至少需要一个子任务", ser.Name)
		}
		for si, step := range ser.Steps {
			if len(step.TaskIDs) == 0 {
				return fmt.Errorf("%s 第 %d 步至少需要一个任务", ser.Name, si+1)
			}
		}
	}
	if s.punishmentStore == nil {
		return fmt.Errorf("系列任务存储不可用")
	}
	if err := s.punishmentStore.replaceSeries(series); err != nil {
		s.errorLog("punishment_series_save_failed", err.Error())
		return fmt.Errorf("保存系列任务失败")
	}
	s.punishmentSeriesCache = series
	return nil
}

// cascadeRemovedTagsFromTasks 在惩罚配置保存后：从任务池 tagIds 里摘除已删除的标签 ID。
// 不阻断保存；空标签任务合法保留，可继续供系列按 ID 引用，但不会进入随机候选池。
func (s *Server) cascadeRemovedTagsFromTasks(oldTags, newTags []types.PunishmentTagConfig) {
	if s.punishmentStore == nil || len(s.punishmentTasksCache) == 0 {
		return
	}
	newSet := map[string]struct{}{}
	for _, t := range newTags {
		newSet[t.ID] = struct{}{}
	}
	removed := map[string]struct{}{}
	for _, t := range oldTags {
		if _, ok := newSet[t.ID]; !ok {
			removed[t.ID] = struct{}{}
		}
	}
	if len(removed) == 0 {
		return
	}
	changed := false
	next := make([]types.PunishmentTaskConfig, len(s.punishmentTasksCache))
	copy(next, s.punishmentTasksCache)
	for i := range next {
		// 重新切片避免共享底层数组副作用
		orig := append([]string(nil), next[i].TagIDs...)
		filtered := orig[:0]
		for _, id := range orig {
			if _, gone := removed[id]; gone {
				changed = true
				continue
			}
			filtered = append(filtered, id)
		}
		next[i].TagIDs = filtered
	}
	if !changed {
		return
	}
	if err := s.punishmentStore.replaceTasks(next); err != nil {
		s.errorLog("punishment_tag_cascade_failed", err.Error())
		return
	}
	s.punishmentTasksCache = next
}

func assertUniqueIDs(ids []string, label string) error {
	seen := map[string]struct{}{}
	for _, id := range ids {
		if id == "" {
			return fmt.Errorf("%s 不能为空", label)
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("%s 重复: %s", label, id)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// adminLoginFailure* 常量：管理员口令校验失败的锁定阈值。只统计失败次数（正确口令的
// 重复调用——如管理员在同一会话里反复保存配置——不计入配额，不会误伤正常使用）。
// 按 IP 单独锁定，另加一个不分 IP 的全局桶防止分布式撞库；两者共用同一套时间窗/冷却时长。
const (
	adminLoginFailurePerIPLimit  = 8
	adminLoginFailureGlobalLimit = 40
	adminLoginFailureWindowMs    = 10 * 60_000
	adminLoginFailureCooldownMs  = 10 * 60_000
)

// adminLoginLockedOut 只读检查（不消耗配额），调用方须持有 s.mu。
func (s *Server) adminLoginLockedOut(ip string) bool {
	now := nowMs()
	if b := s.rateBuckets["admin_login_fail:"+ip]; b != nil && b.CooldownUntil > now {
		return true
	}
	if b := s.rateBuckets["admin_login_fail:*"]; b != nil && b.CooldownUntil > now {
		return true
	}
	return false
}

// recordAdminLoginFailure 只在口令比对失败时调用一次，同时喂给按 IP 和全局两个桶。
func (s *Server) recordAdminLoginFailure(ip string) {
	s.checkRateLimit("admin_login_fail:"+ip, RateLimitOptions{
		Limit: adminLoginFailurePerIPLimit, WindowMs: adminLoginFailureWindowMs, CooldownMs: adminLoginFailureCooldownMs,
	})
	s.checkRateLimit("admin_login_fail:*", RateLimitOptions{
		Limit: adminLoginFailureGlobalLimit, WindowMs: adminLoginFailureWindowMs, CooldownMs: adminLoginFailureCooldownMs,
	})
}

// adminPasswordMatches 校验管理员口令；ip 用于失败锁定与审计日志（调用方须持有 s.mu）。
func (s *Server) adminPasswordMatches(password, ip string) bool {
	expected := s.cfg.Site.AdminPassword
	if expected == "" {
		return false
	}
	if s.adminLoginLockedOut(ip) {
		s.securityLog("admin_login_locked", map[string]any{"ip": ip})
		return false
	}
	// 恒定时间比较，避免按字符早退泄露口令长度/前缀。
	if subtle.ConstantTimeCompare([]byte(password), []byte(expected)) == 1 {
		return true
	}
	s.recordAdminLoginFailure(ip)
	s.securityLog("admin_login_failed", map[string]any{"ip": ip})
	return false
}

// rankedScorePercent 把真实排位分映射到相对展示上下限的百分比（可超出 ±100）。
// 正分：points/Max*100；负分：points/|Min|*100；0 → 0。
func (s *Server) rankedScorePercent(points int) float64 {
	if points == 0 {
		return 0
	}
	if points > 0 {
		max := s.cfg.RankedScore.Max
		if max <= 0 {
			return 0
		}
		return float64(points) / float64(max) * 100
	}
	min := s.cfg.RankedScore.Min
	if min >= 0 {
		return 0
	}
	return float64(points) / float64(-min) * 100
}

func (s *Server) titleSegmentFor(points int) *types.TitleSegment {
	if len(s.cfg.Titles) == 0 {
		return nil
	}
	percent := s.rankedScorePercent(points)
	var nearest *types.TitleSegment
	nearestDist := math.MaxFloat64
	for i := range s.cfg.Titles {
		item := &s.cfg.Titles[i]
		if percent >= item.MinPercent && percent <= item.MaxPercent {
			return item
		}
		// 落在这一段之外（含分段之间的空隙、或超出首尾边界）：记录到该段最近边界的距离，
		// 最终夹到距离最近的分段，而不是无条件夹到最高档——分段表若配置成中间留空隙，
		// 落在空隙里的百分比应该就近取邻近段，不能因为循环顺序只跟最高档比较。
		dist := item.MinPercent - percent
		if percent > item.MaxPercent {
			dist = percent - item.MaxPercent
		}
		if dist < nearestDist {
			nearestDist = dist
			nearest = item
		}
	}
	return nearest
}

func (s *Server) titleNamesForSegment(segment *types.TitleSegment, factionID string) []string {
	if segment == nil {
		return []string{"初心拳手"}
	}
	if factionID != "" && segment.FactionNames != nil {
		if names := segment.FactionNames[s.taskGroupForFaction(factionID)]; len(names) > 0 {
			return names
		}
	}
	if len(segment.Names) > 0 {
		return segment.Names
	}
	return []string{"初心拳手"}
}

func (s *Server) randomTitleFromSegment(segment *types.TitleSegment, factionID string) string {
	names := s.titleNamesForSegment(segment, factionID)
	if len(names) == 0 {
		return "初心拳手"
	}
	return names[rand.Intn(len(names))]
}

func (s *Server) syncTitleForRankSegment(player *PlayerState, force bool) {
	if player.Stats.TitleCustom {
		return
	}
	segment := s.titleSegmentFor(player.Stats.RankedPoints)
	if segment == nil {
		return
	}
	if !force && player.Stats.TitleSegmentID == "" && player.Stats.Title != "" {
		player.Stats.TitleSegmentID = segment.ID
		return
	}
	if force || player.Stats.TitleSegmentID != segment.ID || player.Stats.Title == "" {
		player.Stats.Title = s.randomTitleFromSegment(segment, player.FactionID)
	}
	player.Stats.TitleSegmentID = segment.ID
}

type genderInfoResult struct {
	GenderID      string
	GenderLabel   string
	FactionID     string
	FactionLabel  string
	FactionColors types.GenderColors
}

// findGenderPreset 按 id 查预设性别（与阵营无关的扁平列表）；查不到返回 nil。
func (s *Server) findGenderPreset(genderID string) *types.GenderOption {
	for i := range s.cfg.Genders {
		if s.cfg.Genders[i].ID == genderID {
			return &s.cfg.Genders[i]
		}
	}
	return nil
}

// findFaction 按 id 查阵营；查不到 / id 为空时退回第一个已配置阵营（配置异常兜底）。
func (s *Server) findFaction(factionID string) types.GenderFaction {
	fallback := types.GenderFaction{
		ID: "unknown_faction", Label: "未知阵营", TaskGroup: "default",
		GenderColors: types.GenderColors{TextColor: "#4d5c6f", BackgroundColor: "#eef3f8", BorderColor: "#c9d6e4"},
	}
	if len(s.cfg.GenderFactions) > 0 {
		fallback = s.cfg.GenderFactions[0]
	}
	if factionID != "" {
		for i := range s.cfg.GenderFactions {
			if s.cfg.GenderFactions[i].ID == factionID {
				return s.cfg.GenderFactions[i]
			}
		}
	}
	return fallback
}

// taskGroupForFaction 把阵营 id 解析成任务分组（male/female/default），供称号按分组取文案
// （系统任务已改为按 FactionIDs 直接筛选，不再依赖任务分组，见 punishment.go 的 candidateTasksForFaction）。
func (s *Server) taskGroupForFaction(factionID string) string {
	group := s.findFaction(factionID).TaskGroup
	if group == "" {
		return "default"
	}
	return group
}

// validGenderSubmission 校验一次"明确提交"的性别选择：genderID 必须命中某个预设，阵营
// 由该预设的 FactionID 查表决定，不再单独提交/校验。只用在个人资料保存、后台管理员保存这类
// "改动既有玩家资料"的主动提交入口；不用于 player:join（无论是首次建号还是常规重连同步）——
// resolveGender 自身的默认兜底覆盖了 genderID 为空/未知的合法场景。返回的第二个值在校验失败
// 时是给用户看的具体原因，成功时为空字符串。
func (s *Server) validGenderSubmission(genderID string) (bool, string) {
	if s.findGenderPreset(genderID) == nil {
		return false, "所选性别不存在"
	}
	return true, ""
}

// resolveGender 查表法：genderID 命中预设则用预设文案，阵营按预设的 FactionID 查表得出；
// 未命中（含空串，比如老账号缓存的自定义性别标记）时退回第一个已配置性别及其阵营。
func (s *Server) resolveGender(genderID string) genderInfoResult {
	gid, glabel, factionID := "", "", ""
	if preset := s.findGenderPreset(genderID); preset != nil {
		gid, glabel, factionID = preset.ID, preset.Label, preset.FactionID
	} else if len(s.cfg.Genders) > 0 {
		gid, glabel, factionID = s.cfg.Genders[0].ID, s.cfg.Genders[0].Label, s.cfg.Genders[0].FactionID
	}
	faction := s.findFaction(factionID)
	return genderInfoResult{
		GenderID: gid, GenderLabel: glabel,
		FactionID: faction.ID, FactionLabel: faction.Label,
		FactionColors: types.GenderColors{
			TextColor: faction.TextColor, BackgroundColor: faction.BackgroundColor, BorderColor: faction.BorderColor,
		},
	}
}

func (s *Server) nameWarCode() string {
	chars := "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 4)
	for i := 0; i < 4; i++ {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

func (s *Server) generateNameWarPenaltyName() string {
	prefix := strings.TrimSpace(s.cfg.NameWar.PenaltyPrefix)
	if prefix == "" {
		prefix = "失名者"
	}
	return prefix + "-" + s.nameWarCode()
}

func formatDisplayName(player *PlayerState) string {
	if ptrBool(player.NameWarPunished) && player.NameWarPenaltyName != "" {
		return player.NameWarPenaltyName
	}
	return player.GenderLabel + " - " + player.Stats.Title + " - " + player.Name
}

func playerShortName(player *PlayerState) string {
	if ptrBool(player.NameWarPunished) && player.NameWarPenaltyName != "" {
		return player.NameWarPenaltyName
	}
	return player.Name
}

func (s *Server) refreshGiveawayBoard(player *PlayerState, now int64) {
	if player.GiveawayBoardExpiresAt == nil || *player.GiveawayBoardExpiresAt > now {
		return
	}
	player.GiveawayBoardText = ""
	player.GiveawayBoardSubmittedAt = nil
	player.GiveawayBoardExpiresAt = nil
	player.GiveawayBoardLikes = intPtr(0)
	player.GiveawayBoardDislikes = intPtr(0)
	player.GiveawayBoardLikesThisHour = intPtr(0)
	player.GiveawayBoardLikeWindowStartedAt = nil
}

// addGiveawayValue 按 delta 调整白给值（正负皆可，最终钳制到 0-100 且保留 2 位小数）。
// 白给值归零后自动关闭白给玩法（赢一局的惩罚、被点踩等都会走这里，不用每处调用方各自判断）。
func (s *Server) addGiveawayValue(player *PlayerState, delta float64) {
	v := ptrFloat(player.GiveawayValue) + delta
	clamped := clampGiveawayValue(v)
	player.GiveawayValue = floatPtr(clamped)
	if clamped <= 0 && ptrBool(player.GiveawayEnabled) {
		player.GiveawayEnabled = boolPtr(false)
	}
	s.markPlayerDirty(player)
	s.requestPersist("lazy")
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
}

// applyGiveawayWinPenalty 白给模式已开启的玩家每赢一局（含断线判负）扣减白给值；
// 未开启白给的玩家不受影响，值本就钳制在 0。
func (s *Server) applyGiveawayWinPenalty(winner *PlayerState) {
	if winner == nil || !ptrBool(winner.GiveawayEnabled) {
		return
	}
	s.addGiveawayValue(winner, -s.cfg.Giveaway.WinPenaltyValue)
}

// giveawayVoteWindowMs 是自救板投票额度的滚动窗口长度（每对 actor→target 各自独立起算）。
const giveawayVoteWindowMs = 3_600_000

// giveawayVoteQuota 是某个 actor 对某一个 target 的投票额度状态（PlayerState.GiveawayVoteQuotas
// 的 value）。
type giveawayVoteQuota struct {
	WindowStartedAt int64
	Likes           int
	Dislikes        int
}

// giveawayVoteRulesFor 按 actor 与 target 的认主认宠关系返回这一对的点赞/倒赞每小时次数
// 上限、以及每次投票对白给值的降低/增加百分比——只认直系关系（不含宠物的宠物等二级以上
// 关系），命中则用宠物档/主人档，否则普通档；三档的次数上限和升降值都各自独立配置。
func (s *Server) giveawayVoteRulesFor(actor, target *PlayerState) (likeLimit, dislikeLimit int, likeValue, dislikeValue float64) {
	cfg := s.cfg.Giveaway
	switch {
	case s.getBond(actor.ID, target.ID) != nil: // target 是 actor 的直系宠物
		return cfg.PetLikeVoteLimitPerHour, cfg.PetDislikeVoteLimitPerHour, cfg.PetLikeVoteValue, cfg.PetDislikeVoteValue
	case s.getBond(target.ID, actor.ID) != nil: // target 是 actor 的直系主人
		return cfg.MasterLikeVoteLimitPerHour, cfg.MasterDislikeVoteLimitPerHour, cfg.MasterLikeVoteValue, cfg.MasterDislikeVoteValue
	default:
		return cfg.LikeVoteLimitPerHour, cfg.DislikeVoteLimitPerHour, cfg.LikeVoteValue, cfg.DislikeVoteValue
	}
}

// giveawayVoteQuotaFor 取出（必要时懒重置）actor 对 targetID 这一对的额度状态。窗口过期
// 后清零重新起算，不影响 actor 对其它目标的额度。
func (s *Server) giveawayVoteQuotaFor(actor *PlayerState, targetID string, now int64) *giveawayVoteQuota {
	if actor.GiveawayVoteQuotas == nil {
		actor.GiveawayVoteQuotas = map[string]*giveawayVoteQuota{}
	}
	q := actor.GiveawayVoteQuotas[targetID]
	if q == nil || now-q.WindowStartedAt >= giveawayVoteWindowMs {
		q = &giveawayVoteQuota{WindowStartedAt: now}
		actor.GiveawayVoteQuotas[targetID] = q
	}
	return q
}

// giveawayVoteQuotaView 组装某一对 actor→target 的额度快照，供 RPC 回包给 actor 自己
// （giveaway:vote 的回执 / giveaway:voteQuotas 的查询结果）；未过期的窗口起始时间原样带上，
// 方便前端按本地时钟推算"N 分钟后刷新"，过期或从未投过票则视为满额度、窗口为 0。
func (s *Server) giveawayVoteQuotaView(actor, target *PlayerState, now int64) map[string]any {
	likeLimit, dislikeLimit, likeValue, dislikeValue := s.giveawayVoteRulesFor(actor, target)
	likesUsed, dislikesUsed, windowStartedAt := 0, 0, int64(0)
	if q := actor.GiveawayVoteQuotas[target.ID]; q != nil && now-q.WindowStartedAt < giveawayVoteWindowMs {
		likesUsed, dislikesUsed, windowStartedAt = q.Likes, q.Dislikes, q.WindowStartedAt
	}
	return map[string]any{
		"targetId":        target.ID,
		"likeLimit":       likeLimit,
		"likeValue":       likeValue,
		"likesUsed":       likesUsed,
		"dislikeLimit":    dislikeLimit,
		"dislikeValue":    dislikeValue,
		"dislikesUsed":    dislikesUsed,
		"windowStartedAt": windowStartedAt,
	}
}

// updateLegacyGiveawayVoteStats 维护旧客户端仍读取的 actor 聚合投票字段。新额度限制按
// actor→target 的 GiveawayVoteQuotas 执行，这些字段只作为兼容性展示，不参与新限流判断。
func (s *Server) updateLegacyGiveawayVoteStats(actor *PlayerState, vote string, now int64) {
	if actor == nil {
		return
	}
	if actor.GiveawayVoteWindowStartedAt == nil || now-*actor.GiveawayVoteWindowStartedAt >= giveawayVoteWindowMs {
		actor.GiveawayVoteWindowStartedAt = int64Ptr(now)
		actor.GiveawayVoteCount = intPtr(0)
		actor.GiveawayVoteLikesThisHour = intPtr(0)
		actor.GiveawayVoteDislikesThisHour = intPtr(0)
	}
	actor.GiveawayVoteCount = intPtr(ptrInt(actor.GiveawayVoteCount) + 1)
	if vote == "like" {
		actor.GiveawayVoteLikesThisHour = intPtr(ptrInt(actor.GiveawayVoteLikesThisHour) + 1)
	} else if vote == "dislike" {
		actor.GiveawayVoteDislikesThisHour = intPtr(ptrInt(actor.GiveawayVoteDislikesThisHour) + 1)
	}
}

func (s *Server) refreshNameWarState(player *PlayerState, now int64) bool {
	before := fmt.Sprintf("%s|%s|%v|%s|%v|%s|%s",
		player.Stats.Title, player.DisplayName, ptrBool(player.NameWarPunished),
		player.NameWarPenaltyName, player.NameWarRenameProtectedUntil,
		player.NameWarRenamedBy, player.NameWarRenamedByName)

	if !ptrBool(player.NameWarEnabled) {
		player.NameWarPunished = boolPtr(false)
		player.NameWarPenaltyName = ""
		player.NameWarRenameProtectedUntil = nil
		player.NameWarRenamedBy = ""
		player.NameWarRenamedByName = ""
		s.syncTitleForRankSegment(player, false)
	} else {
		protectedActive := player.NameWarRenameProtectedUntil != nil && *player.NameWarRenameProtectedUntil > now
		if protectedActive && player.NameWarPenaltyName != "" {
			player.NameWarPunished = boolPtr(true)
		} else if player.Stats.RankedPoints <= s.nameWarPenaltyThreshold() {
			player.NameWarPunished = boolPtr(true)
			if player.NameWarPenaltyName == "" {
				player.NameWarPenaltyName = s.generateNameWarPenaltyName()
			}
			if player.NameWarRenameProtectedUntil != nil && *player.NameWarRenameProtectedUntil <= now {
				player.NameWarRenameProtectedUntil = nil
			}
		} else {
			player.NameWarPunished = boolPtr(false)
			player.NameWarPenaltyName = ""
			player.NameWarRenameProtectedUntil = nil
			player.NameWarRenamedBy = ""
			player.NameWarRenamedByName = ""
			s.syncTitleForRankSegment(player, false)
		}
	}
	player.DisplayName = formatDisplayName(player)
	after := fmt.Sprintf("%s|%s|%v|%s|%v|%s|%s",
		player.Stats.Title, player.DisplayName, ptrBool(player.NameWarPunished),
		player.NameWarPenaltyName, player.NameWarRenameProtectedUntil,
		player.NameWarRenamedBy, player.NameWarRenamedByName)
	return before != after
}

func (s *Server) applyGender(player *PlayerState, genderID string) {
	oldFactionID := player.FactionID
	next := s.resolveGender(genderID)
	player.GenderID = next.GenderID
	player.GenderLabel = next.GenderLabel
	player.FactionID = next.FactionID
	player.FactionLabel = next.FactionLabel
	player.FactionColors = next.FactionColors
	if oldFactionID != "" && oldFactionID != next.FactionID && !ptrBool(player.NameWarPunished) {
		s.syncTitleForRankSegment(player, true)
	}
	player.DisplayName = formatDisplayName(player)
}

func (s *Server) publicPlayer(player *PlayerState) types.PublicPlayer {
	p := player.PublicPlayer
	p.SyncTotalsFromGameStats()
	// 真实分留给排序；展示字段按后台配置封顶（含历史最高/最低）。
	p.Stats.SortRankedPoints = player.Stats.RankedPoints
	p.Stats.SortHighestScore = player.Stats.HighestScore
	p.Stats.SortLowestScore = player.Stats.LowestScore
	p.Stats.RankedPoints = s.displayClampScore(player, player.Stats.RankedPoints)
	p.Stats.HighestScore = s.displayClampScore(player, player.Stats.HighestScore)
	p.Stats.LowestScore = s.displayClampScore(player, player.Stats.LowestScore)
	// 累计在线：存储值 + 当前会话（不写回存储，避免与断线累加重叠）。
	// 量化到整分钟：毫秒级实时递增会让每次大厅/玩家广播的树哈希都变，
	// 触发无意义的 DELTA/FULL 流量（见性能分析 #2）。
	p.Stats.TotalOnlineMs = quantizeOnlineMs(s.effectiveOnlineMs(player))
	// 展示称号优先级见 applyDisplayTitle：管理员自定义 > 宠物称号 > 玩家自设称号 > 积分档称号。
	s.applyDisplayTitle(player, &p)
	return p
}

// clientOnlineCreditedFrom 返回本连接尚未记入 TotalOnlineMs 的起点。
// onlineCreditedAt 优先（checkpoint 后会推进）；否则回退到 connectedAt。
func clientOnlineCreditedFrom(client *Client) int64 {
	if client == nil {
		return 0
	}
	if client.onlineCreditedAt > 0 {
		return client.onlineCreditedAt
	}
	return client.connectedAt
}

// onlineMsQuantizeStep 展示用在线时长量化粒度（1 分钟）。
const onlineMsQuantizeStep int64 = 60_000

// quantizeOnlineMs 把在线毫秒向下取整到 quantize 步长，避免广播文档每秒变脏。
func quantizeOnlineMs(ms int64) int64 {
	if ms <= 0 {
		return 0
	}
	return (ms / onlineMsQuantizeStep) * onlineMsQuantizeStep
}

// effectiveOnlineMs 返回用于展示/排行的累计在线毫秒：已落库（含 checkpoint）+ 尚未记账的本段。
func (s *Server) effectiveOnlineMs(player *PlayerState) int64 {
	if player == nil {
		return 0
	}
	total := player.Stats.TotalOnlineMs
	if !player.Connected || player.SocketID == "" {
		return total
	}
	c := s.clients[player.SocketID]
	from := clientOnlineCreditedFrom(c)
	if from <= 0 {
		return total
	}
	if d := nowMs() - from; d > 0 {
		total += d
	}
	return total
}

// accumulateClientOnlineMs 把一条已结束连接尚未记账的时长累加到绑定玩家（须在 s.mu 内调用）。
// 游客连接（未关联玩家）直接跳过。
// 不要求 player.SocketID 仍等于本连接：同 SID 顶替时玩家可能已指向新 socket，
// 但仍应把旧连接的时长记到该玩家上。
// 已 checkpoint 过的段落不会重复计入（从 onlineCreditedAt 起算）。
func (s *Server) accumulateClientOnlineMs(client *Client, disconnectedAt int64) {
	if client == nil || client.playerID == "" {
		return
	}
	from := clientOnlineCreditedFrom(client)
	if from <= 0 {
		return
	}
	player := s.players[client.playerID]
	if player == nil {
		return
	}
	dur := disconnectedAt - from
	if dur <= 0 {
		return
	}
	player.Stats.TotalOnlineMs += dur
	client.onlineCreditedAt = disconnectedAt
	s.markPlayerDirty(player)
	s.requestPersist("lazy")
}

// checkpointOnlineMs 把仍在线连接的已持续时长折叠进 TotalOnlineMs 并推进 onlineCreditedAt。
// 须在 s.mu 内调用。定期调用可让 kill -9 / 非优雅重启最多只丢一个 checkpoint 间隔。
// 返回是否有玩家时长被改动（调用方据此 markPersistDirty）。
func (s *Server) checkpointOnlineMs() bool {
	now := nowMs()
	touched := false
	for _, c := range s.clients {
		if c == nil || c.playerID == "" || c.connectionLogged {
			continue
		}
		from := clientOnlineCreditedFrom(c)
		if from <= 0 {
			continue
		}
		dur := now - from
		if dur <= 0 {
			continue
		}
		player := s.players[c.playerID]
		if player == nil {
			continue
		}
		player.Stats.TotalOnlineMs += dur
		c.onlineCreditedAt = now
		s.markPlayerDirty(player)
		touched = true
	}
	return touched
}

// displayClampScore 仅用于下发展示：真实存储永不改动；按 RankedScore 的 max/min
// （名争玩家用 nameWarMin）夹紧后的副本。排位分、历史最高、历史最低共用。
func (s *Server) displayClampScore(player *PlayerState, score int) int {
	min := s.cfg.RankedScore.Min
	if ptrBool(player.NameWarEnabled) {
		min = s.cfg.RankedScore.NameWarMin
	}
	return clamp(score, min, s.cfg.RankedScore.Max)
}

func (s *Server) refreshPlayerSnapshots(player *PlayerState) {
	pub := s.publicPlayer(player)
	for _, room := range s.rooms {
		for _, seat := range []types.SeatKey{types.SeatA, types.SeatB} {
			occ := room.Seats[seat]
			if occ == nil {
				continue
			}
			if occ.GetID() == player.ID {
				room.Seats[seat] = &HumanSeat{Player: pub}
			}
		}
	}
}

func (s *Server) refreshAllPlayersForConfig() {
	for _, player := range s.players {
		s.applyGender(player, player.GenderID)
		s.refreshNameWarState(player, nowMs())
		s.refreshPlayerSnapshots(player)
	}
}

// onlinePlayersFromDevice 统计同一 deviceKey（IP+指纹）下已连接玩家数。
func (s *Server) onlinePlayersFromDevice(deviceKey string, exceptPlayerID string) int {
	if deviceKey == "" {
		return 0
	}
	return s.onlinePlayersMatching(exceptPlayerID, func(p *PlayerState) bool { return p.DeviceKey == deviceKey })
}

// canCreateFromDevice 同 deviceKey 在 10 分钟内新建玩家次数上限。
func (s *Server) canCreateFromDevice(deviceKey string) bool {
	if deviceKey == "" {
		deviceKey = "unknown"
	}
	return s.canCreateFromKey(s.deviceCreateAttempts, deviceKey, s.cfg.AccessControl.MaxCreatesPer10Min)
}

// onlinePlayersFromIP 统计同一出口 IP（不看指纹）下已连接玩家数，是
// onlinePlayersFromDevice 的兜底——指纹由客户端上报、未做真实性校验，
// 攻击脚本每次随机化指纹即可让 deviceKey 各不相同，绕过按设备计算的限制。
func (s *Server) onlinePlayersFromIP(ipAddress string, exceptPlayerID string) int {
	if ipAddress == "" {
		return 0
	}
	return s.onlinePlayersMatching(exceptPlayerID, func(p *PlayerState) bool { return p.IPAddress == ipAddress })
}

// onlinePlayersMatching 是 onlinePlayersFromDevice/onlinePlayersFromIP 共用的计数逻辑：
// 统计满足 match 的已连接玩家数（exceptPlayerID 排除自身，用于"算上我自己以外还有几个"）。
func (s *Server) onlinePlayersMatching(exceptPlayerID string, match func(*PlayerState) bool) int {
	n := 0
	for _, player := range s.players {
		if player.Connected && player.ID != exceptPlayerID && match(player) {
			n++
		}
	}
	return n
}

// canCreateFromIP 同 canCreateFromDevice，但按纯 IP 维度计算，用作兜底。
func (s *Server) canCreateFromIP(ipAddress string) bool {
	if ipAddress == "" {
		ipAddress = "unknown-ip"
	}
	return s.canCreateFromKey(s.ipCreateAttempts, ipAddress, s.cfg.AccessControl.MaxCreatesPerIP)
}

// canCreateFromKey 是 canCreateFromDevice/canCreateFromIP 共用的滑动窗口计数器：
// key 在 10 分钟窗口内的新建次数达到 limit 后拒绝，否则记一次并放行。
func (s *Server) canCreateFromKey(attempts map[string][]int64, key string, limit int) bool {
	now := nowMs()
	windowMs := int64(10 * 60 * 1000)
	filtered := attempts[key][:0]
	for _, t := range attempts[key] {
		if now-t < windowMs {
			filtered = append(filtered, t)
		}
	}
	if len(filtered) >= limit {
		attempts[key] = filtered
		return false
	}
	attempts[key] = append(filtered, now)
	return true
}

func (s *Server) createPlayer(name, genderID, token string, identityPlayerID, identitySecret string) *PlayerState {
	session := s.verifySessionToken(token)
	persistent := identityPlayerID != "" && identitySecret != ""
	id := randomID()
	if persistent {
		id = generatePublicID()
	} else if session != nil {
		id = session.SID
	}
	gender := s.resolveGender(genderID)
	titleSegment := s.titleSegmentFor(0)
	title := s.randomTitleFromSegment(titleSegment, gender.FactionID)
	titleSegID := ""
	if titleSegment != nil {
		titleSegID = titleSegment.ID
	}
	now := nowMs()
	player := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID:                           id,
			Name:                         name,
			GenderID:                     gender.GenderID,
			GenderLabel:                  gender.GenderLabel,
			FactionID:                    gender.FactionID,
			FactionLabel:                 gender.FactionLabel,
			FactionColors:                gender.FactionColors,
			DisplayName:                  gender.GenderLabel + " - " + title + " - " + name,
			Connected:                    true,
			NameWarOriginalName:          name,
			GiveawayEnabled:              boolPtr(false),
			GiveawayValue:                floatPtr(0),
			GiveawayClicks:               intPtr(0),
			GiveawayBoardLikes:           intPtr(0),
			GiveawayBoardDislikes:        intPtr(0),
			GiveawayVoteLikesThisHour:    intPtr(0),
			GiveawayVoteDislikesThisHour: intPtr(0),
			RankMultiplierUnlocked:       boolPtr(false),
			ExtremeModeEnabled:           boolPtr(false),
			ExtremeWinStreak:             intPtr(0),
			ExtremeLastDecayHour:         int64Ptr(currentExtremeDecayHour(now)),
			ExtremeForceClosed:           boolPtr(false),
			Stats: types.PublicStats{
				RankedPoints:   0,
				Title:          title,
				TitleSegmentID: titleSegID,
			},
			GameStats: freshGameStats(),
		},
		Token:      token,
		Persistent: persistent,
		PlayerID:   identityPlayerID,
		CreatedAt:  now,
		LastSeenAt: now,
	}
	if token == "" {
		player.Token = randomID()
	}
	if persistent {
		player.PlayerSecrets = []string{identitySecret}
		player.ClaimKey = randomClaimKey()
	}
	if session != nil {
		player.CurrentSID = session.SID
	}
	s.players[player.ID] = player
	s.tokenToPlayer[player.Token] = player.ID
	if player.PlayerID != "" {
		s.playerIdToID[player.PlayerID] = player.ID
	}
	if session != nil {
		s.sidToPlayerID[session.SID] = player.ID
	}
	if persistent {
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
	}
	return player
}

// getPlayerByClient 按 client.playerID 直接索引 s.players（O(1)），再用
// player.SocketID == client.id 确认这条连接确实是该玩家当前活跃的连接——
// 语义等价于旧版对 s.players 的线性扫描（顶号/断线重连后旧连接的 playerID
// 仍指向同一个玩家对象，但 SocketID 已指向新连接，这里会正确返回 nil）。
func (s *Server) getPlayerByClient(client *Client) *PlayerState {
	if client == nil || client.playerID == "" {
		return nil
	}
	player := s.players[client.playerID]
	if player == nil || player.SocketID != client.id {
		return nil
	}
	return player
}

func (s *Server) clearDisconnectHold(player *PlayerState) {
	if player.graceTimer != nil {
		player.graceTimer.Stop()
		player.graceTimer = nil
	}
	if player.discTimer != nil {
		player.discTimer.Stop()
		player.discTimer = nil
	}
	player.graceGen++
	player.timerGen++
	player.DisconnectExpiresAt = nil
}

// recordRankedExtremes 更新历史最高/最低排位分记录；这两个字段永不设上下限、永不回退。
func recordRankedExtremes(player *PlayerState) {
	if player.Stats.RankedPoints > player.Stats.HighestScore {
		player.Stats.HighestScore = player.Stats.RankedPoints
	}
	if player.Stats.RankedPoints < player.Stats.LowestScore {
		player.Stats.LowestScore = player.Stats.RankedPoints
	}
}

// updateRankedPoints 按增量结算排位分。真实存储值不设上下限——RankedScore 配置的
// 上下限只在 publicPlayer()/displayRankedPoints() 下发展示时生效。
func (s *Server) updateRankedPoints(player *PlayerState, delta int) {
	player.Stats.RankedPoints += delta
	recordRankedExtremes(player)
	s.refreshNameWarState(player, nowMs())
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	if player.Persistent {
		s.markPlayerDirty(player)
		s.requestPersist("important")
	}
}

func (s *Server) setRankedPointsByAdmin(player *PlayerState, points int) {
	player.Stats.RankedPoints = int(math.Round(float64(points)))
	recordRankedExtremes(player)
	s.refreshNameWarState(player, nowMs())
	if player.Persistent {
		s.markPlayerDirty(player)
		s.requestPersist("important")
	}
}

// extremeSegmentID 与称号分段同一套百分比刻度（相对 RankedScore.Max/|Min|），
// 供极限模式的正分输分/负分赢分/整点扣分表按 pos1~4 / neg1~4 取系数。
func (s *Server) extremeSegmentID(points int) string {
	percent := s.rankedScorePercent(points)
	if percent >= 75 {
		return "pos4"
	}
	if percent >= 50 {
		return "pos3"
	}
	if percent >= 25 {
		return "pos2"
	}
	if percent > 0 {
		return "pos1"
	}
	if percent <= -75 {
		return "neg4"
	}
	if percent <= -50 {
		return "neg3"
	}
	if percent <= -25 {
		return "neg2"
	}
	if percent < 0 {
		return "neg1"
	}
	return "pos0"
}

func (s *Server) adjustedRankedDelta(player *PlayerState, delta float64) int {
	if player == nil || !ptrBool(player.ExtremeModeEnabled) || delta == 0 {
		return int(math.Round(delta))
	}
	segment := s.extremeSegmentID(player.Stats.RankedPoints)
	if delta < 0 && player.Stats.RankedPoints > 0 {
		rate := s.cfg.ExtremeMode.PositiveLossRates[segment]
		if rate == 0 && segment != "pos0" {
			// missing key -> 1
			if _, ok := s.cfg.ExtremeMode.PositiveLossRates[segment]; !ok {
				rate = 1
			}
		} else if _, ok := s.cfg.ExtremeMode.PositiveLossRates[segment]; !ok {
			rate = 1
		}
		return -int(math.Round(math.Abs(delta) * rate))
	}
	if delta > 0 && player.Stats.RankedPoints < 0 {
		rate, ok := s.cfg.ExtremeMode.NegativeWinRates[segment]
		if !ok {
			if segment == "neg4" {
				rate = 0.5
			} else {
				rate = 1
			}
		}
		return int(math.Round(delta * rate))
	}
	return int(math.Round(delta))
}

func (s *Server) applyRankedStake(winner, loser *PlayerState, stake int) (winnerDelta, loserDelta int) {
	winnerDelta = s.adjustedRankedDelta(winner, float64(stake))
	loserDelta = s.adjustedRankedDelta(loser, float64(-stake))
	if winner != nil {
		s.updateRankedPoints(winner, winnerDelta)
	}
	if loser != nil {
		s.updateRankedPoints(loser, loserDelta)
	}
	return
}

func (s *Server) applyRankedDrawPenaltyStake(playerA, playerB *PlayerState, stake int) (deltaA, deltaB int) {
	deltaA = s.adjustedRankedDelta(playerA, float64(-stake))
	deltaB = s.adjustedRankedDelta(playerB, float64(-stake))
	if playerA != nil {
		s.updateRankedPoints(playerA, deltaA)
	}
	if playerB != nil {
		s.updateRankedPoints(playerB, deltaB)
	}
	return
}

func rankMultiplierFor(settings types.RoomSettings) types.RankMultiplier {
	if !settings.EnableRanked || !settings.EnableRankMultiplier {
		return 1
	}
	switch settings.RankMultiplier {
	case 2, 5, 10:
		return settings.RankMultiplier
	default:
		return 1
	}
}

func effectiveRankedStake(settings types.RoomSettings) int {
	return int(settings.Stake) * int(rankMultiplierFor(settings))
}

func (s *Server) extremeHourlyDecayAmount(player *PlayerState) int {
	segment := s.extremeSegmentID(player.Stats.RankedPoints)
	amount, ok := s.cfg.ExtremeMode.HourlyDecay[segment]
	if !ok {
		amount = s.cfg.ExtremeMode.HourlyDecay["default"]
		if amount == 0 {
			amount = 2
		}
	}
	if amount < 0 {
		amount = 0
	}
	return int(math.Round(amount))
}

func (s *Server) applyExtremeHourlyDecay() {
	now := nowMs()
	hour := currentExtremeDecayHour(now)
	changedRoomIDs := map[string]struct{}{}
	changed := false
	for _, player := range s.players {
		if !ptrBool(player.ExtremeModeEnabled) {
			continue
		}
		if player.ExtremeLastDecayHour != nil && *player.ExtremeLastDecayHour == hour {
			continue
		}
		player.ExtremeLastDecayHour = int64Ptr(hour)
		// 即便本小时扣分为 0，也要落盘衰减游标，避免重启后重复扫全表。
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
		amount := s.extremeHourlyDecayAmount(player)
		if amount <= 0 {
			continue
		}
		s.updateRankedPoints(player, -amount)
		changed = true
		if player.RoomID != "" {
			changedRoomIDs[player.RoomID] = struct{}{}
		}
	}
	if !changed {
		return
	}
	for roomID := range changedRoomIDs {
		s.broadcastRoom(roomID, false)
	}
}

func (s *Server) scheduleExtremeHourlyDecay() {
	s.scheduleBoundaryAlignedDecay(3_600_000, currentExtremeDecayHour, time.Hour, s.applyExtremeHourlyDecay)
}

// scheduleBoundaryAlignedDecay 是 scheduleExtremeHourlyDecay/scheduleRankedDailyDecay 共用的
// 调度逻辑：先精确对齐到下一个周期边界（整点/每日零点）触发一次，之后按 period 周期性触发；
// 每次触发都在 s.mu 内跑 apply，与其它状态变更保持串行。periodIndex 把毫秒时间戳换算成
// "第几个周期"（如 currentExtremeDecayHour/currentRankedDecayDay），periodMs 是一个周期的
// 毫秒数（用来算下一个边界），period 是等间隔 ticker 的间隔，应与 periodMs 一致。
func (s *Server) scheduleBoundaryAlignedDecay(periodMs int64, periodIndex func(nowMs int64) int64, period time.Duration, apply func()) {
	now := nowMs()
	nextBoundary := (periodIndex(now) + 1) * periodMs
	delay := nextBoundary - now + 500
	if delay < 1000 {
		delay = 1000
	}
	timeAfterFunc(time.Duration(delay)*time.Millisecond, func() {
		s.mu.Lock()
		apply()
		s.mu.Unlock()
		ticker := time.NewTicker(period)
		go func() {
			for range ticker.C {
				s.mu.Lock()
				apply()
				s.mu.Unlock()
			}
		}()
	})
}

// rankedDailyDecayAmount 按配置的比例算出衰减后的目标分值（向 0 截断小数部分）。
// 与极限模式的整点衰减是两套独立机制：这里对所有玩家生效，不要求开启极限模式。
func (s *Server) rankedDailyDecayAmount(player *PlayerState) int {
	ratio := s.cfg.RankedScore.DailyDecayRatio
	if ratio <= 0 || ratio > 1 {
		ratio = 1
	}
	return int(float64(player.Stats.RankedPoints) * ratio)
}

func (s *Server) applyRankedDailyDecay() {
	now := nowMs()
	day := currentRankedDecayDay(now)
	changedRoomIDs := map[string]struct{}{}
	changed := false
	for _, player := range s.players {
		if player.RankedLastDecayDay != nil && *player.RankedLastDecayDay == day {
			continue
		}
		player.RankedLastDecayDay = int64Ptr(day)
		// 0 分玩家也要落盘日游标，避免次日重启重复扫。
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
		if player.Stats.RankedPoints == 0 {
			continue
		}
		next := s.rankedDailyDecayAmount(player)
		delta := next - player.Stats.RankedPoints
		if delta == 0 {
			continue
		}
		s.updateRankedPoints(player, delta)
		changed = true
		if player.RoomID != "" {
			changedRoomIDs[player.RoomID] = struct{}{}
		}
	}
	if !changed {
		return
	}
	for roomID := range changedRoomIDs {
		s.broadcastRoom(roomID, false)
	}
}

func (s *Server) scheduleRankedDailyDecay() {
	s.scheduleBoundaryAlignedDecay(86_400_000, currentRankedDecayDay, 24*time.Hour, s.applyRankedDailyDecay)
}

func (s *Server) resetExtremeWinStreak(player *PlayerState) {
	if player != nil && ptrBool(player.ExtremeModeEnabled) {
		player.ExtremeWinStreak = intPtr(0)
		s.markPlayerDirty(player)
		s.requestPersist("lazy")
	}
}

func (s *Server) applyExtremeWinStreakRisk(room *RoomState, winner *PlayerState) string {
	if winner == nil || !ptrBool(winner.ExtremeModeEnabled) || !room.Settings.EnableRanked {
		return ""
	}
	streak := ptrInt(winner.ExtremeWinStreak) + 1
	winner.ExtremeWinStreak = intPtr(streak)
	s.markPlayerDirty(winner)
	s.requestPersist("lazy")
	threshold := s.cfg.ExtremeMode.WinStreakThreshold
	if streak < threshold {
		return ""
	}
	if rand.Float64() >= s.cfg.ExtremeMode.WinStreakCrashChance {
		return ""
	}
	penalty := s.cfg.ExtremeMode.CrashTargetPoints
	if penalty < 1 {
		penalty = 1
	}
	s.updateRankedPoints(winner, -penalty)
	s.refreshPlayerSnapshots(winner)
	return fmt.Sprintf("；%s 极限连胜触发风险，额外扣 %d 分", playerShortName(winner), penalty)
}

func (s *Server) nameWarRenameQuota(player *PlayerState, now int64) int {
	if player.NameWarRenameWindowStartedAt == nil || now-*player.NameWarRenameWindowStartedAt >= 10_800_000 {
		player.NameWarRenameWindowStartedAt = int64Ptr(now)
		player.NameWarRenameCount = intPtr(0)
	}
	return 3 - ptrInt(player.NameWarRenameCount)
}

func (s *Server) nameWarPenaltyThreshold() int {
	th := s.cfg.NameWar.PenaltyThreshold
	if th >= 0 {
		// 配置异常时的安全兜底：名争失格必须是负分线。
		return defaultNameWarPenaltyThreshold
	}
	return th
}

func (s *Server) nameWarRenameMinPoints() int {
	pts := s.cfg.NameWar.RenameMinPoints
	if pts < 1 {
		// 配置异常时的安全兜底。
		return defaultNameWarRenameMinPoints
	}
	return pts
}

func (s *Server) isNameWarRenameTarget(player types.PublicPlayer) bool {
	// 用真实分（SortRankedPoints）判定，不用展示封顶后的 RankedPoints。
	return ptrBool(player.NameWarEnabled) && ptrBool(player.NameWarAllowRename) &&
		ptrBool(player.NameWarPunished) && player.Stats.SortRankedPoints <= s.nameWarPenaltyThreshold()
}
