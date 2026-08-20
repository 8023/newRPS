package server

import (
	"database/sql"
	"strings"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// TestSchemaMigrationV38AddsExtremeForceClosedAndGiveawayBoardColumns verifies that an
// existing production-style players table (pre-v38, missing the extreme_force_closed_*
// and giveaway_board_* columns) gains all ten new columns on upgrade without disturbing
// existing rows. These columns back two pieces of state that were previously kept only
// in memory and silently reset on every restart: the "通用改名处" 极限强关 rename markers
// and the 白给自救板 content/vote counts (see schema_migrations.go v38).
func TestSchemaMigrationV38AddsExtremeForceClosedAndGiveawayBoardColumns(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	v37PlayerSchema := playerSchema
	for _, col := range []string{
		"\textreme_force_closed INTEGER,\n",
		"\textreme_force_closed_at INTEGER,\n",
		"\textreme_rename_protected_until INTEGER,\n",
		"\textreme_renamed_by TEXT NOT NULL DEFAULT '',\n",
		"\textreme_renamed_by_name TEXT NOT NULL DEFAULT '',\n",
		"\tgiveaway_board_text TEXT NOT NULL DEFAULT '',\n",
		"\tgiveaway_board_submitted_at INTEGER,\n",
		"\tgiveaway_board_expires_at INTEGER,\n",
		"\tgiveaway_board_likes INTEGER,\n",
		"\tgiveaway_board_dislikes INTEGER,\n",
	} {
		next := strings.Replace(v37PlayerSchema, col, "", 1)
		if next == v37PlayerSchema {
			t.Fatalf("test fixture failed to remove column line %q from player schema", col)
		}
		v37PlayerSchema = next
	}
	if _, err := db.Exec(v37PlayerSchema); err != nil {
		t.Fatalf("create v37 player schema: %v", err)
	}
	if _, err := db.Exec(`
		INSERT INTO players (id, player_id, name, ranked_points)
		VALUES ('p1', 'identity-1', 'Alice', 15);
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version VALUES (37);
	`); err != nil {
		t.Fatalf("seed v37 db: %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("close seed db: %v", err)
	}

	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("run v38 migration: %v", err)
	}
	defer db.Close()

	var (
		name                                        string
		rankedPoints                                int
		forceClosed, forceClosedAt, renameProtected sql.NullInt64
		renamedBy, renamedByName                    string
		boardText                                   string
		boardSubmitted, boardExpires                sql.NullInt64
		boardLikes, boardDislikes                   sql.NullInt64
	)
	if err := db.QueryRow(`
		SELECT name, ranked_points,
			extreme_force_closed, extreme_force_closed_at, extreme_rename_protected_until,
			extreme_renamed_by, extreme_renamed_by_name,
			giveaway_board_text, giveaway_board_submitted_at, giveaway_board_expires_at,
			giveaway_board_likes, giveaway_board_dislikes
		FROM players WHERE id = 'p1'`).Scan(
		&name, &rankedPoints,
		&forceClosed, &forceClosedAt, &renameProtected,
		&renamedBy, &renamedByName,
		&boardText, &boardSubmitted, &boardExpires,
		&boardLikes, &boardDislikes,
	); err != nil {
		t.Fatalf("query migrated columns: %v", err)
	}
	if name != "Alice" || rankedPoints != 15 {
		t.Fatalf("pre-existing row corrupted by migration: name=%q rankedPoints=%d", name, rankedPoints)
	}
	if forceClosed.Valid || forceClosedAt.Valid || renameProtected.Valid || renamedBy != "" || renamedByName != "" {
		t.Fatalf("extreme force-closed columns should default empty for pre-existing row: %v %v %v %q %q",
			forceClosed, forceClosedAt, renameProtected, renamedBy, renamedByName)
	}
	if boardText != "" || boardSubmitted.Valid || boardExpires.Valid || boardLikes.Valid || boardDislikes.Valid {
		t.Fatalf("giveaway board columns should default empty for pre-existing row: text=%q submitted=%v expires=%v likes=%v dislikes=%v",
			boardText, boardSubmitted, boardExpires, boardLikes, boardDislikes)
	}
}
