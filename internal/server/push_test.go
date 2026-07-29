package server

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestShouldPush(t *testing.T) {
	on := boolPtr(true)
	off := boolPtr(false)
	connected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: true}}
	disconnected := &PlayerState{PublicPlayer: types.PublicPlayer{Connected: false}}

	cases := []struct {
		name   string
		player *PlayerState
		pref   *bool
		want   bool
	}{
		{"nil player", nil, on, false},
		{"connected, pref on", connected, on, true},
		{"disconnected, pref on", disconnected, on, true},
		{"disconnected, pref off", disconnected, off, false},
		{"disconnected, pref nil", disconnected, nil, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := shouldPush(c.player, c.pref); got != c.want {
				t.Errorf("shouldPush() = %v, want %v", got, c.want)
			}
		})
	}
}

func TestLoadOrGenerateVAPIDKeysReportsPersistenceFailure(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	root := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(root, []byte("x"), 0o600); err != nil {
		t.Fatalf("seed blocking file: %v", err)
	}

	keys, err := loadOrGenerateVAPIDKeys(root)
	if err == nil {
		t.Fatalf("expected persistence error, got keys public=%t private=%t", keys.PublicKey != "", keys.PrivateKey != "")
	}
}

func TestLoadOrGenerateVAPIDKeysRejectsPartialEnvironment(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "public-only")
	t.Setenv("VAPID_PRIVATE_KEY", "")

	if _, err := loadOrGenerateVAPIDKeys(t.TempDir()); err == nil {
		t.Fatal("expected an error for a partial VAPID environment configuration")
	}
}

func TestLoadOrGenerateVAPIDKeysRejectsCorruptPersistedKeys(t *testing.T) {
	t.Setenv("VAPID_PUBLIC_KEY", "")
	t.Setenv("VAPID_PRIVATE_KEY", "")
	root := t.TempDir()
	workDir := filepath.Join(root, "work")
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		t.Fatalf("create work directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(workDir, "vapid.json"), []byte("{"), 0o600); err != nil {
		t.Fatalf("write corrupt key file: %v", err)
	}

	if _, err := loadOrGenerateVAPIDKeys(root); err == nil {
		t.Fatal("expected an error for corrupt persisted VAPID keys")
	}
}
