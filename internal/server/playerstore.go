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

// playerSchema：players 主表只保留身份字段、聚合战绩缓存（wins/losses/draws 等，由
// SyncTotalsFromGameStats 从下面的 player_game_stats 同步而来的"总榜"展示值）与推送/认主
// 开关这类始终只有几列、不会持续增长的字段。四组"功能域状态"——名争、白给、极限模式、
// 分游戏胜负平——各自拆到独立子表（v47 迁移，见 player_migrate_v47.go），原因：
//  1. 这四组在 SQL 层从未被按列过滤/排序过，只在启动时整表读进内存、写回时整行覆盖，
//     拆分不损失任何查询能力；
//  2. game_stats 这组会随新游戏上线持续增长（jungle_wins/chess_wins 都是后补的 ALTER
//     TABLE），拆成"一行一游戏"之后新游戏只是多插入几行数据，表结构不用再变；
//  3. players 表原本 78 个位置参数的单条 INSERT，字段顺序错位的隐患随字段数下降而下降。
//
// persistedPlayer（persist.go）与 PlayerState 内存形状不变，拆分只发生在这个文件的
// SQL 读写层——业务代码不需要感知这四张子表的存在。
const playerSchema = `
CREATE TABLE IF NOT EXISTS players (
	id TEXT PRIMARY KEY,
	player_id TEXT NOT NULL UNIQUE,
	claim_key TEXT NOT NULL DEFAULT '',
	name TEXT NOT NULL DEFAULT '',
	gender_id TEXT NOT NULL DEFAULT '',
	faction_id TEXT NOT NULL DEFAULT '',
	avatar_url TEXT NOT NULL DEFAULT '',
	rank_multiplier_unlocked INTEGER,
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
-- player_name_war/player_giveaway/player_extreme_mode：与 players 一对一，每个持久化
-- 玩家固定有且只有一行（loadAll/upsertInTx 无条件维护，不做"行缺失=从未使用过"的稀疏
-- 判断），列语义与 v46 之前同名的 players 列完全一致，NULL 仍表示"从未设置过"。
CREATE TABLE IF NOT EXISTS player_name_war (
	player_id TEXT PRIMARY KEY,
	enabled INTEGER,
	allow_rename INTEGER,
	toggled_at INTEGER,
	original_name TEXT NOT NULL DEFAULT '',
	penalty_name TEXT NOT NULL DEFAULT '',
	punished INTEGER,
	rename_protected_until INTEGER,
	renamed_by TEXT NOT NULL DEFAULT '',
	renamed_by_name TEXT NOT NULL DEFAULT '',
	rename_window_started_at INTEGER,
	rename_count INTEGER,
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS player_giveaway (
	player_id TEXT PRIMARY KEY,
	enabled INTEGER,
	value REAL,
	clicks INTEGER,
	board_text TEXT NOT NULL DEFAULT '',
	board_submitted_at INTEGER,
	board_expires_at INTEGER,
	board_likes INTEGER,
	board_dislikes INTEGER,
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
CREATE TABLE IF NOT EXISTS player_extreme_mode (
	player_id TEXT PRIMARY KEY,
	enabled INTEGER,
	toggled_at INTEGER,
	cooldown_until INTEGER,
	win_streak INTEGER,
	last_decay_hour INTEGER,
	force_closed INTEGER,
	force_closed_at INTEGER,
	rename_protected_until INTEGER,
	renamed_by TEXT NOT NULL DEFAULT '',
	renamed_by_name TEXT NOT NULL DEFAULT '',
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
-- player_game_stats：一行一款游戏的胜负平，取代原先 7 款游戏各 3 列、共 21 列的平铺
-- 结构。新游戏上线只需多 INSERT 几行，不用再 ALTER TABLE。每个持久化玩家固定有 7 行
-- （每款游戏各一行，哪怕从没打过也是 0/0/0），与 player_name_war 等表同样的"稠密"
-- 设计：换来的是行数=玩家数×游戏数这个不变量，既让迁移正确性可以直接用行数核对，也让
-- loadAll/upsertInTx 不需要"这一行存在与否"的分支逻辑。
CREATE TABLE IF NOT EXISTS player_game_stats (
	player_id TEXT NOT NULL,
	game_id   TEXT NOT NULL,
	wins      INTEGER NOT NULL DEFAULT 0,
	losses    INTEGER NOT NULL DEFAULT 0,
	draws     INTEGER NOT NULL DEFAULT 0,
	PRIMARY KEY (player_id, game_id),
	FOREIGN KEY (player_id) REFERENCES players(id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_player_game_stats_player ON player_game_stats(player_id);
`

func newPlayerStore(db *sql.DB) *playerStore {
	return &playerStore{db: db}
}

// playerRow 从库中读出的扁平行（不含性别展示字段，加载时再 genderInfo）。
type playerRow struct {
	item    persistedPlayer
	secrets []string
}

// gameIDsForStats：player_game_stats 覆盖的全部游戏，loadAll/upsertInTx 都按这个顺序
// 遍历。新游戏上线只需要在这里加一项，不需要 ALTER TABLE。
var gameIDsForStats = []types.GameID{
	types.GameRPS, types.GameOthello, types.GameTicTacToe, types.GameGomoku,
	types.GameLiarsDice, types.GameJungle, types.GameChess,
}

func knownGameStatsID(gameID types.GameID) bool {
	for _, known := range gameIDsForStats {
		if gameID == known {
			return true
		}
	}
	return false
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

	// 注意：db.SetMaxOpenConns(1)，同一时刻只能有一组 rows 未 Close，不能嵌套 Query（会
	// 死锁）。所以这里和 secrets 一样：players 主表先整个扫完 Close，再依次扫完四张子表
	// 各自 Close，最后在内存里按 player_id 合并——都是独立的只读全表扫描，互不依赖。
	rows, err := ps.db.Query(`
		SELECT id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			rank_multiplier_unlocked, ranked_last_decay_day,
			push_mention_enabled, push_turn_enabled, push_seat_enabled, push_bond_enabled,
			wins, losses, draws, punishments, ranked_points, highest_score, lowest_score, title, title_segment_id, title_custom, self_title,
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
			rankUnlock, rankedDecay         sql.NullInt64
			pushM, pushT, pushS, pushB      sql.NullInt64
			bondMaster, bondPet, bondPublic sql.NullInt64
		)
		err := rows.Scan(
			&item.ID, &item.PlayerID, &item.ClaimKey, &item.Name, &item.GenderID, &item.FactionID, &item.AvatarURL,
			&rankUnlock, &rankedDecay,
			&pushM, &pushT, &pushS, &pushB,
			&item.Stats.Wins, &item.Stats.Losses, &item.Stats.Draws, &item.Stats.Punishments, &item.Stats.RankedPoints, &item.Stats.HighestScore, &item.Stats.LowestScore, &item.Stats.Title, &item.Stats.TitleSegmentID, &item.Stats.TitleCustom, &item.Stats.SelfTitle,
			&item.Stats.TotalOnlineMs,
			&bondMaster, &bondPet, &bondPublic,
			&item.CreatedAt, &item.LastSeenAt,
		)
		if err != nil {
			_ = rows.Close()
			return nil, err
		}
		item.RankMultiplierUnlocked = nullIntToBoolPtr(rankUnlock)
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

	nameWarByPlayer, err := ps.loadAllNameWarLocked()
	if err != nil {
		return nil, err
	}
	giveawayByPlayer, err := ps.loadAllGiveawayLocked()
	if err != nil {
		return nil, err
	}
	extremeByPlayer, err := ps.loadAllExtremeModeLocked()
	if err != nil {
		return nil, err
	}
	gameStatsByPlayer, err := ps.loadAllGameStatsLocked()
	if err != nil {
		return nil, err
	}
	secretsByPlayer, err := ps.loadAllSecretsLocked()
	if err != nil {
		return nil, err
	}

	out := make([]playerRow, 0, len(items))
	for _, item := range items {
		// 四张子表都是"稠密"设计（每个持久化玩家固定一行/一组），正常情况下 ok 恒为
		// true；缺行只会发生在库被手工改过之类的异常场景，此时保留 item 里的零值
		// （nil 指针/空字符串），效果等同于旧版 NULL 列"从未设置过"。
		if nw, ok := nameWarByPlayer[item.ID]; ok {
			applyNameWarRow(&item, nw)
		}
		if ga, ok := giveawayByPlayer[item.ID]; ok {
			applyGiveawayRow(&item, ga)
		}
		if ex, ok := extremeByPlayer[item.ID]; ok {
			applyExtremeModeRow(&item, ex)
		}
		if gs, ok := gameStatsByPlayer[item.ID]; ok {
			item.GameStats = gs
		}
		secrets := secretsByPlayer[item.ID]
		item.PlayerSecrets = secrets
		out = append(out, playerRow{item: item, secrets: secrets})
	}
	return out, nil
}

// nameWarRow/giveawayRow/extremeModeRow：player_name_war/player_giveaway/
// player_extreme_mode 三张子表各自的扫描形状，字段与列一一对应。
type nameWarRow struct {
	Enabled, AllowRename, Punished     sql.NullInt64
	ToggledAt, RenameProtectedUntil    sql.NullInt64
	RenameWindowStartedAt, RenameCount sql.NullInt64
	OriginalName, PenaltyName          string
	RenamedBy, RenamedByName           string
}

func (ps *playerStore) loadAllNameWarLocked() (map[string]nameWarRow, error) {
	rows, err := ps.db.Query(`
		SELECT player_id, enabled, allow_rename, toggled_at, original_name, penalty_name,
			punished, rename_protected_until, renamed_by, renamed_by_name,
			rename_window_started_at, rename_count
		FROM player_name_war`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]nameWarRow{}
	for rows.Next() {
		var pid string
		var r nameWarRow
		if err := rows.Scan(&pid, &r.Enabled, &r.AllowRename, &r.ToggledAt, &r.OriginalName, &r.PenaltyName,
			&r.Punished, &r.RenameProtectedUntil, &r.RenamedBy, &r.RenamedByName,
			&r.RenameWindowStartedAt, &r.RenameCount); err != nil {
			return nil, err
		}
		out[pid] = r
	}
	return out, rows.Err()
}

func applyNameWarRow(item *persistedPlayer, r nameWarRow) {
	item.NameWarEnabled = nullIntToBoolPtr(r.Enabled)
	item.NameWarAllowRename = nullIntToBoolPtr(r.AllowRename)
	item.NameWarToggledAt = nullIntToInt64Ptr(r.ToggledAt)
	item.NameWarOriginalName = r.OriginalName
	item.NameWarPenaltyName = r.PenaltyName
	item.NameWarPunished = nullIntToBoolPtr(r.Punished)
	item.NameWarRenameProtectedUntil = nullIntToInt64Ptr(r.RenameProtectedUntil)
	item.NameWarRenamedBy = r.RenamedBy
	item.NameWarRenamedByName = r.RenamedByName
	item.NameWarRenameWindowStartedAt = nullIntToInt64Ptr(r.RenameWindowStartedAt)
	item.NameWarRenameCount = nullIntToIntPtr(r.RenameCount)
}

type giveawayRow struct {
	Enabled                          sql.NullInt64
	Value                            sql.NullFloat64
	Clicks                           sql.NullInt64
	BoardText                        string
	BoardSubmittedAt, BoardExpiresAt sql.NullInt64
	BoardLikes, BoardDislikes        sql.NullInt64
}

func (ps *playerStore) loadAllGiveawayLocked() (map[string]giveawayRow, error) {
	rows, err := ps.db.Query(`
		SELECT player_id, enabled, value, clicks, board_text, board_submitted_at, board_expires_at,
			board_likes, board_dislikes
		FROM player_giveaway`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]giveawayRow{}
	for rows.Next() {
		var pid string
		var r giveawayRow
		if err := rows.Scan(&pid, &r.Enabled, &r.Value, &r.Clicks, &r.BoardText, &r.BoardSubmittedAt, &r.BoardExpiresAt,
			&r.BoardLikes, &r.BoardDislikes); err != nil {
			return nil, err
		}
		out[pid] = r
	}
	return out, rows.Err()
}

func applyGiveawayRow(item *persistedPlayer, r giveawayRow) {
	item.GiveawayEnabled = nullIntToBoolPtr(r.Enabled)
	if r.Value.Valid {
		v := r.Value.Float64
		item.GiveawayValue = &v
	}
	item.GiveawayClicks = nullIntToIntPtr(r.Clicks)
	item.GiveawayBoardText = r.BoardText
	item.GiveawayBoardSubmitted = nullIntToInt64Ptr(r.BoardSubmittedAt)
	item.GiveawayBoardExpires = nullIntToInt64Ptr(r.BoardExpiresAt)
	item.GiveawayBoardLikes = nullIntToIntPtr(r.BoardLikes)
	item.GiveawayBoardDislikes = nullIntToIntPtr(r.BoardDislikes)
}

type extremeModeRow struct {
	Enabled                    sql.NullInt64
	ToggledAt, CooldownUntil   sql.NullInt64
	WinStreak, LastDecayHour   sql.NullInt64
	ForceClosed, ForceClosedAt sql.NullInt64
	RenameProtectedUntil       sql.NullInt64
	RenamedBy, RenamedByName   string
}

func (ps *playerStore) loadAllExtremeModeLocked() (map[string]extremeModeRow, error) {
	rows, err := ps.db.Query(`
		SELECT player_id, enabled, toggled_at, cooldown_until, win_streak, last_decay_hour,
			force_closed, force_closed_at, rename_protected_until, renamed_by, renamed_by_name
		FROM player_extreme_mode`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]extremeModeRow{}
	for rows.Next() {
		var pid string
		var r extremeModeRow
		if err := rows.Scan(&pid, &r.Enabled, &r.ToggledAt, &r.CooldownUntil, &r.WinStreak, &r.LastDecayHour,
			&r.ForceClosed, &r.ForceClosedAt, &r.RenameProtectedUntil, &r.RenamedBy, &r.RenamedByName); err != nil {
			return nil, err
		}
		out[pid] = r
	}
	return out, rows.Err()
}

func applyExtremeModeRow(item *persistedPlayer, r extremeModeRow) {
	item.ExtremeModeEnabled = nullIntToBoolPtr(r.Enabled)
	item.ExtremeModeToggledAt = nullIntToInt64Ptr(r.ToggledAt)
	item.ExtremeModeCooldownUntil = nullIntToInt64Ptr(r.CooldownUntil)
	item.ExtremeWinStreak = nullIntToIntPtr(r.WinStreak)
	item.ExtremeLastDecayHour = nullIntToInt64Ptr(r.LastDecayHour)
	item.ExtremeForceClosed = nullIntToBoolPtr(r.ForceClosed)
	item.ExtremeForceClosedAt = nullIntToInt64Ptr(r.ForceClosedAt)
	item.ExtremeRenameProtectedUntil = nullIntToInt64Ptr(r.RenameProtectedUntil)
	item.ExtremeRenamedBy = r.RenamedBy
	item.ExtremeRenamedByName = r.RenamedByName
}

// loadAllGameStatsLocked 把 player_game_stats 的全部行按 player_id 合并回
// types.GameStats（WLDFor 与 upsertInTx 写入时用的是同一张 game_id 映射表）。
func (ps *playerStore) loadAllGameStatsLocked() (map[string]types.GameStats, error) {
	rows, err := ps.db.Query(`SELECT player_id, game_id, wins, losses, draws FROM player_game_stats`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string]types.GameStats{}
	for rows.Next() {
		var pid, gameID string
		var w, l, d int
		if err := rows.Scan(&pid, &gameID, &w, &l, &d); err != nil {
			return nil, err
		}
		id := types.GameID(gameID)
		// WLDFor 为兼容历史调用会把未知枚举回落到 RPS；数据库行不能沿用这个行为，
		// 否则手工导入/未来版本留下的未知 game_id 会静默覆盖该玩家真实的 RPS 战绩。
		if !knownGameStatsID(id) {
			continue
		}
		gs := out[pid]
		*gs.WLDFor(id) = types.GameWLD{Wins: w, Losses: l, Draws: d}
		out[pid] = gs
	}
	return out, rows.Err()
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
	_, err := tx.Exec(`
		INSERT INTO players (
			id, player_id, claim_key, name, gender_id, faction_id, avatar_url,
			rank_multiplier_unlocked, ranked_last_decay_day,
			push_mention_enabled, push_turn_enabled, push_seat_enabled, push_bond_enabled,
			wins, losses, draws, punishments, ranked_points, highest_score, lowest_score, title, title_segment_id, title_custom, self_title,
			total_online_ms,
			bond_master_enabled, bond_pet_enabled, bond_public_display,
			created_at, last_seen_at
		) VALUES (
			?,?,?,?,?,?,?,
			?,?,
			?,?,?,?,
			?,?,?,?,?,?,?,?,?,?,?,
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
			rank_multiplier_unlocked=excluded.rank_multiplier_unlocked,
			ranked_last_decay_day=excluded.ranked_last_decay_day,
			push_mention_enabled=excluded.push_mention_enabled,
			push_turn_enabled=excluded.push_turn_enabled,
			push_seat_enabled=excluded.push_seat_enabled,
			push_bond_enabled=excluded.push_bond_enabled,
			wins=excluded.wins, losses=excluded.losses, draws=excluded.draws,
			punishments=excluded.punishments, ranked_points=excluded.ranked_points,
			highest_score=excluded.highest_score, lowest_score=excluded.lowest_score,
			title=excluded.title, title_segment_id=excluded.title_segment_id, title_custom=excluded.title_custom, self_title=excluded.self_title,
			total_online_ms=excluded.total_online_ms,
			bond_master_enabled=excluded.bond_master_enabled,
			bond_pet_enabled=excluded.bond_pet_enabled,
			bond_public_display=excluded.bond_public_display,
			created_at=excluded.created_at, last_seen_at=excluded.last_seen_at
	`,
		item.ID, item.PlayerID, item.ClaimKey, item.Name, item.GenderID, item.FactionID, item.AvatarURL,
		boolPtrToSQL(item.RankMultiplierUnlocked), int64PtrToSQL(item.RankedLastDecayDay),
		boolPtrToSQL(item.PushMentionEnabled), boolPtrToSQL(item.PushTurnEnabled), boolPtrToSQL(item.PushSeatEnabled), boolPtrToSQL(item.PushBondEnabled),
		item.Stats.Wins, item.Stats.Losses, item.Stats.Draws, item.Stats.Punishments, item.Stats.RankedPoints, item.Stats.HighestScore, item.Stats.LowestScore, item.Stats.Title, item.Stats.TitleSegmentID, item.Stats.TitleCustom, item.Stats.SelfTitle,
		item.Stats.TotalOnlineMs,
		boolPtrToSQL(item.BondMasterEnabled), boolPtrToSQL(item.BondPetEnabled), boolPtrToSQL(item.BondPublicDisplay),
		item.CreatedAt, item.LastSeenAt,
	)
	if err != nil {
		return err
	}
	if err := ps.upsertNameWarInTx(tx, item); err != nil {
		return err
	}
	if err := ps.upsertGiveawayInTx(tx, item); err != nil {
		return err
	}
	if err := ps.upsertExtremeModeInTx(tx, item); err != nil {
		return err
	}
	if err := ps.upsertGameStatsInTx(tx, item); err != nil {
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

func (ps *playerStore) upsertNameWarInTx(tx *sql.Tx, item persistedPlayer) error {
	_, err := tx.Exec(`
		INSERT INTO player_name_war (
			player_id, enabled, allow_rename, toggled_at, original_name, penalty_name,
			punished, rename_protected_until, renamed_by, renamed_by_name,
			rename_window_started_at, rename_count
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(player_id) DO UPDATE SET
			enabled=excluded.enabled, allow_rename=excluded.allow_rename, toggled_at=excluded.toggled_at,
			original_name=excluded.original_name, penalty_name=excluded.penalty_name, punished=excluded.punished,
			rename_protected_until=excluded.rename_protected_until, renamed_by=excluded.renamed_by,
			renamed_by_name=excluded.renamed_by_name, rename_window_started_at=excluded.rename_window_started_at,
			rename_count=excluded.rename_count
	`,
		item.ID, boolPtrToSQL(item.NameWarEnabled), boolPtrToSQL(item.NameWarAllowRename), int64PtrToSQL(item.NameWarToggledAt),
		item.NameWarOriginalName, item.NameWarPenaltyName, boolPtrToSQL(item.NameWarPunished),
		int64PtrToSQL(item.NameWarRenameProtectedUntil), item.NameWarRenamedBy, item.NameWarRenamedByName,
		int64PtrToSQL(item.NameWarRenameWindowStartedAt), intPtrToSQL(item.NameWarRenameCount),
	)
	return err
}

func (ps *playerStore) upsertGiveawayInTx(tx *sql.Tx, item persistedPlayer) error {
	_, err := tx.Exec(`
		INSERT INTO player_giveaway (
			player_id, enabled, value, clicks, board_text, board_submitted_at, board_expires_at,
			board_likes, board_dislikes
		) VALUES (?,?,?,?,?,?,?,?,?)
		ON CONFLICT(player_id) DO UPDATE SET
			enabled=excluded.enabled, value=excluded.value, clicks=excluded.clicks,
			board_text=excluded.board_text, board_submitted_at=excluded.board_submitted_at,
			board_expires_at=excluded.board_expires_at, board_likes=excluded.board_likes,
			board_dislikes=excluded.board_dislikes
	`,
		item.ID, boolPtrToSQL(item.GiveawayEnabled), floatPtrToSQL(item.GiveawayValue), intPtrToSQL(item.GiveawayClicks),
		item.GiveawayBoardText, int64PtrToSQL(item.GiveawayBoardSubmitted), int64PtrToSQL(item.GiveawayBoardExpires),
		intPtrToSQL(item.GiveawayBoardLikes), intPtrToSQL(item.GiveawayBoardDislikes),
	)
	return err
}

func (ps *playerStore) upsertExtremeModeInTx(tx *sql.Tx, item persistedPlayer) error {
	_, err := tx.Exec(`
		INSERT INTO player_extreme_mode (
			player_id, enabled, toggled_at, cooldown_until, win_streak, last_decay_hour,
			force_closed, force_closed_at, rename_protected_until, renamed_by, renamed_by_name
		) VALUES (?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(player_id) DO UPDATE SET
			enabled=excluded.enabled, toggled_at=excluded.toggled_at, cooldown_until=excluded.cooldown_until,
			win_streak=excluded.win_streak, last_decay_hour=excluded.last_decay_hour,
			force_closed=excluded.force_closed, force_closed_at=excluded.force_closed_at,
			rename_protected_until=excluded.rename_protected_until, renamed_by=excluded.renamed_by,
			renamed_by_name=excluded.renamed_by_name
	`,
		item.ID, boolPtrToSQL(item.ExtremeModeEnabled), int64PtrToSQL(item.ExtremeModeToggledAt), int64PtrToSQL(item.ExtremeModeCooldownUntil),
		intPtrToSQL(item.ExtremeWinStreak), int64PtrToSQL(item.ExtremeLastDecayHour),
		boolPtrToSQL(item.ExtremeForceClosed), int64PtrToSQL(item.ExtremeForceClosedAt), int64PtrToSQL(item.ExtremeRenameProtectedUntil),
		item.ExtremeRenamedBy, item.ExtremeRenamedByName,
	)
	return err
}

// upsertGameStatsInTx 无条件写全 gameIDsForStats 里的每一款游戏（哪怕这局没有任何变化的
// 游戏也重新写一遍 0/0/0），不做"只有变化的游戏才写"的判断——每个玩家固定 7 行，换来的
// 是 loadAll 侧不需要处理"这一行存在与否"的分支，多出的写入量在每分钟一次的落盘节奏下
// 可以忽略。
func (ps *playerStore) upsertGameStatsInTx(tx *sql.Tx, item persistedPlayer) error {
	gs := item.GameStats
	for _, gameID := range gameIDsForStats {
		wld := gs.WLDFor(gameID)
		if _, err := tx.Exec(`
			INSERT INTO player_game_stats (player_id, game_id, wins, losses, draws)
			VALUES (?,?,?,?,?)
			ON CONFLICT(player_id, game_id) DO UPDATE SET
				wins=excluded.wins, losses=excluded.losses, draws=excluded.draws
		`, item.ID, string(gameID), wld.Wins, wld.Losses, wld.Draws); err != nil {
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
