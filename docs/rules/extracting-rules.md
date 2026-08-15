# Extracting Prevention Rules

How prevention rules travel from markdown source files to live MCP tools. For
the rule format and regex patterns themselves, see
[writing-rules.md](writing-rules.md).

## The flow

```
Markdown file ─▶ RuleParser ─▶ ParsedRule ─▶ PreventionRule (DB) ─▶ MCP tool ─▶ AI agent
(docs/*.md)     (ParseRules)   (struct)       (model)              (ValidateXxx)
```

Rules defined in markdown are parsed, stored in PostgreSQL, and exposed as MCP
validation tools that agents call before running commands.

## Where rules live

| Location | Purpose | Format |
|----------|---------|--------|
| `.guardrails/prevention-rules/pattern-rules.json` | Regex-based rules | JSON array |
| `.guardrails/prevention-rules/semantic-rules.json` | AST-based rules | JSON array |
| `.guardrails/prevention-rules/extracted-rules.json` | Rules extracted from markdown | JSON array |
| `docs/*.md` | Source documentation | Markdown |

### Pattern rule (JSON)

```json
{
  "id": "PREVENT-001",
  "name": "Force Push Prohibition",
  "pattern": "git\\s+push\\s+.*--force",
  "severity": "error",
  "category": "git",
  "description": "Prevents force pushing to git repositories",
  "examples": {
    "violation": "git push --force origin main",
    "compliant": "git push origin main"
  }
}
```

### Semantic rule (JSON, AST-aware)

```json
{
  "id": "PREVENT-101",
  "name": "Hardcoded Credentials",
  "language": "go",
  "pattern": "password|token|secret|key",
  "severity": "error",
  "category": "security",
  "ast_context": "assignment"
}
```

## Parsing markdown into rules

The `RuleParser` scans markdown and extracts rule sections into a `ParsedRule`:

```go
parser := ingest.NewRuleParser()
content, _ := os.ReadFile("docs/AGENT_GUARDRAILS.md")
rules, err := parser.ParseRules(string(content), "AGENT_GUARDRAILS.md")
for _, rule := range rules {
    fmt.Printf("%s: %s\n", rule.ID, rule.Name)
}
```

`ParsedRule` carries the parsed fields:

```go
type ParsedRule struct {
    ID, Name, Pattern, Severity, Category, Description string
    Examples []string
    Language, Fix, FilePath string
}
```

## Storing rules

Parsed rules become `PreventionRule` models in the database:

```go
rule := &models.PreventionRule{
    Code:        parsed.ID,
    Name:        parsed.Name,
    Description: parsed.Description,
    Pattern:     parsed.Pattern,
    Severity:    models.Severity(parsed.Severity),
    Category:    models.Category(parsed.Category),
    Language:    parsed.Language,
    Fix:         parsed.Fix,
    Source:      "markdown",
    Enabled:     true,
}
db.CreateRule(ctx, rule)
```

### Schema

```sql
CREATE TABLE prevention_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code VARCHAR(50) UNIQUE NOT NULL,   -- PREVENT-XXX
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern TEXT,                       -- regex
    severity VARCHAR(20) NOT NULL,      -- error, warning, info
    category VARCHAR(50) NOT NULL,      -- git, bash, security, etc.
    language VARCHAR(50),
    fix TEXT,
    source VARCHAR(50) NOT NULL,        -- markdown, json, manual
    version INTEGER DEFAULT 1,
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);
```

## Generating MCP tools

Each stored rule becomes a validation tool:

```go
func (s *Server) generateRuleTools(rules []models.PreventionRule) []Tool {
    tools := make([]Tool, 0, len(rules))
    for _, rule := range rules {
        tools = append(tools, Tool{
            Name:        fmt.Sprintf("validate_%s", rule.Code),
            Description: fmt.Sprintf("%s: %s", rule.Name, rule.Description),
            InputSchema: ToolInputSchema{
                Type: "object",
                Properties: map[string]SchemaProperty{
                    "command": {Type: "string", Description: "Command or code to validate"},
                },
                Required: []string{"command"},
            },
        })
    }
    return tools
}
```

## Syncing rules

### Manual sync via API

```bash
curl -X POST http://localhost:8081/api/ingest/sync \
  -H "Authorization: Bearer $MCP_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"source":"repo","paths":[".guardrails/prevention-rules"]}'
```

Response:

```json
{
  "job_id": "550e8400-...",
  "status": "completed",
  "files_processed": 3,
  "rules_added": 2,
  "rules_updated": 1,
  "rules_orphaned": 0
}
```

### Programmatic sync

```go
func syncRules(ctx context.Context, service *ingest.Service) error {
    jobID := uuid.New()
    if err := service.SyncFromRepo(ctx, jobID); err != nil {
        return fmt.Errorf("sync failed: %w", err)
    }
    return nil
}
```

### Auto-sync on file changes

The server can watch the rules directories and re-sync automatically. Triggers:

| Event | Action |
|-------|--------|
| Markdown file modified | Re-parse and update rules |
| JSON rule file modified | Reload and validate |
| New file added | Parse and add new rules |
| File deleted | Mark rules as orphaned |

## Web UI

Rules are browsable at the Web UI (`/web`) and via the REST API:

```bash
curl http://localhost:8081/api/rules | jq '.data | length'   # count
curl http://localhost:8081/api/rules/PREVENT-001 | jq         # one rule
```

Toggle a rule's enabled state:

```bash
curl -X PUT http://localhost:8081/api/rules/PREVENT-001 \
  -H "Authorization: Bearer $MCP_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{"enabled": false}'
```

## Troubleshooting

| Issue | Cause | Fix |
|-------|-------|-----|
| Rules not appearing | Sync not run | Trigger manual sync or restart |
| Pattern not matching | Invalid regex | Test the pattern in isolation |
| Duplicate rules | Same rule in multiple files | Check file paths and rule IDs |
| Rules marked orphaned | Source file deleted | Restore the file or disable orphan cleanup |

Debug:

```bash
curl http://localhost:8081/api/rules | jq '.data | length'   # rule count
curl http://localhost:8081/api/ingest/jobs | jq              # sync history
```

## References

- [Writing rules](writing-rules.md) — rule format and regex patterns
- [Agent guardrails](../getting-started/agent-guardrails.md) — core safety protocols
- [MCP tools reference](../mcp-server/tools-reference.md)
