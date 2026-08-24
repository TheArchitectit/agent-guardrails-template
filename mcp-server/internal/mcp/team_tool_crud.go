package mcp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/metrics"
	"github.com/thearchitectit/guardrail-mcp/internal/team"
)

func (s *MCPServer) handleTeamInit(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_init")
	defer func() {
		metrics.DecrementTeamToolActive("team_init")
		metrics.RecordTeamToolDuration("team_init", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_init", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_init", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Use Go implementation instead of Python
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_init", "go_error")
		metrics.RecordTeamToolCall("team_init", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.InitializeProject(); err != nil {
		metrics.RecordTeamToolDuration("team_init", time.Since(goStart))
		metrics.RecordTeamToolError("team_init", "go_error")
		metrics.RecordTeamToolCall("team_init", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error initializing project: %v", err)}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_init", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Initialized project '%s' with %d teams", projectName, len(team.StandardTeams))
	metrics.RecordTeamToolCall("team_init", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamList lists all teams and their status
func (s *MCPServer) handleTeamList(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_list")
	defer func() {
		metrics.DecrementTeamToolActive("team_list")
		metrics.RecordTeamToolDuration("team_list", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_list", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_list", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// Validate phase filter first (SEC-010) so invalid phases are rejected
	// before any project load attempt.
	if phase, ok := args["phase"].(string); ok && phase != "" {
		if err := validatePhase(phase); err != nil {
			metrics.RecordTeamToolError("team_list", "validation_error")
			metrics.RecordTeamToolCall("team_list", false)
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
				IsError: true,
			}, nil
		}
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_list", "go_error")
		metrics.RecordTeamToolCall("team_list", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.Load(); err != nil {
		metrics.RecordTeamToolError("team_list", "go_error")
		metrics.RecordTeamToolCall("team_list", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error loading project: %v", err)}},
			IsError: true,
		}, nil
	}

	var teams []team.Team
	if phase, ok := args["phase"].(string); ok && phase != "" {
		teams = mgr.GetTeamsByPhase(phase)
	} else {
		teams = mgr.GetAllTeams()
	}
	metrics.RecordTeamToolDuration("team_list", time.Since(goStart))

	// Build result
	var sb strings.Builder
	fmt.Fprintf(&sb, "\n📋 Teams for project '%s':\n\n", projectName)
	fmt.Fprintf(&sb, "%-5s %-35s %-30s %s\n", "ID", "Name", "Phase", "Status")
	sb.WriteString(strings.Repeat("-", 100) + "\n")

	for _, t := range teams {
		assignedCount := 0
		for _, role := range t.Roles {
			if role.AssignedTo != nil {
				assignedCount++
			}
		}
		statusStr := string(t.Status)
		if t.Status == team.TeamStatusNotStarted && assignedCount > 0 {
			statusStr = fmt.Sprintf("%s (%d/%d assigned)", t.Status, assignedCount, len(t.Roles))
		}
		fmt.Fprintf(&sb, "%-5d %-35s %-30s %s\n", t.ID, t.Name, t.Phase, statusStr)
	}

	resultText := sb.String()
	metrics.RecordTeamToolCall("team_list", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamAssign assigns a person to a role in a team
func (s *MCPServer) handleTeamAssign(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_assign")
	defer func() {
		metrics.DecrementTeamToolActive("team_assign")
		metrics.RecordTeamToolDuration("team_assign", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	// SEC-005: Check rate limit
	userID := "default" // Could extract from context/auth if available
	allowed, rateHeaders := globalRateLimiter.checkRateLimit(userID)
	if !allowed {
		metrics.RecordTeamToolError("team_assign", "rate_limit_exceeded")
		retryAfter := rateHeaders["X-RateLimit-Reset"]
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{
				Type: "text",
				Text: fmt.Sprintf("Error: Rate limit exceeded. Retry after %s", retryAfter),
			}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	roleName, ok := args["role_name"].(string)
	if !ok || roleName == "" {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: role_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateRoleName(roleName); err != nil {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	person, ok := args["person"].(string)
	if !ok || person == "" {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: person is required"}},
			IsError: true,
		}, nil
	}

	if err := validatePersonName(person); err != nil {
		metrics.RecordTeamToolError("team_assign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_assign", "go_error")
		metrics.RecordTeamToolCall("team_assign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.Load(); err != nil {
		metrics.RecordTeamToolError("team_assign", "go_error")
		metrics.RecordTeamToolCall("team_assign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error loading project: %v", err)}},
			IsError: true,
		}, nil
	}

	if err := mgr.AssignRole(teamIDInt, roleName, person); err != nil {
		metrics.RecordTeamToolDuration("team_assign", time.Since(goStart))
		metrics.RecordTeamToolError("team_assign", "go_error")
		metrics.RecordTeamToolCall("team_assign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error assigning role: %v", err)}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_assign", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Assigned '%s' to '%s' in Team %d (%s)",
		person, roleName, teamIDInt, team.StandardTeams[teamIDInt].Name)
	metrics.RecordTeamToolCall("team_assign", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamUnassign removes a person from a role in a team
func (s *MCPServer) handleTeamUnassign(ctx context.Context, args map[string]any) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_unassign")
	defer func() {
		metrics.DecrementTeamToolActive("team_unassign")
		metrics.RecordTeamToolDuration("team_unassign", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	// Validate team_id range (1-12)
	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	roleName, ok := args["role_name"].(string)
	if !ok || roleName == "" {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: role_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateRoleName(roleName); err != nil {
		metrics.RecordTeamToolError("team_unassign", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: %v", err)}},
			IsError: true,
		}, nil
	}

	// Use Go implementation
	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_unassign", "go_error")
		metrics.RecordTeamToolCall("team_unassign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.Load(); err != nil {
		metrics.RecordTeamToolError("team_unassign", "go_error")
		metrics.RecordTeamToolCall("team_unassign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error loading project: %v", err)}},
			IsError: true,
		}, nil
	}

	if err := mgr.UnassignRole(teamIDInt, roleName); err != nil {
		metrics.RecordTeamToolDuration("team_unassign", time.Since(goStart))
		metrics.RecordTeamToolError("team_unassign", "go_error")
		metrics.RecordTeamToolCall("team_unassign", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error unassigning role: %v", err)}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_unassign", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Unassigned role '%s' from Team %d (%s)",
		roleName, teamIDInt, team.StandardTeams[teamIDInt].Name)
	metrics.RecordTeamToolCall("team_unassign", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleTeamStart starts a team (marks as active)
// FUNC-010: Supports override for admin users
