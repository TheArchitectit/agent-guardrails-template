# Tasks

## 1. Exercise the integrated multi-agent request path

- [ ] 1.1 Start an isolated MCP server through its production startup constructor with a temporary, digest-recorded safety-chain configuration.
- [ ] 1.2 Invoke the registered agent-validation tool and verify the configured chain replaces defaults, ordered validators run, and results bind to the request/config revision.
- [ ] 1.3 Exercise validator unavailable/error paths and all supported conflict strategies; verify outcomes and audit evidence are consistent with actual behavior.
- [ ] 1.4 Test unsupported `agent_registry`/`cross_agent_policies` YAML and unavailable LLM validator behavior; record these as refused/not implemented, not coverage.
- [ ] 1.5 Produce redacted evidence with request/config IDs, validator/strategy, and output digests but no synthetic payload text or credentials.

## 2. Review coverage boundaries

- [ ] 2.1 Independently trace the tool from registry through dispatch to the real registry/chain and result.
- [ ] 2.2 Confirm claims are limited to supported MCP paths and do not imply generic host or Pi integration.

## Workflow follow-up

Do not treat unit/loader tests or a built registry alone as runtime proof. Keep all tasks open until the integrated request path and reviewer checks pass.
