# Sprint: Registry Fail-Closed and Config Secrecy

**Sprint ID:** SPRINT-A0
**Source Spec:** 16 — Gate 0–1 (R16-03, R16-04, R16-05)
**Priority:** P0 (Critical)
**Estimated Effort:** ~2–3 days
**Status:** PARTIALLY IMPLEMENTED (no item ACCEPTED) — see Reconciliation 2026-10-04
**Depends On:** — (first Phase 0 remediation sprint)
**Blocks:** SPRINT-A1, SPRINT-A2, SPRINT-A3

---

## Reconciliation 2026-10-04 — verified against source at commit 1c838f4

Status vocabulary: **WIRED** (code exists, no exact live test), **EXERCISED**
(exact test ran, cited), **NOT_EXERCISED** (live dependency skipped), **NOT_RUN**
(tool/runner unavailable). No item below is ACCEPTED.

- Registry fail-closed on web middleware — **EXERCISED**: `internal/web/middleware_failclosed_test.go:19` (`TestRegistryConfiguredInvalidDeniesAllProtectedTraffic`).
- Registry fail-closed on MCP full path — **EXERCISED**: `internal/mcp/streamable_http_authz_test.go` (`TestStreamableHTTPConfiguredBrokenRegistryDeniesAll`) drives the real StreamableHTTP `/mcp` endpoint behind `requireBearerWithRegistry` and proves a configured-broken registry denies all callers (legacy key, registered-looking token, no token) with no side effect (commit 508a34a).
- MCP resource secrecy (`guardrail://config`) — **WIRED**: helper `readConfigResourceContents` (`resource_registration.go:120`), exercised at `resource_config_secrecy_test.go:48`.
- Argument privacy — **WIRED**: helper `handleToolCall` (`server.go:202`), exercised at `argument_privacy_test.go:56`.
- Real StreamableHTTP `tools/call` end-to-end — **EXERCISED**: `internal/mcp/streamable_http_authz_test.go` (`TestStreamableHTTPToolsCallAuthorizesAndDenies`, allow + scope/role/resource denial with zero-side-effect assertions) (commit 508a34a).
- Gitleaks full-history scan — **NOT_RUN**.

> Gitleaks addendum 2026-10-06: the full-history scan has since been scripted
> and run (see SPRINT-A4 reconciliation 2026-10-05 part 3/part 4): 16 findings
> reviewed, 11 intentional false positives allowlisted in `.gitleaksignore`,
> 5 real-looking findings deliberately left un-ignored so the scan stays red
> until rotation/purge. The NOT_RUN row above reflects the 2026-10-04 snapshot
> at commit 1c838f4 only.

> MCP transport rows above now also reflect commit 508a34a (full-path test added after the 1c838f4 audit). Web rows and all NOT_EXERCISED/NOT_RUN items are unchanged.

---

## Problem Statement

Configured credential-registry state can fail open or silently downgrade to
unrestricted legacy access. Whole-config MCP resource serialization and full
MCP argument-map logging disclose secrets (DB password, API keys, JWT secret,
verifier key, raw credentials) to callers, logs, audit, and error paths. These
are immediate disclosure and fail-closed defects that block all later identity
and lifecycle work.

**Why:** Registry load, resource projection, and argument logging are not
treated as security boundaries.
**Where:** `mcp-server` registry load path, `guardrail://config` resource,
MCP argument logging (see Spec 16 R16-03–05; Spec 17 R17-07–08).

---

## Entry Gate

Before starting this sprint:

- [ ] Spec 16 Gate 0 evidence inventory is available or in progress (R16-01 exposure registry).
- [ ] No prior security gate is claimed green while this one is red.
- [ ] Test environment uses synthetic credentials only (no production secrets/DB).
- [ ] Baseline recorded: `go test ./internal/auth ./internal/web ./internal/mcp -count=1` results.

---

## Exit Gate

This sprint is complete only when all of the following hold:

- [x] Configured malformed/unreadable/empty/invalid registry fails readiness or denies protected traffic on **both** web and MCP; no silent legacy fallback. — web **EXERCISED** (`middleware_failclosed_test.go:19`); MCP full path **EXERCISED** (`streamable_http_authz_test.go`, commit 508a34a).
- [ ] Unknown scopes, duplicate credential IDs/verifiers, invalid records, and unusable verifier-key material are rejected at load. — **NOT_EXERCISED** (no audited verdict; load-time unit checks exist but were not in the 2026-10-04 verdict set).
- [ ] Unregistered credentials cannot inherit authority from registered ones. — **WIRED** (helper `requireBearerWithRegistry`, `auth_test.go:47`).
- [ ] Registry-only cutover path does not reinstate legacy access when the legacy MCP key is empty/absent. — **WIRED** (helper only).
- [ ] `guardrail://config` (or replacement) returns only a reviewed non-secret projection, or is removed; unauthorized reads deny stably. — **WIRED** (helper `readConfigResourceContents`).
- [ ] No DB password, API key, JWT secret, verifier key, or raw credential appears in resource output, errors, diagnostics, or audit. — **WIRED** (resource + argument helpers).
- [ ] Full MCP argument-map logging is removed; only approved fields are recorded. — **WIRED** (helper `handleToolCall`).
- [ ] Nested fake-secret argument markers are absent from logs, responses, metrics, and exported evidence. — **WIRED** (helper-level capture only).
- [x] Positive and negative tests exist and pass for R16-03–05 on real web middleware and StreamableHTTP MCP paths (not registry unit tests alone). — web **EXERCISED** (`middleware_failclosed_test.go:19`); StreamableHTTP MCP **EXERCISED** (`streamable_http_authz_test.go`: `TestStreamableHTTPToolsCallAuthorizesAndDenies`, `TestStreamableHTTPConfiguredBrokenRegistryDeniesAll`, `TestStreamableHTTPResourceReadOmitsSecrets`, `TestStreamableHTTPArgumentPrivacyAbsentFromLogs`; commit 508a34a).

---

## Scope Boundary

```
IN SCOPE (may modify):
  - mcp-server registry load/validation and readiness wiring
  - MCP resource handlers (guardrail://config projection or removal)
  - MCP argument/audit/log redaction paths
  - Tests: registry fail-closed, resource confidentiality, argument privacy
  - Config load boundary for registry/verifier-key material

OUT OF SCOPE (DO NOT TOUCH):
  - Principal/role/resource intersection enforcement (SPRINT-A1 / R16-06–08)
  - Credential lifecycle/rotation APIs (SPRINT-A2 / R16-09)
  - Deployment profiles, TLS, Compose defaults (SPRINT-A3 / R17-01–05)
  - CI workflow gates and matrix (SPRINT-A4 / R19)
  - Spec 15 bilateral / Spec 14 checker (Gate 4; not this phase)
  - History rewrites, production credentials, production databases
```

---

## Numbered Tasks

### Task 1 — Registry integrity fail-closed (R16-03)

**Action:** Distinguish absent registry (temporary, explicitly gated legacy
migration mode) from configured malformed, unreadable, empty, or invalid
registry. Reject startup or all protected traffic for the latter. Reject
unknown scopes, duplicate credential IDs/verifiers, invalid records, and
unusable verifier-key material at the load boundary. Ensure unregistered
credentials cannot inherit authority. Design and test registry-only cutover
without reinstating legacy access when the legacy MCP key is empty.

**Acceptance criteria:**
- Configured JSON malformed / file unreadable / empty records → readiness non-2xx or protected requests deny on web and MCP.
- Legacy key cannot POST `/api/rules` or call `guardrail_project_delete` when registry is configured-invalid.
- After approved legacy-key removal with a valid registry, registered scoped callers work; absent legacy key authenticates nobody.
- No silent downgrade to unrestricted legacy on any failure path.

**Expected test commands** (from `mcp-server/`):
```sh
go test ./internal/auth ./internal/web ./internal/mcp -run 'TestRegistry|TestLoadFromSourcesNoConfigIsNil' -count=1
go test ./internal/auth ./internal/web ./internal/mcp -count=1
```

### Task 2 — MCP resource confidentiality (R16-04)

**Action:** Replace whole-config serialization with an explicitly reviewed
non-secret projection (or remove the resource). Authorize each resource read
with the same principal decision model as tools (deny unknown resources before
any read side effect). Do not leak secrets in errors, diagnostics, or audit.

**Acceptance criteria:**
- Legacy or least-privilege registered key reading `guardrail://config` never sees DB password, API key, JWT secret, verifier key, or raw credentials.
- Unauthorized reads yield a stable denial (no partial config, no stack-trace leak).
- Resource authorization denies unknown resources before read side effects.

**Expected test commands:**
```sh
go test ./internal/mcp -run 'TestResource|TestGuardrailConfig|TestConfigProjection' -count=1
go test ./internal/auth ./internal/web ./internal/mcp -count=1
```

### Task 3 — Argument privacy (R16-05)

**Action:** Remove full MCP argument-map logging. Record only approved
operation name, request ID, credential/principal ID if verified, safe resource
identifier, reason, outcome, and policy version. Redact payloads and nested
secrets in log, audit, and error paths.

**Acceptance criteria:**
- Distinctive fake secret planted at any argument nesting level is omitted from log capture, response, metrics, and exported evidence.
- Error paths and startup failure output are tested, not only success paths.
- Attribution uses opaque IDs only (no raw keys or reversible forms).

**Expected test commands:**
```sh
go test ./internal/mcp ./internal/auth -run 'TestArgumentPrivacy|TestRedact|TestLog' -count=1
go test ./internal/... -count=1
```

### Task 4 — Cross-cutting negative/positive proof (R16-03–05)

**Action:** Add executable positive and negative cases through real web
middleware and StreamableHTTP tool/resource calls. Record owner decision and
source review. Run Secret Validation (full-history Gitleaks + env-file job) on
the change and record finding disposition (do not suppress real secrets).

**Acceptance criteria:**
- Named negative cases deny with stable errors and zero protected side effects.
- Full-history Gitleaks/env-file job run recorded; any real finding triggers incident handling, not blanket allowlist.
- No production credential or production database used.

**Expected test commands:**
```sh
go test ./internal/auth ./internal/web ./internal/mcp -count=1
go test ./internal/... -count=1
go vet ./...
# Plus CI Secret Validation (Gitleaks full history + env-file job) on the PR/commit
```

---

## Dependencies

| Direction | Sprint / Requirement | Relationship |
|-----------|----------------------|--------------|
| Requires | Spec 16 Gate 0 (R16-01 inventory, R16-02 scan recon) | Exposure/permission matrix and scan disposition inform fail-closed design |
| Blocks | SPRINT-A1 (R16-06–08) | Identity/audit cannot bind until registry and secrecy are fail-closed |
| Blocks | SPRINT-A2 (R16-09) | Lifecycle ops need a registry that rejects invalid state |
| Aligns | Spec 17 R17-07–08, Spec 18 R18-10, Spec 19 R19-05–06 | Same fail-closed and secrecy contract |

---

## Rollback Notes

- Roll back only to the last known-good **restrictive** registry load behavior — never to unrestricted legacy fallback.
- If a change reintroduces legacy authority or secret disclosure, revert the change (`git checkout HEAD -- <files>`) and keep protected traffic denied until fixed.
- Do not restore full-config resource serialization or full argument logging as a “temporary” rollback.
- If registry state is unknown after a failed deploy, deny protected traffic rather than serving with a nil/unrestricted registry.

---

## Do Not Proceed If

**STOP this sprint (and do not continue to SPRINT-A1) if any of the following is true:**

- A configured-invalid registry can still authorize any web or MCP effect.
- Any secret marker (DB password, API key, JWT secret, verifier key, raw credential, nested fake secret) appears in resource output, logs, audit, errors, or metrics.
- Legacy fallback can mutate state (`POST /api/rules`, `guardrail_project_delete`, or equivalent) after a configured registry is present.
- Positive/negative tests only exercise registry unit helpers — not real web middleware and StreamableHTTP MCP paths.
- A genuine credential finding from Secret Validation is suppressed or blanket-allowlisted without classification and rotation.
- Test commands fail, or Windows-only failures (`fcntl`, symlink privilege) are silently treated as PASS without a documented supported-runner rerun plan.

---

## Acceptance Criteria Summary

| # | Criterion | Test | Pass Condition |
|---|-----------|------|----------------|
| 1 | Registry fail-closed (R16-03) | Malformed/unreadable/empty/duplicate/invalid fixtures via web + MCP | Readiness fails or protected traffic denies; no legacy fallback |
| 2 | No authority inheritance | Unregistered vs registered credential cases | Unregistered cannot act as registered |
| 3 | Resource secrecy (R16-04) | Read `guardrail://config` with legacy and least-privilege keys | Non-secret projection only; unauthorized → stable deny |
| 4 | Argument privacy (R16-05) | Nested fake-secret in tool args | Absent from logs, responses, metrics, evidence |
| 5 | Real-transport proof | `go test ./internal/auth ./internal/web ./internal/mcp` | All R16-03–05 cases pass; no side effects on deny |
| 6 | Scan disposition | Gitleaks + env-file CI job | Findings classified; no real secret suppressed |

---

## Reference

- [Spec 16](../specs/16-authentication-authorization-and-evidence-remediation.md) — Gate 1, R16-03–05
- [Spec 17](../specs/17-deployment-tls-and-secret-boundary.md) — R17-07–08
- [Spec 18](../specs/18-migration-startup-and-readiness.md) — R18-10
- [Spec 19](../specs/19-phase0-ci-security-gates.md) — R19-05–06
- Template basis: [SPRINT_TEMPLATE.md](../archive/sprints/SPRINT_TEMPLATE.md) (adapted for security remediation)

---

**Created:** 2026-10-04
**Version:** 1.1
**Status:** PARTIALLY IMPLEMENTED (reconciled 2026-10-04 against commit 1c838f4)
