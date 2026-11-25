# go-advisory-evidence Specification

Status: CLOSED (EXERCISED 1)

## Purpose

The Go advisory evidence record captures the toolchain, source-reachability comparison, residual module-only findings, and bounded build evidence for a reviewed release.

## Requirements

### Requirement: QA-05 evidence identifies its toolchain and scan scope
The QA-05 evidence record SHALL identify the exact reviewed Go toolchain and aligned build inputs, compare dated before-and-after source-reachability results, state residual module-only findings, and delimit the verified build scope.

#### Scenario: Evidence record preserves the tested remediation boundary
- **WHEN** a reviewed Go advisory remediation is recorded
- **THEN** its evidence identifies the matching toolchain and build pins, dated before-and-after source-reachability counts, residual module-only findings, and build verification scope.

## Requirement Closure

| Requirement | State | Evidence |
|---|---|---|
| QA-05 evidence identifies its toolchain and scan scope | EXERCISED | The archived evidence records the toolchain selection, build verification, source and production-tag scan results, and four module-only advisories for the merged change. [QA-05 evidence](../../changes/archive/2026-11-25-qa05-source-reachable-go-toolchain-remediation/evidence.md) |
