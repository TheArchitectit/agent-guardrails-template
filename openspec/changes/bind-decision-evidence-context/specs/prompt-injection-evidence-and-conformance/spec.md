# Prompt Injection Evidence and Conformance

## MODIFIED Requirements

### Requirement: per-decision evidence

Every classification and mediation decision SHALL emit an envelope binding the subject, active policy or bundle digest, classifier identity and version, category, outcome, and signature state. A sealed or verifiable decision SHALL have a signature a downstream verifier can validate against active trust configuration. Unsigned or incomplete records SHALL be identified and SHALL NOT be claimed to satisfy the sealed evidence contract.

#### Scenario: blocked injection attempt

- **GIVEN** an input classified as suspected prompt injection and blocked by policy
- **WHEN** the mediation decision completes
- **THEN** an envelope records the subject, active policy digest, classifier identity/version, category, outcome, and verifiable signature state

#### Scenario: unsigned evidence

- **GIVEN** evidence emitted without a configured signer or with required decision fields absent
- **WHEN** a downstream gate evaluates the record
- **THEN** it identifies the missing signature or fields and does not treat the record as satisfying the sealed evidence contract
