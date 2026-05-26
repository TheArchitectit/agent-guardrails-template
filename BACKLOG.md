# Backlog: MCP + Skills Continuous Development (`mcp-skills-continuous`)

> Branch: **mcp-skills-continuous** (forked from `main` v3.1.0)
> Created: May 26, 2026
> Priority Legend: 🔴 Critical | 🟡 High | 🟢 Medium | ⚪ Low

---

## 🔴 Critical — Build Fixes

### CRIT-1: `mcp-server` fails to compile (2 packages broken)

The MCP server has 2 failing packages — the core `internal/mcp` and `internal/domain` packages don't compile.

**Package: `internal/mcp`**
- `buildToolResult` redeclared in `server.go:818` and `tools_extended.go:378` — need to deduplicate or rename one
- `server.go:33-42` references undefined types: `cache.Cache`, `metrics.Metrics`, `audit.AuditLogger`, `validation.Engine`, `VisionTools` — these types need to exist (or be stubs) at compilation time
- Likely a missing or incomplete merge — `cache`, `metrics`, `audit` packages exist but may lack the expected exports

**Package: `internal/domain`**
- `cqrs.go:159`: `h.repo.Toggle` called but `RuleRepository` interface has no `Toggle` method — add it to the interface

> **Effort:** ~2-4h | **Risk:** Blocking all other MCP work

### CRIT-2: Audit — remove `log.Fatal()` from non-critical paths

- `AUDIT-REPORT.md` flags several `log.Fatal()` calls in non-critical paths
- Replace with proper error returns and structured logging (slog or zerolog)

> **Effort:** ~1h | **Risk:** Medium — could crash in production

---

## 🟡 High — Feature Gaps

### HIGH-1: `.guardrails/` is duplicated across root and `mcp-server/`

- Root has `.guardrails/` with `failure-registry.jsonl`, `prevention-rules/`, `pre-work-check.md`, `team-layout-rules.json`, `web-ui-team-rules.json`
- `mcp-server/.guardrails/` has its own `failure-registry.jsonl`, `pre-work-check.md`, `prevention-rules/`
- These should be **unified** — root should be the single source of truth, mcp-server should symlink or reference

> **Approach:** Consolidate to root `.guardrails/`, update mcp-server config to read from there

### HIGH-2: `web/` is a single `index.html` (stub)

- Only 9.4KB of one-page HTML
- README mentions "Web UI: Dashboard, document browser, rules management, failure registry" but none of that exists
- Should be a proper dashboard (React or vanilla) with:
  - MCP server health/status dashboard
  - Skills browser with search
  - Rule editor
  - Failure registry viewer
  - Configuration management

> **Approach:** Start with a minimal but functional dashboard. SPA or multi-page.

### HIGH-3: Skills directory spans too many agent platforms — needs consolidation

Skills are scattered across:
- `.claude/skills/` — 7 JSON skill files
- `.claude/skills-3d/` — 1 JSON skill file
- `.cursor/rules/` — 3 markdown files
- `.cursor/rules-3d/` — 1 markdown file
- `.opencode/skills/` — 4 SKILL.md files
- `.opencode/agents/` — 2 agent configs
- `skills/` — OpenClaw skills (A2A, 3D-game-dev, etc.)

No single index of what's available across all platforms. The format is inconsistent (JSON vs Markdown vs SKILL.md).

> **Approach:** Create a platform-agnostic `SKILLS_INDEX.md` that maps each skill to its platform-specific locations. Standardize the core content.

### HIGH-4: No end-to-end tests run automatically

- `tests/` has only a `regression/README.md` — no actual test files
- `scripts/e2e_tests.py` exists but no CI workflow runs it
- No integration tests between MCP server, web UI, and guardrails

> **Approach:** Add a `make test-e2e` target and a GitHub Actions workflow that runs it

---

## 🟢 Medium — Improvements

### MED-1: Go module path mismatch

- `mcp-server/go.mod` has `module github.com/thearchitectit/guardrail-mcp` but the repo is `TheArchitectit/agent-guardrails-template`
- `cmd/team-cli/go.mod` likely has the same issue
- Import paths need updating across all Go files

> **Effort:** ~1h (find-replace) | **Risk:** Low if imports align

### MED-2: Add MCP end-to-end test suite

- `testing.T` tests exist in `internal/models`, `internal/security`, `internal/config`, etc. (6 packages pass)
- No integration tests that spin up the MCP server and call it via SSE/JSON-RPC
- Should add `mcp-server/internal/mcp/server_test.go` with integration tests

> **Approach:** Use httptest.Server or run the MCP binary. Test tool registration, tool calls, resource access.

### MED-3: `internal/mcp/server.go` and `handlers.go` are too large

- AUDIT-REPORT flags Single Responsibility Principle violations
- Split into: `server.go` (core setup), `tools.go` (tool registry), `handlers.go` (request handlers), `transport.go` (SSE/HTTP transport)

> **Effort:** ~2h | **Risk:** Low if function boundaries are already clean

### MED-4: `internal/cache` has no tests

- No test files for the cache package
- Basic CRUD and TTL expiry tests needed

### MED-5: `internal/database` has no tests

- No test files for the database package
- Test with in-memory SQLite or testcontainers

### MED-6: `internal/team` has no tests

- No test files for the team package
- Test team layout rules, validation

### MED-7: `internal/vision` has no tests

- Vision pipeline tools exist (capture, inference, composite review) but no tests
- At minimum: unit tests for VisionTools

### MED-8: GitHub Actions not running Go tests

- `.github/workflows/` has: `secret-validation.yml`, `team-validation.yml`, `documentation-check.yml`, `regression-guard.yml`, `guardrails-lint.yml`
- **Missing:** `go-test.yml` — should run `go build ./... && go test ./...` on push/PR
- **Missing:** `web-build.yml` — should lint/build the web UI if it grows

### MED-9: `STATUS.md` is outdated

- Last updated "2026-03-14" (v2.8.0) — now on v3.1.0
- Missing: Sprints 005, 006, vision pipeline feature, AgentMCP integration

### MED-10: `INDEX_MAP.md` and `HEADER_MAP.md` may be stale

- These are token-saver indices — need audit vs actual file tree
- `TOC.md` should be regenerated

### MED-11: OpenClaw skills (`skills/`) need an index

- 10+ skills in `skills/` (a2a-client, a2a-register, a2a-server, 3d-game-dev, etc.)
- No master index of which skills exist and what they do
- Create `skills/INDEX.md`

### MED-12: Shell hooks need testing

- `.claude/hooks/` has `pre-commit.sh`, `pre-execution.sh`, `post-execution.sh`
- No tests or validation scripts for these hooks
- Add a `make test-hooks` target

---

## ⚪ Low — Nice to Haves

### LOW-1: Add `.github/workflows/dependency-review.yml`

- GitHub's Dependency Review action per PR — standard supply chain security

### LOW-2: `scripts/` README

- 9 scripts but no documentation on what each does, dependencies, or usage
- Add `scripts/README.md`

### LOW-3: `Makefile` for root project

- `mcp-server/Makefile` exists but no root Makefile
- Add `make build`, `make test`, `make lint`, `make web`, `make docker` at root

### LOW-4: Docker Compose for the full stack

- `mcp-server/deploy/docker-compose.example.yml` exists but references example paths
- Should have a working `docker-compose.yml` at root that starts: MCP server, Web UI, PostgreSQL, Redis

### LOW-5: Add `CONTRIBUTING.md` updates

- Current `CONTRIBUTING.md` is extensive (this doc is good — low priority to change)
- Could add "How mcp-skills-continuous works" section

### LOW-6: `.env.example` audit

- Root `.env.example` exists — audit that all required env vars are documented
- `mcp-server/.env.example` should be kept in sync

### LOW-7: Add OpenClaw ACP agent config for the cron agent

- The cron agent fires into the repo but has no dedicated agent config
- Create `.openclaw/agent-guardrails.md` with the dev agent's personality, project context, and MCP knowledge

### LOW-8: `GAP_REMEDIATION_BRANCH.md` review

- This document exists — check if the gaps it identifies are addressed in current state

### LOW-9: `cpofopencode` file review

- 17KB file — needs audit to ensure it's still relevant

### LOW-10: `.aider` cleanup

- `.aider.chat.history.md`, `.aider.input.history`, `.aider.tags.cache.v4/` — these are aider IDE artifacts
- Should be gitignored or cleaned up periodically

---

## Sprint Planning Recommendations

### Sprint 0: Fix Build (`mcp-skills-continuous`)
- CRIT-1: MCP server compile fix
- CRIT-2: Remove log.Fatal
- MED-1: Fix Go module path
- MED-8: Add go-test.yml GitHub Action

### Sprint 1: Web UI MVP
- HIGH-2: Web dashboard
- HIGH-4: E2E test setup

### Sprint 2: Skills Consolidation
- HIGH-3: Skills index and standardization
- MED-11: OpenClaw skills index

### Sprint 3: Test Infrastructure
- MED-2: MCP integration tests
- MED-4/5/6/7: Coverage for cache, database, team, vision

### Sprint 4: Cleanup & Refactor
- HIGH-1: .guardrails unification
- MED-3: server.go splitting
- MED-9: STATUS.md update
- LOW-3/4: Makefile + Docker Compose

### Sprint 5: Polish
- LOW-1/2/5/6/7/8/9/10: All low-priority items
