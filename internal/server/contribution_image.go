package server

import (
	"database/sql"
	"strings"
)

func (cs *contributionStore) recordImage(path, uploader, submissionID string) error {
	_, err := cs.db.Exec(`
		INSERT INTO contribution_images (path, uploader_player_id, created_at, submission_id, published)
		VALUES (?, ?, ?, ?, 0)`,
		path, uploader, nowMs(), submissionID,
	)
	return err
}

func (cs *contributionStore) imageOwnedBy(path, playerID string) (bool, error) {
	var n int
	err := cs.db.QueryRow(
		`SELECT COUNT(*) FROM contribution_images WHERE path = ? AND uploader_player_id = ?`,
		path, playerID,
	).Scan(&n)
	return n > 0, err
}

func markContributionImagePublishedTx(tx *sql.Tx, submissionID, path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	_, err := tx.Exec(`UPDATE contribution_images SET published = 1, submission_id = ? WHERE path = ?`, submissionID, path)
	return err
}
