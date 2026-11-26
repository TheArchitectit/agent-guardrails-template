# TPL-52 evidence

## Task 1.1: optional surface map

The map below is derived from the registered lists and current action tables at
the worktree's starting revision `7adc384d498a6faf19bd85e072266d253e4e7716`.

### Conditional MCP registries

All actions pass through `auth.ClassifyMCPTool` and the scope × role × resource
decision. `mcp:read` is the required scope for read actions, `mcp:validate`
for validate actions, `mcp:mutate` for mutations, and `admin:manage` for admin
actions. A targeted resource must match the caller's explicit resource grant
or `*`.

| Registry / operations | Registration and effective configuration | Policy class and resource input |
|---|---|---|
| Vision: `vision_get_report` | Added only when `VISION_ENABLED=true` initializes `s.visionTools`; the same non-nil tool set mounts the REST group. | Read / `mcp:read`; requires an opaque `review_id` resource. |
| Vision: `vision_check_health` | Same Vision gate. | Read / `mcp:read`; no scoped resource. It probes configured local/fallback providers. |
| Vision: `vision_capture_screenshot`, `vision_analyze_screenshot`, `vision_iterate_review`, `vision_guardrail_check` | Same Vision gate. | Mutate / `mcp:mutate`; iterate targets opaque `review_id`. Capture and guardrail-check can trigger capture; analyze/iterate/guardrail-check can call a provider and persist review state. File `path` is not a resource grant. |
| Webhooks: `list_webhooks`, `get_webhook_deliveries` | Registered when `webhookStore` is non-nil; the current server command installs this store. | Read / `mcp:read`; delivery lookup has `webhook_id`, listing is team-wide. |
| Webhooks: `test_webhook` | Same webhook store gate. | Validate / `mcp:validate`; `webhook_id` identifies the target and dispatches a test event. |
| Webhooks: `configure_webhook`, `delete_webhook`, `rotate_webhook_key` | Same webhook store gate. | Mutate / `mcp:mutate`; a supplied `webhook_id` targets an existing webhook. Rotation already requires durable audit. |
| Budgets: `get_budget_status`, `list_budgets`, `get_budget_history` | Registered when `budgetStore` is non-nil. No `SetBudget` call exists in the current server command, so these are not exposed by that composition. | Read / `mcp:read`; requests are team-wide; history accepts no safe budget resource ID. |
| Budgets: `configure_budget`, `delete_budget` | Same budget store gate. | Mutate / `mcp:mutate`; delete may carry `budget_id`. |
| Lifecycle: `get_agent_state`, `list_agent_sessions` | Registered when `agentStateStore` is non-nil. No `SetAgentStateStore` call exists in the current server command, so these are not exposed by that composition. | Read / `mcp:read`; get targets `session_id`, list is team-wide. |
| Lifecycle: `create_agent_session`, `transition_agent_state` | Same state-store gate. | Mutate / `mcp:mutate`; transition targets `session_id`. |
| Lifecycle: `force_agent_state` | Same state-store gate. | Admin / `admin:manage`; targets `session_id`; already requires durable audit. |

### Optional Vision REST routes

The global Echo `APIKeyAuthWithRegistry` middleware wraps registered routes,
resolves the server-controlled caller, runs `ClassifyREST` and records the
decision before calling a handler. The optional group is mounted at
`/v1/vision` only when `VisionTools()` is non-nil.

| Method and route | Operation | Required scope | Resource rule |
|---|---|---|---|
| `POST /v1/vision/review` | Provider call and new review persistence | `rest:write` | No known review ID exists before creation. |
| `GET /v1/vision/review/:id` | Read report | `rest:read` | `:id` must be an opaque resource granted to the caller. |
| `POST /v1/vision/review/:id/iterate` | Provider call and review mutation | `rest:write` | `:id` must be an opaque resource granted to the caller. |
| `GET /v1/vision/reviews` | List reports | `rest:read` | Collection read; no per-report target. |
| `GET /v1/vision/events` | Server-sent capture/review events | `rest:read` | Collection stream; no per-report target. |
| `POST /v1/vision/capture/trigger` | Trigger capture | `rest:write` | No report ID; global capture action. |
| `GET /v1/vision/health` | Provider health probe | `rest:read` | No scoped resource. |

Unknown methods and paths under this optional prefix are classified as an
unknown action and denied. The authorization catalog already knows how to map
the read/write classes; report resource extraction was the missing boundary.

### Effective configuration

`NewMCPServer` and `NewVisionToolSet` enable Vision only when the process
environment contains the exact value `VISION_ENABLED=true`; an absent value
keeps MCP discovery and the REST group disabled. When enabled, local provider
defaults are `http://localhost:8080/v1` and `nemotron-vision-local`; the
database and screenshot path defaults are `./vision_reviews.db` and
`./screenshots`. The deployment Compose example previously overrode the
process's disabled default with `${VISION_ENABLED:-true}`. It is changed in
this increment to `${VISION_ENABLED:-false}`; explicit opt-in remains
available. Its configured paths use the Compose-provided screenshot and
database locations.

## Task 1.2: bounded implementation

Vision MCP dispatch now goes through `handleToolCall`, the same trusted
principal, action decision, and redacted audit ordering used by other MCP
tools. The four Vision mutations require a durable allow-decision audit before
their handler may call a provider, persist data, or trigger capture. Vision
report reads and iterations require an explicit safe `review_id`; it cannot
fall back to an unscoped decision. REST authorization now maps only the seven
registered Vision method/path pairs and binds report/iteration paths to their
review ID. REST audit action names use fixed route templates; validated report
IDs appear only in the resource field. Enabled Vision REST routes require a
registered principal even when ordinary REST migration mode has no registry;
legacy two-key behavior on other paths is unchanged. Vision REST mutations
require a durable allow-decision audit before handler dispatch.

The Compose example renders Vision disabled without an explicit override.
This check changes only example configuration; it does not start services.

Verification on this increment: with `VISION_ENABLED` absent from the invoking
process, `docker compose -f mcp-server/deploy/docker-compose.example.yml config
--format json` rendered `services.mcp-server.environment.VISION_ENABLED` as
`false`. No Compose services were started. Targeted Go results are recorded
below after completion.

The StreamableHTTP MCP fixture verifies an allowed report read, a denied
mutation with unchanged provider-call and iteration counts, and a successful
policy decision blocked before invocation when the required durable audit
logger is unavailable. The web middleware fixture verifies that a legacy key
cannot reach an optional Vision REST route without a registered principal,
that a registered read can proceed, and that a registered Vision mutation
cannot reach its handler when durable audit is unavailable. These middleware
checks use a synthetic next handler, not the registered Vision route handlers.
The targeted command `go test ./internal/auth ./internal/web ./internal/mcp
-run 'TestClassifyOptionalVisionRESTSurface|TestClassifyVisionMCPReportResource|TestAPIKeyAuth_OptionalVision|TestOptionalVisionToolsUseCommonAuthorization'
-count=1` passed in `golang:1.26.11-bookworm` after the final authorization and
audit-label edits.

## Task 1.3: registered REST route acceptance

`TestOptionalVisionRegisteredRoutesAuthorizeEffectsAndRedactAudit` exercises
the actual `api.NewVisionServer` route registrations through Echo `ServeHTTP`
and the production `APIKeyAuth` middleware. Its local fixture is an isolated
temporary SQLite Vision store, an in-memory implementation of the existing
audit store interface, two synthetic registered principals, a generated
one-pixel PNG, and a stub inference client. No provider endpoint or service is
called.

The writer principal's `POST /v1/vision/review` succeeds, invokes the stub once,
and creates a persisted report. A reader principal can then retrieve that
report. The same reader's `POST /v1/vision/review/:id/iterate` is denied; the
stub call count and stored iteration count remain unchanged. The test inspects
the real audit records: route names are fixed templates, review IDs appear
only in the resource field, and the synthetic local image path, bearer
credentials, and request payload marker are absent.

`TestOptionalVisionDefaultProfileDoesNotMountRoutes` unsets `VISION_ENABLED`,
constructs the real MCP server with a local SQLite DB wrapper, confirms
`VisionTools()` is nil, and applies the same `VisionTools()!=nil` conditional
used by `cmd/server` to mount the REST group. The Echo route table contains no
Vision routes. Separately, `docker compose ... config --format json` rendered
the example profile's server environment as `VISION_ENABLED=false` when the
invoking process variable was unset. Compose services were not started.

Transport and profile: real Echo request dispatch with global auth middleware,
local deployment profile, base source revision
`7adc384d498a6faf19bd85e072266d253e4e7716` plus the carried staged patch
`2fb828e1b457dc14f753c79b31a031cd8794f563`. The test does not exercise the
SSE events handler, capture trigger/watcher, every collection/health operation,
the full `cmd/server` process, or a real inference provider. The audit store is
an in-memory fixture; it validates the structured records supplied by the real
logger, not production database durability.

The REST test command
`go test ./internal/web -run 'TestOptionalVisionRegisteredRoutesAuthorizeEffectsAndRedactAudit|TestOptionalVisionDefaultProfileDoesNotMountRoutes' -count=1`
passed in `golang:1.26.11-bookworm`.

The runnable regression entry point
The [regression entry point](../../../tests/regression/test_optional_vision_authorization.sh) runs the existing auth, MCP, and REST fixtures.

`bash tests/regression/test_optional_vision_authorization.sh`
passed in the same Go image after `bash -n`; it invokes the named auth, web,
and MCP Go tests above without duplicating their fixtures or assertions.

## Remaining acceptance

Task 1.3 covers the registered review creation/read/iteration routes and the
default-disabled registration condition described above. The additional
unexercised Vision operations and production provider/audit-store integrations
remain outside this fixture. Pinned strict OpenSpec validation passed in the
earlier increment; broader repository gates and independent scenario review
remain outstanding, so task 2.1 stays open.
