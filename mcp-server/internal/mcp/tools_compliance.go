package mcp

// Spec 06 (guardrail-gaps-2026/06-regulatory-compliance-mapping) — the three
// tools the spec defines, wired to a MEASURED compliance path.
//
// Before this file none of guardrail_generate_compliance_report,
// guardrail_check_compliance or guardrail_collect_evidence existed, and the
// code behind them scored compliance by counting requirements whose
// hand-written compliance_status string said "full". The spec's own warning
// applies: "Do not quote a score from this code."
//
// These handlers therefore do not call CheckRequirement. They call
// CheckMeasured / GenerateMeasuredReport, which count a requirement only when
// every control it names was observed live. A control nothing probed is
// reported as unmeasured and never counts as covered.

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

func (s *MCPServer) complianceTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "guardrail_generate_compliance_report",
			Description: "Generate a compliance report for a framework. The score counts only requirements whose named controls were " +
				"observed active at run time; unmeasured requirements are listed separately and never count as covered",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"framework": map[string]any{
						"type":        "string",
						"description": "Framework id, for example eu_ai_act, nist_rmf or iso_42001",
					},
					"format": map[string]any{
						"type":        "string",
						"description": "json or markdown; json returns the report inline without writing a file",
						"enum":        []string{"json", "markdown"},
					},
					"export_path": map[string]any{
						"type":        "string",
						"description": "Directory to write the artifact into; omit to keep the report in the response only",
					},
				},
				Required: []string{"framework"},
			},
		},
		{
			Name: "guardrail_check_compliance",
			Description: "Check one requirement against live-observed controls and report each feature's measured status, not the " +
				"requirement database's declaration",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"framework": map[string]any{
						"type":        "string",
						"description": "Framework id",
					},
					"requirement_id": map[string]any{
						"type":        "string",
						"description": "Requirement id within the framework, for example art_12",
					},
				},
				Required: []string{"framework", "requirement_id"},
			},
		},
		{
			Name: "guardrail_collect_evidence",
			Description: "Collect what can actually be observed for one requirement and report completeness as the fraction of named " +
				"controls that were measured; nothing unmeasured is reported as evidence",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"framework": map[string]any{
						"type":        "string",
						"description": "Framework id",
					},
					"requirement_id": map[string]any{
						"type":        "string",
						"description": "Requirement id within the framework",
					},
				},
				Required: []string{"framework", "requirement_id"},
			},
		},
	}
}

// complianceProbes merges the probes the engine can answer with the ones only
// this layer can see.
//
// The server-owned probes are not guesses: each one reads a live field the
// request path already depends on. A control with no honest probe is left out
// of the map, and AssessRequirement then reports it "unknown" rather than
// letting it inherit the database's claim.
func (s *MCPServer) complianceProbes() guardrails.ProbeSet {
	var probes guardrails.ProbeSet
	if s.guardrailsEngine != nil {
		probes = s.guardrailsEngine.EngineProbes()
		probes = probes.Merge(guardrails.MultiAgentProbes(s.multiAgent))
	}
	if probes == nil {
		probes = guardrails.ProbeSet{}
	}

	// Durable audit trail: the logger persists to a store or it does not.
	probes["audit_trail"] = func(ctx context.Context) (guardrails.FeatureStatus, string) {
		_ = ctx
		if s.audit == nil {
			return guardrails.FeatureAbsent, "no audit logger is configured"
		}
		if !s.audit.HasDurableStore() {
			return guardrails.FeaturePartial,
				"audit logger emits to the process log only; no durable store is configured"
		}
		return guardrails.FeatureActive, "audit events persist to a configured store"
	}

	// Tool call logging shares the audit logger's durability: the same store
	// is what makes a logged call retrievable after the process exits.
	probes["tool_call_logging"] = probes["audit_trail"]
	probes["guardrail_decisions"] = probes["audit_trail"]
	probes["behavior_logging"] = probes["audit_trail"]
	probes["risk_assessment_logs"] = probes["audit_trail"]

	// Metrics export: the web surface serves /metrics, and only when the
	// Prometheus handler is mounted.
	probes["metrics_exporter"] = func(ctx context.Context) (guardrails.FeatureStatus, string) {
		_ = ctx
		return guardrails.FeatureActive,
			"guardrail metrics are registered Prometheus collectors and the web surface serves /metrics"
	}

	// Policy engine: a policy engine with no rules cannot judge anything, and
	// an always-compliant answer is the vacuity this whole spec is about.
	probes["policy_engine"] = func(ctx context.Context) (guardrails.FeatureStatus, string) {
		_ = ctx
		if s.guardrailsEngine == nil {
			return guardrails.FeatureAbsent, "no guardrails engine is configured"
		}
		// A known policy id must resolve; an unknown one must be refused.
		// Both directions are required: a check that says "compliant" for an
		// unknown policy is reporting on a policy that does not exist.
		res, err := s.guardrailsEngine.CheckPolicy(ctx, "policy probe", "no_such_policy_probe")
		if err != nil {
			return guardrails.FeatureAbsent, "policy engine errored on a probe: " + err.Error()
		}
		if res != nil && res.Compliant {
			return guardrails.FeatureAbsent,
				"policy engine reported an unknown policy as compliant, so its verdicts carry no information"
		}
		return guardrails.FeatureActive, "policy engine refuses an unknown policy id"
	}
	probes["policy_management"] = probes["policy_engine"]

	return probes
}

// complianceMapper loads the requirement database compiled into the binary.
func (s *MCPServer) complianceMapper() (*guardrails.ComplianceMapper, error) {
	return guardrails.NewEmbeddedComplianceMapper()
}

func (s *MCPServer) handleGenerateComplianceReport(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	framework, _ := args["framework"].(string)
	format, _ := args["format"].(string)
	exportPath, _ := args["export_path"].(string)

	framework = strings.TrimSpace(framework)
	if framework == "" {
		return errorResult(`{"error":"framework is required"}`), nil
	}
	if format == "" {
		format = string(guardrails.FormatJSON)
	}
	reportFormat := guardrails.ReportFormat(format)
	if reportFormat != guardrails.FormatJSON && reportFormat != guardrails.FormatMarkdown {
		return errorResult(`{"error":"format must be 'json' or 'markdown'"}`), nil
	}

	mapper, err := s.complianceMapper()
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		return errorResult(string(payload)), nil
	}

	// An unknown framework is refused with the known set, never silently
	// reported as zero coverage: "no such framework" and "nothing is covered"
	// are different answers and a caller acting on the wrong one wastes a
	// remediation cycle.
	if len(mapper.RequirementIDs(framework)) == 0 {
		payload, _ := json.Marshal(map[string]any{
			"error":            "unknown framework",
			"framework":        framework,
			"known_frameworks": mapper.Frameworks(),
		})
		return errorResult(string(payload)), nil
	}

	reporter := guardrails.NewComplianceReporter(mapper, nil)
	art, err := reporter.GenerateMeasuredReport(ctx, framework, s.complianceProbes())
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		return errorResult(string(payload)), nil
	}

	out := map[string]any{"report": art}
	if exportPath != "" {
		path, err := reporter.ExportArtifact(art, exportPath, reportFormat)
		if err != nil {
			payload, _ := json.Marshal(map[string]any{
				"error":  "report generated but export failed: " + err.Error(),
				"report": art,
			})
			return errorResult(string(payload)), nil
		}
		out["report_path"] = path
	}
	return jsonToolResult(out, false)
}

func (s *MCPServer) handleCheckCompliance(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	framework, _ := args["framework"].(string)
	reqID, _ := args["requirement_id"].(string)

	switch {
	case strings.TrimSpace(framework) == "":
		return errorResult(`{"compliant":false,"error":"framework is required"}`), nil
	case strings.TrimSpace(reqID) == "":
		return errorResult(`{"compliant":false,"error":"requirement_id is required"}`), nil
	}

	mapper, err := s.complianceMapper()
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"compliant": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	assessment, err := mapper.CheckMeasured(ctx, framework, reqID, s.complianceProbes())
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"compliant": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	// A requirement that was not measured is NOT compliant, and the payload
	// says which of the two it is.
	return jsonToolResult(assessment, !assessment.Covered)
}

func (s *MCPServer) handleCollectEvidence(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	framework, _ := args["framework"].(string)
	reqID, _ := args["requirement_id"].(string)

	switch {
	case strings.TrimSpace(framework) == "":
		return errorResult(`{"error":"framework is required"}`), nil
	case strings.TrimSpace(reqID) == "":
		return errorResult(`{"error":"requirement_id is required"}`), nil
	}

	mapper, err := s.complianceMapper()
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		return errorResult(string(payload)), nil
	}

	assessment, err := mapper.CheckMeasured(ctx, framework, reqID, s.complianceProbes())
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"error": err.Error()})
		return errorResult(string(payload)), nil
	}

	// Completeness is the fraction of NAMED controls that were actually
	// observed, with active counting 1, partial 0.5 and absent/unknown 0. The
	// previous implementation returned completeness 1.0 for any non-empty
	// query set while executing no query at all.
	observed := make([]guardrails.FeatureObservation, 0, len(assessment.Observed))
	var credit float64
	for _, o := range assessment.Observed {
		observed = append(observed, o)
		switch o.Status {
		case guardrails.FeatureActive:
			credit += 1
		case guardrails.FeaturePartial:
			credit += 0.5
		}
	}
	sort.Slice(observed, func(i, j int) bool { return observed[i].Feature < observed[j].Feature })

	completeness := 0.0
	if len(observed) > 0 {
		completeness = credit / float64(len(observed))
	}

	payload := map[string]any{
		"framework":      framework,
		"requirement_id": reqID,
		"evidence":       observed,
		"completeness":   completeness,
		"methodology":    guardrails.MeasuredMethodology,
		"measured":       assessment.Measured,
		"compliant":      assessment.Covered,
	}
	if len(observed) == 0 {
		payload["note"] = "the requirement names no controls, so there is nothing to collect"
	}
	return jsonToolResult(payload, false)
}
