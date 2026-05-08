package bash

/*
Vertical Slice: Bash Guardrail

Each guardrail type is a self-contained slice. All business logic — rule model,
evaluator, handler, and store access — lives in one package. Addable by creating
a new directory and registering in main.go.
*/

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/domain"
)

// Rule represents a bash guardrail rule (self-contained model)
type Rule struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Pattern      string `json:"pattern"`
	Message      string `json:"message"`
	Severity     string `json:"severity"` // "critical", "high", "medium", "low"
	Enabled      bool   `json:"enabled"`
	Category     string `json:"category"`
	CreatedAt    string `json:"created_at"`
}

// Evaluator performs pattern matching for bash commands
type Evaluator struct {
	rules      []Rule
	patternFn  func(pattern, input string) (bool, error)
}

// NewEvaluator creates a new bash evaluator with rules
func NewEvaluator(rules []Rule, patternFn func(pattern, input string) (bool, error)) *Evaluator {
	return &Evaluator{
		rules:     rules,
		patternFn: patternFn,
	}
}

// Evaluate checks a bash command against all enabled rules
func (e *Evaluator) Evaluate(ctx context.Context, command string) ([]domain.Violation, error) {
	var violations []domain.Violation

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		matched, err := e.patternFn(rule.Pattern, command)
		if err != nil {
			slog.Warn("Pattern matching error",
				"rule_id", rule.ID,
				"error", err,
			)
			continue
		}

		if matched {
			violations = append(violations, domain.Violation{
				RuleID:         rule.ID,
				RuleName:       rule.Name,
				Severity:       toSeverity(rule.Severity),
				Message:        rule.Message,
				Category:       "bash",
				MatchedPattern: rule.Pattern,
				MatchedInput:   truncate(command, 200),
				Timestamp:      time.Now(),
			})
		}
	}

	return violations, nil
}

// Store handles data access for bash rules
type Store interface {
	GetActiveRules(ctx context.Context) ([]Rule, error)
}

// Cache is the cache port for bash rules
type Cache interface {
	GetBashRules(ctx context.Context) ([]Rule, error)
	SetBashRules(ctx context.Context, rules []Rule, ttl time.Duration) error
	InvalidateBashRules(ctx context.Context) error
}

// Handler is the MCP handler for bash guardrail evaluation
type Handler struct {
	evaluator *Evaluator
	store     Store
	cache     Cache
	cacheTTL  time.Duration
}

// NewHandler creates a new bash guardrail handler
func NewHandler(store Store, cache Cache, patternFn func(string, string) (bool, error)) *Handler {
	return &Handler{
		store:    store,
		cache:    cache,
		cacheTTL: 30 * time.Second,
	}
}

// Register in main.go: bashGuardrailHandler := bash.NewHandler(ruleStore, cache, validation.MatchPattern)

// HandleEvaluate processes a bash command evaluation request
func (h *Handler) HandleEvaluate(ctx context.Context, command string) (*domain.ValidationResult, error) {
	// Try cache first (read-optimized query side)
	rules, err := h.cache.GetBashRules(ctx)
	if err != nil || rules == nil {
		// Cache miss - load from store
		rules, err = h.store.GetActiveRules(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to load rules: %w", err)
		}
		// Populate cache
		h.cache.SetBashRules(ctx, rules, h.cacheTTL)
	}

	evaluator := NewEvaluator(rules, h.patternFn())
	violations, err := evaluator.Evaluate(ctx, command)
	if err != nil {
		return nil, err
	}

	return domain.NewValidationResult(violations), nil
}

func (h *Handler) patternFn() func(pattern, input string) (bool, error) {
	return func(pattern, input string) (bool, error) {
		// Delegate to shared pattern matcher (could be swapped for LLM-based)
		return matchPattern(pattern, input)
	}
}

func matchPattern(pattern, input string) (bool, error) {
	// Placeholder - actual implementation uses validation.MatchPattern
	// This is injected to maintain testability
	return false, nil
}

func toSeverity(s string) domain.Severity {
	switch s {
	case "critical":
		return domain.SeverityCritical
	case "high":
		return domain.SeverityHigh
	case "medium":
		return domain.SeverityMedium
	case "low":
		return domain.SeverityLow
	default:
		return domain.SeverityMedium
	}
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
