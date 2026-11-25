# Proposal

## Why

`Config.DatabaseURL` interpolates PostgreSQL usernames and passwords directly into URI userinfo. Synthetic or operator credentials containing URI-reserved punctuation produce an invalid connection URL and prevent MCP from connecting even when PostgreSQL and the migration job accept the password.

## What Changes

- Encode PostgreSQL username and password as URL userinfo using Go's standard `net/url` API.
- Preserve the existing connection URL structure and exact output for ordinary credentials.
- Add a focused test that round-trips a synthetic password containing reserved punctuation without including credential values in failure output.

## Capabilities

### New Capabilities

- `postgres-connection-credential-encoding`: Build a valid PostgreSQL connection URL when credentials contain URI-reserved characters.

### Modified Capabilities

None.

## Impact

Changes `mcp-server/internal/config/config.go` and its unit tests. It does not change credential policy, deployment variables, authentication behavior, or database contents.
