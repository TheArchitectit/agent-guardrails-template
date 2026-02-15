package mcp

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
)

// validateProjectName validates project name to prevent command injection
func validateProjectName(name string) error {
	if name == "" {
		return fmt.Errorf("project_name is required")
	}
	if len(name) > 64 {
		return fmt.Errorf("project_name must be 64 characters or less")
	}
	// Allow alphanumeric, hyphen, underscore only
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '-' || r == '_') {
			return fmt.Errorf("project_name must contain only letters, numbers, hyphens, and underscores")
		}
	}
	return nil
}

// Team tool handler implementations for MCP server

// handleTeamInit initializes team structure for a project
func (s *MCPServer) handleTeamInit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	cmd := exec.CommandContext(ctx, "python", "scripts/team_manager.py", "--project", projectName, "init")
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		resultText = fmt.Sprintf("Error initializing team: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamList lists all teams and their status
func (s *MCPServer) handleTeamList(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	cmdArgs := []string{"scripts/team_manager.py", "--project", projectName, "list"}
	if phase, ok := args["phase"].(string); ok && phase != "" {
		cmdArgs = append(cmdArgs, "--phase", phase)
	}

	cmd := exec.CommandContext(ctx, "python", cmdArgs...)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		resultText = fmt.Sprintf("Error listing teams: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamAssign assigns a person to a role in a team
func (s *MCPServer) handleTeamAssign(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	roleName, ok := args["role_name"].(string)
	if !ok || roleName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: role_name is required"}},
			IsError: true,
		}, nil
	}

	person, ok := args["person"].(string)
	if !ok || person == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: person is required"}},
			IsError: true,
		}, nil
	}

	cmd := exec.CommandContext(ctx, "python", "scripts/team_manager.py", "--project", projectName, "assign",
		"--team", strconv.Itoa(teamIDInt),
		"--role", roleName,
		"--person", person)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		resultText = fmt.Sprintf("Error assigning role: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamStatus gets phase or project status
func (s *MCPServer) handleTeamStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	cmdArgs := []string{"scripts/team_manager.py", "--project", projectName, "status"}
	if phase, ok := args["phase"].(string); ok && phase != "" {
		cmdArgs = append(cmdArgs, "--phase", phase)
	}

	cmd := exec.CommandContext(ctx, "python", cmdArgs...)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		resultText = fmt.Sprintf("Error getting status: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handlePhaseGateCheck checks if phase gate requirements are met
func (s *MCPServer) handlePhaseGateCheck(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	fromPhase, ok := args["from_phase"].(float64)
	if !ok {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: from_phase is required"}},
			IsError: true,
		}, nil
	}

	toPhase, ok := args["to_phase"].(float64)
	if !ok {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: to_phase is required"}},
			IsError: true,
		}, nil
	}

	// Load team layout rules
	rules, err := loadTeamLayoutRules()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{
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
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("No phase gate defined from phase %d to phase %d", int(fromPhase), int(toPhase)),
			}},
			IsError: true,
		}, nil
	}

	// Build response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("# Phase Gate: %s\n\n", gate.Name))
	response.WriteString("**Required Teams:**\n")
	for _, teamID := range gate.RequiredTeams {
		response.WriteString(fmt.Sprintf("- Team %d\n", teamID))
	}

	response.WriteString("\n**Required Deliverables:**\n")
	for _, deliverable := range gate.Deliverables {
		response.WriteString(fmt.Sprintf("- [ ] %s\n", deliverable))
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response.String()}},
	}, nil
}

// handleAgentTeamMap gets the team assignment for an agent type
func (s *MCPServer) handleAgentTeamMap(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	agentType, ok := args["agent_type"].(string)
	if !ok || agentType == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: agent_type is required"}},
			IsError: true,
		}, nil
	}

	// Load team layout rules
	rules, err := loadTeamLayoutRules()
	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error loading team rules: %v", err),
			}},
			IsError: true,
		}, nil
	}

	mapping, exists := rules.AgentMapping[agentType]
	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{
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

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: result}},
	}, nil
}

// handleTeamSizeValidate validates team sizes meet 4-6 member requirement
func (s *MCPServer) handleTeamSizeValidate(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	cmdArgs := []string{"scripts/team_manager.py", "--project", projectName, "validate-size"}
	if teamID, ok := args["team_id"].(float64); ok {
		cmdArgs = append(cmdArgs, "--team", strconv.Itoa(int(teamID)))
	}

	cmd := exec.CommandContext(ctx, "python", cmdArgs...)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		// Non-zero exit indicates validation failure
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// Helper types and functions

type TeamLayoutRules struct {
	Name         string                `json:"name"`
	Version      string                `json:"version"`
	Description  string                `json:"description"`
	AppliesTo    []string              `json:"applies_to"`
	Rules        []TeamRule            `json:"rules"`
	PhaseGates   map[string]PhaseGate  `json:"phase_gates"`
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
func (s *MCPServer) handleTeamDelete(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	// Check for confirmation
	confirmed := false
	if conf, ok := args["confirmed"].(bool); ok {
		confirmed = conf
	}

	cmdArgs := []string{"scripts/team_manager.py", "--project", projectName, "delete-team", "--team", strconv.Itoa(teamIDInt)}
	if confirmed {
		cmdArgs = append(cmdArgs, "--confirmed")
	}

	cmd := exec.CommandContext(ctx, "python", cmdArgs...)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		// Check if this is just a confirmation required error
		if strings.Contains(resultText, "requires confirmation") {
			return &mcp.CallToolResult{
				Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			}, nil
		}
		resultText = fmt.Sprintf("Error deleting team: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleProjectDelete deletes an entire project
func (s *MCPServer) handleProjectDelete(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Check for confirmation
	confirmed := false
	if conf, ok := args["confirmed"].(bool); ok {
		confirmed = conf
	}

	cmdArgs := []string{"scripts/team_manager.py", "--project", projectName, "delete-project"}
	if confirmed {
		cmdArgs = append(cmdArgs, "--confirmed")
	}

	cmd := exec.CommandContext(ctx, "python", cmdArgs...)
	output, err := cmd.CombinedOutput()

	resultText := string(output)
	if err != nil {
		// Check if this is just a confirmation required error
		if strings.Contains(resultText, "requires confirmation") {
			return &mcp.CallToolResult{
				Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			}, nil
		}
		resultText = fmt.Sprintf("Error deleting project: %v\nOutput: %s", err, string(output))
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

func loadTeamLayoutRules() (*TeamLayoutRules, error) {
	// Return hardcoded rules matching .guardrails/team-layout-rules.json
	return &TeamLayoutRules{
		Name:        "Team Layout Compliance",
		Version:     "1.0",
		Description: "Enforces standardized team structure",
		PhaseGates: map[string]PhaseGate{
			"1_to_2": {
				Name:             "Architecture Review Board",
				RequiredTeams:    []int{1, 2, 3},
				ApprovalRequired: []int{2},
				Deliverables:     []string{"Architecture Decision Records", "Approved Tech List", "Compliance Checklist"},
			},
			"2_to_3": {
				Name:             "Environment Readiness",
				RequiredTeams:    []int{4, 5, 6},
				ApprovalRequired: []int{4, 5},
				Deliverables:     []string{"Infrastructure Provisioned", "CI/CD Pipelines", "Data Models"},
			},
			"3_to_4": {
				Name:             "Feature Complete + Code Review",
				RequiredTeams:    []int{7, 8},
				ApprovalRequired: []int{7},
				Deliverables:     []string{"Features Implemented", "Code Reviewed", "Documentation Complete"},
			},
			"4_to_5": {
				Name:             "Security + QA Sign-off",
				RequiredTeams:    []int{9, 10},
				ApprovalRequired: []int{9, 10},
				Deliverables:     []string{"Security Review Passed", "Test Coverage Met", "UAT Sign-off"},
			},
		},
		AgentMapping: map[string]AgentTeam{
			"planner":    {Team: 2, Roles: []string{"Solution Architect"}, Phase: "Phase 1"},
			"architect":  {Team: 2, Roles: []string{"Chief Architect", "Domain Architect"}, Phase: "Phase 1"},
			"infrastructure": {Team: 4, Roles: []string{"Cloud Architect", "IaC Engineer"}, Phase: "Phase 2"},
			"platform":   {Team: 5, Roles: []string{"CI/CD Architect", "Kubernetes Administrator"}, Phase: "Phase 2"},
			"backend":    {Team: 7, Roles: []string{"Senior Backend Engineer"}, Phase: "Phase 3"},
			"frontend":   {Team: 7, Roles: []string{"Senior Frontend Engineer", "Accessibility Expert"}, Phase: "Phase 3"},
			"security":   {Team: 9, Roles: []string{"Security Architect"}, Phase: "Phase 4"},
			"qa":         {Team: 10, Roles: []string{"QA Architect", "SDET"}, Phase: "Phase 4"},
			"sre":        {Team: 11, Roles: []string{"SRE Lead", "Observability Engineer"}, Phase: "Phase 5"},
			"ops":        {Team: 12, Roles: []string{"Release Manager", "NOC Analyst"}, Phase: "Phase 5"},
		},
	}, nil
}
