package guardrails

// Spec 06 — a probe must not be borrowed from a control that is not the control
// the requirement names.
//
// The defect these tests exist to prevent was in the measured path, not the
// declared one. `content_filter_watermark` was aliased to the content filter's
// probe, so EU AI Act Art. 50 (marking of AI-generated content) reported
// OBSERVED ACTIVE as soon as any semantic content-filter backend was attached —
// while no watermarking code exists anywhere in the tree. Art. 50 declares
// itself a gap and names the missing implementation; the alias turned an
// acknowledged, declared gap into a measured compliance claim on the exact path
// built to stop restating declarations.

import (
	"context"
	"testing"
)

// probeThatReports is a minimal ProbeSet whose single control is active, used
// to build a set that is non-empty but does not cover the requirement under
// test — the shape a real deployment has.
func probeThatReports(feature string) ProbeSet {
	return ProbeSet{
		feature: func(context.Context) (FeatureStatus, string) {
			return FeatureActive, "synthetic observation"
		},
	}
}

// The content filter's own status must not stand in for watermarking. This is
// the assertion that would have failed before the fix: with the alias in place,
// attaching a content-filter backend made Art. 50 covered.
func TestEngineProbes_ContentFilterStatusIsNotWatermarkingEvidence(t *testing.T) {
	e := NewEngine(nil, nil)
	e.AddContentFilterBackend(stubClassifier{})

	probes := e.EngineProbes()

	// The content filter's control is observable; that is correct and stays.
	if _, ok := probes["content_filter_privacy"]; !ok {
		t.Fatalf("content_filter_privacy has no probe; the content filter is a real, observable control")
	}

	// Watermarking is a different control. Nothing in this tree implements it,
	// so nothing may be registered as measuring it.
	if _, ok := probes["content_filter_watermark"]; ok {
		t.Fatalf("content_filter_watermark has a probe, but no watermarking is implemented anywhere in the tree; " +
			"a probe here can only be borrowing another control's status")
	}

	// And the consequence at the requirement level: Art. 50 names exactly this
	// control, so it must measure UNKNOWN and must never count as covered.
	mapper, err := NewEmbeddedComplianceMapper()
	if err != nil {
		t.Fatalf("mapper: %v", err)
	}
	req, ok := mapper.Requirement("eu_ai_act", "art_50")
	if !ok {
		t.Fatalf("art_50 missing from the requirement database")
	}
	if len(req.GuardrailFeatures) != 1 || req.GuardrailFeatures[0] != "content_filter_watermark" {
		t.Fatalf("art_50 no longer names content_filter_watermark alone: %v", req.GuardrailFeatures)
	}

	a := AssessRequirement(context.Background(), "art_50", req, probes)
	if a.Covered {
		t.Fatalf("art_50 reported covered while no watermarking exists: %#v", a)
	}
	if a.Measured != FeatureUnknown {
		t.Fatalf("art_50 measured = %q, want %q: nothing observed the control", a.Measured, FeatureUnknown)
	}
}

// An unmeasured control cannot contradict a declaration of DEFICIENCY.
//
// Art. 50 is declared "gap" and lists the missing watermarking in its own gaps.
// With no probe able to observe it, the database is not over-claiming — so
// reporting an over-declaration here would be a false finding, and a report
// full of false findings is read as noise and then ignored.
func TestAssessRequirement_UnmeasuredDoesNotContradictADeclaredGap(t *testing.T) {
	req := Requirement{
		Title:             "AI-Generated Content",
		ComplianceStatus:  "gap",
		GuardrailFeatures: []string{"content_filter_watermark"},
	}

	a := AssessRequirement(context.Background(), "art_50", req, probeThatReports("some_other_feature"))

	if a.Covered {
		t.Fatalf("a requirement with no observed control reported covered: %#v", a)
	}
	if !a.DeclarationAgrees {
		t.Fatalf("declared %q with nothing measurable must NOT be reported as an over-declaration: %#v",
			req.ComplianceStatus, a)
	}
}

// A declaration of FULL is the over-claim: it asserts a control works, and
// nothing observed it. This is the direction that matters and it is pinned so
// the previous test cannot be satisfied by weakening the check to always-true.
func TestAssessRequirement_UnmeasuredContradictsADeclaredFull(t *testing.T) {
	req := Requirement{
		Title:             "declares a working control nobody can observe",
		ComplianceStatus:  "full",
		GuardrailFeatures: []string{"feature_with_no_probe"},
	}

	a := AssessRequirement(context.Background(), "x", req, probeThatReports("some_other_feature"))

	if a.DeclarationAgrees {
		t.Fatalf("declared full with no observation must be an over-declaration: %#v", a)
	}
}

// A partial declaration against an unmeasured control is likewise not an
// over-claim: "we do part of this" is not contradicted by "we did not look".
func TestAssessRequirement_UnmeasuredDoesNotContradictAPartialDeclaration(t *testing.T) {
	req := Requirement{
		Title:             "declared partial",
		ComplianceStatus:  "partial",
		GuardrailFeatures: []string{"feature_with_no_probe"},
	}

	a := AssessRequirement(context.Background(), "x", req, probeThatReports("some_other_feature"))
	if !a.DeclarationAgrees {
		t.Fatalf("declared partial with nothing measurable must not be an over-declaration: %#v", a)
	}
}

// The measured report for eu_ai_act must not count Art. 50 as covered, and must
// list it among the unmeasured requirements rather than omitting it.
func TestGenerateMeasuredReport_WatermarkGapIsUnmeasuredNotCovered(t *testing.T) {
	mapper, err := NewEmbeddedComplianceMapper()
	if err != nil {
		t.Fatalf("mapper: %v", err)
	}
	reporter := NewComplianceReporter(mapper, nil)

	engine := NewEngine(nil, nil)
	engine.AddContentFilterBackend(stubClassifier{})

	art, err := reporter.GenerateMeasuredReport(context.Background(), "eu_ai_act", engine.EngineProbes())
	if err != nil {
		t.Fatalf("GenerateMeasuredReport: %v", err)
	}

	for _, r := range art.Requirements {
		if r.RequirementID != "art_50" {
			continue
		}
		if r.Covered {
			t.Fatalf("art_50 counted as covered: %#v", r)
		}
		if r.Measured != FeatureUnknown {
			t.Fatalf("art_50 measured = %q, want %q", r.Measured, FeatureUnknown)
		}
		return
	}
	t.Fatalf("art_50 absent from the report; an unmeasured requirement must be listed, not dropped")
}

// stubClassifier is a semantic backend that is present but never consulted by
// these tests; it exists so AddContentFilterBackend produces a non-zero backend
// count, which is what used to make the watermark alias report active.
type stubClassifier struct{}

func (stubClassifier) Name() string { return "stub" }

func (stubClassifier) Classify(context.Context, string) (map[string]float64, error) {
	return map[string]float64{}, nil
}

func (stubClassifier) Available(context.Context) bool { return true }

// Spec 06 Sec. 5.1 states its own acceptance criterion verbatim: "Requirement
// mapping covers all EU AI Act articles". Art. 11 was absent from the database
// although the spec's own mapping table lists it as a known gap, so the
// criterion could not pass. It is now present and HONESTLY declared: no probe
// observes technical documentation, so it measures UNKNOWN and never counts as
// covered.
func TestRequirementDB_CoversEveryArticleTheSpecMaps(t *testing.T) {
	mapper, err := NewEmbeddedComplianceMapper()
	if err != nil {
		t.Fatalf("mapper: %v", err)
	}

	// The articles the spec's own Sec. 2.2 mapping table lists for EU AI Act.
	for _, id := range []string{"art_9", "art_10", "art_11", "art_12", "art_13", "art_14", "art_15", "art_50"} {
		if _, ok := mapper.Requirement("eu_ai_act", id); !ok {
			t.Errorf("%s is mapped in the spec but absent from the requirement database", id)
		}
	}

	// Coverage of a requirement that maps to an unimplemented control must not
	// be claimed merely because the requirement now exists.
	req, ok := mapper.Requirement("eu_ai_act", "art_11")
	if !ok {
		t.Fatalf("art_11 missing")
	}
	if req.ComplianceStatus != "gap" {
		t.Fatalf("art_11 declared %q; no technical-documentation artifact is produced by this repo",
			req.ComplianceStatus)
	}
	if len(req.Gaps) == 0 {
		t.Fatalf("art_11 declares no gap text; a missing control must say what is missing")
	}

	a := AssessRequirement(context.Background(), "art_11", req,
		probeThatReports("some_other_feature"))
	if a.Covered {
		t.Fatalf("art_11 counted as covered while nothing measured technical documentation: %#v", a)
	}
	if a.Measured != FeatureUnknown {
		t.Fatalf("art_11 measured = %q, want %q", a.Measured, FeatureUnknown)
	}
}
