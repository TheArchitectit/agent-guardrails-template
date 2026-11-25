# Tasks

## 1. Declare development-only SDK support

- [x] 1.1 Add a compatible `@modelcontextprotocol/sdk` version to `devDependencies` and update `package-lock.json`; verify the SDK is absent from production `dependencies` and optional peer metadata is unchanged.
- [x] 1.2 Add SDK-present tests for dynamic subpath loading and bridge initialization, plus an SDK-absent test that verifies standalone package load succeeds and MCP bridge reports unavailable.

## 2. Verify both install shapes

- [x] 2.1 From `pi-extension/`, run `npm ci && npm test`; verify bridge tests run with the lockfile-installed SDK.
- [x] 2.2 Run `npm pack --json --dry-run`, install the packed package into an isolated temp prefix with production dependencies only, and run the package standalone-load smoke test; verify the MCP SDK is not required and absence is reported without throwing.
