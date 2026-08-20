package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (cs *contributionStore) getOwned(id, playerID string) (types.ContributionSubmission, error) {
	sub, err := cs.get(id)
	if err != nil {
		return sub, err
	}
	if sub.SubmitterPlayerID != playerID {
		return types.ContributionSubmission{}, errContributionForbidden
	}
	return sub, nil
}

func (cs *contributionStore) get(id string) (types.ContributionSubmission, error) {
	var s types.ContributionSubmission
	var anon int
	err := cs.db.QueryRow(`
		SELECT id, kind, submitter_player_id, submitter_name_snapshot, anonymous, status,
			published_target_id, published_version, active_version, created_at, updated_at,
			submitted_at, reviewed_at, reviewed_by, review_comment, unpublish_requested_at
		FROM contribution_submissions WHERE id = ?`, id).Scan(
		&s.ID, &s.Kind, &s.SubmitterPlayerID, &s.SubmitterNameSnap, &anon, &s.Status,
		&s.PublishedTargetID, &s.PublishedVersion, &s.ActiveVersion, &s.CreatedAt, &s.UpdatedAt,
		&s.SubmittedAt, &s.ReviewedAt, &s.ReviewedBy, &s.ReviewComment, &s.UnpublishRequestedAt,
	)
	if err == sql.ErrNoRows {
		return s, errContributionNotFound
	}
	s.Anonymous = anon != 0
	return s, err
}

func (cs *contributionStore) listBySubmitter(playerID string) ([]types.ContributionSubmission, error) {
	rows, err := cs.db.Query(`
		SELECT s.id, s.kind, s.submitter_player_id, s.submitter_name_snapshot, s.anonymous, s.status,
			s.published_target_id, s.published_version, s.active_version, s.created_at, s.updated_at,
			s.submitted_at, s.reviewed_at, s.reviewed_by, s.review_comment, s.unpublish_requested_at,
			COALESCE(NULLIF(v.reviewed_content, ''), v.content, '')
		FROM contribution_submissions s
		LEFT JOIN contribution_versions v ON v.submission_id = s.id AND v.version = s.active_version
		WHERE s.submitter_player_id = ?
		ORDER BY s.updated_at DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubmissions(rows)
}

func (cs *contributionStore) countAllByStatus(status string) (int, error) {
	var n int
	err := cs.db.QueryRow(`SELECT COUNT(*) FROM contribution_submissions WHERE status = ?`, status).Scan(&n)
	return n, err
}

func (cs *contributionStore) countBySubmitter(playerID string) (int, error) {
	var n int
	err := cs.db.QueryRow(`SELECT COUNT(*) FROM contribution_submissions WHERE submitter_player_id = ?`, playerID).Scan(&n)
	return n, err
}

func (cs *contributionStore) reviewQueueCounts() (pending, revision, unpublish int, err error) {
	pending, err = cs.countAllByStatus(types.ContributionPending)
	if err != nil {
		return 0, 0, 0, err
	}
	revision, err = cs.countAllByStatus(types.ContributionRevisionPending)
	if err != nil {
		return 0, 0, 0, err
	}
	unpublish, err = cs.countAllByStatus(types.ContributionUnpublishPending)
	if err != nil {
		return 0, 0, 0, err
	}
	return pending, revision, unpublish, nil
}

// listByStatus 按状态/类型筛选投稿；status 为空表示不按单一状态过滤，但仍会排除
// draft（其它玩家仍在编辑、从未提交的私有草稿）与 withdrawn（玩家已自行撤回），
// 因为空 status 目前只服务于后台审核列表——这两种状态既不需要、也不该出现在
// 管理员能看到的清单里（草稿是他人隐私，撤回已经是终态、无需再处理）。
func (cs *contributionStore) listByStatus(status, kind string) ([]types.ContributionSubmission, error) {
	q := `
		SELECT s.id, s.kind, s.submitter_player_id, s.submitter_name_snapshot, s.anonymous, s.status,
			s.published_target_id, s.published_version, s.active_version, s.created_at, s.updated_at,
			s.submitted_at, s.reviewed_at, s.reviewed_by, s.review_comment, s.unpublish_requested_at,
			COALESCE(NULLIF(v.reviewed_content, ''), v.content, '')
		FROM contribution_submissions s
		LEFT JOIN contribution_versions v ON v.submission_id = s.id AND v.version = s.active_version
		WHERE 1=1`
	args := []any{}
	if status != "" {
		q += ` AND s.status = ?`
		args = append(args, status)
	} else {
		q += ` AND s.status NOT IN (?, ?)`
		args = append(args, types.ContributionDraft, types.ContributionWithdrawn)
	}
	if kind != "" {
		q += ` AND s.kind = ?`
		args = append(args, kind)
	}
	q += ` ORDER BY s.updated_at DESC`
	rows, err := cs.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubmissions(rows)
}

// listStatusIn 返回某类型下、状态落在 statuses 集合内的投稿，不排序（调用方按需再排）。
// 供「待处理」板块的性别审批队列使用：待审批/复审中/下架申请中/已驳回都需要管理员
// 能看到并操作（已驳回的还能撤销/改批注），已通过的性别不需要——那些直接体现在
// 「性别与阵营」配置页里，不必在审核队列里重复出现。
func (cs *contributionStore) listStatusIn(kind string, statuses []string) ([]types.ContributionSubmission, error) {
	if len(statuses) == 0 {
		return []types.ContributionSubmission{}, nil
	}
	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(statuses)), ",")
	q := `
		SELECT s.id, s.kind, s.submitter_player_id, s.submitter_name_snapshot, s.anonymous, s.status,
			s.published_target_id, s.published_version, s.active_version, s.created_at, s.updated_at,
			s.submitted_at, s.reviewed_at, s.reviewed_by, s.review_comment, s.unpublish_requested_at,
			COALESCE(NULLIF(v.reviewed_content, ''), v.content, '')
		FROM contribution_submissions s
		LEFT JOIN contribution_versions v ON v.submission_id = s.id AND v.version = s.active_version
		WHERE s.kind = ? AND s.status IN (` + placeholders + `)
		ORDER BY s.updated_at DESC`
	args := make([]any, 0, len(statuses)+1)
	args = append(args, kind)
	for _, st := range statuses {
		args = append(args, st)
	}
	rows, err := cs.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanSubmissions(rows)
}

// reviewCountsMatrix 统计给定 kind 列表下的审核相关状态投稿数。
// 供后台「待处理」板块的总览统计使用。
func (cs *contributionStore) reviewCountsMatrix(kinds []string) (map[string]map[string]int, error) {
	statuses := []string{
		types.ContributionPending,
		types.ContributionRevisionDraft,
		types.ContributionRevisionPending,
		types.ContributionRevisionRejected,
		types.ContributionApproved,
		types.ContributionRejected,
		types.ContributionUnpublishPending,
	}
	out := make(map[string]map[string]int, len(kinds))
	for _, kind := range kinds {
		row := make(map[string]int, len(statuses))
		for _, st := range statuses {
			row[st] = 0
		}
		out[kind] = row
	}
	rows, err := cs.db.Query(`SELECT kind, status, COUNT(*) FROM contribution_submissions WHERE kind IN (`+
		strings.TrimSuffix(strings.Repeat("?,", len(kinds)), ",")+`) GROUP BY kind, status`, toAnySlice(kinds)...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var kind, status string
		var n int
		if err := rows.Scan(&kind, &status, &n); err != nil {
			return nil, err
		}
		if row, ok := out[kind]; ok {
			if _, tracked := row[status]; tracked {
				row[status] = n
			}
		}
	}
	return out, rows.Err()
}

func toAnySlice(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

func scanSubmissions(rows *sql.Rows) ([]types.ContributionSubmission, error) {
	out := make([]types.ContributionSubmission, 0)
	for rows.Next() {
		var s types.ContributionSubmission
		var anon int
		var content string
		if err := rows.Scan(
			&s.ID, &s.Kind, &s.SubmitterPlayerID, &s.SubmitterNameSnap, &anon, &s.Status,
			&s.PublishedTargetID, &s.PublishedVersion, &s.ActiveVersion, &s.CreatedAt, &s.UpdatedAt,
			&s.SubmittedAt, &s.ReviewedAt, &s.ReviewedBy, &s.ReviewComment, &s.UnpublishRequestedAt,
			&content,
		); err != nil {
			return nil, err
		}
		s.Anonymous = anon != 0
		s.Title = contributionPreviewTitle(s.Kind, content)
		out = append(out, s)
	}
	return out, rows.Err()
}

func contributionPreviewTitle(kind, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	switch kind {
	case types.ContributionKindGender:
		var d types.GenderDraft
		if json.Unmarshal([]byte(raw), &d) != nil {
			return ""
		}
		return strings.TrimSpace(d.Label)
	case types.ContributionKindTask:
		d, err := normalizeLegacyStepJSON([]byte(raw))
		if err != nil {
			return ""
		}
		if len(d.Variants) > 0 {
			return truncateRunes(strings.TrimSpace(d.Variants[0].Text), 24)
		}
	case types.ContributionKindSeries:
		draft, _, err := normalizeLegacySeriesJSON([]byte(raw))
		if err != nil {
			return ""
		}
		return strings.TrimSpace(draft.Name)
	}
	return ""
}

func truncateRunes(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "…"
}

func (cs *contributionStore) getVersion(submissionID string, version int) (types.ContributionVersion, error) {
	var v types.ContributionVersion
	err := cs.db.QueryRow(`
		SELECT id, submission_id, version, content, created_by, created_at,
			reviewed_content, reviewed_at, reviewed_by, review_result, review_comment
		FROM contribution_versions WHERE submission_id = ? AND version = ?`,
		submissionID, version,
	).Scan(&v.ID, &v.SubmissionID, &v.Version, &v.Content, &v.CreatedBy, &v.CreatedAt,
		&v.ReviewedContent, &v.ReviewedAt, &v.ReviewedBy, &v.ReviewResult, &v.ReviewComment)
	if err == sql.ErrNoRows {
		return v, errContributionNotFound
	}
	return v, err
}

func (cs *contributionStore) countStatus(playerID, status string) (int, error) {
	var n int
	err := cs.db.QueryRow(
		`SELECT COUNT(*) FROM contribution_submissions WHERE submitter_player_id = ? AND status = ?`,
		playerID, status,
	).Scan(&n)
	return n, err
}

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
	case types.ContributionKindGender, types.ContributionKindTask, types.ContributionKindSeries:
		return true
	default:
		return false
	}
}

func (cs *contributionStore) validateDraft(kind, raw string) ([]string, error) {
	var images []string
	switch kind {
	case types.ContributionKindGender:
		_, err := decodeGenderDraft(raw)
		return nil, err
	case types.ContributionKindTask:
		draft, err := decodeStepDraft(raw)
		if err != nil {
			return nil, err
		}
		if !draft.InRandomPool || draft.Order < 1 || draft.Order > 99 {
			return nil, fmt.Errorf("随机任务必须填写 1 到 99 的难度")
		}
		images = appendDraftImage(images, draft.BackgroundImage)
	case types.ContributionKindSeries:
		draft, skipCoverage, err := decodeSeriesDraft(raw)
		if err != nil {
			return nil, err
		}
		if !skipCoverage && len(draft.TargetFactionIDs) == 0 {
			return nil, fmt.Errorf("请选择本系列面向的阵营")
		}
		if !skipCoverage {
			for i, step := range draft.Steps {
				if missing := missingTargetFactions(step, draft.TargetFactionIDs); len(missing) > 0 {
					return nil, fmt.Errorf("第 %d 步未覆盖阵营：%s", i+1, strings.Join(missing, "、"))
				}
			}
		}
		for _, step := range draft.Steps {
			images = appendDraftImage(images, step.BackgroundImage)
		}
	default:
		return nil, fmt.Errorf("未知投稿类型")
	}
	return images, nil
}

func (cs *contributionStore) validateOwnedDraft(playerID, kind, raw string) ([]string, error) {
	images, err := cs.validateDraft(kind, raw)
	if err != nil {
		return nil, err
	}
	if err := cs.ensureOwnedImages(playerID, images); err != nil {
		return nil, err
	}
	return images, nil
}

// validateAdminReviewedDraft 允许审核管理员使用自己上传的新封面，同时继续允许投稿者原图。
// adminID 是当前已通过 admin:login 的 socket 所绑定玩家 ID；没有绑定玩家时值为 "admin"，
// 此时自然只能沿用投稿者已有图片，无法凭管理员口令冒充任意上传者。
func (cs *contributionStore) validateAdminReviewedDraft(submitterID, adminID, kind, raw string) ([]string, error) {
	images, err := cs.validateDraft(kind, raw)
	if err != nil {
		return nil, err
	}
	for _, path := range images {
		if safeUploadURL(path) != path || !strings.HasPrefix(path, "/uploads/contributions/") {
			return nil, fmt.Errorf("任务背景图地址无效")
		}
		owned, err := cs.imageOwnedBy(path, submitterID)
		if err != nil {
			return nil, err
		}
		if !owned && adminID != "" && adminID != "admin" {
			owned, err = cs.imageOwnedBy(path, adminID)
			if err != nil {
				return nil, err
			}
		}
		if !owned {
			return nil, fmt.Errorf("无权使用该任务背景图")
		}
	}
	return images, nil
}

// inspectDraftForSave 只做「能存成草稿」的最低校验：JSON 形状对、步骤不超过上限、
// 引用的封面图属于当前玩家。文案/名称/最低步数可以空着，提交审批时再走 validateOwnedDraft。
func (cs *contributionStore) inspectDraftForSave(playerID, kind, raw string) ([]string, error) {
	var images []string
	switch kind {
	case types.ContributionKindGender:
		var d types.GenderDraft
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return nil, fmt.Errorf("性别投稿格式错误")
		}
	case types.ContributionKindTask:
		draft, err := normalizeLegacyStepJSON([]byte(raw))
		if err != nil {
			return nil, err
		}
		if len(draft.Variants) > maxStepCandidates {
			return nil, fmt.Errorf("文案变体过多")
		}
		images = appendDraftImage(images, draft.BackgroundImage)
	case types.ContributionKindSeries:
		draft, _, err := normalizeLegacySeriesJSON([]byte(raw))
		if err != nil {
			return nil, err
		}
		if len(draft.Steps) > maxSeriesSteps {
			return nil, fmt.Errorf("系列步骤过多")
		}
		for _, step := range draft.Steps {
			if len(step.Variants) > maxStepCandidates {
				return nil, fmt.Errorf("文案变体过多")
			}
			images = appendDraftImage(images, step.BackgroundImage)
		}
	default:
		return nil, fmt.Errorf("未知投稿类型")
	}
	if err := cs.ensureOwnedImages(playerID, images); err != nil {
		return nil, err
	}
	return images, nil
}

func (cs *contributionStore) ensureOwnedImages(playerID string, images []string) error {
	for _, path := range images {
		if safeUploadURL(path) != path || !strings.HasPrefix(path, "/uploads/contributions/") {
			return fmt.Errorf("任务背景图地址无效")
		}
		owned, err := cs.imageOwnedBy(path, playerID)
		if err != nil {
			return err
		}
		if !owned {
			return fmt.Errorf("无权使用该任务背景图")
		}
	}
	return nil
}

func appendDraftImage(images []string, path string) []string {
	path = strings.TrimSpace(path)
	if path == "" {
		return images
	}
	return append(images, path)
}

func decodeGenderDraft(raw string) (types.GenderDraft, error) {
	var d types.GenderDraft
	if err := json.Unmarshal([]byte(raw), &d); err != nil {
		return d, fmt.Errorf("性别投稿格式错误")
	}
	d.Label = strings.TrimSpace(d.Label)
	d.FactionID = strings.TrimSpace(d.FactionID)
	if d.Label == "" || d.FactionID == "" {
		return d, fmt.Errorf("性别名称和阵营不能为空")
	}
	if len([]rune(d.Label)) > maxGenderLabelRunes {
		return d, fmt.Errorf("性别名称不能超过 %d 个字符", maxGenderLabelRunes)
	}
	if containsControlRune(d.Label) {
		return d, fmt.Errorf("性别名称包含控制字符")
	}
	if normalizeGenderLabel(d.Label) == "" {
		return d, fmt.Errorf("性别名称包含无效字符")
	}
	return d, nil
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
	case types.ContributionKindGender:
		var d types.GenderDraft
		if err := json.Unmarshal([]byte(raw), &d); err != nil {
			return fmt.Errorf("性别投稿格式错误")
		}
		if fid := strings.TrimSpace(d.FactionID); fid != "" && !s.genderFactionExists(fid) {
			return fmt.Errorf("勾选了不存在的阵营")
		}
	case types.ContributionKindTask:
		d, err := normalizeLegacyStepJSON([]byte(raw))
		if err != nil {
			return err
		}
		return checkStepRefs("任务", d)
	case types.ContributionKindSeries:
		draft, _, err := normalizeLegacySeriesJSON([]byte(raw))
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

func (s *Server) genderFactionExists(id string) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, f := range s.cfg.GenderFactions {
		if f.ID == id {
			return true
		}
	}
	return false
}

func (s *Server) validateSeriesMinSteps(raw string) error {
	draft, _, err := decodeSeriesDraft(raw)
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
	draft, _, err := decodeSeriesDraft(raw)
	if err != nil {
		return err
	}
	maxSteps := effectiveMaxSeriesSteps(s.cfg)
	if len(draft.Steps) > maxSteps {
		return fmt.Errorf("系列任务最多 %d 步，当前有 %d 步", maxSteps, len(draft.Steps))
	}
	return nil
}
