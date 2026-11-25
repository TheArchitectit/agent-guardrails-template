# Design

## Context

The injection and semantic harnesses can calculate metrics, and a semantic drift monitor can identify version changes. Current repository workflows do not require a matching report before promotion.

## Decisions

- Bind reports to release candidate, source revision, corpus versions, classifier identity/version/configuration, policy digest, and supported runtime profile where relevant; calculate a content digest and retain the report immutably under this identity.
- Make report production and threshold evaluation a required release status check for verification claims.
- Fail promotion when a report is absent, stale, below a category bar, over a false-positive budget, or missing required fields.
- Retrieve and verify the retained report digest at gate time, disallow overwriting an existing report for the same identity, and retain named failing fixtures; do not infer success from a unit test of the metric function.

## Rollout

Introduce report-only generation, validate artifact completeness, then require the gate for verified promotion. Any input version change invalidates the prior report.
