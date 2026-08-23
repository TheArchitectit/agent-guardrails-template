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
