# IDE API

> See [API.md](API.md) for base URLs, authentication, and shared conventions.

These endpoints are optimized for IDE integration.

## GET /ide/health

Health check for IDE API.

**Response**
```json
{
  "status": "ok"
}
```

## POST /ide/validate/file

Validate file content against guardrails.

**Request Body**
```json
{
  "file_path": "src/main.go",
  "content": "package main\n\nfunc main() {\n  // code here\n}",
  "language": "go",
  "project_slug": "my-project"
}
```

**Response**
```json
{
  "valid": false,
  "violations": [
    {
      "rule_id": "PREVENT-003",
      "rule_name": "Hardcoded Secret",
      "severity": "error",
      "message": "Potential hardcoded secret detected",
      "line": 15,
      "column": 23,
      "suggestion": "Use environment variables instead"
    }
  ]
}
```

## POST /ide/validate/selection

Validate a code selection (for real-time validation).

**Request Body**
```json
{
  "code": "rm -rf /",
  "language": "bash",
  "context": "cleanup script"
}
```

**Response**
```json
{
  "valid": false,
  "violations": [
    {
      "rule_id": "PREVENT-002",
      "rule_name": "No rm -rf /",
      "severity": "error",
      "message": "Dangerous command detected",
      "suggestion": "Use specific paths instead"
    }
  ]
}
```

## GET /ide/rules

Get active rules for a project.

**Query Parameters**
| Name | Type | Required | Description |
|------|------|----------|-------------|
| project | string | No | Project slug (defaults to all active rules) |

**Response**
```json
{
  "data": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440001",
      "rule_id": "PREVENT-001",
      "name": "No Force Push",
      "pattern": "git push --force",
      "severity": "error",
      "message": "Force push is not allowed",
      "category": "git"
    }
  ]
}
```

## GET /ide/quick-reference

Get quick reference documentation.

**Response**
```json
{
  "data": {
    "reference": "# Quick Reference\n\n## Forbidden Commands\n- rm -rf /\n- git push --force\n\n## Required Checks\n- Pre-work check\n- Validate file edits"
  }
}
```
