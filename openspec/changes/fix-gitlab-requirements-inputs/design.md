# Design

## Context

`ci/gitlab-ci.yml` installs `requirements.txt` for `validate_teams` and `test_python`; no root requirements file exists. The declared Python project metadata is under `examples/python/pyproject.toml`, so the implementation must establish the intended shared dependency source before changing job commands.

## Decisions

- Reuse declared repository dependency metadata instead of adding a second unmaintained package list.
- Ensure each job installs the project scripts' actual dependencies, including pytest and coverage tools where tests require them.
- Validate every job's commands in an isolated clean checkout; preserve current stages, job rules, coverage output, and artifacts.

## Compatibility and Rollback

No application interface changes. If the chosen metadata cannot serve the root scripts without pulling example-only dependencies, add a minimal authoritative root manifest and update jobs together. Revert the CI and dependency metadata edits together if clean-checkout jobs cannot resolve the expected packages.

## Verification

Run the GitLab CI jobs in a clean checkout or equivalent container and confirm validate, Python tests, coverage artifact, and export behavior.
