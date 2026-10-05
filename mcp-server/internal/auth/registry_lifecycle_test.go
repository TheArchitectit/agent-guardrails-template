package auth

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestRevocationPropagationBoundPublished pins the published lifecycle bounds
// required by Spec 16 §4 / R16-09 so a silent change fails the build.
func TestRevocationPropagationBoundPublished(t *testing.T) {
	if RevocationPropagationBound != 60*time.Second {
		t.Fatalf("RevocationPropagationBound = %v, want 60s", RevocationPropagationBound)
	}
	if MaxRotationOverlap != 24*time.Hour {
		t.Fatalf("MaxRotationOverlap = %v, want 24h", MaxRotationOverlap)
	}
}

// TestRegistryRevocationPropagationWithinBound revokes a credential via an
// atomic reload and asserts the next resolution is denied well within the
// published bound.
func TestRegistryRevocationPropagationWithinBound(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{
		recordFor("keep-secret", "cred-keep", "principal-keep"),
		recordFor("revoke-me", "cred-revoke", "principal-revoke"),
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := reg.Resolve("revoke-me"); !ok {
		t.Fatal("credential should resolve before revocation")
	}

	revoked := recordFor("revoke-me", "cred-revoke", "principal-revoke")
	revoked.Revoked = true
	newRecords := []Record{
		recordFor("keep-secret", "cred-keep", "principal-keep"),
		revoked,
	}

	start := time.Now()
	if err := reg.Reload(testVerifierKey, newRecords); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if _, ok := reg.Resolve("revoke-me"); ok {
		t.Fatal("revoked credential still resolves after reload")
	}
	if elapsed := time.Since(start); elapsed > RevocationPropagationBound {
		t.Fatalf("revocation propagated in %v, exceeding bound %v", elapsed, RevocationPropagationBound)
	}
	// A non-revoked credential is unaffected.
	if _, ok := reg.Resolve("keep-secret"); !ok {
		t.Fatal("unrelated credential should still resolve")
	}
}

// TestRegistryReloadIsAtomicAndFailClosed covers the replacement contract: a
// valid reload swaps the whole snapshot, while an invalid or empty reload
// leaves the previous snapshot intact so it never enlarges access.
func TestRegistryReloadIsAtomicAndFailClosed(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("original", "cred-1", "principal-alpha")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// A bad replacement (missing verifier) must not change the live snapshot.
	if err := reg.Reload(testVerifierKey, []Record{{CredentialID: "cred-2", PrincipalID: "p2"}}); err == nil {
		t.Fatal("invalid replacement must be rejected")
	}
	if _, ok := reg.Resolve("original"); !ok {
		t.Fatal("failed reload must preserve the previous snapshot")
	}

	// An empty replacement must be rejected rather than silently disabling the
	// registry (which would fall back to the legacy unlimited surface).
	if err := reg.Reload(testVerifierKey, nil); err == nil {
		t.Fatal("empty replacement must be rejected")
	}
	if _, ok := reg.Resolve("original"); !ok {
		t.Fatal("rejected empty reload must preserve the previous snapshot")
	}

	// A valid replacement takes effect atomically.
	if err := reg.Reload(testVerifierKey, []Record{recordFor("replacement", "cred-9", "principal-beta")}); err != nil {
		t.Fatalf("valid Reload: %v", err)
	}
	if _, ok := reg.Resolve("original"); ok {
		t.Fatal("original credential must be gone after replacement")
	}
	if _, ok := reg.Resolve("replacement"); !ok {
		t.Fatal("replacement credential must resolve")
	}
}

// TestRegistryVerifierKeyRotationOverlap covers verifier-key rotation: during
// the overlap window both the outgoing and incoming key digests verify; after
// the window only the new key does.
func TestRegistryVerifierKeyRotationOverlap(t *testing.T) {
	const newKey = "rotated-verifier-key-9876543210"

	oldRec := recordFor("old-secret", "cred-old", "principal-old") // digest under testVerifierKey
	newRec := Record{CredentialID: "cred-new", PrincipalID: "principal-new", Scopes: []string{ScopeMCPRead}, Verifier: Digest(newKey, "new-secret")}

	reg, err := New(testVerifierKey, []Record{oldRec})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if err := reg.RotateVerifierKey(newKey, []Record{oldRec, newRec}, time.Hour); err != nil {
		t.Fatalf("RotateVerifierKey: %v", err)
	}
	// Both keys are valid during the overlap.
	if _, ok := reg.Resolve("old-secret"); !ok {
		t.Fatal("outgoing key must still verify during overlap")
	}
	if _, ok := reg.Resolve("new-secret"); !ok {
		t.Fatal("incoming key must verify during overlap")
	}

	// Deterministic post-overlap behaviour: an outgoing key whose window has
	// elapsed no longer verifies, while the incoming key continues to.
	expired := &Registry{snap: &snapshot{
		keys: []verifierKeyEntry{
			{key: []byte(testVerifierKey), validUntil: time.Now().Add(-time.Second)},
			{key: []byte(newKey)},
		},
		records: []Record{oldRec, newRec},
	}}
	if _, ok := expired.Resolve("old-secret"); ok {
		t.Fatal("expired outgoing key must not verify")
	}
	if _, ok := expired.Resolve("new-secret"); !ok {
		t.Fatal("incoming key must verify after overlap")
	}
}

// TestRegistryRotationOverlapBounds rejects overlaps that would accept an old
// key indefinitely or beyond the 24h maximum.
func TestRegistryRotationOverlapBounds(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("s", "cred-1", "p1")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	records := []Record{recordFor("s", "cred-1", "p1")}

	for _, overlap := range []time.Duration{0, -time.Minute, MaxRotationOverlap + time.Second} {
		if err := reg.RotateVerifierKey("new-verifier-key-1234567890", records, overlap); err == nil {
			t.Fatalf("overlap %v must be rejected", overlap)
		}
	}
	if err := reg.RotateVerifierKey("new-verifier-key-1234567890", records, MaxRotationOverlap); err != nil {
		t.Fatalf("overlap at the maximum must be accepted: %v", err)
	}
}

// TestRegistryRejectsUnknownScope rejects records whose scopes are outside the
// approved catalog at load time.
func TestRegistryRejectsUnknownScope(t *testing.T) {
	base := Record{CredentialID: "cred-1", PrincipalID: "p1", Verifier: Digest(testVerifierKey, "s"), Role: RoleReader}

	unknown := base
	unknown.Scopes = []string{"mcp"}
	if _, err := New(testVerifierKey, []Record{unknown}); err == nil {
		t.Fatal("unknown scope must be rejected at load")
	}

	empty := base
	empty.Scopes = []string{""}
	if _, err := New(testVerifierKey, []Record{empty}); err == nil {
		t.Fatal("empty scope must be rejected at load")
	}

	// Every approved catalog scope is accepted.
	for _, scope := range []string{
		ScopeMCPRead, ScopeMCPValidate, ScopeMCPMutate,
		ScopeRESTRead, ScopeRESTWrite, ScopeIDEValidate, ScopeAdminManage,
	} {
		ok := base
		ok.Scopes = []string{scope}
		if _, err := New(testVerifierKey, []Record{ok}); err != nil {
			t.Fatalf("approved scope %q must be accepted: %v", scope, err)
		}
	}
}

// TestPermitsRestrictedOnly enforces the POSIX secret-file permission rule
// independently of the host OS.
func TestPermitsRestrictedOnly(t *testing.T) {
	cases := []struct {
		mode os.FileMode
		want bool
	}{
		{0o600, true},
		{0o400, true},
		{0o000, true},
		{0o700, true}, // no group/world bits; regular-file check is separate
		{0o640, false},
		{0o644, false},
		{0o660, false},
		{0o604, false},
		{0o606, false},
	}
	for _, tc := range cases {
		if got := permitsRestrictedOnly(tc.mode); got != tc.want {
			t.Fatalf("permitsRestrictedOnly(%#o) = %v, want %v", uint32(tc.mode), got, tc.want)
		}
	}
}

// TestLoadFromSourcesExVerifierKeyFile loads the registry and the verifier key
// from read-only secret files, with the key trimmed of a trailing newline.
func TestLoadFromSourcesExVerifierKeyFile(t *testing.T) {
	dir := t.TempDir()
	regFile := filepath.Join(dir, "registry.json")
	keyFile := filepath.Join(dir, "verifier.key")

	record := recordFor("registered-secret", "cred-1", "principal-alpha")
	regJSON := `[{"credential_id":"` + record.CredentialID + `","principal_id":"` + record.PrincipalID +
		`","scopes":["` + ScopeMCPRead + `"],"verifier":"` + record.Verifier + `"}]`
	if err := os.WriteFile(regFile, []byte(regJSON), SecretFileMode); err != nil {
		t.Fatalf("write registry file: %v", err)
	}
	if err := os.WriteFile(keyFile, []byte(testVerifierKey+"\n"), SecretFileMode); err != nil {
		t.Fatalf("write key file: %v", err)
	}

	reg, err := LoadFromSourcesEx("", regFile, "", keyFile)
	if err != nil {
		t.Fatalf("LoadFromSourcesEx: %v", err)
	}
	if !reg.Enabled() {
		t.Fatal("registry from secret files must be enabled")
	}
	if _, ok := reg.Resolve("registered-secret"); !ok {
		t.Fatal("credential must resolve with a file-sourced verifier key")
	}
}

// TestLoadFromSourcesExMissingKeyFileFailsClosed rejects a configured registry
// whose verifier-key file is absent.
func TestLoadFromSourcesExMissingKeyFileFailsClosed(t *testing.T) {
	dir := t.TempDir()
	regFile := filepath.Join(dir, "registry.json")
	record := recordFor("registered-secret", "cred-1", "principal-alpha")
	regJSON := `[{"credential_id":"` + record.CredentialID + `","principal_id":"` + record.PrincipalID +
		`","scopes":["` + ScopeMCPRead + `"],"verifier":"` + record.Verifier + `"}]`
	if err := os.WriteFile(regFile, []byte(regJSON), SecretFileMode); err != nil {
		t.Fatalf("write registry file: %v", err)
	}

	reg, err := LoadFromSourcesEx("", regFile, "", filepath.Join(dir, "absent.key"))
	if err == nil {
		t.Fatal("missing verifier-key file must fail closed")
	}
	if reg != nil && reg.Enabled() {
		t.Fatal("error path must not return an enabled registry")
	}
}

// TestSecretFilePermissiveRejected rejects a group/world-readable secret file
// on POSIX hosts. Windows ACLs are not observable through os.FileMode, so the
// host check is skipped there.
func TestSecretFilePermissiveRejected(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX mode check is not enforced on Windows")
	}
	dir := t.TempDir()
	regFile := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(regFile, []byte("[]"), 0o644); err != nil {
		t.Fatalf("write registry file: %v", err)
	}
	if _, err := LoadFromSources("", regFile, testVerifierKey); err == nil || !strings.Contains(err.Error(), "group/world") {
		t.Fatalf("group/world-readable secret file must be rejected, got %v", err)
	}
}
