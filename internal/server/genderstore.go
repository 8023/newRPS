package server

import (
	"database/sql"
	"fmt"

	"github.com/doumiao/newRPS/internal/types"
)

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
			id, label, normalized_label, text_color, background_color, border_color, task_group, sort_index, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer fstmt.Close()
	for i, f := range factions {
		if _, err := fstmt.Exec(f.ID, f.Label, normalizeGenderLabel(f.Label), f.TextColor, f.BackgroundColor, f.BorderColor, f.TaskGroup, i, now, now); err != nil {
			return fmt.Errorf("insert faction %s: %w", f.ID, err)
		}
	}
	gstmt, err := tx.Prepare(`
		INSERT INTO gender_options (id, label, normalized_label, faction_id, sort_index, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return err
	}
	defer gstmt.Close()
	for i, g := range genders {
		if _, err := gstmt.Exec(g.ID, g.Label, normalizeGenderLabel(g.Label), g.FactionID, i, now, now); err != nil {
			return fmt.Errorf("insert gender %s: %w", g.ID, err)
		}
	}
	return nil
}

// formalGenderTakenTx：某阵营下是否已存在同名（规范化后）性别选项，exceptID 排除自身
// （编辑已发布性别、走投稿修改版本重新发布时用）。与 upsertGenderTx/deleteGenderTx 一样
// 供共建投稿发布/下架流程（contribution_publish.go/contribution_unpublish.go）在调用方
// 已开的事务里复用，不单独开事务。
func formalGenderTakenTx(tx *sql.Tx, factionID, normalized, exceptID string) (bool, error) {
	var n int
	err := tx.QueryRow(
		`SELECT COUNT(*) FROM gender_options WHERE faction_id = ? AND normalized_label = ? AND id != ?`,
		factionID, normalized, exceptID,
	).Scan(&n)
	return n > 0, err
}

// upsertGenderTx：id 为空则插入新性别选项（sort_index 追加到末尾）并返回新分配的 ID，
// 否则原地更新 label/faction_id 并返回原 id。
func upsertGenderTx(tx *sql.Tx, id, label, normalized, factionID string) (string, error) {
	now := nowMs()
	if id == "" {
		id = "cg_" + randomID()
		if _, err := tx.Exec(`
			INSERT INTO gender_options (id, label, normalized_label, faction_id, sort_index, created_at, updated_at)
			VALUES (?, ?, ?, ?, (SELECT COALESCE(MAX(sort_index),0)+1 FROM gender_options), ?, ?)`,
			id, label, normalized, factionID, now, now); err != nil {
			return "", err
		}
		return id, nil
	}
	res, err := tx.Exec(`
		UPDATE gender_options SET label = ?, normalized_label = ?, faction_id = ?, updated_at = ? WHERE id = ?`,
		label, normalized, factionID, now, id)
	if err != nil {
		return "", err
	}
	if n, err := res.RowsAffected(); err != nil {
		return "", err
	} else if n == 0 {
		// 已下架投稿再次送审时 published_target_id 会保留，但正式 gender_options 行已经
		// 被删除；此时必须用原 ID 重新插入，不能“UPDATE 0 行后仍返回审批成功”。
		if _, err := tx.Exec(`
			INSERT INTO gender_options (id, label, normalized_label, faction_id, sort_index, created_at, updated_at)
			VALUES (?, ?, ?, ?, (SELECT COALESCE(MAX(sort_index),0)+1 FROM gender_options), ?, ?)`,
			id, label, normalized, factionID, now, now); err != nil {
			return "", err
		}
	}
	return id, nil
}

// deleteGenderTx：下架/撤回一条已发布的性别投稿时删除对应 gender_options 行；
// id 为空是合法调用（从未真正发布过），直接视为无操作。
func deleteGenderTx(tx *sql.Tx, id string) error {
	if id == "" {
		return nil
	}
	_, err := tx.Exec(`DELETE FROM gender_options WHERE id = ?`, id)
	return err
}
