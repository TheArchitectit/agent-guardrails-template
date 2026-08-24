package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

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
			SessionID: session.Token,
			FilePath:  filePath,
		}
		return buildToolResult(result, false)
	}

	// File was read - return success with timestamp
	result := models.FileReadVerificationResult{
		Valid:     true,
		WasRead:   true,
		ReadAt:    record.ReadAt.Format(time.RFC3339),
		SessionID: session.Token,
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if filePath == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"file_path is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Record the file read
	fileReadStore := database.NewFileReadStore(s.db)
	err := fileReadStore.CreateWithStrings(ctx, sessionToken, filePath)
	if err != nil {
		slog.Error("Failed to record file read", "error", err, "session_token", sessionToken, "file_path", filePath)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to record file read: %s"}`, jsonEscapeString(err.Error()))}},
			IsError: true,
		}, nil
	}

	// Return success confirmation
	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":true,"session_token":"%s","file_path":"%s","recorded_at":"%s"}`, jsonEscapeString(sessionToken), jsonEscapeString(filePath), time.Now().Format(time.RFC3339))}},
	}, nil
}

// handleRecordAttempt records a failed task attempt for three strikes tracking
func (s *MCPServer) handleRecordAttempt(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Panic recovery to prevent HTTP 500
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in handleRecordAttempt", "recover", r)
		}
	}()

	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)
	errorMsg, _ := args["error_message"].(string)
	errorCategory, _ := args["error_category"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Check if taskAttemptStore is available
	if s.taskAttemptStore == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Task attempt store not available"}`}},
			IsError: true,
		}, nil
	}

	// Record the attempt
	attempt, err := s.taskAttemptStore.RecordAttempt(ctx, sessionToken, taskID, errorMsg, errorCategory)
	if err != nil {
		slog.Error("Failed to record attempt", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to record attempt: %s"}`, jsonEscapeString(err.Error()))}},
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
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleValidateThreeStrikes checks three strikes status and determines if should halt
func (s *MCPServer) handleValidateThreeStrikes(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Panic recovery to prevent HTTP 500
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in handleValidateThreeStrikes", "recover", r)
		}
	}()

	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Check if taskAttemptStore is available
	if s.taskAttemptStore == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Task attempt store not available"}`}},
			IsError: true,
		}, nil
	}

	// Get three strikes status
	status, err := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	if err != nil {
		slog.Error("Failed to get three strikes status", "error", err, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to check status: %s"}`, jsonEscapeString(err.Error()))}},
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
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleResetAttempts resets attempt counter for a task (on successful completion)
func (s *MCPServer) handleResetAttempts(ctx context.Context, args map[string]interface{}) (result *mcp.CallToolResult, err error) {
	// Panic recovery to prevent HTTP 500
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in handleResetAttempts", "recover", r)
			result = &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Internal server error"}`}},
				IsError: true,
			}
		}
	}()

	sessionToken, _ := args["session_token"].(string)
	taskID, _ := args["task_id"].(string)

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Check if taskAttemptStore is available
	if s.taskAttemptStore == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"valid":false,"error":"Task attempt store not available"}`}},
			IsError: true,
		}, nil
	}

	// Get current count before resolving
	status, _ := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
	attemptsCleared := status.AttemptsCount

	// Resolve attempts
	resolveErr := s.taskAttemptStore.ResolveAttempts(ctx, sessionToken, taskID)
	if resolveErr != nil {
		slog.Error("Failed to reset attempts", "error", resolveErr, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"valid":false,"error":"Failed to reset attempts: %s"}`, jsonEscapeString(resolveErr.Error()))}},
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
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleCheckHaltConditions checks if halt conditions should be triggered
