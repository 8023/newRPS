package server

import (
	"encoding/base64"
	"testing"
)

// TestParseWSAuthProtocolsExtractsTokenAndFingerprint 校验 Sec-WebSocket-Protocol 头承载
// token/指纹的新方案：之前 token 直接拼在 /ws?token= 查询串里，会随完整 URL 一起被反向代理
// 的访问日志记录下来。
func TestParseWSAuthProtocolsExtractsTokenAndFingerprint(t *testing.T) {
	fp := base64.RawURLEncoding.EncodeToString([]byte("visitor-abc123"))
	token, fingerprint := parseWSAuthProtocols("auth.sid123.456.sig, fp." + fp)
	if token != "sid123.456.sig" {
		t.Fatalf("token = %q, want %q", token, "sid123.456.sig")
	}
	if fingerprint != "visitor-abc123" {
		t.Fatalf("fingerprint = %q, want %q", fingerprint, "visitor-abc123")
	}
}

func TestParseWSAuthProtocolsHandlesEmptyHeader(t *testing.T) {
	token, fingerprint := parseWSAuthProtocols("")
	if token != "" || fingerprint != "" {
		t.Fatalf("expected empty token/fingerprint for empty header, got %q/%q", token, fingerprint)
	}
}

func TestParseWSAuthProtocolsIgnoresMalformedFingerprint(t *testing.T) {
	// fp. 后面不是合法 base64url：解码失败时 fingerprint 应保持为空，而不是把垃圾数据带下去。
	token, fingerprint := parseWSAuthProtocols("auth.tok, fp.not!!valid==base64")
	if token != "tok" {
		t.Fatalf("token = %q, want %q", token, "tok")
	}
	if fingerprint != "" {
		t.Fatalf("expected empty fingerprint for malformed base64, got %q", fingerprint)
	}
}

func TestParseWSAuthProtocolsOnlyTokenNoFingerprint(t *testing.T) {
	token, fingerprint := parseWSAuthProtocols("auth.onlytoken")
	if token != "onlytoken" {
		t.Fatalf("token = %q, want %q", token, "onlytoken")
	}
	if fingerprint != "" {
		t.Fatalf("expected no fingerprint, got %q", fingerprint)
	}
}
