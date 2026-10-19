#!/usr/bin/env python3
"""check_charset.py - tracked text must be ASCII-clean (fw-charset-01).

Why this exists here, in this repo
----------------------------------
The same standard was landed in DevGate and reached exactly one repository.
Roger reported the defect again in GR's own spec corpus, which is the corpus
this repository ships to consumers. A gate that exists in a sibling repo does
not police this one.

The concrete defect (twice now): U+00A7 SECTION SIGN is a legitimate section
marker in American legal prose, but several clients render it as a dollar-ish
glyph, so `spec 12 <sect> 3.2` reads as `spec 12 $3.2`. The same class covers
emoji used as priority labels, arrows and triangles used as diagram glyphs,
curly quotes and ellipses. None of them is English text, and none of them
renders the same in a terminal, a diff view, a log pipeline and a tokenizer.

Three properties, each fail-closed
----------------------------------
1. **ASCII standard.** A tracked text file may contain only 0x00-0x7F, except
   codepoints on the reviewed, reason-carrying allowlist. The allowlist may
   NARROW the standard but never disable it: every row needs a reason, a
   malformed row is a hard error, and `deny` always beats `allow`. U+00A7 is
   denied by name so a later edit cannot silently reintroduce the defect.

2. **Strict scopes.** A named path set may be declared zero-tolerance. The
   ratchet baseline may NOT grandfather a file inside a strict scope: those
   are the files the standard exists to protect, and "the corpus that keeps
   regressing is the corpus that is exempt" is the failure this closes.

3. **Substitution post-condition.** Replacing a glyph per-occurrence turns a
   doubled source token into a doubled replacement word: two section signs
   before a number list become the same marker written twice, separated by
   a space. The output is pure ASCII, so a codepoint check
   is blind to it by construction. The gate that PRESCRIBES the substitution
   also rejects its degenerate output, and that check is never grandfathered:
   an artifact is a defect, not legacy debt.

Non-vacuity: a scan that reads zero files FAILS. A gate that finds nothing
because it looked nowhere is worse than no gate.

Exit: 0 clean; 1 violations; 2 usage.
"""
from __future__ import annotations

import argparse
import json
import re
import subprocess
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

POLICY = Path(".guardrails") / "charset-allowlist.json"
BASELINE = Path(".guardrails") / "charset-baseline.json"

# The gate's own policy documents are configuration, not content. Excluded
# from the scan so the zero-file guard below measures the actual corpus: a
# repository holding nothing but its allowlist must still FAIL the vacuity
# check. Letting the config count as "one file scanned" would make that guard
# true by construction, which is the vacuity class this repo keeps closing.
GATE_OWN_FILES = frozenset({POLICY.as_posix(), BASELINE.as_posix()})

# First-party text by extension or basename. Binary and vendored trees are
# never text; scope is applied separately.
TEXT_EXTS = frozenset({
    ".md", ".markdown", ".txt", ".rst",
    ".py", ".go", ".rs", ".ts", ".tsx", ".js", ".jsx", ".mjs", ".cjs",
    ".sh", ".bash", ".zsh", ".bat", ".cmd", ".ps1",
    ".yml", ".yaml", ".json", ".jsonl", ".toml", ".ini", ".cfg", ".env",
    ".sql", ".proto", ".html", ".css", ".hbs", ".xml", ".csv",
})
TEXT_BASENAMES = frozenset({
    ".gitignore", ".gitattributes", ".editorconfig", "dockerfile",
    "makefile", "license", "notice",
})

# Directories that are never first-party text. Declared here rather than
# loaded from a contract file: this repo has no DevGate submodule, and an
# unreadable scope contract must not be able to widen a green.
SKIP_DIRS = frozenset({
    ".git", ".github/ISSUE_TEMPLATE", "node_modules", "vendor", "target",
    "dist", "build", "out", "__pycache__", ".venv", "venv", ".coverage",
    ".claude", ".cursor", ".crew", "worktrees",
})

ASCII_REPLACEMENT = {
    0x00A7: "Sec.",
    0x2013: "-", 0x2014: "-", 0x2012: "-",
    0x2018: "'", 0x2019: "'", 0x201C: '"', 0x201D: '"',
    0x2026: "...",
    0x2192: "->", 0x2190: "<-", 0x2194: "<->", 0x21C4: "<->",
    0x25B6: ">", 0x25BC: "v", 0x25C4: "<", 0x25B2: "^",
    0x00D7: "x", 0x00A0: " ",
    0x1F534: "Critical", 0x1F7E1: "Important", 0x1F7E2: "Nice-to-have",
    0x26A0: "Gap",
}

# Substitution artifacts. Literal ASCII; never grandfathered.
SUBSTITUTION_ARTIFACTS = (
    (re.compile(r"\bSec\.\s+Sec\."),
     "doubled section marker after U+00A7 substitution"),
    (re.compile(r"\b(Section|Figure|Table|Chapter|Clause|Appendix|Article)"
                r"\.\s+\1\."),
     "doubled cross-reference token"),
    (re.compile(r"\.\.\.\.+"),
     "doubled ellipsis after U+2026 substitution"),
    # The substitution has a second degenerate output, found in this same pass:
    # replacing the glyph where the source wrote the marker directly against a
    # digit produced the replacement word fused to the number, with no
    # separating space. It reads as a hostname fragment, not a cross-reference.
    # A per-glyph replace has no post-condition on its OUTPUT, so the gate that
    # prescribes the replacement must reject this shape too. Written from parts
    # rather than literally: a gate that quotes its own forbidden output becomes
    # its own first hit, which is how this rule was caught the first time.
    (re.compile(r"\bSec\.[0-9]"),
     "missing space after Sec. substitution"),
)


def _fail(msg: str) -> "NoReturn":  # type: ignore[name-defined]
    print(f"check-charset: FAIL - {msg}", file=sys.stderr)
    sys.exit(EXIT_USAGE)


def _codepoint(value) -> int:
    if isinstance(value, int):
        return value
    try:
        return int(str(value), 0)
    except (TypeError, ValueError):
        _fail(f"not a codepoint: {value!r}")


def _expand(rows, kind: str) -> "tuple[set[int], set[str]]":
    """Return (codepoints, glob-ish path selectors) from a policy list."""
    points: set[int] = set()
    paths: set[str] = set()
    for row in rows or []:
        if not isinstance(row, dict):
            _fail(f"{kind} row must be an object: {row!r}")
        reason = row.get("reason")
        if not isinstance(reason, str) or not reason.strip():
            _fail(f"{kind} row without a reason: {row!r}")
        if "codepoint" in row:
            points.add(_codepoint(row["codepoint"]))
        elif "range" in row:
            lo, hi = row["range"]
            lo, hi = _codepoint(lo), _codepoint(hi)
            if hi < lo:
                _fail(f"{kind} range inverted: {row!r}")
            points.update(range(lo, hi + 1))
        elif "path" in row:
            paths.add(str(row["path"]))
        else:
            _fail(f"{kind} row needs codepoint, range or path: {row!r}")
    return points, paths


def load_policy(root: Path):
    path = root / POLICY
    if not path.is_file():
        _fail(f"allowlist missing at {POLICY} "
              "(a gate that bans nothing is worse than no gate)")
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        _fail(f"allowlist unreadable: {exc}")
    if not isinstance(data, dict):
        _fail("allowlist must be a JSON object")

    allowed, strict = _expand(data.get("allow"), "allow")
    denied, _ = _expand(data.get("deny"), "deny")
    strict_from_deny, strict_paths = _expand(data.get("strict_zero"), "strict_zero")
    strict |= strict_from_deny

    conflict = allowed & denied
    if conflict:
        _fail("codepoints both allowed and denied: "
              + ", ".join(f"U+{c:04X}" for c in sorted(conflict)))
    return allowed, denied, strict, strict_paths


def load_baseline(root: Path) -> dict[str, int]:
    path = root / BASELINE
    if not path.is_file():
        return {}
    try:
        data = json.loads(path.read_text(encoding="utf-8"))
    except (OSError, json.JSONDecodeError) as exc:
        _fail(f"baseline unreadable: {exc}")
    files = data.get("files")
    if files is None:
        return {}
    if not isinstance(files, dict):
        _fail("baseline 'files' must be an object")
    out: dict[str, int] = {}
    for rel, n in files.items():
        try:
            out[str(rel)] = int(n)
        except (TypeError, ValueError):
            _fail(f"baseline row {rel!r} is not an integer")
    return out


def tracked_files(root: Path) -> list[str]:
    res = subprocess.run(["git", "-C", str(root), "ls-files", "-z"],
                         capture_output=True, text=True, encoding="utf-8",
                         errors="replace")
    if res.returncode != 0:
        _fail(f"git ls-files failed: {res.stderr.strip()}")
    return [f for f in res.stdout.split("\0") if f]


def is_text(rel: str) -> bool:
    p = Path(rel)
    return p.suffix.lower() in TEXT_EXTS or p.name.lower() in TEXT_BASENAMES


def in_scope(rel: str) -> bool:
    if rel in GATE_OWN_FILES:
        return False
    parts = set(Path(rel).parts)
    return not (parts & SKIP_DIRS)


def is_strict(rel: str, strict_paths: set[str]) -> bool:
    return any(rel == p or rel.startswith(p.rstrip("/") + "/")
               for p in strict_paths)


def scan(root: Path, allowed, denied, strict, strict_paths,
         baseline: dict[str, int], max_report: int, use_baseline: bool) -> int:
    files = [f for f in tracked_files(root) if is_text(f) and in_scope(f)]
    if not files:
        print("check-charset: FAIL - zero text files scanned (no vacuous green)",
              file=sys.stderr)
        return EXIT_VIOLATIONS

    violations: list[tuple[str, int, int, int, str]] = []
    artifacts: list[tuple[str, int, int, str, str]] = []
    strict_hits: list[tuple[str, int, int, int, str]] = []
    per_file: dict[str, int] = {}

    for rel in files:
        try:
            text = (root / rel).read_text(encoding="utf-8")
        except OSError:
            continue
        except UnicodeDecodeError as exc:
            violations.append((rel, 1, 1, 0xFFFD, f"not valid UTF-8: {exc}"))
            per_file[rel] = per_file.get(rel, 0) + 1
            continue

        zero_tolerance = is_strict(rel, strict_paths)
        for lineno, line in enumerate(text.splitlines(), 1):
            for pat, why in SUBSTITUTION_ARTIFACTS:
                m = pat.search(line)
                if m:
                    artifacts.append((rel, lineno, m.start() + 1, m.group(0), why))
            for col, ch in enumerate(line, 1):
                cp = ord(ch)
                if cp <= 0x7F:
                    continue
                if cp in denied or cp not in allowed:
                    violation = (rel, lineno, col, cp, ch)
                    violations.append(violation)
                    per_file[rel] = per_file.get(rel, 0) + 1
                    if zero_tolerance:
                        strict_hits.append(violation)

    # 1. Substitution artifacts. Checked first and never grandfathered: the
    #    baseline exists for legacy debt, not for defects this standard makes.
    if artifacts:
        print(f"check-charset: FAIL - {len(artifacts)} substitution artifact(s) "
              f"(fw-charset-01); replacing each glyph independently duplicated a "
              f"word, and the output is pure ASCII so a codepoint check cannot "
              f"see it")
        for rel, lineno, col, shown, why in artifacts[:max_report]:
            print(f"  {rel}:{lineno}:{col}  {shown!r}  ({why})")
        print("  fix: collapse the doubled token BEFORE substituting. Two "
              "section signs before a number list become 'Sec. 3 and Sec. 5', "
              "never the doubled-word form.")
        return EXIT_VIOLATIONS

    # 2. Strict scopes. Zero tolerance: the baseline may not exempt them.
    if strict_hits:
        print(f"check-charset: FAIL - {len(strict_hits)} non-ASCII codepoint(s) "
              f"inside a strict-zero scope (fw-charset-01); the ratchet "
              f"baseline does not apply to these paths")
        for rel, lineno, col, cp, ch in strict_hits[:max_report]:
            hint = ASCII_REPLACEMENT.get(cp)
            tail = f"  -> {hint!r}" if hint else ""
            shown = ch if ch.isprintable() else "?"
            print(f"  {rel}:{lineno}:{col}  U+{cp:04X}  {shown!r}{tail}")
        return EXIT_VIOLATIONS

    # 3. Ratchet over the remaining tree.
    if use_baseline:
        over = {rel: n for rel, n in per_file.items()
                if n > baseline.get(rel, 0)}
    else:
        over = dict(per_file)

    if over:
        print(f"check-charset: FAIL - {len(over)} file(s) exceed their charset "
              f"baseline (fw-charset-01); "
              f"{sum(over.values())} non-ASCII occurrence(s) not grandfathered")
        for rel in sorted(over)[:max_report]:
            for rel2, lineno, col, cp, ch in violations:
                if rel2 != rel:
                    continue
                hint = ASCII_REPLACEMENT.get(cp)
                tail = f"  -> {hint!r}" if hint else ""
                shown = ch if ch.isprintable() else "?"
                print(f"  {rel}:{lineno}:{col}  U+{cp:04X}  {shown!r}{tail}")
                break
        print("  fix: replace the glyph with its ASCII equivalent; add a "
              "reasoned allowlist row only for reviewed typography that must "
              "stay non-ASCII. A new file has no baseline row, so it starts "
              "clean.")
        return EXIT_VIOLATIONS

    improving = [rel for rel, n in per_file.items() if n < baseline.get(rel, 0)]
    improving += [rel for rel in baseline if rel not in per_file]
    if improving and use_baseline:
        print(f"check-charset: note - {len(improving)} file(s) improved below "
              f"baseline; tighten {BASELINE}: " + ", ".join(sorted(improving)[:8]))

    grandfathered = sum(baseline.get(rel, 0) for rel in per_file)
    print(f"check-charset: OK - {len(files)} text file(s) scanned; "
          f"{len(strict_paths)} strict-zero scope(s); "
          f"{grandfathered} grandfathered occurrence(s); "
          f"{len(allowed)} allowed / {len(denied)} denied codepoints")
    return EXIT_OK


def write_baseline(root: Path, allowed, denied, strict_paths) -> int:
    per: dict[str, int] = {}
    for rel in [f for f in tracked_files(root) if is_text(f) and in_scope(f)]:
        if is_strict(rel, strict_paths):
            continue
        try:
            text = (root / rel).read_text(encoding="utf-8")
        except (OSError, UnicodeDecodeError):
            continue
        n = sum(1 for line in text.splitlines() for ch in line
                if ord(ch) > 0x7F and ord(ch) not in allowed)
        if n:
            per[rel] = n
    doc = {
        "api_version": "guardrails.charset-baseline/v1",
        "note": [
            "fw-charset-01 ratchet: grandfathered non-ASCII counts for files",
            "outside every strict-zero scope. May only DECREASE. Fix files as",
            "you touch them, then rerun --write-baseline to tighten.",
            "Strict-zero paths are deliberately absent: a baseline row there",
            "would exempt the corpus the standard exists to protect.",
        ],
        "files": dict(sorted(per.items())),
    }
    (root / BASELINE).write_text(json.dumps(doc, indent=1) + "\n",
                                 encoding="utf-8")
    print(f"check-charset: wrote baseline - {len(per)} file(s), "
          f"{sum(per.values())} occurrence(s)")
    return EXIT_OK


def main() -> int:
    ap = argparse.ArgumentParser(
        prog="check_charset",
        description="tracked text must be ASCII-clean (fw-charset-01)")
    ap.add_argument("--repo", default=".")
    ap.add_argument("--max-report", type=int, default=40)
    ap.add_argument("--list", action="store_true",
                    help="print every violation (no cap)")
    ap.add_argument("--no-baseline", action="store_true",
                    help="ignore the ratchet baseline (strict, zero tolerance)")
    ap.add_argument("--write-baseline", action="store_true",
                    help="regenerate the ratchet baseline and exit")
    args = ap.parse_args()

    root = Path(args.repo).resolve()
    allowed, denied, strict, strict_paths = load_policy(root)
    if args.write_baseline:
        return write_baseline(root, allowed, denied, strict_paths)
    baseline = {} if args.no_baseline else load_baseline(root)
    return scan(root, allowed, denied, strict, strict_paths, baseline,
                10 ** 9 if args.list else args.max_report,
                use_baseline=not args.no_baseline)


if __name__ == "__main__":
    sys.exit(main())
