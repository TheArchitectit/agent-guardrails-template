# Tasks

## 1. Deterministic test fixtures

- [x] 1.1 Update the four direct negative-permission fixtures and dependent
  mutation baseline to set and verify the intended permissive mode explicitly;
  from `mcp-server/`, run
  `(umask 0077; go test ./internal/auth ./internal/config ./internal/secretfile ./internal/mutationkill)`
  and confirm all targeted tests pass.

## 2. Production mode regression checks

- [x] 2.1 Preserve the production owner-only secret-file assertions and verify
  the same affected packages under `(umask 0022; go test ./internal/auth ./internal/config ./internal/secretfile ./internal/mutationkill)`
  and `(umask 0077; go test ./internal/auth ./internal/config ./internal/secretfile ./internal/mutationkill)`.
