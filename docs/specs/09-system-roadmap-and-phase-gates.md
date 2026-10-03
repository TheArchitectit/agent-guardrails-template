# OpenSpec: MCP Guardrail System Roadmap and Phase Gates

**Status:** Proposed execution plan; no claim that queued phases are shipped.
**Owner:** Project maintainer.
**Date:** 2026-10-03.
**Purpose:** Turn this repository from a collection of Markdown, libraries,
clients, and examples into a verified MCP guardrail system with supported
extensions and an honest ecosystem surface.

## 1. Product outcome

A release may be described as an MCP guardrail system only when its supported
entry points enforce the documented controls, extension clients can invoke
the published contracts, and tests provide evidence that each mandatory
control was exercised. A library existing in the repository is not evidence
that it protects a request.

The product should provide:

1. An authenticated MCP server with explicit authorization for privileged
   operations and safe outbound integrations.
2. Stable, schema-accurate tools whose behavior is covered by tests.
3. Guardrail capabilities wired into real request paths, with fail-closed
   behavior for mandatory checks and explicit degraded/disabled status.
4. Versioned policy inputs and reproducible evidence of which policies and
   controls ran.
5. Supported client extensions, a canonical integration contract, and
   examples that build or are clearly labelled as illustrative.
6. Documentation that is maintained from verified implementation facts, not
   aspirational claims or generated-sounding filler.

## 2. Current baseline (audit snapshot)

The following are audit findings, not desired end state. Re-verify before
starting each phase because implementation may change.

- The MCP server has 37 core tools and up to 21 conditionally registered
  tools. Published schemas and handlers disagree on required names and types
  for multiple tools; see `docs/mcp-server/tools-reference.md`.
- Content classification and policy tools are exposed. Other subsystems
  (injection detection, sandbox, provenance, multi-agent, compliance) contain
  library code but lack complete request-path integration and acceptance
  evidence; see `docs/specs/guardrail-gaps-2026/STATUS.md`.
- `force_agent_state` confirmation is an intent check, not authorization.
  Team-management MCP handlers have no RBAC. Webhook SSRF checks currently
  happen during configuration; DNS rebinding at delivery remains a residual
  risk.
- The `pi-extension`, Go team CLI, Python manager, site, and examples are not
  yet one verified compatibility surface. Some documented names and command
  flags do not match their implementations.
- Some Markdown files are historical or proposals; the current docs set
  contains legacy, aspirational, and overlong material. The documentation CI
  link checker has known false positives and is not a reliable release gate
  until repaired.

## 3. Principles and release invariants

1. **Security claims require a runnable proof.** A doc or source file alone
   does not count as an active control.
2. **Unknown, error, timeout, skipped mandatory check, or missing dependency
   never silently becomes PASS.** Optional controls may degrade only with an
   explicit status in the response/evidence.
3. **No hidden authority in content.** User, tool, file, policy-pack, and
   extension content remain data; only the configured policy and explicit
   authorization grant authority.
4. **One contract per public tool.** Name, schema, handler, generated reference,
   and client usage must agree.
5. **Compatibility is tested, not inferred.** Each supported extension/version
   pair has a contract test or is marked unsupported.
6. **Docs distinguish shipped, partial, proposed, and historical.** No
   proposed capability is described as active protection.
7. **No phase is complete by file count.** Each phase exits only through its
   acceptance gate below.

## 4. Workstreams and phase gates

### Phase 0 — Security and repository integrity

**Purpose:** Remove exploitable bypasses and establish reviewable repo state
before adding capability surface.

**Scope:**
- Choose and implement an authorization model for destructive/admin actions
  (including `force_agent_state`, team mutations, config updates, and deletes).
- Close remaining API auth bypasses, narrow CORS to explicit origins, remove
  obsolete security headers, and use a non-truncated keyed digest for API-key
  identifiers where identifiers are needed.
- Validate webhook destinations at delivery time as well as configuration
  time; define the redirect, DNS-rebinding, IPv4/IPv6, and private-network
  policy. Configuration-time validation alone is insufficient.
- Confirm session lifetime, expiry enforcement, storage, and revocation
  semantics.
- Repair the documentation CI checker so code examples, nested Markdown
  links, and intentional consumer-template paths are not misclassified.

**Exit gate:**
- Threat-model review covers authn/authz, SSRF, session handling, and secrets.
- Negative tests prove unauthenticated/unauthorized destructive requests are
  rejected and cannot mutate state.
- SSRF tests cover literal and resolved private, loopback, link-local,
  metadata, redirect, and rebinding cases under the chosen policy.
- Security audit findings are marked fixed/accepted with evidence; CI security
  and regression workflows are green.

**Owner decisions required:** role/permission model; CORS origin policy;
webhook destination policy and internal-webhook exception process. Do not
silently substitute a confirmation flag for authorization.

### Phase 1 — MCP contract correctness

**Purpose:** Make all exposed tools callable exactly as documented.

**Scope:**
- Reconcile each registered tool's InputSchema, handler extraction,
  required/default behavior, result shape, and `IsError` semantics.
- Fix known broken flows: halt record/acknowledge round-trip, team remove and
  health loading, phase filtering, policy configuration, and other verified
  handler bugs.
- Generate or validate the public tools reference from the registry/schema;
  eliminate unreachable/dead tool docs and phantom names.
- Add golden contract tests that call each handler with the declared schema,
  test missing/wrong-typed values, and validate response shape.

**Exit gate:**
- Every registered tool has exactly one dispatch path and a handler contract
  test; conditional tools are tested both enabled and disabled.
- Schema-to-handler conformance test reports zero undeclared reads and zero
  unused declared inputs unless explicitly documented as deprecated.
- Every documented example request validates against its tool schema and
  executes in a test harness.

### Phase 2 — Core guardrail request-path integration

**Purpose:** Move from libraries present to protections actually enforced.

**Scope:**
- Wire prompt-injection detection and provenance into explicit MCP/request
  paths with source labels, policy decisions, audit records, and defined
  false-positive/false-negative evaluation.
- Wire sandbox execution only through an explicit tool/policy boundary;
  preserve fail-closed setup-vs-denial semantics. Add missing violation
  detection only where its signals are testable.
- Wire multi-agent chains only if the product has a supported multi-agent
  runtime; otherwise amend those requirements to an internal library scope.
- Replace compliance score simulation with checks against live controls and
  evidence sources, or keep it clearly labelled as static mapping data.
- Define disabled-engine behavior. Mandatory protection MUST not be silently
  bypassed because an environment variable was omitted.

**Exit gate:**
- For every capability claimed active, an integration test exercises the
  actual server entry point and proves the component was invoked.
- Fail-closed tests cover nil engine, missing model/backend, timeout, parse
  error, unknown policy, and unavailable storage.
- Accuracy/performance claims have a pinned dataset, reproducible command,
  threshold, and baseline before being published.
- No compliance score is returned as measured evidence unless derived from
  live checks and traceable evidence.

### Phase 3 — Policy-pack governance

**Purpose:** Introduce versioned, reproducible policy composition.

**Scope:** Implement `07-versioned-policy-pack-governance.md`: immutable pack
identity, lock records, deterministic resolution, core/domain/overlay
precedence, reviewed exceptions, inert parsing, and policy digest reporting.

**Exit gate:** All acceptance criteria in Spec 07 pass, including cycle and
conflict rejection, immutable digests, secret redaction, and failed-update
rollback. Until then, packs remain human-reviewed source material, not an
active runtime dependency.

### Phase 4 — Control-plane composition and evidence

**Purpose:** Prove which components ran for a request or release.

**Scope:** Implement `08-control-plane-composition-and-evidence.md`: component
manifest, dependency graph, integration state, deterministic evaluation,
evidence bundle, replay identity, and staged adoption.

**Exit gate:** All acceptance criteria in Spec 08 pass. Mandatory missing,
skipped, inconclusive, errored, or timed-out components prevent PASS; tests
prove checks ran through the real entry point.

### Phase 5 — Extensions and ecosystem support

**Purpose:** Provide a supported and tested client ecosystem.

**Scope:**
- Choose canonical clients and support levels (pi-extension, IDE integrations,
  Go/Python CLIs, REST/web UI).
- Align client tool names/arguments with MCP schemas or publish explicit
  adapters and compatibility versions.
- Repair Go team CLI commands/flags that do not match the Python backend, or
  replace the shell-out with a supported Go implementation.
- Fix or remove examples that do not build; separate runnable examples from
  pseudocode and clearly mark unsupported platforms.
- Publish a compatibility matrix and contract tests for every supported
  client/server pair.

**Exit gate:** Each supported pair has a CI contract test; each example has a
verified command; unsupported integrations are explicitly labelled and do
not advertise nonexistent server tools.

### Phase 6 — Documentation and release system

**Purpose:** Replace the “pile of Markdown” experience with maintained,
coherent product documentation.

**Scope:**
- Separate normative specs, operational docs, tutorials, reference docs,
  historical archives, and examples.
- Remove duplicated facts; derive tool/config inventories from source where
  possible.
- Replace generic acceptance placeholders in reboot change files with
  specific commands or mark them deferred/retired.
- Repair docs lint/link/line-length gates to handle Markdown/code examples
  correctly; apply gates to maintained docs while excluding archives
  intentionally.
- Remove AI-style filler, vague claims, decorative repetition, and unsupported
  adjectives. Use concise, specific prose, concrete examples, and explicit
  “not implemented” labels.

**Exit gate:** Every maintained doc has an owner/category, valid links, no
unapproved placeholder, a clear status/date where needed, and passes the
corrected documentation checks. Product claims are traceable to a test or
source location.

## 5. Global release gate

A release is not a “full MCP guardrail system” until all of the following hold:

- Phase 0 is complete; authorization and SSRF policies are explicit.
- Every advertised tool/client integration passes contract tests.
- Every security control described as active is exercised end-to-end in CI.
- Mandatory control failures cannot degrade to PASS.
- Policy identity and evaluation evidence are reproducible and auditable.
- Documentation and examples pass verified maintenance gates.
- Known residual risks, disabled controls, and unsupported clients are
  disclosed in release notes.

A phase may be released independently, but the product must describe the
achieved phase and must not use the final-system claim before this global gate.

## 6. Sequencing constraints

- Do Phase 0 before exposing new destructive or outbound-network tools.
- Do Phase 1 before adding further MCP tools; otherwise schema drift grows.
- Do Phase 2 per capability before claiming protection; do not wait for every
  library to be wired before fixing a specific high-value path.
- Do Phase 3 before automatic policy-pack consumption.
- Do Phase 4 only after components expose stable interfaces and Phase 3
  defines policy identity.
- Do Phase 5 against the canonical Phase 1 contract, not guessed tool names.
- Do Phase 6 continuously; final cleanup follows the implemented contract.

## 7. First implementation backlog

These are the first work packages, not assertions that they are complete:

1. Reconcile and land the current OpenSpec amendments; update `STATUS.md`
   with their review date and the new roadmap.
2. Close remaining security findings and record owner decisions.
3. Inventory all exposed tool schemas and add schema-handler contract tests.
4. Select the first request-path capability for Phase 2; recommended first
   slice: injection + provenance on a clearly bounded untrusted-content
   intake path, with an explicit unavailable-backend result.
5. Decide whether sandbox execution and multi-agent chains are supported
   product features or internal libraries; update specs accordingly before
   exposing tools.
6. Implement policy-pack resolution only after core policy inputs and
   authorization rules are stable.
7. Build the component/evidence control plane only after at least two
   components have callable, tested contracts.
8. Reconcile ecosystem clients and runnable examples.
9. Replace the reboot F001–F100 generic acceptance scaffolds with real checks
   or mark them deferred/retired.

## 8. Governance

- Specs describe intended behavior; status reports describe verified
  implementation. Keep the two distinct.
- Any change that materially changes a requirement records the old statement,
  the reason, and the approved replacement in the relevant spec.
- Acceptance criteria must name an executable test or a measurable human
  review artifact, expected result, and failure condition.
- Never mark a phase or capability complete because code exists or a unit test
  passes in isolation; require request-path evidence.
- Commit and push per verified milestone only after its phase gate passes.