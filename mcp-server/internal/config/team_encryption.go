package config

import (
	"fmt"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/keymaterial"
)

// Team data encryption at rest (SEC-007 / Spec 17 R17-06).
//
// Team assignments persist as JSON under `.teams/`, and the assignee names in
// them are personal data. R17-06 requires that such material either live in a
// managed secret store or be encrypted at rest, and the secret inventory has
// documented TEAM_ENCRYPTION_KEY as the key that does it.
//
// Before this change the key was documented and unused: nothing in the server
// constructed the manager it configures, so team files were written as plaintext
// JSON at mode 0644 no matter what an operator set. That is the "library
// mistaken for a shipped control" failure this program keeps finding. The
// wiring now exists (cmd/server builds the manager once and every team tool
// receives it) and the file is written 0600.
//
// Posture, stated rather than implied:
//   - The key is OPTIONAL at the load boundary. Team data is an optional
//     feature, and refusing to start without the key would break every existing
//     deployment in one commit — that is a posture change requiring the owner's
//     decision, not a side effect of a defect fix. The server logs a WARN naming
//     exactly what is unprotected when the key is absent on a nonlocal profile.
//   - A CONFIGURED key must carry real entropy, not merely length: the same
//     floor R17-07 applies to the webhook key, for the same reason. This is the
//     part that is unconditional, because a weak key that appears to protect
//     data is worse than the honest absence of one.
//   - Reading sealed data with no key, a wrong key, or a tampered value is a
//     hard error in internal/team — never a silent fall back to the ciphertext.
//
// OPEN (owner-gated): whether a nonlocal/production profile should REQUIRE the
// key at startup, as WEBHOOK_ENCRYPTION_KEY already does. Doing so is a
// deliberate breaking change for existing deployments and is recorded as a
// pending owner decision rather than taken here.

// ValidateTeamEncryption checks a configured team at-rest key for usable
// entropy. An unset key is not an error at this boundary (see the posture note
// above); the startup warning and the fail-closed read path are what keep an
// unconfigured deployment honest.
func (c *Config) ValidateTeamEncryption() error {
	key := strings.TrimSpace(c.TeamEncryptionKey)
	if key == "" {
		return nil
	}
	minBits := keymaterial.MinVerifierKeyBits
	if c.requiresProductionCredentials() {
		minBits = keymaterial.ProductionVerifierKeyBits
	}
	if err := keymaterial.ValidateVerifierKeyEntropy(key, minBits); err != nil {
		return fmt.Errorf("TEAM_ENCRYPTION_KEY is too weak for profile %q: %w", c.effectiveProfileName(), err)
	}
	return nil
}
