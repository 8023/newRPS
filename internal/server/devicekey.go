package server

import (
	"crypto/sha256"
	"encoding/hex"
	"regexp"
	"strings"
)

var fingerprintSafe = regexp.MustCompile(`[^a-zA-Z0-9_-]+`)

// normalizeFingerprint 清洗客户端上报的 visitorId；空则 "missing"（与 IP 组合后仍可限流）。
func normalizeFingerprint(fp string) string {
	fp = strings.TrimSpace(fp)
	fp = fingerprintSafe.ReplaceAllString(fp, "")
	if len(fp) > 128 {
		fp = fp[:128]
	}
	if fp == "" {
		return "missing"
	}
	return fp
}

// deviceKey 将出口 IP 与浏览器指纹一起哈希，作为防多开的设备维度。
// 同宿舍/公司共享公网 IP 但指纹不同 → 不同 key，互不抢名额。
func deviceKey(ip, fingerprint string) string {
	ip = strings.TrimSpace(ip)
	if ip == "" {
		ip = "unknown-ip"
	}
	fp := normalizeFingerprint(fingerprint)
	sum := sha256.Sum256([]byte(ip + "\x00" + fp))
	return hex.EncodeToString(sum[:])
}
