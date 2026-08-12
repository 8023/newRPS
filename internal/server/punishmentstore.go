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
	name               TEXT NOT NULL DEFAULT '',
	text               TEXT NOT NULL DEFAULT '',
	tag_ids            TEXT NOT NULL DEFAULT '[]',
	faction_ids        TEXT NOT NULL DEFAULT '[]',
	difficulty_order   INTEGER NOT NULL DEFAULT 50,
	background_images  TEXT NOT NULL DEFAULT '[]',
	background_opacity REAL NOT NULL DEFAULT 0.22,
	sort_index         INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS punishment_series (
	id                     TEXT PRIMARY KEY,
	name                   TEXT NOT NULL DEFAULT '',
	room_name_pool         TEXT NOT NULL DEFAULT '{}',
	room_background_images TEXT NOT NULL DEFAULT '[]',
	steps                  TEXT NOT NULL DEFAULT '[]',
	sort_index             INTEGER NOT NULL DEFAULT 0
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

func (ps *punishmentStore) listTasks() ([]types.PunishmentTaskConfig, error) {
	rows, err := ps.db.Query(`
		SELECT id, name, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity
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
		if err := rows.Scan(
			&t.ID, &t.Name, &t.Text, &tagJSON, &factionJSON, &t.Order, &bgJSON, &t.BackgroundOpacity,
		); err != nil {
			return nil, err
		}
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
			id, name, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity, sort_index
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
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
		if _, err := stmt.Exec(
			t.ID, t.Name, t.Text, string(tagJSON), string(factionJSON), t.Order,
			string(bgJSON), t.BackgroundOpacity, i,
		); err != nil {
			return fmt.Errorf("insert task %s: %w", t.ID, err)
		}
	}
	return tx.Commit()
}

func (ps *punishmentStore) listSeries() ([]types.PunishmentSeriesTaskConfig, error) {
	rows, err := ps.db.Query(`
		SELECT id, name, room_name_pool, room_background_images, steps
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
		var poolJSON, bgJSON, stepsJSON string
		if err := rows.Scan(&s.ID, &s.Name, &poolJSON, &bgJSON, &stepsJSON); err != nil {
			return nil, err
		}
		s.RoomNamePool = decodeRoomNamePoolJSON(poolJSON)
		s.RoomBackgroundImages = decodeStringSliceJSON(bgJSON)
		s.Steps = decodeSeriesStepsJSON(stepsJSON)
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
			id, name, room_name_pool, room_background_images, steps, sort_index
		) VALUES (?, ?, ?, ?, ?, ?)
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
		if _, err := stmt.Exec(s.ID, s.Name, string(poolJSON), string(bgJSON), string(stepsJSON), i); err != nil {
			return fmt.Errorf("insert series %s: %w", s.ID, err)
		}
	}
	return tx.Commit()
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
