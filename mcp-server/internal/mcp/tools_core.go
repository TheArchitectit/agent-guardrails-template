package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
	"github.com/thearchitectit/guardrail-mcp/internal/validation"
)

// validationResult is a shared shape for core guardrail validation responses.
type validationResult struct {
	Valid      bool                   `json:"valid"`
	Violations []validation.Violation `json:"violations"`
	CheckedAt  string                 `json:"checked_at"`
	Command    string                 `json:"command,omitempty"`
	FilePath   string                 `json:"file_path,omitempty"`
	WasRead    bool                   `json:"was_read,omitempty"`
	ReadAt     string                 `json:"read_at,omitempty"`
	Message    string                 `json:"message,omitempty"`
}

// handleValidateBash validates a bash command against prevention rules.
func (s *MCPServer) handleValidateBash(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	command, _ := args["command"].(string)
	workingDir, _ := args["working_dir"].(string)

	if command == "" {
		return errorResult(`{"valid":false,"violations":[],"message":"command is required"}`), nil
	}

	if s.validator == nil {
		return errorResult(`{"valid":false,"violations":[],"message":"validation engine not configured"}`), nil
	}

	violations, err := s.validator.ValidateBash(ctx, command)
	if err != nil {
		return errorResult(fmt.Sprintf(`{"valid":false,"violations":[],"message":"validation failed: %s"}`, err.Error())), nil
	}

	result := validationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
		CheckedAt:  time.Now().Format(time.RFC3339),
		Command:    command,
	}
	result.Message = fmt.Sprintf("Bash validation for working_dir=%q", workingDir)

	return jsonToolResult(result, len(violations) > 0)
}

// handleValidateFileEdit validates a file edit against prevention rules and
// confirms the file was read before editing in the current session.
func (s *MCPServer) handleValidateFileEdit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, _ := args["file_path"].(string)
	oldString, _ := args["old_string"].(string)
	newString, _ := args["new_string"].(string)

	if filePath == "" {
		return errorResult(`{"valid":false,"violations":[],"message":"file_path is required"}`), nil
	}

	if s.validator == nil {
		return errorResult(`{"valid":false,"violations":[],"message":"validation engine not configured"}`), nil
	}

	// Determine the session token from args (optional). The validator uses it
	// to confirm the file was read before editing when a store is configured.
	sessionToken, _ := args["session_token"].(string)
	if sessionToken == "" {
		if tok, ok := args["session_id"].(string); ok {
			sessionToken = tok
		}
	}

	editContent := newString
	if editContent == "" {
		editContent = oldString
	}

	violations, err := s.validator.ValidateFileEdit(ctx, filePath, editContent, sessionToken)
	if err != nil {
		return errorResult(fmt.Sprintf(`{"valid":false,"violations":[],"message":"validation failed: %s"}`, err.Error())), nil
	}

	// Re-check read-before-edit explicitly when the file read store exists and a
	// session token was provided (defensive: engine may skip when no session).
	wasRead := true
	if sessionToken != "" && s.fileReadStore != nil {
		verification, verr := s.validator.VerifyFileRead(ctx, sessionToken, filePath)
		if verr == nil {
			wasRead = verification.WasRead
		}
	}

	result := validationResult{
		Valid:      len(violations) == 0 && wasRead,
		Violations: violations,
		CheckedAt:  time.Now().Format(time.RFC3339),
		FilePath:   filePath,
		WasRead:    wasRead,
	}

	if !wasRead {
		result.Message = "File must be read before editing"
		result.Valid = false
	}

	return jsonToolResult(result, !result.Valid)
}

// handleValidateGitOperation validates a git operation against prevention rules.
func (s *MCPServer) handleValidateGitOperation(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	operation, _ := args["operation"].(string)
	gitArgsRaw, _ := args["args"].([]interface{})

	if operation == "" {
		return errorResult(`{"valid":false,"violations":[],"message":"operation is required"}`), nil
	}

	if s.validator == nil {
		return errorResult(`{"valid":false,"violations":[],"message":"validation engine not configured"}`), nil
	}

	// Build a representative git command string from operation + args.
	parts := []string{"git", operation}
	for _, a := range gitArgsRaw {
		if str, ok := a.(string); ok {
			parts = append(parts, str)
		}
	}
	command := strings.Join(parts, " ")

	violations, err := s.validator.ValidateGit(ctx, command)
	if err != nil {
		return errorResult(fmt.Sprintf(`{"valid":false,"violations":[],"message":"validation failed: %s"}`, err.Error())), nil
	}

	result := validationResult{
		Valid:      len(violations) == 0,
		Violations: violations,
		CheckedAt:  time.Now().Format(time.RFC3339),
		Command:    command,
	}
	result.Message = fmt.Sprintf("Git operation %q validated", operation)

	return jsonToolResult(result, len(violations) > 0)
}

// preWorkCheckResult is the response shape for handlePreWorkCheck.
type preWorkCheckResult struct {
	Safe             bool                `json:"safe"`
	TaskDescription  string              `json:"task_description"`
	CheckedAt        string              `json:"checked_at"`
	KnownRegressions []preWorkRegression `json:"known_regressions"`
	Message          string              `json:"message"`
}

// preWorkRegression summarizes a known failure registry entry that matches the
// task being attempted.
type preWorkRegression struct {
	FailureID         string   `json:"failure_id"`
	Category          string   `json:"category"`
	Severity          string   `json:"severity"`
	ErrorMessage      string   `json:"error_message"`
	RootCause         string   `json:"root_cause"`
	AffectedFiles     []string `json:"affected_files"`
	RegressionPattern string   `json:"regression_pattern"`
}

// handlePreWorkCheck performs a pre-work safety check by scanning the failure
// registry for known regressions that match the task description.
func (s *MCPServer) handlePreWorkCheck(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	taskDescription, _ := args["task_description"].(string)

	if taskDescription == "" {
		return errorResult(`{"safe":false,"message":"task_description is required"}`), nil
	}

	if s.db == nil {
		return errorResult(`{"safe":false,"message":"database not configured"}`), nil
	}

	failStore := database.NewFailureStore(s.db)
	failures, err := failStore.GetActiveByFiles(ctx, []string{})
	if err != nil {
		return errorResult(fmt.Sprintf(`{"safe":false,"message":"failed to query failure registry: %s"}`, err.Error())), nil
	}

	// Extract candidate file paths referenced in the task description.
	candidateFiles := extractPathsFromText(taskDescription)

	// If the task references specific files, narrow to failures affecting them.
	if len(candidateFiles) > 0 {
		if narrowed, nerr := failStore.GetActiveByFiles(ctx, candidateFiles); nerr == nil {
			failures = narrowed
		}
	}

	matches := []preWorkRegression{}
	for _, f := range failures {
		// Match against the task description via regression pattern or keyword overlap.
		if failureMatchesTask(f, taskDescription) {
			matches = append(matches, preWorkRegression{
				FailureID:         f.FailureID,
				Category:          f.Category,
				Severity:          f.Severity,
				ErrorMessage:      f.ErrorMessage,
				RootCause:         f.RootCause,
				AffectedFiles:     models.ToStringSlice(f.AffectedFiles),
				RegressionPattern: f.RegressionPattern,
			})
		}
	}

	safe := len(matches) == 0
	message := "No known regressions match this task - safe to proceed"
	if !safe {
		message = fmt.Sprintf("%d known regression(s) match this task - review before proceeding", len(matches))
	}

	result := preWorkCheckResult{
		Safe:             safe,
		TaskDescription:  taskDescription,
		CheckedAt:        time.Now().Format(time.RFC3339),
		KnownRegressions: matches,
		Message:          message,
	}

	return jsonToolResult(result, !safe)
}

// extractPathsFromText finds file-path-like tokens in free-form text.
var pathTokenRegex = regexp.MustCompile(`(?:[A-Za-z0-9_\-./]+/)*[A-Za-z0-9_\-.]+\.[A-Za-z0-9]+`)

func extractPathsFromText(text string) []string {
	seen := make(map[string]bool)
	files := []string{}
	for _, m := range pathTokenRegex.FindAllString(text, -1) {
		if seen[m] {
			continue
		}
		seen[m] = true
		files = append(files, m)
	}
	return files
}

// failureMatchesTask returns true if a failure entry is plausibly relevant to
// the task description (via regression pattern match or keyword/substring overlap).
func failureMatchesTask(f models.FailureEntry, taskDescription string) bool {
	if f.RegressionPattern != "" {
		if re, err := regexp.Compile(f.RegressionPattern); err == nil && re.MatchString(taskDescription) {
			return true
		}
	}

	// Keyword overlap: low-cost heuristic when no usable regex.
	haystack := strings.ToLower(taskDescription)
	needles := []string{
		strings.ToLower(f.Category),
		strings.ToLower(f.ErrorMessage),
		strings.ToLower(f.RootCause),
	}
	for _, n := range needles {
		n = strings.TrimSpace(n)
		if n != "" && len(n) > 3 && strings.Contains(haystack, n) {
			return true
		}
	}

	for _, af := range models.ToStringSlice(f.AffectedFiles) {
		if af != "" && strings.Contains(taskDescription, af) {
			return true
		}
	}

	return false
}

// jsonToolResult marshals any result value to a CallToolResult with proper JSON
// escaping and the given error flag.
func jsonToolResult(result interface{}, isError bool) (*mcp.CallToolResult, error) {
	data, err := json.Marshal(result)
	if err != nil {
		return errorResult(fmt.Sprintf(`{"error":"failed to format result: %s"}`, err.Error())), nil
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: string(data)}},
		IsError: isError,
	}, nil
}
