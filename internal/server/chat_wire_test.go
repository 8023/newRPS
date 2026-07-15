package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/types"
)

// chat:new 走动态 RAW（不在 pbconv 的 typed 事件表里），必须原样带上 mentions/seq。
// 若不慎回退到 typed ChatMessage 编码，这两个字段会被丢掉——本测试守住它。
func TestChatNewCarriesMentionsAndSeqOverWire(t *testing.T) {
	msg := types.ChatMessage{
		ID: "m1", PlayerID: "p1", Author: "阿明", Text: "hi @小红 @小李",
		At: 123, Mentions: []string{"pHong", "pLi"}, Seq: 7,
	}
	body, err := pbconv.BuildRawBody("chat:new", msg)
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
	mentions, ok := m["mentions"].([]any)
	if !ok || len(mentions) != 2 || mentions[0] != "pHong" || mentions[1] != "pLi" {
		t.Fatalf("mentions not preserved over wire: %#v", m["mentions"])
	}
	// structpb 数字是 float64
	if seq, ok := m["seq"].(float64); !ok || int(seq) != 7 {
		t.Fatalf("seq not preserved over wire: %#v", m["seq"])
	}
	if m["text"] != "hi @小红 @小李" {
		t.Fatalf("text mangled: %#v", m["text"])
	}
}

// 模拟重启：写入后关闭再重开同目录的 store，数据应仍在（重启不丢聊天）。
func TestChatStoreSurvivesReopen(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if _, err := store.append("", types.ChatMessage{ID: "a", Text: "before restart", At: 1}); err != nil {
		t.Fatalf("append: %v", err)
	}
	if _, err := store.append("roomX", types.ChatMessage{ID: "b", Text: "room msg", At: 2}); err != nil {
		t.Fatalf("append room: %v", err)
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}

	reopened, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer reopened.Close()

	lobby, _, _ := reopened.recent("", 100)
	if len(lobby) != 1 || lobby[0].Text != "before restart" {
		t.Fatalf("lobby not persisted across reopen: %#v", lobby)
	}
	room, _, _ := reopened.recent("roomX", 100)
	if len(room) != 1 || room[0].Text != "room msg" {
		t.Fatalf("room not persisted across reopen: %#v", room)
	}
}
