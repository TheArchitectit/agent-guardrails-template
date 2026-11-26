# Design

## Context

The initial audit snapshot dated 2026-11-25 reported Pi: 9 development package entries (3 moderate, 4 high, 2 critical), and VS Code: 14 entries (4 moderate, 10 high). `npm audit --omit=dev` reported zero findings for each. The repository has no workflow enforcement for npm advisory audits. The newer dated report in [evidence](evidence/2026-11-25/audit.md) supersedes that snapshot with its own scan time and raw lockfile identities.

## Decisions

- Run audits from lockfile-backed clean installs and retain the date, tool version, lockfile identity, package-entry counts, and scope in the review record.
- Report runtime and development dependency results separately; never infer reachability or exploitability from the advisory count.
- Require disposition for each dated result. A later scan supersedes the snapshot only when it records its own date and lockfile identity.
- Defer severity thresholds and automatic blocking until maintainers approve a policy.

## Compatibility and Rollback

The process does not change runtime dependencies or package exports. Dependency updates remain separate reviewed changes; if an update breaks extension tests or packaging, revert that dependency update and retain the audit record with its original snapshot context.

## Verification

The procedure checks out no mutable package tree. It reads only the `package.json` and `package-lock.json` Git blobs at `HEAD`, verifies their identity/version agreement, installs in temporary directories with lifecycle scripts disabled, and runs full and `--omit=dev` audits against the public npm registry. It accepts only valid npm audit JSON with consistent package/severity totals and expected exit codes. It writes a dated artifact directory only after all four reports validate. No manifests or lockfiles are modified.

## Repeatable JavaScript extension dependency audit

Run with Node `v22.20.0` and its adjacent npm CLI at version `10.9.3`. The procedure rejects other tool versions, resolves its source commit and raw blob digests directly from Git, and ignores inherited Git/NPM configuration and credential variables. It invokes the adjacent `npm-cli.js` with `process.execPath` and argument arrays (no shell), using a temporary npm cache and empty user/global configuration files; stderr diagnostics are discarded and never enter report artifacts. The output directory argument must not already exist.

```sh
node scripts/audit-javascript-dev-dependencies.mjs /path/to/new-audit-output
```

The report records the Git commit, raw Git SHA-256 for manifests and lockfiles, date, tool versions, install flags, and separate full and `--omit=dev` results. A full audit exit code of 1 is expected when npm reports findings; all results must otherwise match the JSON. Failure or malformed output exits nonzero and publishes no report directory.
