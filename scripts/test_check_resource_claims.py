#!/usr/bin/env python3
"""test_check_resource_claims.py — red-on-drift fixture for check_resource_claims.py.

A gate that has only ever been observed passing is not evidence. Each case below
builds a throwaway tree, asserts the gate's verdict on it, and restores nothing
because nothing outside the temp dir is touched.

The cases that matter most are the two the real repository actually failed on
2026-10-18:

  * a documented URI that is not registered (`guardrail://quick-reference`), and
  * a count that disagrees with the registry (11 documented vs 12 served).

Exit: 0 all cases behave; 1 a case behaved wrongly.
"""
from __future__ import annotations

import subprocess
import sys
import tempfile
from pathlib import Path

GATE = Path(__file__).with_name("check_resource_claims.py")

REGISTRY_TEMPLATE = """package mcp

func (s *MCPServer) resourceSpecs() []resourceSpec {{
\treturn []resourceSpec{{
{entries}\t}}
}}
"""

# Each entry mirrors the real registry: `uri:` on its own line inside a struct
# literal. The first version of this fixture wrote `{uri: "…", name: "…"}` on one
# line, which the gate's `^\s*uri:` anchor correctly refused to match — so every
# case failed for an unrelated reason and the self-test proved nothing.
ENTRY = '\t\t{{\n\t\t\turi:  "{uri}",\n\t\t\tname: "{name}",\n\t\t}},\n'

README_TEMPLATE = """# Server

## Web routes

- `GET /ide/quick-reference` - Get quick reference documentation

### MCP Resources

{count} resources exposed via `resources/list`:

{bullets}

### Connecting to MCP Server
"""


def build(root: Path, uris: list[str], documented: list[str], count: int) -> None:
    reg = root / "mcp-server/internal/mcp/resource_registration.go"
    reg.parent.mkdir(parents=True, exist_ok=True)
    entries = "".join(ENTRY.format(uri=u, name=u.split("//")[1]) for u in uris)
    reg.write_text(REGISTRY_TEMPLATE.format(entries=entries), encoding="utf-8")

    bullets = "\n".join(f"- `{u}` - doc" for u in documented)
    readme = root / "mcp-server/README.md"
    readme.parent.mkdir(parents=True, exist_ok=True)
    readme.write_text(
        README_TEMPLATE.format(count=count, bullets=bullets), encoding="utf-8"
    )


def run(root: Path) -> tuple[int, str]:
    proc = subprocess.run(
        [sys.executable, str(GATE), "--repo", str(root)],
        capture_output=True,
        text=True,
    )
    return proc.returncode, proc.stdout + proc.stderr


def expect(name: str, want_ok: bool, want_sub: str = "") -> bool:
    with tempfile.TemporaryDirectory() as tmp:
        root = Path(tmp)
        uris, documented, count = CASES[name]
        build(root, uris, documented, count)
        code, out = run(root)
    ok = (code == 0) == want_ok
    if ok and want_sub and want_sub not in out:
        ok = False
        out += f"\n(expected substring {want_sub!r} was absent)"
    status = "PASS" if ok else "FAIL"
    print(f"{status} {name}: exit={code}")
    if not ok:
        print(out)
    return ok


BASE = ["guardrail://config", "guardrail://stats"]

CASES = {
    # The clean tree must pass, otherwise every refusal below is vacuous.
    "clean": (BASE, BASE, 2),
    # The exact defect: a documented resource that is not registered.
    "phantom-resource": (
        BASE,
        BASE + ["guardrail://quick-reference"],
        2,
    ),
    # The count defect seen on 2026-10-18: documented 11, served 12.
    "stale-count": (BASE, BASE, 11),
    # A registered resource nobody documents is undiscoverable.
    "undocumented-resource": (BASE, ["guardrail://config"], 2),
    # A vacuous scan must never report success.
    "empty-registry": ([], [], 2),
}


def main() -> int:
    failures = 0
    for name in ("clean",):
        if not expect(name, want_ok=True):
            failures += 1
    if not expect("phantom-resource", want_ok=False, want_sub="not registered"):
        failures += 1
    if not expect("stale-count", want_ok=False, want_sub="claims 11 resources"):
        failures += 1
    if not expect("undocumented-resource", want_ok=False, want_sub="is not documented"):
        failures += 1
    if not expect("empty-registry", want_ok=False, want_sub="vacuous scan"):
        failures += 1

    print(f"\nRan {len(CASES)} cases, {failures} failure(s)")
    return 0 if failures == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
