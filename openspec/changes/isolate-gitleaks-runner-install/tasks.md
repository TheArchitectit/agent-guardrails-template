# Tasks

## 1. Isolate and verify scanner setup

- [ ] 1.1 Replace the fixed shared Gitleaks install path with a unique child under `RUNNER_TEMP`; use mocked downloads/binaries and one shared synthetic parent to verify concurrent installers allocate distinct children, remove only their own directories, and preserve a pre-existing `/tmp/gitleaks.tmp` sentinel.
- [ ] 1.2 Verify the v8.24.3 archive against its upstream-published checksum manifest and check `gitleaks version`; use mocked failures to verify download, checksum, and version errors return nonzero without reporting a clean scan.

## 2. Verify scan outcome and cleanup

- [ ] 2.1 Run a known-clean fixture and a separate safe synthetic-detection fixture; verify the expected outcomes and that logs expose no fixture value or credential.
- [ ] 2.2 Verify push/PR range, full-history input, licensed behavior, scan-step `GITHUB_TOKEN`/`GITLEAKS_LICENSE` inputs, read-only permission scope, scan arguments, `--exit-code=2`, and disabled SARIF upload remain unchanged; verify `GITHUB_PATH` carries the verified binary across steps and cleanup removes only the current job's temporary directory.
