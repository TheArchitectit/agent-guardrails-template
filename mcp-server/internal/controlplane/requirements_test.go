package controlplane

// Spec 08 R08-4.1.5 — the per-profile requirement set.
//
// The requirement is one sentence: "Profiles MUST specify required checks and
// minimum evidence, not merely a list of installed components." It is easy to
// satisfy on paper and easy to satisfy vacuously, so the tests below are built
// around the specific ways a requirement set can look like an obligation and
// constrain nothing:
//
//   - an empty list (the pre-change state: components only);
//   - a requirement naming no component and no check;
//   - a floor of `installed`, which restates the component list;
//   - a check that nothing ever runs;
//   - a mandatory component no requirement holds to anything.
//
// Each of those is proven to FAIL resolution below, not merely to be
// discouraged in a comment.

import (
	"strings"
	"testing"
)

// requirementFixture is a manifest whose requirement set is real: one component
// floor and one check that must pass.
func requirementFixture(t *testing.T) *Manifest {
	t.Helper()
	m := validManifest()
	m.Requirements = []Requirement{
		{
			ID:          "policy-identity",
			Description: "a decision is attributable to the policy identity in force",
			RequiredComponents: []ComponentRequirement{
				{ID: "policy-engine", MinimumState: FloorReachable},
			},
		},
		{
			ID:          "injection-proof",
			Description: "the detector must be reachable and must flag a canonical injection",
			RequiredComponents: []ComponentRequirement{
				{ID: "injection-detector", MinimumState: FloorReachable},
			},
			RequiredChecks: []string{"canonical-injection"},
		},
	}
	if err := m.Resolve(); err != nil {
		t.Fatalf("requirement fixture does not resolve: %v", err)
	}
	return m
}

// passingCheck is the check the injection requirement names.
// allExercised observes every component as proven.
func allExercised(c Component) Observation {
	return Observation{ComponentID: c.ID, State: StateExercised, Detail: "measured"}
}

// passingCheck is the check the injection requirement names.
func passingCheck() Check {
	return Check{
		ComponentID: "injection-detector",
		CheckID:     "canonical-injection",
		InputDigest: digest("9"),
		Run:         func() (CheckOutcome, string, string) { return OutcomePass, "detected", "" },
	}
}

// The pre-change manifest declared components and nothing else. That shape is
// now refused at resolve time, because a profile that states no obligations is
// exactly the "merely a list of installed components" the requirement forbids.
func TestManifestWithoutRequirementsIsRefused(t *testing.T) {
	m := validManifest()
	m.Requirements = nil
	err := m.Resolve()
	if err == nil {
		t.Fatal("a manifest declaring no profile requirements resolved")
	}
	if !strings.Contains(err.Error(), "R08-4.1.5") {
		t.Fatalf("refusal does not name the requirement it enforces: %v", err)
	}
}

func TestRequirementRefusals(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{
			name: "requirement names nothing",
			mutate: func(m *Manifest) {
				m.Requirements = append(m.Requirements, Requirement{
					ID: "decoration", Description: "looks like an obligation",
				})
			},
			want: "nothing could ever fail it",
		},
		{
			name: "requirement with no description",
			mutate: func(m *Manifest) {
				m.Requirements[0].Description = " "
			},
			want: "no description",
		},
		{
			name: "requirement id is not stable",
			mutate: func(m *Manifest) {
				m.Requirements[0].ID = "Policy Identity"
			},
			want: "stable lowercase identifier",
		},
		{
			name: "duplicate requirement id",
			mutate: func(m *Manifest) {
				dup := m.Requirements[0]
				m.Requirements = append(m.Requirements, dup)
			},
			want: "declared twice",
		},
		{
			name: "floor of installed",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents[0].MinimumState = EvidenceFloor("installed")
			},
			want: "minimum_state",
		},
		{
			name: "floor of declared",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents[0].MinimumState = EvidenceFloor("declared")
			},
			want: "minimum_state",
		},
		{
			name: "floor of initialized",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents[0].MinimumState = EvidenceFloor("initialized")
			},
			want: "minimum_state",
		},
		{
			name: "floor of failed",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents[0].MinimumState = EvidenceFloor("failed")
			},
			want: "minimum_state",
		},
		{
			name: "requirement names an undeclared component",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents = []ComponentRequirement{
					{ID: "not-in-the-manifest", MinimumState: FloorReachable},
				}
			},
			want: "does not declare",
		},
		{
			name: "component listed twice in one requirement",
			mutate: func(m *Manifest) {
				m.Requirements[0].RequiredComponents = append(
					m.Requirements[0].RequiredComponents,
					ComponentRequirement{ID: "policy-engine", MinimumState: FloorReachable})
			},
			want: "twice",
		},
		{
			name: "check named twice",
			mutate: func(m *Manifest) {
				m.Requirements[1].RequiredChecks = []string{"canonical-injection", "canonical-injection"}
			},
			want: "twice",
		},
		{
			name: "empty check id",
			mutate: func(m *Manifest) {
				m.Requirements[1].RequiredChecks = []string{"  "}
			},
			want: "empty check id",
		},
		{
			name: "minimum_passing_checks without checks",
			mutate: func(m *Manifest) {
				zero := 1
				m.Requirements[0].MinimumPassingChecks = &zero
			},
			want: "without naming any check",
		},
		{
			name: "minimum_passing_checks of zero",
			mutate: func(m *Manifest) {
				zero := 0
				m.Requirements[1].MinimumPassingChecks = &zero
			},
			want: "between 1 and the number named",
		},
		{
			name: "minimum_passing_checks above the count",
			mutate: func(m *Manifest) {
				two := 2
				m.Requirements[1].MinimumPassingChecks = &two
			},
			want: "between 1 and the number named",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			m := requirementFixture(t)
			tc.mutate(m)
			err := m.Resolve()
			if err == nil {
				t.Fatalf("the manifest resolved with a defective requirement set")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %q does not mention %q", err.Error(), tc.want)
			}
		})
	}
}

// A mandatory control that no requirement asks evidence for is a label. The
// resolver refuses it rather than shipping a `mandatory: true` row that
// constrains nothing.
func TestMandatoryComponentWithoutARequirementIsRefused(t *testing.T) {
	m := requirementFixture(t)
	// Drop the requirement covering the detector, leaving the component.
	m.Requirements = m.Requirements[:1]
	err := m.Resolve()
	if err == nil {
		t.Fatal("a mandatory component with no evidence obligation resolved")
	}
	if !strings.Contains(err.Error(), "constrains nothing") {
		t.Fatalf("refusal does not explain the defect: %v", err)
	}
}

// Two requirements holding one component to different floors are refused: the
// weaker one would decide the answer, so the stronger one would be decoration.
func TestConflictingFloorsAreRefused(t *testing.T) {
	m := requirementFixture(t)
	m.Requirements = append(m.Requirements, Requirement{
		ID:          "injection-stronger",
		Description: "an observed invocation, not merely reachability",
		RequiredComponents: []ComponentRequirement{
			{ID: "injection-detector", MinimumState: FloorExercised},
		},
	})
	err := m.Resolve()
	if err == nil {
		t.Fatal("two different floors for one component resolved")
	}
	if !strings.Contains(err.Error(), "one component has one floor") {
		t.Fatalf("refusal does not explain the conflict: %v", err)
	}
}

// The evaluation half: a requirement the profile declared and this run did not
// satisfy must prevent `pass`, and must say which obligation fell short.
func TestUnmetRequirementPreventsPass(t *testing.T) {
	m := requirementFixture(t)
	// Exercised, so the base mandatory-component rule (R08-4.3.4) is satisfied
	// and the only thing left to decide is the PROFILE requirement set. That
	// separation matters: a pass assertion on a reachable-only composition
	// would fail for a reason that has nothing to do with R08-4.1.5.
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		Checks:        []Check{passingCheck()},
		SubjectDigest: digest("e"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict != VerdictPass {
		t.Fatalf("a manifest whose requirements are met did not pass: %s (%v)", b.Verdict, b.Unmet)
	}
	for _, r := range b.Requirements {
		if !r.Met {
			t.Fatalf("requirement %q reported unmet: %v", r.ID, r.Unmet)
		}
	}

	// Now the same manifest with both components exercised but the check that
	// backs the requirement never supplied: the requirement is unevidenced.
	b, err = Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		SubjectDigest: digest("e2"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict == VerdictPass {
		t.Fatal("a requirement with an unevidenced check still produced a pass")
	}
	if !strings.Contains(strings.Join(b.Unmet, "; "), `check "canonical-injection" did not run`) {
		t.Fatalf("unmet reasons do not name the missing check: %v", b.Unmet)
	}

	// And a component measured below its floor is reported against that floor,
	// naming the floor rather than a bare "not exercised".
	m2 := requirementFixture(t)
	m2.Requirements[0].RequiredComponents[0].MinimumState = FloorExercised
	if err := m2.Resolve(); err != nil {
		t.Fatalf("fixture: %v", err)
	}
	b, err = Evaluate(EvaluateRequest{
		Manifest: m2,
		Observer: ObserverFunc(func(c Component) Observation {
			if c.ID == "policy-engine" {
				return Observation{ComponentID: c.ID, State: StateReachable}
			}
			return Observation{ComponentID: c.ID, State: StateExercised}
		}),
		Checks:        []Check{passingCheck()},
		SubjectDigest: digest("e3"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict == VerdictPass {
		t.Fatal("a component below its declared floor still produced a pass")
	}
	joined := strings.Join(b.Unmet, "; ")
	if !strings.Contains(joined, "profile requirement") || !strings.Contains(joined, "below the exercised floor") {
		t.Fatalf("unmet reasons do not name the floor shortfall: %v", b.Unmet)
	}
}

// A check the profile names but the wiring never supplied is `did not run`, and
// that is a defect in the composition rather than a refusal — so the verdict is
// `failed`, not the softer `incomplete`.
func TestNamedCheckThatNeverRanIsAFailureNotAnIncompleteness(t *testing.T) {
	m := requirementFixture(t)
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		SubjectDigest: digest("f"),
		Clock:         fixedClock(),
		// No checks at all: the requirement names one.
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	if b.Verdict != VerdictFailed {
		t.Fatalf("verdict = %s, want failed for a named check that never ran", b.Verdict)
	}
	joined := strings.Join(b.Unmet, "; ")
	if !strings.Contains(joined, "did not run") {
		t.Fatalf("unmet reasons do not say the check never ran: %v", b.Unmet)
	}
}

// A minimum below the number named is honoured, and a requirement that does not
// clear its own floor is unmet even when most of its checks passed.
func TestMinimumPassingChecksFloorIsHonoured(t *testing.T) {
	m := requirementFixture(t)
	two := 2
	m.Requirements[1].RequiredChecks = []string{"a", "b", "c"}
	m.Requirements[1].MinimumPassingChecks = &two
	if err := m.Resolve(); err != nil {
		t.Fatalf("fixture: %v", err)
	}

	// Two pass, one blocks: the declared floor is met, so the requirement is
	// met even though one check refused the subject. The block still prevents a
	// pass verdict through the mandatory-check path — the two rules are
	// independent on purpose.
	checks := []Check{
		{ComponentID: "injection-detector", CheckID: "a", InputDigest: digest("1"),
			Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
		{ComponentID: "injection-detector", CheckID: "b", InputDigest: digest("2"),
			Run: func() (CheckOutcome, string, string) { return OutcomePass, "ok", "" }},
		{ComponentID: "injection-detector", CheckID: "c", InputDigest: digest("3"),
			Run: func() (CheckOutcome, string, string) { return OutcomeSkipped, "not_applicable", "" }},
	}
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		Checks:        checks,
		SubjectDigest: digest("7"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	var inj *RequirementResult
	for i := range b.Requirements {
		if b.Requirements[i].ID == "injection-proof" {
			inj = &b.Requirements[i]
		}
	}
	if inj == nil {
		t.Fatal("the evaluation reported no result for the injection requirement")
	}
	if !inj.Met {
		t.Fatalf("a requirement that met its own floor reported unmet: %v", inj.Unmet)
	}

	// One pass, two skipped: below the floor.
	checks[1] = Check{ComponentID: "injection-detector", CheckID: "b", InputDigest: digest("2"),
		Run: func() (CheckOutcome, string, string) { return OutcomeInconclusive, "unclear", "" }}
	b, err = Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		Checks:        checks,
		SubjectDigest: digest("8"),
		Clock:         fixedClock(),
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	for _, r := range b.Requirements {
		if r.ID == "injection-proof" && r.Met {
			t.Fatalf("a requirement below its passing floor reported met: %v", b.Unmet)
		}
	}
	if b.Verdict == VerdictPass {
		t.Fatal("an unmet passing floor produced a pass")
	}
}

// The requirement set is part of what a manifest IS, so two manifests that
// differ only in their obligations must not share a digest. If they did, an
// evaluation could be replayed against a requirement set it never ran under.
func TestManifestDigestCoversRequirements(t *testing.T) {
	a := requirementFixture(t)
	b := requirementFixture(t)
	b.Requirements[0].RequiredComponents[0].MinimumState = FloorExercised
	// One component is now held at two floors, which must not resolve — so
	// drop the conflicting entry to isolate the digest question.
	b.Requirements = b.Requirements[1:]
	b.Requirements = append(b.Requirements, Requirement{
		ID:          "policy-observation",
		Description: "the engine must be seen evaluating, not merely reachable",
		RequiredComponents: []ComponentRequirement{
			{ID: "policy-engine", MinimumState: FloorExercised},
		},
	})
	if err := b.Resolve(); err != nil {
		t.Fatalf("second fixture does not resolve: %v", err)
	}
	if a.Digest() == b.Digest() {
		t.Fatal("two manifests with different requirement sets share a digest")
	}
}

// Declaration order must not change identity: the same obligations listed in a
// different order are the same manifest.
func TestRequirementOrderDoesNotChangeTheDigest(t *testing.T) {
	a := requirementFixture(t)
	b := requirementFixture(t)
	b.Requirements[0], b.Requirements[1] = b.Requirements[1], b.Requirements[0]
	if a.Digest() != b.Digest() {
		t.Fatal("reordering requirements changed the manifest digest")
	}
}

// Lowering a floor is a governed-root change (R08-4.4.3): it changes what every
// future evaluation counts as proof without touching a component. A diff that
// only walked components would report "no change" for exactly that edit.
func TestLoweringARequirementFloorRequiresReview(t *testing.T) {
	prior := requirementFixture(t)
	prior.Requirements[1].RequiredComponents[0].MinimumState = FloorExercised
	if err := prior.Resolve(); err != nil {
		t.Fatalf("prior fixture: %v", err)
	}

	next := requirementFixture(t) // injection-detector back at `reachable`

	changes := RequiresReview(prior, next)
	if len(changes) == 0 {
		t.Fatal("lowering a requirement floor reported no governed-root change")
	}
	joined := strings.Join(changes, "; ")
	if !strings.Contains(joined, "floor: exercised -> reachable") {
		t.Fatalf("computed differences do not name the lowered floor: %v", changes)
	}

	// And the refusal is real: an unreviewed adoption of that change fails.
	g, err := NewGovernance(prior)
	if err != nil {
		t.Fatalf("governance: %v", err)
	}
	if _, err := g.Adopt(next, Approval{}); err == nil {
		t.Fatal("a lowered evidence floor was adopted with no review")
	}
}

// Removing a requirement entirely is also a governed-root change: the control
// it held stops being required at all.
func TestRemovingARequirementRequiresReview(t *testing.T) {
	prior := requirementFixture(t)
	next := requirementFixture(t)
	next.Requirements = next.Requirements[:1]
	// The detector is mandatory and now held by nothing, so the manifest must
	// not resolve — which is itself the point. Assert both: the diff names the
	// removal, and the resulting manifest is refused.
	changes := RequiresReview(prior, next)
	if !strings.Contains(strings.Join(changes, "; "), `requirement "injection-proof" removed`) {
		t.Fatalf("diff does not name the removed requirement: %v", changes)
	}
	if err := next.Resolve(); err == nil {
		t.Fatal("a manifest whose mandatory control lost its requirement resolved")
	}
}

// Readiness and evidence must not disagree about one process: an unmet profile
// requirement makes the instance not-ready, because the same bundle calls it
// incomplete.
func TestReadinessReflectsUnmetRequirements(t *testing.T) {
	m := requirementFixture(t)
	b, err := Evaluate(EvaluateRequest{
		Manifest:      m,
		Observer:      ObserverFunc(allExercised),
		SubjectDigest: digest("a"),
		Clock:         fixedClock(),
		// The named check never runs.
	})
	if err != nil {
		t.Fatalf("Evaluate: %v", err)
	}
	rd := b.Readiness()
	if rd.Ready {
		t.Fatalf("readiness is ready while a profile requirement is unmet: %+v", rd)
	}
	if !strings.Contains(strings.Join(rd.Unmet, "; "), "profile requirement") {
		t.Fatalf("readiness unmet reasons do not name the profile requirement: %v", rd.Unmet)
	}
}
