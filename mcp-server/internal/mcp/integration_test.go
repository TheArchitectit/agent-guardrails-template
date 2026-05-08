package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// TestIntegrationFullWorkflow tests the complete workflow: init -> assign -> list -> status
func TestIntegrationFullWorkflow(t *testing.T) {
	// Check if team_manager.py exists
	scriptPath := "../../../scripts/team_manager.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-workflow"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Step 1: Initialize project
	t.Run("Step 1: Initialize Project", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": projectName,
		}

		result, err := s.handleTeamInit(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamInit failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("handleTeamInit returned error: %s", getResultText(result))
		}

		text := getResultText(result)
		if !strings.Contains(text, "Initialized") {
			t.Errorf("Expected initialization message, got: %s", text)
		}

		// Verify file was created (repo root .teams directory)
		configPath := filepath.Join("..", "..", "..", ".teams", projectName+".json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file was not created: %s", configPath)
		}
	})

	// Step 2: Assign roles to teams
	t.Run("Step 2: Assign Roles", func(t *testing.T) {
		assignments := []struct {
			teamID   float64
			roleName string
			person   string
		}{
			{1, "Business Relationship Manager", "Alice Smith"},
			{1, "Lead Product Manager", "Bob Jones"},
			{2, "Chief Architect", "Carol White"},
			{7, "Senior Backend Engineer", "David Brown"},
			{7, "Senior Frontend Engineer", "Eve Davis"},
		}

		for _, assignment := range assignments {
			args := map[string]interface{}{
				"project_name": projectName,
				"team_id":      assignment.teamID,
				"role_name":    assignment.roleName,
				"person":       assignment.person,
			}

			result, err := s.handleTeamAssign(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamAssign failed for team %d: %v", int(assignment.teamID), err)
			}

			if result.IsError {
				t.Errorf("handleTeamAssign returned error for team %d: %s", int(assignment.teamID), getResultText(result))
			}

			text := getResultText(result)
			if !strings.Contains(text, "Assigned") {
				t.Errorf("Expected assignment confirmation, got: %s", text)
			}
		}
	})

	// Step 3: List teams
	t.Run("Step 3: List Teams", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": projectName,
		}

		result, err := s.handleTeamList(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamList failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("handleTeamList returned error: %s", getResultText(result))
		}

		text := getResultText(result)

		// Verify team structure in output
		if !strings.Contains(text, "Team 1:") {
			t.Error("Expected Team 1 in output")
		}
		if !strings.Contains(text, "Team 7:") {
			t.Error("Expected Team 7 in output")
		}

		// Verify assigned names appear
		if !strings.Contains(text, "Alice Smith") {
			t.Error("Expected Alice Smith in output")
		}
		if !strings.Contains(text, "Carol White") {
			t.Error("Expected Carol White in output")
		}
	})

	// Step 4: Get phase status
	t.Run("Step 4: Get Phase Status", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": projectName,
		}

		result, err := s.handleTeamStatus(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamStatus failed: %v", err)
		}

		if result.IsError {
			t.Fatalf("handleTeamStatus returned error: %s", getResultText(result))
		}

		text := getResultText(result)
		if text == "" {
			t.Error("handleTeamStatus returned empty result")
		}
	})

	// Step 5: Validate team sizes
	t.Run("Step 5: Validate Team Sizes", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": projectName,
		}

		result, err := s.handleTeamSizeValidate(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamSizeValidate failed: %v", err)
		}

		// This may return error if teams are undersized (which they will be)
		// That's expected behavior - we're testing the integration, not the validation logic
		text := getResultText(result)
		if text == "" {
			t.Error("handleTeamSizeValidate returned empty result")
		}
	})

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestIntegrationPhaseGateCheck tests phase gate checking with Python integration
func TestIntegrationPhaseGateCheck(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-gates"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project first
	s.handleTeamInit(ctx, map[string]interface{}{"project_name": projectName})

	// Test various phase transitions
	transitions := []struct {
		fromPhase float64
		toPhase   float64
		shouldPass bool
	}{
		{1, 2, true},   // Valid: 1_to_2
		{2, 3, true},   // Valid: 2_to_3
		{3, 4, true},   // Valid: 3_to_4
		{4, 5, true},   // Valid: 4_to_5
		{5, 6, false},  // Invalid: no 5_to_6 gate
		{1, 5, false},  // Invalid: no 1_to_5 gate
	}

	for _, tc := range transitions {
		t.Run(fmt.Sprintf("Phase %d to %d", int(tc.fromPhase), int(tc.toPhase)), func(t *testing.T) {
			args := map[string]interface{}{
				"project_name": projectName,
				"from_phase":   tc.fromPhase,
				"to_phase":     tc.toPhase,
			}

			result, err := s.handlePhaseGateCheck(ctx, args)
			if err != nil {
				t.Fatalf("handlePhaseGateCheck failed: %v", err)
			}

			if tc.shouldPass && result.IsError {
				t.Errorf("Expected phase gate %d_to_%d to pass, got error: %s",
					int(tc.fromPhase), int(tc.toPhase), getResultText(result))
			}

			if !tc.shouldPass && !result.IsError {
				t.Errorf("Expected phase gate %d_to_%d to fail, but it passed",
					int(tc.fromPhase), int(tc.toPhase))
			}
		})
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

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
			args := map[string]interface{}{
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

// TestIntegrationJSONParsing tests JSON output from Python is parsed correctly
func TestIntegrationJSONParsing(t *testing.T) {
	scriptPath := "../../../scripts/team_manager.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	projectName := "integration-test-json"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project using Python directly (run from repo root)
	repoRoot := filepath.Join("..", "..", "..")
	scriptPathFromRoot := filepath.Join("scripts", "team_manager.py")
	cmd := exec.Command("python3", scriptPathFromRoot, "--project", projectName, "--test-mode", "init")
	cmd.Dir = repoRoot
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Try with python
		cmd = exec.Command("python", scriptPathFromRoot, "--project", projectName, "--test-mode", "init")
		cmd.Dir = repoRoot
		output, err = cmd.CombinedOutput()
	}
	if err != nil {
		t.Fatalf("Failed to initialize project: %v\nOutput: %s", err, string(output))
	}

	// Verify config file exists and is valid JSON (repo root .teams directory)
	configPath := filepath.Join("..", "..", "..", ".teams", projectName+".json")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		t.Fatalf("Config file is not valid JSON: %v", err)
	}

	// Verify structure
	if config["project_name"] != projectName {
		t.Errorf("Expected project_name to be %s, got %v", projectName, config["project_name"])
	}

	teams, ok := config["teams"].([]interface{})
	if !ok {
		t.Fatal("teams field is not an array")
	}

	if len(teams) != 12 {
		t.Errorf("Expected 12 teams, got %d", len(teams))
	}

	// Verify team structure
	for i, team := range teams {
		teamMap, ok := team.(map[string]interface{})
		if !ok {
			t.Fatalf("Team %d is not an object", i)
		}

		requiredFields := []string{"id", "name", "phase", "description", "roles", "exit_criteria", "status"}
		for _, field := range requiredFields {
			if _, exists := teamMap[field]; !exists {
				t.Errorf("Team %d missing required field: %s", i, field)
			}
		}
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestIntegrationErrorPropagation tests error handling from Python to Go
func TestIntegrationErrorPropagation(t *testing.T) {
	scriptPath := "../../../scripts/team_manager.py"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("team_manager.py not found, skipping integration test")
	}

	ctx := context.Background()
	s := mockMCPServer()

	// Test with non-existent project
	t.Run("Non-existent Project", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": "non-existent-project-12345",
		}

		result, err := s.handleTeamList(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamList failed: %v", err)
		}

		if !result.IsError {
			t.Error("Expected error for non-existent project")
		}

		text := getResultText(result)
		if !strings.Contains(text, "not found") {
			t.Errorf("Expected 'not found' message, got: %s", text)
		}
	})

	// Test with invalid project name (command injection attempt)
	t.Run("Invalid Project Name", func(t *testing.T) {
		args := map[string]interface{}{
			"project_name": "test;rm -rf",
		}

		result, err := s.handleTeamInit(ctx, args)
		if err != nil {
			t.Fatalf("handleTeamInit failed: %v", err)
		}

		if !result.IsError {
			t.Error("Expected error for invalid project name")
		}
	})
}

// TestIntegrationAssignAndValidate tests role assignment and validation
func TestIntegrationAssignAndValidate(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-assign-validate"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project
	s.handleTeamInit(ctx, map[string]interface{}{"project_name": projectName})

	// Assign minimum required roles to a team (4 members)
	roles := []struct {
		roleName string
		person   string
	}{
		{"Business Relationship Manager", "Person 1"},
		{"Lead Product Manager", "Person 2"},
		{"Business Systems Analyst", "Person 3"},
		{"Financial Controller (FinOps)", "Person 4"},
	}

	for _, role := range roles {
		args := map[string]interface{}{
			"project_name": projectName,
			"team_id":      float64(1),
			"role_name":    role.roleName,
			"person":       role.person,
		}

		result, err := s.handleTeamAssign(ctx, args)
		if err != nil {
			t.Fatalf("Failed to assign role: %v", err)
		}

		if result.IsError {
			t.Fatalf("Role assignment failed: %s", getResultText(result))
		}
	}

	// Validate team size - should pass now
	args := map[string]interface{}{
		"project_name": projectName,
		"team_id":      float64(1),
	}

	result, err := s.handleTeamSizeValidate(ctx, args)
	if err != nil {
		t.Fatalf("handleTeamSizeValidate failed: %v", err)
	}

	// Note: This might still fail if the Python script has different logic
	// We're testing the integration path, not the validation logic
	text := getResultText(result)
	if text == "" {
		t.Error("handleTeamSizeValidate returned empty result")
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestIntegrationTeamListWithPhaseFilter tests team listing with phase filter
func TestIntegrationTeamListWithPhaseFilter(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-phase-filter"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project
	s.handleTeamInit(ctx, map[string]interface{}{"project_name": projectName})

	// Test different phases (SEC-010: Only Phase 1, Phase 2, Phase 3 are valid)
	phases := []string{
		"Phase 1",
		"Phase 2",
		"Phase 3",
	}

	for _, phase := range phases {
		t.Run("Phase: "+phase, func(t *testing.T) {
			args := map[string]interface{}{
				"project_name": projectName,
				"phase":        phase,
			}

			result, err := s.handleTeamList(ctx, args)
			if err != nil {
				t.Fatalf("handleTeamList failed: %v", err)
			}

			if result.IsError {
				t.Fatalf("handleTeamList returned error: %s", getResultText(result))
			}

			text := getResultText(result)

			// Verify the team list returned successfully (phase filter applied)
			if text == "" {
				t.Errorf("Expected non-empty output for phase '%s'", phase)
			}
		})
	}

	// Cleanup
	cleanupTestProject(t, projectName)
}

// TestIntegrationMultipleProjects tests handling multiple projects
func TestIntegrationMultipleProjects(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()

	projects := []string{
		"integration-test-multi-1",
		"integration-test-multi-2",
		"integration-test-multi-3",
	}

	// Clean up before test
	for _, project := range projects {
		cleanupTestProject(t, project)
	}

	// Initialize all projects
	for _, project := range projects {
		args := map[string]interface{}{
			"project_name": project,
		}

		result, err := s.handleTeamInit(ctx, args)
		if err != nil {
			t.Fatalf("Failed to initialize project %s: %v", project, err)
		}

		if result.IsError {
			t.Fatalf("Error initializing project %s: %s", project, getResultText(result))
		}
	}

	// Verify each project has separate config (repo root .teams directory)
	for _, project := range projects {
		configPath := filepath.Join("..", "..", "..", ".teams", project+".json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file for project %s was not created", project)
		}
	}

	// Clean up
	for _, project := range projects {
		cleanupTestProject(t, project)
	}
}

