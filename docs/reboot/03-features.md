# 100 next features, each with a plan
Prepared Oct 1, 2026. Planning document only. Every feature is a proposal; nothing is built. Each has a label for what it can honestly enforce. 'Blocking' needs a demonstrated interception and deny path on a named host version, else it is shown as 'not enforced'. Effort: S under 1 week, M 1-3 weeks, L over 3 weeks for one engineer (rough, unvalidated).

Order of work: M0 then M1 in sequence; later milestones can overlap. Each feature also needs its own OpenSpec change before build.

## A. Core server and transport

### F001 Tool manifest generator
- Plan: Run a live server, call tools/list, emit a versioned JSON manifest; CI fails if README tool count differs.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F002 stdio transport
- Plan: Only after B01 proves a host needs it: add stdio entrypoint beside StreamableHTTP, share handler code, test against two hosts.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F003 Zero-config local mode
- Plan: Prove a single-command local start with bundled defaults; document exactly which backing services remain required.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F004 SQLite storage option
- Plan: Introduce a storage interface, port one table set, run the same conformance suite on both backends. No parity claim until it passes.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F005 Optional Redis
- Plan: Make cache a soft dependency with a tested degraded mode; degraded state is reported, never silent.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F006 Health and readiness endpoints
- Plan: Add /healthz and /readyz that report policy loaded, store reachable, classifier available; unknown is not ready.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F007 Graceful error taxonomy
- Plan: Define PASS/FAIL/UNKNOWN/ERROR result codes; map every internal failure to one; add a test that no error maps to PASS.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F008 Profile system
- Plan: Named profiles (core, strict, extended) loaded from config; non-core tools off by default; test that each profile exposes the expected tool set.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F009 Single static binary release
- Plan: Cross-compile per OS/arch in CI, smoke-test each artifact, publish checksums.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F010 Container image
- Plan: Minimal non-root image, pinned base, SBOM attached, run read-only filesystem test.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

## B. Policy engine

### F011 Policy-as-data format
- Plan: Spec a versioned YAML/JSON policy schema; validate on load; reject unknown fields rather than ignore them.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F012 Four Laws pack
- Plan: Ship Four Laws as a versioned policy resource with tests proving each law maps to at least one rule or guidance entry.
- Label: Advisory (host must choose to call it)
- Effort: S  |  Milestone: M1
- Done when: Docs state it is advisory; test shows the host can skip it.

### F013 Scope validator
- Plan: Given declared paths/globs, check proposed file operations against scope; return FAIL on out-of-scope with the matched rule.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F014 Halt conditions
- Plan: Encode halt triggers (uncertainty, unmet precondition) as machine-readable states the host can surface; advisory unless wrapped.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M2
- Done when: Docs state it is advisory; test shows the host can skip it.

### F015 Three-strikes tracker
- Plan: Count repeated failures per task/session; at 3 return HALT_RECOMMENDED. Persist counter, test reset rules.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M2
- Done when: Docs state it is advisory; test shows the host can skip it.

### F016 Policy versioning and pinning
- Plan: Policies carry semver and content hash; every decision records both; pin in config.
- Label: Checked operation (server validates an operation it controls)
- Effort: S  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F017 Policy diff tool
- Plan: Command that shows what changes between two policy versions in plain language and flags newly permissive rules.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F018 Policy test harness
- Plan: Table-driven cases (input, expected verdict) per rule, runnable by users on their own packs.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F019 Domain packs
- Plan: Split core, web, infrastructure, game packs; install by name; each pack has its own tests and owner notes.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F020 Org override layers
- Plan: Layered config (default, org, repo, session) with explicit precedence; show effective policy with source per rule.
- Label: Checked operation (server validates an operation it controls)
- Effort: L  |  Milestone: M4
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

## C. Destructive-action and git safety

### F021 Dangerous command classifier
- Plan: Rule set for rm -rf, force push, reset --hard, branch delete, history rewrite; table-driven tests with benign look-alikes.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F022 Force-push permit
- Plan: Server issues a short-lived permit bound to repo, branch, remote and commit; push wrapper checks it. Without the wrapper it is advisory.
- Label: Blocking only with a tested hook/wrapper; otherwise advisory
- Effort: L  |  Milestone: M3
- Done when: Recorded demo and test of interception AND deny path on a pinned host version. No demo means label stays 'not enforced'.

### F023 Protected branch rules
- Plan: Config list of protected refs; verdict FAIL for operations that rewrite or delete them.
- Label: Checked operation (server validates an operation it controls)
- Effort: S  |  Milestone: M2
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F024 Pre-destructive snapshot prompt
- Plan: Before a risky op, recommend/create a bundle or tag backup and record its ref in the evidence log.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M2
- Done when: Docs state it is advisory; test shows the host can skip it.

### F025 Backup verification check
- Plan: Check that a named backup exists and restores to a temp dir before approving a destructive step.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M3
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F026 Git hook pack
- Plan: Installable pre-push and pre-commit hooks that call the server; document that hooks are bypassable with --no-verify.
- Label: Blocking only with a tested hook/wrapper; otherwise advisory
- Effort: M  |  Milestone: M3
- Done when: Recorded demo and test of interception AND deny path on a pinned host version. No demo means label stays 'not enforced'.

### F027 Mass-delete threshold
- Plan: Flag operations deleting more than N files or more than X% of tree; thresholds configurable.
- Label: Checked operation (server validates an operation it controls)
- Effort: S  |  Milestone: M2
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F028 History-rewrite detector
- Plan: Compare ref before/after in CI to detect non-fast-forward updates on protected refs and alert.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M4
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F029 Recovery runbook generator
- Plan: Given a repo, output tested steps for restoring from tags, reflog, forks and mirrors.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F030 Repo mirror reminder
- Plan: Advisory check that an off-host mirror or tag archive exists and is recent.
- Label: Advisory (host must choose to call it)
- Effort: S  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

## D. Host integrations

### F031 VS Code Copilot MCP setup
- Plan: Extension writes a tested MCP config entry per current VS Code docs; status bar shows connection truth.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M1
- Done when: Docs state it is advisory; test shows the host can skip it.

### F032 Copilot status panel
- Plan: Panel listing server, profile, policy version, last check, and an explicit label per path: advisory, checked, blocking, not enforced.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F033 Copilot custom instructions pack
- Plan: Ship instruction/prompt files that ask the agent to call checks; labeled advisory.
- Label: Advisory (host must choose to call it)
- Effort: S  |  Milestone: M1
- Done when: Docs state it is advisory; test shows the host can skip it.

### F034 Claude Code integration
- Plan: Test native MCP config and hooks on a pinned version; document which hook events can actually deny.
- Label: Blocking only with a tested hook/wrapper; otherwise advisory
- Effort: M  |  Milestone: M2
- Done when: Recorded demo and test of interception AND deny path on a pinned host version. No demo means label stays 'not enforced'.

### F035 Cursor integration
- Plan: Test MCP config and rules files on a pinned version; same evidence rules.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M2
- Done when: Docs state it is advisory; test shows the host can skip it.

### F036 JetBrains adapter revival
- Plan: Restore existing adapter, build on a pinned IDE version, ship only if tests pass.
- Label: Advisory (host must choose to call it)
- Effort: L  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F037 Neovim and Vim adapters
- Plan: Revive existing code, add smoke tests in CI, ship as separate packages.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M4
- Done when: Docs state it is advisory; test shows the host can skip it.

### F038 Windsurf and Zed configs
- Plan: Verify current config formats; publish copy-paste setups with tested versions listed.
- Label: Advisory (host must choose to call it)
- Effort: S  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F039 Gemini CLI and Codex CLI setups
- Plan: Same pattern: pin version, test, document hook limits.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F040 Host compatibility matrix
- Plan: Generated table: host, version, tested date, advisory/checked/blocking per path. Stale rows expire automatically.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

## E. Secrets and content safety

### F041 Secret pattern scanner
- Plan: Scan diffs and files for key/token patterns with a documented false-positive rate; never print matched values.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F042 Pre-commit secret check
- Plan: Hook plus MCP tool; result lists file and rule only, redacted.
- Label: Blocking only with a tested hook/wrapper; otherwise advisory
- Effort: S  |  Milestone: M2
- Done when: Recorded demo and test of interception AND deny path on a pinned host version. No demo means label stays 'not enforced'.

### F043 Secret-in-history scan
- Plan: On-demand scan of full history with redacted output and a rotation checklist.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M3
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F044 Prompt injection screen
- Plan: Port existing layered checks, publish a labeled test corpus and honest recall/precision numbers.
- Label: Advisory (host must choose to call it)
- Effort: L  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F045 Repo-instruction trust boundary
- Plan: Treat instruction text found in repos/docs as untrusted data; flag mandatory-sounding directives to agents.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F046 Content filter pack
- Plan: Port existing categories behind a profile; document limits; no compliance claims.
- Label: Advisory (host must choose to call it)
- Effort: L  |  Milestone: M4
- Done when: Docs state it is advisory; test shows the host can skip it.

### F047 Sandbox level reporting
- Plan: Report which isolation level the host actually provides; unknown stays unknown.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M4
- Done when: Docs state it is advisory; test shows the host can skip it.

### F048 Path traversal and symlink check
- Plan: Normalize paths, resolve symlinks, reject escapes of workspace; test with fixtures.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M2
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F049 Network egress advisory
- Plan: List outbound hosts in a proposed command or config and ask for confirmation; advisory only.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M4
- Done when: Docs state it is advisory; test shows the host can skip it.

### F050 Redaction library
- Plan: Shared redactor used by logs, evidence and error messages; fuzz tests prove no leak of seeded canaries.
- Label: Checked operation (server validates an operation it controls)
- Effort: M  |  Milestone: M1
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

## F. Evidence, audit and reporting

### F051 Decision log
- Plan: Append-only log of each check: input hash, policy version, verdict, time. No raw secrets.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F052 Evidence bundle export
- Plan: Command that exports a run's decisions to a signed JSON/Markdown bundle.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F053 Hash-chained ledger
- Plan: Each entry includes previous hash; verify command detects edits.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: L  |  Milestone: M4
- Done when: Reproducible command or artifact check recorded in CI.

### F054 Session summary
- Plan: Plain-language end-of-session report: what was checked, what was skipped, what is UNKNOWN.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F055 Skipped-check detector
- Plan: Compare expected checks for a task against checks actually called; report gaps.
- Label: Advisory (host must choose to call it)
- Effort: M  |  Milestone: M3
- Done when: Docs state it is advisory; test shows the host can skip it.

### F056 PR comment reporter
- Plan: GitHub Action that posts the evidence summary on a PR.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M3
- Done when: Reproducible command or artifact check recorded in CI.

### F057 SARIF output
- Plan: Export findings as SARIF for code scanning UIs.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M4
- Done when: Reproducible command or artifact check recorded in CI.

### F058 Metrics endpoint
- Plan: Expose counters (checks, verdict mix, latency) for local dashboards; no user content.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M4
- Done when: Reproducible command or artifact check recorded in CI.

### F059 Compliance mapping notes
- Plan: Doc mapping rules to common control language, labeled as mapping and not certification.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M5
- Done when: Reproducible command or artifact check recorded in CI.

### F060 Incident report template
- Plan: Template and generator for a factual write-up after a near-miss, with redaction pass.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

## G. Developer experience and CLI

### F061 guardrails init
- Plan: Interactive setup that detects host, writes config, runs a self-test, prints what is enforced and what is not.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F062 guardrails doctor
- Plan: Diagnose server, policy, host connection, versions; each check ends PASS/FAIL/UNKNOWN.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F063 guardrails check CLI
- Plan: Run any check from the shell for use in scripts and CI.
- Label: Checked operation (server validates an operation it controls)
- Effort: S  |  Milestone: M2
- Done when: Unit tests incl. negative and benign look-alike cases; UNKNOWN/ERROR never returns PASS.

### F064 Dry-run mode
- Plan: Evaluate policy without recording or enforcing; clearly labeled output.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F065 Explain command
- Plan: For any verdict, print the rule, source line and how to fix or override legitimately.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F066 Config schema and editor completion
- Plan: Publish JSON schema; validate in editors.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: S  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F067 Uninstall and rollback
- Plan: One command removes configs, hooks and extensions, leaving a report of what was removed.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F068 Quickstart in under 5 minutes
- Plan: Timed, scripted quickstart tested monthly on clean machines; failures open issues.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M1
- Done when: Reproducible command or artifact check recorded in CI.

### F069 Example repos
- Plan: Three small repos showing a blocked deletion, scope violation and clean run, each reproducible.
- Label: Not a safety check (tooling, docs or reporting)
- Effort: M  |  Milestone: M2
- Done when: Reproducible command or artifact check recorded in CI.

### F070 Sample policy playground
- Plan: Static web page that evaluates sample inputs against a policy in the browser, no uploads.
- Label: Advisory (host must choose to call it)
- Effort: L  |  Milestone: M4
- Done when: Docs state it is advisory; test shows the host can skip it.

---

## Continued

The remaining material is in [100 next features — trust, supply chain, ecosystem and milestone view](./03-features-part2.md).
