package types

const (
	ContributionKindGender = "gender"
	ContributionKindTask   = "task"
	ContributionKindSeries = "series"

	ContributionDraft            = "draft"
	ContributionPending          = "pending"
	ContributionApproved         = "approved"
	ContributionRejected         = "rejected"
	ContributionWithdrawn        = "withdrawn"
	ContributionRevisionDraft    = "revision_draft"
	ContributionRevisionPending  = "revision_pending"
	ContributionRevisionRejected = "revision_rejected"
	ContributionUnpublishPending = "unpublish_pending"

	ContributionVoteUp   = 1
	ContributionVoteDown = -1

	RoundRoleWinner = "winner"
	RoundRoleLoser  = "loser"

	DefaultAnonymousContributorLabel = "匿名贡献者"
	DefaultContributionBGOpacity     = 0.22
)

type GenderDraft struct {
	Label     string `json:"label"`
	FactionID string `json:"factionId"`
}

// TaskVariantDraft 是 step 里的一份文案变体。
type TaskVariantDraft struct {
	Text       string   `json:"text"`
	FactionIDs []string `json:"factionIds"`
}

// StepDraft 是随机任务投稿 / 系列某一步共用的编辑单元。不设人肉可读的"名称"字段——
// 玩家和管理员都不会看到它：游戏内展示的分类文字来自惩罚标签名，系列本身的展示名是
// SeriesDraft.Name，发布后每份变体各自的 PunishmentTaskConfig.ID 已经是唯一标识，
// 不需要再额外维护一个标题。
type StepDraft struct {
	Variants          []TaskVariantDraft `json:"variants"`
	InRandomPool      bool               `json:"inRandomPool"` // false ⇒ 发布时 Order 落库为 -1
	Order             int                `json:"order"`        // InRandomPool 为 true 时必填 1-99
	TagIDs            []string           `json:"tagIds"`
	BackgroundImage   string             `json:"backgroundImage"`
	BackgroundOpacity float64            `json:"backgroundOpacity"`
}

type SeriesDraft struct {
	Name             string      `json:"name"`
	TargetFactionIDs []string    `json:"targetFactionIds"`
	Steps            []StepDraft `json:"steps"`
}

// TaskDraft 是旧版单条文案投稿形状，仅供兼容解码。旧 JSON 里可能仍带 name 字段，
// 解码时按未知字段丢弃即可。
type TaskDraft struct {
	Text              string   `json:"text"`
	FactionIDs        []string `json:"factionIds"`
	TagIDs            []string `json:"tagIds"`
	Order             int      `json:"order"`
	BackgroundImage   string   `json:"backgroundImage"`
	BackgroundOpacity float64  `json:"backgroundOpacity"`
}

// LegacySeriesDraft 是旧版「steps.taskRefs + tasks[]」形状，仅供兼容解码。
type LegacySeriesDraft struct {
	Name  string             `json:"name"`
	Steps []SeriesDraftStep  `json:"steps"`
	Tasks []TaskDraftWithRef `json:"tasks"`
}

type SeriesDraftStep struct {
	TaskRefs []string `json:"taskRefs"`
}

type TaskDraftWithRef struct {
	Ref  string    `json:"ref"`
	Task TaskDraft `json:"task"`
}

type ContributionSubmission struct {
	ID                   string `json:"id"`
	Kind                 string `json:"kind"`
	SubmitterPlayerID    string `json:"submitterPlayerId"`
	SubmitterNameSnap    string `json:"submitterNameSnapshot"`
	Anonymous            bool   `json:"anonymous"`
	Status               string `json:"status"`
	PublishedTargetID    string `json:"publishedTargetId"`
	PublishedVersion     int    `json:"publishedVersion"`
	ActiveVersion        int    `json:"activeVersion"`
	CreatedAt            int64  `json:"createdAt"`
	UpdatedAt            int64  `json:"updatedAt"`
	SubmittedAt          int64  `json:"submittedAt"`
	ReviewedAt           int64  `json:"reviewedAt"`
	ReviewedBy           string `json:"reviewedBy"`
	ReviewComment        string `json:"reviewComment"`
	UnpublishRequestedAt int64  `json:"unpublishRequestedAt,omitempty"`
	// Title 是列表用的展示标题（从当前版本内容解析，不落库）。
	Title string `json:"title,omitempty"`
}

type ContributionVersion struct {
	ID              int64  `json:"id"`
	SubmissionID    string `json:"submissionId"`
	Version         int    `json:"version"`
	Content         string `json:"content"`
	CreatedBy       string `json:"createdBy"`
	CreatedAt       int64  `json:"createdAt"`
	ReviewedContent string `json:"reviewedContent"`
	ReviewedAt      int64  `json:"reviewedAt"`
	ReviewedBy      string `json:"reviewedBy"`
	ReviewResult    string `json:"reviewResult"`
	ReviewComment   string `json:"reviewComment"`
}

type ContributionDetail struct {
	Submission ContributionSubmission `json:"submission"`
	Version    ContributionVersion    `json:"version"`
	LikeCount  int                    `json:"likeCount"`
	DownCount  int                    `json:"downCount"`
	VoteCount  int                    `json:"voteCount"`
	RealRatio  *int                   `json:"realRatio"`
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

type VoteStats struct {
	TargetKind   string `json:"targetKind"`
	TargetID     string `json:"targetId"`
	TargetVer    int    `json:"targetVersion"`
	LikeCount    int    `json:"likeCount"`
	DownCount    int    `json:"downCount"`
	VoteCount    int    `json:"voteCount"`
	RealRatio    *int   `json:"realRatio"`
	DisplayRatio *int   `json:"displayRatio"`
	OverrideOn   bool   `json:"overrideOn"`
	OverrideBy   string `json:"overrideBy,omitempty"`
	OverrideNote string `json:"overrideNote,omitempty"`
	OverrideAt   int64  `json:"overrideAt,omitempty"`
}

type VoteCard struct {
	EventID                string `json:"eventId"`
	RoundID                string `json:"roundId"`
	TargetKind             string `json:"targetKind"`
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

type RoundParticipant struct {
	RoundID  string `json:"roundId"`
	RoomID   string `json:"roomId"`
	PlayerID string `json:"playerId"`
	Role     string `json:"role"`
}
