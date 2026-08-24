package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/metrics"
	"github.com/thearchitectit/guardrail-mcp/internal/team"
)

func (s *MCPServer) handleTeamStatus(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_status")
	defer func() {
		metrics.DecrementTeamToolActive("team_status")
		metrics.RecordTeamToolDuration("team_status", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_status", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_status", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_status", "go_error")
		metrics.RecordTeamToolCall("team_status", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.Load(); err != nil {
		metrics.RecordTeamToolError("team_status", "go_error")
		metrics.RecordTeamToolCall("team_status", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error loading project: %v", err)}},
			IsError: true,
		}, nil
	}

	var resultText string
	if phase, ok := args["phase"].(string); ok && phase != "" {
		status, _ := mgr.GetPhaseStatus(phase)
		data, _ := json.MarshalIndent(status, "", "  ")
		resultText = string(data)
	} else {
		status := mgr.GetProjectStatus()
		data, _ := json.MarshalIndent(status, "", "  ")
		resultText = string(data)
	}
	metrics.RecordTeamToolDuration("team_status", time.Since(goStart))

	metrics.RecordTeamToolCall("team_status", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handlePhaseGateCheck checks if phase gate requirements are met
func (s *MCPServer) handlePhaseGateCheck(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("phase_gate_check")
	defer func() {
		metrics.DecrementTeamToolActive("phase_gate_check")
		metrics.RecordTeamToolDuration("phase_gate_check", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("phase_gate_check", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("phase_gate_check", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	fromPhase, ok := args["from_phase"].(float64)
	if !ok {
		metrics.RecordTeamToolError("phase_gate_check", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: from_phase is required"}},
			IsError: true,
		}, nil
	}

	toPhase, ok := args["to_phase"].(float64)
	if !ok {
		metrics.RecordTeamToolError("phase_gate_check", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: to_phase is required"}},
			IsError: true,
		}, nil
	}

	// Load team layout rules
	rules, err := loadTeamLayoutRules()
	if err != nil {
		metrics.RecordTeamToolError("phase_gate_check", "rules_error")
		metrics.RecordTeamToolCall("phase_gate_check", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error loading team rules: %v", err),
			}},
			IsError: true,
		}, nil
	}

	// Map phases to gate names
	gateName := fmt.Sprintf("%d_to_%d", int(fromPhase), int(toPhase))
	gate, exists := rules.PhaseGates[gateName]
	if !exists {
		metrics.RecordTeamToolError("phase_gate_check", "gate_not_found")
		metrics.RecordTeamToolCall("phase_gate_check", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No phase gate defined from phase %d to phase %d", int(fromPhase), int(toPhase)),
			}},
			IsError: true,
		}, nil
	}

	// Build response
	var response strings.Builder
	fmt.Fprintf(&response, "# Phase Gate: %s\n\n", gate.Name)
	response.WriteString("**Required Teams:**\n")
	for _, teamID := range gate.RequiredTeams {
		fmt.Fprintf(&response, "- Team %d\n", teamID)
	}

	response.WriteString("\n**Required Deliverables:**\n")
	for _, deliverable := range gate.Deliverables {
		fmt.Fprintf(&response, "- [ ] %s\n", deliverable)
	}

	metrics.RecordTeamToolCall("phase_gate_check", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response.String()}},
	}, nil
}

// handleAgentTeamMap gets the team assignment for an agent type
func (s *MCPServer) handleAgentTeamMap(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("agent_team_map")
	defer func() {
		metrics.DecrementTeamToolActive("agent_team_map")
		metrics.RecordTeamToolDuration("agent_team_map", time.Since(start))
	}()

	agentType, ok := args["agent_type"].(string)
	if !ok || agentType == "" {
		metrics.RecordTeamToolError("agent_team_map", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: agent_type is required"}},
			IsError: true,
		}, nil
	}

	// Load team layout rules
	rules, err := loadTeamLayoutRules()
	if err != nil {
		metrics.RecordTeamToolError("agent_team_map", "rules_error")
		metrics.RecordTeamToolCall("agent_team_map", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error loading team rules: %v", err),
			}},
			IsError: true,
		}, nil
	}

	mapping, exists := rules.AgentMapping[agentType]
	if !exists {
		metrics.RecordTeamToolError("agent_team_map", "mapping_not_found")
		metrics.RecordTeamToolCall("agent_team_map", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No team mapping found for agent type: %s", agentType),
			}},
			IsError: true,
		}, nil
	}

	result := fmt.Sprintf(
		"# Agent Team Assignment\n\n"+
			"**Agent Type:** %s\n"+
			"**Assigned Team:** Team %d\n"+
			"**Phase:** %s\n"+
			"**Roles:** %s\n",
		agentType,
		mapping.Team,
		mapping.Phase,
		strings.Join(mapping.Roles, ", "),
	)

	metrics.RecordTeamToolCall("agent_team_map", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: result}},
	}, nil
}

// handleTeamSizeValidate validates team sizes meet 4-6 member requirement
func (s *MCPServer) handleTeamSizeValidate(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_size_validate")
	defer func() {
		metrics.DecrementTeamToolActive("team_size_validate")
		metrics.RecordTeamToolDuration("team_size_validate", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_size_validate", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_size_validate", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName)
	if err != nil {
		metrics.RecordTeamToolError("team_size_validate", "go_error")
		metrics.RecordTeamToolCall("team_size_validate", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()

	// Get all teams
	teams := mgr.GetAllTeams()

	// Validate team sizes
	var violations []string
	for _, t := range teams {
		assignedCount := 0
		for _, role := range t.Roles {
			if role.AssignedTo != nil && *role.AssignedTo != "" {
				assignedCount++
			}
		}
		if assignedCount < 4 || assignedCount > 6 {
			violations = append(violations, fmt.Sprintf("Team %d (%s): %d members (requires 4-6)", t.ID, t.Name, assignedCount))
		}
	}

	metrics.RecordTeamToolDuration("team_size_validate", time.Since(goStart))

	if len(violations) > 0 {
		resultText := fmt.Sprintf("❌ Team size validation failed:\n%s", strings.Join(violations, "\n"))
		metrics.RecordTeamToolCall("team_size_validate", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	resultText := fmt.Sprintf("✅ All teams in project '%s' have valid sizes (4-6 members)", projectName)
	metrics.RecordTeamToolCall("team_size_validate", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// Helper types and functions

type TeamLayoutRules struct {
	Name         string               `json:"name"`
	Version      string               `json:"version"`
	Description  string               `json:"description"`
	AppliesTo    []string             `json:"applies_to"`
	Rules        []TeamRule           `json:"rules"`
	PhaseGates   map[string]PhaseGate `json:"phase_gates"`
	AgentMapping map[string]AgentTeam `json:"agent_mapping"`
}

type TeamRule struct {
	ID       string   `json:"id"`
	Name     string   `json:"name"`
	Severity string   `json:"severity"`
	Check    string   `json:"check"`
	Command  string   `json:"command"`
	Message  string   `json:"message"`
	Trigger  string   `json:"trigger,omitempty"`
	Patterns []string `json:"patterns,omitempty"`
}

type PhaseGate struct {
	Name             string   `json:"name"`
	RequiredTeams    []int    `json:"required_teams"`
	ApprovalRequired []int    `json:"approval_required"`
	Deliverables     []string `json:"deliverables"`
}

type AgentTeam struct {
	Team  int      `json:"team"`
	Roles []string `json:"roles"`
	Phase string   `json:"phase"`
}

// handleTeamDelete deletes a specific team from a project

func (s *MCPServer) handleTeamStart(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_start")
	defer func() {
		metrics.DecrementTeamToolActive("team_start")
		metrics.RecordTeamToolDuration("team_start", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_start", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_start", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		metrics.RecordTeamToolError("team_start", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		metrics.RecordTeamToolError("team_start", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	// FUNC-010: Handle override option
	override := false
	if overrideVal, ok := args["override"].(bool); ok {
		override = overrideVal
	}

	if override {
		// Reason is required when overriding
		reason, ok := args["reason"].(string)
		if !ok || reason == "" {
			metrics.RecordTeamToolError("team_start", "validation_error")
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: reason is required when using override"}},
				IsError: true,
			}, nil
		}
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_start", "go_error")
		metrics.RecordTeamToolCall("team_start", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.Load(); err != nil {
		metrics.RecordTeamToolError("team_start", "go_error")
		metrics.RecordTeamToolCall("team_start", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error loading project: %v", err)}},
			IsError: true,
		}, nil
	}

	if err := mgr.StartTeam(teamIDInt, override, ""); err != nil {
		metrics.RecordTeamToolDuration("team_start", time.Since(goStart))
		metrics.RecordTeamToolError("team_start", "go_error")
		metrics.RecordTeamToolCall("team_start", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error starting team: %v", err)}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_start", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Started Team %d (%s)", teamIDInt, team.StandardTeams[teamIDInt].Name)
	if override {
		resultText += " (with override)"
	}
	metrics.RecordTeamToolCall("team_start", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamStatus gets phase or project status
