# Conditionally Registered and Integration Tools

21 tools registered **only when their backing store is initialised**. A
default deployment exposes the 37 core tools; a fully wired deployment
exposes 58. `toolList()` in `tools_registry.go` governs the gating.

| Group | Count | Registered when | Dispatch route |
|-------|-------|-----------------|----------------|
| Vision | 6 | `s.visionTools != nil` **and** `VISION_ENABLED=true` | Vision sub-router |
| Webhooks | 5 | `s.webhookStore != nil` | Main switch, `server.go:239-248` |
| Budgets | 5 | `s.budgetStore != nil` | Main switch, `server.go:250-259` |
| Agent lifecycle | 5 | `s.agentStateStore != nil` | Main switch, `server.go:261-270` |

All 21 are reachable — none are registered-but-dead.

> **Vision routing differs from everything else.** `setupHandlers`
> (`server.go:147-152`) tries `s.visionTools.dispatch` *first* on every call
> and only falls through to the main switch if that returns an error. The
> main switch's `default:` is therefore never the path for a vision tool.

---

## Vision tools

Gate on both `s.visionTools != nil` and `VISION_ENABLED == "true"`
(`vision_tools.go:26-29`).

### vision_capture_screenshot

Triggers an immediate screenshot capture from the running game.

**Parameters:** none. **Returns** a plain-text trigger message.
**Mutates state** — the watcher writes a screenshot file.

### vision_analyze_screenshot

Runs a vision review over a screenshot.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `path` | string | **handler only** | Optional in the schema, hard-fails when empty |

**Returns** JSON-marshalled `vision.Report`: `review`, `iterations`,
`findings[]`. **Mutates state** — persists reviews and iterations in
SQLite (`vision_reviews.db`).

### vision_iterate_review

Runs another review round against an existing review.

| Parameter | Type | Required |
|-----------|------|----------|
| `review_id` | string | **handler only** (no `Required` array in the schema) |

**Returns** `Report` JSON. **Mutates state.**

### vision_get_report

Fetches a stored report.

| Parameter | Type | Required |
|-----------|------|----------|
| `review_id` | string | schema/handler mismatch |

**Returns** `Report` JSON. Read-only.

### vision_check_health

Reports health of the vision backends.

**Parameters:** none. **Returns** `HealthStatus[]`: `backend`, `healthy`,
`model_loaded`, `error`. Read-only; performs a network read to
`LOCAL_LLAMA_URL`.

### vision_guardrail_check

Capture → review → validate findings against 3D rules.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `path` | string | no | Omitted → triggers capture, then sleeps 2s |

**Returns** `{"report": Report, "validation": {passed, issues[], rule_set,
checked_at}}`. **Mutates state.**

---

## Webhook tools

> **Security: unvalidated outbound URL (SSRF).** `configure_webhook.url` is
> stored verbatim. `url.Parse` and `ParseRequestURI` appear **nowhere** in the
> server — no scheme check, no host check, no IP-range filtering. The schema
> text "must be HTTPS in production" is documentation only. Two paths then
> POST to the stored URL: the domain-event dispatcher and `test_webhook`,
> which reaches it in a single tool call. The HTTP client has a 10s timeout
> but **no redirect policy**, so loopback, RFC1918 and link-local targets are
> reachable. Signed payloads, a circuit breaker and a 4KB body cap limit the
> blast radius but do not close the hole.

### configure_webhook

Creates or updates a webhook.

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `team_id` | number | yes | |
| `url` | string | yes | **Not validated** |
| `events` | array of string | yes | |
| `secret_hmac` | string | yes | |
| `enabled` | bool | optional | **Omitting it creates a disabled webhook** |
| `webhook_id` | string | no | Presence selects the update path |

**Returns** `{success, webhook_id, message}`. **Mutates** `webhook_configs`.

> **Known issue.** The schema documents `enabled` as defaulting to `true`,
> but the handler does a bare `args["enabled"].(bool)` with no default, so an
> omitted `enabled` silently yields `false` (`tools_notifications.go:116`).

### test_webhook

Sends a real test POST to the configured URL.

| Parameter | Type | Required |
|-----------|------|----------|
| `webhook_id` | string | yes |

**Returns** `{success, status_code, response_body, error_message,
delivery_id}`. **Mutates** — records a delivery row.

### list_webhooks

Lists webhooks for a team.

| Parameter | Type | Required |
|-----------|------|----------|
| `team_id` | number | yes |

**Returns** `{webhooks:[…], count}`, each with `id, team_id, url, events,
enabled, created_at, updated_at`. `secret_hmac` is deliberately masked.
Read-only.

### delete_webhook

| Parameter | Type | Required |
|-----------|------|----------|
| `webhook_id` | string | yes |

**Returns** `{success, message}`. **Mutates**.

### get_webhook_deliveries

| Parameter | Type | Required | Notes |
|-----------|------|----------|-------|
| `webhook_id` | string | yes | |
| `limit` | number | no | Handler accepts **float64 only**, default 50 |

**Returns** `{deliveries:[…], count}`. Read-only.

---

## Budget tools

Units are **tokens** and **US cents per 1M tokens**
(`budget/estimator.go:3-19`).

> **Budgets are advisory reporting only — nothing enforces them.**
> `Governor.CheckAndRecord` (`budget/governor.go:31`) holds the enforcement
> logic but has **zero callers** in the repository, and no tool records usage
> at all (`budgetStore.Record` is never invoked; there is no `record` tool).
> Reported usage therefore stays zero, and a budget breach cannot block
> anything. Separately, an unrecognised `model_name` estimates at **0**
> (`budget/estimator.go:23-27`), so a typo under-reports cost rather than
> failing loudly.

### configure_budget

Sets per team + model limits.

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `team_id` | number | yes | |
| `model_name` | string | yes | |
| `max_tokens` | number | no | |
| `max_cost_cents` | number | no | |
| `period` | string | no | `daily` (also `weekly`, `monthly`) |
| `alert_threshold` | number | no | `0.8` |

**Returns** `{success, budget_id, message, team_id, model_name}`.
**Mutates** `budget_configs`.

### get_budget_status

| Parameter | Type | Required |
|-----------|------|----------|
| `team_id` | number | yes |
| `model_name` | string | yes |

**Returns** `budget.BudgetStatus`: `team_id, model_name, period, tokens_used,
tokens_max, cost_used_cents, cost_max_cents, percent_used, within_budget,
reset_at`.

> **Known issue.** Registration gates on `s.budgetStore != nil`, but the
> handler additionally requires `s.budgetGovernor` (`tools_budget.go:206`).
> A server with a store but no governor registers this tool and then errors
> on every call.

### list_budgets

| Parameter | Type | Required |
|-----------|------|----------|
| `team_id` | number | yes |

**Returns** `{budgets:[…], count}`. Read-only.

### get_budget_history

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `team_id` | number | yes | |
| `days` | number | no | 7 (hard-capped at 200 entries) |

**Returns** `{entries:[…], count, total_tokens, total_cost_cents,
period_days}`. Read-only.

### delete_budget

| Parameter | Type | Required |
|-----------|------|----------|
| `budget_id` | string | yes |

**Returns** `{success, message}`. **Mutates**.

---

## Agent lifecycle tools

All five use `s.agentStateStore` (Postgres `agent_sessions` /
`agent_state_transitions`) and **never touch the in-memory `s.sessions` map**.

> **The two session systems are disjoint.** `models.AgentSession` is a
> distinct type from `models.Session` with independent UUIDs. A lifecycle
> `session_id` will **not** satisfy the `s.sessions[sessionToken]` checks used
> by the extended guardrail tools, and `guardrail_init_session` ids are not
> interchangeable with `create_agent_session` ids.

### create_agent_session

| Parameter | Type | Required |
|-----------|------|----------|
| `team_id` | number | yes |
| `agent_name` | string | yes |
| `project_slug` | string | no |

**Returns** `{success, session_id, current_state, agent_name, message}`.
Created in `idle`. **Mutates.**

### transition_agent_state

| Parameter | Type | Required | Default |
|-----------|------|----------|---------|
| `session_id` | string | yes | |
| `to_state` | string | yes | |
| `reason` | string | no | |
| `triggered_by` | string | no | `agent` |

**Returns** `{success, session_id, current_state, previous_state, message}`.
Validated against `models.ValidTransitions`. **Mutates.**

### get_agent_state

| Parameter | Type | Required |
|-----------|------|----------|
| `session_id` | string | yes |

**Returns** `{session: AgentSession, transitions: [StateTransition]}`. A
failure fetching transitions is swallowed and returns `null` rather than an
error. Read-only.

### list_agent_sessions

| Parameter | Type | Required |
|-----------|------|----------|
| `team_id` | number | yes |

**Returns** `{sessions:[AgentSession], count}`. Read-only.

### force_agent_state

Admin override that bypasses transition validation.

| Parameter | Type | Required |
|-----------|------|----------|
| `session_id` | string | yes |
| `to_state` | string | yes |
| `reason` | string | yes |

**Returns** `{success, session_id, new_state, message, warning:"Transition
validation was bypassed"}`. Records `triggered_by="admin_override"`.

> **Security: no authorization whatsoever.** The handler performs no auth
> check — "admin" is enforced only by the client choosing to pass a `reason`
> string (`tools_lifecycle.go:230-254`). Any authenticated bearer caller can
> force any agent into any state.