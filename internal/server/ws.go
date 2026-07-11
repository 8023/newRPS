package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

type wsEnvelope struct {
	E  string          `json:"e,omitempty"`
	ID json.RawMessage `json:"id,omitempty"`
	D  json.RawMessage `json:"d,omitempty"`
}

type wsResponse struct {
	ID  json.RawMessage `json:"id,omitempty"`
	D   any             `json:"d,omitempty"`
	Err string          `json:"err,omitempty"`
}

type wsPush struct {
	E string `json:"e"`
	D any    `json:"d"`
}

func (c *Client) writeJSON(v any) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()
	if c.closed {
		return fmt.Errorf("closed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.conn.Write(ctx, websocket.MessageText, data)
}

func (c *Client) reply(id json.RawMessage, data any, errMsg string) {
	if len(id) == 0 {
		return
	}
	resp := wsResponse{ID: id}
	if errMsg != "" {
		resp.Err = errMsg
	} else {
		resp.D = data
	}
	_ = c.writeJSON(resp)
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

func (s *Server) emitToRoom(room string, event string, data any) {
	msg := wsPush{E: event, D: data}
	for _, c := range s.clients {
		if c.inRoom(room) {
			_ = c.writeJSON(msg)
		}
	}
}

func (s *Server) emitVolatileAll(event string, data any) {
	msg := wsPush{E: event, D: data}
	for _, c := range s.clients {
		_ = c.writeJSON(msg)
	}
}

func (s *Server) emitToClient(clientID string, event string, data any) {
	c := s.clients[clientID]
	if c == nil {
		return
	}
	_ = c.writeJSON(wsPush{E: event, D: data})
}

func (s *Server) handleWS(w http.ResponseWriter, r *http.Request) {
	token := r.URL.Query().Get("token")
	session := s.verifySessionToken(token)
	ipAddress := clientIP(r)
	userAgent := r.UserAgent()
	origin := r.Header.Get("Origin")
	host := r.Host

	if !s.isAllowedOrigin(origin, host) {
		s.securityLog("socket_origin_blocked", map[string]any{"ip": ipAddress, "origin": origin, "userAgent": userAgent})
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
	socketsForIP := s.clientIDsByIP[ipAddress]
	if socketsForIP == nil {
		socketsForIP = map[string]struct{}{}
		s.clientIDsByIP[ipAddress] = socketsForIP
	}
	if len(socketsForIP) >= s.maxSocketsPerIP {
		s.mu.Unlock()
		s.securityLog("socket_ip_limited", map[string]any{"sid": session.SID, "ip": ipAddress, "userAgent": userAgent})
		http.Error(w, "Too many connections", http.StatusTooManyRequests)
		return
	}
	if prevID, ok := s.sidToClientID[session.SID]; ok {
		if prev := s.clients[prevID]; prev != nil {
			s.securityLog("socket_duplicate", map[string]any{"sid": session.SID, "ip": ipAddress, "oldSocketId": prevID, "userAgent": userAgent})
			_ = prev.conn.Close(websocket.StatusPolicyViolation, "replaced")
		}
	}
	s.mu.Unlock()

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		CompressionMode: websocket.CompressionContextTakeover,
		OriginPatterns:  []string{"*"},
	})
	if err != nil {
		return
	}
	conn.SetReadLimit(1_000_000)

	client := &Client{
		id:         randomID(),
		conn:       conn,
		sid:        session.SID,
		token:      token,
		sessionExp: session.Exp,
		ipAddress:  ipAddress,
		rooms:      map[string]struct{}{},
		userAgent:  userAgent,
		host:       host,
		origin:     origin,
	}

	s.mu.Lock()
	s.clients[client.id] = client
	s.sidToClientID[session.SID] = client.id
	s.clientIDToSID[client.id] = session.SID
	s.clientIDsByIP[ipAddress][client.id] = struct{}{}
	s.securityLog("socket_connected", map[string]any{"sid": session.SID, "ip": ipAddress, "socketId": client.id, "userAgent": userAgent})
	// initial pushes
	client.joinRoom(lobbyChannel)
	cfg := s.publicConfig()
	lobby := s.lobbySnapshot(false, true)
	s.mu.Unlock()

	_ = client.writeJSON(wsPush{E: "config:update", D: cfg})
	_ = client.writeJSON(wsPush{E: "lobby:update", D: lobby})

	ctx := r.Context()
	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			break
		}
		var env wsEnvelope
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}
		if env.E == "" {
			continue
		}
		s.handleWSEvent(client, env)
	}

	s.onClientDisconnect(client)
	_ = conn.Close(websocket.StatusNormalClosure, "")
}

func (s *Server) handleWSEvent(client *Client, env wsEnvelope) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// global flood backstop
	if !s.consumeRateLimit(fmt.Sprintf("socket:%s:%s", client.ipAddress, env.E), 60_000, 600) {
		client.reply(env.ID, nil, "操作过于频繁，请稍后再试")
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
	if s.sidToClientID[sid] == client.id {
		delete(s.sidToClientID, sid)
	}
	delete(s.clientIDToSID, client.id)
	if set := s.clientIDsByIP[ipAddress]; set != nil {
		delete(set, client.id)
		if len(set) == 0 {
			delete(s.clientIDsByIP, ipAddress)
		}
	}
	delete(s.adminClientIDs, client.id)
	delete(s.clients, client.id)

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
		// keep IPv6 bracket forms roughly
		h := host[:i]
		if h != "" {
			return h
		}
	}
	return host
}
