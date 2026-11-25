# Spec Delta

## Purpose

The Go advisory evidence record captures the toolchain, source-reachability comparison, residual module-only findings, and bounded build evidence for a reviewed release.

## ADDED Requirements

### Requirement: QA-05 evidence identifies its toolchain and scan scope
The QA-05 evidence record SHALL identify the exact reviewed Go toolchain and aligned build inputs, compare dated before-and-after source-reachability results, state residual module-only findings, and delimit the verified build scope.

#### Scenario: Evidence record preserves the tested remediation boundary
- **WHEN** a reviewed Go advisory remediation is recorded
- **THEN** its evidence identifies the matching toolchain and build pins, dated before-and-after source-reachability counts, residual module-only findings, and build verification scope.
