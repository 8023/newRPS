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
	if err != nil || row.ContributorPlayerID == "" {
		client.reply(env.ID, map[string]any{}, "")
		return
	}
	card, err := s.voteCardFor(p.ID, row)
	if err != nil {
		client.reply(env.ID, map[string]any{}, "")
		return
	}
	ok, _ := s.contributionStore.isRoundParticipant(row.RoundID, p.ID)
	if !ok || row.Status != "approved" {
		card.CanVote = false
	}
	if row.ContributorPlayerID == p.ID {
		card.CanVote = false
		card.CannotVoteReason = "不能评价自己贡献的内容"
	}
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

func (s *Server) castContributionVote(player *PlayerState, eventID string, vote int) (types.VoteCard, error) {
	var card types.VoteCard
	if s.eventDB == nil || s.contributionStore == nil {
		return card, errVoteInvalidTarget
	}
	row, err := s.eventDB.getPunishmentEvent(eventID)
	if err != nil {
		return card, errVoteInvalidTarget
	}
	if row.Status != "approved" || (row.RedoID.Valid && row.RedoID.String != "") {
		return card, errVoteNotEligible
	}
	if row.Source == "player" || row.ContributorPlayerID == "" {
		return card, errVoteInvalidTarget
	}
	ok, err := s.contributionStore.isRoundParticipant(row.RoundID, player.ID)
	if err != nil || !ok {
		return card, errVoteNotEligible
	}
	if row.ContributorPlayerID == player.ID {
		return card, errVoteNotEligible
	}
	kind, tid, ver := formalTarget(row)
	if tid == "" {
		return card, errVoteInvalidTarget
	}
	if err := s.contributionStore.castVote(row.RoundID, eventID, player.ID, kind, tid, ver, vote); err != nil {
		if err != errVoteDuplicate {
			s.errorLog("contribution_vote_failed", err.Error())
			return card, fmt.Errorf("投票失败，请稍后重试")
		}
		return card, err
	}
	return s.voteCardFor(player.ID, row)
}

// formalTarget：一个惩罚事件最终该向哪条具体任务计票——恒定落在 FormalTaskID（TaskGroupID，
// 见 metaForFormalTask），不再按整个系列合并计票，这样系列里每一步才能各自独立点赞点踩。
// FormalSeriesID 只作事件溯源/完成率统计用，不参与这里的目标解析。
func formalTarget(row punishmentEventRow) (kind, id string, ver int) {
	return types.ContributionKindTask, row.FormalTaskID, row.FormalTaskVersion
}

func (s *Server) voteCardFor(playerID string, row punishmentEventRow) (types.VoteCard, error) {
	kind, tid, ver := formalTarget(row)
	st, err := s.contributionStore.voteStats(kind, tid, ver)
	if err != nil {
		return types.VoteCard{}, err
	}
	my, _ := s.contributionStore.myVote(row.RoundID, playerID, kind, tid)
	name := row.ContributorNameSnap
	if row.ContributorAnonymous != 0 {
		name = s.cfg.Site.AnonymousContributorLabel
		if name == "" {
			name = types.DefaultAnonymousContributorLabel
		}
	}
	can := row.ContributorPlayerID != playerID && row.Status == "approved"
	card := types.VoteCard{
		EventID:                row.ID,
		RoundID:                row.RoundID,
		TargetKind:             kind,
		TargetID:               tid,
		CanVote:                can && my == 0,
		MyVote:                 my,
		DisplayRatio:           st.DisplayRatio,
		VoteCount:              st.VoteCount,
		HasVotes:               st.VoteCount > 0,
		ContributorDisplayName: name,
		ContributorAnonymous:   row.ContributorAnonymous != 0,
	}
	if my == 0 {
		// “投票前不剧透”必须是服务端协议保证，不能只靠 React 暂时不渲染：否则攻击者
		// 直接调用 contribution:votePreview 就能提前拿到贡献者和现有点赞率，影响判断。
		card.DisplayRatio = nil
		card.VoteCount = 0
		card.HasVotes = false
		card.ContributorDisplayName = ""
		card.ContributorAnonymous = false
	}
	return card, nil
}
