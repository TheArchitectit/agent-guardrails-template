---
id: policy-config
name: Policy Configuration Hierarchy
description: How guardrail policies cascade from organization through team to project level
version: 1.0.0
tags: [safety, pi, policy]
---

# Policy Configuration Hierarchy

Guardrail policies are loaded from three layers, with later layers overriding earlier ones via deep merge. This allows organizations to set baseline rules, teams to tighten them, and projects to customize further.

## Three Policy Layers

| Layer | Location | Priority | Typical Use |
|-------|----------|----------|-------------|
| Organization | `~/.pi/guardrails-org.json` or `PI_GUARDRAILS_ORG_CONFIG` | Lowest (base) | Company-wide defaults, compliance requirements |
| Team | `~/.pi/teams/{name}/guardrails.json` or `PI_GUARDRAILS_TEAM={name}` | Middle | Team-specific rules, project-type constraints |
| Project | `{cwd}/.pi-guardrails.json` or `{cwd}/.pi/guardrails.json` | Highest | Per-project customization, local overrides |

## Deep Merge Behavior

Later layers **deep-merge** into earlier layers:

- **Primitive values**: Later layer wins (`"maxStrikes": 5` overrides `"maxStrikes": 3`)
- **Objects**: Merged recursively (team can add keys without losing org defaults)
- **Arrays**: Later layer **replaces** earlier array entirely (not concatenated)

Example:

```json
// Organization: ~/.pi/guardrails-org.json
{
  "maxStrikes": 3,
  "toolPermissions": { "tools": { "bash": "ask" } },
  "outputValidation": { "autoRedact": true }
}

// Team: ~/.pi/teams/backend/guardrails.json
{
  "maxStrikes": 5,
  "toolPermissions": { "tools": { "bash": "auto" } }
}

// Merged result for backend team:
{
  "maxStrikes": 5,                    // team overrides org
  "toolPermissions": {
    "tools": { "bash": "auto" }       // team overrides org
  },
  "outputValidation": { "autoRedact": true }  // inherited from org
}
```

## Environment Variables

| Variable | Purpose |
|----------|---------|
| `PI_GUARDRAILS_ORG_CONFIG` | Override org config file path |
| `PI_GUARDRAILS_TEAM` | Team name to load team policy |

## What This Means For You

1. **Check `guardrail_status`** at session start to see the effective configuration
2. **Do not assume** defaults — your organization may have stricter rules
3. **Respect the hierarchy** — if a tool is blocked at org level, a project override does not unblock it (unless the project policy explicitly does so)
4. **When uncertain**, default to the stricter interpretation

## MCP Bridge

When connected, `guardrail_mcp` with action `get_policy` shows the full resolved policy with source annotations, so you can see which layer set each value.

## References

- [[tool-permissions]] — Permission levels controlled by policy
- [[content-safety]] — Content filter configuration in policy
- [[output-security]] — Output validation configuration in policy
