# OpenSpec sprint plans and agent assignment

This is the planning index for the canonical OpenSpec catalog and active change
proposals in [the OpenSpec index](../../../openspec/INDEX.md). The linked plans
are proposals, not claims that work has started, that all platforms implement a
capability, or that a previously closed specification must be reopened. Each
plan identifies the relevant source, intended work or verification, owner roles,
evidence and exit criteria. Confirm scope and current state at sprint start.

## Plan inventory

- [Canonical platform and CI specifications](canonical-plan-platform-ci.md)
- [Canonical runtime and integration specifications](canonical-plan-runtime-integrations.md)
- [Canonical security and conformance specifications](canonical-plan-security-conformance.md)
- [Canonical quality and evidence specifications](canonical-plan-quality-evidence.md)

- [Active platform and compatibility proposals](active-platform-compatibility.md)
- [Active runtime and evidence proposals](active-runtime-evidence.md)
- [Active deployment and operations proposals](active-deployment-operations.md)
- [Active release and toolchain proposals](active-release-and-toolchain.md)

## Agent roles and model allocation

The repository's [agent-team mapping](../../teams/team-tools-agent-mapping.md)
defines the following role types. Assign one accountable lead to each plan;
separate implementation from independent validation when feasible. Model names
below describe a suggested class, not an access guarantee or a mandatory vendor.
The user prefers Luna agents for implementation, test execution and evidence;
the lead retains design decisions, independent validation and final sign-off.

| Agent type | Planning responsibility | Suggested model class | Typical effort per scoped task |
| --- | --- | --- | --- |
| planner | Pin source, dependencies, scope, and sprint acceptance | Luna for inventory; lead for decisions | 0.5–2 hours |
| architect | Resolve boundaries, interfaces, and tradeoffs | Lead-capable reasoning model | 1–4 hours |
| infrastructure | Prepare disposable hosts and controlled access | Luna, lead reviews sensitive changes | 1–4 hours |
| platform | Implement runner, workflow, and artifact wiring | Luna, lead reviews trust boundaries | 1–5 hours |
| backend | Implement MCP/server changes | Luna for bounded code; lead for architecture | 1–6 hours |
| frontend | Implement and package IDE or UI changes | Luna for bounded code; lead for UX decisions | 1–6 hours |
| security | Review boundaries, secrets and negative cases | Independent lead-capable reviewer | 1–4 hours |
| qa | Run independent assertions and preserve receipts | Luna under lead review | 1–5 hours |
| sre | Verify operations, diagnostics and recovery | Luna for collection; lead for risk decisions | 1–4 hours |
| ops | Verify release, rollback and operator documentation | Luna for preparation; lead for release decision | 0.5–3 hours |

These are rough elapsed active-work estimates, not model benchmarks or delivery
commitments. They exclude queue time, model/API availability, host provisioning,
external review, and remediation discovered during execution. Parallel work can
reduce elapsed time; shared fixtures should not be counted repeatedly. Record
actual start/end times, model and reasoning settings, failures and retries in
the sprint evidence before recalibrating. Use Luna for bounded work and evidence
preparation; the lead owns architecture and sign-off. Treat model availability
as environment-specific. Do not infer that a given editor exposes a model.

## Coding-platform qualification matrix

Each plan applies the shared MCP/server contract where its capability is relevant;
client-specific work is required only for platforms that can exercise that
boundary. Record an explicit **not applicable**, **not tested**, **blocked**,
**passed**, or **failed** with version, commit and evidence for every platform
considered. A config file or package build alone does not prove a live client.

| Coding platform or adapter | Existing repository surface | Qualification work |
| --- | --- | --- |
| Codex | MCP client acceptance and smoke command | On demand, connect the actual client to an isolated built image and verify initialization, enumeration and a benign call; unattended builds use deterministic protocol checks without a model |
| Claude Code | `.claude/` and setup installer | Verify installation and an applicable real client session |
| Cursor | `.cursor/rules/` and setup installer | Verify rule installation and applicable editor session |
| Windsurf | `.windsurfrules` and setup installer | Verify rule installation and applicable editor session |
| OpenCode | `.opencode/` and setup installer | Verify configuration and applicable agent session |
| GitHub Copilot | `.github/copilot-instructions.md` and setup installer | Verify instruction placement and applicable client session |
| Pi | `pi-extension/` package and tests | Build and test package, then verify applicable Pi runtime behavior |
| VS Code | `ide/vscode-extension/` | Build and audit package, then verify applicable extension behavior |
| JetBrains IDE | `ide/` plugin and its documented lab package | Verify the exact package/commit in a supported IDE; retain version and license prerequisites |

The [integration setup guide](../../integrations/agents-and-skills-setup.md)
describes its five installer targets; Pi, VS Code, JetBrains and Codex use their
own integration surfaces and must not be described as installer targets without
separate implementation evidence. A plan may designate a platform not
applicable when the OpenSpec concerns repository-only or server-only behavior;
it must still name and test actual clients when it claims client coverage.

## Execution and evidence rules

1. Pin the OpenSpec, candidate source SHA, dependencies, platform versions and
   selected model. Review current requirements and distinguish implemented,
   unverified, and new work before estimating or assigning tasks.
2. Assign the plan's accountable role and parallel work. Capture scope,
   prerequisites, negative controls, tests, and a rollback or teardown method.
3. Run CI on its assigned runner; in this lab UCS03 is CI, while disposable
   application and host tests run on AI03–AI05. Keep synthetic fixtures unique,
   leave the laptop pilot and unrelated projects alone, and preserve data.
4. Record exact commands, SHAs, image digests, model/client versions, start/end
   times, runner/host, links and actual results. Mark skips, timeouts, failures,
   missing prerequisites, and unsupported surfaces; do not turn them green by
   analogy to a different environment.
5. An independent reviewer checks evidence against the plan and source, then the
   lead decides whether its exit criteria are met. Update the linked OpenSpec
   tasks and both indexes when a proposal is changed or archived.
