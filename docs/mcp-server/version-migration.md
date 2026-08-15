# Migration Guide

> Version compatibility, migration instructions, and rollback procedures

---

## Topic Guides

| Topic | File | Covers |
|-------|------|--------|
| Breaking changes | [migration-breaking-changes.md](migration-breaking-changes.md) | What changed in each release, version by version |
| Procedures | [migration-procedures.md](migration-procedures.md) | Step-by-step upgrade paths for v2.6.0, v1.10.0, v1.9.0 |
| Rollback | [migration-rollback.md](migration-rollback.md) | Automatic triggers, manual rollback, emergency recovery |
| Examples | [migration-examples.md](migration-examples.md) | Single-project, batch, and zero-downtime migration scripts |
| Troubleshooting | [migration-troubleshooting.md](migration-troubleshooting.md) | Common migration issues and a verification script |

---

## Version Compatibility Matrix

### Current Version Support

| Version | Status | Compatible With | Implementation |
|---------|--------|-----------------|----------------|
| 3.3.x | Current | 3.x | **Go** (StreamableHTTP) |
| 3.2.x | Maintained | 3.x | Go |
| 2.6.x | Maintained | 2.0.x, 1.10.x | Go |
| 1.10.x | Legacy | 1.9.x | Go |
| 1.8.x | Deprecated | 1.7.x | Python |
| < 1.8.0 | End of life | — | Python |

### Compatibility Legend

| Symbol | Meaning |
|--------|---------|
| Full | All features compatible |
| Partial | Some features require configuration |
| Breaking | Requires migration steps |
| N/A | Not compatible |

### MCP Server Compatibility

| Client Version | MCP Server 1.9 | MCP Server 1.10 | MCP Server 2.0 |
|----------------|----------------|-----------------|----------------|
| Claude Code 1.x | Full | Full | Full |
| Claude Code 2.x | Partial | Full | Full |
| OpenCode 1.x | Full | Full | Full |
| Cursor 1.x | N/A | Partial | Full |
| Custom Clients | Breaking | Partial | Full |

### Database Compatibility

| Database Version | Schema Version | Migration Required |
|------------------|----------------|------------------|
| PostgreSQL 15 | v1.8 | Yes |
| PostgreSQL 16 | v1.9+ | No |
| Redis 6 | v1.8 | Yes |
| Redis 7 | v1.9+ | No |

---

## Quick Start

If you are upgrading from v1.8.x or earlier, you need the full migration path:

1. Review [breaking changes](migration-breaking-changes.md) for your target version
2. Run the [pre-migration checklist](migration-procedures.md) and follow the procedure for your version
3. Keep [rollback procedures](migration-rollback.md) handy in case something goes wrong
4. Use the [migration examples](migration-examples.md) for single-project, batch, or zero-downtime patterns
5. If something breaks, check [migration troubleshooting](migration-troubleshooting.md)

For minor version bumps within v2.x (e.g., v2.0.0 to v2.6.0), the process is straightforward — the Go build and API are compatible. See the v2.6.0 procedure in [migration procedures](migration-procedures.md).
