# Agent Guardrails Template

> AI-first safety framework for agents building software at high velocity. Guardrails don't slow you down — they're your license to move fast.

[![Version](https://img.shields.io/badge/version-v3.0.0-blue.svg)](./CHANGELOG.md)
[![Go Implementation](https://img.shields.io/badge/Implementation-Go-blue.svg?style=flat&logo=go)](https://golang.org)
[![WCAG 3.0+](https://img.shields.io/badge/Accessibility-WCAG_3.0+_Silver-green.svg)](docs/accessibility/ACCESSIBILITY_GUIDE.md)
[![Spatial Computing](https://img.shields.io/badge/Spatial-XR/VR/AR-blue.svg)](docs/spatial/SPATIAL_COMPUTING_UI.md)

---

## What Is This?

**The Agent Guardrails Template** is a framework that enables AI agents to develop software at full velocity with built-in safety. Whether you're vibe coding a game prototype or building production infrastructure, guardrails let agents spend tokens on building instead of second-guessing safety.

It works with any AI system — Claude, GPT, Gemini, LLaMA, and 30+ other model families.

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

Then see [QUICK_SETUP.md](QUICK_SETUP.md) for the 5-minute setup, or [HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) to apply guardrails to an existing repo.

---

## What's Included

### Core Safety (Mandatory)

| Document | Purpose |
|----------|---------|
| [AGENT_GUARDRAILS.md](docs/AGENT_GUARDRAILS.md) | The Four Laws, forbidden actions, halt conditions |
| [TEST_PRODUCTION_SEPARATION.md](docs/standards/TEST_PRODUCTION_SEPARATION.md) | Mandatory test/production isolation |
| [four-laws.md](skills/shared-prompts/four-laws.md) | Canonical Four Laws prompt |
| [halt-conditions.md](skills/shared-prompts/halt-conditions.md) | When to stop and ask |

### AI-First Development (v2.8.0)

| Document | Purpose |
|----------|---------|
| [AI_ASSISTED_DEV.md](docs/ai-dev/AI_ASSISTED_DEV.md) | Vibe coding workflow, decision matrix (ask/decide/halt), design-intent preservation |
| [STATE_MANAGEMENT.md](docs/state/STATE_MANAGEMENT.md) | State architecture decision tree, client/server/offline/CRDT patterns |
| [GENERATIVE_ASSET_SAFETY.md](docs/generative/GENERATIVE_ASSET_SAFETY.md) | AI content disclosure, C2PA metadata, procedural generation safety |
| [vibe-coding.md](skills/shared-prompts/vibe-coding.md) | Canonical vibe coding principles |

### Game Design & UI/UX (Agent-GDUI-2026)

| Document | Purpose |
|----------|---------|
| [2026_GAME_DESIGN.md](docs/game-design/2026_GAME_DESIGN.md) | Game design guardrails, XR/VR comfort zones, performance budgets |
| [3D_GAME_DEVELOPMENT.md](docs/game-design/3D_GAME_DEVELOPMENT.md) | 3D game dev pipeline: assets, Godot conventions, AI workflow, scope, budgets |
| [2026_UI_UX_STANDARD.md](docs/ui-ux/2026_UI_UX_STANDARD.md) | UI component patterns, design tokens, responsive breakpoints |
| [ACCESSIBILITY_GUIDE.md](docs/accessibility/ACCESSIBILITY_GUIDE.md) | WCAG 3.0+ compliance (Bronze/Silver/Gold) |
| [SPATIAL_COMPUTING_UI.md](docs/spatial/SPATIAL_COMPUTING_UI.md) | XR/VR/AR UI patterns, comfort zones, latency requirements |
| [ETHICAL_ENGAGEMENT.md](docs/ethical/ETHICAL_ENGAGEMENT.md) | Dark pattern taxonomy and automated prevention |

### Commerce & Social Safety (v2.8.0)

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
| **17 MCP Tools** | Session init, bash/file/git validation, scope checking, regression prevention, team management |
| **8 MCP Resources** | Quick reference, active rules, documentation access |
| **Web UI** | Dashboard, document browser, rules management, failure registry |
| **Endpoints** | SSE stream (`/mcp/v1/sse`), JSON-RPC (`/mcp/v1/message`), Web UI (`/web`) |

```bash
# Deploy
cd mcp-server && docker compose -f deploy/podman-compose.yml up -d

# Verify
curl http://your-server:8095/health/ready
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
- **Engineering Teams** — Deploying AI coding assistants safely across projects.
- **DevOps & Platform Teams** — Enforcing infrastructure guardrails and preventing configuration drift.
- **AI Agent Developers** — Building safer autonomous agents with real-time validation.
- **Compliance & Security Teams** — Meeting regulatory requirements with documented safety processes.

---

## Project Structure

```
agent-guardrails-template/
├── README.md                    ← You are here
├── QUICK_SETUP.md               ← 5-minute setup guide
├── PROMPTING_GUIDE.md           ← Effective prompting for AI development
├── INDEX_MAP.md / HEADER_MAP.md ← Token-efficient navigation
├── CLAUDE.md                    ← Claude Code CLI context
├── CHANGELOG.md                 ← Release notes
│
├── docs/
│   ├── AGENT_GUARDRAILS.md      # Core safety protocols (MANDATORY)
│   ├── HOW_TO_APPLY.md          # Apply template to your repo
│   ├── ai-dev/                  # AI-assisted development patterns (v2.8.0)
│   ├── state/                   # State management patterns (v2.8.0)
│   ├── generative/              # Generative asset safety (v2.8.0)
│   ├── monetization/            # Monetization guardrails (v2.8.0)
│   ├── multiplayer/             # Multiplayer safety (v2.8.0)
│   ├── analytics/               # Analytics ethics (v2.8.0)
│   ├── deployment/              # Cross-platform deployment (v2.8.0)
│   ├── game-design/             # 2026 game design guardrails
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
├── examples/                    ← 14 language implementations
├── skills/shared-prompts/       ← Four Laws, halt conditions, vibe coding
├── scripts/                     ← Setup and utility tools
└── .github/                     ← CI/CD, templates, secrets management
```

---

## Statistics

| Metric | Count |
|--------|-------|
| **Documentation Files** | 44+ |
| **Guardrail Categories** | 7 (safety, game design, commerce, social, analytics, deployment, generative) |
| **Workflows** | 10 documents |
| **Standards** | 11 documents |
| **Example Languages** | 14 (Go, TS, Rust, Python, Java, Swift, Dart, GDScript, Scala, R, C#, C++, PHP, Ruby) |
| **MCP Tools** | 17 |
| **MCP Resources** | 8 |
| **Supported AI Models** | 30+ LLM families |
| **Implementation** | Go 1.23+ |
| **Infrastructure** | PostgreSQL 16, Redis 7, Docker/Podman |

---

## Version History

**Current:** v2.8.0 (2026-03-14)

| Version | Date | Highlights |
|---------|------|------------|
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

**v2.8.0** · AI-First Rapid Development Framework · [Get Started →](QUICK_SETUP.md)
