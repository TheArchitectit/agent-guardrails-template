package mcp

import (
	"regexp"
	"strings"
)

// readFileIfExists reads a file if it exists, returns empty string otherwise
func readFileIfExists(path string) string {
	// This is a simplified version - in production, this would check if file
	// is within authorized scope and handle errors properly
	return ""
}

// contains checks if a string slice contains a value
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// isPatternMatch checks if text matches a glob pattern
func isPatternMatch(text, pattern string) bool {
	// Simple glob matching - convert glob to regex
	// * -> .*
	// ? -> .
	regexPattern := regexp.QuoteMeta(pattern)
	regexPattern = strings.ReplaceAll(regexPattern, `\*`, `.*`)
	regexPattern = strings.ReplaceAll(regexPattern, `\?`, `.`)
	regexPattern = "^" + regexPattern + "$"

	re, err := regexp.Compile(regexPattern)
	if err != nil {
		return false
	}

	return re.MatchString(text)
}
