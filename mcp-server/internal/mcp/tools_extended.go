package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// handleValidateScope checks if a file path is within authorized scope
func (s *MCPServer) handleValidateScope(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, _ := args["file_path"].(string)
	scope, _ := args["authorized_scope"].(string)

	if filePath == "" {
		result := models.ScopeValidationResult{
			Valid:   false,
			Message: "file_path is required",
		}
		return buildToolResult(result, true)
	}

	if scope == "" {
		result := models.ScopeValidationResult{
			Valid:    true,
			Message:  "No scope restriction specified - file allowed",
			FilePath: filePath,
			Scope:    scope,
		}
		return buildToolResult(result, false)
	}

	// Clean paths for comparison
	cleanPath := filepath.Clean(filePath)
	cleanScope := filepath.Clean(scope)

	// Check if file is within scope
	isValid := strings.HasPrefix(cleanPath, cleanScope)

	var result models.ScopeValidationResult
	if isValid {
		result = models.ScopeValidationResult{
			Valid:    true,
			Message:  fmt.Sprintf("File %s is within authorized scope", filePath),
			FilePath: filePath,
			Scope:    scope,
		}
	} else {
		result = models.ScopeValidationResult{
			Valid:        false,
			Message:      fmt.Sprintf("File %s is OUTSIDE authorized scope %s", filePath, scope),
			FilePath:     filePath,
			Scope:        scope,
			OutsideScope: true,
		}
	}

	return buildToolResult(result, !isValid)
}

// handleValidateCommit validates a commit message against conventional commit format
func (s *MCPServer) handleValidateCommit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	message, _ := args["message"].(string)

	if message == "" {
		result := models.CommitValidationResult{
			Valid:   false,
			Message: "Commit message is required",
			Issues:  []string{"Empty commit message"},
		}
		return buildToolResult(result, true)
	}

	result := validateConventionalCommit(message)
	return buildToolResult(result, !result.Valid)
}

// validateConventionalCommit validates against conventional commit format
// Format: type(scope): description
func validateConventionalCommit(message string) models.CommitValidationResult {
	issues := []string{}

	// Valid conventional commit types
	validTypes := []string{"feat", "fix", "docs", "style", "refactor", "perf", "test", "chore", "build", "ci", "revert"}
	validTypesMap := make(map[string]bool)
	for _, t := range validTypes {
		validTypesMap[t] = true
	}

	// Pattern for conventional commit: type(scope): description
	// Scope is optional
	conventionalPattern := regexp.MustCompile(`^(\w+)(?:\(([^)]+)\))?!?: (.+)$`)

	// Check message length
	if len(message) > 72 {
		issues = append(issues, "Message exceeds 72 characters (consider using body for details)")
	}

	// Check for common issues
	if strings.HasSuffix(message, ".") {
		issues = append(issues, "Message should not end with a period")
	}

	// Check first word capitalization (should be lowercase for conventional commits)
	if len(message) > 0 && message[0] >= 'A' && message[0] <= 'Z' {
		issues = append(issues, "First word should be lowercase (type)")
	}

	// Match against conventional commit pattern
	matches := conventionalPattern.FindStringSubmatch(message)

	if matches == nil {
		// Not in conventional commit format
		return models.CommitValidationResult{
			Valid:           false,
			FormatCompliant: false,
			Issues:          append(issues, "Message does not follow conventional commit format: type(scope): description"),
			Message:         message,
		}
	}

	commitType := matches[1]
	scope := matches[2]
	description := matches[3]

	// Validate type
	if !validTypesMap[commitType] {
		issues = append(issues, fmt.Sprintf("Invalid type '%s' - must be one of: %s", commitType, strings.Join(validTypes, ", ")))
	}

	// Validate description
	if description == "" {
		issues = append(issues, "Description cannot be empty")
	}

	// Check description starts with lowercase (for non-proper nouns)
	if len(description) > 0 && description[0] >= 'A' && description[0] <= 'Z' {
		// This is a warning, not an error - might be a proper noun
		if !isProperNounStart(description) {
			issues = append(issues, "Description should start with lowercase (unless it's a proper noun)")
		}
	}

	valid := len(issues) == 0

	return models.CommitValidationResult{
		Valid:            valid,
		FormatCompliant:  true,
		Issues:           issues,
		Message:          message,
		ConventionalType: commitType,
		Scope:            scope,
	}
}

// isProperNounStart checks if description starts with what might be a proper noun
func isProperNounStart(description string) bool {
	// Common proper nouns in commit messages
	properNouns := []string{"API", "URL", "HTTP", "JSON", "XML", "SQL", "CSS", "HTML", "AWS", "GCP", "UI", "UX"}
	for _, noun := range properNouns {
		if strings.HasPrefix(description, noun) {
			return true
		}
	}
	return false
}

// handlePreventRegression checks failure registry for matching patterns
func (s *MCPServer) handlePreventRegression(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Extract file paths
	filesArg, _ := args["file_paths"].([]interface{})
	files := make([]string, 0, len(filesArg))
	for _, f := range filesArg {
		if str, ok := f.(string); ok {
			files = append(files, str)
		}
	}

	// Extract code content for pattern matching
	codeContent, _ := args["code_content"].(string)

	if len(files) == 0 && codeContent == "" {
		result := models.RegressionCheckResult{
			Matches: []models.RegressionMatch{},
			Checked: 0,
		}
		return buildToolResult(result, false)
	}

	// Query database for active failures affecting these files
	failStore := database.NewFailureStore(s.db)
	failures, err := failStore.GetActiveByFiles(ctx, files)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Failed to check failures: %v", err)}},
			IsError: true,
		}, nil
	}

	// Match failures against code content if provided
	matches := []models.RegressionMatch{}
	for _, failure := range failures {
		// Check if regression pattern matches code content
		if codeContent != "" && failure.RegressionPattern != "" {
			pattern, err := regexp.Compile(failure.RegressionPattern)
			if err == nil && pattern.MatchString(codeContent) {
				matches = append(matches, models.RegressionMatch{
					FailureID:         failure.FailureID,
					Category:          failure.Category,
					Severity:          failure.Severity,
					Message:           failure.ErrorMessage,
					RootCause:         failure.RootCause,
					RegressionPattern: failure.RegressionPattern,
					AffectedFiles:     models.ToStringSlice(failure.AffectedFiles),
				})
			}
		} else {
			// Include failure if it affects any of the files
			matches = append(matches, models.RegressionMatch{
				FailureID:         failure.FailureID,
				Category:          failure.Category,
				Severity:          failure.Severity,
				Message:           failure.ErrorMessage,
				RootCause:         failure.RootCause,
				RegressionPattern: failure.RegressionPattern,
				AffectedFiles:     models.ToStringSlice(failure.AffectedFiles),
			})
		}
	}

	result := models.RegressionCheckResult{
		Matches: matches,
		Checked: len(files),
	}

	return buildToolResult(result, len(matches) > 0)
}

// handleCheckTestProdSeparation verifies test/production environment isolation
func (s *MCPServer) handleCheckTestProdSeparation(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePath, _ := args["file_path"].(string)
	environment, _ := args["environment"].(string)

	if filePath == "" {
		result := models.TestProdSeparationResult{
			Valid:       false,
			Violations:  []string{"file_path is required"},
			Environment: environment,
		}
		return buildToolResult(result, true)
	}

	violations := []string{}

	// Read file content if it exists
	content := ""
	if data, err := os.ReadFile(filePath); err == nil {
		content = string(data)
	}

	switch environment {
	case "prod":
		// In prod code, check for test database usage
		if strings.Contains(content, "test_db") || strings.Contains(content, "test_database") {
			violations = append(violations, "Production code references test database")
		}
		if strings.Contains(content, "localhost:5433") || strings.Contains(content, "localhost:5434") {
			violations = append(violations, "Production code uses test database port")
		}
		// Check for test-only patterns
		if regexp.MustCompile(`testMode\s*=\s*true`).MatchString(content) {
			violations = append(violations, "Production code has test mode enabled")
		}

	case "test":
		// In test code, check for production credentials/patterns
		if strings.Contains(content, "prod_db") || strings.Contains(content, "production_database") {
			violations = append(violations, "Test code references production database")
		}
		// Check for hardcoded production URLs
		if regexp.MustCompile(`https?://api\.production\.`).MatchString(content) {
			violations = append(violations, "Test code contains production API URL")
		}
		// Check for real credentials (basic patterns)
		if regexp.MustCompile(`(?i)(aws_access_key_id|aws_secret_access_key)\s*=\s*["'][A-Z0-9]{20}["']`).MatchString(content) {
			violations = append(violations, "Test code may contain hardcoded AWS credentials")
		}
		// Check for production secrets
		if regexp.MustCompile(`(?i)production.*secret`).MatchString(content) {
			violations = append(violations, "Test code references production secrets")
		}

	default:
		violations = append(violations, fmt.Sprintf("Unknown environment: %s (expected 'test' or 'prod')", environment))
	}

	valid := len(violations) == 0

	result := models.TestProdSeparationResult{
		Valid:       valid,
		Violations:  violations,
		FilePath:    filePath,
		Environment: environment,
	}

	return buildToolResult(result, !valid)
}

// handleValidatePush validates git push safety conditions
func (s *MCPServer) handleValidatePush(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	branch, _ := args["branch"].(string)
	isForce, _ := args["is_force"].(bool)
	hasUnpushedCommits, _ := args["has_unpushed_commits"].(bool)

	warnings := []string{}
	canPush := true
	valid := true

	// Check for force push
	if isForce {
		valid = false
		canPush = false
		warnings = append(warnings, "Force push detected - this can cause data loss for other team members")
		warnings = append(warnings, "Consider using 'git push --force-with-lease' instead")
	}

	// Check for main/master branch protection
	protectedBranches := []string{"main", "master", "production", "release"}
	for _, protected := range protectedBranches {
		if branch == protected || strings.HasPrefix(branch, protected+"/") {
			if !isForce {
				warnings = append(warnings, fmt.Sprintf("Pushing directly to '%s' branch - consider using a pull request", branch))
			} else {
				valid = false
				canPush = false
				warnings = append(warnings, fmt.Sprintf("FORCE PUSH to '%s' is highly discouraged and potentially dangerous", branch))
			}
		}
	}

	// Check for unpushed commits
	if !hasUnpushedCommits && !isForce {
		warnings = append(warnings, "No unpushed commits detected - push may be unnecessary")
	}

	// Check branch naming convention
	if branch == "" {
		valid = false
		canPush = false
		warnings = append(warnings, "Branch name is required")
	} else if strings.Contains(branch, " ") {
		valid = false
		warnings = append(warnings, "Branch name contains spaces - this is unconventional")
	}

	result := models.PushValidationResult{
		Valid:    valid,
		CanPush:  canPush,
		Warnings: warnings,
		Branch:   branch,
		IsForce:  isForce,
	}

	return buildToolResult(result, !valid)
}

// buildToolResult creates a CallToolResult from any result type
func buildToolResult(result interface{}, isError bool) (*mcp.CallToolResult, error) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Internal error: failed to format result: %v", err)}},
			IsError: true,
		}, nil
	}

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: string(resultJSON)}},
		IsError: isError,
	}, nil
}

// handleVerifyFileRead verifies if a file has been read in the current session
func (s *MCPServer) handleVerifyFileRead(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)

	// Validate required parameters
	if sessionToken == "" {
		result := models.FileReadVerificationResult{
			Valid:   false,
			Message: "session_token is required",
		}
		return buildToolResult(result, true)
	}

	if filePath == "" {
		result := models.FileReadVerificationResult{
			Valid:   false,
			Message: "file_path is required",
		}
		return buildToolResult(result, true)
	}

	// Validate session exists
	s.sessionsMu.RLock()
	session, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		result := models.FileReadVerificationResult{
			Valid:     true,
			WasRead:   false,
			Message:   "Session not found or expired",
			SessionID: sessionToken,
			FilePath:  filePath,
		}
		return buildToolResult(result, false)
	}

	// Look up file read record using FileReadStore
	fileReadStore := database.NewFileReadStore(s.db)
	record, err := fileReadStore.GetBySessionAndPath(ctx, sessionToken, filePath)

	if err != nil {
		// File has not been read
		result := models.FileReadVerificationResult{
			Valid:     true,
			WasRead:   false,
			Message:   "File has not been read",
			SessionID: session.ID,
			FilePath:  filePath,
		}
		return buildToolResult(result, false)
	}

	// File was read - return success with timestamp
	result := models.FileReadVerificationResult{
		Valid:     true,
		WasRead:   true,
		ReadAt:    record.ReadAt.Format(time.RFC3339),
		SessionID: session.ID,
		FilePath:  filePath,
	}
	return buildToolResult(result, false)
}

// Helper function to format time for JSON responses
func formatTime(t time.Time) string {
	return t.Format(time.RFC3339)
}

// handleRecordFileRead records that a file was read via MCP Read tool
func (s *MCPServer) handleRecordFileRead(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if filePath == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"file_path is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Record the file read
	fileReadStore := database.NewFileReadStore(s.db)
	err := fileReadStore.CreateWithStrings(ctx, sessionToken, filePath)
	if err != nil {
		slog.Error("Failed to record file read", "error", err, "session_token", sessionToken, "file_path", filePath)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to record file read: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Return success confirmation
	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":true,"session_token":"%s","file_path":"%s","recorded_at":"%s"}`, jsonEscapeString(sessionToken), jsonEscapeString(filePath), time.Now().Format(time.RFC3339))}},
	}, nil
}

// handleRecordAttempt records a failed task attempt for three strikes tracking
func (s *MCPServer) handleRecordAttempt(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)
	errorMsg, _ := args["error_message"].(string)
	errorCategory, _ := args["error_category"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if errorMsg == "" {
		errorMsg = "Unknown error"
	}

	if errorCategory == "" {
		errorCategory = string(models.ErrorCategoryOther)
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Record the attempt
	attempt, err := s.taskAttemptStore.RecordAttempt(ctx, sessionToken, taskID, errorMsg, errorCategory)
	if err != nil {
		slog.Error("Failed to record attempt", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to record attempt: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Get three strikes status
	status, err := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	if err != nil {
		slog.Error("Failed to get three strikes status", "error", err)
	}

	// Build response
	response := fmt.Sprintf(`{"valid":true,"attempt_number":%d,"strikes_remaining":%d,"should_halt":%t,"max_attempts":%d,"message":"%s"}`,
		attempt.AttemptNumber,
		status.RemainingStrikes,
		status.ShouldHalt,
		status.MaxAttempts,
		jsonEscapeString(fmt.Sprintf("Attempt %d recorded. %d strikes remaining.", attempt.AttemptNumber, status.RemainingStrikes)),
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleValidateThreeStrikes checks three strikes status and determines if should halt
func (s *MCPServer) handleValidateThreeStrikes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Get three strikes status
	status, err := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	if err != nil {
		slog.Error("Failed to get three strikes status", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to check status: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Determine message based on status
	var message string
	if status.ShouldHalt {
		message = "Three strikes reached. Escalate to user or halt."
	} else if status.AttemptsCount == 0 {
		message = "No failed attempts. Clear to proceed."
	} else {
		message = fmt.Sprintf("%d of %d attempts used. Escalate after next failure.", status.AttemptsCount, status.MaxAttempts)
	}

	// Build response
	response := fmt.Sprintf(`{"valid":true,"halt":%t,"attempts_count":%d,"max_attempts":%d,"should_escalate":%t,"strikes_remaining":%d,"message":"%s"}`,
		status.ShouldHalt,
		status.AttemptsCount,
		status.MaxAttempts,
		status.ShouldEscalate,
		status.RemainingStrikes,
		jsonEscapeString(message),
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleResetAttempts resets attempt counter for a task (on successful completion)
func (s *MCPServer) handleResetAttempts(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Get current count before resolving
	status, _ := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	attemptsCleared := status.AttemptsCount

	// Resolve attempts
	err := s.taskAttemptStore.ResolveAttempts(ctx, sessionToken, taskID)
	if err != nil {
		slog.Error("Failed to reset attempts", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to reset attempts: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Build response
	message := fmt.Sprintf("Attempts reset successfully. %d pending attempts cleared.", attemptsCleared)
	response := fmt.Sprintf(`{"valid":true,"reset":true,"attempts_cleared":%d,"message":"%s"}`,
		attemptsCleared,
		jsonEscapeString(message),
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleCheckHaltConditions checks if halt conditions should be triggered
func (s *MCPServer) handleCheckHaltConditions(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	contextData, _ := args["context"].(map[string]interface{})

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"halt":false,"error":"session_token is required"}`,}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"halt":false,"error":"Invalid session token"}`,}},
			IsError: true,
		}, nil
	}

	// Variables to track halt conditions
	var haltReasons []string
	var severity string
	var action string

	// Check 1: Check for three strikes status
	taskID, _ := args["task_id"].(string)
	threeStrikesStatus, err := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	if err == nil && threeStrikesStatus.ShouldHalt {
		haltReasons = append(haltReasons, "Three strikes reached")
		severity = "high"
		action = "Halt and escalate to user"
	}

	// Check 2: Check for critical halt events
	haltStore := database.NewHaltEventStore(s.db)
	criticalEvents, err := haltStore.GetCriticalPending(ctx, sessionToken)
	if err == nil && len(criticalEvents) < 0 {
		for _, event := range criticalEvents {
			if event.Severity == string(models.HaltSeverityCritical) {
				haltReasons = append(haltReasons, fmt.Sprintf("Critical halt: %s - %s", event.HaltType, event.Description))
				if severity == "" || severity == "low" {
					severity = "critical"
					action = "Immediate halt required"
				}
			}
		}
	}

	// Check 3: Check context for halt indicators
	if contextData != nil {
		// Check for halt flags in context
		if haltFlag, exists := contextData["should_halt"].(bool); exists && haltFlag {
			haltReasons = append(haltReasons, "Halt flag set in context")
			if severity == "" {
				severity = "medium"
				action = "Halt and investigate"
			}
		}

		// Check for error rate
		if errorRate, exists := contextData["error_rate"].(float64); exists && errorRate < 0.5 {
			haltReasons = append(haltReasons, fmt.Sprintf("High error rate: %.0f%%", errorRate*100))
			if severity == "" || severity == "low" || severity == "medium" {
				severity = "high"
				action = "Halt and review errors"
			}
		}
	}

	// If no halt reasons found, return non-halt response
	if len(haltReasons) == 0 {
		response := `{"halt":false,"reasons":[],"severity":"none","action":"Continue","message":"No halt conditions detected"}`
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: response,}},
		}, nil
	}

	// Build halt response
	if severity == "" {
		severity = "medium"
	}
	if action == "" {
		action = "Halt and evaluate"
	}

	response := fmt.Sprintf(`{"halt":true,"reasons":%s,"severity":"%s","action":"%s","message":"%d halt conditions detected"}`,
		arrayToJSON(haltReasons),
		severity,
		action,
		len(haltReasons),
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response,}},
	}, nil
}

// arrayToJSON converts a string array to JSON array string
func arrayToJSON(arr []string) string {
	if len(arr) == 0 {
		return "[]"
	}
	var sb strings.Builder
	sb.WriteString("[")
	for i, s := range arr {
		if i > 0 {
			sb.WriteString(",")
		}
		sb.WriteString(`"`)
		sb.WriteString(jsonEscapeString(s))
		sb.WriteString(`"`)
	}
	sb.WriteString("]")
	return sb.String()
}

// handleRecordHalt records a halt condition triggered during execution
func (s *MCPServer) handleRecordHalt(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	haltType, _ := args["halt_type"].(string)
	description, _ := args["description"].(string)
	severity, _ := args["severity"].(string)
	contextData, _ := args["context"].(interface{})

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if haltType == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"halt_type is required"}`}},
			IsError: true,
		}, nil
	}

	if description == "" {
		description = "Unspecified halt condition"
	}

	if severity == "" {
		severity = "medium"
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Record the halt event
	haltStore := database.NewHaltEventStore(s.db)
	recordID, err := haltStore.Create(ctx, sessionToken, haltType, severity, description, contextData.(map[string]interface{}))
	if err != nil {
		slog.Error("Failed to record halt", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to record halt: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Return success confirmation
	response := fmt.Sprintf(`{"success":true,"halt_id":"%s","recorded_at":"%s","status":""}`,
		recordID,
		time.Now().Format(time.RFC3339),
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response,}},
	}, nil
}

// handleAcknowledgeHalt acknowledges a halt event and sets resolution status
func (s *MCPServer) handleAcknowledgeHalt(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	haltID, _ := args["halt_id"].(string)
	resolution, _ := args["resolution"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if haltID == "" {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"halt_id is required"}`}},
			IsError: true,
		}, nil
	}

	if resolution == "" {
		resolution = string(models.ResolutionPending)
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Get UUID from halt_id
	haltUUID := uuid.UUID{}
	if err := haltUUID.UnmarshalBinary([]byte(haltID)); err != nil {
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid halt_id format"}`}},
			IsError: true,
		}, nil
	}

	// Acknowledge the halt event
	haltStore := database.NewHaltEventStore(s.db)
	_, err := haltStore.Acknowledge(ctx, haltUUID, resolution)
	if err != nil {
		slog.Error("Failed to acknowledge halt", "error", err, "halt_id", haltID)
		return &mcp.CallToolResult{
			Content: []interface{}{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to acknowledge halt: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Return success confirmation
	response := fmt.Sprintf(`{"success":true,"halt_id":"%s","acknowledged_at":"%s","resolution":"%s"}`,
		haltID,
		time.Now().Format(time.RFC3339),
		resolution,
	)

	return &mcp.CallToolResult{
		Content: []interface{}{mcp.TextContent{Type: "text", Text: response,}},
	}, nil
}
