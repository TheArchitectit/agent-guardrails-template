package models

import (
	"time"

	"github.com/google/uuid"
)

// Project represents a project with guardrail configuration
type Project struct {
	ID           uuid.UUID      `json:"id" db:"id"`
	Name         string         `json:"name" db:"name"`
	Slug         string         `json:"slug" db:"slug"`
	GuardrailContext string     `json:"guardrail_context" db:"guardrail_context"`
	ActiveRules  []string       `json:"active_rules" db:"active_rules"`
	Metadata     map[string]any `json:"metadata" db:"metadata"`
	CreatedAt    time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at" db:"updated_at"`
}

// Violation represents a guardrail violation found during validation
type Violation struct {
	RuleID               string `json:"rule_id"`
	RuleName             string `json:"rule_name"`
	Severity             string `json:"severity"`
	Message              string `json:"message"`
	Category             string `json:"category"`
	Action               string `json:"action"`
	SuggestedAlternative string `json:"suggested_alternative,omitempty"`
	DocumentationURI     string `json:"documentation_uri,omitempty"`
	Line                 int    `json:"line,omitempty"`
	Column               int    `json:"column,omitempty"`
}

// ValidationResult represents the result of a validation check
type ValidationResult struct {
	Valid      bool        `json:"valid"`
	Violations []Violation `json:"violations"`
	Meta       ValidationMeta `json:"meta"`
}

// ValidationMeta contains metadata about the validation
type ValidationMeta struct {
	CheckedAt       time.Time `json:"checked_at"`
	RulesEvaluated  int       `json:"rules_evaluated"`
	DurationMs      int64     `json:"duration_ms"`
	Cached          bool      `json:"cached"`
}

// Session represents an MCP client session
type Session struct {
	Token       string    `json:"token"`
	ProjectSlug string    `json:"project_slug"`
	AgentType   string    `json:"agent_type"`
	ClientVersion string  `json:"client_version"`
	CreatedAt   time.Time `json:"created_at"`
	ExpiresAt   time.Time `json:"expires_at"`
}
