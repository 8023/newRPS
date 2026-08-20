package server

import (
	"strings"

	"github.com/doumiao/newRPS/internal/types"
)

func (s *Server) requirePersistentPlayer(client *Client) *PlayerState {
	p := s.getPlayerByClient(client)
	if p == nil || !p.Persistent {
		return nil
	}
	return p
}

func (s *Server) onContributionList(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	list, err := s.contributionStore.listBySubmitter(p.ID)
	if err != nil {
		client.reply(env.ID, nil, "读取失败")
		return
	}
	client.reply(env.ID, map[string]any{"items": list}, "")
}

func (s *Server) onContributionGet(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	sub, err := s.contributionStore.getOwned(in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	ver, err := s.contributionStore.getVersion(sub.ID, sub.ActiveVersion)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	detail := types.ContributionDetail{Submission: sub, Version: ver}
	if sub.PublishedTargetID != "" {
		st, _ := s.contributionStore.submissionVoteAggregate(sub.ID, sub.PublishedVersion)
		detail.LikeCount = st.LikeCount
		detail.DownCount = st.DownCount
		detail.VoteCount = st.VoteCount
		detail.RealRatio = st.RealRatio
		if sub.Kind == types.ContributionKindSeries {
			detail.Completion = s.seriesCompletionStats(sub.PublishedTargetID, sub.PublishedVersion)
		}
	}
	client.reply(env.ID, detail, "")
}

func (s *Server) onContributionSaveDraft(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	var in struct {
		ID        string `json:"id"`
		Kind      string `json:"kind"`
		Anonymous bool   `json:"anonymous"`
		Content   any    `json:"content"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Kind = strings.TrimSpace(in.Kind)
	if in.ID != "" && len([]rune(in.ID)) > maxContributionIDRunes {
		client.reply(env.ID, nil, "投稿 ID 无效")
		return
	}
	raw, err := marshalDraft(in.Content)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.validateContributionRefs(in.Kind, raw); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	sub, err := s.contributionStore.saveDraft(p.ID, playerShortName(p), in.Kind, in.Anonymous, in.ID, raw)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	client.reply(env.ID, sub, "")
}

func (s *Server) onContributionSubmit(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	sub, err := s.contributionStore.getOwned(in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	ver, err := s.contributionStore.getVersion(sub.ID, sub.ActiveVersion)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if _, err := s.contributionStore.validateOwnedDraft(p.ID, sub.Kind, ver.Content); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.validateContributionRefs(sub.Kind, ver.Content); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if sub.Kind == types.ContributionKindSeries {
		if err := s.validateSeriesMinSteps(ver.Content); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
		if err := s.validateSeriesMaxSteps(ver.Content); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
	}
	if err := s.contributionStore.submit(p.ID, in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	sub, err = s.contributionStore.getOwned(in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, "读取投稿状态失败")
		return
	}
	client.reply(env.ID, sub, "")
}

func (s *Server) onContributionWithdraw(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.contributionStore.withdraw(p.ID, in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}

func (s *Server) onContributionRequestUnpublish(client *Client, env wsEnvelope) {
	p := s.requirePersistentPlayer(client)
	if p == nil {
		client.reply(env.ID, nil, "请先登录持久身份")
		return
	}
	if s.contributionStore == nil {
		client.reply(env.ID, nil, "共建存储不可用")
		return
	}
	var in struct {
		ID string `json:"id"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.contributionStore.requestUnpublish(p.ID, in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	client.reply(env.ID, map[string]any{"ok": true}, "")
}
