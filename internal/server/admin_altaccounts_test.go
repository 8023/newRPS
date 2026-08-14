package server

import (
	"strconv"
	"testing"
	"time"

	"github.com/doumiao/newRPS/internal/types"
)

// newAltAccountsTestServer 复用 newActivityTestServer 的骨架，额外注册一批玩家档案，
// 方便断言 altAccountCandidates 返回的 PublicPlayer 列表。
func newAltAccountsTestServer(t *testing.T, playerIDs ...string) *Server {
	t.Helper()
	s := newActivityTestServer(t)
	for _, id := range playerIDs {
		s.players[id] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: id, Name: id}}
	}
	return s
}

func namesOf(players []types.PublicPlayer) []string {
	out := make([]string, len(players))
	for i, p := range players {
		out[i] = p.Name
	}
	return out
}

func sameSet(got []string, want ...string) bool {
	if len(got) != len(want) {
		return false
	}
	set := map[string]bool{}
	for _, g := range got {
		set[g] = true
	}
	for _, w := range want {
		if !set[w] {
			return false
		}
	}
	return true
}

func mustInsertConnectionEvent(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("insertConnectionEvent: %v", err)
	}
}

// TestAltAccountCandidatesTransitiveChain 验证 A-devA-B-devB-C 这种链式关联能被逐层
// 展开挖出来：底层一跳查询（altAccountPlayerIDs）查 A 只能看到 B，altAccountCandidates 的
// BFS 传递展开要能看到 B 和 C——也就是"小号的小号"。
func TestAltAccountCandidatesTransitiveChain(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	s := newAltAccountsTestServer(t, "a", "b", "c")
	s.activityDB = newActivityStore(db)
	s.db = db

	// a、b 共用 devA；b、c 共用 devB；a 和 c 从没直接共用过设备。
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s1", 1000, 2000, "sid1", "1.1.1.1", "devA", "fpA", "ua", "ctx", "a", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s2", 3000, 4000, "sid2", "2.2.2.2", "devA", "fpA", "ua", "ctx", "b", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s3", 5000, 6000, "sid3", "3.3.3.3", "devB", "fpB", "ua", "ctx", "b", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s4", 7000, 8000, "sid4", "4.4.4.4", "devB", "fpB", "ua", "ctx", "c", "disconnect", "", ""))

	direct, err := altAccountPlayerIDs(db, "a")
	if err != nil {
		t.Fatalf("altAccountPlayerIDs: %v", err)
	}
	if len(direct) != 1 || direct[0] != "b" {
		t.Fatalf("altAccountPlayerIDs(a) = %v, want [b] only（一跳查询不该看到c）", direct)
	}

	all, err := s.altAccountCandidates("a")
	if err != nil {
		t.Fatalf("altAccountCandidates: %v", err)
	}
	if got := namesOf(all); !sameSet(got, "b", "c") {
		t.Fatalf("altAccountCandidates(a) = %v, want [b c]（传递展开应挖到c）", got)
	}
}

// TestAltAccountCandidatesTransitiveCycle 验证 A↔B 互为对方的一跳小号（环）时，
// BFS 不会死循环——visited 集合让每个玩家只入队一次，且结果里不会出现查询者本人。
func TestAltAccountCandidatesTransitiveCycle(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	s := newAltAccountsTestServer(t, "a", "b")
	s.activityDB = newActivityStore(db)
	s.db = db

	// a、b 都用过 devA 和 devB：无论从谁反查，都会重新看到对方，构成一个环。
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s1", 1000, 2000, "sid1", "1.1.1.1", "devA", "fpA", "ua", "ctx", "a", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s2", 3000, 4000, "sid2", "2.2.2.2", "devA", "fpA", "ua", "ctx", "b", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s3", 5000, 6000, "sid3", "3.3.3.3", "devB", "fpB", "ua", "ctx", "a", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s4", 7000, 8000, "sid4", "4.4.4.4", "devB", "fpB", "ua", "ctx", "b", "disconnect", "", ""))

	type result struct {
		players []types.PublicPlayer
		err     error
	}
	done := make(chan result, 1)
	go func() {
		all, err := s.altAccountCandidates("a")
		done <- result{all, err}
	}()
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatalf("altAccountCandidates: %v", r.err)
		}
		if got := namesOf(r.players); !sameSet(got, "b") {
			t.Fatalf("altAccountCandidates(a) = %v, want [b] only（a自己不应出现在结果里）", got)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("altAccountCandidates did not return in time — likely an infinite loop")
	}
}

// 当前连接要在断线时才写 connection_events。验证两名玩家第一次在同一设备在线时，
// 不需要先断线也能互相查到；并且在线关系发现的 b 还能继续沿历史设备查到 c。
func TestAltAccountCandidatesIncludesLiveConnections(t *testing.T) {
	s := newAltAccountsTestServer(t, "a", "b", "c")
	s.activityDB = newActivityStore(s.db)

	s.clients["ca"] = &Client{id: "ca", playerID: "a", deviceKey: "live-shared"}
	s.clients["cb"] = &Client{id: "cb", playerID: "b", deviceKey: "live-shared"}
	s.players["a"].SocketID = "ca"
	s.players["b"].SocketID = "cb"

	// b、c 的历史设备关联已经落库；a、b 当前设备尚未落库。
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s1", 1000, 2000, "sid1", "1.1.1.1", "history-bc", "fp", "ua", "ctx", "b", "disconnect", "", ""))
	mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s2", 3000, 4000, "sid2", "2.2.2.2", "history-bc", "fp", "ua", "ctx", "c", "disconnect", "", ""))

	all, err := s.altAccountCandidates("a")
	if err != nil {
		t.Fatalf("altAccountCandidates: %v", err)
	}
	if got := namesOf(all); !sameSet(got, "b", "c") {
		t.Fatalf("altAccountCandidates(a) = %v, want [b c]（当前连接也必须参与传递查询）", got)
	}
}

// 同一账号可能同时保留多个 WebSocket（不同标签页或不同设备）。在线快照必须保存玩家的
// 全部设备，而不是 map 赋值覆盖成最后一个，否则只能沿其中一条关系找到小号。
func TestAltAccountCandidatesIncludesAllLiveDevices(t *testing.T) {
	s := newAltAccountsTestServer(t, "a", "b", "c")
	s.activityDB = newActivityStore(s.db)

	s.clients["a1"] = &Client{id: "a1", playerID: "a", deviceKey: "live-ab"}
	s.clients["a2"] = &Client{id: "a2", playerID: "a", deviceKey: "live-ac"}
	s.clients["b1"] = &Client{id: "b1", playerID: "b", deviceKey: "live-ab"}
	s.clients["c1"] = &Client{id: "c1", playerID: "c", deviceKey: "live-ac"}

	all, err := s.altAccountCandidates("a")
	if err != nil {
		t.Fatalf("altAccountCandidates: %v", err)
	}
	if got := namesOf(all); !sameSet(got, "b", "c") {
		t.Fatalf("altAccountCandidates(a) = %v, want [b c]（同一玩家的多个在线设备都必须参与查询）", got)
	}
}

// TestAltAccountTraversalLimit 构造一条长度超过 altAccountTraversalLimit 的设备关联链，
// 验证 BFS 会在达到上限后停止，而不是无界展开。
func TestAltAccountTraversalLimit(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()

	n := altAccountTraversalLimit + 20
	ids := make([]string, 0, n)
	for i := 0; i < n; i++ {
		ids = append(ids, "p"+strconv.Itoa(i))
	}
	s := newAltAccountsTestServer(t, ids...)
	s.activityDB = newActivityStore(db)
	s.db = db

	// 链式设备关联：p0-p1 共用一个只属于它俩的设备，p1-p2 共用另一个，以此类推。
	seq := 0
	for i := 0; i+1 < n; i++ {
		dev := "dev" + strconv.Itoa(i)
		seq++
		mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s"+strconv.Itoa(seq), int64(seq*1000), int64(seq*1000+500), "sid"+strconv.Itoa(seq), "1.1.1.1", dev, "fp", "ua", "ctx", ids[i], "disconnect", "", ""))
		seq++
		mustInsertConnectionEvent(t, s.activityDB.insertConnectionEvent("s"+strconv.Itoa(seq), int64(seq*1000), int64(seq*1000+500), "sid"+strconv.Itoa(seq), "1.1.1.1", dev, "fp", "ua", "ctx", ids[i+1], "disconnect", "", ""))
	}

	found, truncated, err := altAccountCandidateIDs(s.db, ids[0], altAccountLiveSnapshot{})
	if err != nil {
		t.Fatalf("altAccountCandidateIDs: %v", err)
	}
	all := s.altAccountsToPublicPlayers(found)
	if len(all) > altAccountTraversalLimit {
		t.Fatalf("result count = %d, want <= %d（应在达到上限后停止）", len(all), altAccountTraversalLimit)
	}
	if len(all) == 0 {
		t.Fatalf("result count = 0, want some players discovered before hitting the limit")
	}
	if !truncated {
		t.Fatalf("truncated=false, want true（达到遍历上限时必须让前端知道结果不完整）")
	}
}
