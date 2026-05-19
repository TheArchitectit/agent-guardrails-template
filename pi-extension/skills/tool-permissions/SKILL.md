---
id: tool-permissions
name: Tool Permission Control
description: Per-tool permission levels (auto/ask/blocked) that gate which tools the agent can use
version: 1.0.0
tags: [safety, pi]
---

# Tool Permission Control

The pi-guardrails extension enforces tool-level permissions. Before any tool call executes, the permission handler checks whether the tool is allowed, requires confirmation, or is blocked.

## Automatic Enforcement

The permission handler runs on every `tool_call` event before execution:

- **auto**: Tool executes immediately (no confirmation needed)
- **ask**: Tool execution is **paused** and you must ask the user for approval
- **blocked**: Tool execution is **refused** with reason

## Default Permission Matrix

| Tool | Level | Rationale |
|------|-------|-----------|
| bash | ask | Shell commands can be destructive |
| write | auto | File creation is scoped to write paths |
| edit | auto | Edits are guarded by scope + read tracking |
| read | auto | Reading is always safe |
| grep | auto | Search is always safe |
| glob | auto | File listing is always safe |
| ls | auto | Directory listing is always safe |

## When You Need Confirmation

If a tool is set to `ask` level:

1. **Stop** before executing the tool
2. **Explain** to the user what the tool will do
3. **Ask** "Should I proceed with [tool] to [action]?"
4. **Wait** for explicit approval
5. **Execute** only after approval — do not assume consent

## Session Overrides

Permissions can be adjusted within a session via `guardrail_mcp` with action `set_permission`:

```
action: set_permission
toolName: bash
level: auto
reason: User approved bash for this task
```

This sets the override for the remainder of the session only.

## Configuration

Permissions are configured in `.pi-guardrails.json`:

```json
{
  "toolPermissions": {
    "defaultLevel": "auto",
    "tools": {
      "bash": "ask",
      "write": "auto",
      "edit": "auto",
      "read": "auto"
    }
  }
}
```

Organization and team policies can override these defaults — see [[policy-config]].

## What To Do When Blocked

1. **Do not retry** the blocked tool call
2. **Inform the user** which tool was blocked and why
3. **Suggest an alternative** approach that uses allowed tools
4. If the block seems incorrect, suggest the user update their permission configuration

## References

- [[policy-config]] — How permission policies cascade from org → team → project
- [[guardrails-core]] — Core guardrail tools overview
