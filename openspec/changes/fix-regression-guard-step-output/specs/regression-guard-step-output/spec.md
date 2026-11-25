# Spec Delta

## Purpose

Defines safe, valid output records for Regression Guard's PR commit-analysis
step.

## ADDED Requirements

### Requirement: PR commit analysis emits valid consumed outputs

The PR commit-analysis step SHALL emit the numeric `bug_fix_count` consumed by
downstream workflow steps, and SHALL NOT write a multiline commit-subject
value as a single-line `GITHUB_OUTPUT` record.

#### Scenario: mixed commit subjects

- **GIVEN** a PR subject list containing ordinary and conventional fix
  subjects
- **WHEN** the analysis step writes its GitHub Actions outputs
- **THEN** `bug_fix_count` equals the number of matching fix subjects and each
  output record is a valid single line
- **AND** no unused subject-list output is emitted
