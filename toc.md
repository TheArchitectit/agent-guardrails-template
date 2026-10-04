# Table of Contents

Flat listing of documentation in this repo. For keyword lookup, see
[index-map.md](index-map.md). Historical docs are in
[docs/archive/](docs/archive/) and not listed here.

## Root

- [README.md](README.md) — project overview
- [platform-current-state.md](docs/platform-current-state.md) — source-verified platform baseline
- [CHANGELOG.md](CHANGELOG.md) — release history
- [CONTRIBUTING.md](CONTRIBUTING.md) — how to contribute
- [CLAUDE.md](CLAUDE.md) — Claude Code context
- [index-map.md](index-map.md) — keyword → file lookup
- [toc.md](toc.md) — this file

## docs/

### getting-started/
- [quick-setup.md](docs/getting-started/quick-setup.md)
- [how-to-apply.md](docs/getting-started/how-to-apply.md)
- [agent-guardrails.md](docs/getting-started/agent-guardrails.md)

### architecture/
- [system-architecture.md](docs/architecture/system-architecture.md)

### designs/
- [halt-conditions-design.md](docs/designs/halt-conditions-design.md)

### mcp-server/
- [tools-reference.md](docs/mcp-server/tools-reference.md) — index of all 58 registered tools
- [tools/core-validation.md](docs/mcp-server/tools/core-validation.md) — 7 tools: bash / file-edit / git validation, scope, session
- [tools/workflow-and-git.md](docs/mcp-server/tools/workflow-and-git.md) — 8 tools: commit, push, regression, production-first
- [tools/halt-attempts-content.md](docs/mcp-server/tools/halt-attempts-content.md) — 12 tools: provenance, three strikes, halt, classification
- [tools/teams-and-advisors.md](docs/mcp-server/tools/teams-and-advisors.md) — 10 tools: project lifecycle, assignment, advisors
- [tools/conditional-and-integrations.md](docs/mcp-server/tools/conditional-and-integrations.md) — 21 conditionally registered tools
- [version-migration.md](docs/mcp-server/version-migration.md) — migration overview
- [python-to-go-migration.md](docs/mcp-server/python-to-go-migration.md)
- [migration-breaking-changes.md](docs/mcp-server/migration-breaking-changes.md)
- [migration-procedures.md](docs/mcp-server/migration-procedures.md)
- [migration-rollback.md](docs/mcp-server/migration-rollback.md)
- [migration-examples.md](docs/mcp-server/migration-examples.md)
- [migration-troubleshooting.md](docs/mcp-server/migration-troubleshooting.md)

### specs/
- [INDEX.md](docs/specs/INDEX.md)
- [09-system-roadmap-and-phase-gates.md](docs/specs/09-system-roadmap-and-phase-gates.md) — master product roadmap and release gates
- [10-mcp-protocol-and-fastmcp-parity.md](docs/specs/10-mcp-protocol-and-fastmcp-parity.md) — MCP contract/test parity informed by FastMCP
- [11-authorization-scopes-and-roles.md](docs/specs/11-authorization-scopes-and-roles.md) — scoped API keys intersected with named roles
- [12-web-exposure-boundary.md](docs/specs/12-web-exposure-boundary.md) — public routes, CORS, trusted proxies, exposure profiles
- [13-webhook-ssrf-hardening.md](docs/specs/13-webhook-ssrf-hardening.md) — delivery-time DNS validation, pinning, and redirect policy
- [14-optional-oap-guardrail-checker/spec.md](docs/specs/14-optional-oap-guardrail-checker/spec.md) — conditional OAP checker; no identity/effect authority
- [15-secure-cross-product-method/spec.md](docs/specs/15-secure-cross-product-method/spec.md) — Cross-Product Evidence Auth v1 for both sides
- [AUTH-01-mcp-endpoint-auth.md](docs/specs/AUTH-01-mcp-endpoint-auth.md) — /mcp bearer auth, merged
- [07-versioned-policy-pack-governance.md](docs/specs/07-versioned-policy-pack-governance.md) — proposed policy distribution and trust contract
- [08-control-plane-composition-and-evidence.md](docs/specs/08-control-plane-composition-and-evidence.md) — proposed composition and evidence contract
- [guardrail-gaps-2026/STATUS.md](docs/specs/guardrail-gaps-2026/STATUS.md) — spec vs shipped code, requirement by requirement
- [guardrail-gaps-2026/index.md](docs/specs/guardrail-gaps-2026/index.md)

### reboot/
- [INDEX.md](docs/reboot/INDEX.md) — reboot planning material (site, story, features, stars)
- [01-site.md](docs/reboot/01-site.md)
- [02-story.md](docs/reboot/02-story.md) — draft, held for review
- [03-features.md](docs/reboot/03-features.md)
- [04-stars.md](docs/reboot/04-stars.md)
- [changes/INDEX.md](docs/reboot/changes/INDEX.md) — F001–F100 change files

### releases/
- [INDEX.md](docs/releases/INDEX.md)

### superpowers/
- [INDEX.md](docs/superpowers/INDEX.md) — working designs and plans

### integrations/
- [agents-and-skills-setup.md](docs/integrations/agents-and-skills-setup.md)
- [claude-code.md](docs/integrations/claude-code.md)
- [opencode.md](docs/integrations/opencode.md)
- [cursor.md](docs/integrations/cursor.md)
- [windsurf.md](docs/integrations/windsurf.md)
- [copilot.md](docs/integrations/copilot.md)

### teams/
- [team-structure.md](docs/teams/team-structure.md)
- [team-tools.md](docs/teams/team-tools.md) — tools overview
- [team-tools-management.md](docs/teams/team-tools-management.md)
- [team-tools-phase-gates.md](docs/teams/team-tools-phase-gates.md)
- [team-tools-agent-mapping.md](docs/teams/team-tools-agent-mapping.md)
- [team-tools-validation.md](docs/teams/team-tools-validation.md)
- [team-tools-errors.md](docs/teams/team-tools-errors.md)
- [team-tools-workflows.md](docs/teams/team-tools-workflows.md)

### rules/
- [writing-rules.md](docs/rules/writing-rules.md)
- [extracting-rules.md](docs/rules/extracting-rules.md)

### standards/
See [standards/INDEX.md](docs/standards/INDEX.md) — prompting, logging,
testing, infrastructure, documentation, dependency governance.

### workflows/
See [workflows/INDEX.md](docs/workflows/INDEX.md) — execution, review,
commits, pushes, rollback, regression prevention, escalation.

### security/
See [security/INDEX.md](docs/security/INDEX.md) — API, code, config,
container, database, dependency audits.

### advisors/
See [advisors/INDEX.md](docs/advisors/INDEX.md) — cost, privacy, resilience.

### enterprise/
See [enterprise/INDEX.md](docs/enterprise/INDEX.md) — charter, governance,
ownership, release calendar, tech stack.

### specs/
#### guardrail-gaps-2026/
- [index.md](docs/specs/guardrail-gaps-2026/index.md) — overview and implementation order
- [01-prompt-injection-defense.md](docs/specs/guardrail-gaps-2026/01-prompt-injection-defense.md) — prompt injection defense (Critical)
- [02-semantic-content-filtering.md](docs/specs/guardrail-gaps-2026/02-semantic-content-filtering.md) — semantic content filtering (Critical)
- [03-runtime-sandbox-isolation.md](docs/specs/guardrail-gaps-2026/03-runtime-sandbox-isolation.md) — runtime sandbox isolation (Important)
- [04-multi-agent-safety-policies.md](docs/specs/guardrail-gaps-2026/04-multi-agent-safety-policies.md) — multi-agent safety policies (Important)
- [05-indirect-prompt-injection.md](docs/specs/guardrail-gaps-2026/05-indirect-prompt-injection.md) — indirect prompt injection handling (Important)
- [06-regulatory-compliance-mapping.md](docs/specs/guardrail-gaps-2026/06-regulatory-compliance-mapping.md) — regulatory compliance mapping (Nice-to-have)

### Domain guides
- [ui-ux/ui-ux-standard.md](docs/ui-ux/ui-ux-standard.md)
- [accessibility/accessibility-guide.md](docs/accessibility/accessibility-guide.md)
- [spatial/spatial-computing-ui.md](docs/spatial/spatial-computing-ui.md)
- [ethical/ethical-engagement.md](docs/ethical/ethical-engagement.md)
- [ai-dev/ai-assisted-dev.md](docs/ai-dev/ai-assisted-dev.md)
- [state/state-management.md](docs/state/state-management.md)
- [generative/generative-asset-safety.md](docs/generative/generative-asset-safety.md)
- [monetization/monetization-guardrails.md](docs/monetization/monetization-guardrails.md)
- [multiplayer/multiplayer-safety.md](docs/multiplayer/multiplayer-safety.md)
- [analytics/analytics-ethics.md](docs/analytics/analytics-ethics.md)
- [deployment/cross-platform-deployment.md](docs/deployment/cross-platform-deployment.md)

### Top-level
- [troubleshooting.md](docs/troubleshooting.md) — index to topic guides
- [status.md](docs/status.md) — project status

## mcp-server/ (root)
- [README.md](mcp-server/README.md)
- [API.md](mcp-server/api.md) — API overview (links to api-*.md)
- [DEPLOYMENT_GUIDE.md](mcp-server/deployment-guide.md)

## Other
- [pi-extension/README.md](pi-extension/README.md) — pi coding agent extension
- [ide/README.md](ide/README.md) — IDE extensions
- [cmd/team-cli/README.md](cmd/team-cli/README.md) — team CLI
- [skills/shared-prompts/](skills/shared-prompts/) — canonical guardrail prompts
- [examples/](examples/) — per-language guardrail examples
