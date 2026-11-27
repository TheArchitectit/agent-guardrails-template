# CI workflows and local validation

This guide describes the repository's GitHub Actions checks, their trigger scopes, and the local commands that cover deterministic validation. The workflow files remain the source of truth for exact job steps and branch/event filters.

## Workflow map

| Workflow | Triggers | Main responsibility | Local counterpart |
| --- | --- | --- | --- |
| `team-validation.yml` | Push and pull request to `main` or `master`; manual dispatch | Team CLI checks, inventory-driven Python and Go suites/builds, coverage, MCP/security integration rows, and trusted repository validation | `scripts/run_local_ci.sh` covers the local CLI, Python/Go inventories, advisory checks and deterministic gates; it does not upload artifacts or create event comments |
| `team-validation-fleet.yml` | Push to `main`; manual dispatch | Trusted-main validation on the configured runner, with bounded concurrency | `scripts/run_local_ci.sh`; local execution does not attest the registered runner or its isolation |
| `documentation-check.yml` | Push to `main` or `master`; pull request; manual dispatch | Markdown line caps, internal links, required sections, and formatting | `scripts/run_local_ci.sh` |
| `go-advisories.yml` | Push and pull request to `main` or `master` | Pinned Go vulnerability scan and advisory classification | `scripts/run_local_ci.sh` (requires the pinned `govulncheck`) |
| `guardrails-lint.yml` | Push and pull request to `main`, `master`, or `develop`; manual dispatch | Changed-file scope, forbidden files, commit format, AI attribution, compliance summary | `scripts/run_local_event_gates.sh` for an explicit event range; summary artifacts remain CI-only |
| `secret-validation.yml` | Push to `main` or `develop`; pull request to `main` or `develop` | Gitleaks history scan, `.env` and credential filenames, and hardcoded-secret patterns | `scripts/run_local_event_gates.sh` for an explicit push or PR commit range |
| `regression-guard.yml` | Pull request opened/synchronized/reopened; push to `main` or `master` | Deterministic regression checks, inventory discovery, repository isolation policy, failure registry, and PR feedback | `scripts/run_local_ci.sh` runs deterministic checks and self-tests; PR analysis/commenting remains CI-only |
| `release-coverage.yml` | Push to `main` | Requires main to equal its annotated, documented release tag (except the documented generated-report-only case) | No local command publishes or certifies release coverage; inspect its exact Git history and tag rules before release |
| `release.yml` | Push of a `v*` tag | Verifies the tag and release notes, then creates and reads back a non-draft GitHub Release | No local equivalent publishes a release |
| `guardrails-build-call.yml` | Canonical trusted branch/tag push, same-repository PR, manual dispatch and rerun; consumer opt-in by service name | Calls the pinned reusable isolated image/protocol acceptance workflow; canonical configuration failures are blocking, consumers opt in with `GUARDRAILS_WORKER_SERVICE` | No local command can reproduce the worker trust boundary; see [per-build worker](guardrails-per-build-worker.md) |

All nine workflows use the approved runner selector. Unless repository variable `CI_RUNNER_LABELS` supplies a JSON array of runner labels, jobs use `ubuntu-latest`. For example, an administrator may set the variable to the JSON array `["self-hosted", "Linux", "X64", "guardrails-ucs03"]`. The actual setting is repository administration state; this document does not change it.

The Team Validation job **Local CI parity (configured runner)** executes the same `scripts/run_local_ci.sh` entry point used locally and records both the configured labels and actual runner identity. It also invokes `scripts/run_local_event_gates.sh` for push and pull-request event ranges, before running the main local suite. A manual dispatch has no push or pull-request range, so that step is explicitly omitted and reported in the job log. The job does not build or start an MCP container. A successful job establishes the result on its assigned runner; verifying the hosted and self-hosted modes requires separate runs under each configuration.

Jobs that can reach a private self-hosted runner include a job-level fork guard. Fork pull requests must not be dispatched to that pool. The regression-guard R19-13 checks enforce the reviewed selector and guard structure; a passing static test does not itself prove a runner was assigned. The GitHub-hosted fallback was exercised separately in the isolated proof documented in the OpenSpec change.

## Local commands

Run deterministic project checks with:

```bash
scripts/run_local_ci.sh
```

Run event-scoped secret and lint checks only after pinning the event's exact commit range:

```bash
scripts/run_local_event_gates.sh \
  --event push \
  --base-sha <event-before-sha> \
  --head-sha <event-after-sha>
```

For a pull request, provide `--event pull_request`, `--base-ref`, `--base-sha`, `--head-sha`, and `--pr-head-sha`. The helper fails closed when the checkout is dirty or the supplied range cannot be resolved. See [local validation](local-validation.md) and [local event gates](local-event-gates.md) for prerequisites and limits.

The Docker CLI transport test sends a synthetic build context to a disposable loopback receiver and verifies filtering. It is a transport/context test, not a production image build or runtime test. Engine-backed Docker/Podman context checks are opt-in. A one-time real-image and Codex test on AI05 is recorded in the [OpenSpec evidence](../../openspec/changes/qualify-ci-environment-and-mcp-client/evidence.md). The reusable Codex smoke command requires an already-running disposable endpoint and client authorization and is on-demand only; unattended builds intentionally skip model calls. The per-build worker gate resolves a stable service name through the pinned AIGGP helper or runner service registry, records the actual runner, verifies the exact PR merge SHA or push SHA, and uploads the complete signed receipt. Find the full catalog of specifications and proposals in the [OpenSpec index](../../openspec/INDEX.md).

## GitHub-only evidence

GitHub Actions artifacts, step summaries, pull-request comments, protected-branch decisions, actual runner assignment and isolation attestations, and release publication must be verified from their GitHub job records. A local pass does not substitute for them. Likewise, a hosted default selector proof does not establish that a configured self-hosted pool is available or that fork guards behave correctly in every workflow; those are separate CI checks.
