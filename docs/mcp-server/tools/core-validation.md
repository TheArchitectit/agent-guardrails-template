# Core Validation Tools

Seven tools that validate raw inputs — bash commands, file edits, git
operations — against the rule store. Handlers live in
`mcp-server/internal/mcp/tools_core.go` and `tools_extended_validation.go`;
schemas in `tools_core_list.go:11-132`.

**Read [Shared behaviour and known issues](../tools-reference.md#shared-behaviour)
first.** Several tools here accept parameters the published schema omits.

---

## guardrail_init_session

Creates a session identifier for provenance tracking.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `user_id` | string | schema yes, **not enforced in code** | Discrarded on read; empty accepted |
| `environment` | string | no | Echoed back; never validated |

**Returns** `models.SessionInfo`:

```json
{"session_id":"<48 hex chars>","user_id":"u1","environment":"dev","start_time":"..."}
```

The `session_id` is 24 crypto-random bytes, hex-encoded.

> **Known issue.** The handler never registers the token in `s.sessions`, so
> tools that validate against that map (`guardrail_record_file_read`,
> `guardrail_record_attempt`, `guardrail_validate_production_first`, …)
> reject it as `"Invalid session token"`. Advisory only; never blocks.

---

## guardrail_validate_bash

Validates a bash command against prevention rules.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `command` | string | yes | Empty → error result |
| `working_dir` | string | no | **Cosmetic** — interpolated into `message` only, never validated |

**Returns** `validationResult`:

```json
{"valid":false,
 "violations":[{"rule_id":"BASH-001","rule_name":"...","severity":"critical",
                "message":"...","category":"bash",
                "matched_pattern":"rm -rf /","matched_input":"rm -rf /"}],
 "checked_at":"2026-01-01T00:00:00Z","command":"rm -rf /",
 "message":"Bash validation for working_dir=\"\""}
```

`matched_input` is truncated to 200 characters. `severity` is one of
`critical|error|warning|info`.

**Fails closed.** Errors if the validator is nil, if the engine errors, and
if a rule regex fails to compile (`engine.go:132` returns ERROR, never PASS).
There is no severity threshold — **any** violation at any severity sets
`IsError=true`.

---

## guardrail_validate_file_edit

Validates a file edit and checks read-before-edit.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `file_path` | string | yes | Regex-matched as well as the content |
| `old_string` | string | schema yes, **not enforced** | Used only if `new_string` is empty |
| `new_string` | string | schema yes, **not enforced** | The content actually scanned |
| `session_token` | string | **not in schema** | Read-before-edit token |
| `session_id` | string | **not in schema** | Fallback spelling of the same value |

Only `new_string` is scanned for content rules, falling back to
`old_string` when it is empty (`tools_core.go:82-85`).

**Returns** `validationResult` plus `file_path`, `was_read`, and — on a read
failure — `message: "File must be read before editing"`. `valid` requires
both no violations *and* `was_read`.

**Fails closed** on nil validator or engine error. Read-before-edit is
enforced **only** when a session token is supplied *and* `s.fileReadStore`
is set; otherwise `was_read` defaults to `true` and the check silently
passes. Violation id `FILE-READ-001` (`critical`).

> **Known issue.** Because `guardrail_init_session` does not register its
> token, `guardrail_record_file_read` rejects it — so the read-before-edit
> path is effectively unreachable through the normal
> init → record → edit flow.

---

## guardrail_validate_git_operation

Validates a git operation against prevention rules.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `operation` | string | yes | Empty → error result |
| `args` | array of string | no | Non-string elements are silently skipped |

The handler synthesizes `git <operation> <args...>` and regex-matches that
string.

**Returns** `validationResult` with `command` set to the synthesized git
string and `message` `Git operation "<cmd>" validated`.

**Fails closed**, same as bash. Categories `git`, plus legacy
`version_control`/`scm`. This validates the *proposed* command only — it is
not connected to `guardrail_validate_commit` or `guardrail_validate_push`.

---

## guardrail_pre_work_check

Scans the failure registry for known regressions matching a planned task.

| Parameter | Type | Required |
|-----------|------|----------|
| `task_description` | string | yes |

**Returns** `preWorkCheckResult`:

```json
{"safe":true,"task_description":"...","checked_at":"...",
 "known_regressions":[],"message":"..."}
```

Each regression carries `failure_id, category, severity, error_message,
root_cause, affected_files[], regression_pattern`.

**Fails closed** — nil DB, query failure, or empty description all yield
`safe:false`. Matching is heuristic: path-like tokens are extracted by
regex, then each failure matches by (a) `regression_pattern` compiled as a
regex, else (b) keyword substring overlap ≥3 chars, else (c) affected-file
substring match.

The schema calls this "mandatory", but nothing server-side enforces that it
runs first — it is caller-enforced convention only. It does not consult the
validation engine.

---

## guardrail_get_context

Reports the context and rule count for a path.

| Parameter | Type | Required |
|-----------|------|----------|
| `path` | string | no (defaults to CWD) |

**Returns** a bare map, not a named struct:

```json
{"path":"/repo","applicable_rules":42,"timestamp":"2026-10-03T00:00:00Z"}
```

**Advisory read**; `IsError` is always false.

> **Known issues.** `applicable_rules` is the validation engine's *total
> cached rule count*, not a per-path filtered set — the name is misleading
> and no path filtering happens. Unlike the `validate_*` tools there is no
> nil-validator guard, so a nil validator panics rather than erroring.

---

## guardrail_validate_scope

Verifies a path is inside an authorized scope.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `file_path` | string | yes | |
| `authorized_scope` | string | optional in schema, **required in practice** | |

**Returns** `models.ScopeValidationResult`:

```json
{"valid":false,"message":"File /repo/src-evil is OUTSIDE authorized scope /repo/src",
 "file_path":"/repo/src-evil","scope":"/repo/src","outside_scope":true}
```

**Fails closed, explicitly.** A missing `authorized_scope` returns
`valid:false` with an `UNKNOWN:`-prefixed message and `IsError=true` —
unknown is not a pass. Containment (`pathWithinScope`,
`tools_extended_validation.go:408-431`) resolves relative paths against
CWD, cleans, resolves symlinks (falling back to the deepest existing
parent for not-yet-created files), then compares with `filepath.Rel`. It is
boundary-aware, so `/repo/src-evil` does **not** pass for scope
`/repo/src`. Never touches the DB, rules, or engine.

---

## Where rules come from

None of these seven use the policy-pack engine (`guardrail_check_policy` /
`guardrail_classify_content` do). Rules are loaded by
`ValidationEngine.loadRulesFromDB`: in-memory TTL cache → Redis →
`ruleStore.GetActiveRules()`. Rules marked disabled are skipped
(`engine_rules.go:119`).