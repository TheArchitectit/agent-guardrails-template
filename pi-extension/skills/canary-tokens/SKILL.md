---
id: canary-tokens
name: Canary Tokens
description: Insert honeypot tokens into sensitive files to detect if an agent leaks their contents in output
version: 1.0.0
tags: [safety, security, pi]
---

# Canary Tokens

Canary tokens are unique identifiers inserted into sensitive files. If an agent's output contains a canary token, it indicates potential data exfiltration — the agent has leaked file contents that should have stayed private.

## How Canary Tokens Work

1. **Insert**: A unique token (e.g., `CANARY_A3F8B2D1E9C4...`) is placed in a sensitive file
2. **Monitor**: Every tool result is scanned for canary token matches
3. **Detect**: If a canary appears in output, it means the agent read the file and leaked its contents
4. **Alert**: The violation is logged and the agent is notified

## When To Use Canary Tokens

Use canaries when working with:

- **Credentials files** (`.env`, `config/credentials.yml`, `secrets.json`)
- **Private key files** (`.pem`, `.key`, SSH keys)
- **Database configurations** (connection strings, migration files)
- **Internal documentation** (architecture diagrams, security policies)
- **Any file the user flags as sensitive**

## Automatic Enforcement

When canary tokens are configured, the output validation handler automatically scans tool results for triggered canaries. Since `tool_result` handlers are side-effect only, the content will still reach you — but the violation is detected and flagged:

1. The violation is **logged** with severity "critical"
2. A **status bar alert** warns of possible data exfiltration
3. The triggered canary is marked as **triggered** in the tracking store

## What To Do When a Canary Is Triggered

1. **STOP** — do not continue sending output that contains file contents
2. **Inform the user** that a canary token was triggered (indicates data exfiltration risk)
3. **Do not reveal** the canary token value itself
4. **Log the violation** with `guardrail_log_violation` if not auto-logged
5. **Suggest remediation** — the file may need to be excluded from agent access

## MCP Bridge

When connected, `guardrail_mcp` provides canary management:

- `action: canary_insert` — Insert a new canary into a file path
- `action: canary_check` — Check text for triggered canaries
- `action: canary_list` — List all active and triggered canaries

## Token Format

- **Prefix**: `CANARY_` (configurable)
- **Suffix**: 32-character hex string (e.g., `A3F8B2D1E9C4017FA3F8B2D1E9C4017F`)
- **Full token**: `CANARY_A3F8B2D1E9C4017FA3F8B2D1E9C4017F`
- **HTML insertion**: `<!-- CANARY_A3F8B2D1E9C4017FA3F8B2D1E9C4017F -->` (for markup files)
- **Plain insertion**: `CANARY_A3F8B2D1E9C4017FA3F8B2D1E9C4017F` (for config/text files)

## References

- [[injection-defense]] — Complementary input-side protection
- [[output-security]] — Secret scanning that works alongside canary detection
