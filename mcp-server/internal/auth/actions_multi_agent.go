package auth

// Spec 04 (multi-agent safety policies) tool classification.
//
// Same reasoning as actions_indirect_injection.go: these live in their own map
// so the audited mcpToolActions table — and the mutation fixture anchored to
// its exact text — stays byte-identical when a tool family is added.
//
// Classification is not optional: ClassifyMCPTool denies any tool it cannot
// classify, so an unregistered name is uncallable however correct its handler
// is. These three tools all analyse inputs and return a verdict; none of them
// writes durable state, so all three are read-class.
var multiAgentToolActions = map[string]ActionKind{
	// validate_agent_output runs a safety chain over text; it records an
	// audit entry but mutates no resource.
	"guardrail_validate_agent_output": ActionRead,
	// check_agent_constraints compares declared constraints to a policy.
	"guardrail_check_agent_constraints": ActionRead,
	// resolve_conflicts applies a strategy to in-memory outputs.
	"guardrail_resolve_conflicts": ActionRead,
}
