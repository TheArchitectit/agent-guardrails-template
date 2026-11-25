# Tasks

## 1. Bound the context

- [x] 1.1 Add `mcp-server/.dockerignore` for VCS metadata, local env/credential files, databases, caches, test/build artifacts, and editor files; explicitly retain `go.mod`, `go.sum`, Go source, `web/`, and `.guardrails/prevention-rules`.
- [ ] 1.2 Add a temporary-context regression test with a benign `.env` marker, `.git` directory, cache fixture, and required-source controls. Use a Docker context inspector target to assert excluded files are unavailable and required inputs are present without printing marker contents; verify with `python3 -m pytest tests/test_mcp_docker_context.py -q`.

## 2. Verify the production build

- [ ] 2.1 Run `cd mcp-server && docker build --no-cache -f deploy/Dockerfile -t guardrail-mcp:context-test .`; verify the build succeeds with the reduced context.
- [ ] 2.2 Review `.dockerignore` against the Dockerfile `COPY` inputs and record the inspected context file list without secret values.
