# Guardrails Control Plane

Composition root for the guardrails architecture. This repo owns no rules and
no enforcement code itself; it pins the independently versioned pieces as
submodules and documents how they compose.

## Architecture

```
guardrails-control-plane (this repo, composition root)
├── engines/guardrail-mcp   -> guardrail-mcp          (runtime MCP enforcement)
├── engines/devgate         -> DevGate-Agentic-Framework (static/CI gates)
└── policy/packs            -> guardrail-policy-packs (versioned rule packs)
```

- **guardrail-mcp** - runtime behavioral enforcement: validates bash/edit/git
  operations, prompt-injection defense, content safety, sandboxing. Extracted
  from `mcp-server/` in agent-guardrails-template.
- **DevGate-Agentic-Framework** - static/CI quality gates: pattern/semantic
  scans, regression check, test isolation, deploy gate, drift scans. Usable
  standalone.
- **guardrail-policy-packs** - the single versioned source for rules (Four
  Laws, halt conditions, scope validation, ...): one `core` pack plus domain
  packs for games, infrastructure, and web.
- **Mission Control** remains the dashboard and ingestion layer; it consumes
  findings, it is not composed here.

## Dependency rule

One-way only: control plane -> engines and policy packs. Engines and packs
never depend back on the control plane. Both engines load rules from the
pinned policy packs; neither owns a private copy of the rules.

## Pin policy

Submodules pin commits (or tags, once engines cut releases). Consumers of this
composition pin a control-plane commit plus their own project overlay pack.
See docs/ARCHITECTURE.md for the full design and migration plan.
