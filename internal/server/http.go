package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
		w.Header().Set("Content-Security-Policy", "default-src 'self'; base-uri 'self'; object-src 'none'; frame-ancestors 'none'; img-src 'self' data: blob:; style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self' ws: wss:;")
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
	if xf := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); xf != "" {
		// 可能是 "a.com, b.com"
		if i := strings.Index(xf, ","); i >= 0 {
			xf = xf[:i]
		}
		return strings.TrimSpace(xf)
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

func (s *Server) isAllowedOrigin(origin, requestHost string) bool {
	if origin == "" {
		return true
	}
	origin = strings.TrimSpace(origin)
	for _, item := range strings.Split(os.Getenv("ALLOWED_ORIGINS"), ",") {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		// 完整 Origin 或仅域名都算匹配
		if item == origin || hostOnly(item) == hostOnly(origin) {
			return true
		}
	}
	return s.sameHostOrLocalDev(origin, requestHost)
}

func (s *Server) sameHostOrLocalDev(origin, requestHost string) bool {
	originHost := hostOnly(origin)
	localHosts := map[string]bool{"localhost": true, "127.0.0.1": true, "::1": true}
	if localHosts[originHost] {
		return true
	}
	// 忽略端口差异：Origin=https://rps.rbq.io 与 Host=rps.rbq.io:443 应放行
	reqHost := hostOnly(requestHost)
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
		// JSON body 可选 { "fingerprint": "..." }
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
	ok := s.checkRateLimit("session:"+devKey, RateLimitOptions{Limit: 10, WindowMs: 60_000, CooldownMs: 60_000})
	if !ok {
		s.mu.Unlock()
		s.securityLog("token_issue_limited", map[string]any{"ip": ipAddress, "device": devKey, "userAgent": r.UserAgent()})
		writeJSON(w, http.StatusTooManyRequests, map[string]string{"message": "请求过于频繁，请稍后再试"})
		return
	}
	token := s.issueSessionToken()
	payload := s.verifySessionToken(token)
	s.securityLog("token_issued", map[string]any{"sid": payload.SID, "ip": ipAddress, "device": devKey, "userAgent": r.UserAgent()})
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
	s.mu.Unlock()

	file, header, err := r.FormFile("image")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	defer file.Close()
	// 后缀名：非 .webp 直接拒绝（用户可见错误）
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
	writeJSON(w, http.StatusOK, map[string]string{"imageUrl": url})
}

func (s *Server) handleAdminImage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 8*1024*1024+1024)
	if err := r.ParseMultipartForm(8 * 1024 * 1024); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片上传失败，请确认格式为 jpg/png/webp 且小于 8MB"})
		return
	}
	password := r.FormValue("password")
	s.mu.Lock()
	ok := s.adminPasswordMatches(password)
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
	buf, err := io.ReadAll(file)
	if err != nil || len(buf) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片格式不支持或图片为空"})
		return
	}
	ct := header.Header.Get("Content-Type")
	mime, _, okKind := imageKind(buf)
	if !okKind || (mime != "image/jpeg" && mime != "image/png" && mime != "image/webp") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片真实格式不正确，请上传 jpg/png/webp"})
		return
	}
	if ct != mime {
		ct = mime
	}
	url, err := s.saveVerifiedImage(buf, ct, "admin")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"message": "图片真实格式不正确，请上传 jpg/png/webp"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"imageUrl": url})
}

func (s *Server) handleConfigExport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	password := r.URL.Query().Get("password")
	s.mu.Lock()
	ok := s.adminPasswordMatches(password)
	s.mu.Unlock()
	if !ok {
		writeJSON(w, http.StatusForbidden, map[string]string{"message": "Admin password is required"})
		return
	}
	text, err := exportConfigText()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"message": err.Error()})
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write([]byte(text))
}

// exportConfigText is wired in persist/server from config package
var exportConfigText = func() (string, error) { return "", fmt.Errorf("not wired") }

func (s *Server) serveStatic(w http.ResponseWriter, r *http.Request) {
	if strings.HasPrefix(r.URL.Path, "/uploads/") {
		// security headers for uploads
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Content-Security-Policy", "default-src 'none'; img-src 'self';")
		w.Header().Set("Cache-Control", "public, max-age=2592000, immutable")
		http.StripPrefix("/uploads/", http.FileServer(http.Dir(s.uploadsDir))).ServeHTTP(w, r)
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
		w.Header().Set("Cache-Control", "no-cache")
		http.ServeFile(w, r, filepath.Join(s.distDir, "index.html"))
		return
	}
	// hashed assets get long cache
	base := filepath.Base(full)
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

func (s *Server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/session", s.handleSession)
	mux.Handle("/api/proof-image", s.httpRateLimit("proof-image", 60_000, 20)(http.HandlerFunc(s.handleProofImage)))
	mux.Handle("/api/admin-image", s.httpRateLimit("admin-image", 60_000, 12)(http.HandlerFunc(s.handleAdminImage)))
	mux.HandleFunc("/api/config/export", s.handleConfigExport)
	mux.HandleFunc("/ws", s.handleWS)
	mux.HandleFunc("/", s.serveStatic)

	var h http.Handler = mux
	h = s.httpRateLimit("http", 60_000, 240)(h)
	h = s.requireTrustedOrigin(h)
	h = s.securityHeaders(h)
	return h
}
