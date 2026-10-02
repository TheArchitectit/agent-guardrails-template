# F021 audit: dangerous command classifier (Oct 2, 2026)

Method: ran the shipped rules in .guardrails/prevention-rules/extracted-rules.json through the server's own MatchPattern against a corpus (mcp-server/internal/validation/corpus_audit_test.go).

## Findings (before)
1. PREVENT-GIT-001 (no force push) used a negative lookahead. Go's regexp (RE2) cannot compile it, and SafeRegex turned the compile error into "no match". Result: `git push --force origin main` was NOT caught. Fail-open.
2. Not covered at all: `git push -f`, `+ref` pushes, remote branch delete (`--delete`, `:ref`), `git branch -D main`, `git filter-repo`/`filter-branch`, `git clean -f`, `rm -rf ~`.
3. PREVENT-SYS-001 `rm\s+-rf\s+/` also matched harmless `rm -rf /tmp/build` (false positive) and missed `rm -rf ~`.

## Changes
- SafeRegex now returns an error for an invalid pattern instead of "no match" (tests updated: the old tests encoded the fail-open behavior).
- PREVENT-GIT-001 rewritten in RE2 syntax. New rules PREVENT-GIT-007..011. PREVENT-SYS-001 tightened.
- Corpus test asserts current behavior, including that `--force-with-lease`, `git push origin feat` and `rm -rf /tmp/build` are NOT flagged.

## Still open
- The engine callers (internal/validation/engine.go) log and `continue` on a pattern error, so an invalid rule is still skipped, now with a visible error instead of silently. A verdict of ERROR (not PASS) for the whole call is the right fix; needs its own change.
- These are regex rules on command text: advisory unless a host hook runs them. A determined command can evade them (aliases, scripts, `sh -c`). Do not market as blocking.
- Rules are stored in the DB; the JSON file seeds it. Existing deployments need the rules reloaded for the new patterns to apply (not verified).
- Behavior change: more commands will be flagged. Owner verdict needed before merge.
