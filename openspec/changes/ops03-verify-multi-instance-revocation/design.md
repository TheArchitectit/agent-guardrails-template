# Design

## Context

`internal/auth/registry.go` documents its propagation bound relative to the registry source becoming visible; `registry_lifecycle_test.go` verifies a local reload. The bank topology and registry synchronization mechanism have not been selected in this review.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Exercise the actual selected distribution path in staging with test credentials, observe each instance and failover, and compare against the documented propagation bound. Alternatives: extrapolating from unit tests is not evidence for inter-process delivery; publishing credential material is unnecessary and prohibited.

## Risks / Trade-offs

[Topology changes invalidate evidence] → bind report to replica count, registry mechanism, image/config revision and repeat on topology change; [test key reused] → use synthetic credentials and destroy them after the drill.
