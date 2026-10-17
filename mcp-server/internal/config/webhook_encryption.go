package config

import (
	"fmt"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/keymaterial"
)

// Webhook signing-material encryption at rest (Spec 17 R17-06).
//
// R17-06 requires that secret material persisting outside a secret store
// "MUST define encrypted-at-rest storage or a managed secret reference". The
// webhook signing key persists as the `webhook_configs.secret_hmac` column (and
// the rotation-overlap column `secret_hmac_previous`), so a database dump, a
// backup, or a replica read exposed the signing secret that authorises every
// delivery to that receiver.
//
// WEBHOOK_ENCRYPTION_KEY is the operator key that seals that material before it
// is written (internal/secretbox, AES-256-GCM). A nonlocal or production
// deployment MUST set it: leaving the signing key plaintext in the database is
// exactly the exposure R17-06 forbids. The `local` profile may leave it unset,
// keeping the documented development posture; that exemption is named, not
// silent.

// ValidateWebhookEncryption requires a strong webhook-encryption key outside the
// local boundary. The key may be supplied inline or through a read-only secret
// file (resolved by ResolveSecretSources before Validate runs).
func (c *Config) ValidateWebhookEncryption() error {
	key := strings.TrimSpace(c.WebhookEncryptionKey)
	if key == "" {
		if c.requiresProductionCredentials() {
			return fmt.Errorf(
				"profile %q requires WEBHOOK_ENCRYPTION_KEY (Spec 17 R17-06): webhook signing material must be encrypted at rest; set the key inline or point WEBHOOK_ENCRYPTION_KEY_FILE at a read-only secret file",
				c.effectiveProfileName())
		}
		return nil
	}
	// The key protects a persisted credential, so it must carry real entropy,
	// not merely length (R17-07's rule, applied here to the same class of key).
	minBits := keymaterial.MinVerifierKeyBits
	if c.requiresProductionCredentials() {
		minBits = keymaterial.ProductionVerifierKeyBits
	}
	if err := keymaterial.ValidateVerifierKeyEntropy(key, minBits); err != nil {
		return fmt.Errorf("WEBHOOK_ENCRYPTION_KEY is too weak for profile %q: %w", c.effectiveProfileName(), err)
	}
	return nil
}
