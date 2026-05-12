# Skills Architecture

This repo follows a canonical skill pattern inspired by [obra/superpowers](https://github.com/obra/superpowers): one canonical skill definition consumed by lightweight IDE-specific plugins.

## Canonical Skill Format

Each skill lives in `skills/<id>/SKILL.md`:

```markdown
---
id: guardrails-enforcer
name: Guardrails Enforcement Agent
description: Enforces the Four Laws of Agent Safety on all operations
version: 1.0.0
tags: [safety, core, mandatory]
applies_to: [claude, cursor, opencode, openclaw, windsurf, copilot]
author: TheArchitectit
references:
  - skills/four-laws/SKILL.md
tools: [Read, Grep, Glob, AskUserQuestion]   # Claude-specific
globs: "**/*"                                # Cursor-specific
alwaysApply: true                            # Cursor-specific
---

# Guardrails Enforcement Agent

Skill body in markdown...
```

## Build Script

`scripts/build_skills.py` reads canonical skills and generates native IDE files:

```bash
python scripts/build_skills.py                    # Build all platforms
python scripts/build_skills.py --check            # Drift check (CI)
python scripts/build_skills.py --dry-run          # Preview changes
python scripts/build_skills.py --platform claude  # Single platform
python scripts/build_skills.py --skill guardrails-enforcer  # Single skill
```

Generated files receive a `GENERATED` header. **Do not edit them directly.**

## Plugin Manifests

Each supported IDE has a lightweight plugin manifest referencing `./skills/`:

- `.claude-plugin/plugin.json`
- `.cursor-plugin/plugin.json`
- `.codex-plugin/plugin.json`
- `.gemini-extension/gemini-extension.json`

## Script-Based Workflows

For large-scale operations, scripts are more efficient than interactive AI sessions.

### When to Use Scripts

| Scenario | Interactive AI | Script |
|----------|---------------|--------|
| Single file changes | ✓ Efficient | ✗ Overhead |
| Multi-file refactoring | △ Token-heavy | ✓ Better |
| Large code review | △ Slow | ✓ Parallelizable |
| Batch migrations | ✗ Impractical | ✓ Essential |
| Repo-wide analysis | △ Expensive | ✓ Scalable |

### Batch Execution Script

Create `scripts/batch_execute.py`:

```python
#!/usr/bin/env python3
"""
Batch Execution Script with Guardrails

Executes operations across multiple files with full guardrails compliance.
Implements Three Strikes Rule and halt conditions.
"""

import json
import sys
from dataclasses import dataclass
from pathlib import Path
from typing import List


@dataclass
class Operation:
    """Single operation with guardrails tracking."""
    file: Path
    operation_type: str
    attempts: int = 0
    max_attempts: int = 3
    completed: bool = False


class GuardrailsExecutor:
    """Executes operations with guardrails enforcement."""

    def __init__(self, config_path: Path):
        self.config = json.loads(config_path.read_text())
        self.read_files: set = set()
        self.failed_operations: List[Operation] = []

    def pre_flight_check(self, op: Operation) -> bool:
        if op.operation_type == "edit" and op.file not in self.read_files:
            print(f"HALT: Attempting to edit unread file: {op.file}")
            return False
        if not self._is_in_scope(op.file):
            print(f"HALT: File outside scope: {op.file}")
            return False
        if op.attempts >= op.max_attempts:
            print(f"HALT: Three strikes on: {op.file}")
            return False
        return True

    def _is_in_scope(self, file: Path) -> bool:
        scope = self.config.get("scope", [])
        if not scope:
            return True
        return any(file.match(pattern) for pattern in scope)

    def execute(self, operations: List[Operation]) -> bool:
        for op in operations:
            if not self.pre_flight_check(op):
                self.failed_operations.append(op)
                continue
            if op.operation_type == "read":
                self.read_files.add(op.file)
            op.completed = True
        return len(self.failed_operations) == 0


def main():
    config = Path(".guardrails.json")
    executor = GuardrailsExecutor(config)
    operations = [
        Operation(Path("src/main.py"), "read"),
        Operation(Path("src/main.py"), "edit"),
    ]
    success = executor.execute(operations)
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
```

## Integration with CI/CD

Add to `.github/workflows/skill-build-check.yml`:

```yaml
name: Skill Build Check

on:
  push:
    branches: [main, develop]
    paths:
      - 'skills/**'
      - 'scripts/build_skills.py'
      - 'scripts/skill_lib/**'
      - 'tests/scripts/**'
  pull_request:
    paths:
      - 'skills/**'
      - 'scripts/build_skills.py'
      - 'scripts/skill_lib/**'
      - 'tests/scripts/**'

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.10'
      - run: pip install pytest pyyaml
      - run: pytest tests/scripts/ -v

  drift-check:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.10'
      - run: pip install pyyaml
      - run: python scripts/build_skills.py --check
```

## Best Practices

1. **Edit canonical sources only** — Never edit generated native files
2. **Run `--check` in CI** — Block PRs with stale generated files
3. **Track reads** — Maintain a log of files that have been read
4. **Batch by scope** — Group operations by scope boundaries
5. **Parallelize safely** — Only parallelize independent operations
6. **Generate reports** — Output machine-readable results for CI integration

## References

- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [TEST_PRODUCTION_SEPARATION.md](standards/TEST_PRODUCTION_SEPARATION.md) - Environment rules
- [COMMIT_WORKFLOW.md](workflows/COMMIT_WORKFLOW.md) - Commit standards
- [AGENT_EXECUTION.md](workflows/AGENT_EXECUTION.md) - Execution protocols
