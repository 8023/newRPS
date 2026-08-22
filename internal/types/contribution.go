package types

const (
	ContributionKindTask   = "task"
	ContributionKindSeries = "series"

	// 状态直接落在 sub_tasks/series 每一行上，不再有单独的"投稿信封"表。全量版本化之后
	// 不需要 revision_* 这类状态——"这是不是一次修订"由"这个 ID 是否已有更早的版本"反映，
	// 不需要单独枚举值区分初审/复审。
	ContributionDraft     = "draft"
	ContributionPending   = "pending"
	ContributionApproved  = "approved"
	ContributionRejected  = "rejected"
	ContributionWithdrawn = "withdrawn"

	ContributionVoteUp   = 1
	ContributionVoteDown = -1

	DefaultAnonymousContributorLabel = "匿名贡献者"
	DefaultContributionBGOpacity     = 0.22
)

// TaskVariantDraft 是一份任务/系列步骤里的一份文案变体。
type TaskVariantDraft struct {
	Text       string   `json:"text"`
	FactionIDs []string `json:"factionIds"`
}

// StepDraft 是随机任务投稿 / 系列某一步共用的编辑单元，同时也是 sub_tasks 一行的
// "内容"部分（不含 ID/Version/Status/审核信息等行元数据，那些由 ContributionItem 承载）。
type StepDraft struct {
	Variants          []TaskVariantDraft `json:"variants"`
	InRandomPool      bool               `json:"inRandomPool"` // false ⇒ 发布时 Order 落库为 -1
	Order             int                `json:"order"`        // InRandomPool 为 true 时必填 1-99
	TagIDs            []string           `json:"tagIds"`
	BackgroundImage   string             `json:"backgroundImage"`
	BackgroundOpacity float64            `json:"backgroundOpacity"`
}

// SeriesDraft 是系列任务的编辑单元：Steps 里每一步都是一个完整的 StepDraft，
// 保存/提交时前后端始终把整个系列（含所有步骤）当一个整体传输，落库时才拆成
// series 表一行 + 每步一行 sub_tasks（见 internal/server/seriesstore.go）。
type SeriesDraft struct {
	Name             string      `json:"name"`
	TargetFactionIDs []string    `json:"targetFactionIds"`
	Steps            []StepDraft `json:"steps"`
}

// ContributionItem 是玩家"参与共建"列表/详情、以及后台共建审核用的统一视图，
// 覆盖 task 和 series 两种 kind——取代旧版 ContributionSubmission+ContributionVersion
// 两张表拼出来的结构：sub_tasks/series 每一行本身就同时是"投稿信封"和"内容"，不用再
// 分两张表 JOIN。Content 对 kind=task 是 StepDraft，对 kind=series 是 SeriesDraft
// （其 Steps 由后端从 sub_tasks 按 SeriesID 查出来拼装，不单独落成一份 JSON）。
type ContributionItem struct {
	ID            string `json:"id"`
	Kind          string `json:"kind"`
	Version       int    `json:"version"`
	Status        string `json:"status"`
	Anonymous     bool   `json:"anonymous"`
	SubmitterID   string `json:"submitterId"`
	SubmitterName string `json:"submitterName"`
	ReviewedBy    string `json:"reviewedBy,omitempty"`
	ReviewedAt    int64  `json:"reviewedAt,omitempty"`
	ReviewComment string `json:"reviewComment,omitempty"`
	CreatedAt     int64  `json:"createdAt"`
	UpdatedAt     int64  `json:"updatedAt"`
	LikeCount     int    `json:"likeCount,omitempty"`
	DownCount     int    `json:"downCount,omitempty"`
	// Content 只在详情（Get）响应里填充；列表（List）响应为节省带宽不带内容，
	// 只用 Title 做展示。
	Content any `json:"content,omitempty"`
	// Title 是列表用的展示标题（从内容解析，不落库）：任务用第一份文案的前若干字，
	// 系列用系列名称。
	Title      string                 `json:"title,omitempty"`
	Completion *SeriesCompletionStats `json:"completion,omitempty"`
}

// SeriesCompletionStats：系列任务"完成率"——房间销毁时，把房间里每个至少走完 1 步的玩家
// 各自的"已走步数/系列总步数"百分比记一笔样本，Rate 是这些样本的算术平均（不是"是否走到
// 最后一步"的二元判定——20 步的系列大多数人走 15 步就退出的话，二元判定会把完成率算得
// 极低，按进度百分比取均值更贴近真实体验，见 punishment.go 的
// recordSeriesRunProgressOnClose）。Participants 是样本数，Rate 为空表示 Participants 为 0
// （这个版本的系列还没有人真正走过），避免除零。
type SeriesCompletionStats struct {
	Participants int  `json:"participants"`
	Rate         *int `json:"rate"`
}

// VoteCard 是某次惩罚事件对应的评价卡片：能不能评价、我的评价、当前点赞率。
// 投票前（MyVote==0）服务端不下发 DisplayRatio/VoteCount/ContributorDisplayName，
// 避免"剧透"影响玩家评价前的独立判断。
type VoteCard struct {
	EventID                string `json:"eventId"`
	TargetID               string `json:"targetId"`
	CanVote                bool   `json:"canVote"`
	CannotVoteReason       string `json:"cannotVoteReason,omitempty"`
	MyVote                 int    `json:"myVote"`
	DisplayRatio           *int   `json:"displayRatio"`
	VoteCount              int    `json:"voteCount"`
	HasVotes               bool   `json:"hasVotes"`
	ContributorDisplayName string `json:"contributorDisplayName"`
	ContributorAnonymous   bool   `json:"contributorAnonymous"`
}
