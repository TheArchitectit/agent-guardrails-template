package models

import (
	"time"

	"github.com/google/uuid"
)

// Document represents a guardrail document stored in the database
type Document struct {
	ID           uuid.UUID       `json:"id" db:"id"`
	Slug         string          `json:"slug" db:"slug"`
	Title        string          `json:"title" db:"title"`
	Content      string          `json:"content" db:"content"`
	SearchVector string          `json:"-" db:"search_vector"`
	Category     string          `json:"category" db:"category"`
	Path         string          `json:"path" db:"path"`
	Version      int             `json:"version" db:"version"`
	Metadata     map[string]any  `json:"metadata" db:"metadata"`
	CreatedAt    time.Time       `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time       `json:"updated_at" db:"updated_at"`
}

// DocumentCategory represents valid document categories
type DocumentCategory string

const (
	CategoryWorkflow DocumentCategory = "workflow"
	CategoryStandard DocumentCategory = "standard"
	CategoryGuide    DocumentCategory = "guide"
	CategoryReference DocumentCategory = "reference"
)

// IsValidCategory checks if a category is valid
func IsValidCategory(cat string) bool {
	switch DocumentCategory(cat) {
	case CategoryWorkflow, CategoryStandard, CategoryGuide, CategoryReference:
		return true
	}
	return false
}
