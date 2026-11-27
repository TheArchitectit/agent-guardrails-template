# Local CI validation

Run the repository's declared Python and Go test/build commands and their
discovery gates from a local checkout:

```bash
scripts/run_local_ci.sh
```

The command reads `.guardrails/python-test-floor.json` and
`.guardrails/test-floor.json`, runs every listed test and build command, and
enforces each positive test floor. It also runs the deterministic Regression
Guard checks and their self-tests, plus the approved AIGGP charset gate and
self-test at commit `7d17d6f1baa4357110e57d3a6ffd55fbacdd77f7`. The AIGGP source
must already be checked out cleanly in the ignored `.devgate/` directory at
that exact commit. The command does not clone it or install anything.

Prerequisites must already be available: Python 3, `pytest`, `pytest-cov`,
PyYAML, Go, a C compiler for the Go race suite, Git, and `jq`. A missing prerequisite is reported before the checks
start. Use a Go toolchain that satisfies the module declarations. Go may download
the required toolchain or modules according to the local Go configuration.
Local Go tests use a 30-minute per-test timeout to allow for slower machines.

Python coverage is produced as `coverage.xml` in the checkout and validated
for a non-empty report. The local command does not upload it. The
Docker/Podman context integration test is skipped by default. To opt in, set
`MCP_DOCKER_CONTEXT_ENGINE=docker` or `MCP_DOCKER_CONTEXT_ENGINE=podman`; the
workflow-compatible `MCP_DOCKER_CONTEXT_DAEMON=available` selects Docker. An
opt-in requires the selected engine and daemon to be available. The command
only checks engine availability; the test owns any temporary resources it
creates. No engine command runs by default.

This is a local code-validation command, not a replacement for event-bound
workflow evidence. It does not create a PR or push range, run PR-only analysis
or comments, attest the actual UCS03 runner host, or upload GitHub artifacts.
