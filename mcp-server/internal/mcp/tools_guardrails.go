package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// guardrailsToolList returns the guardrail tools defined by Spec 01 Sec. 3.1
// (injection detection) and Spec 02 Sec. 3.1 (semantic content filtering).
func (s *MCPServer) guardrailsToolList() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "guardrail_detect_injection",
			Description: "Scan text for prompt injection attempts and return the verdict, the layer that decided it, and the matched categories",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"text": map[string]any{
						"type":        "string",
						"description": "The text to scan for injection attempts",
					},
					"source": map[string]any{
						"type":        "string",
						"description": "Where the text came from; selects the per-source fail policy (user is advisory, the others block)",
						"enum":        []string{"user", "tool_output", "file_content", "api_response"},
					},
					"context": map[string]any{
						"type":        "string",
						"description": "Optional provenance label naming where the text originated (for example 'file:README.md'); recorded on the audit event, never treated as instructions",
					},
				},
				Required: []string{"text"},
			},
		},
		{
			Name:        "guardrail_scan_text_batch",
			Description: "Scan many texts for prompt injection in one call and return a verdict per item",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"texts": map[string]any{
						"type":        "array",
						"description": "Items to scan; each item carries the text, its source, and a caller-chosen id echoed back in the result",
						"items": map[string]any{
							"type": "object",
							"properties": map[string]any{
								"text": map[string]any{
									"type":        "string",
									"description": "The text to scan",
								},
								"source": map[string]any{
									"type":        "string",
									"description": "Where the text came from; selects the per-source fail policy",
									"enum":        []string{"user", "tool_output", "file_content", "api_response"},
								},
								"id": map[string]any{
									"type":        "string",
									"description": "Caller-chosen identifier echoed back so results can be matched to inputs",
								},
							},
							"required": []string{"text", "id"},
						},
					},
				},
				Required: []string{"texts"},
			},
		},
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
func (s *MCPServer) handleClassifyContent(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	text, _ := args["text"].(string)
	if text == "" {
		return errorResult(`{"safe":true,"error":"text is required"}`), nil
	}

	// Fail closed (P5/D-18: fail-closed is not configurable). An unconfigured
	// engine means content filtering is unavailable — the same class of
	// condition the ContentFilter fail-closed policy handles for an unavailable
	// backend. Emitting `"safe":true` here told every caller the content was
	// safe when nothing had classified it, so a deployment that forgot
	// OLLAMA_URL silently lost its content filter. Report not-safe instead.
	if s.guardrailsEngine == nil {
		return errorResult(`{"safe":false,"error":"guardrails engine not configured"}`), nil
	}

	direction := guardrails.DirectionInput
	if dir, ok := args["direction"].(string); ok && dir == string(guardrails.DirectionOutput) {
		direction = guardrails.DirectionOutput
	}

	result, err := s.guardrailsEngine.ClassifyContent(ctx, text, direction)
	if err != nil {
		// Fail closed: a classification error means the content was NOT
		// classified. Do not report it as safe.
		return errorResult(`{"safe":false,"error":"classification failed"}`), nil
	}

	return jsonToolResult(result, result.IsBlocked())
}

// handleCheckPolicy implements the guardrail_check_policy tool: it checks
// whether text complies with a specific named content policy.
func (s *MCPServer) handleCheckPolicy(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
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

	// Classify first so we can tell the fail-open case apart from a genuine
	// clean result. PolicyEngine.CheckPolicy returns Compliant:true with no
	// violations both when a known policy passes AND when the policy_id is
	// unknown (no rule to evaluate). The latter is fail-open: blocked content
	// that the policy should have caught slips through. Fail closed instead.
	classification, err := s.guardrailsEngine.ClassifyContent(ctx, text, guardrails.DirectionOutput)
	if err != nil {
		return errorResult(`{"policy_id":"` + policyID + `","compliant":false,"error":"classification failed"}`), nil
	}

	result, err := s.guardrailsEngine.CheckPolicy(ctx, text, policyID)
	if err != nil {
		return errorResult(`{"policy_id":"` + policyID + `","compliant":false,"error":"policy check failed"}`), nil
	}

	// Fail-open guard: a compliant result with no violations is only valid
	// when the policy actually evaluated the content. If the classification
	// found blocked content that the policy should have caught, the
	// compliant-with-no-violations response means the policy_id was unknown
	// (or the engine has no rules for it). Return an explicit error.
	if result.Compliant && len(result.Violations) == 0 && classification.IsBlocked() {
		return errorResult(`{"policy_id":"` + policyID + `","compliant":false,"error":"unknown policy_id: policy does not exist or has no rules"}`), nil
	}

	return jsonToolResult(result, !result.Compliant)
}

// injectionSources is the closed set of source values the injection tools
// accept. Anything else is refused rather than defaulted, because the source
// selects the fail policy: silently treating an unknown value as "user" would
// apply the weakest policy (Spec 01 Sec. 3.2 source_policies).
var injectionSources = map[string]guardrails.Source{
	"user":         guardrails.SourceUser,
	"tool_output":  guardrails.SourceToolOutput,
	"file_content": guardrails.SourceFileContent,
	"api_response": guardrails.SourceAPIResponse,
}

// resolveInjectionSource maps a caller-supplied source onto the closed set.
// An absent source means user-supplied text, the weakest policy; an unknown
// source is an error, never a silent downgrade to the weakest policy.
func resolveInjectionSource(raw string) (guardrails.Source, bool) {
	if raw == "" {
		return guardrails.SourceUser, true
	}
	src, ok := injectionSources[raw]
	return src, ok
}

// handleDetectInjection implements the guardrail_detect_injection tool from
// Spec 01 Sec. 3.1: it scans one text and returns the verdict, deciding layer,
// confidence, reason and matched categories.
func (s *MCPServer) handleDetectInjection(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	text, _ := args["text"].(string)
	if text == "" {
		return errorResult(`{"safe":false,"error":"text is required"}`), nil
	}

	sourceRaw, _ := args["source"].(string)
	source, ok := resolveInjectionSource(sourceRaw)
	if !ok {
		return errorResult(`{"safe":false,"error":"source must be one of user, tool_output, file_content, api_response"}`), nil
	}

	// Fail closed (P5/D-18). An absent engine means nothing scanned the text;
	// reporting safe:true would assert a verdict that was never reached.
	if s.guardrailsEngine == nil {
		return errorResult(`{"safe":false,"error":"guardrails engine not configured"}`), nil
	}

	// Spec 01 declares SourceTool/ToolCallID on the audit event; attach the
	// calling tool and the optional provenance label so a record can be tied
	// back to the call that produced it.
	provenance, _ := args["context"].(string)
	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_detect_injection", provenance)

	result := s.guardrailsEngine.DetectInjection(ctx, text, source)
	return jsonToolResult(result, !result.Safe)
}

// errorJSON builds a well-formed error result for a message that carries
// caller-supplied data. The message is JSON-escaped rather than concatenated,
// so a quote or control character in an id cannot produce malformed output or
// inject fields into the response.
func errorJSON(msg string) (*mcp.CallToolResult, error) {
	payload, err := json.Marshal(map[string]any{"results": []any{}, "error": msg})
	if err != nil {
		return errorResult(`{"results":[],"error":"internal encoding failure"}`), nil
	}
	return errorResult(string(payload)), nil
}

// maxBatchItems bounds a single batch scan so one call cannot be used to pin
// the server (each item may reach an outbound classifier).
const maxBatchItems = 256

// batchScanItem is one element of the guardrail_scan_text_batch input.
type batchScanItem struct {
	Text   string `json:"text"`
	Source string `json:"source"`
	ID     string `json:"id"`
}

// handleScanTextBatch implements the guardrail_scan_text_batch tool from
// Spec 01 Sec. 3.1. Every item is scanned independently; one malformed item
// fails the whole call rather than being silently dropped, because a caller
// that receives fewer results than it sent could mistake the gap for a pass.
func (s *MCPServer) handleScanTextBatch(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	raw, ok := args["texts"]
	if !ok || raw == nil {
		return errorResult(`{"results":[],"error":"texts is required"}`), nil
	}

	// The MCP layer hands arguments through as decoded JSON, so the array
	// arrives as []any of map[string]any.
	entries, ok := raw.([]any)
	if !ok {
		return errorResult(`{"results":[],"error":"texts must be an array"}`), nil
	}
	if len(entries) == 0 {
		return errorResult(`{"results":[],"error":"texts must contain at least one item"}`), nil
	}
	if len(entries) > maxBatchItems {
		return errorResult(`{"results":[],"error":"texts exceeds the 256 item limit"}`), nil
	}

	if s.guardrailsEngine == nil {
		return errorResult(`{"results":[],"error":"guardrails engine not configured"}`), nil
	}

	items := make([]guardrails.BatchItem, 0, len(entries))
	seen := make(map[string]bool, len(entries))
	for index, entry := range entries {
		obj, ok := entry.(map[string]any)
		if !ok {
			return errorResult(`{"results":[],"error":"each item must be an object with text and id"}`), nil
		}
		text, _ := obj["text"].(string)
		id, _ := obj["id"].(string)
		if text == "" {
			return errorResult(`{"results":[],"error":"item ` + itoa(index) + ` has no text"}`), nil
		}
		if id == "" {
			return errorResult(`{"results":[],"error":"item ` + itoa(index) + ` has no id"}`), nil
		}
		// Duplicate ids would make the echoed results ambiguous, so they are
		// refused rather than silently collapsing two items into one key.
		if seen[id] {
			return errorJSON("duplicate item id: " + id)
		}
		seen[id] = true

		sourceRaw, _ := obj["source"].(string)
		source, ok := resolveInjectionSource(sourceRaw)
		if !ok {
			return errorResult(`{"results":[],"error":"item ` + itoa(index) + ` has an invalid source"}`), nil
		}

		items = append(items, guardrails.BatchItem{ID: id, Text: text, Source: source})
	}

	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_scan_text_batch", "")
	batch := s.guardrailsEngine.DetectInjectionBatch(ctx, items)

	// A batch that came back short is an internal inconsistency: the caller
	// would see a missing verdict as an absent item rather than a failure.
	if len(batch.Results) != len(items) {
		return errorResult(`{"results":[],"error":"batch scan returned an incomplete result set"}`), nil
	}

	anyUnsafe := false
	for _, result := range batch.Results {
		if !result.Safe {
			anyUnsafe = true
			break
		}
	}
	return jsonToolResult(batch, anyUnsafe)
}

// itoa formats a small index without pulling strconv into every caller.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
