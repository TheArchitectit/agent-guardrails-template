# Spec Delta

## Purpose

Defines reviewable evidence for verify multi-instance revocation as part of an enterprise deployment rollout.

## ADDED Requirements

### Requirement: For a rollout with multiple service instances, credential revocation
For a rollout with multiple service instances, credential revocation SHALL become effective across the selected replica topology and failover path within the documented propagation bound.

#### Scenario: revocation is tested against only one process
- **WHEN** revocation is tested against only one process
- **THEN** the result is not accepted as evidence for a multi-instance topology.
