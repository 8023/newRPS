package server

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/doumiao/newRPS/internal/types"
)

// playerStore 玩家档案 + 设备密钥的 SQLite 持久化（与聊天/事件共用 database.db）。
// 运行时权威状态仍在内存 map；本 store 只负责启动加载与脏写落盘。
type playerStore struct {
	db *sql.DB
	mu sync.Mutex
}

const playerSchema = `
CREATE TABLE IF NOT EXISTS players (
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
CREATE INDEX IF NOT EXISTS idx_players_player_id ON players(player_id);
CREATE INDEX IF NOT EXISTS idx_players_created_at ON players(created_at);
CREATE INDEX IF NOT EXISTS idx_players_last_seen_at ON players(last_seen_at);
CREATE TABLE IF NOT EXISTS player_secrets (
	player_id TEXT NOT NULL,
	secret TEXT NOT NULL,
	PRIMARY KEY (player_id, secret),
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_player_secrets_player ON player_secrets(player_id);
`

func newPlayerStore(db *sql.DB) *playerStore {
	return &playerStore{db: db}
}

// playerRow 从库中读出的扁平行（不含性别展示字段，加载时再 genderInfo）。
type playerRow struct {
	item    persistedPlayer
	secrets []string
}

func (ps *playerStore) count() (int, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	var n int
	err := ps.db.QueryRow(`SELECT COUNT(*) FROM players`).Scan(&n)
	return n, err
}

func (ps *playerStore) loadAll() ([]playerRow, error) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	// 注意：db.SetMaxOpenConns(1)，不能在 rows 未 Close 时再 Query（会死锁）。
	// 先扫完 players，再一次性加载全部 secrets。
	rows, err := ps.db.Query(`
		SELECT id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			name_war_enabled, name_war_allow_rename, name_war_toggled_at, name_war_original_name,
			name_war_penalty_name, name_war_punished, name_war_rename_protected_until,
			name_war_renamed_by, name_war_renamed_by_name, name_war_rename_window_started_at, name_war_rename_count,
			giveaway_enabled, giveaway_value, giveaway_clicks,
			rank_multiplier_unlocked,
			extreme_mode_enabled, extreme_mode_toggled_at, extreme_mode_cooldown_until,
			extreme_win_streak, extreme_last_decay_hour, ranked_last_decay_day,
			push_mention_enabled, push_turn_enabled, push_seat_enabled, push_bond_enabled,
			wins, losses, draws, punishments, ranked_points, highest_score, lowest_score, title, title_segment_id, title_custom, self_title,
			rps_wins, rps_losses, rps_draws,
			othello_wins, othello_losses, othello_draws,
			tictactoe_wins, tictactoe_losses, tictactoe_draws,
			gomoku_wins, gomoku_losses, gomoku_draws,
			liarsdice_wins, liarsdice_losses, liarsdice_draws,
			jungle_wins, jungle_losses, jungle_draws,
			chess_wins, chess_losses, chess_draws,
			total_online_ms,
			bond_master_enabled, bond_pet_enabled, bond_public_display,
			created_at, last_seen_at
		FROM players`)
	if err != nil {
		return nil, err
	}

	var items []persistedPlayer
	for rows.Next() {
		var item persistedPlayer
		var (
			nwEnabled, nwAllow, nwPunished          sql.NullInt64
			nwToggled, nwProt, nwWinStart, nwRename sql.NullInt64
			gaEnabled, rankUnlock, exEnabled        sql.NullInt64
			exToggled, exCool, exStreak, exDecay    sql.NullInt64
			pushM, pushT, pushS, pushB              sql.NullInt64
			gaValue                                 sql.NullFloat64
			gaClicks                                sql.NullInt64
			rankedDecay                             sql.NullInt64
			bondMaster, bondPet, bondPublic         sql.NullInt64
		)
		err := rows.Scan(
			&item.ID, &item.PlayerID, &item.ClaimKey, &item.Name, &item.GenderID, &item.FactionID, &item.AvatarURL,
			&nwEnabled, &nwAllow, &nwToggled, &item.NameWarOriginalName,
			&item.NameWarPenaltyName, &nwPunished, &nwProt,
			&item.NameWarRenamedBy, &item.NameWarRenamedByName, &nwWinStart, &nwRename,
			&gaEnabled, &gaValue, &gaClicks,
			&rankUnlock,
			&exEnabled, &exToggled, &exCool,
			&exStreak, &exDecay, &rankedDecay,
			&pushM, &pushT, &pushS, &pushB,
			&item.Stats.Wins, &item.Stats.Losses, &item.Stats.Draws, &item.Stats.Punishments, &item.Stats.RankedPoints, &item.Stats.HighestScore, &item.Stats.LowestScore, &item.Stats.Title, &item.Stats.TitleSegmentID, &item.Stats.TitleCustom, &item.Stats.SelfTitle,
			&item.GameStats.RPS.Wins, &item.GameStats.RPS.Losses, &item.GameStats.RPS.Draws,
			&item.GameStats.Othello.Wins, &item.GameStats.Othello.Losses, &item.GameStats.Othello.Draws,
			&item.GameStats.TicTacToe.Wins, &item.GameStats.TicTacToe.Losses, &item.GameStats.TicTacToe.Draws,
			&item.GameStats.Gomoku.Wins, &item.GameStats.Gomoku.Losses, &item.GameStats.Gomoku.Draws,
			&item.GameStats.LiarsDice.Wins, &item.GameStats.LiarsDice.Losses, &item.GameStats.LiarsDice.Draws,
			&item.GameStats.Jungle.Wins, &item.GameStats.Jungle.Losses, &item.GameStats.Jungle.Draws,
			&item.GameStats.Chess.Wins, &item.GameStats.Chess.Losses, &item.GameStats.Chess.Draws,
			&item.Stats.TotalOnlineMs,
			&bondMaster, &bondPet, &bondPublic,
			&item.CreatedAt, &item.LastSeenAt,
		)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		item.NameWarEnabled = nullIntToBoolPtr(nwEnabled)
		item.NameWarAllowRename = nullIntToBoolPtr(nwAllow)
		item.NameWarPunished = nullIntToBoolPtr(nwPunished)
		item.NameWarToggledAt = nullIntToInt64Ptr(nwToggled)
		item.NameWarRenameProtectedUntil = nullIntToInt64Ptr(nwProt)
		item.NameWarRenameWindowStartedAt = nullIntToInt64Ptr(nwWinStart)
		item.NameWarRenameCount = nullIntToIntPtr(nwRename)
		item.GiveawayEnabled = nullIntToBoolPtr(gaEnabled)
		if gaValue.Valid {
			v := gaValue.Float64
			item.GiveawayValue = &v
		}
		item.GiveawayClicks = nullIntToIntPtr(gaClicks)
		item.RankMultiplierUnlocked = nullIntToBoolPtr(rankUnlock)
		item.ExtremeModeEnabled = nullIntToBoolPtr(exEnabled)
		item.ExtremeModeToggledAt = nullIntToInt64Ptr(exToggled)
		item.ExtremeModeCooldownUntil = nullIntToInt64Ptr(exCool)
		item.ExtremeWinStreak = nullIntToIntPtr(exStreak)
		item.ExtremeLastDecayHour = nullIntToInt64Ptr(exDecay)
		item.RankedLastDecayDay = nullIntToInt64Ptr(rankedDecay)
		item.PushMentionEnabled = nullIntToBoolPtr(pushM)
		item.PushTurnEnabled = nullIntToBoolPtr(pushT)
		item.PushSeatEnabled = nullIntToBoolPtr(pushS)
		item.PushBondEnabled = nullIntToBoolPtr(pushB)
		item.BondMasterEnabled = nullIntToBoolPtr(bondMaster)
		item.BondPetEnabled = nullIntToBoolPtr(bondPet)
		item.BondPublicDisplay = nullIntToBoolPtr(bondPublic)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return nil, err
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}

	secretsByPlayer, err := ps.loadAllSecretsLocked()
	if err != nil {
		return nil, err
	}
	out := make([]playerRow, 0, len(items))
	for _, item := range items {
		secrets := secretsByPlayer[item.ID]
		item.PlayerSecrets = secrets
		out = append(out, playerRow{item: item, secrets: secrets})
	}
	return out, nil
}

func (ps *playerStore) loadAllSecretsLocked() (map[string][]string, error) {
	rows, err := ps.db.Query(`SELECT player_id, secret FROM player_secrets`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]string{}
	for rows.Next() {
		var pid, secret string
		if err := rows.Scan(&pid, &secret); err != nil {
			return nil, err
		}
		if secret == "" {
			continue
		}
		out[pid] = append(out[pid], secret)
	}
	return out, rows.Err()
}

// upsert 写入/更新一名玩家及其全部设备密钥（事务内替换 secrets）。
func (ps *playerStore) upsert(item persistedPlayer) error {
	ps.mu.Lock()
	defer ps.mu.Unlock()
	return ps.upsertLocked(item)
}

func (ps *playerStore) upsertMany(items []persistedPlayer) error {
	if len(items) == 0 {
		return nil
	}
	ps.mu.Lock()
	defer ps.mu.Unlock()
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	for _, item := range items {
		if err := ps.upsertInTx(tx, item); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (ps *playerStore) upsertLocked(item persistedPlayer) error {
	tx, err := ps.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if err := ps.upsertInTx(tx, item); err != nil {
		return err
	}
	return tx.Commit()
}

func (ps *playerStore) upsertInTx(tx *sql.Tx, item persistedPlayer) error {
	if item.ID == "" || item.PlayerID == "" {
		return fmt.Errorf("playerstore: missing id/playerId")
	}
	gs := item.GameStats
	_, err := tx.Exec(`
		INSERT INTO players (
			id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			name_war_enabled, name_war_allow_rename, name_war_toggled_at, name_war_original_name,
			name_war_penalty_name, name_war_punished, name_war_rename_protected_until,
			name_war_renamed_by, name_war_renamed_by_name, name_war_rename_window_started_at, name_war_rename_count,
			giveaway_enabled, giveaway_value, giveaway_clicks,
			rank_multiplier_unlocked,
			extreme_mode_enabled, extreme_mode_toggled_at, extreme_mode_cooldown_until,
			extreme_win_streak, extreme_last_decay_hour, ranked_last_decay_day,
			push_mention_enabled, push_turn_enabled, push_seat_enabled, push_bond_enabled,
			wins, losses, draws, punishments, ranked_points, highest_score, lowest_score, title, title_segment_id, title_custom, self_title,
			rps_wins, rps_losses, rps_draws,
			othello_wins, othello_losses, othello_draws,
			tictactoe_wins, tictactoe_losses, tictactoe_draws,
			gomoku_wins, gomoku_losses, gomoku_draws,
			liarsdice_wins, liarsdice_losses, liarsdice_draws,
			jungle_wins, jungle_losses, jungle_draws,
			chess_wins, chess_losses, chess_draws,
			total_online_ms,
			bond_master_enabled, bond_pet_enabled, bond_public_display,
			created_at, last_seen_at
		) VALUES (
			?,?,?,?,?,?,?,
			?,?,?,?,?,?,?,?,?,?,?,
			?,?,?,
			?,
			?,?,?,?,?,?,
			?,?,?,?,
			?,?,?,?,?,?,?,?,?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,?,?,
			?,
			?,?,?,
			?,?
		)
		ON CONFLICT(id) DO UPDATE SET
			player_id=excluded.player_id,
			claim_key=excluded.claim_key,
			name=excluded.name,
			gender_id=excluded.gender_id,
			faction_id=excluded.faction_id,
			avatar_url=excluded.avatar_url,
			name_war_enabled=excluded.name_war_enabled,
			name_war_allow_rename=excluded.name_war_allow_rename,
			name_war_toggled_at=excluded.name_war_toggled_at,
			name_war_original_name=excluded.name_war_original_name,
			name_war_penalty_name=excluded.name_war_penalty_name,
			name_war_punished=excluded.name_war_punished,
			name_war_rename_protected_until=excluded.name_war_rename_protected_until,
			name_war_renamed_by=excluded.name_war_renamed_by,
			name_war_renamed_by_name=excluded.name_war_renamed_by_name,
			name_war_rename_window_started_at=excluded.name_war_rename_window_started_at,
			name_war_rename_count=excluded.name_war_rename_count,
			giveaway_enabled=excluded.giveaway_enabled,
			giveaway_value=excluded.giveaway_value,
			giveaway_clicks=excluded.giveaway_clicks,
			rank_multiplier_unlocked=excluded.rank_multiplier_unlocked,
			extreme_mode_enabled=excluded.extreme_mode_enabled,
			extreme_mode_toggled_at=excluded.extreme_mode_toggled_at,
			extreme_mode_cooldown_until=excluded.extreme_mode_cooldown_until,
			extreme_win_streak=excluded.extreme_win_streak,
			extreme_last_decay_hour=excluded.extreme_last_decay_hour,
			ranked_last_decay_day=excluded.ranked_last_decay_day,
			push_mention_enabled=excluded.push_mention_enabled,
			push_turn_enabled=excluded.push_turn_enabled,
			push_seat_enabled=excluded.push_seat_enabled,
			push_bond_enabled=excluded.push_bond_enabled,
			wins=excluded.wins, losses=excluded.losses, draws=excluded.draws,
			punishments=excluded.punishments, ranked_points=excluded.ranked_points,
			highest_score=excluded.highest_score, lowest_score=excluded.lowest_score,
			title=excluded.title, title_segment_id=excluded.title_segment_id, title_custom=excluded.title_custom, self_title=excluded.self_title,
			rps_wins=excluded.rps_wins, rps_losses=excluded.rps_losses, rps_draws=excluded.rps_draws,
			othello_wins=excluded.othello_wins, othello_losses=excluded.othello_losses, othello_draws=excluded.othello_draws,
			tictactoe_wins=excluded.tictactoe_wins, tictactoe_losses=excluded.tictactoe_losses, tictactoe_draws=excluded.tictactoe_draws,
			gomoku_wins=excluded.gomoku_wins, gomoku_losses=excluded.gomoku_losses, gomoku_draws=excluded.gomoku_draws,
			liarsdice_wins=excluded.liarsdice_wins, liarsdice_losses=excluded.liarsdice_losses, liarsdice_draws=excluded.liarsdice_draws,
			jungle_wins=excluded.jungle_wins, jungle_losses=excluded.jungle_losses, jungle_draws=excluded.jungle_draws,
			chess_wins=excluded.chess_wins, chess_losses=excluded.chess_losses, chess_draws=excluded.chess_draws,
			total_online_ms=excluded.total_online_ms,
			bond_master_enabled=excluded.bond_master_enabled,
			bond_pet_enabled=excluded.bond_pet_enabled,
			bond_public_display=excluded.bond_public_display,
			created_at=excluded.created_at, last_seen_at=excluded.last_seen_at
	`,
		item.ID, item.PlayerID, item.ClaimKey, item.Name, item.GenderID, item.FactionID, item.AvatarURL,
		boolPtrToSQL(item.NameWarEnabled), boolPtrToSQL(item.NameWarAllowRename), int64PtrToSQL(item.NameWarToggledAt), item.NameWarOriginalName,
		item.NameWarPenaltyName, boolPtrToSQL(item.NameWarPunished), int64PtrToSQL(item.NameWarRenameProtectedUntil),
		item.NameWarRenamedBy, item.NameWarRenamedByName, int64PtrToSQL(item.NameWarRenameWindowStartedAt), intPtrToSQL(item.NameWarRenameCount),
		boolPtrToSQL(item.GiveawayEnabled), floatPtrToSQL(item.GiveawayValue), intPtrToSQL(item.GiveawayClicks),
		boolPtrToSQL(item.RankMultiplierUnlocked),
		boolPtrToSQL(item.ExtremeModeEnabled), int64PtrToSQL(item.ExtremeModeToggledAt), int64PtrToSQL(item.ExtremeModeCooldownUntil),
		intPtrToSQL(item.ExtremeWinStreak), int64PtrToSQL(item.ExtremeLastDecayHour),
		int64PtrToSQL(item.RankedLastDecayDay),
		boolPtrToSQL(item.PushMentionEnabled), boolPtrToSQL(item.PushTurnEnabled), boolPtrToSQL(item.PushSeatEnabled), boolPtrToSQL(item.PushBondEnabled),
		item.Stats.Wins, item.Stats.Losses, item.Stats.Draws, item.Stats.Punishments, item.Stats.RankedPoints, item.Stats.HighestScore, item.Stats.LowestScore, item.Stats.Title, item.Stats.TitleSegmentID, item.Stats.TitleCustom, item.Stats.SelfTitle,
		gs.RPS.Wins, gs.RPS.Losses, gs.RPS.Draws,
		gs.Othello.Wins, gs.Othello.Losses, gs.Othello.Draws,
		gs.TicTacToe.Wins, gs.TicTacToe.Losses, gs.TicTacToe.Draws,
		gs.Gomoku.Wins, gs.Gomoku.Losses, gs.Gomoku.Draws,
		gs.LiarsDice.Wins, gs.LiarsDice.Losses, gs.LiarsDice.Draws,
		gs.Jungle.Wins, gs.Jungle.Losses, gs.Jungle.Draws,
		gs.Chess.Wins, gs.Chess.Losses, gs.Chess.Draws,
		item.Stats.TotalOnlineMs,
		boolPtrToSQL(item.BondMasterEnabled), boolPtrToSQL(item.BondPetEnabled), boolPtrToSQL(item.BondPublicDisplay),
		item.CreatedAt, item.LastSeenAt,
	)
	if err != nil {
		return err
	}
	if _, err := tx.Exec(`DELETE FROM player_secrets WHERE player_id = ?`, item.ID); err != nil {
		return err
	}
	for _, secret := range item.PlayerSecrets {
		if secret == "" {
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO player_secrets (player_id, secret) VALUES (?, ?)`, item.ID, secret); err != nil {
			return err
		}
	}
	return nil
}

func boolPtrToSQL(p *bool) interface{} {
	if p == nil {
		return nil
	}
	if *p {
		return 1
	}
	return 0
}

func intPtrToSQL(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func int64PtrToSQL(p *int64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func floatPtrToSQL(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullIntToBoolPtr(n sql.NullInt64) *bool {
	if !n.Valid {
		return nil
	}
	v := n.Int64 != 0
	return &v
}

func nullIntToIntPtr(n sql.NullInt64) *int {
	if !n.Valid {
		return nil
	}
	v := int(n.Int64)
	return &v
}

func nullIntToInt64Ptr(n sql.NullInt64) *int64 {
	if !n.Valid {
		return nil
	}
	v := n.Int64
	return &v
}

// 编译期用到 types 的占位，避免仅 schema 文件时误删 import。
var _ = types.GameStats{}
