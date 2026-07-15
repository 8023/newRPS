package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestChatStoreAppendRecentOlder(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	// 写入 250 条大厅消息
	for i := 0; i < 250; i++ {
		_, err := store.append("", types.ChatMessage{
			ID: randomID(), PlayerID: "p1", Author: "阿明", Text: "msg", At: int64(i),
		})
		if err != nil {
			t.Fatalf("append %d: %v", i, err)
		}
	}

	// recent 100：应为最新的 100 条（at 150..249），升序，hasMore=true
	recent, hasMore, err := store.recent("", 100)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(recent) != 100 {
		t.Fatalf("recent len = %d, want 100", len(recent))
	}
	if !hasMore {
		t.Fatalf("recent hasMore = false, want true")
	}
	if recent[0].At != 150 || recent[99].At != 249 {
		t.Fatalf("recent range = [%d,%d], want [150,249]", recent[0].At, recent[99].At)
	}
	if recent[0].Seq >= recent[99].Seq {
		t.Fatalf("recent not ascending by seq")
	}

	// older：以最旧一条的 seq 为游标，取更早 100 条（at 50..149）
	older, hasMore2, err := store.older("", recent[0].Seq, 100)
	if err != nil {
		t.Fatalf("older: %v", err)
	}
	if len(older) != 100 {
		t.Fatalf("older len = %d, want 100", len(older))
	}
	if !hasMore2 {
		t.Fatalf("older hasMore = false, want true (still 50 left)")
	}
	if older[0].At != 50 || older[99].At != 149 {
		t.Fatalf("older range = [%d,%d], want [50,149]", older[0].At, older[99].At)
	}

	// 再往前：剩 50 条（at 0..49），hasMore=false
	oldest, hasMore3, err := store.older("", older[0].Seq, 100)
	if err != nil {
		t.Fatalf("oldest: %v", err)
	}
	if len(oldest) != 50 {
		t.Fatalf("oldest len = %d, want 50", len(oldest))
	}
	if hasMore3 {
		t.Fatalf("oldest hasMore = true, want false")
	}
	if oldest[0].At != 0 || oldest[49].At != 49 {
		t.Fatalf("oldest range = [%d,%d], want [0,49]", oldest[0].At, oldest[49].At)
	}
}

func TestChatStoreRoomIsolationAndMentions(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	pub := &types.PublicPlayer{ID: "pX", Name: "小红"}
	if _, err := store.append("roomA", types.ChatMessage{
		ID: "m1", PlayerID: "pX", Author: "小红", Text: "hi @小明", At: 1,
		Mentions: []string{"pMing"}, AuthorPlayer: pub,
	}); err != nil {
		t.Fatalf("append roomA: %v", err)
	}
	if _, err := store.append("roomB", types.ChatMessage{ID: "m2", PlayerID: "pY", Text: "other room", At: 2}); err != nil {
		t.Fatalf("append roomB: %v", err)
	}
	// 大厅表不应看到房间消息
	lobby, _, _ := store.recent("", 100)
	if len(lobby) != 0 {
		t.Fatalf("lobby should be empty, got %d", len(lobby))
	}
	// roomA 只有 1 条，且 mentions/authorPlayer 正确回读
	roomA, _, _ := store.recent("roomA", 100)
	if len(roomA) != 1 {
		t.Fatalf("roomA len = %d, want 1", len(roomA))
	}
	if len(roomA[0].Mentions) != 1 || roomA[0].Mentions[0] != "pMing" {
		t.Fatalf("mentions not round-tripped: %#v", roomA[0].Mentions)
	}
	if roomA[0].AuthorPlayer == nil || roomA[0].AuthorPlayer.Name != "小红" {
		t.Fatalf("authorPlayer not round-tripped: %#v", roomA[0].AuthorPlayer)
	}
	// clearRoom 只清指定房间
	if err := store.clearRoom("roomA"); err != nil {
		t.Fatalf("clearRoom: %v", err)
	}
	roomA2, _, _ := store.recent("roomA", 100)
	roomB2, _, _ := store.recent("roomB", 100)
	if len(roomA2) != 0 {
		t.Fatalf("roomA should be cleared, got %d", len(roomA2))
	}
	if len(roomB2) != 1 {
		t.Fatalf("roomB should survive, got %d", len(roomB2))
	}
}
