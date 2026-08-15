# Platform Review — June 14, 2026

**Reviewer:** Claude (7-agent automated workflow)
**Version:** v2.6.0
**Scope:** Total platform review + new feature proposals

---

## Table of Contents

1. [Architecture Summary](#1-architecture-summary)
2. [Critical Issues (P0)](#2-critical-issues-p0)
3. [Security Concerns](#3-security-concerns)
4. [Test Coverage Gaps](#4-test-coverage-gaps)
5. [Code Quality Issues](#5-code-quality-issues)
6. [Performance Bottlenecks](#6-performance-bottlenecks)
7. [Accessibility Gaps](#7-accessibility-gaps)
8. [Documentation Gaps](#8-documentation-gaps)
9. [Proposed New Features](#9-proposed-new-features)
10. [Implementation Roadmap](#10-implementation-roadmap)
11. [Dependencies Between Features](#11-dependencies-between-features)
12. [Risk Assessment](#12-risk-assessment)

---

## 1. Architecture Summary

| Metric | Value |
|--------|-------|
| Total files (excl. .git/node_modules) | 710 |
| Primary language | Go (119 .go files) |
| Module | `github.com/thearchitectit/guardrail-mcp` |
| Documentation files | ~100 .md files |
| SQL migrations | 33 files |
| IDE plugins | 4 (VS Code, JetBrains, Neovim, Vim) |

### Key Modules

| Module | Path | Files | Purpose |
|--------|------|-------|---------|
| mcp-server | `mcp-server/` | ~210 | Core backend: Go MCP server, validation, DB, web UI |
| docs | `docs/` | ~100 | Architecture, standards, security audits, workflows |
| examples | `examples/` | ~95 | Multi-language reference implementations |
| ide | `ide/` | ~50 | IDE plugins |
| scripts | `scripts/` | 29 | Python utilities |
| .claude | `.claude/` | 13 | Agent skills and hooks |

### Internal Go Packages

| Package | Files | Purpose |
|---------|-------|---------|
| `internal/models` | 20 | Domain entities |
| `internal/mcp` | 17 | MCP protocol handlers |
| `internal/database` | 14 | PostgreSQL/SQLite stores |
| `internal/vision` | 9 | AI vision pipeline |
| `internal/ingest` | 8 | Rule parser |
| `internal/team` | 7 | Multi-team management |
| `internal/validation` | 6 | Rule validation engine |
| `internal/web` | 4 | Echo HTTP server |
| `internal/guardrails` | 4 | Vertical slices (bash, git, fileedit) |
| `internal/circuitbreaker` | 3 | Circuit breaker pattern |
| `internal/adapters` | 3 | Concrete implementations |
| `internal/domain` | 2 | Core domain (CQRS, ports) |
| `internal/config` | 2 | Env-based config |
| `internal/security` | 2 | Secrets scanner |
| `internal/cache` | 1 | Redis client |
| `internal/audit` | 1 | Audit logger |
| `internal/metrics` | 1 | Prometheus metrics |

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Language | Go 1.23+ (toolchain 1.24.13) |
| HTTP framework | Echo v4 |
| MCP protocol | mark3labs/mcp-go v0.4.0 |
| Database | PostgreSQL (pgx/v5), SQLite (mattn/go-sqlite3) |
| Cache | Redis (go-redis/v8) |
| Metrics | Prometheus (client_golang) |
| Circuit breaker | sony/gobreaker |
| Config | caarlos0/env, YAML |
| Logging | slog (structured JSON) |
| Vision AI | Anthropic API, OpenAI API, local LLaMa |
| CI/CD | GitHub Actions (5 workflows), GitLab CI |

### Design Patterns

1. **Clean Architecture / Hexagonal** — Domain ports in `internal/domain/`, adapters in `internal/adapters/`
2. **CQRS** — Command/query separation in `internal/domain/cqrs.go`
3. **Vertical Slices** — Guardrail evaluation split into bash, git, fileedit
4. **Functional Options** — `WithBash()`, `WithGit()`, `WithFileEdit()`
5. **Event Bus (Pub/Sub)** — In-memory `DefaultEventBus` with handler registration
6. **Repository Pattern** — Typed store methods behind interfaces
7. **Circuit Breaker** — Resilience wrapper for external calls

---

## 2. Critical Issues (P0)

### 2.1 Build is Broken

**File:** `mcp-server/internal/domain/cqrs.go`

Unused imports at lines 5-6 (`log/slog`, `sync`) cause compilation failure across 8 downstream packages: cmd/server, internal/adapters, internal/domain, internal/guardrails/*, internal/mcp.

### 2.2 Duplicate Function Name

**File:** `mcp-server/internal/mcp/tools_extended.go`

Two different `buildToolResult` functions exist in the same package with different signatures:
- `internal/mcp/server.go:818` — `(data interface{}, isJson bool)`
- `internal/mcp/tools_extended.go:378` — `(result interface{}, isError bool)`

This causes a compilation error.

### 2.3 Dead Halt Condition

**File:** `mcp-server/internal/mcp/tools_extended.go:788`

```go
if len(criticalEvents) < 0  // Always false — slice length is never negative
```

Should be `> 0`. This means critical halt events are **never evaluated**, making the halt mechanism completely non-functional.

---

## 3. Security Concerns

### 3.1 ReDoS Exposure

**File:** `mcp-server/internal/web/handlers.go:1312`

`validateContentAgainstRules` calls `regexp.Compile(rule.Pattern)` directly on user-defined patterns without the safe regex wrapper (`internal/validation/safe_regex.go`). The validation engine uses safe regex, but the web handler bypasses it.

### 3.2 Unauthenticated POST Endpoints

**File:** `mcp-server/internal/web/middleware.go:88-92`

The following POST endpoints skip authentication:
- `POST /api/ingest`
- `POST /api/ingest/sync`
- `POST /api/updates/check`

An unauthenticated attacker can trigger resource exhaustion via document ingestion.

### 3.3 SSRF via os.ReadFile (3 instances)

| Location | Line | Risk |
|----------|------|------|
| `internal/mcp/tools_extended.go` | 267 | `handleCheckTestProdSeparation` reads arbitrary files |
| `internal/mcp/tools_extended.go` | 2116 | `handleScanCommitPayload` reads arbitrary files |
| `internal/mcp/tools_extended.go` | 2194 | `handleDetectMergeConflicts` reads arbitrary files |

All accept user-supplied file paths with no sandboxing.

### 3.4 Insufficient Session Token Entropy

**File:** `mcp-server/internal/mcp/server.go:841`

```go
token := make([]byte, 8)  // Only 64 bits — should be 128+ bits
```

### 3.5 File Extension Bypass Vector

**File:** `mcp-server/internal/web/middleware.go:56-68`

Middleware grants public access to paths ending in `.js`, `.css`, `.html`, `.json`, etc. An attacker could access files like `/api/rules/../../secret.json` if path normalization is insufficient.

### 3.6 Incomplete Secrets Scanner

**File:** `mcp-server/internal/security/secrets_scanner.go`

Current coverage: AWS keys, private keys, GitHub tokens, Slack tokens, generic passwords.

**Missing patterns:** GCP service account keys, Azure AD tokens, Stripe keys, npm tokens, PyPI tokens, Hugging Face tokens, database connection strings with embedded passwords.

---

## 4. Test Coverage Gaps

**17 test files** cover **92 non-test Go source files** (18.5% file-level coverage).

### Untested Critical Packages

| Package | Files | Risk |
|---------|-------|------|
| `internal/database/` | 14 | All store files untested |
| `internal/web/` | 4 | All handlers untested |
| `internal/vision/` | 9 | Entire AI pipeline untested |
| `internal/audit/` | 1 | Logger untested (ring buffer, panic recovery) |
| `internal/team/` | 7 | All team management untested |
| `internal/cache/` | 1 | Redis client untested |
| `internal/metrics/` | 1 | Prometheus metrics untested |
| `internal/middleware/` | 1 | HTTP logging middleware untested |

### Specific Gaps

- No tests for `handleToolCall` dispatch in `internal/mcp/server.go` (~30 tool handlers)
- CQRS handlers (`CreateRuleHandler`, `UpdateRuleHandler`) have no tests
- `internal/cache/redis.go` `DistributedRateLimiter.Allow()` untested
- `validateContentAgainstRules` in `web/handlers.go` untested (and has ReDoS vulnerability)

---

## 5. Code Quality Issues

### 5.1 Error Swallowing

| Location | Issue |
|----------|-------|
| `internal/audit/logger.go:125-126` | `reqID.(string)` will panic if not a string — needs ok check |
| `internal/web/handlers.go:659` | `limit, _ := strconv.Atoi(...)` — error ignored |
| `internal/web/handlers.go:799` | `_ = c.Bind(&req)` — bind error ignored |
| `internal/mcp/server.go:841-842` | `rand.Read(token)` error ignored |

### 5.2 Audit Buffer Overflow Invisible

**File:** `internal/audit/logger.go:131-132`

When the audit buffer is full, events are dropped with only stderr log. No metric or counter for dropped events — invisible in production monitoring.

### 5.3 Infinite Restart Loop

**File:** `internal/audit/logger.go:138-143`

If the `process()` goroutine panics, it recursively spawns via `go l.process()`. A consistent panic (e.g., nil store) creates an infinite restart loop.

### 5.4 Redis WriteTimeout Bug

**File:** `internal/cache/redis.go:32`

```go
WriteTimeout: cfg.RedisReadTimeout  // Should be cfg.RedisWriteTimeout
```

Copy-paste bug — uses read timeout for write timeout.

### 5.5 Dead Code

- `encoderPool` in `audit/logger.go:24-27` — declared but never used
- `ListRulesQuery.Cache` field in `domain/cqrs.go:265-266` — cache port wired but never called

### 5.6 Custom `contains` Reimplements stdlib

**File:** `internal/database/tx.go:128-139`

Custom `contains` function when `strings.Contains` from the standard library would suffice.

---

## 6. Performance Bottlenecks

### 6.1 Regex Compilation on Every Request

**File:** `internal/web/handlers.go:1312`

`validateContentAgainstRules` compiles regex patterns on every call. The validation engine caches compiled patterns, but the web handler does not use it.

### 6.2 Unbounded File I/O

`os.ReadFile` calls in MCP tool handlers have no size limits. A large file blocks the goroutine and consumes unbounded memory.

### 6.3 Aggressive Connection Pool

**File:** `internal/database/postgres.go:23`

```go
minConnections = 50  // + NumCPU() * connMultiplier
```

On a 16-core machine: 114 minimum connections. This can exhaust PostgreSQL's `max_connections`.

### 6.4 No Pagination on ListReviews

**File:** `internal/vision/storage.go:103`

`ListReviews` has a default limit of 50 but no offset parameter — all results returned from the beginning.

---

## 7. Accessibility Gaps

**File:** `web/index.html`

| Issue | Severity |
|-------|----------|
| No ARIA labels on interactive elements | Critical |
| No keyboard navigation for dynamic team-item divs | Critical |
| Color-only status indicators (no text alternatives) | Serious |
| No skip-navigation link | Serious |
| No `role="main"` or landmark attributes | Moderate |
| `select.innerHTML` used for dynamic content (XSS risk) | Critical |
| Missing `<label>` association for dynamic content | Moderate |
| No reduced-motion media query | Moderate |
| Contrast ratio `#7f8c8d` on white (~4.1:1, needs 4.5:1) | Moderate |

---

## 8. Documentation Gaps

| Gap | Impact |
|-----|--------|
| No OpenAPI/Swagger spec for 30+ MCP tools and REST endpoints | High |
| No runbook or operations guide | High |
| No consolidated database schema documentation | Medium |
| INDEX_MAP.md / HEADER_MAP.md navigation not auto-synced | Medium |
| No CHANGELOG.md (version history only in git) | Medium |
| No dependency security documentation | Medium |
| ~100 .md files mix active docs with historical sprints | Low |

---

## 9. Proposed New Features

### Feature 1: Live Enforcement Pipeline (`/v1/policy/check`)

**Problem:** CI/CD systems have no lightweight REST endpoint to gate PRs on violations.

**Target User:** DevOps engineers, platform teams

**Design:** Single POST endpoint accepting file content and context, delegates to existing `GuardrailService`, returns structured violations response. Supports batch mode for entire commits.

**Effort:** 2-3 days

**Implementation:**
1. Add `PolicyCheckRequest`/`PolicyCheckResponse` models to `internal/models/`
2. Create `internal/web/policy.go` handler
3. Register route behind API key middleware
4. Add batch endpoint with concurrency limiter
5. Update `regression-guard.yml` workflow

---

### Feature 2: Safe Regex & SSRF Hardening

**Problem:** 3 SSRF vectors, ReDoS exposure, unauthenticated endpoints, weak session tokens.

**Target User:** Security engineers, compliance auditors

**Design:** Centralized `SafeFileRead` sandbox, cached safe regex compiler, auth-by-default middleware, expanded secrets scanner.

**Effort:** 1-2 days

**Implementation:**
1. Create `internal/security/boundary.go` with `SafeFileRead(root, userPath)`
2. Create `internal/security/regex_compiler.go` with `sync.Map` cache
3. Replace 3 `os.ReadFile` calls in `tools_extended.go`
4. Replace `regexp.Compile` in `web/handlers.go:1312`
5. Invert auth logic in middleware (require by default)
6. Increase session token to 24 bytes (192 bits)
7. Add 15 new secret patterns

---

### Feature 3: Agent Lifecycle State Machine

**Problem:** No agent lifecycle enforcement. Halt mechanism is broken (`len(criticalEvents) < 0`).

**Target User:** Team leads, QA engineers, compliance officers

**Design:** State machine (IDLE → PLANNING → ACTIVE → REVIEW → RELEASE) with phase-scoped rules and working halt enforcement.

**Effort:** 5-7 days

**Implementation:**
1. Define `AgentState` enum and transition table
2. Create `agent_state.go` store
3. Add `phases` JSONB column to `prevention_rules`
4. Modify `GuardrailService.EvaluateInput()` for phase-scoped evaluation
5. Fix halt condition: `< 0` → `> 0`
6. Add MCP tools: `create_agent_session`, `transition_agent_state`, `get_agent_state`

---

### Feature 4: Token Budget Ledger & Cost Governor

**Problem:** AI API costs are opaque. No budget enforcement. Vision pipeline has no cost tracking.

**Target User:** Engineering managers, platform teams, finance

**Design:** Token estimation model, usage ledger, budget-aware circuit breaker, real-time status endpoint.

**Effort:** 3-4 days

**Implementation:**
1. Create `internal/budget/` package (Ledger, Estimator, Governor)
2. Define pricing constants for Claude, GPT-4o, GPT-4-turbo, local
3. Add database migration for `budget_configs` and `budget_entries`
4. Instrument `review_engine.go` to record usage
5. Wrap circuit breaker with budget check
6. Add MCP tool and REST endpoint for budget status

---

### Feature 5: Violation Webhook Notifications

**Problem:** EventBus publishes in-memory but nothing external consumes events.

**Target User:** Team leads, on-call engineers, project managers

**Design:** Webhook dispatcher subscribing to EventBus, HMAC-SHA256 signed payloads, per-team configuration.

**Effort:** 1 day

**Implementation:**
1. Create `internal/notifications/webhook_dispatcher.go`
2. Define `WebhookConfig` struct with team_id, url, events, secret_hmac
3. Store configs in new `webhook_configs` table
4. Subscribe to EventBus for ViolationDetected, HaltTriggered, BudgetExceeded
5. Wrap HTTP client with circuit breaker
6. Add MCP tools: `configure_notifications`, `test_webhook`, `list_webhooks`

---

### Feature 6: OpenAPI Auto-Generation & API Explorer

**Problem:** 30+ endpoints with no API documentation. API.md is out of date.

**Target User:** External developers, internal developers, QA engineers

**Design:** Auto-generate OpenAPI 3.1 spec from Go handlers, serve Swagger UI at `/docs`, generate MCP tool manifest.

**Effort:** 1-2 days

**Implementation:**
1. Add OpenAPI struct tags to all models in `internal/models/`
2. Write `scripts/generate_openapi.go` code-generation script
3. Mount Swagger UI or Scalar at `/docs`
4. Create `scripts/generate_mcp_manifest.go`
5. Add CI step to catch spec drift

---

### Feature 7: Docker Compose Local Dev Stack

**Problem:** Multi-hour setup process for new contributors. Requires PostgreSQL, Redis, Go server, web UI.

**Target User:** New contributors, QA engineers, demo environments

**Design:** Single `docker-compose.yml` with all services, migration init, health checks, live reloading.

**Effort:** 0.5-1 day

**Implementation:**
1. Create `Dockerfile` (multi-stage: `golang:1.24` builder, `gcr.io/distroless/static` runner)
2. Create `docker-compose.yml` with PostgreSQL, Redis, MCP server
3. Add `docker/init.sh` for migration runner
4. Add health checks for all services
5. Create `.env.example`
6. Add `make dev` target

---

## 10. Implementation Roadmap

### Sprint 1 (Week 1-2): Foundation and Quick Wins

| Order | Task | Effort | Rationale |
|-------|------|--------|-----------|
| 0 | Fix P0 build issues | 0.5 day | Unblocks everything |
| 1 | Docker Compose | 1 day | Unblocks all contributors |
| 2 | Webhook Notifications | 1 day | Zero dependencies, enables alerts |
| 3 | Security Hardening | 2 days | Closes SSRF/ReDoS vulnerabilities |

**Sprint 1 output:** Working local dev stack, webhook notifications, security boundary.

### Sprint 2 (Week 3-4): Integration and Documentation

| Order | Task | Effort | Rationale |
|-------|------|--------|-----------|
| 4 | Enforcement Pipeline | 3 days | Unblocks CI/CD integration |
| 5 | OpenAPI Generation | 2 days | Developer experience |

**Sprint 2 output:** CI/CD can gate PRs on violations. API is documented.

### Sprint 3 (Week 5-7): Governance and Core Platform

| Order | Task | Effort | Rationale |
|-------|------|--------|-----------|
| 6 | Budget Ledger | 4 days | Cost governance |
| 7 | State Machine | 5 days | Most complex, architecturally significant |

**Sprint 3 output:** Token budgets enforced, agent lifecycle managed, halt mechanism functional.

**Total estimated effort:** 13-19 days (2.5-4 weeks)

---

## 11. Dependencies Between Features

```
Feature 7 (Docker) ─────────────── no dependencies
Feature 5 (Webhooks) ───────────── no dependencies
Feature 4 (Security) ───────────── no dependencies
Feature 1 (Enforcement Pipeline) ── depends on: P0 build fix
Feature 6 (OpenAPI) ─────────────── no dependencies
Feature 3 (Budget) ──────────────── depends on: Feature 5 (budget_exceeded → webhook)
Feature 2 (State Machine) ───────── depends on: Feature 4 (halt mechanism is security-critical)
```

---

## 12. Risk Assessment

### Quick Wins Risks

| Feature | Risk | Mitigation |
|---------|------|------------|
| Docker Compose | Migration ordering with 33 existing migrations | Test with fresh DB, retry logic in init.sh |
| Webhooks | Outbound HTTP could block request handlers | Fire-and-forget goroutine with circuit breaker |
| Security | SafeFileRead could break legitimate access | Property-based testing, feature flag rollout |

### Core Improvements Risks

| Feature | Risk | Mitigation |
|---------|------|------------|
| Enforcement Pipeline | Rate limiting blocks legitimate CI/CD traffic | High limits for authenticated API keys |
| OpenAPI | Spec drift between code and docs | CI step runs generator and diffs |
| Security | Replacing regexp.Compile could break patterns | Test all existing rules against safe compiler |

### Strategic Features Risks

| Feature | Risk | Mitigation |
|---------|------|------------|
| Budget Ledger | Token estimation inaccuracy | Pricing constants configurable via DB |
| State Machine | Complex transitions could deadlock agents | Timeout on transitions, FORCE_TERMINAL escape |
| State Machine | Halt bug fix surfaces previously-ignored events | Test in staging first |

---

## Appendix: Files Referenced

### Critical Issue Locations

| File | Line(s) | Issue |
|------|---------|-------|
| `mcp-server/internal/domain/cqrs.go` | 5-6 | Unused imports (build broken) |
| `mcp-server/internal/mcp/server.go` | 818 | Duplicate `buildToolResult` |
| `mcp-server/internal/mcp/tools_extended.go` | 378 | Duplicate `buildToolResult` |
| `mcp-server/internal/mcp/tools_extended.go` | 788 | `len(criticalEvents) < 0` (always false) |
| `mcp-server/internal/mcp/tools_extended.go` | 267, 2116, 2194 | SSRF via os.ReadFile |
| `mcp-server/internal/web/handlers.go` | 1312 | ReDoS exposure |
| `mcp-server/internal/web/middleware.go` | 56-68, 88-92 | File extension bypass, unauth endpoints |
| `mcp-server/internal/mcp/server.go` | 841 | 64-bit session token |
| `mcp-server/internal/cache/redis.go` | 32 | WriteTimeout copy-paste bug |
| `mcp-server/internal/audit/logger.go` | 125-126, 131-132, 138-143 | Panic, dropped events, restart loop |

### New Feature Locations (to be created)

| File | Feature |
|------|---------|
| `mcp-server/internal/models/policy_check.go` | Enforcement Pipeline |
| `mcp-server/internal/web/policy.go` | Enforcement Pipeline |
| `mcp-server/internal/security/boundary.go` | Security Hardening |
| `mcp-server/internal/security/regex_compiler.go` | Security Hardening |
| `mcp-server/internal/models/agent_state.go` | State Machine |
| `mcp-server/internal/database/agent_state.go` | State Machine |
| `mcp-server/internal/mcp/tools_lifecycle.go` | State Machine |
| `mcp-server/internal/budget/` (new package) | Budget Ledger |
| `mcp-server/internal/notifications/webhook_dispatcher.go` | Webhooks |
| `mcp-server/internal/mcp/tools_notifications.go` | Webhooks |
| `mcp-server/internal/mcp/tools_budget.go` | Budget Ledger |
| `mcp-server/internal/web/budget.go` | Budget Ledger |
| `docker-compose.yml` | Docker Compose |
| `Dockerfile` | Docker Compose |
