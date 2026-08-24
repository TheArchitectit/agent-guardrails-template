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

func (s *MCPServer) handleTeamDelete(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_delete")
	defer func() {
		metrics.DecrementTeamToolActive("team_delete")
		metrics.RecordTeamToolDuration("team_delete", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		metrics.RecordTeamToolError("team_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		metrics.RecordTeamToolError("team_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	// Check for confirmation
	confirmed := false
	if conf, ok := args["confirmed"].(bool); ok {
		confirmed = conf
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName)
	if err != nil {
		metrics.RecordTeamToolError("team_delete", "go_error")
		metrics.RecordTeamToolCall("team_delete", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.DeleteTeam(teamIDInt, confirmed); err != nil {
		metrics.RecordTeamToolDuration("team_delete", time.Since(goStart))
		// Check if this is just a confirmation required error
		if strings.Contains(err.Error(), "requires confirmation") {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "⚠️  Deletion requires confirmation. Set confirmed=true to proceed."}},
			}, nil
		}
		resultText := fmt.Sprintf("Error deleting team: %v", err)
		metrics.RecordTeamToolError("team_delete", "go_error")
		metrics.RecordTeamToolCall("team_delete", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_delete", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Deleted team %d from project '%s'", teamIDInt, projectName)
	metrics.RecordTeamToolCall("team_delete", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleProjectDelete deletes an entire project
func (s *MCPServer) handleProjectDelete(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("project_delete")
	defer func() {
		metrics.DecrementTeamToolActive("project_delete")
		metrics.RecordTeamToolDuration("project_delete", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("project_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("project_delete", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Check for confirmation
	confirmed := false
	if conf, ok := args["confirmed"].(bool); ok {
		confirmed = conf
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName)
	if err != nil {
		metrics.RecordTeamToolError("project_delete", "go_error")
		metrics.RecordTeamToolCall("project_delete", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.DeleteProject(confirmed); err != nil {
		metrics.RecordTeamToolDuration("project_delete", time.Since(goStart))
		// Check if this is just a confirmation required error
		if strings.Contains(err.Error(), "requires confirmation") {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "⚠️  Project deletion requires confirmation. Set confirmed=true to proceed."}},
			}, nil
		}
		resultText := fmt.Sprintf("Error deleting project: %v", err)
		metrics.RecordTeamToolError("project_delete", "go_error")
		metrics.RecordTeamToolCall("project_delete", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("project_delete", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Deleted project '%s'", projectName)
	metrics.RecordTeamToolCall("project_delete", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamHealth performs health check on team_manager.py
func (s *MCPServer) handleTeamHealth(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_health")
	defer func() {
		metrics.DecrementTeamToolActive("team_health")
		metrics.RecordTeamToolDuration("team_health", time.Since(start))
	}()

	projectName := "health-check"
	if name, ok := args["project_name"].(string); ok && name != "" {
		if err := validateProjectName(name); err != nil {
			metrics.RecordTeamToolError("team_health", "validation_error")
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
				IsError: true,
			}, nil
		}
		projectName = name
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName)
	if err != nil {
		// For health check, we still want to report status even if project doesn't exist
		health := map[string]any{
			"status":  "healthy",
			"project": projectName,
			"note":    "Project not initialized, but team manager is operational",
		}
		healthJSON, _ := json.MarshalIndent(health, "", "  ")
		resultText := fmt.Sprintf("✅ Team Manager Health:\n%s", string(healthJSON))
		metrics.RecordTeamToolCall("team_health", true)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
		}, nil
	}

	goStart := time.Now()
	health := mgr.Health()
	metrics.RecordTeamToolDuration("team_health", time.Since(goStart))

	healthJSON, err := json.MarshalIndent(health, "", "  ")
	if err != nil {
		resultText := fmt.Sprintf("Health check failed: %v", err)
		metrics.RecordTeamToolError("team_health", "go_error")
		metrics.RecordTeamToolCall("team_health", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}

	resultText := fmt.Sprintf("✅ Team Manager Health:\n%s", string(healthJSON))
	metrics.RecordTeamToolCall("team_health", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
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
			"planner":              {Team: 2, Roles: []string{"Solution Architect"}, Phase: "Phase 1"},
			"architect":            {Team: 2, Roles: []string{"Chief Architect", "Domain Architect"}, Phase: "Phase 1"},
			"infrastructure":       {Team: 4, Roles: []string{"Cloud Architect", "IaC Engineer"}, Phase: "Phase 2"},
			"platform":             {Team: 5, Roles: []string{"CI/CD Architect", "Kubernetes Administrator"}, Phase: "Phase 2"},
			"backend":              {Team: 7, Roles: []string{"Senior Backend Engineer"}, Phase: "Phase 3"},
			"frontend":             {Team: 7, Roles: []string{"Senior Frontend Engineer", "Accessibility Expert"}, Phase: "Phase 3"},
			"security":             {Team: 9, Roles: []string{"Security Architect"}, Phase: "Phase 4"},
			"security-engineer":    {Team: 9, Roles: []string{"DevSecOps Engineer", "Vulnerability Researcher"}, Phase: "Phase 4"},
			"qa":                   {Team: 10, Roles: []string{"QA Architect", "SDET"}, Phase: "Phase 4"},
			"performance-tester":   {Team: 10, Roles: []string{"Performance/Load Engineer"}, Phase: "Phase 4"},
			"accessibility-tester": {Team: 7, Roles: []string{"Accessibility (A11y) Expert"}, Phase: "Phase 3"},
			"ux-researcher":        {Team: 1, Roles: []string{"Business Systems Analyst", "Lead Product Manager"}, Phase: "Phase 1"},
			"sre":                  {Team: 11, Roles: []string{"SRE Lead", "Observability Engineer"}, Phase: "Phase 5"},
			"ops":                  {Team: 12, Roles: []string{"Release Manager", "NOC Analyst"}, Phase: "Phase 5"},
		},
	}, nil
}
