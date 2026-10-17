package auth

import (
	"strings"
	"testing"
	"time"
)

// Spec 16 Sec. 4 R16-09 / Spec 17 R17-10 — registered-credential validity bound.
//
// R16-09 requires new credentials to "expire within 90 days"; Spec 17 R17-10
// adopts the same bound. Before this rule the registry validated scopes,
// verifiers and the <=24h rotation overlap but never checked a record's
// validity window: a record with no expires_at was an unbounded grant, and a
// record could declare an arbitrarily long window. Both are refused at the
// load boundary (buildSnapshot), so a bad registry denies protected traffic
// rather than installing an unbounded credential.

// TestRegistryRequiresExpiry refuses a record that declares no expiry. An
// unbounded credential is exactly the posture the 90-day bound forbids.
func TestRegistryRequiresExpiry(t *testing.T) {
	rec := recordFor("no-expiry-secret", "cred-1", "principal-alpha")
	rec.ExpiresAt = time.Time{}

	_, err := New(testVerifierKey, []Record{rec})
	if err == nil {
		t.Fatal("a credential record without expires_at must be refused")
	}
	if !strings.Contains(err.Error(), "expires_at") {
		t.Fatalf("error must name the missing expires_at, got %v", err)
	}
	if !strings.Contains(err.Error(), "R17-10") {
		t.Fatalf("error must name the bound (R17-10), got %v", err)
	}
}

// TestRegistryRejectsOverLongValidity refuses a record whose declared window
// exceeds the 90-day maximum, and accepts one at the bound.
func TestRegistryRejectsOverLongValidity(t *testing.T) {
	now := time.Now()

	over := recordFor("over-secret", "cred-over", "principal-over")
	over.IssuedAt = now
	over.ExpiresAt = now.Add(MaxCredentialValidity + time.Hour)
	if _, err := New(testVerifierKey, []Record{over}); err == nil {
		t.Fatal("a window longer than the maximum must be refused")
	} else if !strings.Contains(err.Error(), "R17-10") {
		t.Fatalf("error must name the bound (R17-10), got %v", err)
	}

	// Exactly at the bound is accepted (the bound is inclusive).
	atBound := recordFor("at-bound-secret", "cred-bound", "principal-bound")
	atBound.IssuedAt = now
	atBound.ExpiresAt = now.Add(MaxCredentialValidity)
	if _, err := New(testVerifierKey, []Record{atBound}); err != nil {
		t.Fatalf("a window at the maximum must be accepted, got %v", err)
	}
}

// TestRegistryValidityWindowMeasuredFromIssuedAt proves the window is measured
// from issued_at when supplied, so a long-lived record that was re-issued
// recently is still accepted while a stale issuance that would run past the
// bound is refused.
func TestRegistryValidityWindowMeasuredFromIssuedAt(t *testing.T) {
	now := time.Now()

	// Issued 200 days ago, expiring tomorrow: a >90-day window measured from
	// issuance, so it must be refused even though the remaining life is short.
	stale := recordFor("stale-secret", "cred-stale", "principal-stale")
	stale.IssuedAt = now.Add(-200 * 24 * time.Hour)
	stale.ExpiresAt = now.Add(24 * time.Hour)
	if _, err := New(testVerifierKey, []Record{stale}); err == nil {
		t.Fatal("a window measured from issued_at beyond the maximum must be refused")
	}

	// Issued an hour ago with a 30-day expiry: within the bound.
	fresh := recordFor("fresh-secret", "cred-fresh", "principal-fresh")
	fresh.IssuedAt = now.Add(-time.Hour)
	fresh.ExpiresAt = now.Add(30 * 24 * time.Hour)
	if _, err := New(testVerifierKey, []Record{fresh}); err != nil {
		t.Fatalf("a fresh, bounded credential must be accepted, got %v", err)
	}
}

// TestRegistryValidityBoundAppliesToReloadAndRotate proves the bound is
// enforced on every replacement path, not only at initial load: a reload or
// verifier-key rotation carrying an unbounded record is refused and leaves the
// prior snapshot intact.
func TestRegistryValidityBoundAppliesToReloadAndRotate(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("s", "cred-1", "p1")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	unbounded := recordFor("s2", "cred-2", "p2")
	unbounded.ExpiresAt = time.Time{}
	if err := reg.Reload(testVerifierKey, []Record{unbounded}); err == nil {
		t.Fatal("reload with an unbounded record must be refused")
	}
	// The prior snapshot survives the rejected reload.
	if _, ok := reg.Resolve("s"); !ok {
		t.Fatal("failed reload must preserve the previous snapshot")
	}

	if err := reg.RotateVerifierKey("rotated-verifier-key-fixture-000000000002", []Record{unbounded}, time.Hour); err == nil {
		t.Fatal("rotation with an unbounded record must be refused")
	}
}

// TestMaxCredentialValidityPublished pins the published bound so a silent
// change to the 90-day maximum fails the build.
func TestMaxCredentialValidityPublished(t *testing.T) {
	if MaxCredentialValidity != 90*24*time.Hour {
		t.Fatalf("MaxCredentialValidity = %v, want 90 days", MaxCredentialValidity)
	}
}
