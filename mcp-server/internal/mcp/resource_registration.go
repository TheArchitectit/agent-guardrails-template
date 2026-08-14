package mcp

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/config"
)

// resourceSpec pairs a URI with the handler that produces its contents.
type resourceSpec struct {
	uri     string
	name    string
	handler func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error)
}

// setupResources registers all guardrail:// resources with the mcp-go server
// using the v0.58.0 AddResource API.
func (s *MCPServer) setupResources() {
	specs := []resourceSpec{
		{
			uri:  "guardrail://config",
			name: "Guardrail Configuration",
			handler: func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				configJSON, _ := json.MarshalIndent(s.config, "", "  ")
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(configJSON),
					},
				}, nil
			},
		},
		{
			uri:  "guardrail://stats",
			name: "Guardrail Usage Stats",
			handler: func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
				stats, err := json.MarshalIndent(s.usageStats(ctx), "", "  ")
				if err != nil {
					return nil, fmt.Errorf("failed to marshal stats: %w", err)
				}
				return []mcp.ResourceContents{
					mcp.TextResourceContents{
						URI:      req.Params.URI,
						MIMEType: "application/json",
						Text:     string(stats),
					},
				}, nil
			},
		},
		{
			uri:     "guardrail://agent-guardrails",
			name:    "Agent Guardrails Documentation",
			handler: s.adaptResource(s.readAgentGuardrailsResource),
		},
		{
			uri:     "guardrail://workflows",
			name:    "Workflow Documentation",
			handler: s.adaptResource(s.readWorkflowsResource),
		},
		{
			uri:     "guardrail://standards",
			name:    "Standards Documentation",
			handler: s.adaptResource(s.readStandardsResource),
		},
		{
			uri:     "guardrail://four-laws",
			name:    "Four Laws of Agent Safety",
			handler: s.adaptResource(s.readFourLawsResource),
		},
		{
			uri:     "guardrail://halt-conditions",
			name:    "Halt Conditions",
			handler: s.adaptResource(s.readHaltConditionsResource),
		},
		{
			uri:     "guardrail://pre-work-check",
			name:    "Pre-Work Checklist",
			handler: s.adaptResource(s.readPreWorkChecklistResource),
		},
		{
			uri:     "guardrail://git-safety",
			name:    "Git Safety Policy",
			handler: s.adaptResource(s.readGitSafetyPolicyResource),
		},
		{
			uri:     "guardrail://test-prod-separation",
			name:    "Test/Production Separation Policy",
			handler: s.adaptResource(s.readTestProdSeparationPolicyResource),
		},
		{
			uri:     "guardrail://advisors",
			name:    "Available Advisors",
			handler: s.adaptResource(s.readAvailableAdvisorsResource),
		},
	}

	for _, spec := range specs {
		s.mcpServer.AddResource(
			mcp.Resource{URI: spec.uri, Name: spec.name},
			spec.handler,
		)
	}
}

// adaptResource converts a method returning *mcp.ReadResourceResult into the
// ResourceHandlerFunc signature expected by AddResource in v0.58.0.
func (s *MCPServer) adaptResource(
	fn func(ctx context.Context, uri string) (*mcp.ReadResourceResult, error),
) func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
	return func(ctx context.Context, req mcp.ReadResourceRequest) ([]mcp.ResourceContents, error) {
		result, err := fn(ctx, req.Params.URI)
		if err != nil {
			return nil, err
		}
		return result.Contents, nil
	}
}

// usageStats builds a small JSON-serializable snapshot of server usage stats.
func (s *MCPServer) usageStats(ctx context.Context) map[string]interface{} {
	return map[string]interface{}{
		"version": s.version,
		"schema":  config.SchemaVersion,
	}
}
