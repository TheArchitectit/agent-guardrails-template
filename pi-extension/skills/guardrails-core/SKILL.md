---
id: guardrails-core
name: Pi Guardrails Core
description: Available guardrail tools and automatic enforcement behavior for pi agents
version: 1.1.0
tags: [safety, core, pi]
tools: [guardrail_init, guardrail_record_read, guardrail_verify_read,
        guardrail_set_scope, guardrail_check_scope, guardrail_record_attempt,
        guardrail_check_strikes, guardrail_reset_strikes, guardrail_check_halt,
        guardrail_log_violation, guardrail_status, guardrail_mcp,
        guardrail_pre_work_check, guardrail_detect_creep,
        guardrail_check_pattern, guardrail_validate_git]
---

# Pi Guardrails Core

The Four Laws of Agent Safety are enforced automatically via event handlers. You do NOT need to call guardrail tools before every edit — the extension handles enforcement automatically.

## Automatic Enforcement

The following rules are enforced without any explicit tool calls:

| Handler | Event | Law | What It Does |
|---------|-------|-----|-------------|
| Read tracking | `tool_result` | Law 1 | Tracks every file read; blocks edits to unread files |
| Scope enforcement | `tool_call` | Law 2 | Blocks edits to files outside authorized scope |
| Bash safety | `tool_call` | Law 4 | Blocks dangerous commands (rm -rf, sudo, force-push) |
| Injection defense | `tool_call` | Law 4 | Blocks/warns on prompt injection in tool input (bash/write/edit) |
| Output validation | `tool_result` | Law 3 | Detects secrets/PII in tool output, warns via status bar and logs violation |
| Content filtering | `tool_result` | Law 3 | Detects denied topic content in output, warns and logs violation |
| Canary tokens | `tool_result` | Law 3 | Detects canary tokens in output (data exfiltration), alerts via status bar |
| Permissions | `tool_call` | All | Gates tool access by auto/ask/blocked levels |

## Explicit Self-Check Tools

These tools are available for proactive checking before operations:

- `guardrail_verify_read` — Check if a file has been read before editing it
- `guardrail_check_scope` — Check if a path is within the authorized scope
- `guardrail_check_halt` — Evaluate whether an operation should be halted

## Session Management

- `guardrail_init` — Initialize a guardrails session at the start of each conversation
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

When the Go MCP server is available, `guardrail_mcp` proxies calls to it for enhanced enforcement including sandbox execution, canary tokens, extended validation, and policy retrieval.

## Enforcement Coverage Map

| Module | Skill | Automatic? | Explicit Tool? |
|--------|-------|------------|----------------|
| Read tracking | [[guardrails-core]] | Yes (tool_result) | guardrail_verify_read |
| Scope enforcement | [[guardrails-core]] | Yes (tool_call) | guardrail_check_scope |
| Bash safety | [[guardrails-core]] | Yes (tool_call) | guardrail_check_halt |
| Strike tracking | [[guardrails-core]] | No | guardrail_record_attempt/check_strikes |
| Injection defense | [[injection-defense]] | Yes (tool_call) | guardrail_mcp detect_injection |
| Output validation | [[output-security]] | Yes (tool_result) | guardrail_mcp scan_output |
| Content filtering | [[content-safety]] | Warn (tool_result) | guardrail_mcp filter_content |
| Tool permissions | [[tool-permissions]] | Yes (tool_call) | guardrail_mcp set_permission |
| Policy layers | [[policy-config]] | Yes (config load) | guardrail_mcp get_policy |
| Sandbox | [[sandbox-isolation]] | No | guardrail_mcp sandbox_run |
| Canary tokens | [[canary-tokens]] | Warn (tool_result) | guardrail_mcp canary_insert/check |
| Pre-work check | [[guardrails-core]] | No | guardrail_pre_work_check |
| Feature creep | [[guardrails-core]] | No | guardrail_detect_creep |
| Pattern rules | [[guardrails-core]] | No | guardrail_check_pattern |
| Git validation | [[guardrails-core]] | No | guardrail_validate_git |

## References

- [[injection-defense]] — Prompt injection detection and blocking
- [[output-security]] — Secret scanning and auto-redaction
- [[content-safety]] — Topic-based content filtering
- [[tool-permissions]] — Per-tool access control
- [[policy-config]] — Organization → team → project policy hierarchy
- [[sandbox-isolation]] — Docker-based command isolation
- [[canary-tokens]] — Honeypot data exfiltration detection
- [[guardrails-dashboard]] — Status bar and panel UI
