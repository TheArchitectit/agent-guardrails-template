# Design

## Context

Secret Validation run `45556758743`, job `139010560787`, failed in “Run Gitleaks” while installing v8.24.3 because `/tmp/gitleaks.tmp` already existed. The action failed before the scanner ran. The runner may retain scratch across jobs and may execute jobs concurrently.

## Decisions

- Allocate a unique install directory beneath the job-provided `RUNNER_TEMP` using an atomic temporary-directory operation. Do not use a fixed shared path and do not delete or overwrite any pre-existing path owned by another job.
- Download the pinned Gitleaks v8.24.3 release and its upstream-published checksum manifest into the job-owned directory; verify the binary archive against that release's checksum before execution, and verify the installed binary reports v8.24.3. Do not require a signature unless the pinned release actually publishes verifiable signature metadata.
- If installation and scan use separate workflow steps, append the exact job-owned binary directory to the workflow-provided `GITHUB_PATH` so the later step receives it through GitHub's documented step environment. Do not depend on shell-local `PATH` changes or a runner-wide preinstalled binary. Alternatively, install and scan in one step using the explicit path.
- Preserve the workflow's current push and same-repository pull-request events, full-history checkout, scan range/arguments, licensed scan behavior, `GITHUB_TOKEN` and `GITLEAKS_LICENSE` inputs, read-only `contents` permission, disabled SARIF artifact upload, redaction, and nonzero failure semantics. If installation, integrity verification, version verification, or scanning fails, Secret Validation must fail; an install failure must never be reported as a successful clean scan.
- Test concurrency with two simultaneous installers using the same synthetic `RUNNER_TEMP` parent and assert each atomic allocation creates a distinct child. Also test a stale sentinel at the old fixed path that remains unchanged, a clean fixture, and a safe synthetic-detection fixture with output redacted or metadata-only. Do not place real credential material in fixtures or logs.

## Compatibility and Rollback

The scanner version and scan scope remain unchanged. Runner setup requires access to the pinned release and its integrity metadata, or an approved internal mirror containing the same verified artifacts. If installation changes cause failures, restore the prior pinned action only after preserving a non-colliding and fail-closed install path; never restore destructive cleanup of the shared scratch path.

## Verification

Mock downloads and scanner binaries in tests; do not require network access or real credentials. Use a shared synthetic `RUNNER_TEMP` parent for two simultaneous installs; assert distinct children, exact job-owned cleanup, and that a pre-existing `/tmp/gitleaks.tmp` sentinel remains unchanged. Verify the v8.24.3 checksum against its upstream-published checksum manifest and assert version/integrity failures return nonzero. Exercise clean and safe synthetic-detection fixtures. Review workflow wiring to ensure `GITHUB_PATH` carries the job-owned binary across steps, scanner installation does not depend on ambient `PATH`, and events, full-history checkout, scan range/arguments, `GITHUB_TOKEN`/`GITLEAKS_LICENSE` values, read-only `contents` permission, `--exit-code=2`, and disabled SARIF upload remain intact. Review logs to ensure fixture contents and credentials are absent.
