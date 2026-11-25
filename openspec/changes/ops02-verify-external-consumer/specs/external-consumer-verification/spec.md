# Spec Delta

## Purpose

Defines reviewable evidence for verify the external consumer contract as part of an enterprise deployment rollout.

## ADDED Requirements

### Requirement: Where an external consumer participates, rollout evidence
Where an external consumer participates, rollout evidence SHALL show that the named consumer version verifies signed evidence and enforces the declared identity and policy contract; a receipt or observe-only result SHALL NOT authorize an effect.

#### Scenario: only producer or verifier unit tests are available
- **WHEN** only producer or verifier unit tests are available
- **THEN** consumer acceptance is recorded as unverified and the rollout is not declared ready for that consumer.
