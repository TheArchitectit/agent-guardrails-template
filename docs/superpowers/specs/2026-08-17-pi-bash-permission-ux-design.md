# Pi Extension Bash Permission UX — Design

**Date:** 2026-08-17
**Status:** Approved (approach A — unified bash handler)
**Scope:** `pi-extension/` only

## Problem

Bash commands in the pi extension are not "prompted" — they are hard-blocked.

1. The permission handler (`handlers.ts:254`, registered first on `tool_call` in
   `index.ts:511`) defaults bash to level `"ask"`, but `"ask"` is implemented as
   `{ block: true }` with a message telling the agent to get approval in chat.
   There is no interactive prompt.
2. The bash safety handler (`handlers.ts:112`) matches a destructive-command
   denylist via `HaltChecker.checkCommand` and hard-blocks matches. All-or-nothing.
3. `PermissionManager.confirmToolCall` and the `pendingConfirmations` map
   (`permissions.ts:35,83`) are dead code — nothing populates or reads them.
4. Handler ordering means that when bash is `"ask"`, the dangerous-command
   classifier never runs; `ls` and `rm -rf /` get identical treatment.

Net effect: bash is either fully blocked or fully auto. There is no
per-run / just-once / permanent allowance path.

## Goals

- Interactive permission prompts for bash at execution time (run / once /
  always), using `ctx.ui` primitives.
- A persisted danger allow-list managed at runtime by the agent via a new
  `guardrail_allow_danger` tool.
- All dangerous commands are grantable (architect requirement). No
  fundamentally un-allowable tier; catastrophic-tier commands get a stronger
  prompt (type-back) instead of a hard block.
- Every allow/deny decision is audited in the violation/decision log.

## Non-goals

- Non-bash tools keep the existing permission handler unchanged.
- No changes to injection defense, output validation, scope, or Four Laws
  enforcement.
- No per-user/team policy extensions beyond what `PolicyLoader` already merges
  (project-level `.pi-guardrails.json` can override `allowDanger` and
  `toolPermissions` via the existing merge path).

## Architecture

A single handler, `createBashPermissionHandler`, owns the full bash decision.
It **replaces** both `createPermissionHandler` (bash branch) and
`createBashSafetyHandler` on the `tool_call` event. Non-bash tools continue
through `createPermissionHandler` as today.

Decision flow:

```
bash tool_call
  ├─ normalize(cmd)                     # trim + collapse whitespace
  ├─ allowDanger.enabled === false?  ──→ legacy deny behavior (block dangerous, ask blocks)
  ├─ allow-list match(cmd)?          ──→ allow (audit: allowlisted, entry info)
  ├─ classify(cmd) → category         # "safe" | dangerous tier category
  ├─ safe + level "auto"             ──→ allow
  ├─ safe + level "blocked"          ──→ block
  ├─ safe + level "ask"              ──→ simple prompt (scope choices)
  └─ dangerous                       ──→ strong prompt
        └─ catastrophic tier?        ──→ type-back confirmation prompt
  prompt resolves to:
    ├─ deny      ──→ { block: true } + audit denied
    ├─ once      ──→ allow this call only (no state written)
    ├─ session   ──→ allow + record in session allow set (cleared on restart)
    └─ always    ──→ allow + persist exact command to allow-list file
```

Catastrophic tier (type-back required, still grantable):
- fork bomb `:(){ :|:& };:`
- `mkfs`
- `rm -rf /` and `rm -rf /*`
- `dd if=… of=/dev/…`

Everything else matched by `DESTRUCTIVE_PATTERNS` / `DANGEROUS_COMMANDS`
(force-push, `sudo`, `git reset --hard`, `chmod 777`, etc.) is
dangerous-standard: strong prompt, no type-back.

## Components

### 1. `DangerAllowList` — new, `permissions/danger-allow-list.ts`

Persists to `allowlist.json` in the extension storage dir
(`config.ts:getAllowListPath()`, sibling of `config.json`).

Entry shapes:

```ts
interface ExactEntry  { type: "exact";   command: string; addedAt: string; reason?: string; source: "prompt" | "tool"; }
interface PatternEntry{ type: "pattern"; regex: string;   addedAt: string; reason: string;    source: "tool"; }
```

API:

| Method | Behavior |
|--------|----------|
| `matches(cmd): AllowListEntry \| undefined` | Exact entries compared against `normalize(cmd)`; pattern entries tested via `new RegExp(regex).test(normalize(cmd))` (compile once, cache; invalid regex at load time is skipped + logged) |
| `add(entry, reason, source)` | Normalizes exact commands; rejects duplicate exact commands; persists immediately |
| `remove(commandOrRegex): boolean` | Removes by value; persists |
| `list(): AllowListEntry[]` | All entries, insertion order |
| `clear()` | Empties; persists |

Constructor takes an optional storage path for testability (defaults to
`getAllowListPath()`).

### 2. `PermissionManager` changes — `permissions/permissions.ts`

- Add `ApprovalScope = "once" | "session" | "always"` (exported; used by the
  bash handler).
- Add `sessionDangerAllows: Set<string>` + `isSessionAllowed(cmd)` /
  `allowSession(cmd)` — in-memory "once-per-session" memory for bash
  approvals with scope `session`. (Kept here rather than in
  `DangerAllowList` because it must not persist.)
- **Remove** `pendingConfirmations` map and `confirmToolCall()` (dead code).
- `setPermission(toolName, level, opts?: { persist?: boolean })` — when
  `persist`, write `toolPermissions.tools` back to `config.json` (existing
  `getConfigPath()`).
- Non-bash `checkTool` unchanged.

### 3. `createBashPermissionHandler` — `handlers.ts`

Async handler implementing the decision flow above. Prompt primitives:

| Prompt kind | Primitive | Resolution |
|-------------|-----------|------------|
| Simple (safe cmd, level ask) | `ctx.ui.select(title, ["Allow once", "Allow for session", "Always allow", "Deny"])` | Maps to `once/session/always/deny` |
| Strong (dangerous cmd) | `ctx.ui.select(title, ["Allow once", "Allow for session", "Always allow", "Deny"])` with command + category shown in title/message | Same mapping |
| Catastrophic (type-back) | `ctx.ui.input("Type the command to confirm", cmd)` then scope select | Typed text must equal `normalize(cmd)` or the call is denied |

- **No UI available** (`!ctx.hasUI`): fall back to **deny** with reason
  `"Interactive confirmation required but no UI available; bash denied in
  non-interactive mode"`, audited. This preserves today's observable behavior
  for RPC/print mode.
- All decisions (allow via allow-list, allow via prompt, deny) produce an
  audit entry via `violationLog.log` with `law: "halt-when-uncertain"`,
  severity `"info"` for allows / `"warning"` for denies, and the command +
  category + scope in details. The `Violation.severity` type widens from
  `"warning" | "critical"` to `"info" | "warning" | "critical"`; existing
  `ViolationLog` summary counts only branch on critical/warning, so behavior
  is unchanged.

### 4. `HaltChecker` tweak — `standalone/halt-checker.ts`

`checkCommand` already returns `category` on halt, typed
`"destructive" | "elevated" | "network"`. Widen to
`"safe" | "destructive" | "elevated" | "network"` and always populate it
(non-halting commands report `"safe"`) so the prompt can name the risk.
`HaltChecker` also exposes an `isCatastrophic(cmd): boolean` helper used by
the bash handler to choose between strong prompt and type-back.

### 5. `guardrail_allow_danger` tool — `index.ts`, params in `types.ts`

Actions:

| Action | Params | Behavior |
|--------|--------|----------|
| `add` | `command` (exact) or `pattern` (regex), `reason` (required for pattern, optional for exact), `sessionOnly?: boolean` | `sessionOnly` records in session set only (exact only); otherwise persists to allow-list. Audited. |
| `remove` | `command` or `pattern` | Removes entry. Audited. |
| `list` | — | Returns all entries with type/reason/source/addedAt |
| `clear` | — | Empties allow-list. Audited as critical violation-log entry. |

Invalid regex is rejected with an error result, never throws.

### 6. Config — `config.ts` / `types.ts`

New section merged through the existing `loadConfig` / `PolicyLoader` path:

```json
{
  "allowDanger": {
    "enabled": true,
    "requireTypebackForCatastrophic": true
  }
}
```

- `enabled: false` → bash reverts to legacy deny behavior (dangerous blocked,
  `"ask"` blocks with chat-instruction message). The escape hatch for teams
  that want the old ceiling.
- `requireTypebackForCatastrophic: false` → catastrophic tier degrades to the
  strong prompt without type-back.

Storage: `getAllowListPath()` added to `config.ts`.

## Error handling

- Invalid regex in a persisted pattern entry → skipped at load, warning via
  `console.warn`; the allow-list continues to function.
- `ctx.ui.select` returning `undefined` (dialog dismissed/timeout) → treated
  as **deny**.
- `ctx.ui.input` returning `undefined` or a mismatched string → **deny**.
- `guardrail_allow_danger` with both `command` and `pattern` → error result.
- Allow-list file unreadable/corrupt → start empty, log warning, next write
  recreates the file.
- Bash handler throwing → caught by the pi extension runner; treated as deny.

## Audit

All decisions append to `violations.jsonl` via `ViolationLog.log`:
- allows via allow-list: `severity: "info"`, details include entry source
- allows via prompt: `severity: "info"`, details include chosen scope
- denies: `severity: "warning"`, details include reason
- allow-list mutations (`add`/`remove`/`clear` via tool): logged, `clear` at
  `critical`.

## Testing

Vitest (`pi-extension` already uses it):

1. `danger-allow-list.test.ts` — exact match, pattern match, normalization,
   duplicate rejection, persistence round-trip, invalid-regex skip.
2. `permissions.test.ts` (update) — remove dead-code tests, add
   session-danger-allows, `setPermission` persist behavior.
3. `handlers.test.ts` (new) — bash handler flow with a mock `ctx.ui`:
   allow-list short-circuit, safe+ask prompt scopes, dangerous prompt,
   catastrophic type-back (match/mismatch), no-UI deny, `enabled: false`
   legacy path.
4. `halt-checker.test.ts` (update) — category populated on non-halt,
   `isCatastrophic` classification.
5. Tool test for `guardrail_allow_danger` actions incl. invalid regex.

## Files touched

| File | Change |
|------|--------|
| `pi-extension/permissions/danger-allow-list.ts` | New |
| `pi-extension/permissions/permissions.ts` | Modify (scopes, session allows, remove dead code, persist) |
| `pi-extension/handlers.ts` | Replace bash branch; new `createBashPermissionHandler` |
| `pi-extension/standalone/halt-checker.ts` | Category on non-halt; `isCatastrophic` |
| `pi-extension/index.ts` | Register new tool; swap handler registrations |
| `pi-extension/config.ts` | `getAllowListPath()`, `allowDanger` merge |
| `pi-extension/types.ts` | `ApprovalScope`, `AllowDangerConfig`, `AllowListEntry`, tool params |
| `pi-extension/README.md` | Update Tool Permissions + Bash safety sections |
| `pi-extension/**/*.test.ts` | Per Testing above |
