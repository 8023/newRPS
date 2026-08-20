package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestNormalizeStakeTiers(t *testing.T) {
	fallback := []int{5, 10, 20}
	cases := []struct {
		name string
		in   []int
		want []int
	}{
		{"empty falls back", nil, fallback},
		{"all invalid falls back", []int{0, -3}, fallback},
		{"clamped deduped sorted", []int{30, 15, 15, 0, 60}, []int{15, 30, 60}},
		{"capped at four tiers", []int{1, 2, 3, 4, 5}, []int{1, 2, 3, 4}},
		{"upper clamp 9999", []int{100000}, []int{9999}},
	}
	for _, tc := range cases {
		got := normalizeStakeTiers(tc.in, fallback)
		if len(got) != len(tc.want) {
			t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
		}
		for i := range got {
			if got[i] != tc.want[i] {
				t.Fatalf("%s: got %v want %v", tc.name, got, tc.want)
			}
		}
	}
}

func TestNormalizePunishmentRandomSettingsPreservesExplicitInvalidSeriesBounds(t *testing.T) {
	got := normalizePunishmentRandomSettings(types.PunishmentRandomSettings{
		MinSeriesSteps: contributionSeriesTechnicalMax + 500,
		MaxSeriesSteps: contributionSeriesTechnicalMax + 1000,
	})
	if got.MinSeriesSteps != contributionSeriesTechnicalMax+500 || got.MaxSeriesSteps != contributionSeriesTechnicalMax+1000 {
		t.Fatalf("explicit oversized bounds must survive normalization for validation: min=%d max=%d", got.MinSeriesSteps, got.MaxSeriesSteps)
	}
	got = normalizePunishmentRandomSettings(types.PunishmentRandomSettings{MinSeriesSteps: 30, MaxSeriesSteps: 20})
	if got.MinSeriesSteps != 30 || got.MaxSeriesSteps != 20 {
		t.Fatalf("max below min must survive normalization for validation: min=%d max=%d", got.MinSeriesSteps, got.MaxSeriesSteps)
	}
}

func TestValidateConfigRejectsInvalidSeriesBounds(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.PunishmentRandomSettings.MaxSeriesSteps = contributionSeriesTechnicalMax + 1
	if _, err := ValidateConfig(cfg); err == nil {
		t.Fatal("maxSeriesSteps above the technical ceiling must be rejected")
	}
	cfg.PunishmentRandomSettings.MaxSeriesSteps = 20
	cfg.PunishmentRandomSettings.MinSeriesSteps = 21
	if _, err := ValidateConfig(cfg); err == nil {
		t.Fatal("maxSeriesSteps below minSeriesSteps must be rejected")
	}
}

func TestLoadConfigFromSplitJSON(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.Name == "" {
		t.Fatal("empty site name")
	}
	if len(cfg.PunishmentTags) == 0 {
		t.Fatal("punishment tags incomplete")
	}
	// games.json 白名单必须覆盖全部玩法 id，否则建房列表会静默丢掉条目（斗兽棋曾踩坑）。
	wantGames := map[string]bool{"rps": true, "othello": true, "tictactoe": true, "liarsdice": true, "gomoku": true, "jungle": true, "chess": true}
	gotGames := map[string]bool{}
	for _, g := range cfg.Games {
		gotGames[string(g.ID)] = true
	}
	for id := range wantGames {
		if !gotGames[id] {
			t.Fatalf("config.games missing id %q (whitelist or games.json out of sync)", id)
		}
	}
	if cfg.AccessControl.MaxCreatesPer10Min < 1 {
		t.Fatalf("accessControl maxCreates=%d", cfg.AccessControl.MaxCreatesPer10Min)
	}
	for name, v := range map[string]int{
		"ipBackstopMultiplier":     cfg.AccessControl.IPBackstopMultiplier,
		"ipBackstopMinLimit":       cfg.AccessControl.IPBackstopMinLimit,
		"maxSessionIssuePerIp":     cfg.AccessControl.MaxSessionIssuePerIP,
		"maxOnlinePerIpTotal":      cfg.AccessControl.MaxOnlinePerIPTotal,
		"maxCreatesPerIp":          cfg.AccessControl.MaxCreatesPerIP,
		"maxActiveRoomsPerOwner":   cfg.AccessControl.MaxActiveRoomsPerOwner,
		"maxProofUploadsPerPlayer": cfg.AccessControl.MaxProofUploadsPerPlayer,
	} {
		if v < 1 {
			t.Fatalf("accessControl %s=%d", name, v)
		}
	}
	// 拆分文件应存在
	if _, err := os.Stat(filepath.Join(configDir(), "site.json")); err != nil {
		t.Fatal(err)
	}
}

func TestValidateConfigRejectsInvalidIPBackstopFields(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cases := []func(*types.AppConfig){
		func(c *types.AppConfig) { c.AccessControl.IPBackstopMultiplier = 0 },
		func(c *types.AppConfig) { c.AccessControl.IPBackstopMinLimit = 0 },
		func(c *types.AppConfig) { c.AccessControl.MaxSessionIssuePerIP = 0 },
		func(c *types.AppConfig) { c.AccessControl.MaxOnlinePerIPTotal = 0 },
		func(c *types.AppConfig) { c.AccessControl.MaxCreatesPerIP = 0 },
		func(c *types.AppConfig) { c.AccessControl.MaxActiveRoomsPerOwner = 0 },
		func(c *types.AppConfig) { c.AccessControl.MaxProofUploadsPerPlayer = 0 },
	}
	for i, mutate := range cases {
		broken := cfg
		mutate(&broken)
		if _, err := ValidateConfig(broken); err == nil {
			t.Fatalf("case %d: expected validation error", i)
		}
	}
}

func TestFixGiveawayVoteLimitsPreservesExplicitNegative(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	broken := cfg
	broken.Giveaway.PetLikeVoteValue = -1
	broken.Giveaway.MasterDislikeVoteLimitPerHour = 0
	fixed := fixGiveawayVoteLimits(broken)
	if fixed.Giveaway.PetLikeVoteValue != -1 {
		t.Fatalf("negative pet vote value was silently replaced: %v", fixed.Giveaway.PetLikeVoteValue)
	}
	if fixed.Giveaway.MasterDislikeVoteLimitPerHour != cfg.Giveaway.DislikeVoteLimitPerHour {
		t.Fatalf("zero legacy field should fall back, got %d", fixed.Giveaway.MasterDislikeVoteLimitPerHour)
	}
	if _, err := ValidateConfig(fixed); err == nil {
		t.Fatal("negative pet vote value must still fail validation")
	}
}

func TestSaveConfigDoesNotRewriteGenderJSON(t *testing.T) {
	dir := copyConfigDirForTest(t)
	withRootDir(t, dir)
	gendersPath := filepath.Join(dir, "config", "json", "genders.json")
	before, err := os.ReadFile(gendersPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	cfg.Genders = append([]types.GenderOption(nil), cfg.Genders...)
	if len(cfg.Genders) > 0 {
		cfg.Genders[0].Label = cfg.Genders[0].Label + "改"
	}
	if _, err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(gendersPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Fatal("SaveConfig must not rewrite genders.json")
	}
}

func TestSaveConfigRoundTrip(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	orig := cfg.AccessControl.MaxCreatesPer10Min
	cfg.AccessControl.MaxCreatesPer10Min = orig + 1
	saved, err := SaveConfig(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if saved.AccessControl.MaxCreatesPer10Min != orig+1 {
		t.Fatalf("saved=%d", saved.AccessControl.MaxCreatesPer10Min)
	}
	// restore
	cfg.AccessControl.MaxCreatesPer10Min = orig
	if _, err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}
}

func TestSaveConfigBackfillsMissingGiveawayVoteLimits(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	want := cfg.Giveaway
	legacy := cfg
	legacy.Giveaway.PetLikeVoteLimitPerHour = 0
	legacy.Giveaway.PetLikeVoteValue = 0
	legacy.Giveaway.PetDislikeVoteLimitPerHour = 0
	legacy.Giveaway.PetDislikeVoteValue = 0
	legacy.Giveaway.MasterLikeVoteLimitPerHour = 0
	legacy.Giveaway.MasterLikeVoteValue = 0
	legacy.Giveaway.MasterDislikeVoteLimitPerHour = 0
	legacy.Giveaway.MasterDislikeVoteValue = 0

	saved, err := SaveConfig(legacy)
	if err != nil {
		t.Fatalf("legacy config should still save: %v", err)
	}
	if saved.Giveaway.PetLikeVoteLimitPerHour != want.LikeVoteLimitPerHour ||
		saved.Giveaway.PetLikeVoteValue != want.LikeVoteValue ||
		saved.Giveaway.PetDislikeVoteLimitPerHour != want.DislikeVoteLimitPerHour ||
		saved.Giveaway.PetDislikeVoteValue != want.DislikeVoteValue ||
		saved.Giveaway.MasterLikeVoteLimitPerHour != want.LikeVoteLimitPerHour ||
		saved.Giveaway.MasterLikeVoteValue != want.LikeVoteValue ||
		saved.Giveaway.MasterDislikeVoteLimitPerHour != want.DislikeVoteLimitPerHour ||
		saved.Giveaway.MasterDislikeVoteValue != want.DislikeVoteValue {
		t.Fatalf("missing vote limits were not backfilled: %+v", saved.Giveaway)
	}

	if _, err := SaveConfig(cfg); err != nil {
		t.Fatalf("restore config: %v", err)
	}
}

func TestSaveConfigMigratesLegacyPunishments(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, restoreErr := SaveConfig(cfg); restoreErr != nil {
			t.Errorf("restore config: %v", restoreErr)
		}
	}()

	// 模拟旧管理端只提交 Punishments 数组、新标签字段为空。
	// 任务池已迁 SQLite，SaveConfig 只把旧类型拍平成标签。
	legacy := cfg
	legacy.PunishmentTags = nil
	legacy.Punishments = []types.PunishmentConfig{{
		ID: "truth", Name: "真心话",
		Description: "回答一个问题。",
		Variants: map[string]string{
			"default": "回答一个默认问题。",
			"male":    "回答一个勇气问题。",
			"female":  "回答一个心情问题。",
		},
		Tasks: []types.PunishmentTaskConfig{{
			ID: "legacy-task", Variants: map[string]string{
				"default": "回答一个默认问题。",
				"male":    "回答一个勇气问题。",
				"female":  "回答一个心情问题。",
			},
		}},
		RoomNamePool: &types.RoomNamePool{
			Adjectives: []string{"坦白"}, Subjects: []string{"真心话"}, RoomWords: []string{"小屋"},
		},
		RoomBackgroundImages: []string{},
	}}

	saved, err := SaveConfig(legacy)
	if err != nil {
		t.Fatalf("legacy admin config should save after in-memory migration: %v", err)
	}
	if len(saved.PunishmentTags) != 1 || saved.PunishmentTags[0].ID != "truth" {
		t.Fatalf("legacy punishment should flatten to 1 tag: tags=%+v", saved.PunishmentTags)
	}
	if len(saved.Punishments) != 0 {
		t.Fatalf("legacy Punishments field should be cleared after save: %+v", saved.Punishments)
	}
}
