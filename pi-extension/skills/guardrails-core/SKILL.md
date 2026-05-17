---
id: guardrails-core
name: Pi Guardrails Core
description: Available guardrail tools and automatic enforcement behavior for pi agents
version: 1.0.0
tags: [safety, core, pi]
tools: [guardrail_init, guardrail_record_read, guardrail_verify_read,
        guardrail_set_scope, guardrail_check_scope, guardrail_record_attempt,
        guardrail_check_strikes, guardrail_reset_strikes, guardrail_check_halt,
        guardrail_log_violation, guardrail_status, guardrail_mcp]
---

# Pi Guardrails Core

The Four Laws of Agent Safety are enforced automatically via event handlers. You do NOT need to call guardrail tools before every edit — the extension handles enforcement automatically.

## Automatic Enforcement

The following rules are enforced without any explicit tool calls:

- **Read tracking**: Every file you read is tracked automatically. If you try to edit a file you haven't read, the edit is blocked (Law 1).
- **Scope enforcement**: If scope is set, edits to files outside the authorized paths are blocked (Law 2).
- **Bash safety**: Dangerous commands (e.g., `rm -rf /`, `sudo`, `git push --force main`) are blocked automatically (Law 4).

## Explicit Self-Check Tools

These tools are available for proactive checking before operations:

- `guardrail_verify_read` — Check if a file has been read before editing it
- `guardrail_check_scope` — Check if a path is within the authorized scope
- `guardrail_check_halt` — Evaluate whether an operation should be halted

## Session Management

- `guardrail_init` — Initialize a guardrails session at the start of each conversation. Sets up scope and tracking.
- `guardrail_status` — Get the full session state summary (scope, strikes, violations, MCP status)

## Three Strikes Workflow

The Three Strikes rule enforces Law 4 (Halt When Uncertain):

1. `guardrail_record_attempt` — Record each task attempt (success or failure)
2. `guardrail_check_strikes` — Check the strike count for a task
3. `guardrail_reset_strikes` — Reset strikes after a successful resolution or user escalation

After 3 consecutive failures on the same task, the system recommends halting and escalating to the user.

## Scope Management

- `guardrail_set_scope` — Define which file paths the agent is authorized to operate on
- `guardrail_check_scope` — Verify a path is in scope before operating on it

## Violation Logging

- `guardrail_log_violation` — Log a guardrail violation with law, severity, and context

## MCP Bridge

When the Go MCP server is available, `guardrail_mcp` proxies calls to it for enhanced enforcement. Check mode with `guardrail_status` — the `mcpConnected` field indicates whether the bridge is active.
