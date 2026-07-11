package server

import (
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
	root := config.GetRootDir()
	uploadsDir := filepath.Join(root, "work", "uploads")
	proofDir := filepath.Join(uploadsDir, "proofs")
	adminDir := filepath.Join(uploadsDir, "admin")
	dataDir := filepath.Join(root, "data")
	_ = os.MkdirAll(proofDir, 0o755)
	_ = os.MkdirAll(adminDir, 0o755)
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
	maxSockets := (cfg.AccessControl.MaxOnlinePerIP) * 2
	if maxSockets < 5 {
		maxSockets = 5
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
	roomDelay := 60 * time.Millisecond
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
	host := os.Getenv("HOST")
	if host == "" {
		host = "0.0.0.0"
	}
	distDir := filepath.Join(root, "dist")
	if _, err := os.Stat(distDir); err != nil {
		distDir = ""
	}

	s := &Server{
		cfg:                         cfg,
		players:                     map[string]*PlayerState{},
		tokenToPlayer:               map[string]string{},
		playerIdToID:                map[string]string{},
		sidToPlayerID:               map[string]string{},
		rooms:                       map[string]*RoomState{},
		clients:                     map[string]*Client{},
		socketToClient:              map[string]*Client{},
		botTimers:                   map[string]*time.Timer{},
		othelloSettlementTimers:     map[string]*time.Timer{},
		ticTacToeGiveawayTimers:     map[string]*time.Timer{},
		ipCreateAttempts:            map[string][]int64{},
		adminClientIDs:              map[string]struct{}{},
		sidToClientID:               map[string]string{},
		clientIDToSID:               map[string]string{},
		clientIDsByIP:               map[string]map[string]struct{}{},
		rateBuckets:                 map[string]*rateBucket{},
		rateLimitBuckets:            map[string]*rateLimitBucket{},
		roomBroadcastTimers:         map[string]*roomBroadcastPending{},
		lobbyBroadcastDelay:         lobbyDelay,
		roomBroadcastDelay:          roomDelay,
		serverStats:                 types.ServerStats{StartedAt: nowMs()},
		isProduction:                os.Getenv("NODE_ENV") == "production" || os.Getenv("GO_ENV") == "production",
		sessionSecret:               secret,
		sessionTtlMs:                ttl,
		maxSocketsPerIP:             maxSockets,
		host:                        host,
		port:                        port,
		uploadsDir:                  uploadsDir,
		proofUploadsDir:             proofDir,
		adminUploadsDir:             adminDir,
		dataDir:                     dataDir,
		playersFile:                 filepath.Join(dataDir, "players.json"),
		distDir:                     distDir,
		startedAt:                   nowMs(),
	}
	exportConfigText = config.ExportConfigText
	return s, nil
}

// Run starts HTTP server and blocks until shutdown.
func (s *Server) Run() error {
	s.loadPlayersFromDisk()
	s.scheduleExtremeHourlyDecay()
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		for range ticker.C {
			s.flushPersist()
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
	case sig := <-sigCh:
		fmt.Printf("[server] %v received, flushing players...\n", sig)
		s.shuttingDown = true
		s.flushPersist()
		_ = httpServer.Close()
		return nil
	case err := <-errCh:
		if err == http.ErrServerClosed {
			return nil
		}
		return err
	}
}
