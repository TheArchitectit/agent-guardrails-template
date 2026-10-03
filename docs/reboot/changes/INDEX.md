# Change Files

One file per reboot feature, F001–F100.

Each is a 13-line scaffold with the shape:

```
# F0XX: <title>
Status: proposed (not started) | implemented | …
Gap: …
Change: …
Acceptance: Reproducible command or artifact check recorded in CI
```

## Current state

**88 of the 100 files are untouched scaffolds.** They carry a generic
acceptance line and this instruction:

> Replace the generic acceptance line above with exact commands before building.

A file with that line still has no verifiable acceptance criterion, so it
cannot be marked done honestly. Twelve files carry real status text:
F001, F006, F013, F016, F021, F031, F033, F041, F061, F062, F067, F085.

Two features have dedicated audits one level up:
[F013-AUDIT.md](../F013-AUDIT.md), [F021-AUDIT.md](../F021-AUDIT.md).

## Working through these

A change file is finished when its acceptance line names a command someone
can run and a result that must hold — the same bar as
[AUTH-01](../../specs/AUTH-01-mcp-endpoint-auth.md), which is the worked
example of this format:

```
Acceptance (machine-checkable):
1. `cd mcp-server && go test ./internal/mcp/ -run Bearer` — met. …
```

An acceptance criterion that cannot fail is not an acceptance criterion. If
a feature's check would require a container runtime, an external model, or a
judgement call, say so in the file rather than asserting a result.