# Proposal

## Why

The Python discovery gate already finds the root `tests/` suite, but the test-floor inventory does not register it. Consequently, the hosted coverage check fails before CI can run the root regression tests.

## What Changes

- Register the root `tests/` suite in `.guardrails/python-test-floor.json` with its pytest setup and command.
- Set the collection floor to the current 11 collected cases, including the Docker integration case that may be skipped when a daemon is not explicitly available.
- Specify that every discovered test suite must have a runnable inventory row and a positive floor at least as high as the current collection.

## Capabilities

### New Capabilities

- `python-test-suite-inventory`: Declares how discovered Python suites are registered and protected by collection floors.

### Modified Capabilities

None. No existing OpenSpec capability specifies Python test-suite inventory coverage.

## Impact

- `.guardrails/python-test-floor.json` gains one row for the existing root `tests/` suite.
- OpenSpec gains the `python-test-suite-inventory` capability delta.
- The Python discovery gate and its existing self-tests remain unchanged.
