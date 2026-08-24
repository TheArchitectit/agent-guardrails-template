package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/database"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (s *MCPServer) handleVerifyFixesIntact(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)
	modifiedContent, _ := args["modified_content"].(string)
	originalContent, _ := args["original_content"].(string)

	// Validate required parameters
	if sessionToken == "" {
		result := models.FixVerificationResult{
			AllFixesIntact: false,
			VerifySummary:  "Session token is required",
			Fixes:          []models.IndividualFixResult{},
			Recommendation: "Invalid input - session_token required",
		}
		return buildToolResult(result, true)
	}

	if filePath == "" {
		result := models.FixVerificationResult{
			AllFixesIntact: false,
			VerifySummary:  "File path is required",
			Fixes:          []models.IndividualFixResult{},
			Recommendation: "Invalid input - file_path required",
		}
		return buildToolResult(result, true)
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		result := models.FixVerificationResult{
			AllFixesIntact: false,
			VerifySummary:  "Session not found or expired",
			Fixes:          []models.IndividualFixResult{},
			Recommendation: "Create a new session",
		}
		return buildToolResult(result, true)
	}

	if s.db == nil {
		result := models.FixVerificationResult{
			AllFixesIntact: false,
			VerifySummary:  "Database connection is not configured",
			Fixes:          []models.IndividualFixResult{},
			Recommendation: "Configure database and retry",
		}
		return buildToolResult(result, true)
	}

	if s.fixVerificationStore == nil {
		result := models.FixVerificationResult{
			AllFixesIntact: false,
			VerifySummary:  "Fix verification store is not configured",
			Fixes:          []models.IndividualFixResult{},
			Recommendation: "Configure fix verification store and retry",
		}
		return buildToolResult(result, true)
	}

	// Get failure registry store to query active failures for the file
	failStore := database.NewFailureStore(s.db)
	failures, err := failStore.GetActiveByFiles(ctx, []string{filePath})
	if err != nil {
		return &mcp.CallToolResult{
			Content: []mcp.Content{mcp.TextContent{Type: "text", Text: fmt.Sprintf("Failed to check failures: %v", err)}},
			IsError: true,
		}, nil
	}

	// Get current content for verification (use modified_content if provided, otherwise read from file)
	currentContent := modifiedContent
	if currentContent == "" {
		if currentContent, err = s.readFileContent(filePath); err != nil {
			currentContent = originalContent
		}
	}

	// Process each failure as a potential fix to verify
	fixVerificationStore := s.fixVerificationStore
	results := []models.IndividualFixResult{}
	intactCount := 0

	if len(failures) == 0 {
		// No failures found for this file, check if we have any fix tracking records
		if verifications, err := fixVerificationStore.GetBySessionAndFile(ctx, sessionToken, filePath); err == nil && len(verifications) > 0 {
			for _, v := range verifications {
				// Verify against current content
				status, message := fixVerificationStore.VerifyFixContent(ctx, currentContent, &v)
				results = append(results, models.IndividualFixResult{
					FailureID:           v.FailureID,
					Status:              status,
					FixType:             v.FixType,
					AffectedFile:        v.FilePath,
					VerificationMessage: message,
				})
				if status == models.StatusConfirmed {
					intactCount++
				}
			}
		}
	} else {
		// Verify each failure/fix against current content
		for _, failure := range failures {
			// Try to get or create a fix verification record
			fixContent := failure.RootCause
			var fixType models.FixType

			// Determine fix type based on failure data
			if failure.RegressionPattern != "" {
				fixType = models.FixTypeRegex
				// Use regression pattern as fix content for regex fixes
				fixContent = failure.RegressionPattern
			} else if failure.RootCause != "" {
				// This is a bit of a guess - using error message as fix content for code changes
				fixContent = failure.ErrorMessage
				fixType = models.FixTypeCodeChange
			} else {
				fixType = models.FixTypeConfig
				fixContent = failure.ErrorMessage
			}

			verification, err := fixVerificationStore.GetOrCreate(ctx, sessionToken, failure.FailureID, filePath, fixContent, fixType)
			if err != nil {
				slog.Warn("Failed to get or create fix verification", "error", err, "failure_id", failure.FailureID)
				continue
			}

			// Verify if fix is intact
			status, message := fixVerificationStore.VerifyFixContent(ctx, currentContent, verification)

			// Update verification status
			if err := fixVerificationStore.UpdateVerificationStatus(ctx, sessionToken, failure.FailureID, status); err != nil {
				slog.Warn("Failed to update verification status", "error", err, "failure_id", failure.FailureID)
			}

			results = append(results, models.IndividualFixResult{
				FailureID:           failure.FailureID,
				Status:              status,
				FixType:             fixType,
				AffectedFile:        filePath,
				VerificationMessage: message,
			})

			if status == models.StatusConfirmed {
				intactCount++
			}
		}
	}

	// Build summary and recommendation
	totalFixes := len(results)
	var summary, recommendation string
	allIntact := true

	if totalFixes == 0 {
		summary = "No fixes to verify for this file"
		recommendation = "Proceed - no fixes found"
		allIntact = true
	} else {
		summary = fmt.Sprintf("%d/%d fixes verified intact", intactCount, totalFixes)
		if intactCount == totalFixes {
			recommendation = "Continue - all fixes intact"
			allIntact = true
		} else if intactCount > 0 {
			recommendation = "Review - some fixes modified"
			allIntact = false
		} else {
			recommendation = "Halt - fix verification failed"
			allIntact = false
		}
	}

	result := models.FixVerificationResult{
		AllFixesIntact: allIntact,
		VerifySummary:  summary,
		Fixes:          results,
		Recommendation: recommendation,
	}

	return buildToolResult(result, !allIntact)
}

// handleValidateExactReplacement validates that code replacement matches exact specification
func (s *MCPServer) handleValidateExactReplacement(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)
	originalContent, _ := args["original_content"].(string)
	modifiedContent, _ := args["modified_content"].(string)
	replacementType, _ := args["replacement_type"].(string)

	// Validate required parameters
	if sessionToken == "" {
		result := models.ExactReplacementValidationResult{
			ExactMatch:     false,
			Violations:     []models.ExactReplacementViolation{{Type: "validation_error", Severity: "error", Message: "session_token is required"}},
			Recommendation: "Invalid input - session_token required",
		}
		return buildToolResult(result, true)
	}

	if filePath == "" {
		result := models.ExactReplacementValidationResult{
			ExactMatch:     false,
			Violations:     []models.ExactReplacementViolation{{Type: "validation_error", Severity: "error", Message: "file_path is required"}},
			Recommendation: "Invalid input - file_path required",
		}
		return buildToolResult(result, true)
	}

	// If original_content is empty and modified_content is empty or not provided,
	// it's not an error but should be flagged as no validation needed
	if originalContent == "" {
		if modifiedContent == "" {
			result := models.ExactReplacementValidationResult{
				ExactMatch:     true,
				Violations:     []models.ExactReplacementViolation{},
				DiffStats:      models.DiffStats{Additions: 0, Deletions: 0},
				Recommendation: "No content to validate - acceptable for file creation",
			}
			return buildToolResult(result, false)
		}
		// Original content is empty but modified content exists - this is file creation
		result := models.ExactReplacementValidationResult{
			ExactMatch:     true,
			Violations:     []models.ExactReplacementViolation{},
			DiffStats:      models.DiffStats{Additions: len(strings.Split(modifiedContent, "\n")), Deletions: 0},
			Recommendation: "File creation - no exact match validation needed",
		}
		return buildToolResult(result, false)
	}

	// Use provided modified_content or read from file
	actualContent := modifiedContent
	if actualContent == "" {
		if readContent, err := s.readFileContent(filePath); err == nil {
			actualContent = readContent
		}
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		result := models.ExactReplacementValidationResult{
			ExactMatch:     false,
			Violations:     []models.ExactReplacementViolation{{Type: "session_error", Severity: "error", Message: "Session not found or expired"}},
			Recommendation: "Create a new session",
		}
		return buildToolResult(result, true)
	}

	// Analyze the diff between original and modified content
	result := detectExactReplacementViolations(originalContent, actualContent, replacementType)
	return buildToolResult(result, !result.ExactMatch)
}

// detectExactReplacementViolations analyzes content differences for violations
