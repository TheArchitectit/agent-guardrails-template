# 100 next features — trust, supply chain, ecosystem and milestone view

> Sections I and J plus the milestone view, split from the main feature breakdown for the 500-line cap.

**Part of:** [100 next features, each with a plan](./03-features.md)

---

## H. Community, docs and education

### F071 Honest README
- Plan: Lead with what it does, the advisory/checked/blocking table, and a 60-second demo; no unsourced claims.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M0
- Done when: Reproducible command or artifact check recorded in CI.

### F072 The near-loss story post
- Plan: Publish reviewed story as docs/story.md and blog post; factual and restrained.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M0
- Done when: Reproducible command or artifact check recorded in CI.

### F073 Safety principles doc
- Plan: Short doc: boundaries over vigilance, evidence over trust, unknown is not pass.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M0
- Done when: Reproducible command or artifact check recorded in CI.

### F074 Contributor guide and good-first-issues
- Plan: Label 15 small issues with clear acceptance; review within 48 hours.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F075 Threat model doc
- Plan: What the project defends against and what it does not; reviewed by outside readers.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F076 Docs site
- Plan: Searchable docs generated from policy and tool manifests; version selector.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F077 Video series
- Plan: Five short videos: story, install, blocked deletion, writing a rule, limits.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F078 Monthly safety notes
- Plan: Short public changelog of what improved and what failed; includes failures.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F079 Office hours
- Plan: Fortnightly open call or thread; notes published.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F080 Translations
- Plan: Community-led translations of README and quickstart, reviewed by native speakers.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

## I. Trust, supply chain and release

### F081 Signed releases
- Plan: Sign artifacts and checksums; document verification steps.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F082 SBOM per release
- Plan: Generate and attach CycloneDX or SPDX; diff between releases.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F083 Reproducible build check
- Plan: Rebuild in a second environment and compare hashes; publish result.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M4
- Done when: Reproducible command or artifact check recorded in CI.

### F084 Dependency policy and updates
- Plan: Pinned deps, scheduled review, license check; documented exceptions.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F085 Security policy and contact
- Plan: SECURITY.md with report channel and response targets; test the channel.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F086 Branch protection and repo backup
- Plan: Protect main, require review, mirror tags off-host; verify restore quarterly. Directly answers the near-loss.
- Label: Checked operation (server validates an operation it controls)
- Effort: S  |  Milestone: M0
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F087 MCP Registry listing
- Plan: After npm or other package publication and owner approval, publish via the official publisher; verify the listing.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F088 VS Code Marketplace listing
- Plan: Owner-approved publisher identity; prerelease first; verify install on clean profile.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F089 Third-party audit
- Plan: Fund or recruit an outside review of the check logic; publish findings.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F090 Vulnerability response drills
- Plan: Practice a mock disclosure each quarter; publish timing.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

## J. Ecosystem and advanced

### F091 Hook SDK
- Plan: Small library for writing host hooks that call the server and deny on FAIL/UNKNOWN by default.
- Label: Blocking only with a tested hook/wrapper; otherwise advisory
- Effort: L  |  Milestone: M4
- Done when: Recorded demo and test of interception AND deny path on a pinned host version. No demo means label stays 'not enforced'.

### F092 Permit tokens for operations
- Plan: Generalize the force-push permit: operation-bound, expiring, single-use.
- Label: Checked operation (server validates an operation it controls)
- Effort: L  |  Milestone: M4
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F093 Policy pack registry
- Plan: Index of community packs with signatures and test badges.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F094 GitHub Action marketplace action
- Plan: Run checks in CI and fail PRs with clear evidence.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M3
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F095 Agent run replay
- Plan: Replay a recorded session against a new policy version to see what would change.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F096 Team dashboard (optional)
- Plan: Separate optional app reading the evidence log; not part of the core product.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M6
- Done when: Reproducible command or artifact check recorded in CI.

### F097 Slack/Teams notifications
- Plan: Optional webhook on HALT or FAIL with redacted detail.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F098 Policy suggestions from incidents
- Plan: Turn a written incident report into a draft rule plus a failing test.
- Label: Advisory (host must choose to call it)
- Effort: L  |  Milestone: M5
- Done when: Docs state it is advisory; test shows the host can skip it.

### F099 Benchmark suite
- Plan: Public benchmark of agent safety scenarios; results include failures.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F100 Interop with AIGGP/DevGate
- Plan: Share policy schema and evidence format where tested; neither product requires the other.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M6
- Done when: Reproducible command or artifact check recorded in CI.

## Milestone view

**M0 Story and safety basics (target 76 to 100 stars)** (4): F071 Honest README; F072 The near-loss story post; F073 Safety principles doc; F086 Branch protection and repo backup

**M1 Working core and Copilot (100 to 250)** (21): F001 Tool manifest generator; F006 Health and readiness endpoints; F007 Graceful error taxonomy; F008 Profile system; F011 Policy-as-data format; F012 Four Laws pack; F013 Scope validator; F016 Policy versioning and pinning; F021 Dangerous command classifier; F031 VS Code Copilot MCP setup; F032 Copilot status panel; F033 Copilot custom instructions pack; F041 Secret pattern scanner; F050 Redaction library; F051 Decision log; F061 guardrails init; F062 guardrails doctor; F067 Uninstall and rollback; F068 Quickstart in under 5 minutes; F074 Contributor guide and good-first-issues; F085 Security policy and contact

**M2 Second hosts, trust basics (250 to 1,000)** (29): F002 stdio transport; F003 Zero-config local mode; F009 Single static binary release; F010 Container image; F014 Halt conditions; F015 Three-strikes tracker; F018 Policy test harness; F023 Protected branch rules; F024 Pre-destructive snapshot prompt; F027 Mass-delete threshold; F029 Recovery runbook generator; F034 Claude Code integration; F035 Cursor integration; F040 Host compatibility matrix; F042 Pre-commit secret check; F048 Path traversal and symlink check; F054 Session summary; F060 Incident report template; F063 guardrails check CLI; F064 Dry-run mode; F065 Explain command; F066 Config schema and editor completion; F069 Example repos; F075 Threat model doc; F077 Video series; F078 Monthly safety notes; F081 Signed releases; F082 SBOM per release; F084 Dependency policy and updates

**M3 Distribution and ecosystem entry (1,000 to 3,000)** (22): F004 SQLite storage option; F005 Optional Redis; F017 Policy diff tool; F019 Domain packs; F022 Force-push permit; F025 Backup verification check; F026 Git hook pack; F030 Repo mirror reminder; F036 JetBrains adapter revival; F038 Windsurf and Zed configs; F039 Gemini CLI and Codex CLI setups; F043 Secret-in-history scan; F044 Prompt injection screen; F045 Repo-instruction trust boundary; F052 Evidence bundle export; F055 Skipped-check detector; F056 PR comment reporter; F076 Docs site; F079 Office hours; F087 MCP Registry listing; F088 VS Code Marketplace listing; F094 GitHub Action marketplace action

**M4 Depth (3,000 to 10,000)** (13): F020 Org override layers; F028 History-rewrite detector; F037 Neovim and Vim adapters; F046 Content filter pack; F047 Sandbox level reporting; F049 Network egress advisory; F053 Hash-chained ledger; F057 SARIF output; F058 Metrics endpoint; F070 Sample policy playground; F083 Reproducible build check; F091 Hook SDK; F092 Permit tokens for operations

**M5 Standards and community scale (10,000 to 30,000)** (9): F059 Compliance mapping notes; F080 Translations; F089 Third-party audit; F090 Vulnerability response drills; F093 Policy pack registry; F095 Agent run replay; F097 Slack/Teams notifications; F098 Policy suggestions from incidents; F099 Benchmark suite

**M6 Platform interop (30,000 to 100,000)** (2): F096 Team dashboard (optional); F100 Interop with AIGGP/DevGate


