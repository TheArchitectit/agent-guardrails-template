package git

/*
Vertical Slice: Git Guardrail

Self-contained git command validation with pattern matching,
force-push detection, and audit logging.
*/

import (
	"context"
	"log/slog"
	"time"

	"github.com/thearchitectit/guardrail-mcp/internal/domain"
)

// Rule represents a git guardrail rule
type Rule struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Pattern  string `json:"pattern"`
	Message  string `json:"message"`
	Severity string `json:"severity"` // "error", "warning", "info"
	Enabled  bool   `json:"enabled"`
}

// Evaluator performs pattern matching for git commands
type Evaluator struct {
	rules     []Rule
	patternFn func(pattern, input string) (bool, error)
}

// NewEvaluator creates a new git evaluator
func NewEvaluator(rules []Rule, patternFn func(pattern, input string) (bool, error)) *Evaluator {
	return &Evaluator{
		rules:     rules,
		patternFn: patternFn,
	}
}

// Evaluate checks a git command against all enabled rules
func (e *Evaluator) Evaluate(ctx context.Context, command string) ([]domain.Violation, error) {
	var violations []domain.Violation

	for _, rule := range e.rules {
		if !rule.Enabled {
			continue
		}

		matched, err := e.patternFn(rule.Pattern, command)
		if err != nil {
			slog.Warn("Pattern matching error", "rule_id", rule.ID, "error", err)
			continue
		}

		if matched {
			violations = append(violations, domain.Violation{
				RuleID:         rule.ID,
				RuleName:       rule.Name,
				Severity:       toSeverity(rule.Severity),
				Message:        rule.Message,
				Category:       "git",
				MatchedPattern: rule.Pattern,
				MatchedInput:   truncate(command, 200),
				Timestamp:      time.Now(),
			})
		}
	}

	return violations, nil
}

// DetectForcePush detects force push operations
func (e *Evaluator) DetectForcePush(command string) bool {
	return containsForceFlag(command)
}

func containsForceFlag(cmd string) bool {
	forceFlags := []string{"--force", "-f", "--force-with-lease", "-ff"}
	for _, flag := range forceFlags {
		if contains(cmd, flag) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Store handles data access for git rules
type Store interface {
	GetActiveRules(ctx context.Context) ([]Rule, error)
}

// Cache handles caching for git rules
type Cache interface {
	GetGitRules(ctx context.Context) ([]Rule, error)
	SetGitRules(ctx context.Context, rules []Rule, ttl time.Duration) error
}

// Handler is the MCP handler for git guardrail evaluation
type Handler struct {
	evaluator *Evaluator
	store     Store
	cache     Cache
	cacheTTL  time.Duration
}

// NewHandler creates a new git guardrail handler
func NewHandler(store Store, cache Cache, patternFn func(string, string) (bool, error)) *Handler {
	return &Handler{
		store:    store,
		cache:    cache,
		cacheTTL: 30 * time.Second,
		evaluator: &Evaluator{
			rules:     nil, // loaded lazily
			patternFn: patternFn,
		},
	}
}

// HandleEvaluate processes a git command evaluation request
func (h *Handler) HandleEvaluate(ctx context.Context, command string) (*domain.ValidationResult, error) {
	// Try cache first
	rules, err := h.cache.GetGitRules(ctx)
	if err != nil || rules == nil {
		rules, err = h.store.GetActiveRules(ctx)
		if err != nil {
			return nil, err
		}
		h.cache.SetGitRules(ctx, rules, h.cacheTTL)
	}

	evaluator := NewEvaluator(rules, h.evaluator.patternFn)
	violations, err := evaluator.Evaluate(ctx, command)
	if err != nil {
		return nil, err
	}

	return domain.NewValidationResult(violations), nil
}

// HandleWithForceCheck evaluates with automatic force-push detection
func (h *Handler) HandleWithForceCheck(ctx context.Context, command string, isForceFlag bool) (*domain.ValidationResult, error) {
	result, err := h.HandleEvaluate(ctx, command)
	if err != nil {
		return nil, err
	}

	// Auto-detect force flag if not explicitly provided
	shouldCheck := isForceFlag || h.evaluator.DetectForcePush(command)
	if shouldCheck && !containsForceFlag(command) {
		// Only add violation if the command doesn't already have force flag in pattern
		result.Violations = append(result.Violations, domain.Violation{
			RuleID:   "PREVENT-FORCE-001",
			RuleName: "No Force Operation",
			Severity: domain.SeverityError,
			Message:  "Force operations are not allowed. Use --force-with-lease or standard push instead.",
			Category: "git",
			Timestamp: time.Now(),
		})
	}

	return result, nil
}

func toSeverity(s string) domain.Severity {
	switch s {
	case "error":
		return domain.SeverityCritical
	case "warning":
		return domain.SeverityMedium
	case "info":
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
