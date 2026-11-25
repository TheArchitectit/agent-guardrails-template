# Design

## Context

The deployment manifest has placeholder image/version values and shell-style `${NAME:-default}` values inside Kubernetes YAML. `mcp-server/internal/config/profile.go` owns supported profile defaults and validation. Service and health probes target Pod networking, so rendered listener/profile values must be checked as one unit.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Render from an explicit, versioned profile input with a documented tool available to operators; reject unresolved placeholders and profile/bind/reachability disagreement before apply. Keep private endpoint values in the approved deployer configuration, not the repository. Alternatives: raw `envsubst` is insufficient for shell-default expressions and does not validate profile policy; hand-editing YAML is not reproducible.

## Risks / Trade-offs

[Misrendered exposure] → fail validation before apply and test the rendered manifest against the selected profile; [secret leakage] → prohibit secret values in render artifacts and consume secret references from the approved platform store.
