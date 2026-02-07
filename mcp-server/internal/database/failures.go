package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// FailureStore handles failure registry database operations
type FailureStore struct {
	db *DB
}

// NewFailureStore creates a new failure store
func NewFailureStore(db *DB) *FailureStore {
	return &FailureStore{db: db}
}

// GetByID retrieves a failure by ID
func (s *FailureStore) GetByID(ctx context.Context, id uuid.UUID) (*models.FailureEntry, error) {
	var f models.FailureEntry
	err := s.db.QueryRowContext(ctx, `
		SELECT id, failure_id, category, severity, error_message, root_cause, affected_files, regression_pattern, status, project_slug, created_at, updated_at
		FROM failure_registry
		WHERE id = $1
	`, id).Scan(
		&f.ID, &f.FailureID, &f.Category, &f.Severity, &f.ErrorMessage,
		&f.RootCause, &f.AffectedFiles, &f.RegressionPattern, &f.Status,
		&f.ProjectSlug, &f.CreatedAt, &f.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("failure not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get failure: %w", err)
	}
	return &f, nil
}

// List retrieves failures with optional filters
func (s *FailureStore) List(ctx context.Context, status, category, projectSlug string, limit, offset int) ([]models.FailureEntry, error) {
	var args []interface{}
	query := `
		SELECT id, failure_id, category, severity, error_message, root_cause, affected_files, regression_pattern, status, project_slug, created_at, updated_at
		FROM failure_registry
		WHERE 1=1
	`

	argIndex := 1
	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	if projectSlug != "" {
		query += fmt.Sprintf(" AND project_slug = $%d", argIndex)
		args = append(args, projectSlug)
		argIndex++
	}

	query += fmt.Sprintf(" ORDER BY created_at DESC LIMIT $%d OFFSET $%d", argIndex, argIndex+1)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list failures: %w", err)
	}
	defer rows.Close()

	var failures []models.FailureEntry
	for rows.Next() {
		var f models.FailureEntry
		err := rows.Scan(
			&f.ID, &f.FailureID, &f.Category, &f.Severity, &f.ErrorMessage,
			&f.RootCause, &f.AffectedFiles, &f.RegressionPattern, &f.Status,
			&f.ProjectSlug, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan failure: %w", err)
		}
		failures = append(failures, f)
	}

	return failures, rows.Err()
}

// Create inserts a new failure
func (s *FailureStore) Create(ctx context.Context, f *models.FailureEntry) error {
	return s.db.QueryRowContext(ctx, `
		INSERT INTO failure_registry (failure_id, category, severity, error_message, root_cause, affected_files, regression_pattern, status, project_slug)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, f.FailureID, f.Category, f.Severity, f.ErrorMessage, f.RootCause,
		f.AffectedFiles, f.RegressionPattern, f.Status, f.ProjectSlug,
	).Scan(&f.ID, &f.CreatedAt, &f.UpdatedAt)
}

// Update updates an existing failure
func (s *FailureStore) Update(ctx context.Context, f *models.FailureEntry) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE failure_registry
		SET error_message = $1, root_cause = $2, affected_files = $3, regression_pattern = $4, status = $5, updated_at = NOW()
		WHERE id = $6
	`, f.ErrorMessage, f.RootCause, f.AffectedFiles, f.RegressionPattern, f.Status, f.ID)
	if err != nil {
		return fmt.Errorf("failed to update failure: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("failure not found: %s", f.ID)
	}

	return nil
}

// GetActiveByFiles retrieves active failures that affect given files
func (s *FailureStore) GetActiveByFiles(ctx context.Context, files []string) ([]models.FailureEntry, error) {
	if len(files) == 0 {
		return []models.FailureEntry{}, nil
	}

	// Query for failures where affected_files overlaps with input files
	query := `
		SELECT id, failure_id, category, severity, error_message, root_cause, affected_files, regression_pattern, status, project_slug, created_at, updated_at
		FROM failure_registry
		WHERE status = 'active'
		AND affected_files && $1
		ORDER BY severity DESC, created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, files)
	if err != nil {
		return nil, fmt.Errorf("failed to get active failures: %w", err)
	}
	defer rows.Close()

	var failures []models.FailureEntry
	for rows.Next() {
		var f models.FailureEntry
		err := rows.Scan(
			&f.ID, &f.FailureID, &f.Category, &f.Severity, &f.ErrorMessage,
			&f.RootCause, &f.AffectedFiles, &f.RegressionPattern, &f.Status,
			&f.ProjectSlug, &f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan failure: %w", err)
		}
		failures = append(failures, f)
	}

	return failures, rows.Err()
}

// Count returns the total count of failures
func (s *FailureStore) Count(ctx context.Context) (int64, error) {
	var count int64
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM failure_registry`).Scan(&count)
	return count, err
}
