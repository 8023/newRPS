package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
	"github.com/doumiao/newRPS/internal/wire"
	"google.golang.org/protobuf/proto"
)

type wsEnvelope struct {
	E  string          `json:"e,omitempty"`
	ID json.RawMessage `json:"id,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

func (c *Client) writeBinary(data []byte) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return fmt.Errorf("closed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.conn.Write(ctx, websocket.MessageBinary, data)
}

// reply 通过 protobuf RAW 信封返回 RPC 应答（仅 id/err/raw）。
func (c *Client) reply(id json.RawMessage, data any, errMsg string) {
	if len(id) == 0 {
		return
	}
	var reqID int64
	_ = json.Unmarshal(id, &reqID)
	if reqID == 0 {
		// 兼容字符串 id
		var s string
		if json.Unmarshal(id, &s) == nil {
			reqID, _ = strconv.ParseInt(s, 10, 64)
		}
	}
	env := &wire.Envelope{Id: reqID, Kind: wire.PayloadKind_KIND_RAW, Err: errMsg}
	if errMsg == "" && data != nil {
		b, err := json.Marshal(data)
		if err != nil {
			return
		}
		env.Raw = b
	}
	out, err := proto.Marshal(env)
	if err != nil {
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
			s.securityLog("socket_duplicate", map[string]any{"sid": session.SID, "ip": ipAddress, "device": prev.deviceKey, "oldSocketId": prevID, "userAgent": userAgent})
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
			prev.closed = true
			prev.replaced = true
			prevConn := prev.conn
			s.mu.Unlock()
			_ = prevConn.Close(websocket.StatusPolicyViolation, "replaced")
			s.mu.Lock()
		}
	}
	socketsForDevice := s.clientIDsByDevice[devKey]
	if socketsForDevice == nil {
		socketsForDevice = map[string]struct{}{}
		s.clientIDsByDevice[devKey] = socketsForDevice
	}
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
	}

	s.mu.Lock()
	s.clients[client.id] = client
	s.sidToClientID[session.SID] = client.id
	s.clientIDToSID[client.id] = session.SID
	s.clientIDsByDevice[devKey][client.id] = struct{}{}
	s.securityLog("socket_connected", map[string]any{
		"sid": session.SID, "ip": ipAddress, "device": devKey, "fp": fingerprint,
		"socketId": client.id, "userAgent": userAgent, "compression": compressionModeName(compMode),
	})
	client.joinRoom(lobbyChannel)
	cfg := s.publicConfig()
	lobby := s.lobbySnapshot(false, true)
	s.mu.Unlock()

	if env, _, err := s.buildFullEnvelope("config:update", channelConfig(), cfg); err == nil {
		s.emitWireClient(client, env)
	}
	if env, _, err := s.buildFullEnvelope("lobby:update", channelLobby(), lobby); err == nil {
		s.emitWireClient(client, env)
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
		if err := proto.Unmarshal(data, &wenv); err != nil {
			// 兼容旧 JSON 文本帧
			var env wsEnvelope
			if json.Unmarshal(data, &env) != nil || env.E == "" {
				continue
			}
			s.handleWSEvent(client, env)
			continue
		}
		if wenv.Event == "" {
			continue
		}
		// 应用层心跳：客户端周期性 ping，带 id 则 RPC 应答，保持链路活跃（配合 iOS/反代）
		if wenv.Event == "ping" {
			idRaw, _ := json.Marshal(wenv.Id)
			if wenv.Id != 0 {
				client.reply(idRaw, map[string]any{"t": nowMs()}, "")
			}
			continue
		}
		idRaw, _ := json.Marshal(wenv.Id)
		s.handleWSEvent(client, wsEnvelope{E: wenv.Event, ID: idRaw, D: wenv.Raw})
	}

	cancelPing()
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
	if !s.checkRateLimit(rateLimitKey(env.E, client.ipAddress, client.sid), opts) {
		s.securityLog("rate_limit", map[string]any{"sid": client.sid, "ip": client.ipAddress, "event": env.E, "userAgent": client.userAgent})
		client.reply(env.ID, nil, "操作过于频繁，请稍后再试")
		return
	}
	handler(client, env)
}

type eventHandlerFunc func(client *Client, env wsEnvelope)

func (s *Server) onClientDisconnect(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	client.closed = true
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

	// 被同 SID 新连接顶替：只清连接表，不要把玩家打成离线（否则 Safari 重连会闪断+人数错乱）
	if client.replaced {
		return
	}

	player := s.getPlayerByClientID(client.id)
	if player == nil {
		return
	}
	s.securityLog("socket_disconnected", map[string]any{"sid": player.ID, "ip": ipAddress, "socketId": client.id, "userAgent": client.userAgent})
	s.serverStats.Disconnects++
	s.clearDisconnectHold(player)
	player.SocketID = ""

	playerID := player.ID
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

func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
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
