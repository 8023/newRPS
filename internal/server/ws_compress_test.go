package server

import (
	"testing"

	"github.com/coder/websocket"
)

func TestWsCompressionModeSafariDisabled(t *testing.T) {
	cases := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.0.0 Mobile/15E148 Safari/604.1",
	}
	for _, ua := range cases {
		if got := wsCompressionMode(ua); got != websocket.CompressionDisabled {
			t.Fatalf("Safari/iOS UA should disable compression, got %v for %q", got, ua)
		}
	}
}

func TestWsCompressionModeDesktopChromeEnabled(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	if got := wsCompressionMode(ua); got != websocket.CompressionContextTakeover {
		t.Fatalf("desktop Chrome should use context takeover, got %v", got)
	}
}
