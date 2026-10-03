# Halt, Attempts and Content Tools

Twelve tools covering provenance (read-before-edit), three-strikes attempt
tracking, uncertainty and halt conditions, and content classification.

**Read [Shared behaviour and known issues](../tools-reference.md#shared-behaviour)
first.** `session_token` is declared in every handler on this page but is
missing from most published schemas.

## Provenance cluster

### guardrail_record_file_read

Records that an agent read a file (Law-1 enforcement).

| Parameter | Type | Required |
|-----------|------|----------|
| `session_token` | string | yes |
| `file_path` | string | yes |

**Returns** `{"success":true,"session_token":"…","file_path":"…","recorded_at":"…"}`.

**Fails closed** (invalid session → `IsError=true`). **Mutates persistent
state** via `FileReadStore.CreateWithStrings`. Must precede
`guardrail_verify_file_read`.

### guardrail_verify_file_read

Verifies a file was read in this session before editing it.

| Parameter | Type | Required |
|-----------|------|----------|
| `session_token` | string | yes |
| `file_path` | string | yes |

**Returns** `models.FileReadVerificationResult`:

```json
{"valid":true,"was_read":true,"read_at":"2026-01-01T00:00:00Z",
 "session_id":"…","file_path":"…"}
```

**Read-only.** Not fail-closed — it *reports* rather than blocks:

| Situation | Result |
|-----------|--------|
| Missing `session_token`/`file_path` | `valid:false`, `IsError:true` |
| Unknown session | `valid:true, was_read:false, message:"Session not found or expired"`, `IsError:false` |
| No record in DB | `valid:true, was_read:false, message:"File has not been read"`, `IsError:false` |

---

## Attempt tracking (three strikes)

### guardrail_record_attempt

Records a failed attempt.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `session_token` | string | yes | |
| `task_id` | string | no | |
| `error_message` | string | no | `"Unknown error"` |
| `error_category` | string | no | `other` |

**Returns** `{"valid":true,"attempt_number":N,"strikes_remaining":N,
"should_halt":bool,"max_attempts":N,"message":"…"}`.

**Fails closed.** Has a `defer recover()`, but on panic the named return is
left unset, so the recover swallows without producing a result (unlike
`handleResetAttempts` and `handleRecordHalt`, which set an error result).

**Mutates** a persistent counter. Feeds `guardrail_validate_three_strikes`
and Check 1 of `guardrail_check_halt_conditions`.

### guardrail_reset_attempts

Clears a task's attempt counter on success.

| Parameter | Type | Required |
|-----------|------|----------|
| `session_token` | string | yes |
| `task_id` | string | no |

**Returns** `{"valid":true,"reset":true,"attempts_cleared":N,"message":"…"}`.

**Fails closed**, with a recover that sets a proper error result. Marks
pending attempts resolved (not deleted). The opposite transition of
`guardrail_record_attempt`.

### guardrail_validate_three_strikes

Reports whether the failure threshold has been reached.

| Parameter | Type | Required |
|-----------|------|----------|
| `session_token` | string | yes |
| `task_id` | string | no |

**Returns** `{"valid":true,"halt":bool,"attempts_count":N,"max_attempts":N,
"should_escalate":bool,"strikes_remaining":N,"message":"…"}`.

**Read-only / advisory** — `halt` and `should_escalate` are signals, never
`IsError` on the happy path. Requires a non-nil `taskAttemptStore`.

---

## Uncertainty and halt

### guardrail_check_uncertainty

Forces self-reflection and returns an escalation level.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `session_token` | string | yes | **Not validated** — used directly as the session id |
| `current_task` | string | yes | |
| `self_assessment` | string | yes | |
| `context_data` | object | no | Reads `error_count`, `duration_minutes` |

**Returns** `models.UncertaintyCheckResult`: `session_id, current_level,
previous_level, escalated, decision_made, context_summary, recommendation`.
Levels: `critical|blocked|high|medium|investigating|low|resolved`.

**Fails closed on missing args; mutates persistent state** (saves an
`UncertaintyRecord`). `IsError = !escalated`. Escalation thresholds: 3 for
general, 2 for high.

> **Known issue.** Session existence is not validated (the code comment says
> "simplified - no validation"), and despite the conceptual link,
> `guardrail_check_halt_conditions` never consults this result — the two
> are independent.

### guardrail_check_halt_conditions

Evaluates whether the session requires human escalation.

| Parameter | Type | Required |
|-----------|------|----------|
| `session_token` | string | yes |
| `context` | object | no |
| `task_id` | string | no |

Three checks: (1) three-strikes `ShouldHalt`; (2) `HaltEventStore.GetCriticalPending`
where severity is `critical`; (3) context flags `should_halt` (bool) and
`error_rate` (float64).

**Returns** no-halt: `{"halt":false,"reasons":[],"severity":"none",
"action":"Continue","message":"No halt conditions detected"}`; halt:
`{"halt":true,"reasons":[…],"severity":…,"action":…,"message":"N halt conditions detected"}`.

**Read-only / advisory.** Severity: default `medium`; Check 1 → `high`;
critical event → `critical`; context → `medium`/`high`. Depends on
`guardrail_record_attempt` and `guardrail_record_halt`.

> **Fixed 2026-10-03.** The comparison was inverted (`error_rate < 0.5`
> raised "High error rate", so a *low* error rate halted and a high one did
> not). It now fires above a named threshold.
>
> **Still open (feature work).** The halt-condition design
> (`docs/designs/halt-conditions-design.md`) enumerates ~22 conditions; this
> tool implements 3. Closing that is design work, not a bug fix.

### guardrail_record_halt

Records a halt event.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `session_token` | string | yes | |
| `halt_type` | string | yes | |
| `description` | string | no | `"Unspecified halt condition"` |
| `severity` | string | no | `"medium"` |
| `context` | any | no | Safe-asserted to an object |

**Returns** `{"success":true,"halt_id":"<uuid>","recorded_at":"…","status":"recorded"}`.

**Fails closed; mutates persistent state.** Feeds Check 2 of
`guardrail_check_halt_conditions`.

### guardrail_acknowledge_halt

Acknowledges a halt to resume work.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `session_token` | string | yes | |
| `halt_id` | string | yes | |
| `resolution` | string | no | `pending` |

**Returns** `{"success":true,"halt_id":"…","acknowledged_at":"…","resolution":"…"}`.

**Fails closed; mutates state.**

> **Fixed 2026-10-03.** `halt_id` was parsed with `uuid.UnmarshalBinary`,
> which expects 16 raw bytes, so acknowledging a halt that
> `guardrail_record_halt` had just returned always failed. It is now parsed
> with `uuid.Parse`, and the two tools work as a pair.

---

## Content classification

### guardrail_classify_content

Classifies text against the S1–S15 taxonomy.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `text` | string | yes | |
| `direction` | string | no (`input`/`output`) | `input` |
| `context` | **schema only — never read** | | |

**Returns** `guardrails.ClassificationResult`: `safe`, `overall_risk`,
`categories[]` (`id, name, score, action, reason?`), `backend`,
`latency_ms`. `IsError = result.IsBlocked()`.

**Fails open by design when the engine is absent** — returns
`{"safe":true,"error":"guardrails engine not configured"}`
(`tools_guardrails.go:69`). Read-only.

### guardrail_check_policy

Checks text against a named content policy.

| Parameter | Type | Required |
|-----------|------|----------|
| `text` | string | yes |
| `policy_id` | string | yes |

**Returns** `guardrails.PolicyResult`: `policy_id`, `compliant`,
`violations[]` (`category_id, category_name, score, action, reason`).
`IsError = !compliant`.

**Explicit fail-closed guard** (`tools_guardrails.go:122-124`): if the result
is compliant with no violations *and* the classification blocked, it returns
`compliant:false, error:"unknown policy_id: …"` — an unknown policy can
never pass. Read-only.

### guardrail_install_skills

Installs guardrails skill configs by shelling out to `scripts/setup_agents.py`.

| Parameter | Type | Notes |
|-----------|------|-------|
| `action` | string | `clone`, `install-skill`/`skill`, `list-skills`, `list-platforms`, `install` |
| `target_path` | string | |
| `platforms` | **schema says string; handler expects an array** | |
| `skill` | string | |
| `path` | string | |
| `mode` | string | `copy` or `symlink` |
| `dry_run`, `list_skills`, `list_platforms` | bool | |

**Returns** `{"action":…,"success":bool,"output":"<stdout + stderr>"}`.

**Side-effecting**: resolves `python3`, falling back to `python`, and refuses
if the script is missing. **Not fail-closed** — exits 0 with
`success:true`.

> **Known issues.** The `platforms` type mismatch is a genuine bug
> (`tools_missing.go:299` vs `:408`). The handler also reads `action`
> values that are absent from the published enum.