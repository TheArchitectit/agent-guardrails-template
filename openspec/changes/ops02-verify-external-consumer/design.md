# Design

## Context

Local producer/verifier tests establish local mechanics only. The rollout plan’s “external consumer” is not named. Existing control-plane receipts are evidence and must not be treated as authorization.

## Goals / Non-Goals

**Goals:** Produce reviewable, revision-bound evidence for this rollout decision.

**Non-Goals:** Certify bank compliance, authorize production deployment, or publish private operational data.

## Decisions

Bind an acceptance record to consumer version, producer image/commit, policy/config revision, and test outcome; use a synthetic or approved test identity and approved isolated environment. Determine Gate 4 applicability from the selected deployment boundary. Alternatives: a local mock cannot establish consumer acceptance; storing full payloads risks disclosure and is unnecessary for a reviewable outcome.

## Risks / Trade-offs

[Test does not represent production policy] → document test boundary and policy revision; [sensitive records] → store payloads privately and publish only digest/reference, version, and outcome.
