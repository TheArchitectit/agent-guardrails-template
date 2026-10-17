package secretbox

import (
	"strings"
	"testing"
)

// TestSealOpenRoundTrip covers the positive: a sealed value opens to the
// original plaintext and is reported as sealed.
func TestSealOpenRoundTrip(t *testing.T) {
	s, err := New("operator-key-01")
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	const plaintext = "webhook-signing-secret-canary-01"
	stored, err := s.Seal(plaintext)
	if err != nil {
		t.Fatalf("Seal: %v", err)
	}
	if !IsSealed(stored) {
		t.Fatalf("sealed value must carry the versioned prefix, got %q", stored)
	}
	if strings.Contains(stored, plaintext) {
		t.Fatalf("sealed value must not contain the plaintext: %q", stored)
	}
	got, sealed, err := s.Open(stored)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	if !sealed {
		t.Fatal("Open must report a sealed value as sealed")
	}
	if got != plaintext {
		t.Fatalf("round trip mismatch: got %q want %q", got, plaintext)
	}
}

// TestSealIsNonDeterministic proves a fresh nonce per call, so two seals of the
// same plaintext differ and a value cannot be recognised by comparison.
func TestSealIsNonDeterministic(t *testing.T) {
	s, _ := New("operator-key-01")
	a, _ := s.Seal("same-secret")
	b, _ := s.Seal("same-secret")
	if a == b {
		t.Fatalf("two seals of the same plaintext must differ (random nonce): %q", a)
	}
}

// TestOpenLegacyPlaintextReportsUnsealed proves a value with no prefix is
// returned unchanged but reported sealed=false, so a caller cannot mistake
// legacy plaintext for encrypted-at-rest material.
func TestOpenLegacyPlaintextReportsUnsealed(t *testing.T) {
	s, _ := New("operator-key-01")
	got, sealed, err := s.Open("legacy-plaintext-secret")
	if err != nil {
		t.Fatalf("Open legacy: %v", err)
	}
	if sealed {
		t.Fatal("a value with no sealed prefix must not be reported as sealed")
	}
	if got != "legacy-plaintext-secret" {
		t.Fatalf("legacy plaintext must pass through unchanged, got %q", got)
	}
}

// TestOpenRejectsWrongKey is the negative control: a sealed value opened with a
// different key must fail, never return garbage as if it were the secret.
func TestOpenRejectsWrongKey(t *testing.T) {
	sealer, _ := New("operator-key-01")
	stored, _ := sealer.Seal("webhook-signing-secret-canary-01")

	other, _ := New("operator-key-02")
	if _, _, err := other.Open(stored); err == nil {
		t.Fatal("Open with a wrong key must fail authentication")
	}
}

// TestOpenRejectsTamperedCiphertext proves the GCM tag catches a modified
// stored value.
func TestOpenRejectsTamperedCiphertext(t *testing.T) {
	s, _ := New("operator-key-01")
	stored, _ := s.Seal("webhook-signing-secret-canary-01")

	// Flip a byte in the base64 payload (past the prefix).
	body := []byte(stored)
	i := len("v1:") + 2
	if body[i] == 'A' {
		body[i] = 'B'
	} else {
		body[i] = 'A'
	}
	if _, _, err := s.Open(string(body)); err == nil {
		t.Fatal("Open of a tampered sealed value must fail")
	}
}

// TestOpenRejectsTruncatedValue proves a too-short sealed value is refused
// rather than panicking or returning empty as success.
func TestOpenRejectsTruncatedValue(t *testing.T) {
	s, _ := New("operator-key-01")
	if _, _, err := s.Open("v1:AAAA"); err == nil {
		t.Fatal("Open of a truncated sealed value must fail")
	}
}

// TestEmptyValues proves an empty plaintext seals to empty and an empty stored
// value opens to empty with no error, preserving the store's NULL/empty rule.
func TestEmptyValues(t *testing.T) {
	s, _ := New("operator-key-01")
	sealed, err := s.Seal("")
	if err != nil {
		t.Fatalf("Seal empty: %v", err)
	}
	if sealed != "" {
		t.Fatalf("empty plaintext must seal to empty, got %q", sealed)
	}
	got, isSealed, err := s.Open("")
	if err != nil || isSealed || got != "" {
		t.Fatalf("Open empty: got %q sealed=%v err=%v", got, isSealed, err)
	}
}

// TestNewRejectsEmptyKey is the fail-closed control: a sealer built from no key
// must be refused, not silently created.
func TestNewRejectsEmptyKey(t *testing.T) {
	if _, err := New(""); err == nil {
		t.Fatal("New with an empty key must fail")
	}
	if _, err := New("   "); err == nil {
		t.Fatal("New with a whitespace key must fail")
	}
}

// TestNilSealerIsFailClosed proves a nil sealer cannot seal or open a sealed
// value — it does not silently pass material through.
func TestNilSealerIsFailClosed(t *testing.T) {
	var s *Sealer
	if _, err := s.Seal("secret"); err == nil {
		t.Fatal("a nil sealer must refuse to Seal")
	}
	if _, _, err := s.Open("v1:AAAA"); err == nil {
		t.Fatal("a nil sealer must refuse to Open a sealed value")
	}
}
