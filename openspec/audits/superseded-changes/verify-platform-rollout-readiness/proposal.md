# Proposal: Verify platform rollout readiness

## Why

The repository contains reusable authorization, signed-evidence, deployment,
migration, readiness, and CI controls. The pinned review does not identify a
specific enterprise consumer or deployment, demonstrate an approved
nonlocal deployment end to end, or record an operational rollout rehearsal.
The current Kubernetes manifest uses placeholders, omits non-loopback bind
settings and a deployment profile, and has no documented render/apply path.
The production deployment guide predates later profile, preflight, and
migration controls. Cross-product producer and verifier mechanics do not
establish acceptance by a real external consumer; multi-instance revocation
and rollback/availability behavior also lack evidence for a selected
deployment topology.

These are rollout evidence gaps rather than proof that the local controls are
absent. Findings `DEP-01`, `DEP-02`, `OPS-01` through `OPS-04`, and `INV-02`
are recorded in `openspec/audits/2026-11-25-qa-and-gap-review.md`.

## Change

Define the evidence required before describing a supported enterprise rollout:
an approved deployment and consumer inventory, a rendered and version-bound
private deployment configuration, real consumer integration evidence, a
multi-instance revocation result where applicable, named operating owners,
and a migration/readiness/restore/rollback rehearsal for the selected topology.
Keep policy checkers and evidence receipts in their documented observe-only
and evidence roles.

## Impact

This proposal records readiness criteria and does not certify compliance,
authorize production rollout, or change deployed controls. Where evidence is
private, retain it in the approved operating system and publish only its
version, outcome, and reviewable reference.
