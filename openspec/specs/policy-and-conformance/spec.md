# policy-and-conformance Specification

Status: CLOSED (EXERCISED 2)

## Purpose

Apply source-trust-class policy to external content so untrusted content cannot self-elevate, and reconstruct full ancestry when it is acted upon.

## Requirements

### Requirement: class-aware policy

The bundle SHALL be able to express policy on provenance class, including: no destructive action SHALL be justified solely by third-party or unverifiable content.

#### Scenario: destructive action from web content

- **GIVEN** a delete action justified only by fetched third-party content
- **WHEN** policy above is active
- **THEN** mediation SHALL block and escalate.

### Requirement: ancestry reconstruction

Given any mediated action, the full provenance ancestry of its justifying content SHALL be reconstructable from the ledger.

#### Scenario: audit query

- **GIVEN** an action from a week ago
- **WHEN** an auditor queries its ancestry
- **THEN** the result SHALL list every content input with origin class and digest.
