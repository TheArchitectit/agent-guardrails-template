package guardrails

import (
	"fmt"
	"sync"
)

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

// PolicyEngine evaluates category scores against configured policies.
type PolicyEngine struct {
	rules      []PolicyRule
	thresholds ThresholdConfig
	mu         sync.RWMutex
}

// NewPolicyEngine creates a policy engine with the given rules.
func NewPolicyEngine(rules []PolicyRule) *PolicyEngine {
	return &PolicyEngine{
		rules:      rules,
		thresholds: ThresholdConfig{Default: 0.7},
	}
}

// NewPolicyEngineWithThresholds creates a policy engine with rules and threshold config.
func NewPolicyEngineWithThresholds(rules []PolicyRule, thresholds ThresholdConfig) *PolicyEngine {
	if thresholds.Default == 0 {
		thresholds.Default = 0.7
	}
	return &PolicyEngine{
		rules:      rules,
		thresholds: thresholds,
	}
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

		// Determine threshold: per-category > rule-specific > global default
		threshold := pe.thresholds.Default
		if perCat, ok := pe.thresholds.PerCategory[catID]; ok && perCat > 0 {
			threshold = perCat
		}

		// Apply rule-specific settings (first matching rule wins for action/threshold)
		for _, rule := range pe.rules {
			for _, detail := range rule.Rules {
				if detail.Category == catID {
					action = detail.Action
					reason = detail.Description
					// Rule-specific threshold overrides config threshold
					if detail.Threshold > 0 {
						threshold = detail.Threshold
					}
					break
				}
			}
		}

		// Apply threshold: if score is below threshold, downgrade to allow
		if score < threshold {
			action = ActionAllow
			reason = fmt.Sprintf("below threshold %.2f", threshold)
		}

		// Apply context-specific overrides (last matching override wins)
		for _, rule := range pe.rules {
			for _, override := range rule.Overrides {
				if override.Category == catID {
					action = override.Action
					reason = override.Description
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
