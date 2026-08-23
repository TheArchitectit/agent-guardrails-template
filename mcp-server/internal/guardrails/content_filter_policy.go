package guardrails

import (
	"context"
	"log/slog"
	"sync"
)

// StreamingHandler processes text chunks for real-time filtering.
type StreamingHandler struct {
	filter    *ContentFilter
	onBlock   func(result *ClassificationResult)
	onWarn    func(result *ClassificationResult)
	chunkSize int
	mu        sync.Mutex
}

// NewStreamingHandler creates a streaming content filter.
func NewStreamingHandler(filter *ContentFilter, onBlock, onWarn func(*ClassificationResult)) *StreamingHandler {
	return &StreamingHandler{
		filter:    filter,
		onBlock:   onBlock,
		onWarn:    onWarn,
		chunkSize: 512,
	}
}

// ProcessChunk classifies a text chunk. Returns true if generation should continue.
func (sh *StreamingHandler) ProcessChunk(ctx context.Context, chunk string, direction ContentDirection) bool {
	result, err := sh.filter.Classify(ctx, chunk, direction)
	if err != nil {
		slog.Warn("Streaming chunk classification failed", "error", err)
		return true
	}

	if result.IsBlocked() {
		if sh.onBlock != nil {
			sh.onBlock(result)
		}
		return false
	}

	for _, cat := range result.Categories {
		if cat.Action == ActionWarn && sh.onWarn != nil {
			sh.onWarn(result)
			break
		}
	}

	return true
}

// SetChunkSize overrides the default chunk size for streaming.
func (sh *StreamingHandler) SetChunkSize(size int) {
	sh.chunkSize = size
}

// CategoryMetadata returns the name and default action for a category.
func CategoryMetadata(id string) (name string, defaultAction Action) {
	meta := map[string]struct {
		name   string
		action Action
	}{
		"S1":  {"Violent Crimes", ActionBlock},
		"S2":  {"Non-Violent Crimes", ActionBlock},
		"S3":  {"Sex Crimes", ActionBlock},
		"S4":  {"Child Exploitation", ActionBlock},
		"S5":  {"Defamation", ActionWarn},
		"S6":  {"Specialized Advice", ActionWarn},
		"S7":  {"Privacy", ActionBlock},
		"S8":  {"Intellectual Property", ActionWarn},
		"S9":  {"Indiscriminate Weapons", ActionBlock},
		"S10": {"Hate", ActionBlock},
		"S11": {"Self-Harm", ActionBlock},
		"S12": {"Sexual Content", ActionBlock},
		"S13": {"Elections", ActionWarn},
		"S14": {"Code Abuse", ActionBlock},
		"S15": {"Data Exfiltration", ActionBlock},
	}

	m, ok := meta[id]
	if !ok {
		return "Unknown", ActionWarn
	}
	return m.name, m.action
}

// RiskFromScore converts a normalized score to a RiskLevel.
func RiskFromScore(score float64) RiskLevel {
	switch {
	case score >= 0.9:
		return RiskLevelCritical
	case score >= 0.7:
		return RiskLevelHigh
	case score >= 0.4:
		return RiskLevelMedium
	case score > 0:
		return RiskLevelLow
	default:
		return RiskLevelNone
	}
}

// PolicyEngine evaluates category scores against configured policies.
type PolicyEngine struct {
	rules []PolicyRule
	mu    sync.RWMutex
}

// NewPolicyEngine creates a policy engine with the given rules.
func NewPolicyEngine(rules []PolicyRule) *PolicyEngine {
	return &PolicyEngine{rules: rules}
}

// Evaluate maps category scores to actions based on policy rules.
func (pe *PolicyEngine) Evaluate(scores map[string]float64, backendName string) *ClassificationResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var categories []CategoryResult
	var maxRisk float64

	for catID, score := range scores {
		if score <= 0 {
			continue
		}

		name, defaultAction := CategoryMetadata(catID)
		action := defaultAction
		reason := ""

		for _, rule := range pe.rules {
			for _, detail := range rule.Rules {
				if detail.Category == catID {
					action = detail.Action
					reason = detail.Description
					if detail.Threshold > 0 && score < detail.Threshold {
						action = ActionAllow
					}
				}
			}
		}

		categories = append(categories, CategoryResult{
			ID:     catID,
			Name:   name,
			Score:  score,
			Action: action,
			Reason: reason,
		})

		if score > maxRisk {
			maxRisk = score
		}
	}

	return &ClassificationResult{
		Safe:        !hasBlock(categories),
		OverallRisk: maxRisk,
		Categories:  categories,
		Backend:     backendName,
	}
}

// CheckPolicy checks if the classification result complies with a specific policy.
func (pe *PolicyEngine) CheckPolicy(result *ClassificationResult, policyID string) *PolicyResult {
	pe.mu.RLock()
	defer pe.mu.RUnlock()

	var targetRule *PolicyRule
	for i := range pe.rules {
		if pe.rules[i].ID == policyID {
			targetRule = &pe.rules[i]
			break
		}
	}

	if targetRule == nil {
		return &PolicyResult{
			PolicyID:  policyID,
			Compliant: true,
		}
	}

	var violations []ContentViolation
	for _, cat := range result.Categories {
		for _, detail := range targetRule.Rules {
			if detail.Category == cat.ID && cat.Action == ActionBlock {
				violations = append(violations, ContentViolation{
					CategoryID:   cat.ID,
					CategoryName: cat.Name,
					Score:        cat.Score,
					Action:       cat.Action,
					Reason:       detail.Description,
				})
			}
		}
	}

	return &PolicyResult{
		PolicyID:   policyID,
		Compliant:  len(violations) == 0,
		Violations: violations,
	}
}

// UpdateRules replaces the policy rules (for hot-reload).
func (pe *PolicyEngine) UpdateRules(rules []PolicyRule) {
	pe.mu.Lock()
	defer pe.mu.Unlock()
	pe.rules = rules
}

func hasBlock(categories []CategoryResult) bool {
	for _, c := range categories {
		if c.Action == ActionBlock {
			return true
		}
	}
	return false
}
