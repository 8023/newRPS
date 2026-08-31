package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func (s *Server) securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// WebSocket 升级响应尽量少塞响应头，部分 Safari 在 101 上对额外安全头更敏感。
		if strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
			next.ServeHTTP(w, r)
			return
		}
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		// worker-src blob:：heic2any 等库用 blob Worker 解码 HEIC
		// wasm-unsafe-eval：@jsquash/webp 等 WASM 编码器
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; "+
				"img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; "+
				"script-src 'self' 'wasm-unsafe-eval'; worker-src 'self' blob:; "+
				"connect-src 'self' ws: wss:;")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) requireTrustedOrigin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet || r.Method == http.MethodHead || r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		if s.isAllowedOrigin(r.Header.Get("Origin"), publicRequestHost(r)) {
			next.ServeHTTP(w, r)
			return
		}
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "Request origin is not allowed"})
	})
}

func (s *Server) httpRateLimit(scope string, windowMs int64, max int) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			s.mu.Lock()
			ok := s.consumeRateLimit(scope+":"+clientIP(r), windowMs, max)
			s.mu.Unlock()
			if !ok {
				writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "Too many requests, please try again later"})
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// publicRequestHost 反代场景优先 X-Forwarded-Host（可能是 rps.example.com），
// 否则用 r.Host（直连时是浏览器看到的 Host）。
func publicRequestHost(r *http.Request) string {
	// 直连（无可信代理）时不信任客户端可伪造的 X-Forwarded-Host，直接用 r.Host。
	if trustedProxyCount > 0 {
		if xf := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xf != "" {
			// 可能是 "a.com, b.com"
			if i := strings.Index(xf, ","); i >= 0 {
				xf = xf[:i]
			}
			return strings.TrimSpace(xf)
		}
	}
	return r.Host
}

func hostOnly(hostport string) string {
	h := strings.TrimSpace(hostport)
	h = strings.TrimPrefix(h, "https://")
	h = strings.TrimPrefix(h, "http://")
	if i := strings.Index(h, "/"); i >= 0 {
		h = h[:i]
	}
	// host:port 或 [ipv6]:port
	if strings.HasPrefix(h, "[") {
		if i := strings.Index(h, "]"); i >= 0 {
			return h[1:i]
		}
	}
	if i := strings.Index(h, ":"); i >= 0 {
		return h[:i]
	}
	return h
}

// schemeOf 提取 URL 的 scheme（"https"/"http"），没有 "://" 时视为未指定（配置里填了裸域名）。
func schemeOf(raw string) string {
	raw = strings.TrimSpace(raw)
	if i := strings.Index(raw, "://"); i >= 0 {
		return strings.ToLower(raw[:i])
	}
	return ""
}

// refererOrigin 从 Referer 请求头（完整 URL）提取出 "scheme://host" 形式的来源，
// 与浏览器 Origin 请求头同构，供 isAllowedOrigin 复用同一套白名单/本地开发判断逻辑。
// 解析失败或没有 scheme+host（比如 Referer 缺失、或是形如 "about:blank" 的非常规值）
// 一律返回空串——isAllowedOrigin 对空 Origin 本就判定为不放行，行为与"没有可信来源"一致。
func refererOrigin(referer string) string {
	referer = strings.TrimSpace(referer)
	if referer == "" {
		return ""
	}
	u, err := url.Parse(referer)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ""
	}
	return u.Scheme + "://" + u.Host
}

func (s *Server) isAllowedOrigin(origin, requestHost string) bool {
	// 浏览器对 fetch/WS 这类非安全方法请求始终会带 Origin；requireTrustedOrigin 已经对
	// GET/HEAD/OPTIONS 跳过了这层检查。空 Origin 不再直接放行，避免非浏览器脚本绕过。
	if origin == "" {
		return false
	}
	origin = strings.TrimSpace(origin)
	originScheme := schemeOf(origin)
	for _, item := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if item == origin {
			return true
		}
		// 仅域名（未带 scheme）时不限制协议；显式带了 scheme 就必须和 Origin 一致，
		// 避免管理员配置的 https://example.com 被 http://example.com 复用。
		if hostOnly(item) == hostOnly(origin) {
			if itemScheme := schemeOf(item); itemScheme == "" || itemScheme == originScheme {
				return true
			}
		}
	}
	return s.sameHostOrLocalDev(origin, requestHost)
}

func (s *Server) sameHostOrLocalDev(origin, requestHost string) bool {
	originHost := hostOnly(origin)
	reqHost := hostOnly(requestHost)
	localHosts := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}
	// 本地开发：Origin 与请求 Host 必须都是本机地址才放行——否则任何请求只要声称
	// Origin=http://localhost 就能绕过生产环境（Host 是公网域名）的校验。
	if localHosts[originHost] && localHosts[reqHost] {
		return true
	}
	// 忽略端口差异：Origin=https://rps.rbq.io 与 Host=rps.rbq.io:443 应放行
	if reqHost != "" && originHost == reqHost {
		return true
	}
	serverHost := hostOnly(s.host)
	if serverHost != "" && serverHost != "0.0.0.0" && serverHost != "::" && originHost == serverHost {
		return true
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *Server) handleSession(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	ipAddress := clientIP(r)
	fp := r.Header.Get("X-Browser-Fingerprint")
	if fp == "" {
		// JSON body 可选 { "fingerprint": "..." }；上限硬顶 body 大小，避免无认证的匿名端点
		// 被灌超大/超长 body 消耗内存带宽（其它 POST 接口都已有 MaxBytesReader，这里补齐）。
		r.Body = http.MaxBytesReader(w, r.Body, 4*1024)
		var body struct {
			Fingerprint string `json:"fingerprint"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		fp = body.Fingerprint
	}
	devKey := deviceKey(ipAddress, fp)
	if !s.isAllowedOrigin(r.Header.Get("Origin"), publicRequestHost(r)) {
		s.securityLog("session_origin_blocked", map[string]any{"ip": ipAddress, "origin": r.Header.Get("Origin"), "userAgent": r.UserAgent()})
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "Origin not allowed"})
		return
	}
	s.mu.Lock()
	// devKey（IP+指纹）可被客户端随意伪造指纹绕过；额外加一层纯 IP 兜底桶，
	// 使得同一出口 IP 能换发的 session 总量有硬上限，不受指纹伪造影响。
	devOK := s.checkRateLimit("session:"+devKey, RateLimitOptions{Limit: 10, WindowMs: 60_000, CooldownMs: 60_000})
	ipOK := s.checkRateLimit("session:ip:"+ipAddress, RateLimitOptions{Limit: s.cfg.AccessControl.MaxSessionIssuePerIP, WindowMs: 600_000, CooldownMs: 600_000})
	if !devOK || !ipOK {
		s.mu.Unlock()
		s.securityLog("token_issue_limited", map[string]any{"ip": ipAddress, "device": devKey, "userAgent": r.UserAgent()})
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "请求过于频繁，请稍后再试"})
		return
	}
	token := s.issueSessionToken()
	payload := s.verifySessionToken(token)
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"token": token, "expiresAt": payload.Exp})
}

func imageKind(buf []byte) (mime, ext string, ok bool) {
	if len(buf) >= 3 && buf[0] == 0xff && buf[1] == 0xd8 && buf[2] == 0xff {
		return "image/jpeg", ".jpg", true
	}
	if len(buf) >= 8 && buf[0] == 0x89 && buf[1] == 0x50 && buf[2] == 0x4e && buf[3] == 0x47 {
		return "image/png", ".png", true
	}
	if len(buf) >= 12 && string(buf[0:4]) == "RIFF" && string(buf[8:12]) == "WEBP" {
		return "image/webp", ".webp", true
	}
	return "", "", false
}

func (s *Server) saveVerifiedImage(buf []byte, contentType, bucket string) (string, error) {
	mime, ext, ok := imageKind(buf)
	if !ok || mime != contentType {
		return "", fmt.Errorf("图片真实格式不正确")
	}
	filename := fmt.Sprintf("%d-%s%s", time.Now().UnixMilli(), randomID(), ext)
	targetDir := s.proofUploadsDir
	if bucket == "admin" {
		targetDir = s.adminUploadsDir
	} else if bucket == "avatars" {
		targetDir = s.avatarUploadsDir
	} else if bucket == "contributions" {
		targetDir = s.contribUploadsDir
	}
	path := filepath.Join(targetDir, filename)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return "", err
	}
	defer f.Close()
	if _, err := f.Write(buf); err != nil {
		return "", err
	}
	return "/uploads/" + bucket + "/" + filename, nil
}

func (s *Server) handleProofImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 兜底：仅接受前端已压好的 WebP，且不超过 2MB
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024+1024)
	if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片过大或格式错误，仅支持 webp 且不超过 2MB"})
		return
	}
	token := r.FormValue("token")
	s.mu.Lock()
	session := s.verifySessionToken(token)
	var player *PlayerState
	if session != nil {
		if pid := s.tokenToPlayer[token]; pid != "" {
			player = s.players[pid]
		}
		if player != nil && s.sidToPlayerID[session.SID] != player.ID {
			player = nil
		}
	}
	if session == nil || player == nil {
		s.securityLog("upload_denied", map[string]any{"sid": "", "ip": clientIP(r), "event": "proof-image", "userAgent": r.UserAgent()})
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "Invalid session"})
		return
	}
	if !player.Connected {
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "请先进入游戏后再上传证明"})
		return
	}
	// 路由层的 proof-image 限流是纯 IP 维度的，躲不开"分散在多个 IP 背后用同一批账号"的攻击；
	// 这里再按玩家维度加一层限流。
	if !s.checkRateLimit("proof-image:player:"+player.ID, RateLimitOptions{Limit: s.cfg.AccessControl.MaxProofUploadsPerPlayer, WindowMs: 600_000, CooldownMs: 600_000}) {
		s.securityLog("upload_denied", map[string]any{"sid": session.SID, "ip": clientIP(r), "event": "proof-image", "reason": "player_rate_limit", "userAgent": r.UserAgent()})
		s.mu.Unlock()
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "上传过于频繁，请稍后再试"})
		return
	}
	// 记录上传时所在房间，供 serveStatic 做"房间已销毁则拒绝访问"判断——必须在这里（拿到
	// player 的临界区内）取值，出了这段锁 player.RoomID 随时可能因为玩家离房而变化。
	roomID := player.RoomID
	s.mu.Unlock()

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	defer file.Close()
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".webp") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "仅支持 webp 格式，请使用前端压缩后的图片"})
		return
	}
	buf, err := io.ReadAll(file)
	if err != nil || len(buf) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	if len(buf) > 2*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片不能超过 2MB，请压缩后再上传"})
		return
	}
	mime, _, ok := imageKind(buf)
	if !ok || mime != "image/webp" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片内容不是有效的 webp，请重新选择图片"})
		return
	}
	url, err := s.saveVerifiedImage(buf, "image/webp", "proofs")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片真实格式不正确，请上传 webp"})
		return
	}
	s.mu.Lock()
	s.proofImageRooms[filepath.Base(url)] = roomID
	s.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]string{"imageUrl": url})
}

// handleAvatarImage：前端已固定压成正方形 WebP，服务端只需校验后缀/体积/真实格式。
// 传 clear=1 时清空头像，恢复默认首字头像。
func (s *Server) handleAvatarImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 20*1024+1024)
	ct := r.Header.Get("Content-Type")
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(20 * 1024); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片过大或格式错误，仅支持 webp 且不超过 20KB"})
			return
		}
	} else if err := r.ParseForm(); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "请求格式错误"})
		return
	}
	token := r.FormValue("token")
	clearAvatar := r.FormValue("clear") == "1"
	ipAddress := clientIP(r)
	fingerprint := normalizeFingerprint(r.Header.Get("X-Browser-Fingerprint"))
	devKey := deviceKey(ipAddress, fingerprint)
	s.mu.Lock()
	session := s.verifySessionToken(token)
	var player *PlayerState
	if session != nil {
		if pid := s.tokenToPlayer[token]; pid != "" {
			player = s.players[pid]
		}
		if player != nil && s.sidToPlayerID[session.SID] != player.ID {
			player = nil
		}
	}
	if session == nil || player == nil {
		s.securityLog("upload_denied", map[string]any{"sid": "", "ip": clientIP(r), "event": "avatar-image", "userAgent": r.UserAgent()})
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "Invalid session"})
		return
	}
	if !player.Connected {
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "请先登录后再上传头像"})
		return
	}
	if clearAvatar {
		oldAvatarURL := player.AvatarURL
		player.AvatarURL = ""
		if oldAvatarURL != "" {
			// 清除头像并入「更换头像」活动埋点计数，不再单独区分 clear/change。
			s.logPlayerActivity("avatar_change", player.ID, "", oldAvatarURL, ipAddress, devKey, fingerprint, "")
		}
		s.refreshPlayerSnapshots(player)
		s.broadcastPlayerUpdate(player)
		shouldPersist := player.Persistent
		if shouldPersist {
			s.markPlayerDirty(player)
		}
		pub := s.publicPlayer(player)
		s.mu.Unlock()
		if shouldPersist {
			s.requestPersist("important")
		}
		writeJSON(w, http.StatusOK, map[string]any{"avatarUrl": "", "player": pub})
		return
	}
	s.mu.Unlock()

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	defer file.Close()
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".webp") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "仅支持 webp 格式，请使用前端压缩后的图片"})
		return
	}
	buf, err := io.ReadAll(file)
	if err != nil || len(buf) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	if len(buf) > 20*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "头像图片不能超过 20KB，请压缩后再上传"})
		return
	}
	mime, _, ok := imageKind(buf)
	if !ok || mime != "image/webp" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片内容不是有效的 webp，请重新选择图片"})
		return
	}
	url, err := s.saveVerifiedImage(buf, "image/webp", "avatars")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片真实格式不正确，请上传 webp"})
		return
	}

	s.mu.Lock()
	oldAvatarURL := player.AvatarURL
	player.AvatarURL = url
	s.logPlayerActivity("avatar_change", player.ID, url, oldAvatarURL, ipAddress, devKey, fingerprint, "")
	s.refreshPlayerSnapshots(player)
	s.broadcastPlayerUpdate(player)
	shouldPersist := player.Persistent
	if shouldPersist {
		s.markPlayerDirty(player)
	}
	pub := s.publicPlayer(player)
	s.mu.Unlock()
	if shouldPersist {
		// 头像变更立即落盘，避免刷新/重启后丢失
		s.requestPersist("important")
	}
	writeJSON(w, http.StatusOK, map[string]any{"avatarUrl": url, "player": pub})
}

func (s *Server) handleAdminImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	// 与证明图一致：仅接受前端已压好的 WebP，且不超过 2MB
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024+1024)
	if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片过大或格式错误，仅支持 webp 且不超过 2MB"})
		return
	}
	password := r.FormValue("password")
	s.mu.Lock()
	ok := s.adminPasswordMatches(password, clientIP(r))
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "管理员口令不正确或尚未设置"})
		return
	}
	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	defer file.Close()
	filename := strings.ToLower(header.Filename)
	if !strings.HasSuffix(filename, ".webp") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "仅支持 webp 格式，请使用前端压缩后的图片"})
		return
	}
	buf, err := io.ReadAll(file)
	if err != nil || len(buf) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	if len(buf) > 2*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片不能超过 2MB，请压缩后再上传"})
		return
	}
	mime, _, okKind := imageKind(buf)
	if !okKind || mime != "image/webp" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片内容不是有效的 webp，请重新选择图片"})
		return
	}
	url, err := s.saveVerifiedImage(buf, "image/webp", "admin")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片真实格式不正确，请上传 webp"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"imageUrl": url})
}

func (s *Server) handleContributionImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 2*1024*1024+1024)
	if err := r.ParseMultipartForm(2 * 1024 * 1024); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片过大或格式错误，仅支持 webp 且不超过 2MB"})
		return
	}
	token := r.FormValue("token")
	s.mu.Lock()
	session := s.verifySessionToken(token)
	var player *PlayerState
	if session != nil {
		if pid := s.tokenToPlayer[token]; pid != "" {
			player = s.players[pid]
		}
		if player != nil && s.sidToPlayerID[session.SID] != player.ID {
			player = nil
		}
	}
	if session == nil || player == nil || !player.Persistent {
		s.securityLog("upload_denied", map[string]any{"sid": "", "ip": clientIP(r), "event": "contribution-image", "userAgent": r.UserAgent()})
		s.mu.Unlock()
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "请先登录后再上传"})
		return
	}
	if !s.checkRateLimit("contribution-image:player:"+player.ID, RateLimitOptions{Limit: 12, WindowMs: 600_000, CooldownMs: 60_000}) {
		s.mu.Unlock()
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "上传过于频繁，请稍后再试"})
		return
	}
	s.mu.Unlock()
	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	defer file.Close()
	if !strings.HasSuffix(strings.ToLower(header.Filename), ".webp") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "仅支持 webp 格式"})
		return
	}
	buf, err := io.ReadAll(file)
	if err != nil || len(buf) == 0 || len(buf) > 2*1024*1024 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片不能超过 2MB"})
		return
	}
	mime, _, ok := imageKind(buf)
	if !ok || mime != "image/webp" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片内容不是有效的 webp"})
		return
	}
	url, err := s.saveVerifiedImage(buf, "image/webp", "contributions")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "保存失败"})
		return
	}
	// 不再有独立的图片登记表：这张图此刻只是落了盘，要等玩家把它填进草稿的
	// background_image 字段并保存才会被 imageIsLive 判定为"活的"（见
	// contributionImageAccessible）；上传成功但从未被任何草稿引用的文件永久留在磁盘上
	// 不对外可访问，不需要显式回滚删除。
	writeJSON(w, http.StatusOK, map[string]string{"imageUrl": url})
}

// noDirListingFS 禁止目录列表：只允许打开具体文件，访问 /uploads/ 或子目录本身返回 404。
type noDirListingFS struct{ root http.FileSystem }

func (fs noDirListingFS) Open(name string) (http.File, error) {
	f, err := fs.root.Open(name)
	if err != nil {
		return nil, err
	}
	stat, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, err
	}
	if stat.IsDir() {
		_ = f.Close()
		return nil, os.ErrNotExist
	}
	return f, nil
}

// contributionImageAccessible 判断某张共建封面图当前是否仍是它所属投稿"最新版本"引用的
// 图（草稿/待审/已通过/被驳回/已撤回都算数，不看状态），只有"重新上传另一张图覆盖掉"
// 产生的孤儿文件才拒绝——图片文件本身永远不硬删，只是不再对外可访问。走独立只读连接
// （不可用时退回主连接），避免这条高频的静态资源请求占用主写连接（SetMaxOpenConns(1)）。
func (s *Server) contributionImageAccessible(path string) bool {
	if s.contributionStore == nil {
		return false
	}
	db := s.altAccountsReadDB()
	ts := s.contributionStore.tasks
	if db != s.db && db != nil {
		ts = newSubTaskStore(db)
	}
	live, err := ts.imageIsLive(path)
	if err != nil {
		s.errorLog("contribution_image_check_failed", err.Error())
		return false
	}
	return live
}

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/uploads/") {
		// 上传目录（尤其证明图，可能涉及真人隐私内容）的通用访问门槛是文件名不可猜测；
		// 共建图又因监管审计要求永久留存，因此文件名一旦泄露就可能被长期访问。证明图另有
		// 下方的房间存活校验，但两类文件都不能把随机文件名当成真正的身份鉴权。
		// 这里加一层 Referer 校验作为纵深防御：要求请求带着"来自本站页面"的 Referer，
		// 复用 isAllowedOrigin 同一套白名单/本地开发判断（Referer 缺失按空 Origin 处理，
		// 直接拒绝）。正常使用场景下图片都是站内 <img> 子资源加载，同源请求下浏览器默认
		// 会带上 Referer（全局 securityHeaders 设置的 Referrer-Policy: same-origin 允许
		// 同源请求携带），不影响正常展示；能拦住的是站外直接粘贴 URL 打开、跨站热链等场景。
		// ⚠️ 这不是真正的身份鉴权——Referer 是请求方自己声明的，非浏览器客户端（curl 等）
		// 可以随意伪造，只能挡住"浏览器里走正常链接跳转/嵌入"这一类场景，见 review.md 1.2。
		if !s.isAllowedOrigin(refererOrigin(r.Header.Get("Referer")), publicRequestHost(r)) {
			s.securityLog("upload_referer_blocked", map[string]any{
				"ip": clientIP(r), "path": r.URL.Path, "userAgent": r.UserAgent(),
			})
			http.NotFound(w, r)
			return
		}
		// 证明图额外一层限制：房间是纯内存态、不落盘，一旦销毁（正常清空/管理员强制关房/
		// 进程重启）就不该再能看到当时的证明图，不管 Referer 是否合法——作者评估后认为这个
		// 场景（本站是非盈利小站，流量成本比防泄露更值得优先，见下面 Cache-Control 的取舍）
		// 比"细水长流地防陌生人拿到链接"更重要，值得单独拦一道。s.proofImageRooms 记录了
		// 文件名在上传时归属的房间 ID，这里只需要确认那个房间现在还活着；找不到映射（例如
		// 进程重启后，房间状态本就不落盘）同样按"房间已不存在"处理，一律拒绝。
		if strings.HasPrefix(r.URL.Path, "/uploads/proofs/") {
			filename := filepath.Base(r.URL.Path)
			s.mu.Lock()
			roomID, mapped := s.proofImageRooms[filename]
			roomAlive := mapped && s.rooms[roomID] != nil
			s.mu.Unlock()
			if !roomAlive {
				s.securityLog("upload_room_gone_blocked", map[string]any{
					"ip": clientIP(r), "path": r.URL.Path, "userAgent": r.UserAgent(),
				})
				http.NotFound(w, r)
				return
			}
		}
		// 共建封面图额外一层限制：只要这张图当前仍是某份投稿引用的封面就放行，不管那份投稿
		// 是草稿、待审、已通过还是被驳回/撤回——状态完全不影响；只有"上传后又被重新上传
		// 覆盖替换掉"产生的孤儿文件才拒绝（见 contributionImageAccessible 的注释）。图片
		// 文件本身永远不硬删，只是不再对外可访问。
		if strings.HasPrefix(r.URL.Path, "/uploads/contributions/") && !s.contributionImageAccessible(r.URL.Path) {
			s.securityLog("upload_orphaned_image_blocked", map[string]any{
				"ip": clientIP(r), "path": r.URL.Path, "userAgent": r.UserAgent(),
			})
			http.NotFound(w, r)
			return
		}
		// security headers for uploads
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self';")
		// 这是非盈利小站，流量成本比"链接泄露后被陌生人打开"的风险更优先——因此仍标 public
		// 让 CDN/反代等共享缓存能扛流量（而不是像纵深防御该有的做法那样标 private 强制回源，
		// 参考 review.md 1.2 与上面的讨论）。用较短的 max-age（6 小时，多数房间的存活时长）
		// 把"链接泄露后经共享缓存仍可访问"的窗口从原来的 30 天大幅收窄；证明图这一路还额外
		// 有上面那层"房间已销毁则拒绝"的校验兜底——只是那层校验只在缓存未命中、真正回源时才
		// 会执行，6 小时内命中共享缓存的请求仍绕得开它，两者是互补而非互相替代的关系。
		cache := "public, max-age=21600, immutable"
		if strings.HasPrefix(r.URL.Path, "/uploads/contributions/") {
			cache = "public, max-age=2592000, immutable"
		}
		w.Header().Set("Cache-Control", cache)
		// 禁止目录列出：只能用已知文件名访问单文件（证明图/头像 URL）。
		http.StripPrefix("/uploads/", http.FileServer(noDirListingFS{root: http.Dir(s.uploadsDir)})).ServeHTTP(w, r)
		return
	}
	if s.distDir == "" {
		http.NotFound(w, r)
		return
	}
	path := r.URL.Path
	if path == "/" || path == "" {
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(s.distDir, "index.html"))
		return
	}
	full := filepath.Join(s.distDir, filepath.Clean("/"+path))
	if !strings.HasPrefix(full, s.distDir) {
		http.NotFound(w, r)
		return
	}
	fi, err := os.Stat(full)
	if err != nil || fi.IsDir() {
		// /assets/ 下是带 hash 的构建产物（js/wasm/css），缺失多半是浏览器还拿着旧 index.html
		// 引用了上一次构建的文件名——应该让它 404 触发浏览器报错/重试，而不是悄悄回退成
		// index.html（text/html），那样动态 import() 会报 "'text/html' is not a valid
		// JavaScript MIME type" 这种难排查的错误。SPA 客户端路由（如 /room/xyz）仍走下面的回退。
		if strings.HasPrefix(path, "/assets/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(s.distDir, "index.html"))
		return
	}
	// hashed assets get long cache
	base := filepath.Base(full)
	// Service Worker 与 Manifest 必须及时更新，不能落入下面对普通静态文件的一年 immutable。
	isServiceWorker := base == "push-sw-v3.js"
	if isServiceWorker || base == "manifest.webmanifest" {
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.Header().Set("CDN-Cache-Control", "no-store")
		w.Header().Set("Cloudflare-CDN-Cache-Control", "no-store")
		w.Header().Set("Expires", "0")
		if isServiceWorker {
			w.Header().Set("Service-Worker-Allowed", "/")
		}
		http.ServeFile(w, r, full)
		return
	}
	if strings.Contains(base, ".") && (strings.Contains(base, "-") || strings.Contains(base, ".")) {
		// assets with hash typically have long names with dots
		if filepath.Ext(base) != ".html" {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		}
	}
	if strings.HasSuffix(base, ".html") {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFile(w, r, full)
}

// handlePushVapidKey：VAPID 公钥是公开信息（Web Push 协议本来就要求浏览器端拿到它去订阅），
// 不需要鉴权，纯 GET。
func (s *Server) handlePushVapidKey(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	if s.vapid.PublicKey == "" {
		http.Error(w, "push unavailable", http.StatusServiceUnavailable)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"publicKey":       s.vapid.PublicKey,
		"protocolVersion": pushProtocolVersion,
	})
}

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/session", s.handleSession)
	mux.HandleFunc("/api/push/vapid-key", s.handlePushVapidKey)
	mux.Handle("/api/proof-image", s.httpRateLimit("proof-image", 60_000, 20)(http.HandlerFunc(s.handleProofImage)))
	mux.Handle("/api/avatar-image", s.httpRateLimit("avatar-image", 60_000, 12)(http.HandlerFunc(s.handleAvatarImage)))
	mux.Handle("/api/admin-image", s.httpRateLimit("admin-image", 60_000, 12)(http.HandlerFunc(s.handleAdminImage)))
	mux.Handle("/api/contribution-image", s.httpRateLimit("contribution-image", 60_000, 12)(http.HandlerFunc(s.handleContributionImage)))
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/", s.serveStatic)

	var h http.Handler = mux
	h = s.httpRateLimit("http", 60_000, 240)(h)
	h = s.requireTrustedOrigin(h)
	h = s.securityHeaders(h)
	return h
}
