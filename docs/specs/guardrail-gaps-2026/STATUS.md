# Guardrail Gaps 2026 — Spec vs Shipped Code

**Status: reconciliation, 2026-10-03; wave-0 release-gating fixes reconciled
2026-10-09; Spec 01 injection tools implemented 2026-10-18.** This page
supersedes the `Status: Draft` labels on the six specs.

The six specs in this directory are **proposals written before the reboot**,
not descriptions of the system. Since then a Go rewrite implemented much of
the underlying library code. This page records, requirement by requirement,
what actually exists and what can actually be called.

## The headline finding

**Of the 15 MCP tools these specs propose or name, exactly 4 exist.** Verified
by grepping every `.go` file in the repository. As of 2026-10-18 that count
includes the two Spec 01 injection tools, which were added in this pass — the
other 11 names still appear nowhere outside the specs themselves.

| Proposed tool | Spec | Exists? |
|---------------|------|---------|
| `guardrail_detect_injection` | 01 | **Yes (2026-10-18)** |
| `guardrail_scan_text_batch` | 01 | **Yes (2026-10-18)** |
| `guardrail_sandbox_execute` | 03 | **No** |
| `guardrail_sandbox_config` | 03 | **No** |
| `guardrail_validate_agent_output` | 04 | **No** |
| `guardrail_check_agent_constraints` | 04 | **No** |
| `guardrail_resolve_conflicts` | 04 | **No** |
| `guardrail_scan_external_content` | 05 | **No** |
| `guardrail_mark_provenance` | 05 | **No** |
| `guardrail_check_provenance` | 05 | **No** |
| `guardrail_generate_compliance_report` | 06 | **No** |
| `guardrail_check_compliance` | 06 | **No** |
| `guardrail_collect_evidence` | 06 | **No** |
| `guardrail_classify_content` | 02 | **Yes** |
| `guardrail_check_policy` | 02 | **Yes** |

## The second finding: "library exists" ≠ "shipped"

Several subsystems are fully written, well-tested, and **unreachable from
any MCP tool**. As of 2026-10-18 the `internal/mcp` package calls exactly five
methods on the guardrails engine:

```
tools_guardrails.go:136  s.guardrailsEngine.ClassifyContent(...)      // classify_content
tools_guardrails.go:168  s.guardrailsEngine.ClassifyContent(...)      // check_policy, pre-check
tools_guardrails.go:173  s.guardrailsEngine.CheckPolicy(...)          // check_policy
tools_guardrails.go:239  s.guardrailsEngine.DetectInjection(...)      // detect_injection (new)
tools_guardrails.go:325  s.guardrailsEngine.DetectInjectionBatch(...) // scan_text_batch (new)
```

The sandbox manager, the provenance tracker, the multi-agent safety chains and
the compliance mapper remain constructed at startup and never called. The
injection pipeline is no longer in that list: it now has two tools in front of
it, and those tools are what this pass added.

The engine is also no longer conditional. It was previously built only when
`OLLAMA_URL` or `OPENAI_API_KEY` was set (`cmd/server/main.go`), so a
deployment with no LLM had no injection detection at all and both content
tools returned `"guardrails engine not configured"`. It is now constructed
unconditionally; the semantic content-filter backends are still added only
when configured, and with none configured that filter fails closed rather than
reporting content safe.

Honest status wording for the rest: **code complete, not wired to the request
path.**

## Spec-by-spec

### 01 — Prompt injection defense

| Requirement | Status |
|-------------|--------|
| `guardrail_detect_injection` tool | **Implemented 2026-10-18** — registered in `guardrailsToolList()`, dispatched from `server.go`, handler `handleDetectInjection` |
| `guardrail_scan_text_batch` tool | **Implemented 2026-10-18** — `handleScanTextBatch` drives `Engine.DetectInjectionBatch`; bounded to 256 items, rejects duplicate ids / missing text / unknown source rather than dropping an item |
| L1 pattern matching | **Implemented and WIRED** — the shipped `DefaultPipelineConfig` now compiles the built-in `DefaultPatterns()` set; before this pass L1 was "enabled" with zero patterns compiled, so every scan returned `safe:true` without evaluating |
| L2 perplexity | Partial — analyser exists, **disabled by default** |
| L3 classifier backend | **Wired 2026-10-18** — the pipeline is no longer constructed with `NoOpClassifier{}` (which answered "safe" for text it never evaluated). It is built with a **nil** classifier unless a backend is attached; `EnableInjectionClassifier(classifier, cfg)` attaches the backend and its switch together; `cmd/server/main.go` enables the real `OllamaClassifier` when `OLLAMA_URL` is set (no longer dead code). `InjectionResult.layers_evaluated` names the layers that actually ran. An unreachable backend errors on `Classify` and fails closed. Pinned by `injection_l3_wiring_test.go` |
| L4 LLM self-check | **Not implemented, now refused instead of ignored** — `NewPipeline` returns an error when `llm_self_check.enabled` is set, because no code path ever assigned the layer and the scan looked complete anyway |
| Config keys | **Projection implemented 2026-10-18, startup loading still open** — the shipped flat `InjectionConfig` had no consumer at all; new `ToPipelineConfig()` projects L1/L2/L3/L4 switches, patterns, blocklists and source policies onto the nested pipeline config (3 tests drive a real pipeline). `cmd/server/main.go` still builds `NewEngine(nil, …)` and never reads a config file, so a deployment cannot yet supply these by file |
| Blocklist files | `config/blocklists/` does not exist; `BlocklistPaths` is honoured when set but empty by default |
| Authorization (action table) | **Fixed 2026-10-18** — both tools were absent from `mcpToolActions`, so `Decide` denied every call as an unknown action; both are now `ActionRead`. Pinned by `TestEveryRegisteredToolHasActionClass` + `TestInjectionToolsCallableByReadCredential` (real StreamableHTTP call) + mutation case `injection-tools-unclassified-in-action-table` |
| Audit trail | **SourceTool/ToolCallID now populated** — `WithAuditAttribution` carries the calling tool and the caller's provenance label onto the event; the text is recorded as a SHA-256 hash, never plaintext |

The engine is now constructed **unconditionally** in `cmd/server/main.go`. It
was previously built only when `OLLAMA_URL` or `OPENAI_API_KEY` was set, which
meant every deployment without an LLM had no injection detection at all — even
though L1 needs no model. The optional semantic backends are still added only
when configured; with none configured the content filter fails closed.

`Engine.Evaluate`, the full pipeline entry point, is still called only from
tests. The injection tools call `DetectInjection`/`DetectInjectionBatch`
directly rather than routing through `Evaluate`, deliberately: `Evaluate` also
*executes commands* through the SandboxManager when `input.Command` is set, so
routing a text-classification tool through it would start process execution
from the MCP path. That layering decision is unchanged and still open.

**Wiring finding (2026-10-09; injection half RESOLVED 2026-10-18).**
`Engine.Evaluate` — the *full* pipeline entry point — still has **zero
non-test callers**; the other `.Evaluate` references are unrelated
(`PolicyEngine`, `Registry`, and the `bash`/`git`/`fileedit` slice evaluators).
That remains deliberate, not an oversight:
1. **Cross-package integration.** `internal/mcp` never imports
   `internal/domain`; `Engine.Evaluate` also *executes commands* via the
   `SandboxManager` when `input.Command != ""` (real exec), so routing a
   request through it crosses the layering boundary and starts process
   execution from the MCP path.
2. **Fallback semantics.** Only bytes reach the MCP package, so the correct
   `EvalInput.Source` (Provenance trust decision, gap 05) cannot be chosen
   here; and with no `Command` the sandbox layer is skipped, so wiring
   `Evaluate` for a text-only tool would not run sandbox checks anyway.
3. **Product decision.** Whether `guardrail_classify_content` should become a
   *full* pipeline evaluation (injection + provenance + sandbox) or stay a
   content-only classifier is a contract change for an operator. **Still open.**

**What changed:** the injection half did not need `Evaluate`. The engine
exposes `DetectInjection`/`DetectInjectionBatch` — injection-only, no
sandbox, no process execution — and the two Spec 01 tools call those directly.
So the *detector* is now wired to the request path without crossing the
layering boundary. The full-pipeline question in point 3 is unaffected and
still needs an operator decision before `guardrail_classify_content` gains
provenance/sandbox behaviour.

### 02 — Semantic content filtering

The best-executed spec, and the only one whose tools shipped.

| Requirement | Status |
|-------------|--------|
| `guardrail_classify_content` | **Implemented** |
| `guardrail_check_policy` | **Implemented** |
| S1–S15 taxonomy and actions | Implemented |
| Llama Guard backend | Implemented and registered (`main.go:157`, when `OLLAMA_URL` set) |
| 60s result cache | Implemented |
| `fail_policy: block` | Implemented (synthetic block result, not an error) |
| NeMo backend | Not implemented |
| OpenAI Moderation backend | Implemented but never registered |
| Streaming | Not implemented |

**Defect fixed 2026-10-03:** `NewEngine` built the filter as
`NewContentFilter(nil, nil)`, discarding `config.ContentFilter.Policies`. With
no rules loaded, every `guardrail_check_policy` call took the fail-closed
"unknown policy" branch. The configured policy set is now passed to the
filter, so a correctly configured policy resolves; unknown policies still
fail closed by design.

### 03 — Runtime sandbox isolation

| Requirement | Status |
|-------------|--------|
| `guardrail_sandbox_execute` / `guardrail_sandbox_config` tools | Not implemented |
| L0 / L1 (unshare) / L2 (podman→docker) | Implemented, and stronger than spec'd |
| `resource_limits` config | Implemented, field-for-field |
| Network isolation | Implemented — fail-closed, host allowlist via filtering proxy |
| Path-traversal **detection** | Not implemented (only mount-path syntax validation) |
| Privilege-escalation / fork-bomb detection | Not implemented (3 of 5 required violation classes absent) |
| L3 Firecracker | Not implemented (the spec itself marks it optional) |

**Spec amendment, not a gap:** the spec (Sec. 4.3) specifies unconditional
L2→L1→L0 fallback. The implementation deliberately does not: it downgrades
only on genuine setup errors, and treats a command *denied while running
under isolation* as a security breach to be reported. That is the correct
fail-closed behaviour and the spec text is wrong. Sec. 4.3 should be amended.

### 04 — Multi-agent safety policies

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | Not implemented |
| Safety chain engine | Implemented, unwired (`NewSafetyChain` called only from tests) |
| Validators `injection_defense` / `content_filter` / `four_laws_check` | Implemented, unwired |
| `code_review_agent` validator | Not implemented |
| Conflict strategies | Implemented, unwired; named `escalate`, spec says `human_escalate` |
| Config keys (`safety_chains`, `agent_registry`, …) | Not implemented — no loader, zero occurrences in any `.go` file |

**Bug worth noting:** the `four_laws_check` Law-2 scope check
(`multi_agent_validators.go:172`) flags a violation when output *contains* a
declared scope keyword — so merely mentioning a scope word fails the check.
That is not "output exceeds scope" and will misfire on compliant output.

### 05 — Indirect prompt injection

The most completely built of the six, and still unreachable.

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | Not implemented |
| Config keys (`source_trust_policies`, `untrusted_overrides`) | **Implemented**, including the spec's trust table |
| Sanitization / base64 / ROT13 / URL decoding | Implemented, unwired |
| Provenance marker wrapping | Implemented, unwired |
| Redis cache | Deviates — implemented as in-memory (`engine.go:87`) |

The tracker *is* constructed and *is* called from `engine.go:175` with real
trust-based gating — but only from `Engine.Evaluate`, which nothing outside
tests calls.

**`guardrail_record_file_read` / `guardrail_verify_file_read` are not this
spec's provenance tools.** They are session-token-gated file-read
attestation for Law-3 transparency, backed by a database store. They never
touch the provenance tracker, trust levels or content scanning, and do not
satisfy spec 05.

### 06 — Regulatory compliance mapping

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | Not implemented |
| Requirement DB (`compliance_requirements.json`) | Exists, matches the spec's mapping |
| `CalculateComplianceScore` | Implemented, unwired |
| Gap tracking / dashboard | Not implemented |

The data file covers EU AI Act art. 9, 10, 12, 13, 14, 15, 50 and NIST RMF
govern/map/measure/manage. **Art. 11 is absent**, although the spec lists it
as a known gap — so spec Sec. 5.1's own acceptance criterion currently fails.

**Treat these scores as fiction until implemented.** `CheckRequirement`
returns `true` purely because a hand-written string in the JSON says
`"full"`, with the comment *"In a real implementation, this would check if
the features are actually enabled"* (`compliance.go:90-92`). `CollectEvidence`
hardcodes its result and returns `completeness = 1.0` for any non-empty
query (`compliance.go:213`). A score from this code restates the database; it
does not measure the system.

**Simulation made explicit 2026-10-09** (branch `forge/spec-program-wave0`, no
push) — still UNIMPLEMENTED, not fixed by inventing numbers:
- `ComplianceReport.Methodology` is stamped `UnimplementedMethodology =
  "unimplemented"` on every report from `GenerateReport`.
- Declared-`full` evidence from `CheckRequirement` is `Verified:false`,
  `Source: "requirement_db (declared, unverified)"`.
- `CollectEvidence` executes no query; placeholders are `Verified:false` and
  `completeness = verified/total = 0.0` for a non-empty query set.
Exact tests green in that commit: `TestEvidenceCollector_CollectEvidence`,
`TestComplianceReporter_GenerateReport`. `CalculateComplianceScore`'s
arithmetic is unchanged (correct *given* real evidence). This row stays
WIRED/UNIMPLEMENTED — it is not EXERCISED or ACCEPTED.

## Release-gating fixes on this branch (2026-10-09)

Four release-gating gaps were addressed on `forge/spec-program-wave0`. Each is
recorded here with its real status and the fixing commit; none is flipped to
EXERCISED/ACCEPTED without a same-commit green run (per the AUDIT-2026-10-09
status vocabulary).

| Gap | Status | Fixing commit | Present in tree (verified by reading code) |
|-----|--------|---------------|--------------------------------------------|
| 02 content filter failed open when `OLLAMA_URL` unset | **FIXED** | `7f2237b` | `tools_guardrails.go:74-75` returns `safe:false` on unconfigured engine; `:87` returns `safe:false` on classification error; `handleCheckPolicy` fails closed at `:106-107` and `:117` |
| 12-3.3.1 wildcard CORS accepted in production | **FIXED** | `d90cf52` | `config.go:302-306` rejects `*` when `ProductionMode`; reached at startup via `config.Load()`→`Validate()` (`main.go:65`, `config.go:161`) |
| 12-3.6 `X-XSS-Protection` header emitted | **FIXED** | `d90cf52` | `web/server.go:330` no longer emits it (CSP + `nosniff` + `DENY` retained, `:327-329`) |
| 06 compliance scoring restates the DB as a measurement | **UNIMPLEMENTED** (labelled, not fixed) | `beb8c02` | `compliance.go:54,104-107,136,225-250`; see gap 06 above |
| 01/03/04/05 libraries unreachable from the request path | **OPEN** (documented) | `f85aa67` | see gap 01 wiring finding above |

Exact tests re-run green on 2026-10-10 against this tree (HEAD `f85aa67`):
`TestHandleClassifyContent_NoEngine_FailClosed`,
`TestHandleCheckPolicy_NoEngine_FailClosed`,
`TestValidate_ProductionRejectsWildcardCORS`,
`TestSecurityHeaders_NoXSSProtection`,
`TestEvidenceCollector_CollectEvidence`,
`TestComplianceReporter_GenerateReport`.

**Still open and release-gating — do not describe these as fixed:**
- Full-history Gitleaks is RED: 5 real findings, not rotated, history not purged
  (`16-R16-02` / `19-R19-06`). Not weakened, not skipped.
- `/metrics` remains public (`web/server.go:145`; `middleware.go:38,293`) — needs
  the operator (Roger); binding it to auth changes the scrape contract.
- Guardrail capabilities unwired from the request path (`01/03/04/05/06`; `09-P2`).
- Specs `07`/`08`/`15` and the optional OAP checker (`14`) remain ABSENT /
  proposal-only.

**Requirement ledger totals** (recounted 2026-10-10 from the AUDIT-2026-10-09
requirement tables; one status per requirement row): ABSENT 101 · WIRED 75 ·
EXERCISED 59 · NOT_RUN 26 · ACCEPTED 0 · **total 261 requirement rows**.
The AUDIT's earlier `100/78/52/27 = 257` figure is not reproducible from those
tables and is superseded; see
[AUDIT-2026-10-09.md Sec. 5](AUDIT-2026-10-09.md) for the per-section breakdown.

**B7 sync re-verification (2026-10-13, `main` `9fac99b1`):** zero non-test Go
changes since the recount base `f85aa67` — the only deltas are spec-21/22/23
test scaffolding (conformance gate, listing sequence, allowlist test) and
pi-extension TypeScript, both outside this Go requirement ledger's scope.
Totals unchanged: 101/75/59/26/0 = 261.

## Untestable acceptance criteria

Several specs specify acceptance criteria with no oracle in this repository:

- Spec 01: latency budgets — no pipeline benchmark exists.
- Spec 02: FP <5%, FN <1% — no labelled evaluation harness.
- Spec 04 Sec. 5.3: "Agent A instructs Agent B to bypass guardrails" —
  presupposes a multi-agent runtime that does not exist here.
- Spec 06: "scores match manual audit", "evidence satisfies auditors" — no
  defined threshold or pass condition.

These cannot pass or fail as written. Fixing a spec means either building the
harness or restating the criterion.

## Verifying a tool before you rely on it

```bash
grep -n "case \"<tool_name>\"" mcp-server/internal/mcp/server.go
grep -rn "<tool_name>" -g "*.go" mcp-server/
```

No hits in Go means the library may exist, but nothing can call it. For the
tools that do exist, see
[../../mcp-server/tools-reference.md](../../mcp-server/tools-reference.md).