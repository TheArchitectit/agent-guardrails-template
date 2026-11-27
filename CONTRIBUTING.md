# Contributing

Guidelines for contributing to the Agent Guardrails Template and its MCP server.

## Quick start

1. **Fork** the repository, then clone your fork.
2. **Set up** the dev environment (prerequisites below).
3. **Create** a feature branch: `git checkout -b feature/your-feature`.
4. **Make** your changes, **test** them, **commit** with conventional commits.
5. **Push** to your fork and **open** a pull request.

## Prerequisites

| Tool | Version | Purpose |
|------|---------|---------|
| Go | 1.26.11, as declared by `mcp-server/go.mod` | MCP server (primary language) |
| Docker or Podman | recent | Container dev / dep runs |
| PostgreSQL | 16+ | Database (or use the compose stack) |
| Redis | 7+ | Cache (or use the compose stack) |
| Make | 3.81+ | Build automation |

## Repository structure

```
agent-guardrails-template/
├── mcp-server/              # Go MCP server
│   ├── cmd/server/         # entry point
│   ├── internal/           # team, mcp, database, cache, web, security, validation
│   ├── deploy/             # Dockerfile + compose
│   └── go.mod
├── pi-extension/           # TypeScript extension for the pi coding agent
├── docs/                   # documentation
├── skills/shared-prompts/  # canonical guardrail prompts
└── examples/               # per-language guardrail examples
```

The [OpenSpec index](openspec/INDEX.md) lists the canonical specifications,
active proposals, archived changes, and audit records. When a change adds or
archives a specification or proposal, update the index and validate the
affected OpenSpec documents. Preserve the status and evidence in the source
documents rather than assuming a proposal or archive is complete.
Use the [OpenSpec sprint plans](docs/sprints/plans/README.md) to identify role
assignments, client applicability and required evidence for the work.

Before opening a pull request, use the [local CI instructions](docs/ci/local-validation.md)
and, for an explicit event range, the [local event gates](docs/ci/local-event-gates.md).
The [workflow guide](docs/ci/workflows.md) explains hosted and self-hosted runner
selection, the CI parity job, and checks that require GitHub Actions.

## Go development workflow

```bash
cd mcp-server

make deps            # install dependencies
make build           # build the server binary
make dev             # run locally (needs PostgreSQL + Redis, migrations applied)

export DATABASE_URL="postgresql://guardrails:password@localhost:5432/guardrails?sslmode=disable"
make migrate-up      # apply database migrations

make fmt             # gofmt
make lint            # golangci-lint
make test            # run tests
make test-cover      # tests with coverage
make vuln            # govulncheck
make check           # fmt + lint + test
```

## Code standards

All code must pass `gofmt`, `go vet`, and `golangci-lint run`.

| Type | Convention | Example |
|------|------------|---------|
| Packages | lowercase, single word | `team`, `mcp`, `database` |
| Exported | PascalCase | `Manager`, `AssignRole` |
| Unexported | camelCase | `internalFunc` |
| Constants | PascalCase or UPPER_SNAKE | `MaxRetries`, `DEFAULT_TIMEOUT` |
| Interfaces | `-er` suffix | `Reader`, `Writer`, `Manager` |

### Error handling

Always check errors and wrap them with context:

```go
if err := validateProjectName(name); err != nil {
    return fmt.Errorf("invalid project name %q: %w", name, err)
}
```

Use custom error types for business-logic errors so callers can match with
`errors.As`:

```go
var ErrTeamFull = errors.New("team is at maximum capacity")
```

### Context

Accept `context.Context` as the first parameter on anything that does I/O:

```go
func (m *Manager) AssignRole(ctx context.Context, teamID int, role, person string) error
```

## Testing

Table-driven tests are preferred. Mock external dependencies (database, cache)
and cover both success and error paths.

```go
func TestValidateProjectName(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        wantErr bool
    }{
        {"valid", "my-project", false},
        {"empty", "", true},
        {"with space", "my project", true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateProjectName(tt.input)
            if tt.wantErr { assert.Error(t, err) } else { assert.NoError(t, err) }
        })
    }
}
```

```bash
go test ./...                    # all tests
go test -cover ./...             # with coverage
go test -race ./...              # race detector
go test -bench=. ./internal/team/...  # benchmarks
```

Coverage targets: core packages (`internal/team`, `internal/rules`) 80%+,
handlers (`internal/web`) 70%+, utilities 60%+.

## Commit guidelines

Follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>(<scope>): <subject>

[body]

[footer]
```

**Types:** `feat`, `fix`, `docs`, `style`, `refactor`, `test`, `chore`, `perf`, `security`.

**Scopes:** `team`, `mcp`, `db`, `cache`, `security`, `api`, `web`, `config`, `deploy`, `docs`.

```
feat(team): add batch assignment with transaction support

Closes #123
```

## Pull request process

1. Update documentation if your change needs it.
2. Add tests for new functionality.
3. Ensure `go test ./...` passes.
4. Update `CHANGELOG.md` under `[Unreleased]`.
5. Request review and address feedback.

### PR template

```markdown
## Summary
Brief description of changes

## Type
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
```

## Documentation

- New docs go in the appropriate `docs/` subdirectory.
- Keep every document under 500 lines — split if it grows past that.
- Add the file to `index-map.md` and the directory's `INDEX.md`.
- Match the tone and structure of existing docs.

## Questions

- **General:** open a GitHub Discussion
- **Bug reports:** open a GitHub Issue
- **Security:** see the security policy
