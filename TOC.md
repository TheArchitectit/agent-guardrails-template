# Table of Contents

Flat listing of documentation in this repo. For keyword lookup, see
[INDEX_MAP.md](INDEX_MAP.md). Historical docs are in
[docs/archive/](docs/archive/) and not listed here.

## Root

- [README.md](README.md) — project overview
- [CHANGELOG.md](CHANGELOG.md) — release history
- [CONTRIBUTING.md](CONTRIBUTING.md) — how to contribute
- [CLAUDE.md](CLAUDE.md) — Claude Code context
- [INDEX_MAP.md](INDEX_MAP.md) — keyword → file lookup
- [TOC.md](TOC.md) — this file

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
- [tools-reference.md](docs/mcp-server/tools-reference.md)
- [version-migration.md](docs/mcp-server/version-migration.md) — migration overview
- [python-to-go-migration.md](docs/mcp-server/python-to-go-migration.md)
- [migration-breaking-changes.md](docs/mcp-server/migration-breaking-changes.md)
- [migration-procedures.md](docs/mcp-server/migration-procedures.md)
- [migration-rollback.md](docs/mcp-server/migration-rollback.md)
- [migration-examples.md](docs/mcp-server/migration-examples.md)
- [migration-troubleshooting.md](docs/mcp-server/migration-troubleshooting.md)

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
- [API.md](mcp-server/API.md) — API overview (links to api-*.md)
- [DEPLOYMENT_GUIDE.md](mcp-server/DEPLOYMENT_GUIDE.md)

## Other
- [pi-extension/README.md](pi-extension/README.md) — pi coding agent extension
- [ide/README.md](ide/README.md) — IDE extensions
- [cmd/team-cli/README.md](cmd/team-cli/README.md) — team CLI
- [skills/shared-prompts/](skills/shared-prompts/) — canonical guardrail prompts
- [examples/](examples/) — per-language guardrail examples
