# Sprint: Principal Authorization and Decision Audit

**Sprint ID:** SPRINT-A1
**Source Spec:** 16 — Gate 2 (R16-06, R16-07, R16-08)
**Priority:** P0 (Critical)
**Estimated Effort:** ~3–5 days
**Status:** PARTIALLY IMPLEMENTED (no item ACCEPTED) — see Reconciliation 2026-10-04
**Depends On:** SPRINT-A0 (registry fail-closed and config secrecy)
**Blocks:** SPRINT-A2

---

## Reconciliation 2026-10-04 — verified against source at commit 1c838f4

Status vocabulary: **WIRED** (code exists, no exact live test), **EXERCISED**
(exact test ran, cited), **NOT_EXERCISED** (live dependency skipped), **NOT_RUN**
(tool/runner unavailable). No item below is ACCEPTED.

- R16-06 (scope × role × resource intersection) on web middleware — **EXERCISED**: `internal/web/middleware_authz_test.go:55` (`TestAuthzScopeRoleResourceMatrix`).
- R16-06 on MCP full path — **WIRED / NOT_EXERCISED**: exercised only through helpers, not the StreamableHTTP endpoint.
- R16-07 legacy containment — **EXERCISED**: `internal/web/middleware_registry_test.go:97-130`.
- R16-08 decision-audit fail-closed — **EXERCISED**: `internal/web/middleware_authz_test.go:145` (`TestAuthzAdminMutationFailsClosedWithoutAudit`).
- `hashAPIKey` is never used as identity — **WIRED**.

---

## Problem Statement

Credentials are not reliably resolved to a server-controlled principal with
reviewed scopes, role grants, and resource/project membership. Authorization
can accept identity or role from tool arguments/content. Legacy safe-method and
`/ide/` prefix access remain overly broad. Decision and effect audit is
incomplete (including project create/update/delete and denials), and truncated
`hashAPIKey` may be misused as identity. Without a shared principal contract,
later lifecycle work cannot be safe.

**Why:** Scope, role, and resource intersection is not applied before every
effect on both transports; legacy containment and audit are incomplete.
**Where:** Web auth/rate-limit public-route logic, MCP dispatch and resource
handlers, audit persistence (see Spec 16 R16-06–08; Spec 11/12 alignment).

---

## Entry Gate

Before starting this sprint:

- [ ] SPRINT-A0 exit gate is complete (registry fail-closed; no config/argument secret leakage).
- [ ] Exposure inventory and permission matrix (R16-01) identify which legacy clients require which access.
- [ ] Baseline tests recorded from `mcp-server/`:
      `go test ./internal/auth ./internal/web ./internal/mcp -count=1`.

---

## Exit Gate

This sprint is complete only when all of the following hold:

- [ ] A credential resolves to server-controlled principal ID, credential ID, reviewed scopes, role grants, resource/project membership, and status. — **EXERCISED** on web (`middleware_authz_test.go:55`); MCP via helper only (**WIRED**).
- [ ] Identity/role/tenant is never accepted from tool arguments or content. — **WIRED**.
- [ ] `authenticated AND scope_allows AND role_allows AND resource_allows` runs before every effect, including conditional handlers. — **EXERCISED** on web (`middleware_authz_test.go:55`); MCP full path **WIRED/NOT_EXERCISED**.
- [ ] Confirmation flags remain intent checks after authorization only. — **NOT_EXERCISED** (no audited verdict).
- [ ] Missing/unknown role, resource, scope, or operation denies: HTTP 401 unauthenticated, 403 authenticated-forbidden, stable non-leaking MCP permission error. — **EXERCISED** on web (`middleware_authz_test.go:55`); MCP error shape via helper only.
- [ ] Scope/role mismatch and cross-project cases deny without side effects (even with `confirmed=true`). — **EXERCISED** on web (`middleware_authz_test.go:55`, cross-project row).
- [ ] Arbitrary safe-method and `/ide/` prefix access is replaced by the approved method/path/action table. — **EXERCISED** (`middleware_registry_test.go:97-130`).
- [ ] Legacy MCP key calls use the same privilege model; unattributable/unconstrainable legacy principals deny privileged and mutating actions. — **EXERCISED** (`middleware_registry_test.go:97-130`).
- [ ] Public preflight never authorizes its corresponding effect; path normalization and suffix tricks deny. — **NOT_EXERCISED** (no audited verdict).
- [ ] Decision/effect audit records principal + opaque credential ID, action, safe resource/project ID, outcome, reason code, and policy version for both transports (including project create/update/delete and denials). — web **EXERCISED** (`middleware_authz_test.go:145`); MCP **NOT_EXERCISED**.
- [ ] Truncated `hashAPIKey` is never used as identity. — **WIRED**.
- [ ] Security-sensitive mutations fail closed if required durable audit cannot be recorded. — **EXERCISED** (`middleware_authz_test.go:145`).

---

## Scope Boundary

```
IN SCOPE (may modify):
  - Credential → principal resolution and request-context plumbing (web + MCP)
  - Scope/role/resource intersection checks before effects
  - Legacy method/path/action containment and public-route alignment
  - Decision/effect audit records and fail-closed audit persistence
  - Tests: authorization matrix, legacy containment, audit durability

OUT OF SCOPE (DO NOT TOUCH):
  - Registry load fail-closed (completed in SPRINT-A0; read-only verification)
  - Credential generation/rotation/revocation APIs (SPRINT-A2 / R16-09)
  - Deployment TLS/profiles/Compose (SPRINT-A3 / R17)
  - CI matrix/gates (SPRINT-A4 / R19)
  - Spec 15/14 external evidence (Gate 4)
  - New product features, UI work, documentation beyond this sprint file
```

---

## Numbered Tasks

### Task 1 — Shared principal and permission contract (R16-06)

**Action:** Resolve each credential to a server-controlled principal and pass
it through web and MCP contexts. Apply
`authenticated AND scope_allows AND role_allows AND resource_allows` before
every effect. Never take identity/role/tenant from tool arguments or content.
Treat confirmation flags as post-authorization intent checks. Deny missing or
unknown role/resource/scope/operation with the transport-appropriate stable
error (401 / 403 / non-leaking MCP permission error).

**Acceptance criteria:**
- Registered `mcp:read`-only key + admin role → project deletion and force-state denied, zero side effects.
- `mcp:mutate` key + reader role → mutations denied, zero side effects.
- Only explicit scope + role + project grant permits the intended action (still subject to confirmation).
- Caller allowed on project A but not B → identical tool/REST mutation on B denied even with `confirmed=true`.
- Tool arguments claiming admin/tenant/project identity are ignored.

**Expected test commands** (from `mcp-server/`):
```sh
go test ./internal/auth ./internal/web ./internal/mcp -run 'TestAPIKeyAuth_|TestRequireBearer|TestMCPEndpointBehindBearer|TestAuthz|TestScope|TestRole|TestResource' -count=1
go test ./internal/auth ./internal/web ./internal/mcp -count=1
```

### Task 2 — Legacy containment and web exposure (R16-07)

**Action:** Replace arbitrary safe-method and `/ide/` prefix access with the
approved method/path/action table. Apply the same privilege model to legacy
MCP key calls. If a legacy principal cannot be attributed or constrained, deny
privileged and mutating actions (do not restore unrestricted behavior). Align
auth and rate-limit public-route logic with Spec 12. Test path normalization,
suffixes, and preflight separately.

**Acceptance criteria:**
- Legacy key + unlisted GET/HEAD/OPTIONS route → denied under migration policy.
- Legacy key + protected resource → denied.
- Legacy key + `POST /ide/validate/file` (or unlisted `/ide/` action) → denied.
- Legacy key + MCP mutation not explicitly granted → denied.
- Public preflight never authorizes its corresponding effect.
- Filename suffix, `OPTIONS`, or arbitrary GET does not confer access.

**Expected test commands:**
```sh
go test ./internal/web ./internal/mcp -run 'TestLegacy|TestIde|TestPath|TestSuffix|TestPreflight|TestPublicRoute' -count=1
go test ./internal/web ./internal/mcp -count=1
```

### Task 3 — Decision/effect audit (R16-08)

**Action:** Record for both transports: principal and opaque credential ID,
action, safe resource/project ID, outcome, reason code, and policy version —
including project create/update/delete and denied requests. Never use truncated
`hashAPIKey` as identity. Security-sensitive mutations fail closed if required
durable audit cannot be recorded. State the durability boundary and test
failures before and after authorization.

**Acceptance criteria:**
- Allowed project deletion → attributable durable record exists.
- Denied project deletion → attributable record exists; no deletion occurs.
- Audit persistence failure → no deletion (or other sensitive mutation) occurs without the required durable record.
- Raw key and full argument maps are absent from audit records.
- Durability boundary is documented and covered by tests.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/web ./internal/mcp ./internal/database -run 'TestAudit|TestDecision|TestProjectDelete' -count=1
go test ./internal/... -count=1
go vet ./...
```

### Task 4 — Cross-transport negative/positive proof (R16-06–08)

**Action:** Prove all R16-06–08 cases through real web middleware and
StreamableHTTP tool/resource calls, not only helper unit tests. Assert both
stable denial and zero protected side effect on every negative case. Record
owner decision and source review.

**Acceptance criteria:**
- Named matrix rows (scope/role/resource mismatch, legacy fallback, path/method bypass, sensitive resource) deny as specified.
- Unauthorized side-effect assertions pass.
- No production credential or production database used.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/web ./internal/mcp -count=1
go test ./internal/... -count=1
```

---

## Dependencies

| Direction | Sprint / Requirement | Relationship |
|-----------|----------------------|--------------|
| Requires | SPRINT-A0 (R16-03–05) | Registry must be fail-closed and config/args secret-free before binding identity |
| Requires | Spec 16 Gate 0 (R16-01) | Exposure and permission matrix define the action table |
| Blocks | SPRINT-A2 (R16-09) | Rotation/revocation must operate on attributed principals and audit |
| Aligns | Spec 11 roles/scopes, Spec 12 web exposure, Spec 17 R17-07 | Same intersection and exposure contract |

---

## Rollback Notes

- Roll back to the last known-good **deny-by-default** authorization behavior — never to unrestricted safe-method or `/ide/` prefix access.
- If audit cannot be written for a sensitive mutation, fail closed; do not complete the effect and “audit later.”
- Revert files with `git checkout HEAD -- <files>` if a change weakens intersection checks or reintroduces argument-sourced identity.
- Keep legacy principals in contained/denied state during rollback; do not restore pre-migration unrestricted legacy authority.

---

## Do Not Proceed If

**STOP this sprint (and do not continue to SPRINT-A2) if any of the following is true:**

- Identity, role, or tenant can be influenced by tool arguments or request content.
- Any effect can run when scope, role, or resource intersection fails.
- `confirmed=true` (or equivalent) can authorize an effect the principal is not allowed to perform.
- A legacy key can mutate state or use privileged actions that the approved table does not grant.
- Project deletion (or other security-sensitive mutation) can complete without a required durable audit record.
- Truncated `hashAPIKey` or raw key material appears as identity in audit.
- Negative tests do not assert zero side effects, or only helper units are tested (no real web/MCP paths).

---

## Acceptance Criteria Summary

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | Principal resolution (R16-06) | Authn + principal plumbing tests | Server-controlled principal; no arg-sourced identity |
| 2 | Intersection before effects | Scope/role/resource matrix, both transports | Deny on any failed factor; 401/403/MCP error as specified |
| 3 | Cross-project deny | Same mutation on A vs B | B denied even with confirmation |
| 4 | Legacy containment (R16-07) | Unlisted methods, `/ide/`, suffixes, preflight | Denied; preflight never authorizes effects |
| 5 | Decision audit (R16-08) | Allow/deny/audit-failure project delete | Durable attributable record; fail closed without audit |
| 6 | Real-transport proof | Full package tests | R16-06–08 pass; no unauthorized side effects |

---

## Reference

- [Spec 16](../specs/16-authentication-authorization-and-evidence-remediation.md) — Gate 2, R16-06–08
- [Spec 11](../specs/11-authorization-scopes-and-roles.md) — scopes and roles
- [Spec 12](../specs/12-web-exposure-boundary.md) — public routes, CORS, proxies
- [Spec 19](../specs/19-phase0-ci-security-gates.md) — R19-03–04 (negative controls)
- Template basis: [SPRINT_TEMPLATE.md](../archive/sprints/SPRINT_TEMPLATE.md) (adapted for security remediation)

---

**Created:** 2026-10-04
**Version:** 1.1
**Status:** PARTIALLY IMPLEMENTED (reconciled 2026-10-04 against commit 1c838f4)
