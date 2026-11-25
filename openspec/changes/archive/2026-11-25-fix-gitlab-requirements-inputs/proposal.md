# Proposal

## Why

The GitLab pipeline installs `-r requirements.txt` in its validation and Python test jobs, while the repository has no root `requirements.txt`. Those jobs therefore depend on an undeclared input and cannot reproduce the repository's declared Python environment from a clean checkout.

## What Changes

- Make GitLab Python jobs install dependencies from the repository's declared Python project metadata.
- Verify validation, tests, coverage, and export jobs from a clean checkout.

## Capabilities

### New Capabilities
- `gitlab-python-job-dependencies`: reproducible dependency setup for GitLab Python jobs.

### Modified Capabilities
- None.

## Impact

`ci/gitlab-ci.yml`, Python dependency metadata, and GitLab job execution. Jenkins dependency setup is tracked separately by CI-02.
