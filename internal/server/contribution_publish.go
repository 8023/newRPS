package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (cs *contributionStore) reject(id, adminID, comment string) error {
	sub, err := cs.get(id)
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
	now := nowMs()
	next := types.ContributionRejected
	if sub.Status == types.ContributionRevisionPending {
		next = types.ContributionRevisionRejected
	} else if sub.Status == types.ContributionUnpublishPending {
		next, err = statusAfterCancelledUnpublishTx(tx, sub)
		if err != nil {
			return err
		}
	}
	if _, err := tx.Exec(`
		UPDATE contribution_submissions
		SET status = ?, reviewed_at = ?, reviewed_by = ?, review_comment = ?, updated_at = ?,
			unpublish_requested_at = CASE WHEN ? = ? THEN 0 ELSE unpublish_requested_at END
		WHERE id = ?`, next, now, adminID, comment, now,
		sub.Status, types.ContributionUnpublishPending, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE contribution_versions
		SET reviewed_at = ?, reviewed_by = ?, review_result = ?, review_comment = ?
		WHERE submission_id = ? AND version = ?`,
		now, adminID, types.ContributionRejected, comment, id, sub.ActiveVersion); err != nil {
		return err
	}
	return tx.Commit()
}

// updateReviewComment 只改「初审已驳回 / 修订被驳回」投稿的批注文字，不触碰状态——用于管理员事后
// 补充/修正驳回理由，与撤销驳回（revertRejection，见 contribution_save.go）是两个
// 独立操作：改批注不必连带撤销，撤销也不强制先改批注。
func (cs *contributionStore) updateReviewComment(id, comment string) error {
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
	if _, err := tx.Exec(`
		UPDATE contribution_submissions SET review_comment = ?, updated_at = ? WHERE id = ?`,
		comment, now, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE contribution_versions SET review_comment = ? WHERE submission_id = ? AND version = ?`,
		comment, id, sub.ActiveVersion); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Server) publishContribution(id, adminID, reviewedJSON, comment string) error {
	if s.contributionStore == nil || s.genderStore == nil || s.punishmentStore == nil {
		return fmt.Errorf("存储不可用")
	}
	cs := s.contributionStore
	sub, err := cs.get(id)
	if err != nil {
		return err
	}
	canApproveQueued := sub.Status == types.ContributionPending || sub.Status == types.ContributionRevisionPending
	canAdminEditPublished := sub.PublishedTargetID != "" && (sub.Status == types.ContributionApproved ||
		sub.Status == types.ContributionRevisionDraft || sub.Status == types.ContributionRevisionRejected)
	if !canApproveQueued && !canAdminEditPublished {
		return errContributionBadStatus
	}
	ver, err := cs.getVersion(id, sub.ActiveVersion)
	if err != nil {
		return err
	}
	content := reviewedJSON
	if strings.TrimSpace(content) == "" {
		if canAdminEditPublished {
			return fmt.Errorf("管理员即时修订必须提交编辑后的内容")
		}
		content = ver.Content
	}
	if len(content) > maxContributionJSON {
		return fmt.Errorf("审核内容过大")
	}
	// 管理员可以在审核页修订玩家原稿，因此不能复用“玩家提交时已经校验过”的假设；
	// 审核后的最终内容必须重新走同一套引用、形状与图片归属校验，避免构造 admin:action
	// 绕过标签/阵营存在性、图片所有权或任务字段范围检查。
	if err := s.validateContributionRefs(sub.Kind, content); err != nil {
		return err
	}
	if _, err := cs.validateAdminReviewedDraft(sub.SubmitterPlayerID, adminID, sub.Kind, content); err != nil {
		return err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	activeVersion := sub.ActiveVersion
	if sub.Status == types.ContributionApproved {
		// 管理员直接编辑当前已上线版本时，也要新建独立修订版本保留审计链；正式任务仍在
		// 本事务提交前保持旧内容在线。content 保存改稿前的有效发布内容，reviewed_content
		// 在事务末尾写管理员改稿，二者分别保留原稿与最终生效稿。
		activeVersion = sub.ActiveVersion
		if activeVersion < sub.PublishedVersion {
			activeVersion = sub.PublishedVersion
		}
		activeVersion++
		baseContent := ver.Content
		if strings.TrimSpace(ver.ReviewedContent) != "" && ver.ReviewResult == types.ContributionApproved {
			baseContent = ver.ReviewedContent
		}
		if _, err := tx.Exec(`
			INSERT INTO contribution_versions (submission_id, version, content, created_by, created_at)
			VALUES (?, ?, ?, ?, ?)`, id, activeVersion, baseContent, adminID, nowMs()); err != nil {
			return err
		}
	}
	targetID := sub.PublishedTargetID
	pubVer := sub.PublishedVersion + 1
	if pubVer < 1 {
		pubVer = 1
	}
	switch sub.Kind {
	case types.ContributionKindGender:
		draft, err := decodeGenderDraft(content)
		if err != nil {
			return err
		}
		if !s.genderFactionExists(draft.FactionID) {
			return fmt.Errorf("勾选了不存在的阵营")
		}
		norm := normalizeGenderLabel(draft.Label)
		except := targetID
		taken, err := formalGenderTakenTx(tx, draft.FactionID, norm, except)
		if err != nil {
			return err
		}
		if taken {
			return errGenderNameTaken
		}
		targetID, err = upsertGenderTx(tx, targetID, draft.Label, norm, draft.FactionID)
		if err != nil {
			return err
		}
		if err := releaseGenderClaimTx(tx, id); err != nil {
			return err
		}
	case types.ContributionKindTask:
		step, err := decodeStepDraft(content)
		if err != nil {
			return err
		}
		step.InRandomPool = true
		if step.Order < 1 || step.Order > 99 {
			return fmt.Errorf("难度必须在 1 到 99 之间")
		}
		tasks := tasksFromStepDraft(step, sub, pubVer)
		if len(tasks) == 0 {
			return fmt.Errorf("至少需要一份文案")
		}
		for i := range tasks {
			if err := s.validatePunishmentTask(tasks[i]); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DELETE FROM punishment_tasks WHERE submission_id = ?`, id); err != nil {
			return err
		}
		for i := range tasks {
			if err := upsertTaskTx(tx, tasks[i]); err != nil {
				return err
			}
		}
		targetID = tasks[0].ID
		if err := markContributionImagePublishedTx(tx, id, step.BackgroundImage); err != nil {
			return err
		}
	case types.ContributionKindSeries:
		draft, skipCoverage, err := decodeSeriesDraft(content)
		if err != nil {
			return err
		}
		if len(draft.Steps) < effectiveMinSeriesSteps(s.cfg) {
			return fmt.Errorf("系列任务至少需要 %d 步，当前只有 %d 步", effectiveMinSeriesSteps(s.cfg), len(draft.Steps))
		}
		if len(draft.Steps) > effectiveMaxSeriesSteps(s.cfg) {
			return fmt.Errorf("系列任务最多 %d 步，当前有 %d 步", effectiveMaxSeriesSteps(s.cfg), len(draft.Steps))
		}
		if !skipCoverage {
			if len(draft.TargetFactionIDs) == 0 {
				return fmt.Errorf("请选择本系列面向的阵营")
			}
			for i, step := range draft.Steps {
				if missing := missingTargetFactions(step, draft.TargetFactionIDs); len(missing) > 0 {
					return fmt.Errorf("第 %d 步未覆盖阵营：%s", i+1, strings.Join(missing, "、"))
				}
			}
		}
		series, tasks, err := decodeSeriesPublish(content, sub, pubVer)
		if err != nil {
			return err
		}
		for i := range tasks {
			if err := s.validatePunishmentTask(tasks[i]); err != nil {
				return err
			}
		}
		if _, err := tx.Exec(`DELETE FROM punishment_tasks WHERE submission_id = ?`, id); err != nil {
			return err
		}
		for i := range tasks {
			if err := upsertTaskTx(tx, tasks[i]); err != nil {
				return err
			}
		}
		if err := upsertSeriesTx(tx, series); err != nil {
			return err
		}
		for _, step := range draft.Steps {
			if err := markContributionImagePublishedTx(tx, id, step.BackgroundImage); err != nil {
				return err
			}
		}
		targetID = series.ID
	default:
		return fmt.Errorf("未知投稿类型")
	}

	now := nowMs()
	if _, err := tx.Exec(`
		UPDATE contribution_submissions
		SET status = ?, published_target_id = ?, published_version = ?, active_version = ?, reviewed_at = ?, reviewed_by = ?, review_comment = ?, updated_at = ?, unpublish_requested_at = 0
		WHERE id = ?`,
		types.ContributionApproved, targetID, pubVer, activeVersion, now, adminID, comment, now, id); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE contribution_versions
		SET reviewed_content = ?, reviewed_at = ?, reviewed_by = ?, review_result = ?, review_comment = ?
		WHERE submission_id = ? AND version = ?`,
		content, now, adminID, types.ContributionApproved, comment, id, activeVersion); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.reloadGenderCaches()
	s.reloadPunishmentCaches()
	return nil
}
