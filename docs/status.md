# Project Status - Guardrail MCP Server

**Last Updated:** 2026-08-15
**Branch:** main
**Current Version:** v3.3.0

---

## Completed Sprints

### Sprint 001: MCP Gap Implementation - COMPLETED
- **Status:** ✅ COMPLETE
- **Date Completed:** 2026-02-08
- **Coverage:** 11 tools, 8 resources

**Implemented:**
- 5 gap tools (validate_scope, validate_commit, prevent_regression, check_test_prod_separation, validate_push)
- 6 gap resources (agent-guardrails, four-laws, halt-conditions, workflows, standards, pre-work-checklist)

---

### Sprint 002: Web UI Implementation - COMPLETED
- **Status:** ✅ COMPLETE
- **Date Completed:** 2026-02-08
- **Coverage:** 26/26 API endpoints, 6/6 pages

**Implemented:**
- Complete SPA with 6 pages (Dashboard, Documents, Rules, Projects, Failures, IDE Tools)
- 26 API endpoints in api.js
- 5 reusable components (Navigation, DataTable, Forms, Modal, Toast)
- 3 CSS files (variables, components, layout)
- Hash-based routing

---

### Sprint 003: Documentation Parity - COMPLETED
- **Status:** ✅ COMPLETE
- **Date Completed:** v2.9.0 era
- **Focus:** Align documentation with implementation

**Delivered:**
- API documentation updated to match implementation
- Web UI user guide added
- All MCP tools and resources documented
- README updated with deployment instructions

---

### Sprint 004: Document Ingestion System - COMPLETED
- **Status:** ✅ COMPLETE
- **Date Completed:** 2026-02-09
- **Team:** 4 parallel agents

**Implemented:**
- Database migrations for ingest tracking
- Markdown parser with YAML frontmatter support
- Ingest service (repo sync + file upload)
- Update checker (Docker + Guardrail versions)
- Web UI file upload with drag-and-drop
- Update notifier with daily checks

---

## v3.2.0 Platform Review Sprint - COMPLETED

- **Status:** ✅ COMPLETE
- **Released:** 2026-06-16
- **Scope:** Platform review sprint — 7 new features + P0 bug fixes
- **MCP Server Port:** mcp-go v0.4.0 → v0.58.0

**7 Features Delivered:**
1. CI/CD Enforcement Pipeline
2. Webhook Notification System
3. Token Budget Ledger & Cost Governor (incl. vision pipeline instrumentation)
4. Agent Lifecycle State Machine (incl. audit trail)
5. OpenAPI 3.1 Specification + Scalar API Explorer
6. Platform review P0 bug fixes (all resolved)
7. Additional sprint hardening (budget, lifecycle, review gates)

**References:**
- [Platform Review](archive/reviews/PLATFORM_REVIEW_2026-06-14.md)
- [Implementation Report](archive/reviews/IMPLEMENTATION_REPORT_2026-06-14.md)

---

## v3.3.0 Transport & Infrastructure Sprint - COMPLETED

- **Status:** ✅ COMPLETE
- **Released:** 2026-08-15
- **Scope:** Transport migration, build/deploy fixes, repo cleanup

**Delivered:**
1. Stateless StreamableHTTP transport (replaces SSE) — single `POST /mcp` endpoint
2. Fixed broken v3.2.0 build (mcp-go v0.58.0 port completed)
3. All 18 database migrations repaired and renumbered — fresh installs work
4. Container deploy fixes (PostgreSQL capabilities, Redis config, `BIND_ADDR`)
5. JWT secret validation rewritten (Shannon entropy replaces broken popcount check)
6. ~53MB of committed binaries removed; game-design docs moved to private repo

**Reference:** [v3.3.0 release notes](releases/v3.3.0.md)

---

## Deployment Status

The server is deployed and verified on an internal test VM (reachable over
Tailscale). Ports are bound to the Tailscale interface by default via
`BIND_ADDR` — localhost-only unless configured otherwise.

- Version: v3.3.0
- Status: ✅ Running (MCP + Web UI healthy)
- Transport: Stateless StreamableHTTP (`POST /mcp`)
- Features: Full tool set (35 tools, 11 resources), migrations applied cleanly

---

## Code Review Rounds Completed

### Round 1: SPA Fallback Fix
- Fixed static file serving order
- SPA fallback now checks file existence

### Round 2: CORS and JS Fixes
- Added OPTIONS support for CORS preflight
- Fixed Failures.js parameter name

### Round 3: Database Type Fixes
- Fixed Project.ActiveRules type (pq.StringArray)
- Fixed Project.Metadata type (jsonb handling)
- Fixed PreventionRule.PatternHash (nullable)

---

## Next Steps

1. **Enforce 500-Line File Limit**
   - Audit all docs and source files for the 500-line maximum
   - Split any over-limit files (refactor into sections + update INDEX_MAP/HEADER_MAP)

2. **Dead Code Cleanup**
   - Remove unused functions, endpoints, and components flagged during review
   - Verify no orphaned references remain after removal

3. **Maintenance**
   - Keep docs in parity with implementation (see Sprint 003 discipline)
   - Continue platform review follow-through for new feature hardening
