package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/metrics"
	"github.com/thearchitectit/guardrail-mcp/internal/team"
)

// getSetupAgentsPath returns the absolute path to the setup_agents.py script
// at the repository root (<repo>/scripts/setup_agents.py).
func getSetupAgentsPath() string {
	_, filename, _, _ := runtime.Caller(0)
	dir := filepath.Dir(filename)
	// mcp-server/internal/mcp/ -> ../../../scripts/
	return filepath.Join(dir, "..", "..", "..", "scripts", "setup_agents.py")
}

// handleTeamConfigGet returns the on-disk team configuration for a project/team.
func (s *MCPServer) handleTeamConfigGet(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_config_get")
	defer func() {
		metrics.DecrementTeamToolActive("team_config_get")
		metrics.RecordTeamToolDuration("team_config_get", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_config_get", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_config_get", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamName, _ := args["team_name"].(string)

	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_config_get", "go_error")
		metrics.RecordTeamToolCall("team_config_get", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	configPath := mgr.GetConfigPath()

	var raw []byte
	if _, statErr := os.Stat(configPath); statErr == nil {
		raw, err = os.ReadFile(configPath)
		if err != nil {
			metrics.RecordTeamToolDuration("team_config_get", time.Since(goStart))
			metrics.RecordTeamToolError("team_config_get", "go_error")
			metrics.RecordTeamToolCall("team_config_get", false)
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error reading config: %v", err)}},
				IsError: true,
			}, nil
		}
	} else {
		// No config file yet — report an empty config structure.
		raw = []byte("{}")
	}
	metrics.RecordTeamToolDuration("team_config_get", time.Since(goStart))

	result := map[string]interface{}{
		"project_name": projectName,
		"team_name":    teamName,
		"config_path":  configPath,
		"config":       json.RawMessage(raw),
	}

	data, _ := json.MarshalIndent(result, "", "  ")
	metrics.RecordTeamToolCall("team_config_get", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(data)}},
	}, nil
}

// handleTeamConfigUpdate writes the provided config fragment into the team config file.
func (s *MCPServer) handleTeamConfigUpdate(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_config_update")
	defer func() {
		metrics.DecrementTeamToolActive("team_config_update")
		metrics.RecordTeamToolDuration("team_config_update", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_config_update", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_config_update", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamName, _ := args["team_name"].(string)

	configVal, ok := args["config"]
	if !ok || configVal == nil {
		metrics.RecordTeamToolError("team_config_update", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: config is required"}},
			IsError: true,
		}, nil
	}

	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_config_update", "go_error")
		metrics.RecordTeamToolCall("team_config_update", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	configPath := mgr.GetConfigPath()

	// Merge the provided config fragment into the existing config (if any).
	var merged map[string]interface{}
	if existing, statErr := os.ReadFile(configPath); statErr == nil && len(existing) > 0 {
		if err := json.Unmarshal(existing, &merged); err != nil {
			merged = map[string]interface{}{}
		}
	}
	if merged == nil {
		merged = map[string]interface{}{}
	}

	fragment, err := toMap(configVal)
	if err != nil {
		metrics.RecordTeamToolDuration("team_config_update", time.Since(goStart))
		metrics.RecordTeamToolError("team_config_update", "validation_error")
		metrics.RecordTeamToolCall("team_config_update", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: config must be a JSON object: %v", err)}},
			IsError: true,
		}, nil
	}

	for k, v := range fragment {
		merged[k] = v
	}

	out, err := json.MarshalIndent(merged, "", "  ")
	if err != nil {
		metrics.RecordTeamToolDuration("team_config_update", time.Since(goStart))
		metrics.RecordTeamToolError("team_config_update", "go_error")
		metrics.RecordTeamToolCall("team_config_update", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error marshaling config: %v", err)}},
			IsError: true,
		}, nil
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		metrics.RecordTeamToolDuration("team_config_update", time.Since(goStart))
		metrics.RecordTeamToolError("team_config_update", "go_error")
		metrics.RecordTeamToolCall("team_config_update", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating config dir: %v", err)}},
			IsError: true,
		}, nil
	}

	if err := os.WriteFile(configPath, out, 0644); err != nil {
		metrics.RecordTeamToolDuration("team_config_update", time.Since(goStart))
		metrics.RecordTeamToolError("team_config_update", "go_error")
		metrics.RecordTeamToolCall("team_config_update", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error writing config: %v", err)}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_config_update", time.Since(goStart))

	result := map[string]interface{}{
		"project_name": projectName,
		"team_name":    teamName,
		"config_path":  configPath,
		"updated":      true,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	metrics.RecordTeamToolCall("team_config_update", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(data)}},
	}, nil
}

// handleTeamRemove removes a team (or, given a team_id, deletes the team entirely).
// Follows the same confirmation logic as handleProjectDelete.
func (s *MCPServer) handleTeamRemove(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("team_remove")
	defer func() {
		metrics.DecrementTeamToolActive("team_remove")
		metrics.RecordTeamToolDuration("team_remove", time.Since(start))
	}()

	projectName, ok := args["project_name"].(string)
	if !ok || projectName == "" {
		metrics.RecordTeamToolError("team_remove", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: project_name is required"}},
			IsError: true,
		}, nil
	}

	if err := validateProjectName(projectName); err != nil {
		metrics.RecordTeamToolError("team_remove", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: err.Error()}},
			IsError: true,
		}, nil
	}

	teamID, ok := args["team_id"].(float64)
	if !ok {
		metrics.RecordTeamToolError("team_remove", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id is required"}},
			IsError: true,
		}, nil
	}

	teamIDInt := int(teamID)
	if teamIDInt < 1 || teamIDInt > 12 {
		metrics.RecordTeamToolError("team_remove", "validation_error")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "Error: team_id must be between 1 and 12"}},
			IsError: true,
		}, nil
	}

	confirmed := false
	if conf, ok := args["confirmed"].(bool); ok {
		confirmed = conf
	}

	mgr, err := team.NewManager(projectName, team.WithTestMode(true))
	if err != nil {
		metrics.RecordTeamToolError("team_remove", "go_error")
		metrics.RecordTeamToolCall("team_remove", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error creating manager: %v", err)}},
			IsError: true,
		}, nil
	}

	goStart := time.Now()
	if err := mgr.DeleteTeam(teamIDInt, confirmed); err != nil {
		metrics.RecordTeamToolDuration("team_remove", time.Since(goStart))
		if strings.Contains(err.Error(), "requires confirmation") {
			return &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: "⚠️  Team removal requires confirmation. Set confirmed=true to proceed."}},
			}, nil
		}
		resultText := fmt.Sprintf("Error removing team: %v", err)
		metrics.RecordTeamToolError("team_remove", "go_error")
		metrics.RecordTeamToolCall("team_remove", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
			IsError: true,
		}, nil
	}
	metrics.RecordTeamToolDuration("team_remove", time.Since(goStart))

	resultText := fmt.Sprintf("✅ Removed team %d from project '%s'", teamIDInt, projectName)
	metrics.RecordTeamToolCall("team_remove", true)
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: resultText}},
	}, nil
}

// handleInstallSkills installs or clones guardrails skill configurations by
// shelling out to scripts/setup_agents.py.
func (s *MCPServer) handleInstallSkills(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	start := time.Now()
	metrics.IncrementTeamToolActive("install_skills")
	defer func() {
		metrics.DecrementTeamToolActive("install_skills")
		metrics.RecordTeamToolDuration("install_skills", time.Since(start))
	}()

	scriptPath := getSetupAgentsPath()
	if _, statErr := os.Stat(scriptPath); statErr != nil {
		metrics.RecordTeamToolError("install_skills", "script_not_found")
		metrics.RecordTeamToolCall("install_skills", false)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Error: setup script not found at %s", scriptPath)}},
			IsError: true,
		}, nil
	}

	action, _ := args["action"].(string)

	cmdArgs := buildSetupAgentsArgs(args, action)

	// Resolve python interpreter (python3 preferred, fall back to python).
	pyBin := "python3"
	if _, err := exec.LookPath(pyBin); err != nil {
		pyBin = "python"
	}

	fullArgs := append([]string{scriptPath}, cmdArgs...)
	cmd := exec.CommandContext(ctx, pyBin, fullArgs...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	goStart := time.Now()
	runErr := cmd.Run()
	metrics.RecordTeamToolDuration("install_skills", time.Since(goStart))

	output := stdout.String()
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n"
		}
		output += "stderr: " + stderr.String()
	}

	if runErr != nil {
		if output == "" {
			output = runErr.Error()
		}
		metrics.RecordTeamToolError("install_skills", "script_error")
		metrics.RecordTeamToolCall("install_skills", false)
		result := map[string]interface{}{
			"action":  action,
			"success": false,
			"output":  output,
			"error":   runErr.Error(),
		}
		data, _ := json.MarshalIndent(result, "", "  ")
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(data)}},
			IsError: true,
		}, nil
	}

	metrics.RecordTeamToolCall("install_skills", true)
	result := map[string]interface{}{
		"action":  action,
		"success": true,
		"output":  output,
	}
	data, _ := json.MarshalIndent(result, "", "  ")
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(data)}},
	}, nil
}

// buildSetupAgentsArgs translates MCP tool args into setup_agents.py CLI flags.
func buildSetupAgentsArgs(args map[string]interface{}, action string) []string {
	var cmdArgs []string

	switch action {
	case "clone":
		cmdArgs = append(cmdArgs, "--clone")
		if p, ok := args["path"].(string); ok && p != "" {
			cmdArgs = append(cmdArgs, p)
		}
		if tp, ok := args["target_path"].(string); ok && tp != "" {
			cmdArgs = append(cmdArgs, "--target", tp)
		}
	case "install-skill", "skill":
		if skill, ok := args["skill"].(string); ok && skill != "" {
			cmdArgs = append(cmdArgs, "--install-skill", skill)
		}
		if tp, ok := args["target_path"].(string); ok && tp != "" {
			cmdArgs = append(cmdArgs, "--target", tp)
		}
		if plats := extractStringList(args["platforms"]); len(plats) > 0 {
			cmdArgs = append(cmdArgs, "--platform", strings.Join(plats, ","))
		}
		if mode, ok := args["mode"].(string); ok && mode != "" {
			cmdArgs = append(cmdArgs, "--mode", mode)
		}
		if dry, ok := args["dry_run"].(bool); ok && dry {
			cmdArgs = append(cmdArgs, "--dry-run")
		}
	case "list-skills":
		cmdArgs = append(cmdArgs, "--list-skills")
	case "list-platforms":
		cmdArgs = append(cmdArgs, "--list-platforms")
	case "install":
		cmdArgs = append(cmdArgs, "--install")
		if plats := extractStringList(args["platforms"]); len(plats) > 0 {
			cmdArgs = append(cmdArgs, "--platform", strings.Join(plats, ","))
		}
		if tp, ok := args["target_path"].(string); ok && tp != "" {
			cmdArgs = append(cmdArgs, "--target", tp)
		}
		if dry, ok := args["dry_run"].(bool); ok && dry {
			cmdArgs = append(cmdArgs, "--dry-run")
		}
	default:
		// No explicit action: honor the boolean list flags directly.
		if b, ok := args["list_skills"].(bool); ok && b {
			cmdArgs = append(cmdArgs, "--list-skills")
		}
		if b, ok := args["list_platforms"].(bool); ok && b {
			cmdArgs = append(cmdArgs, "--list-platforms")
		}
	}

	return cmdArgs
}

// extractStringList pulls a []interface{} (of strings) or []string out of args.
func extractStringList(v interface{}) []string {
	switch list := v.(type) {
	case []string:
		return list
	case []interface{}:
		out := make([]string, 0, len(list))
		for _, item := range list {
			if s, ok := item.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

// toMap coerces a JSON-style value into a map[string]interface{}.
func toMap(v interface{}) (map[string]interface{}, error) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, nil
	case string:
		var out map[string]interface{}
		if err := json.Unmarshal([]byte(m), &out); err != nil {
			return nil, err
		}
		return out, nil
	}
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	var out map[string]interface{}
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// handleAdvisorQuery is an alias for handleAdvisorConsult, which implements the
// actual advisor consultation logic (defined in advisor_tools.go).
func (s *MCPServer) handleAdvisorQuery(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	return s.handleAdvisorConsult(ctx, args)
}
