package mcp

import (
	"context"
	"strings"
	"testing"
)

// TestIntegrationAgentTeamMap tests agent team mapping
func TestIntegrationAgentTeamMap(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()

	agentTypes := []string{
		"planner",
		"architect",
		"infrastructure",
		"platform",
		"backend",
		"frontend",
		"security",
		"qa",
		"sre",
		"ops",
	}

	for _, agentType := range agentTypes {
		t.Run("Agent: "+agentType, func(t *testing.T) {
			args := map[string]any{
				"agent_type": agentType,
			}

			result, err := s.handleAgentTeamMap(ctx, args)
			if err != nil {
				t.Fatalf("handleAgentTeamMap failed: %v", err)
			}

			if result.IsError {
				t.Errorf("handleAgentTeamMap returned error for agent %s: %s",
					agentType, getResultText(result))
			}

			text := getResultText(result)

			// Verify structure
			if !strings.Contains(text, "Agent Team Assignment") {
				t.Error("Expected 'Agent Team Assignment' header")
			}
			if !strings.Contains(text, "Agent Type:") {
				t.Error("Expected 'Agent Type' field")
			}
			if !strings.Contains(text, "Assigned Team:") {
				t.Error("Expected 'Assigned Team' field")
			}
			if !strings.Contains(text, "Phase:") {
				t.Error("Expected 'Phase' field")
			}
			if !strings.Contains(text, "Roles:") {
				t.Error("Expected 'Roles' field")
			}
		})
	}
}
