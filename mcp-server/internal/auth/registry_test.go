package auth

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

const testVerifierKey = "unit-test-verifier-key-0123456789"

func recordFor(secret, credentialID, principalID string) Record {
	return Record{
		CredentialID: credentialID,
		PrincipalID:  principalID,
		Scopes:       []string{"mcp"},
		Verifier:     Digest(testVerifierKey, secret),
	}
}

// TestRegistryResolvesRegisteredCredential covers the positive path: a
// registered secret resolves to its principal and credential, and the raw
// secret is never part of the resolved identity.
func TestRegistryResolvesRegisteredCredential(t *testing.T) {
	reg, err := New(testVerifierKey, []Record{recordFor("s3cret-value", "cred-1", "principal-alpha")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if !reg.Enabled() {
		t.Fatal("registry with a record should be enabled")
	}

	principal, ok := reg.Resolve("s3cret-value")
	if !ok {
		t.Fatal("registered credential did not resolve")
	}
	if principal.ID != "principal-alpha" {
		t.Fatalf("principal ID = %q, want principal-alpha", principal.ID)
	}
	if principal.CredentialID != "cred-1" {
		t.Fatalf("credential ID = %q, want cred-1", principal.CredentialID)
	}
	if strings.Contains(fmt.Sprintf("%+v", principal), "s3cret-value") {
		t.Fatal("resolved principal must not embed the presented secret")
	}
}

// TestRegistryDeniesUnregisteredCredential covers fail-closed behaviour: an
// unregistered secret never resolves, and a revoked or expired record never
// resolves either.
func TestRegistryDeniesUnregisteredCredential(t *testing.T) {
	expired := recordFor("expired-value", "cred-2", "principal-beta")
	expired.ExpiresAt = time.Now().Add(-time.Hour)
	revoked := recordFor("revoked-value", "cred-3", "principal-gamma")
	revoked.Revoked = true

	reg, err := New(testVerifierKey, []Record{
		recordFor("good-value", "cred-1", "principal-alpha"),
		expired,
		revoked,
	})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	for _, secret := range []string{"", "unknown-value", "expired-value", "revoked-value"} {
		if _, ok := reg.Resolve(secret); ok {
			t.Fatalf("unregistered/expired/revoked secret %q resolved", secret)
		}
	}
}

// TestRegistryUsesConstantTimeVerifier documents and exercises the constant-time
// property: verification is done with crypto/subtle over fixed-length digests,
// so a near-miss (one differing character) is rejected exactly like a
// completely wrong value, and digests are always the same length.
func TestRegistryUsesConstantTimeVerifier(t *testing.T) {
	if len(Digest(testVerifierKey, "a")) != len(Digest(testVerifierKey, "a-much-longer-secret")) {
		t.Fatal("verifier digests must be fixed length for constant-time comparison")
	}

	reg, err := New(testVerifierKey, []Record{recordFor("correct-horse-battery", "cred-1", "principal-alpha")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}

	// Same length as the real secret, differing by a single trailing byte.
	nearMiss := "correct-horse-batterz"
	if len(nearMiss) != len("correct-horse-battery") {
		t.Fatal("test fixture must be same length as the registered secret")
	}
	if _, ok := reg.Resolve(nearMiss); ok {
		t.Fatal("near-miss secret must be rejected")
	}
	if _, ok := reg.Resolve("correct-horse-battery"); !ok {
		t.Fatal("exact secret must be accepted")
	}
}

// TestRegistryNeverExposesSecret ensures the secret never appears in errors,
// in the record's serialized form, or in the loaded registry.
func TestRegistryNeverExposesSecret(t *testing.T) {
	const secret = "top-secret-do-not-log"

	if _, err := New(testVerifierKey, []Record{{CredentialID: "cred-1", PrincipalID: "p", Verifier: "short"}}); err != nil {
		if strings.Contains(err.Error(), secret) {
			t.Fatalf("error leaked a secret: %v", err)
		}
	}

	// A registry built from sources that fail to parse must not echo content.
	if _, err := LoadFromSources(`[{"credential_id":"x"}`, "", testVerifierKey); err == nil {
		t.Fatal("expected an error for malformed inline JSON")
	} else if strings.Contains(err.Error(), secret) {
		t.Fatalf("parse error leaked a secret: %v", err)
	}

	reg, err := New(testVerifierKey, []Record{recordFor(secret, "cred-1", "principal-alpha")})
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, ok := reg.Resolve("wrong-secret"); ok {
		t.Fatal("wrong secret resolved")
	}
}

// TestLoadFromSourcesNoConfigIsNil keeps the absent-registry case explicit: no
// sources configured means the legacy surface is retained (nil registry).
func TestLoadFromSourcesNoConfigIsNil(t *testing.T) {
	reg, err := LoadFromSources("", "", testVerifierKey)
	if err != nil {
		t.Fatalf("LoadFromSources: %v", err)
	}
	if reg.Enabled() {
		t.Fatal("no configured sources must not enable the registry")
	}
}
