package mcp

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mark3labs/mcp-go/mcp"
)

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

// Helper function to cleanup test projects. Cleans both the CWD-relative
// .teams directory (where the Go manager writes when tests run from this
// package dir) and the repo-root .teams directory (where tests that exec
// team_manager.py with Dir=repoRoot write, e.g. TestIntegrationJSONParsing),
// plus timestamped backups the Python manager leaves behind, so repeated test
// runs don't accumulate artifacts.
func cleanupTestProject(t *testing.T, projectName string) {
	t.Helper()
	for _, dir := range []string{".teams", filepath.Join("..", "..", "..", ".teams")} {
		configPath := filepath.Join(dir, projectName+".json")
		if err := os.Remove(configPath); err != nil && !os.IsNotExist(err) {
			t.Logf("Failed to cleanup test project %s: %v", projectName, err)
		}
	}
	backups, err := filepath.Glob(filepath.Join("..", "..", "..", ".teams", "backups", projectName+"_*.json.gz"))
	if err == nil {
		for _, b := range backups {
			_ = os.Remove(b)
		}
	}
}

// TestValidateRoleName tests the role name whitelist validation (SEC-002)
