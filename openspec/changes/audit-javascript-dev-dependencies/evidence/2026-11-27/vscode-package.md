# QA-04 VS Code package-tool candidate — 2026-11-27

Worktree: `guardrails-template-qa04-vsce-nov27`
Baseline: `ec06a4c` (`origin/main`)
Execution image: `qa04-node22-git:latest`
Toolchain: Node `v22.20.0`, npm `10.9.3`, Git `2.39.5`

## Candidate and scoped fix

In a complete disposable checkout, the official npm package
`@vscode/vsce@4.1.1` was tested as the replacement for the deprecated direct
`vsce` dependency. Its engine declares Node `>=22`. The package manifest and
lockfile are updated only in this isolated VS Code worktree.

The baseline package command failed because `.vscodeignore` excluded `out/`
while `package.json` declares `main: ./out/extension.js`. The candidate run
removed only that contradictory ignore entry; this is a packaging defect, not
a dependency risk acceptance.

## Completed measurements

From the repository root copied to a disposable container:

- `npm ci --ignore-scripts --no-audit --no-fund`: PASS
- `npm run compile`: PASS
- `npm run lint`: PASS
- `npm test`: PASS; all three repository-level regression checks passed
- `npm run package`: PASS; produced `guardrail-vscode-1.0.0.vsix` (16 files,
  15.98 KB), including the compiled extension entrypoint
- Full `npm audit --json`: exit 1 with 9 development findings (3 moderate,
  6 high, 0 critical)

The production-only `npm audit --omit=dev --json` rerun was intentionally
stopped when the lab handoff began and is **NOT_RUN**. No runtime-clean claim
is made. OpenSpec strict validation after this VS Code change is also
**NOT_RUN** and must be run in the lab. No publication or VSIX upload occurred.

## Integrated QA-04 follow-up — 2026-11-27

This section records the VS Code disposition after PRs #34 and #36 were integrated in order in the isolated lab worktree at base edadacf008a4f457b7898d36c2168cd91a8429d8. The historical candidate measurements above describe their earlier run and are retained as such; the final integrated measurements below supersede the candidate audit and packaging outcome.

### Dependency disposition

A clean locked baseline audit on Node 22.20.0/npm 10.9.3 reported 9 development package entries: @vscode/vsce, ajv, brace-expansion, flatted, js-yaml, minimatch, picomatch, qs, and underscore. The baseline npm audit --omit=dev reported zero entries. These findings came from the package manager audit and do not establish runtime reachability.

npm audit fix updated the seven independently fixable transitive entries within their parent package ranges: ajv 6.12.6 → 6.15.0, brace-expansion 1.1.12 → 1.1.22, flatted 3.3.3 → 3.4.4, js-yaml 4.1.1 → 4.3.2, picomatch 2.3.1 → 2.3.2, qs 6.14.2 → 6.17.1, and underscore 1.13.7 → 1.13.8. No dependency override was added.

The remaining stable-vsce advisory was GHSA-g3hv-75wg-m249 (minimatch >=7.0.0 <11.0.1). The registry latest tag was @vscode/vsce@4.1.1, whose dependency range was minimatch@^10.2.6. The registry next tag was 4.1.2-4; that published upstream prerelease changes the range to minimatch@^11.0.1. npm's suggested stable fix was the breaking major downgrade to vsce 3.9.1; a trial of 3.9.1 left three moderate findings (vsce, glob, minimatch). The selected package-tool dependency is therefore pinned to the upstream 4.1.2-4 prerelease as a development-only dependency, with Node engine requirement >=22. It was verified with the pinned Node 22.20.0/npm 10.9.3. This is a preview dependency rather than a stable-tag release and should be reconsidered when upstream publishes the patched stable version.

| Final scan | Exit | Findings |
|---|---:|---:|
| npm audit --json | 0 | 0 |
| npm audit --omit=dev --json | 0 | 0 |

### Integrated package verification

After npm ci --ignore-scripts --no-audit --no-fund, npm run compile, npm run lint, and npm test passed. The regression command passed its four assertions (VS Code request callback and JetBrains, Vim, and Neovim source builders). npm run package produced a local 16-file VSIX containing extension/out/extension.js and extension/package.json; it was inspected and not uploaded or published. The official OpenSpec 2.2.0 command openspec validate --all --strict passed all 73 items. Full private command logs and audit JSON are retained in sibling evidence36/ for the isolated review handoff.

The broader QA-04 disposition task remains open because the Pi evidence still has other package entries awaiting maintainer disposition. This VS Code follow-up does not claim those Pi entries are resolved.
