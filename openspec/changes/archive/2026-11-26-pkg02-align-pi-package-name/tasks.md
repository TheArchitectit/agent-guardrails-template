# Tasks

## 1. Align package identity

- [x] 1.1 Update README installation examples and `install.mjs` usage/help to `@architectit/pi-guardrails`; retain the manifest name and `pi-guardrails` install directory.
- [x] 1.2 Add a package-identity test that asserts manifest, README, and installer help use the same canonical identity and that no install example refers to `@thearchitectit/pi-guardrails`.

## 2. Verify packaging and help output

- [x] 2.1 From `pi-extension/`, run `npm ci && npm test && npm pack --json --dry-run`; verify package metadata is `@architectit/pi-guardrails`.
- [x] 2.2 Run `install.mjs --help` with a temporary `HOME`; assert the printed `npx` identity matches metadata and the smoke test does not touch the real user home.
