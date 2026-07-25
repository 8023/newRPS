package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/pbconv"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

// wsEnvelope 是入站解码后的逻辑视图（载荷已是 map / 结构，非 JSON 文本帧）。
type wsEnvelope struct {
	E  string
	ID int64
	D  map[string]any
}

// writeBinary 把数据放进本连接的发送队列（非阻塞）。队列满说明客户端消费不过来，
// 主动断开该连接，绝不在此阻塞——本方法通常在持有 s.mu 时被调用。
func (c *Client) writeBinary(data []byte) error {
	c.writeMu.Lock()
	if c.closed {
		c.writeMu.Unlock()
		return fmt.Errorf("closed")
	}
	select {
	case c.sendCh <- data:
		c.writeMu.Unlock()
		return nil
	default:
		c.writeMu.Unlock()
		// 慢消费者：发送队列已满，断开以免拖累全局锁
		_ = c.conn.Close(websocket.StatusPolicyViolation, "slow consumer")
		c.shutdown()
		return fmt.Errorf("send queue full")
	}
}

// shutdown 幂等地停掉写协程（关闭 done）并标记连接不可再写。
func (c *Client) shutdown() {
	c.closeOnce.Do(func() {
		c.writeMu.Lock()
		c.closed = true
		c.writeMu.Unlock()
		close(c.done)
	})
}

// writeLoop 是本连接唯一的写协程：串行发送，保证顺序，与全局锁解耦。
func (c *Client) writeLoop() {
	for {
		select {
		case <-c.done:
			return
		case data := <-c.sendCh:
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			err := c.conn.Write(ctx, websocket.MessageBinary, data)
			cancel()
			if err != nil {
				_ = c.conn.Close(websocket.StatusInternalError, "write error")
				c.shutdown()
				return
			}
		}
	}
}

// reply 通过 protobuf RAW 信封返回 RPC 应答（RawBody，无 JSON 字节）。
// 编码失败时仍回传 err，避免客户端一直等 ack（表现为按钮无反应直至超时）。
func (c *Client) reply(id int64, data any, errMsg string) {
	if id == 0 {
		return
	}
	env := &wire.Envelope{Id: id, Kind: wire.PayloadKind_KIND_RAW, Err: errMsg}
	if errMsg == "" && data != nil {
		body, err := pbconv.BuildRawBody("", data)
		if err != nil {
			env.Err = "服务器响应编码失败"
		} else {
			env.RawBody = body
		}
	}
	out, err := proto.Marshal(env)
	if err != nil {
		// 最后兜底：仅 id + 错误文案
		fallback, ferr := proto.Marshal(&wire.Envelope{
			Id: id, Kind: wire.PayloadKind_KIND_RAW, Err: "服务器响应序列化失败",
		})
		if ferr == nil {
			_ = c.writeBinary(fallback)
		}
		return
	}
	_ = c.writeBinary(out)
}

func (c *Client) joinRoom(room string) {
	if c.rooms == nil {
		c.rooms = map[string]struct{}{}
	}
	c.rooms[room] = struct{}{}
}

func (c *Client) leaveRoom(room string) {
	delete(c.rooms, room)
}

func (c *Client) inRoom(room string) bool {
	_, ok := c.rooms[room]
	return ok
}

func (s *Server) clientLeaveRoom(c *Client, room string) {
	if c != nil {
		c.leaveRoom(room)
	}
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	// 浏览器指纹：query 优先，其次请求头（兼容后续改造）
	fingerprint := r.URL.Query().Get("fp")
	if fingerprint == "" {
		fingerprint = r.Header.Get("X-Browser-Fingerprint")
	}
	fingerprint = normalizeFingerprint(fingerprint)
	session := s.verifySessionToken(token)
	ipAddress := clientIP(r)
	devKey := deviceKey(ipAddress, fingerprint)
	userAgent := r.UserAgent()
	origin := r.Header.Get("Origin")
	// 反代后 r.Host 可能是 127.0.0.1:9988，必须用公网 Host 做 Origin 比对
	host := publicRequestHost(r)

	if !s.isAllowedOrigin(origin, host) {
		s.securityLog("socket_origin_blocked", map[string]any{
			"ip": ipAddress, "origin": origin, "host": host, "rawHost": r.Host,
			"xffHost": r.Header.Get("X-Forwarded-Host"), "userAgent": userAgent,
		})
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	if token == "" {
		s.securityLog("socket_auth_failed", map[string]any{"ip": ipAddress, "userAgent": userAgent, "reason": "missing"})
		http.Error(w, "Session token missing", http.StatusUnauthorized)
		return
	}
	if session == nil {
		expired := tokenLooksExpired(token)
		reason := "invalid"
		if expired {
			reason = "expired"
		}
		s.securityLog("socket_auth_failed", map[string]any{"ip": ipAddress, "userAgent": userAgent, "reason": reason})
		http.Error(w, "Session invalid", http.StatusUnauthorized)
		return
	}

	s.mu.Lock()
	// 先顶掉同 SID 旧连接并释放设备名额，再检查上限。
	if prevID, ok := s.sidToClientID[session.SID]; ok {
		if prev := s.clients[prevID]; prev != nil {
			// 同 SID 顶替旧连接是正常重连/多标签行为，不是错误，不打日志。
			prevKey := prev.deviceKey
			if prevKey == "" {
				prevKey = deviceKey(prev.ipAddress, prev.fingerprint)
			}
			if set := s.clientIDsByDevice[prevKey]; set != nil {
				delete(set, prev.id)
				if len(set) == 0 {
					delete(s.clientIDsByDevice, prevKey)
				}
			}
			delete(s.clients, prev.id)
			delete(s.clientIDToSID, prev.id)
			delete(s.sidToClientID, session.SID)
			prev.replaced = true
			prevConn := prev.conn
			prev.shutdown()
			s.mu.Unlock()
			_ = prevConn.Close(websocket.StatusPolicyViolation, "replaced")
			s.mu.Lock()
		}
	}
	socketsForDevice := s.ensureDeviceSocketSet(devKey)
	if len(socketsForDevice) >= s.maxSocketsPerDevice {
		s.mu.Unlock()
		s.securityLog("socket_device_limited", map[string]any{
			"sid": session.SID, "ip": ipAddress, "device": devKey, "fp": fingerprint,
			"userAgent": userAgent, "limit": s.maxSocketsPerDevice,
		})
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}
	s.mu.Unlock()

	// 压缩策略见 wsCompressionMode：Safari/iOS 关压缩，其余开 ContextTakeover 省流量。
	compMode := wsCompressionMode(userAgent)
	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: compMode,
		OriginPatterns:  []string{"*"},
	})
	if err != nil {
		s.securityLog("socket_accept_failed", map[string]any{
			"ip": ipAddress, "origin": origin, "host": host, "userAgent": userAgent,
			"compression": compressionModeName(compMode), "err": err.Error(),
		})
		return
	}
	conn.SetReadLimit(1_000_000)

	client := &Client{
		id: randomID(), conn: conn, sid: session.SID, token: token, sessionExp: session.Exp,
		ipAddress: ipAddress, fingerprint: fingerprint, deviceKey: devKey,
		rooms: map[string]struct{}{}, userAgent: userAgent, host: host, origin: origin,
		sendCh: make(chan []byte, 256), done: make(chan struct{}),
		// connectedAt/compression 只是快照在内存里，断连时才和其余字段一起写进
		// connection_events 一行（见 activitylog.go），connect 时不落盘。
		connectedAt: nowMs(), compression: compressionModeName(compMode),
	}
	go client.writeLoop()

	s.mu.Lock()
	s.clients[client.id] = client
	s.sidToClientID[session.SID] = client.id
	s.clientIDToSID[client.id] = session.SID
	// Accept 期间旧连接 onClientDisconnect 可能删掉预创建的空 device map，这里必须再 ensure
	s.ensureDeviceSocketSet(devKey)[client.id] = struct{}{}
	client.joinRoom(lobbyChannel)
	cfg := s.publicConfig()
	lobby := s.lobbySnapshot(false, true)
	// buildFullEnvelope 会读写共享的 syncChans，必须在锁内构建；发送在锁外做。
	cfgEnv, _, cfgErr := s.buildFullEnvelope("config:update", channelConfig(), cfg)
	lobbyEnv, _, lobbyErr := s.buildFullEnvelope("lobby:update", channelLobby(), lobby)
	s.mu.Unlock()

	if cfgErr == nil {
		s.emitWireClient(client, cfgEnv)
	}
	if lobbyErr == nil {
		s.emitWireClient(client, lobbyEnv)
	}

	// 服务端主动 Ping：穿透反向代理空闲超时，并保活 iOS Safari 后台/锁屏场景。
	pingCtx, cancelPing := context.WithCancel(context.Background())
	defer cancelPing()
	go s.wsPingLoop(pingCtx, conn)

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var wenv wire.Envelope
		if err := proto.Unmarshal(data, &wenv); err != nil || wenv.Event == "" {
			continue
		}
		// 应用层心跳：客户端周期性 ping，带 id 则 RPC 应答
		if wenv.Event == "ping" {
			if wenv.Id != 0 {
				client.reply(wenv.Id, map[string]any{"t": nowMs()}, "")
			}
			continue
		}
		payload, _ := pbconv.RawBodyToFront(wenv.RawBody)
		dmap, _ := payload.(map[string]any)
		if dmap == nil {
			// 数组类载荷包一层
			if payload != nil {
				dmap = map[string]any{"_": payload}
			} else {
				dmap = map[string]any{}
			}
		}
		s.handleWSEvent(client, wsEnvelope{E: wenv.Event, ID: wenv.Id, D: dmap})
	}

	cancelPing()
	// onClientDisconnect 内部已统一调用 client.shutdown()，这里不用重复调用。
	s.onClientDisconnect(client)
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

// wsCompressionMode 按 UA 选择压缩。
//
// coder/websocket 文档写明：permessage-deflate 在现代浏览器可用，**Safari 未实现**
// （且曾有 x-webkit-deflate-frame 相关 bug，库已移除）。
// CompressionNoContextTakeover 与 ContextTakeover 都是 RFC7692 同一扩展，只是
// 是否跨消息保留滑动窗口不同——**对 Safari 来说并不是「另一种可支持的压缩」**。
// 若客户端错误地协商了 deflate，Safari 会在握手后/首包阶段直接断线（connection was lost）。
//
// 策略：iOS / 桌面 Safari → 禁用压缩；Chrome/Firefox/Edge 等 → ContextTakeover 省流量。
func wsCompressionMode(userAgent string) websocket.CompressionMode {
	ua := strings.ToLower(userAgent)
	// 所有 iOS 浏览器（含 CriOS/FxiOS）走 WebKit 网络栈
	if strings.Contains(ua, "iphone") || strings.Contains(ua, "ipad") || strings.Contains(ua, "ipod") ||
		strings.Contains(ua, "crios") || strings.Contains(ua, "fxios") || strings.Contains(ua, "edgios") {
		return websocket.CompressionDisabled
	}
	// 桌面 Safari：UA 含 Safari 且不含 Chrome/Chromium/Edg（Chrome 桌面 UA 也带 Safari 字样）
	if strings.Contains(ua, "safari") &&
		!strings.Contains(ua, "chrome") &&
		!strings.Contains(ua, "chromium") &&
		!strings.Contains(ua, "edg/") &&
		!strings.Contains(ua, "android") {
		return websocket.CompressionDisabled
	}
	// 其余浏览器：跨消息上下文压缩，流量最省
	return websocket.CompressionContextTakeover
}

func compressionModeName(m websocket.CompressionMode) string {
	switch m {
	case websocket.CompressionDisabled:
		return "disabled"
	case websocket.CompressionNoContextTakeover:
		return "no_context_takeover"
	case websocket.CompressionContextTakeover:
		return "context_takeover"
	default:
		return "unknown"
	}
}

// wsPingLoop 每 20s 发 WebSocket 协议层 Ping（非业务包），防止代理/NAT/Safari 空闲掐线。
func (s *Server) wsPingLoop(ctx context.Context, conn *websocket.Conn) {
	ticker := time.NewTicker(20 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			pingCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
			err := conn.Ping(pingCtx)
			cancel()
			if err != nil {
				_ = conn.Close(websocket.StatusGoingAway, "ping timeout")
				return
			}
		}
	}
}

func (s *Server) handleWSEvent(client *Client, env wsEnvelope) {
	s.mu.Lock()
	defer s.mu.Unlock()
	// 防止业务 handler panic 导致本连接读循环静默卡死且客户端永远等 ack
	defer func() {
		if rec := recover(); rec != nil {
			s.securityLog("handler_panic", map[string]any{
				"event": env.E, "sid": client.sid, "err": fmt.Sprint(rec),
			})
			if env.ID != 0 {
				client.reply(env.ID, nil, "服务器内部错误")
			}
		}
	}()

	if !s.consumeRateLimit(fmt.Sprintf("socket:%s:%s", client.ipAddress, env.E), 60_000, 600) {
		client.reply(env.ID, nil, "操作过于频繁，请稍后再试")
		return
	}

	// 同步与资料查询
	switch env.E {
	case "sync:full":
		var p struct {
			Channel string `json:"channel"`
		}
		_ = decodeD(env, &p)
		if p.Channel == "" {
			client.reply(env.ID, nil, "channel required")
			return
		}
		s.sendFullChannel(client, p.Channel)
		client.reply(env.ID, map[string]any{"ok": true}, "")
		return
	case "player:get":
		var p struct {
			PlayerID string `json:"playerId"`
		}
		_ = decodeD(env, &p)
		pl := s.players[p.PlayerID]
		if pl == nil {
			client.reply(env.ID, nil, "玩家不存在")
			return
		}
		client.reply(env.ID, map[string]any{"player": s.publicPlayer(pl)}, "")
		return
	}

	opts, handler := s.eventHandler(env.E)
	if handler == nil {
		client.reply(env.ID, nil, "unknown event")
		return
	}
	sidOK := s.checkRateLimit(rateLimitKey(env.E, client.ipAddress, client.sid), opts)
	ipOK := s.checkIPEventBackstop(env.E, client.ipAddress, opts)
	if !sidOK || !ipOK {
		s.securityLog("rate_limit", map[string]any{"sid": client.sid, "ip": client.ipAddress, "event": env.E, "userAgent": client.userAgent})
		client.reply(env.ID, nil, "操作过于频繁，请稍后再试")
		return
	}
	handler(client, env)
}

type eventHandlerFunc func(client *Client, env wsEnvelope)

// ensureDeviceSocketSet 返回 deviceKey 对应的套接字集合（永不返回 nil）。
func (s *Server) ensureDeviceSocketSet(devKey string) map[string]struct{} {
	if s.clientIDsByDevice == nil {
		s.clientIDsByDevice = map[string]map[string]struct{}{}
	}
	set := s.clientIDsByDevice[devKey]
	if set == nil {
		set = map[string]struct{}{}
		s.clientIDsByDevice[devKey] = set
	}
	return set
}

func (s *Server) onClientDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// 通过 shutdown 统一在 writeMu 下置 closed 并停掉写协程，避免跨锁写标志位。
	client.shutdown()
	// 被同 SID 顶替：索引已在 handleWS 里清过。此处若再 delete 空 map，
	// 会把新连接 Accept 前预创建的 device 集合删掉，随后写入触发 nil map panic。
	// 仍需累加本连接已在线时长（否则刷新/重连会丢这一段）。
	if client.replaced {
		if !client.connectionLogged {
			client.connectionLogged = true
			s.accumulateClientOnlineMs(client, nowMs())
		}
		return
	}

	sid := s.clientIDToSID[client.id]
	ipAddress := client.ipAddress
	devKey := client.deviceKey
	if devKey == "" {
		devKey = deviceKey(ipAddress, client.fingerprint)
	}
	if s.sidToClientID[sid] == client.id {
		delete(s.sidToClientID, sid)
	}
	delete(s.clientIDToSID, client.id)
	if set := s.clientIDsByDevice[devKey]; set != nil {
		delete(set, client.id)
		if len(set) == 0 {
			delete(s.clientIDsByDevice, devKey)
		}
	}
	delete(s.adminClientIDs, client.id)
	delete(s.clients, client.id)

	player := s.getPlayerByClient(client)
	playerID := ""
	if player != nil {
		playerID = player.ID
	}
	// 一次性写完整一条 connection_events 行（见 activitylog.go），不再重复打到 stdout；
	// 即使这条连接从没关联到玩家（playerID 为空，比如连上就断的游客）也照样记一行。
	// 优雅关停时 closeLiveStateOnShutdown 可能已经写过并置 connectionLogged，这里跳过避免重复
	// （在线时长累加同样只做一次，避免双计）。
	if !client.connectionLogged {
		client.connectionLogged = true
		now := nowMs()
		s.logConnectionEvent(connectionEventPayload{
			socketID: client.id, connectedAt: client.connectedAt, disconnectedAt: now,
			sessionSID: client.sid, ip: client.ipAddress, device: client.deviceKey,
			fingerprint: client.fingerprint, userAgent: client.userAgent, compression: client.compression,
			playerID: playerID, closeReason: "disconnect",
		})
		s.accumulateClientOnlineMs(client, now)
	}
	if player == nil {
		return
	}
	s.serverStats.Disconnects++
	s.clearDisconnectHold(player)
	player.SocketID = ""

	player.graceGen++
	gen := player.graceGen
	player.graceTimer = timeAfterFunc(30*time.Second, func() {
		s.mu.Lock()
		defer s.mu.Unlock()
		current := s.players[playerID]
		if current == nil || current.SocketID != "" || current.graceGen != gen {
			return
		}
		current.graceTimer = nil
		current.Connected = false
		now := nowMs()
		current.DisconnectedAt = &now
		exp := now + 60_000
		current.DisconnectExpiresAt = &exp
		s.refreshPlayerSnapshots(current)
		if current.RoomID != "" {
			room := s.rooms[current.RoomID]
			if room != nil {
				s.createDisconnectForfeit(room, current)
				s.roomNotice(room, playerShortName(current)+" 断线了，保留座位 60 秒。")
				s.broadcastRoom(room.ID, false)
			}
		}
		s.broadcastPlayerUpdate(current)
		s.broadcastLobby()

		current.timerGen++
		tgen := current.timerGen
		current.discTimer = timeAfterFunc(60*time.Second, func() {
			s.mu.Lock()
			defer s.mu.Unlock()
			expired := s.players[playerID]
			if expired == nil || expired.Connected || expired.timerGen != tgen {
				return
			}
			if expired.RoomID != "" {
				room := s.rooms[expired.RoomID]
				if room != nil {
					s.applyDisconnectForfeit(room, expired)
				}
				s.leaveRoom(expired, LeaveDisconnectTimeout)
			}
			if expired.CurrentSID != "" && s.sidToPlayerID[expired.CurrentSID] == expired.ID {
				delete(s.sidToPlayerID, expired.CurrentSID)
			}
			if expired.Persistent {
				expired.LastSeenAt = nowMs()
				s.requestPersist("lazy")
			} else {
				delete(s.players, expired.ID)
				delete(s.tokenToPlayer, expired.Token)
			}
			expired.DisconnectExpiresAt = nil
			expired.discTimer = nil
			s.broadcastLobby()
		})
	})
}

// trustedProxyCount 表示前置可信反向代理层数（TRUSTED_PROXY_COUNT，默认 1）。
// X-Forwarded-For 由每一跳代理向右追加，客户端可伪造最左侧条目；因此真实客户端
// 位于从右数第 trustedProxyCount 个。设为 0 表示直连、完全忽略 XFF（防伪造）。
var trustedProxyCount = 1

func clientIP(r *http.Request) string {
	if trustedProxyCount > 0 {
		if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
			parts := strings.Split(xff, ",")
			idx := len(parts) - trustedProxyCount
			if idx < 0 {
				idx = 0
			}
			if ip := strings.TrimSpace(parts[idx]); ip != "" {
				return ip
			}
		}
	}
	host := r.RemoteAddr
	if i := strings.LastIndex(host, ":"); i >= 0 {
		h := host[:i]
		if h != "" {
			return h
		}
	}
	return host
}
