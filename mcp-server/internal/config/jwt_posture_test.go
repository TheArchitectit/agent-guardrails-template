package config

import (
	"strings"
	"testing"
	"time"
)

// TestValidateJWTPosture covers Spec 17 R17-10: the JWT rotation window must
// stay within the audited bound and a token must not outlive its signing key's
// rotation. Before this existed, JWT_ROTATION_HOURS was an unbounded env value
// (security-audit-config.md row 9, open) that ValidateTimeout never touched.
func TestValidateJWTPosture(t *testing.T) {
	t.Run("unset adopts the documented default and passes", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.JWTRotationHours = 0
		cfg.JWTExpiry = 0
		if err := cfg.ValidateJWTPosture(); err != nil {
			t.Fatalf("an unset posture must default (168h/15m), got %v", err)
		}
	})

	t.Run("default values pass", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.JWTRotationHours = DefaultJWTRotationWindow
		cfg.JWTExpiry = DefaultJWTTTL
		if err := cfg.ValidateJWTPosture(); err != nil {
			t.Fatalf("defaults must pass, got %v", err)
		}
	})

	t.Run("rotation window at the bound passes", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.JWTRotationHours = MaxJWTRotationWindow
		cfg.JWTExpiry = time.Hour
		if err := cfg.ValidateJWTPosture(); err != nil {
			t.Fatalf("169h is the bound; must pass, got %v", err)
		}
	})

	t.Run("rotation window over the bound is refused naming the requirement", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.JWTRotationHours = MaxJWTRotationWindow + time.Hour
		cfg.JWTExpiry = time.Hour
		err := cfg.ValidateJWTPosture()
		if err == nil {
			t.Fatal("an over-long rotation window must be refused")
		}
		if !strings.Contains(err.Error(), "R17-10") {
			t.Fatalf("error must name R17-10, got %v", err)
		}
	})

	t.Run("ttl longer than the rotation window is refused", func(t *testing.T) {
		cfg := validMinimalConfig()
		cfg.JWTRotationHours = time.Hour
		cfg.JWTExpiry = 2 * time.Hour
		err := cfg.ValidateJWTPosture()
		if err == nil {
			t.Fatal("a token that outlives its key rotation must be refused")
		}
		if !strings.Contains(err.Error(), "JWT_EXPIRY") || !strings.Contains(err.Error(), "R17-10") {
			t.Fatalf("error must name JWT_EXPIRY and R17-10, got %v", err)
		}
	})
}

// TestMaxJWTRotationWindowPublished pins the published bound so a silent change
// fails the build rather than quietly widening the accepted policy.
func TestMaxJWTRotationWindowPublished(t *testing.T) {
	if MaxJWTRotationWindow != 168*time.Hour {
		t.Fatalf("MaxJWTRotationWindow = %v, want 168h", MaxJWTRotationWindow)
	}
}

// TestValidateCouplesJWTPostureToStartup proves the gate runs inside Validate
// (the startup path), not only in the standalone helper: an over-long rotation
// window fails Validate even though every other rule is satisfied.
func TestValidateCouplesJWTPostureToStartup(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.JWTRotationHours = MaxJWTRotationWindow + time.Hour
	cfg.JWTExpiry = time.Hour
	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate must fail on an over-long JWT rotation window")
	}
	if !strings.Contains(err.Error(), "R17-10") {
		t.Fatalf("Validate error must name R17-10, got %v", err)
	}
}
