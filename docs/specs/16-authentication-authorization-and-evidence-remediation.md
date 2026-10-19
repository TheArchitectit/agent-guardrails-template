# OpenSpec 16: Authentication, authorization, and evidence remediation

**Status:** PARTIAL — Gates 1 and 2 exercised; Gate 3 partially exercised (see
§6 Implementation status, 2026-10-18). Source snapshot: `178d076`; every claim
below has been re-verified against the live tree and the stale ones are marked.
This remains a release gate, not a claim that Spec 11, 12, 14, or 15 has
shipped. Owner approval is still required for the permission matrix, public
routes, lifecycle bounds, and cross-product rollout.
**Related:** [AUTH-01](AUTH-01-mcp-endpoint-auth.md) (shipped bearer gate),
[11](11-authorization-scopes-and-roles.md) (proposed intersection),
[12](12-web-exposure-boundary.md) (proposed exposure registry),
[14](14-optional-oap-guardrail-checker/spec.md) (conditional checker),
[15](15-secure-cross-product-method/spec.md) (proposed bilateral evidence method).
This plan does not replace or silently mark any of those specs complete.

## 1. Source-evidenced baseline and discrepancies

| Severity / surface | Observed at snapshot | Required disposition |
|---|---|---|
| Critical: registered web credentials | `mcp-server/internal/web/middleware.go:115-130` stores principal ID, credential ID, and scopes, then calls `next` without a permission decision; registered keys can reach protected mutation handlers regardless of scopes or role. `internal/auth/registry.go:28-45,107-135` has no role or resource binding. | Gate each action by key scope AND principal role AND resource policy before effects. |
| Critical: MCP tools | `internal/mcp/server.go:388-415` accepts any resolved registry key or legacy MCP key and forwards to the MCP server without a principal context; `server.go:143-162,164-274` dispatches tools without a scope/role check. The confirmation in `internal/mcp/team_tool_project.go:128-146` checks intent, not authority. | Preserve verified identity into dispatch and authorize each tool/effect. |
| Critical: MCP resources/config | `internal/mcp/resource_registration.go:21-35,101-105` registers `guardrail://config` and marshals all exported config fields; `internal/config/config.go:45-50,77-92` includes DB password, MCP/IDE keys, verifier key, JWT secret. Bearer-only access is not redaction or resource authorization. | Stop serving secrets; apply per-resource read policy before invoking handlers. |
| Critical: invalid configured registry | Web `middleware.go:190-204` and MCP `server.go:363-374` log load errors and set registry nil. Web then omits its registry-enabled legacy restriction (`middleware.go:150-168`); MCP retains legacy access. `internal/config/config.go:160-166` checks only verifier-key length when configured, not JSON/file validity at startup. | Invalid/empty configured registry must not revert to unrestricted legacy mode. |
| High: legacy-safe web surface | `middleware.go:173-188` allows **any** GET/HEAD/OPTIONS path plus POST `/api/ingest`, `/api/ingest/sync`, `/api/updates/check` and any `/ide/` path when registry is enabled; public route exemptions at `:22-90` occur before authentication. `:206-269` duplicates exemption rules for rate limiting. | Reviewed method-and-route allowlist; narrow public set per Spec 12; no wildcard safe-method exemption. |
| High: registry/lifecycle | `internal/auth/registry.go:62-93` validates IDs and verifier format but not scope vocabulary or duplicate credential IDs; `:138-165` loads JSON/file, returning nil for zero records. `middleware.go:18-20` and MCP `server.go:367-375` load once at construction/Serve. Expiry/revocation are checked at resolution (`registry.go:123-129`), but no issuance, rotation, atomic reload, or propagation bound is visible in these paths. | Validated catalog, unique IDs, managed issuance/rotation/reload, tested revocation bound. |
| High: audit and sensitive arguments | Web audit calls use `api_key_hash`, e.g. `internal/web/handlers_rules.go:92,119,150,212`; `middleware.go:307-315` truncates SHA-256 to eight bytes, not an identity. `internal/web/handlers_projects.go:63-119` has no project-change audit call. MCP `internal/mcp/server.go:161-162` logs entire tool argument maps before dispatch, which may contain secrets. | Stable opaque IDs, redacted structured decision/effect audit, project audit, no raw argument logging. |
| Proposed cross-product controls | Spec 14 `gr-oap-01`–`07` and Spec 15 `gr-xp-01`–`06` propose host-context separation, signatures, trust, replay, revocation and bilateral verification. Current registry HMAC verification is **not** artifact signing or peer trust; `internal/mcp/server.go:363-415` is only bearer authentication. | Separate implementation and conformance gates; never claim signed evidence or OAP enforcement from the registry alone. |
| Coverage / CI | `internal/mcp/auth_test.go:14-116` covers missing/wrong bearer and initialize, but not authorized versus denied tool calls with registered/legacy keys. `internal/web/middleware_registry_test.go:66-125` asserts registered POST succeeds even with scope `mcp` (`:22-27`) and legacy unrestricted POST when registry absent. `internal/auth/registry_test.go:10-17` embeds a test verifier key. `.github/workflows/secret-validation.yml:13-27` runs Gitleaks over full checkout history; `:85-125` warns on patterns but does not fail. Latest Secret Validation failure on a test verifier key is **reported by the audit request**, not reproduced in this document. | Reconcile exact scan finding and tests before a green release claim. |

**Shipped, narrowly:** Static bearer authentication for `/mcp` and 401 on missing
credentials (`server.go:378-415`; AUTH-01); web bearer gate with public skips;
HMAC-backed registry resolution, expiry and revoked checks (`registry.go:53-135`).
**Not shipped:** enforced scope/role/resource intersection, principal-bearing MCP
calls, secure config-resource redaction, managed key lifecycle, signed peer
artifacts/trust roots/replay defense, or an exercised OAP checker. Presence of
proposed specifications, registered tools, or passing isolated registry tests is
not runtime enforcement evidence.

## 2. Ordered dependency and release gates

Implement in order. Each gate requires executable positive and negative tests,
source review, and a recorded owner decision; no production expansion or
cross-product PASS claim while an earlier security gate is red. Deny missing
permission metadata rather than treating it as an allow.

### Gate 0 — freeze exposure and reconcile evidence (blocks all later gates)

**R16-01 — Inventory.** Publish an exact method/path exposure registry and a
complete MCP tool/resource operation-to-permission matrix, including conditional
vision tools and `guardrail://config`; mark public exceptions by named consumer.
Cross-check actual route registrations and dispatch against Spec 11/12, decide
which legacy clients require which access, and record default deny for unmapped
routes/tools/resources.

- **WHEN** a route, tool, resource, or alternate method is newly registered or
  unmatched **THEN** CI fails the inventory check or runtime denies it; a
  filename suffix, `OPTIONS`, or an arbitrary GET does not confer access.

**R16-02 — Secret-scan reconciliation.** Obtain the failing Secret Validation
run's detector, location, commit/history scope and severity; determine whether
the test verifier key is a synthetic false positive or a real exposure. Resolve
via safe fixture replacement or narrowly justified scanner handling with review;
never suppress a genuine credential, print it in CI, rewrite history without
separate authorization, or stage the existing untracked ZIP. Full-history
Gitleaks failure remains a release blocker until a clean rerun or approved
finding disposition, regardless of the warning-only regex job.

- **WHEN** a scanner flags test material **THEN** the finding has documented
  classification and rerun evidence; any real secret triggers incident handling
  and rotation, not a blanket allowlist.

### Gate 1 — fail closed and stop immediate disclosure (depends on Gate 0)

**R16-03 — Configured-registry integrity.** Distinguish absent registry
(temporary, explicitly gated legacy migration mode) from configured malformed,
unreadable, empty, or invalid registry. Reject startup or all protected traffic
for the latter; never silently downgrade to unrestricted legacy. Reject unknown
scopes, duplicate credential IDs/verifiers, invalid records, and unusable
verifier-key material at the load boundary. An unregistered credential cannot
inherit authority from a registered one. MCP currently rejects even registered
keys when the configured legacy MCP key is empty (`server.go:395-400`); design
and test a registry-only cutover without silently reinstating legacy access.

- **WHEN** configured JSON is malformed, a file cannot be read, or records are
  empty **THEN** readiness fails or protected requests deny on **both** web and
  MCP; a legacy key cannot POST `/api/rules` or call `guardrail_project_delete`.
- **WHEN** legacy MCP key removal is approved and a valid registry is loaded
  **THEN** registered scoped callers can use their permitted MCP operations,
  while an absent legacy key still authenticates nobody.

**R16-04 — MCP resource confidentiality.** Replace whole-config serialization
with an explicitly reviewed non-secret projection (or remove the resource), and
authorize each resource read using the same principal decision model as tools.
Deny unknown resources before any read side effect; do not leak secrets in error
messages, diagnostics, or audit.

- **WHEN** either a legacy or least-privilege registered key reads
  `guardrail://config` **THEN** no DB password, API key, JWT secret, verifier key,
  or raw credential is returned; unauthorized reads yield a stable denial.

**R16-05 — Argument privacy.** Remove full MCP argument-map logging. Record only
approved operation name, request ID, credential/principal ID if verified,
resource identifier if safe, reason, outcome, and policy version; redact payloads
and nested secrets in log/audit/error paths.

- **WHEN** a tool argument contains a distinctive fake secret at any nesting
  level **THEN** log capture, response, metrics, and exported evidence omit it.

### Gate 2 — bind identity and enforce decisions (depends on Gate 1)

**R16-06 — Shared principal and permission contract.** Resolve a credential to
server-controlled principal ID, credential ID, reviewed scopes, role grants,
resource/project membership and status; pass it through web and MCP request
contexts without accepting identity/role/tenant from tool arguments or content.
Apply `authenticated AND scope_allows AND role_allows AND resource_allows` before
every effect, including conditional handlers. Confirmation flags remain intent
checks after authorization. Missing/unknown role, resource, scope, or operation
denies; 401 for unauthenticated HTTP, 403 for authenticated forbidden HTTP,
stable non-leaking MCP permission error.

- **WHEN** a registered `mcp:read`-only key is paired with admin role, or a
  `mcp:mutate` key with reader role **THEN** project deletion and force-state are
  denied without side effects; only explicit scope + role + project grant permits
  the intended action, still subject to confirmation.
- **WHEN** one caller is allowed for project A but not B **THEN** identical tool
  or REST mutation on B is denied even with `confirmed=true`.

**R16-07 — Legacy containment and web exposure.** Replace arbitrary safe-method
and `/ide/` prefix access with the approved method/path/action table; apply the
same privilege model to legacy MCP key calls. If a legacy principal cannot be
attributed or constrained, deny privileged and mutating actions rather than
restoring the old unrestricted behavior. Align auth and rate-limit public-route
logic with Spec 12 and test path normalization, suffixes and preflight separately.

- **WHEN** a legacy key requests an unlisted GET/HEAD/OPTIONS route, a protected
  resource, POST `/ide/validate/file`, or an MCP mutation not explicitly granted
  **THEN** it is denied under the approved migration policy; a public preflight
  never authorizes its corresponding effect.

**R16-08 — Decision/effect audit.** Record principal and opaque credential ID,
action, safe resource/project ID, outcome, reason code, and policy version for
both transports, including project create/update/delete and denied requests;
never use the truncated `hashAPIKey` as identity. Security-sensitive mutations
fail closed if required durable audit cannot be recorded; state the durability
boundary and test failures before and after authorization.

- **WHEN** project deletion is allowed, denied, or audit persistence fails
  **THEN** attributable records exist for completed attempts and no deletion
  occurs without the required durable record; raw key and full args are absent.

### Gate 3 — lifecycle and migration (depends on Gate 2)

**R16-09 — Credential operations and bounds.** Define approved key generation,
issuance, distinct principal assignment, expiry, overlap/rotation, revocation,
atomic reload or immutable deployment replacement, verifier-key rotation and
operator audit. Set and publish a measured maximum revocation propagation delay;
reject a bad replacement without restoring privileged access. Do not claim a
lifecycle API exists until one is implemented and protected.

- **WHEN** a key is revoked during an active deployment **THEN** new web and MCP
  calls deny within the approved bound; overlapping replacement works only for
  its intended scopes and role. A failed reload never enlarges access.

### Gate 4 — optional external evidence (depends on Gates 1–3)

**R16-10 — Spec 15 bilateral contract.** Only after both products implement
out-of-band trust roots, direction/audience-scoped identity, detached signed
canonical evidence, bound subject/tenant/policy/request digests, freshness,
nonce/idempotency replay state, revocation and separate transport/artifact/host
policy checks may a cross-product artifact be represented as verified. The
first candidate contract is AIGGP's `contract-v2.md`: one strict body/signature
shape, full key ID and cross-language vectors; both conflicting v1 shapes deny.
Start DevGate→Go OAP observe-only, and test a reverse direction only after a
real second consumer exists. Receipts and signatures are not effect grants.
Registry HMAC verifiers alone do not satisfy this requirement.

- **WHEN** a signed artifact is altered, replayed, expired, wrong-audience,
  wrong-tenant, or signed by a revoked/unknown key **THEN** the receiving product
  records distinct non-PASS reasons and performs no unauthorized effect.

**R16-11 — Spec 14 conditional checker.** Implement only with a real consumer,
exercised production request path and host readiness evidence. Preserve typed
host context as asserted input; checker PASS never overrides host deny; absent,
skipped, timed-out, or library-only checks remain non-PASS. No host identity,
roles, policy or effect mutation from checker results.

- **WHEN** host authorization denies despite checker PASS, or required checker
  is unavailable **THEN** no effect runs and evidence correctly reports the
  non-enforced status; without the prerequisites this gate remains proposed.

## 3. Acceptance and failure criteria

These are **future** commands after tests have been added, not evidence of a
passing implementation at this snapshot. Run from `mcp-server/`:

```sh
go test ./internal/auth ./internal/web ./internal/mcp -count=1
go test ./internal/... -count=1
```

First command must prove all R16-03–09 negative/positive cases through real web
middleware and StreamableHTTP tool/resource calls, not only registry unit tests;
second must have no regression. Run CI **Secret Validation** (full-history
Gitleaks and env-file job) on the change; record the exact run and finding
resolution. For R16-10–11 additionally run approved two-product contract and
host-effect tests in both directions when an integration actually exists; until
then they remain unmet and cross-product/OAP enforcement must not be advertised.
The existing UCS03 Podman runner handles trusted `main` validation only
(`.github/workflows/team-validation-fleet.yml:15-20`); use it for added
same-commit Linux race/integration evidence only after proving per-repo job
isolation and no cross-project credential access. Fork/PR code remains on
hosted runners. A runner label or green build without the named negative tests
is not security acceptance.
**Failure:** any unauthorized effect, leaked config field or argument, missing
project/decision audit, unexpected legacy fallback, stale revocation beyond the
bound, or unresolved real scan finding blocks rollout even if `go test` passes.
No production credential or production database is needed for these tests.

**Snapshot verification (Windows host, documentation-only change):**
`go test ./internal/auth ./internal/web ./internal/mcp -run 'TestRegistry|TestLoadFromSourcesNoConfigIsNil|TestAPIKeyAuth_|TestRequireBearer|TestMCPEndpointBehindBearer' -count=1`
passed all three packages. The broader focused package command above passed
`internal/auth` and `internal/web` but failed `internal/mcp`:
`TestIntegrationJSONParsing` needs Python `fcntl`, and `TestPathWithinScope`
could not create a Windows symlink without the required privilege. Neither is
an authorization acceptance pass; rerun the full gate in a supported environment
and investigate any remaining failures before release. Secret Validation was
not rerun here; its latest reported failure needs the Gate 0 reconciliation.

## 4. Solo-maintainer decisions for the first implementation

These are proposed implementation defaults chosen by the repository owner to
unblock a bounded executor. They do not state that enforcement has shipped.
Changes to these defaults require a new reviewed spec revision and negative
tests, not a runtime permissive fallback.

- **Role/scope matrix:** `reader` can read authenticated, project-scoped data;
  `developer` can run validation and non-destructive project-owned writes but
  cannot assign team membership, change security policy, force agent state or
  delete projects; `security-operator` can change policy only with a separate
  `policy:write` credential scope and cannot issue keys, assign roles or force
  state; `administrator` needs a matching admin key scope, explicit resource
  grant, confirmation where specified and durable audit for destructive/admin
  effects. An empty/unknown scope, role or resource denies. A role never widens
  a key scope; a key never widens a role.
- **Legacy mapping:** `MCP_API_KEY` is limited to named read/validate MCP tools;
  `IDE_API_KEY` to named IDE validation routes. Neither may invoke mutations,
  resources containing config/secrets, team assignment, forced state or admin
  tools. A shared legacy key represents one *legacy workload*, not individual
  humans. Operators migrate each consumer to a distinct registered key before
  removing the legacy key. The transition lasts at most 30 days after
  enforcement begins and requires seven consecutive days of zero measured use
  before removal. No implicit admin mapping or perpetual fallback.
- **Credential authority:** use an operator-owned, versioned local registry
  initially, not a newly invented online IdP. Validate the complete file and
  role/scope catalog before listener startup; absent config can enter only a
  bounded documented legacy mode, while configured-empty/invalid/unreadable
  config is NOT_READY and refuses protected traffic. Reload replaces the whole
  verified snapshot atomically; never restores broader prior permissions on
  failure. No registry bytes or verifier key appear in resource reads/logs.
- **Lifecycle bounds:** new credentials expire within 90 days; planned rotation
  overlap is at most 24 hours; revocation and emergency disable must deny new
  requests on both transports within 60 seconds in the supported deployment.
  If the file-based reload cannot prove that bound, restart/replace the process
  before claiming revocation. The keyed verifier secret is generated out of
  band with at least 256 random bits; a character-count check alone is not an
  entropy guarantee. Rotate it through a separately tested full registry
  replacement, never by silently accepting old and new digests indefinitely.
- **Exposure:** only `GET /health/live`, `GET /health/ready` and `GET /version`
  are anonymous operational endpoints; safe-method web assets are public on
  explicit static paths only. `/metrics`, docs, and all API data are
  authenticated by default in production. Local bind defaults to loopback,
  production CORS uses explicit origins, and no forwarded client IP is trusted
  unless the operator supplies a named proxy CIDR. Preflight grants no action.
- **Audit outage:** security-sensitive mutations and credential/policy changes
  block before effect if their required audit intent cannot persist. A
  post-effect log is not equivalent. Remove or replace `guardrail://config`
  before admitting new peer credentials; no full-config serialization is safe.
- **Rollout:** begin with negative fixtures on a non-production instance,
  inventory the real clients, migrate one at a time, and make the above deny
  rules blocking only after the named tests run on the exact commit. Rollback
  to the last known-good *restrictive* snapshot; otherwise stop protected
  traffic. A green UCS03 build alone cannot approve the cutover.

## 5. Implementation status (2026-10-18)

Ledger of what is enforced in the tree versus still absent. This section exists
because the header above said "NOT implemented" long after most of it was, and
because the program's recurring defect is a requirement described as satisfied
while nothing exercises it.

### Gate 1 — exercised

| Req | State | Mechanism / evidence |
|---|---|---|
| R16-01 resource inventory | EXERCISED | `internal/mcp/resource_policy.go` declares one row per registered `guardrail://` URI; `scripts/check_resource_claims.py` asserts the documented count AND that every documented URI is registered (and vice versa) — a right count with a phantom URI is now a failure |
| R16-02 secret-scan reconciliation | EXERCISED | Secret Validation workflow is a required check; `.gitleaksignore` entries are commit-scoped, and every fix-forward in this program has removed the fixture value rather than adding an allowlist row |
| R16-03 registry fail-closed | EXERCISED | `TestRegistryConfiguredInvalidDeniesAllProtectedTraffic`, `TestStreamableHTTPConfiguredBrokenRegistryDeniesAll` |
| R16-04 resource confidentiality + authorization | EXERCISED | `guardrail://config` serves a reviewed projection; every resource read passes the R16-01 matrix and `auth.Decide` before the handler runs; `resource_policy_test.go` drives the real StreamableHTTP endpoint |
| R16-05 argument privacy | EXERCISED | `TestArgumentPrivacy`, `TestSecretLeakRedactionAcrossRealPaths` |

### Gate 2 — exercised

| Req | State | Mechanism / evidence |
|---|---|---|
| R16-06 principal + intersection | EXERCISED | `TestDecideScopeRoleResourceIntersection`, `TestAuthzScopeRoleResourceMatrix`, `TestStreamableHTTPToolsCallAuthorizesAndDenies` |
| R16-07 legacy containment / exposure | EXERCISED | `internal/web/exposure.go` is the single ordered rule table both middlewares classify with; `TestAPIKeyAuth_LegacyAcceptanceRemovalSwitch` |
| R16-08 decision/effect audit | EXERCISED (both transports) | web: `TestAuthzAdminMutationFailsClosedWithoutAudit`; MCP: `r16_08_audit_durability_test.go` drives the real StreamableHTTP tools/call path and asserts the project file survives a blocked delete and that the record carries principal, opaque credential ID, action, outcome and reason — never a raw key |

### Gate 3 — partially exercised

| Req | State | Mechanism / evidence |
|---|---|---|
| R16-09 revocation propagation | EXERCISED | one process-wide registry, `StartReloadLoop` at 15s inside the published 60s bound; `TestSharedRegistryRevocationPropagatesInProcess` drives both surfaces against ONE registry |
| R16-09 verifier-key rotation + operator audit | EXERCISED | `RotateVerifierKey` with a bounded ≤24h overlap; `rotation_audit.go` emits one opaque event per applied change and nothing for a no-op; `security-matrix-rotation-audit` |
| R16-09 validity bound | EXERCISED | `MaxCredentialValidity` = 90 days enforced inside `buildSnapshot`, so every load path (initial, reload, rotation) fails closed on a missing/over-long window |
| **R16-09 issuance** | **NOW IMPLEMENTED — was ABSENT** | see below |

**R16-09 issuance (2026-10-18).** The requirement names "approved key
generation, issuance, distinct principal assignment, expiry, overlap/rotation,
revocation" and warns *"Do not claim a lifecycle API exists until one is
implemented and protected."* Four of the six existed. **Issuance did not:**
`Digest()` had exactly one non-test caller (`Resolve`), so minting a credential
meant computing an HMAC by hand and hand-writing registry JSON that the loader
might then reject at startup — fail-closed, but only after deployment.

Now: `internal/auth/issuance.go` (`GenerateCredentialSecret` 256-bit,
`IssueRecord`, `LoadRegistryFile`, `WriteRegistryFile`, `RevokeInFile`) and the
operator CLI `cmd/credctl` (`issue` / `revoke` / `list`). Design points that
are requirements, not conveniences:

- **One validation implementation.** Issuance calls `New()` → `buildSnapshot`,
the same code the server uses, so `credctl` cannot author a file the server
rejects, and cannot author one weaker than the server accepts. `WriteRegistryFile`
validates the WHOLE set before touching the file, so an append can never replace
a live registry with something unloadable.
- **The secret is never in argv and never printed.** It is written to a new
0600 file (O_EXCL, no overwrite) after the registry write is durable; an argv
secret would land in `/proc/<pid>/cmdline`, `docker inspect` and shell history
(R17-08). The verifier key likewise comes from a 0600 file or the environment,
never a flag value.
- **Refusals are intent errors, not strength errors.** An unknown role, a scope
outside the published catalog, an empty scope list, a duplicate credential id,
and a non-positive or >90-day validity are all refused — each would otherwise
mint a credential that resolves and then denies every call, which its holder
reads as a broken deployment rather than a policy decision.
- **Atomic, symlink-safe, no partial state.** Serialize → temp file in the same
directory → chmod 0600 → fsync → rename; a symlink target is refused; an empty
registry is refused because the loader treats empty as "not configured" and
falls back to the legacy surface.
- **Revocation is attributable.** `revoked: true` in the file plus the existing
reload loop denies within the published 60s; an unknown credential id is an
error, so a revocation typo is never reported as success.

Evidence: `internal/auth/issuance_test.go` 10 tests driving the REAL issuer
against the REAL loader (`LoadFromSources`) — an issued record resolves to its
intended principal and role; the serialized record carries the verifier and not
the secret; the written file is 0600 and byte-loadable by the server path;
revocation denies inside the bound while an unrelated credential survives.
**Red-on-drift proven three ways**, each PASS→FAIL then restored: removing the
whole-set validation; allowing an empty set; removing the role-catalog check.
CI row `security-matrix-credential-issuance` runs the suite plus a build of
`credctl` that fails if the binary accepts an invocation with no verifier key or
exposes a secret-bearing argv flag. Live operator run verified: issue → 0600
registry + 0600 secret, secret absent from stdout, `list` shows `active`, `revoke`
→ `revoked`, duplicate id refused with exit 1.

**Still absent in Gate 3 (named, not implied):** cross-process revocation for a
multi-instance deployment is not proven (single-process reload only); a
"bounded documented legacy mode" exit date is an operator decision, not code;
Windows ACL equivalence is documented but not observed by any check.

### Gate 4 — unmet by design

R16-10 (bilateral Spec 15 contract) needs a second consumer; Spec 15's sending
half landed (`main@58ceb6c4`) and its receiving half has no submitter.
R16-11 (Spec 14 checker) needs a real consumer and host readiness evidence.
Neither is advertised as enforced.

**Implementation note — R16-08 audit posture (2026-10-19).** The audit
durability path was declared but never wired, and three settings around it
were inert; the dated note and its evidence are in the companion file
[`16-implementation-notes.md`](16-implementation-notes.md) (split out to stay
within the 500-line Markdown limit, Spec 19 R19-11).

## 6. Migration and rollback

1. Record current clients, public consumers, config sources, and tool/resource
   permissions. Owner signs the matrix and bounds; provision separate scoped
   credentials offline. Use shadow decisions only for observation, never to
   elevate a caller. Test both transports and rotate integrations one by one.
2. Enable deny-by-default enforcement with an explicit, time-bounded legacy
   allowlist; monitor denials without storing secrets. Remove fallback after
   verified zero-use window and documented cutover. Readiness must fail when a
   configured registry cannot be loaded.
3. Roll back a faulty policy/config release to the **last known-good restrictive
   policy** and previously valid, separately scoped credentials; never roll back
   to unrestricted legacy acceptance, unredacted config resources, or disabled
   audit. Stop privileged traffic if that safe rollback is unavailable. Preserve
   audit records and announce incompatible client changes before cutover.
4. Signed evidence and the conditional checker roll out independently only after
   bilateral/host conformance. On their failure, mark evidence unavailable or
   non-PASS and disable dependent effects rather than accepting unsigned or
   stale receipts. Retire deprecated trust roots and old keys after the agreed
   overlap, preserving tested revocation and replay bounds.

## 7. Implementation note — R16-01 resource matrix + R16-04 per-resource authorization (2026-10-16)

**Landed on `main`** (strict FF, no force, no PR) via
`forge/gr-spec16-resource-matrix`. Two named rows of Gate 0/Gate 1 are now
mechanisms rather than documents.

**The gap was real.** R16-04 required each resource read to be authorized "using
the same principal decision model as tools", and R16-01 required a complete
operation-to-permission matrix with default deny for an unmapped resource.
Before this change the resource handlers produced contents for **any** caller
that had passed the bearer gate: there was no per-resource decision and no
declared matrix, so a credential that could not call a read tool (for example a
`rest:read`-only key) could still read every `guardrail://` resource.

**R16-01 — declared resource permission matrix.**
`mcp-server/internal/mcp/resource_policy.go` is the single declared table: one
row per registered resource URI, each with an explicit privilege class and a
rationale. `classifyResource` returns `known=false` for a URI that is not in the
table, and `authorizeResourceRead` denies it before any read side effect. The
registration list was split out of `setupResources` into `resourceSpecs()` so
the matrix can be checked against the **real registry** without a live server.
`TestResourcePolicyMatrixCoversEveryRegisteredResource` proves the matrix is
non-vacuous and cannot fall behind the registry: every registered resource has
exactly one row, every row names a registered resource, and an undeclared URI is
unclassifiable (deny by default).

**R16-04 — per-resource authorization.** Every resource handler registered in
`setupResources` is now wrapped by a central gate that calls
`authorizeResourceRead(ctx, uri)` before the handler runs. The gate resolves the
server-controlled principal from the request context (never from the URI or
request content), classifies the resource, and applies the same
`auth.Decide` scope × role × resource intersection the tool path uses. A denial
returns a stable, non-leaking error and produces no contents. The decision is
recorded with the same audit shape as a tool call (`mcpAuditDecision`,
`required=false` — a resource read is not a security-sensitive mutation, so a
missing durable audit is not fatal).

**Evidence.** `resource_policy_test.go` drives the **real StreamableHTTP
endpoint** (`httptest` + `requireBearerWithRegistry`), not the handlers
directly: an `mcp:read` + `reader` credential reads `guardrail://config`
successfully, while an authenticated `rest:read`-only credential is refused and
the denial message leaks no resource detail. A direct-boundary test proves a
read with no principal denies, an undeclared resource denies, and an authorized
principal is allowed (non-vacuous). A new CI row
`security-matrix-resource-permission` runs all three verbatim, and a new
mutation-kill case `resource-read-authz-always-permits` proves the negative
suite flips PASS→FAIL when the decision is neutered.

**Named limits (corrected 2026-10-17 against the landed tree).** R16-06
(role/resource enforcement on the tool path) and R16-07 (legacy containment)
were already implemented in S-A0..A2 and are exercised; this note adds only the
resource half of the decision surface. R16-09 revocation propagation is now
in-process on **both** surfaces: one process-wide registry is loaded once and
exposed by `web.Server.CredentialRegistry()`, the web middleware consumes it via
`APIKeyAuthWithRegistry` (no per-construction copy), `mcp.MCPServer` shares the
same object via `SetCredentialRegistry`, and `cmd/server/main.go` starts
`Registry.StartReloadLoop` at `RevocationReloadInterval` (15s, inside
`RevocationPropagationBound` 60s). `TestSharedRegistryRevocationPropagatesInProcess`
(`internal/mcp/shared_registry_revocation_test.go`) drives the real web
middleware and the real StreamableHTTP endpoint against ONE registry: one
in-place `ReloadOnce` denies the revoked credential on both surfaces on the next
request with no rebuild or restart, an unrelated credential survives, and a
malformed reload keeps the prior validated snapshot. An earlier revision of this
note recorded the web half as a construction-bound registry; that is no longer
true of the tree and has been corrected. R16-10/R16-11 remain unmet by design
(no second product consumer yet).
