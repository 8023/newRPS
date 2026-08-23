package server

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// maxGenderLabelRunes：性别/阵营 ID、名称的字符数上限（adminSaveGenders 用）。
const maxGenderLabelRunes = 40

// handleAdminContributionAction 是后台"共建审核"（action 前缀 "contribution"）+「性别与
// 阵营」批量管理（action 为 "gendersGet"/"gendersSave"，与共建投稿系统无关，见
// genderstore.go 顶部注释）共用的管理员操作入口。共建审核只保留批准/驳回/下架/取消撤回/
// 批注这些审核类操作，内容本身（新增/修改文案）一律走"玩家投稿"，没有绕过投稿记录直接
// 增删改任务池/系列表的入口。
func (s *Server) handleAdminContributionAction(action string, raw map[string]any) (any, error) {
	if action == "gendersGet" || action == "gendersSave" {
		if s.genderStore == nil {
			return nil, fmt.Errorf("性别与阵营存储不可用")
		}
	} else if s.contributionStore == nil {
		return nil, fmt.Errorf("共建存储不可用")
	}
	if action == "gendersGet" {
		return map[string]any{"genders": s.cfg.Genders, "factions": s.cfg.GenderFactions}, nil
	}
	if action == "gendersSave" {
		return s.adminSaveGenders(raw)
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
	if err := validateReviewComment(comment); err != nil {
		return nil, err
	}
	kind, err := optionalStringField(raw, "kind")
	if err != nil {
		return nil, err
	}
	kind = strings.TrimSpace(kind)
	adminID, _ := raw["adminId"].(string)
	if strings.HasPrefix(action, "contribution") && action != "contributionList" && action != "contributionPendingOverview" {
		// kind 为空继续按历史接口解释为 task；显式传入未知值必须拒绝，不能落入 task 分支。
		if kind == "" {
			kind = types.ContributionKindTask
		} else if !validContributionKind(kind) {
			return nil, fmt.Errorf("投稿类型无效")
		}
	}
	switch action {
	case "contributionList":
		status, err := optionalStringField(raw, "status")
		if err != nil {
			return nil, err
		}
		status = strings.TrimSpace(status)
		if status != "" && !validContributionStatus(status) {
			return nil, fmt.Errorf("投稿状态无效")
		}
		if kind != "" && !validContributionKind(kind) {
			return nil, fmt.Errorf("投稿类型无效")
		}
		items, err := s.adminListContributions(kind, status)
		if err != nil {
			return nil, err
		}
		// 已通过投稿的点赞率/系列完成率一次性算好随列表下发，前端不必再对每条已通过
		// 投稿单独发一次 contributionGet——审核队列条数一多，逐条请求很容易在几秒内
		// 撞上 admin:action 的速率限制（见 AdminContributionReview.tsx 的 useEffect 注释）。
		if err := s.fillContributionListMetaWithoutServerLock(items); err != nil {
			return nil, err
		}
		// 必须包成 object：protobuf Struct 不能以顶层数组作为 RPC 载荷。
		return map[string]any{"items": items}, nil
	case "contributionPendingOverview":
		matrix, poolStats, err := s.contributionPendingOverview()
		if err != nil {
			return nil, err
		}
		return map[string]any{"counts": matrix, "poolStats": poolStats}, nil
	case "contributionGet":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if kind == "" {
			kind = types.ContributionKindTask
		}
		item, err := s.contributionStore.get(kind, id)
		if err != nil {
			return nil, err
		}
		one := []types.ContributionItem{item}
		if err := s.fillContributionListMetaWithoutServerLock(one); err != nil {
			return nil, err
		}
		one[0].SubmitterName = s.playerName(one[0].SubmitterID)
		return one[0], nil
	case "contributionUpdateComment":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := s.adminUpdateComment(kind, id, comment); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "contributionReject":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := s.adminReject(kind, id, adminID, comment); err != nil {
			return nil, err
		}
		s.afterContributionChange()
		return map[string]any{"ok": true}, nil
	case "contributionRevertReject":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := s.adminSetStatus(kind, id, types.ContributionPending, adminID, comment); err != nil {
			return nil, err
		}
		return map[string]any{"ok": true}, nil
	case "contributionPublish", "contributionReview":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		reviewedContent, err := optionalStringField(raw, "reviewedContent")
		if err != nil {
			return nil, fmt.Errorf("审核内容格式错误")
		}
		if err := s.adminApprove(kind, id, adminID, comment, reviewedContent); err != nil {
			return nil, err
		}
		s.afterContributionChange()
		return map[string]any{"ok": true}, nil
	case "contributionUnpublish":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := s.adminSetStatus(kind, id, types.ContributionWithdrawn, adminID, comment); err != nil {
			return nil, err
		}
		s.afterContributionChange()
		return map[string]any{"ok": true}, nil
	case "contributionRevertWithdraw":
		if err := validateAdminContributionID(id); err != nil {
			return nil, err
		}
		if err := s.validateContributionForApproval(kind, id, ""); err != nil {
			return nil, err
		}
		if err := s.adminSetStatus(kind, id, types.ContributionApproved, adminID, comment); err != nil {
			return nil, err
		}
		s.afterContributionChange()
		return map[string]any{"ok": true}, nil
	default:
		return nil, fmt.Errorf("未知的后台动作：%s", action)
	}
}

func (s *Server) afterContributionChange() {
	s.reloadPunishmentCaches()
	s.emitConfigUpdate()
	s.broadcastLobby()
}

// adminListContributions 返回后台审核列表：kind 为空时任务+系列都列出。
func (s *Server) adminListContributions(kind, status string) ([]types.ContributionItem, error) {
	var out []types.ContributionItem
	if kind == "" || kind == types.ContributionKindTask {
		rows, err := s.contributionStore.tasks.listAdmin(status)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			if r.SeriesID != "" {
				continue // 系列步骤不在这个列表单独出现，跟着系列一起编辑/展示
			}
			out = append(out, subTaskToItem(r, false))
		}
	}
	if kind == "" || kind == types.ContributionKindSeries {
		rows, err := s.contributionStore.series.listAdmin(status)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, seriesToItem(r, nil, false))
		}
	}
	// 昵称不做快照，统一按 SubmitterID 现查当前显示名（与聊天板块的既有约定一致，
	// 见 subTaskToItem/seriesToItem 顶部注释）。
	for i := range out {
		out[i].SubmitterName = s.playerName(out[i].SubmitterID)
	}
	return out, nil
}

// fillContributionListMeta 就地补全列表里已通过投稿的点赞数与系列完成率。任务和系列都按
// ID 批量查询，复杂度由原来的“每条任务 1 次、每个系列 2 次”降为每 400 条各 1 次；系列
// 每一步的赞踩会跨该步骤的所有历史版本汇总，避免改稿后旧评价从列表里消失。
func (s *Server) fillContributionListMeta(items []types.ContributionItem) error {
	var taskIDs, seriesIDs []string
	for i := range items {
		if items[i].Status != types.ContributionApproved {
			continue
		}
		switch items[i].Kind {
		case types.ContributionKindTask:
			taskIDs = append(taskIDs, items[i].ID)
		case types.ContributionKindSeries:
			seriesIDs = append(seriesIDs, items[i].ID)
		}
	}

	readDB := s.contributionStore.db
	if s.activityRO != nil {
		readDB = s.activityRO
	}
	readTasks := newSubTaskStore(readDB)
	taskVotes, err := readTasks.voteAggregates(taskIDs)
	if err != nil {
		return err
	}
	seriesVotes, err := readTasks.seriesVoteAggregates(seriesIDs)
	if err != nil {
		return err
	}
	seriesRuns := map[string]seriesRunCounts{}
	if s.punishmentStore != nil {
		seriesRuns, err = newPunishmentStore(readDB).currentSeriesRunStats(seriesIDs)
		if err != nil {
			return err
		}
	}

	for i := range items {
		it := &items[i]
		if it.Status != types.ContributionApproved {
			continue
		}
		var counts contributionVoteCounts
		if it.Kind == types.ContributionKindTask {
			counts = taskVotes[it.ID]
		} else if it.Kind == types.ContributionKindSeries {
			counts = seriesVotes[it.ID]
			run := seriesRuns[it.ID]
			stats := &types.SeriesCompletionStats{Participants: run.Participants}
			if run.Participants > 0 {
				rate := int(float64(run.PercentSum)/float64(run.Participants) + 0.5)
				stats.Rate = &rate
			}
			it.Completion = stats
		}
		it.LikeCount, it.DownCount = counts.Likes, counts.Downs
	}
	return nil
}

// fillContributionListMetaWithoutServerLock 供持有 s.mu 的 WebSocket handler 调用。评价与
// 系列完成率是后台列表的批量只读查询，必须使用独立连接并在 SQL 期间释放全局状态锁；回锁
// 后 items 仍是调用方的局部切片，不需要重新读取内存状态。
func (s *Server) fillContributionListMetaWithoutServerLock(items []types.ContributionItem) error {
	s.mu.Unlock()
	defer s.mu.Lock()
	return s.fillContributionListMeta(items)
}

// contributionPendingOverview 统计任务/系列两种类型各状态的投稿条数（后台"待处理"总览用），
// 以及当前已发布的池子规模。
func (s *Server) contributionPendingOverview() (map[string]any, map[string]int, error) {
	statuses := []string{types.ContributionPending, types.ContributionApproved, types.ContributionRejected, types.ContributionWithdrawn}
	matrix := map[string]any{}
	for _, kind := range []string{types.ContributionKindTask, types.ContributionKindSeries} {
		row := map[string]int{}
		for _, st := range statuses {
			row[st] = 0
		}
		items, err := s.adminListContributions(kind, "")
		if err != nil {
			return nil, nil, err
		}
		for _, it := range items {
			if _, ok := row[it.Status]; ok {
				row[it.Status]++
			}
		}
		matrix[kind] = row
	}
	poolStats := map[string]int{}
	if n, err := s.contributionStore.tasks.approvedRandomPoolCount(); err == nil {
		poolStats["randomTasks"] = n
	}
	if n, err := s.contributionStore.series.approvedSeriesCount(); err == nil {
		poolStats["series"] = n
	}
	return matrix, poolStats, nil
}

func (s *Server) adminUpdateComment(kind, id, comment string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	now := nowMs()
	var res interface{ RowsAffected() (int64, error) }
	switch kind {
	case types.ContributionKindSeries:
		r, err := tx.Exec(`UPDATE series SET review_comment=?, updated_at=? WHERE id=? AND version=(SELECT MAX(version) FROM series WHERE id=?)`, comment, now, id, id)
		if err != nil {
			return err
		}
		res = r
	case types.ContributionKindTask:
		r, err := tx.Exec(`UPDATE sub_tasks SET review_comment=?, updated_at=?
			WHERE id=? AND series_id='' AND active=1 AND version=(SELECT MAX(version) FROM sub_tasks WHERE id=?)`, comment, now, id, id)
		if err != nil {
			return err
		}
		res = r
	default:
		return fmt.Errorf("投稿类型无效")
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errContributionNotFound
	}
	return tx.Commit()
}

func (s *Server) adminReject(kind, id, adminID, comment string) error {
	return s.adminSetStatus(kind, id, types.ContributionRejected, adminID, comment)
}

// adminSetStatus 是所有"纯状态流转"型管理员操作的共用实现：驳回/撤销驳回/下架/取消撤回
// 本质都只是把某个 id 最新那一行的 status 改掉，系列还要连带它当前的步骤一起改。
func (s *Server) adminSetStatus(kind, id, status, adminID, comment string) error {
	expected := map[string]string{
		types.ContributionRejected:  types.ContributionPending,
		types.ContributionPending:   types.ContributionRejected,
		types.ContributionWithdrawn: types.ContributionApproved,
		types.ContributionApproved:  types.ContributionWithdrawn,
	}[status]
	if expected == "" {
		return errContributionBadStatus
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	switch kind {
	case types.ContributionKindSeries:
		row, err := s.contributionStore.series.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.Status != expected {
			return errContributionBadStatus
		}
		if err := s.contributionStore.series.setStatus(tx, id, status, adminID, comment); err != nil {
			return err
		}
		stepIDs, err := seriesStepIDsTx(tx, id)
		if err != nil {
			return err
		}
		for _, sid := range stepIDs {
			if err := s.contributionStore.tasks.setStatus(tx, sid, status, adminID, comment); err != nil {
				return err
			}
		}
	case types.ContributionKindTask:
		row, err := s.contributionStore.tasks.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.SeriesID != "" || !row.Active {
			return errContributionNotFound
		}
		if row.Status != expected {
			return errContributionBadStatus
		}
		if err := s.contributionStore.tasks.setStatus(tx, id, status, adminID, comment); err != nil {
			return err
		}
	default:
		return fmt.Errorf("投稿类型无效")
	}
	return tx.Commit()
}

// adminApprove 批准一份投稿：reviewedContent 非空时视为管理员就地改稿后批准
// （插入一个新版本、内容用改后的，直接以 approved 状态生效）；为空则原样批准当前最新版本
// （纯状态流转，不产生新版本）。
func (s *Server) validateContributionForApproval(kind, id, reviewedContent string) error {
	raw := strings.TrimSpace(reviewedContent)
	if raw == "" {
		item, err := s.contributionStore.get(kind, id)
		if err != nil {
			return err
		}
		raw, err = marshalDraft(item.Content)
		if err != nil {
			return err
		}
	}
	// 玩家保存草稿时 marshalDraft 已经限制为 maxContributionJSON；管理员就地改稿走的是
	// reviewedContent 字符串，不能绕过同一上限。否则攻击者一旦拿到后台会话，就能构造一个
	// 远大于正常表单的 JSON 字符串，让后续反复 Unmarshal/校验占用大量内存和 CPU。
	if len(raw) > maxContributionJSON {
		return fmt.Errorf("审核内容过大")
	}
	if err := s.contributionStore.validateOwnedDraft(kind, raw); err != nil {
		return err
	}
	if err := s.validateContributionRefs(kind, raw); err != nil {
		return err
	}
	if kind == types.ContributionKindSeries {
		if err := s.validateSeriesMinSteps(raw); err != nil {
			return err
		}
		if err := s.validateSeriesMaxSteps(raw); err != nil {
			return err
		}
	}
	return nil
}

func (s *Server) adminApprove(kind, id, adminID, comment, reviewedContent string) error {
	if err := s.validateContributionForApproval(kind, id, reviewedContent); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	switch kind {
	case types.ContributionKindSeries:
		prev, err := s.contributionStore.series.latestTx(tx, id)
		if err != nil {
			return err
		}
		// 待审：正常审核批准，reviewedContent 可选（原样批准或就地改稿批准）。
		// 已通过：管理员是终审者，改稿直接生效、不用退回待审再走一遍——但必须带新内容，
		// 否则状态不用变、又没有新内容，等于什么都没做。
		switch prev.Status {
		case types.ContributionPending:
		case types.ContributionApproved:
			if strings.TrimSpace(reviewedContent) == "" {
				return errContributionBadStatus
			}
		default:
			return errContributionBadStatus
		}
		if strings.TrimSpace(reviewedContent) != "" {
			if prev.Version >= maxContributionVersionsPerItem {
				return fmt.Errorf("该投稿版本数量已达上限，不能由审核改稿再新增版本")
			}
			draft, err := decodeSeriesDraft(reviewedContent)
			if err != nil {
				return err
			}
			seriesRow, err := s.contributionStore.series.insertVersion(tx, id, draft.Name, draft.TargetFactionIDs,
				types.ContributionApproved, prev.ContributorPlayerID, prev.ContributorAnonymous, adminID, comment)
			if err != nil {
				return err
			}
			if _, err := s.contributionStore.saveSeriesSteps(tx, seriesRow.ID, prev.ContributorPlayerID, prev.ContributorAnonymous, draft.Steps); err != nil {
				return err
			}
		} else if err := s.contributionStore.series.setStatus(tx, id, types.ContributionApproved, adminID, comment); err != nil {
			return err
		}
		stepIDs, err := seriesStepIDsTx(tx, id)
		if err != nil {
			return err
		}
		for _, sid := range stepIDs {
			if err := s.contributionStore.tasks.setStatus(tx, sid, types.ContributionApproved, adminID, comment); err != nil {
				return err
			}
		}
	case types.ContributionKindTask:
		prev, err := s.contributionStore.tasks.latestTx(tx, id)
		if err != nil {
			return err
		}
		if prev.SeriesID != "" || !prev.Active {
			return errContributionNotFound
		}
		switch prev.Status {
		case types.ContributionPending:
		case types.ContributionApproved:
			if strings.TrimSpace(reviewedContent) == "" {
				return errContributionBadStatus
			}
		default:
			return errContributionBadStatus
		}
		if strings.TrimSpace(reviewedContent) != "" {
			if prev.Version >= maxContributionVersionsPerItem {
				return fmt.Errorf("该投稿版本数量已达上限，不能由审核改稿再新增版本")
			}
			draft, err := decodeStepDraft(reviewedContent)
			if err != nil {
				return err
			}
			if _, err := s.contributionStore.tasks.insertVersion(tx, id, "", 0, draft,
				types.ContributionApproved, prev.ContributorPlayerID, prev.ContributorAnonymous, adminID, comment); err != nil {
				return err
			}
		} else if err := s.contributionStore.tasks.setStatus(tx, id, types.ContributionApproved, adminID, comment); err != nil {
			return err
		}
	default:
		return fmt.Errorf("投稿类型无效")
	}
	return tx.Commit()
}

func validContributionStatus(status string) bool {
	switch status {
	case types.ContributionDraft, types.ContributionPending, types.ContributionApproved,
		types.ContributionRejected, types.ContributionWithdrawn:
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

// sanitizeGenderInput 对性别/阵营批量提交做统一的清洗与校验（trim、长度上限、控制字符、
// 同阵营重名去重、结构校验），供 gendersSave（admin:action）与 config:save 两条写入路径
// 共用，避免两条路径的校验规则跑偏。返回清洗后的副本，不修改入参。
func (s *Server) sanitizeGenderInput(genders []types.GenderOption, factions []types.GenderFaction) ([]types.GenderOption, []types.GenderFaction, error) {
	if len(factions) > 100 || len(genders) > 500 {
		return nil, nil, fmt.Errorf("性别或阵营数量过多")
	}
	factions = append([]types.GenderFaction(nil), factions...)
	genders = append([]types.GenderOption(nil), genders...)
	for i := range factions {
		factions[i].ID = strings.TrimSpace(factions[i].ID)
		factions[i].Label = strings.TrimSpace(factions[i].Label)
		factions[i].TaskGroup = strings.TrimSpace(factions[i].TaskGroup)
		if len([]rune(factions[i].ID)) > 64 || len([]rune(factions[i].Label)) > maxGenderLabelRunes {
			return nil, nil, fmt.Errorf("阵营 ID 或名称过长")
		}
		if containsControlRune(factions[i].ID) || containsControlRune(factions[i].Label) {
			return nil, nil, fmt.Errorf("阵营 ID 或名称包含控制字符")
		}
	}
	seenLabels := map[string]struct{}{}
	for i := range genders {
		genders[i].ID = strings.TrimSpace(genders[i].ID)
		genders[i].Label = strings.TrimSpace(genders[i].Label)
		genders[i].FactionID = strings.TrimSpace(genders[i].FactionID)
		if len([]rune(genders[i].ID)) > 64 || len([]rune(genders[i].Label)) > maxGenderLabelRunes {
			return nil, nil, fmt.Errorf("性别 ID 或名称过长")
		}
		if containsControlRune(genders[i].ID) || containsControlRune(genders[i].Label) || containsControlRune(genders[i].FactionID) {
			return nil, nil, fmt.Errorf("性别字段包含控制字符")
		}
		norm := normalizeGenderLabel(genders[i].Label)
		if norm == "" {
			return nil, nil, fmt.Errorf("性别名称包含无效字符")
		}
		key := genders[i].FactionID + "\x00" + norm
		if _, exists := seenLabels[key]; exists {
			return nil, nil, fmt.Errorf("同一阵营不能有重名性别")
		}
		seenLabels[key] = struct{}{}
	}
	cfg := s.cfg
	cfg.Genders = genders
	cfg.GenderFactions = factions
	if err := config.ValidateGenders(cfg); err != nil {
		return nil, nil, err
	}
	return genders, factions, nil
}

// persistGendersAndFactions 落库（整表覆盖）并刷新运行时缓存/s.cfg，调用前须先经过
// sanitizeGenderInput 校验。同样供 gendersSave 与 config:save 共用。
func (s *Server) persistGendersAndFactions(genders []types.GenderOption, factions []types.GenderFaction) error {
	if s.genderStore == nil {
		return fmt.Errorf("性别与阵营存储不可用")
	}
	if err := s.genderStore.replaceAll(factions, genders); err != nil {
		return err
	}
	s.reloadGenderCaches()
	return nil
}

// adminSaveGenders 是后台「性别与阵营」板块的直接批量增删改（与共建投稿系统无关，见
// genderstore.go 顶部注释）：整份覆盖 gender_factions/gender_options 两张表。
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
	genders, factions, err := s.sanitizeGenderInput(in.Genders, in.Factions)
	if err != nil {
		return nil, err
	}
	if err := s.persistGendersAndFactions(genders, factions); err != nil {
		return nil, err
	}
	s.refreshAllPlayersForConfig()
	s.emitConfigUpdate()
	s.broadcastLobby()
	return map[string]any{"genders": s.cfg.Genders, "factions": s.cfg.GenderFactions}, nil
}
