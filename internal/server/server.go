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
	"strings"
	"syscall"
	"time"

	"github.com/doumiao/newRPS/internal/config"
	"github.com/doumiao/newRPS/internal/geoip"
	"github.com/doumiao/newRPS/internal/types"
)

func envBoolDefault(key string, def bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	switch strings.ToLower(v) {
	case "0", "false", "no", "off":
		return false
	case "1", "true", "yes", "on":
		return true
	default:
		return def
	}
}

func envIntDefault(key string, def int) int {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func loadOrCreateAnalyticsSalt(root string) []byte {
	path := filepath.Join(root, "work", "analytics.salt")
	if data, err := os.ReadFile(path); err == nil && len(data) >= 16 {
		return data
	}
	salt := make([]byte, 32)
	_, _ = rand.Read(salt)
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	_ = os.WriteFile(path, salt, 0o600)
	return salt
}

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
	dataDir := filepath.Join(root, "work", "db")
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
	// 可信反向代理层数：决定 X-Forwarded-For/Host 的信任方式（默认 0=直连；部署在反代之后需显式设置）。
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
		cfg:                       cfg,
		players:                   map[string]*PlayerState{},
		tokenToPlayer:             map[string]string{},
		playerIdToID:              map[string]string{},
		sidToPlayerID:             map[string]string{},
		rooms:                     map[string]*RoomState{},
		clients:                   map[string]*Client{},
		socketToClient:            map[string]*Client{},
		roomClients:               map[string]map[string]struct{}{},
		othelloSettlementTimers:   map[string]*time.Timer{},
		ticTacToeGiveawayTimers:   map[string]*time.Timer{},
		liarsDiceStartTimers:      map[string]*time.Timer{},
		gomokuUndoTimers:          map[string]*time.Timer{},
		othelloClockTimers:        map[string]*time.Timer{},
		gomokuClockTimers:         map[string]*time.Timer{},
		jungleClockTimers:         map[string]*time.Timer{},
		deviceCreateAttempts:      map[string][]int64{},
		ipCreateAttempts:          map[string][]int64{},
		adminClientIDs:            map[string]struct{}{},
		sidToClientID:             map[string]string{},
		clientIDToSID:             map[string]string{},
		clientIDsByDevice:         map[string]map[string]struct{}{},
		rateBuckets:               map[string]*rateBucket{},
		rateLimitBuckets:          map[string]*rateLimitBucket{},
		roomBroadcastTimers:       map[string]*roomBroadcastPending{},
		lobbyBroadcastDelay:       lobbyDelay,
		roomBroadcastDelay:        roomDelay,
		serverStats:               types.ServerStats{StartedAt: nowMs()},
		isProduction:              os.Getenv("NODE_ENV") == "production" || os.Getenv("GO_ENV") == "production",
		sessionSecret:             secret,
		sessionTtlMs:              ttl,
		maxSocketsPerDevice:       maxSockets,
		host:                      host,
		port:                      port,
		uploadsDir:                uploadsDir,
		proofUploadsDir:           proofDir,
		adminUploadsDir:           adminDir,
		avatarUploadsDir:          avatarDir,
		dataDir:                   dataDir,
		playersFile:               filepath.Join(dataDir, "players.json"),
		distDir:                   distDir,
		logCh:                     make(chan activityLogEntry, 1024),
		startedAt:                 nowMs(),
		analyticsKick:             make(chan struct{}, 1),
		analyticsEnabled:          envBoolDefault("ANALYTICS_ENABLED", true),
		analyticsGeoEnabled:       envBoolDefault("ANALYTICS_GEO_ENABLED", true),
		analyticsTZOffsetMin:      envIntDefault("ANALYTICS_TZ_OFFSET_MIN", 480),
		analyticsRawRetentionDays: envIntDefault("ANALYTICS_RAW_RETENTION_DAYS", 90),
	}
	// 分析访客盐：work/analytics.salt，32 随机字节，首启生成，权限 0600。
	s.analyticsSalt = loadOrCreateAnalyticsSalt(root)
	// IP 归属地：默认 config/xdb/ip2region_v4.xdb；ANALYTICS_GEO_ENABLED=0 时不载入。
	// 缺失且开启时直接报错退出（破坏性升级，见 README 升级说明）。
	// 总分析开关关闭时不应再要求地域库存在；否则用户即使设置
	// ANALYTICS_ENABLED=0 仍会因缺少可选 xdb 无法启动。
	// 必须在 openDatabase 之前完成：schema_migrations.go 的 v18 迁移要用已加载的 geoip
	// 搜索器给 connection_events 历史行回填 province/isp，迁移执行时 geoip 必须已就绪。
	if s.analyticsEnabled && s.analyticsGeoEnabled {
		geoPath := os.Getenv("GEOIP_DB_PATH")
		if geoPath == "" {
			geoPath = geoip.DefaultPath(root)
		}
		if err := geoip.Init(geoPath); err != nil {
			return nil, fmt.Errorf(
				"geoip: 未找到 IP 归属地数据库 %s。\n"+
					"请从 https://github.com/lionsoul2014/ip2region/releases 下载 ip2region_v4.xdb 放到该路径，\n"+
					"或设置环境变量 GEOIP_DB_PATH 指定位置，或设置 ANALYTICS_GEO_ENABLED=0 关闭归属地解析。\n"+
					"底层错误: %w", geoPath, err,
			)
		}
		// IPv6 xdb 是可选加成：默认 config/xdb/ip2region_v6.xdb，缺失/损坏只记日志，
		// 不影响已加载的 v4 库——此时 IPv6 来源的访客归属地解析会退化为空。
		geoPathV6 := os.Getenv("GEOIP_DB_PATH_V6")
		if geoPathV6 == "" {
			geoPathV6 = geoip.DefaultPathV6(root)
		}
		if err := geoip.InitV6(geoPathV6); err != nil {
			s.errorLog("geoip_v6_init_failed", err.Error())
		}
	} else {
		s.analyticsGeoEnabled = false
		geoip.Disable()
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
		s.analyticsDB = newAnalyticsStore(db)
		// 只读聚合连接必须在 openDatabase（WAL 初始化）之后；失败只记日志，不阻断启动。
		// ANALYTICS_ENABLED=0 时聚合协程根本不会启动（见 Run()），这个连接也就永远用不上，
		// 不必额外占两个 SQLite 连接句柄。
		if s.analyticsEnabled {
			if ro, err := openAnalyticsReadOnlyDB(dataDir); err != nil {
				s.errorLog("analytics_readonly_open_failed", err.Error())
			} else {
				s.analyticsRO = ro
			}
		}
	}
	s.petBonds = map[string]*petBond{}
	s.petBondRequests = map[string]*petBondRequest{}
	// VAPID 密钥：失败不阻断游戏服务，但会记错误日志；公钥接口返回 503，
	// 设置页会明确显示订阅失败，sendPush 也会因公钥为空跳过。
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
	if s.analyticsEnabled && s.analyticsDB != nil {
		go s.runAnalyticsAggregator()
	}
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
		if s.analyticsRO != nil {
			_ = s.analyticsRO.Close()
		}
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
		province    string
		isp         string
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
					s.markPlayerDirty(p)
				}
			}
		}
		conns = append(conns, openConn{
			id: c.id, connectedAt: c.connectedAt, sid: c.sid, ipAddress: c.ipAddress,
			deviceKey: c.deviceKey, fingerprint: c.fingerprint, userAgent: c.userAgent,
			compression: c.compression, playerID: playerID,
			province: c.anaGeo.Province, isp: c.anaGeo.ISP,
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
				c.playerID, "server_shutdown", c.province, c.isp,
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
