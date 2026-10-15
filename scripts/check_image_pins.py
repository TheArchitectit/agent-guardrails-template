#!/usr/bin/env python3
"""check_image_pins.py — Spec 17 R17-11 / Spec 19 R19-15 image-digest gate.

Spec 19 R19-15: *"Required workflow actions MUST use reviewed full commit SHA
references, and release/base images MUST use immutable digests. A test validates
workflow references and the built image's recorded/executed digest at the exact
commit; a mutable tag, unknown action SHA, omitted identity, or unreviewed
update is FAIL."*

Spec 17 R17-11: *"Pin build/test actions by immutable full commit SHA and
production base/service images by immutable digest ... not a `latest`/floating
tag as release proof. ... verify the pinned image digest at promotion."*

This is the **image half** of the two requirements. The action half is enforced
by ``scripts/check_action_pins.py``.

A tag is mutable: ``redis:7-alpine`` resolves to whatever that tag points at
today, so a rebuilt or compromised upstream image silently changes what runs in
production. A ``@sha256:<64 hex>`` digest is immutable, which is the property
the requirement asks for.

Scope is every container manifest that names an external image:
  - ``docker-compose.yml``
  - ``mcp-server/deploy/podman-compose.yml``
  - ``mcp-server/deploy/docker-compose.example.yml``
  - ``mcp-server/deploy/k8s-deployment.yaml``
  - ``mcp-server/deploy/Dockerfile`` (``FROM`` base images)

Ref forms judged:
  ``name:tag@sha256:<64hex>``  digest-pinned AND present in ``image-pins.json``
                               with the same digest — allowed.
  ``name:tag``                 a mutable tag, no digest — VIOLATION.
  ``name@sha256:<64hex>``      digest-pinned but the tag/name is not recorded in
                               the registry — VIOLATION (an unreviewed pin).
  a first-party image built by this repo (see ALLOWLIST) — allowed; its digest
                               is recorded at promotion, not before the build.

The registry ``image-pins.json`` at the repo root is the reviewed record: each
external ``name:tag`` maps to the digest that was resolved and re-verified. The
gate never needs the network — it checks the manifests against the committed
record, so a drift between a manifest pin and the reviewed digest FAILS.

Fail-closed: a manifest that cannot be read is a violation, not a skip.
Non-vacuity: a scan that finds zero external image refs FAILS (no green from an
empty scope). The allowlist carries a reason per entry; a new entry fails until
it is given one.

Exit: 0 clean; 1 an unpinned/mismatched/unreviewed image ref (or a malformed
scan); 2 usage.
"""
from __future__ import annotations

import argparse
import json
import re
import sys
from pathlib import Path

EXIT_OK = 0
EXIT_VIOLATIONS = 1
EXIT_USAGE = 2

REGISTRY_FILE = Path("image-pins.json")

# Manifests that name external images. Missing files are a violation (fail-closed).
MANIFESTS = [
    "docker-compose.yml",
    "mcp-server/deploy/podman-compose.yml",
    "mcp-server/deploy/docker-compose.example.yml",
    "mcp-server/deploy/k8s-deployment.yaml",
    "mcp-server/deploy/Dockerfile",
]

# First-party images built by this repository, keyed by canonical name -> reason.
# A first-party image cannot carry a digest before it is built; R17-11 records
# its digest at promotion ("verify the pinned image digest at promotion"), so
# the manifest template (a version variable) is correct here. Empty of
# third-party images by design; a new entry fails until it is given a reason.
ALLOWLIST: dict[str, str] = {
    "docker.io/library/guardrail-mcp": (
        "first-party image built from this repo's mcp-server/deploy/Dockerfile; "
        "its digest is recorded at promotion (R17-11), not before the build"
    ),
}

SHA256_RE = re.compile(r"^sha256:[0-9a-f]{64}$")
# Shell variable expansions inside a compose/k8s image ref: ``${VERSION:-latest}``
# or ``$VERSION``. They carry no digest, so the ref is judged on its literal name
# with the expansion removed (a first-party build is allowlisted by name).
VAR_BRACE_RE = re.compile(r"\$\{[^}]*\}")
VAR_PLAIN_RE = re.compile(r"\$[A-Za-z_][A-Za-z0-9_]*")
# A compose/k8s `image:` value, or a Dockerfile `FROM` image.
IMAGE_LINE_RE = re.compile(r"^\s*image:\s*(?P<ref>[^\s#]+)")
FROM_LINE_RE = re.compile(
    r"^\s*FROM\s+(?:--platform=\S+\s+)?(?P<ref>[^\s]+)(?:\s+AS\s+\S+)?\s*$",
    re.IGNORECASE,
)
# A `build:` section marks the local image built from the sibling Dockerfile.
BUILD_LINE_RE = re.compile(r"^\s*build:\s*$|^\s*build:\s*\S+")


def strip_vars(ref: str) -> str:
    """Remove shell variable expansions from an image ref.

    ``guardrail-mcp:${VERSION:-latest}`` -> ``guardrail-mcp:`` so the parser sees
    a name with an empty tag rather than splitting on the colon inside ``${...}``.
    """
    return VAR_PLAIN_RE.sub("", VAR_BRACE_RE.sub("", ref))


def canonical(ref: str) -> tuple[str, str, str]:
    """Split an image ref into (canonical_name, tag, digest).

    ``redis:7-alpine``            -> ("docker.io/library/redis", "7-alpine", "")
    ``docker.io/library/redis:7`` -> ("docker.io/library/redis", "7", "")
    ``gcr.io/distroless/static:nonroot`` -> ("gcr.io/distroless/static", "nonroot", "")
    ``name@sha256:..``            -> digest set, tag "".
    """
    name, _, digest = strip_vars(ref).partition("@")
    tag = ""
    # A colon after the final slash is a tag, not a registry port.
    last = name.rsplit("/", 1)[-1]
    if ":" in last:
        name, _, tag = name.rpartition(":")
    first = name.split("/", 1)[0]
    if "." in first or ":" in first or first == "localhost":
        canonical_name = name
    else:
        repo = name if "/" in name else "library/" + name
        canonical_name = "docker.io/" + repo
    return canonical_name, tag, digest


def read_registry(repo: Path) -> dict[str, str]:
    """Return {canonical_name:tag -> digest} from image-pins.json."""
    path = repo / REGISTRY_FILE
    if not path.is_file():
        raise FileNotFoundError(f"registry {REGISTRY_FILE} not found")
    data = json.loads(path.read_text(encoding="utf-8"))
    images = data.get("images")
    if not isinstance(images, dict) or not images:
        raise ValueError(f"{REGISTRY_FILE} has no non-empty 'images' map")
    out: dict[str, str] = {}
    for key, entry in images.items():
        digest = entry.get("digest") if isinstance(entry, dict) else entry
        if not isinstance(digest, str) or not SHA256_RE.match(digest):
            raise ValueError(f"{REGISTRY_FILE}: '{key}' has no valid sha256 digest")
        out[key] = digest
    return out


def collect_refs(repo: Path) -> tuple[list[tuple[str, str, str, int]], list[str]]:
    """Return (refs, errors).

    refs: list of (manifest, ref, kind, lineno). kind is "image" or "from".
    errors: fail-closed problems (missing/unreadable manifest).
    """
    refs: list[tuple[str, str, str, int]] = []
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
        for lineno, line in enumerate(text.splitlines(), start=1):
            m = IMAGE_LINE_RE.match(line)
            if m:
                refs.append((rel, m.group("ref"), "image", lineno))
                continue
            m = FROM_LINE_RE.match(line)
            if m:
                refs.append((rel, m.group("ref"), "from", lineno))
    return refs, errors


def is_local_build(ref: str, name: str) -> bool:
    """True when the image is the first-party build (allowlisted by name)."""
    return name in ALLOWLIST


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description="Image-digest pin gate (R17-11/R19-15)")
    parser.add_argument("--repo", default=".", help="repository root (default: .)")
    args = parser.parse_args(argv)

    repo = Path(args.repo).resolve()

    try:
        registry = read_registry(repo)
    except (FileNotFoundError, ValueError) as exc:
        print(f"::error::{exc}")
        return EXIT_VIOLATIONS

    refs, errors = collect_refs(repo)
    violations = list(errors)

    external = 0
    for rel, ref, kind, lineno in refs:
        name, tag, digest = canonical(ref)

        if is_local_build(ref, name):
            continue

        external += 1
        key = f"{name}:{tag or 'latest'}"

        if not digest:
            violations.append(
                f"{rel}:{lineno}: '{ref}' has no @sha256 digest "
                f"(mutable tag; R17-11/R19-15 require an immutable digest)"
            )
            continue
        if not SHA256_RE.match(digest):
            violations.append(
                f"{rel}:{lineno}: '{ref}' has a malformed digest (need sha256:<64 hex>)"
            )
            continue
        if key not in registry:
            violations.append(
                f"{rel}:{lineno}: '{ref}' is not in {REGISTRY_FILE} "
                f"(unreviewed pin; add '{key}')"
            )
            continue
        if registry[key] != digest:
            violations.append(
                f"{rel}:{lineno}: '{ref}' digest {digest} does not match the "
                f"reviewed digest {registry[key]} for '{key}'"
            )

    if external == 0:
        violations.append(
            "no external image refs found across the manifests (non-vacuous: "
            "an empty scan cannot pass)"
        )

    if violations:
        for v in violations:
            print(f"::error::{v}")
        print(f"image-digest gate: {len(violations)} violation(s); FAIL")
        return EXIT_VIOLATIONS

    print(
        f"image-digest gate: {external} external image ref(s) pinned and "
        f"matching {REGISTRY_FILE}; OK"
    )
    return EXIT_OK


if __name__ == "__main__":
    sys.exit(main())
