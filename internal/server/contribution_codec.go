package server

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

const (
	defaultMinSeriesSteps = 10
	// defaultMaxSeriesSteps：后台「最高步数」留空/填非正数时的默认值，与 maxSeriesSteps
	// （contributionstore.go）这个纯防御性技术上限是两回事——后者只用来挡住畸形/超大
	// payload，管理员配置的数值在其范围内完全生效，不会被静默收紧到这个默认值。
	defaultMaxSeriesSteps = 20
)

func effectiveMinSeriesSteps(cfg types.AppConfig) int {
	n := cfg.PunishmentRandomSettings.MinSeriesSteps
	if n <= 0 {
		return defaultMinSeriesSteps
	}
	if n > maxSeriesSteps {
		return maxSeriesSteps
	}
	return n
}

// effectiveMaxSeriesSteps 返回系列任务最高步数：读后台「随机任务全局难度」配置，<=0
// 时按 defaultMaxSeriesSteps 兜底；技术上限 maxSeriesSteps（contributionstore.go）只是
// 防止畸形数据的最后一道保护，数值定得很宽松，管理员配置多大都在这道保护线以内生效。
func effectiveMaxSeriesSteps(cfg types.AppConfig) int {
	n := cfg.PunishmentRandomSettings.MaxSeriesSteps
	if n <= 0 {
		n = defaultMaxSeriesSteps
	}
	if n > maxSeriesSteps {
		n = maxSeriesSteps
	}
	if min := effectiveMinSeriesSteps(cfg); n < min {
		return min
	}
	return n
}

// tasksFromStepDraft 把一个 step（独立任务投稿只有一个 step；系列则每一步各调用一次）
// 展开成每个阵营变体各一行。同一次调用产出的所有行共享一个新生成的 TaskGroupID——
// 独立任务因此天然是"一份投稿一个组"；系列因为按步骤逐一调用，天然是"每一步各自一个组"，
// 这正是投票粒度落到"一个具体任务"而不是"整份系列"的关键（见 punishment.go 的
// metaForFormalTask）。
func tasksFromStepDraft(d types.StepDraft, sub types.ContributionSubmission, ver int) []types.PunishmentTaskConfig {
	order := d.Order
	if !d.InRandomPool {
		order = -1
	}
	bg := []string{}
	if strings.TrimSpace(d.BackgroundImage) != "" {
		bg = []string{d.BackgroundImage}
	}
	groupID := "tg_" + randomID()
	createdAt := nowMs()
	out := make([]types.PunishmentTaskConfig, 0, len(d.Variants))
	for _, v := range d.Variants {
		out = append(out, types.PunishmentTaskConfig{
			ID:                   "ct_" + randomID(),
			Text:                 strings.TrimSpace(v.Text),
			TagIDs:               d.TagIDs,
			FactionIDs:           append([]string(nil), v.FactionIDs...),
			Order:                order,
			BackgroundImages:     bg,
			BackgroundOpacity:    d.BackgroundOpacity,
			ContributorPlayerID:  sub.SubmitterPlayerID,
			ContributorAnonymous: sub.Anonymous,
			ContentVersion:       ver,
			VoteVersion:          ver,
			SubmissionID:         sub.ID,
			TaskGroupID:          groupID,
			CreatedAt:            createdAt,
		})
	}
	return out
}

func decodeSeriesPublish(raw string, sub types.ContributionSubmission, ver int) (types.PunishmentSeriesTaskConfig, []types.PunishmentTaskConfig, error) {
	draft, _, err := decodeSeriesDraft(raw)
	if err != nil {
		return types.PunishmentSeriesTaskConfig{}, nil, err
	}
	id := sub.PublishedTargetID
	if id == "" {
		id = "cs_" + randomID()
	}
	outTasks := make([]types.PunishmentTaskConfig, 0)
	steps := make([]types.PunishmentSeriesStep, 0, len(draft.Steps))
	for i, st := range draft.Steps {
		part := tasksFromStepDraft(st, sub, ver)
		if len(part) == 0 {
			return types.PunishmentSeriesTaskConfig{}, nil, fmt.Errorf("第 %d 步没有文案变体", i+1)
		}
		ids := make([]string, 0, len(part))
		for _, t := range part {
			ids = append(ids, t.ID)
		}
		outTasks = append(outTasks, part...)
		steps = append(steps, types.PunishmentSeriesStep{TaskIDs: ids})
	}
	series := types.PunishmentSeriesTaskConfig{
		ID:                   id,
		Name:                 draft.Name,
		RoomNamePool:         &types.RoomNamePool{Subjects: []string{draft.Name}, RoomWords: []string{"小屋"}, Adjectives: []string{"共建"}},
		RoomBackgroundImages: []string{},
		Steps:                steps,
		ContributorPlayerID:  sub.SubmitterPlayerID,
		ContributorAnonymous: sub.Anonymous,
		ContentVersion:       ver,
		VoteVersion:          ver,
		SubmissionID:         sub.ID,
		TargetFactionIDs:     append([]string(nil), draft.TargetFactionIDs...),
		// CreatedAt 只在首次发布（series ID 为空、upsertSeriesTx 走 INSERT 分支）时真正生效；
		// 修订重审命中 ON CONFLICT 时该分支不写这一列，原值保留，这里传入的新时间戳会被忽略。
		CreatedAt: nowMs(),
	}
	return series, outTasks, nil
}

func decodeStepDraft(raw string) (types.StepDraft, error) {
	d, err := normalizeLegacyStepJSON([]byte(raw))
	if err != nil {
		return types.StepDraft{}, err
	}
	return validateStepDraft(d)
}

func decodeSeriesDraft(raw string) (types.SeriesDraft, bool, error) {
	draft, skipCoverage, err := normalizeLegacySeriesJSON([]byte(raw))
	if err != nil {
		return types.SeriesDraft{}, false, err
	}
	draft.Name = strings.TrimSpace(draft.Name)
	if draft.Name == "" || len(draft.Steps) == 0 {
		return types.SeriesDraft{}, false, fmt.Errorf("系列名称和步骤不能为空")
	}
	if len([]rune(draft.Name)) > maxSeriesNameRunes {
		return types.SeriesDraft{}, false, fmt.Errorf("系列名称不能超过 %d 个字符", maxSeriesNameRunes)
	}
	if len(draft.Steps) > maxSeriesSteps {
		return types.SeriesDraft{}, false, fmt.Errorf("系列步骤过多")
	}
	for i, st := range draft.Steps {
		validated, err := validateStepDraft(st)
		if err != nil {
			return types.SeriesDraft{}, false, fmt.Errorf("第 %d 步：%w", i+1, err)
		}
		draft.Steps[i] = validated
	}
	draft.TargetFactionIDs = uniqueNonEmpty(draft.TargetFactionIDs)
	if len(draft.TargetFactionIDs) > maxContributionRefs {
		return types.SeriesDraft{}, false, fmt.Errorf("系列目标阵营过多")
	}
	return draft, skipCoverage, nil
}

func validateStepDraft(d types.StepDraft) (types.StepDraft, error) {
	if len(d.Variants) == 0 {
		return d, fmt.Errorf("至少需要一份文案")
	}
	if len(d.Variants) > maxStepCandidates {
		return d, fmt.Errorf("文案变体过多")
	}
	for i := range d.Variants {
		d.Variants[i].Text = strings.TrimSpace(d.Variants[i].Text)
		if d.Variants[i].Text == "" {
			return d, fmt.Errorf("每份文案都不能为空")
		}
		if len([]rune(d.Variants[i].Text)) > maxTaskTextRunes {
			return d, fmt.Errorf("每份文案不能超过 %d 个字符", maxTaskTextRunes)
		}
		d.Variants[i].FactionIDs = uniqueNonEmpty(d.Variants[i].FactionIDs)
		if len(d.Variants[i].FactionIDs) > maxContributionRefs {
			return d, fmt.Errorf("文案适用阵营过多")
		}
	}
	if len(d.Variants) > 1 {
		seen := map[string]int{}
		for _, v := range d.Variants {
			for _, fid := range v.FactionIDs {
				seen[fid]++
				if seen[fid] > 1 {
					return d, fmt.Errorf("多份文案不能勾选相同阵营")
				}
			}
		}
	}
	if d.InRandomPool && (d.Order < 1 || d.Order > 99) {
		return d, fmt.Errorf("难度必须在 1 到 99 之间")
	}
	if d.BackgroundOpacity == 0 {
		d.BackgroundOpacity = types.DefaultContributionBGOpacity
	}
	if d.BackgroundOpacity < 0 || d.BackgroundOpacity > 1 {
		return d, fmt.Errorf("背景透明率必须在 0 到 1 之间")
	}
	d.TagIDs = uniqueNonEmpty(d.TagIDs)
	if len(d.TagIDs) > maxContributionRefs {
		return d, fmt.Errorf("惩罚标签过多")
	}
	return d, nil
}

func normalizeLegacyStepJSON(raw []byte) (types.StepDraft, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return types.StepDraft{}, fmt.Errorf("任务投稿格式错误")
	}
	if variants, ok := probe["variants"]; ok && len(variants) > 0 && string(variants) != "null" {
		var d types.StepDraft
		if err := json.Unmarshal(raw, &d); err != nil {
			return types.StepDraft{}, fmt.Errorf("任务投稿格式错误")
		}
		return d, nil
	}
	if text, ok := probe["text"]; ok && len(text) > 0 && string(text) != "null" && string(text) != `""` {
		var old types.TaskDraft
		if err := json.Unmarshal(raw, &old); err != nil {
			return types.StepDraft{}, fmt.Errorf("任务投稿格式错误")
		}
		return stepFromLegacyTask(old), nil
	}
	var d types.StepDraft
	if err := json.Unmarshal(raw, &d); err != nil {
		return types.StepDraft{}, fmt.Errorf("任务投稿格式错误")
	}
	return d, nil
}

func stepFromLegacyTask(old types.TaskDraft) types.StepDraft {
	return types.StepDraft{
		Variants: []types.TaskVariantDraft{{
			Text:       old.Text,
			FactionIDs: old.FactionIDs,
		}},
		InRandomPool:      old.Order != -1,
		Order:             old.Order,
		TagIDs:            old.TagIDs,
		BackgroundImage:   old.BackgroundImage,
		BackgroundOpacity: old.BackgroundOpacity,
	}
}

func normalizeLegacySeriesJSON(raw []byte) (types.SeriesDraft, bool, error) {
	var probe struct {
		Name             string          `json:"name"`
		TargetFactionIDs []string        `json:"targetFactionIds"`
		Steps            json.RawMessage `json:"steps"`
		Tasks            json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(raw, &probe); err != nil {
		return types.SeriesDraft{}, false, fmt.Errorf("系列投稿格式错误")
	}
	if isLegacySeriesJSON(probe.Steps, probe.Tasks) {
		var old types.LegacySeriesDraft
		if err := json.Unmarshal(raw, &old); err != nil {
			return types.SeriesDraft{}, false, fmt.Errorf("系列投稿格式错误")
		}
		return expandLegacySeries(old)
	}
	var draft types.SeriesDraft
	if err := json.Unmarshal(raw, &draft); err != nil {
		return types.SeriesDraft{}, false, fmt.Errorf("系列投稿格式错误")
	}
	return draft, false, nil
}

func isLegacySeriesJSON(steps, tasks json.RawMessage) bool {
	if len(tasks) > 0 && string(tasks) != "null" && string(tasks) != "[]" {
		return true
	}
	var rawSteps []map[string]json.RawMessage
	if err := json.Unmarshal(steps, &rawSteps); err != nil || len(rawSteps) == 0 {
		return false
	}
	_, hasRefs := rawSteps[0]["taskRefs"]
	_, hasVariants := rawSteps[0]["variants"]
	return hasRefs && !hasVariants
}

func expandLegacySeries(old types.LegacySeriesDraft) (types.SeriesDraft, bool, error) {
	byRef := make(map[string]types.TaskDraft, len(old.Tasks))
	for _, item := range old.Tasks {
		ref := strings.TrimSpace(item.Ref)
		if ref == "" {
			return types.SeriesDraft{}, false, fmt.Errorf("系列任务引用不能为空")
		}
		if _, exists := byRef[ref]; exists {
			return types.SeriesDraft{}, false, fmt.Errorf("系列任务引用重复")
		}
		byRef[ref] = item.Task
	}
	steps := make([]types.StepDraft, 0, len(old.Steps))
	for i, st := range old.Steps {
		if len(st.TaskRefs) == 0 || len(st.TaskRefs) > maxStepCandidates {
			return types.SeriesDraft{}, false, fmt.Errorf("第 %d 步候选数不合法", i+1)
		}
		var variants []types.TaskVariantDraft
		captured := false
		order := 50
		inPool := true
		var tagIDs []string
		bg := ""
		opacity := types.DefaultContributionBGOpacity
		for _, ref := range st.TaskRefs {
			task, ok := byRef[strings.TrimSpace(ref)]
			if !ok {
				return types.SeriesDraft{}, false, fmt.Errorf("步骤引用了无权使用的任务")
			}
			step := stepFromLegacyTask(task)
			if !captured {
				captured = true
				order = step.Order
				inPool = step.InRandomPool
				tagIDs = step.TagIDs
				bg = step.BackgroundImage
				opacity = step.BackgroundOpacity
			}
			variants = append(variants, step.Variants...)
		}
		steps = append(steps, types.StepDraft{
			Variants:          variants,
			InRandomPool:      inPool,
			Order:             order,
			TagIDs:            tagIDs,
			BackgroundImage:   bg,
			BackgroundOpacity: opacity,
		})
	}
	targets, skip := intersectionOfStepFactions(steps)
	return types.SeriesDraft{
		Name:             old.Name,
		TargetFactionIDs: targets,
		Steps:            steps,
	}, skip, nil
}

func intersectionOfStepFactions(steps []types.StepDraft) ([]string, bool) {
	var inter map[string]struct{}
	for _, st := range steps {
		union := stepFactionSet(st)
		if inter == nil {
			inter = union
			continue
		}
		for id := range inter {
			if _, ok := union[id]; !ok {
				delete(inter, id)
			}
		}
	}
	if len(inter) == 0 {
		return []string{}, true
	}
	out := make([]string, 0, len(inter))
	for id := range inter {
		out = append(out, id)
	}
	sort.Strings(out)
	return out, false
}

func stepFactionSet(step types.StepDraft) map[string]struct{} {
	out := map[string]struct{}{}
	for _, v := range step.Variants {
		for _, id := range v.FactionIDs {
			id = strings.TrimSpace(id)
			if id != "" {
				out[id] = struct{}{}
			}
		}
	}
	return out
}

func missingTargetFactions(step types.StepDraft, targets []string) []string {
	have := stepFactionSet(step)
	var missing []string
	for _, id := range targets {
		if _, ok := have[id]; !ok {
			missing = append(missing, id)
		}
	}
	return missing
}

func uniqueNonEmpty(ids []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
