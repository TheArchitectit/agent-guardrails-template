package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (s *MCPServer) handleCheckHaltConditions(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	// Panic recovery to prevent HTTP 500
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in handleCheckHaltConditions", "recover", r)
		}
	}()

	sessionToken, _ := args["session_token"].(string)
	contextData, _ := args["context"].(map[string]interface{})

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"halt":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"halt":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Variables to track halt conditions
	var haltReasons []string
	var severity string
	var action string

	// Check 1: Check for three strikes status (if store is available)
	taskID, _ := args["task_id"].(string)
	if s.taskAttemptStore != nil {
		threeStrikesStatus, err := s.taskAttemptStore.GetThreeStrikesStatus(ctx, sessionToken, taskID)
		if err == nil && threeStrikesStatus.ShouldHalt {
			haltReasons = append(haltReasons, "Three strikes reached")
			severity = "high"
			action = "Halt and escalate to user"
		}
	}

	// Check 2: Check for critical halt events
	haltStore := database.NewHaltEventStore(s.db)
	criticalEvents, err := haltStore.GetCriticalPending(ctx, sessionToken)
	if err == nil && len(criticalEvents) > 0 {
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
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
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
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
func (s *MCPServer) handleRecordHalt(ctx context.Context, args map[string]interface{}) (result *mcp.CallToolResult, err error) {
	// Panic recovery to prevent HTTP 500
	defer func() {
		if r := recover(); r != nil {
			slog.Error("Panic in handleRecordHalt", "recover", r)
			result = &mcp.CallToolResult{
				Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Internal server error"}`}},
				IsError: true,
			}
		}
	}()

	sessionToken, _ := args["session_token"].(string)
	haltType, _ := args["halt_type"].(string)
	description, _ := args["description"].(string)
	severity, _ := args["severity"].(string)
	contextData, _ := args["context"]

	// Validate required parameters
	if sessionToken == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if haltType == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"halt_type is required"}`}},
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Check if database is available
	if s.db == nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Database not available"}`}},
			IsError: true,
		}, nil
	}

	// Safe type assertion for contextData
	var contextMap map[string]interface{}
	if contextData != nil {
		if cm, ok := contextData.(map[string]interface{}); ok {
			contextMap = cm
		}
	}

	// Record the halt event
	haltStore := database.NewHaltEventStore(s.db)
	recordID, haltErr := haltStore.Create(ctx, sessionToken, haltType, severity, description, contextMap)
	if haltErr != nil {
		slog.Error("Failed to record halt", "error", haltErr, "session_token", sessionToken)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to record halt: %s"}`, jsonEscapeString(haltErr.Error()))}},
			IsError: true,
		}, nil
	}

	// Return success confirmation
	response := fmt.Sprintf(`{"success":true,"halt_id":"%s","recorded_at":"%s","status":"recorded"}`,
		recordID.ID,
		time.Now().Format(time.RFC3339),
	)

	return &mcp.CallToolResult{
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"session_token is required"}`}},
			IsError: true,
		}, nil
	}

	if haltID == "" {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"halt_id is required"}`}},
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
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid session token"}`}},
			IsError: true,
		}, nil
	}

	// Get UUID from halt_id
	haltUUID := uuid.UUID{}
	if err := haltUUID.UnmarshalBinary([]byte(haltID)); err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: `{"success":false,"error":"Invalid halt_id format"}`}},
			IsError: true,
		}, nil
	}

	// Acknowledge the halt event
	haltStore := database.NewHaltEventStore(s.db)
	_, err := haltStore.Acknowledge(ctx, haltUUID, resolution)
	if err != nil {
		slog.Error("Failed to acknowledge halt", "error", err, "halt_id", haltID)
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf(`{"success":false,"error":"Failed to acknowledge halt: %s"}`, jsonEscapeString(err.Error()))}},
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
		Content: []mcp.Content{mcp.TextContent{Type: "text", Text: response}},
	}, nil
}

// handleValidateProductionFirst validates production-first guardrail rules
