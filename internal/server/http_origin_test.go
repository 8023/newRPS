package server

import "testing"

// TestIsAllowedOriginRejectsEmpty 空 Origin 不再直接放行——之前的实现对空 Origin 无条件
// 放行，等价于让非浏览器脚本绕过来源校验。requireTrustedOrigin 已经对 GET/HEAD/OPTIONS
// 跳过这层检查，这里只需确认非安全方法（POST 等）下空 Origin 被拒绝。
func TestIsAllowedOriginRejectsEmpty(t *testing.T) {
	s := &Server{host: "rps.rbq.io"}
	if s.isAllowedOrigin("", "rps.rbq.io") {
		t.Fatal("empty origin should not be allowed")
	}
}

// TestIsAllowedOriginRequiresSchemeMatchWhenConfigured ALLOWED_ORIGINS 里显式带 scheme 的
// 配置项必须要求 Origin 的 scheme 一致，避免 https 配置被 http 复用。
// 请求 Host 用一个和 Origin 域名不同的值（反代场景下常见：内网 Host 与公网域名不一致），
// 确保只走 ALLOWED_ORIGINS 匹配分支，不被 sameHostOrLocalDev 的 Host 兜底分支掩盖。
func TestIsAllowedOriginRequiresSchemeMatchWhenConfigured(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "https://rps.rbq.io")
	s := &Server{host: "0.0.0.0"}
	if s.isAllowedOrigin("http://rps.rbq.io", "127.0.0.1:9988") {
		t.Fatal("http origin should not match an https-only allow-list entry")
	}
	if !s.isAllowedOrigin("https://rps.rbq.io", "127.0.0.1:9988") {
		t.Fatal("matching scheme should be allowed")
	}
}

// TestIsAllowedOriginBareDomainIgnoresScheme 裸域名（未带 scheme）配置项按设计仍不限制协议。
func TestIsAllowedOriginBareDomainIgnoresScheme(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "rps.rbq.io")
	s := &Server{host: "0.0.0.0"}
	if !s.isAllowedOrigin("http://rps.rbq.io", "127.0.0.1:9988") {
		t.Fatal("bare domain entry should match any scheme")
	}
	if !s.isAllowedOrigin("https://rps.rbq.io", "127.0.0.1:9988") {
		t.Fatal("bare domain entry should match any scheme")
	}
}

// TestSameHostOrLocalDevRequiresBothSidesLocal 生产环境（Host 是公网域名）不应该因为请求
// 声称 Origin=http://localhost 就被放行——本地开发豁免只在服务端自己也跑在本机时才成立。
func TestSameHostOrLocalDevRequiresBothSidesLocal(t *testing.T) {
	s := &Server{host: "0.0.0.0"}
	if s.sameHostOrLocalDev("http://localhost:3000", "rps.rbq.io") {
		t.Fatal("localhost origin should not bypass check against a production Host")
	}
	if !s.sameHostOrLocalDev("http://localhost:5173", "localhost:9988") {
		t.Fatal("local dev (both sides local) should still be allowed")
	}
	if !s.sameHostOrLocalDev("http://127.0.0.1:5173", "localhost:9988") {
		t.Fatal("localhost/127.0.0.1 aliases should remain interchangeable in local dev")
	}
}

// TestSameHostOrLocalDevMatchesRequestHost Origin 与 Host 同名（忽略端口/协议）应放行。
func TestSameHostOrLocalDevMatchesRequestHost(t *testing.T) {
	s := &Server{host: "0.0.0.0"}
	if !s.sameHostOrLocalDev("https://rps.rbq.io", "rps.rbq.io:443") {
		t.Fatal("origin host matching request host should be allowed")
	}
	if s.sameHostOrLocalDev("https://evil.example", "rps.rbq.io:443") {
		t.Fatal("mismatched origin host should be rejected")
	}
}
