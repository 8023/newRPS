package server

import "testing"

func TestSafeUploadURLAllowsBase64URLChars(t *testing.T) {
	// randomID 使用 RawURLEncoding，可含 '_'
	ok := []string{
		"/uploads/proofs/1710000000000-A_GwoEDr.webp",
		"/uploads/proofs/1710000000000-abc-def.webp",
		"/uploads/admin/1-xY9.webp",
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
