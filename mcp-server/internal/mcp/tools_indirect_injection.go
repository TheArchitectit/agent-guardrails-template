package mcp

import (
	"context"
	"encoding/json"

	"github.com/mark3labs/mcp-go/mcp"

	"github.com/thearchitectit/guardrail-mcp/internal/guardrails"
)

// indirectInjectionTools returns the tool set defined by Spec 05 Sec. 3.1
// (indirect prompt injection handling).
//
// These exist because Spec 05 shipped as a library that nothing reached: the
// tracker, sanitiser and trust policies were all written, and grep found zero
// non-test callers from the MCP surface, so a deployment could not ask the
// question the spec exists to answer — "this text came from a file, is it safe
// for the agent to read?".
func (s *MCPServer) indirectInjectionTools() []mcp.Tool {
	return []mcp.Tool{
		{
			Name: "guardrail_scan_external_content",
			Description: "Scan content from outside the agent's instruction stream (file, API response, tool output) for indirect " +
				"prompt injection: strips invisible characters, decodes labelled obfuscation, applies the source trust policy, and " +
				"returns the sanitized text the agent should consume",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "The external content to scan; scan the text the agent would actually read",
					},
					"source_type": map[string]any{
						"type":        "string",
						"description": "Kind of source the content came from (file, api, tool_output, web); recorded on the provenance record",
					},
					"source_path": map[string]any{
						"type":        "string",
						"description": "Path, URL or identifier of the source; selects the trust policy by pattern match",
					},
					"agent_id": map[string]any{
						"type":        "string",
						"description": "Identifier of the agent that will read this content",
					},
				},
				Required: []string{"content", "source_type"},
			},
		},
		{
			Name:        "guardrail_mark_provenance",
			Description: "Wrap content in provenance markers so the agent can tell trusted instructions from untrusted data it happens to have read",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"content": map[string]any{
						"type":        "string",
						"description": "The content to mark",
					},
					"provenance_id": map[string]any{
						"type":        "string",
						"description": "Provenance id returned by guardrail_scan_external_content",
					},
				},
				Required: []string{"content"},
			},
		},
		{
			Name:        "guardrail_check_provenance",
			Description: "Look up a recorded provenance verdict by id: what source it came from, its trust level, and whether it was scanned",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: map[string]any{
					"content_id": map[string]any{
						"type":        "string",
						"description": "Provenance id to look up",
					},
				},
				Required: []string{"content_id"},
			},
		},
	}
}

// handleScanExternalContent implements guardrail_scan_external_content.
func (s *MCPServer) handleScanExternalContent(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	content, _ := args["content"].(string)
	sourceType, _ := args["source_type"].(string)
	sourcePath, _ := args["source_path"].(string)
	agentID, _ := args["agent_id"].(string)

	if content == "" {
		return errorResult(`{"safe":false,"error":"content is required"}`), nil
	}
	if sourceType == "" {
		return errorResult(`{"safe":false,"error":"source_type is required"}`), nil
	}

	// Fail closed: an unconfigured engine has tracked nothing, so it cannot
	// call any source trusted. Reporting safe here would be a verdict the
	// server never reached.
	if s.guardrailsEngine == nil {
		return errorResult(`{"safe":false,"error":"guardrails engine not configured"}`), nil
	}

	ctx = guardrails.WithAuditAttribution(ctx, "guardrail_scan_external_content", sourcePath)

	result, err := s.guardrailsEngine.ScanExternalContent(ctx, content, sourceType, sourcePath, agentID)
	if err != nil {
		// A scan that could not run is not a pass. The reason is carried so an
		// operator can tell "provenance disabled" from "pipeline missing".
		payload, _ := json.Marshal(map[string]any{
			"safe":   false,
			"error":  "external content scan unavailable",
			"detail": err.Error(),
		})
		return errorResult(string(payload)), nil
	}

	return jsonToolResult(result, false)
}

// handleMarkProvenance implements guardrail_mark_provenance.
func (s *MCPServer) handleMarkProvenance(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	content, _ := args["content"].(string)
	provenanceID, _ := args["provenance_id"].(string)

	if content == "" {
		return errorResult(`{"marked_content":"","error":"content is required"}`), nil
	}

	// Without a known provenance record the only honest marker is the
	// unknown/untrusted one. Silently marking content as trusted because the
	// id was missing would defeat the control.
	var prov *guardrails.Provenance
	if s.guardrailsEngine != nil && provenanceID != "" {
		if scan, ok := s.guardrailsEngine.LookupProvenance(provenanceID); ok {
			prov = scan.Provenance
		}
	}

	marked := guardrails.MarkContent(content, prov, provenanceID)
	return jsonToolResult(map[string]any{
		"marked_content": marked,
		"provenance_id":  provenanceID,
		"known":          prov != nil,
	}, false)
}

// handleCheckProvenance implements guardrail_check_provenance.
func (s *MCPServer) handleCheckProvenance(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	contentID, _ := args["content_id"].(string)
	if contentID == "" {
		return errorResult(`{"error":"content_id is required"}`), nil
	}
	if s.guardrailsEngine == nil {
		return errorResult(`{"content_id":"` + contentID + `","found":false,"error":"guardrails engine not configured"}`), nil
	}

	scan, ok := s.guardrailsEngine.LookupProvenance(contentID)
	if !ok {
		// "not found" is a real answer, not an error: the id may have expired
		// or may never have been issued. It is reported as found=false so a
		// caller cannot read an absent record as a clean one.
		return jsonToolResult(map[string]any{
			"content_id": contentID,
			"found":      false,
			"reason":     "no recorded scan for this id (unknown, or the record expired)",
		}, false)
	}

	return jsonToolResult(map[string]any{
		"content_id":  contentID,
		"found":       true,
		"provenance":  scan.Provenance,
		"scan_result": scan,
	}, false)
}
