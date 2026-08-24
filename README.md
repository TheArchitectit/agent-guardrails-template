# Agent Guardrails Template

> Safety rails for AI agents that write code. Set the boundaries once, let the agent run at full speed.

[![Version](https://img.shields.io/badge/version-v3.7.1-blue.svg)](./CHANGELOG.md)
[![Powered by Atlas Cloud](https://www.atlascloud.ai/oss-program/powered-by-atlas-cloud.svg)](https://www.atlascloud.ai/?ref=F6TYTG)
[![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?style=flat&logo=go&logoColor=white)](https://golang.org)
[![License](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](./LICENSE)
[![Sponsor](https://img.shields.io/badge/Sponsor-TheArchitectit-FF69B4?style=flat&logo=github-sponsors)](https://github.com/sponsors/TheArchitectit)

---

If you're letting AI agents write most of your code — and increasingly, you are — you need two things: clear boundaries so the agent knows what's allowed, and enforcement that's actually running when you're not watching. This repo gives you both.

The core is a **Go MCP server** that validates every bash command, file edit, and git operation against your rules before it runs. Around that, there's a full set of skills, IDE integrations, and workflows that get the guardrails into your actual development process without a lot of ceremony.

## Quick Start

```bash
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template
cp .env.example .env  # Fill in your API keys
cd mcp-server && make compose-up
curl http://localhost:8081/health/ready
```

That's it. The MCP server is live, the web dashboard is at `http://localhost:8081/web`, and the API explorer is at `http://localhost:8081/docs`.

For applying to an existing repo, see [how-to-apply.md](docs/getting-started/how-to-apply.md). For the 5-minute setup, see [quick-setup.md](docs/getting-started/quick-setup.md).

## The Four Laws

These are the backbone. Everything else extends them.

1. **Read before editing** — Never modify code without reading it first.
2. **Stay in scope** — Only touch files explicitly authorized.
3. **Verify before committing** — Test and check all changes.
4. **Halt when uncertain** — Ask for clarification instead of guessing.

See [four-laws.md](skills/shared-prompts/four-laws.md) and [halt-conditions.md](skills/shared-prompts/halt-conditions.md) for the canonical prompts.

## What's in the Box

**MCP Server** (`mcp-server/`) — Go server with 37 tools, 11 resources, and a stateless StreamableHTTP transport (`POST /mcp`, no session management). Validates bash, file edits, git ops, and commits — and now defends against prompt injection, classifies content against an S1–S15 safety taxonomy, and sandboxes execution across L0–L2 levels. Backed by PostgreSQL 16 and Redis 7. Includes a web UI, OpenAPI 3.1 spec, and 31 REST endpoints including `/api/v1/policy/check` for CI/CD gating.

**IDE Integrations** (`docs/integrations/`) — Native skills and rules for Claude Code, Cursor, OpenCode, Windsurf, and GitHub Copilot. Not generic prompts — each one is tailored to the platform's actual config format.

**Skills** (`skills/shared-prompts/`) — Nine canonical shared prompts covering architecture, error recovery, scope validation, production-first thinking, and vibe coding. The same prompts work across every supported IDE.

**Workflows** (`docs/workflows/`) — Twelve operational procedures: execution, escalation, code review, commit workflow, branch strategy, push safety, regression prevention, rollback, testing, and MCP checkpointing.

**Standards** (`docs/standards/`) — Twenty-five engineering standards covering test/production separation, API specs, dependency governance, logging, rate limiting, retry/degradation, timeouts, operational circuit breakers, and prompting practices.

**Examples** (`examples/`) — Fourteen languages: Go, TypeScript, Rust, Python, Java, Swift, Dart/Flutter, GDScript, Scala, R, C#, C++, PHP, and Ruby. Each demonstrates guardrails patterns in that language's idioms.

## What's New in v3.7

Through the summer we closed the six "2026 guardrail gaps" — the places an AI coding agent could still slip past the rules. Each is a new subsystem in the MCP server, with a design spec under `docs/specs/guardrail-gaps-2026/`:

- **Prompt injection defense (Spec 01)** — a four-layer pipeline (pattern → perplexity → classifier → LLM self-check) that catches injected instructions before they reach your agent, with per-source trust policies.
- **Semantic content filtering (Spec 02)** — safety classification over an S1–S15 taxonomy (Llama Guard plus two code-specific categories), with policies, thresholds, and overrides per category or rule.
- **Runtime sandbox isolation (Spec 03)** — three levels: L0 runs in-process, L1 uses `unshare` namespaces, L2 uses rootless podman/docker. CPU, memory, and PID limits, plus network isolation — and when `AllowedHosts` is set, L2 egress goes through a local proxy instead of the open network.
- **Multi-agent safety policies (Spec 04)** — guardrails for when several agents work the same codebase: scan-and-block or scan-and-warn chains, constraint resolution, and validators for the Four Laws.
- **Indirect injection / provenance (Spec 05)** — tracks where content came from, decodes obfuscated payloads (ROT13, base64), and decides how much to trust a source.
- **Regulatory compliance mapping (Spec 06)** — a map from each guardrail feature to the frameworks it helps you satisfy (GDPR, SOC 2, ISO 27001, the EU AI Act, and more).

Balancing all of that is the point: these guardrails run in the background so the agent can move at full speed and you can stop second-guessing everything it does.

## Atlas Cloud (Sponsored)

This project is sponsored by [Atlas Cloud for Open Source](https://www.atlascloud.ai/?ref=F6TYTG) — $50/month in credits across 300+ image, video, audio, 3D, and LLM models. The guardrails' AI-backed checks (content-safety, AI advisors, output validation) can route through Atlas instead of pay-per-use endpoints. See [atlas-cloud.md](docs/integrations/atlas-cloud.md) for setup.

## Project Structure

```
agent-guardrails-template/
├── README.md                    ← You are here
├── index-map.md                ← Keyword navigation (saves 60-80% tokens)
├── CLAUDE.md                   ← Claude Code context
├── CHANGELOG.md                ← Release history
├── CONTRIBUTING.md             ← How to contribute
├── docker-compose.yml          ← Local dev stack
├── .github/FUNDING.yml         ← Sponsor this project
├── docs/
│   ├── getting-started/        ← Quick setup, apply guide, core rules
│   ├── integrations/           ← Claude Code, Cursor, OpenCode, Windsurf, Copilot, Atlas
│   ├── workflows/              ← 12 operational procedures
│   ├── standards/              ← 25 engineering standards
│   ├── ai-dev/                 ← AI-assisted dev patterns
│   ├── security/               ← Security audit guides
│   ├── enterprise/             ← Enterprise patterns
│   ├── teams/                  ← Team management
│   ├── accessibility/          ← WCAG 3.0+ compliance
│   ├── spatial/                 ← XR/VR/AR patterns
│   ├── ethical/                 ← Dark pattern prevention
│   ├── monetization/            ← IAP and economy guardrails
│   ├── multiplayer/             ← Chat moderation, fairness
│   ├── analytics/               ← Consent and data minimization
│   ├── deployment/              ← Cross-platform deployment
│   ├── ui-ux/                   ← Component standards
│   ├── advisors/                ← Cost, privacy, resilience
│   ├── architecture/            ← Architecture decision records
│   ├── rules/                   ← Rule definitions
│   ├── state/                   ← State management patterns
│   ├── generative/              ← Generative asset safety
│   └── releases/               ← Release archive
├── mcp-server/                 ← Go MCP server (PostgreSQL + Redis)
├── pi-extension/                ← Pi coding agent extension
├── examples/                    ← 14 language implementations
├── skills/shared-prompts/      ← 9 canonical prompts
├── scripts/                    ← Setup and utility tools
├── tests/                      ← Test suite
├── web/                        ← Web dashboard
└── ci/                         ← CI/CD configuration
```

All documents follow the **500-line max** rule for fast context loading. Use `index-map.md` for keyword-based navigation instead of reading the full tree.

## Version

**Current:** v3.7.1 (2026-08-23)

| Version | Date | Highlights |
|---------|------|------------|
| **v3.7.1** | 2026-08-23 | README refresh covering the v3.7 guardrail subsystems |
| **v3.7.0** | 2026-08-23 | Six guardrail subsystems, two QA passes, AllowedHosts egress filtering |
| **v3.6.0** | 2026-08-22 | Atlas Cloud sponsorship integration, GitHub Sponsors, README rewrite |
| **v3.5.0** | 2026-08-18 | Interactive bash permission prompts, danger allow-list, catastrophic type-back |
| **v3.4.0** | 2026-08-15 | Documentation reorganization, game/vision content split to private repos |
| **v3.3.0** | 2026-08-15 | Stateless StreamableHTTP transport, repo cleanup |
| **v2.6.0** | 2026-02-15 | Python → Go migration complete |

Full history in [CHANGELOG.md](CHANGELOG.md).

## License

BSD-3-Clause — see [LICENSE](LICENSE).

## Sponsor

If this project helps you, consider [sponsoring on GitHub](https://github.com/sponsors/TheArchitectit).

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-TheArchitectit-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/TheArchitectit)

---

Built by [TheArchitectit](https://github.com/TheArchitectit) with AI-assisted development.
