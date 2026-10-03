# Team and Advisor Tools

Ten tools that manage team configuration in `.teams/<project>.json` and
consult the built-in advisors.

**Read [Shared behaviour and known issues](../tools-reference.md#shared-behaviour)
first.**

## Cross-cutting facts

- **No RBAC anywhere.** A repo-wide grep for `admin|RBAC|RequireRole|team-lead`
  across `mcp-server/internal` returns zero hits in any team or advisor
  handler. Any authenticated bearer caller can perform any operation.
- `team.WithTestMode(true)` is a **no-op** — the parameter is discarded
  (`internal/team/manager.go:71-76`).
- State lives in `.teams/<project>.json`, path resolved by
  `ValidateProjectPath` (`internal/team/validation.go:58-84`) and written by
  `Manager.save()` via temp-file + rename.
- `project_name` is always validated: required, ≤64 chars, `[A-Za-z0-9_-]`
  only.

---

## Project lifecycle

### guardrail_team_init

Initializes a project with the 12 standard teams.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `project_name` | string | yes | |
| `teams` | array | **schema only — never read** | Marked required in the schema |

**Returns** plain text (not JSON): `✅ Initialized project '<name>' with 12 teams`.

**Mutates state** — writes `.teams/<name>.json`. **Overwrites silently, with
no confirmation.**

### guardrail_team_list

Lists teams as an ASCII table (`ID / Name / Phase / Status`).

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `project_name` | string | yes | |
| `phase` | string | no | **Not in the registered schema** |

Read-only. Status gains an `(n/m assigned)` suffix when `not_started` with
assignments.

> **Known issue.** `phase` must match `^Phase [1-3]$` to pass validation, but
> `Team.Phase` holds full strings like `"Phase 1: Strategy, Governance &
> Planning"` and `GetTeamsByPhase` compares with exact equality. A validated
> `phase` filter therefore **always returns an empty table**.

### guardrail_team_config_get

Returns the raw project configuration file.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `project_name` | string | yes | |
| `team_name` | string | optional in handler | Only echoed; schema marks it required |

**Returns** `{"project_name":…,"team_name":…,"config_path":…,"config":{…}}`,
with `config:{}` when the file is absent. Read-only.

> Does not distinguish teams — always returns the whole project file.

### guardrail_team_config_update

Shallow-merges a configuration fragment into the project file.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `project_name` | string | yes | |
| `team_name` | string | no (echoed) | |
| `config` | object | yes | Accepts an object or a JSON string |

**Returns** `{"project_name":…,"team_name":…,"config_path":…,"updated":true}`.

**Mutates state** via `os.WriteFile(0644)`, **bypassing `Manager.save()`** —
so it skips the temp-file + rename and any backup.

> **Known issues.** Top-level keys collide with `ProjectData`
> (`project_name`/`version`/`updated_at`/`teams`). If the existing file
> fails to parse it is **silently discarded and overwritten**
> (`tools_missing.go:154-161`). No RBAC, no confirmation.

### guardrail_team_remove

Removes a team from the project.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `project_name` | string | yes | |
| `team_id` | number | yes (1–12) | |
| `confirmed` | bool | no | false |

Unconfirmed returns `⚠️  Team removal requires confirmation. Set
confirmed=true to proceed.` with `IsError:false`; success returns
`✅ Removed team <n> from project '<p>'`.

> **Known issue.** The handler never calls `mgr.Load()` first, so
> `m.teams` is empty and `DeleteTeam` reports "team <n> not found" against
> any real, initialized project. As written this tool cannot succeed.

### guardrail_project_delete

Deletes the project file.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `project_name` | string | yes | |
| `confirmed` | bool | no | false |

Unconfirmed returns the same confirmation warning with `IsError:false`;
success returns `✅ Deleted project '<p>'`.

**Destructive** — `os.Remove(.teams/<p>.json)`, tolerating not-exist.
Requires explicit `confirmed=true`; **no RBAC, no backup, no audit trail**.

### guardrail_team_health

Reports team configuration health.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `project_name` | string | no | `"health-check"` |

**Returns** `{"status":"healthy","project":…,"total_teams":N,"active":N,
"completed":N,"not_started":N,"assigned_roles":N,"config_path":"…"}`, or
`{status, project, note:"Project not initialized, but team manager is
operational"}` when the manager fails to construct.

> **Known issues.** The handler never calls `mgr.Load()`, so on the healthy
> path every count is `0`. The schema description claims it validates the
> Python backend — it never invokes `team_manager.py`;
> `getTeamManagerPath()` (`team_tool_handlers.go:13`) is dead code.

---

## Assignment

### guardrail_team_assign

Assigns a person to a role in a team.

| Parameter | Type | Required | Validation |
|-----------|------|----------|------------|
| `project_name` | string | yes | |
| `team_id` | number | yes | Range 1–12 |
| `role_name` | string | yes | Whitelisted to the 48 standard role names |
| `person` | string | yes | ≤256 chars; rejects `; | & $ \` < > .. \` |

The registered schema instead declares `project_name`, `advisor_name`, and
`role` — `advisor_name` and `role` are **never read**.

**Returns** text: `✅ Assigned '<person>' to '<role>' in Team <n> (<name>)`.

**Mutates state** via `AssignRole` → `save()`.

> **Known issues.** Rate-limited by a token bucket of 100 requests per 60s
> keyed on a **hardcoded** `userID = "default"` (`team_tool_crud.go:191-203`),
> so the limit is effectively global rather than per-user. There is **no
> phase gate** — a Phase 3 team can be assigned and started without the
> prior phase being complete.

---

## Advisors

### guardrail_advisor_list

Lists the built-in advisors.

**Parameters:** none. **Returns** compact JSON
`models.AdvisorListResult`: `{"advisors":[…9…],"count":9}`. Each advisor
carries `id, name, alias, enforcement_level, scope, consults_with_teams[],
responsibility, persona_voice, deliverables[], trigger_patterns[],
assigned_to?`.

Static data; read-only. Ordering follows Go map iteration and is therefore
**non-deterministic**.

### guardrail_advisor_query

Consults one advisor.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `advisor_id` | string | yes | |
| `context` | string | no | **Accepted but never used** |
| `file_paths` | array of string | no | |

The registered schema declares `advisor_name` and `query` as required —
**wrong**. A spec-conformant client therefore always receives
`{"error":"advisor_id is required"}`.

**Returns** `models.AdvisorConsultResult`: `advisor_id, advisor_name,
alias, enforcement, severity, message, recommendations[], references?,
persona_voice`.

> **Known issues.** `context` is never passed to `generateAdvisorResponse`,
> and `readFileIfExists` is a stub returning `""`
> (`advisor_helpers.go:9-13`). The content-driven branches — retry,
> timeout, PII, query and audit patterns — therefore always fall through to
> the default "no changes detected" advice. Only filename and extension
> checks (dependency, API, `.html`/`.jsx`/`.tsx`/`.vue`, infra) actually
> fire.

---

## Not exposed

`guardrail_log_violation` and `guardrail_violation` are **not** MCP tools.

- `guardrail_log_violation` appears in zero Go files as a tool name; the
  only Go hit is a stale forward-reference in a comment
  (`tools_rules.go:186`).
- A handler `handleLogViolation` exists at `tools_enforcement.go:16` but has
  **zero callers** — unreachable, since no dispatch case or tool-list entry
  references it.
- `guardrail_violation` in Go is a `FailureEntry.Category` value inside that
  dead handler, not a tool name.
- Both names survive only in the `pi-extension/` TypeScript client harness,
  which registers them client-side.

See also: [`team-structure.md`](../../teams/team-structure.md) and
[`team-tools.md`](../../teams/team-tools.md).