package controlplane

// Spec 08 R08-4.4.2 .. R08-4.4.4 — staged adoption, promotion and rollback.
//
// These cases are written against the rule that makes governance non-vacuous:
// a promotion is evidence about ONE composition, and a rollback restores one
// value. A test satisfied by "the flag changed" would pass for a deployment
// that promoted on a bundle from a different manifest and rolled back to a
// manifest mutated in place — the two failures the requirement names.

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func testManifest(t *testing.T, mutate func(*Manifest)) *Manifest {
	t.Helper()
	m := &Manifest{
		Version:             "2026.10.18",
		Profile:             "local",
		PolicyDigest:        "sha256:" + repeat("a", 64),
		SupportedInterfaces: []string{InterfaceVersion},
		Components: []Component{
			{
				ID:          "policy-engine",
				Version:     "1.2.0",
				Digest:      "sha256:" + repeat("b", 64),
				SourcePaths: []string{"mcp-server/internal/guardrails/engine.go"},
				Owner:       "security",
				Interface:   InterfaceVersion,
				TrustTier:   TrustTrusted,
				Mandatory:   true,
				Signals:     []string{"policy_queries_total"},
				Rationale:   "the policy decision every tool call depends on",
			},
			{
				ID:          "injection-detector",
				Version:     "1.4.2",
				Digest:      "sha256:" + repeat("c", 64),
				SourcePaths: []string{"mcp-server/internal/guardrails/injection_detection.go"},
				Owner:       "security",
				Interface:   InterfaceVersion,
				TrustTier:   TrustVerified,
				Mandatory:   true,
				Signals:     []string{"injection_scans_total"},
				Rationale:   "untrusted content is scanned before use",
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
				Description: "the detector must be reachable before its verdict can be trusted",
				RequiredComponents: []ComponentRequirement{
					{ID: "injection-detector", MinimumState: FloorReachable},
				},
			},
		},
	}
	if mutate != nil {
		mutate(m)
	}
	if err := m.Resolve(); err != nil {
		t.Fatalf("fixture manifest does not resolve: %v", err)
	}
	return m
}

// passingBundle produces the acceptance evidence a promotion requires.
func passingBundle(m *Manifest, clock func() time.Time) *Bundle {
	b, err := Evaluate(EvaluateRequest{
		Manifest: m,
		Observer: ObserverFunc(func(c Component) Observation {
			return Observation{ComponentID: c.ID, State: StateExercised}
		}),
		SubjectDigest: m.Digest(),
		Clock:         clock,
	})
	if err != nil {
		panic(err)
	}
	return b
}

func TestGovernanceStartsObserveOnly(t *testing.T) {
	g, err := NewGovernance(testManifest(t, nil))
	if err != nil {
		t.Fatalf("new governance: %v", err)
	}
	if g.Mode() != "observe_only" {
		t.Fatalf("a freshly adopted composition must be observe-only, got %q", g.Mode())
	}
}

func TestPromoteRequiresReviewRefAndProfileVersion(t *testing.T) {
	m := testManifest(t, nil)
	g, _ := NewGovernance(m)
	clock := fixedClock()
	accept := passingBundle(m, clock)

	cases := []struct {
		name string
		req  PromotionRequest
		want string
	}{
		{
			name: "no review reference",
			req:  PromotionRequest{ProfileVersion: m.Version, ApprovedBy: "roger", Acceptance: accept},
			want: "review reference",
		},
		{
			name: "no approver",
			req:  PromotionRequest{ProfileVersion: m.Version, ReviewRef: "CHG-1", Acceptance: accept},
			want: "accountable approver",
		},
		{
			name: "no profile version",
			req:  PromotionRequest{ReviewRef: "CHG-1", ApprovedBy: "roger", Acceptance: accept},
			want: "explicit profile version",
		},
		{
			name: "profile version names another revision",
			req:  PromotionRequest{ProfileVersion: "2026.10.17", ReviewRef: "CHG-1", ApprovedBy: "roger", Acceptance: accept},
			want: "names profile version",
		},
		{
			name: "no acceptance suite",
			req:  PromotionRequest{ProfileVersion: m.Version, ReviewRef: "CHG-1", ApprovedBy: "roger"},
			want: "passing acceptance suite",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := g.Promote(tc.req)
			if err == nil {
				t.Fatal("promotion succeeded without the required evidence")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %q does not name %q", err, tc.want)
			}
			if g.Mode() != "observe_only" {
				t.Fatalf("a refused promotion changed the mode to %q", g.Mode())
			}
		})
	}
}

func TestPromoteRefusesBundleFromAnotherComposition(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, _ := NewGovernance(m)

	// A bundle produced under a DIFFERENT manifest — green, and about a
	// different system.
	other := testManifest(t, func(m *Manifest) { m.PolicyDigest = "sha256:" + repeat("d", 64) })
	foreign := passingBundle(other, clock)

	err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-1",
		ApprovedBy:     "roger",
		Acceptance:     foreign,
	})
	if err == nil {
		t.Fatal("promotion accepted evidence from a different manifest")
	}
	if !strings.Contains(err.Error(), "policy") && !strings.Contains(err.Error(), "manifest") {
		t.Fatalf("refusal does not name what differed: %v", err)
	}
	if g.Mode() != "observe_only" {
		t.Fatalf("mode changed to %q after a refused promotion", g.Mode())
	}
}

func TestPromoteRefusesObserveOnlyAcceptance(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, _ := NewGovernance(m)

	observe, err := Evaluate(EvaluateRequest{
		Manifest: m,
		Observer: ObserverFunc(func(c Component) Observation {
			return Observation{ComponentID: c.ID, State: StateExercised}
		}),
		SubjectDigest: m.Digest(),
		ObserveOnly:   true,
		Clock:         clock,
	})
	if err != nil {
		t.Fatalf("evaluate: %v", err)
	}
	if observe.Verdict != VerdictPass {
		t.Fatalf("fixture verdict is %s, expected pass", observe.Verdict)
	}

	err = g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-1",
		ApprovedBy:     "roger",
		Acceptance:     observe,
	})
	if err == nil {
		t.Fatal("observe-only evidence promoted a deployment to blocking")
	}
	if !strings.Contains(err.Error(), "observe-only") {
		t.Fatalf("refusal does not name observe-only: %v", err)
	}
}

func TestPromoteSucceedsOnMatchingPassingEvidenceAndRecordsIt(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, _ := NewGovernance(m)
	g.clock = clock
	accept := passingBundle(m, clock)

	if err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-42",
		ApprovedBy:     "roger",
		Acceptance:     accept,
	}); err != nil {
		t.Fatalf("promotion refused valid evidence: %v", err)
	}
	if g.Mode() != "blocking" {
		t.Fatalf("mode is %q after a successful promotion", g.Mode())
	}
	recs := g.Records()
	if len(recs) != 1 || recs[0].Kind != RecordPromote {
		t.Fatalf("audit trail does not hold the promotion: %+v", recs)
	}
	joined := strings.Join(recs[0].Changes, ",")
	if !strings.Contains(joined, accept.EvaluationID) || !strings.Contains(joined, accept.Digest) {
		t.Fatalf("promotion record does not identify its evidence: %v", recs[0].Changes)
	}
	if recs[0].ReviewRef != "CHG-42" || recs[0].ApprovedBy != "roger" {
		t.Fatalf("promotion record does not name the review: %+v", recs[0])
	}
}

func TestPromoteIsRefusedTwice(t *testing.T) {
	m := testManifest(t, nil)
	clock := fixedClock()
	g, _ := NewGovernance(m)
	req := PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-42",
		ApprovedBy:     "roger",
		Acceptance:     passingBundle(m, clock),
	}
	if err := g.Promote(req); err != nil {
		t.Fatalf("first promotion: %v", err)
	}
	if err := g.Promote(req); err == nil {
		t.Fatal("a second promotion succeeded; the mode was already blocking")
	}
}

func TestAdoptRequiresReviewForGovernedRootChange(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{
			name:   "policy root",
			mutate: func(m *Manifest) { m.PolicyDigest = "sha256:" + repeat("e", 64) },
			want:   "policy_digest",
		},
		{
			name: "component trust tier",
			mutate: func(m *Manifest) {
				m.Components[1].TrustTier = TrustUntrusted
				m.Components[1].Digest = "sha256:" + repeat("c", 64)
			},
			want: "trust_tier",
		},
		{
			name: "mandatory class removed",
			mutate: func(m *Manifest) {
				m.Components[0].Mandatory = false
			},
			want: "mandatory",
		},
		{
			name: "component version",
			mutate: func(m *Manifest) {
				m.Components[0].Version = "1.3.0"
			},
			want: "version",
		},
		{
			name: "component removed",
			mutate: func(m *Manifest) {
				// Dropping a component means dropping the requirement that names
				// it: a requirement for an undeclared component cannot resolve, so
				// a manifest cannot claim a control it no longer declares. Both
				// removals are governed-root changes and both must be reported.
				m.Components = m.Components[:1]
				kept := m.Requirements[:0]
				for _, r := range m.Requirements {
					if r.ID != "injection-evidence" {
						kept = append(kept, r)
					}
				}
				m.Requirements = kept
			},
			want: "removed",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fresh, _ := NewGovernance(testManifest(t, nil))
			changes, err := fresh.Adopt(testManifest(t, tc.mutate), Approval{})
			if err == nil {
				t.Fatalf("adoption of a governed-root change (%v) succeeded with no review", changes)
			}
			if !strings.Contains(err.Error(), "review reference") {
				t.Fatalf("refusal does not demand a review: %v", err)
			}
			if len(changes) == 0 {
				t.Fatal("refusal reported no computed differences")
			}
			joined := strings.Join(changes, "; ")
			if !strings.Contains(joined, tc.want) {
				t.Fatalf("computed differences %q do not name %q", joined, tc.want)
			}
			if fresh.Mode() != "observe_only" {
				t.Fatalf("a refused adoption changed the mode")
			}
		})
	}
}

func TestAdoptWithReviewRecordsTheComputedChange(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	g.clock = fixedClock()

	candidate := testManifest(t, func(m *Manifest) {
		m.PolicyDigest = "sha256:" + repeat("e", 64)
	})
	changes, err := g.Adopt(candidate, Approval{
		ReviewRef:  "CHG-7",
		ApprovedBy: "roger",
		ApprovedAt: fixedClock()(),
	})
	if err != nil {
		t.Fatalf("reviewed adoption refused: %v", err)
	}
	if len(changes) == 0 || !strings.Contains(strings.Join(changes, "; "), "policy_digest") {
		t.Fatalf("adoption did not report the computed change: %v", changes)
	}
	if got := g.Active().PolicyDigest; got != candidate.PolicyDigest {
		t.Fatalf("active policy digest is %s, expected %s", got, candidate.PolicyDigest)
	}
	recs := g.Records()
	if len(recs) != 1 || recs[0].Kind != RecordAdopt {
		t.Fatalf("audit trail does not hold the adoption: %+v", recs)
	}
	if len(recs[0].Changes) == 0 {
		t.Fatal("adoption record carries no change list")
	}
	if g.Mode() != "observe_only" {
		t.Fatal("adoption promoted the composition; promotion is a separate reviewed act")
	}
}

func TestAdoptWithoutGovernedChangeNeedsNoReview(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	// A whitespace-only re-serialisation of the same graph: same identity.
	same := g.Active()
	changes, err := g.Adopt(same, Approval{})
	if err != nil {
		t.Fatalf("re-adopting an identical composition was refused: %v", err)
	}
	if len(changes) != 0 {
		t.Fatalf("identical compositions reported changes: %v", changes)
	}
}

func TestRollbackRestoresPriorManifestAndDigestAsAUnit(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	priorDigest := g.Active().Digest()
	priorPolicy := g.Active().PolicyDigest

	candidate := testManifest(t, func(m *Manifest) {
		m.PolicyDigest = "sha256:" + repeat("e", 64)
		m.Components[0].Version = "1.3.0"
	})
	if _, err := g.Adopt(candidate, Approval{
		ReviewRef:  "CHG-7",
		ApprovedBy: "roger",
		ApprovedAt: fixedClock()(),
	}); err != nil {
		t.Fatalf("adopt: %v", err)
	}

	if err := g.Rollback("candidate rejected in review"); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	back := g.Active()
	if got := back.Digest(); got != priorDigest {
		t.Fatalf("rollback restored digest %s, expected %s", got, priorDigest)
	}
	if back.PolicyDigest != priorPolicy {
		t.Fatalf("rollback restored policy %s, expected %s", back.PolicyDigest, priorPolicy)
	}
	// The two are one value: a digest that matches while the policy does not
	// is the partial rollback the requirement forbids.
	if back.Digest() != g.state.ManifestDigest {
		t.Fatalf("state digest %s disagrees with the restored manifest %s",
			g.state.ManifestDigest, back.Digest())
	}
	if g.state.PolicyDigest != back.PolicyDigest {
		t.Fatalf("state policy %s disagrees with the restored manifest %s",
			g.state.PolicyDigest, back.PolicyDigest)
	}
	if g.Mode() != "observe_only" {
		t.Fatal("rollback left the deployment blocking on a composition nobody promoted")
	}
	recs := g.Records()
	last := recs[len(recs)-1]
	if last.Kind != RecordRollback || last.ManifestDigest != priorDigest {
		t.Fatalf("rollback record does not name the restored composition: %+v", last)
	}
}

func TestRollbackWithoutPriorLatchesNotReady(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	err := g.Rollback("nothing to restore to")
	if err == nil {
		t.Fatal("rollback with no prior composition reported success")
	}
	blocked := g.RollbackBlocked()
	if len(blocked) == 0 {
		t.Fatal("a failed rollback did not leave the instance unready")
	}
	if !strings.Contains(blocked[0], "no prior composition") {
		t.Fatalf("readiness reason does not name the cause: %v", blocked)
	}
}

func TestRollbackDetectsInPlaceMutationOfThePriorUnit(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	priorDigest := g.Active().Digest()

	// Active() hands out a copy, so a caller cannot reach the prior unit. This
	// case reaches it the way a shared pointer would: in package, on the stored
	// manifest. The recorded digest then no longer describes the content, and
	// the rollback must refuse rather than restore "the same object" and report
	// success over a composition somebody changed underneath it.
	g.mu.Lock()
	g.prior = &unit{manifest: cloneManifest(g.active.manifest), digest: g.active.digest}
	g.prior.manifest.Components[0].Version = "9.9.9"
	g.mu.Unlock()

	if err := g.Rollback("candidate rejected"); err == nil {
		t.Fatal("rollback restored a prior unit whose content had changed")
	}
	blocked := g.RollbackBlocked()
	if len(blocked) == 0 || !strings.Contains(blocked[len(blocked)-1], "prior composition digest changed") {
		t.Fatalf("rollback failure does not name the digest drift: %v", blocked)
	}
	if g.prior.manifest.Digest() == priorDigest {
		t.Fatal("fixture did not actually mutate the prior composition")
	}
}

func TestRollbackFailureLatchRequiresADeliberateClear(t *testing.T) {
	g, _ := NewGovernance(testManifest(t, nil))
	if err := g.Rollback("no prior"); err == nil {
		t.Fatal("fixture rollback unexpectedly succeeded")
	}
	if err := g.ClearRollbackFailure(""); err == nil {
		t.Fatal("clearing a latched rollback failure required no review")
	}
	if len(g.RollbackBlocked()) == 0 {
		t.Fatal("failed clear removed the latch")
	}
	if err := g.ClearRollbackFailure("CHG-9"); err != nil {
		t.Fatalf("clearing with a review: %v", err)
	}
	if len(g.RollbackBlocked()) != 0 {
		t.Fatalf("latch survived a reviewed clear: %v", g.RollbackBlocked())
	}
}

func TestGovernanceStateSurvivesARestart(t *testing.T) {
	dir := t.TempDir()
	manifestPath := filepath.Join(dir, "manifest.json")
	statePath := filepath.Join(dir, "state.json")

	m := testManifest(t, nil)
	// Write the fixture manifest through the same JSON form the server reads.
	encoded := mustJSON(t, m)
	if err := os.WriteFile(manifestPath, encoded, 0o600); err != nil {
		t.Fatalf("write manifest: %v", err)
	}

	g, err := OpenGovernance(manifestPath, statePath)
	if err != nil {
		t.Fatalf("open governance: %v", err)
	}
	clock := fixedClock()
	g.clock = clock
	if err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-42",
		ApprovedBy:     "roger",
		Acceptance:     passingBundle(m, clock),
	}); err != nil {
		t.Fatalf("promote: %v", err)
	}

	info, err := os.Stat(statePath)
	if err != nil {
		t.Fatalf("state file was not written: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("state file mode is %04o, expected 0600", info.Mode().Perm())
	}

	reopened, err := OpenGovernance(manifestPath, statePath)
	if err != nil {
		t.Fatalf("reopen governance: %v", err)
	}
	if reopened.Mode() != "blocking" {
		t.Fatalf("promotion did not survive a restart: mode is %q", reopened.Mode())
	}
	if len(reopened.Records()) == 0 {
		t.Fatal("audit trail did not survive a restart")
	}
}

func TestPromotionDoesNotTransferToADifferentComposition(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	m := testManifest(t, nil)
	g, _ := NewGovernance(m)
	g.path = statePath
	clock := fixedClock()
	g.clock = clock
	if err := g.Promote(PromotionRequest{
		ProfileVersion: m.Version,
		ReviewRef:      "CHG-42",
		ApprovedBy:     "roger",
		Acceptance:     passingBundle(m, clock),
	}); err != nil {
		t.Fatalf("promote: %v", err)
	}

	// Same state file, a composition that changed underneath it.
	other := testManifest(t, func(m *Manifest) { m.PolicyDigest = "sha256:" + repeat("f", 64) })
	h, err := OpenGovernanceWithManifest(other, statePath)
	if err != nil {
		t.Fatalf("open with manifest: %v", err)
	}
	if h.Mode() != "observe_only" {
		t.Fatalf("a promotion transferred to a different composition: mode is %q", h.Mode())
	}
	found := false
	for _, r := range h.Records() {
		if r.Kind == RecordAdopt && strings.Contains(r.Reason, "does not transfer") {
			found = true
		}
	}
	if !found {
		t.Fatalf("the non-transfer was not recorded: %+v", h.Records())
	}
}

func TestMalformedStateIsAnErrorNotObserveOnly(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	if err := os.WriteFile(statePath, []byte(`{"observe_only": false, "unknown_key": 1}`), 0o600); err != nil {
		t.Fatalf("write state: %v", err)
	}
	m := testManifest(t, nil)
	if _, err := OpenGovernanceWithManifest(m, statePath); err == nil {
		t.Fatal("a state file with an unknown field was accepted")
	}
	if err := os.WriteFile(statePath, []byte("  \n"), 0o600); err != nil {
		t.Fatalf("write empty state: %v", err)
	}
	if _, err := OpenGovernanceWithManifest(m, statePath); err == nil {
		t.Fatal("an empty state file was accepted")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatalf("marshal fixture: %v", err)
	}
	return raw
}
