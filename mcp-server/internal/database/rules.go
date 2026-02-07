package database

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/thearchitectit/guardrail-mcp/internal/models"
)

// RuleStore handles prevention rule database operations
type RuleStore struct {
	db *DB
}

// NewRuleStore creates a new rule store
func NewRuleStore(db *DB) *RuleStore {
	return &RuleStore{db: db}
}

// GetByID retrieves a rule by ID
func (s *RuleStore) GetByID(ctx context.Context, id uuid.UUID) (*models.PreventionRule, error) {
	var rule models.PreventionRule
	err := s.db.QueryRowContext(ctx, `
		SELECT id, rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category, created_at, updated_at
		FROM prevention_rules
		WHERE id = $1
	`, id).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Pattern, &rule.PatternHash,
		&rule.Message, &rule.Severity, &rule.Enabled, &rule.DocumentID,
		&rule.Category, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	return &rule, nil
}

// GetByRuleID retrieves a rule by rule_id
func (s *RuleStore) GetByRuleID(ctx context.Context, ruleID string) (*models.PreventionRule, error) {
	var rule models.PreventionRule
	err := s.db.QueryRowContext(ctx, `
		SELECT id, rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category, created_at, updated_at
		FROM prevention_rules
		WHERE rule_id = $1
	`, ruleID).Scan(
		&rule.ID, &rule.RuleID, &rule.Name, &rule.Pattern, &rule.PatternHash,
		&rule.Message, &rule.Severity, &rule.Enabled, &rule.DocumentID,
		&rule.Category, &rule.CreatedAt, &rule.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found: %s", ruleID)
		}
		return nil, fmt.Errorf("failed to get rule: %w", err)
	}
	return &rule, nil
}

// List retrieves rules with optional filters
func (s *RuleStore) List(ctx context.Context, enabled *bool, category string) ([]models.PreventionRule, error) {
	var args []interface{}
	query := `
		SELECT id, rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category, created_at, updated_at
		FROM prevention_rules
		WHERE 1=1
	`

	argIndex := 1
	if enabled != nil {
		query += fmt.Sprintf(" AND enabled = $%d", argIndex)
		args = append(args, *enabled)
		argIndex++
	}

	if category != "" {
		query += fmt.Sprintf(" AND category = $%d", argIndex)
		args = append(args, category)
		argIndex++
	}

	query += " ORDER BY updated_at DESC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list rules: %w", err)
	}
	defer rows.Close()

	var rules []models.PreventionRule
	for rows.Next() {
		var rule models.PreventionRule
		err := rows.Scan(
			&rule.ID, &rule.RuleID, &rule.Name, &rule.Pattern, &rule.PatternHash,
			&rule.Message, &rule.Severity, &rule.Enabled, &rule.DocumentID,
			&rule.Category, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// GetActiveRules retrieves all enabled rules
func (s *RuleStore) GetActiveRules(ctx context.Context) ([]models.PreventionRule, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category, created_at, updated_at
		FROM prevention_rules
		WHERE enabled = true
		ORDER BY severity DESC, name ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to get active rules: %w", err)
	}
	defer rows.Close()

	var rules []models.PreventionRule
	for rows.Next() {
		var rule models.PreventionRule
		err := rows.Scan(
			&rule.ID, &rule.RuleID, &rule.Name, &rule.Pattern, &rule.PatternHash,
			&rule.Message, &rule.Severity, &rule.Enabled, &rule.DocumentID,
			&rule.Category, &rule.CreatedAt, &rule.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan rule: %w", err)
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// Create inserts a new rule
func (s *RuleStore) Create(ctx context.Context, rule *models.PreventionRule) error {
	return s.db.QueryRowContext(ctx, `
		INSERT INTO prevention_rules (rule_id, name, pattern, pattern_hash, message, severity, enabled, document_id, category)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at, updated_at
	`, rule.RuleID, rule.Name, rule.Pattern, rule.PatternHash, rule.Message,
		rule.Severity, rule.Enabled, rule.DocumentID, rule.Category,
	).Scan(&rule.ID, &rule.CreatedAt, &rule.UpdatedAt)
}

// Update updates an existing rule
func (s *RuleStore) Update(ctx context.Context, rule *models.PreventionRule) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE prevention_rules
		SET name = $1, pattern = $2, pattern_hash = $3, message = $4, severity = $5, enabled = $6, document_id = $7, category = $8, updated_at = NOW()
		WHERE id = $9
	`, rule.Name, rule.Pattern, rule.PatternHash, rule.Message, rule.Severity,
		rule.Enabled, rule.DocumentID, rule.Category, rule.ID)
	if err != nil {
		return fmt.Errorf("failed to update rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found: %s", rule.ID)
	}

	return nil
}

// Delete removes a rule
func (s *RuleStore) Delete(ctx context.Context, id uuid.UUID) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM prevention_rules WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}

// Toggle enables/disables a rule
func (s *RuleStore) Toggle(ctx context.Context, id uuid.UUID, enabled bool) error {
	result, err := s.db.ExecContext(ctx, `
		UPDATE prevention_rules SET enabled = $1, updated_at = NOW() WHERE id = $2
	`, enabled, id)
	if err != nil {
		return fmt.Errorf("failed to toggle rule: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("rule not found: %s", id)
	}

	return nil
}
