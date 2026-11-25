# Tasks

## 1. Register the root suite

- [x] 1.1 Add a `tests/` row to `.guardrails/python-test-floor.json` with pytest setup, the normal root-suite command, always-applicable rationale, and floor 11; verify the row parses and the discovery gate passes.
- [x] 1.2 Run `python3 scripts/test_check_python_tests.py` and verify the existing undocumented-suite fixture still fails closed without changing the gate implementation.

## 2. Verify root-suite execution

- [x] 2.1 Run `python3 -m pytest tests/ -v` and `python3 -m pytest tests/ --collect-only -q -o addopts=""`; verify the tests run with the explicit Docker conditional skip as applicable and the collection count meets the declared floor of 11.
