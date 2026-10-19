package config

import (
	"os"
	"strings"
	"testing"
)

// TestAuditPostureLocalExempt covers the documented development posture: the
// local profile may run with audit logging off (the setting is real, not
// inert), while a nonlocal or production deployment may not.
func TestAuditPostureLocalExempt(t *testing.T) {
	c := &Config{DeploymentProfile: DeploymentProfileLocal, EnableAuditLogging: false}
	if err := c.ValidateAuditPosture(); err != nil {
		t.Fatalf("local profile with audit disabled must be allowed: %v", err)
	}
	if c.AuditDurabilityRequired() {
		t.Fatal("local profile must not require audit durability")
	}
}

// TestAuditPostureNonlocalRefused covers Spec 16 R16-08: a nonlocal/public
// deployment that disables audit logging would record nothing durably, so the
// posture is refused at startup rather than run unaudited.
func TestAuditPostureNonlocalRefused(t *testing.T) {
	for _, profile := range []string{DeploymentProfileTailnet, DeploymentProfilePublic} {
		c := &Config{DeploymentProfile: profile, EnableAuditLogging: false}
		err := c.ValidateAuditPosture()
		if err == nil {
			t.Fatalf("profile %q with ENABLE_AUDIT_LOGGING=false must be refused", profile)
		}
		if !strings.Contains(err.Error(), "ENABLE_AUDIT_LOGGING") {
			t.Fatalf("refusal must name the variable, got: %v", err)
		}
		if !c.AuditDurabilityRequired() {
			t.Fatalf("profile %q must require audit durability", profile)
		}
	}
}

// TestAuditPostureProductionModeRefusedEvenLocal covers the explicit
// PRODUCTION_MODE flag: production mode requires durable audit even when the
// profile is not otherwise nonlocal.
func TestAuditPostureProductionModeRefusedEvenLocal(t *testing.T) {
	c := &Config{DeploymentProfile: DeploymentProfileLocal, ProductionMode: true, EnableAuditLogging: false}
	if err := c.ValidateAuditPosture(); err == nil {
		t.Fatal("PRODUCTION_MODE with audit disabled must be refused")
	}
}

// TestAuditPostureEnabledNonlocalAllowed is the negative control: the gate
// must refuse only a disabled audit posture, never a correctly configured one.
func TestAuditPostureEnabledNonlocalAllowed(t *testing.T) {
	c := &Config{DeploymentProfile: DeploymentProfilePublic, EnableAuditLogging: true}
	if err := c.ValidateAuditPosture(); err != nil {
		t.Fatalf("public profile with audit enabled must be allowed: %v", err)
	}
}

// TestAuditBufferCapacityUsesConfiguredValue is the core regression: the
// startup path previously built the logger with a hardcoded 1000, so the
// validated AUDIT_BUFFER_SIZE never reached the channel.
func TestAuditBufferCapacityUsesConfiguredValue(t *testing.T) {
	c := &Config{AuditBufferSize: 4096}
	if got := c.AuditBufferCapacity(); got != 4096 {
		t.Fatalf("AuditBufferCapacity must return the configured size, got %d", got)
	}

	// A zero/absent value falls back to the published default rather than
	// building a zero-capacity channel.
	empty := &Config{}
	if got := empty.AuditBufferCapacity(); got != DefaultAuditBufferSize {
		t.Fatalf("unset AuditBufferSize must fall back to %d, got %d", DefaultAuditBufferSize, got)
	}
}

// TestValidateCouplesAuditPostureToStartup proves the gate runs inside
// Config.Validate, so a nonlocal/production deployment cannot bypass it by
// constructing a Config directly. Red-on-drift: removing the
// ValidateAuditPosture call from Validate flips this test PASS->FAIL.
func TestValidateCouplesAuditPostureToStartup(t *testing.T) {
	cfg := validMinimalConfig()
	// PRODUCTION_MODE requires durable audit even on the local profile, and it
	// does not pull in the R17-02 nonlocal ingress preflight, so this isolates
	// the audit gate.
	cfg.ProductionMode = true
	cfg.EnableAuditLogging = false

	err := cfg.Validate()
	if err == nil {
		t.Fatal("Validate must refuse production mode with audit logging disabled")
	}
	if !strings.Contains(err.Error(), "ENABLE_AUDIT_LOGGING") {
		t.Fatalf("Validate must surface the audit posture refusal, got: %v", err)
	}

	// Positive control: the same config with audit enabled passes this gate.
	cfg.EnableAuditLogging = true
	cfg.CORSAllowedOrigins = []string{"https://guardrails.example"}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("production config with audit enabled must pass the audit gate, got: %v", err)
	}
}

// TestAuditFlushIntervalRemoved pins the removal of a setting that had no
// implementation: the shipped surface must not advertise a batch/flush
// behaviour the server does not have.
func TestAuditFlushIntervalRemoved(t *testing.T) {
	data, err := os.ReadFile("../../.env.example")
	if err != nil {
		t.Fatalf("cannot read the shipped example env: %v", err)
	}
	if strings.Contains(string(data), "AUDIT_FLUSH_INTERVAL") {
		t.Fatal("AUDIT_FLUSH_INTERVAL has no implementation and must not be documented as a setting")
	}
}
