package server

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/mattn/go-sqlite3"

	"github.com/doumiao/newRPS/internal/types"
)

func (cs *contributionStore) insertRoundParticipants(roundID, roomID string, parts []types.RoundParticipant) error {
	if len(parts) == 0 {
		return nil
	}
	tx, err := cs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	stmt, err := tx.Prepare(`
		INSERT OR IGNORE INTO contribution_round_participants (round_id, room_id, player_id, role)
		VALUES (?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for _, p := range parts {
		if _, err := stmt.Exec(roundID, roomID, p.PlayerID, p.Role); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (cs *contributionStore) isRoundParticipant(roundID, playerID string) (bool, error) {
	var n int
	err := cs.db.QueryRow(
		`SELECT COUNT(*) FROM contribution_round_participants WHERE round_id = ? AND player_id = ?`,
		roundID, playerID,
	).Scan(&n)
	return n > 0, err
}

func (cs *contributionStore) castVote(roundID, eventID, voterID, targetKind, targetID string, targetVer, vote int) error {
	if vote != types.ContributionVoteUp && vote != types.ContributionVoteDown {
		return fmt.Errorf("无效评价")
	}
	now := nowMs()
	_, err := cs.db.Exec(`
		INSERT INTO contribution_votes (
			round_id, punishment_event_id, voter_player_id, target_kind, target_id, target_version, vote, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		roundID, eventID, voterID, targetKind, targetID, targetVer, vote, now, now,
	)
	if err != nil {
		var sqliteErr sqlite3.Error
		if errors.As(err, &sqliteErr) && sqliteErr.Code == sqlite3.ErrConstraint {
			return errVoteDuplicate
		}
		return fmt.Errorf("cast vote: %w", err)
	}
	return nil
}

func (cs *contributionStore) voteStats(kind, id string, version int) (types.VoteStats, error) {
	var st types.VoteStats
	st.TargetKind = kind
	st.TargetID = id
	st.TargetVer = version
	err := cs.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN vote = 1 THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN vote = -1 THEN 1 ELSE 0 END), 0)
		FROM contribution_votes
		WHERE target_kind = ? AND target_id = ? AND target_version = ?`,
		kind, id, version,
	).Scan(&st.LikeCount, &st.DownCount)
	if err != nil {
		return st, err
	}
	st.VoteCount = st.LikeCount + st.DownCount
	if st.VoteCount > 0 {
		ratio := int(float64(st.LikeCount)/float64(st.VoteCount)*100 + 0.5)
		st.RealRatio = &ratio
		st.DisplayRatio = &ratio
	}
	var display int
	var adminID, note string
	var updated int64
	err = cs.db.QueryRow(`
		SELECT display_ratio, admin_id, admin_note, updated_at
		FROM contribution_vote_overrides
		WHERE target_kind = ? AND target_id = ? AND target_version = ?`,
		kind, id, version,
	).Scan(&display, &adminID, &note, &updated)
	if err == sql.ErrNoRows {
		return st, nil
	}
	if err != nil {
		return st, err
	}
	st.OverrideOn = true
	st.OverrideBy = adminID
	st.OverrideNote = note
	st.OverrideAt = updated
	st.DisplayRatio = &display
	return st, nil
}

func (cs *contributionStore) setVoteOverride(kind, id string, version, ratio int, adminID, note string) error {
	if ratio < 0 || ratio > 100 {
		return fmt.Errorf("覆盖点赞率必须在 0 到 100")
	}
	now := nowMs()
	_, err := cs.db.Exec(`
		INSERT INTO contribution_vote_overrides (
			target_kind, target_id, target_version, display_ratio, admin_id, admin_note, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(target_kind, target_id, target_version) DO UPDATE SET
			display_ratio = excluded.display_ratio,
			admin_id = excluded.admin_id,
			admin_note = excluded.admin_note,
			updated_at = excluded.updated_at`,
		kind, id, version, ratio, adminID, note, now, now,
	)
	return err
}

func (cs *contributionStore) clearVoteOverride(kind, id string, version int) error {
	_, err := cs.db.Exec(
		`DELETE FROM contribution_vote_overrides WHERE target_kind = ? AND target_id = ? AND target_version = ?`,
		kind, id, version,
	)
	return err
}

// submissionVoteAggregate 汇总一份投稿名下所有 task_group（独立任务恰好 1 个；系列是每步
// 1 个）各自的点赞情况：LikeCount/DownCount/VoteCount 是所有组的原始计数直接相加（用于展示
// "共 N 人评价"），RealRatio/DisplayRatio 则是各组自己的点赞率取算术平均——只把"有票"的组
// 计入分母，未投票的步骤不拖累均值（RealRatio 依据组自身 VoteCount>0；DisplayRatio 额外把
// "原始 0 票但管理员设了覆盖值"的组也计入，因为那个组确实有一个该展示的比例）。任务/系列
// 两种投稿类型走同一套查询，不用再区分。
func (cs *contributionStore) submissionVoteAggregate(submissionID string, version int) (types.VoteStats, error) {
	st := types.VoteStats{TargetKind: types.ContributionKindTask, TargetID: submissionID, TargetVer: version}
	// 一次按 task_group 聚合票数并 LEFT JOIN 覆盖值。旧实现先查组列表、再对每组调用
	// voteStats，系列 N 步就是 N+1 次 SQLite 查询；这些 RPC 又运行在 s.mu 内，畸形长系列
	// 会直接拖住实时游戏。这里保持“各组点赞率算术平均”的原口径，但恒定只发一条查询。
	rows, err := cs.db.Query(`
		WITH target_groups AS (
			SELECT DISTINCT task_group_id
			FROM punishment_tasks
			WHERE submission_id = ? AND vote_version = ? AND task_group_id != ''
		), vote_counts AS (
			SELECT g.task_group_id,
				COALESCE(SUM(CASE WHEN v.vote = 1 THEN 1 ELSE 0 END), 0) AS likes,
				COALESCE(SUM(CASE WHEN v.vote = -1 THEN 1 ELSE 0 END), 0) AS downs
			FROM target_groups g
			LEFT JOIN contribution_votes v
				ON v.target_kind = ? AND v.target_id = g.task_group_id AND v.target_version = ?
			GROUP BY g.task_group_id
		)
		SELECT c.likes, c.downs, o.display_ratio
		FROM vote_counts c
		LEFT JOIN contribution_vote_overrides o
			ON o.target_kind = ? AND o.target_id = c.task_group_id AND o.target_version = ?`,
		submissionID, version,
		types.ContributionKindTask, version,
		types.ContributionKindTask, version,
	)
	if err != nil {
		return st, err
	}
	defer rows.Close()
	var realSum, realN, displaySum, displayN int
	for rows.Next() {
		var likes, downs int
		var override sql.NullInt64
		if err := rows.Scan(&likes, &downs, &override); err != nil {
			return st, err
		}
		st.LikeCount += likes
		st.DownCount += downs
		votes := likes + downs
		if votes > 0 {
			ratio := int(float64(likes)/float64(votes)*100 + 0.5)
			realSum += ratio
			realN++
			if !override.Valid {
				displaySum += ratio
				displayN++
			}
		}
		if override.Valid {
			displaySum += int(override.Int64)
			displayN++
		}
	}
	if err := rows.Err(); err != nil {
		return st, err
	}
	st.VoteCount = st.LikeCount + st.DownCount
	if realN > 0 {
		ratio := int(float64(realSum)/float64(realN) + 0.5)
		st.RealRatio = &ratio
	}
	if displayN > 0 {
		ratio := int(float64(displaySum)/float64(displayN) + 0.5)
		st.DisplayRatio = &ratio
	} else {
		st.DisplayRatio = st.RealRatio
	}
	return st, nil
}

func (cs *contributionStore) voteTargetExists(kind, id string, version int) (bool, error) {
	if kind != types.ContributionKindTask || id == "" || version < 1 {
		return false, nil
	}
	var n int
	err := cs.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM punishment_tasks
			WHERE task_group_id = ? AND vote_version = ?
			UNION ALL
			SELECT 1 FROM punishment_events
			WHERE formal_task_id = ? AND formal_task_version = ?
		)`, id, version, id, version).Scan(&n)
	return n != 0, err
}

func (cs *contributionStore) myVote(roundID, voterID, kind, id string) (int, error) {
	var v int
	err := cs.db.QueryRow(
		`SELECT vote FROM contribution_votes WHERE round_id = ? AND voter_player_id = ? AND target_kind = ? AND target_id = ?`,
		roundID, voterID, kind, id,
	).Scan(&v)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return v, err
}
