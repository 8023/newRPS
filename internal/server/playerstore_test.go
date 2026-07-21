package server

import (
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
		Name:          "Alice", GenderID: "male", AvatarURL: "/uploads/avatars/a.webp",
		Stats: types.PublicStats{
			Wins: 3, Losses: 1, Draws: 0, Punishments: 2, RankedPoints: 15, Title: "测试",
			HighestScore: 5000, LowestScore: -6000,
		},
		GameStats: types.GameStats{
			RPS:     types.GameWLD{Wins: 2, Losses: 1},
			Othello: types.GameWLD{Wins: 1},
		},
		NameWarEnabled:     boolPtr(true),
		GiveawayValue:      floatPtr(1.5),
		RankedLastDecayDay: int64Ptr(42),
		CreatedAt:          100, LastSeenAt: 200,
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
	if got.Stats.HighestScore != 5000 || got.Stats.LowestScore != -6000 {
		t.Fatalf("highest/lowest score round trip: %+v", got.Stats)
	}
	if got.RankedLastDecayDay == nil || *got.RankedLastDecayDay != 42 {
		t.Fatalf("ranked last decay day round trip: %v", got.RankedLastDecayDay)
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

// TestPlayersLoadFromDiskRoundTrip 验证 loadPlayersFromDisk 只走 SQLite：写入库的玩家
// 重启（新 Server 实例）后能原样加载回内存。
func TestPlayersLoadFromDiskRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	store := newPlayerStore(db)
	if err := store.upsert(persistedPlayer{
		ID: "pub-m1", PlayerID: "ident-m1", ClaimKey: "ck",
		PlayerSecrets: []string{"secret-1"},
		Name:          "已入库玩家", GenderID: "male",
		Stats:     types.PublicStats{Wins: 5, Losses: 2, Draws: 1, RankedPoints: 10, Title: "称号"},
		GameStats: types.GameStats{RPS: types.GameWLD{Wins: 5, Losses: 2, Draws: 1}},
		CreatedAt: 1, LastSeenAt: 2,
	}); err != nil {
		t.Fatal(err)
	}

	s := &Server{
		players:       map[string]*PlayerState{},
		playerIdToID:  map[string]string{},
		tokenToPlayer: map[string]string{},
		dataDir:       dir,
		playersFile:   filepath.Join(dir, "players.json"),
		playerDB:      store,
	}
	s.loadPlayersFromDisk()

	p := s.players["pub-m1"]
	if p == nil {
		t.Fatal("player not loaded into memory")
	}
	if len(p.PlayerSecrets) != 1 || p.PlayerSecrets[0] != "secret-1" {
		t.Fatalf("secrets after sqlite load: %v", p.PlayerSecrets)
	}
	if p.GameStats.RPS.Wins != 5 {
		t.Fatalf("rps wins = %d, want 5", p.GameStats.RPS.Wins)
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
