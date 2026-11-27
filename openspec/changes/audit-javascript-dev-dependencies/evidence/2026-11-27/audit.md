# QA-04 Pi development dependency update — 2026-11-27

Baseline: `ec06a4c` (`origin/main`)
Execution image: `qa04-node22-git:latest`
Toolchain: Node `v22.20.0`, npm `10.9.3`, Git `2.39.5`
Registry lookup: official npm registry; `vitest` current release was `5.1.2`

## Decision

Update the Pi extension's development-only `vitest` dependency from the
lockfile-resolved `2.1.9` to the maintained stable `5.1.2`. The candidate is
compatible with Node 22 and was selected from the official npm registry. No
runtime dependency or package export changed.

## Measurements

The candidate was installed in a clean temporary copy with
`npm ci --ignore-scripts --no-audit --no-fund` (exit 0). The existing Pi test
command then passed all 200 tests in 24 files:

```text
Test Files  24 passed (24)
Tests       200 passed (200)
test_exit=0
```

The final staged Git blob identity recorded for the updated lockfile is
`b6bcd1485282ec76d6a1c5d9db0a4834b69e57a8`; the independently recorded raw
lockfile SHA-256 is
`85de08c913be68fadb774b18f1f16d293ed44a5d1ada4d19549d08e471fe69ed`.

| Scan | Exit | Findings |
|---|---:|---:|
| `npm audit --json` | 0 | 0 |
| `npm audit --omit=dev --json` | 0 | 0 |

The full-audit report is retained in `pi-extension-full.json`; the
production-only report is retained in `pi-extension-omit-dev.json`. The
targeted lock update resolved the `zod` advisory without changing the exact
pinned MCP SDK. Other package entries in the existing review queue remain
pending maintainer disposition.

Both zero-finding JSON reports have SHA-256
`c917bbd29c65824bac30d91875da052ebb739b9ffa267a7f3c965dee151e3979`.

The UCS03 replay at `a5085359a8623d1749017cf2ca86b5f2eaa704a7` passed
all 200 Pi tests in 24 files. Full and production-only npm audits each
returned zero findings. OpenSpec 2.2.0 passed all 73 strict items.
The [separate Go replay](go-replay.md) records both toolchain jobs and the
historical laptop timeout.
