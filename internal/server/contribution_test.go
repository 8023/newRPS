package server

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
	_ "github.com/mattn/go-sqlite3"
)

func TestValidateStepDraft_overlappingFactions(t *testing.T) {
	_, err := decodeStepDraft(`{"name":"任务","inRandomPool":true,"order":10,"variants":[{"text":"甲","factionIds":["f1"]},{"text":"乙","factionIds":["f1"]}]}`)
	if err == nil {
		t.Fatal("overlapping factions must be rejected")
	}
}

func TestDecodeStepDraft_order(t *testing.T) {
	for _, order := range []int{1, 50, 99} {
		draft, err := decodeStepDraft(fmt.Sprintf(`{"name":"任务","inRandomPool":true,"order":%d,"variants":[{"text":"文案","factionIds":["f1"]}]}`, order))
		if err != nil {
			t.Fatalf("order=%d should be accepted, got err=%v", order, err)
		}
		if draft.Order != order || !draft.InRandomPool {
			t.Fatalf("order=%d got %+v", order, draft)
		}
	}
	for _, order := range []int{-1, 0, 100} {
		if _, err := decodeStepDraft(fmt.Sprintf(`{"name":"任务","inRandomPool":true,"order":%d,"variants":[{"text":"文案","factionIds":["f1"]}]}`, order)); err == nil {
			t.Fatalf("in-pool order=%d should be rejected", order)
		}
	}
	draft, err := decodeStepDraft(`{"name":"任务","inRandomPool":false,"order":0,"variants":[{"text":"文案","factionIds":["f1"]}]}`)
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

func TestNormalizeLegacyDraft(t *testing.T) {
	step, err := decodeStepDraft(`{"name":"旧任务","text":"旧文案","order":-1,"factionIds":["f1"],"tagIds":["truth"]}`)
	if err != nil {
		t.Fatal(err)
	}
	if step.InRandomPool || len(step.Variants) != 1 || step.Variants[0].Text != "旧文案" {
		t.Fatalf("legacy task: %+v", step)
	}

	raw := `{"name":"旧系列","steps":[{"taskRefs":["a"]},{"taskRefs":["b"]}],"tasks":[{"ref":"a","task":{"name":"一","text":"第一步","order":10,"factionIds":["f1","f2"]}},{"ref":"b","task":{"name":"二","text":"第二步","order":20,"factionIds":["f1"]}}]}`
	draft, skip, err := decodeSeriesDraft(raw)
	if err != nil {
		t.Fatal(err)
	}
	if skip {
		t.Fatal("intersection f1 should keep coverage")
	}
	if len(draft.Steps) != 2 || len(draft.TargetFactionIDs) != 1 || draft.TargetFactionIDs[0] != "f1" {
		t.Fatalf("legacy series: %+v", draft)
	}
	if draft.Steps[0].Variants[0].Text != "第一步" {
		t.Fatalf("step variants: %+v", draft.Steps[0])
	}
}

func TestNormalizeGenderLabel_when_unicodeAndCase_then_folds(t *testing.T) {
	if got := normalizeGenderLabel("  ＡＢＣ  "); got != "abc" {
		t.Fatalf("got %q", got)
	}
	if got := normalizeGenderLabel("Girl\u0000 "); got != "girl" {
		t.Fatalf("control chars: %q", got)
	}
}

func TestDecodeGenderDraft_when_labelContainsControlRune_then_rejected(t *testing.T) {
	if _, err := decodeGenderDraft(`{"label":"猫\u0000咪","factionId":"f1"}`); err == nil {
		t.Fatal("gender label with control rune must not be stored verbatim")
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
		{ID: "g2", Label: " 猫 ", FactionID: "f1"},
	})
	if err == nil {
		t.Fatal("same normalized label in one faction must fail")
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

func TestContributionDraft_when_saveThenSubmitSameName_then_secondFails(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gs := newGenderStore(db)
	_ = gs.replaceAll([]types.GenderFaction{{
		ID: "f1", Label: "阵营", TaskGroup: "default",
		GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
	}}, nil)
	cs := newContributionStore(db)
	raw, _ := marshalDraft(types.GenderDraft{Label: "狐狸", FactionID: "f1"})
	a, err := cs.saveDraft("p1", "甲", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	b, err := cs.saveDraft("p2", "乙", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.submit("p1", a.ID); err != nil {
		t.Fatal(err)
	}
	if err := cs.submit("p2", b.ID); err != errGenderNameTaken {
		t.Fatalf("second submit want taken, got %v", err)
	}
	if err := cs.withdraw("p1", a.ID); err != nil {
		t.Fatal(err)
	}
	if err := cs.submit("p2", b.ID); err != nil {
		t.Fatalf("after withdraw: %v", err)
	}
}

func TestContributionOwnership_when_otherPlayerReads_then_forbidden(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cs := newContributionStore(db)
	raw, _ := marshalDraft(types.TaskDraft{Text: "t", Order: 10})
	sub, err := cs.saveDraft("p1", "甲", types.ContributionKindTask, true, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.getOwned(sub.ID, "p2"); err != errContributionForbidden {
		t.Fatalf("got %v", err)
	}
}

func TestVoteDedup_when_sameRoundSameTarget_then_oneVote(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cs := newContributionStore(db)
	if err := cs.castVote("r1", "e1", "p1", types.ContributionKindTask, "t1", 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := cs.castVote("r1", "e2", "p1", types.ContributionKindTask, "t1", 1, -1); err != errVoteDuplicate {
		t.Fatalf("got %v", err)
	}
	st, err := cs.voteStats(types.ContributionKindTask, "t1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if st.VoteCount != 1 || st.LikeCount != 1 || st.RealRatio == nil || *st.RealRatio != 100 {
		t.Fatalf("%+v", st)
	}
	if err := cs.setVoteOverride(types.ContributionKindTask, "t1", 1, 12, "admin", "note"); err != nil {
		t.Fatal(err)
	}
	st, _ = cs.voteStats(types.ContributionKindTask, "t1", 1)
	if st.VoteCount != 1 || st.DisplayRatio == nil || *st.DisplayRatio != 12 || st.RealRatio == nil || *st.RealRatio != 100 {
		t.Fatalf("override must keep real counts: %+v", st)
	}
}

func TestVoteStats_when_noVotes_then_nilRatio(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cs := newContributionStore(db)
	st, err := cs.voteStats(types.ContributionKindSeries, "s1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if st.RealRatio != nil || st.VoteCount != 0 {
		t.Fatalf("empty must be no-ratio: %+v", st)
	}
}

func TestVoteCardFor_when_playerHasNotVoted_then_hidesSpoilersAtProtocolLayer(t *testing.T) {
	s := newContributionPublishServer(t)
	row := punishmentEventRow{
		ID: "event1", RoundID: "round1", Status: "approved",
		FormalTaskID: "group1", FormalTaskVersion: 1,
		ContributorPlayerID: "author", ContributorNameSnap: "投稿者",
	}
	if err := s.contributionStore.castVote("other-round", "event0", "other", types.ContributionKindTask, "group1", 1, 1); err != nil {
		t.Fatal(err)
	}
	preview, err := s.voteCardFor("viewer", row)
	if err != nil {
		t.Fatal(err)
	}
	if preview.DisplayRatio != nil || preview.VoteCount != 0 || preview.HasVotes || preview.ContributorDisplayName != "" {
		t.Fatalf("pre-vote RPC leaked spoilers: %+v", preview)
	}
	if err := s.contributionStore.castVote("round1", "event1", "viewer", types.ContributionKindTask, "group1", 1, -1); err != nil {
		t.Fatal(err)
	}
	after, err := s.voteCardFor("viewer", row)
	if err != nil {
		t.Fatal(err)
	}
	if after.DisplayRatio == nil || after.VoteCount != 2 || !after.HasVotes || after.ContributorDisplayName != "投稿者" {
		t.Fatalf("post-vote card must reveal aggregate and attribution: %+v", after)
	}
}

func TestContributionVoteTarget_when_seriesIDProvided_then_rejected(t *testing.T) {
	s := newContributionPublishServer(t)
	ok, err := s.contributionStore.voteTargetExists(types.ContributionKindSeries, "series-1", 1)
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("series itself must never be a vote target")
	}
	if err := validateVoteTarget(types.ContributionKindSeries, "series-1"); err == nil {
		t.Fatal("admin vote override must also reject a whole-series target")
	}
	row := punishmentEventRow{FormalTaskID: "step-task-group", FormalTaskVersion: 3, FormalSeriesID: "series-1", FormalSeriesVersion: 3}
	kind, id, version := formalTarget(row)
	if kind != types.ContributionKindTask || id != "step-task-group" || version != 3 {
		t.Fatalf("series step vote target=%s/%s/v%d", kind, id, version)
	}
}

func TestIntFromAny_when_malformedNumber_then_rejected(t *testing.T) {
	for _, value := range []any{1.5, "1", nil} {
		if _, err := intFromAny(value); err == nil {
			t.Fatalf("malformed integer %v (%T) was accepted", value, value)
		}
	}
	if got, err := intFromAny(float64(7)); err != nil || got != 7 {
		t.Fatalf("valid protobuf Struct number: got=%d err=%v", got, err)
	}
}

// TestSubmissionVoteAggregate_when_multipleTaskGroups_then_averagesRatios 覆盖系列聚合点赞率
// = 各步骤（task_group）点赞率的算术平均，未投票的步骤不拖累均值分母；原始计数直接相加。
func TestSubmissionVoteAggregate_when_multipleTaskGroups_then_averagesRatios(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := newPunishmentStore(db)
	cs := newContributionStore(db)

	tasks := []types.PunishmentTaskConfig{
		{ID: "ct_1", Text: "第一步", FactionIDs: []string{}, SubmissionID: "sub_series", TaskGroupID: "tg_1", VoteVersion: 1, BackgroundImages: []string{}},
		{ID: "ct_2", Text: "第二步", FactionIDs: []string{}, SubmissionID: "sub_series", TaskGroupID: "tg_2", VoteVersion: 1, BackgroundImages: []string{}},
		{ID: "ct_3", Text: "第三步·从没被投过票", FactionIDs: []string{}, SubmissionID: "sub_series", TaskGroupID: "tg_3", VoteVersion: 1, BackgroundImages: []string{}},
	}
	if err := ps.replaceTasks(tasks); err != nil {
		t.Fatal(err)
	}

	// tg_1：2 票全部点赞 → 100%。
	if err := cs.castVote("r1", "e1", "p1", types.ContributionKindTask, "tg_1", 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := cs.castVote("r2", "e2", "p2", types.ContributionKindTask, "tg_1", 1, 1); err != nil {
		t.Fatal(err)
	}
	// tg_2：1 赞 1 踩 → 50%。
	if err := cs.castVote("r3", "e3", "p1", types.ContributionKindTask, "tg_2", 1, 1); err != nil {
		t.Fatal(err)
	}
	if err := cs.castVote("r4", "e4", "p2", types.ContributionKindTask, "tg_2", 1, -1); err != nil {
		t.Fatal(err)
	}
	// tg_3：从没被投过票——不应拖累均值分母。

	st, err := cs.submissionVoteAggregate("sub_series", 1)
	if err != nil {
		t.Fatal(err)
	}
	if st.LikeCount != 3 || st.DownCount != 1 || st.VoteCount != 4 {
		t.Fatalf("raw counts should sum across groups, got %+v", st)
	}
	if st.RealRatio == nil || *st.RealRatio != 75 {
		t.Fatalf("average of 100%% and 50%% (tg_3 excluded, no votes) should be 75%%, got %+v", st.RealRatio)
	}

	// 覆盖值：给 tg_2 设一个管理员覆盖比例，均值应该跟着覆盖值走，但 RealRatio 不变。
	if err := cs.setVoteOverride(types.ContributionKindTask, "tg_2", 1, 90, "admin", "note"); err != nil {
		t.Fatal(err)
	}
	st2, err := cs.submissionVoteAggregate("sub_series", 1)
	if err != nil {
		t.Fatal(err)
	}
	if st2.RealRatio == nil || *st2.RealRatio != 75 {
		t.Fatalf("RealRatio must ignore overrides, got %+v", st2.RealRatio)
	}
	if st2.DisplayRatio == nil || *st2.DisplayRatio != 95 {
		t.Fatalf("DisplayRatio should average tg_1's 100%% with tg_2's overridden 90%% = 95%%, got %+v", st2.DisplayRatio)
	}
}

func TestSchemaV33_when_oldPunishmentTables_then_addsColumns(t *testing.T) {
	dir := t.TempDir()
	path := dir + "/database.db"
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE schema_version (version INTEGER NOT NULL)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (32)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE punishment_tasks (id TEXT PRIMARY KEY, name TEXT, text TEXT, tag_ids TEXT, faction_ids TEXT, difficulty_order INTEGER, background_images TEXT, background_opacity REAL, sort_index INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE punishment_series (id TEXT PRIMARY KEY, name TEXT, room_name_pool TEXT, room_background_images TEXT, steps TEXT, sort_index INTEGER)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE punishment_events (id TEXT PRIMARY KEY, room_id TEXT, task_source TEXT, publisher_id TEXT, publisher_name TEXT, target_id TEXT, target_name TEXT, task_text TEXT, task_at INTEGER, proof_text TEXT, image_file TEXT, proof_at INTEGER, status TEXT, redo_id TEXT)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO punishment_tasks (id, name, text, tag_ids, faction_ids, difficulty_order, background_images, background_opacity, sort_index) VALUES ('legacy1', '旧任务', '文案', '[]', '[]', 10, '[]', 0.22, 0)`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()
	db, err = openDatabase(dir)
	if err != nil {
		t.Fatalf("upgrade: %v", err)
	}
	defer db.Close()
	ok, err := tableHasColumn(db, "punishment_events", "round_id")
	if err != nil || !ok {
		t.Fatalf("round_id missing: %v", err)
	}
	ok, err = tableHasColumn(db, "punishment_tasks", "contributor_player_id")
	if err != nil || !ok {
		t.Fatalf("task contributor missing: %v", err)
	}
	ok, err = tableHasColumn(db, "punishment_series", "target_faction_ids")
	if err != nil || !ok {
		t.Fatalf("target_faction_ids missing after v34: %v", err)
	}
	ok, err = tableHasColumn(db, "punishment_tasks", "task_group_id")
	if err != nil || !ok {
		t.Fatalf("task_group_id missing after v35: %v", err)
	}
	var groupID string
	if err := db.QueryRow(`SELECT task_group_id FROM punishment_tasks WHERE id = 'legacy1'`).Scan(&groupID); err != nil {
		t.Fatal(err)
	}
	if groupID != "legacy1" {
		t.Fatalf("legacy task without submission_id must backfill task_group_id to its own id, got %q", groupID)
	}
	for _, index := range []string{
		"idx_punishment_tasks_submission_vote_group",
		"idx_punishment_tasks_pool_created",
		"idx_punishment_series_created",
		"idx_punishment_events_formal_task",
	} {
		if n := countByQuery(t, db, `SELECT COUNT(*) FROM sqlite_master WHERE type = 'index' AND name = ?`, index); n != 1 {
			t.Fatalf("migration did not create %s", index)
		}
	}
}

func TestContributionDraft_when_incomplete_then_saveOkSubmitRejected(t *testing.T) {
	s := newContributionPublishServer(t)
	s.cfg.PunishmentRandomSettings.MinSeriesSteps = 10
	raw, err := marshalDraft(types.StepDraft{
		InRandomPool: true,
		Order:        50,
	})
	if err != nil {
		t.Fatal(err)
	}
	sub, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindTask, false, "", raw)
	if err != nil {
		t.Fatalf("incomplete task draft should save: %v", err)
	}
	if _, err := s.contributionStore.validateOwnedDraft("p1", types.ContributionKindTask, raw); err == nil {
		t.Fatal("incomplete task (no variant text yet) must not pass submit validation")
	}

	seriesRaw, err := marshalDraft(types.SeriesDraft{
		Name:             "写一半",
		TargetFactionIDs: []string{"f1"},
		Steps: []types.StepDraft{{
			InRandomPool: true, Order: 10,
			Variants: []types.TaskVariantDraft{{Text: "文案", FactionIDs: []string{"f1"}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := s.contributionStore.saveDraft("p1", "甲", types.ContributionKindSeries, false, "", seriesRaw); err != nil {
		t.Fatalf("short series draft should save: %v", err)
	}
	if err := s.validateSeriesMinSteps(seriesRaw); err == nil {
		t.Fatal("short series must fail min-step check")
	}
	if sub.Status != types.ContributionDraft {
		t.Fatalf("status=%s", sub.Status)
	}
}

func TestContributionDraft_when_invalidKind_then_rejected(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cs := newContributionStore(db)
	if _, err := cs.saveDraft("p1", "甲", "unknown", false, "", `{}`); err == nil {
		t.Fatal("unknown contribution kind must be rejected")
	}
}

func TestContributionImage_when_otherPlayerUsesPath_then_rejected(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	cs := newContributionStore(db)
	path := "/uploads/contributions/a.webp"
	if err := cs.recordImage(path, "owner", ""); err != nil {
		t.Fatal(err)
	}
	raw, err := marshalDraft(types.TaskDraft{Text: "t", Order: 10, BackgroundImage: path})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := cs.saveDraft("other", "乙", types.ContributionKindTask, false, "", raw); err == nil {
		t.Fatal("a player must not use another player's image")
	}
}

func TestContributionGenderRevision_when_sameFormalName_then_submitSucceeds(t *testing.T) {
	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	gs := newGenderStore(db)
	if err := gs.replaceAll([]types.GenderFaction{{
		ID: "f1", Label: "阵营", TaskGroup: "default",
		GenderColors: types.GenderColors{TextColor: "#111111", BackgroundColor: "#222222", BorderColor: "#333333"},
	}}, nil); err != nil {
		t.Fatal(err)
	}
	cs := newContributionStore(db)
	raw, _ := marshalDraft(types.GenderDraft{Label: "狐狸", FactionID: "f1"})
	sub, err := cs.saveDraft("p1", "甲", types.ContributionKindGender, false, "", raw)
	if err != nil {
		t.Fatal(err)
	}
	if err := cs.submit("p1", sub.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO gender_options (id,label,normalized_label,faction_id) VALUES ('g1','狐狸','狐狸','f1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE contribution_submissions SET status='approved', published_target_id='g1', published_version=1 WHERE id=?`, sub.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := cs.saveDraft("p1", "甲", types.ContributionKindGender, false, sub.ID, raw); err != nil {
		t.Fatal(err)
	}
	if err := cs.submit("p1", sub.ID); err != nil {
		t.Fatalf("revision may retain its own formal name: %v", err)
	}
}
