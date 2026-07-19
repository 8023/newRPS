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
