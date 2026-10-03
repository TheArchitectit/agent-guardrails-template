# MCP Tools Reference

The guardrails MCP server registers **37 core tools**, dispatched from the
switch in `mcp-server/internal/mcp/server.go:163-270` and defined by
`toolList()` in `tools_registry.go`.

A further **21 tools** are registered only when their backing store is
initialised — 6 vision, 5 webhook, 5 budget, 5 agent-lifecycle. So a
deployment exposes between 37 and 58 tools depending on configuration; see
[Conditionally registered tools](tools/conditional-and-integrations.md).

This page is the index; the detail pages carry per-tool parameters, outputs
and policy behaviour.

> **Previous versions of this file documented only 3 tools.** If you learned
> the API from it, assume anything not listed in the table below is new to
> you — and check the handler before trusting a parameter name, because the
> published schemas are unreliable (see
> [Shared behaviour](#shared-behaviour)).

## Detailed references

| Page | Tools | Covers |
|------|-------|--------|
| [Core validation](tools/core-validation.md) | 7 | bash / file-edit / git validation, scope, session |
| [Workflow and git](tools/workflow-and-git.md) | 8 | commit, push, regression, production-first, replacements |
| [Halt, attempts and content](tools/halt-attempts-content.md) | 12 | provenance, three strikes, halt conditions, classification |
| [Teams and advisors](tools/teams-and-advisors.md) | 10 | project lifecycle, assignment, advisors |
| [Conditionally registered](tools/conditional-and-integrations.md) | 21 | vision, webhooks, budgets, agent lifecycle |

## Quick reference

### Core validation

| Tool | Purpose | Key parameters |
|------|---------|----------------|
| `guardrail_init_session` | Create a session id | `user_id`, `environment` |
| `guardrail_validate_bash` | Validate a bash command | `command`, `working_dir` |
| `guardrail_validate_file_edit` | Validate an edit + read-before-edit | `file_path`, `old_string`, `new_string`, `session_token` |
| `guardrail_validate_git_operation` | Validate a git operation | `operation`, `args[]` |
| `guardrail_pre_work_check` | Check planned work against known regressions | `task_description` |
| `guardrail_get_context` | Rule count for a path | `path` |
| `guardrail_validate_scope` | Path within authorized scope | `file_path`, `authorized_scope` |

### Workflow and git

| Tool | Purpose | Key parameters |
|------|---------|----------------|
| `guardrail_validate_commit` | Conventional Commits check | `message` |
| `guardrail_prevent_regression` | Match planned changes to failure registry | `file_paths[]`, `code_content` |
| `guardrail_check_test_prod_separation` | Test/prod isolation | `file_path`, `environment` |
| `guardrail_validate_push` | Pre-push branch safety | `branch`, `is_force`, `has_unpushed_commits` |
| `guardrail_validate_production_first` | Production code before test code | `session_token`, `file_path`, `code_type` |
| `guardrail_detect_feature_creep` | Diff exceeds task scope | `session_token`, `file_path`, `git_diff` |
| `guardrail_verify_fixes_intact` | Recorded fixes still present | `session_token`, `file_path`, `modified_content` |
| `guardrail_validate_exact_replacement` | Edit was an exact replacement | `session_token`, `file_path`, `original_content`, `modified_content` |

### Halt, attempts and content

| Tool | Purpose | Key parameters |
|------|---------|----------------|
| `guardrail_record_file_read` | Record a file read | `session_token`, `file_path` |
| `guardrail_verify_file_read` | Verify read-before-edit | `session_token`, `file_path` |
| `guardrail_record_attempt` | Record a failed attempt | `session_token`, `task_id`, `error_message`, `error_category` |
| `guardrail_reset_attempts` | Clear attempt counter | `session_token`, `task_id` |
| `guardrail_validate_three_strikes` | Three-strikes threshold | `session_token`, `task_id` |
| `guardrail_check_uncertainty` | Self-reflection + escalation level | `session_token`, `current_task`, `self_assessment`, `context_data` |
| `guardrail_check_halt_conditions` | Does this need a human? | `session_token`, `context`, `task_id` |
| `guardrail_record_halt` | Record a halt event | `session_token`, `halt_type`, `description`, `severity` |
| `guardrail_acknowledge_halt` | Acknowledge and resume | `session_token`, `halt_id`, `resolution` |
| `guardrail_classify_content` | S1–S15 taxonomy classification | `text`, `direction` |
| `guardrail_check_policy` | Check against a named policy | `text`, `policy_id` |
| `guardrail_install_skills` | Install skill configs | `action`, `platforms`, `mode`, `dry_run` |

### Teams and advisors

| Tool | Purpose | Key parameters |
|------|---------|----------------|
| `guardrail_team_init` | Create project + 12 teams | `project_name` |
| `guardrail_team_list` | List teams | `project_name`, `phase` |
| `guardrail_team_config_get` | Read project config | `project_name`, `team_name` |
| `guardrail_team_config_update` | Merge config fragment | `project_name`, `config` |
| `guardrail_team_assign` | Assign a person to a role | `project_name`, `team_id`, `role_name`, `person` |
| `guardrail_team_remove` | Remove a team | `project_name`, `team_id`, `confirmed` |
| `guardrail_project_delete` | Delete the project | `project_name`, `confirmed` |
| `guardrail_team_health` | Config health report | `project_name` |
| `guardrail_advisor_list` | List advisors | — |
| `guardrail_advisor_query` | Consult an advisor | `advisor_id`, `context`, `file_paths[]` |

## Shared behaviour

### Result envelope

Every tool returns a single `mcp.TextContent` holding a JSON object. The
second argument to the result helper becomes MCP's `IsError` flag. There are
three slightly different helpers — `jsonToolResult`, `buildToolResult` and
`errorResult` — with identical observable behaviour, so clients cannot tell
them apart.

### The published schemas are not trustworthy

This is the most important thing to know before integrating a client:

- **Handlers frequently read parameters the schema does not declare.** Nearly
  every tool that needs a session reads `session_token`, which is absent from
  the published `InputSchema`.
- **Schemas frequently declare parameters handlers ignore.** Declared-but-dead
  examples: `files` on `validate_commit`, `remote` on `validate_push`,
  `bug_id` on `verify_fixes_intact`, `target_string` on
  `validate_exact_replacement`, `advisor_name`/`role` on `team_assign`,
  `advisor_name`/`query` on `advisor_query`, `teams` on `team_init`,
  `context` on `classify_content`.
- **Required-ness differs between schema and handler** in both directions —
  `old_string`/`new_string` are schema-required but not enforced, while
  `authorized_scope`, `session_token`, `role_name` and `code_type` are
  required in code but not in the schema.

**Treat the handler as the contract.** Where this document and a schema
disagree, the documented value is what the code actually does.

### Fail-closed vs advisory

Two distinct postures, and neither is universal:

- **Fail closed** — the tool returns `IsError=true` (or `valid:false`) when
  it cannot prove the action is safe. Used by the `validate_*` family.
- **Advisory** — the tool reports a signal the caller decides what to do
  with; it does not block. Used by `validate_three_strikes`,
  `check_halt_conditions`, `verify_file_read` and `get_context`.

Two tools fail **open** by design and should not be relied on as gates:
`classify_content` (returns `safe:true` when the engine is absent) and
`check_test_prod_separation` (produces no violations for an unreadable
file).

### Sessions

Tools taking `session_token` validate it against the in-memory `s.sessions`
map. That map is **never written by `guardrail_init_session`** — see
[Core validation](tools/core-validation.md#guardrail_init_session). In
practice these tools reject freshly issued tokens with `"Invalid session
token"`.

### Security posture

`guardrail_project_delete` and `guardrail_team_config_update` are destructive
and run with no authentication beyond the bearer token, no backup and no
audit trail. `guardrail_team_*` handlers contain no RBAC at all. Treat MCP
bearer auth as the only access control on this server.

## Related

- [Team tools](../teams/team-tools.md)
- [Architecture](../architecture/system-architecture.md)
- [Security audit — API](../security/security-audit-api.md)