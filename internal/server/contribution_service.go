package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// contentToStepDraft/contentToSeriesDraft：RPC 的 content 字段是 google.protobuf.Struct
// 解出来的 any，先重新序列化成 JSON 再按目标形状解析+校验，与 saveDraft 的草稿路径共用
// 同一份 marshalDraft。
func marshalDraft(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	if len(b) > maxContributionJSON {
		return "", fmt.Errorf("投稿内容过大")
	}
	return string(b), nil
}

func validContributionKind(kind string) bool {
	switch kind {
	case types.ContributionKindTask, types.ContributionKindSeries:
		return true
	default:
		return false
	}
}

// toItem 把一行 sub_tasks（独立随机任务）转成对外的 ContributionItem，Content 只在
// withContent 为真时填充。
// SubmitterName 不在这里填：昵称不做快照，调用方（handlers_contribution.go/
// contribution_admin.go）在回给客户端前统一用 SubmitterID 现查 s.playerName。
func subTaskToItem(r subTaskRow, withContent bool) types.ContributionItem {
	item := types.ContributionItem{
		ID: r.ID, Kind: types.ContributionKindTask, Version: r.Version, Status: r.Status,
		Anonymous: r.ContributorAnonymous, SubmitterID: r.ContributorPlayerID,
		ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ReviewComment: r.ReviewComment,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, LikeCount: r.LikeCount, DownCount: r.DownCount,
	}
	if len(r.Draft.Variants) > 0 {
		item.Title = truncateRunes(strings.TrimSpace(r.Draft.Variants[0].Text), 24)
	}
	if withContent {
		item.Content = r.Draft
	}
	return item
}

func seriesToItem(r seriesRow, steps []subTaskRow, withContent bool) types.ContributionItem {
	item := types.ContributionItem{
		ID: r.ID, Kind: types.ContributionKindSeries, Version: r.Version, Status: r.Status,
		Anonymous: r.ContributorAnonymous, SubmitterID: r.ContributorPlayerID,
		ReviewedBy: r.ReviewedBy, ReviewedAt: r.ReviewedAt, ReviewComment: r.ReviewComment,
		CreatedAt: r.CreatedAt, UpdatedAt: r.UpdatedAt, Title: r.Name,
	}
	if withContent {
		draft := types.SeriesDraft{Name: r.Name, TargetFactionIDs: r.TargetFactionIDs, Steps: make([]types.StepDraft, len(steps))}
		for i, st := range steps {
			draft.Steps[i] = st.Draft
			item.LikeCount += st.LikeCount
			item.DownCount += st.DownCount
		}
		item.Content = draft
	}
	return item
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

// listBySubmitter 返回某玩家名下所有投稿（任务+系列）的最新版本概览，供玩家端
// "参与共建"列表使用（不带 Content，节省带宽——详情由 get 单独拉）。
func (cs *contributionStore) listBySubmitter(playerID string) ([]types.ContributionItem, error) {
	tasks, err := cs.tasks.listBySubmitter(playerID)
	if err != nil {
		return nil, err
	}
	series, err := cs.series.listBySubmitter(playerID)
	if err != nil {
		return nil, err
	}
	out := make([]types.ContributionItem, 0, len(tasks)+len(series))
	for _, t := range tasks {
		out = append(out, subTaskToItem(t, false))
	}
	for _, s := range series {
		out = append(out, seriesToItem(s, nil, false))
	}
	return out, nil
}

// get 返回一份投稿的详情（含内容、赞踩汇总、系列完成率）。
func (cs *contributionStore) get(kind, id string) (types.ContributionItem, error) {
	switch kind {
	case types.ContributionKindTask:
		r, err := cs.tasks.latest(id)
		if err != nil {
			return types.ContributionItem{}, err
		}
		if r.SeriesID != "" || !r.Active {
			return types.ContributionItem{}, errContributionNotFound
		}
		likes, downs, err := cs.tasks.voteAggregate(id)
		if err != nil {
			return types.ContributionItem{}, err
		}
		item := subTaskToItem(r, true)
		item.LikeCount, item.DownCount = likes, downs
		return item, nil
	case types.ContributionKindSeries:
		r, err := cs.series.latest(id)
		if err != nil {
			return types.ContributionItem{}, err
		}
		steps, err := cs.tasks.stepsForSeries(id)
		if err != nil {
			return types.ContributionItem{}, err
		}
		item := seriesToItem(r, steps, true)
		return item, nil
	default:
		return types.ContributionItem{}, fmt.Errorf("未知投稿类型")
	}
}

// getOwned 同 get，但要求属于 playerID。
func (cs *contributionStore) getOwned(kind, id, playerID string) (types.ContributionItem, error) {
	item, err := cs.get(kind, id)
	if err != nil {
		return item, err
	}
	if item.SubmitterID != playerID {
		return types.ContributionItem{}, errContributionForbidden
	}
	return item, nil
}

// saveDraft 保存一份草稿（新建或续editing）：kind=task 时 content 是 types.StepDraft；
// kind=series 时是 types.SeriesDraft（含全部步骤）。每次保存都插入新版本行——id 为空即
// 新建；非空则要求属于 playerID，且待审状态禁止编辑。已发布/被驳回/已撤回的投稿可以
// 继续编辑，产生的新版本 status 恒为 draft；运行时只认每个 id 的最新版本，所以开始修订后
// 旧 approved 版本会立即退出正式池，待新版本重新审核通过后恢复。
func (cs *contributionStore) saveDraft(playerID, kind string, anonymous bool, id, content string) (types.ContributionItem, error) {
	if !validContributionKind(kind) {
		return types.ContributionItem{}, fmt.Errorf("未知投稿类型")
	}
	createsDraft := true
	if id != "" {
		if kind == types.ContributionKindTask {
			row, err := cs.tasks.latestOwned(id, playerID)
			if err != nil {
				return types.ContributionItem{}, err
			}
			if row.SeriesID != "" || !row.Active {
				return types.ContributionItem{}, errContributionNotFound
			}
			if row.Version >= maxContributionVersionsPerItem {
				return types.ContributionItem{}, fmt.Errorf("该投稿版本数量已达上限，请新建投稿")
			}
			switch row.Status {
			case types.ContributionDraft, types.ContributionRejected, types.ContributionWithdrawn, types.ContributionApproved:
			default:
				return types.ContributionItem{}, errContributionBadStatus
			}
			createsDraft = row.Status != types.ContributionDraft
		} else {
			row, err := cs.series.latestOwned(id, playerID)
			if err != nil {
				return types.ContributionItem{}, err
			}
			if row.Version >= maxContributionVersionsPerItem {
				return types.ContributionItem{}, fmt.Errorf("该投稿版本数量已达上限，请新建投稿")
			}
			switch row.Status {
			case types.ContributionDraft, types.ContributionRejected, types.ContributionWithdrawn, types.ContributionApproved:
			default:
				return types.ContributionItem{}, errContributionBadStatus
			}
			createsDraft = row.Status != types.ContributionDraft
		}
	} else {
		total, err := cs.countSubmissions(playerID)
		if err != nil {
			return types.ContributionItem{}, err
		}
		if total >= maxSubmissionsPerPlayer {
			return types.ContributionItem{}, fmt.Errorf("投稿总数已达上限")
		}
	}
	if createsDraft {
		drafts, err := cs.countDrafts(playerID)
		if err != nil {
			return types.ContributionItem{}, err
		}
		if drafts >= maxDraftsPerPlayer {
			return types.ContributionItem{}, fmt.Errorf("草稿数量已达上限")
		}
	}
	tx, err := cs.db.Begin()
	if err != nil {
		return types.ContributionItem{}, err
	}
	defer func() { _ = tx.Rollback() }()

	switch kind {
	case types.ContributionKindTask:
		draft, err := parseStepDraftLenient(content)
		if err != nil {
			return types.ContributionItem{}, err
		}
		row, err := cs.tasks.insertVersion(tx, id, "", 0, draft, types.ContributionDraft, playerID, anonymous, "", "")
		if err != nil {
			return types.ContributionItem{}, err
		}
		if err := tx.Commit(); err != nil {
			return types.ContributionItem{}, err
		}
		return subTaskToItem(row, true), nil
	default:
		draft, err := parseSeriesDraftLenient(content)
		if err != nil {
			return types.ContributionItem{}, err
		}
		if len(draft.Steps) > maxSeriesSteps {
			return types.ContributionItem{}, fmt.Errorf("系列步骤过多")
		}
		seriesRow, err := cs.series.insertVersion(tx, id, draft.Name, draft.TargetFactionIDs, types.ContributionDraft, playerID, anonymous, "", "")
		if err != nil {
			return types.ContributionItem{}, err
		}
		steps, err := cs.saveSeriesSteps(tx, seriesRow.ID, playerID, anonymous, draft.Steps)
		if err != nil {
			return types.ContributionItem{}, err
		}
		if err := tx.Commit(); err != nil {
			return types.ContributionItem{}, err
		}
		return seriesToItem(seriesRow, steps, true), nil
	}
}

// saveSeriesSteps 把 SeriesDraft.Steps 落成 sub_tasks 行：位置 i 的内容延续该系列此前
// 位置 i 那一步的 ID（保持版本连续），超出旧步数的新增步骤各自分配新 ID；本次没有
// 出现的旧步骤（系列被缩短）写入 inactive 墓碑版本：历史内容仍完整保留，但不会再被
// 当前成员查询、提交、审批、缓存或图片权限逻辑选中。
func (cs *contributionStore) saveSeriesSteps(tx *sql.Tx, seriesID, playerID string, anonymous bool, steps []types.StepDraft) ([]subTaskRow, error) {
	prevIDs, err := seriesStepIDsTx(tx, seriesID)
	if err != nil {
		return nil, err
	}
	out := make([]subTaskRow, 0, len(steps))
	for i, step := range steps {
		stepID := ""
		if i < len(prevIDs) {
			stepID = prevIDs[i]
		}
		row, err := cs.tasks.insertVersion(tx, stepID, seriesID, i, step, types.ContributionDraft, playerID, anonymous, "", "")
		if err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	if len(prevIDs) > len(steps) {
		for _, removedID := range prevIDs[len(steps):] {
			if err := cs.tasks.deactivateVersion(tx, removedID); err != nil {
				return nil, err
			}
		}
	}
	return out, nil
}

// seriesStepIDsTx 返回某系列当前各步骤（按 step_index 排序）的最新行 ID，供
// saveSeriesSteps 延续同一步骤的版本历史。
func seriesStepIDsTx(tx *sql.Tx, seriesID string) ([]string, error) {
	if seriesID == "" {
		return nil, nil
	}
	rows, err := tx.Query(`
		SELECT t.id FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks WHERE series_id = ? GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.series_id = ? AND t.active = 1
		ORDER BY t.step_index ASC`, seriesID, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, rows.Err()
}

func (cs *contributionStore) countSubmissions(playerID string) (int, error) {
	var n int
	err := cs.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM sub_tasks t INNER JOIN (SELECT id, MAX(version) v FROM sub_tasks WHERE contributor_player_id=? GROUP BY id) m ON t.id=m.id AND t.version=m.v WHERE t.series_id='' AND t.active=1) +
		(SELECT COUNT(DISTINCT id) FROM series WHERE contributor_player_id=?)`, playerID, playerID).Scan(&n)
	return n, err
}

func (cs *contributionStore) countDrafts(playerID string) (int, error) {
	var n int
	err := cs.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM sub_tasks t INNER JOIN (SELECT id, MAX(version) v FROM sub_tasks WHERE contributor_player_id=? GROUP BY id) m ON t.id=m.id AND t.version=m.v WHERE t.series_id='' AND t.active=1 AND t.status='draft') +
		(SELECT COUNT(*) FROM series t INNER JOIN (SELECT id, MAX(version) v FROM series WHERE contributor_player_id=? GROUP BY id) m ON t.id=m.id AND t.version=m.v WHERE t.status='draft')`,
		playerID, playerID).Scan(&n)
	return n, err
}

// submit 把一份草稿从 draft 推进到 pending（提交审批）；系列同时把它名下所有步骤
// 也一并推进（步骤本身没有独立的提交动作，跟系列一起走）。
func (cs *contributionStore) submit(kind, playerID, id string) error {
	pending, err := cs.countPending(playerID)
	if err != nil {
		return err
	}
	if pending >= maxPendingPerPlayer {
		return fmt.Errorf("待审投稿数量已达上限")
	}
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	switch kind {
	case types.ContributionKindTask:
		row, err := cs.tasks.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.SeriesID != "" || !row.Active {
			return errContributionNotFound
		}
		if row.ContributorPlayerID != playerID || row.Status != types.ContributionDraft {
			return errContributionBadStatus
		}
		if err := cs.tasks.setStatus(tx, id, types.ContributionPending, "", ""); err != nil {
			return err
		}
	case types.ContributionKindSeries:
		row, err := cs.series.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.ContributorPlayerID != playerID || row.Status != types.ContributionDraft {
			return errContributionBadStatus
		}
		if err := cs.series.setStatus(tx, id, types.ContributionPending, "", ""); err != nil {
			return err
		}
		stepIDs, err := seriesStepIDsTx(tx, id)
		if err != nil {
			return err
		}
		for _, sid := range stepIDs {
			if err := cs.tasks.setStatus(tx, sid, types.ContributionPending, "", ""); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("未知投稿类型")
	}
	return tx.Commit()
}

func (cs *contributionStore) countPending(playerID string) (int, error) {
	var n int
	err := cs.db.QueryRow(`SELECT
		(SELECT COUNT(*) FROM sub_tasks t INNER JOIN (SELECT id, MAX(version) v FROM sub_tasks WHERE contributor_player_id=? GROUP BY id) m ON t.id=m.id AND t.version=m.v WHERE t.series_id='' AND t.active=1 AND t.status='pending') +
		(SELECT COUNT(*) FROM series t INNER JOIN (SELECT id, MAX(version) v FROM series WHERE contributor_player_id=? GROUP BY id) m ON t.id=m.id AND t.version=m.v WHERE t.status='pending')`,
		playerID, playerID).Scan(&n)
	return n, err
}

// withdraw 是玩家自助下架/撤回：不管当前状态是 pending 还是 approved，直接把最新那一行
// 标记为 withdrawn（纯状态流转，不产生新版本）；系列同时撤回它名下所有步骤。
func (cs *contributionStore) withdraw(kind, playerID, id string) error {
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	switch kind {
	case types.ContributionKindTask:
		row, err := cs.tasks.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.SeriesID != "" || !row.Active {
			return errContributionNotFound
		}
		if row.ContributorPlayerID != playerID || row.Status == types.ContributionWithdrawn || row.Status == types.ContributionDraft {
			return errContributionBadStatus
		}
		if err := cs.tasks.setStatus(tx, id, types.ContributionWithdrawn, "", ""); err != nil {
			return err
		}
	case types.ContributionKindSeries:
		row, err := cs.series.latestTx(tx, id)
		if err != nil {
			return err
		}
		if row.ContributorPlayerID != playerID || row.Status == types.ContributionWithdrawn || row.Status == types.ContributionDraft {
			return errContributionBadStatus
		}
		if err := cs.series.setStatus(tx, id, types.ContributionWithdrawn, "", ""); err != nil {
			return err
		}
		stepIDs, err := seriesStepIDsTx(tx, id)
		if err != nil {
			return err
		}
		for _, sid := range stepIDs {
			if err := cs.tasks.setStatus(tx, sid, types.ContributionWithdrawn, "", ""); err != nil {
				return err
			}
		}
	default:
		return fmt.Errorf("未知投稿类型")
	}
	return tx.Commit()
}

func parseStepDraftLenient(raw string) (types.StepDraft, error) {
	var d types.StepDraft
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return d, fmt.Errorf("任务投稿格式错误")
	}
	if err := validateStepDraftBounds(d); err != nil {
		return d, err
	}
	return d, nil
}

func validateStepDraftBounds(d types.StepDraft) error {
	if len(d.Variants) > maxStepCandidates {
		return fmt.Errorf("文案变体过多")
	}
	if len(d.TagIDs) > maxContributionRefs {
		return fmt.Errorf("惩罚标签过多")
	}
	for _, id := range d.TagIDs {
		if len([]rune(id)) > maxContributionIDRunes {
			return fmt.Errorf("惩罚标签 ID 过长")
		}
	}
	for _, variant := range d.Variants {
		if len([]rune(variant.Text)) > maxTaskTextRunes {
			return fmt.Errorf("每份文案不能超过 %d 个字符", maxTaskTextRunes)
		}
		if len(variant.FactionIDs) > maxContributionRefs {
			return fmt.Errorf("文案适用阵营过多")
		}
		for _, id := range variant.FactionIDs {
			if len([]rune(id)) > maxContributionIDRunes {
				return fmt.Errorf("阵营 ID 过长")
			}
		}
	}
	background := strings.TrimSpace(d.BackgroundImage)
	if len([]rune(background)) > 1024 {
		return fmt.Errorf("背景图片地址过长")
	}
	if background != "" && (!strings.HasPrefix(background, "/uploads/contributions/") || safeUploadURL(background) != background) {
		return fmt.Errorf("背景图片地址无效")
	}
	if d.BackgroundOpacity < 0 || d.BackgroundOpacity > 1 {
		return fmt.Errorf("背景透明率必须在 0 到 1 之间")
	}
	return nil
}

// validateContributionRefs 检查任务/系列引用的标签、阵营是否真实存在。
// 草稿允许未写完，所以这里用宽松解析；结构性错误留给 validateOwnedDraft（提交时）报。
func (s *Server) validateContributionRefs(kind, raw string) error {
	tagSet := make(map[string]struct{}, len(s.cfg.PunishmentTags))
	for _, tag := range s.cfg.PunishmentTags {
		tagSet[tag.ID] = struct{}{}
	}
	factionSet := make(map[string]struct{}, len(s.cfg.GenderFactions))
	for _, f := range s.cfg.GenderFactions {
		factionSet[f.ID] = struct{}{}
	}
	checkStepRefs := func(label string, d types.StepDraft) error {
		for _, tid := range d.TagIDs {
			if _, ok := tagSet[tid]; !ok {
				return fmt.Errorf("%s引用了不存在的标签", label)
			}
		}
		for _, v := range d.Variants {
			for _, fid := range v.FactionIDs {
				if _, ok := factionSet[fid]; !ok {
					return fmt.Errorf("%s勾选了不存在的阵营", label)
				}
			}
		}
		return nil
	}
	switch kind {
	case types.ContributionKindTask:
		d, err := parseStepDraftLenient(raw)
		if err != nil {
			return err
		}
		return checkStepRefs("任务", d)
	case types.ContributionKindSeries:
		draft, err := parseSeriesDraftLenient(raw)
		if err != nil {
			return err
		}
		for _, fid := range draft.TargetFactionIDs {
			if _, ok := factionSet[fid]; !ok {
				return fmt.Errorf("系列勾选了不存在的阵营")
			}
		}
		for i, step := range draft.Steps {
			if err := checkStepRefs(fmt.Sprintf("第 %d 步", i+1), step); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Server) validateSeriesMinSteps(raw string) error {
	draft, err := parseSeriesDraftLenient(raw)
	if err != nil {
		return err
	}
	minSteps := effectiveMinSeriesSteps(s.cfg)
	if len(draft.Steps) < minSteps {
		return fmt.Errorf("系列任务至少需要 %d 步，当前只有 %d 步", minSteps, len(draft.Steps))
	}
	return nil
}

func (s *Server) validateSeriesMaxSteps(raw string) error {
	draft, err := parseSeriesDraftLenient(raw)
	if err != nil {
		return err
	}
	maxSteps := effectiveMaxSeriesSteps(s.cfg)
	if len(draft.Steps) > maxSteps {
		return fmt.Errorf("系列任务最多 %d 步，当前有 %d 步", maxSteps, len(draft.Steps))
	}
	return nil
}

// validateOwnedDraft 是提交审批前的最终校验（比 saveDraft 时更严格）：内容必须真正完整
// 合法（每份文案非空、难度在范围内等），不只是 JSON 形状对。
func (cs *contributionStore) validateOwnedDraft(kind, raw string) error {
	switch kind {
	case types.ContributionKindTask:
		_, err := decodeStepDraft(raw)
		return err
	case types.ContributionKindSeries:
		_, err := decodeSeriesDraft(raw)
		return err
	default:
		return fmt.Errorf("未知投稿类型")
	}
}

func parseSeriesDraftLenient(raw string) (types.SeriesDraft, error) {
	var d types.SeriesDraft
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return d, fmt.Errorf("系列投稿格式错误")
	}
	if len([]rune(d.Name)) > maxSeriesNameRunes {
		return d, fmt.Errorf("系列名称不能超过 %d 个字符", maxSeriesNameRunes)
	}
	if len(d.Steps) > maxSeriesSteps {
		return d, fmt.Errorf("系列步骤过多")
	}
	if len(d.TargetFactionIDs) > maxContributionRefs {
		return d, fmt.Errorf("系列目标阵营过多")
	}
	for _, id := range d.TargetFactionIDs {
		if len([]rune(id)) > maxContributionIDRunes {
			return d, fmt.Errorf("阵营 ID 过长")
		}
	}
	for i, step := range d.Steps {
		if err := validateStepDraftBounds(step); err != nil {
			return d, fmt.Errorf("第 %d 步：%w", i+1, err)
		}
	}
	return d, nil
}
