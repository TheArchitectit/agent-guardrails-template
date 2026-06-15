package mcp

import (
	"context"
	"fmt"
	"time"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/thearchitectit/guardrail-mcp/internal/budget"
)

// budgetToolList returns the tool definitions for budget management tools.
func (s *MCPServer) budgetToolList() []mcp.Tool {
	return []mcp.Tool{
		{
			Name:        "configure_budget",
			Description: "Set or update token budget limits for a team/model combination",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID to set budget for",
					},
					"model_name": map[string]interface{}{
						"type":        "string",
						"description": "Model name (e.g. claude-3-5-sonnet, gpt-4o, local-llama)",
					},
					"max_tokens": map[string]interface{}{
						"type":        "number",
						"description": "Maximum tokens allowed per period (0 = no limit)",
					},
					"max_cost_cents": map[string]interface{}{
						"type":        "number",
						"description": "Maximum cost in cents per period (0 = no limit)",
					},
					"period": map[string]interface{}{
						"type":        "string",
						"enum":        []string{"daily", "weekly", "monthly"},
						"description": "Budget reset period (default: daily)",
					},
					"alert_threshold": map[string]interface{}{
						"type":        "number",
						"description": "Usage fraction (0.0-1.0) that triggers an alert event (default: 0.8)",
					},
				},
				Required: []string{"team_id", "model_name"},
			},
		},
		{
			Name:        "get_budget_status",
			Description: "Get current token/cost usage against budget limits for a team/model",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID",
					},
					"model_name": map[string]interface{}{
						"type":        "string",
						"description": "Model name",
					},
				},
				Required: []string{"team_id", "model_name"},
			},
		},
		{
			Name:        "list_budgets",
			Description: "List all budget configurations for a team",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID to list budgets for",
					},
				},
				Required: []string{"team_id"},
			},
		},
		{
			Name:        "get_budget_history",
			Description: "Get token usage history for a team",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"team_id": map[string]interface{}{
						"type":        "string",
						"description": "Team ID",
					},
					"days": map[string]interface{}{
						"type":        "number",
						"description": "Number of days to look back (default: 7)",
					},
				},
				Required: []string{"team_id"},
			},
		},
		{
			Name:        "delete_budget",
			Description: "Delete a budget configuration",
			InputSchema: mcp.ToolInputSchema{
				Type: "object",
				Properties: mcp.ToolInputSchemaProperties{
					"budget_id": map[string]interface{}{
						"type":        "string",
						"description": "Budget config ID to delete",
					},
				},
				Required: []string{"budget_id"},
			},
		},
	}
}

func (s *MCPServer) handleConfigureBudget(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	modelName, _ := args["model_name"].(string)
	period, _ := args["period"].(string)
	alertThreshold, _ := args["alert_threshold"].(float64)

	if teamID == "" || modelName == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id and model_name are required",
		}, true)
	}

	maxTokens := int64(0)
	if mt, ok := args["max_tokens"].(float64); ok {
		maxTokens = int64(mt)
	}
	maxCostCents := int64(0)
	if mc, ok := args["max_cost_cents"].(float64); ok {
		maxCostCents = int64(mc)
	}

	if period == "" {
		period = "daily"
	}
	if alertThreshold <= 0 {
		alertThreshold = 0.8
	}

	budgetStore := s.budgetStore

	// Try to find existing config
	existing, err := budgetStore.GetConfigByTeamModel(ctx, teamID, modelName)
	if err == nil {
		// Update existing
		existing.MaxTokens = maxTokens
		existing.MaxCostCents = maxCostCents
		existing.Period = budget.Period(period)
		existing.AlertThreshold = alertThreshold
		existing.Enabled = true
		if err := budgetStore.UpdateConfig(ctx, existing); err != nil {
			return buildToolResult(map[string]interface{}{
				"error": fmt.Sprintf("failed to update budget: %v", err),
			}, true)
		}
		return buildToolResult(map[string]interface{}{
			"success":    true,
			"budget_id":  existing.ID,
			"message":    "Budget updated successfully",
			"team_id":    teamID,
			"model_name": modelName,
		}, false)
	}

	// Create new
	config := &budget.BudgetConfig{
		TeamID:         teamID,
		ModelName:      modelName,
		MaxTokens:      maxTokens,
		MaxCostCents:   maxCostCents,
		Period:         budget.Period(period),
		AlertThreshold: alertThreshold,
		Enabled:        true,
	}

	if err := budgetStore.CreateConfig(ctx, config); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to create budget: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success":    true,
		"budget_id":  config.ID,
		"message":    "Budget created successfully",
		"team_id":    teamID,
		"model_name": modelName,
	}, false)
}

func (s *MCPServer) handleGetBudgetStatus(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	modelName, _ := args["model_name"].(string)

	if teamID == "" || modelName == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id and model_name are required",
		}, true)
	}

	if s.budgetGovernor == nil {
		return buildToolResult(map[string]interface{}{
			"error": "budget system not initialized",
		}, true)
	}

	status, err := s.budgetGovernor.Status(ctx, teamID, modelName)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to get budget status: %v", err),
		}, true)
	}

	return buildToolResult(status, false)
}

func (s *MCPServer) handleListBudgets(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	if teamID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id is required",
		}, true)
	}

	configs, err := s.budgetStore.ListConfigsByTeam(ctx, teamID)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to list budgets: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"budgets": configs,
		"count":   len(configs),
	}, false)
}

func (s *MCPServer) handleGetBudgetHistory(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	teamID, _ := args["team_id"].(string)
	if teamID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "team_id is required",
		}, true)
	}

	days := 7
	if d, ok := args["days"].(float64); ok && d > 0 {
		days = int(d)
	}

	since := time.Now().UTC().AddDate(0, 0, -days)
	entries, err := s.budgetStore.ListEntries(ctx, teamID, since, 200)
	if err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to get history: %v", err),
		}, true)
	}

	// Compute summary
	var totalTokens, totalCost int64
	for _, e := range entries {
		totalTokens += e.TotalTokens
		totalCost += e.CostCents
	}

	return buildToolResult(map[string]interface{}{
		"entries":       entries,
		"count":         len(entries),
		"total_tokens":  totalTokens,
		"total_cost_cents": totalCost,
		"period_days":   days,
	}, false)
}

func (s *MCPServer) handleDeleteBudget(ctx context.Context, args map[string]interface{}) (*mcp.CallToolResult, error) {
	budgetID, _ := args["budget_id"].(string)
	if budgetID == "" {
		return buildToolResult(map[string]interface{}{
			"error": "budget_id is required",
		}, true)
	}

	if err := s.budgetStore.DeleteConfig(ctx, budgetID); err != nil {
		return buildToolResult(map[string]interface{}{
			"error": fmt.Sprintf("failed to delete budget: %v", err),
		}, true)
	}

	return buildToolResult(map[string]interface{}{
		"success": true,
		"message": "Budget configuration deleted",
	}, false)
}
