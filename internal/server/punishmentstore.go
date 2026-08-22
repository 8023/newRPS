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

// seriesRunStats 返回某个系列版本累计的样本数与百分比之和；调用方自行相除得算术平均
// （样本数为 0 时不要除，避免除零——见 seriesCompletionStats）。未命中的桶返回 0/0。
func (ps *punishmentStore) seriesRunStats(seriesID string, version int) (participants, percentSum int, err error) {
	err = ps.db.QueryRow(`
		SELECT participant_count, progress_percent_sum FROM punishment_series_run_stats
		WHERE series_id = ? AND series_version = ?`,
		seriesID, version,
	).Scan(&participants, &percentSum)
	if err == sql.ErrNoRows {
		return 0, 0, nil
	}
	return participants, percentSum, err
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}
