# Runtime Conformance Evidence Specification

## Purpose

Ensure conformance claims describe behavior exercised at the named runtime
boundaries and are traceable to the active configuration.

## ADDED Requirements

### Requirement: sandbox runtime evidence

Any claim that a sandbox runtime enforces an isolation boundary SHALL be
supported by a conformance workload executed against a supported runtime, with
the runtime identity and observed result recorded. Static policy-plan checks
MAY support configuration claims but SHALL NOT alone establish runtime
containment.

#### Scenario: only static policy checks are available

- **WHEN** a conformance report contains policy-plan checks but no workload
  executed by a supported runtime
- **THEN** the report identifies the result as policy-plan evidence and does
  not claim runtime containment was exercised

### Requirement: representative semantic conformance evidence

Semantic category conformance SHALL use versioned, representative positive
and benign negative fixtures and SHALL record the configured classifier
identity and category-specific results and budgets.

#### Scenario: reference marker classifier is used

- **WHEN** conformance uses a marker-recognition reference classifier
- **THEN** the report identifies harness behavior as the scope of evidence and
  does not present it as production classifier performance

### Requirement: mediation boundary traceability

Specs that claim automatic injection or provenance enforcement at action or
ingestion boundaries SHALL identify repository call sites or integration
evidence showing those boundaries invoke the guardrail before the action or
content is accepted.

#### Scenario: only an explicit analysis tool is evidenced

- **WHEN** the repository demonstrates a caller-invoked analysis tool but no
  automatic boundary integration
- **THEN** the conformance record limits its claim to the explicit tool

### Requirement: evidence binds active decision context

Decision evidence SHALL identify the active policy digest, classifier identity
and version, category, outcome, and whether the evidence is verifiably signed
when the corresponding spec makes those claims.

#### Scenario: evidence is unsigned or omits decision identity

- **WHEN** evidence is emitted without a verifiable signature or without
  required active decision context
- **THEN** the report states the missing fields and does not claim the stronger
  evidence contract was exercised

### Requirement: redaction action evidence

When a semantic guardrail specification claims redaction, the configured
runtime SHALL remove the declared sensitive spans from returned content and
emit evidence identifying the redaction without including the removed secret.

#### Scenario: redaction is declared but unsupported

- **WHEN** the configured action is redaction but the runtime does not support
  span removal
- **THEN** the policy load or decision fails closed and does not report
  redaction as exercised

### Requirement: release-gated conformance report

Conformance claims used for release promotion SHALL be backed by a
version-bound report that is produced by a required release gate.

#### Scenario: report is missing or stale

- **WHEN** the configured classifier, policy, or supported runtime changes
  after the latest conformance report
- **THEN** release promotion requires a fresh report for the new version
