package guardrails

// Spec 04 (guardrails-gaps-2026/04-multi-agent-safety-policies) — evidence for
// the `safety_chains:` configuration surface.
//
// The tests drive the REAL loader with a verbatim copy of the spec's own
// documented YAML, then the REAL MultiAgentRegistry and the REAL SafetyChain.
// Parsing is not control: a loader test that only asserts the resulting struct
// fields would pass while the chain the server runs is still the built-in one,
// which is exactly the defect this file exists to prevent.

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// specSafetyChainsYAML is a verbatim copy of the block Spec 04 Sec. 2.2
// documents, minus the validators this tree cannot construct (see
// TestUnknownValidatorIsRefused). It is written to a temp file, never embedded
// in the implementation.
const specSafetyChainsYAML = `
safety_chains:
  code_generation:
    description: "Safety chain for code generation agents"
    steps:
      - validator: "injection_defense"
        action: "block"
        description: "Check for prompt injection in output"
      - validator: "content_filter"
        action: "block"
        description: "Classify content against safety taxonomy"
      - validator: "four_laws_check"
        action: "block"
        description: "Verify output follows Four Laws"
    on_failure: "block_and_notify"

  research_output:
    description: "Safety chain for research agents"
    steps:
      - validator: "content_filter"
        action: "block"
        description: "Check for harmful content"
      - validator: "four_laws_check"
        action: "warn"
        description: "Cross-reference claims"
    on_failure: "block_and_notify"
`

func writeTempYAML(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "safety-chains.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// --- The documented shape loads, and the loaded chain is the one that runs ---

func TestLoadSafetyChains_DocumentedShapeLoads(t *testing.T) {
	defs, err := LoadSafetyChainsFromYAML(writeTempYAML(t, specSafetyChainsYAML))
	if err != nil {
		t.Fatalf("the spec's own documented YAML must load, got: %v", err)
	}
	if len(defs) != 2 {
		t.Fatalf("expected 2 chains, got %d", len(defs))
	}
	// Deterministic order by id: code_generation before research_output.
	if defs[0].ID != ChainCodeGeneration || defs[1].ID != ChainResearchOutput {
		t.Fatalf("chains out of order: %q, %q", defs[0].ID, defs[1].ID)
	}
	if len(defs[0].Steps) != 3 {
		t.Fatalf("code_generation should carry 3 steps, got %d", len(defs[0].Steps))
	}
	if defs[0].Steps[0].Validator != "injection_defense" {
		t.Errorf("first step should be injection_defense, got %q", defs[0].Steps[0].Validator)
	}
	// A per-step warn is honoured as warn — this is the implemented way to
	// express "do not block on this check" (the chain-level warn_and_continue
	// key is refused; see TestWarnAndContinueOnFailureIsRefused).
	if defs[1].Steps[1].Action != ActionWarn {
		t.Errorf("research_output step 2 should be warn, got %q", defs[1].Steps[1].Action)
	}
}

// TestLoadedChainBindsAtExecution is the load-bearing test: a chain read from
// the documented file must be the chain the registry executes. The configuration
// is a different shape from the built-in (one step where the built-in has
// three), so a loader that parsed correctly but wired the built-in would still
// run three steps and fail here.
func TestLoadedChainBindsAtExecution(t *testing.T) {
	cfg := `
safety_chains:
  code_generation:
    description: "injection only"
    steps:
      - validator: "injection_defense"
        action: "block"
    on_failure: "block_and_notify"
`
	defs, err := LoadSafetyChainsFromYAML(writeTempYAML(t, cfg))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	engine := NewEngine(nil, nil)
	reg, err := NewMultiAgentRegistryWithChains(engine, nil, defs)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if got := reg.Chains(); len(got) != 1 || got[0] != ChainCodeGeneration {
		t.Fatalf("registry chains = %v, want [code_generation]", got)
	}

	// The configured chain has ONE step. If the built-in chain (three steps)
	// were still wired, this would report three.
	res, err := reg.ValidateOutput(context.Background(), "writer", "The changelog adds one endpoint.", ChainCodeGeneration, "summarise", "")
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if len(res.Steps) != 1 {
		t.Fatalf("the configured one-step chain ran %d steps; the built-in chain is still wired", len(res.Steps))
	}
	if !res.Passed || res.Decision != string(ActionAllow) {
		t.Fatalf("benign output should pass the one-step chain, got passed=%v decision=%q", res.Passed, res.Decision)
	}
}

// A chain id the file did not declare must be unknown, not silently replaced by
// a built-in. This is the fail-closed half of the wiring: disabling a chain in
// the file removes it, and a request for it is refused.
func TestUndeclaredChainIDIsUnknown(t *testing.T) {
	defs, err := LoadSafetyChainsFromYAML(writeTempYAML(t, specSafetyChainsYAML))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// Drop every chain but one, so research_output is genuinely undeclared.
	defs = defs[:1]
	reg, err := NewMultiAgentRegistryWithChains(NewEngine(nil, nil), nil, defs)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if _, err := reg.ValidateOutput(context.Background(), "a", "text", ChainResearchOutput, "", ""); err == nil {
		t.Fatal("an undeclared chain id must be refused, not replaced by a built-in")
	}
	// And the built-in chains must NOT be reachable through this registry.
	for _, id := range []string{ChainResearchOutput, ChainReviewOnly} {
		if _, err := reg.ValidateOutput(context.Background(), "a", "text", id, "", ""); err == nil {
			t.Errorf("chain %q resolved although the file did not declare it", id)
		}
	}
}

// --- Refusal shapes ---

func TestSafetyChainsRefusalShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{
			name: "empty file",
			body: "",
			want: "empty",
		},
		{
			name: "no safety_chains section",
			body: "{}\n",
			want: "declares no safety_chains section",
		},
		{
			name: "unknown key",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: block\n    on_failures: block_and_notify\n",
			want: "field on_failures not found",
		},
		{
			name: "unknown top-level key",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: block\nagent_registry:\n  agents: []\n",
			want: "field agent_registry not found",
		},
		{
			name: "chain with no steps",
			body: "safety_chains:\n  c:\n    description: nothing\n    steps: []\n",
			want: "declares no steps",
		},
		{
			name: "unknown validator refused by name",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: code_review_agent\n        action: warn\n",
			want: "cannot construct",
		},
		{
			name: "missing validator",
			body: "safety_chains:\n  c:\n    steps:\n      - action: block\n",
			want: "declares no validator",
		},
		{
			name: "unknown action",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: quarantine\n",
			want: "unknown action",
		},
		{
			name: "missing action",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n",
			want: "declares no action",
		},
		{
			name: "warn_and_continue on_failure refused",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: block\n    on_failure: warn_and_continue\n",
			want: "fail-open",
		},
		{
			name: "block_and_escalate on_failure refused",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: block\n    on_failure: block_and_escalate\n",
			want: "no escalation mechanism",
		},
		{
			name: "unknown on_failure",
			body: "safety_chains:\n  c:\n    steps:\n      - validator: injection_defense\n        action: block\n    on_failure: ignore\n",
			want: "unknown on_failure",
		},
		{
			name: "disabled chain leaves nothing",
			body: "safety_chains:\n  c:\n    enabled: false\n    steps:\n      - validator: injection_defense\n        action: block\n",
			want: "no enabled chains",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := LoadSafetyChainsFromYAML(writeTempYAML(t, tc.body))
			if err == nil {
				t.Fatalf("expected a refusal mentioning %q, got nil error", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("refusal %q does not mention %q", err.Error(), tc.want)
			}
		})
	}
}

// A disabled chain is dropped rather than loaded-and-ignored: the operator who
// disabled it must not be able to invoke it.
func TestDisabledChainIsAbsent(t *testing.T) {
	body := `
safety_chains:
  code_generation:
    steps:
      - validator: injection_defense
        action: block
  review_only:
    enabled: false
    steps:
      - validator: injection_defense
        action: block
`
	defs, err := LoadSafetyChainsFromYAML(writeTempYAML(t, body))
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	for _, d := range defs {
		if d.ID == ChainReviewOnly {
			t.Fatal("a chain with enabled:false must not load")
		}
	}
}

// A configured chain naming a validator the registry cannot build is refused
// even when it reaches the registry by a path other than the YAML loader.
func TestRegistryRefusesUnconstructableValidator(t *testing.T) {
	bad := []SafetyChainDefinition{{
		ID:        "c",
		OnFailure: OnFailureBlockAndNotify,
		Steps:     []ChainStep{{Validator: "code_review_agent", Action: ActionWarn}},
	}}
	if _, err := NewMultiAgentRegistryWithChains(NewEngine(nil, nil), nil, bad); err == nil {
		t.Fatal("registry must refuse a chain naming a validator it cannot construct")
	}
}

// The default (no file) path must be unchanged: no configuration means the
// built-in chains, and the built-in list still validates.
func TestBuiltinChainsUnchanged(t *testing.T) {
	defs := BuiltinSafetyChainDefinitions()
	if err := ValidateSafetyChainDefinitions(defs); err != nil {
		t.Fatalf("built-in chains must validate: %v", err)
	}
	if len(defs) != 3 {
		t.Fatalf("expected the three built-in chains, got %d", len(defs))
	}
	reg, err := NewMultiAgentRegistry(NewEngine(nil, nil), nil)
	if err != nil {
		t.Fatalf("registry: %v", err)
	}
	if got := reg.Chains(); len(got) != 3 {
		t.Fatalf("expected 3 built-in chains, got %v", got)
	}
}

// Every built-in chain declares the one on_failure semantic the engine
// implements. Before this pass research_output declared `warn_and_continue`,
// which the engine ignores: a blocking step stops the chain and records a block
// decision whatever the field says. A declaration the engine does not honour is
// a false statement about behaviour.
func TestBuiltinChainsDeclareAnImplementedOnFailure(t *testing.T) {
	for _, d := range BuiltinSafetyChainDefinitions() {
		if d.OnFailure != OnFailureBlockAndNotify {
			t.Errorf("chain %q declares on_failure %q, which this engine does not implement", d.ID, d.OnFailure)
		}
	}
}
