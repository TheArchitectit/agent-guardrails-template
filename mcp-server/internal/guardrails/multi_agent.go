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
	Step       int      `json:"step"`
	Validator  string   `json:"validator"`
	Passed     bool     `json:"passed"`
	LatencyMs  int64    `json:"latency_ms"`
	Reason     string   `json:"reason,omitempty"`
	Violations []string `json:"violations,omitempty"`
}

// ChainResult captures the outcome of executing a safety chain.
type ChainResult struct {
	ChainID       string       `json:"chain_id"`
	AgentID       string       `json:"agent_id"`
	ParentAgentID string       `json:"parent_agent_id,omitempty"`
	Passed        bool         `json:"passed"`
	Decision      string       `json:"decision"`
	Steps         []StepResult `json:"steps"`
	Violations    []string     `json:"violations,omitempty"`
	AuditEntry    *AuditEntry  `json:"-"`
}

// AuditEntry is a structured audit log entry for agent chain validation.
type AuditEntry struct {
	Event           string       `json:"event"`
	Timestamp       string       `json:"timestamp"`
	ChainID         string       `json:"chain_id"`
	AgentID         string       `json:"agent_id"`
	ParentAgentID   string       `json:"parent_agent_id,omitempty"`
	Steps           []StepResult `json:"steps"`
	OverallPassed   bool         `json:"overall_passed"`
	Decision        string       `json:"decision"`
	ViolationType   string       `json:"violation_type,omitempty"`
	ViolationDetail string       `json:"violation_detail,omitempty"`
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
type SafetyValidator interface {
	Validate(ctx context.Context, input ValidatorInput) (*ValidatorResult, error)
	Name() string
	Description() string
}

// ChainStep is a single step in a safety chain with its configured action.
type ChainStep struct {
	Validator   string `yaml:"validator"`
	Action      Action `yaml:"action"`
	Description string `yaml:"description"`
}

// SafetyChainDefinition defines an ordered sequence of safety validations.
type SafetyChainDefinition struct {
	ID          string          `yaml:"id"`
	Description string          `yaml:"description"`
	Steps       []ChainStep     `yaml:"steps"`
	OnFailure   OnFailureAction `yaml:"on_failure"`
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
		if _, exists := m[v.Name()]; exists {
			slog.Warn("duplicate validator name, second registration overwrites first",
				"name", v.Name(),
				"chain_id", def.ID,
			)
		}
		m[v.Name()] = v
	}
	return &SafetyChain{
		chainDef:    def,
		validators:  m,
		auditLogger: logger,
	}
}

// Execute runs the safety chain against an agent's output.
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
			// Validator internal error (err != nil) always fails closed:
			// treat as Block regardless of configured action, because a zero
			// or unknown Action must not silently downgrade to warn.
			if err != nil {
				result.Decision = string(ActionBlock)
			} else {
				switch step.Action {
				case ActionBlock, ActionScanAndBlock:
					result.Decision = string(ActionBlock)
				case ActionWarn, ActionScanAndWarn:
					result.Decision = string(ActionWarn)
				default:
					// Unknown/zero action fails closed to block.
					result.Decision = string(ActionBlock)
				}
			}
			if result.Decision == string(ActionBlock) {
				break
			}
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
	AgentID              string            `json:"agent_id"`
	ParentAgentID        string            `json:"parent_agent_id"`
	InheritedConstraints []AgentConstraint `json:"inherited_constraints"`
	OwnConstraints       []AgentConstraint `json:"own_effective_constraints"`
	MergedConstraints    []AgentConstraint `json:"merged_constraints"`
	DowngradePrevented   bool              `json:"downgrade_prevented"`
	EqualReplacements    int               `json:"equal_severity_replacements"`
}

// ResolveConstraints merges parent and child constraints per the inheritance rule.
// Child constraints with the same ID as a parent constraint override it only if
// they are strictly stronger (higher severity). Downgrade is prevented when a
// child tries to weaken a parent constraint, and equal-severity replacements are
// tracked separately so downgrade accounting stays accurate.
func ResolveConstraints(agentID, parentAgentID string, parent, child []AgentConstraint) InheritanceResult {
	result := InheritanceResult{
		AgentID:        agentID,
		ParentAgentID:  parentAgentID,
		OwnConstraints: child,
	}

	severityRank := map[string]int{
		"low": 1, "medium": 2, "high": 3, "critical": 4,
	}
	// An unknown/empty severity ranks below every defined severity, so it can
	// never override a parent that has a defined severity.
	rank := func(s string) int {
		if v, ok := severityRank[s]; ok {
			return v
		}
		return 0
	}

	childByID := make(map[string]AgentConstraint, len(child))
	for _, c := range child {
		childByID[c.ID] = c
	}

	merged := make([]AgentConstraint, 0, len(parent)+len(child))
	downgradePrevented := false
	equalReplacements := 0

	for _, c := range parent {
		result.InheritedConstraints = append(result.InheritedConstraints, c)
		if childOverride, ok := childByID[c.ID]; ok {
			// Child tries to override parent — check severity strictly.
			parentSev := rank(c.Severity)
			childSev := rank(childOverride.Severity)
			if childSev > parentSev {
				// Child is strictly stronger — use child's version.
				merged = append(merged, childOverride)
			} else if childSev == parentSev {
				// Equal severity: replacement is permitted (not a downgrade)
				// but tracked separately so accounting stays accurate.
				merged = append(merged, childOverride)
				equalReplacements++
			} else {
				// Child is weaker (or empty/unknown) — keep parent's
				// version; the downgrade is prevented.
				merged = append(merged, c)
				downgradePrevented = true
			}
		} else {
			merged = append(merged, c)
		}
	}

	for _, c := range child {
		if _, inherited := findConstraintByID(parent, c.ID); !inherited {
			merged = append(merged, c)
		}
	}

	result.MergedConstraints = merged
	result.DowngradePrevented = downgradePrevented
	// Expose equal-severity replacement count for accurate accounting.
	result.EqualReplacements = equalReplacements

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
