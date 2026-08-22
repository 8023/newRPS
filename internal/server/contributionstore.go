package server

import (
	"database/sql"
	"errors"
)

var (
	errContributionNotFound  = errors.New("投稿不存在")
	errContributionForbidden = errors.New("无权操作该投稿")
	errContributionBadStatus = errors.New("当前状态不允许该操作")
	errVoteNotEligible       = errors.New("没有评价资格")
	errVoteInvalidTarget     = errors.New("该任务不能评价")
)

const (
	maxDraftsPerPlayer             = 30
	maxPendingPerPlayer            = 10
	maxSubmissionsPerPlayer        = 500
	maxContributionIDRunes         = 128
	maxContributionVersionsPerItem = 1000
	maxReviewCommentRunes          = 1000
	maxSeriesNameRunes             = 80
	maxTaskTextRunes               = 4000
	maxContributionRefs            = 256
	// maxSeriesSteps 是系列步数纯防御性的技术上限（防畸形/超大 payload），不是玩法层面
	// 的实际约束——真正对外生效的上限完全由后台「最高步数」配置决定（见 contribution_codec.go
	// 的 effectiveMaxSeriesSteps），管理员填多大都在这条线以内生效，不会被静默收紧。
	maxSeriesSteps      = 1000
	maxStepCandidates   = 8
	maxContributionJSON = 64 * 1024
)

// contributionStore 是玩家"参与共建"+后台共建审核共用的业务层，包了 subTaskStore/
// seriesStore 两张版本化表，按 kind（task/series）分发。
type contributionStore struct {
	db     *sql.DB
	tasks  *subTaskStore
	series *seriesStore
}

func newContributionStore(db *sql.DB) *contributionStore {
	return &contributionStore{db: db, tasks: newSubTaskStore(db), series: newSeriesStore(db)}
}

func boolInt(v bool) int {
	if v {
		return 1
	}
	return 0
}
