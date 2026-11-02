> **Re-anchored 2026-10-02:** `aiggp-00`/`aiggp-09` are retired (see `openspec/changes/AIGGP-RETIREMENT-2026-10-02.md`); references below read against DevGate's shipped evidence machinery (`hub/coherence/`) and runner enrollment (`scripts/runner-enroll.sh`).

- [x] Phase 0 (category packages): define the four core categories with corpora, actions, failure modes, budgets.
      **EXERCISED:** S1–S15 safety taxonomy (including coding-specific S14 Code Abuse and S15 Data Exfiltration), per-category thresholds, and mediation actions (`allow`, `warn`, `redact`, `block`) implemented in `mcp-server/internal/guardrails/content_filter.go`, `content_filter_policy.go`, and `content_filter_config.go`; verified by `content_filter_taxonomy_test.go` and `content_filter_policy_test.go`.
- [x] Phase 1 (pipeline): registry, adapters, mediation wiring, redaction engine, envelope emission.
      **EXERCISED:** `ContentFilter` pipeline, Llama Guard (`OllamaBackend`) and OpenAI Moderation backends (`content_filter_backends.go`), 60s result cache (`content_filter_cache.go`), fail-closed default when no backend is configured, and MCP handlers `guardrail_classify_content` / `guardrail_check_policy` (`mcp-server/internal/mcp/tools_guardrails.go`); verified by `content_filter_test.go` and `tools_guardrails_test.go`.
- [ ] Phase 2 (conformance): corpus runners, budget gates, stale-claim detection.
      **PARTIAL (WIRED / ABSENT):** Startup config validation (`CONTENT_FILTER_CONFIG_FILE`) and Spec 07 policy-pack digest binding (`mcp-server/internal/guardrails/policy_pack.go`) detect config/pack drift, while labelled FP/FN evaluation corpus runners and automatic classifier-bump invalidation remain open (`docs/specs/guardrail-gaps-2026/STATUS.md` §02).
- [ ] Phase 3 (drift monitor): scheduled re-runs, alerting, ledger-visible drift events.
      **OPEN (ABSENT):** Scheduled classifier drift re-runs and ledger-visible drift events are not yet implemented.
- [x] Phase 4 (hardening): performance budgets, escalation UX review, red-team content suite.
      **EXERCISED:** Strict YAML config loading (`LoadContentConfig`), taxonomy allowlist enforcement (`enabled_categories` / `custom_categories`), and fail-closed refusal of `fail_policy: allow` implemented in `content_filter_config.go`; verified by `config_file_wiring_test.go` and `content_filter_test.go`.
