package main

import (
	"crypto/x509"
	"testing"
)

func TestEmbeddedFallbackRootsAvailableWithoutSystemBundle(t *testing.T) {
	t.Setenv("SSL_CERT_FILE", "/definitely-missing/ca-certificates.crt")
	t.Setenv("SSL_CERT_DIR", "/definitely-missing/certs")

	pool, err := x509.SystemCertPool()
	if err != nil {
		t.Fatalf("SystemCertPool: %v", err)
	}
	if pool == nil || len(pool.Subjects()) == 0 {
		t.Fatal("expected embedded fallback X.509 roots")
	}
}
