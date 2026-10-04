# OpenSpec 16: Authentication, authorization, and evidence remediation

**Status:** Proposed; NOT implemented. Source snapshot: `178d076` (re-verify against
HEAD before implementation). This is a remediation plan and release gate, not a
claim that Spec 11, 12, 14, or 15 has shipped. Owner approval is required for the
permission matrix, public routes, lifecycle bounds, and cross-product rollout.
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
policy checks may a cross-product artifact be represented as verified. Test
both directions; receipts and signatures are not effect grants. Registry HMAC
verifiers alone do not satisfy this requirement.

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

## 4. Migration and rollback

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
