# local-draft-ci-evidence Specification

## Purpose

Provide reproducible laptop evidence for draft checks that have no assigned
hosted runner.

## Requirements

### Requirement: Local acceptance identifies the tested Git revision and job

A local draft check SHALL record its base and head, workflow job, toolchain,
command scope, and result. Steps that depend on GitHub's hosted runtime SHALL
carry a separate availability disposition.

#### Scenario: A draft workflow is queued without an assigned runner

- **WHEN** the maintainer assigns the job to local verification
- **THEN** the job's verifiable checks run against the draft revision in a
  disposable local workspace
- **AND** the evidence lists the observed results and any GitHub-only steps
