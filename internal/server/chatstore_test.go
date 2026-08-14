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
		_, err := store.append("", "", types.ChatMessage{
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

func TestChatSearchAuthorMatchesNameOnly(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if _, err := store.append("", "", types.ChatMessage{
		ID: "m1", PlayerID: "p1", Author: "男生 - 蓝白新人 - 审核员", Text: "hi", At: 1,
	}); err != nil {
		t.Fatalf("append displayName: %v", err)
	}
	if _, err := store.append("", "", types.ChatMessage{
		ID: "m2", PlayerID: "p2", Author: "小红", Text: "yo", At: 2,
	}); err != nil {
		t.Fatalf("append plain: %v", err)
	}
	if _, err := store.append("", "", types.ChatMessage{
		ID: "m3", PlayerID: "p3", Author: "女生 - 母鸡段位 - 张三", Text: "old", At: 3,
	}); err != nil {
		t.Fatalf("append titled: %v", err)
	}

	mustSearch := func(author string) []adminChatMessage {
		t.Helper()
		out, _, err := store.search(chatSearchQuery{Author: author})
		if err != nil {
			t.Fatalf("search %q: %v", author, err)
		}
		return out
	}
	ids := func(rows []adminChatMessage) []string {
		got := make([]string, len(rows))
		for i, r := range rows {
			got[i] = r.ID
		}
		return got
	}

	if rows := mustSearch("男生"); len(rows) != 0 {
		t.Fatalf("gender label should not match, got %v", ids(rows))
	}
	if rows := mustSearch("蓝白新人"); len(rows) != 0 {
		t.Fatalf("title should not match, got %v", ids(rows))
	}
	if rows := mustSearch("母鸡"); len(rows) != 0 {
		t.Fatalf("title fragment should not match, got %v", ids(rows))
	}
	if rows := mustSearch("审核员"); len(rows) != 1 || rows[0].ID != "m1" {
		t.Fatalf("nickname snapshot = %v, want [m1]", ids(rows))
	}
	if rows := mustSearch("小红"); len(rows) != 1 || rows[0].ID != "m2" {
		t.Fatalf("plain name = %v, want [m2]", ids(rows))
	}

	if _, err := store.db.Exec(`INSERT INTO players (id, player_id, name) VALUES ('p3', 'p3', '李四')`); err != nil {
		t.Fatalf("insert player: %v", err)
	}
	if rows := mustSearch("李四"); len(rows) != 1 || rows[0].ID != "m3" {
		t.Fatalf("current players.name = %v, want [m3]", ids(rows))
	}
	if rows := mustSearch("张三"); len(rows) != 1 || rows[0].ID != "m3" {
		t.Fatalf("historical nickname snapshot = %v, want [m3]", ids(rows))
	}
}

func TestChatSearchTreatsLikeWildcardsAsLiterals(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	for _, row := range []struct {
		roomID, roomName, id, text string
	}{
		{"room-percent", "普通房间", "percent", "进度 100%"},
		{"room-plain", "普通房间", "plain", "没有符号"},
		{"room-underscore", "房_一", "underscore", "下划线房间"},
		{"room-letter", "房A一", "letter", "字母房间"},
	} {
		if _, err := store.append(row.roomID, row.roomName, types.ChatMessage{ID: row.id, Text: row.text, At: 1}); err != nil {
			t.Fatalf("append %s: %v", row.id, err)
		}
	}

	byText, _, err := store.search(chatSearchQuery{Text: "%"})
	if err != nil {
		t.Fatalf("search text: %v", err)
	}
	if len(byText) != 1 || byText[0].ID != "percent" {
		t.Fatalf("search literal %% = %#v, want percent only", byText)
	}
	byRoom, _, err := store.search(chatSearchQuery{Room: "_"})
	if err != nil {
		t.Fatalf("search room: %v", err)
	}
	if len(byRoom) != 1 || byRoom[0].ID != "underscore" {
		t.Fatalf("search literal _ = %#v, want underscore only", byRoom)
	}
}

func TestChatStoreRoomIsolationAndMentions(t *testing.T) {
	dir := t.TempDir()
	store, err := openChatStore(dir)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer store.Close()

	if _, err := store.append("roomA", "房间A", types.ChatMessage{
		ID: "m1", PlayerID: "pX", Author: "小红", Text: "hi @小明", At: 1,
		Mentions: []string{"pMing"},
	}); err != nil {
		t.Fatalf("append roomA: %v", err)
	}
	if _, err := store.append("roomB", "房间B", types.ChatMessage{ID: "m2", PlayerID: "pY", Text: "other room", At: 2}); err != nil {
		t.Fatalf("append roomB: %v", err)
	}
	// 大厅表不应看到房间消息
	lobby, _, _ := store.recent("", 100)
	if len(lobby) != 0 {
		t.Fatalf("lobby should be empty, got %d", len(lobby))
	}
	// roomA 只有 1 条，且 mentions 正确回读
	roomA, _, _ := store.recent("roomA", 100)
	if len(roomA) != 1 {
		t.Fatalf("roomA len = %d, want 1", len(roomA))
	}
	if len(roomA[0].Mentions) != 1 || roomA[0].Mentions[0] != "pMing" {
		t.Fatalf("mentions not round-tripped: %#v", roomA[0].Mentions)
	}
	// 软删除只影响被引用的那一条，且限定 roomId，不会误伤同名 id 的其它房间
	if n, err := store.setDeleted([]chatMessageRef{{RoomID: "roomA", ID: "m1"}}, true, 999); err != nil {
		t.Fatalf("setDeleted: %v", err)
	} else if n != 1 {
		t.Fatalf("setDeleted affected = %d, want 1", n)
	}
	roomA2, _, _ := store.recent("roomA", 100)
	roomB2, _, _ := store.recent("roomB", 100)
	if len(roomA2) != 0 {
		t.Fatalf("roomA should be empty after soft delete, got %d", len(roomA2))
	}
	if len(roomB2) != 1 {
		t.Fatalf("roomB should survive, got %d", len(roomB2))
	}

	// search(IncludeDeleted) 仍能看到软删除的那一条，且带上 Deleted/RoomName
	deleted, _, err := store.search(chatSearchQuery{IncludeDeleted: true, Author: "小红"})
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(deleted) != 1 || !deleted[0].Deleted || deleted[0].RoomName != "房间A" {
		t.Fatalf("search deleted result = %#v", deleted)
	}

	// 恢复后重新出现在普通历史里
	if n, err := store.setDeleted([]chatMessageRef{{RoomID: "roomA", ID: "m1"}}, false, 0); err != nil {
		t.Fatalf("restore: %v", err)
	} else if n != 1 {
		t.Fatalf("restore affected = %d, want 1", n)
	}
	roomA3, _, _ := store.recent("roomA", 100)
	if len(roomA3) != 1 {
		t.Fatalf("roomA should be restored, got %d", len(roomA3))
	}
}
