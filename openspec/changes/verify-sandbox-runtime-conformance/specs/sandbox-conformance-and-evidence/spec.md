# Sandbox Conformance and Evidence

## MODIFIED Requirements

### Requirement: escape-fixture conformance

Each isolation level SHALL be tested against escape fixtures covering filesystem traversal, unauthorized network egress, process/namespace escape, and undeclared secret access. Any successful escape SHALL fail certification. If the suite has not run, escape-fixture conformance SHALL remain unverified.

#### Scenario: release conformance run

- **GIVEN** the approved escape-fixture suite for an isolation level and supported runtime profile
- **WHEN** release conformance executes that suite against the runtime
- **THEN** every fixture records its expected blocked result, and any successful escape fails certification for that profile; if the suite was not run, escape-fixture conformance remains unverified

### Requirement: isolation in evidence

Every executed-workload envelope SHALL record declared and actual isolation, enforcing runtime identity/version, measured host capability, and whether containment was observed through that workload's completion. Its signature SHALL cover these fields and outcome. A downstream verifier SHALL validate the signature and field binding before accepting this run-specific claim. Missing, unsigned, or unverifiable fields SHALL NOT support a successful claim.

#### Scenario: consumer verifies containment claim

- **GIVEN** an evidence envelope for a completed workload
- **WHEN** a downstream gate or auditor verifies the envelope
- **THEN** the declared and actual isolation levels, runtime identity/version, host capability, and containment-through-completion result are present and covered by a valid envelope signature

#### Scenario: workload does not complete under containment

- **GIVEN** a runtime fault or containment loss before a workload completes
- **WHEN** the execution evidence is emitted
- **THEN** the envelope records a failed or incomplete outcome, does not claim containment held through completion, and signs the fault and completion state

#### Scenario: sandbox envelope cannot be verified

- **GIVEN** an envelope with missing runtime fields, an unsigned field set, or a signature that fails verification
- **WHEN** a downstream gate evaluates the containment claim
- **THEN** it rejects the claim as incomplete or invalid and does not report evidence-backed containment

#### Scenario: only static policy checks are available

- **GIVEN** policy-plan checks with no workload executed by a supported runtime
- **WHEN** conformance results are reported
- **THEN** they are labeled as policy-plan evidence and do not claim runtime containment was exercised

## ADDED Requirements

### Requirement: safe runtime capability check evidence

Benign permitted and denied capability checks MAY establish only the observed behavior of the tested capability and runtime profile. They SHALL NOT establish resistance to escape attempts or substitute for escape-fixture conformance.

#### Scenario: benign capability behavior is observed

- **GIVEN** a supported runtime profile and a declared isolation level
- **WHEN** a benign permitted operation and a benign denied operation exercise a claimed capability
- **THEN** the permitted operation succeeds, the denied operation is blocked, and the report labels both results as safe capability-check evidence only
