#!/usr/bin/env python3
"""check_secret_argv.py — Spec 17 R17-08 secret-free-process gate.

Spec 17 R17-08: *"Do not pass credentials via argv, shell-expanded command
strings, healthcheck flags or URLs; replace the Redis `redis-cli -a` probe with
a health method that does not expose a secret in process arguments or logs."*

A secret in argv is exposed to every process on the host: ``/proc/<pid>/cmdline``
and ``docker inspect``/``podman inspect`` render the full command line, and a
crash or verbose error path writes it to logs. A password delivered through the
process *environment* is not in argv, which is the property this requirement
asks for — e.g. ``redis-cli`` reads ``REDISCLI_AUTH`` from the environment
precisely so the password never appears in argv.

Scope is every container manifest that can carry a healthcheck/probe/command:
  - ``docker-compose.yml``
  - ``mcp-server/deploy/podman-compose.yml``
  - ``mcp-server/deploy/docker-compose.example.yml``
  - ``mcp-server/deploy/k8s-deployment.yaml``

Two rule families, applied per command sequence (``healthcheck.test``,
``livenessProbe``/``readinessProbe``/``startupProbe`` ``exec.command``, and a
container ``command:``):

  1. **credential flag with a value** (every sequence) — a token in
     ``{-a, --requirepass, --password, --pass, --auth}`` followed by a value
     (literal or ``${VAR}``), or the ``--flag=value`` form. This is the
     ``redis-cli -a`` / ``redis-server --requirepass`` defect: the value lands in
     the target program's argv.

  2. **secret variable expanded in argv** (healthcheck/probe sequences) — a
     token expanding ``${*_PASSWORD}``/``$*_SECRET``/``$*_TOKEN``/``$*_KEY``/
     ``REDISCLI_AUTH``/``DB_DSN``/``DATABASE_URL``. A probe's argv is inspected
     and logged by orchestrators, so a secret reference there is the named
     defect.

Rule 2 is deliberately **not** applied to a container ``command:``: a command
that *writes a requirepass config file from the environment* (the accepted
argv-free pattern, already used by ``docker-compose.yml``) legitimately contains
the variable reference in its shell script text while the secret value never
enters any program's argv. This limit is named in the spec implementation note,
not hidden.

Fail-closed: a manifest that cannot be read is a violation, not a skip.
Non-vacuity: a scan that finds zero command sequences FAILS (no green from an
empty scope). The allowlist is empty by design; a new entry fails until it is
given a reason.

Exit: 0 clean; 1 a secret-bearing argv token (or a malformed scan); 2 usage.
"""
from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

# Manifests that can carry a healthcheck/probe/command. Missing files are a
# violation (fail-closed).
MANIFESTS = [
    "docker-compose.yml",
    "mcp-server/deploy/podman-compose.yml",
    "mcp-server/deploy/docker-compose.example.yml",
    "mcp-server/deploy/k8s-deployment.yaml",
]

# Flags that take a credential as their value.
SECRET_FLAGS = {"-a", "--requirepass", "--password", "--pass", "--auth"}

# Variable names that hold a credential. Name-based so a plain port/host is not
# flagged; conservative on purpose.
SECRET_VAR_RE = re.compile(
    r"(?:^|_)(?:PASSWORD|PASSWD|SECRET|TOKEN|APIKEY|API_KEY|PRIVATE_KEY|"
    r"CREDENTIAL|CREDENTIALS)(?:$|_)",
    re.IGNORECASE,
)
SECRET_VARS_EXACT = {
    "REDISCLI_AUTH",
    "FALLBACK_API_KEY",
    "OPENAI_API_KEY",
    "DB_DSN",
    "DATABASE_URL",
    "OAP_TEST_PG_DSN",
}
# An expansion of a variable: ``${NAME}`` / ``${NAME:-default}`` / ``$NAME``.
VAR_RE = re.compile(r"\$\{?(?P<name>[A-Za-z_][A-Za-z0-9_]*)(?::-[^}]*)?\}?")

# Deliberate exceptions: "<manifest>:<lineno>:<reason>" -> reason. Empty by design.
ALLOWLIST: dict[str, str] = {}


def is_secret_var(name: str) -> bool:
    return name in SECRET_VARS_EXACT or bool(SECRET_VAR_RE.search(name))


def tokenize(text: str) -> list[str]:
    """Split a command string into tokens, honouring single/double quotes.

    ``redis-cli -a "$REDIS_PASSWORD" ping`` ->
    ['redis-cli', '-a', '$REDIS_PASSWORD', 'ping'].
    """
    tokens: list[str] = []
    cur = ""
    quote = ""
    started = False
    for ch in text:
        if quote:
            if ch == quote:
                quote = ""
            else:
                cur += ch
            continue
        if ch in ("'", '"'):
            quote = ch
            started = True
            continue
        if ch.isspace():
            if started or cur:
                tokens.append(cur)
            cur = ""
            started = False
            continue
        cur += ch
        started = True
    if started or cur:
        tokens.append(cur)
    return tokens


def flag_violations(tokens: list[str]) -> list[str]:
    """Rule 1: a credential flag carrying a value."""
    out: list[str] = []
    idx = 0
    while idx < len(tokens):
        tok = tokens[idx]
        if tok in SECRET_FLAGS:
            nxt = tokens[idx + 1] if idx + 1 < len(tokens) else ""
            out.append(
                f"credential flag '{tok}' with value '{nxt or '<none>'}' in argv"
            )
            idx += 2
            continue
        if "=" in tok:
            flag, _, value = tok.partition("=")
            if flag in SECRET_FLAGS:
                out.append(
                    f"credential flag '{flag}' with value '{value}' in argv"
                )
        idx += 1
    return out


def bare_var_violations(tokens: list[str]) -> list[str]:
    """Rule 2: a secret-named variable expanded in probe argv."""
    out: list[str] = []
    for tok in tokens:
        if tok in SECRET_FLAGS:
            continue
        for m in VAR_RE.finditer(tok):
            if is_secret_var(m.group("name")):
                out.append(
                    f"secret variable '{m.group('name')}' expanded in argv token '{tok}'"
                )
    return out


def _collect_block(lines: list[str], i: int, key_re: re.Pattern[str]):
    """Return (start_lineno, body, next_index) for the value at ``lines[i]``.

    Handles the three compose shapes: an inline JSON list, a scalar on the same
    line, and a block scalar (``>``/``|``) or indented list. Returns
    (0, "", i) when the line is not a ``key:`` value.
    """
    line = lines[i]
    lineno = i + 1
    # Inline JSON list: test: [ ... ]
    m = re.match(rf"^\s*(?:{key_re})\s*:\s*(\[.*\])\s*$", line)
    if m:
        return lineno, m.group(1), i + 1
    # Scalar on the same line, first non-space char not a block indicator.
    m = re.match(rf"^\s*(?:{key_re})\s*:\s*(?P<val>[^>|&\s].*)$", line)
    if m and m.group("val").strip():
        return lineno, m.group("val").strip(), i + 1
    # Block scalar (``command: >`` / ``|``) or indented list.
    if re.match(rf"^\s*(?:{key_re})\s*:\s*(?:[>|][-+]?\s*)?$", line):
        base_indent = len(line) - len(line.lstrip())
        j = i + 1
        buf: list[str] = []
        start = j + 1
        while j < len(lines):
            nxt = lines[j]
            if not nxt.strip():
                j += 1
                continue
            indent = len(nxt) - len(nxt.lstrip())
            if indent <= base_indent:
                break
            item = nxt.strip()
            # A YAML list item is ``- value``; drop only the list dash, never
            # the leading ``--`` of a flag (``- --requirepass`` -> ``--requirepass``).
            if item.startswith("- "):
                item = item[2:].strip()
            elif item == "-":
                item = ""
            buf.append(item)
            j += 1
        if buf:
            return start, " ".join(buf), j
    return 0, "", i


def compose_blocks(text: str) -> list[tuple[str, int, str]]:
    """Return (kind, lineno, command-text) for compose healthcheck/command."""
    blocks: list[tuple[str, int, str]] = []
    lines = text.splitlines()
    i = 0
    while i < len(lines):
        lineno, body, nxt = _collect_block(lines, i, "test")
        if body:
            blocks.append(("probe", lineno, body))
            i = nxt
            continue
        lineno, body, nxt = _collect_block(lines, i, "command")
        if body:
            blocks.append(("command", lineno, body))
            i = nxt
            continue
        i += 1
    return blocks


def k8s_probe_blocks(text: str) -> list[tuple[str, int, str]]:
    """Return ("probe", lineno, command-text) for every k8s exec.command list."""
    blocks: list[tuple[str, int, str]] = []
    lines = text.splitlines()
    i = 0
    while i < len(lines):
        if re.match(r"^\s*command\s*:\s*$", lines[i]):
            base_indent = len(lines[i]) - len(lines[i].lstrip())
            j = i + 1
            buf: list[str] = []
            start = j + 1
            while j < len(lines):
                nxt = lines[j]
                if not nxt.strip():
                    j += 1
                    continue
                indent = len(nxt) - len(nxt.lstrip())
                if indent <= base_indent:
                    break
                item = nxt.strip()
                if item.startswith("- "):
                    item = item[2:].strip()
                elif item == "-":
                    item = ""
                buf.append(item)
                j += 1
            if buf:
                blocks.append(("probe", start, " ".join(buf)))
                i = j
                continue
        i += 1
    return blocks


def collect_blocks(repo: Path) -> tuple[list[tuple[str, str, int, str]], list[str]]:
    """Return (blocks, errors). blocks: (manifest, kind, lineno, command-text)."""
    blocks: list[tuple[str, str, int, str]] = []
    errors: list[str] = []
    for rel in MANIFESTS:
        path = repo / rel
        if not path.is_file():
            errors.append(f"{rel}: manifest not found (fail-closed)")
            continue
        try:
            text = path.read_text(encoding="utf-8")
        except OSError as exc:  # pragma: no cover - unreadable file
            errors.append(f"{rel}: cannot read ({exc})")
            continue
        found = (
            k8s_probe_blocks(text)
            if rel.endswith("k8s-deployment.yaml")
            else compose_blocks(text)
        )
        for kind, lineno, body in found:
            blocks.append((rel, kind, lineno, body))
    return blocks, errors


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Secret-in-argv gate (R17-08)")
    parser.add_argument("--repo", default=".", help="repository root (default: .)")
    args = parser.parse_args(argv)

    repo = Path(args.repo).resolve()
    blocks, errors = collect_blocks(repo)
    violations = list(errors)

    scanned = 0
    for rel, kind, lineno, body in blocks:
        scanned += 1
        tokens = tokenize(body)
        reasons = flag_violations(tokens)
        if kind == "probe":
            reasons += bare_var_violations(tokens)
        for reason in dict.fromkeys(reasons):
            key = f"{rel}:{lineno}:{reason}"
            if key in ALLOWLIST:
                continue
            violations.append(f"{rel}:{lineno}: {reason}")

    if scanned == 0:
        violations.append(
            "no healthcheck/probe/command sequences found across the manifests "
            "(non-vacuous: an empty scan cannot pass)"
        )

    if violations:
        for v in violations:
            print(f"::error::{v}")
        print(f"secret-argv gate: {len(violations)} violation(s); FAIL")
        return EXIT_VIOLATIONS

    print(
        f"secret-argv gate: {scanned} command sequence(s) scanned; "
        f"no credential in argv; OK"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
