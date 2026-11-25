# Design

## Context

`docs/mcp-server/migration-procedures.md`, `mcp-server/scripts/run_migrations.go`, readiness tests, and `docs/workflows/rollback-procedures.md` describe mechanisms. There is no bank-specific rehearsal artifact, selected topology, or service objective in this review.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Have the deployment owner define measurable recovery objectives and safe stop conditions before the drill; perform against synthetic or approved data in isolated staging; record step outcomes and restore integrity. Alternatives: production rehearsal adds unnecessary operational risk; an unbounded “successful” status cannot be evaluated.

## Risks / Trade-offs

[Drill harms shared services] → isolate resources, use synthetic data and preapproved stop conditions; [rollback reintroduces unsafe state] → follow the documented migration/credential controls and require operator approval for each state transition.
