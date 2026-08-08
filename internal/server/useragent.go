package server

import "strings"

// parseUserAgent 粗粒度解析 UA：browser / os / deviceType。
// 刻意不存版本号（避免指纹级信息）。风格照搬 wsCompressionMode 的 strings.Contains。
func parseUserAgent(ua string) (browser, osName, deviceType string) {
	l := strings.ToLower(ua)
	if l == "" {
		return "其他", "其他", "desktop"
	}

	// browser：顺序重要（Edge/微信 会带 Chrome 关键字）
	switch {
	case strings.Contains(l, "micromessenger"):
		browser = "微信"
	case strings.Contains(l, "mqqbrowser") || strings.Contains(l, "qqbrowser"):
		browser = "QQ浏览器"
	case strings.Contains(l, "ucbrowser") || strings.Contains(l, "ubrowser"):
		browser = "UC"
	case strings.Contains(l, "edg/") || strings.Contains(l, "edgios/") || strings.Contains(l, "edga/"):
		browser = "Edge"
	case strings.Contains(l, "fxios/") || strings.Contains(l, "firefox/"):
		browser = "Firefox"
	case strings.Contains(l, "crios/") || (strings.Contains(l, "chrome/") && !strings.Contains(l, "edg")):
		browser = "Chrome"
	case strings.Contains(l, "safari/") && !strings.Contains(l, "chrome") && !strings.Contains(l, "crios") && !strings.Contains(l, "chromium"):
		browser = "Safari"
	default:
		browser = "其他"
	}

	// os
	switch {
	case strings.Contains(l, "windows"):
		osName = "Windows"
	case strings.Contains(l, "iphone") || strings.Contains(l, "ipad") || strings.Contains(l, "ipod") ||
		(strings.Contains(l, "mac os") && strings.Contains(l, "mobile")) || strings.Contains(l, "cpu iphone os"):
		osName = "iOS"
	case strings.Contains(l, "android"):
		osName = "Android"
	case strings.Contains(l, "mac os") || strings.Contains(l, "macintosh"):
		osName = "macOS"
	case strings.Contains(l, "linux"):
		osName = "Linux"
	default:
		osName = "其他"
	}

	// deviceType
	switch {
	case strings.Contains(l, "ipad") || strings.Contains(l, "tablet") || strings.Contains(l, "playbook") ||
		(strings.Contains(l, "android") && !strings.Contains(l, "mobile")):
		deviceType = "tablet"
	case strings.Contains(l, "mobi") || strings.Contains(l, "iphone") || strings.Contains(l, "ipod") ||
		strings.Contains(l, "android") || strings.Contains(l, "phone"):
		deviceType = "mobile"
	default:
		deviceType = "desktop"
	}
	return browser, osName, deviceType
}
