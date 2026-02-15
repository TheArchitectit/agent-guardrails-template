package mcp

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// TeamToolProvider provides MCP tools for team management
type TeamToolProvider struct {
	projectName string
}

// NewTeamToolProvider creates a new team tool provider
func NewTeamToolProvider(projectName string) *TeamToolProvider {
	return &TeamToolProvider{
		projectName: projectName,
	}
}

// GetTools returns the team management tools
func (t *TeamToolProvider) GetTools() []Tool {
	return []Tool{
		{
			Name:        "guardrail_team_init",
			Description: "Initialize team structure for a project",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project to initialize",
					},
				},
				"required": []string{"project_name"},
			},
		},
		{
			Name:        "guardrail_team_list",
			Description: "List all teams and their status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "Filter by phase (optional)",
					},
				},
				"required": []string{"project_name"},
			},
		},
		{
			Name:        "guardrail_team_assign",
			Description: "Assign a person to a role in a team",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"team_id": map[string]interface{}{
						"type":        "integer",
						"description": "Team ID (1-12)",
					},
					"role_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the role to assign",
					},
					"person": map[string]interface{}{
						"type":        "string",
						"description": "Name of the person to assign",
					},
				},
				"required": []string{"project_name", "team_id", "role_name", "person"},
			},
		},
		{
			Name:        "guardrail_team_start",
			Description: "Mark a team as active",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"team_id": map[string]interface{}{
						"type":        "integer",
						"description": "Team ID to start (1-12)",
					},
				},
				"required": []string{"project_name", "team_id"},
			},
		},
		{
			Name:        "guardrail_team_complete",
			Description: "Mark a team as completed",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"team_id": map[string]interface{}{
						"type":        "integer",
						"description": "Team ID to complete (1-12)",
					},
				},
				"required": []string{"project_name", "team_id"},
			},
		},
		{
			Name:        "guardrail_team_status",
			Description: "Get phase or project status",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"phase": map[string]interface{}{
						"type":        "string",
						"description": "Phase name (optional, shows all if omitted)",
					},
				},
				"required": []string{"project_name"},
			},
		},
		{
			Name:        "guardrail_phase_gate_check",
			Description: "Check if phase gate requirements are met",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"project_name": map[string]interface{}{
						"type":        "string",
						"description": "Name of the project",
					},
					"from_phase": map[string]interface{}{
						"type":        "integer",
						"description": "Starting phase (1-4)",
					},
					"to_phase": map[string]interface{}{
						"type":        "integer",
						"description": "Target phase (2-5)",
					},
				},
				"required": []string{"project_name", "from_phase", "to_phase"},
			},
		},
		{
			Name:        "guardrail_agent_team_map",
			Description: "Get the team assignment for an agent type",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"agent_type": map[string]interface{}{
						"type":        "string",
						"description": "Type of agent (planner, coder, security, etc.)",
					},
				},
				"required": []string{"agent_type"},
			},
		},
	}
}

// ExecuteTool executes a team management tool
func (t *TeamToolProvider) ExecuteTool(name string, args json.RawMessage) (string, error) {
	switch name {
	case "guardrail_team_init":
		var params struct {
			ProjectName string `json:"project_name"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamInit(params.ProjectName)

	case "guardrail_team_list":
		var params struct {
			ProjectName string `json:"project_name"`
			Phase       string `json:"phase,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamList(params.ProjectName, params.Phase)

	case "guardrail_team_assign":
		var params struct {
			ProjectName string `json:"project_name"`
			TeamID      int    `json:"team_id"`
			RoleName    string `json:"role_name"`
			Person      string `json:"person"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamAssign(params.ProjectName, params.TeamID, params.RoleName, params.Person)

	case "guardrail_team_start":
		var params struct {
			ProjectName string `json:"project_name"`
			TeamID      int    `json:"team_id"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamStart(params.ProjectName, params.TeamID)

	case "guardrail_team_complete":
		var params struct {
			ProjectName string `json:"project_name"`
			TeamID      int    `json:"team_id"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamComplete(params.ProjectName, params.TeamID)

	case "guardrail_team_status":
		var params struct {
			ProjectName string `json:"project_name"`
			Phase       string `json:"phase,omitempty"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeTeamStatus(params.ProjectName, params.Phase)

	case "guardrail_phase_gate_check":
		var params struct {
			ProjectName string `json:"project_name"`
			FromPhase   int    `json:"from_phase"`
			ToPhase     int    `json:"to_phase"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executePhaseGateCheck(params.ProjectName, params.FromPhase, params.ToPhase)

	case "guardrail_agent_team_map":
		var params struct {
			AgentType string `json:"agent_type"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("invalid arguments: %w", err)
		}
		return t.executeAgentTeamMap(params.AgentType)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

// Implementation methods

func (t *TeamToolProvider) executeTeamInit(projectName string) (string, error) {
	cmd := exec.Command("python", "scripts/team_manager.py", "--project", projectName, "init")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to initialize team: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executeTeamList(projectName, phase string) (string, error) {
	args := []string{"scripts/team_manager.py", "--project", projectName, "list"}
	if phase != "" {
		args = append(args, "--phase", phase)
	}
	cmd := exec.Command("python", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to list teams: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executeTeamAssign(projectName string, teamID int, roleName, person string) (string, error) {
	cmd := exec.Command("python", "scripts/team_manager.py", "--project", projectName, "assign",
		"--team", fmt.Sprintf("%d", teamID),
		"--role", roleName,
		"--person", person)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to assign role: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executeTeamStart(projectName string, teamID int) (string, error) {
	cmd := exec.Command("python", "scripts/team_manager.py", "--project", projectName, "start",
		"--team", fmt.Sprintf("%d", teamID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to start team: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executeTeamComplete(projectName string, teamID int) (string, error) {
	cmd := exec.Command("python", "scripts/team_manager.py", "--project", projectName, "complete",
		"--team", fmt.Sprintf("%d", teamID))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to complete team: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executeTeamStatus(projectName, phase string) (string, error) {
	args := []string{"scripts/team_manager.py", "--project", projectName, "status"}
	if phase != "" {
		args = append(args, "--phase", phase)
	}
	cmd := exec.Command("python", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to get status: %w\nOutput: %s", err, string(output))
	}
	return string(output), nil
}

func (t *TeamToolProvider) executePhaseGateCheck(projectName string, fromPhase, toPhase int) (string, error) {
	// Load team layout rules
	rules, err := t.loadTeamLayoutRules()
	if err != nil {
		return "", fmt.Errorf("failed to load team rules: %w", err)
	}

	// Map phases to gate names
	gateName := fmt.Sprintf("%d_to_%d", fromPhase, toPhase)
	gate, exists := rules.PhaseGates[gateName]
	if !exists {
		return "", fmt.Errorf("no phase gate defined from phase %d to phase %d", fromPhase, toPhase)
	}

	// Check project status
	status, err := t.getProjectStatus(projectName)
	if err != nil {
		return "", fmt.Errorf("failed to get project status: %w", err)
	}

	// Build response
	var response strings.Builder
	response.WriteString(fmt.Sprintf("# Phase Gate: %s\n\n", gate.Name))
	response.WriteString("**Required Teams:**\n")
	for _, teamID := range gate.RequiredTeams {
		team := status.GetTeam(teamID)
		if team != nil {
			icon := "✅"
			if team.Status != "completed" {
				icon = "❌"
			}
			response.WriteString(fmt.Sprintf("%s Team %d: %s (%s)\n", icon, teamID, team.Name, team.Status))
		}
	}

	response.WriteString("\n**Required Deliverables:**\n")
	for _, deliverable := range gate.Deliverables {
		response.WriteString(fmt.Sprintf("- [ ] %s\n", deliverable))
	}

	response.WriteString("\n**Status:** ")
	if status.IsPhaseGateComplete(fromPhase, toPhase) {
		response.WriteString("✅ **APPROVED** - Ready to proceed\n")
	} else {
		response.WriteString("❌ **BLOCKED** - Complete required teams and deliverables first\n")
	}

	return response.String(), nil
}

func (t *TeamToolProvider) executeAgentTeamMap(agentType string) (string, error) {
	// Load team layout rules
	rules, err := t.loadTeamLayoutRules()
	if err != nil {
		return "", fmt.Errorf("failed to load team rules: %w", err)
	}

	mapping, exists := rules.AgentMapping[agentType]
	if !exists {
		return "", fmt.Errorf("no team mapping found for agent type: %s", agentType)
	}

	return fmt.Sprintf(
		"# Agent Team Assignment\n\n"+
			"**Agent Type:** %s\n"+
			"**Assigned Team:** Team %d\n"+
			"**Phase:** %s\n"+
			"**Roles:** %s\n",
		agentType,
		mapping.Team,
		mapping.Phase,
		strings.Join(mapping.Roles, ", "),
	), nil
}

// Helper types and methods

type TeamLayoutRules struct {
	Name        string                `json:"name"`
	Version     string                `json:"version"`
	Description string                `json:"description"`
	AppliesTo   []string              `json:"applies_to"`
	Rules       []TeamRule            `json:"rules"`
	PhaseGates  map[string]PhaseGate  `json:"phase_gates"`
	AgentMapping map[string]AgentTeam `json:"agent_mapping"`
}

type TeamRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Severity    string `json:"severity"`
	Check       string `json:"check"`
	Command     string `json:"command"`
	Message     string `json:"message"`
	Trigger     string `json:"trigger,omitempty"`
	Patterns    []string `json:"patterns,omitempty"`
}

type PhaseGate struct {
	Name            string   `json:"name"`
	RequiredTeams   []int    `json:"required_teams"`
	ApprovalRequired []int   `json:"approval_required"`
	Deliverables    []string `json:"deliverables"`
}

type AgentTeam struct {
	Team   int      `json:"team"`
	Roles  []string `json:"roles"`
	Phase  string   `json:"phase"`
}

type ProjectStatus struct {
	Teams []TeamStatus
}

type TeamStatus struct {
	ID     int
	Name   string
	Status string
}

func (s *ProjectStatus) GetTeam(id int) *TeamStatus {
	for _, t := range s.Teams {
		if t.ID == id {
			return &t
		}
	}
	return nil
}

func (s *ProjectStatus) IsPhaseGateComplete(fromPhase, toPhase int) bool {
	// Simplified check - would need to load actual project data
	return false
}

func (t *TeamToolProvider) loadTeamLayoutRules() (*TeamLayoutRules, error) {
	// This would load from .guardrails/team-layout-rules.json
	// For now, return hardcoded rules
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
		},
		AgentMapping: map[string]AgentTeam{
			"planner": {Team: 2, Roles: []string{"Solution Architect"}, Phase: "Phase 1"},
			"coder":   {Team: 7, Roles: []string{"Senior Backend Engineer"}, Phase: "Phase 3"},
			"security": {Team: 9, Roles: []string{"Security Architect"}, Phase: "Phase 4"},
			"ops":     {Team: 11, Roles: []string{"SRE Lead"}, Phase: "Phase 5"},
		},
	}, nil
}

func (t *TeamToolProvider) getProjectStatus(projectName string) (*ProjectStatus, error) {
	// This would load actual project status from disk
	// For now return empty status
	return &ProjectStatus{Teams: []TeamStatus{}}, nil
}
