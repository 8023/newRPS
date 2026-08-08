package server

import (
	"database/sql"
	"sync"
)

// petBondStore 认主关系与待办申请的 SQLite 持久化。
// 运行时权威仍在 Server 内存 map；本 store 负责启动加载与写穿落盘。
type petBondStore struct {
	db *sql.DB
	mu sync.Mutex
}

const petBondSchema = `
CREATE TABLE IF NOT EXISTS pet_bonds (
	master_id TEXT NOT NULL,
	pet_id TEXT NOT NULL,
	pet_title TEXT NOT NULL DEFAULT '',
	title_updated_at INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (master_id, pet_id)
);
CREATE INDEX IF NOT EXISTS idx_pet_bonds_pet ON pet_bonds(pet_id);
CREATE INDEX IF NOT EXISTS idx_pet_bonds_created ON pet_bonds(created_at);
CREATE TABLE IF NOT EXISTS pet_bond_requests (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	from_id TEXT NOT NULL,
	to_id TEXT NOT NULL,
	master_id TEXT NOT NULL DEFAULT '',
	pet_id TEXT NOT NULL DEFAULT '',
	status TEXT NOT NULL DEFAULT 'pending',
	created_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_pet_bond_requests_status ON pet_bond_requests(status);
`

// petBondRow / petBondRequestRow 与库表一一对应。
type petBondRow struct {
	MasterID       string
	PetID          string
	PetTitle       string
	TitleUpdatedAt int64
	CreatedAt      int64
}

type petBondRequestRow struct {
	ID        string
	Kind      string
	FromID    string
	ToID      string
	MasterID  string
	PetID     string
	Status    string
	CreatedAt int64
}

func newPetBondStore(db *sql.DB) *petBondStore {
	return &petBondStore{db: db}
}

func (ps *petBondStore) loadAllBonds() ([]petBondRow, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	rows, err := ps.db.Query(`SELECT master_id, pet_id, pet_title, title_updated_at, created_at FROM pet_bonds`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []petBondRow
	for rows.Next() {
		var r petBondRow
		if err := rows.Scan(&r.MasterID, &r.PetID, &r.PetTitle, &r.TitleUpdatedAt, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (ps *petBondStore) loadPendingRequests() ([]petBondRequestRow, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	rows, err := ps.db.Query(`
		SELECT id, kind, from_id, to_id, master_id, pet_id, status, created_at
		FROM pet_bond_requests WHERE status = 'pending'`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var reqs []petBondRequestRow
	for rows.Next() {
		var r petBondRequestRow
		if err := rows.Scan(&r.ID, &r.Kind, &r.FromID, &r.ToID, &r.MasterID, &r.PetID, &r.Status, &r.CreatedAt); err != nil {
			return nil, err
		}
		reqs = append(reqs, r)
	}
	return reqs, rows.Err()
}

func (ps *petBondStore) upsertBond(r petBondRow) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	_, err := ps.db.Exec(`
		INSERT INTO pet_bonds (master_id, pet_id, pet_title, title_updated_at, created_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(master_id, pet_id) DO UPDATE SET
			pet_title=excluded.pet_title,
			title_updated_at=excluded.title_updated_at
	`, r.MasterID, r.PetID, r.PetTitle, r.TitleUpdatedAt, r.CreatedAt)
	return err
}

func (ps *petBondStore) deleteBond(masterID, petID string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	_, err := ps.db.Exec(`DELETE FROM pet_bonds WHERE master_id = ? AND pet_id = ?`, masterID, petID)
	return err
}

func (ps *petBondStore) insertRequest(r petBondRequestRow) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	_, err := ps.db.Exec(`
		INSERT INTO pet_bond_requests (id, kind, from_id, to_id, master_id, pet_id, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, r.ID, r.Kind, r.FromID, r.ToID, r.MasterID, r.PetID, r.Status, r.CreatedAt)
	return err
}

func (ps *petBondStore) setRequestStatus(id, status string) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	_, err := ps.db.Exec(`UPDATE pet_bond_requests SET status = ? WHERE id = ?`, status, id)
	return err
}
