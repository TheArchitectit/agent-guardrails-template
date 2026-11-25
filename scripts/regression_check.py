#!/usr/bin/env python3
"""
Regression Check Tool
Scans staged/unstaged changes against failure registry to detect potential regressions.

Usage:
    python scripts/regression_check.py              # Check staged changes
    python scripts/regression_check.py --unstaged   # Check unstaged changes
    python scripts/regression_check.py --all        # Check all changes
    python scripts/regression_check.py --pre-commit # Exit with error code if issues found

Environment Variables:
    FAILURE_REGISTRY_PATH: Path to registry file
    PREVENTION_RULES_PATH: Path to prevention rules directory
"""

import argparse
import fnmatch
import json
import os
import re
import subprocess
import sys
from pathlib import Path
from typing import List, Dict, Tuple, Optional

DEFAULT_REGISTRY_PATH = Path(".guardrails/failure-registry.jsonl")
DEFAULT_RULES_PATH = Path(".guardrails/prevention-rules")


def run_git_command(args: List[str]) -> Tuple[int, str, str]:
    """Run a git command and return (returncode, stdout, stderr)."""
    try:
        result = subprocess.run(
            ["git"] + args,
            capture_output=True,
            text=True,
            cwd=Path.cwd()
        )
        return result.returncode, result.stdout, result.stderr
    except FileNotFoundError:
        return 1, "", "git command not found"


def resolve_ci_range(base: str, head: str, merge_base: bool = False) -> Tuple[str, str]:
    """Resolve and validate the CI comparison endpoints to full commit IDs."""
    if not base or not head:
        raise ValueError("both --base and --head are required for a CI range")
    revisions = []
    for revision in (base, head):
        rc, stdout, stderr = run_git_command(
            ["rev-parse", "--verify", "--end-of-options", f"{revision}^{{commit}}"]
        )
        if rc != 0:
            raise ValueError(f"CI range revision {revision!r} is unavailable: {stderr.strip()}")
        revisions.append(stdout.strip())
    base_commit, head_commit = revisions
    rc, stdout, _ = run_git_command(["rev-parse", "--is-shallow-repository"])
    if rc != 0 or stdout.strip() == "true":
        raise ValueError("CI range history is shallow or cannot be verified completely")
    if merge_base:
        rc, stdout, stderr = run_git_command(["merge-base", "--all", base_commit, head_commit])
        common_bases = stdout.split()
        if rc != 0 or len(common_bases) != 1:
            detail = stderr.strip() or f"expected one common ancestor, found {len(common_bases)}"
            raise ValueError(f"CI PR range {base}..{head} has missing or ambiguous history: {detail}")
        return common_bases[0], head_commit
    rc, _, stderr = run_git_command(["merge-base", "--is-ancestor", base_commit, head_commit])
    if rc != 0:
        detail = "base is not an ancestor of head" if rc == 1 else stderr.strip()
        raise ValueError(f"CI range {base}..{head} is incomplete or invalid: {detail}")
    return base_commit, head_commit


def get_changed_files(staged: bool = True, unstaged: bool = False,
                      base: str | None = None, head: str | None = None,
                      merge_base: bool = False,
                      resolved_range: Tuple[str, str] | None = None) -> List[str]:
    """Get list of changed files from git."""
    if base is not None or head is not None:
        range_base, range_head = resolved_range or resolve_ci_range(base, head, merge_base)
        rc, stdout, stderr = run_git_command(["diff", "--name-only", "-z", range_base, range_head, "--"])
        if rc != 0:
            raise ValueError(f"cannot inspect CI range {base}..{head}: {stderr.strip()}")
        files = [name for name in stdout.split("\0") if name]
        if not files:
            raise ValueError(f"CI range {base}..{head} is empty; refusing a vacuous scan")
        return files

    files = []

    if staged:
        rc, stdout, _ = run_git_command(["diff", "--cached", "--name-only", "-z"])
        if rc == 0:
            files.extend(stdout.split("\0") if stdout else [])

    if unstaged:
        rc, stdout, _ = run_git_command(["diff", "--name-only", "-z"])
        if rc == 0:
            files.extend(stdout.split("\0") if stdout else [])

    return list(dict.fromkeys(f for f in files if f))


def get_diff_content(file_path: str, staged: bool = True,
                     base: str | None = None, head: str | None = None) -> str:
    """Get diff content for a specific file."""
    if base is not None and head is not None:
        rc, stdout, stderr = run_git_command(["diff", base, head, "--", f":(literal){file_path}"])
        if rc not in (0, 1):
            raise ValueError(f"cannot read CI patch for {file_path}: {stderr.strip()}")
        return stdout
    cmd = ["diff", "--cached"] if staged else ["diff"]
    rc, stdout, _ = run_git_command(cmd + ["--", f":(literal){file_path}"])
    return stdout if rc in (0, 1) else ""


def load_failure_registry(registry_path: Path) -> List[Dict]:
    """Load failure entries from registry."""
    if not registry_path.exists():
        return []

    entries = []
    with open(registry_path, "r") as f:
        for line in f:
            line = line.strip()
            if line and not line.startswith("#"):
                try:
                    entry = json.loads(line)
                    if entry.get("status") == "active":
                        entries.append(entry)
                except json.JSONDecodeError:
                    continue
    return entries


def validate_rule_regex(rule: Dict) -> bool:
    """Validate regex patterns in a rule."""
    pattern = rule.get("pattern", "")
    if pattern:
        try:
            re.compile(pattern)
        except re.error as e:
            print(f"Warning: Invalid regex in rule {rule.get('rule_id')}: {e}")
            return False

    forbidden = rule.get("forbidden_context", "")
    if forbidden:
        try:
            re.compile(forbidden)
        except re.error as e:
            print(f"Warning: Invalid forbidden_context in rule {rule.get('rule_id')}: {e}")
            return False

    return True


def load_prevention_rules(rules_path: Path) -> List[Dict]:
    """Load prevention rules from rules directory."""
    rules = []

    pattern_rules_file = rules_path / "pattern-rules.json"
    if pattern_rules_file.exists():
        try:
            with open(pattern_rules_file, "r") as f:
                data = json.load(f)
                for rule in data.get("rules", []):
                    if rule.get("enabled", True):
                        if validate_rule_regex(rule):
                            rule["rule_type"] = "pattern"
                            rules.append(rule)
        except (json.JSONDecodeError, IOError):
            pass

    semantic_rules_file = rules_path / "semantic-rules.json"
    if semantic_rules_file.exists():
        try:
            with open(semantic_rules_file, "r") as f:
                data = json.load(f)
                for rule in data.get("rules", []):
                    if rule.get("enabled", True):
                        rule["rule_type"] = "semantic"
                        rules.append(rule)
        except (json.JSONDecodeError, IOError):
            pass

    return rules


def check_file_against_failures(
    file_path: str,
    failures: List[Dict]
) -> List[Dict]:
    """Check if file is in affected_files of any active failure."""
    matching_failures = []

    for failure in failures:
        affected_files = failure.get("affected_files", [])
        for affected in affected_files:
            # Use fnmatch for proper glob pattern matching
            if fnmatch.fnmatch(file_path, affected):
                matching_failures.append(failure)
                break

    return matching_failures


def _glob_to_regex(glob: str) -> "re.Pattern[str]":
    """Translate a repo-relative glob to a regex (same contract as the Node scanner).

    ``*`` never crosses a path separator and ``**`` does. Python's ``fnmatch``
    lets ``*`` cross ``/``, so reusing it here would silently widen a rule
    scoped to ``src/*.py`` into one matching ``src/a/b.py``.
    """
    out = []
    i = 0
    while i < len(glob):
        ch = glob[i]
        if ch == "*":
            if glob[i:i + 3] == "**/":
                out.append("(?:.*/)?")
                i += 3
                continue
            if glob[i:i + 2] == "**":
                out.append(".*")
                i += 2
                continue
            out.append("[^/]*")
            i += 1
            continue
        if ch == "?":
            out.append("[^/]")
            i += 1
            continue
        out.append(re.escape(ch))
        i += 1
    return re.compile("^" + "".join(out) + "$")


def rule_applies_to(rule: Dict, file_path: str) -> bool:
    """True when the rule's declared ``file_glob`` covers this path.

    A rule may scope itself to a file kind. That declaration was ignored here
    entirely - ``file_glob`` had ZERO readers in this module while all 32
    shipped pattern rules declare one - so a rule written for ``Dockerfile*``
    fired against every changed Markdown, Python and JSON file. The Node
    scanner (``guardrails-scan.mjs``) has always consulted it, which is why the
    two scanners disagreed about the same rule set.
    """
    globs = rule.get("file_glob")
    if not isinstance(globs, list) or not globs:
        return True
    # Basename OR repo-relative path, matching the shared ``glob_matches``
    # contract used by the other gates in this family. The rules are authored
    # as file KINDS ("*.py", "*.gd", "*.js"), and a path-only match would make
    # nearly every one of them dead the moment a file sits in a subdirectory -
    # which is the same silent-scope failure this fix exists to remove.
    base = file_path.rsplit("/", 1)[-1]
    return any(
        _glob_to_regex(g).match(file_path) or ("/" not in g and _glob_to_regex(g).match(base))
        for g in globs
    )


def check_diff_against_patterns(
    diff_content: str,
    rules: List[Dict],
    file_path: str | None = None,
) -> List[Dict]:
    """Check diff content against pattern rules."""
    violations = []

    # Extract added lines only (lines starting with +)
    added_lines = []
    for line in diff_content.split("\n"):
        if line.startswith("+") and not line.startswith("+++"):
            added_lines.append(line[1:])  # Remove the + prefix

    added_content = "\n".join(added_lines)

    for rule in rules:
        if rule.get("rule_type") != "pattern":
            continue

        # Scope first: a rule that does not apply to this file cannot be
        # violated by it, regardless of what the regex would match.
        if file_path is not None and not rule_applies_to(rule, file_path):
            continue

        pattern = rule.get("pattern")
        if not pattern:
            continue

        try:
            if re.search(pattern, added_content, re.MULTILINE):
                # Check forbidden context if specified
                forbidden = rule.get("forbidden_context")
                if forbidden and re.search(forbidden, added_content, re.MULTILINE):
                    continue  # Context suggests this is OK

                violations.append({
                    "rule_id": rule.get("rule_id"),
                    "name": rule.get("name"),
                    "message": rule.get("message"),
                    "severity": rule.get("severity", "warning"),
                    "suggestion": rule.get("suggestion"),
                    "failure_id": rule.get("failure_id"),
                })
        except re.error:
            continue  # Invalid regex, skip

    return violations


def format_severity(severity: str) -> str:
    """Format severity with color codes (if terminal supports it)."""
    colors = {
        "critical": "\033[91m",  # Red
        "high": "\033[93m",      # Yellow
        "medium": "\033[94m",    # Blue
        "low": "\033[90m",       # Gray
        "error": "\033[91m",
        "warning": "\033[93m",
    }
    reset = "\033[0m"

    if sys.stdout.isatty():
        return f"{colors.get(severity.lower(), '')}{severity.upper()}{reset}"
    return severity.upper()


def run_regression_check(
    registry_path: Path,
    rules_path: Path,
    staged: bool = True,
    unstaged: bool = False,
    verbose: bool = False,
    base: str | None = None,
    head: str | None = None,
    merge_base: bool = False,
) -> Tuple[int, List[Dict]]:
    """
    Run full regression check.
    Returns (issue_count, issues_details).
    """
    issues = []

    # Load data
    failures = load_failure_registry(registry_path)
    rules = load_prevention_rules(rules_path)

    if verbose:
        print(f"Loaded {len(failures)} active failures, {len(rules)} enabled rules")

    # Resolve CI endpoints once so both path discovery and every patch use the
    # exact same validated comparison range.
    committed_range = resolve_ci_range(base, head, merge_base) if base is not None or head is not None else None
    changed_files = get_changed_files(staged=staged, unstaged=unstaged, base=base, head=head,
                                      merge_base=merge_base, resolved_range=committed_range)

    if not changed_files:
        if verbose:
            print("No changed files to check")
        return 0, []

    if verbose:
        print(f"Checking {len(changed_files)} changed file(s)...")

    # Check each file
    for file_path in changed_files:
        file_issues = {
            "file": file_path,
            "failures": [],
            "violations": [],
        }

        # Check against failure registry
        matching_failures = check_file_against_failures(file_path, failures)
        if matching_failures:
            file_issues["failures"] = matching_failures

        # Check diff against pattern rules
        if committed_range is not None:
            patches = [get_diff_content(file_path, base=committed_range[0], head=committed_range[1])]
        else:
            patches = []
            if staged:
                patches.append(get_diff_content(file_path, staged=True))
            if unstaged:
                patches.append(get_diff_content(file_path, staged=False))
        violations = []
        for patch in patches:
            if patch:
                violations.extend(check_diff_against_patterns(patch, rules, file_path))
        seen_rules = set()
        for violation in violations:
            rule_id = violation.get("rule_id")
            if rule_id not in seen_rules:
                file_issues["violations"].append(violation)
                seen_rules.add(rule_id)

        if file_issues["failures"] or file_issues["violations"]:
            issues.append(file_issues)

    return len(issues), issues


def print_report(issues: List[Dict], verbose: bool = False):
    """Print formatted report of issues."""
    if not issues:
        print("\n✓ No potential regressions detected")
        return

    print("\n" + "=" * 70)
    print("REGRESSION CHECK REPORT")
    print("=" * 70)

    for issue in issues:
        file_path = issue["file"]
        print(f"\n📄 {file_path}")
        print("-" * 70)

        # Print matching failures
        for failure in issue["failures"]:
            severity = format_severity(failure.get("severity", "medium"))
            print(f"\n  ⚠️  {severity} - Known Bug History")
            print(f"      Failure ID: {failure['failure_id']}")
            print(f"      Category: {failure.get('category', 'unknown')}")
            print(f"      Previous Error: {failure.get('error_message', 'N/A')[:80]}...")
            print(f"      Prevention: {failure.get('prevention_rule', 'N/A')}")

        # Print pattern violations
        for violation in issue["violations"]:
            severity = format_severity(violation.get("severity", "warning"))
            print(f"\n  🚫 {severity} - Pattern Violation")
            print(f"      Rule: {violation.get('name', 'Unknown')}")
            print(f"      Message: {violation.get('message', 'N/A')}")
            if violation.get("failure_id"):
                print(f"      Related Failure: {violation['failure_id']}")
            if violation.get("suggestion"):
                print(f"      Suggestion: {violation['suggestion']}")

    print("\n" + "=" * 70)
    print(f"Total files with potential issues: {len(issues)}")
    print("=" * 70)
    print("\nReview the above carefully before committing.")


def main():
    parser = argparse.ArgumentParser(
        description="Check for potential regressions in changed code",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
    %(prog)s                    # Check staged changes
    %(prog)s --unstaged         # Check unstaged changes
    %(prog)s --all              # Check all changes
    %(prog)s --pre-commit       # Exit with error if issues found
        """
    )

    parser.add_argument("--registry", "-r", type=Path,
                        default=Path(os.getenv("FAILURE_REGISTRY_PATH", DEFAULT_REGISTRY_PATH)),
                        help="Path to failure registry")
    parser.add_argument("--rules", type=Path,
                        default=Path(os.getenv("PREVENTION_RULES_PATH", DEFAULT_RULES_PATH)),
                        help="Path to prevention rules directory")

    # What to check
    group = parser.add_mutually_exclusive_group()
    group.add_argument("--staged", action="store_true", default=True,
                       help="Check staged changes (default)")
    group.add_argument("--unstaged", "-u", action="store_true",
                       help="Check unstaged changes")
    group.add_argument("--all", "-a", action="store_true",
                       help="Check both staged and unstaged changes")
    parser.add_argument("--base", help="CI base commit (requires --head; fails closed on an empty range)")
    parser.add_argument("--head", help="CI head commit (requires --base; fails closed on an empty range)")
    parser.add_argument("--merge-base", action="store_true",
                        help="For PRs, scan the unique merge-base-to-head range")

    # Output options
    parser.add_argument("--pre-commit", action="store_true",
                        help="Exit with non-zero code if issues found (for pre-commit hooks)")
    parser.add_argument("--json", action="store_true",
                        help="Output results as JSON")
    parser.add_argument("--verbose", "-v", action="store_true",
                        help="Verbose output")
    parser.add_argument("--quiet", "-q", action="store_true",
                        help="Only output on issues found")

    args = parser.parse_args()

    if bool(args.base) != bool(args.head):
        parser.error("--base and --head must be supplied together")
    if args.base and (args.unstaged or args.all):
        parser.error("CI range mode cannot be combined with local diff selectors")
    if args.merge_base and not args.base:
        parser.error("--merge-base requires --base and --head")

    # Determine what to check
    staged = args.staged and not args.unstaged and not args.all
    unstaged = args.unstaged or args.all
    if args.all:
        staged = True

    # Run check
    try:
        count, issues = run_regression_check(
            registry_path=args.registry,
            rules_path=args.rules,
            staged=staged,
            unstaged=unstaged,
            verbose=args.verbose and not args.quiet,
            base=args.base,
            head=args.head,
            merge_base=args.merge_base,
        )
    except ValueError as exc:
        print(f"Regression check error: {exc}", file=sys.stderr)
        sys.exit(2)

    # Output results
    if args.json:
        print(json.dumps({
            "issue_count": count,
            "issues": issues
        }, indent=2))
    elif not args.quiet or count > 0:
        print_report(issues, verbose=args.verbose)

    # Exit code
    if args.pre_commit and count > 0:
        sys.exit(1)
    sys.exit(0)


if __name__ == "__main__":
    main()
