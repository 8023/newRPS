package server

import (
	"os"
	"testing"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// TestMigratePunishmentPoolFromJSON_seedsTasksAndSeries 覆盖磁盘遗留 punishments.json
// tasks/seriesTasks 一次性导入 sub_tasks/series：独立任务各自成一行单变体，系列每个 subtask
// 成一步（多份 variants 直接落进该步的 Variants），全部标记 approved、贡献者归
// legacyPunishmentContributorID；表非空时 migrate 幂等（不重复导入）。
func TestMigratePunishmentPoolFromJSON_seedsTasksAndSeries(t *testing.T) {
	dir := t.TempDir()
	origRoot := config.GetRootDir()
	config.SetRootDirForTest(dir)
	t.Cleanup(func() { config.SetRootDirForTest(origRoot) })
	writeLegacyPunishmentsJSON(t, dir, `{
		"tags": [],
		"tasks": [
			{"id": "t1", "text": "原任务", "tagIds": ["truth"], "order": 50}
		],
		"seriesTasks": [
			{
				"id": "s1", "name": "试炼",
				"targetFactionIds": ["female_faction"],
				"roomNamePool": {"subjects": ["b"], "roomWords": ["c"]},
				"roomBackgroundImages": ["/uploads/admin/series.webp"],
				"subtasks": [
					{"variants": [
						{"text": "女变体", "factionIds": ["female_faction"]},
						{"text": "通用变体", "factionIds": []}
					]}
				]
			}
		]
	}`)

	db, err := openDatabase(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{db: db, contributionStore: newContributionStore(db)}
	s.migratePunishmentPoolFromJSONIfNeeded()
	s.reloadPunishmentCaches()

	foundStandalone := false
	foundVariant := false
	for _, task := range s.punishmentTasksCache {
		if task.Text == "原任务" {
			foundStandalone = true
		}
		if task.Text == "女变体" && task.SeriesID != "" {
			foundVariant = true
		}
	}
	if !foundStandalone {
		t.Fatalf("standalone legacy task missing: %#v", s.punishmentTasksCache)
	}
	standaloneID := ""
	for _, task := range s.punishmentTasksCache {
		if task.Text == "原任务" {
			standaloneID = task.ID
		}
	}
	if standaloneID != "t1" {
		t.Fatalf("standalone legacy ID=%q, want t1", standaloneID)
	}
	if !foundVariant {
		t.Fatalf("series step variant missing: %#v", s.punishmentTasksCache)
	}
	if len(s.punishmentSeriesCache) != 1 || s.punishmentSeriesCache[0].Name != "试炼" {
		t.Fatalf("series cache=%#v", s.punishmentSeriesCache)
	}
	if s.punishmentSeriesCache[0].ID != "s1" || len(s.punishmentSeriesCache[0].TargetFactionIDs) != 1 || s.punishmentSeriesCache[0].TargetFactionIDs[0] != "female_faction" {
		t.Fatalf("series identity/targets were not preserved: %#v", s.punishmentSeriesCache[0])
	}
	if s.punishmentSeriesCache[0].RoomNamePool == nil || len(s.punishmentSeriesCache[0].RoomNamePool.Subjects) != 1 || s.punishmentSeriesCache[0].RoomNamePool.Subjects[0] != "b" || len(s.punishmentSeriesCache[0].RoomBackgroundImages) != 1 {
		t.Fatalf("series presentation was not preserved: %#v", s.punishmentSeriesCache[0])
	}

	player := &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1", Name: "小败", FactionID: "female_faction"}, PlayerID: "pid1", Persistent: true}
	s.players = map[string]*PlayerState{player.ID: player}
	seriesID := s.punishmentSeriesCache[0].ID
	room := &RoomState{Settings: types.RoomSettings{PunishmentSource: "series", PunishmentSeriesID: seriesID}}
	r := s.pickSeriesTaskForPlayer(room, player, seriesID, "胜")
	if r == nil || r.TaskText != "女变体" {
		t.Fatalf("pick after migrate: %#v", r)
	}

	// 表非空 → migrate 幂等（不重复导入）。
	n, err := s.contributionStore.tasks.countAll()
	if err != nil {
		t.Fatal(err)
	}
	s.migratePunishmentPoolFromJSONIfNeeded()
	n2, err := s.contributionStore.tasks.countAll()
	if err != nil {
		t.Fatal(err)
	}
	if n2 != n {
		t.Fatalf("idempotent count changed %d -> %d", n, n2)
	}
}

func writeLegacyPunishmentsJSON(t *testing.T, rootDir, body string) {
	t.Helper()
	dir := rootDir + "/config/json"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dir+"/punishments.json", []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}
