package mcp

import (
	"github.com/mark3labs/mcp-go/mcp"
)

// toolList returns the full set of MCP tools registered by the server.
// It includes the core guardrail tools plus conditionally appended vision,
// notification, budget, and lifecycle tools depending on which subsystems
// are initialized on the server.
func (s *MCPServer) toolList() []mcp.Tool {
	tools := coreTools()
	tools = append(tools, coreToolsExtended()...)

	// Semantic content filtering tools (always registered; return a
	// not-configured result until the guardrails engine is set).
	tools = append(tools, s.guardrailsToolList()...)

	if s.visionTools != nil {
		tools = append(tools, s.visionTools.visionToolList()...)
	}

	// Webhook notification tools
	if s.webhookStore != nil {
		tools = append(tools, s.notificationToolList()...)
	}

	// Budget management tools
	if s.budgetStore != nil {
		tools = append(tools, s.budgetToolList()...)
	}

	// Agent lifecycle tools
	if s.agentStateStore != nil {
		tools = append(tools, s.lifecycleToolList()...)
	}

	return tools
}
