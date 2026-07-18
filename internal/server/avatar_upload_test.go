package server

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// fakeWebpBytes 构造一段 imageKind() 能识别为 image/webp 的最小字节流（无需真正可解码）。
func fakeWebpBytes(payloadSize int) []byte {
	buf := make([]byte, 12+payloadSize)
	copy(buf[0:4], "RIFF")
	copy(buf[8:12], "WEBP")
	return buf
}

func newAvatarTestServer(t *testing.T) (*Server, string, *PlayerState) {
	t.Helper()
	s := newTestServer(t)
	s.avatarUploadsDir = filepath.Join(t.TempDir(), "avatars")
	if err := os.MkdirAll(s.avatarUploadsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	s.sessionSecret = []byte("test-secret-please-ignore-1234567890")
	s.sessionTtlMs = 3600_000
	s.tokenToPlayer = map[string]string{}
	s.sidToPlayerID = map[string]string{}
	s.rooms = map[string]*RoomState{}

	token := s.issueSessionToken()
	session := s.verifySessionToken(token)
	if session == nil {
		t.Fatal("expected valid session")
	}
	player := &PlayerState{}
	player.ID = "p1"
	player.Connected = true
	s.players = map[string]*PlayerState{"p1": player}
	s.tokenToPlayer[token] = "p1"
	s.sidToPlayerID[session.SID] = "p1"
	return s, token, player
}

func buildAvatarUploadRequest(t *testing.T, token, filename string, content []byte) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if err := w.WriteField("token", token); err != nil {
		t.Fatal(err)
	}
	part, err := w.CreateFormFile("image", filename)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/avatar-image", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestHandleAvatarImageAcceptsValidWebp(t *testing.T) {
	s, token, player := newAvatarTestServer(t)
	req := buildAvatarUploadRequest(t, token, "avatar.webp", fakeWebpBytes(100))
	rec := httptest.NewRecorder()

	s.handleAvatarImage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		AvatarURL string      `json:"avatarUrl"`
		Player    interface{} `json:"player"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.AvatarURL == "" {
		t.Fatal("expected non-empty avatarUrl in response")
	}
	s.mu.Lock()
	got := player.AvatarURL
	s.mu.Unlock()
	if got != resp.AvatarURL {
		t.Fatalf("player.AvatarURL = %q, want %q", got, resp.AvatarURL)
	}
}

func TestHandleAvatarImageRejectsNonWebpExtension(t *testing.T) {
	s, token, _ := newAvatarTestServer(t)
	req := buildAvatarUploadRequest(t, token, "avatar.png", fakeWebpBytes(100))
	rec := httptest.NewRecorder()

	s.handleAvatarImage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAvatarImageRejectsOversized(t *testing.T) {
	s, token, _ := newAvatarTestServer(t)
	req := buildAvatarUploadRequest(t, token, "avatar.webp", fakeWebpBytes(21*1024))
	rec := httptest.NewRecorder()

	s.handleAvatarImage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for oversized avatar, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestHandleAvatarImageRejectsFakeContent(t *testing.T) {
	s, token, _ := newAvatarTestServer(t)
	req := buildAvatarUploadRequest(t, token, "avatar.webp", []byte("not a real webp file, just text padding to be non-trivially sized"))
	rec := httptest.NewRecorder()

	s.handleAvatarImage(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400 for non-webp content, got %d: %s", rec.Code, rec.Body.String())
	}
}

func buildAvatarClearRequest(t *testing.T, token string) *http.Request {
	t.Helper()
	body := &bytes.Buffer{}
	w := multipart.NewWriter(body)
	if err := w.WriteField("token", token); err != nil {
		t.Fatal(err)
	}
	if err := w.WriteField("clear", "1"); err != nil {
		t.Fatal(err)
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/api/avatar-image", body)
	req.Header.Set("Content-Type", w.FormDataContentType())
	return req
}

func TestHandleAvatarImageClear(t *testing.T) {
	s, token, player := newAvatarTestServer(t)
	player.AvatarURL = "/uploads/avatars/old.webp"

	req := buildAvatarClearRequest(t, token)
	rec := httptest.NewRecorder()
	s.handleAvatarImage(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}
	var resp struct {
		AvatarURL string `json:"avatarUrl"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("bad json: %v", err)
	}
	if resp.AvatarURL != "" {
		t.Fatalf("expected empty avatarUrl, got %q", resp.AvatarURL)
	}
	s.mu.Lock()
	got := player.AvatarURL
	s.mu.Unlock()
	if got != "" {
		t.Fatalf("player.AvatarURL = %q, want empty", got)
	}
}
