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

func (s *Server) buildPunishmentTasks(room *RoomState, punishedPlayers []*PlayerState, result types.RoundResult) []types.PunishmentTask {
	return s.buildPunishmentTasksWithWinnerName(room, punishedPlayers, s.winnerNameForResult(room, result))
}

// buildPunishmentTasksWithWinnerName：与 buildPunishmentTasks 同构，但胜者展示名由调用方直接给出
// （大话骰的赢家不是 Seat 反查得到的，见 buildLiarsDicePunishmentTasks）。
// 始终返回非 nil 切片，避免 history 里 punishmentTasks: null。
func (s *Server) buildPunishmentTasksWithWinnerName(room *RoomState, punishedPlayers []*PlayerState, winnerName string) []types.PunishmentTask {
	out := make([]types.PunishmentTask, 0, len(punishedPlayers))
	// 房间级难度进度整局只 +1：平局双罚/双败一局会惩罚两名玩家，二者仍共用同一个目标难度基数，
	// 且整局最多推进一次进度，而不是每个受罚玩家各推进一次。
	baseCount := room.PunishmentTaskProgress
	advancedRound := false
	src := normalizePunishmentSource(room.Settings.PunishmentSource)
	for _, player := range punishedPlayers {
		var assigner *PlayerState
		var systemTask *punishmentTaskResult
		if src == "player" {
			assigner = s.taskAssigner(room, player.ID)
		} else if src == "series" {
			systemTask = s.pickSeriesTaskForPlayer(room, player, room.Settings.PunishmentSeriesID, winnerName)
		} else {
			var advanced bool
			systemTask, advanced = s.pickSystemTaskForPlayer(room, player, winnerName, baseCount)
			if advanced {
				advancedRound = true
			}
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
			if s.eventDB != nil {
				task.EventID = randomID()
				if err := s.eventDB.insertPunishmentTask(task.EventID, nowMs(), "system", room.ID, "", "", player.ID, task.PlayerName, task.TaskText); err != nil {
					s.errorLog("punishment_event_insert_failed", err.Error())
				}
			}
		}
		if assigner != nil {
			task.AssignedBy = assigner.ID
			task.AssignedByName = assigner.Name
		}
		out = append(out, task)
	}
	if advancedRound {
		room.PunishmentTaskProgress = baseCount + 1
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
}

// pickSystemTaskForPlayer 按玩家阵营 + 房间标签筛选 + 房间级任务难度进度，抽取一条随机任务：
//  1. 阵营过滤（factionIds 为空或不含阵营的任务）；
//  2. 标签过滤：任务标签与拒绝集合无交，且包含集合是任务标签的子集；
//     包含集合为空时退化为只按拒绝集合排除；未勾选任何标签的任务无视筛选、永远入选；
//  3. 难度过滤：难度为 -1 的任务排除出随机候选池（仅供系列任务按 ID 引用）；
//  4. 候选整体丢进 weightedDifficultyPick（单阶段，全局难度参数）。
//
// count 是抽取时用的目标难度基数（通常即 room.PunishmentTaskProgress）。
// 真正抽中难度题时 advanced=true，调用方据此决定房间级进度是否 +1。
// 候选池为空时返回通用兜底话术，advanced=false。
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
	bg := ""
	if len(task.BackgroundImages) > 0 {
		bg = task.BackgroundImages[randIntn(len(task.BackgroundImages))]
	}
	typeName := s.tagNamesForTask(task)
	return &punishmentTaskResult{TaskText: taskText, TypeName: typeName, BackgroundImage: bg, BackgroundOpacity: &op}, true
}

// defaultSeriesFactionFallbackText：管理员未配置兜底文案时，系列任务某一步
// 没有覆盖受罚者阵营的下发的内置默认文案。
const defaultSeriesFactionFallbackText = "该任务不适用于你所在的阵营"

// pickSeriesTaskForPlayer 按房间共享的系列进度取下一步，再从该步 taskIds 按阵营挑任务；
// 产出后推进房间计数器（+1）。进度是房间级的：全员共用，前一个玩家做完离开，后一个
// 玩家从下一步继续；换房间（或房内换系列）从 0 重新计数，房间销毁即释放，不落盘。
// 进度越界（系列步骤被改短等）clamp 到最后一条反复执行。某一步没有覆盖受罚者阵营时
// 系列照常生效：下发管理员可配的兜底文案（见 defaultSeriesFactionFallbackText），
// 进度同样推进。
func (s *Server) pickSeriesTaskForPlayer(room *RoomState, player *PlayerState, seriesID, winnerName string) *punishmentTaskResult {
	fallback := func() *punishmentTaskResult {
		op := 0.22
		return &punishmentTaskResult{TaskText: "请完成本局惩罚。", BackgroundOpacity: &op}
	}
	seriesID = strings.TrimSpace(seriesID)
	series := s.findSeriesByID(seriesID)
	if series == nil || len(series.Steps) == 0 {
		return fallback()
	}
	next := 0
	if room.PunishmentSeriesProgressID == seriesID {
		next = room.PunishmentSeriesProgress
	}
	if next < 0 {
		next = 0
	}
	if next >= len(series.Steps) {
		next = len(series.Steps) - 1
	}
	advance := func() {
		room.PunishmentSeriesProgressID = seriesID
		room.PunishmentSeriesProgress = next + 1
	}
	taskByID := make(map[string]*types.PunishmentTaskConfig, len(s.punishmentTasksCache))
	for i := range s.punishmentTasksCache {
		t := &s.punishmentTasksCache[i]
		taskByID[t.ID] = t
	}
	task := pickSeriesStepTask(series.Steps[next].TaskIDs, player.FactionID, taskByID)
	if task == nil {
		text := strings.TrimSpace(s.cfg.PunishmentRandomSettings.SeriesFactionFallbackText)
		if text == "" {
			text = defaultSeriesFactionFallbackText
		}
		op := 0.22
		advance()
		return &punishmentTaskResult{
			TaskText:          applyPunishmentPlaceholders(text, playerShortName(player), winnerName),
			TypeName:          series.Name,
			BackgroundOpacity: &op,
		}
	}
	if strings.TrimSpace(task.Text) == "" {
		return fallback()
	}
	taskText := applyPunishmentPlaceholders(strings.TrimSpace(task.Text), playerShortName(player), winnerName)
	if taskText == "" {
		return fallback()
	}
	op := task.BackgroundOpacity
	bg := ""
	if len(task.BackgroundImages) > 0 {
		bg = task.BackgroundImages[randIntn(len(task.BackgroundImages))]
	}
	advance()
	return &punishmentTaskResult{
		TaskText:          taskText,
		TypeName:          series.Name,
		BackgroundImage:   bg,
		BackgroundOpacity: &op,
	}
}

// pickSeriesStepTask：收集 taskIds 中所有 FactionIDs 精确命中 factionID 的任务，随机挑一个；
// 未勾选任何阵营的任务永远不参与匹配；指向不存在 ID 的直接跳过；全落空返回 nil——
// 调用方（pickSeriesTaskForPlayer）此时下发阵营兜底文案，系列照常生效。
func pickSeriesStepTask(taskIDs []string, factionID string, taskByID map[string]*types.PunishmentTaskConfig) *types.PunishmentTaskConfig {
	var matches []*types.PunishmentTaskConfig
	for _, id := range taskIDs {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		t := taskByID[id]
		if t == nil {
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

// seriesIsUsable 判断系列任务是否可用：至少要有一步。阵营覆盖不再影响可用性——
// 某一步没覆盖到受罚者阵营时，运行时下发兜底文案（pickSeriesTaskForPlayer），
// 系列仍会出现在建房面板目录里。调用方须持有 s.mu。
func (s *Server) seriesIsUsable(series types.PunishmentSeriesTaskConfig) bool {
	return len(series.Steps) > 0
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
// 保存路径（config.NormalizePunishmentTasks）保证落库值只会是 -1 或 1-99，此处多一层兜底。
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

// weightedDifficultyPick 按房间目标难度，用倒伽马（反射 Gamma）加权随机挑一个候选任务：
//
//	目标难度 target = clamp((count+1)*step, 候选最小难度, 候选最大难度)
//	硬顶 hardCap = target + overshoot（不含该难度：难度 >= hardCap 权重为 0）
//	变换 X = hardCap - 难度；X ~ Gamma(α=4, θ=overshoot/3)，众数在 X=overshoot ⇒ 峰值在 target
//	向简单侧（X 增大）拖尾；更难侧在 hardCap 处截断为 0。
//
// count 是本房间内系统任务已被抽取的总次数（与玩家、与任务类型无关）。
// 若候选全被硬顶滤掉，则退化为只在最简单那一档里抽。
// step/overshoot <=0 时用代码内默认值兜底。传入的 candidates 不能为空。
func weightedDifficultyPick(candidates []types.PunishmentTaskConfig, count int, step, overshoot float64) types.PunishmentTaskConfig {
	if step <= 0 {
		step = defaultPunishmentOrderStep
	}
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
	target := float64(count+1) * step
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
