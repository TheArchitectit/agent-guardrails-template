# Guardrails Gap Analysis — Unified View

Date: 2026-05-17

## Inventory

### 11 Canonical Skills (behavioral teachings, cross-platform)

| # | Skill ID | Tags | Pi Enforcement Added? |
|---|----------|------|----------------------|
| 1 | four-laws | safety, core, mandatory | Yes |
| 2 | guardrails-enforcer | safety, core, mandatory | Yes |
| 3 | halt-conditions | safety, core, mandatory | Yes |
| 4 | scope-validator | safety, core, mandatory | Yes |
| 5 | env-separator | safety, core, mandatory | Yes |
| 6 | three-strikes | safety, core, mandatory | Yes |
| 7 | production-first | safety, core, mandatory | Yes |
| 8 | error-recovery | safety, core, mandatory | Yes |
| 9 | commit-validator | safety, workflow | Yes |
| 10 | vibe-coding | workflow, productivity | Yes |
| 11 | 3d-game-dev | game-dev, 3d, safety | Yes |

### 9 Pi Skills (teach agents how to use pi guardrail tools)

| # | Skill ID | Maps To Code Module | Auto-Enforced? |
|---|----------|---------------------|----------------|
| 1 | guardrails-core | All (index) | Yes (7 handlers) |
| 2 | guardrails-dashboard | tui/guardrails-panel.ts | N/A |
| 3 | injection-defense | injection/detector.ts | Yes (tool_result) |
| 4 | output-security | output-validator/validator.ts | Yes (tool_result) |
| 5 | content-safety | output-validator/content-filter.ts | Yes (tool_result) |
| 6 | tool-permissions | permissions/permissions.ts | Yes (tool_call) |
| 7 | policy-config | policy/policy-loader.ts | Yes (config load) |
| 8 | sandbox-isolation | sandbox/sandbox-runner.ts | No (on-demand) |
| 9 | canary-tokens | injection/canary.ts | Yes (output scan) |

### 22 Pi Code Modules (runtime enforcement)

| Module | Handler Type | Event | Law |
|--------|-------------|-------|-----|
| SessionStore | state management | init | All |
| FileReadStore | tool_result handler | tool_result | Law 1 |
| ScopeValidator | tool_call handler | tool_call | Law 2 |
| StrikeCounter | tool calls | manual | Law 4 |
| HaltChecker + bash-classify | tool_call handler | tool_call | Law 4 |
| ViolationLog | tool_call handler | all | All |
| InjectionDetector | tool_call handler | tool_call | Law 4 |
| OutputValidator | tool_result handler | tool_result | Law 3 |
| ContentFilter | tool_result handler | tool_result | Law 3 |
| CanaryTokenManager | tool_result handler | tool_result | Law 3 |
| PermissionManager | tool_call handler | tool_call | All |
| PolicyLoader | config load | init | All |
| PreWorkChecker | tool | on-demand | Law 4 |
| FeatureCreepDetector | tool | on-demand | Law 2 |
| PatternRuleEngine | tool | on-demand | Law 3 |
| GitValidator | tool | on-demand | Law 4 |
| LanguageDetector | tool | on-demand | Law 3 |
| RegressionGuard | tool | on-demand | Law 3/4 |
| ExactReplacementValidator | tool | on-demand | Law 1 |

### 52 Go MCP Server Tools

| Category | Tools | Pi Bridge Coverage |
|----------|-------|-------------------|
| Session | init_session | Yes (guardrail_mcp) |
| Bash | validate_bash | Yes (auto-enforced) |
| File Edit | validate_file_edit, verify_file_read, record_file_read | Yes (auto-enforced + tools) |
| Git | validate_git_operation, validate_push, validate_commit | Partial (bash safety catches push) |
| Scope | validate_scope | Yes (auto-enforced + tools) |
| Game Dev | validate_game_build | Via guardrail_mcp only |
| Stikes | record_attempt, validate_three_strikes, reset_attempts | Yes (tools) |
| Halt | check_halt_conditions, record_halt, acknowledge_halt, check_uncertainty | Partial (check_halt tool only) |
| Commit | validate_commit, validate_exact_replacement | Via guardrail_mcp only |
| Production | validate_production_first, detect_feature_creep | Via guardrail_mcp only |
| Regression | prevent_regression, verify_fixes_intact | Via guardrail_mcp only |
| Environment | check_test_prod_separation | Via guardrail_mcp only |
| Language | detect_language, get_language_profile, list_languages, validate_language_rules | No pi equivalent |
| Docs | get_standard, get_workflow, search_docs | No pi equivalent |
| Rules | get_prevention_rules, check_pattern | No pi equivalent |
| Pre-work | pre_work_check | No pi equivalent |
| Context | get_context | No pi equivalent |
| Skills | install_skills | No pi equivalent |
| Marketplace | marketplace_add/list/search/remove | No pi equivalent |
| Teams | team_init/list/assign/unassign/start/status/etc. | No pi equivalent (also not implemented in Go) |

---

## Gap Analysis

### GAP-1: Language-Specific Guardrails — CLOSED

**Implemented**: `LanguageDetector` module + 4 language rule files (Python 8 rules, TypeScript 7, Go 6, Rust 6). `guardrail_detect_language` and `guardrail_get_language_profile` tools. Language rules auto-loaded by `PatternRuleEngine`. `language-detection` canonical skill.

### GAP-2: Pre-Work Checklist — CLOSED

**Implemented**: `PreWorkChecker` reads ViolationLog + SessionStore. `guardrail_pre_work_check` tool.

### GAP-3: Feature Creep Detection — CLOSED

**Implemented**: `FeatureCreepDetector` compares modified files against scope paths. `guardrail_detect_creep` tool.

### GAP-4: Regression Prevention — CLOSED

**Implemented**: `RegressionGuard` with cross-session failure registry. `guardrail_check_regression`, `guardrail_verify_fixes`, `guardrail_register_failure` tools. Seeds from ViolationLog critical violations.

### GAP-5: Exact Replacement Validation — CLOSED

**Implemented**: `ExactReplacementValidator` validates edit old_content against actual file. `guardrail_validate_replacement` tool.

### GAP-6: Git Operations Validation — CLOSED

**Implemented**: `GitValidator` with branch protection, commit format, force-push detection. `guardrail_validate_git` tool.

### GAP-7: Halt Event Lifecycle — CLOSED

**Implemented**: `HaltState` in SessionStore (active/halted/acknowledged). Handlers record halts on block. `guardrail_acknowledge_halt` tool.

### GAP-8: Document/Standard Access — CLOSED

**Implemented**: `guardrail_read_skill`, `guardrail_list_skills`, `guardrail_list_languages` tools. `docs-access` pi skill.

### GAP-9: Uncertainty Scoring — CLOSED

**Implemented**: `uncertaintyScore` (0-1) added to HaltResult. Scales: certain (0-0.2), probably (0.2-0.5), uncertain (0.5-0.8), guessing (0.8-1.0).

### GAP-10: Pattern Rules Engine — CLOSED

**Implemented**: `PatternRuleEngine` loads from `.guardrails/prevention-rules/pattern-rules.json`. `guardrail_check_pattern` tool. Auto-loads language rules via LanguageDetector.

---

## Summary: All Gaps Closed

| Priority | Gap | Status |
|----------|-----|--------|
| P1 | GAP-1: Language-specific guardrails | CLOSED — LanguageDetector + 4 language rule files |
| P2 | GAP-10: Pattern rules engine | CLOSED — PatternRuleEngine |
| P3 | GAP-2: Pre-work checklist | CLOSED — PreWorkChecker |
| P4 | GAP-4: Regression prevention | CLOSED — RegressionGuard |
| P5 | GAP-6: Git operations validation | CLOSED — GitValidator |
| P6 | GAP-3: Feature creep detection | CLOSED — FeatureCreepDetector |
| P7 | GAP-5: Exact replacement validation | CLOSED — ExactReplacementValidator |
| P8 | GAP-7: Halt event lifecycle | CLOSED — HaltState in SessionStore |
| P9 | GAP-9: Uncertainty scoring | CLOSED — uncertaintyScore in HaltResult |
| P10 | GAP-8: Document/standard access | CLOSED — read_skill/list_skills/list_languages tools |

## What's Now Covered (After Unification + All Gaps Closed)

| Guardrail Domain | Canonical Skill | Pi Skill | Code Module | Auto-Enforced |
|------------------|----------------|----------|-------------|---------------|
| Four Laws | four-laws | guardrails-core | All | Yes |
| Enforcement agent | guardrails-enforcer | guardrails-core | All | Yes |
| Halting | halt-conditions | guardrails-core | HaltChecker | Yes |
| Scope | scope-validator | guardrails-core | ScopeValidator | Yes |
| Environment separation | env-separator | output-security, canary-tokens | OutputValidator, CanaryTokenManager | Yes |
| Three Strikes | three-strikes | guardrails-core | StrikeCounter | Manual |
| Production-first | production-first | (via MCP bridge) | (none) | No |
| Error recovery | error-recovery | (via MCP bridge + sandbox) | SandboxRunner | On-demand |
| Commit validation | commit-validator | (via MCP bridge) | ViolationLog + bash safety | Partial |
| Vibe coding | vibe-coding | (via MCP bridge) | (none) | No |
| 3D Game dev | 3d-game-dev | (via MCP bridge) | (none) | No |
| Injection defense | (no canonical) | injection-defense | InjectionDetector | Yes |
| Output security | (no canonical) | output-security | OutputValidator | Yes |
| Content filtering | (no canonical) | content-safety | ContentFilter | Yes |
| Tool permissions | (no canonical) | tool-permissions | PermissionManager | Yes |
| Policy hierarchy | (no canonical) | policy-config | PolicyLoader | Yes |
| Sandbox | (no canonical) | sandbox-isolation | SandboxRunner | On-demand |
| Canary tokens | (no canonical) | canary-tokens | CanaryTokenManager | Yes |
| Dashboard | (no canonical) | guardrails-dashboard | GuardrailsPanel | N/A |
| Language detection | language-detection | (via guardrails-core) | LanguageDetector | Auto (on load) |
| Regression prevention | (no canonical) | (via guardrails-core) | RegressionGuard | On-demand |
| Replacement validation | (no canonical) | (via guardrails-core) | ExactReplacementValidator | On-demand |
| Halt lifecycle | (no canonical) | (via guardrails-core) | SessionStore HaltState | Auto (on block) |
| Uncertainty scoring | (no canonical) | (via guardrails-core) | HaltChecker | Yes (in check_halt) |
| Docs access | (no canonical) | docs-access | (inline tools) | No |
