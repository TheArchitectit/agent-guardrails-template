# Design

## Context

`mcp-server/deployment-guide.md` has an older version/date and hand-run Podman commands. Current controls are described across deployment configuration, `scripts/run_migrations.go`, profile validation, and readiness paths; their authoritative commands and prerequisites must be reconciled before the guide is used for a bank rollout.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Use one supported deployment path as the normative sequence and link alternatives as non-production examples. Keep secret values out of command arguments and transcripts. Verify each documented command in a disposable staging deployment. Alternatives: preserving historical commands as equal paths leaves operators unable to tell which controls apply; deleting platform-specific guidance without a tested replacement leaves a gap.

## Risks / Trade-offs

[Guide drifts again] → include source revision/evidence date and an ownership review trigger; [unsafe copy/paste] → use placeholders only for non-secret values and reference platform secret names rather than values.
