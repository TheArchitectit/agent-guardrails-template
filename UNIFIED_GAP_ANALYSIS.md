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

### 16 Pi Code Modules (runtime enforcement)

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

### GAP-1: Language-Specific Guardrails (No Pi Code or Skill)

The MCP server has 4 language tools (`detect_language`, `get_language_profile`, `list_languages`, `validate_language_rules`) for language-specific coding guardrails. Pi has **zero** language-specific enforcement.

**Impact**: HIGH — language-specific patterns (Python type hints, Rust ownership, Go error handling) are not enforced in pi.

**Recommendation**: Create a `pi-extension/languages/` module with LanguageProfileLoader and language-specific rule sets. Add a `language-guardrails` pi skill.

### GAP-2: Pre-Work Checklist (No Pi Code or Skill)

The MCP server has `guardrail_pre_work_check` that loads the failure registry and generates a pre-work checklist. Pi has no equivalent.

**Impact**: MEDIUM — agents don't check for past failures before starting new work.

**Recommendation**: Add a `PreWorkChecker` module that reads the violation log on session start and surfaces relevant past failures. Add a `pre-work-check` pi skill or extend guardrails-core.

### GAP-3: Feature Creep Detection (No Pi Code or Skill)

The MCP server has `guardrail_detect_feature_creep` that compares git diff against authorized scope. Pi has scope enforcement but no diff-based detection.

**Impact**: LOW — scope enforcement catches out-of-scope edits, but can't detect subtle feature creep within in-scope files.

**Recommendation**: Add a `FeatureCreepDetector` that compares planned vs actual changes in scope. Add to `scope-validator` skill as a pi-specific section.

### GAP-4: Regression Prevention (No Pi Code or Skill)

The MCP server has `guardrail_prevent_regression` (failure registry matching) and `guardrail_verify_fixes_intact` (verify past fixes haven't regressed). Pi has neither.

**Impact**: MEDIUM — past violations and fixes are not tracked across sessions for regression prevention.

**Recommendation**: Extend ViolationLog to support cross-session failure registry. Add a `regression-guard` pi skill.

### GAP-5: Exact Replacement Validation (No Pi Code or Skill)

The MCP server has `guardrail_validate_exact_replacement` that validates code replacements match spec. Pi has no equivalent.

**Impact**: LOW — edits are tracked but not validated against specifications.

**Recommendation**: Add an `ExactReplacementValidator` that checks edit operations against expected old content. Add to `guardrails-core` as an edit validation step.

### GAP-6: Git Operations Validation (Partial Pi Coverage)

The MCP server has `guardrail_validate_git_operation` and `guardrail_validate_push` for Git-specific guardrails. Pi has bash safety (catches force-push) but no git-domain-aware validation.

**Impact**: MEDIUM — bash safety catches the most dangerous git commands, but doesn't validate branch names, commit message format, or merge strategies.

**Recommendation**: Add a `GitOperationValidator` with branch protection, commit format, and merge strategy rules. Add a `git-safety` pi skill.

### GAP-7: Halt Event Lifecycle (Partial Pi Coverage)

The MCP server has a full halt lifecycle (`check_halt_conditions`, `record_halt`, `acknowledge_halt`). Pi has `check_halt` and `record_violation` but no explicit halt tracking or acknowledgment flow.

**Impact**: LOW — the current halt + violation flow covers most use cases, but lacks formal halt acknowledgment semantics.

**Recommendation**: Add `halt` and `acknowledge_halt` states to SessionStore. Extend `guardrails-core` skill.

### GAP-8: Document/Standard Access (No Pi Code or Skill)

The MCP server has `guardrail_get_standard`, `guardrail_get_workflow`, `guardrail_search_docs` for accessing guardrail documentation. Pi has no doc retrieval tools.

**Impact**: LOW — agents can read files directly, but structured doc access facilitates better guardrail compliance.

**Recommendation**: Add a `GuardrailDocs` module that indexes and searches `docs/`, `.guardrails/`, and `skills/`. Add to MCP bridge only (not worth a separate pi module).

### GAP-9: Uncertainty Scoring (Partial Pi Coverage)

The MCP server has `guardrail_check_uncertainty` with a structured uncertainty scale (certain/probably/uncertain/guessing). Pi has `check_halt` which returns boolean decisions but no calibrated uncertainty score.

**Impact**: LOW — the current system halts appropriately, but lacks the nuance of calibrated uncertainty.

**Recommendation**: Add an uncertainty score (0-1) to HaltResult. Extend `halt-conditions` skill to mention the pi uncertainty scoring.

### GAP-10: Pattern Rules Engine (No Pi Code or Skill)

The MCP server has `guardrail_get_prevention_rules` and `guardrail_check_pattern` for loading `.guardrails/prevention-rules/` and checking code against them. Pi has no pattern rule engine.

**Impact**: MEDIUM — `.guardrails/prevention-rules/pattern-rules.json` contains language-specific prevention patterns that are not enforced in pi.

**Recommendation**: Add a `PatternRuleEngine` that loads prevention rules from `.guardrails/prevention-rules/` and checks code against them. Add a `pattern-rules` pi skill.

---

## Summary: Priority-Ordered Gap List

| Priority | Gap | Effort | Impact |
|----------|-----|--------|--------|
| P1 | GAP-1: Language-specific guardrails | High | HIGH |
| P2 | GAP-10: Pattern rules engine | Medium | MEDIUM |
| P3 | GAP-2: Pre-work checklist | Low | MEDIUM |
| P4 | GAP-4: Regression prevention | Medium | MEDIUM |
| P5 | GAP-6: Git operations validation | Medium | MEDIUM |
| P6 | GAP-3: Feature creep detection | Low | LOW |
| P7 | GAP-5: Exact replacement validation | Low | LOW |
| P8 | GAP-7: Halt event lifecycle | Low | LOW |
| P9 | GAP-9: Uncertainty scoring | Low | LOW |
| P10 | GAP-8: Document/standard access | Low | LOW |

## What's Now Covered (After Unification)

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
