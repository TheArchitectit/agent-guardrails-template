package auth

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLoadFromSourcesConfiguredInvalidFailsClosed covers every configured-
// invalid registry shape required by S-A0 / R16-03: malformed JSON, unreadable
// file, empty records, and unusable verifier-key material must all return an
// error so callers deny protected traffic instead of falling back to legacy.
func TestLoadFromSourcesConfiguredInvalidFailsClosed(t *testing.T) {
	unreadable := filepath.Join(t.TempDir(), "no-such-registry.json")

	// Empty-record fixtures that parse but authorize nobody.
	emptyArray := "[]"
	emptyFile := filepath.Join(t.TempDir(), "empty.json")
	if err := os.WriteFile(emptyFile, []byte("[]"), 0o600); err != nil {
		t.Fatalf("write empty file: %v", err)
	}

	validRecord := `[{"credential_id":"cred-1","principal_id":"p1","verifier":"` + Digest(testVerifierKey, "s") + `"}]`

	cases := []struct {
		name       string
		inlineJSON string
		filePath   string
		verifier   string
	}{
		{"malformed inline JSON", `[{"credential_id":"x"}`, "", testVerifierKey},
		{"malformed file JSON", "", writeTempFile(t, "{not-json"), testVerifierKey},
		{"unreadable file", "", unreadable, testVerifierKey},
		{"empty inline records", emptyArray, "", testVerifierKey},
		{"empty file records", "", emptyFile, testVerifierKey},
		{"empty both sources", emptyArray, emptyFile, testVerifierKey},
		{"empty verifier key", validRecord, "", ""},
		{"short verifier key", validRecord, "", "short-key"},
		{"whitespace verifier key", validRecord, "", "                "},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			reg, err := LoadFromSources(tc.inlineJSON, tc.filePath, tc.verifier)
			if err == nil {
				t.Fatalf("configured-invalid registry must error, got reg=%v", reg)
			}
			// Fail-closed: never hand back an enabled registry on error.
			if reg != nil && reg.Enabled() {
				t.Fatal("error path must not return an enabled registry")
			}
		})
	}
}

// TestLoadFromSourcesAbsentRemainsNil keeps the unconfigured migration mode
// explicit: no sources means no registry, not an error.
func TestLoadFromSourcesAbsentRemainsNil(t *testing.T) {
	reg, err := LoadFromSources("", "", testVerifierKey)
	if err != nil {
		t.Fatalf("absent registry must not error: %v", err)
	}
	if reg.Enabled() {
		t.Fatal("absent registry must not be enabled")
	}
}

// TestNewRejectsDuplicateCredentialIDs rejects records that would let one
// credential ID appear twice.
func TestNewRejectsDuplicateCredentialIDs(t *testing.T) {
	_, err := New(testVerifierKey, []Record{
		recordFor("secret-a", "cred-1", "principal-alpha"),
		recordFor("secret-b", "cred-1", "principal-beta"),
	})
	if err == nil || !strings.Contains(err.Error(), "duplicate credential_id") {
		t.Fatalf("expected duplicate credential_id error, got %v", err)
	}
}

// TestNewRejectsInvalidVerifierMaterial rejects unusable verifier-key material
// at the load boundary.
func TestNewRejectsInvalidVerifierMaterial(t *testing.T) {
	records := []Record{recordFor("secret", "cred-1", "p1")}
	for _, key := range []string{"", "short", strings.Repeat("x", 15)} {
		if _, err := New(key, records); err == nil {
			t.Fatalf("verifier key %q must be rejected", key)
		}
	}
}

// TestNewRejectsZeroEntropyVerifierKey covers Spec 17 R17-07 at the registry
// load boundary: a key that clears the length floor but carries no real
// randomness (a single repeated byte) must be refused, because "a length check
// is not an entropy proof". Accepting it would make every stored verifier
// forgeable by anyone who guesses the trivial key.
func TestNewRejectsZeroEntropyVerifierKey(t *testing.T) {
	records := []Record{recordFor("secret", "cred-1", "p1")}
	// Length 64 clears the 16-character floor but has one distinct byte.
	_, err := New(strings.Repeat("x", 64), records)
	if err == nil {
		t.Fatal("a zero-entropy verifier key must be refused (R17-07)")
	}
	if !strings.Contains(err.Error(), "R17-07") {
		t.Fatalf("error must name R17-07, got %v", err)
	}

	// Positive control: a strong key is accepted.
	if _, err := New("9f2c4e6a8b1d3f5a7c9e0b2d4f6a8c1e3b5d7f9a0c2e4b6d8f1a3c5e7b9d0f2a", records); err != nil {
		t.Fatalf("a 256-bit verifier key must be accepted, got %v", err)
	}
}

func writeTempFile(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "registry.json")
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	return path
}
