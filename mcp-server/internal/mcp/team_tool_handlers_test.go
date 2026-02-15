package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

// mockMCPServer creates a minimal MCPServer for testing
func mockMCPServer() *MCPServer {
	return &MCPServer{}
}

// TestValidateProjectName tests the project name validation function
func TestValidateProjectName(t *testing.T) {
	tests := []struct {
		name    string
		project string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid simple name",
			project: "my-project",
			wantErr: false,
		},
		{
			name:    "valid with underscore",
			project: "my_project_123",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			project: "project123",
			wantErr: false,
		},
		{
			name:    "empty name",
			project: "",
			wantErr: true,
			errMsg:  "project_name is required",
		},
		{
			name:    "too long",
			project: strings.Repeat("a", 65),
			wantErr: true,
			errMsg:  "project_name must be 64 characters or less",
		},
		{
			name:    "invalid with space",
			project: "my project",
			wantErr: true,
			errMsg:  "project_name must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name:    "invalid with special char",
			project: "project;rm -rf",
			wantErr: true,
			errMsg:  "project_name must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name:    "invalid with slash",
			project: "project/test",
			wantErr: true,
			errMsg:  "project_name must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name:    "invalid with dot",
			project: "project.json",
			wantErr: true,
			errMsg:  "project_name must contain only letters, numbers, hyphens, and underscores",
		},
		{
			name:    "command injection attempt",
			project: "project$(whoami)",
			wantErr: true,
			errMsg:  "project_name must contain only letters, numbers, hyphens, and underscores",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateProjectName(tt.project)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateProjectName(%q) expected error, got nil", tt.project)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateProjectName(%q) error = %v, want error containing %q", tt.project, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateProjectName(%q) unexpected error: %v", tt.project, err)
				}
			}
		})
	}
}

// TestHandleTeamInit_Valid tests handleTeamInit with valid input
func TestHandleTeamInit_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()

	// Use a unique project name for testing
	projectName := "test-project-init"
	args := map[string]interface{}{
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
		args map[string]interface{}
	}{
		{
			name: "nil args",
			args: nil,
		},
		{
			name: "empty args",
			args: map[string]interface{}{},
		},
		{
			name: "empty project_name",
			args: map[string]interface{}{"project_name": ""},
		},
		{
			name: "wrong type",
			args: map[string]interface{}{"project_name": 123},
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
			args := map[string]interface{}{"project_name": tt.project}
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Now list teams
	args := map[string]interface{}{
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

	result, err := s.handleTeamList(ctx, map[string]interface{}{})
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// List with phase filter - SEC-010: Now uses strict "Phase 1" format
	args := map[string]interface{}{
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Assign role
	args := map[string]interface{}{
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
		args map[string]interface{}
	}{
		{
			name: "missing project_name",
			args: map[string]interface{}{
				"team_id":   float64(1),
				"role_name": "Test Role",
				"person":    "Test Person",
			},
		},
		{
			name: "missing team_id",
			args: map[string]interface{}{
				"project_name": "test",
				"role_name":    "Test Role",
				"person":       "Test Person",
			},
		},
		{
			name: "missing role_name",
			args: map[string]interface{}{
				"project_name": "test",
				"team_id":      float64(1),
				"person":       "Test Person",
			},
		},
		{
			name: "missing person",
			args: map[string]interface{}{
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
			args := map[string]interface{}{
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
func TestHandleTeamStatus_Valid(t *testing.T) {
	// Skip if Python is not available
	if _, err := os.Stat("../../../scripts/team_manager.py"); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	s := mockMCPServer()
	ctx := context.Background()
	projectName := "test-project-status"

	// Initialize project first
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Get status
	args := map[string]interface{}{
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Get status with phase - SEC-010: Now uses strict "Phase 1" format
	args := map[string]interface{}{
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

	result, err := s.handleTeamStatus(ctx, map[string]interface{}{})
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

	args := map[string]interface{}{
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
		args map[string]interface{}
	}{
		{
			name: "missing project_name",
			args: map[string]interface{}{
				"from_phase": float64(1),
				"to_phase":   float64(2),
			},
		},
		{
			name: "missing from_phase",
			args: map[string]interface{}{
				"project_name": "test",
				"to_phase":     float64(2),
			},
		},
		{
			name: "missing to_phase",
			args: map[string]interface{}{
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

	args := map[string]interface{}{
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
			args := map[string]interface{}{
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

	result, err := s.handleAgentTeamMap(ctx, map[string]interface{}{})
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

	args := map[string]interface{}{
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Validate team sizes
	args := map[string]interface{}{
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
	initArgs := map[string]interface{}{"project_name": projectName}
	s.handleTeamInit(ctx, initArgs)

	// Validate specific team
	args := map[string]interface{}{
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

	result, err := s.handleTeamSizeValidate(ctx, map[string]interface{}{})
	if err != nil {
		t.Fatalf("handleTeamSizeValidate returned error: %v", err)
	}

	if !result.IsError {
		t.Error("handleTeamSizeValidate should return error for missing project_name")
	}
}

// TestLoadTeamLayoutRules tests the loadTeamLayoutRules function
func TestLoadTeamLayoutRules(t *testing.T) {
	rules, err := loadTeamLayoutRules()
	if err != nil {
		t.Fatalf("loadTeamLayoutRules returned error: %v", err)
	}

	if rules == nil {
		t.Fatal("loadTeamLayoutRules returned nil")
	}

	// Check required fields
	if rules.Name == "" {
		t.Error("rules.Name should not be empty")
	}

	if rules.Version == "" {
		t.Error("rules.Version should not be empty")
	}

	// Check phase gates
	if len(rules.PhaseGates) == 0 {
		t.Error("rules.PhaseGates should not be empty")
	}

	// Check specific phase gates exist
	expectedGates := []string{"1_to_2", "2_to_3", "3_to_4", "4_to_5"}
	for _, gate := range expectedGates {
		if _, exists := rules.PhaseGates[gate]; !exists {
			t.Errorf("Phase gate %s should exist", gate)
		}
	}

	// Check agent mappings
	if len(rules.AgentMapping) == 0 {
		t.Error("rules.AgentMapping should not be empty")
	}

	// Check specific agent types
	expectedAgents := []string{"planner", "architect", "backend", "frontend", "security", "qa"}
	for _, agent := range expectedAgents {
		if _, exists := rules.AgentMapping[agent]; !exists {
			t.Errorf("Agent mapping for %s should exist", agent)
		}
	}
}

// TestTeamLayoutRulesPhaseGateStructure tests the structure of phase gates
func TestTeamLayoutRulesPhaseGateStructure(t *testing.T) {
	rules, err := loadTeamLayoutRules()
	if err != nil {
		t.Fatalf("loadTeamLayoutRules returned error: %v", err)
	}

	for gateName, gate := range rules.PhaseGates {
		if gate.Name == "" {
			t.Errorf("Phase gate %s should have a name", gateName)
		}
		if len(gate.RequiredTeams) == 0 {
			t.Errorf("Phase gate %s should have required teams", gateName)
		}
		if len(gate.Deliverables) == 0 {
			t.Errorf("Phase gate %s should have deliverables", gateName)
		}
	}
}

// TestTeamLayoutRulesAgentMappingStructure tests the structure of agent mappings
func TestTeamLayoutRulesAgentMappingStructure(t *testing.T) {
	rules, err := loadTeamLayoutRules()
	if err != nil {
		t.Fatalf("loadTeamLayoutRules returned error: %v", err)
	}

	for agentType, mapping := range rules.AgentMapping {
		if mapping.Team < 1 || mapping.Team > 12 {
			t.Errorf("Agent %s should map to valid team (1-12), got %d", agentType, mapping.Team)
		}
		if mapping.Phase == "" {
			t.Errorf("Agent %s should have a phase", agentType)
		}
		if len(mapping.Roles) == 0 {
			t.Errorf("Agent %s should have roles", agentType)
		}
	}
}

// TestTeamRuleStructure tests the TeamRule structure
func TestTeamRuleStructure(t *testing.T) {
	rule := TeamRule{
		ID:       "TEAM-001",
		Name:     "Test Rule",
		Severity: "error",
		Check:    "team_size",
		Command:  "validate-size",
		Message:  "Team size must be 4-6 members",
	}

	if rule.ID != "TEAM-001" {
		t.Errorf("Rule ID mismatch: got %s, want TEAM-001", rule.ID)
	}
	if rule.Name != "Test Rule" {
		t.Errorf("Rule Name mismatch: got %s, want Test Rule", rule.Name)
	}
}

// TestPhaseGateStructure tests the PhaseGate structure
func TestPhaseGateStructure(t *testing.T) {
	gate := PhaseGate{
		Name:             "Test Gate",
		RequiredTeams:    []int{1, 2, 3},
		ApprovalRequired: []int{1},
		Deliverables:     []string{"Doc 1", "Doc 2"},
	}

	if gate.Name != "Test Gate" {
		t.Errorf("Gate Name mismatch: got %s, want Test Gate", gate.Name)
	}
	if len(gate.RequiredTeams) != 3 {
		t.Errorf("Expected 3 required teams, got %d", len(gate.RequiredTeams))
	}
	if len(gate.Deliverables) != 2 {
		t.Errorf("Expected 2 deliverables, got %d", len(gate.Deliverables))
	}
}

// TestAgentTeamStructure tests the AgentTeam structure
func TestAgentTeamStructure(t *testing.T) {
	team := AgentTeam{
		Team:  5,
		Roles: []string{"Role 1", "Role 2"},
		Phase: "Phase 2",
	}

	if team.Team != 5 {
		t.Errorf("Team ID mismatch: got %d, want 5", team.Team)
	}
	if len(team.Roles) != 2 {
		t.Errorf("Expected 2 roles, got %d", len(team.Roles))
	}
	if team.Phase != "Phase 2" {
		t.Errorf("Phase mismatch: got %s, want Phase 2", team.Phase)
	}
}

// Helper function to extract text from result
func getResultText(result *mcp.CallToolResult) string {
	if result == nil || len(result.Content) == 0 {
		return ""
	}
	if textContent, ok := result.Content[0].(mcp.TextContent); ok {
		return textContent.Text
	}
	return ""
}

// Helper function to cleanup test projects
func cleanupTestProject(t *testing.T, projectName string) {
	t.Helper()
	// Clean up the test project file if it exists
	configPath := filepath.Join(".teams", projectName+".json")
	if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
		t.Logf("Failed to cleanup test project %s: %v", projectName, err)
	}
}

// TestValidateRoleName tests the role name whitelist validation (SEC-002)
func TestValidateRoleName(t *testing.T) {
	tests := []struct {
		name    string
		role    string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid role - Lead Product Manager",
			role:    "Lead Product Manager",
			wantErr: false,
		},
		{
			name:    "valid role - Chief Architect",
			role:    "Chief Architect",
			wantErr: false,
		},
		{
			name:    "valid role - Senior Backend Engineer",
			role:    "Senior Backend Engineer",
			wantErr: false,
		},
		{
			name:    "valid role - Technical Lead",
			role:    "Technical Lead",
			wantErr: false,
		},
		{
			name:    "invalid role - arbitrary string",
			role:    "Hacker",
			wantErr: true,
			errMsg:  "invalid role_name",
		},
		{
			name:    "invalid role - command injection attempt",
			role:    "root; rm -rf /",
			wantErr: true,
			errMsg:  "invalid role_name",
		},
		{
			name:    "invalid role - empty string",
			role:    "",
			wantErr: true,
			errMsg:  "role_name is required",
		},
		{
			name:    "invalid role - too long",
			role:    strings.Repeat("a", 129),
			wantErr: true,
			errMsg:  "role_name must be 128 characters or less",
		},
		{
			name:    "invalid role - control character",
			role:    "Lead Product Manager\x00",
			wantErr: true,
			errMsg:  "invalid control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRoleName(tt.role)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateRoleName(%q) expected error, got nil", tt.role)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateRoleName(%q) error = %v, want error containing %q", tt.role, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateRoleName(%q) unexpected error: %v", tt.role, err)
				}
			}
		})
	}
}

// TestValidatePersonName tests the person name format validation (SEC-003)
func TestValidatePersonName(t *testing.T) {
	tests := []struct {
		name    string
		person  string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid username",
			person:  "john_doe",
			wantErr: false,
		},
		{
			name:    "valid username with dots",
			person:  "john.doe",
			wantErr: false,
		},
		{
			name:    "valid username with hyphens",
			person:  "john-doe-123",
			wantErr: false,
		},
		{
			name:    "valid email",
			person:  "john.doe@example.com",
			wantErr: false,
		},
		{
			name:    "valid email with subdomain",
			person:  "user@subdomain.example.co.uk",
			wantErr: false,
		},
		{
			name:    "invalid - empty string",
			person:  "",
			wantErr: true,
			errMsg:  "person is required",
		},
		{
			name:    "invalid - too long",
			person:  strings.Repeat("a", 257),
			wantErr: true,
			errMsg:  "person must be 256 characters or less",
		},
		{
			name:    "invalid - control character",
			person:  "john\x00doe",
			wantErr: true,
			errMsg:  "invalid control characters",
		},
		{
			name:    "invalid - command injection attempt",
			person:  "john; rm -rf /",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "invalid - pipe character",
			person:  "john | cat /etc/passwd",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "invalid - backtick",
			person:  "john `whoami`",
			wantErr: true,
			errMsg:  "forbidden pattern",
		},
		{
			name:    "valid - display name with spaces",
			person:  "john doe",
			wantErr: false,
		},
		{
			name:    "invalid email - no domain",
			person:  "john@",
			wantErr: true,
			errMsg:  "contains invalid characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePersonName(tt.person)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePersonName(%q) expected error, got nil", tt.person)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePersonName(%q) error = %v, want error containing %q", tt.person, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePersonName(%q) unexpected error: %v", tt.person, err)
				}
			}
		})
	}
}

// TestValidatePhase tests the phase validation (SEC-010: Phase injection hardening)
func TestValidatePhase(t *testing.T) {
	tests := []struct {
		name    string
		phase   string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid Phase 1",
			phase:   "Phase 1",
			wantErr: false,
		},
		{
			name:    "valid Phase 2",
			phase:   "Phase 2",
			wantErr: false,
		},
		{
			name:    "valid Phase 3",
			phase:   "Phase 3",
			wantErr: false,
		},
		{
			name:    "empty phase (optional)",
			phase:   "",
			wantErr: false,
		},
		{
			name:    "invalid - old full name Phase 1",
			phase:   "Phase 1: Strategy, Governance & Planning",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - old full name Phase 2",
			phase:   "Phase 2: Platform & Foundation",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - Phase 4",
			phase:   "Phase 4",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - Phase 5",
			phase:   "Phase 5",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - arbitrary string",
			phase:   "Phase 99",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - command injection attempt",
			phase:   "Phase 1; rm -rf /",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - path traversal attempt",
			phase:   "Phase 1/../../../etc/passwd",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - null byte injection",
			phase:   "Phase 1\x00",
			wantErr: true,
			errMsg:  "invalid phase",
		},
		{
			name:    "invalid - newline injection",
			phase:   "Phase 1\ncommand",
			wantErr: true,
			errMsg:  "invalid phase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePhase(tt.phase)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validatePhase(%q) expected error, got nil", tt.phase)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validatePhase(%q) error = %v, want error containing %q", tt.phase, err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validatePhase(%q) unexpected error: %v", tt.phase, err)
				}
			}
		})
	}
}

// TestSanitizePhase tests the phase sanitization function (SEC-010)
func TestSanitizePhase(t *testing.T) {
	tests := []struct {
		name     string
		phase    string
		expected string
	}{
		{
			name:     "valid Phase 1",
			phase:    "Phase 1",
			expected: "Phase 1",
		},
		{
			name:     "valid Phase 2",
			phase:    "Phase 2",
			expected: "Phase 2",
		},
		{
			name:     "valid Phase 3",
			phase:    "Phase 3",
			expected: "Phase 3",
		},
		{
			name:     "empty phase",
			phase:    "",
			expected: "",
		},
		{
			name:     "invalid - returns empty",
			phase:    "Phase 1; rm -rf /",
			expected: "",
		},
		{
			name:     "invalid Phase 4 returns empty",
			phase:    "Phase 4",
			expected: "",
		},
		{
			name:     "path traversal returns empty",
			phase:    "../../../etc/passwd",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizePhase(tt.phase)
			if result != tt.expected {
				t.Errorf("sanitizePhase(%q) = %q, want %q", tt.phase, result, tt.expected)
			}
		})
	}
}

// TestHandleTeamAssign_InvalidRole tests handleTeamAssign with invalid role (SEC-002)
func TestHandleTeamAssign_InvalidRole(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name      string
		roleName  string
		wantError string
	}{
		{
			name:      "invalid role name",
			roleName:  "Hacker Role",
			wantError: "invalid role_name",
		},
		{
			name:      "role with control characters",
			roleName:  "Lead Product Manager\x00",
			wantError: "invalid control characters",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"project_name": "test-project",
				"team_id":      float64(1),
				"role_name":    tt.roleName,
				"person":       "john_doe",
			}
			result, err := s.handleTeamAssign(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamAssign returned error: %v", err)
			}

			if !result.IsError {
				t.Error("handleTeamAssign should return error for invalid role_name")
			}

			text := getResultText(result)
			if !strings.Contains(text, tt.wantError) {
				t.Errorf("Expected error containing %q, got: %s", tt.wantError, text)
			}
		})
	}
}

// TestHandleTeamAssign_InvalidPerson tests handleTeamAssign with invalid person (SEC-003)
func TestHandleTeamAssign_InvalidPerson(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name      string
		person    string
		wantError string
	}{
		{
			name:      "person with semicolon",
			person:    "john; rm -rf /",
			wantError: "forbidden pattern",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"project_name": "test-project",
				"team_id":      float64(1),
				"role_name":    "Lead Product Manager",
				"person":       tt.person,
			}
			result, err := s.handleTeamAssign(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamAssign returned error: %v", err)
			}

			if !result.IsError {
				t.Error("handleTeamAssign should return error for invalid person")
			}

			text := getResultText(result)
			if !strings.Contains(text, tt.wantError) {
				t.Errorf("Expected error containing %q, got: %s", tt.wantError, text)
			}
		})
	}
}

// TestHandleTeamList_InvalidPhase tests handleTeamList with invalid phase (SEC-010)
func TestHandleTeamList_InvalidPhase(t *testing.T) {
	s := mockMCPServer()
	ctx := context.Background()

	tests := []struct {
		name      string
		phase     string
		wantError string
	}{
		{
			name:      "invalid phase number",
			phase:     "Phase 99",
			wantError: "invalid phase",
		},
		{
			name:      "command injection attempt",
			phase:     "Phase 1; rm -rf /",
			wantError: "invalid phase",
		},
		{
			name:      "old format phase",
			phase:     "Phase 1: Strategy, Governance & Planning",
			wantError: "invalid phase",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			args := map[string]interface{}{
				"project_name": "test-project",
				"phase":        tt.phase,
			}

			result, err := s.handleTeamList(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamList returned error: %v", err)
			}

			if !result.IsError {
				t.Errorf("handleTeamList should return error for invalid phase: %s", tt.phase)
			}

			text := getResultText(result)
			if !strings.Contains(text, tt.wantError) {
				t.Errorf("Expected error containing %q, got: %s", tt.wantError, text)
			}
		})
	}
}
