# Evidence and verification limits

## Source inspection

- Chain implementation/configuration: `mcp-server/internal/guardrails/multi_agent.go`, `multi_agent_registry.go`, `safety_chain_file.go`.
- Registered MCP tools: `mcp-server/internal/mcp/tools_multi_agent.go`; dispatch/registry surfaces are under `mcp-server/internal/mcp/`.
- Existing evidence pointers include `safety_chain_file_test.go`, `tools_multi_agent_test.go`, and `multi_agent_findings_test.go`. They were identified by source search and were not run for this package.
- `docs/specs/guardrail-gaps-2026/STATUS.md` explicitly states that YAML `agent_registry`/`cross_agent_policies` are not implemented and that `code_review_agent` needs an unavailable backend.

## Not run / not established

No integrated MCP runtime acceptance was run for TPL-46. No real multi-agent environment or user data was accessed. This file records source/test pointers only and does not claim that the required runtime matrix passed.
