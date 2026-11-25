# Spec Delta

## ADDED Requirements

### Requirement: Multi-agent safety-chain selection is evidenced at the MCP boundary
<!-- id: tpl-46-chain-enforcement -->
A multi-agent enforcement claim SHALL identify the loaded chain configuration and demonstrate that the registered MCP validation request executes the selected ordered validators before its result is returned. Parsing or constructing a registry alone SHALL NOT establish enforcement. Unsupported or unavailable validators SHALL be reported as incomplete/refused, never as passed.

#### Scenario: configured chain governs a registered request
- **GIVEN** a disposable MCP server started with an explicit safety-chain configuration distinct from built-in defaults
- **WHEN** a registered agent-output validation request is dispatched
- **THEN** the evidence identifies the active configuration digest, chain, ordered validators, per-step results, final decision, and request boundary used

#### Scenario: unavailable validator does not become a pass
- **GIVEN** a required validator is unavailable or returns an error during chain execution
- **WHEN** the registered validation request completes
- **THEN** the result is incomplete or refused under the actual fail-closed contract and evidence does not claim all validators passed

### Requirement: Conflict resolution evidence identifies the applied strategy
<!-- id: tpl-46-conflict-resolution -->
A multi-agent conflict-resolution claim SHALL be tied to the actual registry/request result and the configured supported strategy. Test evidence SHALL distinguish supported strategies from unsupported or unimplemented configuration surfaces.

#### Scenario: competing agent results are resolved
- **GIVEN** a registered request receives controlled competing results and a supported conflict strategy
- **WHEN** the MCP conflict-resolution path evaluates them
- **THEN** the result and evidence identify the strategy, input result digests, selected/escalated outcome, and any enforced safety-chain decision

#### Scenario: unsupported multi-agent feature is not claimed
- **GIVEN** a caller requests unsupported `agent_registry`/`cross_agent_policies` YAML or an unavailable LLM review validator
- **WHEN** configuration or validation is evaluated
- **THEN** the unsupported feature is refused or identified as unavailable and is not included as exercised coverage
