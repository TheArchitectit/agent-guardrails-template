# Agent Guardrails Template

> AI-first safety framework for agents building software at high velocity. Guardrails don't slow you down — they're your license to move fast.

[![Version](https://img.shields.io/badge/version-v3.2.0-blue.svg)](./CHANGELOG.md)
[![Go Implementation](https://img.shields.io/badge/Implementation-Go-blue.svg?style=flat&logo=go)](https://golang.org)
[![WCAG 3.0+](https://img.shields.io/badge/Accessibility-WCAG_3.0+_Silver-green.svg)](docs/accessibility/ACCESSIBILITY_GUIDE.md)
[![Spatial Computing](https://img.shields.io/badge/Spatial-XR/VR/AR-blue.svg)](docs/spatial/SPATIAL_COMPUTING_UI.md)

---

## What Is This?

**The Agent Guardrails Template** is a production-grade operating system for AI-assisted development. It turns "vibe coding" chaos into shipping software — giving AI agents explicit boundaries so they spend 100% of their context window on building, not on safety-checking.

### What You Actually Get

| Capability | What It Does |
|-----------|-------------|
| **Real-Time Guardrail Enforcement** | Go MCP server validates every bash command, file edit, git operation, and commit before execution |
| **CI/CD Policy Enforcement** | `POST /api/v1/policy/check` endpoint for gating PRs on guardrail violations — with line/column precision |
| **Webhook Notifications** | Real-time violation and halt event delivery via HMAC-signed webhooks with circuit breaker protection |
| **Token Budget Ledger** | Track AI API costs across Claude/GPT models, set team budgets, get alerts when thresholds are crossed |
| **Agent Lifecycle Management** | State machine (idle→planning→active→review→release) with transition validation and audit trails |
| **Multi-Agent Orchestration** | 10-part AI-Powered Development 2026 guide covering MoA (Mixture of Agents), swarm intelligence, and autonomous tool use |
| **Cross-Platform IDE Integration** | Native skills and rules for Claude Code, Cursor, OpenCode, Windsurf, and GitHub Copilot — not generic prompts |
| **3D Game Development Suite** | Engine-agnostic guardrails (Godot, Unity, Unreal), XR/VR/AR comfort zones, mathematical foundations, AI-debuggable architecture |
| **Token-Efficient Documentation** | 68+ modular docs (500-line max), INDEX_MAP keyword lookup, HEADER_MAP section navigation, `.claudeignore` for context savings |
| **Production Infrastructure** | PostgreSQL 16 + Redis 7, Docker Compose, CI/CD validation, 22-pattern secret scanning, regression prevention |
| **14 Language Examples** | Go, Rust, TypeScript, Python, Java, GDScript, Scala, R, C#, C++, PHP, Ruby, Swift, Dart/Flutter |
| **Ethical & Accessible by Default** | WCAG 3.0+ Silver compliance, dark pattern prevention, XR comfort zones, monetization ethics, multiplayer safety |

### Who This Is For

- **AI-First Teams** — Agents generate 80%+ of your code. You need them to move fast without breaking prod.
- **3D Game Developers** — AI-generated shaders, physics, NPCs, and assets need mathematical correctness and comfort-zone enforcement.
- **Platform Engineers** — Enforce infrastructure guardrails, prevent config drift, and maintain separation across environments.
- **Compliance & Security** — Documented safety processes that satisfy regulatory requirements.

### The Paradox: Constraints Enable Speed

Without guardrails, agents waste tokens on safety verification: *"Is this file safe to edit? Will this break something? Should I ask first?"* This constant self-checking burns context and slows output.

With guardrails, agents know the boundaries upfront. They spend tokens on building, not on doubt. The result: faster iteration, fewer rollbacks, and code that ships with confidence.

Think of guardrails like lane markers on a highway — they don't slow you down. They're the reason you can drive at full speed.

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

Then see [QUICK_SETUP.md](docs/QUICK_SETUP.md) for the 5-minute setup, or [HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) to apply guardrails to an existing repo.

---

## What's Included

### Core Safety (Mandatory)

| Document | Purpose |
|----------|---------|
| [AGENT_GUARDRAILS.md](docs/AGENT_GUARDRAILS.md) | The Four Laws, forbidden actions, halt conditions |
| [TEST_PRODUCTION_SEPARATION.md](docs/standards/TEST_PRODUCTION_SEPARATION.md) | Mandatory test/production isolation |
| [four-laws.md](skills/shared-prompts/four-laws.md) | Canonical Four Laws prompt |
| [halt-conditions.md](skills/shared-prompts/halt-conditions.md) | When to stop and ask |

### AI-First Development 

| Document | Purpose |
|----------|---------|
| [AI_ASSISTED_DEV.md](docs/ai-dev/AI_ASSISTED_DEV.md) | Vibe coding workflow, decision matrix (ask/decide/halt), design-intent preservation |
| [STATE_MANAGEMENT.md](docs/state/STATE_MANAGEMENT.md) | State architecture decision tree, client/server/offline/CRDT patterns |
| [GENERATIVE_ASSET_SAFETY.md](docs/generative/GENERATIVE_ASSET_SAFETY.md) | AI content disclosure, C2PA metadata, procedural generation safety |
| [vibe-coding.md](skills/shared-prompts/vibe-coding.md) | Canonical vibe coding principles |

### AI Tool Integration 

| Document | Purpose |
|----------|---------|
| [AGENTS_AND_SKILLS_SETUP.md](docs/AGENTS_AND_SKILLS_SETUP.md) | Setup guide for Claude Code, Cursor, OpenCode, Windsurf, Copilot |
| [.claude/skills/](.claude/skills/) | 7 Claude Code skill files (guardrails-enforcer, commit-validator, etc.) |
| [.claude/hooks/](.claude/hooks/) | Pre/post execution shell hooks |
| [.cursor/rules/](.cursor/rules/) | 3 Cursor rules files |
| [.cursor/rules-3d/](.cursor/rules-3d/) | 3D game dev Cursor rules |
| [.windsurfrules](.windsurfrules) | Windsurf rules preamble |
| [.opencode/](.opencode/) | OpenCode agents and skills |
| [.opencode/skills/3d-game-dev/](.opencode/skills/3d-game-dev/) | 3D game dev OpenCode skill |
| [.claude/skills/](.claude/skills/) | 7 Claude Code skill files |
| [.claude/skills-3d/](.claude/skills-3d/) | 3D game dev Claude skill |
| [.github/copilot-instructions.md](.github/copilot-instructions.md) | GitHub Copilot repo-level instructions |
| [skills/shared-prompts/](skills/shared-prompts/) | 8 canonical shared prompts (3d-game-dev, error-recovery, three-strikes, production-first, scope-validation + existing) |

### UI/UX & Accessibility

| Document | Purpose |
|----------|---------|
| [2026_UI_UX_STANDARD.md](docs/ui-ux/2026_UI_UX_STANDARD.md) | UI component patterns, design tokens, responsive breakpoints |
| [ACCESSIBILITY_GUIDE.md](docs/accessibility/ACCESSIBILITY_GUIDE.md) | WCAG 3.0+ compliance (Bronze/Silver/Gold) |
| [SPATIAL_COMPUTING_UI.md](docs/spatial/SPATIAL_COMPUTING_UI.md) | XR/VR/AR UI patterns, comfort zones, latency requirements |
| [ETHICAL_ENGAGEMENT.md](docs/ethical/ETHICAL_ENGAGEMENT.md) | Dark pattern taxonomy and automated prevention |

> Game design docs (2026_GAME_DESIGN, 3D game development, AI_DEV_2026 guide, Hermes 2026 dossier) have been moved to a separate private repo.

### Commerce & Social Safety 

| Document | Purpose |
|----------|---------|
| [MONETIZATION_GUARDRAILS.md](docs/monetization/MONETIZATION_GUARDRAILS.md) | IAP ethics, loot box transparency, virtual economy balance |
| [MULTIPLAYER_SAFETY.md](docs/multiplayer/MULTIPLAYER_SAFETY.md) | Chat moderation, matchmaking fairness, CSAM detection |
| [ANALYTICS_ETHICS.md](docs/analytics/ANALYTICS_ETHICS.md) | Consent tiers, data minimization, A/B testing ethics |
| [CROSS_PLATFORM_DEPLOYMENT.md](docs/deployment/CROSS_PLATFORM_DEPLOYMENT.md) | App store compliance matrix, CI/CD, feature flags |

### Workflows & Standards

| Document | Purpose |
|----------|---------|
| [AGENT_EXECUTION.md](docs/workflows/AGENT_EXECUTION.md) | Execution protocol, rollback, retry limits |
| [COMMIT_WORKFLOW.md](docs/workflows/COMMIT_WORKFLOW.md) | When and how to commit |
| [CODE_REVIEW.md](docs/workflows/CODE_REVIEW.md) | Review process and escalation |
| [GIT_PUSH_PROCEDURES.md](docs/workflows/GIT_PUSH_PROCEDURES.md) | Push safety and verification |
| [REGRESSION_PREVENTION.md](docs/workflows/REGRESSION_PREVENTION.md) | Failure registry, prevention rules |
| [All workflows →](docs/workflows/INDEX.md) | 10 workflow documents |
| [All standards →](docs/standards/INDEX.md) | 11 standards documents |

### Token Efficiency

| Tool | Purpose |
|------|---------|
| [INDEX_MAP.md](INDEX_MAP.md) | Find docs by keyword — saves 60-80% tokens |
| [HEADER_MAP.md](HEADER_MAP.md) | Jump to specific sections with line numbers |
| [TOC.md](TOC.md) | Complete file listing |
| `.claudeignore` | Skip irrelevant files |

All documents follow the **500-line max** rule for fast context loading.

---

## MCP Server

The **Model Context Protocol Server** provides real-time guardrail enforcement — validating every bash command, file edit, git operation, and commit before execution.

**Implementation:** Go (`mcp-server/internal/`) | **Infra:** PostgreSQL 16 + Redis 7

| Feature | Details |
|---------|---------|
| **32 MCP Tools** | Session, bash/file/git validation, scope, regression, team, webhooks (5), budget (5), lifecycle (5) |
| **8 MCP Resources** | Quick reference, active rules, documentation access |
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
See [DEPLOYMENT_GUIDE.md](mcp-server/DEPLOYMENT_GUIDE.md) for production deployment.

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

- **AI-First Development Teams** — Practicing vibe coding where agents generate most of the code. Guardrails let agents build at full velocity without human bottlenecks.
- **3D Game Development Teams** — Building with Godot, Unity, Unreal, or custom engines. Mathematical correctness, asset safety, shader constraints, and AI-debuggable architecture.
- **Engineering Teams** — Deploying AI coding assistants safely across projects.
- **DevOps & Platform Teams** — Enforcing infrastructure guardrails and preventing configuration drift.
- **AI Agent Developers** — Building safer autonomous agents with real-time validation.
- **Compliance & Security Teams** — Meeting regulatory requirements with documented safety processes.

---

## Project Structure

```
agent-guardrails-template/
├── README.md                    ← You are here
├── INDEX_MAP.md / HEADER_MAP.md ← Token-efficient navigation
├── TOC.md                       ← Complete file listing
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
| **Example Languages** | 13 (Go, TS, Rust, Python, Java, Swift, Dart, Scala, R, C#, C++, PHP, Ruby) |
| **MCP Tools** | 56 |
| **MCP Resources** | 11 |
| **Supported AI Models** | 30+ LLM families |
| **Implementation** | Go 1.25+ |
| **Infrastructure** | PostgreSQL 16, Redis 7, Docker/Podman |

---

## Version History

**Current:** v3.2.0 (2026-06-16)

| Version | Date | Highlights |
|---------|------|------------|
| **v3.2.0** | 2026-06-16 | Platform review sprint: 7 features, all P0 fixes, mcp-go v0.58.0 |
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

### ☕ Support This Project

Help keep this project going — use a referral link below and both of us get credits!

| Service | Your Bonus | Details | Referral Code |
|---------|-----------|---------|---------------|
| [**Neuralwatt**](https://portal.neuralwatt.com/auth/register?ref=NW-ROGER-ET3Y) | $10 in credits | Spend $10+ → you get $10, we get $20 | `NW-ROGER-ET3Y` |
| [**Synthetic**](https://synthetic.new/?referral=UAWqkKQQLFkzMkY) | $10 in credits | Subscribe → both get $10 credit | `UAWqkKQQLFkzMkY` |
---

**v3.2.0** · AI-First Rapid Development Framework · [Get Started →](docs/QUICK_SETUP.md)


## ☁️ Cloud Credits

Power your AI projects with [Ozore.com](https://ozore.com/?ref=cwe4kdx0) — use code **lundrog50** for 50% off your first month.

> `direct-pin` and `custom-router` are available on **Pro** and **Max** plans only.

## ☕ Support

If this project helped you, consider buying me a coffee:

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-TheArchitectit-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/TheArchitectit)
