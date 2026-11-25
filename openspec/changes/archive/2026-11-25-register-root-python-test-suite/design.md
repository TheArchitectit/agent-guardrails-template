# Design

## Context

The discovery and execution loops already read `.guardrails/python-test-floor.json`; the workflows need no additional hardcoded suite path. The root `tests/` directory currently collects 11 cases. One Docker context integration case is included in collection and is skipped unless its explicit daemon condition is met.

## Goals / Non-Goals

**Goals:**

- Make the existing root regression suite discoverable and runnable through the inventory-driven CI path.
- Set a non-vacuous floor equal to the observed collection count.

**Non-Goals:**

- Change discovery logic, workflow control flow, existing suite rows, or root tests.
- Require an available Docker daemon to collect or run the suite.

## Decisions

- Add exactly one row for `tests/` with `python3 -m pytest tests/ -v` as its test command and `python3 -m pip install pytest` as setup. This follows the inventory's existing self-contained suite-command pattern and declares the only required runner package.
- Set `min_tests` to 11, the current pytest collection count. Pytest includes the conditional Docker case in collection even when its daemon-dependent body is skipped, so the floor guards discovery without asserting that Docker executed.
- Leave the gate and CI workflows unchanged. Both already discover suite directories, validate inventory coverage, and execute each inventory row.

## Risks / Trade-offs

- [The root suite gains or loses collected cases] → the floor gate catches reductions below 11; intentional collection changes require reviewing and updating the floor with new collection evidence.
- [A runner has no Docker daemon] → the existing Docker test's explicit skip condition keeps the suite runnable while its test remains counted at collection.

## Migration Plan

Add the row, run the discovery gate and its self-tests, then run the root test command and compare pytest's collected count with the declared floor. Rollback consists of removing the row if the suite itself is removed; the fail-closed gate will then require its test files to be removed or registered elsewhere.
