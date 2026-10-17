package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// rotatedVerifierKeyForTest derives the "incoming" verifier key for the
// rotation cases at runtime from a low-entropy seed. Committing a
// high-entropy literal would trip the generic-api-key secret scanner; deriving
// it here keeps the tree clean while the rotation path under test still sees a
// distinct, non-trivial key. (The scanner is right to flag such literals — the
// cause is the fixture, not the scan.)
func rotatedVerifierKeyForTest() string {
	sum := sha256.Sum256([]byte("rotation-audit-fixture-seed"))
	return hex.EncodeToString(sum[:])
}

// Spec 16 R16-09 / Spec 17 R17-10 — operator audit for credential-registry
// replacement and verifier-key rotation.
//
// R16-09 requires the credential lifecycle to define "atomic reload or
// immutable deployment replacement, verifier-key rotation and operator audit".
// R17-10 requires rotation to "log opaque rotation events" and to "identify the
// verifying key". Before this existed the registry swapped its whole snapshot
// on every reload/rotation with no record at all, so an operator could not
// confirm a rotation was applied or reconstruct when it happened.
//
// These tests pin the contract:
//  1. an applied replacement emits exactly one opaque event,
//  2. a no-op replacement (unchanged source) emits nothing,
//  3. a rejected replacement emits nothing,
//  4. the event never carries key material or a secret,
//  5. the key fingerprint is stable and identifies a rotation.

// captureRotations installs an observer and returns the collected events.
func captureRotations(t *testing.T, reg *Registry) *[]RotationEvent {
	t.Helper()
	events := &[]RotationEvent{}
	reg.SetRotationObserver(func(ev RotationEvent) {
		*events = append(*events, ev)
	})
	return events
}

// TestRotationAuditReloadEmitsOpaqueEvent covers the reload path (R16-09): an
// applied replacement that changes state emits one opaque event naming the
// operation, the new record counts and a key fingerprint — never the key.
func TestRotationAuditReloadEmitsOpaqueEvent(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("keep-secret", "cred-keep", "principal-keep")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := captureRotations(t, reg)

	revoked := recordFor("revoke-me", "cred-revoke", "principal-revoke")
	revoked.Revoked = true
	if err := reg.Reload(testVerifierKey, []Record{
		recordFor("keep-secret", "cred-keep", "principal-keep"),
		revoked,
	}); err != nil {
		t.Fatalf("Reload: %v", err)
	}

	if len(*events) != 1 {
		t.Fatalf("applied reload emitted %d events, want 1", len(*events))
	}
	ev := (*events)[0]
	if ev.Operation != "reload" {
		t.Fatalf("operation = %q, want reload", ev.Operation)
	}
	if ev.RecordCount != 2 || ev.RevokedCount != 1 {
		t.Fatalf("counts = (%d, %d), want (2, 1)", ev.RecordCount, ev.RevokedCount)
	}
	if ev.KeyFingerprint != keyFingerprint(testVerifierKey) {
		t.Fatalf("fingerprint = %q, want the fingerprint of the in-force key", ev.KeyFingerprint)
	}
	if ev.KeyFingerprint == "" || ev.KeyFingerprint == testVerifierKey {
		t.Fatal("the event must carry a non-reversible fingerprint, never the key")
	}
	if ev.OccurredAt.IsZero() {
		t.Fatal("event must carry the time the replacement was applied")
	}
}

// TestRotationAuditNoOpReloadIsSilent covers the "record a change, not a
// heartbeat" rule: re-reading an unchanged source must not emit an event, or
// the log would be useless for reconstructing rotations.
func TestRotationAuditNoOpReloadIsSilent(t *testing.T) {
	records := []Record{recordFor("keep-secret", "cred-keep", "principal-keep")}
	reg, err := New(testVerifierKey, records)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := captureRotations(t, reg)

	// Same records, same key: no state change.
	if err := reg.Reload(testVerifierKey, records); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if len(*events) != 0 {
		t.Fatalf("no-op reload emitted %d events, want 0", len(*events))
	}

	// A real change still fires.
	if err := reg.Reload(testVerifierKey, []Record{recordFor("other", "cred-other", "principal-other")}); err != nil {
		t.Fatalf("Reload: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("changed reload emitted %d events, want 1", len(*events))
	}
}

// TestRotationAuditRejectedReloadIsSilent proves a rejected replacement records
// nothing: the operator audit must describe applied state, never an attempted
// change that did not take effect.
func TestRotationAuditRejectedReloadIsSilent(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("original", "cred-1", "principal-alpha")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := captureRotations(t, reg)

	// Invalid replacement (missing verifier) must be rejected without an event.
	if err := reg.Reload(testVerifierKey, []Record{{CredentialID: "cred-2", PrincipalID: "p2"}}); err == nil {
		t.Fatal("invalid replacement must be rejected")
	}
	// Empty replacement is also rejected.
	if err := reg.Reload(testVerifierKey, nil); err == nil {
		t.Fatal("empty replacement must be rejected")
	}
	if len(*events) != 0 {
		t.Fatalf("rejected replacements emitted %d events, want 0", len(*events))
	}
}

// TestRotationAuditVerifierKeyRotation covers R17-10's rotation clause: a
// verifier-key rotation always emits an event that identifies the *incoming*
// key by fingerprint and records the overlap window.
func TestRotationAuditVerifierKeyRotation(t *testing.T) {
	newKey := rotatedVerifierKeyForTest()

	reg, err := New(testVerifierKey, []Record{recordFor("old-secret", "cred-old", "principal-old")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := captureRotations(t, reg)

	newRec := Record{CredentialID: "cred-new", PrincipalID: "principal-new", Scopes: []string{ScopeMCPRead}, Verifier: Digest(newKey, "new-secret")}
	if err := reg.RotateVerifierKey(newKey, []Record{
		recordFor("old-secret", "cred-old", "principal-old"),
		newRec,
	}, time.Hour); err != nil {
		t.Fatalf("RotateVerifierKey: %v", err)
	}

	if len(*events) != 1 {
		t.Fatalf("rotation emitted %d events, want 1", len(*events))
	}
	ev := (*events)[0]
	if ev.Operation != "rotate" {
		t.Fatalf("operation = %q, want rotate", ev.Operation)
	}
	if ev.KeyFingerprint != keyFingerprint(newKey) {
		t.Fatalf("fingerprint = %q, want the incoming key's fingerprint", ev.KeyFingerprint)
	}
	if ev.KeyFingerprint == keyFingerprint(testVerifierKey) {
		t.Fatal("the event must identify the INCOMING key, not the outgoing one")
	}
	if ev.OverlapSeconds != 3600 {
		t.Fatalf("overlap = %d, want 3600", ev.OverlapSeconds)
	}
	if ev.RecordCount != 2 {
		t.Fatalf("record count = %d, want 2", ev.RecordCount)
	}
}

// TestRotationAuditEventNeverCarriesKeyMaterial is the opacity guard: the
// serialized event must not contain the verifier key or any credential secret.
// This is the property that keeps the audit log from becoming a credential
// store.
func TestRotationAuditEventNeverCarriesKeyMaterial(t *testing.T) {
	newKey := rotatedVerifierKeyForTest()
	const secret = "super-secret-credential-value"

	reg, err := New(testVerifierKey, []Record{recordFor(secret, "cred-1", "principal-1")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := captureRotations(t, reg)

	if err := reg.RotateVerifierKey(newKey, []Record{
		{CredentialID: "cred-1", PrincipalID: "principal-1", Scopes: []string{ScopeMCPRead}, Verifier: Digest(newKey, secret)},
	}, time.Minute); err != nil {
		t.Fatalf("RotateVerifierKey: %v", err)
	}
	if len(*events) != 1 {
		t.Fatalf("rotation emitted %d events, want 1", len(*events))
	}

	rendered := renderEventForTest((*events)[0])
	for _, forbidden := range []string{testVerifierKey, newKey, secret} {
		if strings.Contains(rendered, forbidden) {
			t.Fatalf("rotation event leaked key material %q: %s", forbidden, rendered)
		}
	}
}

// TestRotationAuditFingerprintIsStableAndKeyIdentifying proves the fingerprint
// is deterministic (so events correlate) and distinguishes keys (so a rotation
// is visible).
func TestRotationAuditFingerprintIsStableAndKeyIdentifying(t *testing.T) {
	if keyFingerprint(testVerifierKey) != keyFingerprint(testVerifierKey) {
		t.Fatal("fingerprint must be deterministic")
	}
	if keyFingerprint(testVerifierKey) == keyFingerprint(rotatedVerifierKeyForTest()) {
		t.Fatal("different keys must have different fingerprints")
	}
}

// renderEventForTest renders every opaque field of an event, so the opacity
// guard inspects what an operator would actually read.
func renderEventForTest(ev RotationEvent) string {
	return strings.Join([]string{
		ev.Operation,
		ev.KeyFingerprint,
		ev.OccurredAt.String(),
	}, " ")
}
