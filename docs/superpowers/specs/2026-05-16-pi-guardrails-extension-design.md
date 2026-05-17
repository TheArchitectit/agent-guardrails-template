# Pi Guardrails Extension Design

**Date:** 2026-05-16
**Branch:** feature/pi-extension
**Package:** @thearchitectit/pi-guardrails
**Status:** Draft

---

## Goal

Make the Agent Guardrails Template work as a first-class pi coding agent extension so that pi users get full safety enforcement (Four Laws, Three Strikes, scope validation, halt conditions) without requiring the Go MCP server. When the MCP server is available, the extension bridges to the full 30+ tool suite.

## Why

- The guardrails template already supports Claude Code, Cursor, OpenCode, OpenClaw, Windsurf, and Copilot — but not pi.
- Pi has its own extension system with tools, TUI overlays, event handlers, and skills. A native extension gives tighter integration than MCP alone.
- Users who don't run the Go MCP server (no Docker/Podman, no Postgres) should still get core guardrails. The hybrid approach means standalone works everywhere; MCP bridge unlocks the full suite when deployed.

## Architecture

```
pi agent
  |
  v
pi-guardrails extension (TypeScript)
  |
  +-- Standalone core (no external deps)
  |     FileReadStore       - tracks which files the agent has read
  |     StrikeCounter       - three-strikes rule per task
  |     ScopeValidator      - checks edits stay in authorized scope
  |     HaltChecker         - evaluates halt conditions
  |     ProductionFirst     - enforces test/prod separation
  |     SessionStore        - per-session state in JSON
  |     ViolationLog        - append-only violation log
  |
  +-- MCP bridge (when Go server available)
  |     MCPClient           - connects to guardrail_mcp via stdio transport
  |     MCPTools            - proxies 30+ MCP tools via action dispatch
  |
  +-- TUI
  |     GuardrailsPanel     - full overlay dashboard (/guardrails)
  |     GuardrailsStatus     - status bar entry via ctx.ui.setStatus()
  |
  +-- Skills
        guardrails-core skill       - teaches agents about guardrail_* tools + enforcement
        guardrails-dashboard skill  - teaches agents about TUI panel + status bar
        (parent template SKILL.md files are NOT modified)
```

## Pi Extension API

The extension imports from `@earendil-works/pi-coding-agent` (same as pi-subagents) for the ExtensionAPI, and from `@earendil-works/pi-tui` for TUI Component/Focusable/TUI types. These are peer dependencies provided by the pi runtime.

**Key API surfaces used:**

| Feature | API | Reference |
|---------|-----|-----------|
| Tool registration | `pi.registerTool(tool)` (1-arg object literal) | pi-subagents (extension/index.ts:475), pi-messenger |
| Event blocking | `pi.on("tool_call", handler)` returning `{ block: true, reason }` | pi-messenger reservation enforcement (index.ts:1110) |
| Event tracking | `pi.on("tool_result", handler)` for post-call side effects | pi-messenger activity tracking (index.ts:703) |
| Session lifecycle | `pi.on("session_start", handler)` / `pi.on("session_shutdown", handler)` | pi-messenger (lines 781, 1073), pi-subagents (lines 543, 547) |
| Status display | `ctx.ui.setStatus(key, text)` | pi-messenger (index.ts:286) |
| TUI overlay | `ctx.ui.custom<T>(factory, { overlay: true })` | pi-messenger MessengerOverlay |
| Slash command | `pi.registerCommand(name, { description, handler })` | pi-messenger `/messenger` command |
| Skill discovery | `"pi": { "skills": "./skills" }` in package.json | Both extensions |

## Pi-Native Tools

### Standalone Tools (work without MCP server)

All registered via `pi.registerTool()` at extension init using `ToolDefinition` object literals (same pattern as pi-subagents at extension/index.ts:398-475). All parameter schemas use TypeBox `Type.Object()` from `@sinclair/typebox` (imported directly, same as pi-messenger line 14).

**Name collision note:** Several standalone tools share conceptual overlap with MCP server tools but intentionally use different names to avoid shadowing. When the MCP server is connected, both standalone and bridge tools are available — the standalone tools provide a lightweight local-only path while the bridge tools use the full Go server engine. The extension does NOT need to choose one or the other; the agent can use either set depending on the operation.

| Tool | Purpose | MCP server equivalent |
|------|---------|----------------------|
| `guardrail_standalone_init` | Initialize a guardrails session (standalone mode) | `guardrail_init_session` (server) |
| `guardrail_record_read` | Mark a file as having been read by the agent | `guardrail_record_file_read` (server) |
| `guardrail_verify_read` | Check whether a file was read before an edit is attempted | `guardrail_verify_file_read` (server) |
| `guardrail_set_scope` | Define the authorized file scope for the session | `guardrail_validate_scope` (server) |
| `guardrail_check_scope` | Validate that a file path is within authorized scope | (no direct equivalent) |
| `guardrail_record_attempt` | Log a task attempt (increments strike counter) | `guardrail_record_attempt` (server, same name) |
| `guardrail_check_strikes` | Get current strike count and status for a task | `guardrail_validate_three_strikes` (server) |
| `guardrail_reset_strikes` | Reset strike counter for a task (after user escalation) | `guardrail_reset_attempts` (server) |
| `guardrail_check_halt` | Evaluate all halt conditions and return go/no-go | `guardrail_check_halt_conditions` (server) |
| `guardrail_log_violation` | Record a safety violation with severity and details | `guardrail_log_violation` (server, same name) |
| `guardrail_status` | Get full guardrails state summary (reads, strikes, scope, violations) | (no direct equivalent — status is local state only) |

### MCP Bridge Tools (require Go server running)

All tools registered by the Go MCP server are proxied as stubs at extension init via a single tool (`guardrail_mcp`) with an `action` parameter, following pi-messenger's single-tool pattern. The bridge dynamically reads the server's tool list at connection time rather than hardcoding action names — this ensures new tools added to the Go server are automatically available without extension updates.

The Go MCP server currently registers 50+ tools across these categories (not exhaustive — see `mcp-server/internal/mcp/server.go` for the authoritative list):

| Category | Examples |
|----------|---------|
| Enforcement | `guardrail_validate_bash`, `guardrail_validate_file_edit`, `guardrail_validate_git_operation`, `guardrail_validate_commit`, `guardrail_validate_push`, `guardrail_validate_scope`, `guardrail_validate_three_strikes`, `guardrail_validate_exact_replacement`, `guardrail_validate_production_first` |
| Halt & uncertainty | `guardrail_check_halt_conditions`, `guardrail_record_halt`, `guardrail_acknowledge_halt`, `guardrail_check_uncertainty` |
| File tracking | `guardrail_record_file_read`, `guardrail_verify_file_read` |
| Violations & regression | `guardrail_log_violation`, `guardrail_prevent_regression`, `guardrail_check_test_prod_separation`, `guardrail_record_attempt`, `guardrail_reset_attempts` |
| Language & docs | `guardrail_detect_language`, `guardrail_get_language_profile`, `guardrail_list_languages`, `guardrail_validate_language_rules`, `guardrail_get_standard`, `guardrail_get_workflow`, `guardrail_search_docs` |
| Prevention & detection | `guardrail_get_prevention_rules`, `guardrail_check_pattern`, `guardrail_detect_feature_creep`, `guardrail_verify_fixes_intact` |
| Context & workflow | `guardrail_get_context`, `guardrail_pre_work_check`, `guardrail_validate_game_build` |
| Team management | `guardrail_team_init`, `guardrail_team_list`, `guardrail_team_assign`, `guardrail_team_unassign`, `guardrail_team_start`, `guardrail_team_status`, `guardrail_team_delete`, `guardrail_team_health`, `guardrail_team_size_validate`, `guardrail_agent_team_map`, `guardrail_phase_gate_check`, `guardrail_project_delete` |
| Skills & marketplace | `guardrail_install_skills`, `guardrail_marketplace_add`, `guardrail_marketplace_list`, `guardrail_marketplace_search`, `guardrail_marketplace_remove` |
| Resources | `Quick Reference`, `Active Prevention Rules`, `Agent Guardrails`, `Four Laws of Agent Safety`, `Halt Conditions` |

Each stub's `execute()` method:
1. Checks whether MCP is connected at call time.
2. If connected: proxies the call to the Go server and returns the result.
3. If not connected: returns an informative message: *"Tool requires the Guardrail MCP server. Start it and run guardrail_standalone_init with MCP endpoint configured."*

## Tool Schemas

All tool parameter schemas use TypeBox `Type.Object()` definitions (imported from `@sinclair/typebox` directly, same pattern as pi-messenger line 14). Below is the logical schema for each tool — TypeBox equivalents are straightforward.

### guardrail_standalone_init

```
Input: {
  projectSlug: string,       // project identifier
  agentType?: string,        // e.g. "pi", "claude-code"
  scope?: string[],          // initial authorized file paths/globs
  rules?: string[]           // enabled rule IDs
}
Output: {
  sessionId: string,
  mode: "standalone" | "mcp-bridge",
  availableTools: string[],
  mcpConnected: boolean
}
```

### guardrail_record_read

```
Input: { filePath: string }
Output: { recorded: boolean, filePath: string, readAt: string }
```

### guardrail_verify_read

```
Input: { filePath: string }
Output: { wasRead: boolean, filePath: string, readAt?: string }
```

### guardrail_set_scope

```
Input: { paths: string[], reason?: string }
Output: { scope: string[], reservedAt: string }
```

### guardrail_check_scope

```
Input: { filePath: string, operation: "read" | "edit" | "delete" }
Output: { inScope: boolean, reason?: string }
```

### guardrail_record_attempt

```
Input: { task: string, success: boolean, error?: string }
Output: { task: string, strikeCount: number, maxReached: boolean }
```

### guardrail_check_strikes

```
Input: { task: string }
Output: { task: string, strikeCount: number, maxReached: boolean, details: { attempt: number, success: boolean, timestamp: string }[] }
```

### guardrail_reset_strikes

```
Input: { task: string }
Output: { task: string, reset: boolean }
```

### guardrail_check_halt

```
Input: { operation: string, filePath?: string, details?: string }
Output: {
  shouldHalt: boolean,
  reasons: string[],
  severity: "none" | "warning" | "critical",
  suggestions: string[]
}
```

### guardrail_log_violation

```
Input: {
  law: "read-before-edit" | "stay-in-scope" | "verify-before-commit" | "halt-when-uncertain",
  severity: "warning" | "critical",
  details: string,
  filePath?: string,
  operation?: string
}
Output: { logged: boolean, violationId: string, timestamp: string }
```

### guardrail_status

```
Input: {}
Output: {
  sessionId: string,
  mode: "standalone" | "mcp-bridge",
  scope: string[],
  filesRead: number,
  activeStrikes: { task: string, count: number }[],
  violations: { total: number, critical: number, warning: number },
  mcpConnected: boolean
}
```

## Event-Based Enforcement

Pi does not have distinct lifecycle hooks (no `onBeforeFileEdit`, `onBeforeBash`, etc.). Instead, the extension registers `pi.on("tool_call", ...)` handlers that inspect the event and conditionally block tool calls by returning `{ block: true, reason: "..." }`.

**Input field convention:** pi's built-in tools use `event.input.path` for file paths in edit/write operations (confirmed in pi-messenger lines 714, 1114). The extension uses `input.path` consistently for edit/write, and `input.path` for read operations. Type assertions (`as Record<string, unknown>`) follow pi-messenger's pattern at lines 711, 1113.

### Registered Event Handlers

**0. Session lifecycle handlers** (`session_start` / `session_shutdown`):

```typescript
pi.on("session_start", async (_event, ctx) => {
  // Initialize session store from disk or create fresh
  sessionStore.initialize(ctx.cwd);
  // Attempt MCP connection (non-blocking, fire-and-forget)
  mcpClient.tryConnect(config.mcpEndpoint).catch(() => {});
  // Set initial status bar
  if (config.statusBarEnabled && ctx.hasUI) {
    ctx.ui.setStatus("guardrails", "[g: ok]");
  }
});

pi.on("session_shutdown", async () => {
  // Flush violation log to disk
  violationLog.flush();
  // Close MCP connection if open
  await mcpClient.close();
  // Remove status bar entry
  // (pi runtime cleans this up automatically on shutdown)
});
```

Same pattern as pi-messenger (lines 781, 1073) and pi-subagents (lines 543, 547).

**1. Read tracking handler** (`tool_result` listener):

```typescript
pi.on("tool_result", (event, ctx) => {
  if (event.toolName === "read") {
    const input = event.input as Record<string, unknown>;
    const filePath = typeof input.path === "string" ? input.path : null;
    if (filePath) {
      fileReadStore.record(filePath);
    }
  }
});
```

This runs after the read completes — no blocking needed. Same pattern as pi-messenger's activity tracking at line 703.

**2. Pre-edit enforcement handler** (`tool_call` listener):

```typescript
pi.on("tool_call", (event, _ctx) => {
  if (!["edit", "write"].includes(event.toolName)) return;

  const input = event.input as Record<string, unknown>;
  const filePath = typeof input.path === "string" ? input.path : null;
  if (!filePath) return;

  // Check Law 1: Read before editing
  const wasRead = fileReadStore.wasRead(filePath);
  if (!wasRead) {
    return { block: true, reason: `Guardrail violation: file '${filePath}' was not read before edit (Law 1: Read Before Editing). Use 'read' tool first.` };
  }

  // Check Law 2: Stay in scope
  const inScope = scopeValidator.isInScope(filePath, "edit");
  if (!inScope) {
    return { block: true, reason: `Guardrail violation: file '${filePath}' is outside authorized scope (Law 2: Stay in Scope). Authorized: ${scopeValidator.getScope().join(", ")}` };
  }
});
```

Returns `{ block: true, reason }` to stop the tool call. Same pattern as pi-messenger's reservation enforcement at line 1110.

**3. Bash command safety handler** (`tool_call` listener):

```typescript
pi.on("tool_call", (event, _ctx) => {
  if (event.toolName !== "bash") return;

  const input = event.input as Record<string, unknown>;
  const cmd = typeof input.command === "string" ? input.command : "";

  // Detect dangerous commands (standalone mode)
  const dangerous = haltChecker.checkCommand(cmd);
  if (dangerous.shouldHalt) {
    return { block: true, reason: dangerous.reason };
  }

  // If MCP is connected, validate via server
  if (mcpClient?.isConnected()) {
    // Note: this requires the handler to be async — verify pi runtime supports async tool_call handlers
    // pi-messenger uses async handlers (line 1110: async (event, _ctx) => ...)
  }
});
```

Note: MCP validation in a `tool_call` handler requires async execution. pi-messenger confirms this is supported (line 1110: `pi.on("tool_call", async (event, _ctx) => { ... })`). The standalone check runs synchronously; MCP validation is async and only runs when connected.

Command classification:
- `git commit` → commit validation (secrets-in-diff check, test-existence check)
- `git push` → push validation (no force to main, etc.)
- `rm -rf /`, destructive shell → blocked by denylist

### Handler Ordering

Handlers are registered in `index.ts` in this order:
1. Session lifecycle (session_start — initialization; session_shutdown — cleanup)
2. Read tracking (tool_result — non-blocking, always passes)
3. Pre-edit enforcement (tool_call — blocking)
4. Bash safety (tool_call — blocking)

If the guardrails extension and pi-messenger are both installed and monitoring the same tool calls, whichever handler returns `{ block: true }` first stops the tool call. Guardrails enforcement is independent of reservation enforcement — they check different conditions. The pi runtime processes all handlers.

## TUI Components

### GuardrailsPanel (full overlay)

Registered via `pi.registerCommand("guardrails", { handler: ... })` — opened with `/guardrails`. Also triggerable from the `guardrail_status` tool result via auto-open.

Imports Component/Focusable/TUI types from `@earendil-works/pi-tui`, same pattern as pi-messenger's MessengerOverlay. Shows:

- **Safety Score** — aggregate health (green/yellow/red) based on violation count and strike state
- **Four Laws Status** — which laws have been satisfied/violated in the current session
- **Strike Tracker** — per-task strike counts with attempt history
- **Scope** — authorized paths with visual indicators
- **Violation Log** — chronological list of violations with severity badges
- **MCP Status** — connection state to Go server, available bridge tools

Panel auto-closes on guardrail-clean state after user acknowledgment.

### GuardrailsStatus (status bar entry)

Uses `ctx.ui.setStatus("guardrails", ...)` — a single-line status string rendered by pi's status bar (same API as pi-messenger at line 286). Shows:

```
[g: !!2/3 src/ !3v mcp:*]
```

Segments (each shown only when relevant, from most to least important):
- **Strikes**: `!!2/3` when any strikes active, else omitted
- **Scope**: `src/` (first authorized path prefix) or `unscoped` in dimmed text
- **Violations**: `!3v` when violations exist
- **MCP**: `mcp:*` (connected) or `mcp:.` (not connected)

Updated after each event handler triggers or tool call completes.

## Skills

### Pi-specific skill files (not modifying template SKILL.md)

The extension declares skills in `package.json` via `"pi": { "skills": "./skills" }`. These are **new pi-specific skill files** inside the extension directory. The 11 existing template SKILL.md files are NOT modified — they use `applies_to` for Claude Code / Cursor / etc., which is a different skill discovery mechanism.

New skills in `pi-extension/skills/`:

| Skill | File | Purpose |
|-------|------|---------|
| guardrails-core | `skills/guardrails-core/SKILL.md` | Teaches pi agents about available `guardrail_*` tools and event-based enforcement |
| guardrails-dashboard | `skills/guardrails-dashboard/SKILL.md` | How to use and interpret the guardrails panel and status bar |

### guardrails-core skill

```yaml
id: guardrails-core
name: Pi Guardrails Core
description: Available guardrail tools and automatic enforcement behavior for pi agents
version: 1.0.0
tags: [safety, core, pi]
applies_to: [pi]
tools: [guardrail_standalone_init, guardrail_record_read, guardrail_verify_read,
        guardrail_set_scope, guardrail_check_scope, guardrail_record_attempt,
        guardrail_check_strikes, guardrail_reset_strikes, guardrail_check_halt,
        guardrail_log_violation, guardrail_status]
```

Teaches pi agents:
- The Four Laws are enforced automatically via event handlers — reads are tracked, unread edits are blocked, out-of-scope edits are blocked
- Explicit tool calls are available for agents to self-check before operations
- Use `guardrail_check_halt` before uncertain operations
- Use `guardrail_record_attempt` / `guardrail_check_strikes` for the three-strikes workflow
- Use `guardrail_status` to get a full state summary
- MCP bridge tools require the Go server; check mode with `guardrail_status`

### guardrails-dashboard skill

```yaml
id: guardrails-dashboard
name: Pi Guardrails Dashboard
description: How to use and interpret the guardrails panel and status bar in the pi TUI
version: 1.0.0
tags: [safety, pi, tui]
applies_to: [pi]
```

Teaches pi agents:
- `/guardrails` slash command opens the guardrails panel overlay
- Status bar shows compact safety state at a glance
- What each section of the panel means (safety score, strikes, scope, violations)
- When to proactively check guardrails state vs let automatic enforcement handle it

## Data Storage

All extension state stored at `~/.pi/agent/extensions/pi-guardrails/` (consistent with pi-subagents' convention of storing config under `~/.pi/agent/extensions/subagent/config.json`):

```
~/.pi/agent/extensions/pi-guardrails/
  config.json                    # user config
  sessions/
    {sessionId}.json             # per-session state
  violations.jsonl               # append-only violation log
```

### Session file structure

```json
{
  "id": "session-abc123",
  "projectSlug": "my-project",
  "createdAt": "2026-05-16T18:00:00Z",
  "scope": {
    "paths": ["src/", "tests/"],
    "reason": "User-defined scope for auth refactor"
  },
  "filesRead": {
    "src/auth/login.ts": "2026-05-16T18:01:00Z",
    "src/auth/logout.ts": "2026-05-16T18:02:00Z"
  },
  "strikes": {
    "fix-auth-bug": {
      "attempts": [
        { "success": false, "error": "test failed", "timestamp": "2026-05-16T18:05:00Z" },
        { "success": true, "timestamp": "2026-05-16T18:07:00Z" }
      ]
    }
  },
  "mcpEndpoint": "http://localhost:8094",
  "mcpConnected": true
}
```

### Config file structure

```json
{
  "mcpEndpoint": "http://localhost:8094",
  "enabledRules": ["four-laws", "three-strikes", "scope-validator"],
  "autoRegister": true,
  "defaultScope": [],
  "maxStrikes": 3,
  "statusBarEnabled": true,
  "panelAutoOpen": false
}
```

### Security: API keys

The Guardrail MCP server API key is **not** stored in the config file. Supplied via `PI_GUARDRAILS_MCP_API_KEY` environment variable instead. Extension reads it at init and passes to the MCP client. This prevents plaintext credential storage.

## File Structure

Inside the template repo at `pi-extension/`:

```
pi-extension/
  package.json                    # @thearchitectit/pi-guardrails
  index.ts                        # Extension entry point + all registrations
  install.mjs                     # npx installer script
  config.ts                       # Config loading + defaults (env var for API key)
  types.ts                        # Shared types (TypeBox schemas)
  standalone/
    file-read-store.ts            # File read tracking
    strike-counter.ts             # Three-strikes implementation
    scope-validator.ts            # Scope enforcement
    halt-checker.ts               # Halt condition evaluation (incl. command denylist)
    production-first.ts           # Test/prod separation check
    session-store.ts              # Session persistence (JSON)
    violation-log.ts              # Append-only violation logger
  mcp-bridge/
    mcp-client.ts                 # Stdio MCP client (@modelcontextprotocol/sdk optional dep, StdioClientTransport)
    mcp-tools.ts                  # MCP tool stubs + action dispatch (dynamic tool list from server)
  tui/
    guardrails-panel.ts           # Full overlay dashboard component
    guardrails-status.ts          # Status bar entry (ctx.ui.setStatus)
    render.ts                     # TUI rendering helpers
  skills/
    guardrails-core/
      SKILL.md                    # Core guardrails tool usage skill
    guardrails-dashboard/
      SKILL.md                    # Dashboard/panel usage skill
  README.md
```

## MCP Bridge Behavior

- All MCP bridge tool stubs are **registered at extension init** (no dynamic tool registration).
- At `session_start`, the extension attempts to connect to the configured MCP server binary using `@modelcontextprotocol/sdk` with **stdio transport** (`StdioClientTransport`) — the Go server is spawned as a child process.
- If the spawn succeeds, each stub's `execute()` proxies the call to the Go server.
- If it fails, each stub's `execute()` returns an informative message: *"Tool requires the Guardrail MCP server. Start it and run guardrail_standalone_init to retry connection."*
- Reconnection uses exponential backoff (1s base, 30s max, 5 attempts max) with jitter — the agent can retry manually via `guardrail_standalone_init`.
- The `@modelcontextprotocol/sdk` package is an optional peer dependency — the extension handles its absence gracefully by marking MCP as permanently unavailable.
- The tool list is read dynamically from the server at connection time (via MCP `tools/list`) rather than hardcoded in the extension, so new server tools are automatically available.

## Extension Interaction

When multiple extensions are installed (e.g., guardrails + pi-messenger):

- Guardrails `tool_call` handlers check read-before-edit and scope, returning `{ block: true }` on violation.
- pi-messenger `tool_call` handlers check file reservations, returning `{ block: true }` on conflict.
- These are independent checks — both fire, and pi runtime processes all handlers sequentially.
- If both block, the user sees both reasons.
- No coordination protocol needed — the checks are orthogonal and non-overlapping.

## Error Handling

- All standalone tools handle corrupted/missing session files by recreating from defaults.
- MCP bridge failures are caught per-call and return informative standalone-mode warnings, not hard errors.
- Violation log uses append-only writes to minimize data loss risk.
- TUI components degrade gracefully if session data is unavailable (show "No session" state).
- Missing optional `@modelcontextprotocol/sdk` dep: MCP bridge marked unavailable, standalone mode unaffected.

## Dependencies

| Package | Type | Purpose |
|---------|------|---------|
| `@earendil-works/pi-coding-agent` | peer | Extension API (ExtensionAPI, ToolDefinition, pi.on, pi.registerTool, ui.setStatus) |
| `@earendil-works/pi-tui` | peer | TUI Component/Focusable/TUI types for overlay |
| `@modelcontextprotocol/sdk` | optional peer | MCP client for connecting to Go guardrail server (graceful fallback) |
| `@sinclair/typebox` | (via pi-coding-agent re-export) | Tool parameter schema definitions |

## Testing

- Unit tests for each standalone module (file-read-store, strike-counter, scope-validator, halt-checker, production-first).
- Integration test: create full extension session, run tool calls, verify state persistence.
- TUI snapshot tests for the panel and status bar renders.
- MCP bridge integration test using a mock MCP server (only when optional dep is available).
- Event handler tests: simulate `tool_call` events and verify `{ block: true }` returns.

## Scope Boundaries

**In scope:**
- Pi extension package with standalone tools + MCP bridge
- Full TUI guardrails panel (/guardrails) and status bar entry
- Pi-specific skill files (guardrails-core, guardrails-dashboard)
- Event-based enforcement (tool_call/tool_result listeners for Four Laws)
- npx installer
- Unit and integration tests

**Out of scope:**
- Changes to the Go MCP server code
- Changes to the existing 11 SKILL.md files (no `applies_to: [pi]` added)
- Changes to existing Claude/Cursor/etc. plugin formats
- New guardrail rules or logic beyond what already exists
- NPM publishing (manual step after code is ready)

## Success Criteria

1. `pi install npm:@thearchitectit/pi-guardrails` installs and registers the extension.
2. Running `guardrail_status` in a pi session shows standalone mode with all core tools working.
3. Starting the Go MCP server and re-initializing enables MCP bridge tools (stubs return real results, not "requires server" messages).
4. The `/guardrails` slash command opens the full TUI panel.
5. The status bar shows compact safety state (strikes, scope, violations, MCP status).
6. Event handlers automatically enforce the Four Laws: edits to unread files are blocked with a violation message.
7. Both pi-specific skills appear in pi's skill registry (discovered from `"pi": { "skills": "./skills" }` in package.json).
8. No external runtime dependencies for standalone mode (pure TypeScript + Node fs). MCP SDK is optional.
