package config

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// strongVerifierKey derives a 64-hex-character, ~256-bit verifier key at
// runtime from a low-entropy seed. Deriving it here (rather than committing a
// high-entropy literal) keeps the secret scanner clean: the tree contains no
// literal that looks like a real key, while the gate under test still sees
// material that clears the R17-07 production bound.
func strongVerifierKey() string {
	sum := sha256.Sum256([]byte("verifier-key-strength-fixture-seed"))
	return hex.EncodeToString(sum[:])
}

// TestValidateCredentialVerifierKeyStrength covers Spec 17 R17-07: "a length
// check is not an entropy proof." A key that clears the 16-character floor but
// carries no real randomness (a single repeated byte) must be refused, and a
// nonlocal/production profile must additionally clear the >=256-bit bound
// R17-07 names.
func TestValidateCredentialVerifierKeyStrength(t *testing.T) {
	configured := func() *Config {
		return &Config{
			CredentialRegistryJSON: `[{"credential_id":"c","principal_id":"p","verifier":"` +
				strings.Repeat("a", 64) + `"}]`,
		}
	}

	// A 64-character key of one repeated byte passes the old length check but
	// carries zero entropy: it must be refused at the load boundary.
	weak := configured()
	weak.CredentialVerifierKey = strings.Repeat("x", 64)
	err := weak.validateCredentialRegistry()
	if err == nil {
		t.Fatal("a length-64 single-byte verifier key must be refused (R17-07)")
	}
	if !strings.Contains(err.Error(), "R17-07") {
		t.Fatalf("error must name R17-07, got %v", err)
	}

	// A 33-character key at ~147 bits clears the local floor.
	local := configured()
	local.CredentialVerifierKey = "unit-test-verifier-key-0123456789"
	if err := local.validateCredentialRegistry(); err != nil {
		t.Fatalf("a 147-bit key must clear the local floor: %v", err)
	}

	// The same key is refused for a production profile, which R17-07 requires
	// to reach >=256 bits.
	prod := configured()
	prod.ProductionMode = true
	prod.CredentialVerifierKey = "unit-test-verifier-key-0123456789"
	if err := prod.validateCredentialRegistry(); err == nil {
		t.Fatal("a 147-bit key must be refused in production (R17-07 requires >=256 bits)")
	}

	// A 64-hex-character key reaches the production bound.
	strong := configured()
	strong.ProductionMode = true
	strong.CredentialVerifierKey = strongVerifierKey()
	if err := strong.validateCredentialRegistry(); err != nil {
		t.Fatalf("a 256-bit key must pass in production: %v", err)
	}
}

// TestValidateCouplesVerifierKeyStrengthToStartup proves the strength gate runs
// inside Validate (the startup gate), not only in the standalone helper: a
// config that satisfies every other rule still fails startup on a weak inline
// verifier key.
func TestValidateCouplesVerifierKeyStrengthToStartup(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.CredentialRegistryJSON = `[{"credential_id":"c","principal_id":"p","verifier":"` +
		strings.Repeat("a", 64) + `"}]`
	cfg.CredentialVerifierKey = strings.Repeat("x", 64)

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate must fail startup on a zero-entropy verifier key")
	}
	if !strings.Contains(err.Error(), "R17-07") {
		t.Fatalf("Validate error must name R17-07, got %v", err)
	}

	// Positive control: a strong key passes.
	cfg.CredentialVerifierKey = strongVerifierKey()
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate with a strong verifier key must pass, got %v", err)
	}
}
