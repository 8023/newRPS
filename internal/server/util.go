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
	return clampF(math.Round(value*10)/10, 0, 100)
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
var safeUploadRe = regexp.MustCompile(`(?i)^/uploads/(?:proofs|admin|avatars)/[0-9a-z._-]+\.(?:jpg|png|webp)$`)

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

func escapeRegExp(value string) string {
	special := `.*+?^${}()|[]\`
	var b strings.Builder
	for _, r := range value {
		if strings.ContainsRune(special, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	return b.String()
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
func recordGameOutcome(player *PlayerState, gameID types.GameID, outcome string) {
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

func uniqueStrings(in []string) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, s := range in {
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}
