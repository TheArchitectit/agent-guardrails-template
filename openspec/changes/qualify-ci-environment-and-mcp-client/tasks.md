# Tasks

## 1. Establish the parity contract

- [ ] 1.1 Inventory every deterministic check in the nine workflows and map it to a local command and hosted/self-hosted job; record prerequisites, supported versions, event-only behavior, and artifact-only behavior.
- [x] 1.2 Run the exact `CI_RUNNER_LABELS` unset fallback expression in an isolated public GitHub repository and verify the assigned runner is GitHub-hosted `ubuntu-latest`; receipt: [`evidence.md`](evidence.md).
- [ ] 1.3 Exercise the configured `CI_RUNNER_LABELS` value on the registered UCS03 pool and verify fork pull requests remain ineligible for private-runner assignment. Retain the static R19-13 tests and trusted-main checks.
- [ ] 1.4 Reconcile local, GitHub-hosted, and UCS03 check results. Add missing parity commands only where the source workflows require them. Preserve GitHub-only event comments, artifacts, attestations, and publication semantics.
- [ ] 1.5 Qualify synthetic context isolation on AI03, AI04, and AI05 in unique temporary paths; assert exclusions and required inputs without host sockets, privileged execution, or runner changes.

## 2. Build and exercise the real MCP image

- [x] 2.1 Build the production `mcp-server/deploy/Dockerfile` from the reviewed source commit in a disposable engine and record the image digest and builder/runtime versions; see [AI05 evidence](evidence.md).
- [x] 2.2 Start the built image using the supported MCP transport and connect with an actual Codex MCP client. Verify `initialize`, `tools/list`, and one benign `tools/call`; record Codex version, negotiated protocol, tool name, redacted result summary, image digest, and exit status; see [AI05 evidence](evidence.md).
- [ ] 2.3 Verify the same documented build/client flow in local, GitHub-hosted, and registered self-hosted environments, or record a specific unavailable prerequisite without treating that environment as passed.
- [ ] 2.4 Review logs and image contents for synthetic secret leakage and unintended build-context files. Remove only disposable images and fixtures created by this qualification.
- [ ] 2.5 Wire deterministic exact-image MCP protocol acceptance to each candidate build on a trusted, isolated test host. Publish SHA- and digest-bound results and explicit failure or unavailable states; do not expose host credentials or sockets to untrusted PR code. Unattended builds skip model-backed Codex calls; run those explicitly on demand and record them separately.

## 3. Document and review

- [x] 3.1 Prepare the user-facing workflow guide with trigger scope, workflow responsibilities, runner selection, fork protection, local counterparts, and CI-only evidence.
- [ ] 3.2 Independently review all receipts and the parity matrix, run the pinned strict OpenSpec validator and relevant repository gates, then archive only when every required scenario is evidenced.
