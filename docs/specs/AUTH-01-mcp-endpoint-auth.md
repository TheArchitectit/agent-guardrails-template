# AUTH-01: authenticate the /mcp endpoint
Status: implemented and merged to `main` (`e46c4b7`). Behaviour verified in CI; the docker-compose leg of the fleet gate remains unrun — see Acceptance.
Problem: `/mcp` (StreamableHTTP, 0.0.0.0) accepted unauthenticated requests; MCP_API_KEY was validated at startup but never enforced there.
Change: internal/mcp/server.go wraps `/mcp` with `requireBearer` (constant-time compare, empty key fails closed, 401 + WWW-Authenticate).
Acceptance (machine-checkable):
1. `cd mcp-server && go test ./internal/mcp/ -run Bearer` — **met.** 7 table cases plus `TestMCPEndpointBehindBearer`, which drives a real StreamableHTTP server: 401 without a key, 200 with a valid bearer and a JSON-RPC `initialize` body. Runs on every push via the `test-go` job.
2. Fleet gate: start via docker-compose; `curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/mcp` prints 401; same request with a valid bearer and a JSON-RPC initialize body prints 200. — **behaviour covered, deployment leg not run.** `TestMCPEndpointBehindBearer` asserts the same 401/200 pair in-process, but the containerised, real-transport path has not been exercised on a host with a container runtime. Worth running once against docker-compose before treating the endpoint as verified in deployment.
Rollback: revert the commit; clients then need no key (unsafe).
Compatibility: existing clients must send `Authorization: Bearer <MCP_API_KEY>`. Update host configs (integrations/copilot/.vscode/mcp.json already does).
