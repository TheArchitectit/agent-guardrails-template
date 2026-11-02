package guardrails

import (
	"context"
	"strings"
	"testing"
)

func TestRunSemanticConformance_DefaultCorpusPasses(t *testing.T) {
	ref := NewReferenceSemanticClassifier("reference-s1-s15@1.0.0")
	cf := NewContentFilter([]SemanticClassifier{ref}, nil, WithCacheTTL(0))

	report, err := RunSemanticConformance(
		context.Background(),
		cf,
		DefaultSemanticConformanceCorpus(),
		DefaultSemanticConformancePolicy(),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !report.Passed {
		t.Fatalf("expected conformance report to pass, failures: %v", report.Failures)
	}
	if report.ClassifierVersion != "reference-s1-s15@1.0.0" {
		t.Fatalf("unexpected classifier version: %q", report.ClassifierVersion)
	}
	if !strings.HasPrefix(report.CorpusDigest, "sha256:") {
		t.Fatalf("expected sha256 corpus digest, got %q", report.CorpusDigest)
	}
	if len(report.Categories) != 15 {
		t.Fatalf("expected 15 categories in report, got %d", len(report.Categories))
	}
	if report.OverallFNRate != 0 || report.OverallFPRate != 0 {
		t.Fatalf("expected 0 FP/FN on reference corpus, got FP=%.4f FN=%.4f", report.OverallFPRate, report.OverallFNRate)
	}
}

func TestSemanticDriftMonitor_VersionBumpInvalidatesStaleClaim(t *testing.T) {
	v1 := NewReferenceSemanticClassifier("llama-guard-3@1.0.0")
	cf := NewContentFilter([]SemanticClassifier{v1}, nil)

	monitor := NewSemanticDriftMonitor(cf, DefaultSemanticConformancePolicy())
	corpus := DefaultSemanticConformanceCorpus()

	if _, err := monitor.Certify(context.Background(), corpus); err != nil {
		t.Fatalf("initial certification failed: %v", err)
	}

	claim := monitor.VerifyClaim(context.Background())
	if !claim.Valid || claim.Stale {
		t.Fatalf("expected valid non-stale claim for v1, got %+v", claim)
	}

	// Simulate a classifier/model version bump without re-running conformance.
	v1.version = "llama-guard-3@2.0.0"

	staleClaim := monitor.VerifyClaim(context.Background())
	if staleClaim.Valid || !staleClaim.Stale {
		t.Fatalf("expected version bump to invalidate conformance claim as stale, got %+v", staleClaim)
	}
	if !strings.Contains(staleClaim.InvalidatedReason, "llama-guard-3@2.0.0") {
		t.Fatalf("expected invalidated reason to mention new version, got %q", staleClaim.InvalidatedReason)
	}

	events := monitor.LedgerEvents()
	if len(events) < 2 || events[len(events)-1].EventType != "version_mismatch" {
		t.Fatalf("expected version_mismatch event in ledger, got %+v", events)
	}
}

func TestSemanticDriftMonitor_DetectsAccuracyDriftAndEmitsLedgerEvent(t *testing.T) {
	ref := NewReferenceSemanticClassifier("reference-s1-s15@1.0.0")
	cf := NewContentFilter([]SemanticClassifier{ref}, nil, WithCacheTTL(0))

	monitor := NewSemanticDriftMonitor(cf, DefaultSemanticConformancePolicy())
	corpus := DefaultSemanticConformanceCorpus()

	if _, err := monitor.Certify(context.Background(), corpus); err != nil {
		t.Fatalf("initial certification failed: %v", err)
	}

	// Inject drift on S7 (Privacy) so positive samples drop below threshold.
	ref.SetCategoryScoreOverride("S7", 0.15)

	report, drifted, err := monitor.CheckDrift(context.Background(), corpus)
	if err != nil {
		t.Fatalf("CheckDrift error: %v", err)
	}
	if !drifted {
		t.Fatal("expected CheckDrift to report drift=true when S7 misses positives")
	}
	if report.Passed {
		t.Fatal("expected drifted conformance report to have Passed=false")
	}

	events := monitor.LedgerEvents()
	last := events[len(events)-1]
	if last.EventType != "accuracy_drift" {
		t.Fatalf("expected accuracy_drift ledger event, got %+v", last)
	}
	foundS7 := false
	for _, c := range last.DriftedCategories {
		if c == "S7" {
			foundS7 = true
		}
	}
	if !foundS7 {
		t.Fatalf("expected S7 in drifted categories, got %v", last.DriftedCategories)
	}
}

func TestRunSemanticConformance_RefusesVacuousFilter(t *testing.T) {
	cf := NewContentFilter(nil, nil)
	if _, err := RunSemanticConformance(context.Background(), cf, DefaultSemanticConformanceCorpus(), DefaultSemanticConformancePolicy()); err == nil {
		t.Fatal("expected error when ContentFilter has zero backends")
	}
}
