package server

import (
	"database/sql"
)

// punishmentStore 现在只剩系列任务"完成率"的运行时统计（punishment_series_run_stats，
// 建表 DDL 见 contribution_schema.go）；任务池/系列本身的存取已经搬到 subTaskStore/
// seriesStore（sub_tasks/series 两张版本化表）。
type punishmentStore struct {
	db *sql.DB
}

type seriesRunCounts struct {
	Participants int
	PercentSum   int
}

func newPunishmentStore(db *sql.DB) *punishmentStore {
	return &punishmentStore{db: db}
}

// recordSeriesRunProgress：见 punishment_series_run_stats 建表注释。调用点在
// recordSeriesRunProgressOnClose（房间销毁时，按玩家各自的完成百分比各记一笔样本）——
// 失败只记日志，不阻断惩罚流程（次要旁路统计）。
func (ps *punishmentStore) recordSeriesRunProgress(seriesID string, version int, percent int) error {
	_, err := ps.db.Exec(`
		INSERT INTO punishment_series_run_stats (series_id, series_version, participant_count, progress_percent_sum)
		VALUES (?, ?, 1, ?)
		ON CONFLICT(series_id, series_version) DO UPDATE SET
			participant_count = participant_count + 1,
			progress_percent_sum = progress_percent_sum + excluded.progress_percent_sum`,
		seriesID, version, percent,
	)
	return err
}

// currentSeriesRunStats 批量读取系列当前版本的完成率原料。直接与 series 的最新版本 JOIN，
// 避免列表展示时先逐系列查版本、再逐系列查统计的两层 N+1。
func (ps *punishmentStore) currentSeriesRunStats(seriesIDs []string) (map[string]seriesRunCounts, error) {
	out := make(map[string]seriesRunCounts, len(seriesIDs))
	for start := 0; start < len(seriesIDs); start += contributionQueryBatchSize {
		end := min(start+contributionQueryBatchSize, len(seriesIDs))
		batch := seriesIDs[start:end]
		args := make([]any, len(batch))
		for i := range batch {
			args[i] = batch[i]
		}
		rows, err := ps.db.Query(`
			SELECT t.id, COALESCE(stats.participant_count,0), COALESCE(stats.progress_percent_sum,0)
			FROM series t
			LEFT JOIN punishment_series_run_stats stats
				ON stats.series_id = t.id AND stats.series_version = t.version
			WHERE t.id IN (`+sqlPlaceholders(len(batch))+`)
				AND t.version = (SELECT MAX(version) FROM series WHERE id = t.id)`, args...)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			var id string
			var counts seriesRunCounts
			if err := rows.Scan(&id, &counts.Participants, &counts.PercentSum); err != nil {
				_ = rows.Close()
				return nil, err
			}
			out[id] = counts
		}
		if err := rows.Close(); err != nil {
			return nil, err
		}
		if err := rows.Err(); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
