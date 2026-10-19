package guardrails

import (
	"fmt"
	"strings"
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
	// categories is the taxonomy filter (Spec 02 Sec. 3.2
	// `taxonomy.enabled_categories`). A nil slice means every category is
	// evaluated, which is the shipped default. A non-nil slice is an
	// allow-list: a category that is not in it is never scored, so a backend
	// that reports S5 when the operator enabled only S1-S4 cannot silently
	// widen the taxonomy the deployment declared.
	categories []string
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

// NewPolicyEngineWithConfig creates a policy engine whose thresholds AND
// enabled-category set come from one configuration.
//
// It exists because the taxonomy was declared and had no effect: the
// `enabled_categories` / `custom_categories` keys were parsed into the config
// and never read at request time, so an operator who narrowed the taxonomy
// still got every S1-S15 category evaluated.
func NewPolicyEngineWithConfig(rules []PolicyRule, thresholds ThresholdConfig, enabled, custom []string) *PolicyEngine {
	pe := NewPolicyEngineWithThresholds(rules, thresholds)
	pe.categories = normalizeCategories(enabled, custom)
	return pe
}

// normalizeCategories resolves the declared taxonomy into the allow-list the
// engine enforces.
//
// An EMPTY declared list is a real declaration, not "absent": the spec's own
// example enables a subset (S1,S2,S3,S4,S7,S9,S10,S11,S14,S15), and a config
// that enables nothing must evaluate nothing rather than falling back to all
// fifteen. Callers that want the shipped default must not call this at all
// (DefaultFilterConfig sets every category).
func normalizeCategories(enabled, custom []string) []string {
	seen := make(map[string]bool, len(enabled)+len(custom))
	out := make([]string, 0, len(enabled)+len(custom))
	for _, id := range append(append([]string{}, enabled...), custom...) {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, id)
	}
	return out
}

// categoryEnabled reports whether a category is inside the declared taxonomy.
// A nil allow-list means the engine was built without a taxonomy filter, so
// every category passes (the pre-existing behaviour).
func (pe *PolicyEngine) categoryEnabled(catID string) bool {
	if pe.categories == nil {
		return true
	}
	for _, id := range pe.categories {
		if id == catID {
			return true
		}
	}
	return false
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
		// A category outside the declared taxonomy is not evaluated. Before
		// this it was scored and reported like any other, so a narrowed
		// taxonomy changed nothing an operator could observe.
		if !pe.categoryEnabled(catID) {
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
		// Fail closed: a policy_id that does not exist in the engine must not be
		// treated as compliant. Reporting it as a violation forces every caller
		// (Engine.CheckPolicy, handlers, tests) to reject unknown policies
		// instead of silently passing.
		return &PolicyResult{
			PolicyID:  policyID,
			Compliant: false,
			Violations: []ContentViolation{{
				CategoryID:   "POLICY",
				CategoryName: "Unknown Policy",
				Score:        1.0,
				Action:       ActionBlock,
				Reason:       fmt.Sprintf("unknown policy_id %q: cannot assert compliance", policyID),
			}},
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
