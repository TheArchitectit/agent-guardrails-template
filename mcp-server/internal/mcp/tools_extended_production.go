package mcp

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (s *MCPServer) handleValidateProductionFirst(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)
	codeTypeStr, _ := args["code_type"].(string)

	// Validate required parameters
	if sessionToken == "" {
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: "session_token is required",
		}
		return buildToolResult(result, true)
	}

	if filePath == "" {
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: "file_path is required",
		}
		return buildToolResult(result, true)
	}

	if codeTypeStr == "" {
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: "code_type is required",
		}
		return buildToolResult(result, true)
	}

	// Validate code_type value
	if !models.IsValidCodeType(codeTypeStr) {
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Invalid code_type '%s'. Must be one of: production, test, infrastructure", codeTypeStr),
		}
		return buildToolResult(result, true)
	}

	codeType := models.CodeType(codeTypeStr)

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: "Invalid session token",
		}
		return buildToolResult(result, true)
	}

	// Record the code being created (regardless of type)
	productionCode := &models.ProductionCode{
		SessionID: sessionToken,
		FilePath:  filePath,
		CodeType:  codeType,
	}

	if err := s.productionCodeStore.CreateOrUpdate(ctx, productionCode); err != nil {
		slog.Error("Failed to record production code", "error", err, "session_token", sessionToken, "file_path", filePath)
		result := models.ProductionCodeValidationResult{
			Valid:   false,
			Message: fmt.Sprintf("Failed to record code: %s", err.Error()),
		}
		return buildToolResult(result, true)
	}

	// If code_type is production, mark as verified
	if codeType == models.CodeTypeProduction {
		if err := s.productionCodeStore.MarkAsVerified(ctx, sessionToken, filePath); err != nil {
			slog.Warn("Failed to mark production code as verified", "error", err, "file_path", filePath)
		}
	}

	// Check if this is test or infrastructure code
	if codeType == models.CodeTypeTest || codeType == models.CodeTypeInfrastructure {
		// Check if production code exists in the session
		hasProductionCode, err := s.productionCodeStore.HasProductionCode(ctx, sessionToken)
		if err != nil {
			slog.Error("Failed to check production code existence", "error", err, "session_token", sessionToken)
			result := models.ProductionCodeValidationResult{
				Valid:   false,
				Message: "Failed to check production code existence",
			}
			return buildToolResult(result, true)
		}

		if !hasProductionCode {
			result := models.ProductionCodeValidationResult{
				Valid:                false,
				Message:              "Production code must be created first",
				ProductionCodeExists: false,
			}
			return buildToolResult(result, true)
		}

		// Production code exists, validation passes
		result := models.ProductionCodeValidationResult{
			Valid:                true,
			Message:              fmt.Sprintf("%s code creation validated successfully", codeType),
			ProductionCodeExists: true,
		}
		return buildToolResult(result, false)
	}

	// For production code, always pass
	result := models.ProductionCodeValidationResult{
		Valid:                true,
		Message:              "Production code can always be created",
		ProductionCodeExists: true,
	}
	return buildToolResult(result, false)
}

// handleDetectFeatureCreep detects if changes contain feature creep
func (s *MCPServer) handleDetectFeatureCreep(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	sessionToken, _ := args["session_token"].(string)
	filePath, _ := args["file_path"].(string)
	gitDiff, _ := args["git_diff"].(string)
	changeDescription, _ := args["change_description"].(string)
	isNewFile, _ := args["is_new_file"].(bool)

	// Validate required parameters
	if sessionToken == "" {
		result := models.FeatureCreepDetectionResult{
			CreepDetected:  false,
			Recommendation: "session_token is required",
		}
		return buildToolResult(result, true)
	}

	if filePath == "" {
		result := models.FeatureCreepDetectionResult{
			CreepDetected:  false,
			Recommendation: "file_path is required",
		}
		return buildToolResult(result, true)
	}

	if gitDiff == "" {
		result := models.FeatureCreepDetectionResult{
			CreepDetected:  false,
			Recommendation: "git_diff is required",
		}
		return buildToolResult(result, true)
	}

	// Validate session exists
	s.sessionsMu.RLock()
	_, exists := s.sessions[sessionToken]
	s.sessionsMu.RUnlock()

	if !exists {
		result := models.FeatureCreepDetectionResult{
			CreepDetected: false,
			Violations: []models.FeatureCreepViolation{
				{
					Type:     "session_error",
					Severity: "error",
					Message:  "Session not found or expired",
				},
			},
			Recommendation: "Invalid session token",
		}
		return buildToolResult(result, true)
	}

	// Parse and analyze the git diff
	result := detectFeatureCreep(gitDiff, changeDescription, isNewFile, filePath)
	return buildToolResult(result, result.CreepDetected)
}

// detectFeatureCreep analyzes git diff for feature creep patterns
func detectFeatureCreep(gitDiff string, changeDescription string, isNewFile bool, filePath string) models.FeatureCreepDetectionResult {
	violations := []models.FeatureCreepViolation{}
	additions := 0
	deletions := 0
	newFunctions := 0
	newImports := 0
	newClasses := 0
	newEndpoints := 0
	refactoringIndicators := 0
	improvementIndicators := 0

	// Split diff into lines
	lines := strings.Split(gitDiff, "\n")

	// Precompile regex patterns
	funcPattern := regexp.MustCompile(`^\+.*\bfunc\s+(\w+\s*)?\(`)
	importPattern := regexp.MustCompile(`^\+import\s+`)
	thirdPartyImportPattern := regexp.MustCompile(`^\+.*import.*["'][^"'/]+/[^"']+["']`)
	classPattern := regexp.MustCompile(`^\+.*\b(class|struct|interface)\s+\w+`)
	endpointPattern := regexp.MustCompile(`^\+.*\b(http|endpoint|route|api|REST)\b`)
	refactorPattern := regexp.MustCompile(`(?i)(refactor|rename|restructure|reorganize)`)
	improvePattern := regexp.MustCompile(`(?i)\b(better|improved|optimized|enhanced|cleaned|simplified)\b`)
	commentPattern := regexp.MustCompile(`^\+\s*//\s*(?i)(refactor|improve|optimize|enhance|clean|simplify)`)

	// Analyze each line in the diff
	for _, line := range lines {
		if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			additions++

			// Check for new functions
			if funcPattern.MatchString(line) {
				newFunctions++
			}

			// Check for new imports
			if importPattern.MatchString(line) {
				newImports++
				// Check for third-party imports
				if thirdPartyImportPattern.MatchString(line) {
					violations = append(violations, models.FeatureCreepViolation{
						Type:     "new_import",
						Severity: "warning",
						Message:  "Third-party package import detected",
					})
				}
			}

			// Check for new classes/structs
			if classPattern.MatchString(line) {
				newClasses++
			}

			// Check for new endpoints
			if endpointPattern.MatchString(line) {
				newEndpoints++
			}

			// Check for refactoring indicators in comments
			if commentPattern.MatchString(line) {
				refactoringIndicators++
			}

			// Check for improvement words anywhere in added lines
			if improvePattern.MatchString(line) {
				improvementIndicators++
			}
		} else if strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "---") {
			deletions++
		}
	}

	// Check change description for improvement indicators
	if changeDescription != "" {
		if refactorPattern.MatchString(changeDescription) {
			refactoringIndicators++
		}
		if improvePattern.MatchString(changeDescription) {
			improvementIndicators++
		}
	}

	// Apply detection rules

	// Rule: New file with substantial additions
	if isNewFile && additions > 50 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "large_addition",
			Severity: "warning",
			Message:  fmt.Sprintf("New file with %d additions - potential feature creep", additions),
		})
	}

	// Rule: Multiple new functions
	if newFunctions > 1 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "new_feature",
			Severity: "warning",
			Message:  fmt.Sprintf("Multiple new functions added (%d)", newFunctions),
		})
	}

	// Rule: Multiple new classes/structs
	if newClasses > 1 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "new_feature",
			Severity: "warning",
			Message:  fmt.Sprintf("Multiple new classes/structs added (%d)", newClasses),
		})
	}

	// Rule: New endpoints
	if newEndpoints > 0 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "new_feature",
			Severity: "warning",
			Message:  fmt.Sprintf("New endpoint(s) added (%d)", newEndpoints),
		})
	}

	// Rule: Large additions
	if additions > 100 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "large_addition",
			Severity: "warning",
			Message:  fmt.Sprintf("Large number of additions: %d lines", additions),
		})
	}

	// Rule: Refactoring without clear purpose
	if refactoringIndicators > 0 && newFunctions == 0 && newClasses == 0 && additions < 50 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "refactor",
			Severity: "warning",
			Message:  fmt.Sprintf("Code refactoring detected (%d indicators)", refactoringIndicators),
		})
	}

	// Rule: Improvement indicators
	if improvementIndicators > 0 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "improvement",
			Severity: "warning",
			Message:  fmt.Sprintf("Improvement keywords detected (%d)", improvementIndicators),
		})
	}

	// Rule: Excessive imports
	if newImports > 3 {
		violations = append(violations, models.FeatureCreepViolation{
			Type:     "new_import",
			Severity: "warning",
			Message:  fmt.Sprintf("Many new imports added (%d)", newImports),
		})
	}

	// Build recommendation based on violations
	recommendation := "Continue with caution"
	creepDetected := len(violations) > 0

	if creepDetected {
		criticalCount := 0
		for _, v := range violations {
			if v.Severity == "error" {
				criticalCount++
			}
		}

		if criticalCount > 0 {
			recommendation = "Halt - critical feature creep detected"
		} else if len(violations) > 2 {
			recommendation = "Review carefully - multiple creep indicators"
		} else {
			recommendation = "Proceed with caution - minor creep detected"
		}
	} else {
		recommendation = "No feature creep detected - clear to proceed"
	}

	// Build diff summary
	diffSummary := fmt.Sprintf("+%d/-%d lines", additions, deletions)
	if newFunctions > 0 {
		diffSummary += fmt.Sprintf(", %d new functions", newFunctions)
	}
	if newClasses > 0 {
		diffSummary += fmt.Sprintf(", %d new classes", newClasses)
	}
	if newImports > 0 {
		diffSummary += fmt.Sprintf(", %d new imports", newImports)
	}

	return models.FeatureCreepDetectionResult{
		CreepDetected: creepDetected,
		Violations:    violations,
		DiffSummary:   diffSummary,
		TotalChanges: struct {
			Additions int `json:"additions"`
			Deletions int `json:"deletions"`
		}{
			Additions: additions,
			Deletions: deletions,
		},
		Recommendation: recommendation,
	}
}

// handleVerifyFixesIntact verifies if previously applied fixes are still intact
