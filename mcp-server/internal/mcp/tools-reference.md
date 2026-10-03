# Guardrail MCP Server — Tools Reference (moved)

**This file previously documented 13 MCP tools that do not exist.**

An earlier version of this document described `guardrail_detect_injection`,
`guardrail_scan_text_batch`, `guardrail_sandbox_execute`,
`guardrail_sandbox_config`, `guardrail_validate_agent_output`,
`guardrail_check_agent_constraints`, `guardrail_resolve_conflicts`,
`guardrail_scan_external_content`, `guardrail_mark_provenance`,
`guardrail_check_provenance`, `guardrail_generate_compliance_report`,
`guardrail_check_compliance` and `guardrail_collect_evidence`.

None of those names appear in any `.go` file in this repository, and none is
registered in `toolList()` (`tools_registry.go`) or dispatched from
`server.go`. The library code behind some of them does exist but is not
exposed — for example the injection-detection pipeline in
`internal/guardrails/injection_detection.go` and the sandbox in
`internal/guardrails/sandbox.go` are fully built libraries with no MCP tool
in front of them. The "Error Cases" section was likewise invented: the real
handlers return `{"error": "..."}` payloads, not the symbolic codes it
listed.

Because this file sat inside the Go package, it read as authoritative to
anyone working in `mcp-server/` — including coding agents. It has been
reduced to this notice.

## Where the real reference lives

**[docs/mcp-server/tools-reference.md](../../../docs/mcp-server/tools-reference.md)**

That reference is generated from the source and covers all 58 registered
tools — the 37 core `guardrail_*` tools plus 21 conditionally registered
ones (vision, webhooks, budgets, agent lifecycle) — with each tool's actual
parameters, output shape and policy behaviour.

It also records where the published tool schemas disagree with the handlers,
which is worth knowing before you integrate a client.

## Before you document a tool

Verify it exists:

```bash
# Is it dispatched?
grep -n "case \"<tool_name>\"" mcp-server/internal/mcp/server.go

# Is it registered?
grep -n "<tool_name>" mcp-server/internal/mcp/tools_registry.go \
                mcp-server/internal/mcp/tools_core_list*.go
```

A tool that fails both checks is not callable, however plausible the design
sounds. Several proposed tools in `docs/specs/guardrail-gaps-2026/` remain
in exactly that state.