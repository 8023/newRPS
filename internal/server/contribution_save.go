package server

import (
	"database/sql"
	"fmt"

	"github.com/doumiao/newRPS/internal/types"
)

func (cs *contributionStore) saveDraft(playerID, nameSnap, kind string, anonymous bool, id string, content string) (types.ContributionSubmission, error) {
	if !validContributionKind(kind) {
		return types.ContributionSubmission{}, fmt.Errorf("未知投稿类型")
	}
	images, err := cs.inspectDraftForSave(playerID, kind, content)
	if err != nil {
		return types.ContributionSubmission{}, err
	}
	now := nowMs()
	if kind == types.ContributionKindGender {
		anonymous = false
	}
	if id == "" {
		total, err := cs.countBySubmitter(playerID)
		if err != nil {
			return types.ContributionSubmission{}, err
		}
		if total >= maxSubmissionsPerPlayer {
			return types.ContributionSubmission{}, fmt.Errorf("投稿总数已达上限")
		}
		n, err := cs.countStatus(playerID, types.ContributionDraft)
		if err != nil {
			return types.ContributionSubmission{}, err
		}
		if n >= maxDraftsPerPlayer {
			return types.ContributionSubmission{}, fmt.Errorf("草稿数量已达上限")
		}
		id = randomID()
		tx, err := cs.db.Begin()
		if err != nil {
			return types.ContributionSubmission{}, err
		}
		defer func() { _ = tx.Rollback() }()
		_, err = tx.Exec(`
			INSERT INTO contribution_submissions (
				id, kind, submitter_player_id, submitter_name_snapshot, anonymous, status,
				active_version, created_at, updated_at
			) VALUES (?, ?, ?, ?, ?, ?, 1, ?, ?)`,
			id, kind, playerID, nameSnap, boolInt(anonymous), types.ContributionDraft, now, now,
		)
		if err != nil {
			return types.ContributionSubmission{}, err
		}
		if _, err := tx.Exec(`
			INSERT INTO contribution_versions (submission_id, version, content, created_by, created_at)
			VALUES (?, 1, ?, ?, ?)`, id, content, playerID, now); err != nil {
			return types.ContributionSubmission{}, err
		}
		if err := bindDraftImagesTx(tx, images, playerID, id); err != nil {
			return types.ContributionSubmission{}, err
		}
		if err := tx.Commit(); err != nil {
			return types.ContributionSubmission{}, err
		}
		return cs.get(id)
	}
	sub, err := cs.getOwned(id, playerID)
	if err != nil {
		return sub, err
	}
	if kind != sub.Kind {
		return sub, fmt.Errorf("投稿类型不可修改")
	}
	if sub.Status != types.ContributionDraft && sub.Status != types.ContributionRejected &&
		sub.Status != types.ContributionWithdrawn && sub.Status != types.ContributionApproved &&
		sub.Status != types.ContributionRevisionDraft && sub.Status != types.ContributionRevisionRejected {
		return sub, errContributionBadStatus
	}
	tx, err := cs.db.Begin()
	if err != nil {
		return sub, err
	}
	defer func() { _ = tx.Rollback() }()
	ver := sub.ActiveVersion
	if sub.PublishedTargetID != "" {
		// 已上线内容的修订始终落在独立版本：旧正式任务/系列保持不动，继续可被玩家抽到。
		// 只有尚未开始修订时才新建版本；修订草稿/修订被驳回后继续保存则覆盖同一个未发布
		// 版本，避免每点一次保存就永久堆出无用 contribution_versions 行。
		if sub.ActiveVersion <= sub.PublishedVersion {
			ver = sub.ActiveVersion + 1
			if _, err := tx.Exec(`
				INSERT INTO contribution_versions (submission_id, version, content, created_by, created_at)
				VALUES (?, ?, ?, ?, ?)`, id, ver, content, playerID, now); err != nil {
				return sub, err
			}
		} else {
			if _, err := tx.Exec(`
				UPDATE contribution_versions SET content = ?, created_by = ?, created_at = ?
				WHERE submission_id = ? AND version = ?`, content, playerID, now, id, ver); err != nil {
				return sub, err
			}
		}
		_, err = tx.Exec(`
			UPDATE contribution_submissions
			SET active_version = ?, status = ?, updated_at = ?, anonymous = ?, submitter_name_snapshot = ? WHERE id = ?`,
			ver, types.ContributionRevisionDraft, now, boolInt(anonymous), nameSnap, id)
	} else {
		if _, err := tx.Exec(`
			UPDATE contribution_versions SET content = ?, created_at = ? WHERE submission_id = ? AND version = ?`,
			content, now, id, ver); err != nil {
			return sub, err
		}
		_, err = tx.Exec(`
			UPDATE contribution_submissions SET updated_at = ?, anonymous = ?, submitter_name_snapshot = ? WHERE id = ?`,
			now, boolInt(anonymous), nameSnap, id)
	}
	if err != nil {
		return sub, err
	}
	if err := bindDraftImagesTx(tx, images, playerID, id); err != nil {
		return sub, err
	}
	if err := tx.Commit(); err != nil {
		return sub, err
	}
	return cs.get(id)
}

func bindDraftImagesTx(tx *sql.Tx, images []string, playerID, submissionID string) error {
	// 共建图片属于监管审计材料：上传后永久保留，换图、撤稿、驳回和下架都不能解绑或删除
	// 旧图片。submission_id/published 表示它曾归属于哪份投稿、是否曾正式发布，而不是当前
	// 页面是否仍在引用。日后如需清理，只能由站长另行编写的审计清理工具显式执行。
	for _, path := range images {
		res, err := tx.Exec(`
			UPDATE contribution_images SET submission_id = ?
			WHERE path = ? AND uploader_player_id = ? AND (submission_id = '' OR submission_id = ?)`,
			submissionID, path, playerID, submissionID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n == 0 {
			return fmt.Errorf("任务背景图已被其它投稿占用")
		}
	}
	return nil
}

func (cs *contributionStore) submit(playerID, id string) error {
	sub, err := cs.getOwned(id, playerID)
	if err != nil {
		return err
	}
	pending, err := cs.countStatus(playerID, types.ContributionPending)
	if err != nil {
		return err
	}
	revPending, err := cs.countStatus(playerID, types.ContributionRevisionPending)
	if err != nil {
		return err
	}
	if pending+revPending >= maxPendingPerPlayer {
		return fmt.Errorf("待审投稿数量已达上限")
	}
	next := types.ContributionPending
	if sub.PublishedTargetID != "" {
		if sub.Status != types.ContributionRevisionDraft && sub.Status != types.ContributionRevisionRejected {
			return errContributionBadStatus
		}
		if sub.ActiveVersion <= sub.PublishedVersion {
			return fmt.Errorf("没有待提交的修订内容")
		}
		next = types.ContributionRevisionPending
	} else if sub.Status != types.ContributionDraft && sub.Status != types.ContributionRejected && sub.Status != types.ContributionWithdrawn {
		return errContributionBadStatus
	}
	now := nowMs()
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if sub.Kind == types.ContributionKindGender {
		ver, err := loadVersionTx(tx, id, sub.ActiveVersion)
		if err != nil {
			return err
		}
		draft, err := decodeGenderDraft(ver.Content)
		if err != nil {
			return err
		}
		if err := claimGenderNameTx(tx, draft.FactionID, normalizeGenderLabel(draft.Label), id, sub.PublishedTargetID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		UPDATE contribution_submissions SET status = ?, submitted_at = ?, updated_at = ? WHERE id = ?`,
		next, now, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

func loadVersionTx(tx *sql.Tx, submissionID string, version int) (types.ContributionVersion, error) {
	var v types.ContributionVersion
	err := tx.QueryRow(`
		SELECT id, submission_id, version, content, created_by, created_at,
			reviewed_content, reviewed_at, reviewed_by, review_result, review_comment
		FROM contribution_versions WHERE submission_id = ? AND version = ?`,
		submissionID, version,
	).Scan(&v.ID, &v.SubmissionID, &v.Version, &v.Content, &v.CreatedBy, &v.CreatedAt,
		&v.ReviewedContent, &v.ReviewedAt, &v.ReviewedBy, &v.ReviewResult, &v.ReviewComment)
	return v, err
}

func claimGenderNameTx(tx *sql.Tx, factionID, normalized, submissionID, exceptGenderID string) error {
	if normalized == "" {
		return fmt.Errorf("性别名称无效")
	}
	var n int
	if err := tx.QueryRow(
		`SELECT COUNT(*) FROM gender_options WHERE faction_id = ? AND normalized_label = ? AND id != ?`,
		factionID, normalized, exceptGenderID,
	).Scan(&n); err != nil {
		return err
	}
	if n > 0 {
		return errGenderNameTaken
	}
	var claimingSubID string
	err := tx.QueryRow(
		`SELECT submission_id FROM contribution_gender_claims WHERE faction_id = ? AND normalized_label = ?`,
		factionID, normalized,
	).Scan(&claimingSubID)
	if err != nil && err != sql.ErrNoRows {
		return err
	}
	if err == nil && claimingSubID != submissionID {
		return errGenderNameTaken
	}
	if _, err := tx.Exec(
		`INSERT INTO contribution_gender_claims (faction_id, normalized_label, submission_id) VALUES (?, ?, ?)
		 ON CONFLICT(faction_id, normalized_label) DO UPDATE SET submission_id = excluded.submission_id`,
		factionID, normalized, submissionID,
	); err != nil {
		return errGenderNameTaken
	}
	return nil
}

func releaseGenderClaimTx(tx *sql.Tx, submissionID string) error {
	_, err := tx.Exec(`DELETE FROM contribution_gender_claims WHERE submission_id = ?`, submissionID)
	return err
}

// revertRejection 是「撤销驳回」：把初审驳回或修订驳回的投稿重新放回对应审批队列，供管理员反悔
// 误判时使用。与 submit() 一样，性别投稿要重新占住「阵营+名称」的名字位——驳回时
// releaseGenderClaimTx 已经把这个名字放出去了，这段时间内可能被别的投稿占用，占用
// 冲突时直接报错阻止撤销，逼管理员先处理冲突而不是静默产生两个同名性别。
func (cs *contributionStore) revertRejection(id string) error {
	sub, err := cs.get(id)
	if err != nil {
		return err
	}
	if sub.Status != types.ContributionRejected && sub.Status != types.ContributionRevisionRejected {
		return errContributionBadStatus
	}
	now := nowMs()
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if sub.Kind == types.ContributionKindGender {
		ver, err := loadVersionTx(tx, id, sub.ActiveVersion)
		if err != nil {
			return err
		}
		draft, err := decodeGenderDraft(ver.Content)
		if err != nil {
			return err
		}
		if err := claimGenderNameTx(tx, draft.FactionID, normalizeGenderLabel(draft.Label), id, sub.PublishedTargetID); err != nil {
			return err
		}
	}
	next := types.ContributionPending
	if sub.Status == types.ContributionRevisionRejected {
		next = types.ContributionRevisionPending
	}
	if _, err := tx.Exec(`
		UPDATE contribution_submissions SET status = ?, updated_at = ? WHERE id = ?`,
		next, now, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (cs *contributionStore) withdraw(playerID, id string) error {
	sub, err := cs.getOwned(id, playerID)
	if err != nil {
		return err
	}
	if sub.Status != types.ContributionPending && sub.Status != types.ContributionRevisionPending && sub.Status != types.ContributionUnpublishPending {
		return errContributionBadStatus
	}
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := releaseGenderClaimTx(tx, id); err != nil {
		return err
	}
	next := types.ContributionWithdrawn
	if sub.Status == types.ContributionRevisionPending {
		next = types.ContributionRevisionDraft
	} else if sub.Status == types.ContributionUnpublishPending {
		next, err = statusAfterCancelledUnpublishTx(tx, sub)
		if err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		UPDATE contribution_submissions SET status = ?, updated_at = ?,
			unpublish_requested_at = CASE WHEN ? = ? THEN 0 ELSE unpublish_requested_at END
		WHERE id = ?`, next, nowMs(), sub.Status, types.ContributionUnpublishPending, id); err != nil {
		return err
	}
	return tx.Commit()
}

func (cs *contributionStore) requestUnpublish(playerID, id string) error {
	sub, err := cs.getOwned(id, playerID)
	if err != nil {
		return err
	}
	if sub.PublishedTargetID == "" || (sub.Status != types.ContributionApproved &&
		sub.Status != types.ContributionRevisionDraft && sub.Status != types.ContributionRevisionRejected) {
		return errContributionBadStatus
	}
	_, err = cs.db.Exec(
		`UPDATE contribution_submissions SET status = ?, unpublish_requested_at = ?, updated_at = ? WHERE id = ?`,
		types.ContributionUnpublishPending, nowMs(), nowMs(), id,
	)
	return err
}

func statusAfterCancelledUnpublishTx(tx *sql.Tx, sub types.ContributionSubmission) (string, error) {
	if sub.ActiveVersion <= sub.PublishedVersion {
		return types.ContributionApproved, nil
	}
	ver, err := loadVersionTx(tx, sub.ID, sub.ActiveVersion)
	if err != nil {
		return "", err
	}
	if ver.ReviewResult == types.ContributionRejected {
		return types.ContributionRevisionRejected, nil
	}
	return types.ContributionRevisionDraft, nil
}
