package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/pbconv"
)

// server:hello 与心跳 ping 应答都走动态 RAW 载荷（见 ws.go handleWS），带的
// buildId 必须原样过线——前端拿这个字段跟自身构建版本比对，字段丢了或类型变了，
// 提示刷新的功能就会静默失效。
func TestServerHelloCarriesBuildID(t *testing.T) {
	s := &Server{}
	old := BuildID
	BuildID = "abc1234"
	defer func() { BuildID = old }()

	env, err := s.buildRawEnvelope("server:hello", 0, map[string]any{"buildId": BuildID}, "")
	if err != nil {
		t.Fatalf("buildRawEnvelope: %v", err)
	}
	front, err := pbconv.RawBodyToFront(env.RawBody)
	if err != nil {
		t.Fatalf("RawBodyToFront: %v", err)
	}
	m, ok := front.(map[string]any)
	if !ok {
		t.Fatalf("front is %T, want map", front)
	}
	if got, ok := m["buildId"].(string); !ok || got != "abc1234" {
		t.Fatalf("buildId not preserved over wire: %#v", m["buildId"])
	}
}

// ping 应答里的 buildId 是覆盖长连接不断线场景的兜底通道，必须和 server:hello
// 携带同一个值（都读自 BuildID 这一个全局变量），否则前端两条通道会得出矛盾结论。
func TestPingReplyPayloadCarriesBuildID(t *testing.T) {
	old := BuildID
	BuildID = "def5678"
	defer func() { BuildID = old }()

	payload := map[string]any{"t": nowMs(), "buildId": BuildID}
	body, err := pbconv.BuildRawBody("pong", payload)
	if err != nil {
		t.Fatalf("BuildRawBody: %v", err)
	}
	front, err := pbconv.RawBodyToFront(body)
	if err != nil {
		t.Fatalf("RawBodyToFront: %v", err)
	}
	m, ok := front.(map[string]any)
	if !ok {
		t.Fatalf("front is %T, want map", front)
	}
	if got, ok := m["buildId"].(string); !ok || got != "def5678" {
		t.Fatalf("buildId not preserved over wire: %#v", m["buildId"])
	}
}

// 默认值必须是 "dev"，与前端 vite.config.ts 在拿不到 __APP_BUILD_ID__ 时的兜底
// 字面量一致——这是本地开发环境（npm run dev，未走 -ldflags 注入）两端永远判定
// "版本一致、不提示刷新"的前提，改动默认值会让本地开发环境出现误报横幅。
func TestBuildIDDefaultsToDevPlaceholder(t *testing.T) {
	if BuildID != "dev" {
		t.Fatalf("BuildID default changed to %q; front/back 本地开发环境兜底值必须保持 dev 对齐", BuildID)
	}
}
