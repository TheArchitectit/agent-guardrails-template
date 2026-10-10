# OpenSpec: Tool Schema ↔ Handler Conformance Gate

**Status:** Implemented (gate live) — opened 2026-10-10 (spin-out from spec 20
/ spec 10 §4.1.2); gate landed 2026-10-10. The gate runs and blocks on drift;
the discovered drift inventory is pinned in an allowlist and remains open work.
This is the CI gate that closes the single most important required
FastMCP-parity item.

**Priority:** High — blocking gate; without it "canonical tool contract" is a
claim, not a check.

**Purpose:** A same-commit CI job that enumerates every registered tool,
compares its advertised JSON input schema against the handler's actual
argument reads, and fails on drift unless the drift is explicitly allowlisted
with a deprecation/removal issue.

## 1. Problem

The AUDIT (§10-4.1.2) records this requirement as **ABSENT** — no conformance
test exists. Handlers currently validate inputs, but nothing proves the
advertised schema and the handler agree. Schema/handler drift is the top
recorded risk in spec 10 §1.

## 2. Requirements

**R21.1** The gate SHALL enumerate every registered tool from the live registry
(not a hand-maintained list).

**R21.2** For each tool the gate SHALL decode the advertised input schema and
assert it covers every required argument the handler reads, and that every
declared input is either read or explicitly unused-with-reason.

**R21.3** Undeclared required reads and declared-but-unused inputs SHALL fail
CI unless present in an allowlist file, and every allowlist entry SHALL name a
tracking issue or removal date.

**R21.4** The gate SHALL run on the production registry with conditional tools
in their enabled and disabled states, and SHALL report the active inventory.

**R21.5** The gate SHALL NOT be weakenable by skipping, `continue-on-error`, or
a relaxed threshold; a failure is a failure.

**R21.6** The gate SHALL be demonstrated red on an injected drift and green on
the clean tree in the same evidence turn.

## 3. Acceptance criteria

1. Registry enumeration is live and total.
2. Drift count is zero, or every non-zero entry is allowlisted with a tracked
   reason.
3. Red-on-drift and green-on-clean both captured with run ids.

## 4. Dependencies

Blocks closure of spec 20 R20.2 and spec 10 §4.1.2. Depends on spec 09 Phase 1
(contract integrity).

## 5. Closure status — 2026-10-10

**Implemented.** The gate is
`mcp-server/internal/mcp/schema_handler_conformance_test.go`:
`TestToolSchemaHandlerConformance` (blocking) plus
`TestToolSchemaHandlerConformance_DetectsInjectedDrift` (red-path proof).
Allowlist: `mcp-server/internal/mcp/testdata/tool_contract_drift_allowlist.json`.

How the gate works. It parses the package's own non-test `.go` files with
`go/parser` at test time — never a hand-maintained tool list (R21.1). It
extracts every tool declared in any function/method that returns a
`[]mcp.Tool` (`coreTools`, `coreToolsExtended`, `guardrailsToolList`,
`visionToolList`, `notificationToolList`, `budgetToolList`,
`lifecycleToolList`), so conditional tools are enumerated in both enabled and
disabled states (R21.4). For each tool it unions the `args[...]` keys read in
the matching `case "<tool>"` of `dispatchToolCall` / `VisionToolSet.dispatch`
with the keys read in every package-local helper those cases pass `args` to
(R21.2). It then fails on any of the two drift families — an undeclared
required read, or a declared-but-unused input — that is not present in the
allowlist, and fails on any stale allowlist entry (R21.3). There is no skip,
no `continue-on-error`, and no threshold (R21.5).

Measured result (2026-10-10, branch `forge/spec-program-wave0`):

```
TestToolSchemaHandlerConformance
    enumerated 58 registered tools; 24 drifting, 24 allowlisted
```

Drift is real and matches `docs/mcp-server/tools-reference.md` ("The published
schemas are not trustworthy"). The 24 allowlisted tools are the defect
inventory, not a waiver: two families dominate — `session_token`/`session_id`
threaded through the halt/attempt/file-read and extended guardrail tools but
absent from the schema (undeclared read), and dead schema properties such as
`files` (validate_commit), `remote` (validate_push), `bug_id`
(verify_fixes_intact), `target_string` (validate_exact_replacement),
`advisor_name`/`role` (team_assign), `advisor_name`/`query` (advisor_query),
`teams` (team_init), `context` (classify_content) (declared-but-unused).

Red/green evidence in one turn (R21.6). Commands and results:

- Red — remove one allowlist entry, then run the gate:

  ```
  $ go test ./internal/mcp/ -run 'TestToolSchemaHandlerConformance$' -count=1
  --- FAIL: TestToolSchemaHandlerConformance (0.03s)
      schema_handler_conformance_test.go:373: schema/handler drift in
      "guardrail_validate_push" is not allowlisted: undeclared_reads=
      [has_unpushed_commits is_force] declared_unused=[remote]
      schema_handler_conformance_test.go:390: enumerated 58 registered tools; 24 drifting, 23 allowlisted
  FAIL
  ```

- Green — restore the allowlist, then run the gate:

  ```
  $ go test ./internal/mcp/ -run 'TestToolSchemaHandlerConformance' -count=1
  ok  github.com/thearchitectit/guardrail-mcp/internal/mcp
  ```

The injected-drift path is also proven without editing the tree by the pure
`computeDrift` function under
`TestToolSchemaHandlerConformance_DetectsInjectedDrift`.

Honest limits. (a) The gate is not yet wired into a workflow step in
`.github/workflows/team-validation.yml`; it currently runs inside the existing
`go test ./... -race` job, which is a blocking job, so drift already fails CI —
but there is no standalone named step/run id yet. (b) Drift count is **not yet
zero**; it is 24/58 allowlisted (R21 acceptance criterion 2 is met via the
tracked allowlist; the spec-20 R20.2 "fail on un-allowlisted drift" intent is
met, but the contract is not yet canonical). (c) The parser attribute family
is `args`/`a`/`arguments` (the names this package uses); a future handler using
a different local variable name would need the visitor extended. (d) Engine
closure is deferred until the drift inventory is remediated to zero.
