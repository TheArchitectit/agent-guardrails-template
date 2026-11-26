# TPL-53 MCP session topology inventory — 2026-11-25

## Scope and source identity

This inventory is pinned to template `main` commit `7c52f2091b9b0fa508500daa1d93acd3f091a396`. The hashes below are SHA-256 over raw Git blob bytes at that commit, not Windows checkout bytes. No deployment was started and no production or Windows pilot was changed.

The repository's `DEPLOYMENT_PROFILE` values (`local`, `tailnet`, `public`) describe ingress/bind posture. They do not declare an MCP application-session mode or bind replicas and routing to session state. The bounded mode evidenced by the Go fixture is the stateless Streamable HTTP transport in one MCP server process.

## Published profile inventory

| Entry point | Checked-in posture | Session-topology consequence |
|---|---|---|
| Root `docker-compose.yml` | MCP service has a single static service declaration, `container_name`, restart policy `unless-stopped`, local profile default, local-bridge attestation, and host publishes on `127.0.0.1`. No replica count or session-mode field is declared. | The file describes one service instance by default. Its restart policy does not persist the Go session map. No restart/resume behavior was exercised here. |
| `mcp-server/deploy/docker-compose.example.yml` | Also sets the `local` profile and isolated-local-bridge attestation, and publishes MCP/Web on loopback. No replica count or session-mode field. | Local exposure configuration only; it does not define token continuity. |
| `mcp-server/deploy/podman-compose.yml` | One `mcp-server` service with a fixed container name and `unless-stopped`; loopback bind default. It omits `DEPLOYMENT_PROFILE`, whose config default is `local`. No replica count or session-mode field. | Same process-local token limitation. A restart creates a new in-memory registry. |
| `mcp-server/deploy/k8s-deployment.yaml` | Deployment starts at 2 replicas with rolling updates; the HPA declares 2–10 replicas. The Service is `ClusterIP`; the Service has no `sessionAffinity`. The manifest has no `DEPLOYMENT_PROFILE`, session-mode, or deployment-preflight setting. | This is a multi-pod example with no session-affinity contract. The Go application-token map is not shared between pods. It is not qualified for workflows that depend on that token across requests. |

`ValidateDeploymentBinding` and `ValidateIngressPosture` validate exposure settings (bind host, TLS/private ingress evidence, and direct-backend denial for nonlocal profiles). They do not inspect Kubernetes replica counts, Service affinity, or the MCPServer session map. No session-mode field or renderer/preflight comparison between replicas and session storage was found in the checked-in config/deployment paths. This leaves task 1.2 open.

## Per-process state, identity, and restart boundary

`MCPServer` owns `sessions map[string]*models.Session`; `NewMCPServer` creates a fresh map. `guardrail_init_session` creates a random token and records the token plus creation/expiry timestamps in that map. The expiry is 8 hours. Lookup checks only map presence and expiry; registration prunes expired entries. A process replacement therefore loses these entries (derived from the constructor and in-memory map; no deployment restart was performed).

The MCP transport is separately configured with `server.WithStateLess(true)`. The HTTP test fixture sends separate initialize/list/call requests through a real `httptest.Server`. This verifies a one-process stateless transport sequence; it does not make the application `session_token` portable across MCPServer instances.

`models.Session` has token, project slug, agent type, client version, creation time, and expiry fields, but no principal or credential owner. `handleInitSession` reads `user_id` and `environment` from tool arguments, stores neither in the session record, and returns them in `SessionInfo`. Request authentication does resolve a bearer credential to a server-controlled principal on each HTTP request, and tool authorization runs before dispatch. The session token record is not bound to that principal. Session-using resource handlers validate token existence/expiry and use the token as a resource key; no cross-principal owner check was established.

The application session map has no revoke operation in the reviewed paths. Existing revocation tests concern bearer credentials and an in-process auth registry, not revocation of a `guardrail_init_session` token across pods. The expired/revoked/mismatched-identity scenario is therefore not proven; task 2.1 remains open.

## Existing fixtures and bounded local control

Executed in disposable `golang:1.26.11-bookworm` image `sha256:ea47ee45f1284d60f9cc09cb2c3bb4f508c256b640bd8bb30342ff13d1adcb52` against a read-only source mount, with Go dependencies fetched through the public Go module proxy:

- `go test ./internal/mcp -run '^(TestStreamableHTTPListingSequence|TestSessionLifecycle)$' -count=1` — PASS (`0.023s`). This is the bounded synthetic control. `TestStreamableHTTPListingSequence` starts a disposable local `httptest.Server`, exercises initialize/list/call over the real Streamable HTTP handler, verifies an authorized call registers an application token, and verifies a credential lacking scope is denied without that side effect. It is not a Docker/Kubernetes app deployment.
- `go test ./internal/config -run '^(TestValidateIngressPosture|TestValidateIngressPostureRejectsWeakRecordFile|TestValidateCouplesIngressPostureToStartup|TestIngressPreflightPath|TestProfileBindingLocal|TestProfileBindingPublic|TestProfileBindingTailnet|TestProfileBindingValidationRules)$' -count=1` — PASS (`0.011s`). These fixtures cover ingress/bind profile validation, not session mode or replica routing.

`TestSessionLifecycle` covers valid, unknown, expired, no-expiry, and pruning behavior in one map. The checked-in `TestLiveRevocationPropagationBound` and `TestSharedRegistryRevocationPropagatesInProcess` use in-process HTTP servers and a shared in-process credential registry; they do not exercise a second MCPServer, process restart, or application-token revocation. No fixture found exercises mismatched session owner identity or multi-pod routing.

No application deployment image or rendered topology digest was produced by this synthetic control. The profile/deployment drill, restart/reinitialization, revocation, identity separation, and mode-specific routing remain open under task 1.3. No fresh-connection auto-reinitialization behavior is claimed.

## Acceptance status

- Task 1.1 is complete as a source inventory of the observed profiles and bounds.
- Task 1.2 remains open: current preflight does not reject replicas/routing that conflict with process-local token state.
- Task 1.3 remains open: the bounded test is one process and does not cover replacement, cross-process movement, revocation, or identity separation.
- Task 2.1 remains open until an independent reviewer addresses all three spec scenarios. In particular, scenario 3 lacks stored principal binding and a demonstrated expired/revoked/mismatched-owner denial.

## Raw Git blob SHA-256 pins

| Source path at pinned commit | SHA-256 |
|---|---|
| `docker-compose.yml` | `c78d926f39714fb863ebd21f685a2ec17bddab9dcc19b88b9236d53d87ee5786` |
| `mcp-server/deploy/docker-compose.example.yml` | `060a84c419682dc7f162bfbfe25e9db06b0900e04ef8163d6ce4223e72ffc9c6` |
| `mcp-server/deploy/podman-compose.yml` | `b433ac7307f45cdbfc77224bf8714063e590a400f1b7014ef52da6f3fd7b7bea` |
| `mcp-server/deploy/k8s-deployment.yaml` | `1cd62a94b9b78da2e49b7c283c1bd4260e2a6f7766ebcf6776835bd30cd31aa3` |
| `mcp-server/internal/config/profile.go` | `ee2bc853991596ddeeef7e89e4a62d988e4c1cfbf3f67a80c816c139518b2eb7` |
| `mcp-server/internal/config/config.go` | `45c11f78b83b8e11d23acefbae079dc89c36f229566037296244c84785a1c9da` |
| `mcp-server/internal/config/ingress.go` | `d6071da61f09e9c6f648df69c77f5f378e49476e7dbc889046661c9d1373e682` |
| `mcp-server/internal/config/profile_test.go` | `8aa9ff412dc83f073b81467de5b453df743d22cff2778ccad9b04e78c3bdd163` |
| `mcp-server/internal/config/ingress_test.go` | `eba666e1f73b079dda255c1ebfac1e48cada01de100254be4a704809cbb696fb` |
| `mcp-server/internal/mcp/server.go` | `130f07c014085987890253bb84f24505460ee87b27a64cde73e98e182e5e3bb8` |
| `mcp-server/internal/models/project.go` | `a5d7c90b9e5ce2ac78622e17d1a48a8dd57e2becc3cc311fe5f2cc05a2c6d718` |
| `mcp-server/internal/mcp/tools_extended_fileread.go` | `b3ea7be70d64afdd0714167a8bbe58080b7431b8c6c65c7d302e5928bc9d55fd` |
| `mcp-server/internal/mcp/session_test.go` | `124b5d9084638f2f2afb3c5a698344b986b8cefce321a1931ef264b455b2b1bc` |
| `mcp-server/internal/mcp/streamable_http_authz_test.go` | `1d62a16451e31efb82fd1c638677875799c9732060c8bf464e0239db61c852f2` |
| `mcp-server/internal/mcp/streamable_http_listing_test.go` | `a9b05211efe3e11cc4b3e6054c977bdd12eb6d97d55f0c57edf6f802307af43e` |
| `mcp-server/internal/mcp/streamable_http_revocation_test.go` | `ea90d1f101f704ed8995955fe9c1eae944234b0b708df9b6f7972fe7b6ff4878` |
| `mcp-server/internal/mcp/shared_registry_revocation_test.go` | `ce08b96f1691a6c3c6a374f461c94c04ab882af24d5363a4e3c8bd7af43ba3af` |
| `mcp-server/go.mod` | `343cbf1a2bbc1c6fcf9c5922f31fe6b9606638e5ef9c8f9970647eff0d408535` |
