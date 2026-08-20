package server

import "testing"

func TestSafeUploadURLAllowsBase64URLChars(t *testing.T) {
	// randomID 使用 RawURLEncoding，可含 '_'
	ok := []string{
		"/uploads/proofs/1710000000000-A_GwoEDr.webp",
		"/uploads/proofs/1710000000000-abc-def.webp",
		"/uploads/admin/1-xY9.webp",
		"/uploads/contributions/1710000000000-abc.webp",
	}
	for _, u := range ok {
		if safeUploadURL(u) != u {
			t.Fatalf("should accept %s", u)
		}
	}
	bad := []string{
		"/uploads/proofs/../x.webp",
		"https://evil/x.webp",
		"/uploads/proofs/x.gif",
	}
	for _, u := range bad {
		if safeUploadURL(u) != "" {
			t.Fatalf("should reject %s", u)
		}
	}
}

func TestRandomClaimKeyLength(t *testing.T) {
	// 12 字节 → RawURLEncoding 无 padding 固定 16 字符
	for i := 0; i < 8; i++ {
		k := randomClaimKey()
		if len(k) != 16 {
			t.Fatalf("randomClaimKey len = %d, want 16 (%q)", len(k), k)
		}
	}
}

func TestRandomPlayerSecretLength(t *testing.T) {
	// 16 字节 → RawURLEncoding 无 padding 固定 22 字符
	for i := 0; i < 8; i++ {
		s := randomPlayerSecret()
		if len(s) != 22 {
			t.Fatalf("randomPlayerSecret len = %d, want 22 (%q)", len(s), s)
		}
	}
}
