# Tasks

## 1. Align template toolchains

- [x] 1.1 Update the existing GitLab MCP Go test job to select the MCP module toolchain, assert its reported `go version`, and verify its race/coverage command with the bounded 30-minute timeout.
- [x] 1.2 Update the existing Jenkins MCP tests and CLI build to select toolchains compatible with their respective module directives; verify MCP tests preserve `-race` and the CLI build produces its configured artifact.

## 2. Verify legacy runner behavior

- [x] 2.1 Run both templates on clean local runner containers without relying on automatic toolchain downloads and verify setup fails clearly when the requested version is unavailable; confirm no new example-module job was added.

## 3. Preserve the selector contract

- [x] 3.1 Add a permanent Python regression that reads the actual GitLab and Jenkins selectors and shell steps, exercises matching and mismatched toolchains with isolated stubs, and verifies tests/build stop before execution on mismatch.
- [x] 3.2 Demonstrate the regression fails against the original CI configuration and passes against the corrected configuration without requiring a Go compiler in the Python test environment.
