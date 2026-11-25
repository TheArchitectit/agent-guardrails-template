# Design

## Context

The audit snapshot dated 2026-11-25 reports Pi: 9 development package entries (3 moderate, 4 high, 2 critical), and VS Code: 14 entries (4 moderate, 10 high). `npm audit --omit=dev` reports zero findings for each. The repository has no workflow enforcement for npm advisory audits.

## Decisions

- Run audits from lockfile-backed clean installs and retain the date, tool version, lockfile identity, package-entry counts, and scope in the review record.
- Report runtime and development dependency results separately; never infer reachability or exploitability from the advisory count.
- Require disposition for each dated result. A later scan supersedes the snapshot only when it records its own date and lockfile identity.
- Defer severity thresholds and automatic blocking until maintainers approve a policy.

## Compatibility and Rollback

The process does not change runtime dependencies or package exports. Dependency updates remain separate reviewed changes; if an update breaks extension tests or packaging, revert that dependency update and retain the audit record with its original snapshot context.

## Verification

Use `npm ci`, `npm audit --json`, and `npm audit --omit=dev --json` in each package directory; verify reports identify the lockfile and separate dependency scope.
