package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionDefaults(t *testing.T) {
	if version != "dev" {
		t.Errorf("version = %q, want %q (ldflags should set it at build time)", version, "dev")
	}
	if buildTime != "unknown" {
		t.Errorf("buildTime = %q, want %q", buildTime, "unknown")
	}
	if gitCommit != "unknown" {
		t.Errorf("gitCommit = %q, want %q", gitCommit, "unknown")
	}
}

func TestGetTeamManagerPath_EnvOverride(t *testing.T) {
	t.Setenv("TEAM_MANAGER_PATH", "/custom/team_manager.py")
	if got := getTeamManagerPath(); got != "/custom/team_manager.py" {
		t.Errorf("getTeamManagerPath() = %q, want env override", got)
	}
}

func TestGetTeamManagerPath_Default(t *testing.T) {
	t.Setenv("TEAM_MANAGER_PATH", "")
	if got := getTeamManagerPath(); got != "scripts/team_manager.py" {
		t.Errorf("getTeamManagerPath() = %q, want default %q", got, "scripts/team_manager.py")
	}
}

func TestFindTeamManagerPath_InstalledLayout(t *testing.T) {
	prefix := t.TempDir()
	executable := filepath.Join(prefix, "bin", "team")
	backend := filepath.Join(prefix, "share", "agent-guardrails-template", "team-cli", "scripts", "team_manager.py")
	if err := os.MkdirAll(filepath.Dir(backend), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backend, []byte("# packaged backend\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := findTeamManagerPath(executable)
	if !ok {
		t.Fatal("findTeamManagerPath() did not find the install-owned backend")
	}
	if got != backend {
		t.Errorf("findTeamManagerPath() = %q, want %q", got, backend)
	}
}

func TestFindTeamManagerPath_SourceCheckoutLayout(t *testing.T) {
	root := t.TempDir()
	executable := filepath.Join(root, "cmd", "team-cli", "team")
	backend := filepath.Join(root, "scripts", "team_manager.py")
	if err := os.MkdirAll(filepath.Dir(backend), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(backend, []byte("# source backend\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got, ok := findTeamManagerPath(executable)
	if !ok {
		t.Fatal("findTeamManagerPath() did not preserve source-checkout lookup")
	}
	if got != backend {
		t.Errorf("findTeamManagerPath() = %q, want %q", got, backend)
	}
}

func TestPrettyPrintJSON_Valid(t *testing.T) {
	if err := prettyPrintJSON([]byte(`{"key": "value", "n": 1}`)); err != nil {
		t.Errorf("prettyPrintJSON(valid) error = %v, want nil", err)
	}
}

func TestPrettyPrintJSON_Invalid(t *testing.T) {
	if err := prettyPrintJSON([]byte(`not json`)); err == nil {
		t.Error("prettyPrintJSON(invalid) expected error, got nil")
	}
}

func TestListCmdRequiresProject(t *testing.T) {
	projectName = ""
	cmd := listCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "--project") {
		t.Errorf("list without --project: got err = %v, want error mentioning --project", err)
	}
}

func TestAssignCmdRequiresFlags(t *testing.T) {
	projectName = "test-project"
	cmd := assignCmd()
	cmd.SetArgs([]string{})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "required flag") {
		t.Errorf("assign without flags: got err = %v, want required-flag error", err)
	}
}

func TestTemplateCmdUnsupportedFormat(t *testing.T) {
	projectName = "test-project"
	cmd := templateCmd()
	cmd.SetArgs([]string{"--format", "xml"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported format") {
		t.Errorf("template --format xml: got err = %v, want unsupported-format error", err)
	}
}

func TestExportCmdUnsupportedFormat(t *testing.T) {
	projectName = "test-project"
	cmd := exportCmd()
	cmd.SetArgs([]string{"--format", "xml"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "unsupported export format") {
		t.Errorf("export --format xml: got err = %v, want unsupported-export-format error", err)
	}
}

func TestValidateOutputFormatBeforeCommand(t *testing.T) {
	tests := []struct {
		name       string
		format     string
		command    *cobra.Command
		wantErr    string
	}{
		{name: "text supported for mutation", format: "text", command: assignCmd()},
		{name: "status json supported", format: "json", command: statusCmd()},
		{name: "query json supported", format: "json", command: queryCmd()},
		{name: "history json supported", format: "json", command: historyCmd()},
		{name: "mutation json rejected", format: "json", command: assignCmd(), wantErr: "JSON output is not supported"},
		{name: "unknown format rejected", format: "xml", command: assignCmd(), wantErr: "unsupported output format"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output = tt.format
			err := validateOutputFormat(tt.command, nil)
			if tt.wantErr == "" && err != nil {
				t.Fatalf("validateOutputFormat() error = %v, want nil", err)
			}
			if tt.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tt.wantErr)) {
				t.Fatalf("validateOutputFormat() error = %v, want substring %q", err, tt.wantErr)
			}
		})
	}
	output = "text"
}
