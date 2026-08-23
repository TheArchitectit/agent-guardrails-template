package guardrails

import (
	"context"
	"log/slog"
	"testing"
)

func TestComplianceMapper_CheckRequirement(t *testing.T) {
	// Use the actual JSON file for testing
	mapper, err := NewComplianceMapper("compliance_requirements.json")
	if err != nil {
		t.Fatalf("failed to create mapper: %v", err)
	}

	tests := []struct {
		name       string
		framework  string
		reqID      string
		wantCompl  bool
		wantGaps   int
	}{
		{"Full compliance", "eu_ai_act", "art_12", true, 0},
		{"Partial compliance", "eu_ai_act", "art_9", false, 0},
		{"Gap compliance", "eu_ai_act", "art_50", false, 1},
		{"Invalid framework", "unknown", "art_12", false, 1},
		{"Invalid req", "eu_ai_act", "unknown", false, 1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			compliant, _, gaps := mapper.CheckRequirement(tc.framework, tc.reqID)
			if compliant != tc.wantCompl {
				t.Errorf("compliant = %v, want %v", compliant, tc.wantCompl)
			}
			if len(gaps) != tc.wantGaps {
				t.Errorf("gaps = %d, want %d", len(gaps), tc.wantGaps)
			}
		})
	}
}

func TestComplianceReporter_GenerateReport(t *testing.T) {
	mapper, err := NewComplianceMapper("compliance_requirements.json")
	if err != nil {
		t.Fatalf("NewComplianceMapper failed: %v", err)
	}
	reporter := NewComplianceReporter(mapper, slog.Default())

	ctx := context.Background()
	report, err := reporter.GenerateReport(ctx, "eu_ai_act", FormatJSON)
	if err != nil {
		t.Fatalf("GenerateReport failed: %v", err)
	}

	if report.Framework != "eu_ai_act" {
		t.Errorf("expected framework eu_ai_act, got %s", report.Framework)
	}

	if report.OverallScore <= 0 || report.OverallScore > 100 {
		t.Errorf("invalid overall score: %.2f", report.OverallScore)
	}

	if len(report.Sections) == 0 {
		t.Error("report should have at least one section")
	}
}

func TestCalculateComplianceScore(t *testing.T) {
	reqs := map[string]Requirement{
		"req1": {Title: "R1"},
		"req2": {Title: "R2"},
		"req3": {Title: "R3"},
		"req4": {Title: "R4"},
	}

	tests := []struct {
		name     string
		evidence []Evidence
		want     float64
	}{
		{"No evidence", []Evidence{}, 0.0},
		{"Partial evidence", []Evidence{
			{RequirementID: "req1"},
			{RequirementID: "req2"},
		}, 50.0},
		{"Full evidence", []Evidence{
			{RequirementID: "req1"},
			{RequirementID: "req2"},
			{RequirementID: "req3"},
			{RequirementID: "req4"},
		}, 100.0},
		{"Over-evidence", []Evidence{
			{RequirementID: "req1"},
			{RequirementID: "req1"},
			{RequirementID: "req2"},
		}, 50.0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			score := CalculateComplianceScore(reqs, tc.evidence)
			if score != tc.want {
				t.Errorf("score = %.2f, want %.2f", score, tc.want)
			}
		})
	}
}

func TestEvidenceCollector_CollectEvidence(t *testing.T) {
	collector := NewEvidenceCollector(slog.Default())
	ctx := context.Background()

	queries := []string{"SELECT * FROM logs WHERE x=1", "SELECT * FROM events WHERE y=2"}
	evidence, completeness := collector.CollectEvidence(ctx, "eu_ai_act", "art_12", queries)

	if len(evidence) != len(queries) {
		t.Errorf("expected %d evidence items, got %d", len(queries), len(evidence))
	}

	if completeness != 1.0 {
		t.Errorf("expected completeness 1.0, got %.2f", completeness)
	}

	t.Run("EmptyQueries", func(t *testing.T) {
		ev, comp := collector.CollectEvidence(ctx, "eu_ai_act", "art_12", []string{})
		if len(ev) != 0 {
			t.Errorf("expected 0 evidence, got %d", len(ev))
		}
		if comp != 0.0 {
			t.Errorf("expected completeness 0.0, got %.2f", comp)
		}
	})
}
