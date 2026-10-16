package config

import (
	"strings"
	"testing"
	"time"
)

// TestPProfDeadline covers the RFC3339 parsing contract for the debug window
// (Spec 17 R17-09): unset is "not configured", a valid value parses, and a
// malformed value is a hard error rather than a silent unbounded window.
func TestPProfDeadline(t *testing.T) {
	cfg := validMinimalConfig()
	if _, configured, err := cfg.PProfDeadline(); err != nil || configured {
		t.Fatalf("unset PProfUntil = (configured=%v, err=%v), want (false, nil)", configured, err)
	}

	cfg.PProfUntil = "2026-10-17T00:00:00Z"
	deadline, configured, err := cfg.PProfDeadline()
	if err != nil || !configured {
		t.Fatalf("valid PProfUntil = (configured=%v, err=%v), want (true, nil)", configured, err)
	}
	if !deadline.Equal(time.Date(2026, 10, 17, 0, 0, 0, 0, time.UTC)) {
		t.Fatalf("deadline = %v, want 2026-10-17T00:00:00Z", deadline)
	}

	cfg.PProfUntil = "not-a-timestamp"
	if _, _, err := cfg.PProfDeadline(); err == nil || !strings.Contains(err.Error(), "PPROF_UNTIL") {
		t.Fatalf("malformed PProfUntil err = %v, want error naming PPROF_UNTIL", err)
	}
}

// TestValidateDebugEndpoint covers the R17-09 posture: a disabled endpoint
// validates to nil; an enabled one requires a loopback bind, a bounded future
// window, and (outside local) a no-proxy-route attestation.
func TestValidateDebugEndpoint(t *testing.T) {
	// Disabled (the default) validates cleanly regardless of other fields.
	if err := validMinimalConfig().ValidateDebugEndpoint(); err != nil {
		t.Fatalf("disabled endpoint err = %v, want nil", err)
	}

	future := time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	local := func() *Config {
		cfg := validMinimalConfig()
		cfg.DeploymentProfile = DeploymentProfileLocal
		cfg.PProfEnabled = true
		cfg.PProfBindHost = "127.0.0.1"
		cfg.PProfPort = 6060
		cfg.PProfUntil = future
		return cfg
	}
	if err := local().ValidateDebugEndpoint(); err != nil {
		t.Fatalf("local loopback+window err = %v, want nil", err)
	}

	// Non-loopback bind is refused.
	nonLoop := local()
	nonLoop.PProfBindHost = "0.0.0.0"
	if err := nonLoop.ValidateDebugEndpoint(); err == nil || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("wildcard bind err = %v, want loopback refusal", err)
	}

	// Missing window is refused.
	noWindow := local()
	noWindow.PProfUntil = ""
	if err := noWindow.ValidateDebugEndpoint(); err == nil || !strings.Contains(err.Error(), "PPROF_UNTIL") {
		t.Fatalf("missing window err = %v, want PPROF_UNTIL requirement", err)
	}

	// Expired window is refused.
	expired := local()
	expired.PProfUntil = time.Now().Add(-time.Minute).UTC().Format(time.RFC3339)
	if err := expired.ValidateDebugEndpoint(); err == nil || !strings.Contains(err.Error(), "expired") {
		t.Fatalf("expired window err = %v, want expiry refusal", err)
	}

	// Malformed window is a hard error (never a silent unbounded window).
	malformed := local()
	malformed.PProfUntil = "tomorrow"
	if err := malformed.ValidateDebugEndpoint(); err == nil || !strings.Contains(err.Error(), "PPROF_UNTIL") {
		t.Fatalf("malformed window err = %v, want PPROF_UNTIL error", err)
	}

	// Outside local, the no-proxy-route attestation is required.
	public := local()
	public.DeploymentProfile = DeploymentProfilePublic
	public.TrustedProxies = []string{"10.0.0.1"}
	public.DBSSLMode = "verify-full"
	public.RedisUseTLS = true
	if err := public.ValidateDebugEndpoint(); err == nil || !strings.Contains(err.Error(), "PPROF_NO_PROXY_ROUTE_ATTESTATION") {
		t.Fatalf("public without attestation err = %v, want attestation requirement", err)
	}
	public.PProfNoProxyRouteAttestation = "no proxy route to 127.0.0.1:6060; loopback only"
	if err := public.ValidateDebugEndpoint(); err != nil {
		t.Fatalf("public with attestation err = %v, want nil", err)
	}
}

// TestDebugEndpointActive covers the runtime gate: a disabled endpoint, a
// missing/malformed window, and an elapsed window all refuse to serve.
func TestDebugEndpointActive(t *testing.T) {
	now := time.Now()

	cfg := validMinimalConfig()
	cfg.PProfEnabled = true
	cfg.PProfUntil = now.Add(time.Hour).UTC().Format(time.RFC3339)
	if active, _ := cfg.DebugEndpointActive(now); !active {
		t.Fatal("endpoint within its window reported inactive, want active")
	}

	cfg.PProfUntil = now.Add(-time.Hour).UTC().Format(time.RFC3339)
	if active, _ := cfg.DebugEndpointActive(now); active {
		t.Fatal("elapsed window reported active, want inactive")
	}

	cfg.PProfUntil = ""
	if active, _ := cfg.DebugEndpointActive(now); active {
		t.Fatal("unconfigured window reported active, want inactive")
	}

	cfg.PProfEnabled = false
	cfg.PProfUntil = now.Add(time.Hour).UTC().Format(time.RFC3339)
	if active, _ := cfg.DebugEndpointActive(now); active {
		t.Fatal("disabled endpoint reported active, want inactive")
	}
}

// TestValidateCouplesDebugPostureToStartup proves the debug gate runs inside
// Config.Validate: a config that satisfies every other rule still fails startup
// when the debug endpoint is enabled without a bounded loopback posture.
func TestValidateCouplesDebugPostureToStartup(t *testing.T) {
	cfg := validMinimalConfig()
	cfg.DeploymentProfile = DeploymentProfileLocal
	cfg.PProfEnabled = true
	cfg.PProfBindHost = "0.0.0.0" // non-loopback administrative surface
	cfg.PProfUntil = time.Now().Add(time.Hour).UTC().Format(time.RFC3339)

	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() = nil, want failure when the debug endpoint is not loopback-only")
	}

	cfg.PProfBindHost = "127.0.0.1"
	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() = %v, want nil for a loopback time-boxed debug endpoint", err)
	}
}
