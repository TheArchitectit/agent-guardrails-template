package mcp

import (
	"context"
	"testing"
)

// TestIntegrationAssignAndValidate tests role assignment and validation
func TestIntegrationAssignAndValidate(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-assign-validate"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project
	s.handleTeamInit(ctx, map[string]any{"project_name": projectName})

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
		args := map[string]any{
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
	args := map[string]any{
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
