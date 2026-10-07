# Agent Guardrails Template

Agent Guardrails is a Go MCP/REST service and a set of integration materials for AI-assisted software development. It provides validation tools, advisory agent instructions, policy material, and a web/API surface. It does **not** automatically intercept every action an AI host takes, and a client integration is not blocking unless a tested host hook or CI gate proves that behavior.

[![Version](https://img.shields.io/badge/version-v3.7.1-blue.svg)](./CHANGELOG.md)
[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev/)
[![License](https://img.shields.io/badge/License-BSD--3--Clause-blue.svg)](./LICENSE)
[![Powered by Atlas Cloud](https://www.atlascloud.ai/oss-program/powered-by-atlas-cloud.svg)](https://www.atlascloud.ai/?ref=F6TYTG)

## Current status

This is an active engineering repository, not a finished security product. The source-verified baseline is [`docs/platform-current-state.md`](docs/platform-current-state.md). It distinguishes what is wired and tested from what is conditional, library-only, proposed, or not yet connected to a production request path.

| Area | Current state | Do not infer |
|---|---|---|
| MCP and REST | Go service with bearer-protected `/mcp`, REST routes, health endpoints, web UI, and team tools such as `guardrail_team_health` | The server does not intercept actions an assistant never submits |
| Validation | Rule-based bash, file, and git validation tools are available | Tool availability is not host-level enforcement |
| Guardrail engines | Content classification tools exist; some engines require optional runtime configuration | Injection, sandbox, provenance, multi-agent, and compliance libraries are not automatically active protection |
| Outbound webhooks | Webhook delivery enforces an SSRF policy at delivery time — non-public destinations, redirects, and private/loopback dial attempts are refused (`internal/notifications`, tested) | This covers the webhook dispatch path only; it is not a general egress firewall |
| Authorization | Credentials resolve to scoped principals; scope × role × resource intersection is enforced and tested on web REST and the MCP full path | Credential lifecycle/rotation and some decision-audit rows are not yet exercised end to end |
| Integrations | Copilot, IDE, Pi, and other client material exists at mixed maturity | Every client/server pair is not yet contract-tested |
| CI | Hosted PR checks; trusted-main validation on the repo-scoped UCS03 Podman runner; security-matrix jobs (fail-closed registry, MCP full-path authz, secret redaction, webhook SSRF, mutation-kill) on protected-branch pushes; Gitleaks secret validation with a reviewed baseline; pattern-based regression gates | Green CI does not prove every runtime guardrail is wired; the full-history Gitleaks scan still exits 1 on real-looking findings that are deliberately not allowlisted pending rotation |

The tool registry contains a core set plus conditional tools. The registry maximum is not the live startup inventory; configuration and initialization determine what is actually exposed. See the [tool reference](docs/mcp-server/tools-reference.md) and [gap reconciliation](docs/specs/guardrail-gaps-2026/STATUS.md). Security-row status (WIRED / EXERCISED / NOT_EXERCISED / NOT_RUN) is reconciled against source per sprint in [docs/sprints/](docs/sprints/).

## Product family and responsibility boundaries

This repository is one part of a broader, still-proposed security family:

```text
Agent Guardrails Template   agent/runtime checks, MCP/REST policy, evidence
           │
           │ bounded, signed evidence; never effect authority
           ▼
AIGGP / DevGate             repository gates, coherence, CI evidence, runners
           │
           │ optional organization-consumer contract
           ▼
Go OpenAgentPlatform        native tenant, grant, action and effect authority

Rust radicalopenmcpplatform is a separate MCP/plugin server with its own
transport, filesystem and plugin-trust boundary. It is not the Go OAP authority.
```

The products are not merged by implication. Cross-product evidence is proposed, must bind the exact subject/policy/evaluator context, and never authorizes an OAP effect by itself. See the [system roadmap](docs/specs/09-system-roadmap-and-phase-gates.md), [authorization contract](docs/specs/11-authorization-scopes-and-roles.md), and [cross-product method](docs/specs/15-secure-cross-product-method/spec.md).

## Evaluate the repository locally

For source validation, use Go 1.25.5 as declared by `mcp-server/go.mod`:

```sh
git clone https://github.com/TheArchitectit/agent-guardrails-template.git
cd agent-guardrails-template/mcp-server
go test ./...
go build ./cmd/server
```

Some tests require Linux facilities or configured services. Record skipped or environment-specific failures; do not call an incomplete local run green. Running the service requires PostgreSQL, Redis, credentials, and an explicit migration step. The root Compose file is a local-development starting point, not a production deployment: it contains placeholder DB/Redis fallbacks and the application listeners are HTTP paths. Review [deployment and TLS](docs/specs/17-deployment-tls-and-secret-boundary.md), [readiness and migrations](docs/specs/18-migration-startup-and-readiness.md), and [Phase 0 CI gates](docs/specs/19-phase0-ci-security-gates.md) before exposing it.

## Advisory client setup

For Copilot-style advisory wiring into an existing project, from Bash or WSL:

```sh
bash scripts/guardrails-cli.sh init /path/to/project
bash scripts/guardrails-cli.sh doctor /path/to/project
```

This writes only the files recorded in the target project's manifest. `doctor` reports `UNKNOWN` when the local server is unavailable. The integration is advisory and does not block an agent. See [Copilot integration](integrations/copilot/README.md) and [how to apply the project](docs/getting-started/how-to-apply.md).

## Repository layout

| Path | Purpose |
|---|---|
| [`mcp-server/`](mcp-server/) | Go MCP/REST service and validation components; separate Go module |
| [`cmd/team-cli/`](cmd/team-cli/) | Go frontend for the Python team manager; separate module |
| [`scripts/`](scripts/) | Setup, validation, and team-management utilities |
| [`integrations/`](integrations/) and [`pi-extension/`](pi-extension/) | Client and host integration material |
| [`policy-packs/`](policy-packs/) and [`.guardrails/`](.guardrails/) | Policy and prevention-rule inputs; consumers vary by subsystem |
| [`examples/`](examples/) | Illustrative language examples; verify commands before advertising support |
| [`site/`](site/) and [`web/`](web/) | Separate site and web UI surfaces |
| [`docs/`](docs/) | Onboarding, architecture, operations, specifications, audits, and history |

Documentation is intentionally categorized but not physically reorganized in this pass. Use [`index-map.md`](index-map.md) for keyword navigation and [`toc.md`](toc.md) for the complete inventory. Historical and proposed material is labeled; file presence is not implementation evidence.

## Documentation entry points

- [Current platform baseline](docs/platform-current-state.md)
- [Getting started](docs/getting-started/quick-setup.md)
- [Apply to an existing repository](docs/getting-started/how-to-apply.md)
- [MCP tool reference](docs/mcp-server/tools-reference.md)
- [Architecture](docs/architecture/system-architecture.md)
- [Roadmap and phase gates](docs/specs/09-system-roadmap-and-phase-gates.md)
- [Security policy](SECURITY.md)
- [Contributing](CONTRIBUTING.md)
- [Keyword index](index-map.md)
- [Complete file listing](toc.md)

## Shared agent guidance

The [Four Laws](skills/shared-prompts/four-laws.md) and [halt conditions](skills/shared-prompts/halt-conditions.md) describe safe agent behavior. They are workflow guidance, not a substitute for authorization, host enforcement, or a production security boundary.

## Atlas Cloud sponsorship

This project is sponsored by [Atlas Cloud for Open Source](https://www.atlascloud.ai/?ref=F6TYTG) — $50/month in credits across 300+ image, video, audio, 3D, and LLM models. AI-backed checks can route through Atlas instead of pay-per-use endpoints. See [Atlas Cloud setup](docs/integrations/atlas-cloud.md).

## Version and history

The repository baseline is v3.7.1; current `main` includes subsequent security specifications and remediation planning. Treat [`CHANGELOG.md`](CHANGELOG.md) and the source-verified baseline as separate: release notes describe historical releases, while the baseline describes current wiring.

## License and support

BSD-3-Clause — see [LICENSE](LICENSE).

Built by [TheArchitectit](https://github.com/TheArchitectit) with AI-assisted development. If this project helps you, consider [sponsoring on GitHub](https://github.com/sponsors/TheArchitectit). Support funds development, testing infrastructure, and model/API costs.

| Service | Your Bonus | Details | Referral Code |
|---|---:|---|---|
| [**Neuralwatt**](https://portal.neuralwatt.com/auth/register?ref=NW-ROGER-ET3Y) | $5 in credits | Refer a friend; when they use $25 in compute, both earn $5 | `NW-ROGER-ET3Y` |
| [**Synthetic**](https://synthetic.new/?referral=UAWqkKQQLFkzMkY) | $10 in credits | Subscribe and both receive a $10 credit | `UAWqkKQQLFkzMkY` |
| [**Ozore**](https://ozore.com/?ref=cwe4kdx0) | 50% off first month | AI-ready cloud; use code `lundrog50` | `lundrog50` |

[![Buy Me a Coffee](https://img.shields.io/badge/Buy%20Me%20a%20Coffee-TheArchitectit-FFDD00?style=for-the-badge&logo=buy-me-a-coffee&logoColor=black)](https://www.buymeacoffee.com/TheArchitectit)
