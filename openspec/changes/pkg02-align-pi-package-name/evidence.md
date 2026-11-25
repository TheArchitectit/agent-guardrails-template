# PKG-02 Verification Evidence

Verification ran in the isolated Windows worktree at merged baseline `6ca7b122498db529ae96f83662d76898d33b0587`, using Docker `node:20-bookworm-slim` for Node/npm commands.

- `npm ci`: passed; installed the lockfile dependencies (139 packages).
- `npm test`: **failed** on two runs with 199/200 tests passing. The unrelated `mcp-bridge/mcp-client-optional.test.ts` test `loads the bridge in isolation and reports the SDK unavailable` timed out at its 5-second limit both times. The new `tests/package-identity.test.ts` passed in both runs. A third attempt from a Linux-local copy did not reach test output while `npm ci` was still taking several minutes and was stopped; no further retries were made. No unrelated test or timeout changes were made.
- `npm pack --json --dry-run`: passed separately; package metadata was `@architectit/pi-guardrails` version `0.2.0`, with 70 packed files. The listing excludes `tests/package-identity.test.ts`.
- `install.mjs --help` with `HOME` and `USERPROFILE` set to a fresh temporary directory: passed; output showed `npx @architectit/pi-guardrails` and the `pi-guardrails` extension path under that temporary home. The `.pi` directory was not created.
- Independent registry readback by the coordinating agent: `https://registry.npmjs.org/@architectit%2fpi-guardrails/latest` returned HTTP 200 with `name=@architectit/pi-guardrails`, version `0.2.0`. No registry `npx` command was run and nothing was published.
- `npx --yes @fission-ai/openspec@2.2.0 validate --all --strict`: passed, 67 items, 0 failures. OpenSpec emitted an engine warning because this run used Node 20; validation completed successfully.
