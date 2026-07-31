package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

// drainReplyErrorFor 排干 sendCh 里当前已入队的所有消息，返回 Id 匹配 wantID 的那条的 Err
// 字段。sync:full 每次调用会先推一条 FullState 推送（Id=0），再回一条 RPC ack（Id=请求 ID），
// 不能只读一条消息就当成本次调用的回复。
func drainReplyErrorFor(t *testing.T, client *Client, wantID int64) (string, bool) {
	t.Helper()
	for {
		select {
		case data := <-client.sendCh:
			var env wire.Envelope
			if err := proto.Unmarshal(data, &env); err != nil {
				t.Fatalf("failed to decode message: %v", err)
			}
			if env.Id == wantID {
				return env.Err, true
			}
		default:
			return "", false
		}
	}
}

// TestEventHandlerRegistersSyncFullAndPlayerGet sync:full / player:get 之前在 handleWSEvent
// 里被特殊分支提前处理，完全绕过了 eventHandler 表的按事件限流。修复后它们必须和其它事件一样
// 在这张表里登记，拥有真正的 RateLimitOptions（而不是零值）。
func TestEventHandlerRegistersSyncFullAndPlayerGet(t *testing.T) {
	s := &Server{}
	for _, event := range []string{"sync:full", "player:get"} {
		opts, handler := s.eventHandler(event)
		if handler == nil {
			t.Fatalf("%s should have a registered handler", event)
		}
		if opts.Limit <= 0 {
			t.Fatalf("%s should have a positive rate limit, got %+v", event, opts)
		}
	}
}

// TestHandleWSEventThrottlesSyncFull 复现"绕过限流"漏洞的反面：通过 handleWSEvent（而不是
// 直接调用 onSyncFull）反复发送 sync:full，超过配额后必须被拒绝——此前的实现会一路 return，
// 完全跳过 checkRateLimit/checkIPEventBackstop，只受一个 600 次/分钟的粗粒度 IP 桶约束。
func TestHandleWSEventThrottlesSyncFull(t *testing.T) {
	s := newTestServer(t)
	// IP 兜底桶调宽松一些，确保本测试只触发 sync:full 自己的 event:ip:sid 限流分支。
	s.cfg.AccessControl.IPBackstopMultiplier = 50
	s.cfg.AccessControl.IPBackstopMinLimit = 1000
	s.rateLimitBuckets = map[string]*rateLimitBucket{}
	s.clients = map[string]*Client{}

	client := &Client{id: "sock1", sid: "sid1", ipAddress: "1.2.3.4", sendCh: make(chan []byte, 64), rooms: map[string]struct{}{}}
	s.clients[client.id] = client

	opts, _ := s.eventHandler("sync:full")
	blocked := false
	for i := 0; i < opts.Limit+2; i++ {
		reqID := int64(i + 1)
		s.handleWSEvent(client, wsEnvelope{E: "sync:full", ID: reqID, D: map[string]any{"channel": "lobby"}})
		errMsg, ok := drainReplyErrorFor(t, client, reqID)
		if !ok {
			t.Fatalf("expected a reply for request id %d, got none", reqID)
		}
		if errMsg == "操作过于频繁，请稍后再试" {
			blocked = true
			break
		}
		if errMsg != "" {
			t.Fatalf("unexpected error on attempt %d: %q", i, errMsg)
		}
	}
	if !blocked {
		t.Fatalf("sync:full should eventually be rate-limited after exceeding its quota (%d)", opts.Limit)
	}
}
