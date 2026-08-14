package server

import (
	"testing"

	"github.com/doumiao/newRPS/internal/types"
)

func TestInsertConnectionEventOneShot(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newActivityStore(db)

	// 一次性写完整一行：connected_at/disconnected_at/close_reason 都在同一次 INSERT 里。
	if err := store.insertConnectionEvent("sock1", 1000, 2000, "sid1", "1.2.3.4", "dev1", "fp1", "ua1", "ctx", "p1", "disconnect", "广东省", "电信"); err != nil {
		t.Fatalf("insertConnectionEvent: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM connection_events`).Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Fatalf("row count = %d, want 1", count)
	}
	var connectedAt, disconnectedAt int64
	var playerID, closeReason string
	row := db.QueryRow(`SELECT connected_at, disconnected_at, player_id, close_reason FROM connection_events WHERE socket_id = ?`, "sock1")
	if err := row.Scan(&connectedAt, &disconnectedAt, &playerID, &closeReason); err != nil {
		t.Fatalf("scan: %v", err)
	}
	if connectedAt != 1000 || disconnectedAt != 2000 || playerID != "p1" || closeReason != "disconnect" {
		t.Fatalf("unexpected row: connectedAt=%d disconnectedAt=%d player=%q reason=%q", connectedAt, disconnectedAt, playerID, closeReason)
	}
}

func TestAltAccountPlayerIDs(t *testing.T) {
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	defer db.Close()
	store := newActivityStore(db)

	// p1、p2 共用过 devA；p3 从没共用过设备；p4 干脆没有 connection_events 记录。
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("insertConnectionEvent: %v", err)
		}
	}
	must(store.insertConnectionEvent("s1", 1000, 2000, "sid1", "1.1.1.1", "devA", "fpA", "ua", "ctx", "p1", "disconnect", "", ""))
	must(store.insertConnectionEvent("s2", 3000, 4000, "sid2", "2.2.2.2", "devA", "fpA", "ua", "ctx", "p2", "disconnect", "", ""))
	must(store.insertConnectionEvent("s3", 5000, 6000, "sid3", "3.3.3.3", "devB", "fpB", "ua", "ctx", "p3", "disconnect", "", ""))

	ids, err := altAccountPlayerIDs(db, "p1")
	if err != nil {
		t.Fatalf("altAccountPlayerIDs: %v", err)
	}
	if len(ids) != 1 || ids[0] != "p2" {
		t.Fatalf("altAccountPlayerIDs(p1) = %v, want [p2]", ids)
	}

	if ids, err := altAccountPlayerIDs(db, "p3"); err != nil || len(ids) != 0 {
		t.Fatalf("altAccountPlayerIDs(p3) = %v, err=%v, want empty", ids, err)
	}
	if ids, err := altAccountPlayerIDs(db, "p4"); err != nil || len(ids) != 0 {
		t.Fatalf("altAccountPlayerIDs(p4) = %v, err=%v, want empty", ids, err)
	}

	var indexed int
	if err := db.QueryRow(`SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND name='idx_connection_events_device'`).Scan(&indexed); err != nil {
		t.Fatalf("check index: %v", err)
	}
	if indexed != 1 {
		t.Fatalf("idx_connection_events_device missing")
	}
}

func newActivityTestServer(t *testing.T) *Server {
	t.Helper()
	dir := t.TempDir()
	db, err := openDatabase(dir)
	if err != nil {
		t.Fatalf("openDatabase: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &Server{
		players:    map[string]*PlayerState{},
		rooms:      map[string]*RoomState{},
		clients:    map[string]*Client{},
		activityDB: newActivityStore(db),
		eventDB:    newEventStore(db),
		db:         db,
	}
}

func TestCloseLiveStateOnShutdown(t *testing.T) {
	s := newActivityTestServer(t)

	// 一条已经绑定玩家的连接，一条从没关联过玩家的（比如连上就断的游客）。
	s.clients["sock-bound"] = &Client{id: "sock-bound", connectedAt: 1000, sid: "sid-a", ipAddress: "1.1.1.1", compression: "ctx", playerID: "p1"}
	s.clients["sock-guest"] = &Client{id: "sock-guest", connectedAt: 1100, sid: "sid-b", ipAddress: "2.2.2.2"}
	s.players["p1"] = &PlayerState{PublicPlayer: types.PublicPlayer{ID: "p1"}, SocketID: "sock-bound"}
	s.players["p1"].Stats.TotalOnlineMs = 500

	s.rooms["room1"] = &RoomState{
		ID: "room1", OwnerID: "p1", CreatorName: "小明",
		Settings:  types.RoomSettings{Name: "测试房间", GameID: types.GameRPS},
		CreatedAt: 500,
	}

	s.closeLiveStateOnShutdown()

	// 关停必须累加本会话时长并置 dirty，否则随后 flushPersist 会跳过写盘。
	if got := s.players["p1"].Stats.TotalOnlineMs; got <= 500 {
		t.Fatalf("TotalOnlineMs after shutdown=%d, want >500 (session accumulated)", got)
	}
	s.persistMu.Lock()
	dirty := s.persistDirty
	s.persistMu.Unlock()
	if !dirty {
		t.Fatalf("persistDirty should be true after shutdown online-ms accumulation")
	}

	var connCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM connection_events`).Scan(&connCount); err != nil {
		t.Fatalf("count connection_events: %v", err)
	}
	if connCount != 2 {
		t.Fatalf("connection_events count = %d, want 2", connCount)
	}
	var boundPlayerID, boundReason string
	if err := s.db.QueryRow(`SELECT player_id, close_reason FROM connection_events WHERE socket_id = ?`, "sock-bound").
		Scan(&boundPlayerID, &boundReason); err != nil {
		t.Fatalf("scan sock-bound: %v", err)
	}
	if boundPlayerID != "p1" || boundReason != "server_shutdown" {
		t.Fatalf("sock-bound row: player=%q reason=%q", boundPlayerID, boundReason)
	}
	var guestPlayerID string
	if err := s.db.QueryRow(`SELECT player_id FROM connection_events WHERE socket_id = ?`, "sock-guest").
		Scan(&guestPlayerID); err != nil {
		t.Fatalf("scan sock-guest: %v", err)
	}
	if guestPlayerID != "" {
		t.Fatalf("sock-guest player_id = %q, want empty (never bound to a player)", guestPlayerID)
	}

	var roomCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM room_events WHERE action = 'close'`).Scan(&roomCount); err != nil {
		t.Fatalf("count room_events: %v", err)
	}
	if roomCount != 1 {
		t.Fatalf("room_events close count = %d, want 1", roomCount)
	}
	var roomName, userName, reason string
	if err := s.db.QueryRow(`SELECT room_name, user_name, reason FROM room_events WHERE room_id = ? AND action = 'close'`, "room1").
		Scan(&roomName, &userName, &reason); err != nil {
		t.Fatalf("scan room1: %v", err)
	}
	if roomName != "测试房间" || userName != "小明" || reason != "server_shutdown" {
		t.Fatalf("room1 row: name=%q creator=%q reason=%q", roomName, userName, reason)
	}
}
