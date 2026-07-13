package config

import (
	"os"
	"path/filepath"
	"testing"
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
	// 拆分文件应存在
	if _, err := os.Stat(filepath.Join(configDir(), "site.json")); err != nil {
		t.Fatal(err)
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
