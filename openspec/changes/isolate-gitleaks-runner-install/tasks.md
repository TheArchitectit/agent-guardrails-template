# Tasks

## 1. Isolate and verify scanner setup

- [x] 1.1 Give the pinned action an atomically allocated, job-owned `TMPDIR` beneath `RUNNER_TEMP`; use a mocked action/process harness and one shared synthetic parent to verify concurrent jobs get distinct directories, clean only their own directories, and preserve a pre-existing `/tmp/gitleaks.tmp` sentinel.
- [ ] 1.2 **Open upstream capability gap:** the pinned action downloads and extracts v8.24.3 but has no checksum-manifest verification or post-install `gitleaks version` check. Verify the installed action source and release metadata, then decide whether a future approved action release adds these checks or the workflow replaces the action with a maintained equivalent. Do not claim checksum/version verification for this change.

## 2. Verify scan outcome and cleanup

- [x] 2.1 Run a known-clean fixture and a separate safe synthetic-detection fixture; verify the expected outcomes and that logs expose no fixture value or credential. (Local v8.24.3 CLI only; pinned action installer remains unverified.)
- [x] 2.2 Verify push/PR range, full-history checkout, licensed behavior, scan-step `GITHUB_TOKEN`/`GITLEAKS_LICENSE` inputs, read-only permission scope, scan arguments, `--exit-code=2`, and disabled SARIF upload remain unchanged; verify the pinned action resolves its binary under the explicit job-owned `TMPDIR` and cleanup removes only that directory.
