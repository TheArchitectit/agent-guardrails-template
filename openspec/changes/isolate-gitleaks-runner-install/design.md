# Design

## Context

Secret Validation run `45556758743`, job `139010560787`, failed in “Run Gitleaks” while installing v8.24.3 because `/tmp/gitleaks.tmp` already existed. The action failed before the scanner ran. The runner may retain scratch across jobs and may execute jobs concurrently.

## Decisions

- Allocate a unique install directory beneath the job-provided `RUNNER_TEMP` using an atomic temporary-directory operation. Do not use a fixed shared path and do not delete or overwrite any pre-existing path owned by another job.
- Preserve the official action at its existing immutable pin. Its `src/gitleaks.js` at `ff98106e4c7b2bc287b24eaf42907196329070c7` uses Node `os.tmpdir()` for both `gitleaks-${version}` and `gitleaks.tmp`. On Linux, Node honors `TMPDIR`; provide a unique, job-owned directory so the action's existing fixed child names cannot collide across jobs.
- Explicitly set `GITLEAKS_VERSION=8.24.3`, matching the action's default at the pin, and pass the allocated directory as the action step's `TMPDIR`. The action retains its own versioned install, cache, scan, license, event, API, and reporting behavior.
- Source review of the same immutable pin confirms its effective scan flags remain `detect --redact -v --exit-code=2 --report-format=sarif --report-path=results.sarif --log-level=debug`; push scans the event commit range (or one commit when identical), and pull requests fetch their commit range through the token-authenticated API. The existing `GITLEAKS_ENABLE_UPLOAD_ARTIFACT=false` only disables artifact upload; SARIF generation remains enabled.
- **Open integrity gap:** the pinned upstream action source downloads and extracts the archive but does not fetch or verify the release checksum manifest and does not run `gitleaks version` after installation. Do not claim those checks. Keep task 1.2 open until upstream supports them or a future approved replacement can preserve the existing action contract.

- Preserve the workflow's current push and same-repository pull-request events, full-history checkout, scan range/arguments, licensed scan behavior, `GITHUB_TOKEN` and `GITLEAKS_LICENSE` inputs, read-only `contents` permission, disabled SARIF artifact upload, redaction, and nonzero failure semantics. If installation, integrity verification, version verification, or scanning fails, Secret Validation must fail; an install failure must never be reported as a successful clean scan.
- Test concurrency with two simultaneous installers using the same synthetic `RUNNER_TEMP` parent and assert each atomic allocation creates a distinct child. Also test a stale sentinel at the old fixed path that remains unchanged, a clean fixture, and a safe synthetic-detection fixture with output redacted or metadata-only. Do not place real credential material in fixtures or logs.

## Open Acceptance

Checksum-manifest verification and post-install version verification remain open. The immutable action pin lacks both checks, and this change intentionally retains its installer so its event handling, license behavior, token use, and reporting contract stay intact. Resolve task 1.2 only after upstream support or a separately reviewed replacement can preserve those behaviors.

## Compatibility and Rollback

The scanner version and scan scope remain unchanged. Runner setup requires access to the pinned release and its integrity metadata, or an approved internal mirror containing the same verified artifacts. If installation changes cause failures, restore the prior pinned action only after preserving a non-colliding and fail-closed install path; never restore destructive cleanup of the shared scratch path.

## Verification

Run `pytest tests/test_gitleaks_temp_isolation.py` against synthetic runner-temp parents, including a symlinked parent; assert concurrent allocations resolve to distinct canonical job-owned directories, scoped cleanup preserves a synthetic `gitleaks.tmp` sentinel, and out-of-parent cleanup is rejected. This test never touches the global `/tmp/gitleaks.tmp` and never invokes a scanner. Review the pinned upstream action source and preserve its event-specific scan range and flags, token/license inputs, read-only `contents` permission, explicit v8.24.3, and disabled SARIF artifact upload. Treat checksum-manifest and post-install version verification as open because the pinned upstream action does not implement them. The isolation test does not exercise Gitleaks scans; keep fixture scan task 2.1 open.
