package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// copyConfigDirForTest 把当前真实 config/json/ 目录复制一份到临时目录，供迁移测试改动，
// 不触碰仓库里的真实文件。
func copyConfigDirForTest(t *testing.T) string {
	t.Helper()
	src := configDir()
	entries, err := os.ReadDir(src)
	if err != nil {
		t.Fatal(err)
	}
	dst := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dst, "config", "json"), 0o755); err != nil {
		t.Fatal(err)
	}
	dstConfig := filepath.Join(dst, "config", "json")
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

// stripPunishmentFactionIDs 曾用于清空 punishments.json 任务上的阵营引用。
// 任务池已迁出 AppConfig，ValidateConfig 不再校验任务阵营，此函数保留为空操作以兼容调用点。
func stripPunishmentFactionIDs(t *testing.T, cfgDir string) {
	t.Helper()
	_ = cfgDir
}

// withRootDir 临时把包级 rootDir 指向 dir，测试结束后还原，避免污染后续测试。
func withRootDir(t *testing.T, dir string) {
	t.Helper()
	orig := rootDir
	rootDir = dir
	t.Cleanup(func() { rootDir = orig })
}

func TestLoadConfigMigratesLegacyPunishmentsFile(t *testing.T) {
	dir := copyConfigDirForTest(t)
	cfgDir := filepath.Join(dir, "config", "json")
	legacy := `[
  {
    "id": "truth",
    "name": "真心话",
    "description": "回答一个问题。",
    "variants": {
      "default": "回答一个默认问题。",
      "male": "回答一个勇气问题。",
      "female": "回答一个心情问题。"
    },
    "tasks": [{
      "id": "task1",
      "name": "默认任务",
      "variants": {
        "default": "回答一个默认问题。",
        "male": "回答一个勇气问题。",
        "female": "回答一个心情问题。"
      },
      "backgroundOpacity": 0.22
    }],
    "roomNamePool": {
      "adjectives": ["坦白"],
      "subjects": ["真心话"],
      "roomWords": ["小屋"]
    }
  }
]`
	if err := os.WriteFile(filepath.Join(cfgDir, "punishments.json"), []byte(legacy), 0o600); err != nil {
		t.Fatal(err)
	}

	withRootDir(t, dir)
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig should migrate legacy punishment tasks, got error: %v", err)
	}
	if len(cfg.PunishmentTags) != 1 || cfg.PunishmentTags[0].ID != "truth" {
		t.Fatalf("legacy punishment should flatten to 1 tag: tags=%+v", cfg.PunishmentTags)
	}
	// 任务池不再进 AppConfig，但磁盘上仍应写出拍平后的 tasks，供 server 导入 SQLite。
	tasks, _, err := ReadLegacyPunishmentPoolFromDisk()
	if err != nil {
		t.Fatalf("read legacy pool: %v", err)
	}
	if len(tasks) != 3 {
		t.Fatalf("legacy disk tasks should be 3, got %d: %+v", len(tasks), tasks)
	}
	texts := map[string]string{}
	for _, task := range tasks {
		texts[task.ID] = task.Text
		if task.Order != 50 {
			t.Fatalf("legacy task order should use the neutral default, got %d", task.Order)
		}
	}
	if texts["truth_task1_default"] != "回答一个默认问题。" || texts["truth_task1_male"] != "回答一个勇气问题。" || texts["truth_task1_female"] != "回答一个心情问题。" {
		t.Fatalf("legacy task texts were not preserved: %#v", texts)
	}
	if _, err := os.Stat(filepath.Join(cfgDir, "punishments.json.bak")); err != nil {
		t.Fatalf("legacy punishment file should be backed up: %v", err)
	}
	if _, err := LoadConfig(); err != nil {
		t.Fatalf("second LoadConfig should be idempotent after punishment migration: %v", err)
	}
}

func TestLoadConfigMigratesLegacyPunishmentsFromMonolithicConfig(t *testing.T) {
	dir := copyConfigDirForTest(t)
	cfgDir := filepath.Join(dir, "config", "json")
	withRootDir(t, dir)
	cfg, err := readSplitConfig()
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]json.RawMessage
	data, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatal(err)
	}
	legacy := []byte(`[{"id":"truth","name":"真心话","description":"回答一个问题。","variants":{"default":"回答默认问题。","male":"回答勇气问题。","female":"回答心情问题。"},"tasks":[{"id":"task1","name":"默认任务","variants":{"default":"回答默认问题。","male":"回答勇气问题。","female":"回答心情问题。"}}],"roomNamePool":{"adjectives":["坦白"],"subjects":["真心话"],"roomWords":["小屋"]}}]`)
	document["punishments"] = legacy
	// 清除新结构字段，模拟真正的旧单体配置。
	delete(document, "punishmentTags")
	delete(document, "punishmentSeriesSummaries")
	delete(document, "punishmentRandomSettings")
	data, err = json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "default.json"), data, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, name := range splitConfigFiles {
		if err := os.Remove(filepath.Join(cfgDir, name)); err != nil {
			t.Fatal(err)
		}
	}

	loaded, err := LoadConfig()
	if err != nil {
		t.Fatalf("monolithic legacy config should migrate without losing punishment text: %v", err)
	}
	if len(loaded.PunishmentTags) != 1 {
		t.Fatalf("monolithic punishment migration incomplete: tags=%+v", loaded.PunishmentTags)
	}
	// 任务文案落在磁盘 tasks 字段，供后续 SQLite 导入，不进 AppConfig。
	tasks, _, err := ReadLegacyPunishmentPoolFromDisk()
	if err != nil {
		t.Fatalf("read legacy pool: %v", err)
	}
	found := false
	for _, task := range tasks {
		if task.Text == "回答默认问题。" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("monolithic punishment migration lost task text on disk: %+v", tasks)
	}
}

// TestLoadConfigMigratesLegacyDeployDir 模拟官方升级流程（README「升级服务器且不丢玩家数据」）
// ——旧版 config/ 目录整体保留、只换 bin/dist——升级到本版本后仍是旧文件名/缺新文件。
// announcement-board.json 之前叫 daily-announcement.json 且多出已废弃的 buttonText/version
// 字段，security-disclaimer.json 在旧版本里根本不存在。LoadConfig 必须能在这种目录状态下
// 自愈启动，而不是直接报错退出。
func TestLoadConfigMigratesLegacyDeployDir(t *testing.T) {
	dir := copyConfigDirForTest(t)
	cfgDir := filepath.Join(dir, "config", "json")

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
	cfgDir := filepath.Join(dir, "config", "json")

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
	stripPunishmentFactionIDs(t, cfgDir)

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
