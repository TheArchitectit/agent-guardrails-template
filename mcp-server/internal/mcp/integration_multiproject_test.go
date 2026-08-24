package mcp

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

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
		args := map[string]any{
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

	// Verify each project has separate config (CWD-relative .teams directory)
	for _, project := range projects {
		configPath := filepath.Join(".teams", project+".json")
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			t.Errorf("Config file for project %s was not created", project)
		}
	}

	// Clean up
	for _, project := range projects {
		cleanupTestProject(t, project)
	}
}
