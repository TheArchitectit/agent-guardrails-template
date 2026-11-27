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
PyYAML, Go, a C compiler for the Go race suite, Git, `jq`, the Docker CLI,
and `govulncheck` v1.10.0. Install the pinned scanner with
`GOTOOLCHAIN=local go install golang.org/x/vuln/cmd/govulncheck@v1.10.0`. A missing prerequisite is reported before the checks
start. Use a Go toolchain that satisfies the module declarations. Go may download
the required toolchain or modules according to the local Go configuration.
Local Go tests use a 30-minute per-test timeout to allow for slower machines.

Python coverage is produced as `coverage.xml` in the checkout and validated
for a non-empty report. The local command does not upload it. The
Docker CLI transport test runs without a daemon. The Docker/Podman
engine-backed integration test is skipped by default. To opt in, set
`MCP_DOCKER_CONTEXT_ENGINE=docker` or `MCP_DOCKER_CONTEXT_ENGINE=podman`; the
workflow-compatible `MCP_DOCKER_CONTEXT_DAEMON=available` selects Docker. An
opt-in requires the selected engine and daemon to be available. The command
only checks engine availability; the test owns any temporary resources it
creates. No engine command runs by default.

Use [local event gates](local-event-gates.md) with an explicit push or pull
request commit range to run the event-scoped secret and lint checks locally.
Neither command creates GitHub comments or artifacts, or attests a remote runner.
The Team Validation parity job invokes these same entry points on its selected
runner; manual dispatch has no event range, so its event gate is explicitly
skipped. See the [workflow guide](workflows.md) for job and runner details.
The current suite verifies Docker build-context transport; it does not build
the production MCP image or prove a Codex connection. The associated work and
evidence are tracked in the [OpenSpec change](../../openspec/changes/qualify-ci-environment-and-mcp-client/proposal.md).
