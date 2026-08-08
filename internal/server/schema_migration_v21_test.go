package server

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV21AddsBondPushPreference verifies that an existing production-style
// players table gains the fourth push preference without changing the three existing values.
func TestSchemaMigrationV21AddsBondPushPreference(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	v20PlayerSchema := strings.Replace(playerSchema, "\tpush_bond_enabled INTEGER,\n", "", 1)
	if v20PlayerSchema == playerSchema {
		t.Fatal("test fixture failed to remove push_bond_enabled from player schema")
	}
	if _, err := db.Exec(v20PlayerSchema); err != nil {
		t.Fatalf("create v20 player schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO players (id, player_id, push_mention_enabled, push_turn_enabled, push_seat_enabled)
		VALUES ('p1', 'identity-1', 1, 0, NULL);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version VALUES (20);
	`); err != nil {
		t.Fatalf("seed v20 db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("run v21 migration: %v", err)
	}
	defer db.Close()

	var mention, turn, seat, bond sql.NullInt64
	if err := db.QueryRow(`SELECT push_mention_enabled, push_turn_enabled, push_seat_enabled, push_bond_enabled FROM players WHERE id = 'p1'`).
		Scan(&mention, &turn, &seat, &bond); err != nil {
		t.Fatalf("query migrated preferences: %v", err)
	}
	if !mention.Valid || mention.Int64 != 1 || !turn.Valid || turn.Int64 != 0 || seat.Valid || bond.Valid {
		t.Fatalf("migrated preferences: mention=%v turn=%v seat=%v bond=%v", mention, turn, seat, bond)
	}
}
