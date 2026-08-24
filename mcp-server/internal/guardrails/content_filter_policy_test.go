package guardrails

import (
	"testing"
)

// === PolicyEngine ===

func TestPolicyEngine_EvaluateDefaultActions(t *testing.T) {
	pe := NewPolicyEngine(nil)
	res := pe.Evaluate(map[string]float64{
		"S10": 0.9, // Hate -> block
		"S13": 0.8, // Elections -> warn
	}, "backend")
	if res.Safe {
		t.Error("S10 block should make result unsafe")
	}
	if res.OverallRisk != 0.9 {
		t.Errorf("expected overall risk 0.9, got %v", res.OverallRisk)
	}
}

func TestPolicyEngine_ThresholdDowngrades(t *testing.T) {
	pe := NewPolicyEngineWithThresholds(nil, ThresholdConfig{Default: 0.9})
	res := pe.Evaluate(map[string]float64{"S10": 0.5}, "backend")
	// 0.5 < 0.9 default threshold -> downgraded to allow
	if !res.Safe {
		t.Error("score below threshold should downgrade to allow")
	}
	if len(res.Categories) == 0 || res.Categories[0].Action != ActionAllow {
		t.Error("expected action allow for below-threshold score")
	}
}

func TestPolicyEngine_PerCategoryThreshold(t *testing.T) {
	pe := NewPolicyEngineWithThresholds(nil, ThresholdConfig{
		Default:     0.9,
		PerCategory: map[string]float64{"S10": 0.3},
	})
	// 0.5 >= 0.3 per-category threshold -> block retained
	res := pe.Evaluate(map[string]float64{"S10": 0.5}, "backend")
	if res.Safe {
		t.Error("score above per-category threshold should remain block")
	}
}

func TestPolicyEngine_RuleOverrides(t *testing.T) {
	pe := NewPolicyEngineWithThresholds([]PolicyRule{{
		ID: "strict",
		Rules: []PolicyDetail{{
			Category:  "S13",
			Action:    ActionBlock,
			Threshold: 0.0,
		}},
		Overrides: []PolicyOverride{{
			Category:    "S13",
			Action:      ActionAllow,
			Description: "elections exempt",
		}},
	}}, ThresholdConfig{Default: 0.7})

	// S13 default is Warn; override forces Allow even at high score
	res := pe.Evaluate(map[string]float64{"S13": 0.95}, "backend")
	if !res.Safe {
		t.Error("S13 override to Allow should make result safe")
	}
	if len(res.Categories) != 1 || res.Categories[0].Action != ActionAllow {
		t.Errorf("expected S13 override action allow, got %+v", res.Categories)
	}
}

func TestPolicyEngine_CheckPolicy(t *testing.T) {
	pe := NewPolicyEngine([]PolicyRule{{
		ID: "p1",
		Rules: []PolicyDetail{{
			Category:    "S10",
			Action:      ActionBlock,
			Description: "no hate",
		}},
	}})

	res := pe.Evaluate(map[string]float64{"S10": 0.9}, "backend")
	pr := pe.CheckPolicy(res, "p1")
	if pr.Compliant {
		t.Error("blocking result should be non-compliant with policy")
	}
	if len(pr.Violations) == 0 {
		t.Error("expected violation for blocked S10")
	}

	// Unknown policy must fail closed: not compliant, with a violation.
	pr2 := pe.CheckPolicy(res, "unknown")
	if pr2.Compliant {
		t.Error("unknown policy must not be compliant (fail closed)")
	}
	if len(pr2.Violations) == 0 || pr2.Violations[0].CategoryID != "POLICY" {
		t.Errorf("expected unknown-policy violation, got %+v", pr2.Violations)
	}
}

func TestPolicyEngine_UpdateRules(t *testing.T) {
	pe := NewPolicyEngine(nil)
	pe.UpdateRules([]PolicyRule{{ID: "new"}})
	res := pe.Evaluate(map[string]float64{"S10": 0.9}, "backend")
	if res.Safe {
		t.Error("S10 default block should still apply after update")
	}
}

// === CategoryMetadata taxonomy ===

func TestCategoryMetadata_CanonicalTaxonomy(t *testing.T) {
	cases := map[string]struct {
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

	for id, want := range cases {
		name, action := CategoryMetadata(id)
		if name != want.name {
			t.Errorf("%s: expected name %q, got %q", id, want.name, name)
		}
		if action != want.action {
			t.Errorf("%s: expected action %q, got %q", id, want.action, action)
		}
	}

	// Unknown category
	name, action := CategoryMetadata("S99")
	if name != "Unknown" || action != ActionWarn {
		t.Errorf("unknown category: expected (Unknown, warn), got (%q, %q)", name, action)
	}
}
