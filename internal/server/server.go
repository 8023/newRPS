package server

import (
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/types"
)

// New creates a Server with default configuration from disk.
func New() (*Server, error) {
	cfg, err := config.LoadConfig()
	if err != nil {
		return nil, err
	}
	config.EnsureConfigPermissions()
	root := config.GetRootDir()
	uploadsDir := filepath.Join(root, "work", "uploads")
	proofDir := filepath.Join(uploadsDir, "proofs")
	adminDir := filepath.Join(uploadsDir, "admin")
	avatarDir := filepath.Join(uploadsDir, "avatars")
	dataDir := filepath.Join(root, "data")
	_ = os.MkdirAll(proofDir, 0o755)
	_ = os.MkdirAll(adminDir, 0o755)
	_ = os.MkdirAll(avatarDir, 0o755)
	_ = os.MkdirAll(dataDir, 0o755)

	// 会话密钥：优先环境变量；否则落盘 work/session.secret，避免每次重启使浏览器 token 全部失效。
	secret := []byte(os.Getenv("SESSION_SECRET"))
	if len(secret) == 0 {
		secretPath := filepath.Join(root, "work", "session.secret")
		if data, err := os.ReadFile(secretPath); err == nil && len(data) >= 16 {
			secret = data
		} else {
			secret = make([]byte, 32)
			_, _ = rand.Read(secret)
			_ = os.MkdirAll(filepath.Dir(secretPath), 0o755)
			_ = os.WriteFile(secretPath, secret, 0o600)
		}
	}
	ttl := int64(24 * 60 * 60 * 1000)
	if v := os.Getenv("SESSION_TTL_MS"); v != "" {
		if n, err := strconv.ParseInt(v, 10, 64); err == nil && n >= 5*60_000 {
			ttl = n
		}
	}
	// 每设备（IP+指纹）套接字上限：Safari 重连/多标签会短暂占多个连接，过紧会直接握手失败。
	maxSockets := (cfg.AccessControl.MaxOnlinePerIP) * 4
	if maxSockets < 12 {
		maxSockets = 12
	}
	if v := os.Getenv("MAX_SOCKETS_PER_IP"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 1 {
			maxSockets = n
		}
	}
	lobbyDelay := 300 * time.Millisecond
	if v := os.Getenv("LOBBY_BROADCAST_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 50 {
			lobbyDelay = time.Duration(n) * time.Millisecond
		}
	}
	roomDelay := 100 * time.Millisecond
	if v := os.Getenv("ROOM_BROADCAST_DELAY_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 20 {
			roomDelay = time.Duration(n) * time.Millisecond
		}
	}
	port := 9988
	if v := os.Getenv("PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			port = n
		}
	}
	// 可信反向代理层数：决定 X-Forwarded-For/Host 的信任方式（默认 1，直连可设 0）。
	if v := os.Getenv("TRUSTED_PROXY_COUNT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			trustedProxyCount = n
		}
	}
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	// 优先仓库根 dist/（vite outDir: ../dist），兼容旧路径 web/dist
	distDir := filepath.Join(root, "dist")
	if _, err := os.Stat(distDir); err != nil {
		alt := filepath.Join(root, "web", "dist")
		if _, err2 := os.Stat(alt); err2 == nil {
			distDir = alt
		} else {
			distDir = ""
		}
	}

	s := &Server{
		cfg:                     cfg,
		players:                 map[string]*PlayerState{},
		tokenToPlayer:           map[string]string{},
		playerIdToID:            map[string]string{},
		sidToPlayerID:           map[string]string{},
		rooms:                   map[string]*RoomState{},
		clients:                 map[string]*Client{},
		socketToClient:          map[string]*Client{},
		othelloSettlementTimers: map[string]*time.Timer{},
		ticTacToeGiveawayTimers: map[string]*time.Timer{},
		liarsDiceStartTimers:    map[string]*time.Timer{},
		gomokuUndoTimers:        map[string]*time.Timer{},
		othelloClockTimers:      map[string]*time.Timer{},
		gomokuClockTimers:       map[string]*time.Timer{},
		jungleClockTimers:       map[string]*time.Timer{},
		deviceCreateAttempts:    map[string][]int64{},
		ipCreateAttempts:        map[string][]int64{},
		adminClientIDs:          map[string]struct{}{},
		sidToClientID:           map[string]string{},
		clientIDToSID:           map[string]string{},
		clientIDsByDevice:       map[string]map[string]struct{}{},
		rateBuckets:             map[string]*rateBucket{},
		rateLimitBuckets:        map[string]*rateLimitBucket{},
		roomBroadcastTimers:     map[string]*roomBroadcastPending{},
		lobbyBroadcastDelay:     lobbyDelay,
		roomBroadcastDelay:      roomDelay,
		serverStats:             types.ServerStats{StartedAt: nowMs()},
		isProduction:            os.Getenv("NODE_ENV") == "production" || os.Getenv("GO_ENV") == "production",
		sessionSecret:           secret,
		sessionTtlMs:            ttl,
		maxSocketsPerDevice:     maxSockets,
		host:                    host,
		port:                    port,
		uploadsDir:              uploadsDir,
		proofUploadsDir:         proofDir,
		adminUploadsDir:         adminDir,
		avatarUploadsDir:        avatarDir,
		dataDir:                 dataDir,
		playersFile:             filepath.Join(dataDir, "players.json"),
		distDir:                 distDir,
		logCh:                   make(chan activityLogEntry, 1024),
		startedAt:               nowMs(),
	}
	// SQLite 持久化（聊天/房间事件/惩罚事件/玩家档案共用一个连接）：失败不阻断启动，仅记录
	// （内存降级为不落盘，功能仍可用）。
	if db, err := openDatabase(dataDir); err != nil {
		s.errorLog("database_open_failed", err.Error())
	} else {
		s.db = db
		s.chatDB = newChatStore(db)
		s.eventDB = newEventStore(db)
		s.pushDB = newPushStore(db)
		s.playerDB = newPlayerStore(db)
		s.activityDB = newActivityStore(db)
		s.petBondDB = newPetBondStore(db)
	}
	s.petBonds = map[string]*petBond{}
	s.petBondRequests = map[string]*petBondRequest{}
	// VAPID 密钥：失败不阻断启动，Web Push 功能会静默不可用（sendPush 会因 vapid.PublicKey=="" 直接跳过）。
	if keys, err := loadOrGenerateVAPIDKeys(root); err != nil {
		s.errorLog("vapid_keys_failed", err.Error())
	} else {
		s.vapid = keys
	}
	exportConfigText = config.ExportConfigText
	return s, nil
}

// Run starts HTTP server and blocks until shutdown.
func (s *Server) Run() error {
	s.loadPlayersFromDisk()
	s.loadPetBondsFromDisk()
	s.scheduleExtremeHourlyDecay()
	s.scheduleRankedDailyDecay()
	go s.runActivityLogConsumer()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			// 先把仍在线会话折叠进 TotalOnlineMs 再刷盘，避免进程被 kill -9 /
			// npm 热重启等非 SIGTERM 路径丢掉整段长会话。
			s.mu.Lock()
			if s.checkpointOnlineMs() {
				s.markPersistDirty()
			}
			s.mu.Unlock()
			s.flushPersist()
		}
	}()
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		for range ticker.C {
			s.mu.Lock()
			s.pruneEphemeralState()
			s.mu.Unlock()
		}
	}()

	addr := fmt.Sprintf("%s:%d", s.host, s.port)
	httpServer := &http.Server{
		Addr:              addr,
		Handler:           s.routes(),
		ReadHeaderTimeout: 10 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		fmt.Printf("RPS Online server listening on http://%s\n", addr)
		errCh <- httpServer.ListenAndServe()
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-sigCh:
		s.flushPersist()
		// 优雅关停：给在途请求 / WebSocket 5 秒收尾，而非硬切。
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = httpServer.Shutdown(shutdownCtx)
		// httpServer.Shutdown 不会主动断开已升级的 WebSocket 连接，也不会触发正常的房间关闭
		// 流程；不补这一步的话，这些连接/房间这次的生命周期就完全没有落盘记录（见
		// closeLiveStateOnShutdown）。真正的进程被杀（kill -9/OOM）仍然覆盖不到，属于已知的、
		// 影响有限的边界情况。
		s.closeLiveStateOnShutdown()
		// closeLiveState 会把仍在线会话的时长累进 TotalOnlineMs，再刷一次盘。
		s.flushPersist()
		if s.db != nil {
			_ = s.db.Close()
		}
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}

// closeLiveStateOnShutdown 在优雅关停（SIGINT/SIGTERM）时，把内存里仍处于"存活"状态的
// 连接（s.clients）和房间（s.rooms）各写一条收尾记录——它们的 connection_events/rooms 行只在
// 结束时一次性 insert（见 activitystore.go/eventstore.go），正常情况下分别由 onClientDisconnect
// 和 cleanupRoomIfEmpty/admin 关房触发；但 httpServer.Shutdown 不会主动断开已升级的 WebSocket，
// 进程直接退出时也不会有玩家主动关闭房间，这些连接/房间这次的生命周期就不会经过那两条路径，
// 需要在这里补写。
//
// 字段读取与"摘出存活集合"必须整段放在 s.mu 内完成：房间会在 empty_cleanup/admin_close 路径下
// 被并发关闭并 delete(s.rooms, id)，若在这里只是复制指针、解锁后才读字段/判断是否已关闭，会与
// 那两条路径产生竞争——既可能读到理论上会变化的字段，也可能出现同一房间被两条路径各写一行
// close 记录（rooms.room_id 主键冲突，只有先写入的一行生效，另一次会被当作错误记进日志）。这里
// 把需要的字段值复制出来，并且把 delete(s.rooms, id) 一起放进同一段临界区：谁先拿到锁，谁就
// 独占了"关闭这个房间"的资格，之后任何路径按 s.rooms[id]==nil 都会直接跳过，不会重复处理。
// 真正的 SQLite 写入不做 I/O 前不涉及共享状态，放到锁外执行。
func (s *Server) closeLiveStateOnShutdown() {
	now := nowMs()

	type openConn struct {
		id          string
		connectedAt int64
		sid         string
		ipAddress   string
		deviceKey   string
		fingerprint string
		userAgent   string
		compression string
		playerID    string
	}
	type closedRoom struct {
		id          string
		name        string
		gameID      string
		ownerID     string
		creatorName string
		createdAt   int64
	}

	s.mu.Lock()
	conns := make([]openConn, 0, len(s.clients))
	onlineMsTouched := false
	for _, c := range s.clients {
		// 先打标，防止随后 WS 拆线触发 onClientDisconnect 再写一条 disconnect / 再累加时长。
		if c.connectionLogged {
			continue
		}
		c.connectionLogged = true
		playerID := ""
		if p := s.getPlayerByClient(c); p != nil {
			playerID = p.ID
			// 优雅关停时本会话时长也计入累计在线。必须 markPersistDirty：调用方紧接着的
			// flushPersist 只在 dirty 时写盘；此前这里只改内存不置脏，重启后长会话会丢
			// （本地库曾出现 connection_events 合计数小时、total_online_ms 只剩最后一次
			// 正常断线的几十秒）。
			// 与 accumulateClientOnlineMs 一致：只计尚未 checkpoint 的段落。
			from := clientOnlineCreditedFrom(c)
			if from > 0 {
				if d := now - from; d > 0 {
					p.Stats.TotalOnlineMs += d
					c.onlineCreditedAt = now
					onlineMsTouched = true
				}
			}
		}
		conns = append(conns, openConn{
			id: c.id, connectedAt: c.connectedAt, sid: c.sid, ipAddress: c.ipAddress,
			deviceKey: c.deviceKey, fingerprint: c.fingerprint, userAgent: c.userAgent,
			compression: c.compression, playerID: playerID,
		})
	}
	if onlineMsTouched {
		// 持 s.mu 时调用 markPersistDirty 是安全的（与 accumulateClientOnlineMs 同锁序）。
		s.markPersistDirty()
	}
	rooms := make([]closedRoom, 0, len(s.rooms))
	for id, r := range s.rooms {
		rooms = append(rooms, closedRoom{
			id: id, name: r.Settings.Name, gameID: string(r.Settings.GameID),
			ownerID: r.OwnerID, creatorName: r.CreatorName, createdAt: r.CreatedAt,
		})
		delete(s.rooms, id)
	}
	s.mu.Unlock()

	if s.activityDB != nil {
		for _, c := range conns {
			if err := s.activityDB.insertConnectionEvent(
				c.id, c.connectedAt, now, c.sid, c.ipAddress,
				c.deviceKey, c.fingerprint, c.userAgent, c.compression,
				c.playerID, "server_shutdown",
			); err != nil {
				s.errorLog("connection_event_persist_failed", err.Error())
			}
		}
	}
	if s.eventDB != nil {
		for _, r := range rooms {
			if err := s.eventDB.insertRoomEvent(roomEventInput{
				At: now, RoomID: r.id, RoomName: r.name, GameID: r.gameID,
				UserID: r.ownerID, UserName: r.creatorName, Action: "close", Reason: "server_shutdown",
			}); err != nil {
				s.errorLog("room_event_close_failed", err.Error())
			}
		}
	}
}
