package server

// contributionSchema：共建投稿的存储结构。核心是两张版本化、插入不更新的表——
// sub_tasks（随机任务 + 系列每一步，二合一）与 series（系列元信息，不含步骤内容）。
// 同一个逻辑任务/系列永远用同一个 id，每次内容编辑都插入新的 (id, version) 行、版本历史
// 永久保留；纯状态流转（提交审批/通过/驳回/下架）不产生新版本，直接在当前那一行上
// UPDATE status。"取当前生效版本"统一是
// "SELECT * FROM <table> WHERE id=? AND status='approved' ORDER BY version DESC LIMIT 1"。
//
// 图片、点赞点踩都不再单独建表：background_image 是 sub_tasks 一行上的单值字段（改图=
// 插入新版本行，旧图所在的旧版本行永久保留，不会被覆盖/清空）；like_count/down_count
// 直接累加在任务对应的精确版本行上；防重复依据 punishment_events 两侧投票列，
// 事件防重与计数累加在同一事务完成（见 eventstore.go），这里的计数也是持久化审计数据。
const contributionSchema = `
CREATE TABLE IF NOT EXISTS sub_tasks (
	id                  TEXT NOT NULL,
	version             INTEGER NOT NULL,
	series_id           TEXT NOT NULL DEFAULT '',
	step_index          INTEGER NOT NULL DEFAULT 0,
	active              INTEGER NOT NULL DEFAULT 1,
	variants            TEXT NOT NULL DEFAULT '[]',
	tag_ids             TEXT NOT NULL DEFAULT '[]',
	background_image    TEXT NOT NULL DEFAULT '',
	background_opacity  REAL NOT NULL DEFAULT 0.22,
	difficulty_order    INTEGER NOT NULL DEFAULT 50,
	status              TEXT NOT NULL,
	like_count          INTEGER NOT NULL DEFAULT 0,
	down_count          INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	-- contributor_name：v48 迁移已 DROP（昵称不做快照，统一按 contributor_player_id
	-- 现查 s.players）。这里仍保留在 CREATE TABLE 里，是因为更早的迁移
	-- （execSchemaQuarantiningLegacyTables 的隔离重建、punishment_migrate_v43.go 的
	-- 存量搬迁）按列名显式写过它，DDL 提前少一列会让那些老库升级路径直接报错；
	-- v48 会在它们都跑完后把这列删掉，新库与升级完的老库最终结构一致。
	contributor_name      TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	reviewed_by         TEXT NOT NULL DEFAULT '',
	reviewed_at         INTEGER NOT NULL DEFAULT 0,
	review_comment      TEXT NOT NULL DEFAULT '',
	created_at          INTEGER NOT NULL DEFAULT 0,
	updated_at          INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (id, version)
);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_series   ON sub_tasks(series_id, active, step_index);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_status   ON sub_tasks(status, difficulty_order);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_bg       ON sub_tasks(background_image);
CREATE INDEX IF NOT EXISTS idx_sub_tasks_owner    ON sub_tasks(contributor_player_id, id, version);
CREATE TABLE IF NOT EXISTS series (
	id                     TEXT NOT NULL,
	version                INTEGER NOT NULL,
	name                   TEXT NOT NULL DEFAULT '',
	target_faction_ids     TEXT NOT NULL DEFAULT '[]',
	room_name_pool         TEXT NOT NULL DEFAULT '{}',
	room_background_images TEXT NOT NULL DEFAULT '[]',
	status                 TEXT NOT NULL,
	contributor_player_id  TEXT NOT NULL DEFAULT '',
	-- contributor_name：与 sub_tasks 同一约定，v48 迁移已 DROP，见上方注释。
	contributor_name       TEXT NOT NULL DEFAULT '',
	contributor_anonymous  INTEGER NOT NULL DEFAULT 0,
	reviewed_by            TEXT NOT NULL DEFAULT '',
	reviewed_at            INTEGER NOT NULL DEFAULT 0,
	review_comment         TEXT NOT NULL DEFAULT '',
	created_at             INTEGER NOT NULL DEFAULT 0,
	updated_at             INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (id, version)
);
CREATE INDEX IF NOT EXISTS idx_series_status ON series(status);
CREATE INDEX IF NOT EXISTS idx_series_owner  ON series(contributor_player_id, id, version);
-- punishment_series_run_stats：系列任务"完成率"的运行时统计——房间销毁时，把房间里每个
-- 至少走完 1 步的玩家各自的"已走步数 / 系列总步数"百分比记一笔样本；participant_count
-- 是样本数，progress_percent_sum 是样本值之和，完成率 = sum/count（算术平均，不是"是否
-- 走到最后一步"的二元判定）。按 series_id+series_version 分桶，系列改版重发后从 0 重新计。
-- 见 punishment.go 的 recordSeriesRunProgressOnClose。
CREATE TABLE IF NOT EXISTS punishment_series_run_stats (
	series_id      TEXT NOT NULL,
	series_version INTEGER NOT NULL,
	participant_count     INTEGER NOT NULL DEFAULT 0,
	progress_percent_sum  INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (series_id, series_version)
);
`
