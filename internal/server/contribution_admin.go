package server

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) handleAdminContributionAction(action string, raw map[string]any) (any, error) {
	if action == "gendersGet" || action == "gendersSave" {
		if s.genderStore == nil {
			return nil, fmt.Errorf("性别与阵营存储不可用")
		}
	} else if s.contributionStore == nil {
		return nil, fmt.Errorf("共建存储不可用")
	}
	id, err := optionalStringField(raw, "id")
	if err != nil {
		return nil, err
	}
	id = strings.TrimSpace(id)
	comment, err := optionalStringField(raw, "comment")
	if err != nil {
		return nil, err
	}
	adminID, _ := raw["adminId"].(string)
	switch action {
	case "contributionList":
		status, err := optionalStringField(raw, "status")
		if err != nil {
			return nil, err
		}
		kind, err := optionalStringField(raw, "kind")
		if err != nil {
			return nil, err
		}
		status = strings.TrimSpace(status)
		kind = strings.TrimSpace(kind)
		if status != "" && !validContributionStatus(status) {
			return nil, fmt.Errorf("投稿状态无效")
		}
		if kind != "" && !validContributionKind(kind) {
			return nil, fmt.Errorf("投稿类型无效")
		}
		list, err := s.contributionStore.listByStatus(status, kind)
		if err != nil {
			return nil, err
		}
		counts, err := contributionReviewCountsMap(s.contributionStore)
		if err != nil {
			return nil, err
		}
		// 必须包成 object：protobuf Struct 不能以顶层数组作为 RPC 载荷，
		// 直接返回 []Submission 会被编成 {"_": [...]}，前端 Array.isArray 为假，列表永远是空的。
		return map[string]any{"items": list, "counts": counts}, nil
	case "contributionCounts":
		return contributionReviewCountsMap(s.contributionStore)
	case "contributionPendingOverview":
		genders, err := s.contributionStore.listStatusIn(types.ContributionKindGender, []string{
			types.ContributionPending, types.ContributionRevisionPending,
			types.ContributionUnpublishPending, types.ContributionRejected,
			types.ContributionRevisionDraft, types.ContributionRevisionRejected,
		})
		if err != nil {
			return nil, err
		}
		matrix, err := s.contributionStore.reviewCountsMatrix([]string{types.ContributionKindTask, types.ContributionKindSeries})
		if err != nil {
			return nil, err
		}
		// poolStats：当前已审批通过、实际躺在池子里的条数/组数（与「总览」里按投稿状态数的
		// 计数是两回事——那边数的是投稿流水，这里数的是任务池/系列表本身的行数），供管理员
		// 一眼看出内容库规模。口径与数据分析「随机任务」「系列任务」增长图一致，见
		// punishmentStore.randomTaskGroupCount / seriesGroupCount。
		poolStats := map[string]int{}
		if s.punishmentStore != nil {
			if n, err := s.punishmentStore.randomTaskGroupCount(); err == nil {
				poolStats["randomTasks"] = n
			}
			if n, err := s.punishmentStore.seriesGroupCount(); err == nil {
				poolStats["series"] = n
			}
		}
		return map[string]any{"genders": genders, "counts": matrix, "poolStats": poolStats}, nil
	case "contributionUpdateComment":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := validateReviewComment(comment); err != nil {
			return nil, err
		}
		return nil, s.contributionStore.updateReviewComment(id, comment)
	case "contributionRevertReject":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		return nil, s.contributionStore.revertRejection(id)
	case "contributionGet":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		sub, err := s.contributionStore.get(id)
		if err != nil {
			return nil, err
		}
		ver, err := s.contributionStore.getVersion(id, sub.ActiveVersion)
		if err != nil {
			return nil, err
		}
		st, _ := s.contributionStore.submissionVoteAggregate(sub.ID, sub.PublishedVersion)
		var completion *types.SeriesCompletionStats
		if sub.Kind == types.ContributionKindSeries {
			completion = s.seriesCompletionStats(sub.PublishedTargetID, sub.PublishedVersion)
		}
		return map[string]any{"submission": sub, "version": ver, "votes": st, "completion": completion}, nil
	case "contributionReject":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := validateReviewComment(comment); err != nil {
			return nil, err
		}
		return nil, s.contributionStore.reject(id, adminID, comment)
	case "contributionPublish", "contributionReview":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := validateReviewComment(comment); err != nil {
			return nil, err
		}
		reviewed, err := optionalStringField(raw, "reviewedContent")
		if err != nil {
			return nil, fmt.Errorf("审核内容格式错误")
		}
		if reviewed == "" {
			if obj, ok := raw["reviewed"].(map[string]any); ok {
				b, err := json.Marshal(obj)
				if err != nil {
					return nil, fmt.Errorf("审核内容格式错误")
				}
				reviewed = string(b)
			} else if _, exists := raw["reviewed"]; exists {
				return nil, fmt.Errorf("审核内容格式错误")
			}
		}
		// 只有性别发布会改变玩家档案解析结果。任务/系列审批是低频管理操作，但不该因此
		// 在全局游戏锁内遍历“全部玩家 × 全部房间”刷新座位快照。
		sub, err := s.contributionStore.get(id)
		if err != nil {
			return nil, err
		}
		if err := s.publishContribution(id, adminID, reviewed, comment); err != nil {
			return nil, err
		}
		if sub.Kind == types.ContributionKindGender {
			s.refreshAllPlayersForConfig()
		}
		s.emitConfigUpdate()
		s.broadcastLobby()
		return map[string]any{"ok": true}, nil
	case "contributionUnpublish":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := validateReviewComment(comment); err != nil {
			return nil, err
		}
		sub, err := s.contributionStore.get(id)
		if err != nil {
			return nil, err
		}
		if err := s.unpublishContribution(id, adminID, comment); err != nil {
			return nil, err
		}
		if sub.Kind == types.ContributionKindGender {
			s.refreshAllPlayersForConfig()
		}
		s.emitConfigUpdate()
		s.broadcastLobby()
		return map[string]any{"ok": true}, nil
	case "contributionSetVoteOverride":
		kind, _ := raw["targetKind"].(string)
		tid, _ := raw["targetId"].(string)
		kind, tid = strings.TrimSpace(kind), strings.TrimSpace(tid)
		ver, err := intFromAny(raw["targetVersion"])
		if err != nil || ver < 1 {
			return nil, fmt.Errorf("评价版本无效")
		}
		ratio, err := intFromAny(raw["displayRatio"])
		if err != nil {
			return nil, fmt.Errorf("覆盖点赞率格式无效")
		}
		note, err := optionalStringField(raw, "note")
		if err != nil {
			return nil, err
		}
		if err := validateVoteTarget(kind, tid); err != nil {
			return nil, err
		}
		if err := validateReviewComment(note); err != nil {
			return nil, err
		}
		exists, err := s.contributionStore.voteTargetExists(kind, tid, ver)
		if err != nil {
			return nil, err
		}
		if !exists {
			return nil, errVoteInvalidTarget
		}
		return nil, s.contributionStore.setVoteOverride(kind, tid, ver, ratio, adminID, note)
	case "contributionClearVoteOverride":
		kind, _ := raw["targetKind"].(string)
		tid, _ := raw["targetId"].(string)
		kind, tid = strings.TrimSpace(kind), strings.TrimSpace(tid)
		ver, err := intFromAny(raw["targetVersion"])
		if err != nil || ver < 1 {
			return nil, fmt.Errorf("评价版本无效")
		}
		if err := validateVoteTarget(kind, tid); err != nil {
			return nil, err
		}
		return nil, s.contributionStore.clearVoteOverride(kind, tid, ver)
	case "gendersGet":
		return map[string]any{"genders": s.cfg.Genders, "factions": s.cfg.GenderFactions}, nil
	case "gendersSave":
		return s.adminSaveGenders(raw)
	default:
		// 路由到这里的都是前缀匹配 "contribution*" 或 genders* 的动作；真正未识别的动作
		// 报错而不是悄悄返回成功，避免未来同前缀但语义无关的新动作被这里吞掉。
		return nil, fmt.Errorf("未知的后台动作：%s", action)
	}
}

// seriesCompletionStats 读一个系列版本的完成率统计——各玩家"已走步数/总步数"百分比样本的
// 算术平均（见 punishmentstore.go 的 punishment_series_run_stats）。store 不可用、seriesID
// 为空（该投稿还没真正发布过）、或查询出错都返回 nil，调用方按"没有完成率数据"处理，
// 不当错误传播。
func (s *Server) seriesCompletionStats(seriesID string, version int) *types.SeriesCompletionStats {
	if s.punishmentStore == nil || seriesID == "" {
		return nil
	}
	participants, percentSum, err := s.punishmentStore.seriesRunStats(seriesID, version)
	if err != nil {
		return nil
	}
	stats := &types.SeriesCompletionStats{Participants: participants}
	if participants > 0 {
		rate := int(float64(percentSum)/float64(participants) + 0.5)
		stats.Rate = &rate
	}
	return stats
}

func contributionReviewCountsMap(cs *contributionStore) (map[string]any, error) {
	pending, revision, unpublish, err := cs.reviewQueueCounts()
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"pending":          pending,
		"revisionPending":  revision,
		"unpublishPending": unpublish,
	}, nil
}

func intFromAny(v any) (int, error) {
	switch n := v.(type) {
	case float64:
		if math.IsNaN(n) || math.IsInf(n, 0) || math.Trunc(n) != n || n > float64(math.MaxInt) || n < float64(math.MinInt) {
			return 0, fmt.Errorf("整数格式无效")
		}
		return int(n), nil
	case int:
		return n, nil
	default:
		return 0, fmt.Errorf("整数格式无效")
	}
}

func optionalStringField(raw map[string]any, key string) (string, error) {
	v, exists := raw[key]
	if !exists || v == nil {
		return "", nil
	}
	s, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("%s 格式无效", key)
	}
	return s, nil
}

func validContributionStatus(status string) bool {
	switch status {
	case types.ContributionDraft, types.ContributionPending, types.ContributionApproved,
		types.ContributionRejected, types.ContributionWithdrawn, types.ContributionRevisionPending,
		types.ContributionRevisionDraft, types.ContributionRevisionRejected, types.ContributionUnpublishPending:
		return true
	default:
		return false
	}
}

func validateAdminContributionID(id string) error {
	if id == "" || len([]rune(id)) > maxContributionIDRunes {
		return fmt.Errorf("投稿 ID 无效")
	}
	return nil
}

func validateReviewComment(comment string) error {
	if len([]rune(comment)) > maxReviewCommentRunes {
		return fmt.Errorf("审核批注不能超过 %d 个字符", maxReviewCommentRunes)
	}
	return nil
}

func validateVoteTarget(kind, id string) error {
	if kind != types.ContributionKindTask || id == "" || len([]rune(id)) > maxContributionIDRunes {
		return errVoteInvalidTarget
	}
	return nil
}

func (s *Server) adminSaveGenders(raw map[string]any) (any, error) {
	b, err := json.Marshal(raw)
	if err != nil {
		return nil, err
	}
	var in struct {
		Genders  []types.GenderOption  `json:"genders"`
		Factions []types.GenderFaction `json:"factions"`
	}
	if err := json.Unmarshal(b, &in); err != nil {
		return nil, err
	}
	if len(in.Factions) > 100 || len(in.Genders) > 500 {
		return nil, fmt.Errorf("性别或阵营数量过多")
	}
	for i := range in.Factions {
		in.Factions[i].ID = strings.TrimSpace(in.Factions[i].ID)
		in.Factions[i].Label = strings.TrimSpace(in.Factions[i].Label)
		in.Factions[i].TaskGroup = strings.TrimSpace(in.Factions[i].TaskGroup)
		if len([]rune(in.Factions[i].ID)) > 64 || len([]rune(in.Factions[i].Label)) > maxGenderLabelRunes {
			return nil, fmt.Errorf("阵营 ID 或名称过长")
		}
		if containsControlRune(in.Factions[i].ID) || containsControlRune(in.Factions[i].Label) {
			return nil, fmt.Errorf("阵营 ID 或名称包含控制字符")
		}
	}
	seenLabels := map[string]struct{}{}
	for i := range in.Genders {
		in.Genders[i].ID = strings.TrimSpace(in.Genders[i].ID)
		in.Genders[i].Label = strings.TrimSpace(in.Genders[i].Label)
		in.Genders[i].FactionID = strings.TrimSpace(in.Genders[i].FactionID)
		if len([]rune(in.Genders[i].ID)) > 64 || len([]rune(in.Genders[i].Label)) > maxGenderLabelRunes {
			return nil, fmt.Errorf("性别 ID 或名称过长")
		}
		if containsControlRune(in.Genders[i].ID) || containsControlRune(in.Genders[i].Label) || containsControlRune(in.Genders[i].FactionID) {
			return nil, fmt.Errorf("性别字段包含控制字符")
		}
		norm := normalizeGenderLabel(in.Genders[i].Label)
		if norm == "" {
			return nil, fmt.Errorf("性别名称包含无效字符")
		}
		key := in.Genders[i].FactionID + "\x00" + norm
		if _, exists := seenLabels[key]; exists {
			return nil, fmt.Errorf("同一阵营不能有重名性别")
		}
		seenLabels[key] = struct{}{}
	}
	cfg := s.cfg
	cfg.Genders = in.Genders
	cfg.GenderFactions = in.Factions
	if err := config.ValidateGenders(cfg); err != nil {
		return nil, err
	}
	if err := s.genderStore.replaceAll(in.Factions, in.Genders); err != nil {
		return nil, err
	}
	s.reloadGenderCaches()
	s.refreshAllPlayersForConfig()
	s.emitConfigUpdate()
	s.broadcastLobby()
	return map[string]any{"genders": s.cfg.Genders, "factions": s.cfg.GenderFactions}, nil
}
