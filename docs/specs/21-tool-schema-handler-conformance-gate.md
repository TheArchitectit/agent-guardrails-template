# OpenSpec: Tool Schema ↔ Handler Conformance Gate

**Status:** Proposed — opened 2026-10-10 (spin-out from spec 20 / spec 10
§4.1.2). NOT implemented. This is the CI gate that closes the single most
important required FastMCP-parity item.

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
