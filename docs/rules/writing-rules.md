# Writing Prevention Rules

How to write prevention rules — the regex patterns, the markdown format, and
the categories and severities. For how rules get extracted into the server,
see [extracting-rules.md](extracting-rules.md).

## Rule format

Rules are defined in markdown using a standardized section format:

```markdown
## PREVENT-001: Force Push Prohibition

**Pattern:** `git\s+push\s+.*--force`
**Severity:** error
**Category:** git
**Language:** bash
**Fix:** Use git push with standard options; never force push to shared branches

Prevents force pushing, which overwrites commit history and causes data loss
for collaborators.

### Examples

**Violations:**
```bash
git push --force origin main
git push -f origin feature-branch
```

**Compliant:**
```bash
git push origin main
```
```

### Required fields

| Field | Description | Values |
|-------|-------------|--------|
| `## PREVENT-XXX: Title` | Rule ID and name | PREVENT-001 through PREVENT-999 |
| **Pattern** | Regex to match violations | Valid Go regex (RE2) |
| **Severity** | Impact level | `critical`, `error`, `warning`, `info` |
| **Category** | Rule classification | see below |

### Optional fields

| Field | Description | Example |
|-------|-------------|---------|
| **Language** | Target language | `go`, `python`, `javascript` |
| **Fix** | Suggested remediation | `Use rm -i instead` |
| **References** | Related docs | `See agent-guardrails.md` |

## Regex patterns

The engine is Go's `regexp` (RE2 syntax). Patterns are pre-compiled and cached.
Use `(?i)` for case-insensitive matching.

| Anchor | Meaning | Example |
|--------|---------|---------|
| `^` | Start of string | `^git\s+` |
| `$` | End of string | `main$` |
| `\b` | Word boundary | `\brm\b` (matches "rm", not "remove") |

### Common patterns

**Dangerous commands**

| Pattern | Blocks |
|---------|--------|
| `(?i)rm\s+-[a-z]*rf\s*/` | `rm -rf /` |
| `(?i):\(\)\s*\{\s*:\|\:&\s*\};` | Fork bomb |
| `(?i)mkfs\.\w+\s+/dev/` | Filesystem format |
| `(?i)dd\s+.*of=/dev/[sh]` | Direct disk writes |
| `(?i)>\s*/etc/` | Overwriting system files |

**Git operations**

| Pattern | Blocks |
|---------|--------|
| `git\s+push\s+.*--force` | Force pushes |
| `git\s+push\s+.*--delete` | Branch deletion |
| `git\s+.*--hard\s+` | Hard resets |

**Secrets & credentials**

| Pattern | Detects |
|---------|---------|
| `(?i)api[_-]?key\s*[:=]\s*["'][^"']{10,}` | API keys |
| `(?i)password\s*[:=]\s*["'][^"']{6,}` | Passwords |
| `(?i)token\s*[:=]\s*["'][^"']{20,}` | Tokens |
| `bearer\s+[a-zA-Z0-9]{20,}` | Bearer tokens |
| `-----BEGIN\s+(RSA\s+)?PRIVATE\s+KEY-----` | Private keys |

**Database URIs (with embedded credentials)**

| Pattern | Detects |
|---------|---------|
| `mongodb(\+srv)?://[^:]+:[^@]+@` | MongoDB |
| `postgres(ql)?://[^:]+:[^@]+@` | PostgreSQL |
| `mysql://[^:]+:[^@]+@` | MySQL |
| `redis://:[^@]+@` | Redis |

**Container security**

| Pattern | Detects |
|---------|---------|
| `USER\s+root` | Root user in Dockerfile |
| `chmod\s+777` | World-writable permissions |
| `--privileged` | Privileged container |
| `FROM.*:latest` | Latest tag usage |

### Pattern breakdown: blocking `rm -rf /`

```
(?i)(rm\s+-[a-z]*rf|rm\s+-[a-z]*f[a-z]*r)
```

- `(?i)` — case insensitive
- `rm\s+-` — "rm" followed by whitespace and a dash
- `[a-z]*rf` — any flags ending in "rf"
- `[a-z]*f[a-z]*r` — flags with "f" before "r" (handles `-fr`, `-Rf`)

## Severity

| Severity | When to use | Example |
|----------|-------------|---------|
| **critical** | Immediate security risk, data loss | Private keys, destructive commands |
| **error** | Policy violation, potential harm | Force push, secrets in code |
| **warning** | Caution needed, review recommended | Large deletions, debug mode |
| **info** | Informational, no blocking | Statistics, suggestions |

## Categories

| Category | Use for |
|----------|---------|
| `bash` | Shell commands, system operations |
| `git` | Git operations, version control |
| `security` | Secrets, credentials, vulnerabilities |
| `docker` | Container operations |
| `code` | Code patterns (unchecked errors, race conditions) |
| `test` | Test code (test DB in prod, mock issues) |
| `general` | Cross-cutting concerns, file protections |

## Testing patterns

Test with Go's regex directly before committing:

```go
re := regexp.MustCompile(`(?i)(rm\s+-[a-z]*rf)`)
for _, cmd := range []string{"rm -rf /", "rm -Rf /tmp", "rm -f -r /var"} {
    fmt.Printf("%q matches: %v\n", cmd, re.MatchString(cmd))
}
```

Or test through the validation endpoint:

```bash
curl -X POST http://localhost:8081/mcp/validate \
  -H "Content-Type: application/json" \
  -d '{"tool":"guardrail_validate_bash","arguments":{"command":"rm -rf /tmp/test"}}'
```

Validate JSON rule files: `cat .guardrails/prevention-rules/pattern-rules.json | jq empty`.

## Best practices

1. **Be specific** — overly broad patterns cause false positives.
2. **Use word boundaries** — `\brm\b`, not `rm`.
3. **Account for whitespace** — `\s*` for optional spaces.
4. **Case-insensitive by default** — `(?i)` unless you have a reason not to.
5. **Document intent** — a comment on what the pattern targets saves the next reader.
6. **Test both ways** — verify it catches violations and passes compliant input.
