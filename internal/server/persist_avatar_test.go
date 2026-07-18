package server

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestAvatarURLPersistedAndReloaded(t *testing.T) {
	s := newTestServer(t)
	s.playerIdToID = map[string]string{}
	s.tokenToPlayer = map[string]string{}

	player := &PlayerState{
		PublicPlayer: types.PublicPlayer{
			ID:        "pub-avatar-1",
			Name:      "头像玩家",
			GenderID:  "male",
			AvatarURL: "/uploads/avatars/demo.webp",
			Stats:     types.PublicStats{Title: "测试称号"},
		},
		Persistent:    true,
		PlayerID:      "identity-1",
		PlayerSecrets: []string{"secret-a"},
		ClaimKey:      "claim-a",
		Token:         "token-a",
		CreatedAt:     1000,
		LastSeenAt:    2000,
	}
	s.players[player.ID] = player
	s.playerIdToID[player.PlayerID] = player.ID

	s.writeSnapshot()

	raw, err := os.ReadFile(s.playersFile)
	if err != nil {
		t.Fatal(err)
	}
	var onDisk []persistedPlayer
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		t.Fatal(err)
	}
	if len(onDisk) != 1 || onDisk[0].AvatarURL != "/uploads/avatars/demo.webp" {
		t.Fatalf("avatarUrl not written to disk: %+v", onDisk)
	}

	// 模拟重启后从磁盘恢复
	s2 := newTestServer(t)
	s2.playersFile = s.playersFile
	s2.playerIdToID = map[string]string{}
	s2.tokenToPlayer = map[string]string{}
	s2.loadPlayersFromDisk()

	restored := s2.players["pub-avatar-1"]
	if restored == nil {
		t.Fatal("player not reloaded")
	}
	if restored.AvatarURL != "/uploads/avatars/demo.webp" {
		t.Fatalf("AvatarURL after reload = %q, want /uploads/avatars/demo.webp", restored.AvatarURL)
	}
}
