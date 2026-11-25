# Tasks

## 1. Align template toolchains

- [ ] 1.1 Update the existing GitLab MCP Go test job to select the MCP module toolchain, then verify its reported `go version` and existing test command.
- [ ] 1.2 Update the existing Jenkins MCP tests and CLI build to select toolchains compatible with their respective module directives, then verify MCP tests preserve `-race` and the CLI build produces its configured artifact.

## 2. Verify legacy runner behavior

- [ ] 2.1 Run both templates on clean runners without relying on automatic toolchain downloads and verify setup fails clearly when the requested version is unavailable; confirm no new example-module job was added.
