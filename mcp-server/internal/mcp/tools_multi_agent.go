package mcp

// Spec 04 (guardrail-gaps-2026/04-multi-agent-safety-policies) — the three
// tools the spec defines, wired to the chain engine.
//
// Before this file the entire multi-agent domain model was unreachable:
// NewSafetyChain, ResolveConstraints and ResolveConflicts had test-only
// callers, and none of guardrail_validate_agent_output,
// guardrail_check_agent_constraints or guardrail_resolve_conflicts existed. A
// safety chain that no request can invoke validates nothing, so the model is
// only worth what this file exposes of it.

import (
	"context"
	"encoding/json"
	"sort"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// maxConflictOutputs bounds the parallel-output fan-in. Conflict resolution is
// quadratic in the number of agents for the pairwise strategies, and an
// unbounded list is a cheap denial-of-service against the server.
const maxConflictOutputs = 32

func (s *MCPServer) multiAgentTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "guardrail_validate_agent_output",
			Description: "Run an agent's output through a named safety chain before another agent or a user consumes it, and " +
				"return per-step results, the decision, and any violations",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"agent_id": map[string]any{
						"type":        "string",
						"description": "Identifier of the agent whose output is being validated",
					},
					"output": map[string]any{
						"type":        "string",
						"description": "The output to validate",
					},
					"chain_id": map[string]any{
						"type":        "string",
						"description": "Safety chain to run; an unknown id is refused rather than replaced by a default",
					},
					"context": map[string]any{
						"type":        "string",
						"description": "The scope declared for this task; required by the Four Laws scope check",
					},
					"parent_agent_id": map[string]any{
						"type":        "string",
						"description": "Agent that delegated to this one, recorded on the audit entry",
					},
				},
				Required: []string{"agent_id", "output", "chain_id"},
			},
		},
		{
			Name: "guardrail_check_agent_constraints",
			Description: "Check whether an agent's declared constraints satisfy a named policy: every policy constraint must " +
				"survive at equal or higher severity",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"agent_id": map[string]any{
						"type":        "string",
						"description": "Identifier of the agent to check",
					},
					"policy_id": map[string]any{
						"type":        "string",
						"description": "Policy to check against; an unknown id is refused",
					},
					"constraints": map[string]any{
						"type":        "array",
						"description": "Constraints the agent declares for itself; at least one is required",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"id":       map[string]any{"type": "string", "description": "Constraint id"},
								"severity": map[string]any{"type": "string", "description": "low, medium, high or critical"},
							},
							"required": []string{"id", "severity"},
						},
					},
				},
				Required: []string{"agent_id", "policy_id", "constraints"},
			},
		},
		{
			Name: "guardrail_resolve_conflicts",
			Description: "Resolve disagreements between parallel agents' outputs under an explicit strategy and report which " +
				"actions conflicted",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"outputs": map[string]any{
						"type":        "array",
						"description": "One entry per agent, at most 32",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"agent_id": map[string]any{"type": "string", "description": "Agent identifier"},
								"output":   map[string]any{"type": "string", "description": "Agent output"},
								"priority": map[string]any{"type": "integer", "description": "Higher wins under the priority strategy"},
								"actions": map[string]any{
									"type":        "array",
									"items":       map[string]any{"type": "string"},
									"description": "Actions the agent proposes; used by intersection, union and conflict reporting",
								},
							},
							"required": []string{"agent_id"},
						},
					},
					"strategy": map[string]any{
						"type":        "string",
						"description": "priority, intersection, union or escalate",
						"enum":        []string{"priority", "intersection", "union", "escalate"},
					},
				},
				Required: []string{"outputs", "strategy"},
			},
		},
	}
}

// multiAgentUnavailable is the fail-closed answer when no registry is
// configured: the chain did not run, so nothing may be reported as passed.
func multiAgentUnavailable(tool string) *mcp.CallToolResult {
	payload, _ := json.Marshal(map[string]any{
		"error": "multi-agent safety chains are not configured",
		"tool":  tool,
	})
	return errorResult(string(payload))
}

// handleValidateAgentOutput implements guardrail_validate_agent_output.
func (s *MCPServer) handleValidateAgentOutput(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	agentID, _ := args["agent_id"].(string)
	output, _ := args["output"].(string)
	chainID, _ := args["chain_id"].(string)
	scopeContext, _ := args["context"].(string)
	parentID, _ := args["parent_agent_id"].(string)

	switch {
	case strings.TrimSpace(agentID) == "":
		return errorResult(`{"passed":false,"error":"agent_id is required"}`), nil
	case output == "":
		return errorResult(`{"passed":false,"error":"output is required"}`), nil
	case strings.TrimSpace(chainID) == "":
		// A missing chain must not fall through to "no steps", which executes
		// nothing and reports passed=true.
		return errorResult(`{"passed":false,"error":"chain_id is required"}`), nil
	case s.multiAgent == nil:
		return multiAgentUnavailable("guardrail_validate_agent_output"), nil
	}

	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_validate_agent_output", chainID)

	result, err := s.multiAgent.ValidateOutput(ctx, agentID, output, chainID, scopeContext, parentID)
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"passed": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	// A blocked chain is a verdict, not a transport error, but it is reported
	// as isError so a caller that only inspects IsError cannot read a block as
	// a pass. The full result is still present for a caller that wants reasons.
	return jsonToolResult(result, !result.Passed)
}

// handleCheckAgentConstraints implements guardrail_check_agent_constraints.
func (s *MCPServer) handleCheckAgentConstraints(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	agentID, _ := args["agent_id"].(string)
	policyID, _ := args["policy_id"].(string)

	switch {
	case strings.TrimSpace(agentID) == "":
		return errorResult(`{"compliant":false,"error":"agent_id is required"}`), nil
	case strings.TrimSpace(policyID) == "":
		return errorResult(`{"compliant":false,"error":"policy_id is required"}`), nil
	case s.multiAgent == nil:
		return multiAgentUnavailable("guardrail_check_agent_constraints"), nil
	}

	own, err := decodeConstraints(args["constraints"])
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"compliant": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}
	if len(own) == 0 {
		// No declared constraints is not "everything inherited and fine" when
		// the agent had to declare its own posture; it is a refusal.
		return errorResult(`{"compliant":false,"error":"constraints must contain at least one entry"}`), nil
	}

	result, err := s.multiAgent.CheckConstraints(agentID, policyID, own)
	if err != nil {
		payload, _ := json.Marshal(map[string]any{"compliant": false, "error": err.Error()})
		return errorResult(string(payload)), nil
	}

	return jsonToolResult(result, !result.Compliant)
}

// decodeConstraints converts the MCP-decoded array into []AgentConstraint,
// refusing entries that carry no usable severity: an empty severity ranks below
// every requirement and would silently satisfy the merge rule.
func decodeConstraints(raw any) ([]guardrails.AgentConstraint, error) {
	if raw == nil {
		return nil, errText("constraints is required")
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, errText("constraints must be an array")
	}

	seen := make(map[string]bool, len(entries))
	out := make([]guardrails.AgentConstraint, 0, len(entries))
	for i, entry := range entries {
		obj, ok := entry.(map[string]any)
		if !ok {
			return nil, errText("each constraint must be an object with id and severity")
		}
		id, _ := obj["id"].(string)
		severity, _ := obj["severity"].(string)
		if strings.TrimSpace(id) == "" {
			return nil, errText("constraint " + itoa(i) + " has no id")
		}
		if guardrails.SeverityRank(severity) == 0 {
			return nil, errText("constraint " + itoa(i) + " has an unknown severity " + quote(severity))
		}
		if seen[id] {
			return nil, errText("duplicate constraint id: " + id)
		}
		seen[id] = true
		out = append(out, guardrails.AgentConstraint{ID: id, Severity: strings.ToLower(strings.TrimSpace(severity))})
	}
	return out, nil
}

// handleResolveConflicts implements guardrail_resolve_conflicts.
func (s *MCPServer) handleResolveConflicts(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	strategyName, _ := args["strategy"].(string)
	strategy, ok := resolveConflictStrategy(strategyName)
	if !ok {
		return errorResult(`{"resolved":false,"error":"strategy must be one of priority, intersection, union, escalate"}`), nil
	}

	raw, ok := args["outputs"].([]any)
	if !ok || len(raw) == 0 {
		return errorResult(`{"resolved":false,"error":"outputs must contain at least one entry"}`), nil
	}
	if len(raw) > maxConflictOutputs {
		return errorResult(`{"resolved":false,"error":"outputs exceeds the 32 entry limit"}`), nil
	}

	outputs := make([]guardrails.AgentOutput, 0, len(raw))
	seen := make(map[string]bool, len(raw))
	for i, entry := range raw {
		obj, ok := entry.(map[string]any)
		if !ok {
			return errorResult(`{"resolved":false,"error":"each output must be an object with agent_id"}`), nil
		}
		agentID, _ := obj["agent_id"].(string)
		if strings.TrimSpace(agentID) == "" {
			return errorResult(`{"resolved":false,"error":"output ` + itoa(i) + ` has no agent_id"}`), nil
		}
		if seen[agentID] {
			return errorResult(`{"resolved":false,"error":"duplicate agent_id: ` + agentID + `"}`), nil
		}
		seen[agentID] = true

		text, _ := obj["output"].(string)
		actions, err := decodeActions(obj["actions"])
		if err != nil {
			payload, _ := json.Marshal(map[string]any{"resolved": false, "error": err.Error()})
			return errorResult(string(payload)), nil
		}
		outputs = append(outputs, guardrails.AgentOutput{
			AgentID:  agentID,
			Output:   text,
			Priority: intOf(obj["priority"]),
			Actions:  actions,
		})
	}

	result := guardrails.ResolveConflicts(outputs, strategy)

	// A strategy that could not reach a decision (escalate, or a divergent
	// union) is reported as isError so a caller cannot treat "we did not
	// resolve this" as a resolution. resolved=false is in the payload too.
	return jsonToolResult(result, !result.Resolved)
}

func resolveConflictStrategy(name string) (guardrails.ConflictStrategy, bool) {
	switch guardrails.ConflictStrategy(strings.ToLower(strings.TrimSpace(name))) {
	case guardrails.ConflictPriority:
		return guardrails.ConflictPriority, true
	case guardrails.ConflictIntersection:
		return guardrails.ConflictIntersection, true
	case guardrails.ConflictUnion:
		return guardrails.ConflictUnion, true
	case guardrails.ConflictEscalate:
		return guardrails.ConflictEscalate, true
	}
	return "", false
}

func decodeActions(raw any) ([]string, error) {
	if raw == nil {
		return nil, nil
	}
	entries, ok := raw.([]any)
	if !ok {
		return nil, errText("actions must be an array of strings")
	}
	out := make([]string, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for _, e := range entries {
		a, ok := e.(string)
		if !ok {
			return nil, errText("actions must be an array of strings")
		}
		a = strings.TrimSpace(a)
		if a == "" || seen[a] {
			// An empty or repeated action changes what "every agent agreed"
			// means under the intersection and union strategies.
			continue
		}
		seen[a] = true
		out = append(out, a)
	}
	sort.Strings(out)
	return out, nil
}

func intOf(v any) int {
	switch n := v.(type) {
	case int:
		return n
	case int64:
		return int(n)
	case float64:
		return int(n)
	}
	return 0
}

// errText is a tiny error type so decode helpers can return a message without
// importing fmt into the mcp package (it already does, but the intent here is
// to keep these messages call-site readable).
type errText string

func (e errText) Error() string { return string(e) }

func quote(s string) string { return `"` + s + `"` }
