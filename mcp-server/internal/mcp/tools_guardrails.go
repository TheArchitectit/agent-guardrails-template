package mcp

import (
	"context"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// guardrailsToolList returns the semantic content filtering tools defined by
// Spec 02 §3.1: guardrail_classify_content and guardrail_check_policy.
func (s *MCPServer) guardrailsToolList() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "guardrail_classify_content",
			Description: "Classify text against the safety taxonomy (S1-S15) and return per-category scores and actions",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The text to classify",
					},
					"direction": map[string]any{
						"type":        "string",
						"description": "Whether the text is entering ('input') or leaving ('output') the agent",
						"enum":        []string{"input", "output"},
					},
					"context": map[string]any{
						"type":        "string",
						"description": "Optional context for classification (e.g. source tool or channel)",
					},
				},
				Required: []string{"text"},
			},
		},
		{
			Name:        "guardrail_check_policy",
			Description: "Check if text complies with a specific named content policy and return violations",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The text to check against the policy",
					},
					"policy_id": map[string]any{
						"type":        "string",
						"description": "Identifier of the policy to check (e.g. 'coding-safety')",
					},
				},
				Required: []string{"text", "policy_id"},
			},
		},
	}
}

// handleClassifyContent implements the guardrail_classify_content tool:
// it classifies text against the S1-S15 safety taxonomy and returns the
// per-category scores and actions.
func (s *MCPServer) handleClassifyContent(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	text, _ := args["text"].(string)
	if text == "" {
		return errorResult(`{"safe":true,"error":"text is required"}`), nil
	}

	if s.guardrailsEngine == nil {
		return errorResult(`{"safe":true,"error":"guardrails engine not configured"}`), nil
	}

	direction := guardrails.DirectionInput
	if dir, ok := args["direction"].(string); ok && dir == string(guardrails.DirectionOutput) {
		direction = guardrails.DirectionOutput
	}

	result, err := s.guardrailsEngine.ClassifyContent(ctx, text, direction)
	if err != nil {
		return errorResult(`{"safe":true,"error":"classification failed"}`), nil
	}

	return jsonToolResult(result, result.IsBlocked())
}

// handleCheckPolicy implements the guardrail_check_policy tool: it checks
// whether text complies with a specific named content policy.
func (s *MCPServer) handleCheckPolicy(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	text, _ := args["text"].(string)
	policyID, _ := args["policy_id"].(string)

	if text == "" {
		return errorResult(`{"policy_id":"","compliant":false,"error":"text is required"}`), nil
	}
	if policyID == "" {
		return errorResult(`{"policy_id":"","compliant":false,"error":"policy_id is required"}`), nil
	}

	if s.guardrailsEngine == nil {
		return errorResult(`{"policy_id":"` + policyID + `","compliant":false,"error":"guardrails engine not configured"}`), nil
	}

	result, err := s.guardrailsEngine.CheckPolicy(ctx, text, policyID)
	if err != nil {
		return errorResult(`{"policy_id":"` + policyID + `","compliant":false,"error":"policy check failed"}`), nil
	}

	return jsonToolResult(result, !result.Compliant)
}
