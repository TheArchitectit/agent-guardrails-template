# Design

## Context

Semantic corpus utilities currently include marker-based examples and a reference implementation. Category conformance claims need evidence tied to actual configured classifiers and reviewed fixture versions.

## Decisions

- Require a declared minimum detection rate and maximum false-positive rate per category; pin corpus identity and version, classifier identity/version/configuration, both thresholds, and measured results in the report.
- Include representative positive and benign negative cases for every shipped category; report fixture-level misses and false positives.
- Label reference-classifier results as harness checks and do not use them as production-classifier performance evidence.
- Keep promotion gating and stale-report invalidation in a separate change.

## Rollout

Run the corpus in report-only mode against each configured classifier first. Mark only categories with matching results and satisfied budgets as evaluated.
