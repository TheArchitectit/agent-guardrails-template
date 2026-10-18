package controlplane

// Spec 08 R08-4.2.4 — readiness reports per-mandatory-component state and is
// not-ready when a mandatory component is unavailable.
//
// The cases below are written against the rule that makes this requirement
// non-vacuous: presence is not availability. A component that is `installed`
// or `initialized` exists in the process and is still refused, because
// nothing proves a request can reach it. If these tests were satisfied by
// "the component is in the list", the requirement would be met by a report
// that says nothing.

import (
	"testing"

	"time"
)

func readinessBundle(t *testing.T, components []ComponentState, checks []CheckRecord) *Bundle {
	t.Helper()
	return &Bundle{
		Profile:    "test",
		Checks:     checks,
		Components: components,
	}
}

func TestReadinessMandatoryInstalledIsNotAvailable(t *testing.T) {
	b := readinessBundle(t, []ComponentState{
		{ID: "policy-engine", Mandatory: true, State: StateInstalled},
	}, nil)

	rd := b.Readiness()
	if rd.Ready {
		t.Fatalf("a mandatory component that is only installed must not be ready: %+v", rd)
	}
	if rd.Status != ReadinessNotReady {
		t.Fatalf("expected %q, got %q", ReadinessNotReady, rd.Status)
	}
	if len(rd.Mandatory) != 1 {
		t.Fatalf("expected one mandatory row, got %d", len(rd.Mandatory))
	}
	m := rd.Mandatory[0]
	if m.Available {
		t.Fatalf("installed must not be reported available: %+v", m)
	}
	if m.Reason != "not_reachable_installed" {
		t.Fatalf("expected a named reason, got %q", m.Reason)
	}
	if len(rd.Unmet) == 0 {
		t.Fatal("a not-ready decision must name what was unmet")
	}
}

func TestReadinessMandatoryInitializedIsNotAvailable(t *testing.T) {
	// `initialized` is the most tempting false positive: the component was
	// constructed and configured at startup, so every struct field says it is
	// there. It still has no proven entry point.
	b := readinessBundle(t, []ComponentState{
		{ID: "content-filter", Mandatory: true, State: StateInitialized},
	}, nil)
	if rd := b.Readiness(); rd.Ready {
		t.Fatalf("initialized-but-unreachable mandatory component must not be ready: %+v", rd)
	}
}

func TestReadinessMandatoryReachableIsDegradedNotReady(t *testing.T) {
	// A freshly started instance has legitimately exercised nothing. Blocking
	// on "exercised" would make readiness permanently false; reporting `ready`
	// would claim proof that does not exist. Degraded is the honest answer and
	// it must remain distinguishable from `ready`.
	b := readinessBundle(t, []ComponentState{
		{ID: "injection-detector", Mandatory: true, State: StateReachable},
	}, nil)

	rd := b.Readiness()
	if !rd.Ready {
		t.Fatalf("reachable mandatory component must not block traffic: %+v", rd)
	}
	if rd.Status != ReadinessDegraded {
		t.Fatalf("expected %q, got %q — reachable is not exercised", ReadinessDegraded, rd.Status)
	}
	if !rd.Mandatory[0].Available || rd.Mandatory[0].Exercised {
		t.Fatalf("reachable must be available and not exercised: %+v", rd.Mandatory[0])
	}
}

func TestReadinessMandatoryExercisedIsReady(t *testing.T) {
	b := readinessBundle(t, []ComponentState{
		{ID: "injection-detector", Mandatory: true, State: StateExercised},
		{ID: "content-filter", Mandatory: false, State: StateInitialized},
	}, nil)

	rd := b.Readiness()
	if !rd.Ready || rd.Status != ReadinessReady {
		t.Fatalf("expected ready, got %+v", rd)
	}
	if len(rd.Mandatory) != 1 || rd.Mandatory[0].ID != "injection-detector" {
		t.Fatalf("optional components must not appear as mandatory rows: %+v", rd.Mandatory)
	}
}

func TestReadinessMandatoryFailedIsNotReady(t *testing.T) {
	b := readinessBundle(t, []ComponentState{
		{ID: "injection-detector", Mandatory: true, State: StateExercised},
		{ID: "policy-engine", Mandatory: true, State: StateFailed},
	}, nil)

	rd := b.Readiness()
	if rd.Ready {
		t.Fatalf("a failed mandatory component must make the instance not-ready: %+v", rd)
	}
	// `failed` must outrank a previously exercised state on the same
	// component: measurement that found it broken is newer than the fact that
	// it once worked.
	b2 := readinessBundle(t, []ComponentState{
		{ID: "policy-engine", Mandatory: true, State: StateFailed},
	}, nil)
	if rd2 := b2.Readiness(); rd2.Ready || rd2.Mandatory[0].Reason != "component_failed" {
		t.Fatalf("failed must be reported as failed: %+v", rd2)
	}
}

func TestReadinessMandatoryCheckFailureBlocks(t *testing.T) {
	// The component is reachable, but a mandatory check on it ran and refused
	// to prove anything. A reachable component is not a passing control.
	b := readinessBundle(t, []ComponentState{
		{ID: "injection-detector", Mandatory: true, State: StateExercised},
	}, []CheckRecord{
		{ComponentID: "injection-detector", CheckID: "canonical-injection-detected",
			Outcome: OutcomeError, Mandatory: true, ReasonCode: "detector_reported_safe"},
	})

	rd := b.Readiness()
	if rd.Ready {
		t.Fatalf("a failed mandatory check must block readiness: %+v", rd)
	}
}

func TestReadinessOptionalFailureIsReportedNotBlocking(t *testing.T) {
	// The manifest itself declared this component non-essential. Readiness that
	// blocks on an optional input is readiness nobody can satisfy.
	b := readinessBundle(t, []ComponentState{
		{ID: "injection-detector", Mandatory: true, State: StateExercised},
		{ID: "sandbox-isolation", Mandatory: false, State: StateFailed},
	}, nil)

	rd := b.Readiness()
	if !rd.Ready {
		t.Fatalf("an optional failure must not block readiness: %+v", rd)
	}
	if len(rd.Mandatory) != 1 {
		t.Fatalf("optional components must not be counted as mandatory: %+v", rd.Mandatory)
	}
}

func TestReadinessNoMandatoryComponentIsNotReady(t *testing.T) {
	// A profile requiring nothing would make readiness a claim about an empty
	// set — the same vacuity class as a gate that scans zero files and exits 0.
	b := readinessBundle(t, []ComponentState{
		{ID: "content-filter", Mandatory: false, State: StateExercised},
	}, nil)

	rd := b.Readiness()
	if rd.Ready {
		t.Fatalf("a manifest with no mandatory component must not report ready: %+v", rd)
	}
	if len(rd.Unmet) == 0 {
		t.Fatal("expected the absence of mandatory components to be named")
	}
}

func TestReadinessNilBundleIsNotConfigured(t *testing.T) {
	var b *Bundle
	rd := b.Readiness()
	if rd.Status != ReadinessNotConfigured || rd.Ready {
		t.Fatalf("a nil bundle must be not_configured and not ready: %+v", rd)
	}
}

// The decision must be derived from the same evaluation the resource reports,
// so a status page and readiness cannot disagree about one process.
func TestReadinessAgreesWithEvaluation(t *testing.T) {
	m := &Manifest{
		Version:             "test",
		Profile:             "local",
		PolicyDigest:        "sha256:" + repeat("c", 64),
		SupportedInterfaces: []string{InterfaceVersion},
		Components: []Component{
			{
				ID: "injection-detector", Version: "1.0.0",
				Digest:      "sha256:" + repeat("a", 64),
				SourcePaths: []string{"mcp-server/internal/controlplane/evaluate.go"},
				Owner:       "guardrails", Interface: InterfaceVersion,
				TrustTier: TrustTrusted, Mandatory: true,
				Signals:   []string{"injection_scans_total"},
				Rationale: "mandatory detector",
			},
		},
		Requirements: []Requirement{
			{
				ID:          "injection-evidence",
				Description: "the detector must be reachable before its verdict can be trusted",
				RequiredComponents: []ComponentRequirement{
					{ID: "injection-detector", MinimumState: FloorReachable},
				},
			},
		},
	}
	clock := func() time.Time { return time.Date(2026, 10, 18, 0, 0, 0, 0, time.UTC) }

	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		SubjectDigest: "sha256:" + repeat("b", 64),
		Observer:      ObserverFunc(func(c Component) Observation { return Observation{ComponentID: c.ID, State: StateReachable} }),
		Clock:         clock,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if b.Verdict == VerdictPass {
		t.Fatalf("an unexercised mandatory component must not produce a pass verdict: %s", b.Verdict)
	}
	rd := b.Readiness()
	if rd.Status != ReadinessDegraded {
		t.Fatalf("readiness must agree the composition is not proven exercised: %+v", rd)
	}
}

func repeat(s string, n int) string {
	out := make([]byte, 0, len(s)*n)
	for i := 0; i < n; i++ {
		out = append(out, s...)
	}
	return string(out)
}
