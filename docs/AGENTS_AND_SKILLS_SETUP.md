# Agent Guardrails - Agents and Skills Setup

This guide explains how to set up custom agents, skills, and hooks for Claude Code and OpenCode.

## Quick Start

Run the setup script to generate configuration files:

```bash
# Minimal setup (guardrails-enforcer only)
python scripts/setup_agents.py --claude --minimal
python scripts/setup_agents.py --opencode --minimal

# Full setup (all skills, agents, and hooks)
python scripts/setup_agents.py --claude --opencode --full
```

## What Gets Created

### Claude Code Configuration (`.claude/`)

**Minimal:**
- `.claude/skills/guardrails-enforcer.json` - Core safety enforcement

**Full:**
- `.claude/skills/guardrails-enforcer.json` - Four Laws enforcement
- `.claude/skills/commit-validator.json` - Commit message validation
- `.claude/skills/env-separator.json` - Test/production separation
- `.claude/hooks/pre-execution.sh` - Pre-operation checks
- `.claude/hooks/post-execution.sh` - Post-operation validation
- `.claude/hooks/pre-commit.sh` - Pre-commit validation

### OpenCode Configuration (`.opencode/`)

**Minimal:**
- `.opencode/oh-my-opencode.jsonc` - Main configuration
- `.opencode/skills/guardrails-enforcer/SKILL.md` - Core safety skill

**Full:**
- `.opencode/oh-my-opencode.jsonc` - Main configuration with all agents
- `.opencode/skills/guardrails-enforcer/SKILL.md` - Four Laws enforcement
- `.opencode/skills/commit-validator/SKILL.md` - Commit validation
- `.opencode/skills/env-separator/SKILL.md` - Environment separation
- `.opencode/agents/guardrails-auditor.json` - Post-work auditor
- `.opencode/agents/doc-indexer.json` - Documentation indexer

## Platform-Specific Guides

- [Claude Code Integration](CLCODE_INTEGRATION.md) - Detailed Claude Code setup
- [OpenCode Integration](OPENCODE_INTEGRATION.md) - Detailed OpenCode setup

## Available Skills

### guardrails-enforcer

Enforces the **Four Laws of Agent Safety**:
1. Read before editing
2. Stay in scope
3. Verify before committing
4. Halt when uncertain

**Activates:** Automatically on all operations

### commit-validator

Validates commits against `COMMIT_WORKFLOW.md`:
- AI attribution required
- Single focus per commit
- No secrets in diff
- Tests pass before commit

**Activates:** Before git commit operations

### env-separator

Enforces `TEST_PRODUCTION_SEPARATION.md`:
- Production code before test code
- Separate service instances
- No test data in production

**Activates:** When creating test code or modifying environments

## Customization

### Modifying Skills

After setup, edit the generated files:

**Claude Code:**
```bash
# Edit a skill
.claude/skills/<skill-name>.json
```

**OpenCode:**
```bash
# Edit a skill
.opencode/skills/<skill-name>/SKILL.md
```

### Creating Custom Skills

**For Claude Code:**

Create a new JSON file in `.claude/skills/`:

```json
{
  "name": "my-custom-skill",
  "description": "What this skill does",
  "tools": ["Read", "Grep"],
  "prompt": "Your skill instructions here..."
}
```

**For OpenCode:**

Create a new directory in `.opencode/skills/` with a `SKILL.md`:

```markdown
---
name: my-custom-skill
description: "What this skill does"
---

# Skill Title

Your skill instructions here...
```

## Removing Configuration

To remove the setup:

```bash
# Remove Claude Code configuration
rm -rf .claude/

# Remove OpenCode configuration
rm -rf .opencode/

# Remove both
rm -rf .claude/ .opencode/
```

## Troubleshooting

### Skills Not Loading

**Claude Code:**
- Ensure `.claude/skills/` directory exists
- Check JSON syntax in skill files
- Restart Claude Code

**OpenCode:**
- Verify `.opencode/oh-my-opencode.jsonc` syntax
- Check that skills are listed in the `enable` array
- Restart OpenCode

### Hooks Not Running

- Ensure hooks are executable: `chmod +x .claude/hooks/*.sh`
- Check hook syntax with `bash -n <hook-file>`
- Verify hook paths in your configuration

## Script-Based Workflows

For large-scale operations, scripts are more efficient than interactive AI sessions. The guardrails framework supports scripted execution.

### When to Use Scripts

| Scenario | Interactive AI | Script |
|----------|---------------|--------|
| Single file changes | ✓ Efficient | ✗ Overhead |
| Multi-file refactoring | △ Token-heavy | ✓ Better |
| Large code review | △ Slow | ✓ Parallelizable |
| Batch migrations | ✗ Impractical | ✓ Essential |
| Repo-wide analysis | △ Expensive | ✓ Scalable |

### Large Code Review Script

Create `scripts/large_code_review.py`:

```python
#!/usr/bin/env python3
"""
Large Code Review Script

Performs guardrails-compliant code review across multiple files.
Uses parallel processing for efficiency.
"""

import argparse
import json
import subprocess
from concurrent.futures import ThreadPoolExecutor
from pathlib import Path


def review_file(file_path: Path, guardrails_config: dict) -> dict:
    """Review a single file against guardrails."""
    result = {
        "file": str(file_path),
        "violations": [],
        "passed": True
    }

    # Check: File was read (simulated - in real use, track reads)
    # Check: Scope compliance
    if not is_in_scope(file_path, guardrails_config.get("scope", [])):
        result["violations"].append("File outside authorized scope")
        result["passed"] = False

    # Check: No secrets
    if contains_secrets(file_path):
        result["violations"].append("Potential secrets detected")
        result["passed"] = False

    # Check: Test/Production separation
    if mixes_environments(file_path):
        result["violations"].append("Test/Production environment mix detected")
        result["passed"] = False

    return result


def is_in_scope(file_path: Path, scope_patterns: list) -> bool:
    """Check if file is within authorized scope."""
    for pattern in scope_patterns:
        if file_path.match(pattern):
            return True
    return len(scope_patterns) == 0  # No scope = all allowed


def contains_secrets(file_path: Path) -> bool:
    """Scan for common secret patterns."""
    secret_patterns = [
        r"password\s*=\s*['\"][^'\"]+['\"]",
        r"api_key\s*=\s*['\"][^'\"]+['\"]",
        r"secret\s*=\s*['\"][^'\"]+['\"]",
        r"sk-[a-zA-Z0-9]{48}",  # OpenAI key pattern
    ]
    # Implementation would scan file content
    return False


def mixes_environments(file_path: Path) -> bool:
    """Check for test/production environment mixing."""
    content = file_path.read_text()
    # Check for production URLs in test files
    if "test" in file_path.name.lower():
        if "production.com" in content or "prod-db" in content:
            return True
    return False


def main():
    parser = argparse.ArgumentParser(description="Large-scale guardrails review")
    parser.add_argument("--files", nargs="+", required=True, help="Files to review")
    parser.add_argument("--config", default=".guardrails.json", help="Guardrails config")
    parser.add_argument("--parallel", type=int, default=4, help="Parallel workers")
    args = parser.parse_args()

    # Load guardrails config
    config = json.loads(Path(args.config).read_text())

    # Review files in parallel
    files = [Path(f) for f in args.files]
    with ThreadPoolExecutor(max_workers=args.parallel) as executor:
        results = list(executor.map(lambda f: review_file(f, config), files))

    # Output results
    violations = [r for r in results if not r["passed"]]
    if violations:
        print(f"\n❌ {len(violations)} files failed guardrails check:")
        for v in violations:
            print(f"  - {v['file']}: {', '.join(v['violations'])}")
        return 1
    else:
        print(f"\n✅ All {len(results)} files passed guardrails check")
        return 0


if __name__ == "__main__":
    exit(main())
```

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
from typing import List, Optional


@dataclass
class Operation:
    """Single operation with guardrails tracking."""
    file: Path
    operation_type: str  # 'read', 'edit', 'delete'
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
        """Verify operation is safe to proceed."""
        # Law 1: Read before editing
        if op.operation_type == "edit" and op.file not in self.read_files:
            print(f"❌ HALT: Attempting to edit unread file: {op.file}")
            return False

        # Law 2: Stay in scope
        if not self._is_in_scope(op.file):
            print(f"❌ HALT: File outside scope: {op.file}")
            return False

        # Law 4: Halt when uncertain (simulated)
        if self._is_uncertain(op):
            print(f"❌ HALT: Uncertain about operation on: {op.file}")
            return False

        return True

    def _is_in_scope(self, file: Path) -> bool:
        """Check if file is within authorized scope."""
        scope = self.config.get("scope", [])
        if not scope:
            return True
        return any(file.match(pattern) for pattern in scope)

    def _is_uncertain(self, op: Operation) -> bool:
        """Determine if we should halt due to uncertainty."""
        # Three Strikes Rule
        if op.attempts >= op.max_attempts:
            return True
        return False

    def execute(self, operations: List[Operation]) -> bool:
        """Execute operations with guardrails."""
        for op in operations:
            print(f"Processing: {op.file}")

            # Pre-flight check
            if not self.pre_flight_check(op):
                self.failed_operations.append(op)
                continue

            # Track reads
            if op.operation_type == "read":
                self.read_files.add(op.file)

            # Execute (simulated)
            try:
                self._do_operation(op)
                op.completed = True
                print(f"  ✅ Completed")
            except Exception as e:
                op.attempts += 1
                print(f"  ⚠️  Failed (attempt {op.attempts}/{op.max_attempts}): {e}")
                if op.attempts >= op.max_attempts:
                    print(f"  ❌ Three strikes - halting")
                    self.failed_operations.append(op)

        # Summary
        success = len(self.failed_operations) == 0
        if not success:
            print(f"\n❌ {len(self.failed_operations)} operations failed")
        return success

    def _do_operation(self, op: Operation):
        """Actually perform the operation."""
        # Implementation would do the actual work
        pass


def main():
    # Example usage
    config = Path(".guardrails.json")
    executor = GuardrailsExecutor(config)

    operations = [
        Operation(Path("src/main.py"), "read"),
        Operation(Path("src/main.py"), "edit"),
        Operation(Path("src/utils.py"), "read"),
        Operation(Path("src/utils.py"), "edit"),
    ]

    success = executor.execute(operations)
    return 0 if success else 1


if __name__ == "__main__":
    sys.exit(main())
```

### Integration with CI/CD

Add to `.github/workflows/guardrails-batch.yml`:

```yaml
name: Batch Guardrails Check

on:
  pull_request:
    paths:
      - "src/**"
      - "tests/**"

jobs:
  guardrails:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Run batch guardrails review
        run: |
          python scripts/large_code_review.py \
            --files $(git diff --name-only HEAD^) \
            --config .guardrails.json \
            --parallel 8
```

### Best Practices

1. **Track reads**: Maintain a log of files that have been read
2. **Batch by scope**: Group operations by scope boundaries
3. **Parallelize safely**: Only parallelize independent operations
4. **Halt on failure**: Stop the batch if critical violations found
5. **Generate reports**: Output machine-readable results for CI integration

## References

- [AGENT_GUARDRAILS.md](AGENT_GUARDRAILS.md) - Core safety protocols
- [TEST_PRODUCTION_SEPARATION.md](standards/TEST_PRODUCTION_SEPARATION.md) - Environment rules
- [COMMIT_WORKFLOW.md](workflows/COMMIT_WORKFLOW.md) - Commit standards
- [AGENT_EXECUTION.md](workflows/AGENT_EXECUTION.md) - Execution protocols
