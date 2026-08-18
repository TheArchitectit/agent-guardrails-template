# Agent Guardrails Template

> AI-first safety framework for agents building software at high velocity. Guardrails don't slow you down — they're your license to move fast.

[![Version](https://img.shields.io/badge/version-v3.4.0-blue.svg)](./CHANGELOG.md)
[![Go Implementation](https://img.shields.io/badge/Implementation-Go-blue.svg?style=flat&logo=go)](https://golang.org)
[![WCAG 3.0+](https://img.shields.io/badge/Accessibility-WCAG_3.0+_Silver-green.svg)](docs/accessibility/accessibility-guide.md)
[![Spatial Computing](https://img.shields.io/badge/Spatial-XR/VR/AR-blue.svg)](docs/spatial/spatial-computing-ui.md)

---

## What Is This?

Agent Guardrails Template is a set of safety rules and tooling for teams that let AI agents write most of their code. The idea is simple: give the agent clear boundaries up front, so it can spend its time building instead of second-guessing every file edit.

The centerpiece is a Go MCP server that checks bash commands, file edits, and git operations against your guardrails before they run. Everything else — the docs, the per-IDE skill files, the examples — is there to get those guardrails into your workflow with minimal setup.

### What You Actually Get

| Capability | What It Does |
|-----------|-------------|
| **Real-Time Guardrail Enforcement** | Go MCP server validates every bash command, file edit, git operation, and commit before execution |
| **CI/CD Policy Enforcement** | `POST /api/v1/policy/check` endpoint for gating PRs on guardrail violations — with line/column precision |
| **Webhook Notifications** | Real-time violation and halt event delivery via HMAC-signed webhooks with circuit breaker protection |
| **Token Budget Ledger** | Track AI API costs across Claude/GPT models, set team budgets, get alerts when thresholds are crossed |
| **Agent Lifecycle Management** | State machine (idle→planning→active→review→release) with transition validation and audit trails |
| **Multi-Agent Orchestration** | Patterns for MoA (Mixture of Agents), swarm intelligence, and autonomous tool use |
| **Cross-Platform IDE Integration** | Native skills and rules for Claude Code, Cursor, OpenCode, Windsurf, and GitHub Copilot — not generic prompts |
| **Token-Efficient Documentation** | 68+ modular docs (500-line max), index-map keyword lookup, `.claudeignore` for context savings |
| **Production Infrastructure** | PostgreSQL 16 + Redis 7, Docker Compose, CI/CD validation, 22-pattern secret scanning, regression prevention |
| **14 Language Examples** | Go, Rust, TypeScript, Python, Java, GDScript, Scala, R, C#, C++, PHP, Ruby, Swift, Dart/Flutter |
| **Ethical & Accessible by Default** | WCAG 3.0+ Silver compliance, dark pattern prevention, XR comfort zones, monetization ethics, multiplayer safety |

### Who This Is For

- **AI-first teams** — If agents write most of your code, you want them moving fast without breaking production.
- **Platform engineers** — Enforce infrastructure guardrails, prevent config drift, and keep test and production separated.
- **Compliance & security folks** — Documented safety processes that hold up in an audit.

### Why Constraints Speed Things Up

Left on their own, agents burn tokens asking themselves *"Is this safe to edit? Should I check first?"* That constant self-verification eats context and slows output.

With the boundaries defined up front, the agent knows what's allowed and just gets on with the work. Fewer rollbacks, faster iteration.

The analogy we keep coming back to: lane markers on a highway don't slow you down — they're the reason you can drive at full speed.

### The Four Laws of Agent Safety

1. **Read before editing** — Never modify code without reading it first
2. **Stay in scope** — Only touch files explicitly authorized
3. **Verify before committing** — Test and check all changes
4. **Halt when uncertain** — Ask for clarification instead of guessing

---

## Quick Start

```bash
# Clone the template
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template
```

Then see [QUICK_SETUP.md](docs/getting-started/quick-setup.md) for the 5-minute setup, or [HOW_TO_APPLY.md](docs/getting-started/how-to-apply.md) to apply guardrails to an existing repo.

---

## What's Included

### Core Safety (Mandatory)

| Document | Purpose |
|----------|---------|
| [AGENT_GUARDRAILS.md](docs/getting-started/agent-guardrails.md) | The Four Laws, forbidden actions, halt conditions |
| [TEST_PRODUCTION_SEPARATION.md](docs/standards/test-production-separation.md) | Mandatory test/production isolation |
| [four-laws.md](skills/shared-prompts/four-laws.md) | Canonical Four Laws prompt |
| [halt-conditions.md](skills/shared-prompts/halt-conditions.md) | When to stop and ask |

### AI-First Development

| Document | Purpose |
|----------|---------|
| [AI_ASSISTED_DEV.md](docs/ai-dev/ai-assisted-dev.md) | Vibe coding workflow, decision matrix (ask/decide/halt), design-intent preservation |
| [STATE_MANAGEMENT.md](docs/state/state-management.md) | State architecture decision tree, client/server/offline/CRDT patterns |
| [GENERATIVE_ASSET_SAFETY.md](docs/generative/generative-asset-safety.md) | AI content disclosure, C2PA metadata, procedural generation safety |
| [vibe-coding.md](skills/shared-prompts/vibe-coding.md) | Canonical vibe coding principles |

### AI Tool Integration

| Document | Purpose |
|----------|---------|
| [AGENTS_AND_SKILLS_SETUP.md](docs/integrations/agents-and-skills-setup.md) | Setup guide for Claude Code, Cursor, OpenCode, Windsurf, Copilot |
| [.claude/skills/](.claude/skills/) | 7 Claude Code skill files (guardrails-enforcer, commit-validator, etc.) |
| [.claude/hooks/](.claude/hooks/) | Pre/post execution shell hooks |
| [.cursor/rules/](.cursor/rules/) | 3 Cursor rules files |
| [.windsurfrules](.windsurfrules) | Windsurf rules preamble |
| [.opencode/](.opencode/) | OpenCode agents and skills |
| [.claude/skills/](.claude/skills/) | 8 Claude Code skill files |
| [.github/copilot-instructions.md](.github/copilot-instructions.md) | GitHub Copilot repo-level instructions |
| [skills/shared-prompts/](skills/shared-prompts/) | 7 canonical shared prompts (error-recovery, three-strikes, production-first, scope-validation, four-laws, halt-conditions, vibe-coding) |

> 3D game-development skills and docs moved to the private [agent-guardrails-game-design](https://github.com/TheArchitectit/agent-guardrails-game-design) repo. Vision capture/review lives in [radredeye](https://github.com/TheArchitectit/radredeye).

### UI/UX & Accessibility

| Document | Purpose |
|----------|---------|
| [2026_UI_UX_STANDARD.md](docs/ui-ux/ui-ux-standard.md) | UI component patterns, design tokens, responsive breakpoints |
| [ACCESSIBILITY_GUIDE.md](docs/accessibility/accessibility-guide.md) | WCAG 3.0+ compliance (Bronze/Silver/Gold) |
| [SPATIAL_COMPUTING_UI.md](docs/spatial/spatial-computing-ui.md) | XR/VR/AR UI patterns, comfort zones, latency requirements |
| [ETHICAL_ENGAGEMENT.md](docs/ethical/ethical-engagement.md) | Dark pattern taxonomy and automated prevention |

> Game design docs (2026_GAME_DESIGN, 3D game development, AI_DEV_2026 guide, Hermes 2026 dossier) live in the private [agent-guardrails-game-design](https://github.com/TheArchitectit/agent-guardrails-game-design) repo.

### Commerce & Social Safety

| Document | Purpose |
|----------|---------|
| [MONETIZATION_GUARDRAILS.md](docs/monetization/monetization-guardrails.md) | IAP ethics, loot box transparency, virtual economy balance |
| [MULTIPLAYER_SAFETY.md](docs/multiplayer/multiplayer-safety.md) | Chat moderation, matchmaking fairness, CSAM detection |
| [ANALYTICS_ETHICS.md](docs/analytics/analytics-ethics.md) | Consent tiers, data minimization, A/B testing ethics |
| [CROSS_PLATFORM_DEPLOYMENT.md](docs/deployment/cross-platform-deployment.md) | App store compliance matrix, CI/CD, feature flags |

### Workflows & Standards

| Document | Purpose |
|----------|---------|
| [AGENT_EXECUTION.md](docs/workflows/agent-execution.md) | Execution protocol, rollback, retry limits |
| [COMMIT_WORKFLOW.md](docs/workflows/commit-workflow.md) | When and how to commit |
| [CODE_REVIEW.md](docs/workflows/code-review.md) | Review process and escalation |
| [GIT_PUSH_PROCEDURES.md](docs/workflows/git-push-procedures.md) | Push safety and verification |
| [REGRESSION_PREVENTION.md](docs/workflows/regression-prevention.md) | Failure registry, prevention rules |
| [All workflows →](docs/workflows/INDEX.md) | 10 workflow documents |
| [All standards →](docs/standards/INDEX.md) | 11 standards documents |

### Token Efficiency

| Tool | Purpose |
|------|---------|
| [index-map.md](index-map.md) | Find docs by keyword — saves 60-80% tokens |
| [toc.md](toc.md) | Complete file listing |
| `.claudeignore` | Skip irrelevant files |

All documents follow the **500-line max** rule for fast context loading.

---

## MCP Server

The MCP server does the actual enforcement. Agents connect to it over HTTP, and it validates bash commands, file edits, git operations, and commits against your rules before anything runs. Written in Go, backed by PostgreSQL 16 and Redis 7.

As of v3.3.0 the server speaks **stateless StreamableHTTP**: one `POST /mcp` endpoint, no session IDs to keep track of, each request stands on its own.

| Feature | Details |
|---------|---------|
| **35 MCP Tools** | Session, bash/file/git validation, scope, regression, team, webhooks, budget, lifecycle |
| **11 MCP Resources** | Quick reference, active rules, four laws, halt conditions, and documentation |
| **Web UI** | Dashboard, document browser, rules management, failure registry |
| **REST API** | 31 endpoints including `/api/v1/policy/check` for CI/CD enforcement |
| **API Docs** | OpenAPI 3.1 spec + Scalar explorer at `/docs` |
| **Endpoints** | StreamableHTTP (`POST /mcp`), Web UI (`/web`) |

```bash
# Quick start with Docker Compose
cp .env.example .env  # Edit with your API keys
cd mcp-server && make compose-up

# Or deploy production
cd mcp-server && docker compose -f deploy/podman-compose.yml up -d

# Verify
curl http://localhost:8081/health/ready
curl http://localhost:8081/docs  # API explorer
```

See [mcp-server/README.md](mcp-server/README.md) for full setup, API docs, and troubleshooting.
See [DEPLOYMENT_GUIDE.md](mcp-server/deployment-guide.md) for production deployment.

---

## Examples

Multi-language implementation examples demonstrating guardrails patterns:

| Language | Directory | Highlights |
|----------|-----------|------------|
| **Go** | `examples/go/` | Admin UI, HTMX patterns |
| **TypeScript** | `examples/typescript/` | Game UI, UI components |
| **Rust** | `examples/rust/` | Bevy UI, egui overlay |
| **Python** | `examples/python/` | Game tools, UI dashboard |
| **Java** | `examples/java/` | Compose UI |
| **Swift** | `examples/swift/` | SwiftUI game |
| **Dart/Flutter** | `examples/flutter/` | Cross-platform: ethical widgets, accessibility wrappers |
| **GDScript** | `examples/gdscript/` | Godot: comfort zones, ethical UI, accessibility |
| **Scala** | `examples/scala/` | Functional UI, type-safe CSS, DDA telemetry |
| **R** | `examples/r/` | Game analytics, ethics auditing |
| **C#** | `examples/csharp/` | Unity UI |
| **C++** | `examples/cpp/` | Unreal UI |
| **PHP** | `examples/php/` | Laravel UI |
| **Ruby** | `examples/ruby/` | Rails UI |

---

## Who Should Use This

- **Teams where agents write most of the code** — Guardrails let agents build at full velocity without a human reviewing every command.
- **Engineering teams** rolling out AI coding assistants across multiple projects.
- **DevOps & platform teams** — Enforce infrastructure guardrails and prevent configuration drift.
- **Agent developers** building autonomous agents that need real-time validation.
- **Compliance & security teams** who need documented safety processes that hold up under audit.

> Building games? The 3D game development suite (Godot/Unity/Unreal guardrails, XR comfort zones, AI_DEV_2026 and Hermes 2026 guides) lives in the private [agent-guardrails-game-design](https://github.com/TheArchitectit/agent-guardrails-game-design) repo. Vision capture and screenshot review live in [radredeye](https://github.com/TheArchitectit/radredeye).

---

## Project Structure

```
agent-guardrails-template/
├── README.md                    ← You are here
├── index-map.md ← Token-efficient navigation
├── toc.md                       ← Complete file listing
├── CLAUDE.md                    ← Claude Code CLI context
├── CONTRIBUTING.md              ← How to contribute
├── CHANGELOG.md                 ← Release notes
│
├── docs/
│   ├── AGENT_GUARDRAILS.md      # Core safety protocols (MANDATORY)
│   ├── HOW_TO_APPLY.md          # Apply template to your repo
│   ├── QUICK_SETUP.md           # 5-minute setup guide
│   ├── STATUS.md                # Project status
│   ├── ai-dev/                  # AI-assisted development patterns
│   ├── state/                   # State management patterns
│   ├── generative/              # Generative asset safety
│   ├── monetization/            # Monetization guardrails
│   ├── multiplayer/             # Multiplayer safety
│   ├── analytics/               # Analytics ethics
│   ├── deployment/              # Cross-platform deployment
│   ├── ui-ux/                   # UI/UX component standards
│   ├── accessibility/           # WCAG 3.0+ compliance
│   ├── spatial/                 # XR/VR/AR patterns
│   ├── ethical/                 # Dark pattern prevention
│   ├── security/                # Security audit guides
│   ├── advisors/                # Cost, privacy, resilience advisors
│   ├── workflows/               # 10 operational procedure docs
│   ├── standards/               # 11 engineering standards docs
│   └── sprints/                 # Task framework and templates
│
├── mcp-server/                  ← Go MCP server (PostgreSQL + Redis)
│   ├── docs/openapi.yaml        # OpenAPI 3.1 spec (31 endpoints)
│   └── internal/budget/         # Token budget ledger
├── docker-compose.yml           ← Local dev stack (postgres, redis, mcp-server)
├── examples/                    ← 14 language implementations
├── skills/shared-prompts/       ← Four Laws, halt conditions, vibe coding
├── scripts/                     ← Setup and utility tools
└── .github/                     ← CI/CD, templates, secrets management
```

---

## Statistics

| Metric | Count |
|--------|-------|
| **Documentation Files** | 68+ |
| **Guardrail Categories** | 6 (safety, commerce, social, analytics, deployment, generative) |
| **Workflows** | 10 documents |
| **Standards** | 11 documents |
| **Example Languages** | 14 (Go, TS, Rust, Python, Java, Swift, Dart, GDScript, Scala, R, C#, C++, PHP, Ruby) |
| **MCP Tools** | 35 |
| **MCP Resources** | 11 |
| **Implementation** | Go 1.25+ |
| **Infrastructure** | PostgreSQL 16, Redis 7, Docker/Podman |

---

## Version History

**Current:** v3.4.0 (2026-08-15)

| Version | Date | Highlights |
|---------|------|------------|
| **v3.4.0** | 2026-08-15 | Documentation redo, pi-extension restored, game/vision content split to private repos |
| **v3.3.0** | 2026-08-15 | Stateless StreamableHTTP transport, repo cleanup, migration & deploy fixes |
| **v3.2.0** | 2026-06-16 | Platform review sprint: 7 features, all P0 fixes |
| **v3.1.0** | 2026-05-12 | Structural reorganization: split docs into 3d/ subfolder, README link fixes, stats update |
| **v3.0.0** | 2026-05-12 | 3D game development suite, AI-Powered Development 2026 guide, Hermes 2026 dossier |
| **v2.9.0** | 2026-05-08 | AI tool integration suite (Claude Code, Cursor, Windsurf, Copilot, OpenCode) |
| **v2.8.0** | 2026-03-14 | AI-first reframe, 7 new guardrail docs, vibe coding, Flutter/Godot examples |
| **v2.7.0** | 2026-03-14 | Agent-GDUI-2026, game design suite, WCAG 3.0+, spatial computing |
| **v2.6.0** | 2026-02-15 | Python → Go migration complete |

See [CHANGELOG.md](CHANGELOG.md) for full history.

---

## License

BSD-3-Clause — See [LICENSE](LICENSE)

---

## Credits

- **Maintainer:** [TheArchitectit](https://github.com/TheArchitectit)
- **Built with:** Claude Code + Opus

## Support This Project

If this project helped you, consider buying me a coffee:

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-TheArchitectit-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/TheArchitectit)

Or use one of the referral links below — you get credits, and so do we:

| Service | Your Bonus | Details | Referral Code |
|---------|-----------|---------|---------------|
| [**Synthetic**](https://synthetic.new/?referral=UAWqkKQQLFkzMkY) | $10 in credits | Subscribe → both get $10 credit | `UAWqkKQQLFkzMkY` |
| [**Ozore.com**](https://ozore.com/?ref=cwe4kdx0) | 50% off first month |  — code `lundrog50` | — |
| [**Neuralwatt**](https://portal.neuralwatt.com/auth/register?ref=NW-ROGER-ET3Y) | When you use $25+ in compute, you both earn $5. Bonuses are credited after a 72-hour review period. — |

---

**v3.4.0** · AI-First Rapid Development Framework · [Get Started →](docs/getting-started/quick-setup.md)
