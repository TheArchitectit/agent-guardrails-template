# Spec Delta

## Purpose

Defines reviewable evidence for render and verify a supported kubernetes profile as part of an enterprise deployment rollout.

## ADDED Requirements

### Requirement: A supported Kubernetes rollout
A supported Kubernetes rollout SHALL produce a version-bound manifest whose application bind, Pod reachability, private ingress, authentication, and probes agree with the selected supported profile.

#### Scenario: the rendered Service or health probes target Pod networking but the selected profile does not permit the application listener to be reached there
- **WHEN** the rendered Service or health probes target Pod networking but the selected profile does not permit the application listener to be reached there
- **THEN** preflight rejects the render and records the failed check without applying it.
