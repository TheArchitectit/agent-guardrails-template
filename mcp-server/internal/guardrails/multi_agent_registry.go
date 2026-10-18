package guardrails

// Spec 04 — Multi-Agent Safety Policies: the request path.
//
// The domain model in multi_agent.go was complete and unreachable: NewSafetyChain
// had zero non-test callers, the three validators had zero production call sites,
// and none of the three tools the spec defines existed. A safety chain nothing
// can invoke is not a control, so this file supplies the two things that were
// missing — a registry that binds chain ids to real validators built from the
// live engine, and the entry points the MCP tools call.
//
// Two rules are enforced here rather than left to the caller:
//
//  1. An unknown chain id or policy id is an ERROR. Defaulting to a shorter
//     chain, or to "no constraints", would let a typo silently reduce the
//     validation an agent's output receives.
//  2. A validator that cannot run (nil engine, nil filter, broken pattern) is
//     an error, never a pass. The chain reports why.

import (
	"context"
	"fmt"
	"sort"
	"strings"
)

// Constraint IDs recognised by the built-in policies.
const (
	ConstraintFourLaws              = "four_laws"
	ConstraintNoDestructiveCommands = "no_destructive_commands"
	ConstraintAdversarialReview     = "adversarial_review"
	ConstraintScopeLimitation       = "scope_limitation"
)

// Built-in chain IDs (Spec 04 Sec. 2.2).
const (
	ChainCodeGeneration = "code_generation"
	ChainResearchOutput = "research_output"
	ChainReviewOnly     = "review_only"
)

// injectionPipeline is the narrow view of Engine the injection validator needs.
// Engine exposes DetectInjection, not Detect, so the adapter below is the only
// thing that has to change if that signature moves.
type injectionPipeline interface {
	Detect(ctx context.Context, text string, source Source) InjectionResult
}

type engineInjectionPipeline struct{ engine *Engine }

func (p engineInjectionPipeline) Detect(ctx context.Context, text string, source Source) InjectionResult {
	if p.engine == nil {
		// Never reached through the registry (construction refuses a nil
		// engine), but a nil here must not answer "safe".
		return InjectionResult{Safe: false, Reason: "guardrails engine not configured", Decision: string(FailPolicyBlock)}
	}
	return p.engine.DetectInjection(ctx, text, source)
}

// EngineContentFilterValidator runs Spec 02's classifier for a chain step.
//
// It calls Engine.ClassifyContent rather than holding a *ContentFilter so it
// inherits the engine's fail-closed posture: with no backend configured the
// filter returns a blocking SYSTEM result, which this validator turns into a
// failed step. A validator holding a filter it built itself could be
// constructed with no backend and report clean.
type EngineContentFilterValidator struct {
	engine *Engine
	filter *ContentFilter
}

// NewEngineContentFilterValidator builds the Spec 02 step for a chain.
func NewEngineContentFilterValidator(engine *Engine, filter *ContentFilter) *EngineContentFilterValidator {
	return &EngineContentFilterValidator{engine: engine, filter: filter}
}

// Validate implements SafetyValidator.
func (v *EngineContentFilterValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	var (
		result *ClassificationResult
		err    error
	)
	switch {
	case v.filter != nil:
		result, err = v.filter.Classify(ctx, input.Output, input.Direction)
	case v.engine != nil:
		// Distinguish "no classifier configured" from "the classifier found a
		// problem". Classify fails closed in both cases, which is right for a
		// filter asked a direct question and wrong as a chain step: reporting
		// an absent backend as a content block tells an operator that agents
		// are emitting harmful material when the truth is that nothing
		// examined the text. The layer is reported unavailable instead, and
		// Sec. 5's chain still runs its remaining steps.
		if v.engine.ContentFilterBackendCount() == 0 {
			return nil, &UnavailableError{
				Validator: v.Name(),
				Reason:    "no semantic classification backend is configured (set OLLAMA_URL or OPENAI_API_KEY)",
			}
		}
		result, err = v.engine.ClassifyContent(ctx, input.Output, input.Direction)
	default:
		return nil, fmt.Errorf("content filter step cannot run: no engine or filter configured")
	}
	if err != nil {
		return nil, err
	}
	if result == nil {
		// A nil result with no error is an absent verdict, not a pass.
		return nil, fmt.Errorf("content filter returned no verdict")
	}

	violations := make([]string, 0, len(result.Categories))
	for _, c := range result.Categories {
		if c.Action == ActionBlock {
			violations = append(violations, fmt.Sprintf("%s: %s", c.ID, c.Name))
		}
	}
	return &ValidatorResult{
		Passed:     result.Safe,
		Reason:     formatFilterReason(result),
		Violations: violations,
		Confidence: result.OverallRisk,
	}, nil
}

// Name implements SafetyValidator.
func (v *EngineContentFilterValidator) Name() string { return "content_filter" }

// Description implements SafetyValidator.
func (v *EngineContentFilterValidator) Description() string {
	return "Classifies agent output against the safety taxonomy (Spec 02)"
}

// ConstraintPolicy is a named set of constraints a caller can hand to an agent.
type ConstraintPolicy struct {
	ID          string            `json:"id"`
	Description string            `json:"description"`
	Constraints []AgentConstraint `json:"constraints"`
}

// MultiAgentRegistry binds chain ids and policy ids to configured instances.
//
// It is built once at startup from the live engine, so every chain resolves to
// the same detectors the rest of the server uses. Handing callers a
// constructor instead would let two call sites validate against two different
// pipelines.
type MultiAgentRegistry struct {
	chains   map[string]*SafetyChain
	policies map[string]ConstraintPolicy
	// order preserves a deterministic listing for the operator.
	order []string
}

// NewMultiAgentRegistry builds the built-in chains over the given engine.
//
// A nil engine is refused rather than producing chains whose validators answer
// "safe": the point of the registry is that a chain which runs has run
// something.
func NewMultiAgentRegistry(engine *Engine, scopePatterns []string) (*MultiAgentRegistry, error) {
	if engine == nil {
		return nil, fmt.Errorf("multi-agent registry requires a guardrails engine")
	}

	fourLaws, err := NewFourLawsValidatorChecked(scopePatterns)
	if err != nil {
		return nil, fmt.Errorf("four_laws validator: %w", err)
	}

	injection := NewInjectionDefenseValidator(engineInjectionPipeline{engine: engine})
	filter := NewEngineContentFilterValidator(engine, nil)

	r := &MultiAgentRegistry{
		chains:   make(map[string]*SafetyChain, 3),
		policies: map[string]ConstraintPolicy{},
	}

	defs := []SafetyChainDefinition{
		{
			ID:          ChainCodeGeneration,
			Description: "Safety chain for agents that emit code other agents will run",
			OnFailure:   OnFailureBlockAndNotify,
			Steps: []ChainStep{
				{Validator: injection.Name(), Action: ActionBlock, Description: "prompt injection in generated output"},
				{Validator: filter.Name(), Action: ActionBlock, Description: "harmful content taxonomy"},
				{Validator: fourLaws.Name(), Action: ActionBlock, Description: "Four Laws of agent safety"},
			},
		},
		{
			ID:          ChainResearchOutput,
			Description: "Safety chain for research agents producing text for humans",
			OnFailure:   OnFailureWarnAndContinue,
			Steps: []ChainStep{
				{Validator: filter.Name(), Action: ActionBlock, Description: "harmful content taxonomy"},
				{Validator: fourLaws.Name(), Action: ActionBlock, Description: "Four Laws of agent safety"},
			},
		},
		{
			ID:          ChainReviewOnly,
			Description: "Injection-only check for output that is reviewed before use",
			OnFailure:   OnFailureBlockAndNotify,
			Steps: []ChainStep{
				{Validator: injection.Name(), Action: ActionBlock, Description: "prompt injection in reviewed output"},
			},
		},
	}

	validators := []SafetyValidator{injection, filter, fourLaws}
	for _, def := range defs {
		chain := NewSafetyChain(def, validators, &DefaultMultiAgentAuditLogger{})
		r.chains[def.ID] = chain
		r.order = append(r.order, def.ID)
	}
	sort.Strings(r.order)

	for _, p := range defaultConstraintPolicies() {
		r.policies[p.ID] = p
	}

	return r, nil
}

// Chains lists the registered chain ids, sorted.
func (r *MultiAgentRegistry) Chains() []string {
	out := make([]string, len(r.order))
	copy(out, r.order)
	return out
}

// ValidateOutput runs a named safety chain over one agent's output.
//
// An unknown chain id is an error: falling back to the shortest chain would
// turn a typo into less validation without telling anyone.
func (r *MultiAgentRegistry) ValidateOutput(ctx context.Context, agentID, output, chainID, scopeContext, parentAgentID string) (*ChainResult, error) {
	if output == "" {
		return nil, fmt.Errorf("output is required")
	}
	chain, ok := r.chains[chainID]
	if !ok {
		return nil, fmt.Errorf("unknown chain_id %q; registered chains: %s", chainID, strings.Join(r.order, ", "))
	}
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}
	return chain.Execute(ctx, agentID, output, scopeContext, parentAgentID), nil
}

// CheckConstraints merges an agent's declared constraints onto a named policy
// and reports whether any parent constraint was weakened.
//
// Compliance is not "the call returned": an agent is compliant only when every
// policy constraint survives at equal-or-higher severity. A downgrade attempt
// is reported per constraint so the caller can name it.
func (r *MultiAgentRegistry) CheckConstraints(agentID, policyID string, own []AgentConstraint) (*ConstraintCheckResult, error) {
	policy, ok := r.policies[policyID]
	if !ok {
		ids := make([]string, 0, len(r.policies))
		for id := range r.policies {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return nil, fmt.Errorf("unknown policy_id %q; registered policies: %s", policyID, strings.Join(ids, ", "))
	}
	if agentID == "" {
		return nil, fmt.Errorf("agent_id is required")
	}

	// Compliance is measured against what the agent DECLARED, not against the
	// merged set. Comparing the merge output is the wrong question twice over:
	// ResolveConstraints keeps the parent's constraint when a child weakens it,
	// so the merge always satisfies the policy and the check could never fail.
	// A gate that cannot fail is worse than no gate — it reports "compliant" for
	// precisely the agent that tried to shed a constraint.
	declared := make(map[string]AgentConstraint, len(own))
	for _, c := range own {
		declared[c.ID] = c
	}

	results := make([]ConstraintResult, 0, len(policy.Constraints))
	compliant := true
	for _, want := range policy.Constraints {
		got, present := declared[want.ID]
		ok := present && severityRank(got.Severity) >= severityRank(want.Severity)
		if !ok {
			compliant = false
		}
		results = append(results, ConstraintResult{
			ID:        want.ID,
			Required:  want.Severity,
			Effective: got.Severity,
			Compliant: ok,
			Reason:    constraintReason(want, got, present, ok),
		})
	}

	// The merge machinery still reports whether it would have blocked the
	// downgrade, which is useful for an operator reading the result; it is not
	// what decides compliance.
	inherited := ResolveConstraints(agentID, "", policy.Constraints, own)

	return &ConstraintCheckResult{
		AgentID:            agentID,
		PolicyID:           policyID,
		Compliant:          compliant,
		Constraints:        results,
		DowngradePrevented: inherited.DowngradePrevented,
	}, nil
}

func constraintReason(want, got AgentConstraint, present, ok bool) string {
	switch {
	case !present:
		return "constraint missing from the agent's effective set"
	case ok:
		return "constraint holds at equal or higher severity"
	default:
		return fmt.Sprintf("constraint weakened: policy requires %q, agent declares %q", want.Severity, got.Severity)
	}
}

// ConstraintResult is one row of a constraint compliance check.
type ConstraintResult struct {
	ID        string `json:"id"`
	Required  string `json:"required_severity"`
	Effective string `json:"effective_severity"`
	Compliant bool   `json:"compliant"`
	Reason    string `json:"reason,omitempty"`
}

// ConstraintCheckResult is the output of CheckConstraints.
type ConstraintCheckResult struct {
	AgentID            string             `json:"agent_id"`
	PolicyID           string             `json:"policy_id"`
	Compliant          bool               `json:"compliant"`
	Constraints        []ConstraintResult `json:"constraints"`
	DowngradePrevented bool               `json:"downgrade_prevented"`
}

// SeverityRank orders constraint severities for callers outside this package.
//
// It is exported because the MCP boundary must refuse a constraint whose
// severity it cannot rank: an unknown severity ranks 0, below every
// requirement, and a caller that passed one through as "declared" would be
// describing a constraint that can never hold.
func SeverityRank(s string) int { return severityRank(s) }

// severityRank orders constraint severities. An unknown or empty severity ranks
// below every defined value so it can never satisfy a required constraint.
func severityRank(s string) int {
	switch strings.ToLower(s) {
	case "low":
		return 1
	case "medium":
		return 2
	case "high":
		return 3
	case "critical":
		return 4
	}
	return 0
}

func defaultConstraintPolicies() []ConstraintPolicy {
	return []ConstraintPolicy{
		{
			ID:          "four_laws",
			Description: "Every agent bound by the Four Laws of agent safety",
			Constraints: []AgentConstraint{
				{ID: ConstraintFourLaws, Name: "Four Laws", Description: "output obeys the Four Laws", Severity: "critical"},
				{ID: ConstraintNoDestructiveCommands, Name: "No destructive commands", Description: "output issues no destructive shell command", Severity: "critical"},
			},
		},
		{
			ID:          "code_writer",
			Description: "Constraints for an agent that writes code other agents run",
			Constraints: []AgentConstraint{
				{ID: ConstraintFourLaws, Name: "Four Laws", Description: "output obeys the Four Laws", Severity: "critical"},
				{ID: ConstraintNoDestructiveCommands, Name: "No destructive commands", Description: "output issues no destructive shell command", Severity: "critical"},
				{ID: ConstraintScopeLimitation, Name: "Scope limitation", Description: "output stays inside the declared scope", Severity: "high"},
			},
		},
		{
			ID:          "code_reviewer",
			Description: "Constraints for an agent that reviews another agent's work",
			Constraints: []AgentConstraint{
				{ID: ConstraintFourLaws, Name: "Four Laws", Description: "output obeys the Four Laws", Severity: "critical"},
				{ID: ConstraintAdversarialReview, Name: "Adversarial review", Description: "review assumes the author made a mistake", Severity: "high"},
			},
		},
	}
}
