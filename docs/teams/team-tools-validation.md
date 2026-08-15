# Validation and Security

> Team size validation tool, input validation rules, and compliance requirements

All team tools validate their parameters to prevent command injection and ensure consistent naming. This document covers the `guardrail_team_size_validate` tool, input validation rules for each parameter, and the TEAM-007 team size compliance rule.

---

## guardrail_team_size_validate

Validate team sizes meet the 4-6 member requirement.

**Purpose:** Ensures all teams have between 4 and 6 members (inclusive) per TEAM-007 compliance rule.

**Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `project_name` | string | Yes | Name of the project |
| `team_id` | number | No | Optional: Specific team ID to validate |

**Example:**

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_size_validate",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Response:**

```
✅ All 12 teams have valid size (4-6 members)
```

Or if violations found:

```
❌ Team size violations found:
   Team 3 (GRC) has 3 members, minimum is 4
   Team 7 (Core Feature Squad) has 8 members, maximum is 6
```

---

## Input Validation Rules

### Project Name Validation

All team tools validate the `project_name` parameter to prevent command injection and ensure consistent naming:

- **Maximum Length:** 64 characters
- **Allowed Characters:**
  - Letters (a-z, A-Z)
  - Numbers (0-9)
  - Hyphens (`-`)
  - Underscores (`_`)

**Valid Examples:**
- `my-project`
- `project_123`
- `team-alpha-v2`

**Invalid Examples:**
- `my project` (contains space)
- `project;rm -rf /` (contains special characters)
- `../etc/passwd` (path traversal attempt)

### Role Name Validation

The `role_name` parameter is validated for security and consistency:

- **Maximum Length:** 128 characters
- **Required:** Yes (cannot be empty)
- **Allowed Characters:**
  - Letters (a-z, A-Z)
  - Numbers (0-9)
  - Spaces
  - Hyphens (`-`)
  - Underscores (`_`)
  - Forward slashes (`/`)
  - Ampersands (`&`)
  - Parentheses (`(` `)`)
  - Periods (`.`)
- **Forbidden Patterns:** Shell metacharacters (`;`, `|`, `&&`, `||`, backticks, `$`, `<`, `>`)

**Valid Examples:**
- `Technical Lead`
- `Senior Backend Engineer`
- `DevOps/SRE`
- `QA Architect (Automation)`

**Invalid Examples:**
- `role; rm -rf /` (contains shell metacharacters)
- `$(whoami)` (contains command substitution)
- (empty string)

### Person Name Validation

The `person` parameter is validated to ensure safe input:

- **Maximum Length:** 128 characters
- **Required:** Yes (cannot be empty)
- **Allowed Characters:**
  - Letters (a-z, A-Z)
  - Spaces
  - Hyphens (`-`)
  - Apostrophes (`'`) for names like "O'Connor"
- **Forbidden Patterns:** Path traversal, shell metacharacters, special symbols

**Valid Examples:**
- `Alice Johnson`
- `Bob O'Connor`
- `Mary-Jane Watson`

**Invalid Examples:**
- `user; cat /etc/passwd` (contains shell metacharacters)
- `../../../etc/shadow` (path traversal attempt)
- (empty string)

### Phase Validation

The optional `phase` parameter must be one of the valid phase names:

- **Valid Values:** `Phase 1`, `Phase 2`, `Phase 3`, `Phase 4`, `Phase 5`
- **Case Sensitive:** Yes
- **Required:** No (optional filter)

**Valid Examples:**
- `Phase 1`
- `Phase 3`

**Invalid Examples:**
- `phase 1` (wrong case)
- `Phase One` (invalid format)
- `1` (missing "Phase" prefix)

---

## Team Size Compliance (TEAM-007)

All teams **MUST** comply with the 4-6 member size requirement:

- **Minimum:** 4 members per team
- **Maximum:** 6 members per team
- **Rule ID:** TEAM-007
- **Severity:** Error

**Validation:**
Use `guardrail_team_size_validate` to check compliance:

```json
{
  "method": "tools/call",
  "params": {
    "name": "guardrail_team_size_validate",
    "arguments": {
      "project_name": "my-project"
    }
  }
}
```

**Why This Matters:**
- Teams with fewer than 4 members lack adequate role coverage
- Teams with more than 6 members suffer from coordination overhead
- This rule applies to human teams, AI agent teams, and mixed teams

---

## Implementation Details

Team tools use the native Go `team` package (Go 1.25+) for persistence. Project data is stored in `.teams/{project_name}.json`. The Go implementation provides the same functionality as the previous Python script with improved performance and security.

---

## Related Documentation

- [team-tools.md](./team-tools.md) - Overview of all team tools
- [team-tools-errors.md](./team-tools-errors.md) - Error code reference including validation errors
- [team-structure.md](./team-structure.md) - Complete team structure and role definitions
