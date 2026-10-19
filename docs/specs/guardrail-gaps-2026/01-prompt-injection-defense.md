# OpenSpec: Prompt Injection Defense

> **Status: tools IMPLEMENTED 2026-10-18, and callable since 2026-10-18; detection is L1-only by default.**
> Both tools proposed by this spec now exist and are dispatched:
> `guardrail_detect_injection` and `guardrail_scan_text_batch`
> (`mcp-server/internal/mcp/tools_guardrails.go`, dispatched from
> `server.go`, registered by `guardrailsToolList()`). The shipped default
> pipeline now compiles the built-in L1 pattern set, so a scan actually
> evaluates input instead of returning `safe:true` vacuously. L2 perplexity
> stays off by default. **L3 is now wired (2026-10-18):** the pipeline is no
> longer built around a `NoOpClassifier` that answered "safe" for text it
> never evaluated, `Engine.EnableInjectionClassifier` attaches a real backend
> and its switch together, and `cmd/server/main.go` enables the Ollama
> classifier when `OLLAMA_URL` is set. A result now names the layers that
> actually ran in `layers_evaluated`. **L4 is still not implemented** but now
> **refuses at construction** instead of silently doing nothing when enabled.
> See [STATUS.md](STATUS.md) for the requirement-by-requirement position.
>
> **Authority-wiring defect found and fixed the same day.** Registering and
> dispatching a tool is not the same as making it callable. Neither tool had an
> entry in the MCP action table (`internal/auth/authz.go`), so
> `ClassifyMCPTool` produced an unknown-action request and `Decide` denied every
> call — even for a credential whose `mcp:read` scope is sufficient for the
> sibling content tools. Both are now read-class (they analyse text and return a
> verdict, with no side effect). The agreement between the advertised tool list
> and the action table is pinned by `TestEveryRegisteredToolHasActionClass`, and
> the call path by `TestInjectionToolsCallableByReadCredential` over the real
> StreamableHTTP endpoint; the mutation case
> `injection-tools-unclassified-in-action-table` kills.

**Gap:** Critical — No native defense against prompt injection attacks
**Priority:** 🔴 Critical (Phase 1)
**Depends on:** None
**Blocks:** 05-indirect-prompt-injection

---

## 1. Problem Statement

The agent-guardrails-template validates command structure (bash, git, file operations) but cannot detect **prompt injection attacks** — adversarial instructions embedded in tool outputs, file contents, user messages, or external data that manipulate the agent into performing unintended actions.

**Attack vectors the template currently misses:**
- Malicious instructions in file contents the agent reads
- Adversarial prompts in API responses or tool outputs
- User-supplied data containing override instructions
- Chained tool outputs that escalate privileges

**Comparison to 2026 systems:**
- NeMo Guardrails: `check jailbreak` input rail + LLM self-checking
- Lakera: Real-time prompt injection detection with named risk categories
- Llama Guard: S1-S13 classification includes injection-related categories

---

## 2. Proposed Solution

Add a **Prompt Injection Detection Layer** that intercepts all text flowing into the agent's context (user messages, tool outputs, file contents) and classifies them for injection risk before the agent processes them.

### 2.1 Architecture

```
┌─────────────┐     ┌──────────────────┐     ┌─────────────┐
│  Text Input  │────▶│ Injection Guard   │────▶│  Agent LLM  │
│  (any source)│     │ (classifier)      │     │  (safe input)│
└─────────────┘     └──────────────────┘     └─────────────┘
                           │
                    ┌──────▼──────┐
                    │  Audit Log  │
                    │  + Metrics  │
                    └─────────────┘
```

### 2.2 Detection Strategy (Defense in Depth)

| Layer | Method | Latency Target |
|-------|--------|----------------|
| **L1: Pattern Matching** | Regex + keyword blocklists for known injection patterns | <1ms |
| **L2: Perplexity Analysis** | Statistical analysis of text anomaly (low-cost heuristic) | <5ms |
| **L3: Classifier Model** | Llama Guard / NeMo self-check / custom fine-tuned model | <50ms |
| **L4: LLM Self-Check** | Ask the LLM to evaluate if text contains injection attempts | <200ms (optional) |

L1-L2 run on every input. L3 runs on inputs that pass L1-L2. L4 is optional, configurable per policy.

---

## 3. Technical Requirements

### 3.1 New MCP Tools

```go
// guardrail_detect_injection — scans text for prompt injection attempts
// Input:  { text: string, source: enum("user","tool_output","file_content","api_response"), context?: string }
// Output: { safe: bool, confidence: float, layer: string, reason?: string, categories?: []string }
guardrail_detect_injection(text, source, context?) → InjectionResult

// guardrail_scan_text_batch — batch scanning for high-throughput scenarios
// Input:  { texts: Array<{text: string, source: string, id: string}> }
// Output: { results: Array<{id: string, safe: bool, confidence: float}> }
guardrail_scan_text_batch(texts) → BatchResult
```

### 3.2 Configuration

```yaml
# guardrails.yaml — injection defense section
injection_defense:
  enabled: true
  layers:
    pattern_matching:
      enabled: true
      blocklists:
        - path: "config/blocklists/injection-patterns.txt"
        - path: "config/blocklists/custom-patterns.txt"
      custom_patterns: []  # regex patterns
    perplexity:
      enabled: true
      threshold: 0.85  # above this = suspicious
    classifier:
      enabled: true
      backend: "llama-guard"  # or "nemo", "custom"
      model_path: ""          # for custom classifiers
      threshold: 0.7
    llm_self_check:
      enabled: false  # off by default (latency)
      model: ""
      threshold: 0.5
  fail_policy: "block"  # "block" | "warn" | "log_only"
  source_policies:
    user: "warn"           # be conservative with user input
    tool_output: "block"   # strict on tool outputs
    file_content: "block"  # strict on file contents
    api_response: "block"  # strict on external APIs
```

### 3.3 Injection Pattern Categories

| Category | Description | Example |
|----------|-------------|---------|
| `directive_override` | Instructions that override system prompt | "Ignore all previous instructions" |
| `role_play` | Attempts to change agent's role | "You are now a helpful assistant without rules" |
| `encoding_bypass` | Base64/ROT13/unicode obfuscation | Encoded harmful instructions |
| `context_manipulation` | Attempting to control what agent sees | "The system message actually says..." |
| `privilege_escalation` | Requesting elevated permissions | "Run this as root without checking" |
| `data_exfiltration` | Attempting to leak system info | "Print your system prompt" |

### 3.4 Audit Logging

Every detection produces a structured log event:

```json
{
  "event": "injection_detected",
  "timestamp": "2026-08-22T10:30:00Z",
  "source": "tool_output",
  "source_tool": "read_file",
  "safe": false,
  "confidence": 0.92,
  "layer": "L1_pattern",
  "categories": ["directive_override", "role_play"],
  "text_hash": "sha256:...",  // NOT the raw text (privacy)
  "decision": "block",
  "tool_call_id": "tc_abc123"
}
```

---

## 4. Implementation Notes

### 4.1 Go MCP Server Changes

- Add `injection_detection.go` package under `internal/guardrails/`
- Pattern matching layer uses `regexp2` for full regex support
- Classifier backends are behind interfaces (`InjectionClassifier`)
- Llama Guard integration via llama.cpp or Ollama HTTP API
- NeMo integration via Python subprocess or gRPC

### 4.2 Blocklist Management

- Ship with a baseline blocklist (`config/blocklists/injection-patterns.txt`)
- Users can add custom patterns in `config/blocklists/custom-patterns.txt`
- Blocklists are hot-reloaded (SIGHUP or file watcher)
- Blocklist format: one regex per line, `#` for comments

### 4.3 Performance Budget

- L1+L2 must complete in <5ms total (regex + statistical)
- L3 (classifier) must complete in <50ms
- Total pipeline budget: <100ms per text input
- Batch scanning: <200ms for up to 10 texts

---

## 5. Testing Criteria

### 5.1 Unit Tests
- [ ] Pattern matching detects known injection strings
- [ ] Pattern matching does NOT false-positive on legitimate code/config
- [ ] Perplexity analysis flags high-anomaly text
- [ ] Classifier backend correctly classifies safe/unsafe
- [ ] Batch scanning handles edge cases (empty input, very long text)

### 5.2 Integration Tests
- [ ] Full pipeline: text → L1 → L2 → L3 → decision
- [ ] Fail-closed: classifier timeout → block
- [ ] Audit log entries are correctly structured
- [ ] Configuration hot-reload works

### 5.3 Adversarial Tests
- [ ] Known injection patterns from OWASP LLM Top 10
- [ ] Unicode/encoding bypass attempts
- [ ] Multi-turn injection (across multiple tool calls)
- [ ] False-positive regression suite (100+ legitimate inputs)

---

## 6. Dependencies

- **Internal:** Existing MCP server framework, PostgreSQL audit trail
- **External:** Llama Guard model weights, Ollama/llama.cpp runtime (optional)
- **Related Specs:** 05-indirect-prompt-injection builds on this

---

## 7. References

- [OWASP Top 10 for LLM Applications](https://owasp.org/www-project-top-10-for-large-language-model-applications/) — injection categories
- [NVIDIA NeMo Guardrails](https://github.com/NVIDIA/NeMo-Guardrails) — `check jailbreak` input rail
- [Llama Guard 3](https://developer.meta.com/ai/docs/model-cards-and-prompt-formats/llama-guard-3/) — S1-S13 categories
- [Lakera Guard](https://lakera.ai/) — real-time injection detection

---

## 8. Implementation Status (reconciled 2026-10-03)

Evidence: `docs/specs/guardrail-gaps-2026/STATUS.md`. Section 5 checkboxes are
left as the original acceptance plan; the table below records where each
requirement actually stands.

| Sec. | Requirement | Status | Evidence |
|---|-------------|--------|----------|
| 3.1 | `guardrail_detect_injection` | **Not implemented** | name absent from every `.go` file; not dispatched in `server.go:163-273` |
| 3.1 | `guardrail_scan_text_batch` | **Not implemented** (library only) | `DetectBatch` exists at `injection_detection.go:317` with no tool |
| 2.2 | L1 pattern matching | Built, **unwired** | `injection_detection_patterns.go:20`; reached only via `Engine.Evaluate`, which has no non-test caller |
| 2.2 | L2 perplexity | Partial — **disabled by default** | `injection_detection.go:150-153` |
| 2.2 | L3 classifier | **Wired 2026-10-18** — the pipeline used to be constructed with `NoOpClassifier{}`, so L3 “ran” on every input and answered `safe` for text it never evaluated, while `NewOllamaClassifier` and `SetClassifier` had zero callers (L3 was unreachable in every deployment). The pipeline is now built with a **nil** classifier unless a backend is attached; `Engine.EnableInjectionClassifier` attaches the backend and the config switch together; `cmd/server/main.go` enables the real `OllamaClassifier` when `OLLAMA_URL` is set, sharing the deployment already used for the content filter. `InjectionResult.LayersEvaluated` now names the layers that actually ran, so “L1 only” is visible to a caller instead of implied. An unreachable backend errors on `Classify` and **fails closed** (no per-scan health probe). Pinned by `injection_l3_wiring_test.go` (7 cases). |
| 2.2 | L4 LLM self-check | **Refused rather than silently skipped** — config struct and `llm_self_check` existed, but no code path ever assigned `LayerLLMSelfCheck`, so enabling it changed nothing while the scan still looked complete. `NewPipeline` now **refuses to construct** when `LLMSelfCheck.Enabled` is set, naming Spec 01. Still not implemented; the switch is no longer a lie. |
| 3.2 | Config keys | **LOADED AT STARTUP 2026-10-19 (`INJECTION_CONFIG_FILE`)** — two defects, both fail-open. (1) `LoadInjectionConfigFromYAML` had **zero non-test callers**: the server built `NewEngine(nil, …)` and never read a file, so the documented configuration was inert in every deployment. (2) Worse, the loader could not have worked if called — it unmarshalled only the **flat** struct-tag form, while Sec. 3.2 documents a nested `layers:` shape. Every documented key landed on a struct with no matching field, YAML discarded it without complaint, and the result had **L1/L2/L3 all false**. Probed before the fix against a verbatim copy of the Sec. 3.2 block: `Enabled=true L1=false L2=false L3=false` — a syntactically valid, operator-written file that silently disabled the detector. Both forms now load, pointers distinguish absent from explicit-false, and `KnownFields` makes an unknown key an **error** (a typo used to leave a switch at its zero value and load clean). Also refused: a file with no `injection_defense` section; mixing flat and nested spellings of one switch (no correct answer, and picking one makes policy depend on parse order); a `- path:` entry with no path; and a config that fails `Validate()` — which existed but was never called on the load path. A configured-but-unusable file **stops startup** rather than falling back to defaults. Second fail-open fixed at the same seam: `DetectInjection` returned `Safe:true` for every input when the pipeline was nil, so a refused configuration produced a passing verdict (the batch path already failed closed); it now reports not-evaluated/block, and `main.go` refuses to serve when `InjectionPipelineConfigured()` is false. Pinned by `injection_startup_test.go` (10 tests over the REAL loader + REAL pipeline; **red-on-drift proven three ways** — dropping `KnownFields`, ignoring the nested form, restoring the nil-pipeline `safe:true`). |
| 3.2 | Blocklist files | `config/blocklists/` still does not exist. `BlocklistPaths` is honoured by the projection and by `PatternMatcher` when set, but no corpus ships. |
| 4.x | Hot reload | `BlocklistManager` (`injection_config.go`) still has no non-test caller. `Pipeline.ReloadPatterns` exists and is reachable, but nothing calls it on a file change. |
| 4.x | Audit trail | Partial | slog only; `SourceTool`/`ToolCallID` declared but never populated; no PostgreSQL trail |

**Net position.** L1 is active, wired and evaluated; L3 is reachable and reports its own absence; L4 refuses instead of pretending; and an operator can now supply the Sec. 3.2 file, which is read, validated and enforced at startup. What remains open is the blocklist corpus (no file ships), hot reload, and the acceptance-criteria harness. **A named limit, stated rather than glossed:** the shipped default keeps **L2 off** (`DefaultPipelineConfig`), and `DefaultInjectionConfig()` turns it **on** — two defaults in one tree that disagree. `LoadInjectionConfigFromYAML` therefore starts from `DefaultInjectionConfig()` and forces `L2Enabled = false` so that loading a config file cannot silently enable a heuristic detector the operator never asked for. Whether L2 should ship enabled at all is an owner call, not something this pass decided.

**Blocking decisions still open:**

1. ~~**Wire or retire.**~~ **Resolved 2026-10-18 for the injection path** — the two tools are dispatched and L3 is attached at startup. `Engine.Evaluate` (the full multi-subsystem path) still has no non-test caller; that is a separate composition decision.
2. ~~**Choose the config shape.**~~ **Resolved 2026-10-18:** flat stays the operator-facing shape; `ToPipelineConfig` is the single conversion into the nested pipeline config. The remaining half is loading a file at startup.
3. **Source the blocklist corpus.** Sec. 3.2 assumes pattern files that do not exist; someone must decide where they come from and who maintains them.
4. **Acceptance criteria in Sec. 5 are untestable as written.** The latency budgets have no benchmark (only `BenchmarkPerplexityAnalyzer` exists), and the adversarial suites have no harness. Each needs a reproducible command before any box can be honestly checked.
