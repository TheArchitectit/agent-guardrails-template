package web

import (
	"context"
	"time"
)

// This file implements the ordered readiness contract of Spec 18 Sec. 3/Sec. 5
// (R18-07 through R18-13). It is deliberately free of database/cache imports so
// the ordered decision can be exercised without a live PostgreSQL or Redis: the
// live probes are injected as function fields by NewServer.

// LifecycleState is the explicit process lifecycle state observed by
// /health/ready (Spec 18 Sec. 5.2 / R18-12).
type LifecycleState string

const (
	// LifecycleStarting is the state before the ordered checks have passed and
	// before listeners accept normal traffic.
	LifecycleStarting LifecycleState = "STARTING"
	// LifecycleReady means the instance may receive normal protected traffic.
	LifecycleReady LifecycleState = "READY"
	// LifecycleDraining is entered before graceful shutdown begins; readiness
	// must be false while requests drain.
	LifecycleDraining LifecycleState = "DRAINING"
	// LifecycleStopped is terminal.
	LifecycleStopped LifecycleState = "STOPPED"
)

// CheckResult is the terminal result of one readiness check (Spec 18 Sec. 3).
type CheckResult string

const (
	ResultPass     CheckResult = "PASS"
	ResultFail     CheckResult = "FAIL"
	ResultNotReady CheckResult = "NOT_READY"
	ResultDegraded CheckResult = "DEGRADED"
	ResultSkipped  CheckResult = "SKIPPED"
	ResultNotRun   CheckResult = "NOT_RUN"
)

// CheckState is the bounded evidence for one readiness check (R18-13). It never
// carries a credential or a raw dependency error: only a stable reason code.
type CheckState struct {
	Name      string      `json:"name"`
	Required  bool        `json:"required"`
	Result    CheckResult `json:"result"`
	Reason    string      `json:"reason,omitempty"`
	CheckedAt time.Time   `json:"checked_at"`
	Version   string      `json:"version,omitempty"`
}

// ReadinessReport is the whole ordered readiness decision.
type ReadinessReport struct {
	Status    string         `json:"status"`
	Lifecycle LifecycleState `json:"lifecycle"`
	Ready     bool           `json:"ready"`
	Version   string         `json:"version,omitempty"`
	Timestamp time.Time      `json:"timestamp"`
	Checks    []CheckState   `json:"checks"`
}

// PolicyState reports the policy engine state distinctly (Spec 18 Sec. 6 / R18-11):
// installed, enabled, and exercised are distinct facts, and a disabled or merely
// installed engine is never an enforcement PASS.
type PolicyState struct {
	// Configured is true when a policy source is present in the deployment.
	Configured bool
	// Enabled is true when the policy engine is actually running.
	Enabled bool
	// Exercised is true once at least one policy evaluation has completed.
	Exercised bool
	// Version is the selected policy version/digest, when known.
	Version string
}

// ReadinessDeps are the live dependency probes and static posture inputs. All
// probe fields may be nil (the check then reports NOT_RUN rather than PASS).
type ReadinessDeps struct {
	// DBHealth is the authenticated application-level database ping (required).
	DBHealth func(ctx context.Context) error
	// CacheHealth is the authenticated cache ping (required for the current
	// server profile).
	CacheHealth func(ctx context.Context) error
	// SchemaCompatibility verifies the applied schema is at or above the
	// image's minimum (required).
	SchemaCompatibility func(ctx context.Context) error

	// RegistryConfigured is true when a credential registry source is set.
	RegistryConfigured bool
	// RegistryLoadErr is non-nil when a configured registry failed to load.
	RegistryLoadErr error

	// Policy is the policy engine posture (Spec 18 Sec. 6 / R18-11).
	Policy PolicyState
	// PolicyRequired is true when the selected profile is an enforcement
	// profile, where a disabled policy engine is non-ready (Spec 18 Sec. 5.3).
	PolicyRequired bool

	// Clock is injectable for deterministic tests; defaults to time.Now.
	Clock func() time.Time
}

// ReadinessEvaluator runs the ordered checks of Spec 18 Sec. 3 and produces a
// bounded report. The order is: lifecycle, config, database, cache, schema,
// credential registry, policy. It never converts FAIL/NOT_READY/NOT_RUN/SKIPPED
// (for a required check) into PASS.
type ReadinessEvaluator struct {
	version string
	deps    ReadinessDeps
}

// NewReadinessEvaluator builds an evaluator.
func NewReadinessEvaluator(version string, deps ReadinessDeps) *ReadinessEvaluator {
	if deps.Clock == nil {
		deps.Clock = time.Now
	}
	return &ReadinessEvaluator{version: version, deps: deps}
}

func (e *ReadinessEvaluator) check(name string, required bool, reason string, version string, run func(ctx context.Context) error) CheckState {
	now := e.deps.Clock().UTC()
	st := CheckState{Name: name, Required: required, CheckedAt: now, Version: version}
	if run == nil {
		// A missing probe is NOT_RUN, never PASS (Spec 18 Sec. 3: unexercised is
		// not evidence of health).
		st.Result = ResultNotRun
		st.Reason = reason + "_probe_absent"
		return st
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := run(ctx); err != nil {
		st.Result = ResultNotReady
		st.Reason = reason + "_failed"
		return st
	}
	st.Result = ResultPass
	st.Reason = reason + "_ok"
	return st
}

// Evaluate runs the ordered checks and returns the report.
func (e *ReadinessEvaluator) Evaluate(lifecycle LifecycleState) ReadinessReport {
	now := e.deps.Clock().UTC()
	rep := ReadinessReport{
		Lifecycle: lifecycle,
		Version:   e.version,
		Timestamp: now,
	}

	// Step 0 (R18-12): lifecycle must be READY. STARTING/DRAINING/STOPPED are
	// non-ready by construction.
	rep.Checks = append(rep.Checks, CheckState{
		Name: "lifecycle", Required: true, CheckedAt: now,
		Result: lifecycleResult(lifecycle),
		Reason: "lifecycle_" + string(lifecycle),
	})

	// Step 1: configuration was parsed and validated before the server was
	// constructed; reaching this evaluator with a non-nil server is the
	// evidence. Recorded explicitly rather than assumed.
	rep.Checks = append(rep.Checks, CheckState{
		Name: "config", Required: true, Result: ResultPass,
		Reason: "config_validated", CheckedAt: now, Version: e.version,
	})

	// Step 2: authenticated dependency pings (required).
	rep.Checks = append(rep.Checks, e.check("database", true, "database_ping", "", e.deps.DBHealth))
	rep.Checks = append(rep.Checks, e.check("cache", true, "cache_ping", "", e.deps.CacheHealth))

	// Step 3: schema compatibility (required, R18-05).
	rep.Checks = append(rep.Checks, e.check("schema", true, "schema_compatibility", "", e.deps.SchemaCompatibility))

	// Step 4: credential registry (R18-10). Absent is the documented legacy
	// migration posture and does not block; configured-but-invalid is a hard
	// non-ready, never a silent fall back to legacy access.
	rep.Checks = append(rep.Checks, registryCheck(e.deps, now))

	// Step 5: policy honesty (R18-11).
	rep.Checks = append(rep.Checks, policyCheck(e.deps, now))

	rep.Ready, rep.Status = overall(rep.Checks, lifecycle)
	return rep
}

func lifecycleResult(s LifecycleState) CheckResult {
	if s == LifecycleReady {
		return ResultPass
	}
	return ResultNotReady
}

func registryCheck(d ReadinessDeps, now time.Time) CheckState {
	st := CheckState{Name: "credential_registry", Required: d.RegistryConfigured, CheckedAt: now}
	switch {
	case !d.RegistryConfigured:
		// Legacy migration mode: no registry configured. Recorded, not a PASS
		// claim, and does not broaden access (R18-10 is about invalid
		// *configured* state).
		st.Required = false
		st.Result = ResultSkipped
		st.Reason = "registry_not_configured"
	case d.RegistryLoadErr != nil:
		st.Result = ResultNotReady
		st.Reason = "registry_load_failed"
	default:
		st.Result = ResultPass
		st.Reason = "registry_loaded"
	}
	return st
}

func policyCheck(d ReadinessDeps, now time.Time) CheckState {
	st := CheckState{Name: "policy", Required: d.PolicyRequired, CheckedAt: now, Version: d.Policy.Version}
	switch {
	case !d.Policy.Configured && !d.Policy.Enabled:
		st.Required = d.PolicyRequired
		st.Result = ResultSkipped
		st.Reason = "policy_not_configured"
	case d.Policy.Enabled && d.Policy.Exercised:
		st.Result = ResultPass
		st.Reason = "policy_enabled_and_exercised"
	case d.Policy.Enabled && !d.Policy.Exercised:
		// Enabled but not yet exercised: never an enforcement PASS (R18-11).
		st.Result = ResultDegraded
		st.Reason = "policy_enabled_not_yet_exercised"
	default:
		// Configured but disabled: non-enforcing, never PASS.
		st.Result = ResultDegraded
		st.Reason = "policy_disabled_non_enforcing"
	}
	return st
}

// overall derives the aggregate status. Required checks must be PASS; a
// DEGRADED/FAIL on an optional check yields DEGRADED (still admissible); any
// required non-PASS, or a non-READY lifecycle, yields NOT_READY.
func overall(checks []CheckState, lifecycle LifecycleState) (bool, string) {
	if lifecycle != LifecycleReady {
		return false, "not ready"
	}
	degraded := false
	for _, c := range checks {
		if c.Required {
			if c.Result != ResultPass {
				return false, "not ready"
			}
			continue
		}
		if c.Result == ResultFail || c.Result == ResultNotReady || c.Result == ResultDegraded {
			degraded = true
		}
	}
	if degraded {
		return true, "degraded"
	}
	return true, "ready"
}
