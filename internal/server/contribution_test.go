package server

import (
	"fmt"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestValidateStepDraft_overlappingFactions(t *testing.T) {
	_, err := decodeStepDraft(`{"inRandomPool":true,"order":10,"variants":[{"text":"甲","factionIds":["f1"]},{"text":"乙","factionIds":["f1"]}]}`)
	if err == nil {
		t.Fatal("overlapping factions must be rejected")
	}
}

func TestDecodeStepDraft_order(t *testing.T) {
	for _, order := range []int{1, 50, 99} {
		draft, err := decodeStepDraft(fmt.Sprintf(`{"inRandomPool":true,"order":%d,"variants":[{"text":"文案","factionIds":["f1"]}]}`, order))
		if err != nil {
			t.Fatalf("order=%d should be accepted, got err=%v", order, err)
		}
		if draft.Order != order || !draft.InRandomPool {
			t.Fatalf("order=%d got %+v", order, draft)
		}
	}
	for _, order := range []int{-1, 0, 100} {
		if _, err := decodeStepDraft(fmt.Sprintf(`{"inRandomPool":true,"order":%d,"variants":[{"text":"文案","factionIds":["f1"]}]}`, order)); err == nil {
			t.Fatalf("in-pool order=%d should be rejected", order)
		}
	}
	draft, err := decodeStepDraft(`{"inRandomPool":false,"order":0,"variants":[{"text":"文案","factionIds":["f1"]}]}`)
	if err != nil {
		t.Fatalf("series-only step should ignore order: %v", err)
	}
	if draft.InRandomPool {
		t.Fatal("expected inRandomPool=false")
	}
}

func TestDecodeStepDraft_when_backgroundOpacityOutOfRange_then_rejected(t *testing.T) {
	for _, opacity := range []string{"-0.1", "1.1"} {
		raw := fmt.Sprintf(`{"inRandomPool":true,"order":10,"backgroundOpacity":%s,"variants":[{"text":"文案","factionIds":["f1"]}]}`, opacity)
		if _, err := decodeStepDraft(raw); err == nil {
			t.Fatalf("backgroundOpacity=%s must be rejected before entering review", opacity)
		}
	}
}

func TestDecodeSeriesDraft_when_nameOrStepsMissing_then_rejected(t *testing.T) {
	if _, err := decodeSeriesDraft(`{"name":"","steps":[]}`); err == nil {
		t.Fatal("empty name and steps must be rejected")
	}
	if _, err := decodeSeriesDraft(`{"name":"系列","steps":[]}`); err == nil {
		t.Fatal("no steps must be rejected")
	}
	if _, err := decodeSeriesDraft(`{"name":"系列","targetFactionIds":["f1","f2"],"steps":[{"variants":[{"text":"只覆盖一营","factionIds":["f1"]}]}]}`); err == nil {
		t.Fatal("every step must cover every declared target faction")
	}
	draft, err := decodeSeriesDraft(`{"name":"系列","targetFactionIds":["f1"],"steps":[{"inRandomPool":true,"order":10,"variants":[{"text":"第一步","factionIds":["f1"]}]}]}`)
	if err != nil {
		t.Fatal(err)
	}
	if len(draft.Steps) != 1 || draft.Steps[0].Variants[0].Text != "第一步" {
		t.Fatalf("draft=%+v", draft)
	}
}

func TestNormalizeGenderLabel_when_unicodeAndCase_then_folds(t *testing.T) {
	if got := normalizeGenderLabel("  ＡＢＣ  "); got != "abc" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeGenderLabel("Girl\x00 "); got != "girl" {
		t.Fatalf("control chars: %q", got)
	}
}

func TestGenderStore_when_sameFactionSameName_then_unique(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gs := newGenderStore(db)
	factions := []types.GenderFaction{{
		ID: "f1", Label: "阵营一", TaskGroup: "default",
		GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
	}}
	genders := []types.GenderOption{{ID: "g1", Label: "猫", FactionID: "f1"}}
	if err := gs.replaceAll(factions, genders); err != nil {
		t.Fatal(err)
	}
	err = gs.replaceAll(factions, []types.GenderOption{
		{ID: "g1", Label: "猫", FactionID: "f1"},
		{ID: "g2", Label: "猫", FactionID: "f1"},
	})
	if err == nil {
		t.Fatal("same label in one faction must fail")
	}
	// 存储层唯一约束是字面量 UNIQUE(faction_id, label)，只挡完全相同的字符串；大小写/
	// 全半角/空白不敏感的近似查重是应用层职责（adminSaveGenders 的 seenLabels 检查，见
	// normalizeGenderLabel），不在这一层，近似重复在 store 层应当能正常写入。
	if err := gs.replaceAll(factions, []types.GenderOption{
		{ID: "g1", Label: "猫", FactionID: "f1"},
		{ID: "g2", Label: " 猫 ", FactionID: "f1"},
	}); err != nil {
		t.Fatalf("near-duplicate (whitespace) label should pass at store level: %v", err)
	}
	err = gs.replaceAll(append(factions, types.GenderFaction{
		ID: "f2", Label: "阵营二", TaskGroup: "default",
		GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
	}), []types.GenderOption{
		{ID: "g1", Label: "猫", FactionID: "f1"},
		{ID: "g2", Label: "猫", FactionID: "f2"},
	})
	if err != nil {
		t.Fatalf("same name in different factions: %v", err)
	}
}

func TestGenderJSONSeed_when_tablesEmpty_then_import(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, genderStore: newGenderStore(db)}
	if err := s.importGendersFromJSONIfNeeded(); err != nil {
		t.Fatal(err)
	}
	n, err := s.genderStore.countGenders()
	if err != nil || n == 0 {
		t.Fatalf("seed genders n=%d err=%v", n, err)
	}
	if err := s.importGendersFromJSONIfNeeded(); err != nil {
		t.Fatal(err)
	}
	n2, _ := s.genderStore.countGenders()
	if n2 != n {
		t.Fatalf("second import changed count %d -> %d", n, n2)
	}
}

func newContributionTestServer(t *testing.T) *Server {
	t.Helper()
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return &Server{
		db:                db,
		players:           map[string]*PlayerState{},
		contributionStore: newContributionStore(db),
		punishmentStore:   newPunishmentStore(db),
		eventDB:           newEventStore(db),
		cfg: types.AppConfig{
			GenderFactions:           []types.GenderFaction{{ID: "f1", Label: "阵营"}, {ID: "f2", Label: "阵营二"}},
			PunishmentRandomSettings: types.PunishmentRandomSettings{MinSeriesSteps: 1},
		},
	}
}

func TestContributionOwnership_when_otherPlayerReads_then_forbidden(t *testing.T) {
	s := newContributionTestServer(t)
	raw, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "t"}}})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, true, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.getOwned(types.ContributionKindTask, sub.ID, "p2"); err != errContributionForbidden {
		t.Fatalf("got %v", err)
	}
}

// TestContributionLifecycle_task 覆盖任务投稿的完整生命周期：草稿 -> 提交 -> 管理员批准 ->
// 进入随机池 -> 撤回。
func TestContributionLifecycle_task(t *testing.T) {
	s := newContributionTestServer(t)
	raw, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 30,
		Variants: []types.TaskVariantDraft{{Text: "给甲", FactionIDs: []string{"f1"}}},
	})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if sub.Status != types.ContributionDraft {
		t.Fatalf("new draft status=%s", sub.Status)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p2", sub.ID); err == nil {
		t.Fatal("other player must not submit someone else's draft")
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	got, err := s.contributionStore.get(types.ContributionKindTask, sub.ID)
	if err != nil || got.Status != types.ContributionPending {
		t.Fatalf("after submit status=%s err=%v", got.Status, err)
	}

	if err := s.adminApprove(types.ContributionKindTask, sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	s.reloadPunishmentCaches()
	found := false
	for _, task := range s.punishmentTasksCache {
		if task.ID == sub.ID {
			found = true
		}
	}
	if !found {
		t.Fatalf("approved task should enter the random pool cache: %#v", s.punishmentTasksCache)
	}

	if err := s.contributionStore.withdraw(types.ContributionKindTask, "p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	s.reloadPunishmentCaches()
	for _, task := range s.punishmentTasksCache {
		if task.ID == sub.ID {
			t.Fatalf("withdrawn task must leave the pool: %#v", s.punishmentTasksCache)
		}
	}
}

// TestContributionAdminReject_then_revertReject 覆盖管理员驳回 / 撤销驳回这一对纯状态流转。
func TestContributionAdminReject_then_revertReject(t *testing.T) {
	s := newContributionTestServer(t)
	raw, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "t", FactionIDs: []string{"f1"}}}})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminReject(types.ContributionKindTask, sub.ID, "admin", "文案不合适"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.contributionStore.get(types.ContributionKindTask, sub.ID)
	if got.Status != types.ContributionRejected {
		t.Fatalf("status=%s", got.Status)
	}
	if err := s.adminSetStatus(types.ContributionKindTask, sub.ID, types.ContributionPending, "admin", ""); err != nil {
		t.Fatal(err)
	}
	got, _ = s.contributionStore.get(types.ContributionKindTask, sub.ID)
	if got.Status != types.ContributionPending {
		t.Fatalf("after revert-reject status=%s", got.Status)
	}
}

// TestContributionAdminApprove_withReviewedContent_then_insertsNewApprovedVersion 覆盖管理员
// 就地改稿后批准：内容变了，必须落一个新版本，且新版本直接是 approved。
func TestContributionAdminApprove_withReviewedContent_then_insertsNewApprovedVersion(t *testing.T) {
	s := newContributionTestServer(t)
	raw, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "原文案", FactionIDs: []string{"f1"}}}})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	edited, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 20, Variants: []types.TaskVariantDraft{{Text: "改后文案", FactionIDs: []string{"f1"}}}})
	if err := s.adminApprove(types.ContributionKindTask, sub.ID, "admin", "改了下措辞", edited); err != nil {
		t.Fatal(err)
	}
	got, err := s.contributionStore.get(types.ContributionKindTask, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != types.ContributionApproved || got.Version != 2 {
		t.Fatalf("got=%+v", got)
	}
	draft, ok := got.Content.(types.StepDraft)
	if !ok || draft.Variants[0].Text != "改后文案" {
		t.Fatalf("content not updated: %+v", got.Content)
	}
}

// TestSaveSeriesSteps_when_savedTwice_then_reusesPositionalStepID 覆盖系列步骤"位置 i 的内容
// 延续该系列此前位置 i 那一步的 ID"这条续版规则。
func TestSaveSeriesSteps_when_savedTwice_then_reusesPositionalStepID(t *testing.T) {
	s := newContributionTestServer(t)
	first, _ := marshalDraft(types.SeriesDraft{
		Name: "共建系列", TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{
			{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "第一步", FactionIDs: []string{"f1"}}}},
		},
	})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, "", first)
	if err != nil {
		t.Fatal(err)
	}
	steps, err := s.contributionStore.tasks.stepsForSeries(sub.ID)
	if err != nil || len(steps) != 1 {
		t.Fatalf("steps=%v err=%v", steps, err)
	}
	stepID := steps[0].ID

	next, _ := marshalDraft(types.SeriesDraft{
		Name: "共建系列", TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{
			{InRandomPool: true, Order: 15, Variants: []types.TaskVariantDraft{{Text: "第一步(改)", FactionIDs: []string{"f1"}}}},
			{InRandomPool: true, Order: 25, Variants: []types.TaskVariantDraft{{Text: "第二步", FactionIDs: []string{"f1"}}}},
		},
	})
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, sub.ID, next); err != nil {
		t.Fatal(err)
	}
	steps2, err := s.contributionStore.tasks.stepsForSeries(sub.ID)
	if err != nil || len(steps2) != 2 {
		t.Fatalf("steps2=%v err=%v", steps2, err)
	}
	if steps2[0].ID != stepID {
		t.Fatalf("position 0 should reuse the same step id across saves, got %q want %q", steps2[0].ID, stepID)
	}
	if steps2[0].Version != 2 {
		t.Fatalf("reused step should bump its own version, got %d", steps2[0].Version)
	}
	if steps2[1].ID == stepID {
		t.Fatal("newly appended step must get its own id")
	}
}

// TestContributionImage_when_replacedByReupload_then_oldVersionNoLongerLive 覆盖"这张图当前
// 是不是它所属 id 最新版本仍在引用的图"这条访问控制规则：不看审核状态，重新上传覆盖之后，
// 旧版本行不再是 MAX(version)，其 background_image 自然不再被判定为"活的"。
func TestContributionImage_when_replacedByReupload_then_oldVersionNoLongerLive(t *testing.T) {
	s := newContributionTestServer(t)
	pathA := "/uploads/contributions/a.webp"
	pathB := "/uploads/contributions/b.webp"
	rawA, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, BackgroundImage: pathA, Variants: []types.TaskVariantDraft{{Text: "t", FactionIDs: []string{"f1"}}}})
	sub, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", rawA)
	if err != nil {
		t.Fatal(err)
	}
	live, err := s.contributionStore.tasks.imageIsLive(pathA)
	if err != nil || !live {
		t.Fatalf("just-referenced image must be live: live=%v err=%v", live, err)
	}

	rawB, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, BackgroundImage: pathB, Variants: []types.TaskVariantDraft{{Text: "t", FactionIDs: []string{"f1"}}}})
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, sub.ID, rawB); err != nil {
		t.Fatal(err)
	}
	if live, err := s.contributionStore.tasks.imageIsLive(pathA); err != nil || live {
		t.Fatalf("replaced image must no longer be live: live=%v err=%v", live, err)
	}
	if live, err := s.contributionStore.tasks.imageIsLive(pathB); err != nil || !live {
		t.Fatalf("newly referenced image must be live: live=%v err=%v", live, err)
	}
}

// TestContributionVote_dedupAndSpoilerFree 覆盖评价 RPC 的两条规则：投票前协议层不下发
// 点赞率/贡献者（voteCardFor 的 MyVote==0 分支），以及同一侧只能投一次（数据库层的
// WHERE <col>=0 原子去重，不需要单独的投票表）。
func TestContributionVote_dedupAndSpoilerFree(t *testing.T) {
	s := newContributionTestServer(t)
	author := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "author", Name: "投稿者"}}
	performer := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "performer"}}
	approver := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "approver"}}
	for _, p := range []*PlayerState{author, performer, approver} {
		s.players[p.ID] = p
	}

	taskDraft, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}}})
	taskItem, err := s.contributionStore.saveDraft(author.ID, types.ContributionKindTask, false, "", taskDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, author.ID, taskItem.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindTask, taskItem.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	if err := s.eventDB.insertPunishmentTask("e1", nowMs(), "room1", approver.ID, "审批者", performer.ID, "执行者", "文案", punishmentEventMeta{
		FormalTaskID: taskItem.ID, FormalTaskVersion: taskItem.Version,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.eventDB.updatePunishmentStatus("e1", "approved"); err != nil {
		t.Fatal(err)
	}

	row, err := s.eventDB.getPunishmentEvent("e1")
	if err != nil {
		t.Fatal(err)
	}
	preview := s.voteCardFor(performer.ID, row)
	if preview.ContributorDisplayName != "" || preview.HasVotes || preview.DisplayRatio != nil {
		t.Fatalf("pre-vote card leaked spoilers: %+v", preview)
	}
	if !preview.CanVote {
		t.Fatalf("performer should be eligible to vote: %+v", preview)
	}

	if _, err := s.castContributionVote(performer, "e1", types.ContributionVoteUp); err != nil {
		t.Fatal(err)
	}
	if _, err := s.castContributionVote(performer, "e1", types.ContributionVoteDown); err == nil {
		t.Fatal("same side must not be able to vote twice")
	}
	if _, err := s.castContributionVote(author, "e1", types.ContributionVoteUp); err == nil {
		t.Fatal("contributor must not vote on their own content")
	}
	if _, err := s.castContributionVote(approver, "e1", types.ContributionVoteUp); err != nil {
		t.Fatal(err)
	}

	row2, err := s.eventDB.getPunishmentEvent("e1")
	if err != nil {
		t.Fatal(err)
	}
	after := s.voteCardFor(performer.ID, row2)
	if after.ContributorDisplayName != "投稿者" || !after.HasVotes || after.DisplayRatio == nil || *after.DisplayRatio != 100 {
		t.Fatalf("post-vote card should reveal aggregate and attribution: %+v", after)
	}
	likes, downs, err := s.contributionStore.tasks.voteAggregate(taskItem.ID)
	if err != nil || likes != 2 || downs != 0 {
		t.Fatalf("likes=%d downs=%d err=%v", likes, downs, err)
	}
}

// TestVoteCardFor_beforeProofApproved_thenAlreadyVotable 惩罚任务一发布、证明还没提交/
// 还没被对手审核完（status 还是 assigned/pending）时评价卡片就已经在渲染（AppViews.tsx 的
// punishment-card 不等证明状态就挂载 ContributionVoteLazy）——评价的是任务文案本身好不好，
// 执行者/审批者从看到文案那一刻起就已经在"执行"或"审批"这条任务了，不需要等证明审核通过
// 才能打分：证明审核通过往往和本局结算/进入下一局同时发生（finishPunishmentIfComplete
// 与批准同一次处理），若硬性要求已批准，玩家几乎抓不到 phase==="punishment" 与刚批准同时
// 成立的那一瞬间，评价入口等于形同虚设。
func TestVoteCardFor_beforeProofApproved_thenAlreadyVotable(t *testing.T) {
	s := newContributionTestServer(t)
	author := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "author", Name: "投稿者"}}
	performer := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "performer"}}
	s.players[author.ID], s.players[performer.ID] = author, performer

	taskDraft, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}}})
	taskItem, err := s.contributionStore.saveDraft(author.ID, types.ContributionKindTask, false, "", taskDraft)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, author.ID, taskItem.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindTask, taskItem.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	// 任务刚发布，还没有 updatePunishmentStatus("approved")——对应真实流程里"证明还没提交/
	// 还没被对手审核"这段时间，row.Status 停在建行时写死的 "assigned"。
	if err := s.eventDB.insertPunishmentTask("e1", nowMs(), "room1", "approver", "审批者", performer.ID, "执行者", "文案", punishmentEventMeta{
		FormalTaskID: taskItem.ID, FormalTaskVersion: taskItem.Version,
	}); err != nil {
		t.Fatal(err)
	}
	row, err := s.eventDB.getPunishmentEvent("e1")
	if err != nil {
		t.Fatal(err)
	}
	card := s.voteCardFor(performer.ID, row)
	if !card.CanVote {
		t.Fatalf("should already be votable before proof is approved: %+v", card)
	}
	if card.CannotVoteReason != "" {
		t.Fatalf("votable card must not carry a reason, got %+v", card)
	}
}

// TestAdminGendersRoundTrip 覆盖后台「性别与阵营」批量管理走 gendersGet/gendersSave——
// 这条路径与共建投稿系统无关（不经过 sub_tasks/series），但复用同一个
// handleAdminContributionAction 入口分发，历史上曾在一次重构里被误删，这里钉一个回归测试。
func TestAdminGendersRoundTrip(t *testing.T) {
	s := newContributionTestServer(t)
	s.genderStore = newGenderStore(s.db)

	saveRaw := map[string]any{
		"factions": []map[string]any{
			{"id": "f1", "label": "阵营一", "taskGroup": "default", "textColor": "#111111", "backgroundColor": "#222222", "borderColor": "#333333"},
		},
		"genders": []map[string]any{
			{"id": "g1", "label": "猫", "factionId": "f1"},
		},
	}
	if _, err := s.handleAdminContributionAction("gendersSave", saveRaw); err != nil {
		t.Fatal(err)
	}
	got, err := s.handleAdminContributionAction("gendersGet", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := got.(map[string]any)
	if !ok {
		t.Fatalf("gendersGet returned %#v", got)
	}
	factions, _ := m["factions"].([]types.GenderFaction)
	genders, _ := m["genders"].([]types.GenderOption)
	if len(factions) != 1 || factions[0].ID != "f1" {
		t.Fatalf("factions=%#v", m["factions"])
	}
	if len(genders) != 1 || genders[0].ID != "g1" {
		t.Fatalf("genders=%#v", m["genders"])
	}
}

func TestSaveSeriesSteps_whenSeriesShrinks_thenRemovedStepsStayInactive(t *testing.T) {
	s := newContributionTestServer(t)
	step := func(text string) types.StepDraft {
		return types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: text, FactionIDs: []string{"f1"}}}}
	}
	first, _ := marshalDraft(types.SeriesDraft{Name: "缩短测试", TargetFactionIDs: []string{"f1"}, Steps: []types.StepDraft{step("一"), step("二"), step("三")}})
	item, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, "", first)
	if err != nil {
		t.Fatal(err)
	}
	before, err := s.contributionStore.tasks.stepsForSeries(item.ID)
	if err != nil || len(before) != 3 {
		t.Fatalf("before=%v err=%v", before, err)
	}
	removed := []string{before[1].ID, before[2].ID}

	shortened, _ := marshalDraft(types.SeriesDraft{Name: "缩短测试", TargetFactionIDs: []string{"f1"}, Steps: []types.StepDraft{step("一（改）")}})
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, item.ID, shortened); err != nil {
		t.Fatal(err)
	}
	current, err := s.contributionStore.tasks.stepsForSeries(item.ID)
	if err != nil || len(current) != 1 {
		t.Fatalf("current=%v err=%v", current, err)
	}
	for _, id := range removed {
		row, err := s.contributionStore.tasks.latest(id)
		if err != nil || row.Active {
			t.Fatalf("removed step %s active=%v err=%v", id, row.Active, err)
		}
	}
	if err := s.contributionStore.submit(types.ContributionKindSeries, "p1", item.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindSeries, item.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.withdraw(types.ContributionKindSeries, "p1", item.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.validateContributionForApproval(types.ContributionKindSeries, item.ID, ""); err != nil {
		t.Fatal(err)
	}
	if err := s.adminSetStatus(types.ContributionKindSeries, item.ID, types.ContributionApproved, "admin", ""); err != nil {
		t.Fatal(err)
	}
	s.reloadPunishmentCaches()
	for _, task := range s.punishmentTasksCache {
		for _, id := range removed {
			if task.ID == id {
				t.Fatalf("removed step %s returned to cache", id)
			}
		}
	}
}

func TestContributionVote_whenTaskWasEdited_thenCountsExperiencedVersion(t *testing.T) {
	s := newContributionTestServer(t)
	author := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "author", Name: "作者"}}
	performer := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "performer"}}
	s.players[author.ID], s.players[performer.ID] = author, performer
	v1, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "第一版", FactionIDs: []string{"f1"}}}})
	item, err := s.contributionStore.saveDraft(author.ID, types.ContributionKindTask, false, "", v1)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, author.ID, item.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindTask, item.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.eventDB.insertPunishmentTask("event-v1", nowMs(), "room", "reviewer", "审核方", performer.ID, "执行方", "第一版", punishmentEventMeta{
		FormalTaskID: item.ID, FormalTaskVersion: 1,
	}); err != nil {
		t.Fatal(err)
	}
	if err := s.eventDB.updatePunishmentStatus("event-v1", "approved"); err != nil {
		t.Fatal(err)
	}
	v2, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 20, Variants: []types.TaskVariantDraft{{Text: "第二版", FactionIDs: []string{"f1"}}}})
	if _, err := s.contributionStore.saveDraft(author.ID, types.ContributionKindTask, false, item.ID, v2); err != nil {
		t.Fatal(err)
	}
	if _, err := s.castContributionVote(performer, "event-v1", types.ContributionVoteUp); err != nil {
		t.Fatal(err)
	}
	var v1Likes, v2Likes int
	if err := s.db.QueryRow(`SELECT like_count FROM sub_tasks WHERE id=? AND version=1`, item.ID).Scan(&v1Likes); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT like_count FROM sub_tasks WHERE id=? AND version=2`, item.ID).Scan(&v2Likes); err != nil {
		t.Fatal(err)
	}
	if v1Likes != 1 || v2Likes != 0 {
		t.Fatalf("votes landed on wrong version: v1=%d v2=%d", v1Likes, v2Likes)
	}
}

func TestContributionAdminStateAndKindValidation(t *testing.T) {
	s := newContributionTestServer(t)
	raw, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 10, Variants: []types.TaskVariantDraft{{Text: "任务", FactionIDs: []string{"f1"}}}})
	item, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindTask, item.ID, "admin", "", ""); err != errContributionBadStatus {
		t.Fatalf("draft must not be approved directly, got %v", err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p1", item.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, item.ID, raw); err != errContributionBadStatus {
		t.Fatalf("pending item must not be edited, got %v", err)
	}
	if _, err := s.handleAdminContributionAction("contributionUpdateComment", map[string]any{"id": item.ID, "kind": "not-a-kind"}); err == nil {
		t.Fatal("unknown contribution kind must be rejected")
	}
	if _, err := s.handleAdminContributionAction("contributionUpdateComment", map[string]any{"id": item.ID, "kind": 123}); err == nil {
		t.Fatal("non-string contribution kind must be rejected")
	}
}

func TestContributionSaveDraft_whenVersionLimitReached_thenRejected(t *testing.T) {
	s := newContributionTestServer(t)
	taskRaw, _ := marshalDraft(types.StepDraft{
		InRandomPool: true,
		Order:        10,
		Variants:     []types.TaskVariantDraft{{Text: "任务", FactionIDs: []string{"f1"}}},
	})
	task, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", taskRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE sub_tasks SET version=? WHERE id=?`, maxContributionVersionsPerItem, task.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, task.ID, taskRaw); err == nil {
		t.Fatal("task at the version limit must reject another save")
	}

	seriesRaw, _ := marshalDraft(types.SeriesDraft{
		Name:             "系列",
		TargetFactionIDs: []string{"f1"},
		Steps:            []types.StepDraft{{Variants: []types.TaskVariantDraft{{Text: "一步", FactionIDs: []string{"f1"}}}}},
	})
	series, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, "", seriesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.db.Exec(`UPDATE series SET version=? WHERE id=?`, maxContributionVersionsPerItem, series.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, series.ID, seriesRaw); err == nil {
		t.Fatal("series at the version limit must reject another save")
	}
}

func TestAnalyticsPunishmentPoolMetricsUseVersionedTables(t *testing.T) {
	s := newContributionTestServer(t)
	s.analyticsRO = s.db

	taskRaw, _ := marshalDraft(types.StepDraft{
		InRandomPool: true,
		Order:        10,
		Variants:     []types.TaskVariantDraft{{Text: "独立任务", FactionIDs: []string{"f1"}}},
	})
	task, err := s.contributionStore.saveDraft("p1", types.ContributionKindTask, false, "", taskRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindTask, "p1", task.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindTask, task.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	seriesRaw, _ := marshalDraft(types.SeriesDraft{
		Name:             "系列",
		TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{{
			InRandomPool: true,
			Order:        20,
			Variants:     []types.TaskVariantDraft{{Text: "系列步骤", FactionIDs: []string{"f1"}}},
		}},
	})
	series, err := s.contributionStore.saveDraft("p1", types.ContributionKindSeries, false, "", seriesRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit(types.ContributionKindSeries, "p1", series.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.adminApprove(types.ContributionKindSeries, series.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	day := nowMs() / 86_400_000
	rows, err := s.rebuildDay(day, 0)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]int64{}
	for _, row := range rows {
		if row.Key == "" {
			got[row.Metric] = row.Value
		}
	}
	if got[metricPunishTaskPoolNew] != 2 || got[metricPunishTaskPoolTotal] != 2 {
		t.Fatalf("task pool metrics new/total=%d/%d", got[metricPunishTaskPoolNew], got[metricPunishTaskPoolTotal])
	}
	if got[metricPunishSeriesPoolNew] != 1 || got[metricPunishSeriesPoolTotal] != 1 {
		t.Fatalf("series pool metrics new/total=%d/%d", got[metricPunishSeriesPoolNew], got[metricPunishSeriesPoolTotal])
	}
}
