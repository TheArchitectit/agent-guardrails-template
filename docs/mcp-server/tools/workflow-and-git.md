# Workflow and Git Tools

Eight tools covering the commit → push → regression-prevention workflow.
Handlers live in `tools_extended_validation.go`, `tools_extended_production.go`,
and `tools_extended_replacement.go`.

**Read [Shared behaviour and known issues](../tools-reference.md#shared-behaviour)
first.** Every tool on this page has at least one published-schema parameter
the handler never reads.

## Workflow order

```
pre_work_check → (work) → validate_commit → validate_push
        ↓                                ↓
prevent_regression            detect_feature_creep
        ↓                                ↓
validate_production_first → validate_exact_replacement → verify_fixes_intact
```

`validate_production_first` **only** passes for `test`/`infrastructure` code
if production code was already recorded in the same session — so the
production call must come first.

---

## guardrail_validate_commit

Validates a commit message against Conventional Commits.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `message` | string | yes | |
| `files` | array | **schema only — never read** | |

**Returns:**

```json
{"valid":false,"format_compliant":true,
 "issues":["Message exceeds 72 characters (consider using body for details)"],
 "message":"Feat: Add X","conventional_type":"Feat","scope":""}
```

**Fails closed** — `IsError = !valid`. No state, no ordering dependency.

Rules: `^(\w+)(?:\(([^)]+)\))?!?: (.+)$`; types
feat/fix/docs/style/refactor/perf/test/chore/build/ci/revert; ≤72 chars; no
trailing period; lowercase first character unless it is in a proper-noun
allowlist (`isProperNounStart`).

---

## guardrail_prevent_regression

Checks planned changes against active failure-registry entries.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `file_paths` | array of string | no | Schema declares `file_path`/`changes` — **wrong** |
| `code_content` | string | no | |

Both empty returns `{"matches":[],"checked":0}` with `IsError=false`.

**Returns** `matches[]` (`failure_id, category, severity, message,
root_cause, regression_pattern, affected_files[]`) and `checked` = count of
`file_paths`.

**`IsError = len(matches) > 0`** — any match blocks. Reads active rows via
`GetActiveByFiles`; a DB error returns a plain-text error result.

> **Known issue.** When `code_content` is supplied but the failure's
> `regression_pattern` is empty, the code takes the else branch
> (`tools_extended_validation.go:240`) and includes the failure
> unconditionally.

---

## guardrail_check_test_prod_separation

Verifies test/production isolation for one file.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `file_path` | string | yes | |
| `environment` | string | effectively yes | Only `prod`/`test` are matched |

**Returns** `valid`, `violations[]` (plain strings — no rule_id or
severity), `file_path`, `environment`. **Fails closed**.

prod rules: `test_db`/`test_database`, `localhost:5433|5434`,
`testMode = true`. test rules: `prod_db`/`production_database`,
`https?://api.production.`, hardcoded AWS keys, `(?i)production.*secret`.

> **Known issue.** An unreadable file yields empty content and **no
> violations** — this path fails open silently.

---

## guardrail_validate_push

Pre-push safety validation.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `branch` | string | no | |
| `is_force` | bool | no | |
| `has_unpushed_commits` | bool | no | |
| `remote` | **schema only — never read** | | |

**Returns** `valid`, `can_push`, `warnings[]`, `branch`, `is_force`.
**Fails closed** on `IsError = !valid`.

Force push → invalid, `can_push=false`. Protected branches `main, master,
production, release` (and `<branch>/…` prefixes) warn when not forced and
invalidate when forced. No unpushed commits → warning only.

> **Known issue.** A branch containing spaces sets `valid=false` but leaves
> `can_push=true` (`tools_extended_validation.go:375`).

---

## guardrail_validate_production_first

Requires production code to exist before test/infrastructure code.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `session_token` | string | yes | Must exist in `s.sessions` |
| `file_path` | string | yes | Schema declares only `path` — wrong and not required |
| `code_type` | string | yes | `production`, `test`, or `infrastructure` |

**Returns** `valid`, `message`, `production_code_exists`. **Writes state**
on every call; marks verified when `code_type` is `production`.

**Ordering:** `test`/`infrastructure` passes only if
`HasProductionCode(session_token)` is true — a prior production call in the
same session is mandatory. Production code always passes.

> **Known issue.** Unlike `verify_fixes_intact`, there is **no nil-guard** on
> `s.productionCodeStore`; with no DB configured it will nil-panic.

---

## guardrail_detect_feature_creep

Decides whether a diff exceeds the original task scope.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `session_token` | string | yes | Schema declares `task_id`/`current_changes` — **wrong** |
| `file_path` | string | yes | |
| `git_diff` | string | yes | |
| `change_description` | string | no | |
| `is_new_file` | bool | no | |

**Returns** `creep_detected`, `violations[]` (`type, severity, message`),
`diff_summary` (`"+42/-3 lines, 2 new functions"`), `total_changes`
(`{additions, deletions}`), `recommendation`.

**`IsError = creep_detected` — a single warning blocks.** Rules: new file
>50 additions; >1 new function; >1 new struct; any added line matching
`http|endpoint|route|api|REST`; >100 additions; refactor keywords with no
new symbols; any "better/improved/optimized/…" keyword; >3 new imports.

> **Known issue.** The `"Halt - critical feature creep detected"` branch is
> unreachable from diff analysis — only the invalid-session path sets
> severity `error`, and that returns `CreepDetected:false` with
> `IsError=true`.

---

## guardrail_verify_fixes_intact

Verifies recorded bug fixes are still present in a file.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `session_token` | string | yes | |
| `file_path` | string | yes | |
| `modified_content` | string | no | Falls back to reading from disk, then `original_content` |
| `original_content` | string | no | |
| `bug_id` | **schema only — never read** | | |

**Returns** `all_fixes_intact`, `verify_summary` (`"1/2 fixes verified
intact"`), `fixes[]` (`failure_id, status, fix_type, affected_file,
verification_message`), `recommendation`.

**This tool creates the fix records** it then verifies: `GetOrCreate`
defaults status to `confirmed` and hashes the content, so the first call
freezes the fix content for that `(session, failure_id)`. `fix_type` is
inferred: `regression_pattern` → `regex`; else `root_cause` →
`code_change`; else `config`. Status is `confirmed|modified|removed`
(`fix_verification.go:202`). Requires both `s.db` and
`s.fixVerificationStore` (nil-guarded).

Zero fixes → `all_intact=true`, `IsError=false` (no-op pass).

---

## guardrail_validate_exact_replacement

Confirms an edit was an exact replacement.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `session_token` | string | yes | |
| `file_path` | string | yes | |
| `original_content` | string | no | |
| `modified_content` | string | no | |
| `replacement_type` | string | no | **Dead parameter** — never referenced by the analysis |
| `target_string` | **schema only — never read** | | |

**Returns** `exact_match`, `violations[]` (`type, severity, message`),
`diff_stats{additions,deletions}`, `recommendation`.

Violation severities: `new_import`/`type_change`/`variable_rename`/
`code_reorganized` = warning; `debug_added`/`extra_function`/
`function_removed` = error; `formatting`/`comment_change` = info.

**`IsError = !exact_match`**, but info-only violations are silently
downgraded to `exact_match=true` (`tools_extended_helpers.go:232`) while
still being listed. More than one critical → "Reject and use exact code".

> **Known issues.** (a) The empty-`original_content` file-creation path
> returns *before* the session-existence check, so `session_token` is not
> validated on that path. (b) `diff_stats` comes from an index-wise line
> comparison while violations come from a set difference — the two are
> computed differently and can disagree.