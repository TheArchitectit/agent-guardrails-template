# Design: CI environment and MCP client qualification

## Context

At main `e8949f5fed43bbeda75343bd48b9e68985a1eb9f`, the workflow selector is `runs-on: ${{ fromJSON(vars.CI_RUNNER_LABELS || '["ubuntu-latest"]') }}` and `scripts/check_runner_isolation.py` statically enforces the approved selector and fork guard. The selector's unset-variable branch has now run in a separate public proof repository. The template's main CI continues to use its existing UCS03 variable.

`scripts/run_local_ci.sh` runs inventory-driven Python and Go checks, docs and CLI checks, Go advisory classification, the pinned charset gate, deterministic regression gates, and Docker CLI context-transport verification. The Docker engine-backed context test is opt-in. `tests/test_mcp_docker_context_transport.py` explicitly says that it does not build an image. The repository therefore lacks evidence that the production Dockerfile produces a runnable server image and that an actual Codex client can initialize it, enumerate its tools, and invoke one safely.

## Decisions

1. Treat parity as an explicit matrix of named deterministic checks and their owning commands. Every supported environment must run the same required check or record a blocking reason; a skipped or unavailable check is never recorded as passed.
2. Keep GitHub event behavior, artifacts, comments, branch protection, and remote-runner attestations as environment-specific checks. Do not pretend a local run reproduces those GitHub services.
3. Verify the hosted fallback in an isolated repository where `CI_RUNNER_LABELS` is absent. Verify configured-label routing on the registered fleet separately. Retain job-level fork guards on every job that can select private runners.
4. Build the actual `mcp-server/deploy/Dockerfile` into a unique disposable image. Record the source commit, Dockerfile, builder/runtime versions, image digest, and build result. Exercise that image with a real Codex client over the supported MCP transport; require successful `initialize`, `tools/list`, and one non-mutating tool call with a bounded expected result.
5. Keep synthetic Docker context checks distinct from image-build and protocol acceptance. Run synthetic boundary fixtures on AI03, AI04, and AI05 in unique temporary directories. Do not mount `/var/run/docker.sock`, use privileged containers, or modify runner services. The engine-backed build/client test must use an explicitly approved disposable engine and test environment.
6. Store redacted receipts. Do not store credentials, environment values, request secrets, or full client transcripts when a summary is sufficient.

## Proposed unattended lab integration

The current workflow does not dispatch the image harness to AI03–AI05. A
candidate design is a narrowly scoped test worker on an authorized AI03–AI05
host, invoked by the UCS03 job using a short-lived GitHub Actions identity. The
worker would validate the token issuer, audience, repository, workflow, event,
ref, run identity and expiry against an explicit trust policy; reject replays;
fetch the exact candidate SHA independently; and run only the fixed, bounded,
rootless image/protocol harness. Its receipt would bind source SHA, image
digest, host, protocol stages and outcome to the requesting run. It would not
expose a shell, a host container socket, reusable private-host credentials, or
model credentials to the job. Unattended execution must not invoke Codex or a
model; the separately authorized Codex smoke remains on demand.

This is a design option, not a deployed gate. The lab currently has no such
worker or trusted runner-to-AI05 endpoint, and routing, TLS, token-claim
validation, denial behavior, resource limits and isolation still need design
review and live negative testing. Repository-only events are not automatically
trusted merely because they can trigger a workflow. Implement the worker and
fleet wiring through the infrastructure repository before reporting per-build
runtime coverage.

## Risks and verification

Hosted runner image drift can reveal missing prerequisites that are hidden by the UCS03 image. The parity matrix must name version requirements and distinguish tool installation from test execution. Self-hosted runners share a fleet, so tests must be serialized or use unique disposable paths and avoid cleanup outside their own fixtures. MCP initialization success alone is insufficient; tool enumeration and one real tool call are required to demonstrate client compatibility.

## Rollout

First publish the workflow guide and parity matrix. Add or adjust tests and workflow steps only after the matrix identifies a concrete missing gate. Run the exact checks locally, on a GitHub-hosted runner, and on the registered fleet. Exercise synthetic isolation on AI03-AI05, then build and inspect the production image and run the real Codex client acceptance in disposable environments. Archive only after evidence identifies every source SHA, image digest, runner/environment, client version, and result.
