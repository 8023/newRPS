package server

import (
	"encoding/json"
	"fmt"
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

func decodeStepDraft(raw string) (types.StepDraft, error) {
	var d types.StepDraft
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return types.StepDraft{}, fmt.Errorf("任务投稿格式错误")
	}
	return validateStepDraft(d)
}

func decodeSeriesDraft(raw string) (types.SeriesDraft, error) {
	var draft types.SeriesDraft
	if err := json.Unmarshal([]byte(raw), &draft); err != nil {
		return types.SeriesDraft{}, fmt.Errorf("系列投稿格式错误")
	}
	draft.Name = strings.TrimSpace(draft.Name)
	if draft.Name == "" || len(draft.Steps) == 0 {
		return types.SeriesDraft{}, fmt.Errorf("系列名称和步骤不能为空")
	}
	if len([]rune(draft.Name)) > maxSeriesNameRunes {
		return types.SeriesDraft{}, fmt.Errorf("系列名称不能超过 %d 个字符", maxSeriesNameRunes)
	}
	if len(draft.Steps) > maxSeriesSteps {
		return types.SeriesDraft{}, fmt.Errorf("系列步骤过多")
	}
	for i, st := range draft.Steps {
		validated, err := validateStepDraft(st)
		if err != nil {
			return types.SeriesDraft{}, fmt.Errorf("第 %d 步：%w", i+1, err)
		}
		draft.Steps[i] = validated
	}
	draft.TargetFactionIDs = uniqueNonEmpty(draft.TargetFactionIDs)
	if len(draft.TargetFactionIDs) > maxContributionRefs {
		return types.SeriesDraft{}, fmt.Errorf("系列目标阵营过多")
	}
	if len(draft.TargetFactionIDs) == 0 {
		return types.SeriesDraft{}, fmt.Errorf("系列至少需要选择一个目标阵营")
	}
	for i, step := range draft.Steps {
		if missing := missingTargetFactions(step, draft.TargetFactionIDs); len(missing) > 0 {
			return types.SeriesDraft{}, fmt.Errorf("第 %d 步未覆盖目标阵营：%s", i+1, strings.Join(missing, "、"))
		}
	}
	return draft, nil
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
