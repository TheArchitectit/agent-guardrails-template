package guardrails

import (
	"context"
	"testing"
)

// --- Finding 4: validator error fails closed (block) ---

func TestFourLaws_ValidatorErrorFailsClosed(t *testing.T) {
	// Simulate a validator that returns an error with zero/default Action.
	v := &erroringValidator{}
	chain := NewSafetyChain(SafetyChainDefinition{
		ID:        "c1",
		OnFailure: OnFailureBlockAndNotify,
		Steps: []ChainStep{
			{Validator: "err", Action: ActionWarn}, // even warn-configured
		},
	}, []SafetyValidator{v}, nil)

	res := chain.Execute(context.Background(), "a", "output", "", "")

	if res.Passed {
		t.Fatal("expected chain to fail when validator errors")
	}
	if res.Decision != string(ActionBlock) {
		t.Fatalf("expected block decision on validator error, got %q", res.Decision)
	}
}

type erroringValidator struct{}

func (e *erroringValidator) Validate(ctx context.Context, in ValidatorInput) (*ValidatorResult, error) {
	return nil, context.Canceled // internal error
}
func (e *erroringValidator) Name() string     { return "err" }
func (e *erroringValidator) Description() string { return "err" }

// --- Finding 5: FourLaws Law 2 runs even with empty context ---

func TestFourLaws_ScopeCheckRunsWithEmptyContext(t *testing.T) {
	v := NewFourLawsValidator([]string{"delete-db"})
	res, err := v.Validate(context.Background(), ValidatorInput{Output: "do the thing", Context: ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Fatal("expected scope_unverified failure with empty context and declared scope")
	}
	found := false
	for _, viol := range res.Violations {
		if viol == "scope_unverified" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected scope_unverified violation, got %v", res.Violations)
	}
}

func TestFourLaws_ScopeExceededWithContext(t *testing.T) {
	v := NewFourLawsValidator([]string{"delete-db"})
	res, err := v.Validate(context.Background(), ValidatorInput{
		Output:  "now delete-db everything",
		Context: "read-only reports",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Passed {
		t.Fatal("expected scope_violation")
	}
}

// --- Finding 6: destructive-command blacklist bypasses ---

func TestFourLaws_DestructiveCommandBypassCases(t *testing.T) {
	v := NewFourLawsValidator(nil)
	cases := []string{
		"rm -fr /",
		"rm -rf --no-preserve-root /",
		`rm -rf "$HOME"`,
		"rm -rf /home",
		"rm -r -f /",
		"chmod 777 /etc",
		"chmod -R 777 /",
	}
	for _, c := range cases {
		res, err := v.Validate(context.Background(), ValidatorInput{Output: c, Context: "ok"})
		if err != nil {
			t.Fatalf("case %q error: %v", c, err)
		}
		if res.Passed {
			t.Errorf("case %q should be flagged as destructive", c)
		}
	}
}

func TestFourLaws_SafeCommandsAllowed(t *testing.T) {
	v := NewFourLawsValidator(nil)
	cases := []string{
		"rm -rf ./build",        // relative path, not absolute
		"chmod 777 ./script.sh", // relative path
		"ls -la /tmp",
	}
	for _, c := range cases {
		res, err := v.Validate(context.Background(), ValidatorInput{Output: c, Context: "ok"})
		if err != nil {
			t.Fatalf("case %q error: %v", c, err)
		}
		if !res.Passed {
			t.Errorf("case %q should be allowed, got violations %v", c, res.Violations)
		}
	}
}

// --- Finding 8: ResolveConstraints override accounting ---

func TestResolveConstraints_NoEqualOverride(t *testing.T) {
	parent := []AgentConstraint{{ID: "p1", Name: "n", Severity: "high"}}
	child := []AgentConstraint{{ID: "p1", Name: "n", Severity: "high"}} // equal
	res := ResolveConstraints("c", "p", parent, child)
	if res.DowngradePrevented {
		t.Error("equal severity should not count as downgrade")
	}
	if res.EqualReplacements != 1 {
		t.Errorf("expected 1 equal-severity replacement, got %d", res.EqualReplacements)
	}
}

func TestResolveConstraints_UnknownSeverityCannotOverride(t *testing.T) {
	parent := []AgentConstraint{{ID: "p1", Name: "n", Severity: "medium"}}
	child := []AgentConstraint{{ID: "p1", Name: "n", Severity: ""}} // empty/unknown
	res := ResolveConstraints("c", "p", parent, child)
	if !res.DowngradePrevented {
		t.Error("empty child severity must not override a defined parent severity")
	}
}

func TestResolveConstraints_StrictUpgradeAllowed(t *testing.T) {
	parent := []AgentConstraint{{ID: "p1", Name: "n", Severity: "low"}}
	child := []AgentConstraint{{ID: "p1", Name: "n", Severity: "critical"}}
	res := ResolveConstraints("c", "p", parent, child)
	if res.DowngradePrevented {
		t.Error("strict upgrade should not be flagged as downgrade")
	}
	if len(res.MergedConstraints) != 1 || res.MergedConstraints[0].Severity != "critical" {
		t.Errorf("expected critical merged constraint, got %v", res.MergedConstraints)
	}
}

// --- Finding 9: conflict resolution ---

func TestResolveConflicts_UnionOnlyWhenConverged(t *testing.T) {
	outputs := []AgentOutput{
		{AgentID: "a", Output: "x", Actions: []string{"read"}},
		{AgentID: "b", Output: "y", Actions: []string{"read", "write"}},
	}
	res := ResolveConflicts(outputs, ConflictUnion)
	if res.Resolved {
		t.Error("union of divergent actions should NOT report Resolved=true")
	}
	if !res.Partial {
		t.Error("divergent union should set Partial=true")
	}
}

func TestResolveConflicts_UnionConverged(t *testing.T) {
	outputs := []AgentOutput{
		{AgentID: "a", Output: "x", Actions: []string{"read"}},
		{AgentID: "b", Output: "y", Actions: []string{"read"}},
	}
	res := ResolveConflicts(outputs, ConflictUnion)
	if !res.Resolved {
		t.Error("union of identical actions should report Resolved=true")
	}
}

func TestResolveConflicts_PriorityTieBreak(t *testing.T) {
	outputs := []AgentOutput{
		{AgentID: "zeta", Output: "z", Priority: 5},
		{AgentID: "alpha", Output: "a", Priority: 5}, // equal priority
	}
	res := ResolveConflicts(outputs, ConflictPriority)
	if res.ResolvedOutput != "a" {
		t.Errorf("tie should break to lexicographically smallest agent (alpha), got %q", res.ResolvedOutput)
	}
}
