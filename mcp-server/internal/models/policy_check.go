package models

import "time"

// PolicyCheckRequest is the request body for the /api/v1/policy/check endpoint.
type PolicyCheckRequest struct {
	// Input is the content to check against guardrail policies.
	Input string `json:"input"`
	// FilePath is the optional file path context for the check.
	FilePath string `json:"file_path,omitempty"`
	// Language is the optional programming language hint (e.g. "go", "python").
	Language string `json:"language,omitempty"`
	// Categories filters which rule categories to check.
	// Valid values: "bash", "git", "file_edit". Empty means all categories.
	Categories []string `json:"categories,omitempty"`
}

// PolicyCheckResponse is the response from the /api/v1/policy/check endpoint.
type PolicyCheckResponse struct {
	// Passed is true if no violations were found.
	Passed bool `json:"passed"`
	// Violations lists all policy violations found.
	Violations []PolicyViolation `json:"violations"`
	// CheckedAt is when the check was performed.
	CheckedAt time.Time `json:"checked_at"`
	// DurationMs is the elapsed time of the check in milliseconds.
	DurationMs int `json:"duration_ms"`
	// RulesCount is the number of rules evaluated.
	RulesCount int `json:"rules_evaluated"`
}

// PolicyViolation represents a single policy violation found during a check.
type PolicyViolation struct {
	// RuleID is the identifier of the rule that was violated.
	RuleID string `json:"rule_id"`
	// RuleName is the human-readable name of the rule.
	RuleName string `json:"rule_name"`
	// Severity is the violation severity: critical, error, warning, info.
	Severity string `json:"severity"`
	// Message is the human-readable violation message.
	Message string `json:"message"`
	// Category is the rule category (bash, git, file_edit).
	Category string `json:"category"`
	// MatchedPattern is the regex pattern that matched.
	MatchedPattern string `json:"matched_pattern,omitempty"`
	// Line is the 1-based line number where the violation was found (0 if not applicable).
	Line int `json:"line,omitempty"`
	// Column is the 1-based column number where the violation was found (0 if not applicable).
	Column int `json:"column,omitempty"`
}
