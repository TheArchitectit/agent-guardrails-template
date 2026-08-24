package mcp

import (
	"context"
	"fmt"
	"strings"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// handleAdvisorList returns all available advisors
func (s *MCPServer) handleAdvisorList(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	advisors := models.StandardAdvisors()

	advisorList := make([]models.Advisor, 0, len(advisors))
	for _, advisor := range advisors {
		advisorList = append(advisorList, advisor)
	}

	result := models.AdvisorListResult{
		Advisors: advisorList,
		Count:    len(advisorList),
	}

	return buildToolResult(result, false)
}

// handleAdvisorTriggerCheck checks if code changes trigger an advisor consultation
func (s *MCPServer) handleAdvisorTriggerCheck(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	filePathsArg, _ := args["file_paths"].([]interface{})
	fileDiffsArg, _ := args["file_diffs"].(map[string]interface{})

	if len(filePathsArg) == 0 {
		result := models.AdvisorTriggerResult{
			Triggered: []models.TriggeredAdvisor{},
			Count:     0,
		}
		return buildToolResult(result, false)
	}

	// Convert file paths
	filePaths := make([]string, 0, len(filePathsArg))
	for _, fp := range filePathsArg {
		if path, ok := fp.(string); ok {
			filePaths = append(filePaths, path)
		}
	}

	// Convert file diffs
	fileDiffs := make(map[string]string)
	for path, diff := range fileDiffsArg {
		if diffStr, ok := diff.(string); ok {
			fileDiffs[path] = diffStr
		}
	}

	// Check each advisor
	advisors := models.StandardAdvisors()
	triggered := make([]models.TriggeredAdvisor, 0)

	for _, advisor := range advisors {
		if isTriggered, matchedPatterns, reason := checkAdvisorTriggers(advisor, filePaths, fileDiffs); isTriggered {
			triggered = append(triggered, models.TriggeredAdvisor{
				ID:               advisor.ID,
				Name:             advisor.Name,
				EnforcementLevel: advisor.EnforcementLevel,
				MatchedPatterns:  matchedPatterns,
				Reason:           reason,
			})
		}
	}

	result := models.AdvisorTriggerResult{
		Triggered: triggered,
		Count:     len(triggered),
	}

	return buildToolResult(result, false)
}

// checkAdvisorTriggers checks if an advisor should be triggered
func checkAdvisorTriggers(advisor models.Advisor, filePaths []string, fileDiffs map[string]string) (bool, []string, string) {
	matchedPatterns := make([]string, 0)
	reasons := make([]string, 0)

	for _, pattern := range advisor.TriggerPatterns {
		// Remove wildcards for matching
		cleanPattern := strings.Trim(pattern, "*")

		// Check file paths
		for _, path := range filePaths {
			if strings.Contains(strings.ToLower(path), strings.ToLower(cleanPattern)) {
				if !contains(matchedPatterns, pattern) {
					matchedPatterns = append(matchedPatterns, pattern)
					reasons = append(reasons, fmt.Sprintf("File path '%s' matches pattern '%s'", path, pattern))
				}
			}
		}

		// Check file diffs if available
		for path, diff := range fileDiffs {
			if strings.Contains(strings.ToLower(diff), strings.ToLower(cleanPattern)) {
				if !contains(matchedPatterns, pattern) {
					matchedPatterns = append(matchedPatterns, pattern)
					reasons = append(reasons, fmt.Sprintf("Diff in '%s' matches pattern '%s'", path, pattern))
				}
			}
		}
	}

	if len(matchedPatterns) > 0 {
		return true, matchedPatterns, strings.Join(reasons, "; ")
	}

	return false, nil, ""
}

// handleAdvisorConsult gets advice from a specific advisor
func (s *MCPServer) handleAdvisorConsult(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	advisorID, _ := args["advisor_id"].(string)
	context, _ := args["context"].(string)
	filePathsArg, _ := args["file_paths"].([]interface{})

	if advisorID == "" {
		return buildToolResult(map[string]string{
			"error": "advisor_id is required",
		}, true)
	}

	advisors := models.StandardAdvisors()
	advisor, ok := advisors[advisorID]
	if !ok {
		return buildToolResult(map[string]string{
			"error": fmt.Sprintf("Advisor not found: %s", advisorID),
		}, true)
	}

	// Convert file paths
	filePaths := make([]string, 0, len(filePathsArg))
	for _, fp := range filePathsArg {
		if path, ok := fp.(string); ok {
			filePaths = append(filePaths, path)
		}
	}

	// Generate advice based on advisor type and context
	result := generateAdvisorResponse(advisor, context, filePaths)

	return buildToolResult(result, false)
}
