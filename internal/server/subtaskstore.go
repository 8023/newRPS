package server

import (
	"database/sql"
	"encoding/json"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

// subTaskStore 是 sub_tasks 表（随机任务 + 系列每一步，二合一）的存取层。核心不变式：
// "id X 当前状态/内容" 恒定等于"sub_tasks 里 id=X 的所有行中 version 最大的那一行"——
// 插入新版本只管 INSERT，从不 UPDATE/DELETE 内容列；纯状态流转（提交/通过/驳回/下架/
// 取消撤回）直接 UPDATE 最新那一行的 status，不产生新版本。没有单独的"投稿信封"表，
// 这张表本身就是内容 + 审核状态的合一记录。
type subTaskStore struct {
	db *sql.DB
}

func newSubTaskStore(db *sql.DB) *subTaskStore {
	return &subTaskStore{db: db}
}

// subTaskRow 是 sub_tasks 一行的完整 Go 映射。
type subTaskRow struct {
	ID                   string
	Version              int
	SeriesID             string
	StepIndex            int
	Active               bool
	Draft                types.StepDraft
	Status               string
	LikeCount            int
	DownCount            int
	ContributorPlayerID  string
	ContributorAnonymous bool
	ReviewedBy           string
	ReviewedAt           int64
	ReviewComment        string
	CreatedAt            int64
	UpdatedAt            int64
}

func scanSubTaskRow(row interface {
	Scan(dest ...any) error
}) (subTaskRow, error) {
	var r subTaskRow
	var variantsJSON, tagJSON string
	var active, anon int
	err := row.Scan(
		&r.ID, &r.Version, &r.SeriesID, &r.StepIndex, &active, &variantsJSON, &tagJSON,
		&r.Draft.BackgroundImage, &r.Draft.BackgroundOpacity, &r.Draft.Order, &r.Status,
		&r.LikeCount, &r.DownCount, &r.ContributorPlayerID, &anon,
		&r.ReviewedBy, &r.ReviewedAt, &r.ReviewComment, &r.CreatedAt, &r.UpdatedAt,
	)
	if err != nil {
		return r, err
	}
	r.Active = active != 0
	r.ContributorAnonymous = anon != 0
	if err := json.Unmarshal([]byte(variantsJSON), &r.Draft.Variants); err != nil {
		r.Draft.Variants = nil
	}
	if err := json.Unmarshal([]byte(tagJSON), &r.Draft.TagIDs); err != nil {
		r.Draft.TagIDs = nil
	}
	r.Draft.InRandomPool = r.Draft.Order != -1
	return r, nil
}

// subTaskSelectCols 全部带 t. 前缀：多数调用点把 sub_tasks 起别名 t 并和一张同样带 id/version
// 列的派生表 JOIN（取每个 id 的 MAX(version)），不加前缀会被 SQLite 判为 ambiguous column。
//
// 不选 contributor_name：投稿人昵称不做快照，统一由调用方拿 ContributorPlayerID 现查
// s.players（改名立即生效，不显示改名前的旧昵称）——该列仍留在建表 DDL 供老库迁移路径
// 显式写入，业务代码只是不再读它，最终由 v48 迁移物理 DROP。
const subTaskSelectCols = `t.id, t.version, t.series_id, t.step_index, t.active, t.variants, t.tag_ids,
	t.background_image, t.background_opacity, t.difficulty_order, t.status,
	t.like_count, t.down_count, t.contributor_player_id, t.contributor_anonymous,
	t.reviewed_by, t.reviewed_at, t.review_comment, t.created_at, t.updated_at`

// latest 返回某个 id 当前最新的一行（不管状态）；不存在返回 errContributionNotFound。
func (ts *subTaskStore) latest(id string) (subTaskRow, error) {
	row := ts.db.QueryRow(`SELECT `+subTaskSelectCols+` FROM sub_tasks t
		WHERE id = ? ORDER BY version DESC LIMIT 1`, id)
	r, err := scanSubTaskRow(row)
	if err == sql.ErrNoRows {
		return r, errContributionNotFound
	}
	return r, err
}

// atVersion 返回某个 id 在指定 version 上的那一行；不存在返回 errContributionNotFound。
// 供 contribution_vote_rpc.go 按惩罚事件记录的 formal_task_id/formal_task_version 精确
// 定位玩家当时实际体验的那一份内容——贡献者是谁、要不要匿名都现查这一行，不依赖
// punishment_events 自己再存一份快照。
func (ts *subTaskStore) atVersion(id string, version int) (subTaskRow, error) {
	row := ts.db.QueryRow(`SELECT `+subTaskSelectCols+` FROM sub_tasks t
		WHERE id = ? AND version = ?`, id, version)
	r, err := scanSubTaskRow(row)
	if err == sql.ErrNoRows {
		return r, errContributionNotFound
	}
	return r, err
}

// latestOwned 同 latest，但要求属于 playerID，否则报无权限。
func (ts *subTaskStore) latestOwned(id, playerID string) (subTaskRow, error) {
	r, err := ts.latest(id)
	if err != nil {
		return r, err
	}
	if r.ContributorPlayerID != playerID {
		return subTaskRow{}, errContributionForbidden
	}
	return r, nil
}

// insertVersion 插入一个新版本行（id 为空则新建 id，version=1）；submissionID 为空时
// 复用当前时间戳生成一个新 id。status 由调用方指定（draft/pending/approved 均可，
// 后者用于管理员编辑后直接发布）。
func (ts *subTaskStore) insertVersion(tx *sql.Tx, id string, seriesID string, stepIndex int, draft types.StepDraft, status, contributorID string, anonymous bool, reviewedBy, reviewComment string) (subTaskRow, error) {
	now := nowMs()
	version := 1
	createdAt := now
	if id == "" {
		id = "st_" + randomID()
	} else {
		if prev, err := ts.latestTx(tx, id); err == nil {
			version = prev.Version + 1
			createdAt = prev.CreatedAt
		} else if err != errContributionNotFound {
			return subTaskRow{}, err
		}
	}
	order := draft.Order
	if !draft.InRandomPool {
		order = -1
	}
	variantsJSON, err := json.Marshal(nonNilVariants(draft.Variants))
	if err != nil {
		return subTaskRow{}, err
	}
	tagJSON, err := json.Marshal(nonNilStrings(draft.TagIDs))
	if err != nil {
		return subTaskRow{}, err
	}
	anon := 0
	if anonymous {
		anon = 1
	}
	reviewedAt := int64(0)
	if reviewedBy != "" {
		reviewedAt = now
	}
	_, err = tx.Exec(`INSERT INTO sub_tasks (
		id, version, series_id, step_index, active, variants, tag_ids, background_image, background_opacity,
		difficulty_order, status, like_count, down_count, contributor_player_id,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, updated_at
	) VALUES (?, ?, ?, ?, 1, ?, ?, ?, ?, ?, ?, 0, 0, ?, ?, ?, ?, ?, ?, ?)`,
		id, version, seriesID, stepIndex, string(variantsJSON), string(tagJSON),
		strings.TrimSpace(draft.BackgroundImage), draft.BackgroundOpacity, order, status,
		contributorID, anon, reviewedBy, reviewedAt, reviewComment, createdAt, now,
	)
	if err != nil {
		return subTaskRow{}, err
	}
	return ts.latestTx(tx, id)
}

// deactivateVersion 为系列缩短时被移除的步骤写入一条 inactive 新版本。保留旧版本内容
// 和投票历史，但所有“当前成员”查询只认该 id 的最新版本，因此不会在后续提交、审批或
// 取消下架时把已经删除的尾部步骤重新带回来。
func (ts *subTaskStore) deactivateVersion(tx *sql.Tx, id string) error {
	prev, err := ts.latestTx(tx, id)
	if err != nil {
		return err
	}
	if !prev.Active {
		return nil
	}
	_, err = tx.Exec(`INSERT INTO sub_tasks (
		id, version, series_id, step_index, active, variants, tag_ids, background_image, background_opacity,
		difficulty_order, status, like_count, down_count, contributor_player_id,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, updated_at
	) SELECT id, version + 1, series_id, step_index, 0, variants, tag_ids, background_image, background_opacity,
		difficulty_order, status, 0, 0, contributor_player_id,
		contributor_anonymous, reviewed_by, reviewed_at, review_comment, created_at, ?
		FROM sub_tasks WHERE id = ? AND version = ?`, nowMs(), id, prev.Version)
	return err
}

func (ts *subTaskStore) latestTx(tx *sql.Tx, id string) (subTaskRow, error) {
	row := tx.QueryRow(`SELECT `+subTaskSelectCols+` FROM sub_tasks t
		WHERE id = ? ORDER BY version DESC LIMIT 1`, id)
	r, err := scanSubTaskRow(row)
	if err == sql.ErrNoRows {
		return r, errContributionNotFound
	}
	return r, err
}

// setStatus 是纯状态流转：只改最新那一行的 status（不产生新版本）。
func (ts *subTaskStore) setStatus(tx *sql.Tx, id, status, reviewedBy, reviewComment string) error {
	now := nowMs()
	reviewedAt := int64(0)
	if reviewedBy != "" {
		reviewedAt = now
	}
	res, err := tx.Exec(`UPDATE sub_tasks SET status = ?, reviewed_by = ?, reviewed_at = ?, review_comment = ?, updated_at = ?
		WHERE id = ? AND version = (SELECT MAX(version) FROM sub_tasks WHERE id = ?)`,
		status, reviewedBy, reviewedAt, reviewComment, now, id, id)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return errContributionNotFound
	}
	return nil
}

// listBySubmitter 返回某玩家名下所有"独立随机任务"投稿的最新版本（不含系列步骤——
// 系列步骤只随系列一起编辑/展示，不在这里单独列出）。
func (ts *subTaskStore) listBySubmitter(playerID string) ([]subTaskRow, error) {
	rows, err := ts.db.Query(`
		SELECT `+subTaskSelectCols+` FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks WHERE contributor_player_id = ? GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.series_id = '' AND t.active = 1
		ORDER BY t.updated_at DESC`, playerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subTaskRow
	for rows.Next() {
		r, err := scanSubTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// listAdmin 返回所有"独立随机任务"（不含系列步骤）的最新版本，可选按 status 过滤，
// 供后台共建审核使用。
func (ts *subTaskStore) listAdmin(status string) ([]subTaskRow, error) {
	q := `SELECT ` + subTaskSelectCols + ` FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks GROUP BY id) m
			ON t.id = m.id AND t.version = m.v`
	args := []any{}
	if status != "" {
		q += ` WHERE t.series_id = '' AND t.active = 1 AND t.status = ?`
		args = append(args, status)
	} else {
		q += ` WHERE t.series_id = '' AND t.active = 1`
	}
	q += ` ORDER BY t.updated_at DESC`
	rows, err := ts.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subTaskRow
	for rows.Next() {
		r, err := scanSubTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// stepsForSeries 返回某系列当前生效（各步骤各自最新版本）的所有步骤行，按 step_index 排序。
func (ts *subTaskStore) stepsForSeries(seriesID string) ([]subTaskRow, error) {
	rows, err := ts.db.Query(`
		SELECT `+subTaskSelectCols+` FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks WHERE series_id = ? GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.series_id = ? AND t.active = 1
		ORDER BY t.step_index ASC`, seriesID, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subTaskRow
	for rows.Next() {
		r, err := scanSubTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// listApprovedForCache 加载随机池/系列步骤运行时缓存：所有"最新版本且状态为 approved"的行，
// 供 rebuildPunishmentCacheIndexes 展开成 types.PunishmentTaskConfig（一份变体一条）。
func (ts *subTaskStore) listApprovedForCache() ([]subTaskRow, error) {
	rows, err := ts.db.Query(`
		SELECT ` + subTaskSelectCols + ` FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks GROUP BY id) m
			ON t.id = m.id AND t.version = m.v
		WHERE t.status = 'approved' AND t.active = 1
		ORDER BY t.series_id, t.step_index, t.created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []subTaskRow
	for rows.Next() {
		r, err := scanSubTaskRow(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// imageIsLive 判断某张共建封面图当前是不是它所属 id 的最新版本仍在引用的图——不看状态
// （草稿/待审/已通过/被驳回/已撤回都不影响），只要还是"最新那一版"引用的就放行；
// 重新上传另一张图覆盖掉之后，被替换下来的旧图不再是任何 id 的最新版本，查询自然查不到。
func (ts *subTaskStore) imageIsLive(path string) (bool, error) {
	if path == "" {
		return false, nil
	}
	var n int
	err := ts.db.QueryRow(`
		SELECT COUNT(*) FROM sub_tasks t
		WHERE t.background_image = ? AND t.active = 1
			AND t.version = (SELECT MAX(version) FROM sub_tasks WHERE id = t.id)`, path).Scan(&n)
	return n > 0, err
}

// voteAggregate 汇总某 id 名下所有历史版本行的赞踩计数之和（不止最新那一行）。
func (ts *subTaskStore) voteAggregate(id string) (likes, downs int, err error) {
	err = ts.db.QueryRow(`SELECT COALESCE(SUM(like_count),0), COALESCE(SUM(down_count),0)
		FROM sub_tasks WHERE id = ?`, id).Scan(&likes, &downs)
	return
}

func nonNilVariants(v []types.TaskVariantDraft) []types.TaskVariantDraft {
	if v == nil {
		return []types.TaskVariantDraft{}
	}
	return v
}

// countAll 统计 sub_tasks 表当前总行数（含所有版本），只供
// migratePunishmentPoolFromJSONIfNeeded 判断"这是不是一次全新安装"用。
func (ts *subTaskStore) countAll() (int, error) {
	var n int
	err := ts.db.QueryRow(`SELECT COUNT(*) FROM sub_tasks`).Scan(&n)
	return n, err
}

// approvedRandomPoolCount 统计当前随机任务候选池的"条数"：最新版本为 approved 且
// difficulty_order<>-1（会真正进入随机候选池）、不属于任何系列步骤的独立任务 ID 数——
// 后台共建审核总览 + 数据分析「随机任务」增长图共用这个口径。仅供系列内部引用
// （order=-1）的步骤不算；"同时发布到随机任务"的系列步骤（series_id 非空但 order<>-1）
// 也算在内。
func (ts *subTaskStore) approvedRandomPoolCount() (int, error) {
	var n int
	err := ts.db.QueryRow(`
		SELECT COUNT(*) FROM sub_tasks t
		INNER JOIN (SELECT id, MAX(version) AS v FROM sub_tasks GROUP BY id) m ON t.id=m.id AND t.version=m.v
		WHERE t.status = 'approved' AND t.active = 1 AND t.difficulty_order <> -1`).Scan(&n)
	return n, err
}
