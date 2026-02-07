package models

import (
	"time"

	"github.com/google/uuid"
)

// PreventionRule represents a guardrail prevention rule
type PreventionRule struct {
	ID           uuid.UUID `json:"id" db:"id"`
	RuleID       string    `json:"rule_id" db:"rule_id"`
	Name         string    `json:"name" db:"name"`
	Pattern      string    `json:"pattern" db:"pattern"`
	PatternHash  string    `json:"pattern_hash" db:"pattern_hash"`
	Message      string    `json:"message" db:"message"`
	Severity     Severity  `json:"severity" db:"severity"`
	Enabled      bool      `json:"enabled" db:"enabled"`
	DocumentID   *uuid.UUID `json:"document_id,omitempty" db:"document_id"`
	Category     string    `json:"category" db:"category"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// Severity represents rule severity levels
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// IsValidSeverity checks if a severity level is valid
func IsValidSeverity(sev string) bool {
	switch Severity(sev) {
	case SeverityError, SeverityWarning, SeverityInfo:
		return true
	}
	return false
}

// Action returns the recommended action for a severity level
func (s Severity) Action() string {
	switch s {
	case SeverityError:
		return "halt"
	case SeverityWarning:
		return "confirm"
	case SeverityInfo:
		return "log"
	default:
		return "log"
	}
}
