# OpenSpec: API-Key Scopes and Named Roles

**Status:** Proposed; owner decision recorded 2026-10-04: use **both** scoped
API keys and named roles. **Implementation status (2026-10-16): substantially
implemented and exercised on both transports (see Sec. 9); the role matrix itself
still awaits owner acceptance.**
**Priority:** Critical — prerequisite to destructive/admin operations,
per-tool access hooks, and Phase 0 exit.
**Related:** `docs/specs/AUTH-01-mcp-endpoint-auth.md`,
`docs/specs/09-system-roadmap-and-phase-gates.md` Phase 0, and
`docs/platform-current-state.md` Sec. Sec. 3, 5.

## 1. Problem

The service currently has two static bearer secrets (`MCP_API_KEY` and
`IDE_API_KEY`). The MCP endpoint checks the MCP key, while web middleware
recognizes MCP/IDE keys and stores a key type/hash in request context. The
server does not resolve a caller principal or apply named roles to MCP tool
operations. Thus possession of a valid key can authorize operations far
beyond the caller’s intended purpose; confirmation flags only express intent
and do not authorize a caller.

This spec defines the authorization contract before implementation. It does
not claim that principals, scopes, roles, or role administration exist today.

## 2. Goals and non-goals

### Goals

1. Keep API-key scopes and named roles as separate, composable controls.
2. Compute effective permission by **intersection**: an action is allowed
   only when both the key scope and the principal’s role grant it.
3. Apply one authorization decision model to MCP and REST operations while
   preserving protocol-appropriate error responses.
4. Protect destructive, administrative, security-sensitive, and data-reading
   operations with least privilege.
5. Support gradual migration from the existing MCP/IDE keys without silently
   granting administrative authority to legacy credentials.
6. Make key issuance, rotation, revocation, role assignment, and audit
   attributable to an operator identity.

### Non-goals

- Treating `confirm_override`, `confirmed`, or free-text `reason` as
  authorization. These remain explicit-intent/confirmation signals only.
- Replacing network isolation, TLS, rate limiting, or SSRF controls.
- Requiring OAuth/OIDC in the first implementation. The principal/key model
  must permit a later identity-provider adapter without making it a prerequisite.
- Persisting raw API-key values. Only a one-way keyed verifier or equivalent
  secure representation may be stored.

## 3. Terminology

- **Principal:** stable service identity (human, CI workload, or integration)
  with a role assignment and audit identity.
- **Key credential:** secret bearer credential mapped to exactly one principal
  and one or more scopes.
- **Scope:** maximum API surface a key may invoke (for example MCP tools,
  read-only REST, IDE validation, or administration).
- **Role:** maximum set of operations a principal may perform (for example
  reader, developer, security-operator, administrator).
- **Effective permission:** intersection of the key’s scopes and the
  principal’s role permissions, further restricted by resource/project policy.
- **Legacy key:** current static MCP_API_KEY or IDE_API_KEY configuration
  before per-principal credential records are enabled.

## 4. Requirements

### 4.1 Principal and credential model

1. Each credential MUST resolve to a principal ID, credential ID, scopes,
   issue time, optional expiry, and revocation state. The secret value MUST
   never be returned by read/list APIs or logged.
2. Credential verification MUST use a constant-time comparison or a vetted
   password/token verifier designed for high-entropy secrets. Logs, metrics,
   and error messages MUST NOT include raw keys or reversible key material.
3. Multiple credentials MAY map to one principal to support rotation. A new
   credential MUST be verifiable before the old one is revoked; rotation MUST
   not require sharing a key across principals.
4. Revocation MUST take effect within a documented bound and MUST be testable.
5. Missing, malformed, expired, revoked, unknown, or unverifiable credentials
   MUST be rejected before route/tool side effects.

### 4.2 Scope model

1. Scopes MUST be named and versioned, not inferred from a key prefix or
   caller-supplied request field.
2. Initial scope candidates are `mcp:read`, `mcp:validate`, `mcp:mutate`,
   `rest:read`, `rest:write`, `ide:validate`, and `admin:manage`. The
   implementation MUST publish the final catalog before enabling scoped keys.
3. A credential with a scope MUST NOT gain permissions from a broader role.
   A principal with a role MUST NOT gain access outside the credential’s
   scopes.
4. Public routes MUST be explicitly enumerated and tested. An unlisted route
   is authenticated by default. Static-file extension matching MUST never
   exempt `/api/` routes or unsafe HTTP methods.
5. Key type labels such as `mcp` and `ide` are not, by themselves, scopes;
   legacy mapping MUST be explicit, documented, and temporary.

### 4.3 Role and permission model

1. Roles MUST map to explicit permission identifiers; code MUST authorize by
   permission, not by comparing role-name strings ad hoc.
2. Initial role candidates are `reader`, `developer`, `security-operator`,
   and `administrator`. The final mapping MUST be a reviewed table of
   operations and permissions, including resource constraints.
3. Administrative permissions MUST be required for at minimum: forcing agent
   state, changing policy/configuration, issuing/revoking keys, assigning
   roles, deleting projects, overriding halt decisions, and changing
   deployment/security settings.
4. Mutating team operations MUST require an authenticated principal with the
   corresponding project/team permission. A bearer key alone MUST NOT imply
   admin.
5. Resource/project ownership or team scope MUST be checked after coarse
   role/scope authorization and before a mutation.
6. Confirmation fields such as `confirmed=true` MAY remain as a second-step
   safety control but MUST NOT substitute for authorization.

### 4.4 Authorization decision and failure behavior

For principal `P`, credential `K`, action `A`, and resource `R`:

```text
allow(P, K, A, R) =
    authenticated(K)
    AND not_revoked(K)
    AND scope_allows(K.scopes, A)
    AND role_allows(P.role, A)
    AND resource_policy_allows(P, A, R)
```

1. Any missing/unknown authorization input MUST deny by default.
2. Denials MUST be distinguishable as unauthenticated (401) versus
   authenticated but forbidden (403) at HTTP boundaries; MCP responses MUST
   use a stable permission-denied error shape without leaking whether a
   protected resource exists.
3. Authorization MUST be evaluated server-side on every request, not only in
   the UI/client or at key-issuance time.
4. Decisions MUST record principal ID, credential ID (not secret), action,
   resource identifier where safe, decision, reason code, and policy version.
   Audit failure behavior MUST be specified per operation; security-sensitive
   mutations MUST fail closed if a required audit record cannot be written.
5. Public-route exceptions MUST be method-specific and route-specific; CORS
   preflight is not authorization for the corresponding request.

### 4.5 Legacy migration and compatibility

1. The current `MCP_API_KEY` and `IDE_API_KEY` MUST be mapped through an
   explicit migration table to named principals, roles, and least-privilege
   scopes. Neither key may silently map to unrestricted administrator.
2. The migration MUST define a transition period, warning/metrics for legacy
   credential use, rotation procedure, rollback procedure, and removal date.
3. If the deployment cannot distinguish clients sharing one legacy key, the
   limitation MUST be explicit: those clients share a principal and therefore
   cannot receive distinct role assignments until separate credentials are
   issued.
4. Existing clients MUST receive a tested migration path. A rejected request
   must return actionable guidance without revealing credentials.
5. The API-key identity hash currently truncated for metrics MUST not be used
   as a principal or credential identity. Identity storage requires a
   collision-resistant keyed digest or opaque credential ID.

## 5. Acceptance criteria

1. Unit tests cover every scope × role combination for the permission matrix,
   including explicit deny cases and resource/team boundaries.
2. Integration tests prove the same action is denied when either the key
   scope or role is insufficient, and allowed only when both permit it.
3. Tests prove legacy MCP/IDE keys map only to the reviewed migration roles
   and cannot invoke admin-only operations unless explicitly migrated to an
   authorized principal.
4. Tests prove missing, malformed, expired, revoked, and rotated credentials
   are rejected/accepted according to policy, with rotation overlap bounded
   and revocation effective within the documented interval.
5. REST tests cover public-route allowlists, method restrictions, CORS
   preflight, and attempts to bypass auth with suffixes, path normalization,
   alternate methods, and route-pattern mismatches.
6. MCP integration tests cover authorized and denied tool calls through the
   real `/mcp` transport, including `force_agent_state`, team mutations,
   project deletion, and policy/config mutation.
7. Audit records contain principal/action/outcome/reason and never contain
   the raw credential; tests seed a recognizable fake secret and assert it
   is absent from logs, metrics, and evidence.
8. A compatibility test exercises documented client migration with both
   legacy and scoped credentials; the deprecation/removal date is enforced by
   a testable configuration switch.

## 6. Initial permission matrix (proposal for review)

| Operation class | reader | developer | security-operator | administrator |
|-----------------|--------|-----------|-------------------|---------------|
| Read public status/docs | allow by route policy | allow | allow | allow |
| Read protected project/team data | deny by default | project-scoped | project-scoped | allow |
| Run validation/check tools | deny unless key has `mcp:validate` | allow within key scope | allow | allow |
| Mutate team/project state | deny | project/team-scoped | policy-dependent | allow |
| Change security policy or credentials | deny | deny | scoped policy management | allow |
| Force agent state / bypass transition rules | deny | deny | deny unless an explicit delegated permission is approved | allow + reason + confirmation |
| Delete project / destructive admin action | deny | deny | deny | allow + confirmation + audit |

This table is a proposal, not a shipped permission matrix. Before coding,
confirm whether a security-operator may manage policy without managing
credentials, and whether any developer role may mutate team assignments.

## 7. Migration plan

1. **Inventory:** enumerate routes/tools, side effects, existing auth bypasses,
   public exceptions, and current key consumers.
2. **Introduce data model:** add principal, role, credential, scope, and audit
   records plus migrations; do not change live authorization yet.
3. **Shadow evaluation:** resolve legacy keys to proposed principal/scope and
   emit decision metrics without granting new access. Compare observed use to
   the proposed matrix.
4. **Provision distinct credentials:** issue scoped keys per integration and
   update integrations; verify each with contract tests.
5. **Enforce:** turn on deny-by-default scope-role intersection; keep only
   explicitly approved public routes.
6. **Deprecate legacy keys:** measure zero use during the agreed window, then
   remove fallback and rotate remaining clients.
7. **Rollback:** define how to restore the prior credential set without
   re-enabling unrestricted admin access. Rollback MUST preserve auditability.

## 8. Open decisions and first-implementation defaults

The solo-maintainer proposed choices in
[Spec 16 Sec. 4](16-authentication-authorization-and-evidence-remediation.md)
answer the initial matrix and migration posture for an executor: no developer
team-assignment changes, security-operator policy write only under a distinct
key scope, no implicit legacy administrator, file-backed registry validated
before startup, 90-day key expiry, at most 24-hour rotation overlap and
60-second revocation target. These are **implementation targets**, not claims
that authorization is wired or that production rollout is approved. If a live
client cannot migrate under them, stop and revise the spec rather than grant a
silent exception.

For the first implementation, use `reader`, `developer`,
`security-operator`, and `administrator` with the restrictive matrix in
Spec 16 Sec. 4. Store the initial credential-to-principal and role/scope mapping
in an operator-owned versioned local registry; validate and atomically replace
the whole snapshot before use, with no permissive fallback on invalid config.
Postgres-backed administration is deferred until it has an equally testable
trust and migration boundary. Public routes and credential bounds are the
Spec 16 defaults. Legacy acceptance lasts **at most 30 days after enforcement
is enabled** and requires seven consecutive days of zero observed use before
removal; an integration that cannot migrate stops rollout rather than extending
the window silently. Emergency disable must deny new calls within 60 seconds
by verified reload or process replacement. Destructive/admin operations block
if required audit persistence fails. These choices are proposed targets and
require the negative tests in Sec. 5 before deployment.

## 9. Implementation status

**Substantially implemented; owner acceptance of the role matrix still pending.**
The Sec. 9 text below was re-derived from live source on **2026-10-16** (`main@dccaee9`);
the prior version of this section described the pre-Spec-16 tree and is
superseded.

**Implemented and exercised (live-verified 2026-10-16):**

- **Principal/credential model (Sec. 4.1).** `mcp-server/internal/auth/registry.go`
  loads credential/principal IDs, scopes, expiry and revocation state; each
  record stores only a keyed one-way HMAC-SHA256 verifier, never the secret.
  `Resolve` uses a constant-time comparison; the secret is never returned or
  logged (`TestRegistryNeverExposesSecret`, `TestRegistryUsesConstantTimeVerifier`).
- **Scope catalog (Sec. 4.2).** `mcp-server/internal/auth/authz.go` publishes
  `mcp:read`, `mcp:validate`, `mcp:mutate`, `rest:read`, `rest:write`,
  `ide:validate`, `admin:manage`; an unknown scope on a credential denies
  (`TestDecideScopeRoleResourceIntersection` case "unknown scope on credential
  denies"). Public routes are enumerated in the Spec 12 exposure registry, and
  the static-asset rule is method- and `/api/`-constrained (no suffix bypass).
- **Role/permission model (Sec. 4.3).** Named roles `reader`, `developer`,
  `security-operator`, `administrator` map to an explicit permission-class
  table (`roleActionAllowed`); authorization is by permission class, not
  role-name string comparison.
- **Authorization decision (Sec. 4.4).** `auth.Decide` implements exactly
  `authenticated AND not_revoked AND scope_allows AND role_allows AND
  resource_allows`; missing/unknown inputs deny by default. It is applied on
  **both** transports before any effect: REST at
  `mcp-server/internal/web/middleware.go` (`auth.Decide` on the request path)
  and MCP at `mcp-server/internal/mcp/server.go` `authorizeToolCall` (before
  tool dispatch), plus the resource-read gate (Spec 16 R16-04). Denials are
  401 (unauthenticated) / 403 (forbidden) at HTTP and a stable non-leaking
  permission error on MCP. Security-sensitive mutations fail closed when a
  required durable audit record cannot be written
  (`TestAuthzAdminMutationFailsClosedWithoutAudit`).
- **Legacy migration/containment (Sec. 4.5).** `MCP_API_KEY`/`IDE_API_KEY` map to
  explicit constrained legacy principals (`auth.LegacyPrincipal`); a legacy key
  is confined to a named read/validate table and can never reach admin
  (`TestLegacyMCPCallAllowedNamedOnly`, `TestMatchLegacyRouteNoWildcards`,
  `TestAPIKeyAuth_UnregisteredCredentialDeniedPrivilegedAction`). The truncated
  `hashAPIKey` is retained only as a non-authoritative log/rate-limit
  correlation value, never as identity.
- **Fail-closed registry load.** A configured-but-unloadable registry denies
  all protected traffic on both transports — there is no legacy fallback
  (`TestLoadFromSourcesConfiguredInvalidFailsClosed`,
  `TestStreamableHTTPConfiguredBrokenRegistryDeniesAll`, the web `regErr`
  branch). The prior Sec. 9 claim that an invalid registry "can fall back to legacy
  access" is no longer true.
- **Removal-date switch (Sec. 4.5.2 / acceptance criterion 8) — added 2026-10-16.**
  `LEGACY_ACCEPTANCE_UNTIL` (RFC3339) is a **testable** switch: before the
  deadline the legacy key keeps its surface; once it has passed the key is
  refused on both transports regardless of registry state, while a registered
  credential is unaffected; an unset value never retires legacy keys; a
  malformed value fails closed (503 on REST, startup failure on MCP).
  `TestLegacyAcceptanceDeadline`, `TestLegacyAcceptanceExpired`,
  `TestValidateRejectsMalformedLegacyDeadline`,
  `TestAPIKeyAuth_LegacyAcceptanceRemovalSwitch`,
  `TestRequireBearerWithPolicyLegacyRemovalSwitch`; CI row
  `security-matrix-legacy-acceptance`; mutation-kill
  `legacy-acceptance-removal-switch-ignored` (kills).
- **Legacy-use observability (Sec. 4.5.2) — added 2026-10-16.** Accepted legacy use
  is counted by `guardrail_legacy_credential_uses_total{surface,class}`, so the
  migration window is measured (a zero rate gates removal), not assumed.

**Still open (honest limits, not claimed here):**

- **Owner acceptance of the role matrix (Sec. 6/Sec. 8).** The restrictive matrix and
  the security-operator/developer postures remain *proposed*; this spec must be
  accepted by the owner before the matrix is treated as ratified.
- **Postgres-backed administration** is deferred (Sec. 8): the credential store is
  the operator-owned versioned local registry.
- **Sec. 4.5.2 transition-period mechanics** beyond the removal switch (warning/
  metric for legacy use is implemented; the documented seven-consecutive-day
  zero-use precondition is an operator procedure, not a code gate).
- **R16-06/R16-07** tool-path role/resource enforcement and legacy containment
  are implemented and exercised (Spec 16); R16-10/R16-11 remain unmet by design
  (no second product consumer).

Acceptance criteria 1–8 in Sec. 5 are now covered by the suites above; criterion 8
(the deprecation/removal switch) is the mechanism added this pass.