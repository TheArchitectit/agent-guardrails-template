# CI-02 Verification Evidence

## Version selection

- The module directives are `go 1.26.11` in `mcp-server/go.mod` and
  `go 1.23.2` in `cmd/team-cli/go.mod`. The legacy templates previously
  declared Go 1.21.
- GitLab now uses `golang:1.26.11`, sets `GOTOOLCHAIN=local`, and asserts the
  selected `go env GOVERSION`. Jenkins uses the exact module version with
  `GOTOOLCHAIN=go<version>+path`, checks the selected version, and fails with a
  clear message if it differs. The `+path` selector disables download fallback.
- Positive selection passed for Go 1.26.11 in the cached Linux toolchain image.
  A missing `go1.29.0` with `+path` failed with `cannot find "go1.29.0" in
  PATH`, confirming no download fallback. A Go 1.26.11 runner selected for the
  CLI version was rejected by the exact-version check.
- In a clean detached checkout of 109c952 with only the candidate Jenkinsfile
  overlaid, the `golang:1.23.2-bookworm` image passed the Jenkins CLI selector
  and `make build`. The configured `cmd/team-cli/team` artifact was written at
  7,266,600 bytes.

## MCP race-suite runtime finding

- The first full MCP race/coverage replay exited 1. Its disposable container
  was removed before the full failure log could be preserved; the exact timeout
  panic was not captured.
- A narrower `TestNegativeSuitesKillInjectedBypass` replay completed with an
  explicit 20-minute Go test timeout in 1,035.716 seconds (17 minutes 15.7
  seconds), passed, and killed all 52 injected bypasses. This is longer than
  Go's default 10-minute per-test timeout. The original failure is therefore
  consistent with a default test timeout; that cause is an inference because
  the original terminal panic text was not captured.
- The legacy MCP test commands now use a bounded 30-minute test timeout while
  retaining all packages, race, and coverage flags. A clean full-suite replay
  completed successfully in isolated Linux containers with Go 1.26.11.

## Candidate job replays

- The full GitLab MCP command passed on a canonical LF archive of base
  `109c95213a1a64518707dcfd27d3b81a2433021d`, with the candidate selector
  `GOTOOLCHAIN=local`, exact `go env GOVERSION` check, and
  `go test -timeout=30m ./... -v -race -coverprofile=coverage.out -count=1`.
  All 21 test-bearing packages passed and the coverage summary was 43.9%.
  This run used the pre-TPL-51 validation source and did not emit a race report;
  it does not negate the separately reproduced race finding.
- The full Jenkins MCP command passed on the same canonical source with only
  the TPL-51 `safe_regex.go` and native test changes overlaid. The path-only
  selector `GOTOOLCHAIN=go1.26.11+path`, exact version check, and
  `go test -timeout=30m ./... -v -race -count=1` all completed successfully;
  all 21 test-bearing packages passed. `internal/mutationkill` took 259.110s
  and `internal/validation` took 1.230s. This is a local shell replay, not a
  live Jenkins run.
- With the TPL-51 source overlay, the GitLab selector (`GOTOOLCHAIN=local`)
  and exact Go 1.26.11 assertion passed for the validation package using
  `go test -timeout=30m ./internal/validation -race -coverprofile=coverage.out
  -count=1`; package coverage was 30.4%. `CompilePattern` and
  `ClearRegexCache` each measured 100.0% statement coverage. This focused
  source replay supplements, but does not replace, the full baseline-source
  GitLab run above.
- Negative controls ran without network access. A Go 1.23.2 image under the
  GitLab selector failed with `Required Go 1.26.11, got go1.23.2`; a Jenkins
  Go 1.23.2 agent requesting `go1.26.11+path` failed because that toolchain
  could not be found in PATH. Earlier CLI mismatch and CLI build checks are
  recorded above.

Sanitized local receipts are retained in `guardrails-template-ci02-evidence/`:
`gitlab-mcp-full-baseline.log`, `jenkins-mcp-fixed-source.log`,
`gitlab-validation-fixed.log`, `gitlab-validation-fixed.cover.out`,
`gitlab-wrong-toolchain.log`, and `jenkins-missing-toolchain.log`.
No live GitLab or Jenkins runner acceptance is claimed.

## Permanent regression

- `tests/regression/test_ci_go_toolchain_regression_FAIL_6a1595be.py` reads
  the checked-in GitLab and Jenkins configuration and their module directives.
  It executes the extracted selector/test/build shell commands with isolated
  fake `go` and `make` tools; no compiler or dependency download is needed.
- The corrected candidate passed all three regression tests. Matching GitLab,
  Jenkins MCP, and Jenkins CLI versions reached their test/coverage/build
  markers. GitLab version mismatch, Jenkins MCP mismatch, and unavailable CLI
  version failed before those markers. Assertions also preserve GitLab race,
  coverage, and coverage report commands; Jenkins race test, CLI build, and
  configured `cmd/team-cli/team` artifact path; and the absence of an example
  Go job.
- The same regression file was run against the original CI files from base
  `109c95213a1a64518707dcfd27d3b81a2433021d` in a disposable fixture checkout.
  All three tests failed on the missing/old selectors, as expected. The
  original configuration had GitLab Go 1.21 and no `GOTOOLCHAIN` guard, and
  Jenkins had no per-module Go selectors.
