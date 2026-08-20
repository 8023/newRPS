package server

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/doumiao/newRPS/internal/types"
)

// punishmentStore 任务池 + 系列任务的 SQLite 持久化。
// 运行时权威读路径是 Server 内存缓存；本 store 只负责 list/replace 与启动加载。
type punishmentStore struct {
	db *sql.DB
}

const punishmentPoolSchema = `
CREATE TABLE IF NOT EXISTS punishment_tasks (
	id                 TEXT PRIMARY KEY,
	text               TEXT NOT NULL DEFAULT '',
	tag_ids            TEXT NOT NULL DEFAULT '[]',
	faction_ids        TEXT NOT NULL DEFAULT '[]',
	difficulty_order   INTEGER NOT NULL DEFAULT 50,
	background_images  TEXT NOT NULL DEFAULT '[]',
	background_opacity REAL NOT NULL DEFAULT 0.22,
	sort_index         INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	content_version INTEGER NOT NULL DEFAULT 0,
	vote_version INTEGER NOT NULL DEFAULT 0,
	submission_id TEXT NOT NULL DEFAULT '',
	task_group_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS punishment_series (
	id                     TEXT PRIMARY KEY,
	name                   TEXT NOT NULL DEFAULT '',
	room_name_pool         TEXT NOT NULL DEFAULT '{}',
	room_background_images TEXT NOT NULL DEFAULT '[]',
	steps                  TEXT NOT NULL DEFAULT '[]',
	sort_index             INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	content_version INTEGER NOT NULL DEFAULT 0,
	vote_version INTEGER NOT NULL DEFAULT 0,
	submission_id TEXT NOT NULL DEFAULT '',
	target_faction_ids TEXT NOT NULL DEFAULT '[]',
	created_at INTEGER NOT NULL DEFAULT 0
);
-- punishment_series_run_stats：系列任务"完成率"的运行时统计——房间销毁时，把房间里每个
-- 至少走完 1 步的玩家各自的"已走步数 / 系列总步数"百分比记一笔样本；participant_count
-- 是样本数，progress_percent_sum 是样本值之和，完成率 = sum/count（算术平均，不是"是否
-- 走到最后一步"的二元判定）。按 series_id+series_version 分桶，系列改版重发后从 0 重新计
-- （对齐点赞按 vote_version 分桶重置的既有约定）。全新表，不需要走版本化迁移，随
-- CREATE TABLE IF NOT EXISTS 自动建好。见 punishment.go 的 recordSeriesRunProgressOnClose。
CREATE TABLE IF NOT EXISTS punishment_series_run_stats (
	series_id      TEXT NOT NULL,
	series_version INTEGER NOT NULL,
	participant_count     INTEGER NOT NULL DEFAULT 0,
	progress_percent_sum  INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (series_id, series_version)
);
`

func newPunishmentStore(db *sql.DB) *punishmentStore {
	return &punishmentStore{db: db}
}

func (ps *punishmentStore) countTasks() (int, error) {
	var n int
	err := ps.db.QueryRow(`SELECT COUNT(*) FROM punishment_tasks`).Scan(&n)
	return n, err
}

// randomTaskGroupCount 统计当前随机任务候选池的"条数"：按 task_group_id 去重，只算
// difficulty_order <> -1（会真正进入随机候选池）的任务，与 candidateTasksForRandomDifficulty
// 同一口径——独立发布的随机任务与"同时发布到随机任务"的系列步骤都算在内，仅供系列内部
// 引用（order=-1）的步骤不算。后台共建审核总览 + 数据分析「随机任务」增长图共用这个口径。
func (ps *punishmentStore) randomTaskGroupCount() (int, error) {
	var n int
	err := ps.db.QueryRow(`SELECT COUNT(DISTINCT task_group_id) FROM punishment_tasks WHERE difficulty_order <> -1 AND task_group_id != ''`).Scan(&n)
	return n, err
}

// seriesGroupCount 统计当前系列任务"组数"：punishment_series 本就一行一系列，直接计数。
func (ps *punishmentStore) seriesGroupCount() (int, error) {
	var n int
	err := ps.db.QueryRow(`SELECT COUNT(*) FROM punishment_series`).Scan(&n)
	return n, err
}

func (ps *punishmentStore) listTasks() ([]types.PunishmentTaskConfig, error) {
	rows, err := ps.db.Query(`
		SELECT id, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id, task_group_id, created_at
		FROM punishment_tasks
		ORDER BY sort_index ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]types.PunishmentTaskConfig, 0)
	for rows.Next() {
		var t types.PunishmentTaskConfig
		var tagJSON, factionJSON, bgJSON string
		var anon int
		if err := rows.Scan(
			&t.ID, &t.Text, &tagJSON, &factionJSON, &t.Order, &bgJSON, &t.BackgroundOpacity,
			&t.ContributorPlayerID, &anon, &t.ContentVersion, &t.VoteVersion, &t.SubmissionID, &t.TaskGroupID, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		t.ContributorAnonymous = anon != 0
		t.TagIDs = decodeStringSliceJSON(tagJSON)
		t.FactionIDs = decodeStringSliceJSON(factionJSON)
		t.BackgroundImages = decodeStringSliceJSON(bgJSON)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (ps *punishmentStore) replaceTasks(tasks []types.PunishmentTaskConfig) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM punishment_tasks`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO punishment_tasks (
			id, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity, sort_index,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id, task_group_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, t := range tasks {
		tagJSON, err := json.Marshal(nonNilStrings(t.TagIDs))
		if err != nil {
			return err
		}
		factionJSON, err := json.Marshal(nonNilStrings(t.FactionIDs))
		if err != nil {
			return err
		}
		bgJSON, err := json.Marshal(nonNilStrings(t.BackgroundImages))
		if err != nil {
			return err
		}
		anon := 0
		if t.ContributorAnonymous {
			anon = 1
		}
		if _, err := stmt.Exec(
			t.ID, t.Text, string(tagJSON), string(factionJSON), t.Order,
			string(bgJSON), t.BackgroundOpacity, i,
			t.ContributorPlayerID, anon, t.ContentVersion, t.VoteVersion, t.SubmissionID, t.TaskGroupID, t.CreatedAt,
		); err != nil {
			return fmt.Errorf("insert task %s: %w", t.ID, err)
		}
	}
	return tx.Commit()
}

func (ps *punishmentStore) listSeries() ([]types.PunishmentSeriesTaskConfig, error) {
	rows, err := ps.db.Query(`
		SELECT id, name, room_name_pool, room_background_images, steps,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id,
			target_faction_ids, created_at
		FROM punishment_series
		ORDER BY sort_index ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]types.PunishmentSeriesTaskConfig, 0)
	for rows.Next() {
		var s types.PunishmentSeriesTaskConfig
		var poolJSON, bgJSON, stepsJSON, targetJSON string
		var anon int
		if err := rows.Scan(&s.ID, &s.Name, &poolJSON, &bgJSON, &stepsJSON,
			&s.ContributorPlayerID, &anon, &s.ContentVersion, &s.VoteVersion, &s.SubmissionID, &targetJSON, &s.CreatedAt); err != nil {
			return nil, err
		}
		s.ContributorAnonymous = anon != 0
		s.RoomNamePool = decodeRoomNamePoolJSON(poolJSON)
		s.RoomBackgroundImages = decodeStringSliceJSON(bgJSON)
		s.Steps = decodeSeriesStepsJSON(stepsJSON)
		s.TargetFactionIDs = decodeStringSliceJSON(targetJSON)
		out = append(out, s)
	}
	return out, rows.Err()
}

func (ps *punishmentStore) replaceSeries(series []types.PunishmentSeriesTaskConfig) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.Exec(`DELETE FROM punishment_series`); err != nil {
		return err
	}
	stmt, err := tx.Prepare(`
		INSERT INTO punishment_series (
			id, name, room_name_pool, room_background_images, steps, sort_index,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id,
			target_faction_ids, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer stmt.Close()
	for i, s := range series {
		poolJSON, err := json.Marshal(s.RoomNamePool)
		if err != nil {
			return err
		}
		if s.RoomNamePool == nil {
			poolJSON = []byte("{}")
		}
		bgJSON, err := json.Marshal(nonNilStrings(s.RoomBackgroundImages))
		if err != nil {
			return err
		}
		stepsJSON, err := json.Marshal(nonNilSteps(s.Steps))
		if err != nil {
			return err
		}
		targetJSON, err := json.Marshal(nonNilStrings(s.TargetFactionIDs))
		if err != nil {
			return err
		}
		anon := 0
		if s.ContributorAnonymous {
			anon = 1
		}
		if _, err := stmt.Exec(s.ID, s.Name, string(poolJSON), string(bgJSON), string(stepsJSON), i,
			s.ContributorPlayerID, anon, s.ContentVersion, s.VoteVersion, s.SubmissionID, string(targetJSON), s.CreatedAt); err != nil {
			return fmt.Errorf("insert series %s: %w", s.ID, err)
		}
	}
	return tx.Commit()
}

func upsertTaskTx(tx *sql.Tx, t types.PunishmentTaskConfig) error {
	tagJSON, err := json.Marshal(nonNilStrings(t.TagIDs))
	if err != nil {
		return err
	}
	factionJSON, err := json.Marshal(nonNilStrings(t.FactionIDs))
	if err != nil {
		return err
	}
	bgJSON, err := json.Marshal(nonNilStrings(t.BackgroundImages))
	if err != nil {
		return err
	}
	anon := 0
	if t.ContributorAnonymous {
		anon = 1
	}
	// sort_index：追加到当前列表末尾（而不是硬编码 0），避免每次审批通过的投稿都插到
	// 管理员手工排好的顺序最前面——UPDATE 分支不写 sort_index，已有任务的顺序不受影响。
	// created_at 同理不写进 UPDATE 分支：task_group_id 每次发布都是新生成的随机 ID
	// （见 tasksFromStepDraft），ON CONFLICT 实际上几乎不会命中，这里只是与
	// upsertSeriesTx 保持同样的"首次写入即定、修订不覆盖"防御性写法。
	_, err = tx.Exec(`
		INSERT INTO punishment_tasks (
			id, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity, sort_index,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id, task_group_id, created_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, (SELECT COALESCE(MAX(sort_index), -1) + 1 FROM punishment_tasks), ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			text = excluded.text,
			tag_ids = excluded.tag_ids,
			faction_ids = excluded.faction_ids,
			difficulty_order = excluded.difficulty_order,
			background_images = excluded.background_images,
			background_opacity = excluded.background_opacity,
			contributor_player_id = excluded.contributor_player_id,
			contributor_anonymous = excluded.contributor_anonymous,
			content_version = excluded.content_version,
			vote_version = excluded.vote_version,
			submission_id = excluded.submission_id,
			task_group_id = excluded.task_group_id`,
		t.ID, t.Text, string(tagJSON), string(factionJSON), t.Order, string(bgJSON), t.BackgroundOpacity,
		t.ContributorPlayerID, anon, t.ContentVersion, t.VoteVersion, t.SubmissionID, t.TaskGroupID, t.CreatedAt,
	)
	return err
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

func upsertSeriesTx(tx *sql.Tx, s types.PunishmentSeriesTaskConfig) error {
	poolJSON, err := json.Marshal(s.RoomNamePool)
	if err != nil {
		return err
	}
	if s.RoomNamePool == nil {
		poolJSON = []byte("{}")
	}
	bgJSON, err := json.Marshal(nonNilStrings(s.RoomBackgroundImages))
	if err != nil {
		return err
	}
	stepsJSON, err := json.Marshal(nonNilSteps(s.Steps))
	if err != nil {
		return err
	}
	targetJSON, err := json.Marshal(nonNilStrings(s.TargetFactionIDs))
	if err != nil {
		return err
	}
	anon := 0
	if s.ContributorAnonymous {
		anon = 1
	}
	// sort_index：同 upsertTaskTx，追加到列表末尾，UPDATE 分支不改已有顺序。created_at
	// 同理故意不进 UPDATE 分支：series ID 跨修订稳定复用（decodeSeriesPublish 沿用
	// sub.PublishedTargetID），修订重审会真正命中 ON CONFLICT，不写它才能让"首次发布日期"
	// 在多次修订后仍保持不变，供数据分析「系列任务」增长图使用。
	_, err = tx.Exec(`
		INSERT INTO punishment_series (
			id, name, room_name_pool, room_background_images, steps, sort_index,
			contributor_player_id, contributor_anonymous, content_version, vote_version, submission_id,
			target_faction_ids, created_at
		) VALUES (?, ?, ?, ?, ?, (SELECT COALESCE(MAX(sort_index), -1) + 1 FROM punishment_series), ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			name = excluded.name,
			room_name_pool = excluded.room_name_pool,
			room_background_images = excluded.room_background_images,
			steps = excluded.steps,
			contributor_player_id = excluded.contributor_player_id,
			contributor_anonymous = excluded.contributor_anonymous,
			content_version = excluded.content_version,
			vote_version = excluded.vote_version,
			submission_id = excluded.submission_id,
			target_faction_ids = excluded.target_faction_ids`,
		s.ID, s.Name, string(poolJSON), string(bgJSON), string(stepsJSON),
		s.ContributorPlayerID, anon, s.ContentVersion, s.VoteVersion, s.SubmissionID, string(targetJSON), s.CreatedAt,
	)
	return err
}

func nonNilStrings(s []string) []string {
	if s == nil {
		return []string{}
	}
	return s
}

func nonNilSteps(steps []types.PunishmentSeriesStep) []types.PunishmentSeriesStep {
	if steps == nil {
		return []types.PunishmentSeriesStep{}
	}
	out := make([]types.PunishmentSeriesStep, len(steps))
	for i, st := range steps {
		out[i] = types.PunishmentSeriesStep{TaskIDs: nonNilStrings(st.TaskIDs)}
	}
	return out
}

func decodeStringSliceJSON(raw string) []string {
	if raw == "" {
		return []string{}
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []string{}
	}
	return out
}

func decodeRoomNamePoolJSON(raw string) *types.RoomNamePool {
	if raw == "" || raw == "{}" || raw == "null" {
		return &types.RoomNamePool{}
	}
	var pool types.RoomNamePool
	if err := json.Unmarshal([]byte(raw), &pool); err != nil {
		return &types.RoomNamePool{}
	}
	if pool.Adjectives == nil {
		pool.Adjectives = []string{}
	}
	if pool.Subjects == nil {
		pool.Subjects = []string{}
	}
	if pool.RoomWords == nil {
		pool.RoomWords = []string{}
	}
	return &pool
}

func decodeSeriesStepsJSON(raw string) []types.PunishmentSeriesStep {
	if raw == "" {
		return []types.PunishmentSeriesStep{}
	}
	var out []types.PunishmentSeriesStep
	if err := json.Unmarshal([]byte(raw), &out); err != nil || out == nil {
		return []types.PunishmentSeriesStep{}
	}
	for i := range out {
		if out[i].TaskIDs == nil {
			out[i].TaskIDs = []string{}
		}
	}
	return out
}
