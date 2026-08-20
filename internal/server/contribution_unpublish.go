package server

import (
	"database/sql"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) unpublishContribution(id, adminID, comment string) error {
	sub, err := s.contributionStore.get(id)
	if err != nil {
		return err
	}
	// 只要确实存在正式发布目标就允许管理员下架；修订草稿/审核中/被驳回时旧版本仍在线，
	// 因此同样属于可下架状态。未发布过的初审稿 published_target_id 为空，明确拒绝。
	if sub.PublishedTargetID == "" {
		return errContributionBadStatus
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := removePublishedContributionTx(tx, sub); err != nil {
		return err
	}
	if _, err := tx.Exec(`
		UPDATE contribution_submissions SET status = ?, reviewed_at = ?, reviewed_by = ?, review_comment = ?, updated_at = ?, unpublish_requested_at = 0
		WHERE id = ?`, types.ContributionWithdrawn, nowMs(), adminID, comment, nowMs(), id); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	s.reloadGenderCaches()
	s.reloadPunishmentCaches()
	return nil
}

func removePublishedContributionTx(tx *sql.Tx, sub types.ContributionSubmission) error {
	switch sub.Kind {
	case types.ContributionKindGender:
		return deleteGenderTx(tx, sub.PublishedTargetID)
	case types.ContributionKindTask:
		_, err := tx.Exec(`DELETE FROM punishment_tasks WHERE submission_id = ? OR id = ?`, sub.ID, sub.PublishedTargetID)
		return err
	case types.ContributionKindSeries:
		if _, err := tx.Exec(`DELETE FROM punishment_tasks WHERE submission_id = ?`, sub.ID); err != nil {
			return err
		}
		_, err := tx.Exec(`DELETE FROM punishment_series WHERE submission_id = ? OR id = ?`, sub.ID, sub.PublishedTargetID)
		return err
	default:
		return nil
	}
}
