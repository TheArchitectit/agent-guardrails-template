# categories-and-actions Specification

Status: CLOSED (EXERCISED 3)

## Purpose

Declare semantic content categories and their mediation actions in the policy bundle, name the classifier behind each decision, and fail conservatively at boundaries no category claims.

## Requirements

### Requirement: bundle-declared categories

Content categories and their actions SHALL be declared in the policy bundle. A category without a declared action set, corpus reference, and failure-mode statement SHALL be invalid.

#### Scenario: vague category rejected

- **GIVEN** a bundle adding a category with no corpus or failure modes
- **WHEN** the bundle is validated
- **THEN** validation SHALL FAIL naming the missing fields.

### Requirement: named classifier per decision

Every filtering decision SHALL record classifier provider, model, version, and runtime where applicable. A decision lacking classifier identity SHALL fail envelope validation.

#### Scenario: unnamed model

- **GIVEN** a decision emitted without model version
- **WHEN** the envelope is validated
- **THEN** it SHALL be rejected.

### Requirement: conservative boundary defaults

At send, commit, and publish boundaries, sub-threshold or errored classification SHALL default to escalate or block per category policy, never silent allow.

#### Scenario: classifier error at send

- **GIVEN** the classifier times out while evaluating an outbound message
- **WHEN** mediation decides
- **THEN** the send SHALL be held for escalation.
