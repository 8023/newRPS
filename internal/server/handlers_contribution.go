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
	for i := range list {
		list[i].SubmitterName = s.playerName(list[i].SubmitterID)
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
		ID   string `json:"id"`
		Kind string `json:"kind"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Kind = strings.TrimSpace(in.Kind)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if !validContributionKind(in.Kind) {
		client.reply(env.ID, nil, "投稿类型无效")
		return
	}
	item, err := s.contributionStore.getOwned(in.Kind, in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if in.Kind == types.ContributionKindSeries {
		item.Completion = s.seriesCompletionStats(in.ID)
	}
	item.SubmitterName = s.playerName(item.SubmitterID)
	client.reply(env.ID, item, "")
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
	if !validContributionKind(in.Kind) {
		client.reply(env.ID, nil, "投稿类型无效")
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
	item, err := s.contributionStore.saveDraft(p.ID, in.Kind, in.Anonymous, in.ID, raw)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	item.SubmitterName = s.playerName(item.SubmitterID)
	client.reply(env.ID, item, "")
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
		ID   string `json:"id"`
		Kind string `json:"kind"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Kind = strings.TrimSpace(in.Kind)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if !validContributionKind(in.Kind) {
		client.reply(env.ID, nil, "投稿类型无效")
		return
	}
	item, err := s.contributionStore.getOwned(in.Kind, in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	raw, err := marshalDraft(item.Content)
	if err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if err := s.validateContributionRefs(in.Kind, raw); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if in.Kind == types.ContributionKindTask {
		if err := s.contributionStore.validateOwnedDraft(in.Kind, raw); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
	} else {
		if err := s.contributionStore.validateOwnedDraft(in.Kind, raw); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
		if err := s.validateSeriesMinSteps(raw); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
		if err := s.validateSeriesMaxSteps(raw); err != nil {
			client.reply(env.ID, nil, err.Error())
			return
		}
	}
	if err := s.contributionStore.submit(in.Kind, p.ID, in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	item, err = s.contributionStore.getOwned(in.Kind, in.ID, p.ID)
	if err != nil {
		client.reply(env.ID, nil, "读取投稿状态失败")
		return
	}
	item.SubmitterName = s.playerName(item.SubmitterID)
	client.reply(env.ID, item, "")
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
		ID   string `json:"id"`
		Kind string `json:"kind"`
	}
	if err := decodeD(env, &in); err != nil {
		client.reply(env.ID, nil, "参数格式错误")
		return
	}
	in.ID = strings.TrimSpace(in.ID)
	in.Kind = strings.TrimSpace(in.Kind)
	if err := validateAdminContributionID(in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	if !validContributionKind(in.Kind) {
		client.reply(env.ID, nil, "投稿类型无效")
		return
	}
	if err := s.contributionStore.withdraw(in.Kind, p.ID, in.ID); err != nil {
		client.reply(env.ID, nil, err.Error())
		return
	}
	s.reloadPunishmentCaches()
	s.emitConfigUpdate()
	s.broadcastLobby()
	client.reply(env.ID, map[string]any{"ok": true}, "")
}
