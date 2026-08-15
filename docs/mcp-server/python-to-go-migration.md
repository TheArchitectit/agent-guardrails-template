# Python to Go Migration

The MCP server's team management was rewritten from Python to Go in v2.6.0.
Everything that lived in `scripts/team_manager.py` is now native Go in
`mcp-server/internal/team/`. This is a reference for anyone who worked with the
old Python implementation or is porting from it.

## Why the move

Go compiles to a single static binary, which unlocked real security and
deployment improvements the Python runtime couldn't give us:

- **Distroless containers** — no shell, no package manager, minimal attack
  surface, non-root execution with dropped capabilities.
- **Memory safety** — type checking at compile time, no interpreter
  vulnerabilities.
- **No runtime dependencies** — no `pip install`, no virtualenvs, no version
  conflicts. `go mod download` and you're done.

| Metric | Python | Go | Improvement |
|--------|--------|-----|-------------|
| Startup time | ~500ms | ~50ms | 10× faster |
| Memory usage | ~40MB | ~15MB | 2.7× less |
| Team operation | ~100ms | ~5ms | 20× faster |
| Container size | ~80MB | ~20MB | 4× smaller |

## What moved where

```
# Before (Python)
scripts/
├── team_manager.py      # team management logic
├── export_teams.py      # data export
└── migrate_config.py    # config migration

# After (Go)
mcp-server/internal/team/
├── manager.go           # team init, listing, assignment, phase tracking
├── encryption.go        # Fernet encryption at rest
├── validation.go        # project/role/name validation
├── rules.go             # layout rules, phase gates, agent-to-team mapping
├── metrics.go           # operation metrics
├── types.go             # data structures, phase definitions
└── migrations.go        # data format migration, version compat
```

## API changes

Python methods became Go methods with a `context.Context` first argument.
Exceptions became Go errors.

| Python (old) | Go (new) | Notes |
|--------------|----------|-------|
| `TeamManager.init_project()` | `Manager.InitProject(ctx)` | Added context |
| `TeamManager.list_teams()` | `Manager.ListTeams(ctx)` | Returns slice, not dict |
| `TeamManager.assign_role()` | `Manager.AssignRole(ctx, teamID, role, person)` | Param order preserved |
| `TeamManager.unassign_role()` | `Manager.UnassignRole(ctx, teamID, role)` | |
| `TeamManager.get_status()` | `Manager.GetStatus(ctx, phase)` | Phase optional |
| `EncryptionManager.encrypt()` | `Encrypt(data, key)` | Standalone function |
| `EncryptionManager.decrypt()` | `Decrypt(data, key)` | Standalone function |

```go
mgr, err := team.NewManager("my-project")
if err != nil { return err }
if err := mgr.AssignRole(ctx, 7, "Technical Lead", "Alice"); err != nil {
    var valErr *team.ValidationError
    if errors.As(err, &valErr) {
        log.Printf("invalid: %v", valErr)
    }
}
```

## Backward compatibility

**MCP tools** — fully compatible. Names, parameters, and responses are
unchanged: `guardrail_team_init`, `guardrail_team_list`, `guardrail_team_assign`,
`guardrail_team_unassign`, `guardrail_team_status`, `guardrail_phase_gate_check`,
`guardrail_agent_team_map`, `guardrail_team_size_validate`.

**Data format** — unchanged. `.teams/*.json` files work as-is.

**Config** — only Python-specific env vars were removed (`PYTHONPATH`,
`TEAM_MANAGER_SCRIPT`). `TEAM_ENCRYPTION_KEY` is still used.

## Container changes

```dockerfile
# Before — Python runtime required
FROM gcr.io/distroless/python3-debian12
COPY scripts/ /app/scripts/
RUN pip install cryptography

# After — pure Go static binary
FROM gcr.io/distroless/static:nonroot
COPY server /server
ENTRYPOINT ["/server"]
```

Drop the scripts volume and `PYTHONPATH` from compose:

```yaml
# Before
services:
  mcp-server:
    volumes: [./scripts:/app/scripts:ro]
    environment: [PYTHONPATH=/app]

# After
services:
  mcp-server:
    environment: [TEAM_ENCRYPTION_KEY=${TEAM_ENCRYPTION_KEY}]
```

## FAQ

**Do I need to learn Go to use the server?** No — it's a black box from the
client side. You interact through MCP or the REST API.

**Will my existing `.teams/*.json` files work?** Yes, unchanged.

**What happened to `scripts/team_manager.py`?** Migrated to
`mcp-server/internal/team/`. The Python script is deprecated.

**Do I need Go installed to run the server?** No — use the pre-built Docker
image. You only need Go to build from source (Go 1.25+).

**Any breaking changes?** None from the MCP client perspective.

**How do I debug?**
```bash
go build -gcflags="-N -l" -o bin/server ./cmd/server   # debug symbols
dlv exec ./bin/server                                   # delve
LOG_LEVEL=debug ./bin/server                            # debug logging
```

## Troubleshooting

**"team_manager.py not found"** — old code references the Python script. Replace
`exec.Command("python", "team_manager.py", ...)` with a direct Go call:
`mgr, _ := team.NewManager(projectName); mgr.ListTeams(ctx)`.

**Encryption key not working** — the key must be 32 bytes (Fernet format).
Generate one: `openssl rand -base64 32`.

**Build failures** — `cd mcp-server && go mod download && go build ./cmd/server`.

## References

- [Team tools reference](../teams/team-tools.md) — MCP tool documentation
- [Architecture](../architecture/system-architecture.md)
- [Go documentation](https://golang.org/doc/)
