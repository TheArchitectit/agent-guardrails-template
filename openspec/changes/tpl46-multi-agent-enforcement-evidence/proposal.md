# Proposal: Verify multi-agent enforcement through the registered request path

## Why

The audit distinguishes a wired multi-agent registry and safety-chain engine from proof that configured chain selection, validator ordering, conflict strategy, and failure behavior are enforced through the actual registered MCP tools. The legacy Spec 04 is a docs-only requirements source, not an indexed OpenSpec capability. Existing helper and loader tests do not by themselves provide one version-bound runtime evidence record for the integrated request path.

## What Changes

- Add an OpenSpec capability delta for active chain selection and ordered safety enforcement.
- Require a runtime acceptance matrix for conflict outcomes, validator unavailable/error behavior, and configured chain replacement.
- Bind tool responses and effect/no-effect observations to the registry/chain/config revision actually used.

## Scope mapping

- Primary source: `docs/specs/guardrail-gaps-2026/04-multi-agent-safety-policies.md`, Sec. 2.2 to Sec. 2.3 and Sec. 3.1.
- Runtime status source: `docs/specs/guardrail-gaps-2026/STATUS.md`, Sec. 04. It records chain loading/wiring, conflict strategies, and remaining non-implemented `agent_registry`/`cross_agent_policies` YAML as separate facts.
- Related standard requirements: `classification-and-mediation/fail-closed destructive boundaries` (RUN-03) covers generic action interception; TPL-46 specifically covers multi-agent registry, conflict resolution, and safety-chain MCP requests. TPL-44 covers the separate Pi host path.

## Capabilities

### New Capabilities
- `multi-agent-runtime-enforcement`: evidence the selected multi-agent chain and conflict policy govern the actual MCP request result.

## Impact

MCP multi-agent configuration, registered tool calls, and a redacted runtime conformance artifact. This package does not claim automatic coverage of arbitrary external agent frameworks.
