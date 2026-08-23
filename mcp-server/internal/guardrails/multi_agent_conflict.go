package guardrails

import (
	"sort"
	"strings"
)

// AgentOutput is a single output from one agent in a parallel group.
type AgentOutput struct {
	AgentID  string   `json:"agent_id"`
	Output   string   `json:"output"`
	Priority int      `json:"priority"`
	Actions  []string `json:"actions,omitempty"`
}

// ConflictResult captures the resolution of parallel agent conflicts.
//
// Resolved is true only when the chosen strategy produced a definitive
// answer. Union reports Resolved only when the merged actions actually
// converged (all agents agreed on the same set of actions); otherwise it sets
// Partial=true to signal that the merge is a best-effort union, not a
// convergence. Priority resolution always picks the highest-priority agent,
// breaking ties deterministically by agent name.
type ConflictResult struct {
	Resolved         bool     `json:"resolved"`
	ResolvedOutput   string   `json:"resolved_output"`
	Conflicts        []string `json:"conflicts"`
	ResolutionMethod string   `json:"resolution_method"`
	Escalation       bool     `json:"escalation"`
	Partial          bool     `json:"partial,omitempty"`
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
	// Deterministic tie-break: for equal priority, pick the agent with the
	// lexicographically smallest name so the result is reproducible.
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Priority != sorted[j].Priority {
			return sorted[i].Priority > sorted[j].Priority
		}
		return sorted[i].AgentID < sorted[j].AgentID
	})
	return ConflictResult{
		Resolved:         true,
		ResolvedOutput:   sorted[0].Output,
		ResolutionMethod: string(ConflictPriority),
	}
}

func resolveByIntersection(outputs []AgentOutput) ConflictResult {
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
	total := len(outputs)
	// Count how many agents carry each action; union "converges" only when
	// every agent agreed on the exact same action set.
	actionCounts := make(map[string]int)
	for _, o := range outputs {
		local := make(map[string]bool)
		for _, a := range o.Actions {
			if !seen[a] {
				seen[a] = true
				all = append(all, a)
			}
			if !local[a] {
				local[a] = true
				actionCounts[a]++
			}
		}
	}
	sort.Strings(all)

	converged := len(all) > 0
	for _, count := range actionCounts {
		if count != total {
			converged = false
			break
		}
	}

	return ConflictResult{
		Resolved:         converged,
		Partial:          !converged,
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
