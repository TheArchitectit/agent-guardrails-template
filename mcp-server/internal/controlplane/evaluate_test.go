package controlplane

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func digest(ch string) string {
	// A stable full sha256-shaped digest for fixtures. Using a real length keeps
	// the manifest validator honest rather than special-casing test input.
	return "sha256:" + strings.Repeat(ch, 64)
}

func validManifest() *Manifest {
	return &Manifest{
		Version:             "2026.10.18",
		Profile:             "local",
		PolicyDigest:        digest("a"),
		SupportedInterfaces: []string{InterfaceVersion},
		Components: []Component{
			{
				ID:          "policy-engine",
				Version:     "1.2.0",
				Digest:      digest("b"),
				SourcePaths: []string{"internal/guardrails/engine.go"},
				Owner:       "guardrails",
				Interface:   InterfaceVersion,
				TrustTier:   TrustTrusted,
				Mandatory:   true,
				Signals:     []string{"policy_evaluations_total"},
				Rationale:   "a request cannot be judged without the policy engine",
			},
			{
				ID:          "injection-detector",
				Version:     "1.4.2",
				Digest:      digest("c"),
				SourcePaths: []string{"internal/guardrails/injection_detection.go"},
				Owner:       "guardrails",
				Interface:   InterfaceVersion,
				TrustTier:   TrustTrusted,
				DependsOn:   []string{"policy-engine"},
				Mandatory:   true,
				Signals:     []string{"injection_scans_total"},
				Rationale:   "prompt-injection detection is a mandatory check for this profile",
			},
			{
				ID:          "webhook-notifier",
				Version:     "0.9.0",
				Digest:      digest("d"),
				SourcePaths: []string{"internal/notifications/dispatcher.go"},
				Owner:       "guardrails",
				Interface:   InterfaceVersion,
				TrustTier:   TrustVerified,
				Mandatory:   false,
				Signals:     []string{"webhook_deliveries_total"},
				Rationale:   "notifications are optional for the local profile",
			},
		},
		Requirements: []Requirement{
			{
				ID:          "policy-evidence",
				Description: "the policy engine must be reachable so a decision can be attributed to a policy identity",
				RequiredComponents: []ComponentRequirement{
					{ID: "policy-engine", MinimumState: FloorReachable},
				},
			},
			{
				ID:          "injection-evidence",
				Description: "the mandatory detector must be reachable before it can be trusted to scan",
				RequiredComponents: []ComponentRequirement{
					{ID: "injection-detector", MinimumState: FloorReachable},
				},
			},
		},
	}
}

// R08-4.1.2 — a graph that cannot be trusted is refused before anything starts.
func TestResolveRejectsUntrustworthyGraphs(t *testing.T) {
	cases := []struct {
		name    string
		mutate  func(*Manifest)
		wantSub string
	}{
		{
			name:    "dependency cycle",
			mutate:  func(m *Manifest) { m.Components[0].DependsOn = []string{"injection-detector"} },
			wantSub: "dependency cycle",
		},
		{
			name:    "self dependency",
			mutate:  func(m *Manifest) { m.Components[1].DependsOn = []string{"injection-detector"} },
			wantSub: "depends on itself",
		},
		{
			name:    "missing dependency",
			mutate:  func(m *Manifest) { m.Components[1].DependsOn = []string{"absent-component"} },
			wantSub: "does not declare",
		},
		{
			name:    "duplicate component",
			mutate:  func(m *Manifest) { m.Components[2].ID = "policy-engine" },
			wantSub: "declared twice",
		},
		{
			name: "mutable version ref",
			mutate: func(m *Manifest) {
				m.Components[2].Version = "latest"
			},
			wantSub: "mutable or unversioned ref",
		},
		{
			name:    "branch version ref",
			mutate:  func(m *Manifest) { m.Components[2].Version = "main" },
			wantSub: "mutable or unversioned ref",
		},
		{
			name:    "unsupported interface",
			mutate:  func(m *Manifest) { m.Components[2].Interface = "controlplane.component/v9" },
			wantSub: "does not support",
		},
		{
			name:    "unknown trust tier",
			mutate:  func(m *Manifest) { m.Components[2].TrustTier = TrustTier("probably-fine") },
			wantSub: "not a known tier",
		},
		{
			name:    "component with no signal",
			mutate:  func(m *Manifest) { m.Components[2].Signals = nil },
			wantSub: "no runtime signal",
		},
		{
			name:    "component with no source path",
			mutate:  func(m *Manifest) { m.Components[2].SourcePaths = nil },
			wantSub: "no source_paths",
		},
		{
			name:    "component with no owner",
			mutate:  func(m *Manifest) { m.Components[2].Owner = "" },
			wantSub: "no owner",
		},
		{
			name:    "short digest",
			mutate:  func(m *Manifest) { m.Components[2].Digest = "sha256:abc" },
			wantSub: "full sha256",
		},
		{
			name:    "empty component set",
			mutate:  func(m *Manifest) { m.Components = nil },
			wantSub: "declares no components",
		},
		{
			name:    "empty interface set",
			mutate:  func(m *Manifest) { m.SupportedInterfaces = nil },
			wantSub: "no supported interface",
		},
		{
			name:    "bad policy digest",
			mutate:  func(m *Manifest) { m.PolicyDigest = "not-a-digest" },
			wantSub: "policy_digest",
		},
		{
			name: "mandatory depends on optional",
			mutate: func(m *Manifest) {
				// Make the mandatory detector depend on the optional notifier.
				m.Components[1].DependsOn = []string{"policy-engine", "webhook-notifier"}
			},
			wantSub: "may not rest on an optional one",
		},
		{
			name: "lower trust supplies higher trust",
			mutate: func(m *Manifest) {
				// An optional verified component consuming an untrusted one. This
				// is deliberately optional-on-optional so the mandatory/optional
				// rule cannot fire first and mask the trust rule.
				m.Components = append(m.Components, Component{
					ID:          "untrusted-scanner",
					Version:     "0.1.0",
					Digest:      digest("f"),
					SourcePaths: []string{"internal/scanner/scanner.go"},
					Owner:       "external",
					Interface:   InterfaceVersion,
					TrustTier:   TrustUntrusted,
					Signals:     []string{"scanner_runs_total"},
					Rationale:   "third-party scanner, optional",
				})
				m.Components[2].DependsOn = []string{"untrusted-scanner"}
			},
			wantSub: "lower-trust component may not supply",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := validManifest()
			tc.mutate(m)
			err := m.Resolve()
			if err == nil {
				t.Fatalf("Resolve accepted a manifest it should refuse (%s)", tc.name)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Fatalf("refusal did not name the cause: got %q, want substring %q", err, tc.wantSub)
			}
		})
	}
}

// Positive control: the same fixture without a mutation resolves. Without this
// the refusal suite above could pass because the fixture is broken for an
// unrelated reason.
func TestResolveAcceptsValidManifest(t *testing.T) {
	m := validManifest()
	if err := m.Resolve(); err != nil {
		t.Fatalf("valid manifest refused: %v", err)
	}
	if got := len(m.MandatoryIDs()); got != 2 {
		t.Fatalf("mandatory set = %d, want 2", got)
	}
}

// R08-4.2 — a library that is linked but unreachable must not be reported as
// an active protection (acceptance criterion 2).
func TestInstalledIsNotReachableIsNotExercised(t *testing.T) {
	m := validManifest()

	states := map[string]IntegrationState{
		// Present in the build, never initialized.
		"policy-engine": StateInstalled,
		// Initialized and wired to an entry point, never invoked.
		"injection-detector": StateReachable,
		// The optional one is genuinely exercised.
		"webhook-notifier": StateExercised,
	}
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{
			ComponentID: c.ID,
			State:       states[c.ID],
			Detail:      "measured",
		}
	})

	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      obs,
		SubjectDigest: digest("e"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}

	if b.Verdict == VerdictPass {
		t.Fatalf("verdict was pass while two mandatory components were unexercised")
	}
	if b.Verdict != VerdictIncomplete {
		t.Fatalf("verdict = %s, want incomplete", b.Verdict)
	}

	byID := map[string]ComponentState{}
	for _, cs := range b.Components {
		byID[cs.ID] = cs
	}
	// The installed-only component must not read as active.
	if byID["policy-engine"].Active {
		t.Fatalf("installed-but-uninitialized component reported active")
	}
	if byID["injection-detector"].Active {
		t.Fatalf("reachable-but-never-invoked component reported active")
	}
	if byID["policy-engine"].State != StateInstalled {
		t.Fatalf("policy-engine state = %s, want installed", byID["policy-engine"].State)
	}
	// The exercised optional component is active, and optional, so it neither
	// rescued nor blocked the verdict.
	if !byID["webhook-notifier"].Active {
		t.Fatalf("exercised component reported not active")
	}

	// The report must say WHICH controls were unproven, naming the state, so an
	// operator can tell "not wired" from "wired and idle".
	joined := strings.Join(b.Unmet, "\n")
	if !strings.Contains(joined, `"policy-engine" is installed`) {
		t.Fatalf("unmet list did not name the installed-only component: %v", b.Unmet)
	}
	if !strings.Contains(joined, `"injection-detector" is reachable`) {
		t.Fatalf("unmet list did not distinguish reachable from exercised: %v", b.Unmet)
	}
}

// R08-4.3.4 — a mandatory non-pass check prevents a pass, and the verdict
// distinguishes block from error (criterion 3).
func TestMandatoryNonPassPreventsPass(t *testing.T) {
	cases := []struct {
		outcome CheckOutcome
		want    Verdict
	}{
		{OutcomeBlock, VerdictBlock},
		{OutcomeError, VerdictFailed},
		{OutcomeSkipped, VerdictIncomplete},
		{OutcomeInconclusive, VerdictIncomplete},
	}

	for _, tc := range cases {
		t.Run(string(tc.outcome), func(t *testing.T) {
			m := validManifest()
			allExercised := ObserverFunc(func(c Component) Observation {
				return Observation{ComponentID: c.ID, State: StateExercised, Detail: "measured"}
			})

			b, err := Evaluate(EvaluateRequest{
				Manifest:      m,
				Observer:      allExercised,
				SubjectDigest: digest("f"),
				Clock:         fixedClock(),
				Checks: []Check{{
					ComponentID: "injection-detector",
					CheckID:     "canonical-injection",
					InputDigest: digest("1"),
					Mandatory:   true,
					Run: func() (CheckOutcome, string, string) {
						return tc.outcome, "fixture_reason", "fixture detail"
					},
				}},
			})
			if err != nil {
				t.Fatalf("Evaluate: %v", err)
			}
			if b.Verdict == VerdictPass {
				t.Fatalf("mandatory %s check produced a pass verdict", tc.outcome)
			}
			if b.Verdict != tc.want {
				t.Fatalf("verdict = %s, want %s", b.Verdict, tc.want)
			}
			if len(b.Unmet) == 0 {
				t.Fatalf("no unmet reason recorded for a non-pass verdict")
			}
		})
	}
}

// Positive control for the above: with every mandatory component exercised and
// its check passing, the verdict IS pass. Without this the suite could pass by
// never producing pass at all.
func TestAllMandatoryExercisedYieldsPass(t *testing.T) {
	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised, Detail: "measured"}
	})
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      obs,
		SubjectDigest: digest("2"),
		Clock:         fixedClock(),
		Checks: []Check{
			{ComponentID: "policy-engine", CheckID: "policy-rules-loaded", InputDigest: digest("3"),
				Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
			{ComponentID: "injection-detector", CheckID: "canonical-injection", InputDigest: digest("4"),
				Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
		},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict != VerdictPass {
		t.Fatalf("verdict = %s, want pass; unmet=%v", b.Verdict, b.Unmet)
	}
	if len(b.Unmet) != 0 {
		t.Fatalf("pass verdict carried unmet reasons: %v", b.Unmet)
	}
}

// A check without an implementation is an error, never a skip and never a pass.
func TestUnimplementedCheckIsError(t *testing.T) {
	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised}
	})
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      obs,
		SubjectDigest: digest("5"),
		Clock:         fixedClock(),
		Checks:        []Check{{ComponentID: "policy-engine", CheckID: "wired-but-empty"}},
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Checks[0].Outcome != OutcomeError {
		t.Fatalf("nil Run produced %s, want error", b.Checks[0].Outcome)
	}
	if b.Verdict == VerdictPass {
		t.Fatalf("an unimplemented mandatory check produced a pass")
	}
}

// Evidence for an undeclared component is refused: it is evidence about
// something the profile does not cover, which would let a check outside the
// manifest influence the verdict.
func TestCheckForUndeclaredComponentIsRefused(t *testing.T) {
	_, err := Evaluate(EvaluateRequest{
		Manifest:      validManifest(),
		SubjectDigest: digest("6"),
		Clock:         fixedClock(),
		Checks: []Check{{
			ComponentID: "not-in-the-manifest",
			CheckID:     "rogue",
			Run:         func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" },
		}},
	})
	if err == nil {
		t.Fatalf("Evaluate accepted a check for an undeclared component")
	}
	if !strings.Contains(err.Error(), "does not declare") {
		t.Fatalf("refusal did not name the cause: %v", err)
	}
}

// R08-4.2.2 — an observer that answers about a different component must not
// have its answer attributed to this one.
func TestMismatchedObservationIsFailed(t *testing.T) {
	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: "some-other-component", State: StateExercised}
	})
	b, err := Evaluate(EvaluateRequest{Manifest: m, Observer: obs, SubjectDigest: digest("7"), Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	for _, cs := range b.Components {
		if cs.State != StateFailed {
			t.Fatalf("component %q state = %s, want failed for a mismatched observation", cs.ID, cs.State)
		}
		if cs.Active {
			t.Fatalf("component %q reported active from a mismatched observation", cs.ID)
		}
	}
	if b.Verdict != VerdictFailed {
		t.Fatalf("verdict = %s, want failed", b.Verdict)
	}
}

// A missing observer means nothing measured the system. Every component is
// `declared` and the verdict cannot be pass.
func TestNilObserverCannotPass(t *testing.T) {
	b, err := Evaluate(EvaluateRequest{
		Manifest:      validManifest(),
		SubjectDigest: digest("8"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict == VerdictPass {
		t.Fatalf("evaluation with no observer produced a pass")
	}
	for _, cs := range b.Components {
		if cs.State != StateDeclared {
			t.Fatalf("component %q state = %s, want declared", cs.ID, cs.State)
		}
	}
}

// R08-4.3.1 / criterion 4 — identical inputs produce an identical result and
// evidence digest.
func TestEvaluationIsStableAcrossRuns(t *testing.T) {
	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised, Detail: "measured"}
	})
	checks := func() []Check {
		return []Check{
			{ComponentID: "injection-detector", CheckID: "canonical-injection", InputDigest: digest("9"),
				Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
			{ComponentID: "policy-engine", CheckID: "policy-rules-loaded", InputDigest: digest("9"),
				Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
		}
	}

	first, err := Evaluate(EvaluateRequest{Manifest: m, Observer: obs, SubjectDigest: digest("9"), Clock: fixedClock(), Checks: checks()})
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := Evaluate(EvaluateRequest{Manifest: m, Observer: obs, SubjectDigest: digest("9"), Clock: fixedClock(), Checks: checks()})
	if err != nil {
		t.Fatalf("second: %v", err)
	}

	if first.EvaluationID != second.EvaluationID {
		t.Fatalf("evaluation id unstable: %s vs %s", first.EvaluationID, second.EvaluationID)
	}
	if first.Digest != second.Digest {
		t.Fatalf("bundle digest unstable: %s vs %s", first.Digest, second.Digest)
	}
	// Declaration order must not change the digest: the same graph is the same
	// manifest.
	reordered := validManifest()
	reordered.Components[0], reordered.Components[2] = reordered.Components[2], reordered.Components[0]
	if reordered.Digest() != m.Digest() {
		t.Fatalf("manifest digest depends on declaration order")
	}
}

// criterion 6 — a replay under a changed component version or policy digest is
// marked non-equivalent, and the difference is named.
func TestReplayWithChangedVersionIsNotEquivalent(t *testing.T) {
	base := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised}
	})
	stored, err := Evaluate(EvaluateRequest{Manifest: base, Observer: obs, SubjectDigest: digest("a"), Clock: fixedClock()})
	if err != nil {
		t.Fatalf("stored: %v", err)
	}

	t.Run("component version changed", func(t *testing.T) {
		changed := validManifest()
		changed.Components[1].Version = "1.4.3"
		changed.Components[1].Digest = digest("f")
		fresh, err := Evaluate(EvaluateRequest{Manifest: changed, Observer: obs, SubjectDigest: digest("a"), Clock: fixedClock()})
		if err != nil {
			t.Fatalf("fresh: %v", err)
		}
		cmp := CompareReplay(stored, fresh)
		if cmp.Equivalent {
			t.Fatalf("a changed component version was reported equivalent")
		}
		joined := strings.Join(cmp.Reasons, "\n")
		if !strings.Contains(joined, "manifest digest changed") {
			t.Fatalf("reasons did not name the manifest change: %v", cmp.Reasons)
		}
	})

	t.Run("policy digest changed", func(t *testing.T) {
		changed := validManifest()
		changed.PolicyDigest = digest("f")
		fresh, err := Evaluate(EvaluateRequest{Manifest: changed, Observer: obs, SubjectDigest: digest("a"), Clock: fixedClock()})
		if err != nil {
			t.Fatalf("fresh: %v", err)
		}
		cmp := CompareReplay(stored, fresh)
		if cmp.Equivalent {
			t.Fatalf("a changed policy digest was reported equivalent")
		}
		if !strings.Contains(strings.Join(cmp.Reasons, "\n"), "policy digest changed") {
			t.Fatalf("reasons did not name the policy change: %v", cmp.Reasons)
		}
	})

	t.Run("identical inputs are equivalent", func(t *testing.T) {
		fresh, err := Evaluate(EvaluateRequest{Manifest: validManifest(), Observer: obs, SubjectDigest: digest("a"), Clock: fixedClock()})
		if err != nil {
			t.Fatalf("fresh: %v", err)
		}
		cmp := CompareReplay(stored, fresh)
		if !cmp.Equivalent {
			t.Fatalf("an identical replay was reported non-equivalent: %v", cmp.Reasons)
		}
	})
}

// R08-4.4.1 — observe-only results are labelled and can never satisfy a
// blocking release gate (criterion 7).
func TestObserveOnlyIsLabelledAndCannotSatisfyBlockingGate(t *testing.T) {
	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		return Observation{ComponentID: c.ID, State: StateExercised}
	})
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      obs,
		SubjectDigest: digest("b"),
		ObserveOnly:   true,
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if !b.ObserveOnly {
		t.Fatalf("observe-only evaluation was not labelled")
	}
	// Even a would-be pass must not be usable as a blocking gate.
	if err := b.Enforce(); err == nil {
		t.Fatalf("observe-only bundle satisfied a blocking gate")
	}

	blocking, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      obs,
		SubjectDigest: digest("b"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if blocking.ObserveOnly {
		t.Fatalf("blocking evaluation was labelled observe-only")
	}
	if err := blocking.Enforce(); err != nil {
		t.Fatalf("a passing blocking bundle failed Enforce: %v", err)
	}
}

// Evidence must carry no seeded secret material (criterion 5).
func TestBundleCarriesNoCredentialMaterial(t *testing.T) {
	const seededSecret = "sk-live-SEEDED-do-not-copy-0123456789"

	m := validManifest()
	obs := ObserverFunc(func(c Component) Observation {
		// A careless observer is exactly the case worth testing: the bundle is
		// serialized and shipped, so whatever it records is what leaves.
		return Observation{
			ComponentID: c.ID,
			State:       StateExercised,
			Detail:      "measured; credential=" + seededSecret,
		}
	})
	b, err := Evaluate(EvaluateRequest{Manifest: m, Observer: obs, SubjectDigest: digest("c"), Clock: fixedClock()})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	raw, err := json.Marshal(b)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(raw), seededSecret) {
		t.Fatalf("evidence bundle carried the seeded secret value")
	}
	if strings.Contains(string(raw), "sk-live-") {
		t.Fatalf("evidence bundle carried a credential-shaped token")
	}
}

func fixedClock() func() time.Time {
	at := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	return func() time.Time { return at }
}
