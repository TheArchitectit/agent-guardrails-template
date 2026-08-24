package mcp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (s *MCPServer) readFileContent(filePath string) (string, error) {
	data, err := safeReadFile(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %w", err)
	}
	return string(data), nil
}

// handleValidateExactReplacement validates that code replacement matches exact specification

func detectExactReplacementViolations(originalContent, actualContent, replacementType string) models.ExactReplacementValidationResult {
	violations := []models.ExactReplacementViolation{}
	additions := 0
	deletions := 0

	// If content matches exactly, return early
	if originalContent == actualContent {
		return models.ExactReplacementValidationResult{
			ExactMatch:     true,
			Violations:     []models.ExactReplacementViolation{},
			DiffStats:      models.DiffStats{Additions: 0, Deletions: 0},
			Recommendation: "Accept changes - exact match confirmed",
		}
	}

	// Split content into lines
	originalLines := strings.Split(originalContent, "\n")
	actualLines := strings.Split(actualContent, "\n")

	// Precompile regex patterns
	importPattern := regexp.MustCompile(`^\+*\s*import\s+`)
	typeHintPattern := regexp.MustCompile(`^\+*\s*.*:\s*\w+`) // e.g., name: Type
	debugPattern := regexp.MustCompile(`^\+*\s*.*(fmt\.Print|console\.log|println|echo)`)
	funcPattern := regexp.MustCompile(`^\+*\s*\b(func|def|function)\s+\w+`)
	commentPattern := regexp.MustCompile(`^[-+]\s*//.*`)
	variableRenamePattern := regexp.MustCompile(`^\+*\s*.*=.*//\s*renamed`)
	formattingPattern := regexp.MustCompile(`^\+*\s*\s*$`) // Lines with only whitespace changes
	reorderPattern := regexp.MustCompile(`^[-+]\s*package\s+|^\+*\s*package\s+`)

	// Calculate line-by-line diff summary using a simple line comparison
	// This is a simplified diff - in production you might use a proper diff library
	minLines := len(originalLines)
	if len(actualLines) < minLines {
		minLines = len(actualLines)
	}

	// Track changes line by line
	contentChanged := false
	for i := 0; i < minLines; i++ {
		if originalLines[i] != actualLines[i] {
			contentChanged = true
			// Count as modification (both add and delete)
			additions++
			deletions++
		}
	}

	// Add extra lines
	if len(actualLines) > minLines {
		additions += len(actualLines) - minLines
		contentChanged = true
	}

	// Removed lines
	if len(originalLines) > minLines {
		deletions += len(originalLines) - minLines
		contentChanged = true
	}

	// If no content changes detected, return exact match
	if !contentChanged && additions == 0 && deletions == 0 {
		return models.ExactReplacementValidationResult{
			ExactMatch:     true,
			Violations:     []models.ExactReplacementViolation{},
			DiffStats:      models.DiffStats{Additions: 0, Deletions: 0},
			Recommendation: "Accept changes - exact match confirmed",
		}
	}

	// Analyze actual content for violations
	// Split into added and removed lines
	addedLines := []string{}
	removedLines := []string{}

	// Simple diff calculation - find adds and removes
	// In production, consider using a proper diff algorithm
	originalSet := make(map[string]bool)
	for _, line := range originalLines {
		if strings.TrimSpace(line) != "" {
			originalSet[line] = true
		}
	}

	actualSet := make(map[string]bool)
	for _, line := range actualLines {
		if strings.TrimSpace(line) != "" {
			actualSet[line] = true
			if !originalSet[line] {
				addedLines = append(addedLines, line)
			}
		}
	}

	for _, line := range originalLines {
		if strings.TrimSpace(line) != "" && !actualSet[line] {
			removedLines = append(removedLines, line)
		}
	}

	// Check for violations in added lines
	var criticalCount, warningCount int

	for _, line := range addedLines {
		// Check for new imports
		if importPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "new_import",
				Severity: "warning",
				Message:  "Additional import added: " + strings.TrimSpace(line),
			})
			warningCount++
		}

		// Check for type hint changes (particularly relevant for Python/JavaScript)
		if typeHintPattern.MatchString(line) && !strings.Contains(line, "//") {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "type_change",
				Severity: "warning",
				Message:  "Type hint added or changed: " + strings.TrimSpace(line),
			})
			warningCount++
		}

		// Check for debug statements
		if debugPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "debug_added",
				Severity: "error",
				Message:  "Debug statement added: " + strings.TrimSpace(line),
			})
			criticalCount++
		}

		// Check for new functions
		if funcPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "extra_function",
				Severity: "error",
				Message:  "New function added: " + strings.TrimSpace(line),
			})
			criticalCount++
		}

		// Check for variable renames (pattern: old = new // renamed)
		if variableRenamePattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "variable_rename",
				Severity: "warning",
				Message:  "Variable renamed: " + strings.TrimSpace(line),
			})
			warningCount++
		}

		// Check for pure formatting changes (only whitespace)
		if formattingPattern.MatchString(line) && len(strings.TrimSpace(line)) == 0 {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "formatting",
				Severity: "info",
				Message:  "Formatting change (whitespace only)",
			})
		}

		// Check for package reorganization (Go files)
		if reorderPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "code_reorganized",
				Severity: "warning",
				Message:  "Code structure changed (package reordering)",
			})
			warningCount++
		}
	}

	// Check for comment changes
	for _, line := range append(addedLines, removedLines...) {
		if commentPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "comment_change",
				Severity: "info",
				Message:  "Comment modified: " + strings.TrimSpace(line),
			})
		}
	}

	// Check removed lines for significant deletions
	for _, line := range removedLines {
		// Check if a function was removed
		if funcPattern.MatchString(line) {
			violations = append(violations, models.ExactReplacementViolation{
				Type:     "function_removed",
				Severity: "error",
				Message:  "Original function removed: " + strings.TrimSpace(line),
			})
			criticalCount++
		}
	}

	// Determine recommendation based on violations
	recommendation := "Accept changes - exact match confirmed"
	exactMatch := len(violations) == 0

	if !exactMatch {
		if criticalCount > 0 {
			recommendation = "Reject and use exact code - critical violations detected"
		} else if warningCount > 2 {
			recommendation = "Review modifications - multiple warnings"
		} else if warningCount > 0 {
			recommendation = "Proceed with caution - minor changes acceptable"
		} else {
			// Only info-level violations
			recommendation = "Accept changes - improvements only"
			exactMatch = true // Consider info-only as acceptable
		}
	}

	return models.ExactReplacementValidationResult{
		ExactMatch:     exactMatch,
		Violations:     violations,
		DiffStats:      models.DiffStats{Additions: additions, Deletions: deletions},
		Recommendation: recommendation,
	}
}

// handleVerifyTestsBeforeCommit validates that tests pass before committing
