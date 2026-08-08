package server

import "testing"

func TestParseUserAgent(t *testing.T) {
	cases := []struct {
		ua                   string
		browser, os, device  string
	}{
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
			"Chrome", "Windows", "desktop",
		},
		{
			"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0",
			"Edge", "Windows", "desktop",
		},
		{
			"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
			"Safari", "macOS", "desktop",
		},
		{
			"Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) CriOS/120.0.6099.119 Mobile/15E148 Safari/604.1",
			"Chrome", "iOS", "mobile",
		},
		{
			"Mozilla/5.0 (iPhone; CPU iPhone OS 16_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 MicroMessenger/8.0.42(0x18002a2a) NetType/WIFI Language/zh_CN",
			"微信", "iOS", "mobile",
		},
		{
			"Mozilla/5.0 (Linux; Android 13; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Mobile Safari/537.36",
			"Chrome", "Android", "mobile",
		},
		{
			"Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.0 Mobile/15E148 Safari/604.1",
			"Safari", "iOS", "tablet",
		},
		{
			"Mozilla/5.0 (X11; Linux x86_64; rv:121.0) Gecko/20100101 Firefox/121.0",
			"Firefox", "Linux", "desktop",
		},
		{
			"",
			"其他", "其他", "desktop",
		},
	}
	for _, c := range cases {
		b, o, d := parseUserAgent(c.ua)
		if b != c.browser || o != c.os || d != c.device {
			t.Errorf("parseUserAgent(%q) = (%q,%q,%q), want (%q,%q,%q)",
				c.ua, b, o, d, c.browser, c.os, c.device)
		}
	}
}
