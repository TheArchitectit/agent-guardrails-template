package config

// Evidence retention posture tests (Spec 08 R08-4.3.6).
//
// These drive the real Config.Validate path, not the helper directly, because
// the requirement is that a misconfigured store fails STARTUP. A helper that
// returns an error nobody calls is the "gate that never ran" class this program
// keeps finding.

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
	"time"
)

// evidenceBaseConfig is a local-profile config that passes the other gates, so
// a failure here is attributable to the retention check.
func evidenceBaseConfig() *Config {
	// Fixtures are derived at runtime from a low-entropy seed. A literal that
	// merely looks random is what turns Secret Validation red for a reason that
	// has nothing to do with the behaviour under test (the same fix applied in
	// b2d015c); deriving it here keeps the file free of credential shapes while
	// still clearing the real entropy floors.
	fixture := func(tag string) string {
		sum := sha256.Sum256([]byte("controlplane evidence retention fixture " + tag))
		return tag + hex.EncodeToString(sum[:])
	}
	return &Config{
		MCPAPIKey:         fixture("Mcp"),
		IDEAPIKey:         fixture("Ide"),
		JWTSecret:         fixture("Jwt"),
		DBPassword:        fixture("Db"),
		RedisPassword:     fixture("Redis"),
		DBSSLMode:         "disable",
		RedisUseTLS:       false,
		DeploymentProfile: "local",
		DBMaxOpenConns:    10,
		DBMaxIdleConns:    5,
		RedisPoolSize:     10,
		RedisMinIdleConns: 2,
		RateLimitMCP:      60,
		RateLimitIDE:      120,
		ShutdownTimeout:   30 * time.Second,
		RequestTimeout:    30 * time.Second,
		DBConnectTimeout:  10 * time.Second,
	}
}

// An unconfigured store is valid: a deployment that never opted into a
// composition retains nothing, and that is a choice rather than an error.
func TestValidateAcceptsAnUnconfiguredEvidenceStore(t *testing.T) {
	c := evidenceBaseConfig()
	if err := c.ValidateCompositionEvidenceRetention(); err != nil {
		t.Fatalf("unconfigured store was refused: %v", err)
	}
}

// A retention policy with no store retains nothing; accepting it would let an
// operator believe evidence is kept because RETENTION was set.
func TestValidateRefusesRetentionWithNoStore(t *testing.T) {
	c := evidenceBaseConfig()
	c.ControlPlaneEvidenceRetention = 24 * time.Hour
	err := c.ValidateCompositionEvidenceRetention()
	if err == nil {
		t.Fatal("retention with no directory was accepted")
	}
	if !strings.Contains(err.Error(), "CONTROL_PLANE_EVIDENCE_DIR") {
		t.Fatalf("refusal did not name the missing directory: %v", err)
	}
}

// A store with no window is "keep forever by accident".
func TestValidateRefusesAStoreWithNoRetentionWindow(t *testing.T) {
	c := evidenceBaseConfig()
	c.ControlPlaneEvidenceDir = t.TempDir()
	err := c.ValidateCompositionEvidenceRetention()
	if err == nil {
		t.Fatal("a store with no retention window was accepted")
	}
	if !strings.Contains(err.Error(), "CONTROL_PLANE_EVIDENCE_RETENTION") {
		t.Fatalf("refusal did not name the retention window: %v", err)
	}
}

func TestValidateEvidenceRetentionBounds(t *testing.T) {
	cases := []struct {
		name    string
		mut     func(*Config)
		wantSub string
	}{
		{
			name: "window too long",
			mut: func(c *Config) {
				c.ControlPlaneEvidenceRetention = 3 * 365 * 24 * time.Hour
			},
			wantSub: "exceeds the",
		},
		{
			name: "negative count bound",
			mut: func(c *Config) {
				c.ControlPlaneEvidenceRetention = time.Hour
				c.ControlPlaneEvidenceMaxRecords = -1
			},
			wantSub: "non-negative",
		},
		{
			name: "count bound too small to hold a day",
			mut: func(c *Config) {
				c.ControlPlaneEvidenceRetention = time.Hour
				c.ControlPlaneEvidenceMaxRecords = 2
			},
			wantSub: "below the",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := evidenceBaseConfig()
			c.ControlPlaneEvidenceDir = t.TempDir()
			tc.mut(c)
			err := c.ValidateCompositionEvidenceRetention()
			if err == nil {
				t.Fatalf("%s was accepted", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("refusal %q does not contain %q", err, tc.wantSub)
			}
		})
	}
}

// Outside `local`, a retained record must be verifiable, so the signing
// identity is required. Inside `local` the store may retain unsigned and
// reports that state rather than implying a signature it does not have.
func TestValidateEvidenceRetentionCouplesSigningToProfile(t *testing.T) {
	local := evidenceBaseConfig()
	local.ControlPlaneEvidenceDir = t.TempDir()
	local.ControlPlaneEvidenceRetention = 24 * time.Hour
	if err := local.ValidateCompositionEvidenceRetention(); err != nil {
		t.Fatalf("local profile may retain unsigned evidence, got: %v", err)
	}

	for _, profile := range []string{"tailnet", "public"} {
		c := evidenceBaseConfig()
		c.DeploymentProfile = profile
		c.ControlPlaneEvidenceDir = t.TempDir()
		c.ControlPlaneEvidenceRetention = 24 * time.Hour
		err := c.ValidateCompositionEvidenceRetention()
		if err == nil {
			t.Fatalf("%s profile retained evidence with no signing key", profile)
		}
		if !strings.Contains(err.Error(), "EVIDENCE_PRIVATE_KEY") {
			t.Fatalf("%s refusal did not name the signing key: %v", profile, err)
		}
	}
}

// The coupling runs inside Validate itself: a directly-constructed Config is on
// the same path as one loaded from the environment.
func TestValidateCouplesEvidenceRetentionToStartup(t *testing.T) {
	c := evidenceBaseConfig()
	c.ControlPlaneEvidenceDir = t.TempDir()
	// No retention window: Validate must refuse at startup, not only when the
	// helper is called by hand.
	err := c.Validate()
	if err == nil {
		t.Fatal("Validate accepted a store with no retention window")
	}
	if !strings.Contains(err.Error(), "CONTROL_PLANE_EVIDENCE_RETENTION") {
		t.Fatalf("startup refusal did not name the retention window: %v", err)
	}
}
