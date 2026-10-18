#!/usr/bin/env python3
"""check_resource_claims.py — MCP resource truth gate.

Why this exists (and why it is separate from check_tool_count.py):

On 2026-10-18 the checked-in `mcp-server/README.md` advertised
``11 resources exposed via `resources/list` `` and listed
``guardrail://quick-reference`` among them. The live registry served **12**
resources and `guardrail://quick-reference` **was not one of them** — it is a web
route (`GET /ide/quick-reference`), not an MCP resource. A client reading the
README would send `resources/read` for a URI the server never registered.

`check_tool_count.py` could not catch this for two reasons, and both are
structural rather than an oversight:

  1. It counts *tools*. Resource counts were never in its scope.
  2. It compares **numbers**. Every number here could have been right and the
     wrong URI would still have shipped.

So this gate asserts two independent things:

  * **Count.** The number of registered resources, parsed from the one declared
    registry (`resourceSpecs()`), must match every canonical claim site.
  * **Existence.** Every `guardrail://` URI the README documents as an MCP
    resource must actually be registered. This is the check that would have
    caught the phantom resource.

Fail-closed and non-vacuous:
  * zero resources discovered in the registry  -> FAIL (a vacuous scan)
  * a claim site that stops matching           -> FAIL (never a skip)
  * a documented URI with no registry entry     -> FAIL
  * a registered URI missing from the README    -> FAIL (the README is the
    client-facing list; a resource nobody documents is undiscoverable)

Scope limit, stated rather than implied: this gate reads the *declared* registry
from source, the same way check_tool_count.py does. It does not start a server.
Spec 21's conformance gate and the live `resources/list` sequence test cover the
wire behaviour.

Exit: 0 clean; 1 a violation; 2 usage.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

REGISTRY = Path("mcp-server/internal/mcp/resource_registration.go")
README = Path("mcp-server/README.md")
STATUS = Path("docs/status.md")

# A registered resource declaration. The registry uses `uri:` with varying
# alignment, so match the key and a quoted guardrail:// value.
REGISTERED_RE = re.compile(r'^\s*uri:\s*"(guardrail://[a-z0-9-]+)"', re.MULTILINE)

# URIs documented inside a Markdown code span, e.g. `guardrail://config`.
DOCUMENTED_RE = re.compile(r'`(guardrail://[a-z0-9-]+)`')


class GateError(Exception):
    """A structural violation that must fail the gate, not crash it."""


def registered_uris(root: Path) -> list[str]:
    """The declared resource URIs, in declaration order."""
    path = root / REGISTRY
    if not path.is_file():
        raise GateError(f"FAIL missing resource registry {REGISTRY}")
    text = path.read_text(encoding="utf-8")
    uris = REGISTERED_RE.findall(text)
    if not uris:
        # The vacuity guard. A regex that quietly matches nothing must never
        # report success — that is how the F001 tool gate went blind.
        raise GateError(
            f"FAIL zero resources discovered in {REGISTRY}: refusing to pass a vacuous scan"
        )
    duplicates = sorted({u for u in uris if uris.count(u) > 1})
    if duplicates:
        raise GateError(f"FAIL resource(s) registered twice: {', '.join(duplicates)}")
    return uris


def readme_section(text: str) -> str:
    """The README's MCP Resources section, up to the next heading.

    Scoped on purpose: the README also mentions `/ide/quick-reference` as a web
    route, and a gate that flagged that legitimate mention would be wrong and
    would soon be disabled.
    """
    marker = "### MCP Resources"
    start = text.find(marker)
    if start < 0:
        raise GateError(f"FAIL {README} has no '{marker}' section")
    rest = text[start + len(marker):]
    end = rest.find("\n### ")
    body = rest if end < 0 else rest[:end]
    # A second-level heading also terminates the section.
    h2 = body.find("\n## ")
    if h2 >= 0:
        body = body[:h2]
    return body


def check(root: Path) -> int:
    try:
        uris = registered_uris(root)
    except GateError as error:
        print(error)
        return EXIT_VIOLATIONS

    live = len(uris)
    bad = 0

    # 1. Count claim in the README.
    readme_path = root / README
    if not readme_path.is_file():
        print(f"FAIL {README} is missing")
        return EXIT_VIOLATIONS
    readme_text = readme_path.read_text(encoding="utf-8")

    try:
        section = readme_section(readme_text)
    except GateError as error:
        print(error)
        return EXIT_VIOLATIONS

    count_matches = re.findall(r"(\d+)\s+resources exposed via", section)
    if not count_matches:
        print(f"FAIL {README}: the resource section no longer states a count "
              f"('N resources exposed via'); a claim site that stops matching is a failure")
        bad += 1
    else:
        for claimed in count_matches:
            if int(claimed) != live:
                print(f"FAIL {README}: claims {claimed} resources, registry declares {live}")
                bad += 1

    # 2. Status-page count, when it states one.
    status_path = root / STATUS
    if status_path.is_file():
        status_text = status_path.read_text(encoding="utf-8")
        for claimed in re.findall(r"\d+ core tools, (\d+) resources", status_text):
            if int(claimed) != live:
                print(f"FAIL {STATUS}: claims {claimed} resources, registry declares {live}")
                bad += 1

    # 3. Every documented resource URI must exist (the phantom-resource check).
    documented = DOCUMENTED_RE.findall(section)
    if not documented:
        print(f"FAIL {README}: the resource section documents no guardrail:// URIs")
        bad += 1
    for uri in sorted(set(documented)):
        if uri not in uris:
            print(f"FAIL {README}: documents {uri} as an MCP resource, but it is not registered")
            bad += 1

    # 4. Every registered resource must be documented, or it is undiscoverable.
    for uri in uris:
        if uri not in documented:
            print(f"FAIL {README}: registered resource {uri} is not documented")
            bad += 1

    if bad:
        print(f"\n{bad} resource-claim violation(s)")
        return EXIT_VIOLATIONS

    print(f"OK {live} resources: count claims agree and every documented URI is registered")
    return EXIT_OK


def main(argv: list[str]) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root")
    args = parser.parse_args(argv)
    return check(Path(args.repo).resolve())


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
