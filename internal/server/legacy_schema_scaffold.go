package server

// legacyPunishmentPoolSchema/legacyContributionEnvelopeSchema：v43（punishment_migrate_v43.go）
// 之前的旧存储结构（"整表替换式"任务池/系列表 + 投稿信封两层模型），已被 sub_tasks/series
// 取代，运行时代码不再读写这里任何一张表。仅当数据库版本低于 v43 时，ensureSchema
// 才会临时创建它们：v25-v42 的历史迁移假设这些表存在，v43 完成数据搬运后立即删除。
// 不能把它们放回 allSchemas，否则已经升级的数据库会在每次重启时重新长出一批空旧表。
var legacyMigrationSchemas = []string{
	legacyPunishmentPoolSchema,
	legacyContributionEnvelopeSchema,
}

const legacyPunishmentPoolSchema = `
CREATE TABLE IF NOT EXISTS punishment_tasks (
	id                 TEXT PRIMARY KEY,
	text               TEXT NOT NULL DEFAULT '',
	tag_ids            TEXT NOT NULL DEFAULT '[]',
	faction_ids        TEXT NOT NULL DEFAULT '[]',
	difficulty_order   INTEGER NOT NULL DEFAULT 50,
	background_images  TEXT NOT NULL DEFAULT '[]',
	background_opacity REAL NOT NULL DEFAULT 0.22,
	sort_index         INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	content_version INTEGER NOT NULL DEFAULT 0,
	vote_version INTEGER NOT NULL DEFAULT 0,
	submission_id TEXT NOT NULL DEFAULT '',
	task_group_id TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS punishment_series (
	id                     TEXT PRIMARY KEY,
	name                   TEXT NOT NULL DEFAULT '',
	room_name_pool         TEXT NOT NULL DEFAULT '{}',
	room_background_images TEXT NOT NULL DEFAULT '[]',
	steps                  TEXT NOT NULL DEFAULT '[]',
	sort_index             INTEGER NOT NULL DEFAULT 0,
	contributor_player_id TEXT NOT NULL DEFAULT '',
	contributor_anonymous INTEGER NOT NULL DEFAULT 0,
	content_version INTEGER NOT NULL DEFAULT 0,
	vote_version INTEGER NOT NULL DEFAULT 0,
	submission_id TEXT NOT NULL DEFAULT '',
	target_faction_ids TEXT NOT NULL DEFAULT '[]',
	created_at INTEGER NOT NULL DEFAULT 0
);
`

const legacyContributionEnvelopeSchema = `
CREATE TABLE IF NOT EXISTS contribution_submissions (
	id TEXT PRIMARY KEY,
	kind TEXT NOT NULL,
	submitter_player_id TEXT NOT NULL,
	submitter_name_snapshot TEXT NOT NULL DEFAULT '',
	anonymous INTEGER NOT NULL DEFAULT 0,
	status TEXT NOT NULL,
	published_target_id TEXT NOT NULL DEFAULT '',
	published_version INTEGER NOT NULL DEFAULT 0,
	active_version INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0,
	submitted_at INTEGER NOT NULL DEFAULT 0,
	reviewed_at INTEGER NOT NULL DEFAULT 0,
	reviewed_by TEXT NOT NULL DEFAULT '',
	review_comment TEXT NOT NULL DEFAULT '',
	unpublish_requested_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_contrib_sub_submitter ON contribution_submissions(submitter_player_id, updated_at);
CREATE INDEX IF NOT EXISTS idx_contrib_sub_status ON contribution_submissions(status, kind, updated_at);
CREATE TABLE IF NOT EXISTS contribution_versions (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	submission_id TEXT NOT NULL,
	version INTEGER NOT NULL,
	content TEXT NOT NULL DEFAULT '{}',
	created_by TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL DEFAULT 0,
	reviewed_content TEXT NOT NULL DEFAULT '',
	reviewed_at INTEGER NOT NULL DEFAULT 0,
	reviewed_by TEXT NOT NULL DEFAULT '',
	review_result TEXT NOT NULL DEFAULT '',
	review_comment TEXT NOT NULL DEFAULT '',
	UNIQUE (submission_id, version),
	FOREIGN KEY (submission_id) REFERENCES contribution_submissions(id)
);
CREATE TABLE IF NOT EXISTS contribution_gender_claims (
	faction_id TEXT NOT NULL,
	normalized_label TEXT NOT NULL,
	submission_id TEXT NOT NULL,
	PRIMARY KEY (faction_id, normalized_label)
);
CREATE TABLE IF NOT EXISTS contribution_votes (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	round_id TEXT NOT NULL,
	punishment_event_id TEXT NOT NULL DEFAULT '',
	voter_player_id TEXT NOT NULL,
	target_kind TEXT NOT NULL,
	target_id TEXT NOT NULL,
	target_version INTEGER NOT NULL DEFAULT 0,
	vote INTEGER NOT NULL,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0,
	UNIQUE (round_id, voter_player_id, target_kind, target_id)
);
CREATE INDEX IF NOT EXISTS idx_contrib_votes_target ON contribution_votes(target_kind, target_id, target_version);
CREATE TABLE IF NOT EXISTS contribution_vote_overrides (
	target_kind TEXT NOT NULL,
	target_id TEXT NOT NULL,
	target_version INTEGER NOT NULL DEFAULT 0,
	display_ratio INTEGER NOT NULL,
	admin_id TEXT NOT NULL DEFAULT '',
	admin_note TEXT NOT NULL DEFAULT '',
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (target_kind, target_id, target_version)
);
CREATE TABLE IF NOT EXISTS contribution_round_participants (
	round_id TEXT NOT NULL,
	room_id TEXT NOT NULL DEFAULT '',
	player_id TEXT NOT NULL,
	role TEXT NOT NULL DEFAULT '',
	PRIMARY KEY (round_id, player_id)
);
CREATE TABLE IF NOT EXISTS contribution_images (
	path TEXT PRIMARY KEY,
	uploader_player_id TEXT NOT NULL,
	created_at INTEGER NOT NULL DEFAULT 0,
	submission_id TEXT NOT NULL DEFAULT '',
	published INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_contrib_images_uploader ON contribution_images(uploader_player_id);
`
