# Design

## Context

`Config.DatabaseURL` builds a PostgreSQL URL by formatting `DBUser` and `DBPassword` into the `scheme://user:password@host` form. The root Compose generator currently emits URL-safe credentials, but the configuration API accepts arbitrary credentials and the migration runner consumes the same password through a URL-independent PostgreSQL environment interface.

## Goals / Non-Goals

**Goals:**

- Encode credential userinfo so PostgreSQL URL parsing restores the original username and password.
- Keep current output unchanged for ordinary credentials and preserve the existing query parameter ordering.
- Avoid printing credential values from the new regression test.

**Non-Goals:**

- Change credential generation, validation, or storage.
- Change host, database-name, SSL, or timeout semantics.
- Change the migration runner's DSN interface.

## Decisions

- Use `url.UserPassword(c.DBUser, c.DBPassword).String()` only for the URI userinfo, then retain the current formatting for host, database path, query values, and ordering. This keeps `TestConfig_DatabaseURL`'s ordinary expected string stable while escaping reserved username/password characters.
- Parse the generated URL in the focused test and compare decoded username/password fields with synthetic inputs. Failure messages are fixed text and never format either credential or the URL.

## Risks / Trade-offs

- [An encoder changes existing ordinary URLs] → Keep the current exact-output test and verify it remains unchanged.
- [A regression test leaks its fixture] → Use generic failure messages with no credential or URL formatting.

## Migration Plan

No migration is needed. Existing connection settings retain their meaning; only URI-reserved bytes in credential userinfo are escaped during URL construction.
