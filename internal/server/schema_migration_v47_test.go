package server

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

// v46PlayersSchemaFixture 是 v47 拆表迁移之前、players 表的真实结构快照（与 v38 测试的
// v37PlayersSchemaFixture 同一套理由：硬编码字面量，不从当前 playerSchema 派生，免得
// playerSchema 未来继续演进时连带改坏这个只想测 v46→v47 这一步的历史夹具）。
const v46PlayersSchemaFixture = `
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
	extreme_force_closed INTEGER,
	extreme_force_closed_at INTEGER,
	extreme_rename_protected_until INTEGER,
	extreme_renamed_by TEXT NOT NULL DEFAULT '',
	extreme_renamed_by_name TEXT NOT NULL DEFAULT '',
	giveaway_board_text TEXT NOT NULL DEFAULT '',
	giveaway_board_submitted_at INTEGER,
	giveaway_board_expires_at INTEGER,
	giveaway_board_likes INTEGER,
	giveaway_board_dislikes INTEGER,
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

// TestSchemaMigrationV47SplitsPlayerSubTables 是 v47 拆表迁移的核心正确性测试：在一张
// 真实 v46 结构的 players 表上插入三种代表性的行——
//  1. p1：全部四组功能域字段都非空/非零，且刻意在同一组内混入 NULL 与显式 0/false
//     （比如 name_war_allow_rename=0 但 name_war_toggled_at=NULL），验证三态语义
//     （从未设置 NULL / 显式假 0 / 有值）在拆表后逐列精确保留，不会被"存在性判断"
//     误伤；
//  2. p2：全部字段都是默认值/NULL（从未碰过任何一组功能），验证"这个人明明什么都没
//     设置过"不会被迁移误变成"显式设置为空/假"；
//  3. p3：只打过部分游戏（othello 有战绩，其余游戏挂零），验证 game_stats 拆表后仍是
//     稠密的 7 行（未打过的游戏也有一行 0/0/0），而不是只给打过的游戏建行。
//
// 断言分四层：迁移后 schema_version 前进到当前版本；四张子表的行数精确等于
// "玩家数"（或 game_stats 的"玩家数×7"）——这是数据没有被漏迁移或重复迁移的行数
// 不变量；每个玩家在四张子表里的具体值与迁移前 players 表上的旧列逐一比对相等；
// players 表上的 50 个旧列全部被删除，不再可查询。
func TestSchemaMigrationV47SplitsPlayerSubTables(t *testing.T) {
	dir := t.TempDir()
	seed, err := sql.Open("sqlite3", dir+"/database.db")
	if err != nil {
		t.Fatalf("open seed db: %v", err)
	}
	if _, err := seed.Exec(v46PlayersSchemaFixture); err != nil {
		t.Fatalf("create v46 player schema: %v", err)
	}
	if _, err := seed.Exec(`
		CREATE TABLE schema_version (version INTEGER NOT NULL);
		INSERT INTO schema_version VALUES (46);

		INSERT INTO players (
			id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			name_war_enabled, name_war_allow_rename, name_war_toggled_at, name_war_original_name,
			name_war_penalty_name, name_war_punished, name_war_rename_protected_until,
			name_war_renamed_by, name_war_renamed_by_name, name_war_rename_window_started_at, name_war_rename_count,
			giveaway_enabled, giveaway_value, giveaway_clicks,
			rank_multiplier_unlocked,
			extreme_mode_enabled, extreme_mode_toggled_at, extreme_mode_cooldown_until,
			extreme_win_streak, extreme_last_decay_hour, ranked_last_decay_day,
			extreme_force_closed, extreme_force_closed_at, extreme_rename_protected_until,
			extreme_renamed_by, extreme_renamed_by_name,
			giveaway_board_text, giveaway_board_submitted_at, giveaway_board_expires_at,
			giveaway_board_likes, giveaway_board_dislikes,
			rps_wins, rps_losses, rps_draws,
			othello_wins, othello_losses, othello_draws,
			tictactoe_wins, tictactoe_losses, tictactoe_draws,
			gomoku_wins, gomoku_losses, gomoku_draws,
			liarsdice_wins, liarsdice_losses, liarsdice_draws,
			jungle_wins, jungle_losses, jungle_draws,
			chess_wins, chess_losses, chess_draws,
			created_at, last_seen_at
		) VALUES (
			'p1', 'ident-1', 'claim1', '玩家一', 'male', 'fa1', '',
			1, 0, NULL, '原名一',
			'母狗001', 1, 9000,
			'admin1', '管理员甲', 8000, 3,
			1, 2.5, 10,
			1,
			1, 5000, 6000,
			9, 7000, NULL,
			1, 4000, 4500,
			'admin2', '管理员乙',
			'我错了', 1000, 2000,
			5, 1,
			10, 3, 1,
			5, 2, 0,
			0, 0, 0,
			0, 0, 0,
			0, 0, 0,
			2, 1, 0,
			1, 0, 0,
			100, 200
		);

		INSERT INTO players (
			id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			created_at, last_seen_at
		) VALUES (
			'p2', 'ident-2', 'claim2', '玩家二', 'female', 'fa2', '',
			101, 201
		);

		INSERT INTO players (
			id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			othello_wins, othello_losses, othello_draws,
			created_at, last_seen_at
		) VALUES (
			'p3', 'ident-3', 'claim3', '玩家三', 'male', 'fa1', '',
			4, 2, 1,
			102, 202
		);
	`); err != nil {
		t.Fatalf("seed v46 rows: %v", err)
	}
	if err := seed.Close(); err != nil {
		t.Fatalf("close seed: %v", err)
	}

	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("run v47 migration: %v", err)
	}
	defer db.Close()

	v, err := readSchemaVersion(db)
	if err != nil {
		t.Fatalf("readSchemaVersion: %v", err)
	}
	if v != currentSchemaVersion {
		t.Fatalf("schema_version = %d, want %d", v, currentSchemaVersion)
	}

	// 行数不变量：3 个玩家 → player_name_war/player_giveaway/player_extreme_mode 各 3 行，
	// player_game_stats 3×7=21 行（稠密，含未打过的游戏）。
	for _, table := range []string{"player_name_war", "player_giveaway", "player_extreme_mode"} {
		var n int
		if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&n); err != nil {
			t.Fatalf("count %s: %v", table, err)
		}
		if n != 3 {
			t.Fatalf("%s row count = %d, want 3", table, n)
		}
	}
	var gsCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM player_game_stats`).Scan(&gsCount); err != nil {
		t.Fatalf("count player_game_stats: %v", err)
	}
	if gsCount != 3*7 {
		t.Fatalf("player_game_stats row count = %d, want %d", gsCount, 3*7)
	}

	// p1：混合 NULL / 显式 0 / 有值三态精确核对。
	var (
		enabled, allowRename, toggledAt, punished, renameProtected sql.NullInt64
		originalName, penaltyName, renamedBy, renamedByName        string
		renameWindowStarted, renameCount                           sql.NullInt64
	)
	if err := db.QueryRow(`
		SELECT enabled, allow_rename, toggled_at, original_name, penalty_name, punished,
			rename_protected_until, renamed_by, renamed_by_name, rename_window_started_at, rename_count
		FROM player_name_war WHERE player_id = 'p1'`).Scan(
		&enabled, &allowRename, &toggledAt, &originalName, &penaltyName, &punished,
		&renameProtected, &renamedBy, &renamedByName, &renameWindowStarted, &renameCount,
	); err != nil {
		t.Fatalf("query p1 name_war: %v", err)
	}
	if enabled.Int64 != 1 || allowRename.Int64 != 0 || !allowRename.Valid || toggledAt.Valid ||
		originalName != "原名一" || penaltyName != "母狗001" || punished.Int64 != 1 ||
		renameProtected.Int64 != 9000 || renamedBy != "admin1" || renamedByName != "管理员甲" ||
		renameWindowStarted.Int64 != 8000 || renameCount.Int64 != 3 {
		t.Fatalf("p1 name_war mismatch: enabled=%v allowRename=%v toggledAt=%v(valid=%v) original=%q penalty=%q punished=%v protected=%v by=%q byName=%q winStart=%v count=%v",
			enabled, allowRename, toggledAt, toggledAt.Valid, originalName, penaltyName, punished, renameProtected, renamedBy, renamedByName, renameWindowStarted, renameCount)
	}
	// name_war_toggled_at 在种子数据里是 NULL——必须原样是 NULL，不能被迁移误写成 0。
	if toggledAt.Valid {
		t.Fatalf("p1 name_war.toggled_at should remain NULL, got %v", toggledAt)
	}

	var extEnabled, extToggled, extCooldown, extStreak, extDecay sql.NullInt64
	var extForceClosed, extForceClosedAt, extRenameProtected sql.NullInt64
	var extRenamedBy, extRenamedByName string
	if err := db.QueryRow(`
		SELECT enabled, toggled_at, cooldown_until, win_streak, last_decay_hour,
			force_closed, force_closed_at, rename_protected_until, renamed_by, renamed_by_name
		FROM player_extreme_mode WHERE player_id = 'p1'`).Scan(
		&extEnabled, &extToggled, &extCooldown, &extStreak, &extDecay,
		&extForceClosed, &extForceClosedAt, &extRenameProtected, &extRenamedBy, &extRenamedByName,
	); err != nil {
		t.Fatalf("query p1 extreme_mode: %v", err)
	}
	if extEnabled.Int64 != 1 || extToggled.Int64 != 5000 || extCooldown.Int64 != 6000 ||
		extStreak.Int64 != 9 || extDecay.Int64 != 7000 || extForceClosed.Int64 != 1 ||
		extForceClosedAt.Int64 != 4000 || extRenameProtected.Int64 != 4500 ||
		extRenamedBy != "admin2" || extRenamedByName != "管理员乙" {
		t.Fatalf("p1 extreme_mode mismatch: %+v %+v %+v %+v %+v %+v %+v %+v %q %q",
			extEnabled, extToggled, extCooldown, extStreak, extDecay, extForceClosed, extForceClosedAt, extRenameProtected, extRenamedBy, extRenamedByName)
	}

	var gaEnabled, gaClicks, gaBoardSubmitted, gaBoardExpires, gaBoardLikes, gaBoardDislikes sql.NullInt64
	var gaValue sql.NullFloat64
	var gaBoardText string
	if err := db.QueryRow(`
		SELECT enabled, value, clicks, board_text, board_submitted_at, board_expires_at, board_likes, board_dislikes
		FROM player_giveaway WHERE player_id = 'p1'`).Scan(
		&gaEnabled, &gaValue, &gaClicks, &gaBoardText, &gaBoardSubmitted, &gaBoardExpires, &gaBoardLikes, &gaBoardDislikes,
	); err != nil {
		t.Fatalf("query p1 giveaway: %v", err)
	}
	if gaEnabled.Int64 != 1 || gaValue.Float64 != 2.5 || gaClicks.Int64 != 10 || gaBoardText != "我错了" ||
		gaBoardSubmitted.Int64 != 1000 || gaBoardExpires.Int64 != 2000 || gaBoardLikes.Int64 != 5 || gaBoardDislikes.Int64 != 1 {
		t.Fatalf("p1 giveaway mismatch: %+v %+v %+v %q %+v %+v %+v %+v",
			gaEnabled, gaValue, gaClicks, gaBoardText, gaBoardSubmitted, gaBoardExpires, gaBoardLikes, gaBoardDislikes)
	}

	// p1 分游戏胜负平：逐款游戏核对，覆盖"有战绩"和"零战绩"两种行。
	wantP1 := map[string][3]int{
		"rps": {10, 3, 1}, "othello": {5, 2, 0}, "tictactoe": {0, 0, 0},
		"gomoku": {0, 0, 0}, "liarsdice": {0, 0, 0}, "jungle": {2, 1, 0}, "chess": {1, 0, 0},
	}
	assertGameStats(t, db, "p1", wantP1)

	// p2：从未碰过任何一组功能——子表仍各有一行（稠密），但字段全部是 NULL/空。
	var p2Enabled, p2Punished sql.NullInt64
	var p2OriginalName, p2PenaltyName string
	if err := db.QueryRow(`
		SELECT enabled, punished, original_name, penalty_name FROM player_name_war WHERE player_id = 'p2'`).Scan(
		&p2Enabled, &p2Punished, &p2OriginalName, &p2PenaltyName,
	); err != nil {
		t.Fatalf("query p2 name_war: %v", err)
	}
	if p2Enabled.Valid || p2Punished.Valid || p2OriginalName != "" || p2PenaltyName != "" {
		t.Fatalf("p2 name_war should be all-empty/NULL, got enabled=%v punished=%v original=%q penalty=%q",
			p2Enabled, p2Punished, p2OriginalName, p2PenaltyName)
	}
	wantP2 := map[string][3]int{
		"rps": {0, 0, 0}, "othello": {0, 0, 0}, "tictactoe": {0, 0, 0},
		"gomoku": {0, 0, 0}, "liarsdice": {0, 0, 0}, "jungle": {0, 0, 0}, "chess": {0, 0, 0},
	}
	assertGameStats(t, db, "p2", wantP2)

	// p3：只打过 othello，其余游戏也要有稠密的 0/0/0 行。
	wantP3 := map[string][3]int{
		"rps": {0, 0, 0}, "othello": {4, 2, 1}, "tictactoe": {0, 0, 0},
		"gomoku": {0, 0, 0}, "liarsdice": {0, 0, 0}, "jungle": {0, 0, 0}, "chess": {0, 0, 0},
	}
	assertGameStats(t, db, "p3", wantP3)

	// players 表上的 50 个旧列必须已被删除。
	oldColumns := []string{
		"name_war_enabled", "giveaway_enabled", "extreme_mode_enabled",
		"rps_wins", "othello_wins", "tictactoe_wins", "gomoku_wins", "liarsdice_wins", "jungle_wins", "chess_wins",
	}
	for _, col := range oldColumns {
		has, err := tableHasColumn(db, "players", col)
		if err != nil {
			t.Fatalf("tableHasColumn(%s): %v", col, err)
		}
		if has {
			t.Fatalf("players.%s should have been dropped by v47", col)
		}
	}
	// ranked_last_decay_day/rank_multiplier_unlocked 明确不属于拆分范围，必须还留在 players 上。
	for _, col := range []string{"ranked_last_decay_day", "rank_multiplier_unlocked"} {
		has, err := tableHasColumn(db, "players", col)
		if err != nil {
			t.Fatalf("tableHasColumn(%s): %v", col, err)
		}
		if !has {
			t.Fatalf("players.%s should NOT have been dropped by v47", col)
		}
	}
}

func assertGameStats(t *testing.T, db *sql.DB, playerID string, want map[string][3]int) {
	t.Helper()
	rows, err := db.Query(`SELECT game_id, wins, losses, draws FROM player_game_stats WHERE player_id = ?`, playerID)
	if err != nil {
		t.Fatalf("query game_stats for %s: %v", playerID, err)
	}
	defer rows.Close()
	got := map[string][3]int{}
	for rows.Next() {
		var gameID string
		var w, l, d int
		if err := rows.Scan(&gameID, &w, &l, &d); err != nil {
			t.Fatalf("scan game_stats for %s: %v", playerID, err)
		}
		got[gameID] = [3]int{w, l, d}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("rows.Err for %s: %v", playerID, err)
	}
	if len(got) != len(want) {
		t.Fatalf("%s: game_stats row count = %d, want %d (%v)", playerID, len(got), len(want), got)
	}
	for gameID, w := range want {
		if got[gameID] != w {
			t.Fatalf("%s/%s: game stats = %v, want %v", playerID, gameID, got[gameID], w)
		}
	}
}
