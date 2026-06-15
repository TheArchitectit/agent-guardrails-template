package budget

import (
	"time"
)

// Period defines the budget reset period.
type Period string

const (
	PeriodDaily   Period = "daily"
	PeriodWeekly  Period = "weekly"
	PeriodMonthly Period = "monthly"
)

// BudgetConfig defines spending limits for a team/model combination.
type BudgetConfig struct {
	ID              string    `json:"id"`
	TeamID          string    `json:"team_id"`
	ModelName       string    `json:"model_name"`
	MaxTokens       int64     `json:"max_tokens"`
	MaxCostCents    int64     `json:"max_cost_cents"`
	Period          Period    `json:"period"`
	AlertThreshold  float64   `json:"alert_threshold"` // 0.0-1.0, triggers event when exceeded
	Enabled         bool      `json:"enabled"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// BudgetEntry records a single token usage event.
type BudgetEntry struct {
	ID           string    `json:"id"`
	TeamID       string    `json:"team_id"`
	ModelName    string    `json:"model_name"`
	InputTokens  int64     `json:"input_tokens"`
	OutputTokens int64     `json:"output_tokens"`
	TotalTokens  int64     `json:"total_tokens"`
	CostCents    int64     `json:"cost_cents"`
	Endpoint     string    `json:"endpoint"`
	CreatedAt    time.Time `json:"created_at"`
}

// BudgetStatus reports current spend against a budget limit.
type BudgetStatus struct {
	TeamID       string  `json:"team_id"`
	ModelName    string  `json:"model_name"`
	Period       Period  `json:"period"`
	TokensUsed   int64   `json:"tokens_used"`
	TokensMax    int64   `json:"tokens_max"`
	CostUsedCents int64  `json:"cost_used_cents"`
	CostMaxCents int64   `json:"cost_max_cents"`
	PercentUsed  float64 `json:"percent_used"`
	WithinBudget bool    `json:"within_budget"`
	ResetAt      string  `json:"reset_at"`
}
