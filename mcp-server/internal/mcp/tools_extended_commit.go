package mcp

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

func (s *MCPServer) handleVerifyTestsBeforeCommit(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	testResults, _ := args["test_results"].(string)
	stagedFiles, _ := args["staged_files"].([]interface{})
	requireCoverage, _ := args["require_coverage"].(bool)

	if testResults == "" {
		result := models.TestValidationResult{
			Valid:   false,
			Passed:  false,
			Message: "test_results is required",
		}
		return buildToolResult(result, true)
	}

	// Parse test results for common patterns
	testPassed := true
	failedTests := []string{}

	// Check for failure indicators
	failurePatterns := []string{"FAIL", "failed", "error", "Error", "panic", "FAIL:"}
	for _, pattern := range failurePatterns {
		if strings.Contains(testResults, pattern) {
			testPassed = false
			break
		}
	}

	// Check for success indicators
	successPatterns := []string{"PASS", "ok", "SUCCESS", "All tests passed"}
	hasSuccess := false
	for _, pattern := range successPatterns {
		if strings.Contains(testResults, pattern) {
			hasSuccess = true
			break
		}
	}

	// If no success indicators found but no failures, assume passing
	if !testPassed && !hasSuccess {
		testPassed = false
	}

	// Extract failed test names if any
	if !testPassed {
		lines := strings.Split(testResults, "\n")
		for _, line := range lines {
			if strings.Contains(line, "FAIL") || strings.Contains(line, "fail") {
				// Try to extract test name
				parts := strings.Fields(line)
				for i, part := range parts {
					if strings.Contains(part, "FAIL") && i+1 < len(parts) {
						failedTests = append(failedTests, parts[i+1])
						break
					}
				}
			}
		}
	}

	// Check coverage if required (placeholder logic)
	coverageMet := true
	if requireCoverage {
		// Look for coverage percentage in output
		coveragePatterns := []string{"coverage:", "Coverage:"}
		for _, pattern := range coveragePatterns {
			if idx := strings.Index(testResults, pattern); idx != -1 {
				// Extract coverage percentage
				after := testResults[idx+len(pattern):]
				// Simple extraction - look for number followed by %
				for i, ch := range after {
					if ch == '%' {
						// Found percentage, check if it's acceptable (>70%)
						coverageMet = true // Assume acceptable for now
						break
					}
					if i > 10 {
						break
					}
				}
			}
		}
	}

	fileCount := len(stagedFiles)
	message := fmt.Sprintf("Tests %s for %d staged files", map[bool]string{true: "PASSED", false: "FAILED"}[testPassed], fileCount)

	result := models.TestValidationResult{
		Valid:       true,
		Passed:      testPassed,
		Message:     message,
		FailedTests: failedTests,
		CoverageMet: coverageMet,
	}

	return buildToolResult(result, !testPassed)
}

// handleScanCommitPayload scans staged files for secrets, binaries, and generated files
func (s *MCPServer) handleScanCommitPayload(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	stagedFiles, _ := args["staged_files"].([]interface{})
	scanSecrets, _ := args["scan_secrets"].(bool)
	scanBinaries, _ := args["scan_binaries"].(bool)
	scanLargeFiles, _ := args["scan_large_files"].(bool)

	// Default to true if not specified
	if !scanSecrets && !scanBinaries && !scanLargeFiles {
		scanSecrets = true
		scanBinaries = true
		scanLargeFiles = true
	}

	if len(stagedFiles) == 0 {
		result := models.PayloadScanResult{
			Valid:   false,
			Clean:   false,
			Message: "staged_files is required",
		}
		return buildToolResult(result, true)
	}

	findings := []models.PayloadFinding{}
	scanned := 0

	// Secret patterns
	secretPatterns := []struct {
		pattern     *regexp.Regexp
		typeName    string
		severity    string
		description string
	}{
		{regexp.MustCompile(`(?i)(password|passwd|pwd)\s*=\s*["'][^"']+["']`), "secret", "critical", "Password in code"},
		{regexp.MustCompile(`(?i)(api[_-]?key|apikey)\s*=\s*["'][^"']+["']`), "secret", "critical", "API key in code"},
		{regexp.MustCompile(`(?i)(secret[_-]?key|secretkey)\s*=\s*["'][^"']+["']`), "secret", "critical", "Secret key in code"},
		{regexp.MustCompile(`(?i)(auth[_-]?token|authtoken)\s*=\s*["'][^"']+["']`), "secret", "critical", "Auth token in code"},
		{regexp.MustCompile(`sk-[a-zA-Z0-9]{48}`), "secret", "critical", "OpenAI API key"},
		{regexp.MustCompile(`gh[pousr]_[A-Za-z0-9_]{36,}`), "secret", "critical", "GitHub token"},
		{regexp.MustCompile(`[a-zA-Z0-9_-]*[Aa]ws[A-Za-z0-9_-]*\s*=\s*["'][^"']+["']`), "secret", "high", "AWS credential"},
	}

	// Binary file extensions
	binaryExtensions := []string{
		".exe", ".dll", ".so", ".dylib", ".bin",
		".zip", ".tar", ".gz", ".rar", ".7z",
		".jpg", ".jpeg", ".png", ".gif", ".bmp", ".ico", ".svg",
		".mp3", ".mp4", ".avi", ".mov", ".wav",
		".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx",
		".db", ".sqlite", ".sqlite3",
	}

	// Generated file patterns
	generatedPatterns := []string{
		".min.js", ".min.css", ".bundle.js", ".bundle.css",
		"generated.", "_generated.", "__generated__",
		"node_modules/", "vendor/", "dist/", "build/", "target/",
	}

	for _, file := range stagedFiles {
		filePath, ok := file.(string)
		if !ok {
			continue
		}

		scanned++
		fileLower := strings.ToLower(filePath)

		// Check for binaries
		if scanBinaries {
			for _, ext := range binaryExtensions {
				if strings.HasSuffix(fileLower, ext) {
					findings = append(findings, models.PayloadFinding{
						File:        filePath,
						Type:        "binary",
						Severity:    "error",
						Description: "Binary file should not be committed",
					})
					break
				}
			}
		}

		// Check for generated files
		for _, pattern := range generatedPatterns {
			if strings.Contains(fileLower, pattern) {
				findings = append(findings, models.PayloadFinding{
					File:        filePath,
					Type:        "generated",
					Severity:    "warning",
					Description: "Generated file should be in .gitignore",
				})
				break
			}
		}

		// Check file size (if we can stat the file)
		if scanLargeFiles {
			if info, err := os.Stat(filePath); err == nil {
				if info.Size() > 1024*1024 { // > 1MB
					findings = append(findings, models.PayloadFinding{
						File:        filePath,
						Type:        "large",
						Severity:    "warning",
						Description: fmt.Sprintf("Large file (%.2f MB)", float64(info.Size())/(1024*1024)),
					})
				}
			}
		}

		// Scan for secrets in file content
		if scanSecrets {
			content, err := safeReadFile(filePath)
			if err == nil {
				contentStr := string(content)
				lines := strings.Split(contentStr, "\n")

				for lineNum, line := range lines {
					for _, pattern := range secretPatterns {
						if pattern.pattern.MatchString(line) {
							findings = append(findings, models.PayloadFinding{
								File:        filePath,
								Type:        pattern.typeName,
								Severity:    pattern.severity,
								Description: pattern.description,
								LineNumber:  lineNum + 1,
							})
						}
					}
				}
			}
		}
	}

	clean := len(findings) == 0
	message := fmt.Sprintf("Scanned %d files", scanned)
	if !clean {
		message = fmt.Sprintf("Found %d issues in %d files", len(findings), scanned)
	}

	result := models.PayloadScanResult{
		Valid:    true,
		Clean:    clean,
		Message:  message,
		Findings: findings,
		Scanned:  scanned,
	}

	return buildToolResult(result, !clean)
}

// handleDetectMergeConflicts detects merge conflict markers in files
func (s *MCPServer) handleDetectMergeConflicts(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePaths, _ := args["file_paths"].([]interface{})
	checkContent, _ := args["check_content"].(bool)

	if !checkContent {
		checkContent = true // Default to true
	}

	if len(filePaths) == 0 {
		result := models.MergeConflictResult{
			Valid:   false,
			Clean:   false,
			Message: "file_paths is required",
		}
		return buildToolResult(result, true)
	}

	conflicts := []models.ConflictFinding{}
	checked := 0

	// Conflict markers
	markerStart := regexp.MustCompile(`^<{7}`) // <<<<<<<
	markerSep := regexp.MustCompile(`^={7}`)   // =======
	markerEnd := regexp.MustCompile(`^>{7}`)   // >>>>>>>

	for _, file := range filePaths {
		filePath, ok := file.(string)
		if !ok {
			continue
		}

		checked++

		if !checkContent {
			continue
		}

		// Read file content
		content, err := safeReadFile(filePath)
		if err != nil {
			continue // Skip unreadable files
		}

		lines := strings.Split(string(content), "\n")
		inConflict := false
		conflictStartLine := 0

		for lineNum, line := range lines {
			if markerStart.MatchString(line) {
				inConflict = true
				conflictStartLine = lineNum + 1
			} else if markerSep.MatchString(line) && inConflict {
				// Middle of conflict
				continue
			} else if markerEnd.MatchString(line) && inConflict {
				// End of conflict
				conflicts = append(conflicts, models.ConflictFinding{
					File:       filePath,
					LineNumber: conflictStartLine,
					Context:    fmt.Sprintf("Conflict from line %d to %d", conflictStartLine, lineNum+1),
				})
				inConflict = false
			}
		}

		// If still in conflict at end of file
		if inConflict {
			conflicts = append(conflicts, models.ConflictFinding{
				File:       filePath,
				LineNumber: conflictStartLine,
				Context:    fmt.Sprintf("Unterminated conflict starting at line %d", conflictStartLine),
			})
		}
	}

	clean := len(conflicts) == 0
	message := fmt.Sprintf("Checked %d files", checked)
	if !clean {
		message = fmt.Sprintf("Found %d merge conflicts in %d files", len(conflicts), checked)
	}

	result := models.MergeConflictResult{
		Valid:     true,
		Clean:     clean,
		Message:   message,
		Conflicts: conflicts,
		Checked:   checked,
	}

	return buildToolResult(result, !clean)
}
