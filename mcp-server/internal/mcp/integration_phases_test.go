package mcp

import (
	"context"
	"fmt"
	"testing"
)

// TestIntegrationPhaseGateCheck tests phase gate checking with Python integration
func TestIntegrationPhaseGateCheck(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-gates"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project first
	s.handleTeamInit(ctx, map[string]any{"project_name": projectName})

	// Test various phase transitions
	transitions := []struct {
		fromPhase  float64
		toPhase    float64
		shouldPass bool
	}{
		{1, 2, true},  // Valid: 1_to_2
		{2, 3, true},  // Valid: 2_to_3
		{3, 4, true},  // Valid: 3_to_4
		{4, 5, true},  // Valid: 4_to_5
		{5, 6, false}, // Invalid: no 5_to_6 gate
		{1, 5, false}, // Invalid: no 1_to_5 gate
	}

	for _, tc := range transitions {
		t.Run(fmt.Sprintf("Phase %d to %d", int(tc.fromPhase), int(tc.toPhase)), func(t *testing.T) {
			args := map[string]any{
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

// TestIntegrationTeamListWithPhaseFilter tests team listing with phase filter
func TestIntegrationTeamListWithPhaseFilter(t *testing.T) {
	ctx := context.Background()
	s := mockMCPServer()
	projectName := "integration-test-phase-filter"

	// Clean up before test
	cleanupTestProject(t, projectName)

	// Initialize project
	s.handleTeamInit(ctx, map[string]any{"project_name": projectName})

	// Test different phases (SEC-010: Only Phase 1, Phase 2, Phase 3 are valid)
	phases := []string{
		"Phase 1",
		"Phase 2",
		"Phase 3",
	}

	for _, phase := range phases {
		t.Run("Phase: "+phase, func(t *testing.T) {
			args := map[string]any{
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
