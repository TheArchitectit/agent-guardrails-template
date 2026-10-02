# AUTH-01: authenticate the /mcp endpoint
Status: implemented on branch, awaiting fleet-CI gate and owner verdict.
Problem: `/mcp` (StreamableHTTP, 0.0.0.0) accepted unauthenticated requests; MCP_API_KEY was validated at startup but never enforced there.
Change: internal/mcp/server.go wraps `/mcp` with `requireBearer` (constant-time compare, empty key fails closed, 401 + WWW-Authenticate).
Acceptance (machine-checkable):
1. `cd mcp-server && go test ./internal/mcp/ -run Bearer` passes (7 table cases plus real initialize: 401 without key, 200 with key).
2. Fleet gate: start via docker-compose; `curl -s -o /dev/null -w '%{http_code}' -X POST http://localhost:8080/mcp` prints 401; same request with a valid bearer and a JSON-RPC initialize body prints 200.
Rollback: revert the commit; clients then need no key (unsafe).
Compatibility: existing clients must send `Authorization: Bearer <MCP_API_KEY>`. Update host configs (integrations/copilot/.vscode/mcp.json already does).
