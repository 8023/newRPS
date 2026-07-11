package server

import "testing"

func TestDeviceKeySeparatesFingerprintsOnSameIP(t *testing.T) {
	ip := "1.2.3.4"
	a := deviceKey(ip, "fp-alice")
	b := deviceKey(ip, "fp-bob")
	if a == b {
		t.Fatal("different fingerprints on same IP must produce different device keys")
	}
	if deviceKey(ip, "fp-alice") != a {
		t.Fatal("device key must be stable")
	}
}

func TestDeviceKeyIncludesIP(t *testing.T) {
	fp := "same-browser"
	if deviceKey("1.1.1.1", fp) == deviceKey("2.2.2.2", fp) {
		t.Fatal("same fingerprint on different IPs must produce different keys")
	}
}

func TestNormalizeFingerprint(t *testing.T) {
	if normalizeFingerprint("") != "missing" {
		t.Fatal("empty fingerprint should become missing")
	}
	if normalizeFingerprint("  AbC-123  ") != "AbC-123" {
		t.Fatalf("got %q", normalizeFingerprint("  AbC-123  "))
	}
}
