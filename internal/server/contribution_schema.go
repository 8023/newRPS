package server

const contributionSchema = `
CREATE TABLE IF NOT EXISTS gender_factions (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL DEFAULT '',
	normalized_label TEXT NOT NULL DEFAULT '',
	text_color TEXT NOT NULL DEFAULT '',
	background_color TEXT NOT NULL DEFAULT '',
	border_color TEXT NOT NULL DEFAULT '',
	task_group TEXT NOT NULL DEFAULT 'default',
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS gender_options (
	id TEXT PRIMARY KEY,
	label TEXT NOT NULL DEFAULT '',
	normalized_label TEXT NOT NULL DEFAULT '',
	faction_id TEXT NOT NULL,
	sort_index INTEGER NOT NULL DEFAULT 0,
	created_at INTEGER NOT NULL DEFAULT 0,
	updated_at INTEGER NOT NULL DEFAULT 0,
	FOREIGN KEY (faction_id) REFERENCES gender_factions(id),
	UNIQUE (faction_id, normalized_label)
);
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
