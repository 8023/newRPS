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
		botTimers:               map[string]*time.Timer{},
		othelloSettlementTimers: map[string]*time.Timer{},
		ticTacToeGiveawayTimers: map[string]*time.Timer{},
		liarsDiceStartTimers:    map[string]*time.Timer{},
		gomokuUndoTimers:        map[string]*time.Timer{},
		deviceCreateAttempts:    map[string][]int64{},
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
	}
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
	s.scheduleExtremeHourlyDecay()
	go s.runActivityLogConsumer()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
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
