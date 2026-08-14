package mcp

import (
	"encoding/json"
	"os"
)

// getRepoPath returns the base path to the guardrails repository
// where docs/ and .guardrails/ directories are located.
// Uses GUARDRAILS_REPO_PATH env var, or current working directory.
func (s *MCPServer) getRepoPath() string {
	if path := os.Getenv("GUARDRAILS_REPO_PATH"); path != "" {
		return path
	}
	path, err := os.Getwd()
	if err != nil {
		return "."
	}
	return path
}

// jsonEscapeString escapes a string for safe embedding in JSON string literals.
func jsonEscapeString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return s
	}
	return string(b[1 : len(b)-1])
}
