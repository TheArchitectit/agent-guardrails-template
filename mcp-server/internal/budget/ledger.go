package budget

import (
	"context"
	"fmt"
	"time"
)

// Store defines the persistence interface for budget operations.
type Store interface {
	CreateConfig(ctx context.Context, config *BudgetConfig) error
	GetConfig(ctx context.Context, id string) (*BudgetConfig, error)
	GetConfigByTeamModel(ctx context.Context, teamID, modelName string) (*BudgetConfig, error)
	ListConfigsByTeam(ctx context.Context, teamID string) ([]*BudgetConfig, error)
	UpdateConfig(ctx context.Context, config *BudgetConfig) error
	DeleteConfig(ctx context.Context, id string) error
	RecordEntry(ctx context.Context, entry *BudgetEntry) error
	SumUsage(ctx context.Context, teamID, modelName string, since time.Time) (tokens int64, costCents int64, err error)
	ListEntries(ctx context.Context, teamID string, since time.Time, limit int) ([]*BudgetEntry, error)
}

// Ledger tracks token usage and enforces budgets.
type Ledger struct {
	store Store
}

// NewLedger creates a new budget ledger.
func NewLedger(store Store) *Ledger {
	return &Ledger{store: store}
}

// Record logs a token usage event.
func (l *Ledger) Record(ctx context.Context, entry *BudgetEntry) error {
	entry.TotalTokens = entry.InputTokens + entry.OutputTokens
	return l.store.RecordEntry(ctx, entry)
}

// GetStatus returns the current budget status for a team/model in the active period.
func (l *Ledger) GetStatus(ctx context.Context, teamID, modelName string) (*BudgetStatus, error) {
	config, err := l.store.GetConfigByTeamModel(ctx, teamID, modelName)
	if err != nil {
		return nil, fmt.Errorf("no budget config for %s/%s: %w", teamID, modelName, err)
	}

	since := periodStart(config.Period)
	tokensUsed, costUsed, err := l.store.SumUsage(ctx, teamID, modelName, since)
	if err != nil {
		return nil, fmt.Errorf("failed to sum usage: %w", err)
	}

	pctTokens := 0.0
	if config.MaxTokens > 0 {
		pctTokens = float64(tokensUsed) / float64(config.MaxTokens) * 100
	}
	pctCost := 0.0
	if config.MaxCostCents > 0 {
		pctCost = float64(costUsed) / float64(config.MaxCostCents) * 100
	}
	pct := pctTokens
	if pctCost > pct {
		pct = pctCost
	}

	within := true
	if config.MaxTokens > 0 && tokensUsed >= config.MaxTokens {
		within = false
	}
	if config.MaxCostCents > 0 && costUsed >= config.MaxCostCents {
		within = false
	}

	resetAt := nextReset(config.Period)

	return &BudgetStatus{
		TeamID:        teamID,
		ModelName:     modelName,
		Period:        config.Period,
		TokensUsed:    tokensUsed,
		TokensMax:     config.MaxTokens,
		CostUsedCents: costUsed,
		CostMaxCents:  config.MaxCostCents,
		PercentUsed:   pct,
		WithinBudget:  within,
		ResetAt:       resetAt.Format(time.RFC3339),
	}, nil
}

// CheckBudget returns nil if within budget, error if exceeded.
func (l *Ledger) CheckBudget(ctx context.Context, teamID, modelName string) error {
	status, err := l.GetStatus(ctx, teamID, modelName)
	if err != nil {
		return nil // No config = no limit
	}
	if !status.WithinBudget {
		return fmt.Errorf("budget exceeded for %s/%s: %.1f%% used (tokens: %d/%d, cost: %d/%d cents)",
			teamID, modelName, status.PercentUsed,
			status.TokensUsed, status.TokensMax,
			status.CostUsedCents, status.CostMaxCents)
	}
	return nil
}

// IsAlertThresholdCrossed checks if usage exceeds the alert threshold.
func (l *Ledger) IsAlertThresholdCrossed(ctx context.Context, teamID, modelName string) (bool, float64, error) {
	config, err := l.store.GetConfigByTeamModel(ctx, teamID, modelName)
	if err != nil {
		return false, 0, nil
	}

	since := periodStart(config.Period)
	tokensUsed, costUsed, err := l.store.SumUsage(ctx, teamID, modelName, since)
	if err != nil {
		return false, 0, err
	}

	pctTokens := 0.0
	if config.MaxTokens > 0 {
		pctTokens = float64(tokensUsed) / float64(config.MaxTokens)
	}
	pctCost := 0.0
	if config.MaxCostCents > 0 {
		pctCost = float64(costUsed) / float64(config.MaxCostCents)
	}
	pct := pctTokens
	if pctCost > pct {
		pct = pctCost
	}

	return pct >= config.AlertThreshold, pct, nil
}

// periodStart returns the start of the current budget period.
func periodStart(p Period) time.Time {
	now := time.Now().UTC()
	switch p {
	case PeriodDaily:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case PeriodWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(now.Year(), now.Month(), now.Day()-weekday+1, 0, 0, 0, 0, time.UTC)
	case PeriodMonthly:
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	}
}

// nextReset returns when the current budget period ends.
func nextReset(p Period) time.Time {
	start := periodStart(p)
	switch p {
	case PeriodDaily:
		return start.AddDate(0, 0, 1)
	case PeriodWeekly:
		return start.AddDate(0, 0, 7)
	case PeriodMonthly:
		return start.AddDate(0, 1, 0)
	default:
		return start.AddDate(0, 0, 1)
	}
}
