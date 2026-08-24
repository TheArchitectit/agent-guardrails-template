package mcp

import (
	"strings"
	"testing"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// mockMCPServer creates a minimal MCPServer for testing
func mockMCPServer() *MCPServer {
	return &MCPServer{
		sessions: make(map[string]*models.Session),
	}
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
