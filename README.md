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
| **Multi-Agent Orchestration** | 10-part AI-Powered Development 2026 guide covering MoA (Mixture of Agents), swarm intelligence, and autonomous tool use |
| **Skills Architecture** | One canonical skill definition (`skills/<id>/SKILL.md`) generates native formats for Claude Code, Cursor, OpenCode, Windsurf, Copilot, OpenClaw — no duplication drift |
| **3D Game Development Suite** | Engine-agnostic guardrails (Godot, Unity, Unreal), XR/VR/AR comfort zones, mathematical foundations, AI-debuggable architecture |
| **Token-Efficient Documentation** | 68+ modular docs (500-line max), INDEX_MAP keyword lookup, HEADER_MAP section navigation, `.claudeignore` for context savings |
| **Production Infrastructure** | PostgreSQL 16 + Redis 7, CI/CD validation, secret scanning, regression prevention, test/production separation |
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

Then see [QUICK_SETUP.md](QUICK_SETUP.md) for the 5-minute setup, or [HOW_TO_APPLY.md](docs/HOW_TO_APPLY.md) to apply guardrails to an existing repo.

---

## What's Included

### Core Safety (Mandatory)

| Document | Purpose |
|----------|---------|
| [AGENT_GUARDRAILS.md](docs/AGENT_GUARDRAILS.md) | The Four Laws, forbidden actions, halt conditions |
| [TEST_PRODUCTION_SEPARATION.md](docs/standards/TEST_PRODUCTION_SEPARATION.md) | Mandatory test/production isolation |
| [four-laws](skills/four-laws/SKILL.md) | Canonical Four Laws prompt |
| [halt-conditions](skills/halt-conditions/SKILL.md) | When to stop and ask |

### AI-First Development 

| Document | Purpose |
|----------|---------|
| [AI_ASSISTED_DEV.md](docs/ai-dev/AI_ASSISTED_DEV.md) | Vibe coding workflow, decision matrix (ask/decide/halt), design-intent preservation |
| [STATE_MANAGEMENT.md](docs/state/STATE_MANAGEMENT.md) | State architecture decision tree, client/server/offline/CRDT patterns |
| [GENERATIVE_ASSET_SAFETY.md](docs/generative/GENERATIVE_ASSET_SAFETY.md) | AI content disclosure, C2PA metadata, procedural generation safety |
| [vibe-coding](skills/vibe-coding/SKILL.md) | Canonical vibe coding principles |

### Skills Architecture

One canonical skill definition → generated native formats for every IDE.

| Layer | What It Is | Path |
|-------|-----------|------|
| **Canonical Skills** | YAML frontmatter + markdown body. Source of truth for all 11 skills | `skills/<id>/SKILL.md` |
| **Build Script** | Generates all native formats from canonical sources. Supports `--check`, `--dry-run`, `--platform`, `--skill` | `scripts/build_skills.py` |
| **Plugin Manifests** | Per-IDE wrappers pointing to `./skills/` | `.claude-plugin/`, `.cursor-plugin/`, `.codex-plugin/`, `.gemini-extension/` |
| **Claude Code Plugin** | Native plugin: install via `/plugin install` or `claude --plugin-dir ./` | `.claude-plugin/plugin.json` + `skills/<id>/SKILL.md` |
| **Cross-Platform Marketplace** | Install any skill for any platform from any GitHub repo | `marketplace.json` + `scripts/marketplace.py` |
| **Generated: Claude Code** | JSON skill files | `.claude/skills/*.json` |
| **Generated: Cursor** | Markdown rule files | `.cursor/rules/*.md` |
| **Generated: OpenCode** | SKILL.md per skill | `.opencode/skills/*/SKILL.md` |
| **Generated: OpenClaw** | SKILL.md per skill | `.openclaw/skills/*/SKILL.md` |
| **Generated: Copilot** | Monolithic instructions | `.github/copilot-instructions.md` |
| **Generated: Windsurf** | Monolithic rules | `.windsurfrules` |
| **Hooks** | Pre/post execution shell hooks | `.claude/hooks/*.sh` |

**Installation:** See [AGENTS_AND_SKILLS_SETUP.md](docs/AGENTS_AND_SKILLS_SETUP.md) for quick start, [SKILL_REGISTRY.md](docs/SKILL_REGISTRY.md) for per-skill details, [SKILLS_ARCHITECTURE.md](docs/SKILLS_ARCHITECTURE.md) for build script and CI/CD integration, [MARKETPLACE.md](docs/MARKETPLACE.md) for cross-platform marketplace CLI, and [CLAUDE_CODE_PLUGIN.md](docs/CLAUDE_CODE_PLUGIN.md) for Claude Code native plugin testing and submission.

### Game Design & UI/UX (Agent-GDUI-2026)

| Document | Purpose |
|----------|---------|
| [2026_GAME_DESIGN.md](docs/game-design/2026_GAME_DESIGN.md) | Game design guardrails, XR/VR comfort zones, performance budgets |
| [3D_GAME_DEVELOPMENT.md](docs/game-design/3d/3D_GAME_DEVELOPMENT.md) | 3D game dev pipeline: assets, Godot conventions, AI workflow, scope, budgets |
| [3D_MATHEMATICAL_FOUNDATIONS.md](docs/game-design/3d/3D_MATHEMATICAL_FOUNDATIONS.md) | Linear algebra, quaternions, collision math reference |
| [3D_MODULE_ARCHITECTURE.md](docs/game-design/3d/3D_MODULE_ARCHITECTURE.md) | LLM-to-3D-engine bridging architecture |
| [AI_DEBUGGABLE_3D_ARCHITECTURE.md](docs/game-design/3d/AI_DEBUGGABLE_3D_ARCHITECTURE.md) | Autonomous 3D troubleshooting patterns |
| [3D_GUARDREL_PROPOSALS_V1.2.md](docs/game-design/3d/3D_GUARDREL_PROPOSALS_V1.2.md) | v1.2 proposed guardrails (neural radiance fields, AI NPCs) |
| [2026_UI_UX_STANDARD.md](docs/ui-ux/2026_UI_UX_STANDARD.md) | UI component patterns, design tokens, responsive breakpoints |
| [ACCESSIBILITY_GUIDE.md](docs/accessibility/ACCESSIBILITY_GUIDE.md) | WCAG 3.0+ compliance (Bronze/Silver/Gold) |
| [SPATIAL_COMPUTING_UI.md](docs/spatial/SPATIAL_COMPUTING_UI.md) | XR/VR/AR UI patterns, comfort zones, latency requirements |
| [ETHICAL_ENGAGEMENT.md](docs/ethical/ETHICAL_ENGAGEMENT.md) | Dark pattern taxonomy and automated prevention |

### AI-Powered Development 2026

| Document | Purpose |
|----------|---------|
| [AI_DEV_2026_PART01_INTRO_AND_FOUNDATIONS.md](docs/game-design/AI_DEV_2026_PART01_INTRO_AND_FOUNDATIONS.md) | Introduction & Foundations (Ch 1–2) |
| [AI_DEV_2026_PART02_PROMPTING.md](docs/game-design/AI_DEV_2026_PART02_PROMPTING.md) | Prompt Engineering for Code |
| [AI_DEV_2026_PART03_CONTEXT_AND_ITERATION.md](docs/game-design/AI_DEV_2026_PART03_CONTEXT_AND_ITERATION.md) | Context & Iterative Development |
| [AI_DEV_2026_PART04_QUALITY_AND_ARCHITECTURE.md](docs/game-design/AI_DEV_2026_PART04_QUALITY_AND_ARCHITECTURE.md) | Quality & Architecture |
| [AI_DEV_2026_PART05_LEGACY_AND_AGENTS.md](docs/game-design/AI_DEV_2026_PART05_LEGACY_AND_AGENTS.md) | Legacy Refactoring & Agent Paradigm |
| [AI_DEV_2026_PART06_BUILDING_AGENTS.md](docs/game-design/AI_DEV_2026_PART06_BUILDING_AGENTS.md) | Building Agents & Tool Use |
| [AI_DEV_2026_PART07_MULTI_AGENT_SYSTEMS.md](docs/game-design/AI_DEV_2026_PART07_MULTI_AGENT_SYSTEMS.md) | Multi-Agent Systems |
| [AI_DEV_2026_PART08_SECURITY_ETHICS_FUTURE.md](docs/game-design/AI_DEV_2026_PART08_SECURITY_ETHICS_FUTURE.md) | Security, Ethics & Future |
| [AI_DEV_2026_PART09_APPENDICES_ABC.md](docs/game-design/AI_DEV_2026_PART09_APPENDICES_ABC.md) | Appendices A, B & C |
| [AI_DEV_2026_PART10_APPENDIX_D.md](docs/game-design/AI_DEV_2026_PART10_APPENDIX_D.md) | Appendix D: Complete MoA Reference |

### Hermes 2026: AI in 3D Game Development

| Document | Purpose |
|----------|---------|
| [HERMES_2026_PART01_INTRO_AND_EXECUTIVE.md](docs/game-design/3d/HERMES_2026_PART01_INTRO_AND_EXECUTIVE.md) | Introduction & Executive Summary |
| [HERMES_2026_PART02_ASSETS_AND_ENGINES.md](docs/game-design/3d/HERMES_2026_PART02_ASSETS_AND_ENGINES.md) | 3D Asset Generation & Engine Integration |
| [HERMES_2026_PART03_WORLD_AND_RENDERING.md](docs/game-design/3d/HERMES_2026_PART03_WORLD_AND_RENDERING.md) | World Generation & Neural Rendering |
| [HERMES_2026_PART04_NPCS_AND_ANIMATION.md](docs/game-design/3d/HERMES_2026_PART04_NPCS_AND_ANIMATION.md) | NPCs, Dialogue & Animation |
| [HERMES_2026_PART05_CODE_AND_PHYSICS.md](docs/game-design/3d/HERMES_2026_PART05_CODE_AND_PHYSICS.md) | Code Generation & Neural Physics |
| [HERMES_2026_PART06_QA_AND_BUSINESS.md](docs/game-design/3d/HERMES_2026_PART06_QA_AND_BUSINESS.md) | QA, Testing & Business Landscape |
| [HERMES_2026_PART07_LEGAL_AND_CASES.md](docs/game-design/3d/HERMES_2026_PART07_LEGAL_AND_CASES.md) | Legal, Ethics & Case Studies |
| [HERMES_2026_PART08_DEEP_DIVES_AND_FUTURE.md](docs/game-design/3d/HERMES_2026_PART08_DEEP_DIVES_AND_FUTURE.md) | Technology Deep-Dives & Future Outlook |
| [HERMES_2026_PART09_APPENDICES.md](docs/game-design/3d/HERMES_2026_PART09_APPENDICES.md) | Appendices |

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
├── QUICK_SETUP.md               ← 5-minute setup guide
├── CLAUDE.md                    ← Claude Code CLI context (auto-loaded)
├── GEMINI.md                    ← Gemini CLI context (auto-loaded)
├── AGENTS.md                    ← Generic agents context (auto-loaded)
├── CHANGELOG.md                 ← Release notes
│
├── skills/                      ← CANONICAL SKILL SOURCE (one truth)
│   ├── guardrails-enforcer/     # Four Laws enforcement
│   ├── commit-validator/        # Commit message validation
│   ├── env-separator/           # Test/production separation
│   ├── scope-validator/         # File modification scope checking
│   ├── production-first/        # Production before test code
│   ├── three-strikes/           # 3 failure attempts → halt
│   ├── error-recovery/          # Recovery without escalation
│   ├── four-laws/               # Shared Four Laws prompt
│   ├── halt-conditions/         # Shared halt conditions prompt
│   ├── vibe-coding/             # Shared vibe coding principles
│   └── 3d-game-dev/             # 3D game development guardrails
│
├── scripts/                     ← Build tooling
│   ├── build_skills.py          # Canonical → native format generator
│   ├── setup_agents.py          # Platform install (copy/symlink/clone/named)
│   ├── skill_lib/               # Core logic: parser, generator, assembler, rewriter
│   └── ...
│
├── docs/
│   ├── AGENT_GUARDRAILS.md      # Core safety protocols (MANDATORY)
│   ├── AGENTS_AND_SKILLS_SETUP.md  # Quick start: platform install
│   ├── SKILL_REGISTRY.md        # All skills, customization guide
│   ├── SKILLS_ARCHITECTURE.md   # Build script, CI/CD, batch workflows
│   ├── HOW_TO_APPLY.md          # Apply template to your repo
│   ├── ai-dev/                  # AI-assisted development patterns
│   ├── state/                   # State management patterns
│   ├── generative/              # Generative asset safety
│   ├── monetization/            # Monetization guardrails
│   ├── multiplayer/             # Multiplayer safety
│   ├── analytics/               # Analytics ethics
│   ├── deployment/              # Cross-platform deployment
│   ├── game-design/             # 2026 game design guardrails
│   │   └── 3d/                  # 3D game development docs
│   ├── ui-ux/                   # UI/UX component standards
│   ├── accessibility/           # WCAG 3.0+ compliance
│   ├── spatial/                 # XR/VR/AR patterns
│   ├── ethical/                 # Dark pattern prevention
│   ├── security/                # Security audit guides
│   ├── workflows/               # 10 operational procedure docs
│   └── standards/               # 11 engineering standards docs
│
├── tests/scripts/               ← Build script tests (44 pytest tests)
├── mcp-server/                  ← Go MCP server (PostgreSQL + Redis)
├── examples/                    ← 14 language implementations
│
├── .claude/                     ← GENERATED (Claude Code native)
│   ├── skills/*.json            # 11 JSON skill files
│   └── hooks/*.sh               # Pre/post execution hooks
├── .claude-plugin/plugin.json   # Plugin manifest (Claude Code native)
├── .cursor-plugin/plugin.json   # Plugin manifest
├── .codex-plugin/plugin.json    # Plugin manifest
├── .gemini-extension/           # Plugin manifest
├── marketplace.json             # Cross-platform skill catalog
├── .cursor/rules/*.md           # GENERATED (Cursor native)
├── .opencode/skills/*/SKILL.md  # GENERATED (OpenCode native)
├── .openclaw/skills/*/SKILL.md  # GENERATED (OpenClaw native)
├── .github/copilot-instructions.md  # GENERATED (Copilot, monolithic)
├── .windsurfrules               # GENERATED (Windsurf, monolithic)
│
└── .github/                     ← CI/CD, templates, secrets management
```

---

## Statistics

| Metric | Count |
|--------|-------|
| **Canonical Skills** | 11 (auto-generated to 6 native formats) |
| **Documentation Files** | 68+ |
| **Guardrail Categories** | 7 (safety, game design, commerce, social, analytics, deployment, generative) |
| **Workflows** | 10 documents |
| **Standards** | 11 documents |
| **Example Languages** | 14 (Go, TS, Rust, Python, Java, Swift, Dart, GDScript, Scala, R, C#, C++, PHP, Ruby) |
| **MCP Tools** | 17 |
| **MCP Resources** | 8 |
| **Supported AI Models** | 30+ LLM families |
| **Build Script Tests** | 44 (pytest) |
| **Supported IDEs** | 6 (Claude Code, Cursor, OpenCode, OpenClaw, Windsurf, Copilot) |
| **Implementation** | Go 1.23+, Python 3.10+ |
| **Infrastructure** | PostgreSQL 16, Redis 7, Docker/Podman |

---

## Version History

**Current:** v3.2.0 (2026-05-12)

| Version | Date | Highlights |
|---------|------|------------|
| **v3.2.0** | 2026-05-12 | Skills Architecture: canonical `skills/<id>/SKILL.md` → 6 native IDE formats via `scripts/build_skills.py`. 44 tests, drift-check CI, plugin manifests. Eliminates duplication drift. |
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

**v3.2.0** · AI-First Rapid Development Framework · [Get Started →](QUICK_SETUP.md)
