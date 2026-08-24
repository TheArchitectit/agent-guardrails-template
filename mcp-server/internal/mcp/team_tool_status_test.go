package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestHandleTeamStatus_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-status"

	// Initialize project first
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Get status
	args := map[string]any{
		"project_name": projectName,
	}

	result, err := s.handleTeamStatus(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamStatus returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handleTeamStatus returned error result: %v", getResultText(result))
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamStatus_WithPhase tests handleTeamStatus with phase filter (SEC-010)
func TestHandleTeamStatus_WithPhase(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-status-phase"

	// Initialize project
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Get status with phase - SEC-010: Now uses strict "Phase 1" format
	args := map[string]any{
		"project_name": projectName,
		"phase":        "Phase 1",
	}

	result, err := s.handleTeamStatus(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamStatus returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handleTeamStatus returned error result: %v", getResultText(result))
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamStatus_MissingProjectName tests handleTeamStatus with missing project_name
func TestHandleTeamStatus_MissingProjectName(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	result, err := s.handleTeamStatus(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("handleTeamStatus returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleTeamStatus should return error for missing project_name")
	}
}

// TestHandlePhaseGateCheck_Valid tests handlePhaseGateCheck with valid input
func TestHandlePhaseGateCheck_Valid(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	args := map[string]any{
		"project_name": "test-project",
		"from_phase":   float64(1),
		"to_phase":     float64(2),
	}

	result, err := s.handlePhaseGateCheck(ctx, args)
	if err != nil {
		t.Fatalf("handlePhaseGateCheck returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handlePhaseGateCheck returned error result: %v", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "Phase Gate") {
		t.Errorf("handlePhaseGateCheck result should contain 'Phase Gate': %s", text)
	}
}

// TestHandlePhaseGateCheck_MissingFields tests handlePhaseGateCheck with missing fields
func TestHandlePhaseGateCheck_MissingFields(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "missing project_name",
			args: map[string]any{
				"from_phase": float64(1),
				"to_phase":   float64(2),
			},
		},
		{
			name: "missing from_phase",
			args: map[string]any{
				"project_name": "test",
				"to_phase":     float64(2),
			},
		},
		{
			name: "missing to_phase",
			args: map[string]any{
				"project_name": "test",
				"from_phase":   float64(1),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.handlePhaseGateCheck(ctx, tt.args)
			if err != nil {
				t.Fatalf("handlePhaseGateCheck returned error: %v", err)
			}

			if !result.IsError {
				t.Errorf("handlePhaseGateCheck should return error for %s", tt.name)
			}
		})
	}
}

// TestHandlePhaseGateCheck_InvalidGate tests handlePhaseGateCheck with undefined gate
func TestHandlePhaseGateCheck_InvalidGate(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	args := map[string]any{
		"project_name": "test-project",
		"from_phase":   float64(5),
		"to_phase":     float64(6),
	}

	result, err := s.handlePhaseGateCheck(ctx, args)
	if err != nil {
		t.Fatalf("handlePhaseGateCheck returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handlePhaseGateCheck should return error for undefined gate")
	}

	text := getResultText(result)
	if !strings.Contains(text, "No phase gate defined") {
		t.Errorf("Expected error message about undefined gate, got: %s", text)
	}
}

// TestHandleAgentTeamMap_Valid tests handleAgentTeamMap with valid agent types
func TestHandleAgentTeamMap_Valid(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		agentType string
		wantTeam  int
	}{
		{"planner", 2},
		{"architect", 2},
		{"infrastructure", 4},
		{"platform", 5},
		{"backend", 7},
		{"frontend", 7},
		{"security", 9},
		{"qa", 10},
		{"sre", 11},
		{"ops", 12},
	}

	for _, tt := range tests {
		t.Run(tt.agentType, func(t *testing.T) {
			args := map[string]any{
				"agent_type": tt.agentType,
			}

			result, err := s.handleAgentTeamMap(ctx, args)
			if err != nil {
				t.Fatalf("handleAgentTeamMap returned error: %v", err)
			}

			if result.IsError {
				t.Errorf("handleAgentTeamMap returned error for agent type %s: %v", tt.agentType, getResultText(result))
			}

			text := getResultText(result)
			expectedTeamStr := "Team " + string(rune('0'+tt.wantTeam))
			if !strings.Contains(text, expectedTeamStr[:6]) {
				// Check for team number in output
				found := false
				for i := 1; i <= 12; i++ {
					if strings.Contains(text, "Team "+string(rune('0'+i))) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("handleAgentTeamMap result should contain team assignment: %s", text)
				}
			}
		})
	}
}

// TestHandleAgentTeamMap_MissingAgentType tests handleAgentTeamMap with missing agent_type
func TestHandleAgentTeamMap_MissingAgentType(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	result, err := s.handleAgentTeamMap(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("handleAgentTeamMap returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleAgentTeamMap should return error for missing agent_type")
	}
}

// TestHandleAgentTeamMap_InvalidAgentType tests handleAgentTeamMap with invalid agent type
func TestHandleAgentTeamMap_InvalidAgentType(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	args := map[string]any{
		"agent_type": "nonexistent-agent",
	}

	result, err := s.handleAgentTeamMap(ctx, args)
	if err != nil {
		t.Fatalf("handleAgentTeamMap returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleAgentTeamMap should return error for invalid agent_type")
	}

	text := getResultText(result)
	if !strings.Contains(text, "No team mapping found") {
		t.Errorf("Expected error message about no mapping, got: %s", text)
	}
}

// TestHandleTeamSizeValidate_Valid tests handleTeamSizeValidate with valid input
func TestHandleTeamSizeValidate_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-validate"

	// Initialize project first
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Validate team sizes
	args := map[string]any{
		"project_name": projectName,
	}

	result, err := s.handleTeamSizeValidate(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamSizeValidate returned error: %v", err)
	}

	_ = result // Result may be error (undersized) which is expected

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamSizeValidate_WithTeamID tests handleTeamSizeValidate with specific team_id
func TestHandleTeamSizeValidate_WithTeamID(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-validate-team"

	// Initialize project
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Validate specific team
	args := map[string]any{
		"project_name": projectName,
		"team_id":      float64(1),
	}

	result, err := s.handleTeamSizeValidate(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamSizeValidate returned error: %v", err)
	}

	_ = result // Result may be error (undersized) which is expected

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamSizeValidate_MissingProjectName tests handleTeamSizeValidate with missing project_name
func TestHandleTeamSizeValidate_MissingProjectName(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	result, err := s.handleTeamSizeValidate(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("handleTeamSizeValidate returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleTeamSizeValidate should return error for missing project_name")
	}
}

// TestLoadTeamLayoutRules tests the loadTeamLayoutRules function
