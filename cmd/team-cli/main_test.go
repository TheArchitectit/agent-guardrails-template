package main

import (
	"strings"
	"testing"
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
