# VS Code + GitHub Copilot wiring (advisory)

Copy `.vscode/mcp.json` into a project, start the server (`mcp-server`, needs Postgres and Redis, see `docker-compose.yml`),
then in Copilot Chat agent mode enable the `guardrails` tools.

## What this does and does not do
- **Advisory.** Copilot may call `guardrail_*` tools. Nothing forces it to. It can skip them.
- **Checked operation / Blocking: not enforced here.** Blocking needs a tested hook or wrapper on a named Copilot/VS Code version. Until a test record exists, the compatibility matrix stays UNKNOWN.
- Add `.github/copilot-instructions.md` (already in this repo) so Copilot is told to call `guardrail_pre_work_check` before edits. That is still a request, not enforcement.
- Bind the server to localhost. Authentication on `/mcp` was not found in `cmd/server/main.go` or `internal/mcp/server.go`; verify before any network exposure.

## Verify
1. `tools/list` over the endpoint returns the tools in `site/src/data/tools.json`.
2. Call `guardrail_validate_bash` with a destructive command and record the verdict.
3. Record the VS Code and Copilot versions and the date in `site/src/data/compat.json`. Only then may a row leave UNKNOWN.
