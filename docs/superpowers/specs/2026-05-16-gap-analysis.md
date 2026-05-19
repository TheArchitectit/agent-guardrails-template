# Gap Analysis: Pi Guardrails vs. Existing Systems

**Date:** 2026-05-16
**Author:** PO (generated)
**Purpose:** Identify what pi's guardrails framework does well, where it lags, and where it has no competition. Informs backlog prioritization.

---

## Executive Summary

Pi's Four Laws framework is **unique in its focus on agent accountability** — no other system implements progressive discipline (three-strikes + HALT) or the read-before-edit requirement as a hard enforcement. However, pi is weak or absent in several areas that competitors treat as table stakes: prompt injection defense, sandboxing, output validation, CI/CD integration, and team-level policy enforcement. This gap analysis identifies 23 specific gaps and 7 areas where pi is ahead.

---

## Systems Compared

| System | Type | License | Primary Focus |
|--------|------|---------|---------------|
| **Claude Code** (Anthropic) | Native agent safety | Proprietary | Permission modes, tool approval, sandboxing |
| **Guardrails AI** | Python/JS validation framework | Apache 2.0 | LLM output validation, structured generation |
| **NeMo Guardrails** (NVIDIA) | Programmable guardrails | Apache 2.0 | Input/output/dialog rails, Colang DSL |
| **OpenHands** | Open source AI dev agent | MIT | Docker sandbox isolation |
| **Cursor** | IDE AI assistant | Proprietary | .cursorrules, agent mode confirmations |
| **Rebuff** (Protect AI) | Prompt injection detector | Apache 2.0 | Self-hardening injection defense |
| **Prompt Armor** | Third-party AI risk | Commercial | Vendor monitoring, injection detection |
| **LangChain/LangGraph** | Agent framework | MIT | HumanApproval, tool restrictions |
| **agent-guardrails-template** (pi) | Extension + MCP server | Template | Four Laws, progressive discipline |

---

## Safety Categories Matrix

| Category | Claude Code | Guardrails AI | NeMo | OpenHands | Rebuff | Pi |
|----------|:-----------:|:------------:|:----:|:---------:|:------:|:--:|
| Scope enforcement | Yes | No | No | Docker | No | **Yes** |
| Read-before-edit | No | No | No | No | No | **Yes** |
| Progressive discipline | No | No | No | No | No | **Yes** |
| Bash/shell safety | Yes | No | No | Docker | No | **Partial** |
| Prompt injection defense | No | Partial | Yes | No | Yes | No |
| Output validation | No | Yes | Yes | No | No | No |
| Sensitive data detection | No | Yes | Yes | Docker | No | No |
| Sandbox/isolation | Yes | No | Docker | Docker | No | No |
| Permission modes | Yes | No | No | No | No | No |
| Tool-level permissions | Yes | No | Execution rails | Docker | No | No |
| Content filtering | No | Yes | Yes | No | Yes | No |
| Canary tokens | No | No | No | No | Yes | No |
| CI/CD integration | Yes (GH Actions) | No | No | Docker | No | No |
| Team policy enforcement | No | No | No | No | No | No |
| Real-time dashboard | No | No | No | No | No | **Yes** |
| Session tracking | No | No | No | No | No | **Yes** |

---

## Detailed Gap Analysis

### GAPS: Where Pi Is Behind

#### GAP-1: No Prompt Injection Defense
**Severity: HIGH**
**Competitors:** NeMo Guardrails (built-in jailbreak/injection detection), Rebuff (self-hardening injection detector with canary tokens + vector DB), Prompt Armor (third-party monitoring)

Pi has zero detection or defense against prompt injection attacks. An attacker could craft a prompt that instructs the pi agent to bypass guardrails entirely. The agent would comply because there's no input rail filtering malicious instructions.

**What's needed:**
- Input rail that scans user prompts and tool outputs for injection patterns
- Canary token insertion to detect when injection succeeds
- Integration with an existing detection library (Rebuff is Apache 2.0)
- Severity scoring: low confidence = warn, high confidence = block + log

#### GAP-2: No Output Validation / Sensitive Data Filtering
**Severity: HIGH**
**Competitors:** Guardrails AI (Pydantic-based output validators), NeMo (output rails with sensitive data masking), Guardrails AI (detects toxic content, competitor mentions, PII)

Pi never validates what the agent outputs. If the agent reads a file containing API keys, it could echo them in a response or commit them to git. No PII detection, no secret scanning of agent output.

**What's needed:**
- Output rail that scans agent responses before delivery
- PII/entity detection (PERSON, EMAIL, API_KEY, etc.)
- Secret scanning on all output (grep for patterns: AWS keys, GitHub tokens, etc.)
- Configurable redaction or blocking

#### GAP-3: No Sandbox / Process Isolation
**Severity: MEDIUM**
**Competitors:** Claude Code (Docker-based sandbox mode), OpenHands (Docker container isolation), Claude Code (permission modes restrict what tools can do)

Pi runs in the user's full environment with no process isolation. The bash safety denylist is a soft guard — a sophisticated agent or injected prompt could circumvent it.

**What's needed:**
- Docker-based sandbox for tool execution (read-only filesystem mounts)
- Network isolation for untrusted operations
- Resource limits (CPU, memory, disk)
- This is a significant architectural change — consider as Phase 3+

#### GAP-4: No Tool-Level Permission System
**Severity: MEDIUM**
**Competitors:** Claude Code (per-tool approval: plan/auto/bypassPermissions), LangGraph (HumanApproval callback), Claude Code (allowlist/denylist for specific tools)

Pi treats all tools equally — if a tool is registered, it can be called. There's no per-tool approval workflow, no "ask before executing" for dangerous tools, and no allowlist/denylist.

**What's needed:**
- Per-tool permission levels: `auto`, `ask`, `blocked`
- Configurable tool permission matrix (e.g., `Bash: ask`, `Write: auto`, `Edit: auto`)
- Session-level and project-level permission overrides
- Integration with the existing event handler system

#### GAP-5: No CI/CD Integration
**Severity: MEDIUM**
**Competitors:** Claude Code (GitHub Actions, GitLab CI/CD), Prompt Armor (continuous monitoring)

Pi guardrails only run inside the pi agent session. There's no way to enforce guardrails in CI pipelines, no pre-commit hooks, no PR-level validation.

**What's needed:**
- GitHub Action that runs guardrails checks on PRs
- Pre-commit hook that validates files before commit
- Git hook integration for push validation
- CI report format (SARIF, GitHub annotations)

#### GAP-6: No Team-Level Policy Enforcement
**Severity: MEDIUM**
**Competitors:** Prompt Armor (vendor-level policies), Claude Code (CLAUDE.md for project-level rules)

Pi's guardrails are per-session. There's no way for a team lead to define policies that apply to all pi agents across the organization, no RBAC for who can modify guardrails settings.

**What's needed:**
- Organization-level guardrails config (loaded from a shared location)
- RBAC: who can modify scope, who can reset strikes, who can approve HALT overrides
- Policy inheritance: org > team > project > session
- Audit log for policy changes

#### GAP-7: No Bash Command Classification
**Severity: MEDIUM**
**Competitors:** Claude Code (command allowlist/denylist with pattern matching)

Pi's bash safety uses a hardcoded denylist of ~15 patterns. This is fragile and doesn't adapt to project-specific needs. No allowlist support, no command classification (read-only vs. destructive vs. network).

**What's needed:**
- Command classification: `read_only`, `constructive`, `destructive`, `network`
- Configurable allowlist and denylist per project
- Pattern-based matching (glob/regex) instead of hardcoded list
- Destructive command confirmation workflow

#### GAP-8: No Content Filtering / Topic Control
**Severity: LOW**
**Competitors:** NeMo Guardrails (topic control, topic rails), Guardrails AI (toxicity detection)

Pi doesn't filter what the agent can discuss or generate. NeMo can prevent agents from discussing certain topics entirely.

**What's needed:**
- Configurable topic allowlist/denylist
- Content safety checks on agent output
- This is likely low priority for a coding agent, but relevant for agents with broader capabilities

#### GAP-9: No Canary Token / Leak Detection
**Severity: LOW**
**Competitors:** Rebuff (canary tokens inserted into prompts), Prompt Armor (leak detection)

If the pi agent leaks sensitive information through an indirect prompt injection, there's no way to detect it after the fact.

**What's needed:**
- Canary tokens inserted into sensitive files that the agent reads
- Detection of canary tokens in agent output or external API calls
- Alert when a canary is triggered

---

### GAPS: Where Pi Is Equal

#### GAP-10: File Read Tracking
**Status: Equivalent**
NeMo and Guardrails AI track input/output but not at the file level. Pi's file read tracking is more granular than most systems. However, it only tracks reads within the pi session — no cross-session aggregation.

#### GAP-11: Session Lifecycle Management
**Status: Equivalent**
Most systems have session concepts but don't persist state across sessions like pi does. Pi's session store is adequate but could benefit from the team-level aggregation mentioned in GAP-6.

---

### AHEAD: Where Pi Leads

#### AHEAD-1: Progressive Discipline (Three Strikes + HALT)
**Unique in market.** No other system implements a graduated response to policy violations. This is pi's strongest differentiator.

#### AHEAD-2: Read-Before-Edit Enforcement
**Unique in market.** No other system requires the agent to read a file before editing it. This is a powerful defense against uninformed edits.

#### AHEAD-3: Real-Time TUI Dashboard
**Ahead of most.** While some systems have dashboards (Guardrails AI has a server UI), pi's in-session TUI overlay with safety score is unique. Most competitors require switching to a separate web UI.

#### AHEAD-4: Scope Enforcement with File-Level Granularity
**Ahead of most.** Claude Code has some scope restrictions but not at the file-path-prefix level. Pi's scope enforcement is more precise than most systems.

#### AHEAD-5: Dual-Mode Architecture (Standalone + MCP Bridge)
**Ahead of most.** The ability to run fully standalone (no external dependencies) or with the Go MCP server for full validation is a unique architectural choice that makes deployment flexible.

#### AHEAD-6: Session-Aware State Tracking
**Ahead of most.** Pi tracks cumulative state (reads, strikes, scope) across a session with persistence. Most systems reset state per-interaction or don't track it at all.

#### AHEAD-7: Extension-Based Architecture
**Ahead of most.** As a pi extension, guardrails are opt-in and composable with other extensions. Most competitors are monolithic — you get their guardrails whether you want them or not.

---

## Priority Recommendations

Based on severity, competitor coverage, and implementation effort:

| Priority | Gap | Effort | Impact |
|----------|-----|--------|--------|
| P0 | GAP-1: Prompt injection defense | Medium | Critical safety gap |
| P0 | GAP-2: Output validation / sensitive data filtering | Medium | Prevents data leaks |
| P1 | GAP-4: Tool-level permission system | Low-Medium | Table stakes for agent safety |
| P1 | GAP-7: Bash command classification | Low | Improves existing feature |
| P2 | GAP-5: CI/CD integration | Medium | Extends guardrails outside session |
| P2 | GAP-6: Team-level policy enforcement | Medium | Needed for enterprise adoption |
| P3 | GAP-3: Sandbox / process isolation | High | Architectural change, defer |
| P3 | GAP-8: Content filtering | Low | Low priority for coding agents |
| P3 | GAP-9: Canary token / leak detection | Low | Niche, defer |

---

## What Pi Should NOT Copy

These are features competitors have that don't fit pi's architecture:

1. **NeMo's Colang DSL** — Too heavy for a coding agent. Pi's event handler model is simpler and more appropriate.
2. **Guardrails AI's Pydantic validators** — Designed for structured LLM output, not file edits. Pi's TypeBox schemas serve the same purpose differently.
3. **Rebuff's self-hardening model** — Requires training/fine-tuning. Pi should use pre-trained detection, not build its own.
4. **OpenHands' full Docker isolation** — Too restrictive for a coding agent that needs filesystem access. Use sandboxing selectively, not globally.
5. **Prompt Armor's vendor monitoring** — Enterprise TPRM, not relevant to a coding agent extension.

---

## Competitive Positioning

Pi's unique value proposition: **"The only agent guardrails system that holds agents accountable through progressive discipline."**

No other system:
- Requires agents to read before editing
- Implements three-strikes escalation
- Halts sessions after repeated violations
- Provides real-time in-session safety dashboards
- Works as an opt-in extension (not forced)

The gaps above are real but fixable. The unique differentiators are hard to replicate. Prioritize closing the gaps while doubling down on what makes pi unique.
