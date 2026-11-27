"""Shared synthetic context fixture for Docker and Podman context checks."""

from __future__ import annotations

import shutil
from pathlib import Path

EXCLUDED = (
    ".env", ".git/config", ".cache/fixture", "__pycache__/fixture.pyc",
    "web/.env", "web/.git/config", "web/.cache/fixture",
    ".guardrails/prevention-rules/.env", ".guardrails/prevention-rules/.git/config",
    ".guardrails/prevention-rules/.cache/fixture",
)
REQUIRED = (
    "go.mod", "go.sum", "cmd/server/main.go", "web/index.html",
    ".guardrails/prevention-rules/extracted-rules.json",
)


def write_synthetic_context(context: Path, dockerignore: Path, dockerfile: str | None = None) -> None:
    """Create only synthetic files plus the repository's real ignore rules."""
    shutil.copy2(dockerignore, context / ".dockerignore")
    for relative in EXCLUDED + REQUIRED:
        path = context / relative
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_text("synthetic context fixture\n", encoding="utf-8")
    if dockerfile is not None:
        (context / "Dockerfile").write_text(dockerfile, encoding="utf-8")
