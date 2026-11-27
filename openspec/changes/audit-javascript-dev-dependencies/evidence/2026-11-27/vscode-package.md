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
