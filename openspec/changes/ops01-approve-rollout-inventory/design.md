# Design

## Context

Existing docs define profiles, auth, deployment, and operational controls but no selected bank deployment or consumer. No evidence located in this review names the bank environment or accountable operators.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Use an operator-approved inventory artifact with a revision and review date; store private identities/endpoints and approval records in the bank’s approved evidence store. Alternatives: publishing those details in git increases exposure; a generic checklist does not identify applicability or ownership.

## Risks / Trade-offs

[Inventory becomes stale] → require owner/date and change-triggered review; [private detail copied into repo] → use redacted evidence references and review artifacts before publication.
