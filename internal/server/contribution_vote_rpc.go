package server

import (
	"fmt"
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) onContributionVotePreview(client *Client, env wsEnvelope) {
	p := s.getPlayerByClient(client)
	if p == nil || s.eventDB == nil || s.contributionStore == nil {
		client.reply(env.ID, nil, "")
		return
	}
	var in struct {
		EventID string `json:"eventId"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.EventID = strings.TrimSpace(in.EventID)
	if in.EventID == "" || len([]rune(in.EventID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "惩罚事件 ID 无效")
		return
	}
	row, err := s.eventDB.getPunishmentEvent(in.EventID)
	if err != nil || row.FormalTaskID == "" {
		client.reply(env.ID, map[string]any{}, "")
		return
	}
	card := s.voteCardFor(p.ID, row)
	client.reply(env.ID, card, "")
}

func (s *Server) onContributionVote(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	var in struct {
		EventID string `json:"eventId"`
		Vote    int    `json:"vote"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.EventID = strings.TrimSpace(in.EventID)
	if in.EventID == "" || len([]rune(in.EventID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "惩罚事件 ID 无效")
		return
	}
	card, err := s.castContributionVote(p, in.EventID, in.Vote)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	client.reply(env.ID, card, "")
}

// castContributionVote 给一次惩罚事件对应的正式任务投一票赞/踩。防重复投票不落任何
// 独立的投票表/内存态：直接原子 UPDATE punishment_events 的 performer_vote/approver_vote
// 列（WHERE 当前还是 0），数据库本身就是"这一侧投过没有"的权威记录，见
// eventstore.go 的 castPunishmentEventVoteAndAggregate。
func (s *Server) castContributionVote(player *PlayerState, eventID string, vote int) (types.VoteCard, error) {
	var card types.VoteCard
	if s.eventDB == nil || s.contributionStore == nil {
		return card, errVoteInvalidTarget
	}
	if vote != types.ContributionVoteUp && vote != types.ContributionVoteDown {
		return card, errVoteInvalidTarget
	}
	row, err := s.eventDB.getPunishmentEvent(eventID)
	if err != nil {
		return card, errVoteInvalidTarget
	}
	if row.RedoID.Valid && row.RedoID.String != "" {
		return card, errVoteNotEligible
	}
	if row.FormalTaskID == "" || row.FormalTaskVersion <= 0 {
		return card, errVoteInvalidTarget
	}
	// 贡献者归属不落在事件行自己身上，按玩家当时实际体验的那个具体版本现查
	// sub_tasks——见 subTaskStore.atVersion 顶部注释。
	task, err := s.contributionStore.tasks.atVersion(row.FormalTaskID, row.FormalTaskVersion)
	if err != nil || task.ContributorPlayerID == "" {
		return card, errVoteInvalidTarget
	}
	if task.ContributorPlayerID == player.ID {
		return card, errVoteNotEligible
	}
	asPerformer := row.PerformerID == player.ID
	asApprover := row.ApproverID == player.ID
	if !asPerformer && !asApprover {
		return card, errVoteNotEligible
	}
	ok, err := s.eventDB.castPunishmentEventVoteAndAggregate(eventID, asPerformer, vote, row.FormalTaskID, row.FormalTaskVersion)
	if err != nil {
		s.errorLog("contribution_vote_failed", err.Error())
		return card, fmt.Errorf("投票失败，请稍后重试")
	}
	if !ok {
		return card, fmt.Errorf("已经评价过该内容")
	}
	row, err = s.eventDB.getPunishmentEvent(eventID)
	if err != nil {
		return card, nil
	}
	return s.voteCardFor(player.ID, row), nil
}

// voteCardFor 组装某玩家看到的评价卡片：能不能投、我投过没、当前点赞率。
// "投票前不剧透"是服务端协议保证，不能只靠前端暂时不渲染——MyVote==0 时不下发
// 点赞率/票数/贡献者，避免攻击者直接调用 contribution:votePreview 提前拿到这些信息
// 影响判断。
//
// 能不能投只看 structurallyVotable（来源是不是玩家发布、有没有正式任务归属、有没有被
// 驳回重做），不要求证明已经审核通过——评价的是任务文案本身好不好，执行者/审批者从任务
// 一发布、双方都看到文案那一刻起就已经在"执行"或"审批"这条任务了，没必要等对方审完才能
// 打分；而且证明审核通过往往和本局结算/进入下一局同时发生（finishPunishmentIfComplete
// 与批准在同一次处理里发生），若硬性要求已批准，玩家几乎抓不到 phase==="punishment" 与刚
// 批准同时成立的那一瞬间，评价入口等于形同虚设。
func (s *Server) voteCardFor(playerID string, row punishmentEventRow) types.VoteCard {
	myVote := 0
	canVote := false
	reason := ""
	// 贡献者/匿名与否都现查玩家当时实际体验的那个具体版本，不依赖事件行自己的快照。
	var task subTaskRow
	haveTask := false
	if row.FormalTaskID != "" && row.FormalTaskVersion > 0 && s.contributionStore != nil {
		if t, err := s.contributionStore.tasks.atVersion(row.FormalTaskID, row.FormalTaskVersion); err == nil && t.ContributorPlayerID != "" {
			task, haveTask = t, true
		}
	}
	structurallyVotable := haveTask && !(row.RedoID.Valid && row.RedoID.String != "")
	if haveTask && task.ContributorPlayerID == playerID {
		reason = "不能评价自己贡献的内容"
	} else if row.PerformerID == playerID {
		myVote = row.PerformerVote
		canVote = structurallyVotable && myVote == 0
	} else if row.ApproverID == playerID {
		myVote = row.ApproverVote
		canVote = structurallyVotable && myVote == 0
	}
	if reason == "" && !structurallyVotable {
		reason = "该任务不能评价"
	}
	name := ""
	anon := false
	if haveTask {
		anon = task.ContributorAnonymous
		if anon {
			name = s.cfg.Site.AnonymousContributorLabel
			if name == "" {
				name = types.DefaultAnonymousContributorLabel
			}
		} else {
			name = s.playerName(task.ContributorPlayerID)
		}
	}
	card := types.VoteCard{
		EventID: row.ID, TargetID: row.FormalTaskID, CanVote: canVote,
		CannotVoteReason: reason, MyVote: myVote,
		ContributorDisplayName: name, ContributorAnonymous: anon,
	}
	if myVote == 0 {
		card.ContributorDisplayName = ""
		card.ContributorAnonymous = false
		return card
	}
	if !haveTask {
		return card
	}
	likes, downs, err := s.contributionStore.tasks.voteAggregate(row.FormalTaskID)
	if err == nil {
		total := likes + downs
		card.VoteCount = total
		card.HasVotes = total > 0
		if total > 0 {
			ratio := int(float64(likes)/float64(total)*100 + 0.5)
			card.DisplayRatio = &ratio
		}
	}
	return card
}
