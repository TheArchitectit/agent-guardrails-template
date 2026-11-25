# Platform Rollout Evidence Specification

## Purpose

Define reviewable evidence for an enterprise deployment using the repository's
guardrail platform controls.

## ADDED Requirements

### Requirement: approved rollout inventory and ownership

A rollout SHALL have an approved record of consumers, deployment topology,
supported profile, data and network boundaries, and named service,
control, and operations owners before the deployment is described as ready.

#### Scenario: rollout target or owner is unspecified

- **WHEN** the consumer, supported profile, or accountable owner is not
  identified
- **THEN** the readiness record marks the rollout inventory incomplete and
  does not declare the deployment ready

### Requirement: rendered private deployment evidence

A supported Kubernetes deployment SHALL have a rendered, version-bound
configuration in which service binds, Pod reachability, ingress, authentication,
and health probes agree with the selected private profile.

#### Scenario: service probes a Pod IP with loopback binds

- **WHEN** the Service or health probes target the Pod IP but the application
  binds only to loopback
- **THEN** deployment validation rejects the configuration as not ready

### Requirement: current deployment procedure

The production deployment procedure SHALL describe the current supported
profile, ingress preflight, secret handling, staging checks, and migration job
sequence.

#### Scenario: guide predates required controls

- **WHEN** the deployment guide omits a currently required profile, preflight,
  secret, or migration control
- **THEN** the guide is marked stale for rollout use until reconciled

### Requirement: external consumer verification

Where an external consumer is part of the rollout, readiness SHALL include
version-bound evidence that the actual consumer verifies signed evidence and
enforces its declared identity and policy contract. A receipt or observe-only
checker result SHALL NOT itself authorize an effect.

#### Scenario: only local producer tests are available

- **WHEN** producer and verifier unit tests pass but no external consumer
  integration evidence is recorded
- **THEN** readiness identifies consumer acceptance as unverified

### Requirement: multi-instance revocation evidence

For deployments with multiple service instances, credential revocation claims
SHALL be tested across the selected replica topology, including propagation
and failover behavior.

#### Scenario: only one instance is tested

- **WHEN** revocation propagation is demonstrated against a single process
  only
- **THEN** the result is not applied as evidence for a multi-instance
  propagation bound

### Requirement: operational recovery rehearsal

A rollout SHALL have a recorded rehearsal for migrations, readiness, backup
and restore, and operator-controlled recovery or rollback against the selected
topology, with service objectives and outcomes recorded.

#### Scenario: rollback behavior has only design evidence

- **WHEN** the repository documents a rollback procedure but no rehearsal
  result exists for the selected topology
- **THEN** the readiness record marks operational recovery evidence pending

### Requirement: current platform baseline

The platform current-state document SHALL identify its source revision and
evidence date, and SHALL be reconciled when current implementation evidence
supersedes a stated capability gap.

#### Scenario: audit persistence is implemented after the baseline date

- **WHEN** the current-state document says audit persistence is unattached but
  current server wiring attaches an audit store
- **THEN** the baseline is refreshed or explicitly marked as historical with
  a link to the newer evidence
