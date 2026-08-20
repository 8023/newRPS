package server

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/doumiao/newRPS/internal/types"
)

func nowMs() int64 {
	return time.Now().UnixMilli()
}

func randomID() string {
	b := make([]byte, 6)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// randomClaimKey 生成认领密钥（ClaimKey）：12 字节 ≈ 96 bit，base64url 约 16 字符。
// 与 randomID（房间/消息等非秘密短 ID）刻意分开，避免把弱熵 ID 复用到安全边界上。
// 展示认领码时前端会把 playerId 从 UUID 压成 base64；库内 playerId 仍存 UUID，不变。
// Claim 用后即轮换，熵可略低于长期设备凭据 randomPlayerSecret。
func randomClaimKey() string {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

// randomPlayerSecret 生成服务端签发的设备凭据（PlayerSecrets 条目）：16 字节 ≈ 128 bit，
// base64url 约 22 字符。用于认领换发、旧格式 secret 静默换发等路径。
// 前端首次注册仍可用 UUID；本函数只约束「服务端新签发」的强度。比 ClaimKey 更长，
// 因为 secret 长期有效、反复用于 player:join，而 claim 是一次性的。
func randomPlayerSecret() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func generatePublicID() string {
	b := make([]byte, 9)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func clamp(value, min, max int) int {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampF(value, min, max float64) float64 {
	if value < min {
		return min
	}
	if value > max {
		return max
	}
	return value
}

func clampGiveawayValue(value float64) float64 {
	return clampF(math.Round(value*100)/100, 0, 100)
}

func currentExtremeDecayHour(now int64) int64 {
	return now / 3_600_000
}

func currentRankedDecayDay(now int64) int64 {
	return now / 86_400_000
}

func cleanText(value string, max int) string {
	var b strings.Builder
	for _, r := range value {
		if r < 0x20 || r == 0x7f {
			continue
		}
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(r)
	}
	s := strings.TrimSpace(b.String())
	runes := []rune(s)
	if len(runes) > max {
		return string(runes[:max])
	}
	return s
}

// randomID 使用 RawURLEncoding，可能含 '_' 与 '-'
var safeUploadRe = regexp.MustCompile(`(?i)^/uploads/(?:proofs|admin|avatars|contributions)/[0-9a-z._-]+\.(?:jpg|png|webp)$`)

func safeUploadURL(value string) string {
	if safeUploadRe.MatchString(value) {
		return value
	}
	return ""
}

func boolPtr(b bool) *bool        { return &b }
func intPtr(i int) *int           { return &i }
func floatPtr(f float64) *float64 { return &f }
func int64Ptr(i int64) *int64     { return &i }

func ptrBool(p *bool) bool {
	if p == nil {
		return false
	}
	return *p
}

func ptrInt(p *int) int {
	if p == nil {
		return 0
	}
	return *p
}

func ptrFloat(p *float64) float64 {
	if p == nil {
		return 0
	}
	return *p
}

func ptrInt64(p *int64) int64 {
	if p == nil {
		return 0
	}
	return *p
}

func randomFrom[T any](values []T) T {
	if len(values) == 0 {
		var zero T
		return zero
	}
	// crypto/rand based index
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	return values[int(b[0])%len(values)]
}

// randomFromF 历史上用纳秒取模选择，连续调用（同一房名的多个词）随机性极差；
// 改为走 crypto/rand 的 randomFrom。
func randomFromF[T any](values []T) T {
	return randomFrom(values)
}

func oppositeSeat(seat types.SeatKey) types.SeatKey {
	if seat == types.SeatA {
		return types.SeatB
	}
	return types.SeatA
}

func randomSeat() types.SeatKey {
	b := make([]byte, 1)
	_, _ = rand.Read(b)
	if b[0]%2 == 0 {
		return types.SeatA
	}
	return types.SeatB
}

func emptySeatStats() types.SeatStats {
	return types.SeatStats{}
}

func freshGameStats() types.GameStats {
	return types.GameStats{}
}

// recordGameOutcome 记一局某游戏的胜/负/平，并同步合计到 Stats.Wins/Losses/Draws。
// 须在 s.mu 内调用；会 markPlayerDirty，避免只靠 60s checkpoint 兜底写盘。
func (s *Server) recordGameOutcome(player *PlayerState, gameID types.GameID, outcome string) {
	if player == nil {
		return
	}
	if gameID == "" {
		gameID = types.GameRPS
	}
	wld := player.GameStats.WLDFor(gameID)
	if wld == nil {
		return
	}
	switch outcome {
	case "win":
		wld.Wins++
	case "loss":
		wld.Losses++
	case "draw":
		wld.Draws++
	default:
		return
	}
	player.SyncTotalsFromGameStats()
	s.markPlayerDirty(player)
	s.requestPersist("lazy")
}

func formatSigned(n int) string {
	if n >= 0 {
		return fmt.Sprintf("+%d", n)
	}
	return fmt.Sprintf("%d", n)
}

func containsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}
