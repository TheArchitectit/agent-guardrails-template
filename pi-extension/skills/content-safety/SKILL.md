---
id: content-safety
name: Content Safety & Filtering
description: Topic-based content filtering to block harmful or unauthorized content in agent output
version: 1.0.0
tags: [safety, security, pi]
---

# Content Safety & Filtering

Content filtering supplements output validation by scanning for **topic-level** harmful content rather than specific secret patterns. It enforces organizational content policies.

## Automatic Enforcement

When configured, the content filter runs alongside output validation on every tool result. Since `tool_result` handlers are side-effect only (they cannot block content from reaching the agent), the filter detects and warns about denied content via violation logging and status bar alerts.

## Filtering Modes

### Denylist Mode

Block specific topics from appearing in any output:

```json
{
  "outputValidation": {
    "contentFilter": {
      "deniedTopics": ["violence", "hate", "self_harm", "sexual", "credentials"]
    }
  }
}
```

### Allowlist Mode (Strict)

In strict mode, **everything not on the allowed list is blocked**. Use this for highly regulated environments:

```json
{
  "outputValidation": {
    "contentFilter": {
      "allowedTopics": ["code", "documentation"],
      "strictMode": true
    }
  }
}
```

## Built-In Topic Patterns

| Topic | Patterns |
|-------|----------|
| violence | kill, murder, attack, assault, bomb, weapon, shoot, stab |
| hate | hate, bigot, racist, slur, discrimination |
| self_harm | suicide, self-harm, cutting, overdose |
| sexual | pornograph, explicit, nsfw, sexual |
| credentials | `password = ...`, `secret = ...`, `api_key = ...` |

## Custom Topics

Add organization-specific topics with custom patterns:

```json
{
  "outputValidation": {
    "contentFilter": {
      "deniedTopics": ["internal_project_names"],
      "topicPatterns": {
        "internal_project_names": ["\\bProjectX\\b", "\\bAurora\\b", "\\bTitan\\b"]
      }
    }
  }
}
```

## What To Do When Content Is Blocked

1. **Do not rephrase** the flagged content to work around the filter
2. **Inform the user** that content was flagged by policy
3. **Proceed** with the safe portions of your task
4. If the block seems incorrect, suggest the user review their content filter configuration

## MCP Bridge

When connected, `guardrail_mcp` with action `filter_content` applies server-side content policies that may be stricter than local configuration.

## References

- [[output-security]] — Secret/PII pattern scanning (complementary)
- [[injection-defense]] — Injection attack detection (complementary)
- [[policy-config]] — How policies cascade from org → team → project
