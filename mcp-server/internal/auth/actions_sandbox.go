package auth

// Spec 03 (runtime sandbox isolation) tool classification.
//
// Same reasoning as actions_multi_agent.go: these live in their own map so the
// audited mcpToolActions alignment group — and the mutation fixture anchored to
// its exact text — stays byte-identical when a tool family is added.
//
// The split here is deliberate and is the security-relevant part:
//
//   - guardrail_sandbox_execute RUNS A COMMAND. That is not a read. It is
//     classified mutate because it has a side effect on the host, and any
//     credential that can call it can run code. Calling it read-class because
//     it "returns a result" would hand every reader credential an execution
//     path.
//   - guardrail_sandbox_config CHANGES SECURITY POSTURE. A credential that can
//     lower a sandbox policy can weaken every later execution, so it is an
//     admin mutation and takes the audit-before-effect path.
var sandboxToolActions = map[string]ActionKind{
	"guardrail_sandbox_execute": ActionMutate,
	"guardrail_sandbox_config":  ActionAdmin,
}

// sandboxSensitiveTools require a durable audit record before the effect runs.
// Both qualify: one executes arbitrary commands, the other redefines the
// isolation those commands run under.
var sandboxSensitiveTools = map[string]bool{
	"guardrail_sandbox_execute": true,
	"guardrail_sandbox_config":  true,
}
