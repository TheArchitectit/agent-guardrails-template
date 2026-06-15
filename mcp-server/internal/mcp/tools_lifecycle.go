package mcp

import (
	"context"
	"fmt"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// lifecycleToolList returns the tool definitions for agent lifecycle management.
func (s *MCPServer) lifecycleToolList() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "create_agent_session",
			Description: "Create a new agent session in idle state. The agent must transition through planning → active → review → release lifecycle.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID the agent belongs to",
					},
					"agent_name": map[string]interface{}{
						"type":        "string",
						"description": "Agent name or identifier",
					},
					"project_slug": map[string]interface{}{
						"type":        "string",
						"description": "Optional project slug the agent is working on",
					},
				},
				Required: []string{"team_id", "agent_name"},
			},
		},
		{
			Name:        "transition_agent_state",
			Description: "Move an agent to a new lifecycle state. Valid transitions: idle→planning, planning→active|idle, active→review|halted|idle, review→release|active|halted, release→idle, halted→idle.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Agent session ID",
					},
					"to_state": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"idle", "planning", "active", "review", "release", "halted"},
						"description": "Target state",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Reason for the transition (for audit trail)",
					},
					"triggered_by": map[string]interface{}{
						"type":        "string",
						"description": "Who triggered this transition (default: agent)",
					},
				},
				Required: []string{"session_id", "to_state"},
			},
		},
		{
			Name:        "get_agent_state",
			Description: "Get the current state and recent transition history of an agent session.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Agent session ID",
					},
				},
				Required: []string{"session_id"},
			},
		},
		{
			Name:        "list_agent_sessions",
			Description: "List all agent sessions for a team, showing current states.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID to list sessions for",
					},
				},
				Required: []string{"team_id"},
			},
		},
		{
			Name:        "force_agent_state",
			Description: "Admin override: force an agent to any state, bypassing transition validation. Requires justification.",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"session_id": map[string]interface{}{
						"type":        "string",
						"description": "Agent session ID",
					},
					"to_state": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"idle", "planning", "active", "review", "release", "halted"},
						"description": "Target state",
					},
					"reason": map[string]interface{}{
						"type":        "string",
						"description": "Justification for the override (required)",
					},
				},
				Required: []string{"session_id", "to_state", "reason"},
			},
		},
	}
}

func (s *MCPServer) handleCreateAgentSession(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	agentName, _ := args["agent_name"].(string)
	projectSlug, _ := args["project_slug"].(string)

	if teamID == "" || agentName == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id and agent_name are required",
		}, true)
	}

	session := &models.AgentSession{
		TeamID:       teamID,
		AgentName:    agentName,
		CurrentState: models.AgentStateIdle,
		ProjectSlug:  projectSlug,
	}

	if err := s.agentStateStore.CreateSession(ctx, session); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to create session: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success":       true,
		"session_id":    session.ID,
		"current_state": session.CurrentState,
		"agent_name":    agentName,
		"message":       "Agent session created in idle state",
	}, false)
}

func (s *MCPServer) handleTransitionAgentState(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionID, _ := args["session_id"].(string)
	toState, _ := args["to_state"].(string)
	reason, _ := args["reason"].(string)
	triggeredBy, _ := args["triggered_by"].(string)

	if sessionID == "" || toState == "" {
		return buildToolResult(map[string]interface{}{
			"error": "session_id and to_state are required",
		}, true)
	}

	if triggeredBy == "" {
		triggeredBy = "agent"
	}

	if err := s.agentStateStore.Transition(ctx, sessionID, models.AgentState(toState), reason, triggeredBy); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("transition failed: %v", err),
		}, true)
	}

	session, _ := s.agentStateStore.GetSession(ctx, sessionID)

	return buildToolResult(map[string]interface{}{
		"success":        true,
		"session_id":     sessionID,
		"current_state":  toState,
		"previous_state": session.PreviousState,
		"message":        fmt.Sprintf("Transitioned to %s", toState),
	}, false)
}

func (s *MCPServer) handleGetAgentState(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionID, _ := args["session_id"].(string)
	if sessionID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "session_id is required",
		}, true)
	}

	session, err := s.agentStateStore.GetSession(ctx, sessionID)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("session not found: %v", err),
		}, true)
	}

	transitions, err := s.agentStateStore.GetTransitions(ctx, sessionID)
	if err != nil {
		transitions = nil // Non-fatal
	}

	return buildToolResult(map[string]interface{}{
		"session":      session,
		"transitions":  transitions,
	}, false)
}

func (s *MCPServer) handleListAgentSessions(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	if teamID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id is required",
		}, true)
	}

	sessions, err := s.agentStateStore.ListByTeam(ctx, teamID)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to list sessions: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"sessions": sessions,
		"count":    len(sessions),
	}, false)
}

func (s *MCPServer) handleForceAgentState(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionID, _ := args["session_id"].(string)
	toState, _ := args["to_state"].(string)
	reason, _ := args["reason"].(string)

	if sessionID == "" || toState == "" || reason == "" {
		return buildToolResult(map[string]interface{}{
			"error": "session_id, to_state, and reason are all required for admin override",
		}, true)
	}

	if err := s.agentStateStore.ForceState(ctx, sessionID, models.AgentState(toState), reason); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("force state failed: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success":    true,
		"session_id": sessionID,
		"new_state":  toState,
		"message":    fmt.Sprintf("State forced to %s (admin override)", toState),
		"warning":    "Transition validation was bypassed",
	}, false)
}
