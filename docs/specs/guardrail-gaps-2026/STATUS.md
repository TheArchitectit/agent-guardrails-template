# Guardrail Gaps 2026 — Spec vs Shipped Code

**Status: reconciliation, 2026-10-03; wave-0 release-gating fixes reconciled
2026-10-09; Spec 01 injection tools and Spec 05 provenance tools implemented
2026-10-18; the Spec 02 (`CONTENT_FILTER_CONFIG_FILE`) and Spec 05
(`PROVENANCE_CONFIG_FILE`) config files are now read and validated at startup,
2026-10-19.** This page supersedes the `Status: Draft` labels on the six specs.

The six specs in this directory are **proposals written before the reboot**,
not descriptions of the system. Since then a Go rewrite implemented much of
the underlying library code. This page records, requirement by requirement,
what actually exists and what can actually be called.

## The headline finding

**Of the 15 MCP tools these specs propose or name, all 15 now exist**, verified
by grepping every `.go` file in the repository (2026-10-19). The table below is
the per-tool record; when it was first written (2026-10-03) only 7 existed,
and the headline said so. As of 2026-10-18 the Spec 01, 03, 04, 05 and 06
tools were added, so the stale count is corrected here rather than left to
contradict the table underneath it.

| Proposed tool | Spec | Exists? |
|---------------|------|---------|
| `guardrail_detect_injection` | 01 | **Yes (2026-10-18)** |
| `guardrail_scan_text_batch` | 01 | **Yes (2026-10-18)** |
| `guardrail_sandbox_execute` | 03 | **Yes (2026-10-18)** |
| `guardrail_sandbox_config` | 03 | **Yes (2026-10-18)** |
| `guardrail_validate_agent_output` | 04 | **Yes (2026-10-18)** |
| `guardrail_check_agent_constraints` | 04 | **Yes (2026-10-18)** |
| `guardrail_resolve_conflicts` | 04 | **Yes (2026-10-18)** |
| `guardrail_scan_external_content` | 05 | **Yes (2026-10-18)** |
| `guardrail_mark_provenance` | 05 | **Yes (2026-10-18)** |
| `guardrail_check_provenance` | 05 | **Yes (2026-10-18)** |
| `guardrail_generate_compliance_report` | 06 | **Yes (2026-10-18)** |
| `guardrail_check_compliance` | 06 | **Yes (2026-10-18)** |
| `guardrail_collect_evidence` | 06 | **Yes (2026-10-18)** |
| `guardrail_classify_content` | 02 | **Yes** |
| `guardrail_check_policy` | 02 | **Yes** |

## The second finding: "library exists" ≠ "shipped"

Several subsystems are fully written, well-tested, and **unreachable from
any MCP tool**. As of 2026-10-18 the `internal/mcp` package calls these
methods on the guardrails engine (line numbers move; the names do not):

```
handleClassifyContent      ClassifyContent(...)        // classify_content
handleCheckPolicy          ClassifyContent(...)        // check_policy pre-check
handleCheckPolicy          CheckPolicy(...)            // check_policy
handleDetectInjection      DetectInjection(...)        // detect_injection
handleScanTextBatch        DetectInjectionBatch(...)   // scan_text_batch
handleScanExternalContent  ScanExternalContent(...)    // scan_external_content
handleCheckProvenance      LookupProvenance(...)       // check_provenance
```

The sandbox manager and the compliance mapper remain constructed at startup
and never called. Three subsystems have left that list: the injection pipeline
(two Spec 01 tools), the provenance tracker (three Spec 05 tools) and the
multi-agent safety chains (three Spec 04 tools). Note what the tracker wiring required — the
Spec 05 tools call `ScanExternalContent`, not `Evaluate`, because `Evaluate`
executes commands through the sandbox manager when `input.Command` is set, and
routing a content-scan tool through it would start process execution from the
MCP path.

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
| L1 pattern matching | **Implemented, WIRED, and fail-closed** — the shipped `DefaultPipelineConfig` compiles the built-in `DefaultPatterns()` set; before that pass L1 was "enabled" with zero patterns compiled, so every scan returned `safe:true` without evaluating. Since 2026-10-19 the matcher also REFUSES to degrade: an unreadable declared blocklist, an uncompilable pattern, or an enabled matcher with zero compiled patterns are errors, and a failed `Reload` leaves the prior set in force (pinned by `injection_l1_failclosed_test.go`) |
| L2 perplexity | Partial — analyser exists, **disabled by default** |
| L3 classifier backend | **Wired 2026-10-18** — the pipeline is no longer constructed with `NoOpClassifier{}` (which answered "safe" for text it never evaluated). It is built with a **nil** classifier unless a backend is attached; `EnableInjectionClassifier(classifier, cfg)` attaches the backend and its switch together; `cmd/server/main.go` enables the real `OllamaClassifier` when `OLLAMA_URL` is set (no longer dead code). `InjectionResult.layers_evaluated` names the layers that actually ran. An unreachable backend errors on `Classify` and fails closed. Pinned by `injection_l3_wiring_test.go` |
| L4 LLM self-check | **Not implemented, now refused instead of ignored** — `NewPipeline` returns an error when `llm_self_check.enabled` is set, because no code path ever assigned the layer and the scan looked complete anyway |
| Config keys | **LOADED AT STARTUP 2026-10-19** (`INJECTION_CONFIG_FILE`) — two fail-opens closed. The loader had zero non-test callers, so a configured file was inert; and it unmarshalled only the flat struct-tag form while Sec. 3.2 documents a nested `layers:` shape, so a verbatim copy of the documented block loaded with **L1/L2/L3 all false** (probed: `Enabled=true L1=false L2=false L3=false`). Both forms now load; unknown keys, a missing `injection_defense` section, mixed spellings, empty blocklist paths and a config failing `Validate()` are all errors; a configured-but-unusable file stops startup. `DetectInjection` also no longer reports `safe:true` when the pipeline is nil |
| Blocklist files | `config/blocklists/` does not exist (no corpus ships — an owner decision, not a mechanism gap). `BlocklistPaths` is honoured when set and empty by default; the *mechanism* is fail-closed as of 2026-10-19 — a declared-but-unreadable blocklist, an uncompilable declared pattern, or an enabled L1 compiling to zero patterns are all construction errors, and a failed `Reload` keeps the prior compiled set |
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
| OpenAI Moderation backend | **Registered 2026-10-18** — constructed in `cmd/server/main.go` when `OPENAI_API_KEY` is set |
| Streaming | Not implemented |
| `taxonomy.enabled_categories` / `custom_categories` | **Enforced 2026-10-19** — was parsed and never read; the policy engine now carries the declared taxonomy as an allow-list |
| YAML loader (`LoadContentConfig`) | **READ AT STARTUP 2026-10-19** (`CONTENT_FILTER_CONFIG_FILE`) — was zero-caller; the documented `content_filter:` wrapper loaded with **zero policies** because the loader only accepted the flat struct-tag form. Both forms load now (not both at once); unknown keys, a missing section, an unimplemented backend, an invalid action and `fail_policy: allow` are all refused |

**Defect fixed 2026-10-03:** `NewEngine` built the filter as
`NewContentFilter(nil, nil)`, discarding `config.ContentFilter.Policies`. With
no rules loaded, every `guardrail_check_policy` call took the fail-closed
"unknown policy" branch. The configured policy set is now passed to the
filter, so a correctly configured policy resolves; unknown policies still
fail closed by design.

### 03 — Runtime sandbox isolation

| Requirement | Status |
|-------------|--------|
| `guardrail_sandbox_execute` / `guardrail_sandbox_config` tools | **Implemented 2026-10-18** — registered, dispatched, mutate/admin-classified, audit-before-effect |
| Per-tool policy applied at execution | **Implemented 2026-10-18** — `ExecuteFor`/`ExecuteForLimits` resolve level + limits from `tool_policies`; previously the table had no consumer |
| `DefaultLevel` validated | **Implemented 2026-10-18** — an empty or unknown level is a load error, not a silent L0 |
| L0 / L1 (unshare) / L2 (podman→docker) | Implemented, and stronger than spec'd |
| `resource_limits` config | Implemented, field-for-field |
| Network isolation | Implemented — fail-closed, host allowlist via filtering proxy |
| Path-traversal **detection** | Not implemented (only mount-path syntax validation) |
| Privilege-escalation / fork-bomb detection | Not implemented (3 of 5 required violation classes absent) |
| L3 Firecracker | Not implemented (the spec itself marks it optional) |

**What the wiring deliberately adds beyond the spec's signature.** The spec
asks for a tool that runs a command at a level the caller names. Shipped as
specified, that is a host-execution surface with a reassuring name: any
caller could pass `L0`, and any caller could pass limits wider than the
deployment's policy. The two tools therefore constrain the request:

- **L0 is refused outright**, even when a policy declares it. `file_edit`
  ships an L0 policy, so a policy level of L0 is legal *config*; the tool
  that claims to isolate still refuses to be the path to the host.
- **Per-request overrides narrow only.** A larger memory/time/CPU value or a
  path outside the policy's writable set is refused with the field named —
  not clamped, because a clamp reads as "your override applied".
- **A requested level may raise isolation, never lower it.**
- **An unknown `tool_type` is refused** rather than run at the global
  defaults, and the refusal names the types that exist.

Executing under L1 where namespace creation is unavailable fails closed: the
command does not run on the host, and the result reports the level that was
attempted. Verified by a test that touches a marker file and asserts its
absence.

**Still open in 03** (stated, not stubbed): path-traversal *detection*
beyond mount-syntax validation; privilege-escalation and fork-bomb violation
classes; a YAML loader for `sandbox_policies` (the Go key remains
`tool_policies`); and Sec. 5.x security criteria that need a defined
container-escape/fork-bomb harness.

**Spec amendment, not a gap:** the spec (Sec. 4.3) specifies unconditional
L2→L1→L0 fallback. The implementation deliberately does not: it downgrades
only on genuine setup errors, and treats a command *denied while running
under isolation* as a security breach to be reported. That is the correct
fail-closed behaviour and the spec text is wrong. Sec. 4.3 should be amended.

### 04 — Multi-agent safety policies

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | **Implemented 2026-10-18** — registered, dispatched, read-class |
| Safety chain engine | **Wired 2026-10-18** — `MultiAgentRegistry` built from the live engine at startup |
| Validators `injection_defense` / `content_filter` / `four_laws_check` | **Wired** — bound to the same engine the rest of the server uses |
| `code_review_agent` validator | Not implemented (needs an LLM review backend — owner call) |
| Conflict strategies | **Wired**; shipped name is `escalate` (spec's `human_escalate` maps to it) |
| Config keys as YAML (`safety_chains`) | **READ AT STARTUP 2026-10-19** (`SAFETY_CHAINS_CONFIG_FILE`) — the loader had **zero callers**, so the documented Sec. 2.2 block changed nothing and the server ran its built-in chains regardless. Both the loader and the wiring landed this pass. Unknown keys, a missing/empty section, a chain with no steps, an unknown validator (refused by name), a missing/unknown step action, and `on_failure: warn_and_continue`/`block_and_escalate` are all refused; a configured-but-unusable file stops startup; a declared chain list REPLACES the built-in one so a disabled chain is genuinely absent |
| Config keys as YAML (`agent_registry`, `cross_agent_policies`) | Not implemented — no loader for either; the constraint policies ship built-in and the tools take the policy id directly |
| `on_failure` semantics | **Corrected 2026-10-19** — the built-in `research_output` chain declared `warn_and_continue`, which `SafetyChain.Execute` never reads: a blocking step stops the chain and records a block decision whatever the field says. The declaration was inert and fail-open if honoured. Built-in chains now declare the one implemented semantic (`block_and_notify`); a warning is a step's own `action: warn`, and the loader refuses the two unimplemented chain-level values by name |
| Law-2 keyword false positive | **Fixed 2026-10-18** — quoted/mentioned occurrences no longer fail |

**Law 2 was a false-positive generator, and wiring it would have shipped the
bug.** The check failed any output that merely *contained* a configured scope
keyword, so a runbook line, a quoted policy, or the prohibition itself
("never run delete-db") was reported as a scope violation. It now treats an
occurrence as discussion when it is quoted or introduced by a mention marker
(`never`, `documented`, `forbidden`, `the word`, …), and still fails an
unqualified occurrence. Named limit: this is a heuristic over prose, not a
parser — a real directive quoted as an example passes Law 2.

**An absent classifier is now reported as absent.** A chain step whose backend
is not configured (Spec 02 with no `OLLAMA_URL`/`OPENAI_API_KEY`) used to fail
closed, which is right for a direct classification question and wrong as a
chain step: it labelled benign output "Content blocked: SYSTEM". The step now
returns an `UnavailableError`; the chain records it in `UnavailableSteps`, keeps
running its remaining detectors, and reports `decision: incomplete` — never
`passed`. "Every step passed" and "one step never ran" are different answers.

**The `safety_chains:` file was inert, and one declared behaviour was never
honoured (both fixed 2026-10-19).** Two defects on one seam, the same class as
the Spec 01/02/03/05 config files fixed earlier in this program. (1) The
registry built its three chains as Go literals and `LoadSafetyChainsFromYAML`
did not exist, so an operator who wrote the documented Sec. 2.2 block got the
built-in chain instead — a review trail asserting validation that never ran.
`SAFETY_CHAINS_CONFIG_FILE` now loads it, strict and fail-closed: unknown keys,
a missing/empty section, a chain with no steps, an unknown validator (named,
not ignored — `code_review_agent` is refused because this tree has no LLM review
backend), a missing or unknown step action, and both unimplemented chain-level
`on_failure` values are refused; a declared chain list REPLACES the built-in one,
so a disabled chain is genuinely absent and its id is unknown (already refused).
(2) The built-in `research_output` chain declared `on_failure:
warn_and_continue`, which `SafetyChain.Execute` never reads — a blocking step
stops the chain and records a block decision regardless. The declaration was
inert *and* fail-open if honoured (P5/D-18: fail-closed is not configurable), so
the built-in chains now declare `block_and_notify` and a warning is expressed
with a step's own `action: warn`, which is implemented and enforced. Evidence:
`safety_chain_file_test.go` drives the REAL loader (with a verbatim copy of the
spec's documented YAML) and the REAL registry/chain — the loaded chain is proven
to be the chain that RUNS, not merely the chain that parsed; 13 refusal shapes;
**red-on-drift proven three ways** (registry wires built-ins regardless; drop
`KnownFields`; remove the fail-open guard), each PASS→FAIL then restored.
`go test ./...` EXIT=0 (21 pkgs); `regression_check --all --pre-commit` exit 0;
all repo gates green.

### 05 — Indirect prompt injection

The most completely built of the six, and as of 2026-10-18 also **wired**.

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | **Implemented 2026-10-18** |
| Config keys (`source_trust_policies`, `untrusted_overrides`) | **Implemented**, including the spec's trust table |
| Sanitization / base64 / ROT13 / URL decoding | Implemented, **now applied before the scan** |
| Provenance marker wrapping | Implemented, **now reachable via `guardrail_mark_provenance`** |
| Redis cache | Deviates — implemented as in-memory (`provenance_cache.go`) |
| YAML loader (`LoadProvenanceConfigFromYAML`) | **READ AT STARTUP 2026-10-19** (`PROVENANCE_CONFIG_FILE`) — was zero-caller, so the documented trust table was inert. Now strict: unknown keys, a missing `indirect_injection` section and a config failing `Validate()` are errors; a configured-but-unusable file stops startup |

**The defect this pass fixed was an ordering defect, and it made the whole
feature cosmetic.** `DecodeObfuscation`/`SanitizeContent` existed and worked;
they simply ran *after* anything that could act on their output. The only
injection scan in the tree took RAW text, so L1's literal regexes were trivially
evaded: `"Ig\u200bnore all previous instructions"` inserts one zero-width
character and `ignore\s+` no longer matches. The sanitiser was a function
nobody called with the scan's input.

`Engine.ScanExternalContent` now runs the pipeline in the order the spec
describes: sanitise → decode labelled obfuscation → resolve the source trust
policy → scan **the sanitised text** → return the sanitised text the agent
should consume. The trust decision still governs whether a deep scan happens
(the spec's `action: allow` for trusted sources), but sanitisation is
unconditional: invisible characters are never meaningful in trusted prose
either.

**A second, narrower defect found in the existing sanitiser.** `hasBase64Marker`
accepted `base64: <payload>` as an encoded string, and `tryBase64Decode` then
tried to decode the *entire* string including the label — which cannot succeed.
The marker announced "I am encoded" and the decoder could never agree, so the
base64 branch was unreachable for exactly the inputs that declared themselves
encoded. Fixed by extracting the payload from its label first (`base64Payload`:
prefix label, `data:` URI, `;base64,` suffix).

The tracker is constructed at startup and now reachable three ways from the
request path.

**`guardrail_record_file_read` / `guardrail_verify_file_read` are not this
spec's provenance tools.** They are session-token-gated file-read
attestation for Law-3 transparency, backed by a database store. They never
touch the provenance tracker, trust levels or content scanning, and do not
satisfy spec 05.

### 06 — Regulatory compliance mapping

| Requirement | Status |
|-------------|--------|
| 3 proposed tools | **Implemented 2026-10-18** — registered, dispatched, read-class, on the MEASURED path |
| Requirement DB (`compliance_requirements.json`) | Exists; covers EU AI Act art. 9, 10, **11**, 12, 13, 14, 15, 50 and NIST RMF govern/map/measure/manage |
| `CalculateComplianceScore` | Implemented (legacy declared-status path, stamped `unimplemented`) |
| `compliance_reports:` config file (Sec. 2.3) | **Read + enforced 2026-10-19** — `COMPLIANCE_CONFIG_FILE` → `LoadComplianceConfigFromYAML` at startup; the block had **no reader** before (`ComplianceConfig` had zero production call sites) |
| `sections:` (Sec. 2.3) | **Not implemented — refused by name.** The measured report emits a fixed structure; a declared list would be inert |
| `config_history` evidence source (Sec. 2.3/3.3) | **Not implemented — refused by name.** No code path records a restorable configuration snapshot |
| Gap tracking / dashboard | Not implemented |

The data file covers EU AI Act art. 9, 10, 11, 12, 13, 14, 15, 50 and NIST RMF
govern/map/measure/manage. **Art. 11 is now present**, added 2026-10-19 with
`compliance_status: "gap"` and explicit gap text, so spec Sec. 5.1's own
acceptance criterion is met honestly — the article is mapped, and mapped as an
acknowledged gap rather than as a control this repo has.

**Two score paths exist; only one of them measures.** `CheckRequirement` /
`GenerateReport` is the legacy declared-status path: it returns `true` purely
because a hand-written string in the JSON says `"full"`, and `CollectEvidence`
hardcodes its result. That path is stamped `UnimplementedMethodology =
"unimplemented"` and its evidence is `Verified:false`. The three Sec. 3.1 tools
do NOT call it: they call `CheckMeasured` / `GenerateMeasuredReport`, which
count a requirement only when every control it names was observed live, list
unmeasured requirements separately, and stamp `MeasuredMethodology =
"measured-runtime-probes-v1"`.

**A probe must observe the control the requirement NAMES** (fixed 2026-10-19).
`content_filter_watermark` was aliased to the content filter's probe, so EU AI
Act Art. 50 reported an OBSERVED ACTIVE control the moment any semantic
content-filter backend was attached — while `grep -ri watermark` over the tree
returns **1 hit: the alias itself**. A declared, acknowledged gap was being
converted into a measured compliance claim on the exact path built to stop
restating declarations. The alias is removed; with no probe able to observe it,
Art. 50 measures `unknown` and never counts as covered.
`declarationAtLeast` now also distinguishes UNKNOWN: nothing observed the
control, so nothing can contradict a declaration of *deficiency* — Art. 50
(declared `gap`) is no longer falsely reported as an over-declaration, while a
declaration of `full` still is.

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
| 01/03/04/06 libraries unreachable from the request path | **CLOSED** — 01, 03, 04, 05, 06 all reachable | `f85aa67` + 2026-10-18 | every gap-spec library now has a registered, dispatched, classified tool: 01 (2 tools), 03 (2), 04 (3), 05 (3), 06 (3). Residual per-spec gaps are listed under each spec and none of them is reachability |

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
- Guardrail capabilities unwired from the request path (`03`/`06`; `09-P2`). Gaps `01`, `04` and `05` are wired as of 2026-10-18.
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