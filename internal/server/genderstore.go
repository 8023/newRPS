package server

import (
	"database/sql"
	"fmt"

	"github.com/doumiao/newRPS/internal/types"
)

// genderSchema：性别与阵营预设表。运行时权威在库里，config/json/genders.json 与
// gender-factions.json 只作空库首次导入种子（见 gender_import.go）。这两张表与共建投稿
// 系统无关（性别投稿功能已下线，见 CLAUDE.md「共建投稿」一节）——后台「性别与阵营」走
// gendersGet/gendersSave 直接增删改，不经过 sub_tasks/series 那套版本化投稿流程。
const genderSchema = `
CREATE TABLE IF NOT EXISTS gender_factions (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL DEFAULT '',
	text_color TEXT NOT NULL DEFAULT '',
	background_color TEXT NOT NULL DEFAULT '',
	border_color TEXT NOT NULL DEFAULT '',
	task_group TEXT NOT NULL DEFAULT 'default',
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS gender_options (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL DEFAULT '',
	faction_id TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (faction_id) REFERENCES gender_factions(id),
	UNIQUE (faction_id, label)
);
`

type genderStore struct {
	db *sql.DB
}

func newGenderStore(db *sql.DB) *genderStore {
	return &genderStore{db: db}
}

func (gs *genderStore) countFactions() (int, error) {
	var n int
	err := gs.db.QueryRow(`SELECT COUNT(*) FROM gender_factions`).Scan(&n)
	return n, err
}

func (gs *genderStore) countGenders() (int, error) {
	var n int
	err := gs.db.QueryRow(`SELECT COUNT(*) FROM gender_options`).Scan(&n)
	return n, err
}

func (gs *genderStore) listFactions() ([]types.GenderFaction, error) {
	rows, err := gs.db.Query(`
		SELECT id, label, text_color, background_color, border_color, task_group
		FROM gender_factions ORDER BY sort_index ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]types.GenderFaction, 0)
	for rows.Next() {
		var f types.GenderFaction
		if err := rows.Scan(&f.ID, &f.Label, &f.TextColor, &f.BackgroundColor, &f.BorderColor, &f.TaskGroup); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (gs *genderStore) listGenders() ([]types.GenderOption, error) {
	rows, err := gs.db.Query(`
		SELECT id, label, faction_id FROM gender_options ORDER BY sort_index ASC, id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make([]types.GenderOption, 0)
	for rows.Next() {
		var g types.GenderOption
		if err := rows.Scan(&g.ID, &g.Label, &g.FactionID); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (gs *genderStore) replaceAll(factions []types.GenderFaction, genders []types.GenderOption) error {
	tx, err := gs.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := replaceGendersTx(tx, factions, genders); err != nil {
		return err
	}
	return tx.Commit()
}

func replaceGendersTx(tx *sql.Tx, factions []types.GenderFaction, genders []types.GenderOption) error {
	if _, err := tx.Exec(`DELETE FROM gender_options`); err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM gender_factions`); err != nil {
		return err
	}
	now := nowMs()
	fstmt, err := tx.Prepare(`
		INSERT INTO gender_factions (
			id, label, text_color, background_color, border_color, task_group, sort_index, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer fstmt.Close()
	for i, f := range factions {
		if _, err := fstmt.Exec(f.ID, f.Label, f.TextColor, f.BackgroundColor, f.BorderColor, f.TaskGroup, i, now, now); err != nil {
			return fmt.Errorf("insert faction %s: %w", f.ID, err)
		}
	}
	gstmt, err := tx.Prepare(`
		INSERT INTO gender_options (id, label, faction_id, sort_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer gstmt.Close()
	for i, g := range genders {
		if _, err := gstmt.Exec(g.ID, g.Label, g.FactionID, i, now, now); err != nil {
			return fmt.Errorf("insert gender %s: %w", g.ID, err)
		}
	}
	return nil
}
