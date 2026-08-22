package server

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// migratePunishmentToSubTasksV43 是共建投稿存储重构（punishment_tasks/punishment_series
// 两张"整表替换式"表 + contribution_submissions/versions 投稿信封 → sub_tasks/series 两张
// 版本化、插入不更新的表，见 CLAUDE.md「共建投稿」一节）的落地迁移：
//  1. 把旧 punishment_tasks（按 task_group_id 分组＝新模型里"一行多阵营变体"的 Variants）
//     和 punishment_series（steps JSON 里的 taskIds → 对应 sub_tasks 行的 SeriesID/StepIndex）
//     转换进新表，全部标记为 status='approved'（这些都是迁移前已经在线上生效的内容）。
//  2. punishment_events 把 publisher_id/publisher_name/target_id/target_name 改名为
//     approver_id/approver_name/performer_id/performer_name，新增 performer_vote/
//     approver_vote/trigger/tie_double_punish 列。
//  3. 性别投稿功能下线，gender_options.submission_id 列（v41 加）随之作废丢弃；
//     gender_factions/gender_options 两张表本身保留（后台「性别与阵营」批量管理仍在用，
//     与共建投稿系统无关）。
//  4. 删除所有被取代的旧表：punishment_tasks/punishment_series、
//     contribution_submissions/versions/images/votes/vote_overrides/gender_claims/
//     round_participants。
//
// 旧投稿信封、版本和投票会先迁入新表；只有全部转换成功后，才在同一迁移事务末尾删除
// 被取代的旧表。任一步失败都会回滚，因此不会出现“旧表已删、新表未写完”的半迁移状态。
// 对旧任务池中确实没有记录的贡献者姓名/审核信息，按“补全不了的不填”原则留空。
func migratePunishmentToSubTasksV43(db sqlExecer) error {
	now := nowMs()

	// v41 建的 idx_gender_options_submission 索引依赖这一列，SQLite 不允许在还有索引引用
	// 该列时直接 DROP COLUMN，必须先把索引丢掉。
	if _, err := db.Exec(`DROP INDEX IF EXISTS idx_gender_options_submission`); err != nil {
		return err
	}
	if err := dropColumnIfExists(db, "gender_options", "submission_id"); err != nil {
		return err
	}

	hasTasks, err := tableExists(db, "punishment_tasks")
	if err != nil {
		return err
	}
	hasSeries, err := tableExists(db, "punishment_series")
	if err != nil {
		return err
	}
	if hasTasks || hasSeries {
		if err := migrateLegacyPunishmentPool(db, now, hasTasks, hasSeries); err != nil {
			return err
		}
	}
	if err := migrateLegacyContributionEnvelopes(db, now); err != nil {
		return err
	}

	if err := renameColumnIfExists(db, "punishment_events", "publisher_id", "approver_id"); err != nil {
		return err
	}
	if err := renameColumnIfExists(db, "punishment_events", "publisher_name", "approver_name"); err != nil {
		return err
	}
	if err := renameColumnIfExists(db, "punishment_events", "target_id", "performer_id"); err != nil {
		return err
	}
	if err := renameColumnIfExists(db, "punishment_events", "target_name", "performer_name"); err != nil {
		return err
	}
	for _, col := range []struct{ name, decl string }{
		{"performer_vote", "INTEGER NOT NULL DEFAULT 0"},
		{"approver_vote", "INTEGER NOT NULL DEFAULT 0"},
		{"trigger", "TEXT NOT NULL DEFAULT 'round_end'"},
		{"tie_double_punish", "INTEGER NOT NULL DEFAULT 0"},
	} {
		if err := addColumnIfMissing(db, "punishment_events", col.name, col.decl); err != nil {
			return err
		}
	}
	// performer_id 改名完成后才能对它建索引（punishmentEventSchema 那段无条件 DDL 里刻意
	// 没放这条，见那边的注释）。
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_punishment_events_target ON punishment_events(performer_id, task_at)`); err != nil {
		return err
	}
	if err := migrateLegacyContributionVotes(db); err != nil {
		return err
	}

	// contribution_versions 外键指向 contribution_submissions，必须先删子表再删主表；
	// 非空旧库在 foreign_keys=1 下若顺序相反会使整个 v43 事务失败。
	return dropLegacyPunishmentTables(db)
}

func dropLegacyPunishmentTables(db sqlExecer) error {
	for _, table := range []string{
		"contribution_versions", "contribution_images", "contribution_votes",
		"contribution_vote_overrides", "contribution_gender_claims", "contribution_round_participants",
		"contribution_submissions", "punishment_tasks", "punishment_series",
	} {
		if _, err := db.Exec(`DROP TABLE IF EXISTS ` + table); err != nil {
			return err
		}
	}
	return nil
}

// migrateLegacyContributionVotes 把旧独立投票表折叠进新模型：聚合数写回精确的任务版本，
// 有 punishment_event_id 的票同时回填事件两侧防重列，避免升级后同一玩家对同一旧事件再投。
func migrateLegacyContributionVotes(db sqlExecer) error {
	type aggregate struct {
		id           string
		version      int
		likes, downs int
	}
	rows, err := db.Query(`SELECT target_id, target_version,
		SUM(CASE WHEN vote=1 THEN 1 ELSE 0 END), SUM(CASE WHEN vote=-1 THEN 1 ELSE 0 END)
		FROM contribution_votes WHERE target_kind='task' GROUP BY target_id, target_version`)
	if err != nil {
		return err
	}
	var aggregates []aggregate
	for rows.Next() {
		var a aggregate
		if err := rows.Scan(&a.id, &a.version, &a.likes, &a.downs); err != nil {
			rows.Close()
			return err
		}
		aggregates = append(aggregates, a)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, a := range aggregates {
		if _, err := db.Exec(`UPDATE sub_tasks SET like_count=like_count+?, down_count=down_count+?
			WHERE id=? AND version=?`, a.likes, a.downs, a.id, legacyContentVersion(a.version)); err != nil {
			return err
		}
	}

	type eventVote struct {
		eventID, voterID string
		vote             int
	}
	rows, err = db.Query(`SELECT punishment_event_id, voter_player_id, vote FROM contribution_votes
		WHERE punishment_event_id<>'' AND vote IN (-1,1)`)
	if err != nil {
		return err
	}
	var votes []eventVote
	for rows.Next() {
		var v eventVote
		if err := rows.Scan(&v.eventID, &v.voterID, &v.vote); err != nil {
			rows.Close()
			return err
		}
		votes = append(votes, v)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	for _, v := range votes {
		if _, err := db.Exec(`UPDATE punishment_events SET
			performer_vote=CASE WHEN performer_id=? AND performer_vote=0 THEN ? ELSE performer_vote END,
			approver_vote=CASE WHEN approver_id=? AND approver_vote=0 THEN ? ELSE approver_vote END
			WHERE id=?`, v.voterID, v.vote, v.voterID, v.vote, v.eventID); err != nil {
			return err
		}
	}
	return nil
}

// legacyContributionSubmission/Version 是 v42 以前投稿信封的迁移快照。只搬仍受支持的
// task/series；gender 投稿功能已明确下线，不重新映射成其它业务数据。
type legacyContributionSubmission struct {
	id, kind, submitterID, submitterName, status, publishedTargetID string
	anonymous                                                       bool
	activeVersion                                                   int
	createdAt, updatedAt, reviewedAt                                int64
	reviewedBy, reviewComment                                       string
	versions                                                        []legacyContributionVersion
}

type legacyContributionVersion struct {
	version                                 int
	content, reviewedContent                string
	createdAt, reviewedAt                   int64
	reviewedBy, reviewResult, reviewComment string
}

func normalizeLegacyContributionStatus(status string) string {
	switch status {
	case "draft", "revision_draft":
		return types.ContributionDraft
	case "pending", "revision_pending":
		return types.ContributionPending
	case "approved", "unpublish_pending":
		// 旧版 unpublish_pending 在管理员真正下架前仍在线。
		return types.ContributionApproved
	case "rejected", "revision_rejected":
		return types.ContributionRejected
	case "withdrawn":
		return types.ContributionWithdrawn
	default:
		return types.ContributionDraft
	}
}

func legacyVersionStatus(sub legacyContributionSubmission, v legacyContributionVersion) string {
	if v.version == sub.activeVersion {
		return normalizeLegacyContributionStatus(sub.status)
	}
	if v.reviewResult == "approved" || v.reviewResult == "rejected" {
		return v.reviewResult
	}
	return types.ContributionDraft
}

func legacyVersionContent(v legacyContributionVersion) string {
	if strings.TrimSpace(v.reviewedContent) != "" {
		return v.reviewedContent
	}
	return v.content
}

func migrateLegacyContributionEnvelopes(db sqlExecer, now int64) error {
	rows, err := db.Query(`
		SELECT id, kind, submitter_player_id, submitter_name_snapshot, anonymous, status,
			published_target_id, active_version, created_at, updated_at, reviewed_at, reviewed_by, review_comment
		FROM contribution_submissions
		WHERE kind IN ('task', 'series') ORDER BY created_at, id`)
	if err != nil {
		return err
	}
	var submissions []legacyContributionSubmission
	for rows.Next() {
		var sub legacyContributionSubmission
		var anonymous int
		if err := rows.Scan(&sub.id, &sub.kind, &sub.submitterID, &sub.submitterName, &anonymous, &sub.status,
			&sub.publishedTargetID, &sub.activeVersion, &sub.createdAt, &sub.updatedAt, &sub.reviewedAt,
			&sub.reviewedBy, &sub.reviewComment); err != nil {
			rows.Close()
			return err
		}
		sub.anonymous = anonymous != 0
		submissions = append(submissions, sub)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()

	for i := range submissions {
		vrows, err := db.Query(`
			SELECT version, content, reviewed_content, created_at, reviewed_at, reviewed_by, review_result, review_comment
			FROM contribution_versions WHERE submission_id = ? ORDER BY version`, submissions[i].id)
		if err != nil {
			return err
		}
		for vrows.Next() {
			var v legacyContributionVersion
			if err := vrows.Scan(&v.version, &v.content, &v.reviewedContent, &v.createdAt, &v.reviewedAt,
				&v.reviewedBy, &v.reviewResult, &v.reviewComment); err != nil {
				vrows.Close()
				return err
			}
			submissions[i].versions = append(submissions[i].versions, v)
		}
		if err := vrows.Err(); err != nil {
			vrows.Close()
			return err
		}
		vrows.Close()
	}

	for _, sub := range submissions {
		logicalID := sub.id
		if sub.publishedTargetID != "" {
			logicalID = sub.publishedTargetID
		}
		switch sub.kind {
		case types.ContributionKindTask:
			if err := migrateLegacyTaskSubmission(db, now, logicalID, sub); err != nil {
				return fmt.Errorf("migrate legacy task submission %s: %w", sub.id, err)
			}
		case types.ContributionKindSeries:
			if err := migrateLegacySeriesSubmission(db, now, logicalID, sub); err != nil {
				return fmt.Errorf("migrate legacy series submission %s: %w", sub.id, err)
			}
		}
	}
	return nil
}

func legacyReviewMeta(sub legacyContributionSubmission, v legacyContributionVersion) (string, int64, string) {
	by, at, comment := v.reviewedBy, v.reviewedAt, v.reviewComment
	if v.version == sub.activeVersion {
		if by == "" {
			by = sub.reviewedBy
		}
		if at == 0 {
			at = sub.reviewedAt
		}
		if comment == "" {
			comment = sub.reviewComment
		}
	}
	return by, at, comment
}

func migrateLegacyTaskSubmission(db sqlExecer, now int64, id string, sub legacyContributionSubmission) error {
	for _, v := range sub.versions {
		var draft types.StepDraft
		if err := json.Unmarshal([]byte(legacyVersionContent(v)), &draft); err != nil {
			return err
		}
		variants, err := json.Marshal(nonNilVariants(draft.Variants))
		if err != nil {
			return err
		}
		tags, err := json.Marshal(nonNilStrings(draft.TagIDs))
		if err != nil {
			return err
		}
		version := legacyContentVersion(v.version)
		createdAt := v.createdAt
		if createdAt == 0 {
			createdAt = sub.createdAt
		}
		updatedAt := createdAt
		if v.version == sub.activeVersion && sub.updatedAt != 0 {
			updatedAt = sub.updatedAt
		}
		if updatedAt == 0 {
			updatedAt = now
		}
		order := draft.Order
		if !draft.InRandomPool {
			order = -1
		}
		reviewedBy, reviewedAt, reviewComment := legacyReviewMeta(sub, v)
		if _, err := db.Exec(`INSERT INTO sub_tasks (
			id, version, series_id, step_index, active, variants, tag_ids, background_image, background_opacity,
			difficulty_order, status, like_count, down_count, contributor_player_id, contributor_name,
			contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, updated_at
		) VALUES (?, ?, '', 0, 1, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, version) DO UPDATE SET
			status=excluded.status, contributor_player_id=excluded.contributor_player_id,
			contributor_name=excluded.contributor_name, contributor_anonymous=excluded.contributor_anonymous,
			reviewed_by=excluded.reviewed_by, reviewed_at=excluded.reviewed_at,
			review_comment=excluded.review_comment, updated_at=excluded.updated_at`,
			id, version, string(variants), string(tags), strings.TrimSpace(draft.BackgroundImage), draft.BackgroundOpacity,
			order, legacyVersionStatus(sub, v), sub.submitterID, sub.submitterName, boolInt(sub.anonymous),
			reviewedBy, reviewedAt, reviewComment, createdAt, updatedAt); err != nil {
			return err
		}
	}
	return nil
}

func latestLegacySeriesStorage(db sqlExecer, id, name string) (string, string, error) {
	var roomPool, backgrounds string
	err := db.QueryRow(`SELECT room_name_pool, room_background_images FROM series WHERE id = ? ORDER BY version DESC LIMIT 1`, id).
		Scan(&roomPool, &backgrounds)
	if err == nil {
		return roomPool, backgrounds, nil
	}
	if err != nil && err != sql.ErrNoRows {
		return "", "", err
	}
	pool, err := json.Marshal(seriesRoomNamePool{Subjects: []string{name}, RoomWords: []string{"小屋"}, Adjectives: []string{"共建"}})
	if err != nil {
		return "", "", err
	}
	return string(pool), "[]", nil
}

func legacySeriesStepIDs(db sqlExecer, seriesID string) ([]string, error) {
	rows, err := db.Query(`
		SELECT t.id FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks WHERE series_id = ? GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.series_id = ? AND t.active = 1 ORDER BY t.step_index`, seriesID, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func migrateLegacySeriesSubmission(db sqlExecer, now int64, id string, sub legacyContributionSubmission) error {
	stepIDs, err := legacySeriesStepIDs(db, id)
	if err != nil {
		return err
	}
	for _, v := range sub.versions {
		var draft types.SeriesDraft
		if err := json.Unmarshal([]byte(legacyVersionContent(v)), &draft); err != nil {
			return err
		}
		status := legacyVersionStatus(sub, v)
		createdAt := v.createdAt
		if createdAt == 0 {
			createdAt = sub.createdAt
		}
		updatedAt := createdAt
		if v.version == sub.activeVersion && sub.updatedAt != 0 {
			updatedAt = sub.updatedAt
		}
		if updatedAt == 0 {
			updatedAt = now
		}
		roomPool, backgrounds, err := latestLegacySeriesStorage(db, id, draft.Name)
		if err != nil {
			return err
		}
		targets, err := json.Marshal(nonNilStrings(draft.TargetFactionIDs))
		if err != nil {
			return err
		}
		version := legacyContentVersion(v.version)
		reviewedBy, reviewedAt, reviewComment := legacyReviewMeta(sub, v)
		if _, err := db.Exec(`INSERT INTO series (
			id, version, name, target_faction_ids, room_name_pool, room_background_images, status,
			contributor_player_id, contributor_name, contributor_anonymous, reviewed_by, reviewed_at,
			review_comment, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id, version) DO UPDATE SET
			status=excluded.status, contributor_player_id=excluded.contributor_player_id,
			contributor_name=excluded.contributor_name, contributor_anonymous=excluded.contributor_anonymous,
			reviewed_by=excluded.reviewed_by, reviewed_at=excluded.reviewed_at,
			review_comment=excluded.review_comment, updated_at=excluded.updated_at`, id, version, strings.TrimSpace(draft.Name),
			string(targets), roomPool, backgrounds, status, sub.submitterID, sub.submitterName, boolInt(sub.anonymous),
			reviewedBy, reviewedAt, reviewComment, createdAt, updatedAt); err != nil {
			return err
		}
		for len(stepIDs) < len(draft.Steps) {
			stepIDs = append(stepIDs, fmt.Sprintf("%s_step_%d", id, len(stepIDs)+1))
		}
		for index, step := range draft.Steps {
			if err := insertLegacySeriesStepVersion(db, now, stepIDs[index], id, index, version, step, status, sub,
				createdAt, updatedAt, reviewedBy, reviewedAt, reviewComment); err != nil {
				return err
			}
		}
		for _, removedID := range stepIDs[len(draft.Steps):] {
			if err := deactivateLegacySeriesStep(db, removedID, version, status, updatedAt); err != nil {
				return err
			}
		}
	}
	return nil
}

func insertLegacySeriesStepVersion(db sqlExecer, now int64, id, seriesID string, stepIndex, version int, draft types.StepDraft,
	status string, sub legacyContributionSubmission, createdAt, updatedAt int64, reviewedBy string, reviewedAt int64, reviewComment string) error {
	variants, err := json.Marshal(nonNilVariants(draft.Variants))
	if err != nil {
		return err
	}
	tags, err := json.Marshal(nonNilStrings(draft.TagIDs))
	if err != nil {
		return err
	}
	order := draft.Order
	if !draft.InRandomPool {
		order = -1
	}
	if updatedAt == 0 {
		updatedAt = now
	}
	_, err = db.Exec(`INSERT INTO sub_tasks (
		id, version, series_id, step_index, active, variants, tag_ids, background_image, background_opacity,
		difficulty_order, status, like_count, down_count, contributor_player_id, contributor_name,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, updated_at
	) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(id, version) DO UPDATE SET
		status=excluded.status, contributor_player_id=excluded.contributor_player_id,
		contributor_name=excluded.contributor_name, contributor_anonymous=excluded.contributor_anonymous,
		reviewed_by=excluded.reviewed_by, reviewed_at=excluded.reviewed_at,
		review_comment=excluded.review_comment, updated_at=excluded.updated_at`, id, version, seriesID, stepIndex,
		string(variants), string(tags), strings.TrimSpace(draft.BackgroundImage), draft.BackgroundOpacity, order, status,
		sub.submitterID, sub.submitterName, boolInt(sub.anonymous), reviewedBy, reviewedAt, reviewComment, createdAt, updatedAt)
	return err
}

func deactivateLegacySeriesStep(db sqlExecer, id string, version int, status string, updatedAt int64) error {
	_, err := db.Exec(`INSERT INTO sub_tasks (
		id, version, series_id, step_index, active, variants, tag_ids, background_image, background_opacity,
		difficulty_order, status, like_count, down_count, contributor_player_id, contributor_name,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, updated_at
	) SELECT id, ?, series_id, step_index, 0, variants, tag_ids, background_image, background_opacity,
		difficulty_order, ?, 0, 0, contributor_player_id, contributor_name,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, ?
		FROM sub_tasks WHERE id = ? AND version < ? ORDER BY version DESC LIMIT 1
		ON CONFLICT(id, version) DO UPDATE SET active=0, status=excluded.status, updated_at=excluded.updated_at`,
		version, status, updatedAt, id, version)
	return err
}

// legacyPoolTaskRow 是旧 punishment_tasks 一行的原始映射（迁移专用，不复用运行时类型）。
type legacyPoolTaskRow struct {
	id              string
	text            string
	tagIDs          []string
	factionIDs      []string
	order           int
	bgImages        []string
	bgOpacity       float64
	sortIndex       int
	contentVersion  int
	contributorID   string
	contributorAnon bool
	taskGroupID     string
	createdAt       int64
}

// legacyPoolGroup 是按 task_group_id 聚合后的新 sub_tasks 一行：旧模型里同一个
// task_group_id 下的多行（各阵营一份文案）就是新模型里一行的多个 Variants。
type legacyPoolGroup struct {
	id              string
	order           int
	bgImage         string
	bgOpacity       float64
	tagIDs          []string
	contributorID   string
	contributorAnon bool
	createdAt       int64
	contentVersion  int
	variants        []types.TaskVariantDraft
	seriesID        string
	stepIndex       int
	hasStep         bool
}

func legacyContentVersion(version int) int {
	if version > 0 {
		return version
	}
	return 1
}

func migrateLegacyPunishmentPool(db sqlExecer, now int64, hasTasks, hasSeries bool) error {
	var tasks []legacyPoolTaskRow
	if hasTasks {
		rows, err := db.Query(`
			SELECT id, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity,
				sort_index, contributor_player_id, contributor_anonymous, content_version, task_group_id, created_at
			FROM punishment_tasks ORDER BY sort_index ASC, id ASC`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var r legacyPoolTaskRow
			var tagJSON, factionJSON, bgJSON string
			var anon int
			if err := rows.Scan(&r.id, &r.text, &tagJSON, &factionJSON, &r.order, &bgJSON, &r.bgOpacity,
				&r.sortIndex, &r.contributorID, &anon, &r.contentVersion, &r.taskGroupID, &r.createdAt); err != nil {
				rows.Close()
				return err
			}
			r.contributorAnon = anon != 0
			_ = json.Unmarshal([]byte(tagJSON), &r.tagIDs)
			_ = json.Unmarshal([]byte(factionJSON), &r.factionIDs)
			_ = json.Unmarshal([]byte(bgJSON), &r.bgImages)
			if r.taskGroupID == "" {
				r.taskGroupID = r.id
			}
			tasks = append(tasks, r)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// 按 task_group_id 聚合，首次出现的成员行决定难度/背景图/贡献者等"整组共享"的字段。
	groups := make(map[string]*legacyPoolGroup)
	var groupOrder []string
	taskGroupOf := make(map[string]string, len(tasks)) // 旧任务 id -> 所属 group id
	for _, t := range tasks {
		taskGroupOf[t.id] = t.taskGroupID
		g, ok := groups[t.taskGroupID]
		if !ok {
			g = &legacyPoolGroup{
				id:              t.taskGroupID,
				order:           t.order,
				bgOpacity:       t.bgOpacity,
				contributorID:   t.contributorID,
				contributorAnon: t.contributorAnon,
				createdAt:       t.createdAt,
				contentVersion:  t.contentVersion,
			}
			if len(t.bgImages) > 0 {
				g.bgImage = t.bgImages[0]
			}
			groups[t.taskGroupID] = g
			groupOrder = append(groupOrder, t.taskGroupID)
		}
		if t.contentVersion > g.contentVersion {
			g.contentVersion = t.contentVersion
		}
		g.tagIDs = append(g.tagIDs, t.tagIDs...)
		g.variants = append(g.variants, types.TaskVariantDraft{
			Text: t.text, FactionIDs: append([]string(nil), t.factionIDs...),
		})
		if t.createdAt != 0 && (g.createdAt == 0 || t.createdAt < g.createdAt) {
			g.createdAt = t.createdAt
		}
	}

	type legacySeriesRow struct {
		id, name             string
		roomNamePoolJSON     string
		bgImagesJSON         string
		targetFactionIDsJSON string
		contributorID        string
		contributorAnon      bool
		contentVersion       int
		createdAt            int64
		steps                []struct {
			TaskIDs []string `json:"taskIds"`
		}
	}
	var series []legacySeriesRow
	if hasSeries {
		rows, err := db.Query(`
			SELECT id, name, room_name_pool, room_background_images, steps,
				contributor_player_id, contributor_anonymous, content_version, target_faction_ids, created_at
			FROM punishment_series ORDER BY sort_index ASC, id ASC`)
		if err != nil {
			return err
		}
		for rows.Next() {
			var s legacySeriesRow
			var stepsJSON string
			var anon int
			if err := rows.Scan(&s.id, &s.name, &s.roomNamePoolJSON, &s.bgImagesJSON, &stepsJSON,
				&s.contributorID, &anon, &s.contentVersion, &s.targetFactionIDsJSON, &s.createdAt); err != nil {
				rows.Close()
				return err
			}
			s.contributorAnon = anon != 0
			_ = json.Unmarshal([]byte(stepsJSON), &s.steps)
			series = append(series, s)
		}
		if err := rows.Err(); err != nil {
			return err
		}
		rows.Close()
	}

	// 把每一步的 taskIds 挂到对应 group 上（同一步的 taskIds 若跨多个 task_group_id，
	// 按第一个 taskId 所在的 group 为准——生产数据里从未出现这种情况，这里只是兜底）。
	for _, s := range series {
		for stepIdx, step := range s.steps {
			var anchor string
			for _, tid := range step.TaskIDs {
				if gid, ok := taskGroupOf[tid]; ok {
					anchor = gid
					break
				}
			}
			if anchor == "" {
				continue // 悬空引用：这一步在旧库里就已经找不到任务了，直接跳过
			}
			if g, ok := groups[anchor]; ok {
				g.seriesID = s.id
				g.stepIndex = stepIdx
				g.hasStep = true
			}
		}
	}

	// 写入新 series 表。
	for _, s := range series {
		roomPool := s.roomNamePoolJSON
		if roomPool == "" {
			roomPool = "{}"
		}
		bgImages := s.bgImagesJSON
		if bgImages == "" {
			bgImages = "[]"
		}
		targets := s.targetFactionIDsJSON
		if targets == "" {
			targets = "[]"
		}
		if _, err := db.Exec(`INSERT INTO series (
			id, version, name, target_faction_ids, room_name_pool, room_background_images, status,
			contributor_player_id, contributor_name, contributor_anonymous, reviewed_by, reviewed_at,
			review_comment, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, 'approved', ?, '', ?, '', 0, '', ?, ?)`,
			s.id, legacyContentVersion(s.contentVersion), s.name, targets, roomPool, bgImages,
			s.contributorID, boolInt(s.contributorAnon), s.createdAt, now,
		); err != nil {
			return err
		}
	}

	// 写入新 sub_tasks 表，按首次出现顺序（groupOrder）保证结果确定。
	for _, gid := range groupOrder {
		g := groups[gid]
		tagJSON, err := json.Marshal(dedupStringsLocal(g.tagIDs))
		if err != nil {
			return err
		}
		variantsJSON, err := json.Marshal(g.variants)
		if err != nil {
			return err
		}
		seriesID := ""
		stepIndex := 0
		if g.hasStep {
			seriesID = g.seriesID
			stepIndex = g.stepIndex
		}
		if _, err := db.Exec(`INSERT INTO sub_tasks (
			id, version, series_id, step_index, variants, tag_ids,
			background_image, background_opacity, difficulty_order, status,
			like_count, down_count, contributor_player_id, contributor_name, contributor_anonymous,
			reviewed_by, reviewed_at, review_comment, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'approved', 0, 0, ?, '', ?, '', 0, '', ?, ?)`,
			g.id, legacyContentVersion(g.contentVersion), seriesID, stepIndex, string(variantsJSON), string(tagJSON),
			g.bgImage, g.bgOpacity, g.order, g.contributorID, boolInt(g.contributorAnon), g.createdAt, now,
		); err != nil {
			return err
		}
	}
	return nil
}

func dedupStringsLocal(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	out := make([]string, 0, len(values))
	for _, v := range values {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}
