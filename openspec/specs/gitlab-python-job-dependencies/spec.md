# gitlab-python-job-dependencies Specification

## Purpose

Defines how GitLab Python jobs obtain declared dependencies so clean-checkout pipeline execution is reproducible.

## Requirements

### Requirement: GitLab Python jobs use declared dependency inputs

GitLab Python jobs SHALL install dependencies from a repository-maintained manifest that exists in a clean checkout.

#### Scenario: execute a Python validation job

- **WHEN** GitLab runs Python validation, test, or export jobs from a clean checkout
- **THEN** dependency installation succeeds using a tracked manifest and the jobs preserve their configured scripts and artifacts

### Requirement: GitLab test dependencies are explicit

GitLab Python test jobs SHALL declare test and coverage dependencies through their installation steps or maintained project metadata.

#### Scenario: run Python tests in the pipeline

- **WHEN** the GitLab Python test job installs dependencies and runs its test command
- **THEN** the test and coverage tools are available without relying on undeclared runner state
