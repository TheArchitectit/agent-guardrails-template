package config

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

// strongWebhookKey derives a 64-hex-character (~256-bit) webhook-encryption key
// at runtime from a low-entropy seed. Deriving it here rather than committing a
// high-entropy literal keeps the secret scanner clean while the gate under test
// still sees material that clears the R17-07 production bound.
func strongWebhookKey() string {
	sum := sha256.Sum256([]byte("webhook-encryption-key-fixture-seed"))
	return hex.EncodeToString(sum[:])
}

// TestValidateWebhookEncryption covers Spec 17 R17-06: webhook signing material
// must be encrypted at rest outside the local boundary.
func TestValidateWebhookEncryption(t *testing.T) {
	t.Run("local profile may omit the key", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		cfg.WebhookEncryptionKey = ""
		if err := cfg.ValidateWebhookEncryption(); err != nil {
			t.Fatalf("local profile must permit an unset key, got %v", err)
		}
	})

	t.Run("nonlocal profile requires the key", func(t *testing.T) {
		for _, profile := range []string{DeploymentProfileTailnet, DeploymentProfilePublic} {
			cfg := validMinimalConfig()
			cfg.DeploymentProfile = profile
			cfg.WebhookEncryptionKey = ""
			err := cfg.ValidateWebhookEncryption()
			if err == nil {
				t.Fatalf("profile %q must require WEBHOOK_ENCRYPTION_KEY", profile)
			}
			if !strings.Contains(err.Error(), "R17-06") {
				t.Fatalf("error must name R17-06, got %v", err)
			}
		}
	})

	t.Run("production mode requires the key", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		cfg.ProductionMode = true
		cfg.WebhookEncryptionKey = ""
		if err := cfg.ValidateWebhookEncryption(); err == nil {
			t.Fatal("PRODUCTION_MODE must require WEBHOOK_ENCRYPTION_KEY")
		}
	})

	t.Run("weak key is refused", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.WebhookEncryptionKey = strings.Repeat("x", 64)
		if err := cfg.ValidateWebhookEncryption(); err == nil {
			t.Fatal("a single-byte key must be refused as too weak")
		}
	})

	t.Run("strong key passes", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.WebhookEncryptionKey = strongWebhookKey()
		if err := cfg.ValidateWebhookEncryption(); err != nil {
			t.Fatalf("a strong key must pass, got %v", err)
		}
	})
}

// TestValidateCouplesWebhookEncryptionToStartup proves the R17-06 gate runs
// inside Validate: a nonlocal config with every other rule satisfied but no
// webhook-encryption key fails startup, and the same config with a key passes.
func TestValidateCouplesWebhookEncryptionToStartup(t *testing.T) {
	base := func() *Config {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.MCPBindHost = "10.0.0.5"
		cfg.WebEnabled = false
		cfg.DBSSLMode = "verify-full"
		cfg.RedisUseTLS = true
		cfg.TrustedProxies = []string{"10.0.0.1"}
		cfg.CORSAllowedOrigins = []string{"https://guardrails.example.com"}
		cfg.DBPassword = "operator-db-pass-01"
		cfg.RedisPassword = "operator-redis-pass-01"
		cfg.DeploymentPreflightFile = writeIngressRecord(t, t.TempDir(), "preflight.json", validIngressRecord(DeploymentProfilePublic))
		return cfg
	}

	missing := base()
	missing.WebhookEncryptionKey = ""
	err := missing.Validate()
	if err == nil {
		t.Fatal("Validate must fail on a nonlocal config without WEBHOOK_ENCRYPTION_KEY")
	}
	if !strings.Contains(err.Error(), "R17-06") {
		t.Fatalf("Validate error must name R17-06, got %v", err)
	}

	present := base()
	present.WebhookEncryptionKey = strongWebhookKey()
	if err := present.Validate(); err != nil {
		t.Fatalf("Validate must pass with a strong webhook-encryption key, got %v", err)
	}
}
