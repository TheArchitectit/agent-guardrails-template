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

	// Multi-agent safety tools (Spec 04). Registered unconditionally for the
	// same reason: a tool that disappears when its backend is missing cannot
	// tell a caller "not configured" — it tells them "this server has no such
	// capability", and the absence is invisible to tools/list diffs.
	tools = append(tools, s.multiAgentTools()...)

	// Regulatory compliance mapping (Spec 06 gap-spec). Registered
	// unconditionally: a compliance report whose probes are missing reports
	// every control as unmeasured, which is a truthful answer, whereas a tool
	// that vanishes cannot say "not configured" at all.
	tools = append(tools, s.complianceTools()...)

	// Runtime sandbox isolation (Spec 03 gap-spec). Registered
	// unconditionally for the same reason as the two groups above: a caller
	// must be able to learn that the sandbox is not configured, rather than
	// inferring from a missing tool that this server has no such capability.
	tools = append(tools, s.sandboxTools()...)

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
