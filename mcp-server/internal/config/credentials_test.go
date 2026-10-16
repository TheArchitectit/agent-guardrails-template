package config

import (
	"strings"
	"testing"
)

// TestIsPlaceholderCredential covers the reviewed placeholder set and the
// obvious sample forms (Spec 17 R17-05).
func TestIsPlaceholderCredential(t *testing.T) {
	placeholders := []string{
		"", "changeme", "CHANGEME", " change-me ", "password", "secret",
		"your-password-here", "replace-me", "<password>", "xxxxxxxx",
		"guardrails", "postgres", "redis", "admin",
	}
	for _, p := range placeholders {
		if !IsPlaceholderCredential(p) {
			t.Errorf("IsPlaceholderCredential(%q) = false, want true", p)
		}
	}
	strong := []string{
		"operator-db-pass-01",
		"9f2c7a1e4b6d8f0a2c4e6b8d0f2a4c6e",
		"correct horse battery staple 42!",
	}
	for _, s := range strong {
		if IsPlaceholderCredential(s) {
			t.Errorf("IsPlaceholderCredential(%q) = true, want false", s)
		}
	}
}

// TestValidateNoProductionDefaultCredentials covers Spec 17 R17-05: nonlocal
// and production profiles reject known placeholder/default credentials, while
// the local profile keeps the documented Compose development posture working.
func TestValidateNoProductionDefaultCredentials(t *testing.T) {
	strong := func() *Config {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfilePublic
		cfg.DBSSLMode = "verify-full"
		cfg.RedisUseTLS = true
		cfg.TrustedProxies = []string{"10.0.0.1"}
		cfg.DBPassword = "operator-db-pass-01"
		cfg.RedisPassword = "operator-redis-pass-01"
		return cfg
	}

	t.Run("local profile permits the Compose fallback", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		cfg.DBPassword = "changeme"
		cfg.RedisPassword = "changeme"
		if err := cfg.ValidateNoProductionDefaultCredentials(); err != nil {
			t.Fatalf("local profile must keep the documented development posture, got %v", err)
		}
	})

	t.Run("nonlocal profile rejects the placeholder DB password", func(t *testing.T) {
		cfg := strong()
		cfg.DBPassword = "changeme"
		err := cfg.ValidateNoProductionDefaultCredentials()
		if err == nil {
			t.Fatal("nonlocal + placeholder DB_PASSWORD must fail closed")
		}
		if !strings.Contains(err.Error(), "R17-05") || !strings.Contains(err.Error(), "DB_PASSWORD") {
			t.Fatalf("error must name R17-05 and DB_PASSWORD, got %v", err)
		}
	})

	t.Run("nonlocal profile rejects an empty Redis password", func(t *testing.T) {
		cfg := strong()
		cfg.RedisPassword = ""
		err := cfg.ValidateNoProductionDefaultCredentials()
		if err == nil {
			t.Fatal("nonlocal + empty REDIS_PASSWORD must fail closed")
		}
		if !strings.Contains(err.Error(), "R17-05") || !strings.Contains(err.Error(), "REDIS_PASSWORD") {
			t.Fatalf("error must name R17-05 and REDIS_PASSWORD, got %v", err)
		}
	})

	t.Run("production mode rejects a placeholder JWT secret", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.ProductionMode = true
		cfg.DBPassword = "operator-db-pass-01"
		cfg.RedisPassword = "operator-redis-pass-01"
		cfg.JWTSecret = "changeme"
		err := cfg.ValidateNoProductionDefaultCredentials()
		if err == nil {
			t.Fatal("production + placeholder JWT_SECRET must fail closed")
		}
		if !strings.Contains(err.Error(), "R17-05") || !strings.Contains(err.Error(), "JWT_SECRET") {
			t.Fatalf("error must name R17-05 and JWT_SECRET, got %v", err)
		}
	})

	t.Run("nonlocal profile accepts strong operator credentials", func(t *testing.T) {
		if err := strong().ValidateNoProductionDefaultCredentials(); err != nil {
			t.Fatalf("nonlocal + strong credentials must pass, got %v", err)
		}
	})
}

// TestValidateCouplesDefaultCredentialGateToProfile proves the gate runs inside
// Validate (the startup gate), not only in the standalone helper: a public
// config with the Compose fallback DB password fails startup while every other
// rule is satisfied.
func TestValidateCouplesDefaultCredentialGateToProfile(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.DeploymentProfile = DeploymentProfilePublic
	cfg.MCPBindHost = "10.0.0.5"
	cfg.WebEnabled = false
	cfg.DBSSLMode = "verify-full"
	cfg.RedisUseTLS = true
	cfg.TrustedProxies = []string{"10.0.0.1"}
	cfg.CORSAllowedOrigins = []string{"https://guardrails.example.com"}
	cfg.DBPassword = "changeme"
	cfg.RedisPassword = "operator-redis-pass-01"

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate must fail on a nonlocal placeholder DB credential")
	}
	if !strings.Contains(err.Error(), "R17-05") {
		t.Fatalf("Validate error must name R17-05, got %v", err)
	}

	// Positive control: the same config with a strong credential passes.
	cfg.DBPassword = "operator-db-pass-01"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("public config with strong credentials must pass Validate, got %v", err)
	}
}
