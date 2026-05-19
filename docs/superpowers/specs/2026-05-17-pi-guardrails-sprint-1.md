# Pi Guardrails Sprint 1: Gap Closure

**Date:** 2026-05-17
**Author:** PO (generated)
**Goal:** Close the top gaps identified in the gap analysis while preserving pi's unique differentiators (progressive discipline, read-before-edit, TUI dashboard)
**Duration:** 2 sprints (Sprint 1A: 1 week, Sprint 1B: 1 week)

---

## Sprint Strategy

Gap closure is split across two repos:

- **Template repo** (`agent-guardrails-template/`): shared patterns that benefit all agents (Claude Code, Cursor, pi, others). These are guardrails rules and reference implementations.
- **Pi extension** (`pi-extension/`): enforcement logic that requires pi-specific hooks (tool_call, tool_result handlers). These only benefit pi agents.

The split: **shared patterns in the template, enforcement logic in the pi extension.**

---

## Sprint 1A: Shared Guardrails Patterns (Template Repo)

These items go into the template repo so all agents benefit.

### Item 1: Bash Command Classification Engine
**Repo:** template
**Effort:** S (2 days)
**Files:** `guardrails/bash-classify.ts` (new), reference implementations for each agent
**Category:** Safety infrastructure

Create a reusable command classification system that replaces the hardcoded denylist.

**Implementation:**
```
guardrails/
  bash-classify.ts         # Command classifier (shared)
  bash-classify.test.ts    # Tests for classifier
  commands.json            # Command → category mapping (shared config)
```

**Command categories:**
- `read_only`: `ls`, `cat`, `find`, `grep`, `git status`, `git log`, `git diff`
- `constructive`: `git add`, `git commit`, `npm install`, `mkdir`, `touch`, `echo >`
- `destructive`: `rm`, `rm -rf`, `git reset`, `git checkout --`, `git stash drop`
- `network`: `curl`, `wget`, `ssh`, `scp`, `rsync`, `git push`, `git clone`
- `elevated`: `sudo`, `chmod 777`, `chown`, `rm -rf /`

**Rules:**
1. Each command classified as exactly one category
2. Configurable allowlist and denylist per project (in `.guardrails/config.json`)
3. Glob/regex pattern matching for flexible matching
4. Default deny: unknown commands blocked unless explicitly allowed
5. Override levels: project > user > system defaults

**Acceptance criteria:**
- [ ] All common shell commands classified
- [ ] Glob/regex pattern matching works
- [ ] Override levels work (project > user > system)
- [ ] Reference implementation for Claude Code and pi
- [ ] Tests pass

**Dependency:** None
**Blocked by:** None

---

### Item 2: Scope Patterns Library
**Repo:** template
**Effort:** S (1 day)
**Files:** `guardrails/scope-patterns.ts` (new), `guardrails/scope-patterns.test.ts`
**Category:** Scope enforcement

Provide reusable scope patterns that agents can import and configure.

**Implementation:**
```
guardrails/
  scope-patterns.ts        # Reusable scope patterns (shared)
  scope-patterns.test.ts   # Tests
```

**Pre-built patterns:**
- `src_only`: Only allow changes in `src/` directories
- `no_config`: Block changes to `*.config.*`, `.env`, `package.json`
- `no_lockfiles`: Block changes to `package-lock.json`, `yarn.lock`, `pnpm-lock.yaml`
- `src_and_tests`: Allow `src/` and `test/`/`__tests__/` only
- `full_project`: Allow everything (default)
- `custom`: User-defined pattern (glob or regex)

**Rules:**
1. Patterns are composable (e.g., `src_only` + `no_lockfiles`)
2. Patterns can be inverted (denylist mode vs. allowlist mode)
3. Patterns can be project-specific (in `.guardrails/config.json`)

**Acceptance criteria:**
- [ ] Pre-built patterns work
- [ ] Patterns are composable
- [ ] Patterns support both allowlist and denylist modes
- [ ] Tests pass

**Dependency:** None
**Blocked by:** None

---

### Item 3: Pre-commit Hook for Guardrails
**Repo:** template
**Effort:** S (2 days)
**Files:** `guardrails/hooks/pre-commit.sh` (new), `guardrails/hooks/install.sh` (new)
**Category:** CI/CD integration

Create a pre-commit hook that validates files before commit.

**Implementation:**
```
guardrails/
  hooks/
    pre-commit.sh          # Pre-commit hook (validates staged files)
    install.sh             # Installer for git hooks
```

**Hook behavior:**
1. When `git commit` runs, the hook fires before commit
2. Validates staged files against scope patterns
3. Validates staged files against sensitive data patterns (API keys, tokens)
4. If violations found: commit blocked, violations printed
5. If clean: commit proceeds

**Sensitive data patterns (regex):**
- AWS access key: `AKIA[0-9A-Z]{16}`
- GitHub token: `ghp_[a-zA-Z0-9]{36}`
- Generic API key: `(?i)(api_key|apikey|api-key)\s*[:=]\s*['"]?([a-zA-Z0-9\-_]{20,})['"]?`
- Private key block: `-----BEGIN (RSA |EC )?PRIVATE KEY-----`

**Acceptance criteria:**
- [ ] Pre-commit hook blocks commits with scope violations
- [ ] Pre-commit hook detects secrets in staged files
- [ ] Hook can be installed via `npm run install-hooks` or `./guardrails/hooks/install.sh`
- [ ] Hook can be skipped with `--no-verify` (standard git behavior)
- [ ] Tests pass

**Dependency:** Item 2 (scope patterns)
**Blocked by:** Item 2

---

### Item 4: Guardrails CI Action (GitHub Actions)
**Repo:** template
**Effort:** M (3 days)
**Files:** `.github/actions/guardrails/action.yml` (new), `guardrails/ci/validate.ts` (new)
**Category:** CI/CD integration

Create a GitHub Action that runs guardrails checks on PRs.

**Implementation:**
```
.github/
  actions/
    guardrails/
      action.yml           # GitHub Action definition
guardrails/
  ci/
    validate.ts            # CI validation logic (shared)
    format.ts              # Output formatters (SARIF, GitHub annotations)
```

**Action behavior:**
1. Triggered on `pull_request` events (opened, synchronize, reopened)
2. Checks out the code
3. Runs scope validation on changed files
4. Runs secret scanning on changed files
5. Outputs results as GitHub annotations
6. Can optionally output SARIF for GitHub Code Scanning

**Acceptance criteria:**
- [ ] GitHub Action works on PRs
- [ ] Annotations appear in PR diff view
- [ ] SARIF output works (optional)
- [ ] Action is reusable (workflow_call)
- [ ] Tests pass

**Dependency:** Item 1 (bash classification), Item 2 (scope patterns)
**Blocked by:** Items 1-2

---

## Sprint 1B: Pi Extension Enforcement (Pi Extension)

These items go into the pi extension and require pi-specific hooks.

### Item 5: Prompt Injection Defense
**Repo:** pi extension
**Effort:** M (4 days)
**Files:** `pi-extension/injection/` (new directory)
**Category:** Security (P0)

Add input rail that detects prompt injection attacks before they reach the pi agent.

**Implementation:**
```
pi-extension/
  injection/
    detector.ts            # Injection detection logic
    patterns.json          # Known injection patterns (regex/heuristic)
    canary.ts              # Canary token management
    detector.test.ts       # Tests
```

**Detection layers (in order):**
1. **Pattern matching** — regex against known injection patterns (OWASP LLM01 top patterns)
2. **Heuristic scoring** — token frequency analysis, instruction override attempts
3. **Canary detection** — detect when injected canary tokens appear in agent output

**Known injection patterns to detect:**
- `ignore previous instructions`
- `you are now in developer mode`
- `disregard all safety`
- `sudo mode`
- `override system prompt`
- `new instruction:`
- `### System:`
- `[SYSTEM]`
- Attempted tool_call injection (fake JSON in responses)

**Enforcement:**
- High confidence (>0.8): block the prompt, log violation, increment strikes
- Medium confidence (0.5-0.8): warn the agent, log for review, don't block
- Low confidence (<0.5): log only, no action

**Acceptance criteria:**
- [ ] Detects known injection patterns
- [ ] Canary tokens work (insert → detect in output → alert)
- [ ] High-confidence injections blocked + logged
- [ ] Medium-confidence injections warned
- [ ] Low-confidence injections logged
- [ ] Tests pass (include known attack vectors)
- [ ] Performance: <5ms per check

**Dependency:** None
**Blocked by:** None

---

### Item 6: Output Validation / Sensitive Data Filter
**Repo:** pi extension
**Effort:** M (3 days)
**Files:** `pi-extension/output-validator/` (new directory)
**Category:** Security (P0)

Add output rail that scans agent responses for sensitive data before delivery.

**Implementation:**
```
pi-extension/
  output-validator/
    validator.ts           # Output validation logic
    secrets.json           # Secret patterns (regex)
    pii.ts                 # PII detection
    redactor.ts            # Data redaction
    validator.test.ts      # Tests
```

**Validation layers:**
1. **Secret scanning** — regex against known secret patterns (AWS keys, GitHub tokens, API keys, private keys)
2. **PII detection** — detect emails, phone numbers, SSNs, credit card numbers
3. **Context-aware detection** — detect if agent is echoing back file contents that contain secrets

**Enforcement modes (configurable):**
- `block`: refuse to deliver the response, log violation
- `redact`: replace sensitive data with `[REDACTED]`, deliver redacted version
- `warn`: deliver the response but warn the user
- `log`: log only, no user-facing action

**Acceptance criteria:**
- [ ] Detects AWS keys, GitHub tokens, API keys, private keys
- [ ] Detects PII (emails, phones, SSNs, credit cards)
- [ ] Configurable enforcement modes work
- [ ] Redaction mode produces clean output
- [ ] Performance: <10ms per check
- [ ] Tests pass

**Dependency:** None
**Blocked by:** None

---

### Item 7: Per-Tool Permission System
**Repo:** pi extension
**Effort:** M (3 days)
**Files:** `pi-extension/tool-permissions/` (new directory)
**Category:** Safety infrastructure

Add per-tool permission levels that can be configured per project.

**Implementation:**
```
pi-extension/
  tool-permissions/
    permissions.ts         # Permission checking logic
    defaults.json          # Default permission levels
    permissions.test.ts    # Tests
```

**Permission levels:**
- `auto`: tool executes without asking (default for safe tools: Read, Glob, Grep, TaskList)
- `ask`: user must approve before execution (default for write tools: Edit, Write, NotebookEdit)
- `blocked`: tool cannot be called at all (configurable per project)

**Default permission matrix:**

| Tool | Default Level | Rationale |
|------|:---:|---|
| Read | auto | Safe, read-only |
| Glob | auto | Safe, read-only |
| Grep | auto | Safe, read-only |
| Edit | ask | Modifies files |
| Write | ask | Creates/overwrites files |
| NotebookEdit | ask | Modifies notebooks |
| Bash | ask | Can run arbitrary commands |
| Agent | ask | Spawns sub-agents |
| LSP | auto | Safe, read-only |
| TaskCreate | auto | Safe, internal |
| TaskUpdate | auto | Safe, internal |
| SendMessage | ask | Can message other agents |

**Configuration:**
```json
// .guardrails/config.json
{
  "tool_permissions": {
    "Bash": "blocked",
    "Edit": "auto",
    "Write": "auto"
  }
}
```

**Acceptance criteria:**
- [ ] Default permission matrix works
- [ ] Project-level overrides work
- [ ] `ask` level triggers user confirmation (if pi supports it)
- [ ] `blocked` level prevents tool execution
- [ ] Tests pass

**Dependency:** None
**Blocked by:** None

---

### Item 8: Team Policy Configuration
**Repo:** pi extension
**Effort:** M (3 days)
**Files:** `pi-extension/team-policy/` (new directory)
**Category:** Enterprise (P2)

Add organization-level guardrails configuration that applies to all pi sessions.

**Implementation:**
```
pi-extension/
  team-policy/
    policy.ts              # Policy loading and enforcement
    schema.ts              # Policy schema (TypeBox)
    policy.test.ts         # Tests
```

**Policy hierarchy (highest to lowest priority):**
1. Organization policy: `/etc/guardrails/policy.json`
2. User policy: `~/.pi/agent/extensions/pi-guardrails/policy.json`
3. Project policy: `.guardrails/config.json`
4. Session overrides: runtime only (not persisted)

**Policy schema:**
```json
{
  "organization": "acme-corp",
  "version": "1.0",
  "rules": {
    "scope": { "paths": ["src/", "lib/"], "deny": [".env", "*.config.*"] },
    "tool_permissions": { "Bash": "blocked", "Write": "ask" },
    "three_strikes": { "max_strikes": 2, "halt_action": "notify_admin" },
    "injection_defense": { "enabled": true, "confidence_threshold": 0.7 },
    "output_validation": { "enabled": true, "mode": "redact" },
    "audit_log": { "enabled": true, "retention_days": 90 }
  },
  "rbac": {
    "can_modify_policy": ["admin", "security_lead"],
    "can_reset_strikes": ["admin"],
    "can_override_halt": ["admin", "security_lead"]
  }
}
```

**Acceptance criteria:**
- [ ] Policy hierarchy works (org > user > project)
- [ ] Policy schema validation works
- [ ] RBAC rules enforced (who can modify, who can reset, who can override)
- [ ] Audit log captures policy changes
- [ ] Tests pass

**Dependency:** Item 5 (injection defense), Item 6 (output validation)
**Blocked by:** Items 5-6

---

### Item 9: Enhanced Bash Safety
**Repo:** pi extension
**Effort:** S (2 days)
**Files:** Update `pi-extension/standalone/bash-safety.ts`
**Category:** Safety improvement

Replace the hardcoded denylist with the shared bash classification engine from Item 1.

**Implementation:**
1. Import `guardrails/bash-classify.ts` from the template
2. Replace hardcoded `BLOCKED_COMMANDS` with classifier output
3. Add configurable project-level overrides
4. Add command confirmation for destructive commands (not just block)

**Acceptance criteria:**
- [ ] Uses shared classifier from template
- [ ] Project-level overrides work
- [ ] Destructive commands can be confirmed instead of blocked
- [ ] All existing tests still pass
- [ ] New tests for classifier integration

**Dependency:** Item 1 (bash classification engine)
**Blocked by:** Item 1

---

## Sprint Summary

| Sprint | Items | Effort | Key Deliverable |
|--------|-------|--------|-----------------|
| 1A | 1-4 | ~8 days | Shared patterns library + CI integration |
| 1B | 5-9 | ~15 days | Pi extension enforcement (injection, output, permissions) |

**Total effort:** ~23 person-days across both sprints

**Sprint 1A dependency:** None (can start immediately)
**Sprint 1B dependency:** Items 1-2 needed for Item 9 only; Items 5-8 are independent

**Recommended sprint order:**
1. **Sprint 1A** (template repo): Items 1, 2, 3, 4
2. **Sprint 1B** (pi extension): Items 5, 6, 7, 8, 9 (Items 5-8 parallelizable, Item 9 after Item 1)

---

## What's NOT in this Sprint

These items from the gap analysis are deferred:

| Item | Reason Deferred |
|------|----------------|
| Sandbox / process isolation (GAP-3) | High architectural effort, defer to Sprint 2+ |
| Canary token / leak detection (GAP-9) | Niche, defer to Sprint 2+ |
| Content filtering / topic control (GAP-8) | Low priority for coding agents |

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|:---------:|:------:|------------|
| Prompt injection patterns evolve fast | High | High | Use community-maintained pattern list (OWASP) |
| Tool permission system conflicts with pi session management | Medium | Medium | Test with pi-subagents early |
| CI Action performance on large PRs | Low | Medium | Only check changed files, not entire repo |
| Team policy hierarchy causes confusion | Medium | Low | Clear error messages showing which policy applied |
| Bash classifier misclassifies a command | Medium | High | Conservative defaults, allowlist override |

---

## Definition of Done

- [ ] All acceptance criteria met
- [ ] Tests passing (unit + integration)
- [ ] Documentation updated (skill files, README)
- [ ] Gap analysis updated to reflect completed items
- [ ] No regressions in existing guardrails functionality
- [ ] Code reviewed and merged
