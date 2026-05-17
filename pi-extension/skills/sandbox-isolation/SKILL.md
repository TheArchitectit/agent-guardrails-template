---
id: sandbox-isolation
name: Sandbox Isolation
description: Run untrusted or risky commands in Docker-based sandbox with resource limits and network isolation
version: 1.0.0
tags: [safety, pi, security]
---

# Sandbox Isolation

The pi-guardrails extension provides a Docker-based sandbox for executing commands that might be risky, untrusted, or potentially destructive. Sandboxed commands run with network isolation, memory limits, and CPU constraints.

## When To Use the Sandbox

Use the sandbox when:

- Running **untrusted scripts** from external sources
- Executing **build commands** in unfamiliar projects
- Testing **destructive operations** (file deletion, schema changes)
- Running **user-provided code** snippets
- Any command where the blast radius is unclear

Do NOT use the sandbox when:

- Running standard `git`, `read`, or `edit` operations
- The command is well-understood and low-risk
- Docker is not available (check `isAvailable` first)

## How It Works

The sandbox wraps `docker run` with:

- **No network access** by default (`--network=none`)
- **Read-only mounts** for source code inspection
- **Read-write mounts** only for explicitly designated output paths
- **Memory limits** (e.g., `512m`) to prevent resource exhaustion
- **CPU limits** (e.g., `1.0` core) to prevent CPU abuse
- **Timeouts** (default 30s) to prevent hanging processes
- **Auto-cleanup** (`--rm`) — containers are removed after execution

## MCP Bridge

When connected, use `guardrail_mcp` with action `sandbox_run`:

```
action: sandbox_run
command: ["npm", "test"]
readOnlyMounts: ["/project/src"]
readWriteMounts: ["/project/test-results"]
networkAccess: false
memoryLimit: "512m"
timeout: 60000
```

The result includes `exitCode`, `stdout`, `stderr`, and `timedOut` fields.

## Fallback When Docker Is Unavailable

If Docker is not installed or the daemon is not running, `sandbox_run` will return an error. In this case:

1. **Inform the user** that sandbox is unavailable
2. **Ask for explicit permission** before running the command outside the sandbox
3. **Proceed only after approval** — do not assume consent

## What To Do When a Sandbox Run Fails

- **Exit code != 0**: The command itself failed — investigate stderr
- **timedOut: true**: The command exceeded the timeout — increase or simplify the command
- **exit code null**: Docker failed to start — check Docker availability

## References

- [[injection-defense]] — Detect injection in sandboxed command output
- [[output-security]] — Scan sandboxed output for secrets
- [[tool-permissions]] — Permission levels for sandbox use
