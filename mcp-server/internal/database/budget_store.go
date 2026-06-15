package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/budget"
)

// BudgetStore handles budget config and entry database operations.
type BudgetStore struct {
	db *DB
}

// NewBudgetStore creates a new budget store.
func NewBudgetStore(db *DB) *BudgetStore {
	return &BudgetStore{db: db}
}

// CreateConfig inserts a new budget configuration.
func (s *BudgetStore) CreateConfig(ctx context.Context, config *budget.BudgetConfig) error {
	config.ID = generateUUID()
	now := time.Now().UTC()
	config.CreatedAt = now
	config.UpdatedAt = now

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO budget_configs (id, team_id, model_name, max_tokens, max_cost_cents, period, alert_threshold, enabled, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`, config.ID, config.TeamID, config.ModelName, config.MaxTokens, config.MaxCostCents,
		string(config.Period), config.AlertThreshold, config.Enabled, config.CreatedAt, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to create budget config: %w", err)
	}
	return nil
}

// GetConfig retrieves a budget config by ID.
func (s *BudgetStore) GetConfig(ctx context.Context, id string) (*budget.BudgetConfig, error) {
	var config budget.BudgetConfig
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, model_name, max_tokens, max_cost_cents, period, alert_threshold, enabled, created_at, updated_at
		FROM budget_configs WHERE id = $1
	`, id).Scan(
		&config.ID, &config.TeamID, &config.ModelName, &config.MaxTokens, &config.MaxCostCents,
		&config.Period, &config.AlertThreshold, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("budget config not found: %s", id)
		}
		return nil, fmt.Errorf("failed to get budget config: %w", err)
	}
	return &config, nil
}

// GetConfigByTeamModel retrieves a budget config by team and model.
func (s *BudgetStore) GetConfigByTeamModel(ctx context.Context, teamID, modelName string) (*budget.BudgetConfig, error) {
	var config budget.BudgetConfig
	err := s.db.QueryRowContext(ctx, `
		SELECT id, team_id, model_name, max_tokens, max_cost_cents, period, alert_threshold, enabled, created_at, updated_at
		FROM budget_configs WHERE team_id = $1 AND model_name = $2 AND enabled = true
	`, teamID, modelName).Scan(
		&config.ID, &config.TeamID, &config.ModelName, &config.MaxTokens, &config.MaxCostCents,
		&config.Period, &config.AlertThreshold, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("no active budget config for %s/%s", teamID, modelName)
		}
		return nil, fmt.Errorf("failed to get budget config: %w", err)
	}
	return &config, nil
}

// ListConfigsByTeam retrieves all budget configs for a team.
func (s *BudgetStore) ListConfigsByTeam(ctx context.Context, teamID string) ([]*budget.BudgetConfig, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, model_name, max_tokens, max_cost_cents, period, alert_threshold, enabled, created_at, updated_at
		FROM budget_configs WHERE team_id = $1 ORDER BY model_name
	`, teamID)
	if err != nil {
		return nil, fmt.Errorf("failed to list budget configs: %w", err)
	}
	defer rows.Close()

	var configs []*budget.BudgetConfig
	for rows.Next() {
		var config budget.BudgetConfig
		if err := rows.Scan(
			&config.ID, &config.TeamID, &config.ModelName, &config.MaxTokens, &config.MaxCostCents,
			&config.Period, &config.AlertThreshold, &config.Enabled, &config.CreatedAt, &config.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan budget config: %w", err)
		}
		configs = append(configs, &config)
	}
	return configs, rows.Err()
}

// UpdateConfig modifies an existing budget configuration.
func (s *BudgetStore) UpdateConfig(ctx context.Context, config *budget.BudgetConfig) error {
	config.UpdatedAt = time.Now().UTC()
	result, err := s.db.ExecContext(ctx, `
		UPDATE budget_configs
		SET max_tokens = $2, max_cost_cents = $3, period = $4, alert_threshold = $5, enabled = $6, updated_at = $7
		WHERE id = $1
	`, config.ID, config.MaxTokens, config.MaxCostCents, string(config.Period),
		config.AlertThreshold, config.Enabled, config.UpdatedAt)
	if err != nil {
		return fmt.Errorf("failed to update budget config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("budget config not found: %s", config.ID)
	}
	return nil
}

// DeleteConfig removes a budget configuration.
func (s *BudgetStore) DeleteConfig(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM budget_configs WHERE id = $1`, id)
	if err != nil {
		return fmt.Errorf("failed to delete budget config: %w", err)
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("budget config not found: %s", id)
	}
	return nil
}

// RecordEntry inserts a token usage entry.
func (s *BudgetStore) RecordEntry(ctx context.Context, entry *budget.BudgetEntry) error {
	if entry.ID == "" {
		entry.ID = generateUUID()
	}
	entry.CreatedAt = time.Now().UTC()

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO budget_entries (id, team_id, model_name, input_tokens, output_tokens, total_tokens, cost_cents, endpoint, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, entry.ID, entry.TeamID, entry.ModelName, entry.InputTokens, entry.OutputTokens,
		entry.TotalTokens, entry.CostCents, entry.Endpoint, entry.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to record budget entry: %w", err)
	}
	return nil
}

// SumUsage returns total tokens and cost for a team/model since the given time.
func (s *BudgetStore) SumUsage(ctx context.Context, teamID, modelName string, since time.Time) (tokens int64, costCents int64, err error) {
	err = s.db.QueryRowContext(ctx, `
		SELECT COALESCE(SUM(total_tokens), 0), COALESCE(SUM(cost_cents), 0)
		FROM budget_entries
		WHERE team_id = $1 AND model_name = $2 AND created_at >= $3
	`, teamID, modelName, since).Scan(&tokens, &costCents)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to sum usage: %w", err)
	}
	return tokens, costCents, nil
}

// ListEntries returns recent budget entries for a team.
func (s *BudgetStore) ListEntries(ctx context.Context, teamID string, since time.Time, limit int) ([]*budget.BudgetEntry, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, team_id, model_name, input_tokens, output_tokens, total_tokens, cost_cents, endpoint, created_at
		FROM budget_entries
		WHERE team_id = $1 AND created_at >= $2
		ORDER BY created_at DESC
		LIMIT $3
	`, teamID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to list entries: %w", err)
	}
	defer rows.Close()

	var entries []*budget.BudgetEntry
	for rows.Next() {
		var e budget.BudgetEntry
		if err := rows.Scan(
			&e.ID, &e.TeamID, &e.ModelName, &e.InputTokens, &e.OutputTokens,
			&e.TotalTokens, &e.CostCents, &e.Endpoint, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan entry: %w", err)
		}
		entries = append(entries, &e)
	}
	return entries, rows.Err()
}
