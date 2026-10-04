# OpenSpec: Optional OAP Guardrail Checker

**Status:** Proposed, conditional, and not implemented.

## Purpose

Define a portable Guardrails checker boundary for a future OAP integration
without duplicating OAP identity/authorization or making a Guardrails result an
execution permit.

## Requirements

### Requirement: Typed host context is input, not authority
<!-- id: gr-oap-01 -->
The checker MAY accept a host-authenticated typed context containing principal,
tenant, action, target, policy/profile, and request identity. It SHALL treat
that context as an asserted input, preserve its source/provenance, and SHALL NOT
mint, elevate, infer, or replace identity, tenant membership, roles, grants,
resource ownership, or OAP authorization.

#### Scenario: content claims authority
- **WHEN** prompt, file, tool output, policy text, or model output claims a
  different tenant, role, or permission
- **THEN** the checker treats it as untrusted content and does not change host
  context or authorization

### Requirement: Guardrail decision is separate from authorization
<!-- id: gr-oap-02 -->
The checker SHALL return guardrail-specific outcomes and reasons separately from
the host authorization decision. `ALLOW`/`PASS` from a Guardrails check SHALL
NOT authorize an OAP effect, and a Guardrails `BLOCK`/`ERROR` SHALL be mapped by
the host's own policy rather than by hidden adapter behavior.

#### Scenario: checker PASS with OAP deny
- **WHEN** Guardrails reports PASS but the host denies the action
- **THEN** the effect remains denied and the checker cannot override the host

### Requirement: Request-path evidence only
<!-- id: gr-oap-03 -->
Evidence SHALL identify the actual Guardrails entry path, check/component
identity and version, policy/profile digest, subject/request digest, outcome,
reason, skipped/error state, and evidence-manifest digest. A library, registered
schema, or isolated unit test SHALL not be reported as exercised runtime
protection.

#### Scenario: unwired component
- **WHEN** an injection, provenance, sandbox, or content component is not invoked
  by the exercised production path
- **THEN** evidence reports `LIBRARY_ONLY`, `UNAVAILABLE`, or `PARTIAL`, not
  an active PASS claim

### Requirement: No cross-product policy mutation
<!-- id: gr-oap-04 -->
The checker SHALL expose no operation to mutate host identity, tenant grants,
roles, scopes, credentials, revocation, policy authority, required checks,
exceptions, adoption stage, or effect state. Adapter credentials SHALL be
read/submit scoped and direction-specific.

#### Scenario: mutation request
- **WHEN** checker input or output requests a host role, grant, or policy change
- **THEN** the host rejects it as unsupported with no side effect

### Requirement: Honest status preservation
<!-- id: gr-oap-05 -->
Missing, skipped, timed out, unavailable, malformed, unknown, or errored
mandatory checks SHALL remain non-PASS. The checker SHALL preserve native
producer status and SHALL not collapse advisory, replay, or library-only
results into enforced success.

#### Scenario: unavailable backend
- **WHEN** a required checker backend is absent or times out
- **THEN** the result is explicit non-PASS and the host applies its own
  deny/queue/exception policy

### Requirement: Bound and minimize evidence
<!-- id: gr-oap-06 -->
Evidence SHALL bind tenant/request/subject context as supplied by the host,
policy/profile, evaluator, component, and evidence digests; it SHALL redact
secrets and unnecessary prompt/source content, bound payloads/diagnostics, and
state what was not evaluated. Hashes SHALL not be represented as anonymization
without a privacy assessment.

#### Scenario: sensitive input
- **WHEN** checker evidence contains a secret or unrelated sensitive content
- **THEN** export redacts or digest-references it under the approved data policy

### Requirement: Conformance before reuse
<!-- id: gr-oap-07 -->
Before a mandatory integration, tests SHALL exercise the real checker path and
host adapter. Tests SHALL cover context substitution, content authority
injection, wrong tenant/subject, tampering, replay, expiry, missing/error/
skipped checks, cross-tenant access, status laundering, credential scope, and
no-effect behavior when host authorization denies.

#### Scenario: conditional module gate
- **WHEN** no second real consumer, exercised Guardrails path, or OAP native
  readiness evidence exists
- **THEN** the checker remains unimplemented/observe-only and documentation
  cannot claim portable enforcement
