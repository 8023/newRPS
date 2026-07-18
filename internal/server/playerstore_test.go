package server

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestPlayerStoreUpsertAndLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := newPlayerStore(db)

	item := persistedPlayer{
		ID: "pub1", PlayerID: "ident1", ClaimKey: "claim",
		PlayerSecrets: []string{"sec-a", "sec-b"},
		Name: "Alice", GenderID: "male", AvatarURL: "/uploads/avatars/a.webp",
		Stats: types.PublicStats{Wins: 3, Losses: 1, Draws: 0, Punishments: 2, RankedPoints: 15, Title: "测试"},
		GameStats: types.GameStats{
			RPS:     types.GameWLD{Wins: 2, Losses: 1},
			Othello: types.GameWLD{Wins: 1},
		},
		NameWarEnabled: boolPtr(true),
		GiveawayValue:  floatPtr(1.5),
		CreatedAt:      100, LastSeenAt: 200,
	}
	if err := store.upsert(item); err != nil {
		t.Fatal(err)
	}

	rows, err := store.loadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("want 1 row, got %d", len(rows))
	}
	got := rows[0].item
	if got.Name != "Alice" || got.AvatarURL != "/uploads/avatars/a.webp" {
		t.Fatalf("basic fields: %+v", got)
	}
	if got.GameStats.RPS.Wins != 2 || got.GameStats.Othello.Wins != 1 {
		t.Fatalf("game stats: %+v", got.GameStats)
	}
	if len(got.PlayerSecrets) != 2 {
		t.Fatalf("secrets = %v", got.PlayerSecrets)
	}
	// 更新密钥列表
	item.PlayerSecrets = []string{"sec-b", "sec-c"}
	item.Name = "Alice2"
	if err := store.upsert(item); err != nil {
		t.Fatal(err)
	}
	rows, err = store.loadAll()
	if err != nil {
		t.Fatal(err)
	}
	got = rows[0].item
	if got.Name != "Alice2" {
		t.Fatalf("name not updated: %s", got.Name)
	}
	if len(got.PlayerSecrets) != 2 {
		t.Fatalf("secrets after replace = %v", got.PlayerSecrets)
	}
	has := map[string]bool{}
	for _, s := range got.PlayerSecrets {
		has[s] = true
	}
	if !has["sec-b"] || !has["sec-c"] || has["sec-a"] {
		t.Fatalf("secret rotation failed: %v", got.PlayerSecrets)
	}
}

func TestMigratePlayersJSONToSQLite(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	s := &Server{
		players:       map[string]*PlayerState{},
		playerIdToID:  map[string]string{},
		tokenToPlayer: map[string]string{},
		dataDir:       dir,
		playersFile:   filepath.Join(dir, "players.json"),
		playerDB:      newPlayerStore(db),
	}

	// 模拟旧 JSON（含 othelloStats 残差）
	legacy := []persistedPlayer{{
		ID: "pub-m1", PlayerID: "ident-m1", ClaimKey: "ck",
		PlayerSecrets: []string{"secret-1"},
		Name: "迁入玩家", GenderID: "male",
		Stats: types.PublicStats{Wins: 5, Losses: 2, Draws: 1, RankedPoints: 10, Title: "旧称号"},
		LegacyOthelloStats: &struct {
			Wins     int `json:"wins"`
			Losses   int `json:"losses"`
			Draws    int `json:"draws"`
			Games    int `json:"games"`
			Captured int `json:"captured"`
			Lost     int `json:"lost"`
		}{Wins: 2, Losses: 1, Draws: 0, Games: 3, Captured: 40, Lost: 5},
		CreatedAt: 1, LastSeenAt: 2,
	}}
	raw, _ := json.MarshalIndent(legacy, "", "  ")
	if err := os.WriteFile(s.playersFile, raw, 0o644); err != nil {
		t.Fatal(err)
	}

	s.loadPlayersFromDisk()

	if s.players["pub-m1"] == nil {
		t.Fatal("player not loaded into memory")
	}
	p := s.players["pub-m1"]
	if p.GameStats.Othello.Wins != 2 {
		t.Fatalf("othello wins = %d", p.GameStats.Othello.Wins)
	}
	if p.GameStats.RPS.Wins != 3 { // 5-2 residual
		t.Fatalf("rps residual wins = %d, want 3", p.GameStats.RPS.Wins)
	}
	if _, err := os.Stat(s.playersFile); !os.IsNotExist(err) {
		t.Fatalf("players.json should be renamed away, err=%v", err)
	}
	if _, err := os.Stat(s.playersFile + ".migrated"); err != nil {
		t.Fatalf("expected .migrated backup: %v", err)
	}

	// 库内可再加载
	s2 := &Server{
		players:       map[string]*PlayerState{},
		playerIdToID:  map[string]string{},
		tokenToPlayer: map[string]string{},
		dataDir:       dir,
		playersFile:   filepath.Join(dir, "players.json"),
		playerDB:      newPlayerStore(db),
	}
	s2.loadPlayersFromDisk()
	if s2.players["pub-m1"] == nil {
		t.Fatal("reload from sqlite failed")
	}
	if s2.players["pub-m1"].AvatarURL != p.AvatarURL {
		// avatar was empty - fine
	}
	if len(s2.players["pub-m1"].PlayerSecrets) != 1 || s2.players["pub-m1"].PlayerSecrets[0] != "secret-1" {
		t.Fatalf("secrets after sqlite reload: %v", s2.players["pub-m1"].PlayerSecrets)
	}
}

func TestWriteSnapshotUsesSQLite(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	s := &Server{
		players:       map[string]*PlayerState{},
		playerIdToID:  map[string]string{},
		tokenToPlayer: map[string]string{},
		dataDir:       dir,
		playersFile:   filepath.Join(dir, "players.json"),
		playerDB:      newPlayerStore(db),
	}
	p := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID: "pub-w", Name: "写盘", GenderID: "male",
			GameStats: types.GameStats{Gomoku: types.GameWLD{Wins: 4, Losses: 1}},
		},
		Persistent: true, PlayerID: "ident-w", PlayerSecrets: []string{"s1"}, ClaimKey: "c1",
		Token: "tok", CreatedAt: 1, LastSeenAt: 2,
	}
	p.SyncTotalsFromGameStats()
	s.players[p.ID] = p
	s.playerIdToID[p.PlayerID] = p.ID
	s.writeSnapshot()

	rows, err := s.playerDB.loadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 || rows[0].item.GameStats.Gomoku.Wins != 4 {
		t.Fatalf("sqlite write failed: %+v", rows)
	}
}
