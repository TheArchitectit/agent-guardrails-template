package budget

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/thearchitectit/guardrail-mcp/internal/domain"
)

// Governor wraps the Ledger and Estimator to provide budget-aware usage recording.
// It checks budgets before recording, emits events when thresholds are crossed,
// and returns errors when limits are exceeded.
type Governor struct {
	ledger  *Ledger
	estimator func(modelName string, inputTokens, outputTokens int64) int64
	bus     domain.EventBus
}

// NewGovernor creates a new budget governor.
func NewGovernor(ledger *Ledger, bus domain.EventBus) *Governor {
	return &Governor{
		ledger:    ledger,
		estimator: EstimateCost,
		bus:       bus,
	}
}

// CheckAndRecord checks if the team/model is within budget, records usage, and emits alerts.
// Returns an error if the budget is exceeded (callers should halt or throttle).
func (g *Governor) CheckAndRecord(ctx context.Context, teamID, modelName string, inputTokens, outputTokens int64, endpoint string) error {
	// Check budget before recording
	if err := g.ledger.CheckBudget(ctx, teamID, modelName); err != nil {
		return err
	}

	costCents := g.estimator(modelName, inputTokens, outputTokens)

	entry := &BudgetEntry{
		TeamID:       teamID,
		ModelName:    modelName,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		CostCents:    costCents,
		Endpoint:     endpoint,
	}

	if err := g.ledger.Record(ctx, entry); err != nil {
		return fmt.Errorf("failed to record budget entry: %w", err)
	}

	// Check if alert threshold crossed
	crossed, pct, err := g.ledger.IsAlertThresholdCrossed(ctx, teamID, modelName)
	if err != nil {
		slog.Warn("failed to check budget threshold", "error", err)
		return nil
	}

	if crossed && g.bus != nil {
		g.bus.Publish(ctx, domain.Event{
			Type: domain.EventBudgetExceeded,
			Payload: map[string]interface{}{
				"team_id":     teamID,
				"model_name":  modelName,
				"percent_used": pct,
				"tokens":      inputTokens + outputTokens,
				"cost_cents":  costCents,
			},
		})
		slog.Warn("budget alert threshold crossed",
			"team_id", teamID,
			"model", modelName,
			"percent", fmt.Sprintf("%.1f%%", pct*100),
		)
	}

	return nil
}

// Status returns the current budget status for a team/model.
func (g *Governor) Status(ctx context.Context, teamID, modelName string) (*BudgetStatus, error) {
	return g.ledger.GetStatus(ctx, teamID, modelName)
}
