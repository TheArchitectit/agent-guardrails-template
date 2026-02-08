package models

import (
	"time"
)

// CheckResult represents the result of a pre-work check
type CheckResult struct {
	Passed        bool           `json:"passed"`
	Checks        []FailureCheck `json:"checks"`
	FilesAffected []string       `json:"files_affected"`
	Summary       string         `json:"summary,omitempty"`
}

// FailureCheck represents a single check result from the failure registry
type FailureCheck struct {
	FailureID         string   `json:"failure_id"`
	Category          string   `json:"category"`
	Severity          string   `json:"severity"`
	Message           string   `json:"message"`
	RootCause         string   `json:"root_cause,omitempty"`
	AffectedFiles     []string `json:"affected_files,omitempty"`
	RegressionPattern string   `json:"regression_pattern,omitempty"`
}

// CommitValidationResult represents the result of validating a commit message
type CommitValidationResult struct {
	Valid            bool     `json:"valid"`
	FormatCompliant  bool     `json:"format_compliant"`
	Issues           []string `json:"issues,omitempty"`
	Message          string   `json:"message,omitempty"`
	ConventionalType string   `json:"conventional_type,omitempty"`
	Scope            string   `json:"scope,omitempty"`
}

// ScopeValidationResult represents the result of validating a file scope
type ScopeValidationResult struct {
	Valid        bool   `json:"valid"`
	Message      string `json:"message"`
	FilePath     string `json:"file_path"`
	Scope        string `json:"scope"`
	OutsideScope bool   `json:"outside_scope,omitempty"`
}

// RegressionCheckResult represents the result of a regression check
type RegressionCheckResult struct {
	Matches []RegressionMatch `json:"matches"`
	Checked int               `json:"checked"`
}

// RegressionMatch represents a single regression pattern match
type RegressionMatch struct {
	FailureID         string   `json:"failure_id"`
	Category          string   `json:"category"`
	Severity          string   `json:"severity"`
	Message           string   `json:"message"`
	RootCause         string   `json:"root_cause"`
	RegressionPattern string   `json:"regression_pattern"`
	AffectedFiles     []string `json:"affected_files"`
}

// TestProdSeparationResult represents the result of test/production separation check
type TestProdSeparationResult struct {
	Valid       bool     `json:"valid"`
	Violations  []string `json:"violations,omitempty"`
	FilePath    string   `json:"file_path"`
	Environment string   `json:"environment"`
}

// PushValidationResult represents the result of validating a git push
type PushValidationResult struct {
	Valid    bool     `json:"valid"`
	CanPush  bool     `json:"can_push"`
	Warnings []string `json:"warnings,omitempty"`
	Branch   string   `json:"branch"`
	IsForce  bool     `json:"is_force"`
}

// MetaInfo contains metadata about the validation (used by some handlers)
type MetaInfo struct {
	CheckedAt      time.Time `json:"checked_at"`
	RulesEvaluated int       `json:"rules_evaluated"`
	DurationMs     int       `json:"duration_ms"`
	Command        string    `json:"command,omitempty"`
	File           string    `json:"file,omitempty"`
	ChangesSize    int       `json:"changes_size,omitempty"`
}
