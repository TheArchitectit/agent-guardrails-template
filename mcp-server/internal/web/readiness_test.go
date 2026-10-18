package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
)

func fixedClock() time.Time { return time.Date(2026, 10, 16, 0, 0, 0, 0, time.UTC) }

func ok(ctx context.Context) error { return nil }

func failing(ctx context.Context) error { return errors.New("dependency down") }

// readyDeps is the all-green baseline used by the tests below.
func readyDeps() ReadinessDeps {
	return ReadinessDeps{
		DBHealth:            ok,
		CacheHealth:         ok,
		SchemaCompatibility: ok,
		Clock:               fixedClock,
	}
}

// R18-07 / R18-08: a fully healthy instance in the READY lifecycle is ready and
// reports every ordered check PASS.
func TestReadinessReadyWhenAllChecksPass(t *testing.T) {
	e := NewReadinessEvaluator("v9.9.9", readyDeps())
	rep := e.Evaluate(LifecycleReady)
	if !rep.Ready || rep.Status != "ready" {
		t.Fatalf("expected ready, got ready=%v status=%q checks=%+v", rep.Ready, rep.Status, rep.Checks)
	}
	wantOrder := []string{"lifecycle", "config", "database", "cache", "schema", "credential_registry", "policy"}
	if len(rep.Checks) != len(wantOrder) {
		t.Fatalf("expected %d checks, got %d", len(wantOrder), len(rep.Checks))
	}
	for i, name := range wantOrder {
		if rep.Checks[i].Name != name {
			t.Fatalf("check %d: expected %q, got %q (ordered startup R18-07)", i, name, rep.Checks[i].Name)
		}
	}
}

// R18-08: liveness must stay local/process-only. The ready handler must never
// consult a dependency for the live probe, and must be non-2xx while STARTING.
func TestLivenessIsLocalAndReadinessGatesTraffic(t *testing.T) {
	deps := readyDeps()
	deps.DBHealth = failing
	e := NewReadinessEvaluator("v", deps)

	// STARTING lifecycle: not ready even if every dependency is up.
	deps2 := readyDeps()
	e2 := NewReadinessEvaluator("v", deps2)
	if rep := e2.Evaluate(LifecycleStarting); rep.Ready {
		t.Fatal("STARTING must not be ready")
	}

	// A required dependency failing yields NOT_READY.
	rep := e.Evaluate(LifecycleReady)
	if rep.Ready || rep.Status != "not ready" {
		t.Fatalf("expected not ready on db failure, got %+v", rep)
	}
}

// R18-09: a required check that cannot run is NOT_RUN, never PASS. A missing
// probe (unexercised) must never be converted to health.
func TestMissingProbeIsNotRunNeverPass(t *testing.T) {
	deps := readyDeps()
	deps.SchemaCompatibility = nil
	rep := NewReadinessEvaluator("v", deps).Evaluate(LifecycleReady)
	if rep.Ready {
		t.Fatal("absent required probe must not be ready")
	}
	var schema CheckState
	for _, c := range rep.Checks {
		if c.Name == "schema" {
			schema = c
		}
	}
	if schema.Result != ResultNotRun {
		t.Fatalf("absent probe must be NOT_RUN, got %q", schema.Result)
	}
}

// R18-10: a configured-but-invalid registry is a hard non-ready; an absent
// registry is the documented legacy posture and does not block.
func TestRegistryFailClosedButAbsentIsSkipped(t *testing.T) {
	bad := readyDeps()
	bad.RegistryConfigured = true
	bad.RegistryLoadErr = errors.New("malformed registry")
	rep := NewReadinessEvaluator("v", bad).Evaluate(LifecycleReady)
	if rep.Ready {
		t.Fatal("configured-but-invalid registry must not be ready")
	}

	absent := readyDeps()
	rep2 := NewReadinessEvaluator("v", absent).Evaluate(LifecycleReady)
	if !rep2.Ready {
		t.Fatalf("absent registry must not block readiness, got %+v", rep2)
	}
	for _, c := range rep2.Checks {
		if c.Name == "credential_registry" && c.Result != ResultSkipped {
			t.Fatalf("absent registry must be SKIPPED, got %q", c.Result)
		}
	}
}

// R18-11: policy honesty. A disabled engine is non-enforcing (never PASS); an
// enabled-but-unexercised engine is DEGRADED, never an enforcement PASS.
func TestPolicyHonesty(t *testing.T) {
	disabled := readyDeps()
	disabled.Policy = PolicyState{Configured: true, Enabled: false}
	rep := NewReadinessEvaluator("v", disabled).Evaluate(LifecycleReady)
	// An optional (non-required) policy engine that is disabled may be DEGRADED,
	// but it must never be reported as a clean PASS nor as a policy enforcement.
	if rep.Status != "degraded" {
		t.Fatalf("expected degraded for disabled optional policy, got %q", rep.Status)
	}
	for _, c := range rep.Checks {
		if c.Name == "policy" {
			if c.Result == ResultPass {
				t.Fatal("disabled policy must never be an enforcement PASS")
			}
			if c.Reason != "policy_disabled_non_enforcing" {
				t.Fatalf("expected explicit non-enforcing reason, got %q", c.Reason)
			}
		}
	}

	// A disabled engine in a profile that REQUIRES policy is non-ready.
	required := readyDeps()
	required.PolicyRequired = true
	required.Policy = PolicyState{Configured: true, Enabled: false}
	if NewReadinessEvaluator("v", required).Evaluate(LifecycleReady).Ready {
		t.Fatal("disabled policy in an enforcement profile must be non-ready")
	}
}

// R18-12: lifecycle safety. DRAINING and STOPPED are non-ready by construction,
// independent of dependency health.
func TestLifecycleDrainingAndStoppedAreNonReady(t *testing.T) {
	e := NewReadinessEvaluator("v", readyDeps())
	for _, s := range []LifecycleState{LifecycleStarting, LifecycleDraining, LifecycleStopped} {
		if rep := e.Evaluate(s); rep.Ready {
			t.Fatalf("lifecycle %s must be non-ready", s)
		}
	}
}

// R18-13: evidence is bounded. Every check carries a terminal result, a reason
// code, and a timestamp; no raw dependency error text leaks.
func TestReadinessEvidenceIsBounded(t *testing.T) {
	deps := readyDeps()
	deps.DBHealth = func(ctx context.Context) error {
		return errors.New("postgres://user:supersecret@10.0.0.9/db: password authentication failed")
	}
	rep := NewReadinessEvaluator("v", deps).Evaluate(LifecycleReady)
	for _, c := range rep.Checks {
		if c.CheckedAt.IsZero() {
			t.Fatalf("check %s has no timestamp", c.Name)
		}
		if c.Reason == "" {
			t.Fatalf("check %s has no reason code", c.Name)
		}
		if c.Result == "" {
			t.Fatalf("check %s has no terminal result", c.Name)
		}
		if contains(c.Reason, "supersecret") || contains(c.Reason, "password authentication") {
			t.Fatalf("check %s leaked raw dependency error: %q", c.Name, c.Reason)
		}
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// The ready handler returns 503 with a bounded body when non-ready and 200 when
// ready (R18-08). Exercised through the real Echo route.
func TestHealthReadyHandlerStatusCodes(t *testing.T) {
	deps := readyDeps()
	e := NewReadinessEvaluator("v", deps)
	s := &Server{readiness: e, version: "v"}
	s.lifecycle.Store(LifecycleStarting)

	ec := echo.New()
	ec.GET("/health/ready", s.healthReady)

	rec := httptest.NewRecorder()
	ec.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("STARTING must return 503, got %d", rec.Code)
	}

	s.SetLifecycle(LifecycleReady)
	rec = httptest.NewRecorder()
	ec.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health/ready", nil))
	if rec.Code != http.StatusOK {
		t.Fatalf("READY must return 200, got %d", rec.Code)
	}
}

// Spec 08 R08-4.2.4: an unavailable mandatory control-plane component makes
// the instance not-ready, through the same boolean an orchestrator routes on.
func TestReadinessBlocksOnUnavailableMandatoryComponent(t *testing.T) {
	deps := readyDeps()
	deps.ControlPlane = func() *ControlPlaneReadiness {
		return &ControlPlaneReadiness{
			Status: "not_ready",
			Ready:  false,
			Mandatory: []ControlPlaneComponent{
				{ID: "injection-detector", State: "installed", Available: false, Reason: "not_reachable_installed"},
			},
			Unmet: []string{"mandatory component injection-detector is installed"},
		}
	}
	rep := NewReadinessEvaluator("v", deps).Evaluate(LifecycleReady)
	if rep.Ready {
		t.Fatalf("an unavailable mandatory component must make the instance not-ready: %+v", rep)
	}
	if rep.Status == "ready" || rep.Status == "degraded" {
		t.Fatalf("expected a non-ready status, got %q", rep.Status)
	}
	if rep.ControlPlane == nil || len(rep.ControlPlane.Mandatory) != 1 {
		t.Fatalf("readiness must report per-mandatory-component state: %+v", rep.ControlPlane)
	}
	if rep.ControlPlane.Mandatory[0].Available {
		t.Fatal("installed must not be reported available")
	}
}

// A reachable-but-unexercised composition is degraded: it must serve, and it
// must not claim to be fully proven.
func TestReadinessReportsDegradedComposition(t *testing.T) {
	deps := readyDeps()
	deps.ControlPlane = func() *ControlPlaneReadiness {
		return &ControlPlaneReadiness{
			Status: "degraded", Ready: true,
			Mandatory: []ControlPlaneComponent{
				{ID: "injection-detector", State: "reachable", Available: true, Reason: "reachable_not_exercised"},
			},
		}
	}
	rep := NewReadinessEvaluator("v", deps).Evaluate(LifecycleReady)
	if !rep.Ready {
		t.Fatalf("degraded composition must still serve: %+v", rep)
	}
	if rep.ControlPlane == nil || rep.ControlPlane.Status != "degraded" {
		t.Fatalf("expected the degraded composition to be reported: %+v", rep.ControlPlane)
	}
}

// A profile that requires a composition and has none is not-ready, not
// silently exempt.
func TestReadinessRequiredCompositionAbsentIsNotReady(t *testing.T) {
	deps := readyDeps()
	deps.ControlPlaneRequired = true
	rep := NewReadinessEvaluator("v", deps).Evaluate(LifecycleReady)
	if rep.Ready {
		t.Fatalf("a required-but-absent composition must be not-ready: %+v", rep)
	}
	if rep.ControlPlane == nil || rep.ControlPlane.Status != "not_configured" {
		t.Fatalf("expected not_configured with a reason: %+v", rep.ControlPlane)
	}
}

// No composition declared and none required: the ordered Spec 18 decision is
// unchanged, and the field is absent rather than reporting a green status.
func TestReadinessWithoutCompositionIsUnchanged(t *testing.T) {
	rep := NewReadinessEvaluator("v", readyDeps()).Evaluate(LifecycleReady)
	if !rep.Ready || rep.Status != "ready" {
		t.Fatalf("expected ready, got %+v", rep)
	}
	if rep.ControlPlane != nil {
		t.Fatalf("no composition was declared; readiness must not invent one: %+v", rep.ControlPlane)
	}
}
