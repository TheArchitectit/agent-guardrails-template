package mcp

// Spec 06 — the compliance tools must report measurements, not declarations.
//
// The defect these tests exist to prevent is not a bug in a helper: it is a
// report whose score restates compliance_requirements.json. A test that only
// asserted "score is between 0 and 100" passed for months while the number
// meant nothing (see the pre-existing compliance_test.go). These tests drive
// the REAL dispatched handlers and assert the properties that make a
// measurement distinguishable from a restatement:
//
//   - an unprobed control is never counted as covered;
//   - a requirement is covered only when every control it names is ACTIVE;
//   - over-declaration (DB says full, measurement disagrees) is surfaced;
//   - an empty probe set is refused, not silently scored zero.

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/audit"
	"github.com/thearchitectit/guardrail-mcp/internal/auth"
	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// callComplianceTool drives the handler the dispatch switch routes to.
//
// It calls the handler directly rather than going through handleToolCall: the
// latter runs the authorization gate first, which needs a principal, and these
// tests are about measurement rather than authorization. Dispatch reachability
// and authz classification are asserted separately by
// TestComplianceToolsAreRegisteredAndClassified and the authz suite.
func callComplianceTool(t *testing.T, s *MCPServer, name string, args map[string]any) map[string]any {
	t.Helper()
	var res *mcp.CallToolResult
	var err error
	switch name {
	case "guardrail_generate_compliance_report":
		res, err = s.handleGenerateComplianceReport(context.Background(), args)
	case "guardrail_check_compliance":
		res, err = s.handleCheckCompliance(context.Background(), args)
	case "guardrail_collect_evidence":
		res, err = s.handleCollectEvidence(context.Background(), args)
	default:
		t.Fatalf("unknown compliance tool %q", name)
	}
	if err != nil {
		t.Fatalf("%s: transport error: %v", name, err)
	}
	if res == nil || len(res.Content) == 0 {
		t.Fatalf("%s: empty result", name)
	}
	tc, ok := res.Content[0].(mcp.TextContent)
	if !ok {
		t.Fatalf("%s: unexpected content type %T", name, res.Content[0])
	}
	return decodeJSON(t, tc.Text)
}

func decodeJSON(t *testing.T, raw string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("result is not JSON: %v\nraw: %s", err, raw)
	}
	return out
}

func newComplianceServer(t *testing.T) *MCPServer {
	t.Helper()
	s := &MCPServer{}
	// A live engine with default config: L1 patterns present, no semantic
	// backend, sandbox configured. This is the shipped posture, so the tests
	// measure what a real deployment would report.
	s.guardrailsEngine = guardrails.NewEngine(nil, nil)
	return s
}

// A probe that never ran must not be able to say "covered". This is the exact
// failure the requirement database produced: every requirement with
// compliance_status=full counted as compliant with no observation at all.
//
// The two audit postures are deliberately distinguished: no logger at all is
// ABSENT, a logger that only writes to the process log is PARTIAL. Neither is
// covered, and a deployment cannot reach "active" by having the struct.
func TestCheckCompliance_UnprobedFeatureIsNotCovered(t *testing.T) {
	t.Run("no audit logger", func(t *testing.T) {
		s := newComplianceServer(t)
		out := callComplianceTool(t, s, "guardrail_check_compliance", map[string]any{
			"framework":      "eu_ai_act",
			"requirement_id": "art_12",
		})

		if covered, _ := out["covered"].(bool); covered {
			t.Fatalf("art_12 reported covered with no audit logger: %#v", out)
		}
		if got, _ := out["measured_status"].(string); got != string(guardrails.FeatureAbsent) {
			t.Fatalf("art_12 measured_status = %q, want %q", got, guardrails.FeatureAbsent)
		}
	})

	t.Run("logger without durable store", func(t *testing.T) {
		s := newComplianceServer(t)
		s.audit = audit.NewLogger(8)
		defer s.audit.Stop()

		out := callComplianceTool(t, s, "guardrail_check_compliance", map[string]any{
			"framework":      "eu_ai_act",
			"requirement_id": "art_12",
		})

		if covered, _ := out["covered"].(bool); covered {
			t.Fatalf("art_12 reported covered from a non-durable logger: %#v", out)
		}
		if got, _ := out["measured_status"].(string); got != string(guardrails.FeaturePartial) {
			t.Fatalf("art_12 measured_status = %q, want %q (process-log only is not durability)",
				got, guardrails.FeaturePartial)
		}
	})
}

// An over-declaration is a finding: the database claiming a stronger state than
// the system exhibits must not be averaged away.
func TestCheckCompliance_OverDeclarationIsReported(t *testing.T) {
	s := newComplianceServer(t)

	// art_12 is declared "full" in the database.
	out := callComplianceTool(t, s, "guardrail_check_compliance", map[string]any{
		"framework":      "eu_ai_act",
		"requirement_id": "art_12",
	})
	if agrees, _ := out["declaration_agrees"].(bool); agrees {
		t.Fatalf("database declares art_12 full but measurement disagrees; declaration_agrees must be false: %#v", out)
	}
	if decl, _ := out["declared_status"].(string); decl != "full" {
		t.Fatalf("declared_status = %q, want full", decl)
	}
}

// An unknown framework is refused with the known set, not scored as zero.
func TestGenerateComplianceReport_UnknownFrameworkRefused(t *testing.T) {
	s := newComplianceServer(t)

	res, err := s.handleToolCall(context.Background(), "guardrail_generate_compliance_report",
		map[string]any{"framework": "not_a_framework"})
	if err != nil {
		t.Fatalf("transport error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("unknown framework must be an error result")
	}
}

// The report must never count an unmeasured requirement as covered, and must
// name the unmeasured ones rather than silently omitting them.
func TestGenerateComplianceReport_UnmeasuredNotCounted(t *testing.T) {
	s := newComplianceServer(t)

	out := callComplianceTool(t, s, "guardrail_generate_compliance_report", map[string]any{
		"framework": "eu_ai_act",
	})

	report, ok := out["report"].(map[string]any)
	if !ok {
		t.Fatalf("report missing: %#v", out)
	}

	methodology, _ := report["methodology"].(string)
	if methodology != guardrails.MeasuredMethodology {
		t.Fatalf("methodology = %q, want %q — a restated score must never be labelled measured",
			methodology, guardrails.MeasuredMethodology)
	}

	reqs, _ := report["requirements"].([]any)
	if len(reqs) == 0 {
		t.Fatalf("report has no requirements")
	}

	var covered, unmeasured int
	for _, r := range reqs {
		m, _ := r.(map[string]any)
		if c, _ := m["covered"].(bool); c {
			covered++
			// A covered requirement must have EVERY named control observed
			// active — prove it rather than trusting the flag.
			obs, _ := m["observed_features"].([]any)
			for _, o := range obs {
				om, _ := o.(map[string]any)
				if st, _ := om["status"].(string); st != string(guardrails.FeatureActive) {
					t.Fatalf("%s counted as covered with a non-active control: %s=%s",
						m["requirement_id"], om["feature"], st)
				}
			}
		}
		if st, _ := m["measured_status"].(string); st == string(guardrails.FeatureUnknown) {
			unmeasured++
		}
	}

	if unmeasured == 0 {
		t.Logf("note: every eu_ai_act requirement had at least one probed feature this run")
	}
	if covered == len(reqs) {
		t.Fatalf("every requirement counted as covered — with no semantic backends and no durable store that cannot be true")
	}
}

// Completeness must reflect observation. The old CollectEvidence returned 1.0
// for any non-empty query set while executing no query.
func TestCollectEvidence_CompletenessReflectsObservation(t *testing.T) {
	s := newComplianceServer(t)

	out := callComplianceTool(t, s, "guardrail_collect_evidence", map[string]any{
		"framework":      "eu_ai_act",
		"requirement_id": "art_15",
	})

	completeness, ok := out["completeness"].(float64)
	if !ok {
		t.Fatalf("completeness missing: %#v", out)
	}
	if completeness >= 1.0 {
		t.Fatalf("completeness = %.2f: art_15 names injection_defense and sandbox_isolation, "+
			"neither of which can be fully active without a semantic backend and an isolation runtime",
			completeness)
	}

	evidence, _ := out["evidence"].([]any)
	if len(evidence) == 0 {
		t.Fatalf("evidence empty: the requirement names controls, so they must be reported")
	}
	for _, e := range evidence {
		em, _ := e.(map[string]any)
		if v, ok := em["verified"].(bool); ok && v {
			t.Fatalf("an unmeasured observation reported verified: %#v", em)
		}
	}
}

// A requirement that names no controls cannot be "compliant": nothing was
// observed, so nothing was demonstrated.
func TestAssessRequirement_NoFeaturesIsNotCompliant(t *testing.T) {
	a := guardrails.AssessRequirement(context.Background(), "x",
		guardrails.Requirement{Title: "no controls", ComplianceStatus: "full"},
		guardrails.ProbeSet{"anything": func(context.Context) (guardrails.FeatureStatus, string) {
			return guardrails.FeatureActive, "ok"
		}})

	if a.Covered {
		t.Fatalf("a requirement naming no controls reported covered")
	}
	if a.DeclarationAgrees {
		t.Fatalf("declared full with no observable control must be an over-declaration")
	}
}

// A requirement whose features have NO registered probe must never be counted
// as covered. This is the exact vacuity class the file exists to remove: the
// old code counted every requirement with compliance_status=full, so a control
// nobody had a probe for inherited the declaration's verdict.
//
// This test exists because a mutation made the unprobed branch report ACTIVE
// and the suite still passed — the proof had a hole exactly where the defect
// used to be.
func TestAssessRequirement_UnprobedFeatureIsUnknownNotCovered(t *testing.T) {
	req := guardrails.Requirement{
		Title:             "declares controls nobody can measure",
		ComplianceStatus:  "full",
		GuardrailFeatures: []string{"feature_with_no_probe", "another_unprobed_feature"},
	}
	// A probe set that exists but does NOT cover this requirement's features.
	probes := guardrails.ProbeSet{
		"some_other_feature": func(context.Context) (guardrails.FeatureStatus, string) {
			return guardrails.FeatureActive, "ok"
		},
	}

	a := guardrails.AssessRequirement(context.Background(), "x", req, probes)

	if a.Covered {
		t.Fatalf("a requirement with no probed controls reported covered: %#v", a)
	}
	if a.Measured != guardrails.FeatureUnknown {
		t.Fatalf("measured = %q, want %q: nothing observed the controls, so the state is unknown",
			a.Measured, guardrails.FeatureUnknown)
	}
	for _, o := range a.Observed {
		if o.Status == guardrails.FeatureActive {
			t.Fatalf("unprobed feature %q reported active: %#v", o.Feature, o)
		}
	}
	if a.DeclarationAgrees {
		t.Fatalf("declared full while measured unknown must be an over-declaration")
	}
}

// The vacuity guard: an empty probe set must be refused, not scored.
func TestGenerateMeasuredReport_EmptyProbesRefused(t *testing.T) {
	mapper, err := guardrails.NewEmbeddedComplianceMapper()
	if err != nil {
		t.Fatalf("mapper: %v", err)
	}
	reporter := guardrails.NewComplianceReporter(mapper, nil)
	if _, err := reporter.GenerateMeasuredReport(context.Background(), "eu_ai_act", guardrails.ProbeSet{}); err == nil {
		t.Fatalf("empty probe set produced a report; it must be refused")
	}
}

// An unknown requirement is refused rather than reported as non-compliant: the
// two answers mean different things to a caller.
func TestCheckMeasured_UnknownRequirementRefused(t *testing.T) {
	mapper, err := guardrails.NewEmbeddedComplianceMapper()
	if err != nil {
		t.Fatalf("mapper: %v", err)
	}
	probes := guardrails.ProbeSet{"x": func(context.Context) (guardrails.FeatureStatus, string) {
		return guardrails.FeatureActive, "ok"
	}}
	if _, err := mapper.CheckMeasured(context.Background(), "eu_ai_act", "no_such_req", probes); err == nil {
		t.Fatalf("unknown requirement accepted")
	}
}

// Every registered compliance tool must be advertised and dispatched. An
// unclassified tool is denied before its handler runs, which is how two of the
// Spec 04/05 tools would have been unreachable had they not been added to the
// table — so classification is asserted here too.
func TestComplianceToolsAreRegisteredAndClassified(t *testing.T) {
	s := &MCPServer{}
	names := map[string]bool{}
	for _, tool := range s.toolList() {
		names[tool.Name] = true
	}
	want := []string{
		"guardrail_generate_compliance_report",
		"guardrail_check_compliance",
		"guardrail_collect_evidence",
	}
	for _, name := range want {
		if !names[name] {
			t.Errorf("tool %q is not advertised by toolList()", name)
		}
		kind, ok := auth.MCPToolAction(name)
		if !ok {
			t.Errorf("tool %q has no authz classification, so it is denied before dispatch", name)
			continue
		}
		if kind != auth.ActionRead {
			t.Errorf("tool %q classified %q, want read", name, kind)
		}
	}
}
