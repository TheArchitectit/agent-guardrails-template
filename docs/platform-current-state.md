# Platform Current-State Baseline

**Audit date:** 2026-10-03  
**Purpose:** Source-verified picture of what this repository currently is,
what runs, and what is still only a library, proposal, or example. This is a
baseline for the system roadmap, not a claim that the target platform is
complete.

## 1. Executive summary

This repository is a mixed platform today: a Go MCP/REST service, Postgres
and Redis dependencies, a web UI, policy and skill material, several client
adapters, a Python team manager, a Go CLI wrapper, and language examples. It
is **not yet one coherent, fully enforced MCP guardrail platform**.

The important distinctions are:

- The service starts and exposes a real MCP endpoint, REST API, and web UI.
- The MCP tool registry has 37 always-registered core tools and up to 21
  conditional tools. Published schema/handler agreement is not complete.
- Some security checks are active in request paths, but several guardrail
  subsystems exist only as libraries or are enabled only by optional runtime
  configuration.
- MCP and web REST use different authentication paths. MCP is bearer-key
  gated; the web middleware intentionally exposes a set of read routes and
  static UI routes. Team MCP tools have no role-based authorization.
- The root Docker Compose file binds host ports to localhost but has weak
  default DB/Redis passwords and disables DB SSL/Redis TLS. The application
  itself binds to `0.0.0.0` inside the container.
- Integrations, scripts, examples, and docs have compatibility drift. A
  client/tool name or example README is not proof that the server supports
  that contract.
- CI is functional for the current tested paths, but does not prove all
  documented controls or all ecosystem clients. Documentation checks are
  PR-only and the link checker has known false positives.

## 2. Repository components

| Component | Location | Current role | Confidence/status |
|-----------|----------|---------------|-------------------|
| Go MCP + REST server | `mcp-server/` | Main server binary, MCP tool/resource registration, REST API, persistence adapters | Active; separate module |
| Guardrails library | `mcp-server/internal/guardrails/` | Injection, content filter, sandbox, provenance, multi-agent, compliance components | Mixed: some wired, several library-only |
| Validation engine | `mcp-server/internal/validation/` | Rule-based bash/git/file checks, DB/Redis rule access | Active in MCP validation tools |
| Web app served by Go | `mcp-server/internal/web/`, `/app/web` | Echo routes, middleware, API and embedded/static UI | Active; distinguish from `site/` |
| React documentation/marketing site | `site/` | Separate Vite/React app and generated data | Separate build; deployment into `/app/web` not established by source audit |
| Pi extension | `pi-extension/` | Client-side tools/hooks for Pi | Separate client contract; not a 1:1 MCP tool mirror |
| IDE extensions/configs | `ide/`, `.cursor/`, `.opencode/`, `.windsurfrules`, `integrations/` | Host-specific configuration and adapters | Mixed maturity; compatibility needs contract tests |
| Python team manager | `scripts/team_manager.py` | Team/project CLI and file-backed management | Real implementation; CLI auth context is not wired, mutating operations require test mode |
| Go team CLI | `cmd/team-cli/` | Go frontend that shells out to the Python manager | Not an independent Go implementation; audit found subcommand/flag drift |
| Policy packs | `policy-packs/`, `.guardrails/` | Four Laws and regression/prevention data | Files exist; runtime consumption differs by subsystem |
| Examples | `examples/` | Language-specific demonstrations | Mixed runnable/illustrative; verify each command before advertising support |
| Docs | `docs/`, root Markdown | Specs, standards, procedures, integrations, history | Large mixed corpus: normative, aspirational, historical, and stale content |

## 3. Runtime architecture and data flow

### 3.1 Startup

`mcp-server/cmd/server/main.go` loads and validates environment configuration,
initializes audit logging, connects PostgreSQL and Redis, constructs the web
server and validation/MCP engines, and starts separate web and MCP listeners.
The process binds both listeners to `0.0.0.0` inside its runtime/container.
Startup is fail-fast for Postgres/Redis connection errors.

Database migrations are present, but automatic migration execution was not
found in the inspected startup path. Treat migration application as an
external deployment step until verified otherwise.

### 3.2 MCP path

The MCP endpoint is stateless Streamable HTTP at `POST /mcp`, wrapped in
bearer authentication using `MCP_API_KEY`. Registered tool schemas are
returned by `toolList()`; most core guardrail tools dispatch through
`handleToolCall`, while vision uses a first-chance sub-router. Resources are
registered separately.

The registry defines 37 always-registered core tools and code paths for up
to 21 conditional tools (vision, webhooks, budgets, lifecycle). This is a
**registry maximum, not the live startup inventory**. `NewMCPServer` builds
and registers the tool list before `main.go` calls `SetWebhookStore`, so
webhook tools are not included in the normal startup registration even
though the dispatcher is later configured. Vision is feature-gated. Budget
and lifecycle setters are not called in the inspected `main.go` startup
path. The live inventory must be measured after startup rather than copied
from generated static data.

### 3.3 Web/REST/UI path

The separate Echo web listener provides health routes, REST APIs, IDE
validation routes, OpenAPI/docs endpoints, and optionally serves the UI when
`WEB_ENABLED` is true. The SPA fallback is `/web/*`; static files are mounted
at both `/web` and `/` from the container path `/app/web`
(`internal/web/server.go:212-236`). The root Compose file maps host ports
8080/8081 to MCP/web ports but binds the host side to `127.0.0.1`.

Web API authentication is not equivalent to “all API routes require a key”:
health/docs/static paths, `/version`, and a set of read endpoints are
intentionally public. Deployment behind another proxy or with different port
binding can change exposure. The React `site/` project is a separate build;
no verified build/deploy pipeline connecting it to `/app/web` was established
here.

### 3.4 State and persistence

- Postgres stores rules, failures, webhook configuration/deliveries, task
  attempts, halt events, uncertainty, and agent lifecycle state through
  dedicated stores. Although startup constructs an audit logger, it does not
  attach a Postgres audit store in the inspected path; do not assume all
  `slog`/audit events persist to Postgres.
- Redis supports cache/rate-limit functions.
- Team projects are a separate file-backed subsystem using
  `.teams/<project>.json`; they do not share the MCP server’s Postgres team
  state as a single source of truth.
- Guardrail sessions are process-local in-memory state with an eight-hour
  expiry; a process restart loses them. They are distinct from Postgres
  agent-lifecycle sessions.
- Policy-pack JSON files and prevention rules exist, but their runtime
  consumers must be checked individually; file presence does not prove use.

## 4. Capability reality

| Capability | Present in source | Exposed/called in runtime | Current caveat |
|------------|------------------|---------------------------|----------------|
| Rule-based bash/git/file validation | Yes | Yes, via validation engine/tools | Schema alignment and caller enforcement still require work |
| Content classification (S1–S15) | Yes | Two MCP tools; engine only initialized when `OLLAMA_URL` is set | No URL means not-configured; classification tool has a fail-open response in that state |
| Named content policy | Yes | MCP tool exists | Configured policies are now loaded by `NewEngine`; real deployment config/loading and evaluation data remain gaps |
| Prompt injection detection | Yes, library | No dedicated MCP tool; `Engine.Evaluate` has no production caller | Engine uses a no-op classifier by default; not active request-path protection |
| Runtime sandbox | Yes, L0–L2 library | No MCP execute/config tool; `ExecuteSandbox` has no production caller | Do not claim user commands are sandboxed by MCP service |
| Provenance / indirect injection | Yes, library | Tracker only reached via the non-production `Engine.Evaluate` path | In-memory cache; file-read attestation tools are a separate feature |
| Multi-agent safety chains | Yes, library/tests | No registered chain tool or production caller | Config loader and runtime absent; game/agent transport not present |
| Compliance mapping | Data and library exist | No MCP tool/request-path caller | `CheckRequirement` reads static `full` status; evidence collection is simulated; not compliance evidence |
| Halt/attempt family | Several MCP tools and stores | Partially reachable | Halt design enumerates far more conditions than implemented; `record_halt`/ack and error-rate defects were fixed in recent commits, but broad condition coverage remains open |
| Vision | Yes | Conditional MCP tools and optional REST routes | Requires `VISION_ENABLED` and initialized vision services; separate from general code safety |
| Webhooks | Yes | Conditional tools and dispatcher | Destination validation is configuration-time; DNS rebinding/delivery-time policy remains residual risk |
| Budgets | Yes | Conditional tools | Usage enforcement/recording is not demonstrated as wired; verify before claiming budgets block calls |
| Agent lifecycle | Yes | Conditional tools if store set | Separate store/session namespace; force-state now has explicit confirmation, but no role authorization |

## 5. Configuration and deployment

### 5.1 Required services and ports

Root `docker-compose.yml` defines Redis 7, Postgres 16, and the server. It maps
MCP/web container ports to localhost by default. `.env.example` documents
MCP/IDE API keys, JWT, DB/Redis settings, and optional provider settings.
The Go server’s primary configuration is loaded by `internal/config` from
environment variables; do not assume every `.env.example` flag is actually
consumed.

### 5.2 Material deployment risks

- Root Compose defaults Redis and Postgres passwords to `changeme`.
- Root Compose defaults `PRODUCTION_MODE=false`, DB SSL to `disable`, and
  Redis TLS to false.
- Container listeners bind `0.0.0.0`; localhost-only exposure comes from the
  checked-in Compose host-port binding, not from the process itself.
- MCP API keys are required in the root Compose interpolation, but a copied
  config or alternate deployment may have different protections.
- The Docker image uses a non-root/distroless runtime. Hardening differs by
  deployment manifest: root `docker-compose.yml` does not show read-only
  filesystem, capability drops, or resource limits, while the separate
  `mcp-server/deploy/podman-compose.yml` and Kubernetes manifest include
  additional restrictions. Do not transfer one manifest's controls to
  another.
- Webhook SSRF filtering is incomplete against rebinding and delivery-time
  changes.
- Database migrations are not visibly run automatically at server startup.

Production deployment is therefore not “secure by default” solely because
the Compose host ports are local. Operators must set strong secrets, choose
TLS/CORS/port exposure deliberately, apply migrations, and verify mandatory
features are enabled.

## 6. Ecosystem and compatibility

The repository contains multiple client surfaces, not yet a single proven
compatibility matrix. Verified examples from the audit:

- **Pi bridge is incompatible with the current MCP server.** The bridge uses
  SSE at `/mcp/v1/sse` (`pi-extension/mcp-bridge/mcp-client.ts:24-56`);
  the server exposes stateless Streamable HTTP at `/mcp`
  (`mcp-server/internal/mcp/server.go:355-363`). The standalone Pi extension
  registers local tools, which is not proof of server interoperability.
- **IDE selection payloads mismatch the handler.** VS Code, Neovim and Vim
  send `code`; the handler expects `selection`
  (`ide/vscode-extension/src/utils/client.ts:45-47`, Neovim
  `lua/guardrail/validation.lua:67-80`, Vim `autoload/guardrail.vim:161-169`,
  versus `mcp-server/internal/web/handlers_ide.go:78-96`). VS Code defaults
  to port 8095; the documented server ports are 8080/8081.
- **The Go `team-cli` is a wrapper, not the canonical Go implementation.**
  It shells out to Python (`cmd/team-cli/main.go:84-110,113-148`). The audit
  found unsupported flags on reassign/export, an advertised but unregistered
  `agent-map` command, and `phase-gate-check` calls that do not exist in the
  Python argparse contract.
- **Pi tools are not a 1:1 mirror of MCP tools.** Pi registers local names and
  behaviors; only a subset maps to server names. Integrations must be marked
  advisory/experimental unless an invocation contract test passes.
- **The site inventory is generated from declarations, not runtime discovery.**
  `site/scripts/gen-data.mjs` scans source; runtime conditionally adds tools,
  so the displayed count can differ from a live server.
- **Examples are not uniformly executable.** Some READMEs name absent
  language directories or pseudocode. Treat examples as illustrative unless
  their stated command is run in CI.

Until Phase 5 of `docs/specs/09-system-roadmap-and-phase-gates.md` passes,
call an integration supported only where a versioned contract test proves the
client/server pair; label all others experimental or illustrative.

## 7. Verification and operations

### CI workflows currently present

- `team-validation.yml`: fixture-based team validation/status, Python unit
  tests with coverage upload, `go test ./... -v -race`, Go build, and team-cli
  build. It does not start Postgres/Redis, run migrations, launch Docker or
  Podman, or test MCP transport end-to-end.
- `secret-validation.yml`: Gitleaks history scan; `.env` files fail, while
  credential-file and hard-coded-secret scans are warning-only.
- `regression-guard.yml`: invokes regression commands with `|| true`, so those
  command results cannot fail the workflow; only the later critical-finding
  condition can block. Missing regression tests are warning-only. Commenting
  on a PR also depends on permissions not explicitly declared by the workflow.
- `guardrails-lint.yml`: forbidden-file changes fail; scope, commit-format and
  attribution checks are advisory; runs only for PRs to `main`/`develop`.
- `documentation-check.yml`: PR-only; >500-line files fail, while required
  section checks warn. Its Markdown link extractor has false positives.

No active workflow builds/tests the `site/` app, tests `pi-extension`, runs
`validate-policy-packs.mjs`, runs `scripts/test-guardrails-cli.sh`, validates
examples across languages, or performs deployment smoke/rollback tests. The
GitLab/Jenkins files are copy templates, not active GitHub workflows; they
contain stale assumptions such as root `requirements.txt` and Go 1.21 versus
the server module's Go 1.25.5.

All checked-in workflows currently target `ubuntu-latest`; none targets a
UCS03/fleet runner label. The private `infra-info` runbook identifies this
repository as public and recommends hosted runners for public repos unless
there is a specific reason to consume fleet capacity. It does not identify
this repo as a configured fleet target. A live runner-registration query was
blocked by GitHub API rate limiting, so a registered-but-unused runner cannot
be ruled out here. Do not switch workflows to UCS03 without an explicit owner
decision and runner-registration verification.

CI green means those particular jobs passed. It does not prove unwired
subsystems enforce anything, the full client ecosystem interoperates, a
production deployment is secure, or rollback works. The custom migration
runner has no checksum validation or down path, and neither migration path
is exercised in CI. The rollback guide's automatic rollback triggers and
example `/mcp/v1/health` route are not implemented/current routes.

### Health and operational signals

The service exposes liveness, readiness and Prometheus routes. Readiness
checks Postgres and Redis, but the binary `--health-check` calls only
liveness; the example Compose healthcheck can therefore report healthy while
dependencies are unavailable. Kubernetes probes readiness directly.

Prometheus collectors exist, but health-check/SLO/error-budget recording
functions have no production callers. Ingest status currently returns a
placeholder `completed` with zero counts. Treat these values as interface
plumbing, not operational evidence, until backing work and request-path tests
are added.

### Observability and reliability

The Go service uses structured `slog`, Prometheus metrics, health/readiness
routes, Redis rate limiting, a circuit breaker, retry helpers, request
timeouts, and graceful shutdown. Coverage is uneven: some config fields are
dead or have no consumer; health response shapes differ from generic standard
docs; structured guardrail-decision evidence is not unified; not every
component’s enabled/ready/reachable/exercised state is surfaced.

### Tests and known environment constraints

The main Go module is `mcp-server/`; `cmd/team-cli/` is a separate Go module;
`examples/go/` is a separate example module. Python team tests run on Linux
because the manager imports POSIX `fcntl`; two MCP tests also fail on the
Windows development host for known platform reasons (symlink privilege and
Python `fcntl`), while Linux CI is the authoritative result for those cases.

## 8. Documentation quality and trust

Documentation currently mixes normative policy, proposed OpenSpecs, code
references, historical records, generated/reference inventories, onboarding
examples, and aspirational standards. The six guardrail-gap specs and Specs
07–09 now explicitly separate proposal from implementation status.

Some security audits are stale relative to the current transport: for example,
`docs/security/security-audit-code.md` describes `/mcp/v1/sse`, but the server
uses stateless Streamable HTTP at `/mcp`. Treat audit reports as dated findings
to re-verify, not as evidence of the current route or a current control.

Remaining work is to:

1. maintain this baseline as code and CI evolve;
2. remove or correct README and deployment-guide claims that imply unwired
   protections are active;
3. label standards that are examples/stubs rather than enforced repo
   requirements;
4. make the doc checker accurate before making it a blocking release gate;
5. retire fabricated tool names, stale CLI flags, and non-runnable examples;
6. replace generic reboot acceptance placeholders with executable checks or
   mark the items deferred/retired.

Do not infer platform capability from Markdown. For each security claim,
require a source path, configuration path, and request-path test.

## 9. Source map

| Area | Primary source |
|------|----------------|
| Startup/config | `mcp-server/cmd/server/main.go`, `mcp-server/internal/config/` |
| MCP registry/dispatch | `mcp-server/internal/mcp/tools_registry.go`, `server.go`, `tools_core_list*.go` |
| Guardrails engine | `mcp-server/internal/guardrails/` |
| Web API/UI | `mcp-server/internal/web/`, `mcp-server/app/web/`, `site/` |
| Persistence | `mcp-server/internal/database/`, `internal/cache/`, `internal/team/` |
| Client extensions | `pi-extension/`, `ide/`, `integrations/`, `.cursor/`, `.opencode/` |
| Team CLI | `scripts/team_manager.py`, `cmd/team-cli/` |
| Deployment | `docker-compose.yml`, `mcp-server/deploy/`, `.env.example` |
| CI | `.github/workflows/` |
| Specs and roadmap | `docs/specs/`, especially `09-system-roadmap-and-phase-gates.md` |
