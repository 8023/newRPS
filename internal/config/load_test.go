package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestLoadConfigFromSplitJSON(t *testing.T) {
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Site.Name == "" {
		t.Fatal("empty site name")
	}
	if len(cfg.Punishments) == 0 || len(cfg.Punishments[0].Tasks) == 0 {
		t.Fatal("punishments incomplete")
	}
	// games.json 白名单必须覆盖全部玩法 id，否则建房列表会静默丢掉条目（斗兽棋曾踩坑）。
	wantGames := map[string]bool{"rps": true, "othello": true, "tictactoe": true, "liarsdice": true, "gomoku": true, "jungle": true}
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
