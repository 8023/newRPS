package config

import (
	"os"
	"path/filepath"
	"testing"
)

// copyConfigDirForTest 把当前真实 config/ 目录复制一份到临时目录，供迁移测试改动，
// 不触碰仓库里的真实文件。
func copyConfigDirForTest(t *testing.T) string {
	t.Helper()
	src := configDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dst, "config"), 0o755); err != nil {
		t.Fatal(err)
	}
	dstConfig := filepath.Join(dst, "config")
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(src, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dstConfig, e.Name()), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return dst
}

// withRootDir 临时把包级 rootDir 指向 dir，测试结束后还原，避免污染后续测试。
func withRootDir(t *testing.T, dir string) {
	t.Helper()
	orig := rootDir
	rootDir = dir
	t.Cleanup(func() { rootDir = orig })
}

// TestLoadConfigMigratesLegacyDeployDir 模拟官方升级流程（README「升级服务器且不丢玩家数据」）
// ——旧版 config/ 目录整体保留、只换 bin/dist——升级到本版本后仍是旧文件名/缺新文件。
// announcement-board.json 之前叫 daily-announcement.json 且多出已废弃的 buttonText/version
// 字段，security-disclaimer.json 在旧版本里根本不存在。LoadConfig 必须能在这种目录状态下
// 自愈启动，而不是直接报错退出。
func TestLoadConfigMigratesLegacyDeployDir(t *testing.T) {
	dir := copyConfigDirForTest(t)
	cfgDir := filepath.Join(dir, "config")

	legacy := `{
		"enabled": true,
		"title": "旧版公告标题",
		"content": "旧版公告内容",
		"buttonText": "知道了",
		"version": "2025-01-01"
	}`
	if err := os.Rename(filepath.Join(cfgDir, "announcement-board.json"), filepath.Join(cfgDir, "daily-announcement.json")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "daily-announcement.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(cfgDir, "security-disclaimer.json")); err != nil {
		t.Fatal(err)
	}

	withRootDir(t, dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should self-heal a legacy config dir, got error: %v", err)
	}
	if !cfg.AnnouncementBoard.Enabled || cfg.AnnouncementBoard.Title != "旧版公告标题" || cfg.AnnouncementBoard.Content != "旧版公告内容" {
		t.Fatalf("announcement board not migrated correctly: %+v", cfg.AnnouncementBoard)
	}
	if !cfg.SecurityDisclaimer.Enabled {
		t.Fatalf("security disclaimer should default to enabled when file is missing, got %+v", cfg.SecurityDisclaimer)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "announcement-board.json")); err != nil {
		t.Fatalf("announcement-board.json should have been written: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "daily-announcement.json.bak")); err != nil {
		t.Fatalf("old daily-announcement.json should have been renamed to .bak: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "security-disclaimer.json")); err != nil {
		t.Fatalf("security-disclaimer.json should have been created: %v", err)
	}

	// 再次加载应是幂等的（第二次不应再有 daily-announcement.json 可迁移，也不该报错）。
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("second LoadConfig should be idempotent, got error: %v", err)
	}
}

// TestLoadConfigMigratesLegacyGenderFactionsFile 模拟性别/阵营解耦前的旧版部署：
// gender-factions.json 是"性别嵌套在阵营里"的旧格式（没有 taskGroup），且没有独立的
// genders.json（那时候性别是从阵营展开得到的，不单独存文件）。LoadConfig 必须能在这种
// 目录状态下自愈：拆出扁平的 genders.json，并把 gender-factions.json 改写成新格式。
func TestLoadConfigMigratesLegacyGenderFactionsFile(t *testing.T) {
	dir := copyConfigDirForTest(t)
	cfgDir := filepath.Join(dir, "config")

	legacy := `[
		{
			"textColor": "#225c8d", "backgroundColor": "#dff2ff", "borderColor": "#92cdf2",
			"id": "male_faction", "label": "男性阵营",
			"genders": [
				{"id": "boy", "label": "男生", "factionId": "male_faction"},
				{"id": "male", "label": "男性", "factionId": "male_faction"}
			]
		},
		{
			"textColor": "#6650a4", "backgroundColor": "#eee9ff", "borderColor": "#c7b5ff",
			"id": "femboy_faction", "label": "男娘阵营",
			"genders": [
				{"id": "femboy", "label": "男娘", "factionId": "femboy_faction"}
			]
		}
	]`
	if err := os.WriteFile(filepath.Join(cfgDir, "gender-factions.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(cfgDir, "genders.json")); err != nil {
		t.Fatal(err)
	}

	withRootDir(t, dir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should self-heal a legacy gender-factions.json, got error: %v", err)
	}
	if len(cfg.Genders) != 3 {
		t.Fatalf("expected 3 flattened genders (boy/male/femboy), got %d: %+v", len(cfg.Genders), cfg.Genders)
	}
	genderIDs := map[string]bool{}
	for _, g := range cfg.Genders {
		genderIDs[g.ID] = true
	}
	for _, want := range []string{"boy", "male", "femboy"} {
		if !genderIDs[want] {
			t.Fatalf("expected gender %q to survive migration, got %+v", want, cfg.Genders)
		}
	}
	groupByFaction := map[string]string{}
	for _, f := range cfg.GenderFactions {
		groupByFaction[f.ID] = f.TaskGroup
	}
	if groupByFaction["male_faction"] != "male" {
		t.Fatalf("male_faction taskGroup = %q, want male", groupByFaction["male_faction"])
	}
	if groupByFaction["femboy_faction"] != "male" {
		t.Fatalf("femboy_faction taskGroup = %q, want male", groupByFaction["femboy_faction"])
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "genders.json")); err != nil {
		t.Fatalf("genders.json should have been written: %v", err)
	}

	// 幂等：genders.json 已存在后，第二次加载不应再改写 gender-factions.json。
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("second LoadConfig should be idempotent, got error: %v", err)
	}
}
