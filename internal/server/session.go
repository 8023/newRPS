package server

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"
)

func (s *Server) hmac(input string) string {
	mac := hmac.New(sha256.New, s.sessionSecret)
	_, _ = mac.Write([]byte(input))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) signSession(payload SessionPayload) string {
	body := fmt.Sprintf("%s.%d", payload.SID, payload.Exp)
	return body + "." + s.hmac(body)
}

func (s *Server) issueSessionToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return s.signSession(SessionPayload{
		SID: hex.EncodeToString(b),
		Exp: nowMs() + s.sessionTtlMs,
	})
}

func (s *Server) verifySessionToken(token string) *SessionPayload {
	value := strings.TrimSpace(token)
	parts := strings.Split(value, ".")
	if len(parts) != 3 {
		return nil
	}
	sid, rawExp, signature := parts[0], parts[1], parts[2]
	if len(sid) != 32 {
		return nil
	}
	for _, c := range sid {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return nil
		}
	}
	exp, err := strconv.ParseInt(rawExp, 10, 64)
	if err != nil || exp <= nowMs() {
		return nil
	}
	expected := s.hmac(sid + "." + rawExp)
	if subtle.ConstantTimeCompare([]byte(signature), []byte(expected)) != 1 {
		return nil
	}
	return &SessionPayload{SID: sid, Exp: exp}
}

func tokenLooksExpired(token string) bool {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	return err == nil && exp <= nowMs()
}

func (s *Server) checkRateLimit(key string, options RateLimitOptions) bool {
	now := nowMs()
	bucket := s.rateBuckets[key]
	if bucket == nil {
		bucket = &rateBucket{}
		s.rateBuckets[key] = bucket
	}
	if bucket.CooldownUntil > now {
		return false
	}
	window := options.WindowMs
	filtered := bucket.Hits[:0]
	for _, t := range bucket.Hits {
		if now-t < window {
			filtered = append(filtered, t)
		}
	}
	bucket.Hits = filtered
	if len(bucket.Hits) >= options.Limit {
		cd := options.CooldownMs
		if cd <= 0 {
			cd = options.WindowMs
		}
		bucket.CooldownUntil = now + cd
		bucket.Hits = nil
		return false
	}
	bucket.Hits = append(bucket.Hits, now)
	bucket.CooldownUntil = 0
	return true
}

func rateLimitKey(event, ipAddress, sid string) string {
	if sid == "" {
		sid = "anonymous"
	}
	return event + ":" + ipAddress + ":" + sid
}

func (s *Server) consumeRateLimit(key string, windowMs int64, max int) bool {
	now := nowMs()
	current := s.rateLimitBuckets[key]
	if current == nil || current.ResetAt <= now {
		s.rateLimitBuckets[key] = &rateLimitBucket{ResetAt: now + windowMs, Count: 1}
		return true
	}
	current.Count++
	return current.Count <= max
}

func (s *Server) securityLog(event string, details map[string]any) {
	// JSON-ish structured log
	parts := []string{fmt.Sprintf(`"ts":"%s"`, time.Now().UTC().Format(time.RFC3339)), fmt.Sprintf(`"event":"%s"`, event)}
	for k, v := range details {
		parts = append(parts, fmt.Sprintf(`"%s":%q`, k, fmt.Sprint(v)))
	}
	fmt.Printf("{%s}\n", strings.Join(parts, ","))
}
