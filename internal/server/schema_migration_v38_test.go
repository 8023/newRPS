package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// v37PlayersSchemaFixture 是 v38 迁移引入 extreme_force_closed_*/giveaway_board_* 十列
// 之前、players 表的真实结构快照。刻意写成硬编码字面量而不是从当前 playerSchema 常量
// 派生——playerSchema 后续版本（如 v47）会把列整体搬到子表甚至删除，若这里继续"从当前
// playerSchema 减去几列"反推历史结构，任何一次那样的重构都会连带改坏这个明明只想测
// v38 这一步迁移的历史夹具。这里的字段顺序/类型是 v37 时代的真实产物快照，之后不应
// 再随 playerSchema 变化而同步修改。
const v37PlayersSchemaFixture = `
CREATE TABLE players (
	id TEXT PRIMARY KEY,
	player_id TEXT NOT NULL UNIQUE,
	claim_key TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL DEFAULT '',
	gender_id TEXT NOT NULL DEFAULT '',
	faction_id TEXT NOT NULL DEFAULT '',
	avatar_url TEXT NOT NULL DEFAULT '',
	name_war_enabled INTEGER,
	name_war_allow_rename INTEGER,
	name_war_toggled_at INTEGER,
	name_war_original_name TEXT NOT NULL DEFAULT '',
	name_war_penalty_name TEXT NOT NULL DEFAULT '',
	name_war_punished INTEGER,
	name_war_rename_protected_until INTEGER,
	name_war_renamed_by TEXT NOT NULL DEFAULT '',
	name_war_renamed_by_name TEXT NOT NULL DEFAULT '',
	name_war_rename_window_started_at INTEGER,
	name_war_rename_count INTEGER,
	giveaway_enabled INTEGER,
	giveaway_value REAL,
	giveaway_clicks INTEGER,
	rank_multiplier_unlocked INTEGER,
	extreme_mode_enabled INTEGER,
	extreme_mode_toggled_at INTEGER,
	extreme_mode_cooldown_until INTEGER,
	extreme_win_streak INTEGER,
	extreme_last_decay_hour INTEGER,
	ranked_last_decay_day INTEGER,
	push_mention_enabled INTEGER,
	push_turn_enabled INTEGER,
	push_seat_enabled INTEGER,
	push_bond_enabled INTEGER,
	wins INTEGER NOT NULL DEFAULT 0,
	losses INTEGER NOT NULL DEFAULT 0,
	draws INTEGER NOT NULL DEFAULT 0,
	punishments INTEGER NOT NULL DEFAULT 0,
	ranked_points INTEGER NOT NULL DEFAULT 0,
	title TEXT NOT NULL DEFAULT '',
	title_segment_id TEXT NOT NULL DEFAULT '',
	title_custom INTEGER NOT NULL DEFAULT 0,
	self_title TEXT NOT NULL DEFAULT '',
	highest_score INTEGER NOT NULL DEFAULT 0,
	lowest_score INTEGER NOT NULL DEFAULT 0,
	rps_wins INTEGER NOT NULL DEFAULT 0,
	rps_losses INTEGER NOT NULL DEFAULT 0,
	rps_draws INTEGER NOT NULL DEFAULT 0,
	othello_wins INTEGER NOT NULL DEFAULT 0,
	othello_losses INTEGER NOT NULL DEFAULT 0,
	othello_draws INTEGER NOT NULL DEFAULT 0,
	tictactoe_wins INTEGER NOT NULL DEFAULT 0,
	tictactoe_losses INTEGER NOT NULL DEFAULT 0,
	tictactoe_draws INTEGER NOT NULL DEFAULT 0,
	gomoku_wins INTEGER NOT NULL DEFAULT 0,
	gomoku_losses INTEGER NOT NULL DEFAULT 0,
	gomoku_draws INTEGER NOT NULL DEFAULT 0,
	liarsdice_wins INTEGER NOT NULL DEFAULT 0,
	liarsdice_losses INTEGER NOT NULL DEFAULT 0,
	liarsdice_draws INTEGER NOT NULL DEFAULT 0,
	jungle_wins INTEGER NOT NULL DEFAULT 0,
	jungle_losses INTEGER NOT NULL DEFAULT 0,
	jungle_draws INTEGER NOT NULL DEFAULT 0,
	chess_wins INTEGER NOT NULL DEFAULT 0,
	chess_losses INTEGER NOT NULL DEFAULT 0,
	chess_draws INTEGER NOT NULL DEFAULT 0,
	total_online_ms INTEGER NOT NULL DEFAULT 0,
	bond_master_enabled INTEGER,
	bond_pet_enabled INTEGER,
	bond_public_display INTEGER,
	created_at INTEGER NOT NULL DEFAULT 0,
	last_seen_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE player_secrets (
	player_id TEXT NOT NULL,
	secret TEXT NOT NULL,
	PRIMARY KEY (player_id, secret),
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
`

// TestSchemaMigrationV38AddsExtremeForceClosedAndGiveawayBoardColumns verifies that an
// existing production-style players table (pre-v38, missing the extreme_force_closed_*
// and giveaway_board_* columns) survives the full migration chain up to the current
// version without disturbing existing rows. These columns back two pieces of state that
// were previously kept only in memory and silently reset on every restart: the
// "通用改名处" 极限强关 rename markers and the 白给自救板 content/vote counts (see
// schema_migrations.go v38). v38 itself adds them as flat columns on players; v47 later
// moves them (along with name_war/giveaway/game_stats) into player_extreme_mode/
// player_giveaway — this test's final assertions read from those two sub-tables rather
// than from players directly, since that's where the data ends up after the full chain.
func TestSchemaMigrationV38AddsExtremeForceClosedAndGiveawayBoardColumns(t *testing.T) {
	dir := t.TempDir()
	db, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := db.Exec(v37PlayersSchemaFixture); err != nil {
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
		t.Fatalf("run v38..latest migration: %v", err)
	}
	defer db.Close()

	var name string
	var rankedPoints int
	if err := db.QueryRow(`SELECT name, ranked_points FROM players WHERE id = 'p1'`).Scan(&name, &rankedPoints); err != nil {
		t.Fatalf("query pre-existing row: %v", err)
	}
	if name != "Alice" || rankedPoints != 15 {
		t.Fatalf("pre-existing row corrupted by migration: name=%q rankedPoints=%d", name, rankedPoints)
	}

	var forceClosed, forceClosedAt, renameProtected sql.NullInt64
	var renamedBy, renamedByName string
	if err := db.QueryRow(`
		SELECT force_closed, force_closed_at, rename_protected_until, renamed_by, renamed_by_name
		FROM player_extreme_mode WHERE player_id = 'p1'`).Scan(
		&forceClosed, &forceClosedAt, &renameProtected, &renamedBy, &renamedByName,
	); err != nil {
		t.Fatalf("query migrated extreme mode row: %v", err)
	}
	if forceClosed.Valid || forceClosedAt.Valid || renameProtected.Valid || renamedBy != "" || renamedByName != "" {
		t.Fatalf("extreme force-closed fields should default empty for pre-existing row: %v %v %v %q %q",
			forceClosed, forceClosedAt, renameProtected, renamedBy, renamedByName)
	}

	var boardText string
	var boardSubmitted, boardExpires, boardLikes, boardDislikes sql.NullInt64
	if err := db.QueryRow(`
		SELECT board_text, board_submitted_at, board_expires_at, board_likes, board_dislikes
		FROM player_giveaway WHERE player_id = 'p1'`).Scan(
		&boardText, &boardSubmitted, &boardExpires, &boardLikes, &boardDislikes,
	); err != nil {
		t.Fatalf("query migrated giveaway row: %v", err)
	}
	if boardText != "" || boardSubmitted.Valid || boardExpires.Valid || boardLikes.Valid || boardDislikes.Valid {
		t.Fatalf("giveaway board fields should default empty for pre-existing row: text=%q submitted=%v expires=%v likes=%v dislikes=%v",
			boardText, boardSubmitted, boardExpires, boardLikes, boardDislikes)
	}

	// players 表上的旧列应已被 v47 删除，不再是查询目标。
	has, err := tableHasColumn(db, "players", "extreme_force_closed")
	if err != nil {
		t.Fatalf("tableHasColumn: %v", err)
	}
	if has {
		t.Fatal("players.extreme_force_closed should have been dropped by v47")
	}
}
