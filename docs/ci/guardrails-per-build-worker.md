# Per-build isolated Guardrails acceptance

`guardrails-build-call.yml` invokes the public reusable workflow for every
branch push, protected-main push, tag push, same-repository pull request,
manual dispatch, and rerun. The reusable workflow is pinned by commit and
contains the fixed client checkout and approved AIGGP checkout. It never checks
out candidate source in the OIDC-capable job.

The canonical Guardrails repository always runs this gate for its trusted
pushes, tags, same-repository PRs, dispatches, and reruns; missing service
configuration fails closed. Other consumers opt in with the operator-owned
`GUARDRAILS_WORKER_SERVICE` variable. The endpoint is resolved by the pinned
AIGGP `templates/ci/service-endpoint.sh` helper, with the runner-provisioned
`/etc/guardrails-worker-services.json` stable-name registry as fallback. The
registry supplies a tailnet HTTPS endpoint; no static host address is stored in
the public repository.

Consumers configure only `GUARDRAILS_WORKER_SERVICE`,
`GUARDRAILS_WORKER_TLS_CERT_SHA256`, `GUARDRAILS_WORKER_CA_PEM`, and
`GUARDRAILS_WORKER_RECEIPT_PUBLIC_KEY`. Empty or malformed values fail closed.

The worker validates OIDC issuer, audience, repository, workflow, event, ref,
SHA and run claims, then independently retrieves the exact SHA in a disposable
VM. Observed acceptance returned a nonzero tool inventory and a benign
`guardrail_advisor_list` result containing nine advisors, plus image digest and
resource-cleanup evidence. Unattended acceptance never invokes a model. The VM
adapter and endpoint are deployment-owned, so repository tests cannot claim
live worker coverage.

