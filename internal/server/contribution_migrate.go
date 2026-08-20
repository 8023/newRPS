package server

const legacyPunishmentContributorID = "QEuBIrirIxpd"

func migrateContributionV33(db sqlExecer) error {
	taskCols := [][2]string{
		{"contributor_player_id", "TEXT NOT NULL DEFAULT ''"},
		{"contributor_anonymous", "INTEGER NOT NULL DEFAULT 0"},
		{"content_version", "INTEGER NOT NULL DEFAULT 0"},
		{"vote_version", "INTEGER NOT NULL DEFAULT 0"},
		{"submission_id", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, col := range taskCols {
		if err := addColumnIfMissing(db, "punishment_tasks", col[0], col[1]); err != nil {
			return err
		}
		if err := addColumnIfMissing(db, "punishment_series", col[0], col[1]); err != nil {
			return err
		}
	}
	eventCols := [][2]string{
		{"round_id", "TEXT NOT NULL DEFAULT ''"},
		{"formal_task_id", "TEXT NOT NULL DEFAULT ''"},
		{"formal_task_version", "INTEGER NOT NULL DEFAULT 0"},
		{"formal_series_id", "TEXT NOT NULL DEFAULT ''"},
		{"formal_series_version", "INTEGER NOT NULL DEFAULT 0"},
		{"contributor_player_id", "TEXT NOT NULL DEFAULT ''"},
		{"contributor_name_snapshot", "TEXT NOT NULL DEFAULT ''"},
		{"contributor_anonymous", "INTEGER NOT NULL DEFAULT 0"},
		{"completed_at", "INTEGER"},
	}
	for _, col := range eventCols {
		if err := addColumnIfMissing(db, "punishment_events", col[0], col[1]); err != nil {
			return err
		}
	}
	// 已有任务/系列（在共建投稿功能上线前由管理员直接在后台添加，从未走过投稿+审批流程）
	// 统一按「8023 本人非匿名投稿、已审批」补记归属，让展示效果与投票逻辑和新投稿内容一致——
	// 这是产品决策：上线后管理员不再直接后台加任务，而是自己注册玩家账号投稿再审批。
	// 只覆盖 contributor_player_id 为空（即从未被 upsertTaskTx/upsertSeriesTx 写过）的旧行，
	// 不会碰到任何已经走过投稿流程的新数据。
	if _, err := db.Exec(
		`UPDATE punishment_tasks SET contributor_player_id = ?, contributor_anonymous = 0, content_version = 1, vote_version = 1
		 WHERE contributor_player_id = ''`, legacyPunishmentContributorID); err != nil {
		return err
	}
	if _, err := db.Exec(
		`UPDATE punishment_series SET contributor_player_id = ?, contributor_anonymous = 0, content_version = 1, vote_version = 1
		 WHERE contributor_player_id = ''`, legacyPunishmentContributorID); err != nil {
		return err
	}
	return nil
}
