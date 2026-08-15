---
name: guardrails-docs-access
description: Access and read guardrails skill documentation, configuration examples, and usage guides at runtime.
version: 1.0.0
applies_to:
  - pi
---

# Guardrails Docs Access

Provides runtime access to guardrails skill documentation and configuration guides.

## Skill Documentation Index

| Skill | Description | Config Reference |
|-------|-------------|------------------|
| `guardrails-core` | Core guardrails lifecycle, tools, and coverage map | `.guardrails/config.json` |
| `injection-defense` | Prompt injection detection and blocking | `injection.patterns[]`, `injection.confidenceThreshold` |
| `output-security` | Output validation for secrets, PII, and code patterns | `outputValidation.rules[]` |
| `content-safety` | Topic-based content filtering (deny/allow lists) | `outputValidation.contentFilter.deniedTopics` |
| `tool-permissions` | Per-tool allow/block/ask permission policies | `toolPermissions.tools{}` |
| `policy-config` | Dynamic policy management with rollback | `policy.name`, `policy.version` |
| `sandbox-isolation` | Container sandbox configuration and enforcement | `sandbox.enabled`, `sandbox.type` |
| `canary-tokens` | Canary token generation, embedding, and detection | `canary.prefix`, `canary.tokenLength` |
| `guardrails-dashboard` | Dashboard UI for violation monitoring and controls | N/A |

## Usage

### List All Skills
```typescript
const skills = guardrail_list_skills(); // Returns array of skill metadata
```

### Read a Specific Skill
```typescript
const content = guardrail_read_skill({ skill: "injection-defense" }); // Returns SKILL.md content
```

### Read Language Rule Files
```typescript
const rules = guardrail_read_language_rules({ language: "python" }); // Returns rules JSON
```

### Get Available Languages
```typescript
const langs = guardrail_list_languages(); // Returns ["python", "typescript", "go", "rust"]
```

## Configuration Templates

### Minimal Config
```json
{
  "enabled": true,
  "enabledRules": ["four-laws", "scope-validator", "halt-when-uncertain"]
}
```

### With Output Validation
```json
{
  "enabled": true,
  "enabledRules": ["four-laws", "scope-validator", "halt-when-uncertain"],
  "outputValidation": {
    "enabled": true,
    "rules": ["api_key_generic", "aws_access_key"],
    "contentFilter": {
      "deniedTopics": ["malicious code"]
    }
  }
}
```

### With Tool Permissions
```json
{
  "enabled": true,
  "toolPermissions": {
    "bash": "ask",
    "write": "allow",
    "edit": "allow"
  }
}
```

### With Sandbox
```json
{
  "enabled": true,
  "sandbox": {
    "enabled": true,
    "type": "docker",
    "image": "node:18-slim"
  }
}
```

### With Canary Tokens
```json
{
  "enabled": true,
  "canary": {
    "enabled": true,
    "prefix": "CATALOG:",
    "tokenLength": 32
  }
}
```
