package server

import (
	"strings"
	"unicode"

	"golang.org/x/text/unicode/norm"
)

func normalizeGenderLabel(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = norm.NFKC.String(s)
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if unicode.IsControl(r) {
			continue
		}
		b.WriteRune(unicode.ToLower(r))
	}
	return strings.TrimSpace(b.String())
}

func containsControlRune(raw string) bool {
	for _, r := range raw {
		if unicode.IsControl(r) {
			return true
		}
	}
	return false
}
