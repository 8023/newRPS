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
	if s.rateBuckets == nil {
		s.rateBuckets = map[string]*rateBucket{}
	}
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
	if s.rateLimitBuckets == nil {
		s.rateLimitBuckets = map[string]*rateLimitBucket{}
	}
	current := s.rateLimitBuckets[key]
	if current == nil || current.ResetAt <= now {
		s.rateLimitBuckets[key] = &rateLimitBucket{ResetAt: now + windowMs, Count: 1}
		return true
	}
	current.Count++
	return current.Count <= max
}

// pruneEphemeralState 定期清理只增不减的限流/防多开 map，避免长期运行内存泄漏。
// 调用方须持有 s.mu。
func (s *Server) pruneEphemeralState() {
	now := nowMs()
	const idleMs = int64(10 * 60 * 1000)
	for k, b := range s.rateBuckets {
		if b == nil {
			delete(s.rateBuckets, k)
			continue
		}
		if b.CooldownUntil > now {
			continue
		}
		newest := int64(0)
		for _, t := range b.Hits {
			if t > newest {
				newest = t
			}
		}
		if newest == 0 || now-newest > idleMs {
			delete(s.rateBuckets, k)
		}
	}
	for k, b := range s.rateLimitBuckets {
		if b == nil || b.ResetAt <= now {
			delete(s.rateLimitBuckets, k)
		}
	}
	for k, attempts := range s.deviceCreateAttempts {
		fresh := false
		for _, t := range attempts {
			if now-t < idleMs {
				fresh = true
				break
			}
		}
		if !fresh {
			delete(s.deviceCreateAttempts, k)
		}
	}
}

// errorLogHeader 是 error.csv 的固定表头：所有拒绝/失败类事件的字段并集。
// 每行只填该事件用到的列，其余列留空——和 connections/rooms 等表一样是标准 CSV，
// 不再把明细塞成一整块 JSON 字符串。
var errorLogHeader = []string{
	"time", "event", "sid", "ip", "device", "fingerprint", "userAgent",
	"origin", "host", "rawHost", "xffHost", "reason", "limit", "compression", "wsEvent", "err",
}

// errorDetailColumn 把 securityLog 调用方用的字段名映射到 error.csv 的固定列名。
// - "fp" 是浏览器指纹，和 connections.csv 里的 "fingerprint" 列对齐。
// - 调用方 details 里的 "event" 是触发问题的具体业务事件名（如 room:move），
//   和 securityLog 自身的 event 参数（"rate_limit" 这类事件分类）是两回事，
//   映射到 "wsEvent" 列，避免同名覆盖。
func errorDetailColumn(key string) string {
	switch key {
	case "fp":
		return "fingerprint"
	case "event":
		return "wsEvent"
	default:
		return key
	}
}

// securityLog 记录真正的拒绝/失败类安全事件（鉴权失败、限流、越权等）。
// 这些事件不会中断服务进程，只落盘 work/logs/{周}/error.csv，不再打到终端——
// 命令行只保留"进程无法继续运行"级别的致命错误（见 cmd/server/main.go：New()/Run() 失败直接 os.Exit）。
func (s *Server) securityLog(event string, details map[string]any) {
	row := make(map[string]string, len(errorLogHeader))
	row["time"] = time.Now().Format(time.RFC3339)
	row["event"] = event
	for k, v := range details {
		row[errorDetailColumn(k)] = fmt.Sprint(v)
	}
	fields := make([]string, len(errorLogHeader))
	for i, col := range errorLogHeader {
		fields[i] = row[col]
	}
	s.activityLog("error", errorLogHeader, fields)
}

// errorLog 供只有一句错误信息、没有结构化字段的场景使用（如持久化读写失败）。
func (s *Server) errorLog(event, errMsg string) {
	s.securityLog(event, map[string]any{"err": errMsg})
}
