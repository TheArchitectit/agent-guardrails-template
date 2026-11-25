# Proposal: Verify Pi consumer guardrail behavior at runtime

## Why

The Pi extension contains Four Laws checks, classifier integration, permission decisions, and output validation. Unit tests for isolated helpers do not establish that the real Pi event lifecycle invokes each control in the expected order, stops a denied operation before its effect, or distinguishes actual redaction from a side-effect warning. This is a consumer-specific boundary and cannot be inferred from MCP scan/action APIs.

## What Changes

- Require an isolated Pi consumer test that drives representative permitted and denied read/edit/bash events through the registered handlers.
- Bind observed classifier and permission decisions to the effect or denial that followed.
- Require output fixtures to prove whether redaction actually changes content; warning-only behavior must remain labeled as observation, not redaction enforcement.

## Scope mapping

- `classification-and-mediation/fail-closed destructive boundaries` and `trace-action-and-ingress-boundaries` (RUN-03) address general action mediation; TPL-44 scopes proof to the Pi extension lifecycle and its registered hooks.
- `semantic-filtering-evidence-and-conformance/auditable redaction` and `implement-semantic-redaction-evidence` (RUN-05) define the underlying redaction contract; TPL-44 requires proof of the actual Pi consumer behavior and its output semantics.
- `docs/specs/guardrail-gaps-2026/02-semantic-content-filtering.md`, `docs/integrations/pi.md`, and the Pi extension tests identify the consumer surface.

## Capabilities

### New Capabilities
- `pi-consumer-runtime-evidence`: prove consumer-level guardrail decisions at the Pi event boundary.

## Impact

Pi extension integration fixtures and a redacted runtime evidence record. No MCP behavior is treated as Pi evidence.
