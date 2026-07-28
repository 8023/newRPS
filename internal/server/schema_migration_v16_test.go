package server

import (
	"database/sql"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV16DropsPetBondRequestApprovalsTable 模拟停在 v15 的旧库，
// pet_bond_request_approvals 表还在且有历史数据。认主/认宠/解除关系申请统一为
// "双方须持续在线，任一方提前下线即自动撤销"后，逐条同意记录只需要在内存里追踪，
// 重启清空重新来过完全可以接受，不必落库——验证升级后该表被彻底删除。
func TestSchemaMigrationV16DropsPetBondRequestApprovalsTable(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "database.db")

	seed, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE pet_bond_requests (
			id TEXT PRIMARY KEY,
			kind TEXT NOT NULL,
			from_id TEXT NOT NULL,
			to_id TEXT NOT NULL,
			master_id TEXT NOT NULL DEFAULT '',
			pet_id TEXT NOT NULL DEFAULT '',
			status TEXT NOT NULL DEFAULT 'pending',
			created_at INTEGER NOT NULL DEFAULT 0
		);
		CREATE TABLE pet_bond_request_approvals (
			request_id TEXT NOT NULL,
			player_id TEXT NOT NULL,
			approved_at INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (request_id, player_id)
		);
		INSERT INTO pet_bond_requests (id, kind, from_id, to_id, master_id, pet_id, status) VALUES
			('req1', 'release', 'a', 'b', 'a', 'b', 'pending');
		INSERT INTO pet_bond_request_approvals (request_id, player_id, approved_at) VALUES
			('req1', 'a', 1000);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version (version) VALUES (15);
	`); err != nil {
		t.Fatalf("seed v15 schema: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase should run v16 migration cleanly: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	if ok, err := tableExists(db, "pet_bond_request_approvals"); err != nil {
		t.Fatalf("tableExists: %v", err)
	} else if ok {
		t.Fatal("pet_bond_request_approvals table should be dropped after v16 migration")
	}
	if ok, err := tableExists(db, "pet_bond_requests"); err != nil {
		t.Fatalf("tableExists: %v", err)
	} else if !ok {
		t.Fatal("pet_bond_requests table should still exist after v16 migration")
	}
}
