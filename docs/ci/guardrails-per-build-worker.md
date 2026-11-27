# Per-build MCP acceptance

For every branch or tag push that creates or updates a ref, the Guardrails
repository runs **Guardrails per-build acceptance**. It also runs when a
same-repository PR opens, reopens or changes. A PR builds GitHub's synthetic
merge commit. The signed receipt identifies that commit. Fork PRs skip the
private job before runner assignment.

To repeat the check for an existing revision, rerun its acceptance job or
dispatch **Guardrails per-build acceptance** on its ref. Manually dispatching
**Team Validation** or **Trusted Main Validation** also invokes acceptance.
The latter requires `main`. On pushes and PRs those manual-only jobs show as
skipped because the dedicated acceptance workflow owns the build.

## What runs where

1. A UCS03 runner checks out the fixed client at its pinned commit and verifies
   the approved AIGGP checkout. The job obtains a short-lived GitHub Actions
   identity and sends the requested SHA to AI05 over certificate-pinned TLS.
2. The AI05 worker verifies the identity and workflow pin, fetches the exact Git
   objects from the public repository, and attaches them read-only to a fresh
   KVM guest. The guest verifies the checkout and builds the production MCP
   Dockerfile with rootless Podman. Its PostgreSQL and Redis instances use
   synthetic data. The guest has no external network interface.
3. The probe applies migrations, checks health, initializes MCP, lists tools
   and calls `guardrail_advisor_list`. Acceptance requires a nonempty tool list
   and nine returned advisors. AI05 removes the per-run disks and signs the
   result. UCS03 verifies the signature, execution profile, source SHA, check
   and run IDs, image digest, protocol results and cleanup.
4. In the workflow run, open the acceptance job for its runner log and the
   signed JSON receipt under **Artifacts**. A failed probe produces a failed
   check. A signed terminal failure is also retained. Configuration or transport
   failures before the worker replies have no signed receipt.

Unattended acceptance uses **zero model tokens**. An operator can request the
real Codex client separately on an authorized application test host with
`GUARDRAILS_SMOKE_CODEX_AUTH=1 scripts/run_mcp_image_codex_harness.sh --receipt FILE --run-codex`.
See the [qualification evidence](../../openspec/changes/qualify-ci-environment-and-mcp-client/evidence.md).

## Consumer configuration

The lab uses its UCS03 self-hosted runner labels. Other repositories can run
the shared deterministic suites locally, on default GitHub-hosted Ubuntu, or
on their own runners. A consumer opting into a private worker must configure
a reachable self-hosted runner, a worker that authorizes that repository, and
these operator-owned Actions variables.

| Variable | Purpose |
| --- | --- |
| `GUARDRAILS_WORKER_SERVICE` | Stable name resolved by AIGGP's service resolver |
| `GUARDRAILS_WORKER_TLS_CERT_SHA256` | Certificate fingerprint |
| `GUARDRAILS_WORKER_CA_PEM` | Public CA certificate for HTTPS validation |
| `GUARDRAILS_WORKER_RECEIPT_PUBLIC_KEY` | Public Ed25519 key for signed receipts |

UCS03 supplies the resolver fallback through
`/etc/guardrails-worker-services.json`. Missing required configuration fails
the canonical lab job. Consumers that have not configured a worker skip this
optional private integration. See [local validation](local-validation.md).
