package server

import "testing"

// 版本化迁移必须能处理"建索引探测"处理不了的场景：改列名不会让任何 CREATE INDEX
// 缺列报错，旧列会悄悄一直留在表里——只有显式迁移才能覆盖。这里模拟"下一个版本"
// 把 push_subscriptions.endpoint 改名为 push_endpoint，验证：
//  1. 迁移真的执行了，旧数据在新列名下还能查到；
//  2. schema_version 被正确回写；
//  3. 下次启动不会重复执行同一条迁移（RENAME COLUMN 对已经改过名的表会报错，
//     重复执行就会直接暴露出来）。
func TestSchemaMigrationsHandleColumnRenameAndPersistVersion(t *testing.T) {
	dir := t.TempDir()

	db1, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	if _, err := db1.Exec(`INSERT INTO push_subscriptions (player_id, endpoint, p256dh, auth, created_at) VALUES ('p1','e1','k1','a1',1)`); err != nil {
		t.Fatalf("seed row: %v", err)
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	origMigrations, origVersion := migrations, currentSchemaVersion
	t.Cleanup(func() { migrations, currentSchemaVersion = origMigrations, origVersion })
	currentSchemaVersion = origVersion + 1
	ran := false
	migrations = append(append([]schemaMigration{}, origMigrations...), schemaMigration{
		version: currentSchemaVersion,
		migrate: func(db sqlExecer) error {
			ran = true
			_, err := db.Exec(`ALTER TABLE push_subscriptions RENAME COLUMN endpoint TO push_endpoint`)
			return err
		},
	})

	db2, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase after adding a rename migration: %v", err)
	}
	defer db2.Close()
	if !ran {
		t.Fatalf("expected the new migration to run")
	}

	var endpoint string
	if err := db2.QueryRow(`SELECT push_endpoint FROM push_subscriptions WHERE player_id = 'p1'`).Scan(&endpoint); err != nil {
		t.Fatalf("renamed column should be queryable, old row preserved: %v", err)
	}
	if endpoint != "e1" {
		t.Fatalf("push_endpoint = %q, want e1", endpoint)
	}

	v, err := readSchemaVersion(db2)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	ran = false
	db3, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("reopening an already-migrated database should not re-run the migration: %v", err)
	}
	defer db3.Close()
	if ran {
		t.Fatalf("migration re-ran on an already-migrated database")
	}
}
