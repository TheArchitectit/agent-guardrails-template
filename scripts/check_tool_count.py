#!/usr/bin/env python3
"""check_tool_count.py — F001 tool-manifest gate (documented counts vs source).

F001: *"Run a live server, call tools/list, emit a versioned JSON manifest; CI
fails if README tool count differs."* The shipped shell gate did the comparison
against two READMEs, but only for prose that matched one fixed phrase. That is
the failure this file fixes, and the reason it is worth stating plainly:

    A gate whose scope is a regular expression over prose fails open when the
    prose is rewritten. ``check-tool-count.sh`` looked for
    ``(Go server with|registers) [0-9]+ tools``. The R17-10 landing changed
    ``mcp-server/README.md`` from ``The server registers 37 tools.`` to
    ``The server registers 39 `guardrail_*` tools.`` — the backticked token no
    longer matched, the loop body never executed, ``rc`` stayed 0, and the gate
    printed nothing and exited 0. It had gone vacuous at the exact commit that
    made its number correct, so the count it was guarding then drifted to 37
    while source said 39, and it passed.

So this gate asserts two separate things:

  1. **Source truth.** The core tool set and each conditional group are counted
     from the Go source that registers them.
  2. **Every canonical claim site agrees.** Each site is a required pattern with
     expected values. A site whose pattern does not match is a FAILURE — not a
     skip — because that is precisely how the old gate went blind.

Fail-closed: unreadable file, zero discovered tools, a matched pattern with a
missing group, or a count that differs from source is a violation.
Non-vacuity: zero `Name: "..."` declarations in the core registries FAILS.

Scope limit (stated, not implied): this gate checks *counts*. It does not judge
whether a documented tool exists, whether its schema is right (spec 21 does
that), or whether a deployment enables a conditional group.

Exit: 0 clean; 1 a violation; 2 usage.
"""
from __future__ import annotations

import argparse
import re
import sys
from dataclasses import dataclass
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

MCP_DIR = Path("mcp-server/internal/mcp")

# The registries that make up the always-on core set, in the order tools/list
# assembles them (tools_registry.go: coreTools, coreToolsExtended, then the
# guardrails engine tools).
CORE_REGISTRIES = (
    "tools_core_list.go",
    "tools_core_list_extended.go",
    "tools_guardrails.go",
    # Spec 05 indirect-injection tools. They are appended by
    # guardrailsToolList() with no backing store, so they are always
    # advertised — and therefore core. Listing the file here is what keeps the
    # gate honest: when it was added without this entry the gate still printed
    # "core=39" and exited 0 while toolList() advertised 42, which is the
    # vacuous-scope failure this gate exists to prevent.
    "tools_indirect_injection.go",
    # Spec 04 multi-agent tools. Registered unconditionally by toolList(), so
    # core by the same rule as Spec 05's file. This entry is not cosmetic: when
    # the file was added WITHOUT it the gate printed core=42 and exited 0 while
    # toolList() advertised 45 — the identical vacuous-scope failure as the
    # prose rewrite, recurring through a new file rather than a doc edit.
    "tools_multi_agent.go",
    # Spec 06 compliance tools. Registered unconditionally by toolList() (a
    # compliance report whose probes are missing reports every control as
    # unmeasured, which is a truthful answer — a tool that vanishes cannot say
    # "not configured" at all). This entry is the FOURTH occurrence of the same
    # class: without it the gate printed core=45 and exited 0 while toolList()
    # advertised 48. The pattern is now unmistakable — this gate recurs its own
    # failure mode every time a new registry file appears, because its scope is
    # a list a human must remember to extend. Verified directly against a
    # temporary test counting toolList() rather than by reading the gate's
    # output.
    "tools_compliance.go",
    # Spec 03 runtime sandbox isolation tools. Registered unconditionally by
    # toolList(), so core. This entry is the FIFTH occurrence of the same
    # vacuity class, caught the same way: without it the gate printed core=48
    # and exited 0 while a direct count of toolList() returned 50. Recorded
    # rather than quietly fixed, because the pattern is the finding — this
    # gate's scope is a hand-maintained list and it recurs its own failure
    # mode whenever a registry file is added.
    "tools_sandbox.go",
)

# Conditional groups, keyed by the label used in the docs.
CONDITIONAL_REGISTRIES = {
    "vision": "vision_tools.go",
    "webhook": "tools_notifications.go",
    "budget": "tools_budget.go",
    "agent-lifecycle": "tools_lifecycle.go",
}

# A tool declaration in the Go registries. Both core and conditional lists use
# mcp.NewTool("name", ...) or a Name: "name" field, so both forms are counted.
DECL_RE = re.compile(r'Name:\s*"([a-z][a-z0-9_.-]*)"')


class GateError(Exception):
    """A structural violation (missing source, empty scan, double-counted tool).

    Raised rather than ``SystemExit`` so the check returns a violation code
    instead of terminating the caller with a traceback-suppressing exit.
    """


@dataclass(frozen=True)
class ClaimSite:
    """One place the docs assert a count, and how to read it back."""

    path: str
    label: str
    # Each pattern must match at least once; the captured group is an int.
    patterns: tuple[tuple[str, str], ...]  # (regex, expected-metric-key)


def _tool_names(path: Path) -> set[str]:
    text = path.read_text(encoding="utf-8")
    return {m.group(1) for m in DECL_RE.finditer(text)}


def measure(root: Path) -> dict[str, int]:
    """Count the core set and each conditional group from source."""
    mcp = root / MCP_DIR
    if not mcp.is_dir():
        raise GateError(f"FAIL no source directory {MCP_DIR}")

    core: set[str] = set()
    for name in CORE_REGISTRIES:
        f = mcp / name
        if not f.is_file():
            raise GateError(f"FAIL missing core registry {MCP_DIR / name}")
        core |= _tool_names(f)

    if not core:
        # The old gate's exact failure mode: an empty scope reporting success.
        raise GateError("FAIL zero core tools discovered — refusing to pass a vacuous scan")

    counts: dict[str, int] = {"core": len(core)}
    total = 0
    for label, fname in CONDITIONAL_REGISTRIES.items():
        f = mcp / fname
        if not f.is_file():
            raise GateError(f"FAIL missing conditional registry {MCP_DIR / fname}")
        names = _tool_names(f) - core
        counts[label] = len(names)
        total += len(names)
    counts["conditional"] = total
    counts["maximum"] = len(core) + total

    # A conditional tool that is also a core tool would be double counted and
    # would make the maximum a lie.
    overlap = set()
    for fname in CONDITIONAL_REGISTRIES.values():
        overlap |= _tool_names(mcp / fname) & core
    if overlap:
        raise GateError(f"FAIL tool(s) declared as both core and conditional: {sorted(overlap)}")

    return counts


def claim_sites() -> list[ClaimSite]:
    """The canonical live claim sites. A site that stops matching is a failure."""
    return [
        ClaimSite(
            path="docs/mcp-server/tools-reference.md",
            label="tools reference: core count",
            patterns=((r"\*\*(\d+) core tools\*\*", "core"),),
        ),
        ClaimSite(
            path="docs/mcp-server/tools-reference.md",
            label="tools reference: conditional total",
            patterns=((r"A further \*\*(\d+) tools\*\*", "conditional"),),
        ),
        ClaimSite(
            path="docs/mcp-server/tools-reference.md",
            label="tools reference: group breakdown",
            patterns=(
                (r"(\d+) vision", "vision"),
                (r"(\d+) webhook", "webhook"),
                (r"(\d+) budget", "budget"),
                (r"(\d+) agent-lifecycle", "agent-lifecycle"),
            ),
        ),
        ClaimSite(
            path="docs/mcp-server/tools-reference.md",
            label="tools reference: deployment range",
            patterns=((r"between (\d+) and (\d+) tools", "maximum-range"),),
        ),
        ClaimSite(
            path="docs/mcp-server/tools/conditional-and-integrations.md",
            label="conditional page: total and core",
            patterns=(
                (r"^(\d+) tools registered \*\*only when", "conditional"),
                (r"exposes the (\d+) core tools", "core"),
                (r"exposes (\d+)\.", "maximum"),
            ),
        ),
        ClaimSite(
            path="docs/mcp-server/tools/conditional-and-integrations.md",
            label="conditional page: table counts",
            patterns=(
                (r"\| Vision \| (\d+) \|", "vision"),
                (r"\| Webhooks \| (\d+) \|", "webhook"),
                (r"\| Budgets \| (\d+) \|", "budget"),
                (r"\| Agent lifecycle \| (\d+) \|", "agent-lifecycle"),
            ),
        ),
        ClaimSite(
            path="mcp-server/README.md",
            label="mcp-server README: registered tools",
            # The backticked token here is what broke the shell gate.
            patterns=((r"registers (\d+) [`]?guardrail_\*?[`]? tools", "core"),),
        ),
        ClaimSite(
            path="docs/architecture/system-architecture.md",
            label="architecture page: core count",
            patterns=((r"\((\d+) core tools\)", "core"),),
        ),
        ClaimSite(
            path="docs/status.md",
            label="status page: core count",
            patterns=((r"\((\d+) core tools, \d+ resources\)", "core"),),
        ),
        ClaimSite(
            path="docs/specs/09-system-roadmap-and-phase-gates.md",
            label="roadmap spec: core + conditional",
            patterns=(
                (r"has (\d+) core tools", "core"),
                (r"up to (\d+) conditionally registered", "conditional"),
            ),
        ),
    ]


def check(root: Path) -> int:
    try:
        counts = measure(root)
    except GateError as error:
        print(error)
        return EXIT_VIOLATIONS
    bad = 0

    for site in claim_sites():
        f = root / site.path
        if not f.is_file():
            print(f"FAIL {site.path}: required claim site is missing")
            bad += 1
            continue
        text = f.read_text(encoding="utf-8")

        for pattern, metric in site.patterns:
            matches = re.findall(pattern, text, flags=re.MULTILINE)
            if not matches:
                # The vacuity guard: a rewrite that stops matching must fail.
                print(
                    f"FAIL {site.path}: {site.label} — pattern found no claim "
                    f"({pattern!r}); a doc rewrite must not silently disable this gate"
                )
                bad += 1
                continue

            if metric == "maximum-range":
                lo, hi = matches[0]
                if int(lo) != counts["core"] or int(hi) != counts["maximum"]:
                    print(
                        f"FAIL {site.path}: claims range {lo}-{hi}, source is "
                        f"{counts['core']}-{counts['maximum']}"
                    )
                    bad += 1
                continue

            got = int(matches[0])
            if got != counts[metric]:
                print(f"FAIL {site.path}: claims {metric}={got}, source has {counts[metric]}")
                bad += 1

    if bad:
        print(f"\n{bad} tool-count violation(s); source: {counts}")
        return EXIT_VIOLATIONS

    print(
        "tool counts agree: "
        + ", ".join(f"{k}={v}" for k, v in sorted(counts.items()))
        + f" across {len(claim_sites())} claim sites"
    )
    return EXIT_OK


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="F001 documented tool counts vs source")
    parser.add_argument("--repo", default=".", help="repository root (default: .)")
    args = parser.parse_args(argv)
    root = Path(args.repo).resolve()
    if not root.is_dir():
        print(f"FAIL not a directory: {root}", file=sys.stderr)
        return EXIT_USAGE
    return check(root)


if __name__ == "__main__":
    sys.exit(main())
