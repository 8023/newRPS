package server

import (
	"fmt"
	"math"
	"math/rand"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// defaultPunishmentOrderStep/defaultPunishmentDifficultyOvershoot：
// 任务类型未配置 orderStep/maxDifficultyOvershoot（即 <=0）时的兜底默认值，见 weightedDifficultyPick。
// maxDifficultyOvershoot 为更难一侧硬顶偏移：难度 >= target+overshoot 永不抽中（不含该边界难度）；
// 更简单的不受限。抽取权重为倒伽马（反射 Gamma，shape 固定为 punishmentDifficultyGammaShape）。
const (
	defaultPunishmentOrderStep           = 2.0
	defaultPunishmentDifficultyOvershoot = 5.0
	// punishmentDifficultyGammaShape：倒伽马形状参数 α。众数约束 mode_X = overshoot = (α-1)θ，
	// 故 scale θ = overshoot/(α-1)。α=4 ⇒ θ = overshoot/3。
	punishmentDifficultyGammaShape = 4.0
)

func (s *Server) punishmentPlayersForResult(room *RoomState, result types.RoundResult) []*PlayerState {
	if !room.Settings.EnablePunishment {
		return nil
	}
	var punishSeats []types.SeatKey
	if result == types.ResultDoubleLoss {
		punishSeats = []types.SeatKey{types.SeatA, types.SeatB}
	} else if result == types.ResultDraw {
		if room.Settings.TieDoublePunish {
			punishSeats = []types.SeatKey{types.SeatA, types.SeatB}
		}
	} else {
		punishSeats = []types.SeatKey{oppositeSeat(types.SeatKey(result))}
	}
	var out []*PlayerState
	for _, seat := range punishSeats {
		occ := room.Seats[seat]
		if occ == nil {
			continue
		}
		if p := s.players[occ.GetID()]; p != nil {
			out = append(out, p)
		}
	}
	return out
}

func (s *Server) addRoundHistory(room *RoomState, item types.RoundHistoryItem) {
	item = sanitizeRoundHistoryItem(item)
	s.bindPunishmentEventsToRound(room, item)
	room.RoundHistory = append([]types.RoundHistoryItem{item}, room.RoundHistory...)
	if len(room.RoundHistory) > roomHistoryMaxKeep {
		room.RoundHistory = room.RoundHistory[:roomHistoryMaxKeep]
	}
	s.emitToRoom(room.ID, "room:historyAppend", map[string]any{
		"roomId": room.ID,
		"item":   item,
		"total":  len(room.RoundHistory),
	})
	s.requestPersist("lazy")
}

func randIntn(n int) int {
	if n <= 0 {
		return 0
	}
	return rand.Intn(n)
}

// normalizePunishmentSource 把历史值 "system" 与空串归一为 "random"。
func normalizePunishmentSource(src string) string {
	switch strings.TrimSpace(src) {
	case "", "system":
		return "random"
	case "random", "series", "player":
		return src
	default:
		return "random"
	}
}

// punishmentRoundLabel 是本局惩罚在历史记录里的标题：玩家发布任务模式固定文案；
// 随机/系列模式下按本局实际抽中任务的标签名（TypeName）去重列出，
// 全员都落到兜底话术时给通用占位文案。
func (s *Server) punishmentRoundLabel(room *RoomState, tasks []types.PunishmentTask) string {
	src := normalizePunishmentSource(room.Settings.PunishmentSource)
	if src == "player" {
		return "玩家发布任务"
	}
	if src == "series" {
		if id := strings.TrimSpace(room.Settings.PunishmentSeriesID); id != "" {
			if series := s.findSeriesByID(id); series != nil {
				return series.Name
			}
		}
		return "系列任务"
	}
	seen := map[string]struct{}{}
	var names []string
	for _, t := range tasks {
		if t.TypeName == "" {
			continue
		}
		if _, ok := seen[t.TypeName]; ok {
			continue
		}
		seen[t.TypeName] = struct{}{}
		names = append(names, t.TypeName)
	}
	if len(names) == 0 {
		return "随机任务"
	}
	return strings.Join(names, "、")
}

func (s *Server) buildPunishmentTasks(room *RoomState, punishedPlayers []*PlayerState, result types.RoundResult, trigger string) []types.PunishmentTask {
	return s.buildPunishmentTasksWithWinnerName(room, punishedPlayers, s.winnerNameForResult(room, result), trigger)
}

// buildPunishmentTasksWithWinnerName：与 buildPunishmentTasks 同构，但胜者展示名由调用方直接给出
// （大话骰的赢家不是 Seat 反查得到的，见 buildLiarsDicePunishmentTasks）。
// 始终返回非 nil 切片，避免 history 里 punishmentTasks: null。
//
// approver（原"publisher"）恒定是 punishmentReviewer(room, 受罚者)——不管任务来源是随机/
// 系列/玩家发布，都是"这局里另一个座位上的人"（大话骰是本局赢家）；玩家发布任务场景下
// "谁发布的"和"谁审批的"天然是同一个人，taskAssigner 本就是 punishmentReviewer 的别名。
// tie_double_punish：一次结算同时惩罚 >1 人时，这几个人之间没有真正的赢家、只是互相审批
// （平局双罚/双输判负都属于这种情况），供审计参考，不参与任何业务判断。
func (s *Server) buildPunishmentTasksWithWinnerName(room *RoomState, punishedPlayers []*PlayerState, winnerName, trigger string) []types.PunishmentTask {
	out := make([]types.PunishmentTask, 0, len(punishedPlayers))
	src := normalizePunishmentSource(room.Settings.PunishmentSource)
	tieDoublePunish := len(punishedPlayers) > 1
	for _, player := range punishedPlayers {
		reviewer := s.punishmentReviewer(room, player.ID)
		var systemTask *punishmentTaskResult
		if src == "series" {
			systemTask = s.pickSeriesTaskForPlayer(room, player, room.Settings.PunishmentSeriesID, winnerName)
		} else if src != "player" {
			systemTask = s.pickSystemTaskForPlayerAdvancing(room, player, winnerName)
		}
		task := types.PunishmentTask{
			PlayerID:     player.ID,
			PlayerName:   playerShortName(player),
			FactionID:    player.FactionID,
			FactionLabel: player.FactionLabel,
			TaskText:     "",
		}
		if systemTask != nil {
			task.TaskText = systemTask.TaskText
			task.TypeName = systemTask.TypeName
			task.BackgroundImage = systemTask.BackgroundImage
			if systemTask.BackgroundOpacity != nil {
				task.BackgroundOpacity = systemTask.BackgroundOpacity
			}
		}
		if reviewer != nil {
			task.AssignedBy = reviewer.ID
			task.AssignedByName = reviewer.Name
		}
		// player 来源不在这里建事件行：此时玩家还没输入任务文案，事件要等
		// handlers_room.go 里对方真正提交自定义任务文案时才创建（approver/performer 的
		// 位置参数跟这里改名前完全一致，那两处调用不用跟着改）。
		if s.eventDB != nil && systemTask != nil {
			eventID := randomID()
			approverID, approverName := "", ""
			if reviewer != nil {
				approverID, approverName = reviewer.ID, playerShortName(reviewer)
			}
			meta := systemTask.EventMeta
			meta.Trigger = trigger
			meta.TieDoublePunish = tieDoublePunish
			if err := s.eventDB.insertPunishmentTask(eventID, nowMs(), room.ID, approverID, approverName, player.ID, task.PlayerName, task.TaskText, meta); err != nil {
				s.errorLog("punishment_event_insert_failed", err.Error())
			} else {
				task.EventID = eventID
			}
		}
		out = append(out, task)
	}
	return out
}

// latestPunishmentTask 返回 room.RoundHistory[0] 中某玩家当前的惩罚任务（指针，可原地改写）。
func latestPunishmentTask(room *RoomState, playerID string) *types.PunishmentTask {
	if len(room.RoundHistory) == 0 {
		return nil
	}
	latest := &room.RoundHistory[0]
	for i := range latest.PunishmentTasks {
		if latest.PunishmentTasks[i].PlayerID == playerID {
			return &latest.PunishmentTasks[i]
		}
	}
	return nil
}

func (s *Server) taskAssigner(room *RoomState, punishedPlayerID string) *PlayerState {
	return s.punishmentReviewer(room, punishedPlayerID)
}

// punishmentReviewer 返回有权给 punishedID 发布任务 / 审核其证明的玩家：
// 座位制游戏是对手座位上的真人；大话骰不进 Seats 体系，是本局赢家。
func (s *Server) punishmentReviewer(room *RoomState, punishedID string) *PlayerState {
	if room.Settings.GameID == types.GameLiarsDice {
		return s.liarsDicePunishmentReviewer(room, punishedID)
	}
	return s.humanOpponent(room, punishedID)
}

type punishmentTaskResult struct {
	TaskText          string
	TypeName          string
	BackgroundImage   string
	BackgroundOpacity *float64
	EventMeta         punishmentEventMeta
}

// pickSystemTaskForPlayer 按玩家阵营 + 房间标签筛选 + 该玩家自己的任务难度进度，抽取一条随机任务：
//  1. 阵营过滤（factionIds 为空或不含阵营的任务）；
//  2. 标签过滤：任务标签与拒绝集合无交，且包含集合是任务标签的子集；
//     包含集合为空时退化为只按拒绝集合排除；未勾选任何标签的任务无视筛选、永远入选；
//  3. 难度过滤：难度为 -1 的任务排除出随机候选池（仅供系列任务按 ID 引用）；
//  4. 候选整体丢进 weightedDifficultyPick（单阶段，全局难度参数）。
//
// count 是抽取时用的目标难度基数（通常即 room.PunishmentTaskProgress[player.ID]，房间内
// 这名玩家自己的进度）。真正抽中难度题时 advanced=true，调用方据此决定该玩家自己的进度是否
// +1（见 pickSystemTaskForPlayerAdvancing）。候选池为空时返回通用兜底话术，advanced=false。
func (s *Server) pickSystemTaskForPlayer(room *RoomState, player *PlayerState, winnerName string, count int) (*punishmentTaskResult, bool) {
	fallback := func() (*punishmentTaskResult, bool) {
		op := 0.22
		return &punishmentTaskResult{TaskText: "请完成本局惩罚。", BackgroundOpacity: &op}, false
	}
	included := room.Settings.PunishmentTagsIncluded
	excluded := room.Settings.PunishmentTagsExcluded
	pool := candidateTasksForFaction(s.punishmentTasksCache, player.FactionID)
	pool = candidateTasksForTags(pool, included, excluded)
	pool = candidateTasksForRandomDifficulty(pool)
	if len(pool) == 0 {
		return fallback()
	}
	rs := s.cfg.PunishmentRandomSettings
	task := weightedDifficultyPick(pool, count, rs.OrderStep, rs.MaxDifficultyOvershoot)

	taskText := applyPunishmentPlaceholders(strings.TrimSpace(task.Text), playerShortName(player), winnerName)
	if taskText == "" {
		return fallback()
	}
	op := task.BackgroundOpacity
	bg := task.BackgroundImage
	typeName := s.tagNamesForTask(task)
	return &punishmentTaskResult{
		TaskText: taskText, TypeName: typeName, BackgroundImage: bg, BackgroundOpacity: &op,
		EventMeta: s.metaForFormalTask(task),
	}, true
}

// pickSystemTaskForPlayerAdvancing 是 pickSystemTaskForPlayer 的落地入口：读取这名玩家在
// 本房间内自己的难度进度（RoomState.PunishmentTaskProgress[player.ID]，与
// PunishmentSeriesPlayerProgress 同构、按玩家 persistent ID 分槽），抽中难度题（非兜底）时
// 把这名玩家自己的计数器 +1，房间里其他人抽多抽少都不受影响。
func (s *Server) pickSystemTaskForPlayerAdvancing(room *RoomState, player *PlayerState, winnerName string) *punishmentTaskResult {
	baseCount := room.PunishmentTaskProgress[player.ID]
	task, advanced := s.pickSystemTaskForPlayer(room, player, winnerName, baseCount)
	if advanced {
		if room.PunishmentTaskProgress == nil {
			room.PunishmentTaskProgress = map[string]int{}
		}
		room.PunishmentTaskProgress[player.ID] = baseCount + 1
	}
	return task
}

// pickSeriesTaskForPlayer 按该玩家在本房间内自己的系列进度取下一步，再从该步 taskIds 按
// 阵营挑任务；产出后推进这名玩家自己的计数器（+1，见 RoomState.PunishmentSeriesPlayerProgress）。
// 进度越界（系列步骤被改短等）clamp 到最后一条反复执行。某一步没有覆盖受罚者阵营时
// 从随机任务池抽一条替补（见 pickSeriesReplacementTask），进度同样推进。「完成率」不在这里
// 统计——那是房间销毁时才结算的快照，见 recordSeriesRunProgressOnClose。
func (s *Server) pickSeriesTaskForPlayer(room *RoomState, player *PlayerState, seriesID, winnerName string) *punishmentTaskResult {
	fallback := func() *punishmentTaskResult {
		op := 0.22
		return &punishmentTaskResult{TaskText: "请完成本局惩罚。", BackgroundOpacity: &op}
	}
	seriesID = strings.TrimSpace(seriesID)
	series := s.findSeriesByID(seriesID)
	if series == nil || series.StepCount == 0 {
		return fallback()
	}
	next := 0
	if prog := room.PunishmentSeriesPlayerProgress[player.ID]; prog != nil && prog.SeriesID == seriesID {
		next = prog.Step
	}
	if next < 0 {
		next = 0
	}
	if next >= series.StepCount {
		next = series.StepCount - 1
	}
	advance := func() {
		if room.PunishmentSeriesPlayerProgress == nil {
			room.PunishmentSeriesPlayerProgress = map[string]*seriesPlayerProgress{}
		}
		room.PunishmentSeriesPlayerProgress[player.ID] = &seriesPlayerProgress{SeriesID: seriesID, Step: next + 1}
	}
	task := pickSeriesStepTask(s.punishmentSeriesSteps[seriesID], next, player.FactionID)
	if task == nil {
		advance()
		return s.pickSeriesReplacementTask(player, *series, next, winnerName)
	}
	if strings.TrimSpace(task.Text) == "" {
		return fallback()
	}
	taskText := applyPunishmentPlaceholders(strings.TrimSpace(task.Text), playerShortName(player), winnerName)
	if taskText == "" {
		return fallback()
	}
	op := task.BackgroundOpacity
	bg := task.BackgroundImage
	advance()
	return &punishmentTaskResult{
		TaskText:          taskText,
		TypeName:          series.Name,
		BackgroundImage:   bg,
		BackgroundOpacity: &op,
		EventMeta:         s.metaForFormalTask(*task),
	}
}

// recordSeriesRunProgressOnClose 在房间销毁时结算这个房间里每个玩家的系列完成度：只统计
// 「至少走完 1 步」的玩家，样本值是「自己走完的步数 / 系列总步数」的百分比，落库后在
// seriesRunStats 里取这些样本的算术平均——刻意不用"是否走到最后一步"的二元判定，
// 否则一个 20 步的系列，大多数人走了 15 步就退出，完成率会被算得极低，看不出真实体验；
// 按进度百分比取均值（上面例子约 75%）更贴近实际。调用方须持有 s.mu，在
// delete(s.rooms, room.ID) 前后均可（房间对象本身，不依赖它还在不在 s.rooms 里）。三处
// 房间销毁路径都要调：room.go 的 cleanupRoomIfEmpty（正常清空关房）、handlers_room.go 的
// admin closeRoom（管理员强制关房）、server.go 优雅关停时的批量清空。
func (s *Server) recordSeriesRunProgressOnClose(room *RoomState) {
	if s.punishmentStore == nil || len(room.PunishmentSeriesPlayerProgress) == 0 {
		return
	}
	for _, prog := range room.PunishmentSeriesPlayerProgress {
		if prog == nil || prog.Step <= 0 {
			continue
		}
		series := s.findSeriesByID(prog.SeriesID)
		if series == nil || series.StepCount == 0 {
			continue
		}
		steps := prog.Step
		if steps > series.StepCount {
			steps = series.StepCount
		}
		percent := int(float64(steps)/float64(series.StepCount)*100 + 0.5)
		if err := s.punishmentStore.recordSeriesRunProgress(series.ID, series.Version, percent); err != nil {
			s.errorLog("series_run_progress_record_failed", err.Error())
		}
	}
}

// pickSeriesReplacementTask 在系列当前步没有变体命中受罚者阵营时，从随机任务池抽一条顶替。
// 难度按「当前步序 / 总步数」定目标；标签三态由系列自身步骤的标签词频推导；票记到抽中的
// 那条随机任务自己的贡献者，不记到系列头上。
func (s *Server) pickSeriesReplacementTask(player *PlayerState, series types.PunishmentSeriesTaskConfig, stepIndex int, winnerName string) *punishmentTaskResult {
	op := 0.22
	fallback := &punishmentTaskResult{TaskText: "请完成本局惩罚。", BackgroundOpacity: &op, TypeName: series.Name}
	if series.StepCount == 0 {
		return fallback
	}
	ratio := float64(stepIndex+1) / float64(series.StepCount)
	incl, excl := s.seriesTagTriState(series)
	pool := candidateTasksForFaction(s.punishmentTasksCache, player.FactionID)
	pool = candidateTasksForRandomDifficulty(pool)
	p1 := candidateTasksForTags(pool, incl, excl)
	candidates := p1
	if len(candidates) == 0 {
		candidates = candidateTasksForTags(pool, nil, excl)
	}
	if len(candidates) == 0 {
		candidates = pool
	}
	if len(candidates) == 0 {
		return fallback
	}
	task := weightedDifficultyPickByRatio(candidates, ratio, s.cfg.PunishmentRandomSettings.MaxDifficultyOvershoot)
	taskText := applyPunishmentPlaceholders(strings.TrimSpace(task.Text), playerShortName(player), winnerName)
	if taskText == "" {
		return fallback
	}
	bg := task.BackgroundImage
	taskOp := task.BackgroundOpacity
	return &punishmentTaskResult{
		TaskText:          taskText,
		TypeName:          series.Name,
		BackgroundImage:   bg,
		BackgroundOpacity: &taskOp,
		EventMeta:         s.metaForFormalTask(task),
	}
}

// seriesTagTriState 按系列所有步骤引用的任务行统计标签词频：
// included 只取词频最高的恰好一个（并列按标签 ID 字典序），excluded 为配置里一次都没出现的标签。
func (s *Server) seriesTagTriState(series types.PunishmentSeriesTaskConfig) (included, excluded []string) {
	freq := map[string]int{}
	seenStep := map[string]struct{}{}
	for _, variant := range s.punishmentSeriesSteps[series.ID] {
		// 同一步的多份变体共享同一个 ID/TagIDs，只按步骤计一次，不按变体重复计数。
		if _, ok := seenStep[variant.ID]; ok {
			continue
		}
		seenStep[variant.ID] = struct{}{}
		for _, tag := range variant.TagIDs {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			freq[tag]++
		}
	}
	bestID := ""
	bestN := 0
	for id, n := range freq {
		if n > bestN || (n == bestN && n > 0 && (bestID == "" || id < bestID)) {
			bestID = id
			bestN = n
		}
	}
	if bestN > 0 {
		included = []string{bestID}
	}
	for _, tag := range s.cfg.PunishmentTags {
		if freq[tag.ID] == 0 {
			excluded = append(excluded, tag.ID)
		}
	}
	return included, excluded
}

// pickSeriesStepTask：从某系列指定步骤（已展开的变体列表里筛出 StepIndex==stepIndex 的
// 那些）中收集所有 FactionIDs 精确命中 factionID 的变体，随机挑一个；未勾选任何阵营的
// 变体永远不参与匹配；全落空返回 nil——调用方（pickSeriesTaskForPlayer）此时从随机池抽
// 替补任务，系列照常生效。
func pickSeriesStepTask(stepVariants []*types.PunishmentTaskConfig, stepIndex int, factionID string) *types.PunishmentTaskConfig {
	var matches []*types.PunishmentTaskConfig
	for _, t := range stepVariants {
		if t.StepIndex != stepIndex {
			continue
		}
		if containsString(t.FactionIDs, factionID) {
			matches = append(matches, t)
		}
	}
	if len(matches) == 0 {
		return nil
	}
	return matches[randIntn(len(matches))]
}

// seriesIsUsable 判断系列任务的运行时结构是否至少有一步。新系列在提交和审批时已保证
// 每一步覆盖其声明的目标阵营；目标阵营外玩家强行进入房间，或遇到历史异常数据时，
// pickSeriesReplacementTask 仍会从玩家对应阵营的随机池抽取替补。调用方须持有 s.mu。
func (s *Server) seriesIsUsable(series types.PunishmentSeriesTaskConfig) bool {
	return series.StepCount > 0
}

// metaForFormalTask 组装惩罚事件的正式任务溯源信息。投票目标恒定落在「这一条具体任务」
// 的 ID 上（独立随机任务/系列的每一步各自的 ID，都是 sub_tasks 的行 ID），系列的不同步骤
// 天然各有各的 ID，不再按整个系列合并计票——见 eventStore.castPunishmentEventVoteAndAggregate/
// subTaskStore.voteAggregate。贡献者是谁、系列归属、要不要匿名都不再存进这份 meta——
// 需要时由 contribution_vote_rpc.go 按 FormalTaskID/FormalTaskVersion 现查 sub_tasks
// （atVersion），昵称统一现查 s.players，不做快照。
func (s *Server) metaForFormalTask(task types.PunishmentTaskConfig) punishmentEventMeta {
	return punishmentEventMeta{FormalTaskID: task.ID, FormalTaskVersion: task.Version}
}

// bindPunishmentEventsToRound 把本局产生的惩罚事件行都打上 round_id（纯审计用途）。
// 投票资格不再靠"这局的参与者名单"这张单独的表判断——approver_id/performer_id 已经在
// insertPunishmentTask 那一刻直接写死在事件行自己身上（见 buildPunishmentTasksWithWinnerName
// 的注释），这里不需要再额外记录一遍。
func (s *Server) bindPunishmentEventsToRound(room *RoomState, item types.RoundHistoryItem) {
	if s.eventDB == nil || len(item.PunishmentTasks) == 0 {
		return
	}
	for _, t := range item.PunishmentTasks {
		if t.EventID == "" {
			continue
		}
		if err := s.eventDB.bindEventRound(t.EventID, item.ID); err != nil {
			s.errorLog("punishment_event_bind_round_failed", err.Error())
		}
	}
}

// candidateTasksForFaction 筛出勾选了该阵营的任务；未勾选任何阵营的任务永远不参与匹配。
func candidateTasksForFaction(tasks []types.PunishmentTaskConfig, factionID string) []types.PunishmentTaskConfig {
	out := make([]types.PunishmentTaskConfig, 0, len(tasks))
	for _, t := range tasks {
		if containsString(t.FactionIDs, factionID) {
			out = append(out, t)
		}
	}
	return out
}

// candidateTasksForTags：T∩R=∅ 且 S⊆T；S 为空时只按 R 排除。T 为空（任务未勾选任何标签）
// 的任务不受标签筛选控制——无视 S/R 永远留在候选池里，房主无法通过勾选/拒绝标签把它挡在外面。
func candidateTasksForTags(tasks []types.PunishmentTaskConfig, included, excluded []string) []types.PunishmentTaskConfig {
	exclSet := map[string]struct{}{}
	for _, id := range excluded {
		if id = strings.TrimSpace(id); id != "" {
			exclSet[id] = struct{}{}
		}
	}
	incl := make([]string, 0, len(included))
	for _, id := range included {
		if id = strings.TrimSpace(id); id != "" {
			incl = append(incl, id)
		}
	}
	out := make([]types.PunishmentTaskConfig, 0, len(tasks))
	for _, t := range tasks {
		if len(t.TagIDs) == 0 {
			out = append(out, t) // 无标签任务无视标签筛选，永远可能被抽到
			continue
		}
		tagSet := map[string]struct{}{}
		for _, id := range t.TagIDs {
			tagSet[id] = struct{}{}
		}
		rejected := false
		for id := range exclSet {
			if _, ok := tagSet[id]; ok {
				rejected = true
				break
			}
		}
		if rejected {
			continue
		}
		ok := true
		for _, id := range incl {
			if _, has := tagSet[id]; !has {
				ok = false
				break
			}
		}
		if ok {
			out = append(out, t)
		}
	}
	return out
}

// candidateTasksForRandomDifficulty 过滤掉难度为 -1 的任务：-1 是"仅供系列任务按 ID
// 引用、不参与随机抽取"的唯一合法负数标记（现在也是把任务挡在随机池外的唯一手段，
// 标签留空已不再排除随机候选），不参与 weightedDifficultyPick 的难度加权计算。判断用 < 0（而非 == -1）是防御性写法——
// 保存路径（decodeStepDraft，投稿提交/审批时经 contributionStore.validateOwnedDraft 校验）拒绝落库值不是 -1 或 1-99 的任务，此处多一层兜底。
func candidateTasksForRandomDifficulty(tasks []types.PunishmentTaskConfig) []types.PunishmentTaskConfig {
	out := make([]types.PunishmentTaskConfig, 0, len(tasks))
	for _, t := range tasks {
		if t.Order < 0 {
			continue
		}
		out = append(out, t)
	}
	return out
}

// tagNamesForTask 把任务自身携带的标签名去重后用顿号连接，写入 TypeName 供历史标题使用。
func (s *Server) tagNamesForTask(task types.PunishmentTaskConfig) string {
	if len(task.TagIDs) == 0 {
		return ""
	}
	nameByID := map[string]string{}
	for _, tag := range s.cfg.PunishmentTags {
		nameByID[tag.ID] = tag.Name
	}
	seen := map[string]struct{}{}
	var names []string
	for _, id := range task.TagIDs {
		name := nameByID[id]
		if name == "" {
			name = id
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		names = append(names, name)
	}
	return strings.Join(names, "、")
}

// weightedDifficultyPick 按房间目标难度，用倒伽马（反射 Gamma）加权随机挑一个候选任务。
// count 是本房间内系统任务已被抽取的总次数。传入的 candidates 不能为空。
func weightedDifficultyPick(candidates []types.PunishmentTaskConfig, count int, step, overshoot float64) types.PunishmentTaskConfig {
	if step <= 0 {
		step = defaultPunishmentOrderStep
	}
	return pickByDifficultyTarget(candidates, float64(count+1)*step, overshoot)
}

// weightedDifficultyPickByRatio 按系列步序比例定目标难度：target = min + ratio*(max-min)。
func weightedDifficultyPickByRatio(candidates []types.PunishmentTaskConfig, ratio, overshoot float64) types.PunishmentTaskConfig {
	minDiff, maxDiff := candidates[0].Order, candidates[0].Order
	for _, c := range candidates[1:] {
		if c.Order < minDiff {
			minDiff = c.Order
		}
		if c.Order > maxDiff {
			maxDiff = c.Order
		}
	}
	if ratio < 0 {
		ratio = 0
	}
	if ratio > 1 {
		ratio = 1
	}
	target := float64(minDiff) + ratio*float64(maxDiff-minDiff)
	return pickByDifficultyTarget(candidates, target, overshoot)
}

// pickByDifficultyTarget 用倒伽马加权随机挑一个候选：
//
//	目标难度 target = clamp(target, 候选最小难度, 候选最大难度)
//	硬顶 hardCap = target + overshoot（不含该难度：难度 >= hardCap 权重为 0）
//	变换 X = hardCap - 难度；X ~ Gamma(α=4, θ=overshoot/3)，众数在 X=overshoot ⇒ 峰值在 target
//	向简单侧（X 增大）拖尾；更难侧在 hardCap 处截断为 0。
//
// 若候选全被硬顶滤掉，则退化为只在最简单那一档里抽。overshoot <=0 时用代码内默认值兜底。
func pickByDifficultyTarget(candidates []types.PunishmentTaskConfig, target, overshoot float64) types.PunishmentTaskConfig {
	if overshoot <= 0 {
		overshoot = defaultPunishmentDifficultyOvershoot
	}
	minDiff, maxDiff := candidates[0].Order, candidates[0].Order
	for _, c := range candidates[1:] {
		if c.Order < minDiff {
			minDiff = c.Order
		}
		if c.Order > maxDiff {
			maxDiff = c.Order
		}
	}
	if target < float64(minDiff) {
		target = float64(minDiff)
	}
	if target > float64(maxDiff) {
		target = float64(maxDiff)
	}

	// 硬顶不含边界：难度 >= hardCap 不得入池。
	hardCap := target + overshoot
	pool := make([]types.PunishmentTaskConfig, 0, len(candidates))
	for _, c := range candidates {
		if float64(c.Order) < hardCap {
			pool = append(pool, c)
		}
	}
	if len(pool) == 0 {
		// 全部都 >= 硬顶：只能退到最简单的那一档，绝不越级抽更难的。
		for _, c := range candidates {
			if c.Order == minDiff {
				pool = append(pool, c)
			}
		}
	}

	// θ = overshoot / (α-1)，使 Gamma 众数落在 X=overshoot（难度=target）。
	theta := overshoot / (punishmentDifficultyGammaShape - 1)
	weights := make([]float64, len(pool))
	var total float64
	for i, c := range pool {
		x := hardCap - float64(c.Order) // > 0
		w := invGammaDifficultyWeight(x, punishmentDifficultyGammaShape, theta)
		weights[i] = w
		total += w
	}
	if total <= 0 {
		return pool[randIntn(len(pool))]
	}
	r := rand.Float64() * total
	for i, w := range weights {
		r -= w
		if r <= 0 {
			return pool[i]
		}
	}
	return pool[len(pool)-1]
}

// invGammaDifficultyWeight 是 Gamma(α, θ) 在 x>0 上的密度（比例即可；调用方会归一化）。
// x<=0 返回 0。使用 log 域计算避免溢出。
func invGammaDifficultyWeight(x, alpha, theta float64) float64 {
	if x <= 0 || alpha <= 0 || theta <= 0 {
		return 0
	}
	// pdf ∝ x^{α-1} * exp(-x/θ) / (Γ(α) θ^α)；Lgamma 返回 (ln|Γ|, sign)
	lg, _ := math.Lgamma(alpha)
	return math.Exp((alpha-1)*math.Log(x) - x/theta - lg - alpha*math.Log(theta))
}

// applyPunishmentPlaceholders 替换系统/自定义任务文案中的占位符：
//
//	{loser}  → 败者昵称（本条任务对应的受罚玩家）
//	{winner} → 胜者昵称（本局唯一胜者座位；平局双罚/双败时为空字符串）
func applyPunishmentPlaceholders(taskText, loserName, winnerName string) string {
	if taskText == "" {
		return taskText
	}
	taskText = strings.ReplaceAll(taskText, "{loser}", loserName)
	taskText = strings.ReplaceAll(taskText, "{winner}", winnerName)
	return taskText
}

// winnerNameForResult 返回胜者展示名；无明确胜者时返回 ""。
func (s *Server) winnerNameForResult(room *RoomState, result types.RoundResult) string {
	var seat types.SeatKey
	switch result {
	case types.ResultA:
		seat = types.SeatA
	case types.ResultB:
		seat = types.SeatB
	default:
		return ""
	}
	return s.seatShortName(room, seat)
}

func (s *Server) seatShortName(room *RoomState, seat types.SeatKey) string {
	if room == nil {
		return ""
	}
	occ := room.Seats[seat]
	if occ == nil {
		return ""
	}
	if p := s.players[occ.GetID()]; p != nil {
		return playerShortName(p)
	}
	return occupantName(occ)
}

func (s *Server) attachProofToLatestHistory(room *RoomState, proof types.HistoryProof) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	var taskText string
	for _, t := range latest.PunishmentTasks {
		if t.PlayerID == proof.PlayerID {
			taskText = t.TaskText
			break
		}
	}
	filtered := latest.Proofs[:0]
	for _, p := range latest.Proofs {
		if p.PlayerID != proof.PlayerID {
			filtered = append(filtered, p)
		}
	}
	if proof.TaskText == "" {
		proof.TaskText = taskText
	}
	latest.Proofs = append(filtered, proof)
}

func (s *Server) updateProofInLatestHistory(room *RoomState, playerID string, next types.HistoryProof) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	for i := range latest.Proofs {
		if latest.Proofs[i].PlayerID == playerID {
			p := latest.Proofs[i]
			if next.Status != "" {
				p.Status = next.Status
			}
			if next.ReviewedBy != "" {
				p.ReviewedBy = next.ReviewedBy
			}
			if next.ReviewedAt != nil {
				p.ReviewedAt = next.ReviewedAt
			}
			if next.RejectReason != "" {
				p.RejectReason = next.RejectReason
			}
			if next.RedoTaskText != "" {
				p.RedoTaskText = next.RedoTaskText
			}
			if next.Text != "" {
				p.Text = next.Text
			}
			if next.ImageURL != "" {
				p.ImageURL = next.ImageURL
			}
			latest.Proofs[i] = p
			return
		}
	}
}

// proofRejectionPenaltyPoints：同一惩罚任务在同一局内被胜方连续第 N 次审核不通过时的额外扣分。
// 第 1、2 次不扣分；第 3 次起 100，第 4 次 200，第 5 次 300……此后每多一次连续不通过再加罚 100。
func proofRejectionPenaltyPoints(rejectCount int) int {
	if rejectCount < 3 {
		return 0
	}
	return (rejectCount - 2) * 100
}

// applyProofRejectionPenalty 在审核方驳回证明（要求重做）后调用：递增该任务的连续驳回计数，
// 从第 3 次起扣分并通过房间系统提示让双方都能看到扣除情况。全游戏通用（走同一套 onPunishmentReview）。
func (s *Server) applyProofRejectionPenalty(room *RoomState, punishedPlayerID string) {
	task := latestPunishmentTask(room, punishedPlayerID)
	if task == nil {
		return
	}
	task.RejectCount++
	penalty := proofRejectionPenaltyPoints(task.RejectCount)
	if penalty <= 0 {
		return
	}
	punished := s.players[punishedPlayerID]
	if punished == nil {
		return
	}
	s.updateRankedPoints(punished, -penalty)
	s.roomNotice(room, fmt.Sprintf(
		"%s 的惩罚任务已连续第 %d 次审核不通过，系统额外扣除其 %d 积分。",
		playerShortName(punished), task.RejectCount, penalty,
	))
}

func (s *Server) updatePunishmentTask(room *RoomState, playerID, taskText string, assignedBy *PlayerState) {
	if len(room.RoundHistory) == 0 {
		return
	}
	latest := &room.RoundHistory[0]
	for i := range latest.PunishmentTasks {
		if latest.PunishmentTasks[i].PlayerID == playerID {
			latest.PunishmentTasks[i].TaskText = taskText
			if assignedBy != nil {
				latest.PunishmentTasks[i].AssignedBy = assignedBy.ID
				latest.PunishmentTasks[i].AssignedByName = assignedBy.Name
			}
			return
		}
	}
}

func (s *Server) oppositeForgiveProof(room *RoomState, reviewerID, targetID string) *types.PunishmentProof {
	for i := range room.Proofs {
		proof := &room.Proofs[i]
		if proof.PlayerID == reviewerID && proof.Status == "approved" &&
			proof.ReviewedBy == targetID && proof.RejectReason == "对方选择放过你" {
			return proof
		}
	}
	return nil
}

func (s *Server) applyForgiveReview(room *RoomState, reviewerID, targetID string) string {
	opposite := s.oppositeForgiveProof(room, reviewerID, targetID)
	if opposite == nil {
		room.ForgiveAdvantage = &forgiveAdvantage{BeneficiaryID: reviewerID, TargetID: targetID}
		return "对方选择放过你"
	}
	room.ForgiveAdvantage = nil
	opposite.RejectReason = "双方互相放过，下一局正常开始。"
	s.updateProofInLatestHistory(room, reviewerID, types.HistoryProof{RejectReason: "双方互相放过，下一局正常开始。"})
	return "双方互相放过，下一局正常开始。"
}

func (s *Server) setupPunishmentOrNext(room *RoomState, result types.RoundResult) {
	if !room.Settings.EnablePunishment {
		return
	}
	humanIDs := make([]string, 0)
	for _, p := range s.punishmentPlayersForResult(room, result) {
		humanIDs = append(humanIDs, p.ID)
	}
	s.setupPunishmentForPlayers(room, humanIDs)
}

// setupPunishmentForPlayers：进入惩罚阶段的公共尾段（不依赖 Seat/RoundResult，
// 直接吃 playerID 列表）——大话骰（不进 Seats 体系）和其它三个游戏共用这一段。
func (s *Server) setupPunishmentForPlayers(room *RoomState, humanIDs []string) {
	if !room.Settings.EnablePunishment || len(humanIDs) == 0 {
		return
	}
	room.Phase = types.PhasePunishment
	room.Status = "punishment"
	room.PunishedPlayerIDs = humanIDs
	room.LockedSeatIDs = map[string]struct{}{}
	for _, playerID := range humanIDs {
		room.LockedSeatIDs[playerID] = struct{}{}
		if player := s.players[playerID]; player != nil {
			player.Stats.Punishments++
			s.markPlayerDirty(player)
			s.requestPersist("lazy")
		}
		if seat, ok := s.seatOf(room, playerID); ok {
			ss := room.SeatStats[seat]
			ss.Punishments++
			room.SeatStats[seat] = ss
		}
	}
}

func (s *Server) punishmentComplete(room *RoomState) bool {
	for _, playerID := range room.PunishedPlayerIDs {
		var task *types.PunishmentTask
		if len(room.RoundHistory) > 0 {
			for i := range room.RoundHistory[0].PunishmentTasks {
				if room.RoundHistory[0].PunishmentTasks[i].PlayerID == playerID {
					task = &room.RoundHistory[0].PunishmentTasks[i]
					break
				}
			}
		}
		if room.Settings.PunishmentSource == "player" && (task == nil || strings.TrimSpace(task.TaskText) == "") {
			return false
		}
		var proof *types.PunishmentProof
		for i := range room.Proofs {
			if room.Proofs[i].PlayerID == playerID {
				proof = &room.Proofs[i]
				break
			}
		}
		if proof == nil || proof.Status == "rejected" {
			return false
		}
		// 仅 approved（或带 ConfirmedBy）算完成。pending 必须等胜方审批；
		// 关闭「需对手确认」时提交路径会立刻写成 approved。
		if proof.Status != "approved" && proof.ConfirmedBy == "" {
			return false
		}
	}
	return true
}

func (s *Server) humanOpponent(room *RoomState, playerID string) *PlayerState {
	seat, ok := s.seatOf(room, playerID)
	if !ok {
		return nil
	}
	return s.humanPlayerFromSeat(room, oppositeSeat(seat))
}

func proofNeedsReview(proof types.PunishmentProof) bool {
	return proof.Status == "pending" || proof.Status == "rejected"
}

func (s *Server) canReviewPlayer(room *RoomState, reviewerID, targetID string) bool {
	if reviewerID == "" || reviewerID == targetID {
		return false
	}
	if room.Settings.GameID == types.GameLiarsDice {
		reviewer := s.liarsDicePunishmentReviewer(room, targetID)
		return reviewer != nil && reviewer.ID == reviewerID
	}
	rs, ok1 := s.seatOf(room, reviewerID)
	ts, ok2 := s.seatOf(room, targetID)
	return ok1 && ok2 && rs != ts
}

func (s *Server) approveProofBySystem(room *RoomState, playerID, message string) bool {
	var proof *types.PunishmentProof
	for i := range room.Proofs {
		if room.Proofs[i].PlayerID == playerID {
			proof = &room.Proofs[i]
			break
		}
	}
	if proof == nil || proof.Status == "approved" {
		return false
	}
	reviewedAt := nowMs()
	proof.Status = "approved"
	proof.ConfirmedBy = "system-auto-forgive"
	proof.ReviewedBy = "system-auto-forgive"
	proof.ReviewedAt = &reviewedAt
	proof.RejectReason = message
	s.updateProofInLatestHistory(room, playerID, types.HistoryProof{
		Status: "approved", ReviewedBy: "system-auto-forgive", ReviewedAt: &reviewedAt, RejectReason: message,
	})
	s.markPunishmentEventApproved(room, playerID)
	return true
}

func (s *Server) submitSystemPunishmentProof(room *RoomState, player *PlayerState, message string) {
	var taskText string
	if len(room.RoundHistory) > 0 {
		for _, t := range room.RoundHistory[0].PunishmentTasks {
			if t.PlayerID == player.ID {
				taskText = t.TaskText
				break
			}
		}
	}
	for _, p := range room.Proofs {
		if p.PlayerID == player.ID && p.RedoTaskText != "" {
			taskText = p.RedoTaskText
		}
	}
	submittedAt := nowMs()
	filtered := room.Proofs[:0]
	for _, p := range room.Proofs {
		if p.PlayerID != player.ID {
			filtered = append(filtered, p)
		}
	}
	room.Proofs = append(filtered, types.PunishmentProof{
		PlayerID: player.ID, Text: message, TaskText: taskText, Status: "approved",
		ConfirmedBy: "system-timeout", ReviewedBy: "system-timeout", ReviewedAt: &submittedAt,
		RejectReason: message, SubmittedAt: submittedAt,
	})
	s.attachProofToLatestHistory(room, types.HistoryProof{
		PlayerID: player.ID, PlayerName: playerShortName(player), Text: message, TaskText: taskText,
		Status: "approved", ReviewedBy: "system-timeout", ReviewedAt: &submittedAt,
		RejectReason: message, SubmittedAt: submittedAt,
	})
	s.markPunishmentEventApproved(room, player.ID)
}

func (s *Server) markPunishmentEventApproved(room *RoomState, playerID string) {
	if s.eventDB == nil {
		return
	}
	task := latestPunishmentTask(room, playerID)
	if task == nil || task.EventID == "" {
		return
	}
	if err := s.eventDB.updatePunishmentStatus(task.EventID, "approved"); err != nil {
		s.errorLog("punishment_event_update_failed", err.Error())
	}
}

func (s *Server) finishPunishmentIfComplete(room *RoomState) bool {
	if room.Phase == types.PhasePunishment && s.punishmentComplete(room) {
		if room.midGamePunishment {
			s.resumeAfterMidGamePunishment(room)
			return true
		}
		s.resetForNextRound(room)
		return true
	}
	return false
}

func (s *Server) resetForNextRound(room *RoomState) {
	s.prepareNextChoice(room)
	room.PunishedPlayerIDs = []string{}
	room.Proofs = []types.PunishmentProof{}
	room.LockedSeatIDs = map[string]struct{}{}
	s.broadcastRoom(room.ID, true)
}

func (s *Server) handlePunishmentDeparture(room *RoomState, player *PlayerState, reason LeaveReason) {
	if room.Phase != types.PhasePunishment {
		return
	}
	isPunished := containsString(room.PunishedPlayerIDs, player.ID)
	var latest *types.RoundHistoryItem
	if len(room.RoundHistory) > 0 {
		latest = &room.RoundHistory[0]
	}
	if isPunished && (reason == LeaveDisconnectTimeout || reason == LeaveAdminKick) {
		playerName := playerShortName(player)
		message := fmt.Sprintf("%s 超时未返回，系统已处理本局惩罚。", playerName)
		if reason == LeaveAdminKick {
			message = fmt.Sprintf("%s 被管理员移出，系统已处理本局惩罚。", playerName)
		}
		s.submitSystemPunishmentProof(room, player, message)
		delete(room.LockedSeatIDs, player.ID)
		s.roomNotice(room, message)
		if s.finishPunishmentIfComplete(room) {
			return
		}
	}
	if latest != nil {
		for _, task := range latest.PunishmentTasks {
			if task.AssignedBy == player.ID && strings.TrimSpace(task.TaskText) == "" {
				s.updatePunishmentTask(room, task.PlayerID, "对方已离开，请提交文字说明完成本局惩罚。", nil)
				s.roomNotice(room, fmt.Sprintf("%s 离开，系统已为 %s 发布兜底任务。", playerShortName(player), task.PlayerName))
			}
		}
	}
	for _, proof := range append([]types.PunishmentProof{}, room.Proofs...) {
		if proofNeedsReview(proof) && s.canReviewPlayer(room, player.ID, proof.PlayerID) {
			target := s.players[proof.PlayerID]
			s.approveProofBySystem(room, proof.PlayerID, "审核方离开，系统已自动放过对方。")
			name := "对方"
			if target != nil {
				name = playerShortName(target)
			}
			s.roomNotice(room, fmt.Sprintf("%s 离开，系统已自动放过 %s。", playerShortName(player), name))
		}
	}
	s.finishPunishmentIfComplete(room)
}
