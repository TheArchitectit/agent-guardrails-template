// Package guardrails provides multi-agent safety policies for the MCP server.
//
// This module implements Spec 04: Multi-Agent Safety Policies — an inter-agent
// validation system where one agent's output is verified by a configurable chain
// of safety validators before reaching downstream agents or users.
//
// Core capabilities:
//   - SafetyValidator interface for pluggable validation logic
//   - SafetyChain orchestrator for ordered validator execution
//   - Constraint inheritance (child agents inherit parent constraints)
//   - Conflict resolution strategies (priority, intersection, union, escalate)
//
// The Four Laws are enforced across the agent graph, not just on individual agents.
package guardrails

import (
	"context"
	"fmt"
	"log/slog"
	"regexp"
	"sort"
	"strings"
	"time"
)

// OnFailureAction determines chain behavior when a validator fails.
type OnFailureAction string

const (
	OnFailureBlockAndNotify   OnFailureAction = "block_and_notify"
	OnFailureWarnAndContinue  OnFailureAction = "warn_and_continue"
	OnFailureBlockAndEscalate OnFailureAction = "block_and_escalate"
)

// ConflictStrategy resolves disagreements between parallel agent outputs.
type ConflictStrategy string

const (
	ConflictPriority     ConflictStrategy = "priority"
	ConflictIntersection ConflictStrategy = "intersection"
	ConflictUnion        ConflictStrategy = "union"
	ConflictEscalate     ConflictStrategy = "escalate"
)

// StepResult captures the outcome of a single validator in a safety chain.
type StepResult struct {
	Step       int            `json:"step"`
	Validator  string         `json:"validator"`
	Passed     bool           `json:"passed"`
	LatencyMs  int64          `json:"latency_ms"`
	Reason     string         `json:"reason,omitempty"`
	Violations []string       `json:"violations,omitempty"`
}

// ChainResult captures the outcome of executing a safety chain.
type ChainResult struct {
	ChainID      string        `json:"chain_id"`
	AgentID      string        `json:"agent_id"`
	ParentAgentID string       `json:"parent_agent_id,omitempty"`
	Passed       bool          `json:"passed"`
	Decision     string        `json:"decision"`
	Steps        []StepResult  `json:"steps"`
	Violations   []string      `json:"violations,omitempty"`
	AuditEntry   *AuditEntry   `json:"-"`
}

// AuditEntry is a structured audit log entry for agent chain validation.
type AuditEntry struct {
	Event          string       `json:"event"`
	Timestamp      string       `json:"timestamp"`
	ChainID        string       `json:"chain_id"`
	AgentID        string       `json:"agent_id"`
	ParentAgentID  string       `json:"parent_agent_id,omitempty"`
	Steps          []StepResult `json:"steps"`
	OverallPassed  bool         `json:"overall_passed"`
	Decision       string       `json:"decision"`
	ViolationType  string       `json:"violation_type,omitempty"`
	ViolationDetail string      `json:"violation_detail,omitempty"`
}

// ValidatorInput is the input to a SafetyValidator.
type ValidatorInput struct {
	AgentID       string
	Output        string
	ChainID       string
	Context       string
	Direction     ContentDirection
	InheritedFrom string
}

// ValidatorResult is the output from a SafetyValidator.
type ValidatorResult struct {
	Passed     bool     `json:"passed"`
	Reason     string   `json:"reason,omitempty"`
	Violations []string `json:"violations,omitempty"`
	Confidence float64  `json:"confidence"`
}

// SafetyValidator is the common interface all validators implement.
//
// Validators evaluate agent outputs against specific safety criteria.
// Built-in validators: injection_defense, content_filter, four_laws_check.
type SafetyValidator interface {
	// Validate evaluates the input and returns a result.
	Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error)

	// Name returns the unique validator identifier (e.g., "injection_defense").
	Name() string

	// Description returns a human-readable description of the validator.
	Description() string
}

// ChainStep is a single step in a safety chain with its configured action.
type ChainStep struct {
	Validator  string  `yaml:"validator"`
	Action     Action  `yaml:"action"`
	Description string  `yaml:"description"`
}

// SafetyChainDefinition defines an ordered sequence of safety validations.
type SafetyChainDefinition struct {
	ID          string           `yaml:"id"`
	Description string           `yaml:"description"`
	Steps       []ChainStep      `yaml:"steps"`
	OnFailure   OnFailureAction  `yaml:"on_failure"`
}

// SafetyChain orchestrates ordered validator execution.
type SafetyChain struct {
	chainDef    SafetyChainDefinition
	validators  map[string]SafetyValidator
	auditLogger MultiAgentAuditLogger
}

// MultiAgentAuditLogger is the interface for multi-agent audit events.
type MultiAgentAuditLogger interface {
	LogChainValidation(ctx context.Context, entry AuditEntry)
}

// NewSafetyChain creates a safety chain from a definition and a validator registry.
func NewSafetyChain(def SafetyChainDefinition, validators []SafetyValidator, logger MultiAgentAuditLogger) *SafetyChain {
	m := make(map[string]SafetyValidator, len(validators))
	for _, v := range validators {
		m[v.Name()] = v
	}
	return &SafetyChain{
		chainDef:    def,
		validators:  m,
		auditLogger: logger,
	}
}

// Execute runs the safety chain against an agent's output.
//
// Validators execute in order. On the first failing step whose action is "block",
// the chain stops and returns a blocking result. "warn" steps log the violation
// but continue execution.
func (sc *SafetyChain) Execute(ctx context.Context, agentID, output, context string, parentAgentID string) *ChainResult {
	result := &ChainResult{
		ChainID:       sc.chainDef.ID,
		AgentID:       agentID,
		ParentAgentID: parentAgentID,
		Passed:        true,
		Steps:         make([]StepResult, 0, len(sc.chainDef.Steps)),
	}

	for i, step := range sc.chainDef.Steps {
		validator, ok := sc.validators[step.Validator]
		if !ok {
			result.Steps = append(result.Steps, StepResult{
				Step:      i + 1,
				Validator: step.Validator,
				Passed:    false,
				Reason:    fmt.Sprintf("validator %q not registered", step.Validator),
			})
			result.Violations = append(result.Violations, fmt.Sprintf("missing validator: %s", step.Validator))
			result.Passed = false
			result.Decision = string(ActionBlock)
			break
		}

		start := time.Now()
		vResult, err := validator.Validate(ctx, ValidatorInput{
			AgentID:        agentID,
			Output:         output,
			ChainID:        sc.chainDef.ID,
			Context:        context,
			Direction:      DirectionOutput,
			InheritedFrom: parentAgentID,
		})
		latency := time.Since(start).Milliseconds()

		stepResult := StepResult{
			Step:      i + 1,
			Validator: step.Validator,
			LatencyMs: latency,
		}

		if err != nil {
			stepResult.Passed = false
			stepResult.Reason = fmt.Sprintf("validator error: %v", err)
			result.Violations = append(result.Violations, fmt.Sprintf("%s error: %v", step.Validator, err))
		} else {
			stepResult.Passed = vResult.Passed
			stepResult.Reason = vResult.Reason
			stepResult.Violations = vResult.Violations
			if !vResult.Passed {
				result.Violations = append(result.Violations, vResult.Reason)
			}
		}

		result.Steps = append(result.Steps, stepResult)

		if !stepResult.Passed {
			result.Passed = false
			if step.Action == ActionBlock {
				result.Decision = string(ActionBlock)
				break
			}
			// warn: continue but track
			result.Decision = string(ActionWarn)
		}
	}

	if result.Passed {
		result.Decision = string(ActionAllow)
	}

	result.AuditEntry = sc.buildAudit(result)
	if sc.auditLogger != nil && result.AuditEntry != nil {
		sc.auditLogger.LogChainValidation(ctx, *result.AuditEntry)
	}

	return result
}

func (sc *SafetyChain) buildAudit(result *ChainResult) *AuditEntry {
	entry := &AuditEntry{
		Event:         "agent_chain_validated",
		Timestamp:     time.Now().UTC().Format(time.RFC3339),
		ChainID:       result.ChainID,
		AgentID:       result.AgentID,
		ParentAgentID: result.ParentAgentID,
		Steps:         result.Steps,
		OverallPassed: result.Passed,
		Decision:      result.Decision,
	}

	for _, step := range result.Steps {
		if !step.Passed {
			entry.ViolationType = step.Validator
			entry.ViolationDetail = step.Reason
			break
		}
	}

	return entry
}

// ChainID returns the chain definition ID.
func (sc *SafetyChain) ChainID() string {
	return sc.chainDef.ID
}

// === Constraint Inheritance ===

// AgentConstraint represents a single safety constraint on an agent.
type AgentConstraint struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// InheritanceResult tracks the merged constraints for a child agent.
type InheritanceResult struct {
	AgentID             string            `json:"agent_id"`
	ParentAgentID       string            `json:"parent_agent_id"`
	InheritedConstraints []AgentConstraint `json:"inherited_constraints"`
	OwnConstraints      []AgentConstraint `json:"own_effective_constraints"`
	MergedConstraints   []AgentConstraint `json:"merged_constraints"`
	DowngradePrevented  bool              `json:"downgrade_prevented"`
}

// ResolveConstraints merges parent and child constraints per the inheritance rule:
// child inherits all parent constraints and cannot remove any (only add).
func ResolveConstraints(agentID, parentAgentID string, parent, child []AgentConstraint) InheritanceResult {
	result := InheritanceResult{
		AgentID:        agentID,
		ParentAgentID:  parentAgentID,
		OwnConstraints: child,
	}

	// Build a map of existing child constraints by ID
	childByID := make(map[string]AgentConstraint, len(child))
	for _, c := range child {
		childByID[c.ID] = c
	}

	// All parent constraints must be present
	merged := make([]AgentConstraint, 0, len(parent)+len(child))
	for _, c := range parent {
		merged = append(merged, c)
		result.InheritedConstraints = append(result.InheritedConstraints, c)
	}

	// Add child-specific constraints that aren't inherited
	for _, c := range child {
		if _, inherited := findConstraintByID(parent, c.ID); !inherited {
			merged = append(merged, c)
		}
	}

	result.MergedConstraints = merged
	result.DowngradePrevented = len(merged) >= len(parent)

	return result
}

func findConstraintByID(constraints []AgentConstraint, id string) (AgentConstraint, bool) {
	for _, c := range constraints {
		if c.ID == id {
			return c, true
		}
	}
	return AgentConstraint{}, false
}

// === Conflict Resolution ===

// AgentOutput is a single output from one agent in a parallel group.
type AgentOutput struct {
	AgentID  string   `json:"agent_id"`
	Output   string   `json:"output"`
	Priority int      `json:"priority"`
	Actions  []string `json:"actions,omitempty"`
}

// ConflictResult captures the resolution of parallel agent conflicts.
type ConflictResult struct {
	Resolved         bool     `json:"resolved"`
	ResolvedOutput   string   `json:"resolved_output"`
	Conflicts        []string `json:"conflicts"`
	ResolutionMethod string   `json:"resolution_method"`
	Escalation       bool     `json:"escalation"`
}

// ResolveConflicts applies the chosen strategy to resolve parallel agent outputs.
func ResolveConflicts(outputs []AgentOutput, strategy ConflictStrategy) ConflictResult {
	if len(outputs) == 0 {
		return ConflictResult{
			Resolved:         true,
			ResolvedOutput:   "",
			ResolutionMethod: "none",
		}
	}
	if len(outputs) == 1 {
		return ConflictResult{
			Resolved:         true,
			ResolvedOutput:   outputs[0].Output,
			ResolutionMethod: "single_source",
		}
	}

	switch strategy {
	case ConflictPriority:
		return resolveByPriority(outputs)
	case ConflictIntersection:
		return resolveByIntersection(outputs)
	case ConflictUnion:
		return resolveByUnion(outputs)
	case ConflictEscalate:
		return resolveByEscalation(outputs)
	default:
		return resolveByPriority(outputs)
	}
}

func resolveByPriority(outputs []AgentOutput) ConflictResult {
	sorted := make([]AgentOutput, len(outputs))
	copy(sorted, outputs)
	sort.SliceStable(sorted, func(i, j int) bool {
		return sorted[i].Priority > sorted[j].Priority
	})
	return ConflictResult{
		Resolved:         true,
		ResolvedOutput:   sorted[0].Output,
		ResolutionMethod: string(ConflictPriority),
	}
}

func resolveByIntersection(outputs []AgentOutput) ConflictResult {
	// Find actions present in all agents' outputs
	actionCounts := make(map[string]int)
	for _, o := range outputs {
		// Deduplicate within a single agent's actions
		seen := make(map[string]bool)
		for _, a := range o.Actions {
			if !seen[a] {
				actionCounts[a]++
				seen[a] = true
			}
		}
	}

	var agreed []string
	total := len(outputs)
	for action, count := range actionCounts {
		if count == total {
			agreed = append(agreed, action)
		}
	}
	sort.Strings(agreed)

	return ConflictResult{
		Resolved:         len(agreed) > 0,
		ResolvedOutput:   strings.Join(agreed, "\n"),
		ResolutionMethod: string(ConflictIntersection),
		Conflicts:        findConflictingActions(outputs, total),
	}
}

func resolveByUnion(outputs []AgentOutput) ConflictResult {
	seen := make(map[string]bool)
	var all []string
	for _, o := range outputs {
		for _, a := range o.Actions {
			if !seen[a] {
				seen[a] = true
				all = append(all, a)
			}
		}
	}
	sort.Strings(all)

	return ConflictResult{
		Resolved:         true,
		ResolvedOutput:   strings.Join(all, "\n"),
		ResolutionMethod: string(ConflictUnion),
	}
}

func resolveByEscalation(outputs []AgentOutput) ConflictResult {
	conflicts := findConflictingActions(outputs, len(outputs))
	return ConflictResult{
		Resolved:         false,
		ResolutionMethod: string(ConflictEscalate),
		Escalation:       true,
		Conflicts:        conflicts,
	}
}

// findConflictingActions returns actions not agreed on by all agents.
func findConflictingActions(outputs []AgentOutput, total int) []string {
	actionCounts := make(map[string]int)
	for _, o := range outputs {
		seen := make(map[string]bool)
		for _, a := range o.Actions {
			if !seen[a] {
				actionCounts[a]++
				seen[a] = true
			}
		}
	}

	var conflicts []string
	for action, count := range actionCounts {
		if count < total {
			conflicts = append(conflicts, action)
		}
	}
	sort.Strings(conflicts)
	return conflicts
}

// === Built-in Validators ===

// InjectionDefenseValidator delegates to the Spec 01 injection detection pipeline.
type InjectionDefenseValidator struct {
	pipeline interface { // *injection.Pipeline, but avoid import cycle
		Detect(ctx context.Context, text string, source Source) InjectionResult
	}
}

// NewInjectionDefenseValidator creates an injection validator wrapping a pipeline.
func NewInjectionDefenseValidator(pipeline interface {
	Detect(ctx context.Context, text string, source Source) InjectionResult
}) *InjectionDefenseValidator {
	return &InjectionDefenseValidator{pipeline: pipeline}
}

// Validate implements SafetyValidator.
func (v *InjectionDefenseValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	result := v.pipeline.Detect(ctx, input.Output, SourceToolOutput)
	return &ValidatorResult{
		Passed:     result.Safe,
		Reason:     result.Reason,
		Confidence: result.Confidence,
		Violations: result.Categories,
	}, nil
}

// Name implements SafetyValidator.
func (v *InjectionDefenseValidator) Name() string {
	return "injection_defense"
}

// Description implements SafetyValidator.
func (v *InjectionDefenseValidator) Description() string {
	return "Detects prompt injection in agent output (Spec 01)"
}

// ContentFilterValidator delegates to the Spec 02 semantic content filter.
type ContentFilterValidator struct {
	filter *ContentFilter
}

// NewContentFilterValidator creates a content filter validator.
func NewContentFilterValidator(filter *ContentFilter) *ContentFilterValidator {
	return &ContentFilterValidator{filter: filter}
}

// Validate implements SafetyValidator.
func (v *ContentFilterValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	result, err := v.filter.Classify(ctx, input.Output, input.Direction)
	if err != nil {
		return nil, err
	}

	var violations []string
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

func formatFilterReason(result *ClassificationResult) string {
	if result.Safe {
		return ""
	}
	var blocked []string
	for _, c := range result.Categories {
		if c.Action == ActionBlock {
			blocked = append(blocked, c.ID)
		}
	}
	if len(blocked) > 0 {
		return fmt.Sprintf("Content blocked: %s", strings.Join(blocked, ", "))
	}
	return "Content flagged by policy"
}

// Name implements SafetyValidator.
func (v *ContentFilterValidator) Name() string {
	return "content_filter"
}

// Description implements SafetyValidator.
func (v *ContentFilterValidator) Description() string {
	return "Classifies content against safety taxonomy (Spec 02)"
}

// FourLawsValidator enforces the "Four Laws" of agent safety on agent output.
//
// The Four Laws:
//  1. An agent must not harm humanity, or through inaction allow humanity to come to harm.
//  2. An agent must obey orders given by authorized humans, except where they conflict with Law 1.
//  3. An agent must protect its existence, except where it conflicts with Laws 1 or 2.
//  4. An agent must preserve confidential information, except where it conflicts with Laws 1-3.
type FourLawsValidator struct {
	scopePatterns []string
}

// NewFourLawsValidator creates a Four Laws validator.
func NewFourLawsValidator(scopePatterns []string) *FourLawsValidator {
	return &FourLawsValidator{scopePatterns: scopePatterns}
}

// normalizeWhitespace strips zero-width characters and collapses runs of
// whitespace to a single space so harmful-command checks can't be trivially
// bypassed with invisible or decorative spacing (e.g. "rm -rf /​" or
// "r'''m -rf /").
func normalizeWhitespace(s string) string {
	// Drop zero-width / control characters used to evade pattern matching.
	zeroWidth := regexp.MustCompile(`[\x{200B}\x{200C}\x{200D}\x{2060}\x{FEFF}\x{00AD}]`)
	s = zeroWidth.ReplaceAllString(s, "")
	// Collapse all whitespace runs (spaces, tabs, newlines, etc.) to one space.
	ws := regexp.MustCompile(`\s+`)
	return ws.ReplaceAllString(s, " ")
}

// Validate implements SafetyValidator.
func (v *FourLawsValidator) Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error) {
	output := input.Output

	// Normalize so malicious spacing/zero-width tricks can't bypass checks.
	norm := normalizeWhitespace(output)
	lowerNorm := strings.ToLower(norm)

	// Law 1: Check for harmful commands (destructive, fork bomb, etc.)
	// Patterns use flexible whitespace so variants like "rm -rf /",
	// "rm -rf /​" (zero-width), or ":(){ :|:&};:" still match.
	harmPatterns := []*regexp.Regexp{
		regexp.MustCompile(`rm\s+-rf\s+/`),
		regexp.MustCompile(`mkfs\.[a-z0-9]+`),
		regexp.MustCompile(`dd\s+if=/dev/zero`),
		regexp.MustCompile(`:\(\)\s*\{\s*:\s*\|\s*:\s*&\s*\}\s*;\s*:`), // fork bomb
		regexp.MustCompile(`chmod\s+-R\s+777\s+/`),
		regexp.MustCompile(`>\s*/dev/sda`),
	}

	for _, re := range harmPatterns {
		if re.MatchString(norm) {
			return &ValidatorResult{
				Passed:     false,
				Reason:     fmt.Sprintf("Law 1 violation: destructive command detected (%s)", re.String()),
				Violations: []string{"harmful_command"},
				Confidence: 1.0,
			}, nil
		}
	}

	// Law 2: Scope limitation - output must not modify files outside declared scope
	if input.Context != "" {
		for _, pattern := range v.scopePatterns {
			if strings.Contains(output, pattern) {
				return &ValidatorResult{
					Passed:     false,
					Reason:     fmt.Sprintf("Law 2 violation: output exceeds declared scope (%s)", pattern),
					Violations: []string{"scope_violation"},
					Confidence: 0.9,
				}, nil
			}
		}
	}

	// Law 4: Check for data exfiltration patterns. These use real regex so
	// "curl http://pastebin.com" actually matches the intended "curl.*pastebin".
	exfilPatterns := []*regexp.Regexp{
		regexp.MustCompile(`curl.*pastebin`),
		regexp.MustCompile(`wget.*pastebin`),
		regexp.MustCompile(`curl.*webhook`),
		regexp.MustCompile(`nslookup.*control`),
	}
	for _, re := range exfilPatterns {
		if re.MatchString(lowerNorm) {
			return &ValidatorResult{
				Passed:     false,
				Reason:     fmt.Sprintf("Law 4 violation: potential data exfiltration (%s)", re.String()),
				Violations: []string{"data_exfiltration"},
				Confidence: 0.95,
			}, nil
		}
	}

	return &ValidatorResult{
		Passed:     true,
		Confidence: 0.99,
	}, nil
}

// Name implements SafetyValidator.
func (v *FourLawsValidator) Name() string {
	return "four_laws_check"
}

// Description implements SafetyValidator.
func (v *FourLawsValidator) Description() string {
	return "Verifies output follows the Four Laws of agent safety"
}

// === NoOp Audit Logger ===

// DefaultMultiAgentAuditLogger logs multi-agent events via slog.
type DefaultMultiAgentAuditLogger struct{}

// LogChainValidation logs a chain validation event.
func (d *DefaultMultiAgentAuditLogger) LogChainValidation(ctx context.Context, entry AuditEntry) {
	slog.Info("MULTI_AGENT_CHAIN",
		"event", entry.Event,
		"chain_id", entry.ChainID,
		"agent_id", entry.AgentID,
		"passed", entry.OverallPassed,
		"decision", entry.Decision,
	)
}
