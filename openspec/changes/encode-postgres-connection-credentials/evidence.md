# TPL-42 evidence

`Config.DatabaseURL` now encodes username/password userinfo with `net/url.UserPassword`. The existing ordinary-credential exact string test remains unchanged. The new test parses a URL containing synthetic reserved punctuation and verifies the decoded credentials match without printing values in any failure message.

Focused Go tests ran in the pinned Go 1.26.11 image with the checkout mounted read-only:

```text
go test ./internal/config ./internal/web
ok  github.com/thearchitectit/guardrail-mcp/internal/config
ok  github.com/thearchitectit/guardrail-mcp/internal/web
```

The Windows Compose candidate used a synthetic DB password containing URI-reserved punctuation. After the encoding fix, MCP started healthy and `GET http://127.0.0.1:18081/health/ready` returned HTTP 200 with lifecycle `READY`. The same candidate also exercised the TPL-39 migration job against its already initialized version-019 database. The recovered `guardrails-laptop-pilot` project and volumes were not changed.
