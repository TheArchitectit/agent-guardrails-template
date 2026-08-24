package mcp

import (
	"context"
	"os"
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
		args := map[string]any{
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

		// Verify file was created (CWD-relative .teams directory, same location
		// the manager reads from when tests run from this package dir)
		configPath := filepath.Join(".teams", projectName+".json")
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
			args := map[string]any{
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
		args := map[string]any{
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

		// Verify team structure in output (list format prints team IDs and
		// assignment counts, e.g. "1  Business Relationship Manager  ...  Not Started (2/2 assigned)")
		if !strings.Contains(text, "1  ") {
			t.Error("Expected Team 1 in output")
		}
		if !strings.Contains(text, "7  ") {
			t.Error("Expected Team 7 in output")
		}

		// Verify assignments from Step 2 are reflected as assigned counts
		if !strings.Contains(text, "assigned") {
			t.Error("Expected assigned counts in output")
		}
	})

	// Step 4: Get phase status
	t.Run("Step 4: Get Phase Status", func(t *testing.T) {
		args := map[string]any{
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
		args := map[string]any{
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
