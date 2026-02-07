package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// FailureEntry represents an entry in the failure registry
type FailureEntry struct {
	ID                uuid.UUID      `json:"id" db:"id"`
	FailureID         string         `json:"failure_id" db:"failure_id"`
	Category          string         `json:"category" db:"category"`
	Severity          string         `json:"severity" db:"severity"`
	ErrorMessage      string         `json:"error_message" db:"error_message"`
	RootCause         string         `json:"root_cause" db:"root_cause"`
	AffectedFiles     pgtype.Array[string] `json:"affected_files" db:"affected_files"`
	RegressionPattern string         `json:"regression_pattern" db:"regression_pattern"`
	Status            string         `json:"status" db:"status"`
	ProjectSlug       string         `json:"project_slug" db:"project_slug"`
	CreatedAt         time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at" db:"updated_at"`
}

// FailureStatus represents the status of a failure entry
type FailureStatus string

const (
	StatusActive     FailureStatus = "active"
	StatusResolved   FailureStatus = "resolved"
	StatusDeprecated FailureStatus = "deprecated"
)

// IsValidFailureStatus checks if a status is valid
func IsValidFailureStatus(status string) bool {
	switch FailureStatus(status) {
	case StatusActive, StatusResolved, StatusDeprecated:
		return true
	}
	return false
}

// ToStringSlice converts pgtype.Array[string] to []string for convenience
func ToStringSlice(arr pgtype.Array[string]) []string {
	if !arr.Valid {
		return nil
	}
	return arr.Elements
}

// ToTextArray converts []string to pgtype.Array[string] for database storage
func ToTextArray(slice []string) pgtype.Array[string] {
	return pgtype.Array[string]{
		Elements: slice,
		Valid:    slice != nil,
	}
}
