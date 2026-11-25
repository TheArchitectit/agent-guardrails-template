# CI-01 Verification Evidence

- Added root `requirements.txt` with PyYAML, pytest, and pytest-cov dependencies
  required by root validation scripts and the configured Python test job.
- Replayed in the supplied `aiggp-pr41-lead:20261125` container with a fresh
  Python virtual environment. Manifest installation succeeded. The validate
  and status commands passed against a disposable `ci-verify` project initialized
  with valid team assignments. The Python job passed 268 tests and 68 subtests,
  and wrote `coverage.xml`. The export job produced a non-empty
  `teams-export.json`.
- Parsed `ci/gitlab-ci.yml` with PyYAML after the update. The merge-request and
  default-branch rules, `coverage.xml` Cobertura path, `teams-export.json`
  artifact, and manifest install in each Python job (validate, test, export)
  were confirmed.
- The caller's project data is a prerequisite of the existing validation and
  status commands: a clean template checkout has no `.teams/<project>.json`.
  The local replay created and populated disposable test data for verification;
  CI commands and project behavior were not changed.

Independent local replay used the staged tree
`1bb3f92c545c43c9fc7be2c94207a223b7d15310` and a fresh Python environment.
It verified dependency installation runs first in all three GitLab Python
jobs and that their trigger conditions match the baseline. Validation and
status passed with disposable project data. The suite passed 268 tests
and 68 subtests, generated coverage XML, and exported a nonempty JSON
artifact. The known Python deprecation warnings remain tracked in QA-03.
