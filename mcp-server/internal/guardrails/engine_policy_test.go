package guardrails

import (
	"context"
	"testing"
)

// TestNewEngine_ConfiguredPoliciesAreLoaded is a regression test: NewEngine
// built the content filter with a nil policy set, ignoring
// config.ContentFilter.Policies entirely, so the policy engine had no rules
// and every guardrail_check_policy call fell through to the fail-closed
// "unknown policy_id" branch no matter which policy the caller asked for.
func TestNewEngine_ConfiguredPoliciesAreLoaded(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.ContentFilter.Enabled = true
	cfg.ContentFilter.Policies = []PolicyRule{{
		ID:          "coding-safety",
		Description: "test policy",
		Rules: []PolicyDetail{
			{Category: "S1", Action: ActionBlock, Threshold: 0.7},
		},
	}}

	e := NewEngine(cfg, nil)

	res, err := e.CheckPolicy(context.Background(), "benign text", "coding-safety")
	if err != nil {
		t.Fatalf("CheckPolicy returned error: %v", err)
	}
	if res == nil {
		t.Fatal("CheckPolicy returned nil result")
	}

	// The failure mode being guarded against is the synthetic violation the
	// policy engine emits for an unknown policy.
	for _, v := range res.Violations {
		if v.CategoryID == "POLICY" {
			t.Fatalf("configured policy was reported as unknown: %+v", res)
		}
	}
}

// TestNewEngine_UnknownPolicyStillFailsClosed pins the intentional behaviour
// that makes the bug above visible: a policy that was never configured must
// not be reported as compliant.
func TestNewEngine_UnknownPolicyStillFailsClosed(t *testing.T) {
	cfg := DefaultEngineConfig()
	cfg.ContentFilter.Enabled = true

	e := NewEngine(cfg, nil)

	res, err := e.CheckPolicy(context.Background(), "benign text", "never-configured")
	if err != nil {
		t.Fatalf("CheckPolicy returned error: %v", err)
	}
	if res.Compliant {
		t.Fatal("an unconfigured policy must not be reported as compliant")
	}

	found := false
	for _, v := range res.Violations {
		if v.CategoryID == "POLICY" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a POLICY violation for an unknown policy, got %+v", res.Violations)
	}
}
