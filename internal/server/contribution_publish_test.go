package server

import (
	"database/sql"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func newContributionPublishServer(t *testing.T) *Server {
	t.Helper()
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	gs := newGenderStore(db)
	factions := []types.GenderFaction{
		{
			ID: "f1", Label: "阵营", TaskGroup: "default",
			GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
		},
		{
			ID: "f2", Label: "阵营二", TaskGroup: "default",
			GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
		},
	}
	if err := gs.replaceAll(factions, nil); err != nil {
		t.Fatal(err)
	}
	return &Server{
		db:                db,
		genderStore:       gs,
		contributionStore: newContributionStore(db),
		punishmentStore:   newPunishmentStore(db),
		cfg: types.AppConfig{
			GenderFactions:           factions,
			PunishmentRandomSettings: types.PunishmentRandomSettings{MinSeriesSteps: 1},
		},
	}
}

func countByQuery(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRow(q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestUnpublishContribution_when_taskApproved_then_removesOfficialTask(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.TaskDraft{Text: "做这个", Order: 20, FactionIDs: []string{"f1"}})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 1 {
		t.Fatalf("published task count=%d", n)
	}

	if err := s.unpublishContribution(sub.ID, "admin", "下架"); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 0 {
		t.Fatalf("unpublished task still in pool: %d", n)
	}
	got, err := s.contributionStore.get(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != types.ContributionWithdrawn {
		t.Fatalf("status=%s", got.Status)
	}
}

func TestUnpublishContribution_when_seriesApproved_then_removesSeriesAndChildTasks(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.LegacySeriesDraft{
		Name:  "共建系列",
		Steps: []types.SeriesDraftStep{{TaskRefs: []string{"a"}}, {TaskRefs: []string{"b"}}},
		Tasks: []types.TaskDraftWithRef{
			{Ref: "a", Task: types.TaskDraft{Text: "第一步", Order: 10, FactionIDs: []string{"f1"}}},
			{Ref: "b", Task: types.TaskDraft{Text: "第二步", Order: 20, FactionIDs: []string{"f1"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_series WHERE submission_id = ?`, sub.ID); n != 1 {
		t.Fatalf("published series count=%d", n)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 2 {
		t.Fatalf("published series tasks=%d", n)
	}

	if err := s.unpublishContribution(sub.ID, "admin", "下架"); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_series WHERE submission_id = ?`, sub.ID); n != 0 {
		t.Fatalf("unpublished series still in pool: %d", n)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 0 {
		t.Fatalf("unpublished series child tasks remain: %d", n)
	}
}

func TestPublishContribution_when_seriesRevised_then_replacesOldChildTasks(t *testing.T) {
	s := newContributionPublishServer(t)
	first, err := marshalDraft(types.LegacySeriesDraft{
		Name:  "共建系列",
		Steps: []types.SeriesDraftStep{{TaskRefs: []string{"a"}}},
		Tasks: []types.TaskDraftWithRef{
			{Ref: "a", Task: types.TaskDraft{Text: "旧文案", Order: 10, FactionIDs: []string{"f1"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, "", first)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	oldIDs := map[string]struct{}{}
	rows, err := s.db.Query(`SELECT id FROM punishment_tasks WHERE submission_id = ?`, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		oldIDs[id] = struct{}{}
	}
	_ = rows.Close()
	if len(oldIDs) != 1 {
		t.Fatalf("first publish tasks=%d", len(oldIDs))
	}

	next, err := marshalDraft(types.LegacySeriesDraft{
		Name:  "共建系列",
		Steps: []types.SeriesDraftStep{{TaskRefs: []string{"n1"}}, {TaskRefs: []string{"n2"}}},
		Tasks: []types.TaskDraftWithRef{
			{Ref: "n1", Task: types.TaskDraft{Text: "新第一步", Order: 15, FactionIDs: []string{"f1"}}},
			{Ref: "n2", Task: types.TaskDraft{Text: "新第二步", Order: 25, FactionIDs: []string{"f1"}}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, sub.ID, next); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	var texts []string
	rows, err = s.db.Query(`SELECT id, text FROM punishment_tasks WHERE submission_id = ?`, sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var id, text string
		if err := rows.Scan(&id, &text); err != nil {
			t.Fatal(err)
		}
		if _, ok := oldIDs[id]; ok {
			t.Fatalf("old child task %s still present after revision", id)
		}
		texts = append(texts, text)
	}
	_ = rows.Close()
	if len(texts) != 2 {
		t.Fatalf("revised series tasks=%d texts=%v", len(texts), texts)
	}
}

func TestPublishContribution_when_taskHasVariants_then_insertsRows(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.StepDraft{
		InRandomPool: true,
		Order:        30,
		Variants: []types.TaskVariantDraft{
			{Text: "给甲", FactionIDs: []string{"f1"}},
			{Text: "给乙", FactionIDs: []string{"f2"}},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 2 {
		t.Fatalf("variant rows=%d", n)
	}
}

func TestContributionSaveDraft_when_approvedDraftSavedRepeatedly_then_reusesRevisionVersion(t *testing.T) {
	s := newContributionPublishServer(t)
	first, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "第一版", FactionIDs: []string{"f1"}}},
	})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", first)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	second, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 30,
		Variants: []types.TaskVariantDraft{{Text: "第二版草稿", FactionIDs: []string{"f1"}}},
	})
	third, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 40,
		Variants: []types.TaskVariantDraft{{Text: "第三次保存", FactionIDs: []string{"f1"}}},
	})
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, sub.ID, second); err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, sub.ID, third); err != nil {
		t.Fatal(err)
	}
	got, err := s.contributionStore.get(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ActiveVersion != 2 || got.PublishedVersion != 1 {
		t.Fatalf("active=%d published=%d", got.ActiveVersion, got.PublishedVersion)
	}
	if got.Status != types.ContributionRevisionDraft {
		t.Fatalf("saved revision status=%s, want %s", got.Status, types.ContributionRevisionDraft)
	}
	var liveText string
	if err := s.db.QueryRow(`SELECT text FROM punishment_tasks WHERE submission_id = ? LIMIT 1`, sub.ID).Scan(&liveText); err != nil {
		t.Fatal(err)
	}
	if liveText != "第一版" {
		t.Fatalf("unapproved revision replaced the live version: %q", liveText)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM contribution_versions WHERE submission_id = ?`, sub.ID); n != 2 {
		t.Fatalf("repeated saves must reuse one unpublished revision row, versions=%d", n)
	}
	ver, err := s.contributionStore.getVersion(sub.ID, got.ActiveVersion)
	if err != nil {
		t.Fatal(err)
	}
	draft, err := decodeStepDraft(ver.Content)
	if err != nil {
		t.Fatal(err)
	}
	if draft.Variants[0].Text != "第三次保存" {
		t.Fatalf("latest save was not retained: %+v", draft.Variants)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if pending, _ := s.contributionStore.get(sub.ID); pending.Status != types.ContributionRevisionPending {
		t.Fatalf("submitted revision status=%s", pending.Status)
	}
	if err := s.contributionStore.reject(sub.ID, "admin", "需要修改"); err != nil {
		t.Fatal(err)
	}
	if rejected, _ := s.contributionStore.get(sub.ID); rejected.Status != types.ContributionRevisionRejected {
		t.Fatalf("rejected revision status=%s", rejected.Status)
	}
	if err := s.publishContribution(sub.ID, "admin", third, "管理员直接修订"); err != nil {
		t.Fatal(err)
	}
	approved, _ := s.contributionStore.get(sub.ID)
	if approved.Status != types.ContributionApproved || approved.PublishedVersion != 2 || approved.ActiveVersion != 2 {
		t.Fatalf("admin immediate publish state=%+v", approved)
	}
	if err := s.db.QueryRow(`SELECT text FROM punishment_tasks WHERE submission_id = ? LIMIT 1`, sub.ID).Scan(&liveText); err != nil {
		t.Fatal(err)
	}
	if liveText != "第三次保存" {
		t.Fatalf("admin edit did not immediately replace formal content: %q", liveText)
	}
}

func TestPublishContribution_when_adminEditsApprovedContent_thenCreatesAuditedVersion(t *testing.T) {
	s := newContributionPublishServer(t)
	first, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "原发布稿", FactionIDs: []string{"f1"}}},
	})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", first)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	edited, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 25,
		Variants: []types.TaskVariantDraft{{Text: "管理员修订稿", FactionIDs: []string{"f1"}}},
	})
	if err := s.publishContribution(sub.ID, "admin-player", edited, "即时修订"); err != nil {
		t.Fatal(err)
	}
	got, _ := s.contributionStore.get(sub.ID)
	if got.ActiveVersion != 2 || got.PublishedVersion != 2 || got.Status != types.ContributionApproved {
		t.Fatalf("submission state=%+v", got)
	}
	ver, err := s.contributionStore.getVersion(sub.ID, 2)
	if err != nil {
		t.Fatal(err)
	}
	if ver.Content != first || ver.ReviewedContent != edited || ver.CreatedBy != "admin-player" {
		t.Fatalf("admin audit version did not preserve before/after content: %+v", ver)
	}
}

func TestPublishContribution_when_adminReviewedContentHasInvalidReference_then_rejected(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "原稿", FactionIDs: []string{"f1"}}},
	})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	reviewed, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "审核改稿", FactionIDs: []string{"missing"}}},
	})
	if err := s.publishContribution(sub.ID, "admin", reviewed, ""); err == nil {
		t.Fatal("admin-reviewed content must not bypass final faction/reference validation")
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 0 {
		t.Fatalf("invalid reviewed content was partially published: %d", n)
	}
}

func TestAdminContributionPublish_when_reviewedContentHasWrongType_then_rejected(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, _ := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "原稿", FactionIDs: []string{"f1"}}},
	})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.handleAdminContributionAction("contributionPublish", map[string]any{
		"id": sub.ID, "reviewedContent": []any{"not-a-string"},
	}); err == nil {
		t.Fatal("wrong reviewedContent type must not silently fall back to publishing the original")
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM punishment_tasks WHERE submission_id = ?`, sub.ID); n != 0 {
		t.Fatalf("malformed admin payload published content: %d", n)
	}
}

func TestPublishContribution_when_genderRepublishedAfterUnpublish_then_restoresFormalRow(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, _ := marshalDraft(types.GenderDraft{Label: "狐狸", FactionID: "f1"})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	approved, _ := s.contributionStore.get(sub.ID)
	formalID := approved.PublishedTargetID
	if err := s.unpublishContribution(sub.ID, "admin", "下架"); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM gender_options WHERE id = ?`, formalID); n != 0 {
		t.Fatalf("gender was not unpublished: %d", n)
	}
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindGender, false, sub.ID, raw); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if n := countByQuery(t, s.db, `SELECT COUNT(*) FROM gender_options WHERE id = ?`, formalID); n != 1 {
		t.Fatalf("republish reported success without restoring the formal gender row: %d", n)
	}
}

func TestContributionAdminList_wrapsItemsAndCounts(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	out, err := s.handleAdminContributionAction("contributionList", map[string]any{"status": types.ContributionPending})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("list must be an object for protobuf Struct, got %T", out)
	}
	items, ok := m["items"].([]types.ContributionSubmission)
	if !ok || len(items) != 1 {
		t.Fatalf("items=%v", m["items"])
	}
	counts, ok := m["counts"].(map[string]any)
	if !ok {
		t.Fatalf("counts=%v", m["counts"])
	}
	if counts["pending"] != 1 {
		t.Fatalf("pending count=%v", counts["pending"])
	}
	if items[0].Title != "文案" {
		t.Fatalf("title should fall back to the first variant text, got %q", items[0].Title)
	}
}

func TestValidateContributionRefs_minSeriesSteps(t *testing.T) {
	s := newContributionPublishServer(t)
	s.cfg.PunishmentRandomSettings.MinSeriesSteps = 10
	raw, err := marshalDraft(types.SeriesDraft{
		Name:             "太短",
		TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{{
			InRandomPool: true, Order: 10,
			Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.validateContributionRefs(types.ContributionKindSeries, raw); err != nil {
		t.Fatalf("refs must still pass for a short complete series: %v", err)
	}
	if err := s.validateSeriesMinSteps(raw); err == nil {
		t.Fatal("short series must be rejected at submit/publish")
	}
}

func TestValidateContributionRefs_maxSeriesSteps(t *testing.T) {
	s := newContributionPublishServer(t)
	s.cfg.PunishmentRandomSettings.MaxSeriesSteps = 2
	step := types.StepDraft{
		InRandomPool: true, Order: 10,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	}
	raw, err := marshalDraft(types.SeriesDraft{
		Name:             "太长",
		TargetFactionIDs: []string{"f1"},
		Steps:            []types.StepDraft{step, step, step},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.validateContributionRefs(types.ContributionKindSeries, raw); err != nil {
		t.Fatalf("refs must still pass regardless of step count: %v", err)
	}
	if err := s.validateSeriesMaxSteps(raw); err == nil {
		t.Fatal("series over the configured max steps must be rejected at submit/publish")
	}

	// 不填（<=0）按默认值兜底（与纯防御性的技术上限 maxSeriesSteps 是两码事）；
	// 见 contribution_codec.go 的 effectiveMaxSeriesSteps——这里只验证"配置了合理值时按配置生效"。
	s.cfg.PunishmentRandomSettings.MaxSeriesSteps = 0
	if got := effectiveMaxSeriesSteps(s.cfg); got != defaultMaxSeriesSteps {
		t.Fatalf("unset max steps should fall back to the default %d, got %d", defaultMaxSeriesSteps, got)
	}
	s.cfg.PunishmentRandomSettings.MaxSeriesSteps = maxSeriesSteps + 100
	if got := effectiveMaxSeriesSteps(s.cfg); got != maxSeriesSteps {
		t.Fatalf("configured max steps must be clamped to the hard ceiling %d, got %d", maxSeriesSteps, got)
	}
}

func TestContributionUpdateCommentAndRevertReject_when_taskRejected_then_flows(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}

	// 改批注只对 rejected 生效：pending 状态下应报错。
	if err := s.contributionStore.updateReviewComment(sub.ID, "改早了"); err == nil {
		t.Fatal("updateReviewComment must reject non-rejected submissions")
	}
	if err := s.contributionStore.revertRejection(sub.ID); err == nil {
		t.Fatal("revertRejection must reject non-rejected submissions")
	}

	if err := s.contributionStore.reject(sub.ID, "admin", "不够劲爆"); err != nil {
		t.Fatal(err)
	}
	got, err := s.contributionStore.get(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != types.ContributionRejected || got.ReviewComment != "不够劲爆" {
		t.Fatalf("status=%s comment=%q", got.Status, got.ReviewComment)
	}

	if err := s.contributionStore.updateReviewComment(sub.ID, "换个理由"); err != nil {
		t.Fatal(err)
	}
	got, err = s.contributionStore.get(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != types.ContributionRejected || got.ReviewComment != "换个理由" {
		t.Fatalf("after comment update: status=%s comment=%q", got.Status, got.ReviewComment)
	}
	ver, err := s.contributionStore.getVersion(sub.ID, got.ActiveVersion)
	if err != nil {
		t.Fatal(err)
	}
	if ver.ReviewComment != "换个理由" {
		t.Fatalf("version review comment not kept in sync: %q", ver.ReviewComment)
	}

	if err := s.contributionStore.revertRejection(sub.ID); err != nil {
		t.Fatal(err)
	}
	got, err = s.contributionStore.get(sub.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != types.ContributionPending {
		t.Fatalf("revert must put the submission back into the review queue, status=%s", got.Status)
	}
}

func TestContributionRevertReject_when_genderNameTakenMeanwhile_then_blocked(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, _ := marshalDraft(types.GenderDraft{Label: "狐狸", FactionID: "f1"})
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.reject(sub.ID, "admin", "先驳回"); err != nil {
		t.Fatal(err)
	}
	// 驳回释放了名字占用；期间另一份同名投稿抢先占住。
	other, err := s.contributionStore.saveDraft("p2", "乙", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p2", other.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.revertRejection(sub.ID); err == nil {
		t.Fatal("revert must fail when the name was claimed by someone else in the meantime")
	}
}

func TestListByStatus_when_statusEmpty_then_excludesDraftAndWithdrawn(t *testing.T) {
	s := newContributionPublishServer(t)
	rawA, _ := marshalDraft(types.StepDraft{InRandomPool: true, Order: 20})
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", rawA); err != nil {
		t.Fatal(err)
	}
	rawB, err := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	subB, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", rawB)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", subB.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.withdraw("p1", subB.ID); err != nil {
		t.Fatal(err)
	}
	rawC, err := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	subC, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", rawC)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", subC.ID); err != nil {
		t.Fatal(err)
	}
	list, err := s.contributionStore.listByStatus("", types.ContributionKindTask)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ID != subC.ID {
		t.Fatalf("empty-status listing must skip draft/withdrawn, got %d items", len(list))
	}
}

func TestContributionPendingOverview_when_mixedSubmissions_then_reportsGendersAndCounts(t *testing.T) {
	s := newContributionPublishServer(t)
	genderRaw, _ := marshalDraft(types.GenderDraft{Label: "狐狸", FactionID: "f1"})
	genderSub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindGender, false, "", genderRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", genderSub.ID); err != nil {
		t.Fatal(err)
	}
	taskRaw, err := marshalDraft(types.StepDraft{
		InRandomPool: true, Order: 20,
		Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
	})
	if err != nil {
		t.Fatal(err)
	}
	taskSub, err := s.contributionStore.saveDraft("p2", "乙", types.ContributionKindTask, false, "", taskRaw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p2", taskSub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(taskSub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}

	out, err := s.handleAdminContributionAction("contributionPendingOverview", map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	m, ok := out.(map[string]any)
	if !ok {
		t.Fatalf("overview must be an object for protobuf Struct, got %T", out)
	}
	genders, ok := m["genders"].([]types.ContributionSubmission)
	if !ok || len(genders) != 1 || genders[0].ID != genderSub.ID {
		t.Fatalf("genders=%v", m["genders"])
	}
	counts, ok := m["counts"].(map[string]map[string]int)
	if !ok {
		t.Fatalf("counts=%v (%T)", m["counts"], m["counts"])
	}
	if counts[types.ContributionKindTask][types.ContributionApproved] != 1 {
		t.Fatalf("task approved count=%v", counts[types.ContributionKindTask])
	}
	if counts[types.ContributionKindTask][types.ContributionPending] != 0 {
		t.Fatalf("task pending count=%v", counts[types.ContributionKindTask])
	}
}

func TestPublishContribution_seriesMarksStepBackgroundImagesPublished(t *testing.T) {
	s := newContributionPublishServer(t)
	path := "/uploads/contributions/cover.webp"
	if err := s.contributionStore.recordImage(path, "p1", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := marshalDraft(types.SeriesDraft{
		Name:             "带图系列",
		TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{{
			Variants:          []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
			InRandomPool:      true,
			Order:             10,
			BackgroundImage:   path,
			BackgroundOpacity: 0.22,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	var published int
	var bound string
	if err := s.db.QueryRow(`SELECT published, submission_id FROM contribution_images WHERE path = ?`, path).Scan(&published, &bound); err != nil {
		t.Fatal(err)
	}
	if published != 1 || bound != sub.ID {
		t.Fatalf("series step image published=%d submission=%s", published, bound)
	}

	// 已发布图片不能被后续修订草稿解绑；上传图片属于永久审计材料。
	next, err := marshalDraft(types.SeriesDraft{
		Name:             "带图系列",
		TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{{
			Variants:     []types.TaskVariantDraft{{Text: "改过的文案", FactionIDs: []string{"f1"}}},
			InRandomPool: true,
			Order:        10,
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, sub.ID, next); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT published, submission_id FROM contribution_images WHERE path = ?`, path).Scan(&published, &bound); err != nil {
		t.Fatal(err)
	}
	if published != 1 || bound != sub.ID {
		t.Fatalf("published series image must survive draft rewrite, published=%d submission=%s", published, bound)
	}
	if err := s.contributionStore.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.publishContribution(sub.ID, "admin", "", ""); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT published, submission_id FROM contribution_images WHERE path = ?`, path).Scan(&published, &bound); err != nil {
		t.Fatal(err)
	}
	if published != 1 || bound != sub.ID {
		t.Fatalf("an image omitted by the approved revision must remain as permanent audit material, published=%d submission=%s", published, bound)
	}
	if err := s.unpublishContribution(sub.ID, "admin", "监管下架"); err != nil {
		t.Fatal(err)
	}
	if err := s.db.QueryRow(`SELECT published, submission_id FROM contribution_images WHERE path = ?`, path).Scan(&published, &bound); err != nil {
		t.Fatal(err)
	}
	if published != 1 || bound != sub.ID {
		t.Fatalf("unpublishing must not retire permanent audit image, published=%d submission=%s", published, bound)
	}
}

func TestValidateContributionRefs_genderUnknownFaction(t *testing.T) {
	s := newContributionPublishServer(t)
	raw, err := marshalDraft(types.GenderDraft{Label: "新性别", FactionID: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.validateContributionRefs(types.ContributionKindGender, raw); err == nil {
		t.Fatal("unknown gender faction must be rejected")
	}
	okRaw, err := marshalDraft(types.GenderDraft{Label: "新性别", FactionID: "f1"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.validateContributionRefs(types.ContributionKindGender, okRaw); err != nil {
		t.Fatalf("existing faction should pass: %v", err)
	}
}
