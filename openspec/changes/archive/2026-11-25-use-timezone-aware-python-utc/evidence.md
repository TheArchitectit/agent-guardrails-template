# QA-03 UTC timestamp migration evidence

## Changes

- `scripts/team_manager.py` now obtains UTC instants through `datetime.now(timezone.utc)`, keeps report and audit boundaries aware during arithmetic and filtering, and normalizes legacy naive or explicit-offset parsed timestamps before comparisons. Log, metric, audit, report, and health timestamp fields retain their names and terminal `Z` form; request ID digits retain their existing UTC format.
- `scripts/log_aggregator.py` now creates aware UTC relative windows and parses `Z`, explicit-offset, and legacy naive timestamps into aware UTC values. Query filtering, chronological sorting, and summary range selection compare UTC instants; summary fields continue returning the original timestamp strings.
- Tests cover structured logs and metrics with the existing timestamp keys and `Z` output; aware report windows; legacy naive metric compatibility; relative windows; `Z`, offset, and naive ISO parsing; mixed-offset filtering, sorting, and summary ranges. A mixed fixture with malformed string, `None`, and numeric timestamps confirms window filters reject invalid times, summaries skip them for range endpoints while retaining entry counts, and unfiltered queries retain the records with stable invalid-last ordering.

## Validation

- Pre-work `python scripts/regression_check.py --all` — passed: `No potential regressions detected`.
- Focused affected tests on Python 3.12 with deprecations promoted to errors: `python3 -m pytest -q -W error::DeprecationWarning scripts/test_team_manager.py scripts/test_log_aggregator.py` — **71 passed, 44 subtests passed**.
- Full Python script suite on Python 3.12: `python3 -m pytest scripts/ -q` — **276 passed, 68 subtests passed**. Pytest reported no warnings. The independent baseline replay had reported 363 deprecation warnings across 268 tests; the migrated code paths also passed the focused warnings-as-errors run.
- Python test discovery: `python3 scripts/check_python_tests.py --repo .` — **4 discovered suites, all covered** by `.guardrails/python-test-floor.json`; `python3 scripts/test_check_python_tests.py` — **14 passed**.
- Pinned OpenSpec CLI `@fission-ai/openspec@2.2.0`, `openspec validate --all --strict` — **64 passed, 0 failed**.
- Independent replay at draft `1ff334b04a7e5b70eb3ae1fc2519ac06d2e26e1a`
  with the four candidate Python files passed 276 tests and 68 subtests.
  Python 3.12.16 promoted deprecation warnings to errors for the entire
  script suite. Test discovery and all 14 discovery controls also passed.
- A canonical Linux replay with Go 1.26.11 and Python 3.14.10 passed the
  MCP integration JSON parsing test and the checked-in manifest verification.
