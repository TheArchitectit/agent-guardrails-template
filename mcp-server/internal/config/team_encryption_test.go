package config

import (
	"crypto/rand"
	"encoding/base64"
	"os"
	"strings"
	"testing"
)

// TEAM_ENCRYPTION_KEY at-rest key posture (SEC-007 / Spec 17 R17-06).
//
// The key was documented in the secret inventory for months while nothing
// constructed the manager it configures, so team assignments were written
// plaintext. The wiring now exists; these cases pin the posture so the gate
// cannot drift into either extreme — refusing to start an existing deployment,
// or silently accepting a key too weak to mean anything.

// testTeamKey returns a freshly generated key with enough entropy to clear the
// production floor. It is generated rather than written as a literal on
// purpose: a high-entropy string in source is indistinguishable from a leaked
// credential, and the repository's own secret scanner correctly refuses it.
func testTeamKey(t *testing.T) string {
	t.Helper()
	buf := make([]byte, 48)
	if _, err := rand.Read(buf); err != nil {
		t.Fatalf("generate test key: %v", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf)
}

func TestValidateTeamEncryption_AbsentKeyIsNotAFailure(t *testing.T) {
	// The key is optional at the load boundary by decision: requiring it would
	// break every existing nonlocal deployment in one commit. The startup
	// warning and the fail-closed read path cover the unprotected case.
	for _, profile := range []string{"local", "public", "tailnet", "production"} {
		cfg := &Config{DeploymentProfile: profile}
		if err := cfg.ValidateTeamEncryption(); err != nil {
			t.Fatalf("profile %q: an unset TEAM_ENCRYPTION_KEY must not fail validation: %v", profile, err)
		}
	}
}

func TestValidateTeamEncryption_RejectsWeakKey(t *testing.T) {
	cfg := &Config{DeploymentProfile: "local", TeamEncryptionKey: "short"}
	err := cfg.ValidateTeamEncryption()
	if err == nil {
		t.Fatalf("a 5-character key must be rejected")
	}
	if !strings.Contains(err.Error(), "TEAM_ENCRYPTION_KEY") {
		t.Fatalf("error must name the variable, got: %v", err)
	}
	// The error must never echo the key material (Spec 17 R17-08).
	if strings.Contains(err.Error(), "short") {
		t.Fatalf("error leaked key material: %v", err)
	}
}

func TestValidateTeamEncryption_AcceptsStrongKey(t *testing.T) {
	cfg := &Config{
		DeploymentProfile: "production",
		TeamEncryptionKey: testTeamKey(t),
	}
	if err := cfg.ValidateTeamEncryption(); err != nil {
		t.Fatalf("a strong key must pass: %v", err)
	}
}

// The file source is part of R17-06's requirement, and TEAM_ENCRYPTION_KEY was
// the last secret the inventory recorded as environment-only. This asserts the
// wiring rather than the docs: the file variant must be a recognized source.
func TestTeamEncryptionKeyAcceptsFileSource(t *testing.T) {
	dir := t.TempDir()
	key := testTeamKey(t)
	path := writeSecretFile(t, dir, "team.key", key)

	cfg := &Config{TeamEncryptionKeyFile: path}
	if err := cfg.ResolveSecretSources(); err != nil {
		t.Fatalf("ResolveSecretSources: %v", err)
	}
	if cfg.TeamEncryptionKey != key {
		t.Fatalf("TEAM_ENCRYPTION_KEY_FILE did not populate the key")
	}
}

// A configured-but-group-readable key file must fail closed, exactly like every
// other secret file (R17-06).
func TestTeamEncryptionKeyFileRejectsLooseMode(t *testing.T) {
	dir := t.TempDir()
	path := writeSecretFile(t, dir, "team-loose.key", testTeamKey(t))
	if err := os.Chmod(path, 0o644); err != nil {
		t.Fatalf("chmod: %v", err)
	}
	cfg := &Config{TeamEncryptionKeyFile: path}
	if err := cfg.ResolveSecretSources(); err == nil {
		t.Fatalf("a group/world-readable secret file must be refused")
	}
}
