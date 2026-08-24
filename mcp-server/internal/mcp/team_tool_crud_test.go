package mcp

import (
	"context"
	"os"
	"strings"
	"testing"
)

func TestHandleTeamInit_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()

	// Use a unique project name for testing
	projectName := "test-project-init"
	args := map[string]any{
		"project_name": projectName,
	}

	result, err := s.handleTeamInit(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamInit returned error: %v", err)
	}

	if result == nil {
		t.Fatal("handleTeamInit returned nil result")
	}

	// Check that result is not an error
	if result.IsError {
		t.Errorf("handleTeamInit returned error result: %v", getResultText(result))
	}

	// Check for expected content
	text := getResultText(result)
	if !strings.Contains(text, "Initialized") && !strings.Contains(text, "Initialized project") {
		t.Errorf("handleTeamInit result does not contain expected content: %s", text)
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamInit_MissingProjectName tests handleTeamInit with missing project_name
func TestHandleTeamInit_MissingProjectName(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "nil args",
			args: nil,
		},
		{
			name: "empty args",
			args: map[string]any{},
		},
		{
			name: "empty project_name",
			args: map[string]any{"project_name": ""},
		},
		{
			name: "wrong type",
			args: map[string]any{"project_name": 123},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.handleTeamInit(ctx, tt.args)
			if err != nil {
				t.Fatalf("handleTeamInit returned error: %v", err)
			}

			if result == nil {
				t.Fatal("handleTeamInit returned nil result")
			}

			if !result.IsError {
				t.Error("handleTeamInit should return error result for invalid input")
			}

			text := getResultText(result)
			if !strings.Contains(text, "project_name is required") {
				t.Errorf("handleTeamInit error should mention 'project_name is required', got: %s", text)
			}
		})
	}
}

// TestHandleTeamInit_InvalidProjectName tests handleTeamInit with invalid project names
func TestHandleTeamInit_InvalidProjectName(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name    string
		project string
	}{
		{
			name:    "with spaces",
			project: "invalid project",
		},
		{
			name:    "with semicolon",
			project: "project;rm -rf",
		},
		{
			name:    "too long",
			project: strings.Repeat("a", 65),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{"project_name": tt.project}
			result, err := s.handleTeamInit(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamInit returned error: %v", err)
			}

			if !result.IsError {
				t.Error("handleTeamInit should return error result for invalid project name")
			}
		})
	}
}

// TestHandleTeamList_Valid tests handleTeamList with valid input
func TestHandleTeamList_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-list"

	// Initialize project first
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Now list teams
	args := map[string]any{
		"project_name": projectName,
	}

	result, err := s.handleTeamList(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamList returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handleTeamList returned error result: %v", getResultText(result))
	}

	text := getResultText(result)
	if text == "" {
		t.Error("handleTeamList returned empty result")
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamList_MissingProjectName tests handleTeamList with missing project_name
func TestHandleTeamList_MissingProjectName(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	result, err := s.handleTeamList(ctx, map[string]any{})
	if err != nil {
		t.Fatalf("handleTeamList returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleTeamList should return error for missing project_name")
	}
}

// TestHandleTeamList_WithPhaseFilter tests handleTeamList with phase filter (SEC-010)
func TestHandleTeamList_WithPhaseFilter(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-list-phase"

	// Initialize project
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// List with phase filter - SEC-010: Now uses strict "Phase 1" format
	args := map[string]any{
		"project_name": projectName,
		"phase":        "Phase 1",
	}

	result, err := s.handleTeamList(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamList returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handleTeamList returned error result: %v", getResultText(result))
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamAssign_Valid tests handleTeamAssign with valid input
func TestHandleTeamAssign_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-assign"

	// Initialize project first
	initArgs := map[string]any{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Assign role
	args := map[string]any{
		"project_name": projectName,
		"team_id":      float64(1),
		"role_name":    "Business Relationship Manager",
		"person":       "John Doe",
	}

	result, err := s.handleTeamAssign(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamAssign returned error: %v", err)
	}

	if result.IsError {
		t.Errorf("handleTeamAssign returned error result: %v", getResultText(result))
	}

	text := getResultText(result)
	if !strings.Contains(text, "Assigned") && !strings.Contains(text, "John Doe") {
		t.Errorf("handleTeamAssign result does not contain expected content: %s", text)
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestHandleTeamAssign_MissingFields tests handleTeamAssign with missing required fields
func TestHandleTeamAssign_MissingFields(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name string
		args map[string]any
	}{
		{
			name: "missing project_name",
			args: map[string]any{
				"team_id":   float64(1),
				"role_name": "Test Role",
				"person":    "Test Person",
			},
		},
		{
			name: "missing team_id",
			args: map[string]any{
				"project_name": "test",
				"role_name":    "Test Role",
				"person":       "Test Person",
			},
		},
		{
			name: "missing role_name",
			args: map[string]any{
				"project_name": "test",
				"team_id":      float64(1),
				"person":       "Test Person",
			},
		},
		{
			name: "missing person",
			args: map[string]any{
				"project_name": "test",
				"team_id":      float64(1),
				"role_name":    "Test Role",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := s.handleTeamAssign(ctx, tt.args)
			if err != nil {
				t.Fatalf("handleTeamAssign returned error: %v", err)
			}

			if !result.IsError {
				t.Errorf("handleTeamAssign should return error for %s", tt.name)
			}
		})
	}
}

// TestHandleTeamAssign_InvalidTeamID tests handleTeamAssign with invalid team_id
func TestHandleTeamAssign_InvalidTeamID(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name   string
		teamID float64
	}{
		{
			name:   "team_id zero",
			teamID: 0,
		},
		{
			name:   "team_id negative",
			teamID: -1,
		},
		{
			name:   "team_id too high",
			teamID: 13,
		},
		{
			name:   "team_id 99",
			teamID: 99,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]any{
				"project_name": "test-project",
				"team_id":      tt.teamID,
				"role_name":    "Test Role",
				"person":       "Test Person",
			}
			result, err := s.handleTeamAssign(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamAssign returned error: %v", err)
			}

			if !result.IsError {
				t.Error("handleTeamAssign should return error for invalid team_id")
			}

			text := getResultText(result)
			if !strings.Contains(text, "team_id must be between 1 and 12") {
				t.Errorf("Expected error message about team_id range, got: %s", text)
			}
		})
	}
}

// TestHandleTeamStatus_Valid tests handleTeamStatus with valid input
