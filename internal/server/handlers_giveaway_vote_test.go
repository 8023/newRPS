package server

import (
	"math"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

// setupGiveawayVoteServer 造一个 actor + target 玩家，target 已上白给自救板，
// actor 通过 sock-actor 连接。调用方按需再改字段（比如调小配额触发上限分支）。
func setupGiveawayVoteServer(t *testing.T) (*Server, *Client) {
	t.Helper()
	s := newTestServer(t)
	s.cfg.Giveaway.LikeVoteLimitPerHour = 3
	s.cfg.Giveaway.LikeVoteValue = 1
	s.cfg.Giveaway.DislikeVoteLimitPerHour = 10
	s.cfg.Giveaway.DislikeVoteValue = 0.1
	s.cfg.Giveaway.PetLikeVoteLimitPerHour = 5
	s.cfg.Giveaway.PetLikeVoteValue = 2
	s.cfg.Giveaway.PetDislikeVoteLimitPerHour = 15
	s.cfg.Giveaway.PetDislikeVoteValue = 0.2
	s.cfg.Giveaway.MasterLikeVoteLimitPerHour = 7
	s.cfg.Giveaway.MasterLikeVoteValue = 3
	s.cfg.Giveaway.MasterDislikeVoteLimitPerHour = 20
	s.cfg.Giveaway.MasterDislikeVoteValue = 0.3

	now := nowMs()
	target := &PlayerState{PublicPlayer: types.PublicPlayer{
		ID:                     "target1",
		Name:                   "自救玩家",
		GiveawayEnabled:        boolPtr(true),
		GiveawayValue:          floatPtr(50),
		GiveawayBoardText:      "求放过",
		GiveawayBoardExpiresAt: int64Ptr(now + 3_600_000),
	}}
	target.GiveawayBoardSubmittedAt = int64Ptr(now)
	actor := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "actor1", Name: "投票玩家"}, SocketID: "sock-actor"}
	s.players["target1"] = target
	s.players["actor1"] = actor

	client := &Client{id: "sock-actor", playerID: "actor1", sendCh: make(chan []byte, 8)}
	return s, client
}

func voteEnv(id int64, targetID, vote string) wsEnvelope {
	return wsEnvelope{E: "giveaway:vote", ID: id, D: map[string]any{"targetId": targetID, "vote": vote}}
}

// TestOnGiveawayVoteAllowsRepeatedVotesUpToHourlyLimit 是本次修复的核心用例：管理员配的是
// "每小时最多 N 个赞/踩"，同一个 actor 对同一个 target 就应该能在这个额度内反复投同类型的票，
// 而不是投过一次就被"你已经对这条自救内容投过票了"卡死——这正是被报告的 bug。
func TestOnGiveawayVoteAllowsRepeatedVotesUpToHourlyLimit(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	target := s.players["target1"]

	for i := int64(1); i <= 3; i++ {
		s.onGiveawayVote(client, voteEnv(i, "target1", "like"))
		data := lastReplyData(t, client)
		if data["ok"] != true {
			t.Fatalf("第 %d 次点赞应成功，got %#v", i, data)
		}
	}
	if got := ptrFloat(target.GiveawayValue); got != 47 {
		t.Fatalf("3 次点赞应各扣减 1%%，got %v", got)
	}
	if got := ptrInt(target.GiveawayBoardLikes); got != 3 {
		t.Fatalf("点赞计数应为 3，got %d", got)
	}

	// 第 4 次超出 LikeVoteLimitPerHour=3，应被拒绝且不再扣值。
	s.onGiveawayVote(client, voteEnv(4, "target1", "like"))
	if errMsg := lastReplyError(t, client); errMsg == "" {
		t.Fatal("超出每小时点赞额度应被拒绝")
	}
	if got := ptrFloat(target.GiveawayValue); got != 47 {
		t.Fatalf("超额后不应再次扣减白给值，got %v", got)
	}

	// 倒赞额度独立计算，此时应仍然可用。
	s.onGiveawayVote(client, voteEnv(5, "target1", "dislike"))
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("点赞额度耗尽不应影响倒赞，got %#v", data)
	}
	if got := ptrInt(target.GiveawayBoardDislikes); got != 1 {
		t.Fatalf("倒赞计数应为 1，got %d", got)
	}
	actor := s.players["actor1"]
	if ptrInt(actor.GiveawayVoteCount) != 4 || ptrInt(actor.GiveawayVoteLikesThisHour) != 3 || ptrInt(actor.GiveawayVoteDislikesThisHour) != 1 {
		t.Fatalf("legacy vote stats not updated: count=%d likes=%d dislikes=%d", ptrInt(actor.GiveawayVoteCount), ptrInt(actor.GiveawayVoteLikesThisHour), ptrInt(actor.GiveawayVoteDislikesThisHour))
	}
}

// TestOnGiveawayVoteQuotaPerTargetIndependent 确认额度按 actor→target 这一对独立计算：
// 对某个目标投满额度，不影响对另一个目标投票。
func TestOnGiveawayVoteQuotaPerTargetIndependent(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	now := nowMs()
	other := &PlayerState{PublicPlayer: types.PublicPlayer{
		ID: "target2", Name: "另一个自救玩家",
		GiveawayEnabled: boolPtr(true), GiveawayValue: floatPtr(50),
		GiveawayBoardText: "也求放过", GiveawayBoardExpiresAt: int64Ptr(now + 3_600_000),
	}}
	other.GiveawayBoardSubmittedAt = int64Ptr(now)
	s.players["target2"] = other

	for i := int64(1); i <= 3; i++ {
		s.onGiveawayVote(client, voteEnv(i, "target1", "like"))
		if data := lastReplyData(t, client); data["ok"] != true {
			t.Fatalf("对 target1 第 %d 次点赞应成功，got %#v", i, data)
		}
	}
	s.onGiveawayVote(client, voteEnv(4, "target1", "like"))
	if errMsg := lastReplyError(t, client); errMsg == "" {
		t.Fatal("target1 额度应已耗尽")
	}

	// target2 是独立的一对，额度未受影响。
	s.onGiveawayVote(client, voteEnv(5, "target2", "like"))
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("对 target2 投票不应受 target1 额度影响，got %#v", data)
	}
	if got := ptrFloat(other.GiveawayValue); got != 49 {
		t.Fatalf("target2 应正常扣减白给值，got %v", got)
	}
}

// TestOnGiveawayVoteQuotaResetsAfterWindow 窗口过期后额度应重新计算。
func TestOnGiveawayVoteQuotaResetsAfterWindow(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	target := s.players["target1"]
	for i := int64(1); i <= 3; i++ {
		s.onGiveawayVote(client, voteEnv(i, "target1", "like"))
		if data := lastReplyData(t, client); data["ok"] != true {
			t.Fatalf("第 %d 次点赞应成功，got %#v", i, data)
		}
	}
	s.onGiveawayVote(client, voteEnv(4, "target1", "like"))
	if errMsg := lastReplyError(t, client); errMsg == "" {
		t.Fatal("额度应已耗尽")
	}

	// 手动把窗口起始时间拨到 1 小时以前，模拟窗口过期。
	actor := s.players["actor1"]
	actor.GiveawayVoteQuotas["target1"].WindowStartedAt = nowMs() - 3_600_000 - 1
	target.GiveawayBoardExpiresAt = int64Ptr(nowMs() + 3_600_000)

	s.onGiveawayVote(client, voteEnv(5, "target1", "like"))
	data := lastReplyData(t, client)
	if data["ok"] != true {
		t.Fatalf("窗口过期后应重新允许投票，got %#v", data)
	}
}

// TestOnGiveawayVoteQuotasRPC 校验 giveaway:voteQuotas 按目标批量返回 actor 自己的额度，
// 且投票后额度会随之更新。
func TestOnGiveawayVoteQuotasRPC(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	s.onGiveawayVoteQuotas(client, wsEnvelope{E: "giveaway:voteQuotas", ID: 1, D: map[string]any{"targetIds": []string{"target1"}}})
	data := lastReplyData(t, client)
	quotas, _ := data["quotas"].(map[string]any)
	if quotas == nil {
		t.Fatalf("quotas 缺失: %#v", data)
	}
	q1, _ := quotas["target1"].(map[string]any)
	if q1 == nil {
		t.Fatalf("target1 额度缺失: %#v", quotas)
	}
	if q1["likeLimit"].(float64) != 3 {
		t.Fatalf("likeLimit=%#v", q1["likeLimit"])
	}
	if q1["likesUsed"].(float64) != 0 {
		t.Fatalf("初始 likesUsed 应为 0，got %#v", q1["likesUsed"])
	}

	s.onGiveawayVote(client, voteEnv(2, "target1", "like"))
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("投票应成功，got %#v", data)
	}

	s.onGiveawayVoteQuotas(client, wsEnvelope{E: "giveaway:voteQuotas", ID: 3, D: map[string]any{"targetIds": []string{"target1"}}})
	data = lastReplyData(t, client)
	quotas, _ = data["quotas"].(map[string]any)
	q1, _ = quotas["target1"].(map[string]any)
	if q1 == nil || q1["likesUsed"].(float64) != 1 {
		t.Fatalf("投票后 likesUsed 应为 1，got %#v", q1)
	}
}

// TestGiveawayVoteRulesForRelationshipTiers 校验点赞/倒赞每小时次数上限、以及每次投票的
// 升/降值百分比都按认主认宠关系分档：投给自己直系宠物走宠物档，投给自己直系主人走主人档，
// 其余（含二级以上关系）走普通档——三档的次数上限和升降值都各自独立，不会互相复用。
func TestGiveawayVoteRulesForRelationshipTiers(t *testing.T) {
	s, _ := setupGiveawayVoteServer(t)
	actor := s.players["actor1"]
	target := s.players["target1"]

	if likeLimit, dislikeLimit, likeValue, dislikeValue := s.giveawayVoteRulesFor(actor, target); likeLimit != 3 || dislikeLimit != 10 || likeValue != 1 || dislikeValue != 0.1 {
		t.Fatalf("无关系应走普通档，got likeLimit=%d dislikeLimit=%d likeValue=%v dislikeValue=%v", likeLimit, dislikeLimit, likeValue, dislikeValue)
	}

	// actor 是 target 的主人（target 是 actor 的宠物）：actor 投给 target 应走宠物档。
	s.petBonds = map[string]*petBond{bondKey(actor.ID, target.ID): {MasterID: actor.ID, PetID: target.ID}}
	if likeLimit, dislikeLimit, likeValue, dislikeValue := s.giveawayVoteRulesFor(actor, target); likeLimit != 5 || dislikeLimit != 15 || likeValue != 2 || dislikeValue != 0.2 {
		t.Fatalf("投给自己宠物应走宠物档，got likeLimit=%d dislikeLimit=%d likeValue=%v dislikeValue=%v", likeLimit, dislikeLimit, likeValue, dislikeValue)
	}

	// actor 是 target 的宠物（target 是 actor 的主人）：actor 投给 target 应走主人档。
	s.petBonds = map[string]*petBond{bondKey(target.ID, actor.ID): {MasterID: target.ID, PetID: actor.ID}}

	if likeLimit, dislikeLimit, likeValue, dislikeValue := s.giveawayVoteRulesFor(actor, target); likeLimit != 7 || dislikeLimit != 20 || likeValue != 3 || dislikeValue != 0.3 {
		t.Fatalf("投给自己主人应走主人档，got likeLimit=%d dislikeLimit=%d likeValue=%v dislikeValue=%v", likeLimit, dislikeLimit, likeValue, dislikeValue)
	}
}

func TestGiveawayVotePreservesHundredthValue(t *testing.T) {
	s, client := setupGiveawayVoteServer(t)
	s.cfg.Giveaway.LikeVoteValue = 0.01
	target := s.players["target1"]
	s.onGiveawayVote(client, voteEnv(1, "target1", "like"))
	if data := lastReplyData(t, client); data["ok"] != true {
		t.Fatalf("投票应成功，got %#v", data)
	}
	if got := ptrFloat(target.GiveawayValue); math.Abs(got-49.99) > 1e-9 {
		t.Fatalf("0.01%% 投票值应生效，got %v", got)
	}
}

func lastReplyError(t *testing.T, client *Client) string {
	t.Helper()
	select {
	case data := <-client.sendCh:
		var env wire.Envelope
		if err := proto.Unmarshal(data, &env); err != nil {
			t.Fatalf("failed to decode reply: %v", err)
		}
		return env.Err
	default:
		t.Fatal("expected a reply on sendCh, got none")
		return ""
	}
}
