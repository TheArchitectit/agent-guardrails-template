# Design

## Context

The legacy Spec 04 and `STATUS.md` describe an implemented chain engine, configured YAML loading and MCP tools, plus explicitly missing `agent_registry`/`cross_agent_policies` YAML and an unavailable LLM validator. Tests such as `safety_chain_file_test.go` and multi-agent MCP tests cover important paths, but TPL-46 asks for an integrated runtime record showing which chain and conflict strategy actually governed tool outcomes.

## Goals / Non-Goals

**Goals:** exercise the real MCP tool registration/dispatch path with a temporary configured registry/chain, conflict inputs, safe/unsafe outputs, and unavailable/error validators; bind observations to the loaded configuration.

**Non-Goals:** claim unimplemented agent-registry YAML, claim `code_review_agent` exists, change policy behavior, or generalize MCP results to Pi/OpenCode/CrewAI or all agent frameworks.

## Decisions

- Use an isolated MCP server process or in-process server built with the production startup constructor and a temporary `SAFETY_CHAINS_CONFIG_FILE`.
- Demonstrate that a declared chain replaces built-ins; invoke the registered validation tool and verify each expected step, order, decision, unavailable status, and failure behavior.
- Exercise each supported conflict strategy with controlled competing results and assert the selected output and audit record.
- Include a negative control with an unknown validator/unsupported chain behavior and verify startup/validation refuses it without reporting a pass.
- Bind evidence to source revision, config digest, chain ID, validator identities, request ID, output digest, and result. Exclude synthetic payload text and secrets.

## Risks / Trade-offs

- [A loader test is misreported as enforcement] -> invoke the actual registered tool or transport entry point after startup.
- [A configured chain is parsed but not selected] -> use a chain that differs observably from built-ins and assert the returned step sequence.
- [Scope silently expands to unimplemented features] -> list unsupported registry YAML and LLM review validator as explicit not-claimed items.

## Migration Plan

Run a disposable server fixture and evidence review. If the real tool path cannot supply requested observations, update capability status as incomplete; do not fill gaps by testing helper functions only.
