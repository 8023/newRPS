package server

import (
	"database/sql"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// currentSchemaVersion 是代码期望的数据库结构版本号。每次改动某张表的列（加/删/改名/
// 拆表）都必须：
//  1. 把对应 store 文件里的 xxxSchema 常量改成目标结构；
//  2. 在下面的 migrations 里追加一条 {version: currentSchemaVersion+1, migrate: ...}，
//     用 ALTER TABLE / 建新表倒数据 / DROP+RENAME 之类的显式语句把旧数据搬过去；
//  3. 把这个常量加一。
//
// 是 var 不是 const，只是为了让测试能临时替换掉去验证迁移机制本身；正常代码路径里
// 把它当常量对待，不要在业务逻辑里修改它。
var currentSchemaVersion = 3

// schemaVersionSchema：只有一行的版本表，openDatabase 每次启动都会先确保它存在。
const schemaVersionSchema = `
CREATE TABLE IF NOT EXISTS schema_version (
	version INTEGER NOT NULL
);
`

// schemaMigration 是一步"从更早版本升到 version"的显式迁移。migrate 拿到的是当前事务
// （*sql.Tx，满足 sqlExecer），迁移语句和版本号回写在同一个事务里提交，保证原子——
// 不会出现"迁移做了一半、版本号却没更新，下次重启重复执行报错"的情况。
//
// 迁移必须幂等：version==0 的旧库（CREATE TABLE IF NOT EXISTS 不会给已有表补列）
// 与"已经是最新结构的全新库"都会跑这些步骤；对已存在的目标列/已改名的列要跳过，
// 不能假设"旧列一定还在"。
type schemaMigration struct {
	version int
	migrate func(sqlExecer) error
}

// sqlExecer 是 *sql.DB / *sql.Tx 的公共子集，migrate 函数只需要这几个方法就够用。
type sqlExecer interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
	QueryRow(query string, args ...any) *sql.Row
}

// migrations 按 version 升序排列。
var migrations = []schemaMigration{
	// v2：player_activity_events.name/old_name 改名为 new_value/old_value；
	// 移除聊天消息的 author_player 快照列。
	{version: 2, migrate: func(db sqlExecer) error {
		if err := renameColumnIfExists(db, "player_activity_events", "name", "new_value"); err != nil {
			return err
		}
		if err := renameColumnIfExists(db, "player_activity_events", "old_name", "old_value"); err != nil {
			return err
		}
		if err := dropColumnIfExists(db, "lobby_messages", "author_player"); err != nil {
			return err
		}
		if err := dropColumnIfExists(db, "room_messages", "author_player"); err != nil {
			return err
		}
		return nil
	}},
	// v3：排位分历史最高/最低 + 每日衰减去重标记；以当前 ranked_points 回填极值。
	{version: 3, migrate: func(db sqlExecer) error {
		if err := addColumnIfMissing(db, "players", "highest_score", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "lowest_score", "INTEGER NOT NULL DEFAULT 0"); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "players", "ranked_last_decay_day", "INTEGER"); err != nil {
			return err
		}
		// 回填可重复执行：只在极值仍是 0 且 ranked_points 非 0 时写，避免覆盖已有历史。
		if _, err := db.Exec(`UPDATE players SET highest_score = ranked_points WHERE highest_score = 0 AND ranked_points > 0`); err != nil {
			return fmt.Errorf("backfill highest_score: %w", err)
		}
		if _, err := db.Exec(`UPDATE players SET lowest_score = ranked_points WHERE lowest_score = 0 AND ranked_points < 0`); err != nil {
			return fmt.Errorf("backfill lowest_score: %w", err)
		}
		return nil
	}},
}

// ensureSchema 是 openDatabase 建表/升级结构的唯一入口：
//  1. 确保 schema_version 表存在，读出当前记录的版本号（表刚建出来、还没写过版本号
//     视为 0）。
//  2. version==0 时，先跑 quarantine：历史遗留库结构无法追溯时，把不兼容的旧表改名隔离。
//  3. 建出/补全所有表（CREATE TABLE/INDEX IF NOT EXISTS）。
//  4. 依次跑 version 之后的 migrations（幂等）。注意：version==0 的旧库也会跑迁移——
//     仅靠 CREATE IF NOT EXISTS 不会给已有 players 表补 highest_score 等新列；
//     跳过迁移会把 schema_version 误标成最新，导致永久缺列、玩家加载失败。
func ensureSchema(db *sql.DB) error {
	if _, err := db.Exec(schemaVersionSchema); err != nil {
		return err
	}
	version, err := readSchemaVersion(db)
	if err != nil {
		return err
	}
	freshBootstrap := version == 0

	for _, schema := range allSchemas {
		if freshBootstrap {
			if err := execSchemaQuarantiningLegacyTables(db, schema); err != nil {
				return err
			}
			continue
		}
		if _, err := db.Exec(schema); err != nil {
			return fmt.Errorf("%s: %w", schema, err)
		}
	}

	for _, m := range migrations {
		if m.version <= version {
			continue
		}
		tx, err := db.Begin()
		if err != nil {
			return err
		}
		if err := m.migrate(tx); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("migration v%d: %w", m.version, err)
		}
		if err := writeSchemaVersion(tx, m.version); err != nil {
			_ = tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		version = m.version
	}

	if version < currentSchemaVersion {
		return writeSchemaVersion(db, currentSchemaVersion)
	}
	return nil
}

func readSchemaVersion(db sqlExecer) (int, error) {
	var version int
	err := db.QueryRow(`SELECT version FROM schema_version LIMIT 1`).Scan(&version)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	return version, err
}

func writeSchemaVersion(db sqlExecer, version int) error {
	res, err := db.Exec(`UPDATE schema_version SET version = ?`, version)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		return nil
	}
	_, err = db.Exec(`INSERT INTO schema_version (version) VALUES (?)`, version)
	return err
}

// tableHasColumn 查 sqlite_master / PRAGMA table_info，判断列是否存在。
func tableHasColumn(db sqlExecer, table, column string) (bool, error) {
	// PRAGMA 不支持占位符绑表名；table/column 仅来自代码常量，不接用户输入。
	rows, err := db.Query(fmt.Sprintf(`PRAGMA table_info(%s)`, table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, ctype string
		var notnull, pk int
		var dflt sql.NullString
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return false, err
		}
		if strings.EqualFold(name, column) {
			return true, nil
		}
	}
	return false, rows.Err()
}

func tableExists(db sqlExecer, table string) (bool, error) {
	var name string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&name)
	if err == sql.ErrNoRows {
		return false, nil
	}
	return err == nil, err
}

func renameColumnIfExists(db sqlExecer, table, from, to string) error {
	ok, err := tableExists(db, table)
	if err != nil || !ok {
		return err
	}
	hasFrom, err := tableHasColumn(db, table, from)
	if err != nil {
		return err
	}
	if !hasFrom {
		return nil
	}
	hasTo, err := tableHasColumn(db, table, to)
	if err != nil {
		return err
	}
	if hasTo {
		// 目标列已在：旧列残留时尽量丢掉旧列，避免双列并存。
		return dropColumnIfExists(db, table, from)
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s RENAME COLUMN %s TO %s`, table, from, to))
	if err != nil {
		return fmt.Errorf("rename %s.%s -> %s: %w", table, from, to, err)
	}
	return nil
}

func dropColumnIfExists(db sqlExecer, table, column string) error {
	ok, err := tableExists(db, table)
	if err != nil || !ok {
		return err
	}
	has, err := tableHasColumn(db, table, column)
	if err != nil || !has {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s DROP COLUMN %s`, table, column))
	if err != nil {
		return fmt.Errorf("drop %s.%s: %w", table, column, err)
	}
	return nil
}

func addColumnIfMissing(db sqlExecer, table, column, decl string) error {
	ok, err := tableExists(db, table)
	if err != nil {
		return err
	}
	if !ok {
		// 表还不存在时由 CREATE TABLE IF NOT EXISTS 建出完整结构，这里无需 ADD。
		return nil
	}
	has, err := tableHasColumn(db, table, column)
	if err != nil || has {
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %s ADD COLUMN %s %s`, table, column, decl))
	if err != nil {
		return fmt.Errorf("add %s.%s: %w", table, column, err)
	}
	return nil
}

// createIndexTableRE 从 "CREATE INDEX IF NOT EXISTS xxx ON <table>(...)" 里取出表名，
// 供下面 execSchemaQuarantiningLegacyTables 判断建索引失败时该隔离哪张表。
var createIndexTableRE = regexp.MustCompile(`(?is)CREATE\s+INDEX\s+IF\s+NOT\s+EXISTS\s+\S+\s+ON\s+([a-zA-Z_][a-zA-Z0-9_]*)\s*\(`)

// execSchemaQuarantiningLegacyTables 只在 ensureSchema 判定 version==0（未纳入版本管理
// 的历史库）时使用：逐条执行一段 schema 里的语句；某条 CREATE INDEX 因为列不存在报错，
// 说明该索引所在的表是旧结构（CREATE TABLE IF NOT EXISTS 对已存在的同名旧表是空操作，
// 不会补列），就把整张旧表改名隔离，再回到这段 schema 的开头重新执行。
func execSchemaQuarantiningLegacyTables(db *sql.DB, schema string) error {
	stmts := splitSQLStatements(schema)
	quarantined := map[string]bool{}
	for i := 0; i < len(stmts); i++ {
		stmt := stmts[i]
		if _, err := db.Exec(stmt); err != nil {
			table := createIndexTableRE.FindStringSubmatch(stmt)
			if table == nil || !isMissingColumnErr(err) || quarantined[table[1]] {
				return fmt.Errorf("%s: %w", stmt, err)
			}
			if err := renameLegacyTable(db, table[1]); err != nil {
				return fmt.Errorf("quarantine legacy %s: %w", table[1], err)
			}
			quarantined[table[1]] = true
			i = -1 // 回到这段 schema 的开头，让对应的 CREATE TABLE 重新跑一遍
			continue
		}
	}
	return nil
}

func splitSQLStatements(schema string) []string {
	parts := strings.Split(schema, ";")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p := strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func isMissingColumnErr(err error) bool {
	return strings.Contains(err.Error(), "no such column")
}

// renameLegacyTable 把 table 改名让路给即将重新创建的新结构；名字冲突时追加时间戳。
func renameLegacyTable(db *sql.DB, table string) error {
	legacyName := table + "_legacy"
	var exists string
	err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type='table' AND name=?`, legacyName).Scan(&exists)
	switch {
	case err == nil:
		legacyName = fmt.Sprintf("%s_legacy_%d", table, time.Now().Unix())
	case err != sql.ErrNoRows:
		return err
	}
	_, err = db.Exec(fmt.Sprintf(`ALTER TABLE %q RENAME TO %q`, table, legacyName))
	return err
}
