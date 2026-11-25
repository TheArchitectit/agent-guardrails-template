# Tasks

## 1. Encode PostgreSQL credentials

- [x] 1.1 Use standard URL userinfo encoding in `Config.DatabaseURL` while preserving ordinary output and the existing host/database/query format.
- [x] 1.2 Add a synthetic reserved-character password round-trip test whose failure output contains no credential or URL values.

## 2. Verify

- [x] 2.1 Run the focused configuration package tests and strict OpenSpec validation.
- [x] 2.2 Rebuild the isolated TPL-39 Compose project using the existing synthetic database and verify MCP reaches HTTP 200 readiness without modifying the recovered pilot.
