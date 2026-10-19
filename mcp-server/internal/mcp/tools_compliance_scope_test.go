package mcp

// Spec 06 Sec. 2.3 — the compliance configuration scope must reach the REQUEST
// PATH, not merely parse.
//
// The loader tests in internal/guardrails prove the file is read strictly. These
// tests exist for the property a parse-only fix would miss: an operator who
// scopes the audit to one framework must not receive a report for another. They
// drive the REAL dispatched handlers through the same entry point a client uses,
// with a config produced by the REAL loader.

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

func loadComplianceConfigForTest(t *testing.T, body string) *guardrails.ComplianceConfig {
	t.Helper()
	p := filepath.Join(t.TempDir(), "compliance.yml")
	if err := os.WriteFile(p, []byte(body), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	cfg, err := guardrails.LoadComplianceConfigFromYAML(p)
	if err != nil {
		t.Fatalf("fixture did not load: %v", err)
	}
	return cfg
}

// An operator who activates one framework must be refused the other BY NAME —
// not scored as zero coverage, which would read as a measurement.
func TestComplianceScope_UnactivatedFrameworkIsRefusedByEveryTool(t *testing.T) {
	cfg := loadComplianceConfigForTest(t, `compliance_reports:
  nist_rmf:
    enabled: true
    output_format: "json"
    evidence_sources: ["audit_logs"]
`)
	s := newComplianceServer(t)
	s.SetComplianceConfig(cfg)

	for _, name := range []string{
		"guardrail_generate_compliance_report",
		"guardrail_check_compliance",
		"guardrail_collect_evidence",
	} {
		t.Run(name, func(t *testing.T) {
			out := callComplianceTool(t, s, name, map[string]any{
				"framework":      "eu_ai_act",
				"requirement_id": "art_12",
			})
			msg, _ := out["error"].(string)
			if !strings.Contains(msg, "not activated") {
				t.Fatalf("%s accepted an unactivated framework; error = %q, out = %#v", name, msg, out)
			}
			// The refusal names the active set so the caller is not left guessing.
			active, _ := out["active_frameworks"].([]any)
			if len(active) != 1 || active[0] != "nist_rmf" {
				t.Fatalf("%s refusal does not name the active set: %#v", name, out)
			}
			// A refusal must never look like a measurement.
			if score, ok := out["overall_score"]; ok {
				t.Fatalf("%s returned a score for a framework it refused: %v", name, score)
			}
		})
	}
}

// The activated framework keeps working: scoping is a restriction, not a
// disablement, and the tool the spec defines must still produce a measured
// report for the scope the operator chose.
func TestComplianceScope_ActivatedFrameworkStillReports(t *testing.T) {
	cfg := loadComplianceConfigForTest(t, `compliance_reports:
  nist_rmf:
    enabled: true
    output_format: "json"
    evidence_sources: ["audit_logs"]
`)
	s := newComplianceServer(t)
	s.SetComplianceConfig(cfg)

	out := callComplianceTool(t, s, "guardrail_generate_compliance_report", map[string]any{
		"framework": "nist_rmf",
	})
	if msg, ok := out["error"]; ok {
		t.Fatalf("the activated framework was refused: %v", msg)
	}
	report, ok := out["report"].(map[string]any)
	if !ok {
		t.Fatalf("no report in result: %#v", out)
	}
	if got, _ := report["methodology"].(string); got != guardrails.MeasuredMethodology {
		t.Fatalf("report methodology = %q, want %q — a scoped report must still be MEASURED",
			got, guardrails.MeasuredMethodology)
	}
	if got, _ := report["framework"].(string); got != "nist_rmf" {
		t.Fatalf("report framework = %q, want nist_rmf", got)
	}
}

// With no file configured the server holds a nil config and behaves exactly as
// it did before this change: every framework the database carries is reportable.
// This is the regression guard for the zero-configuration deployment.
func TestComplianceScope_NoConfigStillAllowsEveryFramework(t *testing.T) {
	s := newComplianceServer(t)

	for _, fw := range []string{"eu_ai_act", "nist_rmf", "iso_42001"} {
		out := callComplianceTool(t, s, "guardrail_generate_compliance_report", map[string]any{
			"framework": fw,
		})
		if msg, ok := out["error"]; ok {
			t.Fatalf("framework %q was refused with no config file configured: %v", fw, msg)
		}
	}
}
