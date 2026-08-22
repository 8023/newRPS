package server

// migratePlayerSubTablesV47 把 players 表里四组"功能域状态"——名争、白给、极限模式、
// 分游戏胜负平，共 50 列——拆分到四张独立子表（player_name_war/player_giveaway/
// player_extreme_mode/player_game_stats，建表 DDL 见 playerstore.go 的 playerSchema）。
//
// 这四组列在 SQL 层从未被单独过滤/排序过，只在启动时整表读进内存、写回时整行覆盖
// （playerstore.go 的 loadAll/upsertInTx），拆分不损失任何查询能力；game_stats 这组会
// 随新游戏上线持续增长（jungle_wins/chess_wins 都是后补的 ALTER TABLE），拆成"一行一
// 游戏"之后新游戏只是多插入几行数据，表结构不用再变。
//
// 迁移分两步，全部在同一个迁移事务内完成（ensureSchema 已经把每条 migrate 包在事务里，
// 失败整体回滚，不会出现"新表已填、旧列未删"或反过来的半迁移状态）：
//  1. 用纯 SQL 的 INSERT...SELECT 把旧列的值原样搬进新表——每个持久化玩家在
//     player_name_war/player_giveaway/player_extreme_mode 各得到恰好一行，在
//     player_game_stats 恰好得到 7 行（每款游戏一行，从没打过也是 0/0/0）。用
//     INSERT...SELECT 而不是 Go 里读出来再拼回去，是因为 SQL 的 SELECT 对 NULL 的
//     透传是无损的——旧列是 NULL（"从未设置过"）就原样搬成新列的 NULL，不需要在 Go
//     侧手写一套 sql.NullInt64 判断 + 类型转换，也就不存在"位置参数数错、把两个字段
//     的值写反"这类改一次错一次的隐患。
//  2. 用 dropColumnIfExists 删掉 players 表上被搬空的 50 列。
//
// 四组各自独立用组内某一列是否存在来判断"这组旧数据还在不在"，而不是共用同一个判断——
// 现实中一张 players 表要么完整具备某一组的全部列，要么（该功能在这张表的历史上从未
// 存在过，比如全新库跑完整迁移链）完全没有，不会出现"组内缺列"这种半吊子状态，所以
// 逐组独立判断既够用又更松耦合：任何一组的判断失误不会连累另外三组。
func migratePlayerSubTablesV47(db sqlExecer) error {
	hasNameWar, err := tableHasColumn(db, "players", "name_war_enabled")
	if err != nil {
		return err
	}
	if hasNameWar {
		if _, err := db.Exec(`
			INSERT INTO player_name_war (
				player_id, enabled, allow_rename, toggled_at, original_name, penalty_name,
				punished, rename_protected_until, renamed_by, renamed_by_name,
				rename_window_started_at, rename_count
			)
			SELECT id, name_war_enabled, name_war_allow_rename, name_war_toggled_at, name_war_original_name,
				name_war_penalty_name, name_war_punished, name_war_rename_protected_until,
				name_war_renamed_by, name_war_renamed_by_name, name_war_rename_window_started_at, name_war_rename_count
			FROM players`); err != nil {
			return err
		}
	}

	hasGiveaway, err := tableHasColumn(db, "players", "giveaway_enabled")
	if err != nil {
		return err
	}
	if hasGiveaway {
		if _, err := db.Exec(`
			INSERT INTO player_giveaway (
				player_id, enabled, value, clicks, board_text, board_submitted_at, board_expires_at,
				board_likes, board_dislikes
			)
			SELECT id, giveaway_enabled, giveaway_value, giveaway_clicks, giveaway_board_text,
				giveaway_board_submitted_at, giveaway_board_expires_at, giveaway_board_likes, giveaway_board_dislikes
			FROM players`); err != nil {
			return err
		}
	}

	hasExtreme, err := tableHasColumn(db, "players", "extreme_mode_enabled")
	if err != nil {
		return err
	}
	if hasExtreme {
		if _, err := db.Exec(`
			INSERT INTO player_extreme_mode (
				player_id, enabled, toggled_at, cooldown_until, win_streak, last_decay_hour,
				force_closed, force_closed_at, rename_protected_until, renamed_by, renamed_by_name
			)
			SELECT id, extreme_mode_enabled, extreme_mode_toggled_at, extreme_mode_cooldown_until,
				extreme_win_streak, extreme_last_decay_hour, extreme_force_closed, extreme_force_closed_at,
				extreme_rename_protected_until, extreme_renamed_by, extreme_renamed_by_name
			FROM players`); err != nil {
			return err
		}
	}

	// game_id 取值须与 gameIDsForStats（playerstore.go）/ types.GameID 常量完全一致。
	// 每款游戏各自的三列独立判断是否存在——jungle_wins/chess_wins 是后补的 ALTER TABLE，
	// 在"卡在更早版本"的库上可能确实还没有，不能被其余游戏的存在与否连累。
	gameColumns := []struct {
		gameID                       string
		winsCol, lossesCol, drawsCol string
	}{
		{"rps", "rps_wins", "rps_losses", "rps_draws"},
		{"othello", "othello_wins", "othello_losses", "othello_draws"},
		{"tictactoe", "tictactoe_wins", "tictactoe_losses", "tictactoe_draws"},
		{"gomoku", "gomoku_wins", "gomoku_losses", "gomoku_draws"},
		{"liarsdice", "liarsdice_wins", "liarsdice_losses", "liarsdice_draws"},
		{"jungle", "jungle_wins", "jungle_losses", "jungle_draws"},
		{"chess", "chess_wins", "chess_losses", "chess_draws"},
	}
	for _, g := range gameColumns {
		hasGame, err := tableHasColumn(db, "players", g.winsCol)
		if err != nil {
			return err
		}
		if !hasGame {
			continue
		}
		q := `INSERT INTO player_game_stats (player_id, game_id, wins, losses, draws)
			SELECT id, '` + g.gameID + `', ` + g.winsCol + `, ` + g.lossesCol + `, ` + g.drawsCol + ` FROM players`
		if _, err := db.Exec(q); err != nil {
			return err
		}
	}

	oldColumns := []string{
		"name_war_enabled", "name_war_allow_rename", "name_war_toggled_at", "name_war_original_name",
		"name_war_penalty_name", "name_war_punished", "name_war_rename_protected_until",
		"name_war_renamed_by", "name_war_renamed_by_name", "name_war_rename_window_started_at", "name_war_rename_count",
		"giveaway_enabled", "giveaway_value", "giveaway_clicks",
		"giveaway_board_text", "giveaway_board_submitted_at", "giveaway_board_expires_at",
		"giveaway_board_likes", "giveaway_board_dislikes",
		"extreme_mode_enabled", "extreme_mode_toggled_at", "extreme_mode_cooldown_until",
		"extreme_win_streak", "extreme_last_decay_hour",
		"extreme_force_closed", "extreme_force_closed_at", "extreme_rename_protected_until",
		"extreme_renamed_by", "extreme_renamed_by_name",
		"rps_wins", "rps_losses", "rps_draws",
		"othello_wins", "othello_losses", "othello_draws",
		"tictactoe_wins", "tictactoe_losses", "tictactoe_draws",
		"gomoku_wins", "gomoku_losses", "gomoku_draws",
		"liarsdice_wins", "liarsdice_losses", "liarsdice_draws",
		"jungle_wins", "jungle_losses", "jungle_draws",
		"chess_wins", "chess_losses", "chess_draws",
	}
	for _, col := range oldColumns {
		if err := dropColumnIfExists(db, "players", col); err != nil {
			return err
		}
	}
	return nil
}
