package guardrails

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// Spec 04 (guardrails-gaps-2026/04-multi-agent-safety-policies) Sec. 2.2
// documents a `safety_chains:` configuration block and Sec. 3.2 an
// `agent_registry:`. Before this file **neither had a reader**: the only safety
// chains in the tree were three Go literals inside NewMultiAgentRegistry, so an
// operator who wrote the documented YAML changed nothing at all.
//
// That is the same defect class as the Spec 01/02/03/05 config files fixed
// earlier in this program, and it is the last of the six gap specs carrying a
// documented configuration surface with no consumer. It matters here because
// the artifact is a *safety boundary*: a chain an operator wrote to require
// `four_laws_check` before an agent's output reaches another agent, and which
// the server silently replaces with its built-in three-step chain, leaves a
// review trail asserting validation that never ran.
//
// The loader is strict for the same reasons as its four siblings:
//
//  1. Unknown keys are an ERROR (KnownFields). A typo in `four_laws_checks`
//     must not leave a chain one validator short and load clean.
//  2. A file that declares nothing is not a configuration: a missing section,
//     an empty file and a zero-chain list are each refused by name.
//  3. A declared chain with no steps, an unknown validator or an unknown step
//     action is refused rather than shortened, because a chain that silently
//     runs fewer checks than it names reports on a chain nobody wrote.
//  4. A validator the tree cannot construct is refused, never ignored. The
//     registry can build `injection_defense`, `content_filter` and
//     `four_laws_check`; `code_review_agent` (Sec. 4.1) needs an LLM review
//     backend this tree does not have, so naming it is an error that says so
//     rather than a step that quietly never runs.
//  5. `on_failure` has exactly ONE implemented semantic. The engine stops the
//     chain at the first blocking failure and records the decision on the audit
//     entry — that is `block_and_notify`. `warn_and_continue` at chain level
//     would downgrade a blocking failure to a warning, which is a fail-open a
//     configuration file may not request (P5/D-18: fail-closed is not
//     configurable). `block_and_escalate` has no escalation mechanism in this
//     tree. Both are refused by name; a warning is expressed with a step's own
//     `action: warn`, which is implemented and enforced.

// chainStepYAML mirrors Spec 04 Sec. 2.2's per-step keys.
type chainStepYAML struct {
	Validator   string `yaml:"validator"`
	Action      string `yaml:"action"`
	Description string `yaml:"description"`
}

// chainSectionYAML mirrors Sec. 2.2's per-chain keys. The chain id is the map
// key, exactly as the spec writes it (`safety_chains: { code_generation: ... }`),
// so a chain cannot be renamed by editing a field inside it.
type chainSectionYAML struct {
	Description string          `yaml:"description"`
	Enabled     *bool           `yaml:"enabled"`
	OnFailure   string          `yaml:"on_failure"`
	Steps       []chainStepYAML `yaml:"steps"`
}

// safetyChainsFile is the wrapper: Sec. 2.2 nests the chains under a top-level
// `safety_chains:` key.
type safetyChainsFile struct {
	SafetyChains map[string]chainSectionYAML `yaml:"safety_chains"`
}

// ConfigurableSafetyChainValidators lists the validator names a configured
// chain MAY name, sorted. It is the set the registry can actually construct,
// and it is published so the loader's refusal can name the alternatives rather
// than leaving an operator to guess.
func ConfigurableSafetyChainValidators() []string {
	return []string{"content_filter", "four_laws_check", "injection_defense"}
}

// ConfigurableSafetyChainActions lists the per-step actions a configured chain
// MAY name, sorted. `scan_and_block`/`scan_and_warn` are accepted because
// SafetyChain.Execute already treats them as block/warn respectively.
func ConfigurableSafetyChainActions() []string {
	return []string{"block", "scan_and_block", "scan_and_warn", "warn"}
}

// BuiltinSafetyChainDefinitions returns the chains the server ships with.
//
// It is exported because two callers now need the same list: the registry's
// zero-configuration constructor, and a deployment that configures some chains
// and wants the shipped ones for the rest.
func BuiltinSafetyChainDefinitions() []SafetyChainDefinition {
	return []SafetyChainDefinition{
		{
			ID:          ChainCodeGeneration,
			Description: "Safety chain for agents that emit code other agents will run",
			OnFailure:   OnFailureBlockAndNotify,
			Steps: []ChainStep{
				{Validator: "injection_defense", Action: ActionBlock, Description: "prompt injection in generated output"},
				{Validator: "content_filter", Action: ActionBlock, Description: "harmful content taxonomy"},
				{Validator: "four_laws_check", Action: ActionBlock, Description: "Four Laws of agent safety"},
			},
		},
		{
			ID:          ChainResearchOutput,
			Description: "Safety chain for research agents producing text for humans",
			// Was declared `warn_and_continue`, which the engine does not
			// implement: a blocking step stops the chain and records a block
			// decision whatever this field says. The declaration was inert and
			// misleading, so it now states the semantic the engine has.
			OnFailure: OnFailureBlockAndNotify,
			Steps: []ChainStep{
				{Validator: "content_filter", Action: ActionBlock, Description: "harmful content taxonomy"},
				{Validator: "four_laws_check", Action: ActionBlock, Description: "Four Laws of agent safety"},
			},
		},
		{
			ID:          ChainReviewOnly,
			Description: "Injection-only check for output that is reviewed before use",
			OnFailure:   OnFailureBlockAndNotify,
			Steps: []ChainStep{
				{Validator: "injection_defense", Action: ActionBlock, Description: "prompt injection in reviewed output"},
			},
		},
	}
}

// LoadSafetyChainsFromYAML reads the `safety_chains:` block from a YAML file.
//
// Chains are returned in a deterministic order (by id) so a listing and a diff
// do not depend on map iteration.
func LoadSafetyChainsFromYAML(path string) ([]SafetyChainDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var root safetyChainsFile
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&root); err != nil {
		// An empty file is not a configuration. Reporting the decoder's bare
		// "EOF" would send an operator looking for a YAML syntax error in a
		// file that is simply empty, so name the actual problem.
		if errors.Is(err, io.EOF) {
			return nil, fmt.Errorf("config %s is empty; it declares no safety_chains section", path)
		}
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if root.SafetyChains == nil {
		return nil, fmt.Errorf("config %s declares no safety_chains section", path)
	}

	defs, err := BuildSafetyChainDefinitions(root.SafetyChains)
	if err != nil {
		return nil, err
	}
	if err := ValidateSafetyChainDefinitions(defs); err != nil {
		return nil, err
	}
	return defs, nil
}

// BuildSafetyChainDefinitions converts the parsed YAML shape into chain
// definitions, refusing anything the engine cannot honour.
//
// A chain with `enabled: false` is dropped rather than loaded-and-ignored: an
// operator who disables a chain is removing it, and a disabled chain that still
// resolves would validate outputs under a name the operator believes they
// switched off. Dropping it means the chain id becomes unknown, which the
// registry already refuses loudly.
func BuildSafetyChainDefinitions(sections map[string]chainSectionYAML) ([]SafetyChainDefinition, error) {
	if len(sections) == 0 {
		return nil, fmt.Errorf("safety_chains declares no chains")
	}

	knownValidators := make(map[string]bool)
	for _, n := range ConfigurableSafetyChainValidators() {
		knownValidators[n] = true
	}
	knownActions := make(map[string]bool)
	for _, a := range ConfigurableSafetyChainActions() {
		knownActions[a] = true
	}

	ids := make([]string, 0, len(sections))
	for id := range sections {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	defs := make([]SafetyChainDefinition, 0, len(ids))
	for _, id := range ids {
		section := sections[id]
		if strings.TrimSpace(id) == "" {
			return nil, fmt.Errorf("a safety_chains entry has an empty chain id")
		}
		if id != strings.TrimSpace(id) {
			return nil, fmt.Errorf("safety chain id %q has surrounding whitespace", id)
		}
		if section.Enabled != nil && !*section.Enabled {
			continue
		}
		if len(section.Steps) == 0 {
			return nil, fmt.Errorf("safety chain %q declares no steps; a chain with no checks validates nothing", id)
		}

		onFailure := OnFailureAction(strings.TrimSpace(section.OnFailure))
		switch onFailure {
		case "":
			onFailure = OnFailureBlockAndNotify
		case OnFailureBlockAndNotify:
			// The one implemented semantic: stop at the first blocking failure
			// and record the decision on the audit entry.
		case OnFailureWarnAndContinue:
			return nil, fmt.Errorf("safety chain %q declares on_failure: warn_and_continue, which this engine does not implement: a blocking step stops the chain whatever this field says, and downgrading a blocking failure to a warning is fail-open (P5/D-18). Express a warning with a step's own `action: warn`", id)
		case OnFailureBlockAndEscalate:
			return nil, fmt.Errorf("safety chain %q declares on_failure: block_and_escalate, which this engine does not implement: there is no escalation mechanism. Remove the key (the default is block_and_notify)", id)
		default:
			return nil, fmt.Errorf("safety chain %q declares unknown on_failure %q; supported: block_and_notify", id, section.OnFailure)
		}

		steps := make([]ChainStep, 0, len(section.Steps))
		for i, s := range section.Steps {
			validator := strings.TrimSpace(s.Validator)
			if validator == "" {
				return nil, fmt.Errorf("safety chain %q step %d declares no validator", id, i+1)
			}
			if !knownValidators[validator] {
				return nil, fmt.Errorf("safety chain %q step %d names validator %q, which this server cannot construct; available: %s", id, i+1, s.Validator, strings.Join(ConfigurableSafetyChainValidators(), ", "))
			}
			action := strings.TrimSpace(s.Action)
			if action == "" {
				return nil, fmt.Errorf("safety chain %q step %d (%s) declares no action", id, i+1, validator)
			}
			if !knownActions[action] {
				return nil, fmt.Errorf("safety chain %q step %d (%s) declares unknown action %q; supported: %s", id, i+1, validator, s.Action, strings.Join(ConfigurableSafetyChainActions(), ", "))
			}
			steps = append(steps, ChainStep{
				Validator:   validator,
				Action:      Action(action),
				Description: s.Description,
			})
		}

		defs = append(defs, SafetyChainDefinition{
			ID:          id,
			Description: section.Description,
			Steps:       steps,
			OnFailure:   onFailure,
		})
	}

	if len(defs) == 0 {
		return nil, fmt.Errorf("safety_chains declares no enabled chains")
	}
	return defs, nil
}

// ValidateSafetyChainDefinitions checks a resolved chain list against the
// validators the registry can build. It exists so a caller that assembled
// definitions in Go — not through the YAML loader — cannot hand the registry a
// chain naming a validator nothing implements.
func ValidateSafetyChainDefinitions(defs []SafetyChainDefinition) error {
	if len(defs) == 0 {
		return fmt.Errorf("no safety chains to validate")
	}
	known := make(map[string]bool)
	for _, n := range ConfigurableSafetyChainValidators() {
		known[n] = true
	}
	seen := make(map[string]bool, len(defs))
	for _, def := range defs {
		if strings.TrimSpace(def.ID) == "" {
			return fmt.Errorf("safety chain has an empty id")
		}
		if seen[def.ID] {
			return fmt.Errorf("safety chain id %q is declared twice", def.ID)
		}
		seen[def.ID] = true
		if len(def.Steps) == 0 {
			return fmt.Errorf("safety chain %q declares no steps", def.ID)
		}
		switch def.OnFailure {
		case OnFailureBlockAndNotify:
		case "":
			return fmt.Errorf("safety chain %q declares no on_failure", def.ID)
		default:
			return fmt.Errorf("safety chain %q declares on_failure %q, which this engine does not implement", def.ID, def.OnFailure)
		}
		for i, step := range def.Steps {
			if !known[step.Validator] {
				return fmt.Errorf("safety chain %q step %d names validator %q, which this server cannot construct", def.ID, i+1, step.Validator)
			}
			switch step.Action {
			case ActionBlock, ActionWarn, ActionScanAndBlock, ActionScanAndWarn:
			default:
				return fmt.Errorf("safety chain %q step %d (%s) declares unknown action %q", def.ID, i+1, step.Validator, step.Action)
			}
		}
	}
	return nil
}
