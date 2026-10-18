#!/usr/bin/env python3
"""check_policy_pack_lock.py — Spec 07 policy-pack resolution + lock-record gate.

Spec 07 (`docs/specs/07-versioned-policy-pack-governance.md`) requires that a
consumer "MUST be able to inspect the resolved policy set without executing
policy text" (4.1.5), that resolution be deterministic (4.1.2), that a lock
record carry identity + digest + resolver version + overlay digest + resolution
time (4.1.3), that pack schemas "reject unknown critical fields and invalid
types" (4.3.2), and that "CI verifies the checked-in lock record matches the
declared pack revisions and content digests" (5.8).

Why this file exists
--------------------
`scripts/validate-policy-packs.mjs` implemented most of the *lock* half, and
**no workflow ever ran it.** The repository's own audit (`docs/platform-current-state.md`)
said so in plain text: "No active workflow ... runs `validate-policy-packs.mjs`."
A verifier nobody invokes is a comment.

It also failed open in the way that matters: given a tree with **zero** packs and
an empty `LOCK.json`, it printed `lock ok` and exited **0**. The loop that builds
the expected lock record simply iterated over nothing, so "the lock matches all
packs" was vacuously true. That is the same failure class as the F001 tool-count
gate and the R19-11 docs gate in this repository: a gate that reports success
for the set it never examined.

So this gate asserts, fail-closed and non-vacuously:

  1. **Discovery is not vacuous.** Zero `*.pack.json` files, or zero known tools,
     is a FAILURE — never "nothing to check".
  2. **Each pack is schema-valid as data.** Unknown top-level fields (other than
     an explicit ``x-`` annotation prefix), bad types, non-semver version, bad
     enforcement class, missing ``evidence`` on a non-advisory rule, a mutable
     ``on_unknown``, duplicate rule ids, and a ``pack`` name that does not match
     its file path are all violations. Nothing here executes pack text.
  3. **Every rule names a real tool.** A policy rule referencing a tool the
     server does not register is a rule that can never fire.
  4. **The checked-in lock record matches the bytes on disk.** Present, complete,
     free of stale entries, and equal on every identity field for every pack.
  5. **Resolution is deterministic.** The effective policy digest is recomputed
     from the resolved set; it must equal the recorded one, and re-deriving it in
     the same run must be stable.

  6. **The declared dependency graph is resolvable (R07-4.1.4).** Every
     ``depends_on`` entry names a pack in the resolved set, the resolved version
     satisfies the declared range, and the graph is acyclic. Each failure names
     the dependency path, because "resolution failed" without the path is not
     actionable. The runtime consumer runs the same walk, so the gate and the
     server cannot disagree about the graph.

Named limits (stated, not implied): this gate verifies in-repo packs against an
in-repo lock. It does not fetch remote packs and does not verify signatures
(R07-4.3.4) — a signature check runs in the consumer only when a trusted-key set
is configured. It does not prove any runtime engine consumes the resolved set
(R07-4.4). Those stay open, and the spec stays PARTIAL for them.

Usage:
  python scripts/check_policy_pack_lock.py --repo .
  python scripts/check_policy_pack_lock.py --repo . --relock

Exit: 0 clean; 1 violation; 2 usage.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import re
import sys
from dataclasses import dataclass
from datetime import datetime, timezone
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

PACK_ROOT = Path("policy-packs")
LOCK_PATH = PACK_ROOT / "LOCK.json"
TOOLS_PATH = Path("site/src/data/tools.json")

RESOLVER_VERSION = "1"
SEMVER_RE = re.compile(r"^\d+\.\d+\.\d+$")
ENFORCEMENTS = ("advisory", "checked", "blocking")
# Spec 07 R07-4.2.1 — declared precedence tiers, strongest first. The tier is
# declared in the pack and must agree with the pack's path prefix, so a domain
# policy cannot be filed as core (or the reverse) by moving a file.
TIERS = ("core", "domain", "overlay")
# Top-level fields a pack may carry. Anything else is an unknown *critical*
# field (R07-4.3.2) and fails; ``x-`` prefixed keys are explicit non-critical
# annotations and do not participate in resolution.
PACK_FIELDS = ("pack", "version", "tier", "rules", "seed", "depends_on")
RULE_FIELDS = ("id", "law", "enforcement", "tool", "on_unknown", "evidence")
DEP_FIELDS = ("pack", "range")
ANNOTATION_PREFIX = "x-"


@dataclass(frozen=True)
class Violation:
    path: str
    message: str

    def render(self) -> str:
        return f"{self.path}: {self.message}"


def _load_json(path: Path):
    """Return (data, error). Never raises: a parse failure is a violation."""
    try:
        raw = path.read_bytes()
    except OSError as exc:
        return None, f"unreadable ({exc.__class__.__name__})"
    try:
        return json.loads(raw.decode("utf-8")), None
    except (UnicodeDecodeError, json.JSONDecodeError):
        return None, "not valid UTF-8 JSON"


def discover_packs(root: Path) -> list[Path]:
    pack_root = root / PACK_ROOT
    if not pack_root.is_dir():
        return []
    return sorted(p for p in pack_root.rglob("*.pack.json") if p.is_file())


def rel_path(root: Path, path: Path) -> str:
    return path.relative_to(root).as_posix()


def expected_pack_name(root: Path, path: Path) -> str:
    rel = rel_path(root, path)
    inner = rel[len(PACK_ROOT.as_posix()) + 1:]
    return inner[: -len(".pack.json")]


def load_known_tools(root: Path):
    data, err = _load_json(root / TOOLS_PATH)
    if err:
        return set(), f"{TOOLS_PATH.as_posix()}: {err}"
    tools = data.get("tools") if isinstance(data, dict) else None
    if not isinstance(tools, list):
        return set(), f"{TOOLS_PATH.as_posix()}: 'tools' is not a list"
    names = {t for t in tools if isinstance(t, str) and t}
    return names, None


def _is_nonempty_str(value) -> bool:
    return isinstance(value, str) and value.strip() != ""


def validate_pack(root: Path, path: Path, known_tools: set[str]) -> list[Violation]:
    """Schema-validate one pack as inert data. Returns violations, never raises."""
    where = rel_path(root, path)
    out: list[Violation] = []

    data, err = _load_json(path)
    if err:
        return [Violation(where, err)]
    if not isinstance(data, dict):
        return [Violation(where, "pack root is not an object")]

    for key in data:
        if key not in PACK_FIELDS and not key.startswith(ANNOTATION_PREFIX):
            out.append(Violation(where, f"unknown critical field '{key}' (R07-4.3.2)"))

    name = data.get("pack")
    if not _is_nonempty_str(name):
        out.append(Violation(where, "'pack' must be a non-empty string"))
    else:
        expected = expected_pack_name(root, path)
        if name != expected:
            out.append(
                Violation(where, f"'pack' is '{name}' but its path implies '{expected}'")
            )

    version = data.get("version")
    if not isinstance(version, str) or not SEMVER_RE.match(version):
        out.append(Violation(where, f"'version' must be semver MAJOR.MINOR.PATCH, got {version!r}"))

    tier = data.get("tier")
    if tier not in TIERS:
        out.append(Violation(where, f"'tier' must be one of {TIERS}, got {tier!r} (R07-4.2.1)"))
    elif isinstance(name, str) and name:
        implied = name.split("/", 1)[0]
        if tier != implied:
            out.append(
                Violation(
                    where,
                    f"'tier' is {tier!r} but the pack name implies {implied!r}"
                    " — precedence must be visible in the declaration, not only the path (R07-4.2.1)",
                )
            )

    seed = data.get("seed", False)
    if not isinstance(seed, bool):
        out.append(Violation(where, "'seed' must be a boolean when present"))

    rules = data.get("rules")
    if not isinstance(rules, list):
        out.append(Violation(where, "'rules' must be a list"))
        return out
    if not rules:
        # R07-4.2.4: an empty pack is only valid when explicitly marked a seed,
        # and a seed must not be represented as domain coverage.
        if not seed:
            out.append(
                Violation(where, "empty 'rules' requires 'seed': true (R07-4.2.4)")
            )

    seen_ids: set[str] = set()
    for index, rule in enumerate(rules):
        at = f"{where}: rules[{index}]"
        if not isinstance(rule, dict):
            out.append(Violation(at, "rule is not an object"))
            continue
        for key in rule:
            if key not in RULE_FIELDS and not key.startswith(ANNOTATION_PREFIX):
                out.append(Violation(at, f"unknown critical field '{key}'"))

        rule_id = rule.get("id")
        if not _is_nonempty_str(rule_id):
            out.append(Violation(at, "'id' must be a non-empty string"))
        elif rule_id in seen_ids:
            out.append(Violation(at, f"duplicate rule id '{rule_id}'"))
        else:
            seen_ids.add(rule_id)

        enforcement = rule.get("enforcement")
        if enforcement not in ENFORCEMENTS:
            out.append(
                Violation(at, f"'enforcement' must be one of {ENFORCEMENTS}, got {enforcement!r}")
            )
        elif enforcement != "advisory" and not _is_nonempty_str(rule.get("evidence")):
            # A blocking/checked claim with no test record is exactly the
            # "advisory dressed as enforcement" failure this schema forbids.
            out.append(Violation(at, f"{enforcement} rule requires a non-empty 'evidence'"))

        tool = rule.get("tool")
        if not _is_nonempty_str(tool):
            out.append(Violation(at, "'tool' must be a non-empty string"))
        elif known_tools and tool not in known_tools:
            out.append(Violation(at, f"tool '{tool}' is not registered by the server"))

        if rule.get("on_unknown") != "stop":
            out.append(
                Violation(at, f"'on_unknown' must be 'stop', got {rule.get('on_unknown')!r}")
            )

    # R07-4.1.4 — dependency DECLARATIONS are validated here; existence,
    # compatibility and cycles need the whole set and are checked in
    # check_dependency_graph. A malformed edge never reaches that walk.
    depends = data.get("depends_on", [])
    if not isinstance(depends, list):
        out.append(Violation(where, "'depends_on' must be a list when present"))
        return out
    seen_deps: set[str] = set()
    for index, dep in enumerate(depends):
        at = f"{where}: depends_on[{index}]"
        if not isinstance(dep, dict):
            out.append(Violation(at, "dependency is not an object"))
            continue
        for key in dep:
            if key not in DEP_FIELDS:
                out.append(Violation(at, f"unknown critical field '{key}'"))
        target = dep.get("pack")
        if not _is_nonempty_str(target):
            out.append(Violation(at, "'pack' must be a non-empty string"))
            continue
        if isinstance(name, str) and target == name:
            out.append(Violation(at, f"pack '{target}' depends on itself"))
        if target in seen_deps:
            out.append(Violation(at, f"duplicate dependency on '{target}'"))
        seen_deps.add(target)
        _, range_err = parse_version_range(dep.get("range"))
        if range_err:
            out.append(Violation(at, f"unusable range: {range_err}"))

    return out


SEMVER_RE_PARTS = re.compile(r"^(\d+)\.(\d+)\.(\d+)$")
_RANGE_OPERATORS = (">=", "^", "~")


def parse_version_range(rng):
    """Validate a dependency range; return (operator, (major, minor, patch)).

    Accepted forms are deliberately few: an exact MAJOR.MINOR.PATCH, or one of
    ``>=``, ``^`` (same major, at least) and ``~`` (same major.minor, at least)
    followed by one. Empty, ``*``, ``latest`` and anything else is an ERROR, not
    "any version": a constraint that silently widens accepts a breaking change
    with no review, which is the failure R07-4.1.4 exists to prevent. The Go
    loader implements the same list (parseVersionRange), so the gate and the
    server cannot disagree about what a range means.
    """
    if not isinstance(rng, str) or not rng.strip():
        return None, "no version range declared (R07-4.1.4); write an exact version or >=/^/~ before it"
    trimmed = rng.strip()
    op = "="
    spec = trimmed
    for candidate in _RANGE_OPERATORS:
        if trimmed.startswith(candidate):
            op = candidate
            spec = trimmed[len(candidate):].strip()
            break
    match = SEMVER_RE_PARTS.match(spec)
    if not match:
        return None, f"range {rng!r} is not an exact MAJOR.MINOR.PATCH or one of >=/^/~ followed by one"
    return (op, tuple(int(part) for part in match.groups())), None


def satisfies_version_range(version, rng):
    """Return (ok, error) for version against rng, using the same forms."""
    parsed, err = parse_version_range(rng)
    if err:
        return False, err
    match = SEMVER_RE_PARTS.match(version or "")
    if not match:
        return False, f"declared version {version!r} is not MAJOR.MINOR.PATCH"
    got = tuple(int(part) for part in match.groups())
    op, want = parsed
    if op == "=":
        return got == want, None
    if op == ">=":
        return got >= want, None
    if op == "^":
        return got[0] == want[0] and got >= want, None
    if op == "~":
        return got[0] == want[0] and got[1] == want[1] and got >= want, None
    return False, f"unreachable range operator {op!r}"


def check_dependency_graph(packs):
    """Validate the declared inter-pack graph (R07-4.1.4).

    ``packs`` maps declared pack name -> parsed pack body. Returns violations
    naming the dependency PATH for an absent dependency, an unsatisfied range and
    a cycle. A single-pack set with no declarations is a legal one-node graph:
    the same walk runs and finds nothing to reject, so this is not a check that
    silently skips when the tree is small.
    """
    out: list[Violation] = []
    where = PACK_ROOT.as_posix()

    for name in sorted(packs):
        body = packs[name]
        for dep in body.get("depends_on") or []:
            if not isinstance(dep, dict):
                continue
            target = dep.get("pack")
            if not isinstance(target, str) or target not in packs:
                out.append(
                    Violation(
                        where,
                        f"pack '{name}' depends on '{target}', which is not in the resolved set"
                        f" (dependency path: {name} -> {target})",
                    )
                )
                continue
            version = packs[target].get("version")
            ok, err = satisfies_version_range(version, dep.get("range"))
            if err:
                out.append(Violation(where, f"pack '{name}' dependency '{target}': {err}"))
            elif not ok:
                out.append(
                    Violation(
                        where,
                        f"pack '{name}' requires '{target}' {dep.get('range')} but the resolved"
                        f" set has {version} (dependency path: {name} -> {target})",
                    )
                )

    # Cycle detection with an explicit path so the error names the closing edge.
    state: dict[str, int] = {}
    stack: list[str] = []

    def walk(node: str) -> str | None:
        current = state.get(node, 0)
        if current == 2:
            return None
        if current == 1:
            start = stack.index(node) if node in stack else 0
            return " -> ".join(stack[start:] + [node])
        state[node] = 1
        stack.append(node)
        for dep in packs[node].get("depends_on") or []:
            if not isinstance(dep, dict):
                continue
            target = dep.get("pack")
            if isinstance(target, str) and target in packs:
                cycle = walk(target)
                if cycle:
                    return cycle
        stack.pop()
        state[node] = 2
        return None

    for name in sorted(packs):
        cycle = walk(name)
        if cycle:
            out.append(Violation(where, f"policy pack dependency cycle: {cycle}"))
            break

    return out


def resolve(root: Path, packs: list[Path], known_tools: set[str]):
    """Deterministic resolution: identical bytes -> identical rows.

    Returns (rows, violations, effective_digest). ``resolved_at`` is metadata and
    is deliberately excluded from the effective digest so that re-resolution is
    reproducible (R07-4.1.2 / acceptance 1).
    """
    violations: list[Violation] = []
    rows: dict[str, dict] = {}
    # Declared pack name -> parsed body, for the R07-4.1.4 dependency walk. A
    # pack whose JSON did not parse is absent here, so an edge pointing at it is
    # reported as an absent dependency rather than silently ignored.
    by_name: dict[str, dict] = {}
    duplicate_names: list[Violation] = []
    for path in packs:
        where = rel_path(root, path)
        violations.extend(validate_pack(root, path, known_tools))
        try:
            digest = hashlib.sha256(path.read_bytes()).hexdigest()
        except OSError:
            violations.append(Violation(where, "unreadable while hashing"))
            continue
        data, err = _load_json(path)
        if err:
            continue
        declared = data.get("pack") if isinstance(data, dict) else None
        if isinstance(declared, str) and declared:
            if declared in by_name:
                duplicate_names.append(
                    Violation(where, f"another pack already declares the name '{declared}'"
                                     " — dependency edges would be ambiguous")
                )
            else:
                by_name[declared] = data
        rows[where] = {
            "pack": data.get("pack"),
            "version": data.get("version"),
            "digest": f"sha256:{digest}",
            "resolver_version": RESOLVER_VERSION,
            # No overlay mechanism ships yet; the field is recorded as the empty
            # value rather than omitted, so the record shape cannot drift silently.
            "overlay_digest": "none",
            "resolved_at": datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%SZ"),
        }

    canonical = json.dumps(
        [
            [key, row["pack"], row["version"], row["digest"], row["resolver_version"], row["overlay_digest"]]
            for key, row in sorted(rows.items())
        ],
        separators=(",", ":"),
        sort_keys=False,
    )
    effective_digest = f"sha256:{hashlib.sha256(canonical.encode('utf-8')).hexdigest()}"
    if duplicate_names:
        violations.extend(duplicate_names)
    else:
        violations.extend(check_dependency_graph(by_name))
    return rows, violations, effective_digest


def compare_lock(root: Path, rows: dict[str, dict]) -> list[Violation]:
    where = LOCK_PATH.as_posix()
    data, err = _load_json(root / LOCK_PATH)
    if err:
        return [Violation(where, f"{err} — the lock record must exist (run --relock)")]
    if not isinstance(data, dict):
        return [Violation(where, "lock record is not an object")]

    recorded = data.get("packs")
    if recorded is None:
        return [Violation(where, "'packs' is missing; this is not a lock record this gate can read")]
    if not isinstance(recorded, dict):
        return [Violation(where, "'packs' must be an object keyed by pack path")]

    out: list[Violation] = []
    for key in sorted(set(recorded) - set(rows)):
        out.append(Violation(f"{where}: {key}", "stale entry — no such pack on disk"))
    for key in sorted(set(rows) - set(recorded)):
        out.append(Violation(f"{where}: {key}", "missing entry — pack is not locked"))
    for key in sorted(set(rows) & set(recorded)):
        got = recorded[key]
        want = rows[key]
        if not isinstance(got, dict):
            out.append(Violation(f"{where}: {key}", "entry is not an object"))
            continue
        for field in ("pack", "version", "digest", "resolver_version", "overlay_digest"):
            if got.get(field) != want[field]:
                out.append(
                    Violation(
                        f"{where}: {key}",
                        f"{field} is {got.get(field)!r}, resolved {want[field]!r}"
                        " — pack content changed without a deliberate re-lock",
                    )
                )
        stamp = got.get("resolved_at")
        if not _is_nonempty_str(stamp):
            out.append(Violation(f"{where}: {key}", "'resolved_at' must be a non-empty string"))
        else:
            try:
                datetime.strptime(stamp, "%Y-%m-%dT%H:%M:%SZ")
            except ValueError:
                out.append(Violation(f"{where}: {key}", f"'resolved_at' is not UTC ISO-8601: {stamp!r}"))

    recorded_digest = data.get("effective_digest")
    if not isinstance(recorded_digest, str) or not recorded_digest.startswith("sha256:"):
        out.append(Violation(where, "'effective_digest' missing or malformed"))
    return out


def write_lock(root: Path, rows: dict[str, dict], effective_digest: str) -> None:
    payload = {"resolver_version": RESOLVER_VERSION, "effective_digest": effective_digest, "packs": rows}
    (root / LOCK_PATH).write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")


def run(root: Path, relock: bool = False) -> list[Violation]:
    known_tools, tools_err = load_known_tools(root)
    if tools_err:
        return [Violation(tools_err, "cannot validate rule->tool references")]
    if not known_tools:
        return [Violation(TOOLS_PATH.as_posix(), "zero known tools — refusing to pass vacuously")]

    packs = discover_packs(root)
    if not packs:
        return [
            Violation(
                PACK_ROOT.as_posix(),
                "zero '*.pack.json' files discovered — a policy gate over an empty set proves nothing",
            )
        ]

    rows, violations, effective_digest = resolve(root, packs, known_tools)
    if relock:
        write_lock(root, rows, effective_digest)
        print(f"relocked {len(rows)} pack(s); effective_digest {effective_digest}")
        return violations

    # Determinism check (R07-4.1.2): resolving twice in one run must agree.
    _, _, again = resolve(root, packs, known_tools)
    if again != effective_digest:
        violations.append(
            Violation(PACK_ROOT.as_posix(), "resolution is not deterministic across two passes")
        )

    violations.extend(compare_lock(root, rows))
    recomputed = effective_digest
    lock_data, _ = _load_json(root / LOCK_PATH)
    if isinstance(lock_data, dict):
        recorded = lock_data.get("effective_digest")
        if isinstance(recorded, str) and recorded != recomputed:
            violations.append(
                Violation(
                    LOCK_PATH.as_posix(),
                    f"effective_digest is {recorded}, recomputed {recomputed}",
                )
            )
    return violations


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--repo", default=".", help="repository root (default: .)")
    parser.add_argument("--relock", action="store_true", help="rewrite LOCK.json deliberately")
    args = parser.parse_args(argv)

    root = Path(args.repo).resolve()
    if not root.is_dir():
        print(f"check_policy_pack_lock: not a directory: {root}", file=sys.stderr)
        return EXIT_USAGE

    violations = run(root, relock=args.relock)
    for violation in violations:
        print(f"FAIL {violation.render()}", file=sys.stderr)
    if violations:
        print(f"POLICY PACK LOCK: {len(violations)} violation(s)", file=sys.stderr)
        return EXIT_VIOLATIONS
    print("POLICY PACK LOCK: OK")
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
