package config

import (
	"strings"
	"testing"
)

// TestValidateStorageTransit covers Spec 17 R17-04: DB/Redis transit must be
// authenticated and encrypted outside the isolated local boundary, and a
// plaintext posture must fail startup rather than connect in the clear.
func TestValidateStorageTransit(t *testing.T) {
	t.Run("local profile allows plaintext storage transit", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		cfg.DBSSLMode = "disable"
		cfg.RedisUseTLS = false
		if err := cfg.ValidateStorageTransit(); err != nil {
			t.Fatalf("local profile must permit the isolated-bridge exception, got %v", err)
		}
	})

	t.Run("empty profile defaults to local", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = ""
		cfg.DBSSLMode = "disable"
		cfg.RedisUseTLS = false
		if err := cfg.ValidateStorageTransit(); err != nil {
			t.Fatalf("empty profile must default to local, got %v", err)
		}
	})

	t.Run("tailnet requires verify-full and redis TLS", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileTailnet
		cfg.DBSSLMode = "verify-full"
		cfg.RedisUseTLS = true
		if err := cfg.ValidateStorageTransit(); err != nil {
			t.Fatalf("tailnet + verify-full + redis TLS must pass, got %v", err)
		}
	})

	t.Run("public requires verify-full and redis TLS", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DBSSLMode = "verify-full"
		cfg.RedisUseTLS = true
		if err := cfg.ValidateStorageTransit(); err != nil {
			t.Fatalf("public + verify-full + redis TLS must pass, got %v", err)
		}
	})

	// The named negative cases: every posture weaker than authenticated
	// encryption must be refused outside local.
	for _, mode := range []string{"disable", "prefer", "require", "verify-ca"} {
		t.Run("nonlocal rejects DB_SSLMODE="+mode, func(t *testing.T) {
			cfg := validMinimalConfig()
			cfg.DeploymentProfile = DeploymentProfileTailnet
			cfg.DBSSLMode = mode
			cfg.RedisUseTLS = true
			err := cfg.ValidateStorageTransit()
			if err == nil {
				t.Fatalf("nonlocal DB_SSLMODE=%q must fail closed", mode)
			}
			if !strings.Contains(err.Error(), "R17-04") || !strings.Contains(err.Error(), "DB_SSLMODE") {
				t.Fatalf("error must name R17-04 and DB_SSLMODE, got %v", err)
			}
		})
	}

	t.Run("nonlocal rejects redis without TLS", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DBSSLMode = "verify-full"
		cfg.RedisUseTLS = false
		err := cfg.ValidateStorageTransit()
		if err == nil {
			t.Fatal("nonlocal Redis without TLS must fail closed")
		}
		if !strings.Contains(err.Error(), "R17-04") || !strings.Contains(err.Error(), "REDIS_USE_TLS") {
			t.Fatalf("error must name R17-04 and REDIS_USE_TLS, got %v", err)
		}
	})
}

// TestValidateCouplesStorageTransitToProfile proves the coupling runs inside
// Validate (the startup gate), not only in the standalone helper: a tailnet
// config with plaintext storage fails Validate even though every other rule is
// satisfied.
func TestValidateCouplesStorageTransitToProfile(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.DeploymentProfile = DeploymentProfileTailnet
	cfg.MCPBindHost = "100.64.0.5"
	cfg.WebEnabled = false
	cfg.DBSSLMode = "require"
	cfg.RedisUseTLS = false
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate must fail on nonlocal plaintext storage transit")
	}
	if !strings.Contains(err.Error(), "R17-04") {
		t.Fatalf("Validate error must name R17-04, got %v", err)
	}

	// Positive control: the same config with authenticated encryption passes.
	cfg.DBSSLMode = "verify-full"
	cfg.RedisUseTLS = true
	if err := cfg.Validate(); err != nil {
		t.Fatalf("nonlocal config with verify-full + redis TLS must pass Validate, got %v", err)
	}
}
