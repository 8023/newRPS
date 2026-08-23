package server

import (
	"database/sql"
	"encoding/json"
)

// seriesStore 是 series 表（系列任务元信息，不含步骤内容）的存取层，与 subTaskStore
// 同一套"版本化、插入不更新、状态直接 UPDATE 最新行"的规矩（见 contribution_schema.go）。
// 步骤内容全部在 sub_tasks（SeriesID=系列 ID），这里只管系列自己的标题/面向阵营/审核状态。
type seriesStore struct {
	db *sql.DB
}

func newSeriesStore(db *sql.DB) *seriesStore {
	return &seriesStore{db: db}
}

type seriesRow struct {
	ID                   string
	Version              int
	Name                 string
	TargetFactionIDs     []string
	RoomNamePool         *seriesRoomNamePool
	RoomBackgroundImages []string
	Status               string
	ContributorPlayerID  string
	ContributorAnonymous bool
	ReviewedBy           string
	ReviewedAt           int64
	FirstApprovedAt      int64
	ReviewComment        string
	CreatedAt            int64
	UpdatedAt            int64
}

// seriesRoomNamePool 镜像 types.RoomNamePool 的 JSON 形状，避免这个包依赖 types 包的
// 具体字段名（拆开写更清楚这里只是原样透传 JSON，不做任何解释）。
type seriesRoomNamePool struct {
	Subjects   []string `json:"subjects"`
	RoomWords  []string `json:"roomWords"`
	Adjectives []string `json:"adjectives"`
}

// seriesSelectCols 全部带 t. 前缀：多数调用点把 series 起别名 t 并和一张同样带 id/version
// 列的派生表 JOIN（取每个 id 的 MAX(version)），不加前缀会被 SQLite 判为 ambiguous column。
//
// 不选 contributor_name：与 subTaskSelectCols 同一约定——昵称不做快照，调用方拿
// ContributorPlayerID 现查 s.players。
const seriesSelectCols = `t.id, t.version, t.name, t.target_faction_ids, t.room_name_pool, t.room_background_images,
	t.status, t.contributor_player_id, t.contributor_anonymous,
	t.reviewed_by, t.reviewed_at, t.first_approved_at, t.review_comment, t.created_at, t.updated_at`

func scanSeriesRow(row interface{ Scan(dest ...any) error }) (seriesRow, error) {
	var r seriesRow
	var targetJSON, poolJSON, bgJSON string
	var anon int
	err := row.Scan(
		&r.ID, &r.Version, &r.Name, &targetJSON, &poolJSON, &bgJSON,
		&r.Status, &r.ContributorPlayerID, &anon,
		&r.ReviewedBy, &r.ReviewedAt, &r.FirstApprovedAt, &r.ReviewComment, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return r, err
	}
	r.ContributorAnonymous = anon != 0
	_ = json.Unmarshal([]byte(targetJSON), &r.TargetFactionIDs)
	_ = json.Unmarshal([]byte(bgJSON), &r.RoomBackgroundImages)
	var pool seriesRoomNamePool
	if json.Unmarshal([]byte(poolJSON), &pool) == nil {
		r.RoomNamePool = &pool
	}
	return r, nil
}

func (ss *seriesStore) latest(id string) (seriesRow, error) {
	row := ss.db.QueryRow(`SELECT `+seriesSelectCols+` FROM series t WHERE id = ? ORDER BY version DESC LIMIT 1`, id)
	r, err := scanSeriesRow(row)
	if err == sql.ErrNoRows {
		return r, errContributionNotFound
	}
	return r, err
}

func (ss *seriesStore) latestTx(tx *sql.Tx, id string) (seriesRow, error) {
	row := tx.QueryRow(`SELECT `+seriesSelectCols+` FROM series t WHERE id = ? ORDER BY version DESC LIMIT 1`, id)
	r, err := scanSeriesRow(row)
	if err == sql.ErrNoRows {
		return r, errContributionNotFound
	}
	return r, err
}

func (ss *seriesStore) latestOwned(id, playerID string) (seriesRow, error) {
	r, err := ss.latest(id)
	if err != nil {
		return r, err
	}
	if r.ContributorPlayerID != playerID {
		return seriesRow{}, errContributionForbidden
	}
	return r, nil
}

// insertVersion 插入一个新版本行；id 为空则新建。roomNamePool/roomBackgroundImages
// 只在首次创建（id 为空）时由调用方按系列名生成，后续版本延续沿用上一版的值
// （这两个字段不是玩家可编辑的表单项，见旧版 decodeSeriesPublish 的同等行为）。
func (ss *seriesStore) insertVersion(tx *sql.Tx, id, name string, targetFactionIDs []string, status, contributorID string, anonymous bool, reviewedBy, reviewComment string) (seriesRow, error) {
	now := nowMs()
	version := 1
	createdAt := now
	firstApprovedAt := int64(0)
	pool := &seriesRoomNamePool{Subjects: []string{name}, RoomWords: []string{"小屋"}, Adjectives: []string{"共建"}}
	bgImages := []string{}
	if id == "" {
		id = "sr_" + randomID()
	} else if prev, err := ss.latestTx(tx, id); err == nil {
		version = prev.Version + 1
		createdAt = prev.CreatedAt
		firstApprovedAt = prev.FirstApprovedAt
		if prev.RoomNamePool != nil {
			pool = prev.RoomNamePool
		}
		bgImages = prev.RoomBackgroundImages
	} else if err != errContributionNotFound {
		return seriesRow{}, err
	}
	targetJSON, err := json.Marshal(nonNilStrings(targetFactionIDs))
	if err != nil {
		return seriesRow{}, err
	}
	poolJSON, err := json.Marshal(pool)
	if err != nil {
		return seriesRow{}, err
	}
	bgJSON, err := json.Marshal(nonNilStrings(bgImages))
	if err != nil {
		return seriesRow{}, err
	}
	anon := 0
	if anonymous {
		anon = 1
	}
	reviewedAt := int64(0)
	if reviewedBy != "" {
		reviewedAt = now
	}
	if status == "approved" && firstApprovedAt == 0 {
		firstApprovedAt = now
	}
	_, err = tx.Exec(`INSERT INTO series (
		id, version, name, target_faction_ids, room_name_pool, room_background_images, status,
		contributor_player_id, contributor_anonymous, reviewed_by, reviewed_at,
		first_approved_at, review_comment, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, version, name, string(targetJSON), string(poolJSON), string(bgJSON), status,
		contributorID, anon, reviewedBy, reviewedAt, firstApprovedAt, reviewComment, createdAt, now,
	)
	if err != nil {
		return seriesRow{}, err
	}
	return ss.latestTx(tx, id)
}

func (ss *seriesStore) setStatus(tx *sql.Tx, id, status, reviewedBy, reviewComment string) error {
	now := nowMs()
	reviewedAt := int64(0)
	if reviewedBy != "" {
		reviewedAt = now
	}
	res, err := tx.Exec(`UPDATE series SET status = ?, reviewed_by = ?, reviewed_at = ?,
		first_approved_at = CASE WHEN ? = 'approved' AND first_approved_at = 0 THEN ? ELSE first_approved_at END,
		review_comment = ?, updated_at = ?
		WHERE id = ? AND version = (SELECT MAX(version) FROM series WHERE id = ?)`,
		status, reviewedBy, reviewedAt, status, now, reviewComment, now, id, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errContributionNotFound
	}
	return nil
}

func (ss *seriesStore) listBySubmitter(playerID string) ([]seriesRow, error) {
	rows, err := ss.db.Query(`
		SELECT `+seriesSelectCols+` FROM series t
		INNER JOIN (SELECT id, MAX(version) AS v FROM series WHERE contributor_player_id = ? GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		ORDER BY t.updated_at DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []seriesRow
	for rows.Next() {
		r, err := scanSeriesRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (ss *seriesStore) listAdmin(status string) ([]seriesRow, error) {
	q := `SELECT ` + seriesSelectCols + ` FROM series t
		INNER JOIN (SELECT id, MAX(version) AS v FROM series GROUP BY id) m
			ON t.id = m.id AND t.version = m.v`
	args := []any{}
	if status != "" {
		q += ` WHERE t.status = ?`
		args = append(args, status)
	}
	q += ` ORDER BY t.updated_at DESC`
	rows, err := ss.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []seriesRow
	for rows.Next() {
		r, err := scanSeriesRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// approvedSeriesCount 统计当前已发布（最新版本状态为 approved）的系列"组数"。
func (ss *seriesStore) approvedSeriesCount() (int, error) {
	var n int
	err := ss.db.QueryRow(`
		SELECT COUNT(*) FROM series t
		INNER JOIN (SELECT id, MAX(version) AS v FROM series GROUP BY id) m ON t.id=m.id AND t.version=m.v
		WHERE t.status = 'approved'`).Scan(&n)
	return n, err
}

// listApprovedForCache 加载建房目录用的系列缓存：所有"最新版本且状态为 approved"的系列。
func (ss *seriesStore) listApprovedForCache() ([]seriesRow, error) {
	rows, err := ss.db.Query(`
		SELECT ` + seriesSelectCols + ` FROM series t
		INNER JOIN (SELECT id, MAX(version) AS v FROM series GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.status = 'approved'
		ORDER BY t.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []seriesRow
	for rows.Next() {
		r, err := scanSeriesRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
